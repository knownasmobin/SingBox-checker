package subscription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/knownasmobin/singbox-checker/config"
	"github.com/knownasmobin/singbox-checker/models"
	"github.com/knownasmobin/singbox-checker/parser"
	"github.com/knownasmobin/singbox-checker/utils"
)


func InitializeConfiguration(configFile string) (*[]*models.ProxyConfig, error) {
	configs, err := ReadFromSource(config.CLIConfig.Subscription.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing subscription: %v", err)
	}
	proxyConfigs := &configs

	// Initialize SingBox configuration
	if err := InitializeSingBoxConfig(*proxyConfigs, configFile); err != nil {
		return nil, fmt.Errorf("error generating SingBox config: %v", err)
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
	// First, read the entire file content
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// First, try to parse as a SingBox config file
	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err == nil {
		// If it's a valid JSON, try to parse as a SingBox config
		if _, ok := config["outbounds"]; ok {
			return parseSingBoxConfig(bytes.NewReader(content))
		}
	}

	// If not a SingBox config, try to parse as a list of proxy URLs
	links := strings.Split(strings.TrimSpace(string(content)), "\n")
	return parseProxyLinks(links)
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
				fmt.Printf("Warning: error parsing file %s: %v\n", path, err)
				return nil
			}

			allConfigs = append(allConfigs, configs...)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk folder: %w", err)
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
					fmt.Printf("Skipped %s config with info ports(0,1)%s\n", protocol, name)
				}
				continue
			}

			if !isCommonInvalidString(link) {
				fmt.Printf("Warning: error parsing proxy URL: %v\n", err)
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

// parseSingBoxConfig parses a SingBox configuration from the given reader.
func parseSingBoxConfig(reader io.Reader) ([]*models.ProxyConfig, error) {
	var config map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SingBox config: %w", err)
	}

	outbounds, ok := config["outbounds"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid SingBox config: missing outbounds")
	}

	var proxyConfigs []*models.ProxyConfig

	for _, outbound := range outbounds {
		out, ok := outbound.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip non-proxy outbounds
		tag, _ := out["tag"].(string)
		if tag == "" || tag == "direct" || tag == "block" || tag == "dns-out" {
			continue
		}

		// Create a new proxy config
		proxy := &models.ProxyConfig{
			Name:     tag,
			Protocol: out["type"].(string),
		}

		// Parse server and port
		if server, ok := out["server"].(string); ok {
			proxy.Server = server
		}

		if port, ok := out["server_port"].(float64); ok {
			proxy.Port = int(port)
		}

		// Parse protocol-specific settings
		switch proxy.Protocol {
		case "vmess", "vless":
			if settings, ok := out["settings"].(map[string]interface{}); ok {
				if v, ok := settings["uuid"].(string); ok {
					proxy.UUID = v
				}
			}

		case "trojan":
			if settings, ok := out["settings"].(map[string]interface{}); ok {
				if v, ok := settings["password"].(string); ok {
					proxy.Password = v
				}
			}

		case "shadowsocks":
			if settings, ok := out["settings"].(map[string]interface{}); ok {
				if v, ok := settings["password"].(string); ok {
					proxy.Password = v
				}
				if v, ok := settings["method"].(string); ok {
					proxy.Method = v
				}
			}
		}

		// Parse transport settings
		if transport, ok := out["transport"].(map[string]interface{}); ok {
			if v, ok := transport["type"].(string); ok {
				proxy.Type = v
			}

			if v, ok := transport["path"].(string); ok {
				proxy.Path = v
			}

			if headers, ok := transport["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					proxy.Host = host
				}
			}

			if v, ok := transport["service_name"].(string); ok {
				proxy.ServiceName = v
			}
		}

		// Parse TLS settings
		if tls, ok := out["tls"].(map[string]interface{}); ok {
			if v, ok := tls["enabled"].(bool); ok && v {
				proxy.Security = "tls"
			}

			if v, ok := tls["server_name"].(string); ok {
				proxy.SNI = v
			}

			if v, ok := tls["insecure"].(bool); ok {
				proxy.AllowInsecure = v
			}

			if v, ok := tls["alpn"].([]interface{}); ok && len(v) > 0 {
				alpn := make([]string, len(v))
				for i, a := range v {
					if s, ok := a.(string); ok {
						alpn[i] = s
					}
				}
				proxy.ALPN = alpn
			}
		}

		// Parse reality settings
		if reality, ok := out["reality"].(map[string]interface{}); ok {
			proxy.Security = "reality"

			if v, ok := reality["server_name"].(string); ok {
				proxy.SNI = v
			}

			if v, ok := reality["public_key"].(string); ok {
				proxy.PublicKey = v
			}

			if v, ok := reality["short_id"].(string); ok {
				proxy.ShortID = v
			}

			if v, ok := reality["fingerprint"].(string); ok {
				proxy.Fingerprint = v
			}
		}

		// Skip invalid configs
		if err := proxy.Validate(); err != nil {
			fmt.Printf("Warning: skipping invalid config %s: %v\n", proxy.Name, err)
			continue
		}

		proxyConfigs = append(proxyConfigs, proxy)
	}

	if len(proxyConfigs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found in SingBox config")
	}

	return proxyConfigs, nil
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

	req.Header.Set("User-Agent", "SingBox-Checker")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{
		Timeout: time.Second * 30, // Add timeout
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
