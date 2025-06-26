package subscription

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"xray-checker/config"
	"xray-checker/models"
	"xray-checker/parser"
	singbox "xray-checker/singbox"
	"xray-checker/utils"
)

func InitializeConfiguration(configFile string) (*[]*models.ProxyConfig, error) {
	configs, err := ReadFromSource(config.CLIConfig.Subscription.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing subscription: %v", err)
	}
	proxyConfigs := &configs

	singbox.PrepareProxyConfigs(*proxyConfigs)
	if err := singbox.GenerateAndSaveConfig(*proxyConfigs, config.CLIConfig.Singbox.StartPort, configFile, config.CLIConfig.Singbox.LogLevel); err != nil {
		return nil, fmt.Errorf("error generating Singbox config: %v", err)
	}

	return proxyConfigs, nil
}

func DetectSourceType(input string) models.SourceType {
	if strings.HasPrefix(input, "file://") {
		return models.SourceTypeFile
	}
	if strings.HasPrefix(input, "folder://") {
		return models.SourceTypeFolder
	}
	if strings.Contains(input, "://") {
		return models.SourceTypeURL
	}
	return models.SourceTypeBase64
}

func ReadFromSource(source string) ([]*models.ProxyConfig, error) {
	sourceType := DetectSourceType(source)

	switch sourceType {
	case models.SourceTypeURL:
		return readFromURL(source)
	case models.SourceTypeBase64:
		return readFromBase64(source)
	case models.SourceTypeFile:
		return readFromFile(strings.TrimPrefix(source, "file://"))
	case models.SourceTypeFolder:
		return readFromFolder(strings.TrimPrefix(source, "folder://"))
	default:
		return nil, fmt.Errorf("unknown source type")
	}
}

func readFromURL(url string) ([]*models.ProxyConfig, error) {
	return ParseSubscription(url)
}

func readFromBase64(encodedData string) ([]*models.ProxyConfig, error) {
	decoded, err := utils.AutoDecode(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}

	links := strings.Split(string(decoded), "\n")
	return parseProxyLinks(links)
}

func readFromFile(filepath string) ([]*models.ProxyConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	return parseXrayConfig(file)
}

func readFromFolder(folderPath string) ([]*models.ProxyConfig, error) {
	var allConfigs []*models.ProxyConfig

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			configs, err := readFromFile(path)
			if err != nil {
				log.Printf("Warning: error parsing file %s: %v", path, err)
				return nil
			}

			allConfigs = append(allConfigs, configs...)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk folder: %v", err)
	}

	if len(allConfigs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found in folder")
	}

	return allConfigs, nil
}

func parseProxyLinks(links []string) ([]*models.ProxyConfig, error) {
	var configs []*models.ProxyConfig

	for _, link := range links {
		link = strings.TrimSpace(link)
		if link == "" || link == "False" {
			continue
		}

		config, err := parser.ParseProxyURL(link)
		if err != nil {
			if strings.Contains(err.Error(), "skipping port:") {
				if u, parseErr := url.Parse(link); parseErr == nil {
					protocol := u.Scheme
					name := u.Fragment
					if name != "" {
						name = ", name: " + name
					}
					log.Printf("Skipped %s config with info ports(0,1)%s", protocol, name)
				}
				continue
			}

			if !isCommonInvalidString(link) {
				log.Printf("Warning: error parsing proxy URL: %v", err)
			}
			continue
		}

		configs = append(configs, config)
	}

	if len(configs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found")
	}

	return configs, nil
}

func isCommonInvalidString(s string) bool {
	invalidStrings := []string{
		"False",
		"True",
		"null",
		"undefined",
		"{",
		"}",
		"[",
		"]",
	}

	s = strings.TrimSpace(strings.ToLower(s))
	for _, invalid := range invalidStrings {
		if s == strings.ToLower(invalid) {
			return true
		}
	}

	specials := `"',.;:!@#$%^&*()_+-={}[]<>?/\|`
	onlySpecials := true
	for _, char := range s {
		if !strings.ContainsRune(specials, char) {
			onlySpecials = false
			break
		}
	}

	return onlySpecials
}

func parseXrayConfig(reader io.Reader) ([]*models.ProxyConfig, error) {
	var xrayConfigs []models.XrayConfig
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&xrayConfigs); err != nil {
		if seeker, ok := reader.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		} else {
			return nil, fmt.Errorf("failed to decode JSON array: %v", err)
		}

		var singleConfig models.XrayConfig
		if err := json.NewDecoder(reader).Decode(&singleConfig); err != nil {
			return nil, fmt.Errorf("failed to decode JSON: %v", err)
		}
		xrayConfigs = []models.XrayConfig{singleConfig}
	}

	var allConfigs []*models.ProxyConfig

	for _, xrayConfig := range xrayConfigs {
		for _, outbound := range xrayConfig.Outbounds {
			if outbound.Tag == "direct" || outbound.Tag == "block" || outbound.Tag == "dns-out" {
				continue
			}

			config := &models.ProxyConfig{
				Protocol: outbound.Protocol,
				Name:     outbound.Tag,
			}

			if err := parseOutboundSettings(config, outbound); err != nil {
				return nil, fmt.Errorf("failed to parse outbound %s: %v", outbound.Tag, err)
			}

			if outbound.StreamSettings != nil {
				parseStreamSettings(config, outbound.StreamSettings)
			}

			if err := config.Validate(); err != nil {
				log.Printf("Warning: skipping invalid config %s: %v", config.Name, err)
				continue
			}

			allConfigs = append(allConfigs, config)
		}
	}

	if len(allConfigs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found")
	}

	return allConfigs, nil
}

func parseOutboundSettings(config *models.ProxyConfig, outbound models.XrayOutbound) error {
	var settings map[string]interface{}
	if err := json.Unmarshal(outbound.Settings, &settings); err != nil {
		return fmt.Errorf("failed to parse outbound settings: %v", err)
	}

	switch config.Protocol {
	case "vmess", "vless":
		if vnext, ok := settings["vnext"].([]interface{}); ok && len(vnext) > 0 {
			if server, ok := vnext[0].(map[string]interface{}); ok {
				config.Server = server["address"].(string)
				config.Port = int(server["port"].(float64))

				if users, ok := server["users"].([]interface{}); ok && len(users) > 0 {
					if user, ok := users[0].(map[string]interface{}); ok {
						config.UUID = user["id"].(string)
						if aid, ok := user["alterId"].(float64); ok {
							config.AlterId = int(aid)
						}
						if flow, ok := user["flow"].(string); ok {
							config.Flow = flow
						}
					}
				}
			}
		}
	case "trojan":
		if servers, ok := settings["servers"].([]interface{}); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]interface{}); ok {
				config.Server = server["address"].(string)
				config.Port = int(server["port"].(float64))
				config.Password = server["password"].(string)
				if flow, ok := server["flow"].(string); ok {
					config.Flow = flow
				}
			}
		}
	case "shadowsocks":
		if servers, ok := settings["servers"].([]interface{}); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]interface{}); ok {
				config.Server = server["address"].(string)
				config.Port = int(server["port"].(float64))
				config.Password = server["password"].(string)
				config.Method = server["method"].(string)
			}
		}
	}

	return nil
}

func parseStreamSettings(config *models.ProxyConfig, settings *models.StreamSettings) {
	config.Type = settings.Network
	config.Security = settings.Security

	if settings.TLSSettings != nil {
		config.AllowInsecure = settings.TLSSettings.AllowInsecure
	}

	if settings.RealitySettings != nil {
		config.SNI = settings.RealitySettings.ServerName
		config.Fingerprint = settings.RealitySettings.Fingerprint
		config.PublicKey = settings.RealitySettings.PublicKey
		config.ShortID = settings.RealitySettings.ShortID
	}

	if settings.WSSettings != nil {
		config.Path = settings.WSSettings.Path
		if host, ok := settings.WSSettings.Headers["Host"]; ok {
			config.Host = host
		}
	}

	if settings.HTTPUpgradeSettings != nil {
		config.Path = settings.HTTPUpgradeSettings.Path
		if settings.HTTPUpgradeSettings.Headers != nil {
			if host, ok := settings.HTTPUpgradeSettings.Headers["Host"]; ok {
				config.Host = host
			}
		}
	}
}

func ParseSubscription(source string) ([]*models.ProxyConfig, error) {
	sourceType := DetectSourceType(source)

	switch sourceType {
	case models.SourceTypeURL:
		links, err := ParseSubscriptionURL(source)
		if err != nil {
			return nil, fmt.Errorf("error parsing subscription URL: %v", err)
		}
		return parseProxyLinks(links)

	case models.SourceTypeFile:
		return readFromFile(strings.TrimPrefix(source, "file://"))

	case models.SourceTypeFolder:
		return readFromFolder(strings.TrimPrefix(source, "folder://"))

	case models.SourceTypeBase64:
		return readFromBase64(source)

	default:
		return nil, fmt.Errorf("unknown source type")
	}
}

func ParseSubscriptionURL(subscriptionURL string) ([]string, error) {
	parsedURL, err := url.Parse(subscriptionURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported protocol scheme for subscription URL: %s", parsedURL.Scheme)
	}

	req, err := http.NewRequest("GET", subscriptionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("User-Agent", "Xray-Checker")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{
		Timeout: time.Second * 30, // Добавляем таймаут
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting subscription: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	decoded, err := utils.AutoDecode(string(body))
	if err != nil {
		return filterEmptyLinks(strings.Split(string(body), "\n")), nil
	}

	return filterEmptyLinks(strings.Split(string(decoded), "\n")), nil
}

func filterEmptyLinks(links []string) []string {
	var filtered []string
	for _, link := range links {
		if link = strings.TrimSpace(link); link != "" && !isCommonInvalidString(link) {
			filtered = append(filtered, link)
		}
	}
	return filtered
}
