package subscription

import (
	"fmt"
	"net"
	"sync"
	"xray-checker/config"
	"xray-checker/logger"
	"xray-checker/models"
	"xray-checker/xray"
)

type subscriptionResult struct {
	URL     string
	Configs []*models.ProxyConfig
	Error   error
}

func InitializeConfiguration(configFile string, version string) (*[]*models.ProxyConfig, error) {
	configs, err := ReadFromMultipleSources(config.CLIConfig.Subscription.URLs)
	if err != nil {
		return nil, err
	}

	proxyConfigs := configs

	if config.CLIConfig.Proxy.ResolveDomains {
		proxyConfigs, err = ResolveDomainsForConfigs(configs)
		if err != nil {
			return nil, err
		}
	}

	xray.PrepareProxyConfigs(proxyConfigs)

	configGenerator := xray.NewConfigGenerator()
	if err := configGenerator.GenerateAndSaveConfig(
		proxyConfigs,
		config.CLIConfig.Xray.StartPort,
		configFile,
		config.CLIConfig.Xray.LogLevel,
	); err != nil {
		return nil, err
	}

	return &proxyConfigs, nil
}

func ReadFromMultipleSources(urls []string) ([]*models.ProxyConfig, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no subscription URLs provided")
	}

	if len(urls) == 1 {
		return ReadFromSource(urls[0])
	}

	logger.Debug("Fetching %d subscriptions in parallel", len(urls))

	results := make(chan subscriptionResult, len(urls))

	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			configs, err := ReadFromSource(u)
			results <- subscriptionResult{
				URL:     u,
				Configs: configs,
				Error:   err,
			}
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allConfigs []*models.ProxyConfig
	var errors []error
	successCount := 0

	for result := range results {
		if result.Error != nil {
			logger.Warn("Failed to fetch subscription %s: %v", result.URL, result.Error)
			errors = append(errors, fmt.Errorf("%s: %v", result.URL, result.Error))
			continue
		}
		logger.Debug("Fetched %d proxies from %s", len(result.Configs), result.URL)
		allConfigs = append(allConfigs, result.Configs...)
		successCount++
	}

	if successCount == 0 {
		return nil, fmt.Errorf("failed to fetch any subscription: %v", errors)
	}

	for i := range allConfigs {
		allConfigs[i].Index = i
	}

	logger.Debug("Total: %d proxies from %d/%d subscriptions", len(allConfigs), successCount, len(urls))
	return allConfigs, nil
}

func ReadFromSource(source string) ([]*models.ProxyConfig, error) {
	parser := NewParser()
	return parser.Parse(source)
}

func ResolveDomainsForConfigs(configs []*models.ProxyConfig) ([]*models.ProxyConfig, error) {
	var out []*models.ProxyConfig
	for _, cfg := range configs {
		if ip := net.ParseIP(cfg.Server); ip != nil {
			out = append(out, cfg)
			continue
		}

		ips, err := net.LookupIP(cfg.Server)
		if err != nil || len(ips) == 0 {
			logger.Warn("Failed to resolve domain %s: %v", cfg.Server, err)
			out = append(out, cfg)
			continue
		}

		for i, ip := range ips {
			clone := *cfg
			clone.Server = ip.String()
			clone.StableID = ""
			if len(ips) > 1 {
				clone.Name = fmt.Sprintf("%s #%d", cfg.Name, i+1)
			}
			out = append(out, &clone)
		}
	}
	return out, nil
}
