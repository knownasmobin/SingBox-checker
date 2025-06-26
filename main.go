package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
	"github.com/knownasmobin/singbox-checker/checker"
	"github.com/knownasmobin/singbox-checker/config"
	"github.com/knownasmobin/singbox-checker/metrics"
	"github.com/knownasmobin/singbox-checker/models"
	"github.com/knownasmobin/singbox-checker/runner"
	"github.com/knownasmobin/singbox-checker/subscription"
	"github.com/knownasmobin/singbox-checker/web"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = "unknown"
)

func main() {
	config.Parse(version)
	log.Printf("SingBox Checker %s starting...\n", version)

	// Ensure config directory exists
	if err := os.MkdirAll(config.CLIConfig.SingBox.ConfigDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	configFile := filepath.Join(config.CLIConfig.SingBox.ConfigDir, "singbox.json")
	
	// Read proxy configs from subscription
	proxyConfigs, err := subscription.ReadFromSource(config.CLIConfig.Subscription.URL)
	if err != nil {
		log.Fatalf("Error reading subscription: %v", err)
	}

	// Initialize SingBox configuration
	if err := subscription.InitializeSingBoxConfig(proxyConfigs, configFile); err != nil {
		log.Fatalf("Error initializing SingBox config: %v", err)
	}

	singboxRunner := runner.NewSingBoxRunner(configFile)
	if err := singboxRunner.Start(); err != nil {
		log.Fatalf("Error starting SingBox: %v", err)
	}

	defer func() {
		if err := singboxRunner.Stop(); err != nil {
			log.Printf("Error stopping SingBox: %v", err)
		}
	}()

	metrics.InitMetrics(config.CLIConfig.Metrics.Instance)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.GetProxyStatusMetric())
	registry.MustRegister(metrics.GetProxyLatencyMetric())

	proxyChecker := checker.NewProxyChecker(
		proxyConfigs,
		config.CLIConfig.SingBox.StartPort,
		config.CLIConfig.Proxy.IpCheckUrl,
		config.CLIConfig.Proxy.Timeout,
		config.CLIConfig.Proxy.StatusCheckUrl,
		config.CLIConfig.Proxy.CheckMethod,
		config.CLIConfig.Metrics.Instance,
	)

	// Create a slice to hold the current proxy configs
	currentConfigs := make([]*models.ProxyConfig, len(proxyConfigs))
	copy(currentConfigs, proxyConfigs)

	var needsUpdate atomic.Bool
	s := gocron.NewScheduler(time.UTC)
	s.Every(config.CLIConfig.Proxy.CheckInterval).Seconds().Do(func() {
		if config.CLIConfig.Subscription.Update && needsUpdate.Swap(false) {
			log.Printf("Updating subscription...")
			newConfigs, err := subscription.ReadFromSource(config.CLIConfig.Subscription.URL)
			if err != nil {
				log.Printf("Error checking subscription updates: %v", err)
			} else if !isConfigsEqual(currentConfigs, newConfigs) {
				// Update SingBox configuration with new proxy configs
				if err := subscription.UpdateSingBoxConfig(singboxRunner, newConfigs); err != nil {
					log.Printf("Error updating SingBox configuration: %v", err)
				}
				// Update proxy checker with new configs
				proxyChecker.UpdateProxies(newConfigs)
				// Update current configs
				currentConfigs = make([]*models.ProxyConfig, len(newConfigs))
				copy(currentConfigs, newConfigs)
			}
		}
		runCheckIteration := func() {
			log.Printf("Starting proxy check iteration...")
			proxyChecker.CheckAllProxies()

			if config.CLIConfig.Metrics.PushURL != "" {
				pushConfig, err := metrics.ParseURL(config.CLIConfig.Metrics.PushURL)
				if err != nil {
					log.Printf("Error parsing push URL: %v", err)
					return
				}

				if pushConfig != nil {
					if err := metrics.PushMetrics(pushConfig, registry); err != nil {
						log.Printf("Error pushing metrics: %v", err)
					}
				}
			}
		}
		runCheckIteration()
	})
	s.StartAsync()

	if config.CLIConfig.Subscription.Update {
		updateScheduler := gocron.NewScheduler(time.UTC)
		updateScheduler.Every(config.CLIConfig.Subscription.UpdateInterval).Seconds().Do(func() {
			needsUpdate.Store(true)
		})
		updateScheduler.StartAsync()
	}

	mux, err := web.NewPrefixServeMux(config.CLIConfig.Metrics.BasePath)
	if err != nil {
		log.Fatalf("Error create web server: %v", err)
	}
	mux.Handle("/health", web.HealthHandler())

	protectedHandler := http.NewServeMux()
	protectedHandler.Handle("/", web.IndexHandler(version, proxyChecker))
	protectedHandler.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	web.RegisterConfigEndpoints(proxyConfigs, proxyChecker, config.CLIConfig.SingBox.StartPort)
	protectedHandler.Handle("/config/", web.ConfigStatusHandler(proxyChecker))

	if config.CLIConfig.Metrics.Protected {
		middlewareHandler := web.BasicAuthMiddleware(
			config.CLIConfig.Metrics.Username,
			config.CLIConfig.Metrics.Password,
		)(protectedHandler)
		mux.Handle("/", middlewareHandler)
	} else {
		mux.Handle("/", protectedHandler)
	}

	if !config.CLIConfig.RunOnce {
		log.Printf("Starting server on %s:%s",
			config.CLIConfig.Metrics.Host,
			config.CLIConfig.Metrics.Port+config.CLIConfig.Metrics.BasePath,
		)
		if err := http.ListenAndServe(config.CLIConfig.Metrics.Host+":"+config.CLIConfig.Metrics.Port, mux); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}
}

// isConfigsEqual compares two slices of proxy configurations for equality
func isConfigsEqual(a, b []*models.ProxyConfig) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps to store the stable IDs of the configurations
	aMap := make(map[string]bool)
	for _, config := range a {
		aMap[config.GenerateStableID()] = true
	}

	// Check if all configs in b exist in a
	for _, config := range b {
		if !aMap[config.GenerateStableID()] {
			return false
		}
	}

	return true
}
