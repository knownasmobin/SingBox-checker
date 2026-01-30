package main

import (
	"net/http"
	"strings"
	"time"
	"xray-checker/checker"
	"xray-checker/config"
	"xray-checker/logger"
	"xray-checker/metrics"
	"xray-checker/models"
	"xray-checker/runner"
	"xray-checker/singbox"
	"xray-checker/subscription"
	"xray-checker/web"
	"xray-checker/xray"

	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version   = "unknown"
	startTime = time.Now()
)

func main() {
	config.Parse(version)

	logLevel := logger.ParseLevel(config.CLIConfig.LogLevel)
	logger.SetLevel(logLevel)

	logger.Startup("Proxy Checker %s (backend: %s)", version, config.CLIConfig.Backend)
	if logLevel == logger.LevelNone {
		logger.Startup("Log level: none (silent mode)")
	}

	if err := web.InitAssetLoader(config.CLIConfig.Web.CustomAssetsPath); err != nil {
		logger.Fatal("Failed to initialize custom assets: %v", err)
	}

	geoManager := xray.NewGeoFileManager("")
	if err := geoManager.EnsureGeoFiles(); err != nil {
		logger.Fatal("Failed to ensure geo files: %v", err)
	}

	var configFile string
	if config.CLIConfig.Backend == "singbox" {
		configFile = "singbox_config.json"
	} else {
		configFile = "xray_config.json"
	}

	proxyConfigs, err := subscription.InitializeConfiguration(configFile, version)
	if err != nil {
		logger.Fatal("Error initializing configuration: %v", err)
	}

	logger.Info("Loaded %d proxy configurations", len(*proxyConfigs))

	if config.CLIConfig.Web.Public {
		if name := subscription.GetSubscriptionName(); name != "" {
			logger.Info("Subscription name for public status page: %s", name)
		}
	} else {
		subNames := web.CollectSubscriptionNames(*proxyConfigs)
		if len(subNames) > 0 {
			logger.Info("Subscriptions: %s", strings.Join(subNames, ", "))
		}
	}

	if logLevel == logger.LevelDebug {
		logger.Debug("=== Parsed Proxy Configurations ===")
		for _, pc := range *proxyConfigs {
			logger.Debug("%s", pc.DebugString())
		}
	}

	var proxyRunner runner.Runner
	if config.CLIConfig.Backend == "singbox" {
		proxyRunner = singbox.NewRunner(configFile)
		if err := proxyRunner.Start(); err != nil {
			logger.Fatal("Error starting sing-box: %v", err)
		}
	} else {
		proxyRunner = xray.NewRunner(configFile)
		if err := proxyRunner.Start(); err != nil {
			logger.Fatal("Error starting Xray: %v", err)
		}
	}

	defer func() {
		if err := proxyRunner.Stop(); err != nil {
			logger.Error("Error stopping proxy backend: %v", err)
		}
	}()

	metrics.InitMetrics(config.CLIConfig.Metrics.Instance)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.GetProxyStatusMetric())
	registry.MustRegister(metrics.GetProxyLatencyMetric())

	proxyChecker := checker.NewProxyChecker(
		*proxyConfigs,
		config.CLIConfig.GetStartPort(),
		config.CLIConfig.Proxy.IpCheckUrl,
		config.CLIConfig.Proxy.Timeout,
		config.CLIConfig.Proxy.StatusCheckUrl,
		config.CLIConfig.Proxy.DownloadUrl,
		config.CLIConfig.Proxy.DownloadTimeout,
		config.CLIConfig.Proxy.DownloadMinSize,
		config.CLIConfig.Proxy.CheckMethod,
	)

	runCheckIteration := func() {
		logger.Info("Starting proxy check iteration")
		proxyChecker.CheckAllProxies()

		if config.CLIConfig.Metrics.PushURL != "" {
			pushConfig, err := metrics.ParseURL(config.CLIConfig.Metrics.PushURL)
			if err != nil {
				logger.Error("Error parsing push URL: %v", err)
				return
			}

			if pushConfig != nil {
				if err := metrics.PushMetrics(pushConfig, registry); err != nil {
					logger.Error("Error pushing metrics: %v", err)
				}
			}
		}
	}

	if config.CLIConfig.RunOnce {
		runCheckIteration()
		logger.Info("Check completed")
		return
	}

	checkScheduler := gocron.NewScheduler(time.UTC)
	checkScheduler.Every(config.CLIConfig.Proxy.CheckInterval).Seconds().Do(func() {
		runCheckIteration()
	})
	checkScheduler.StartAsync()

	if config.CLIConfig.Subscription.Update {
		updateScheduler := gocron.NewScheduler(time.UTC)
		updateScheduler.Every(config.CLIConfig.Subscription.UpdateInterval).Seconds().WaitForSchedule().Do(func() {
			logger.Info("Checking subscriptions for updates...")

			var newConfigs []*models.ProxyConfig

			// Load from subscription URLs
			if len(config.CLIConfig.Subscription.URLs) > 0 {
				subConfigs, err := subscription.ReadFromMultipleSources(config.CLIConfig.Subscription.URLs)
				if err != nil {
					logger.Error("Error fetching subscriptions: %v", err)
				} else {
					newConfigs = append(newConfigs, subConfigs...)
				}
			}

			// Load WireGuard configs
			if len(config.CLIConfig.WireGuard.Configs) > 0 {
				wgConfigs, err := subscription.ParseWireGuardConfigs(config.CLIConfig.WireGuard.Configs)
				if err != nil {
					logger.Error("Error loading WireGuard configs: %v", err)
				} else {
					for _, cfg := range wgConfigs {
						cfg.SubName = "wireguard"
					}
					newConfigs = append(newConfigs, wgConfigs...)
				}
			}

			if len(newConfigs) == 0 {
				logger.Error("No proxy configurations loaded during update")
				return
			}

			if config.CLIConfig.Proxy.ResolveDomains {
				resolved, err := subscription.ResolveDomainsForConfigs(newConfigs)
				if err != nil {
					logger.Error("Error resolving domains: %v", err)
				} else {
					newConfigs = resolved
				}
			}

			configsEqual := false
			if config.CLIConfig.Backend == "singbox" {
				configsEqual = singbox.IsConfigsEqual(*proxyConfigs, newConfigs)
			} else {
				configsEqual = xray.IsConfigsEqual(*proxyConfigs, newConfigs)
			}
			if !configsEqual {
				if err := updateConfiguration(newConfigs, proxyConfigs, proxyRunner, proxyChecker, configFile); err != nil {
					logger.Error("Error updating configuration: %v", err)
				}
			} else {
				logger.Info("Subscriptions checked, no changes")
			}
		})
		updateScheduler.StartAsync()
	}

	mux, err := web.NewPrefixServeMux(config.CLIConfig.Metrics.BasePath)
	if err != nil {
		logger.Fatal("Error creating web server: %v", err)
	}
	mux.Handle("/health", web.HealthHandler())
	mux.Handle("/static/", web.StaticHandler())
	mux.Handle("/api/v1/public/proxies", web.APIPublicProxiesHandler(proxyChecker))

	web.RegisterConfigEndpoints(*proxyConfigs, proxyChecker, config.CLIConfig.GetStartPort())

	protectedHandler := http.NewServeMux()
	protectedHandler.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	protectedHandler.Handle("/config/", web.ConfigStatusHandler(proxyChecker))
	protectedHandler.Handle("/api/v1/proxies/", web.APIProxyHandler(proxyChecker, config.CLIConfig.GetStartPort()))
	protectedHandler.Handle("/api/v1/proxies", web.APIProxiesHandler(proxyChecker, config.CLIConfig.GetStartPort()))
	protectedHandler.Handle("/api/v1/config", web.APIConfigHandler(proxyChecker))
	protectedHandler.Handle("/api/v1/status", web.APIStatusHandler(proxyChecker))
	protectedHandler.Handle("/api/v1/system/info", web.APISystemInfoHandler(version, startTime))
	protectedHandler.Handle("/api/v1/system/ip", web.APISystemIPHandler(proxyChecker))
	protectedHandler.Handle("/api/v1/docs", web.APIDocsHandler())
	protectedHandler.Handle("/api/v1/openapi.yaml", web.APIOpenAPIHandler())

	if config.CLIConfig.Web.Public {
		mux.Handle("/", web.IndexHandler(version, proxyChecker))
		mux.Handle("/config/", web.ConfigStatusHandler(proxyChecker))
		middlewareHandler := web.BasicAuthMiddleware(
			config.CLIConfig.Metrics.Username,
			config.CLIConfig.Metrics.Password,
		)(protectedHandler)
		mux.Handle("/metrics", middlewareHandler)
		mux.Handle("/api/", middlewareHandler)
	} else if config.CLIConfig.Metrics.Protected {
		protectedHandler.Handle("/", web.IndexHandler(version, proxyChecker))
		middlewareHandler := web.BasicAuthMiddleware(
			config.CLIConfig.Metrics.Username,
			config.CLIConfig.Metrics.Password,
		)(protectedHandler)
		mux.Handle("/", middlewareHandler)
	} else {
		protectedHandler.Handle("/", web.IndexHandler(version, proxyChecker))
		mux.Handle("/", protectedHandler)
	}

	if !config.CLIConfig.RunOnce {
		logger.Info("Server listening on %s:%s%s",
			config.CLIConfig.Metrics.Host,
			config.CLIConfig.Metrics.Port,
			config.CLIConfig.Metrics.BasePath,
		)
		if err := http.ListenAndServe(config.CLIConfig.Metrics.Host+":"+config.CLIConfig.Metrics.Port, mux); err != nil {
			logger.Fatal("Error starting server: %v", err)
		}
	}
}

func updateConfiguration(newConfigs []*models.ProxyConfig, currentConfigs *[]*models.ProxyConfig,
	proxyRunner runner.Runner, proxyChecker *checker.ProxyChecker, configFile string) error {

	logger.Info("Subscription changed, updating configuration...")

	if config.CLIConfig.Backend == "singbox" {
		singbox.PrepareProxyConfigs(newConfigs)
		configGenerator := singbox.NewConfigGenerator()
		if err := configGenerator.GenerateAndSaveConfig(
			newConfigs,
			config.CLIConfig.GetStartPort(),
			configFile,
			config.CLIConfig.GetLogLevel(),
		); err != nil {
			return err
		}
	} else {
		xray.PrepareProxyConfigs(newConfigs)
		configGenerator := xray.NewConfigGenerator()
		if err := configGenerator.GenerateAndSaveConfig(
			newConfigs,
			config.CLIConfig.GetStartPort(),
			configFile,
			config.CLIConfig.GetLogLevel(),
		); err != nil {
			return err
		}
	}

	if err := proxyRunner.Stop(); err != nil {
		return err
	}

	if err := proxyRunner.Start(); err != nil {
		return err
	}

	proxyChecker.UpdateProxies(newConfigs)

	*currentConfigs = newConfigs

	web.RegisterConfigEndpoints(newConfigs, proxyChecker, config.CLIConfig.GetStartPort())

	logger.Info("Configuration updated: %d proxies", len(newConfigs))
	return nil
}
