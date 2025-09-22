package main

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
	"xray-checker/checker"
	"xray-checker/config"
	"xray-checker/metrics"
	"xray-checker/runner"
	"xray-checker/subscription"
	"xray-checker/web"
	"xray-checker/xray"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = "unknown"
)

func main() {
	config.Parse(version)
	log.Printf("Xray Checker %s starting...\n", version)

	configFile := "xray_config.json"
	proxyConfigs, err := subscription.InitializeConfiguration(configFile)
	if err != nil {
		log.Fatalf("Error initializing configuration: %v", err)
	}

	xrayRunner := runner.NewXrayRunner(configFile)
	if err := xrayRunner.Start(); err != nil {
		log.Fatalf("Error starting Xray: %v", err)
	}

	defer func() {
		if err := xrayRunner.Stop(); err != nil {
			log.Printf("Error stopping Xray: %v", err)
		}
	}()

	metrics.InitMetrics(config.CLIConfig.Metrics.Instance)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.GetProxyStatusMetric())
	registry.MustRegister(metrics.GetProxyLatencyMetric())

	proxyChecker := checker.NewProxyChecker(
		*proxyConfigs,
		config.CLIConfig.Xray.StartPort,
		config.CLIConfig.Proxy.IpCheckUrl,
		config.CLIConfig.Proxy.Timeout,
		config.CLIConfig.Proxy.StatusCheckUrl,
		config.CLIConfig.Proxy.DownloadUrl,
		config.CLIConfig.Proxy.DownloadTimeout,
		config.CLIConfig.Proxy.DownloadMinSize,
		config.CLIConfig.Proxy.CheckMethod,
		config.CLIConfig.Metrics.Instance,
	)

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

	if config.CLIConfig.RunOnce {
		runCheckIteration()
		log.Println("Single check iteration completed, exiting...")
		return
	}

	var needsUpdate atomic.Bool
	s := gocron.NewScheduler(time.UTC)
	s.Every(config.CLIConfig.Proxy.CheckInterval).Seconds().Do(func() {
		if config.CLIConfig.Subscription.Update && needsUpdate.Swap(false) {
			log.Printf("Updating subscription...")
			newConfigs, err := subscription.ReadFromSource(config.CLIConfig.Subscription.URL)
			if err != nil {
				log.Printf("Error checking subscription updates: %v", err)
			} else if !xray.IsConfigsEqual(*proxyConfigs, newConfigs) {
				if err := xray.UpdateConfiguration(newConfigs, proxyConfigs, xrayRunner, proxyChecker); err != nil {
					log.Printf("Error updating configuration: %v", err)
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

	web.RegisterConfigEndpoints(*proxyConfigs, proxyChecker, config.CLIConfig.Xray.StartPort)
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
