package subscription

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knownasmobin/singbox-checker/config"
	"github.com/knownasmobin/singbox-checker/models"
	"github.com/knownasmobin/singbox-checker/runner"
)

// InitializeSingBoxConfig initializes the SingBox configuration
func InitializeSingBoxConfig(proxyConfigs []*models.ProxyConfig, configFile string) error {
	// Create a new default SingBox config
	singboxConfig := models.NewDefaultSingBoxConfig()

	// Convert proxy configs to SingBox outbounds
	outbounds, err := ConvertToSingBoxOutbounds(proxyConfigs, config.CLIConfig.SingBox.StartPort)
	if err != nil {
		return fmt.Errorf("failed to convert to SingBox outbounds: %v", err)
	}
	
	// Add the outbounds to the config
	singboxConfig.Outbounds = outbounds

	// Create the config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Write the config to a file
	configData, err := json.MarshalIndent(singboxConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SingBox config: %v", err)
	}

	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		return fmt.Errorf("failed to write SingBox config: %v", err)
	}

	return nil
}

// ConvertToSingBoxOutbounds converts proxy configs to SingBox outbound configurations
func ConvertToSingBoxOutbounds(proxyConfigs []*models.ProxyConfig, startPort int) ([]models.Outbound, error) {
	var outbounds []models.Outbound

	for i, proxy := range proxyConfigs {
		outbound, err := convertProxyToOutbound(proxy, startPort+i)
		if err != nil {
			return nil, fmt.Errorf("error converting proxy to outbound: %v", err)
		}
		outbounds = append(outbounds, *outbound)
	}

	// Add a direct outbound for routing
	directOutbound := models.Outbound{
		Type: "direct",
		Tag:  "direct",
	}
	outbounds = append(outbounds, directOutbound)

	return outbounds, nil
}

// convertProxyToOutbound converts a single proxy config to a SingBox outbound
func convertProxyToOutbound(proxy *models.ProxyConfig, port int) (*models.Outbound, error) {
	outbound := &models.Outbound{
		Type:       proxy.Protocol,
		Tag:        fmt.Sprintf("%s-%d", proxy.Protocol, port),
		Server:     proxy.Server,
		ServerPort: proxy.Port,
	}

	switch proxy.Protocol {
	case "vmess":
		outbound.UUID = proxy.UUID
		outbound.Security = "auto"
		outbound.Transport = &models.TransportConfig{
			Type: "ws",
		}

		if proxy.Path != "" {
			outbound.Transport.Path = proxy.Path
		}

		if proxy.Host != "" {
			outbound.Transport.Headers = map[string]string{
				"Host": proxy.Host,
			}
		}

	case "vless":
		outbound.UUID = proxy.UUID
		outbound.Flow = proxy.Flow

		transport := &models.TransportConfig{
			Type: "ws",
		}

		if proxy.Path != "" {
			transport.Path = proxy.Path
		}

		if proxy.Host != "" {
			transport.Headers = map[string]string{
				"Host": proxy.Host,
			}
		}

		outbound.Transport = transport

		// Enable TLS if SNI is provided
		if proxy.SNI != "" {
			outbound.TLS = &models.TLSConfig{
				Enabled:     true,
				ServerName:  proxy.SNI,
				Insecure:    proxy.AllowInsecure,
				ALPN:        proxy.ALPN,
				Fingerprint: proxy.Fingerprint,
			}
		}

	case "trojan":
		outbound.Password = proxy.Password

		transport := &models.TransportConfig{
			Type: "ws",
		}

		if proxy.Path != "" {
			transport.Path = proxy.Path
		}

		if proxy.Host != "" {
			transport.Headers = map[string]string{
				"Host": proxy.Host,
			}
		}

		outbound.Transport = transport

		// Enable TLS by default for trojan
		outbound.TLS = &models.TLSConfig{
			Enabled:    true,
			ServerName: proxy.SNI,
			Insecure:   proxy.AllowInsecure,
		}

	case "shadowsocks":
		outbound.Password = proxy.Password
		outbound.Method = proxy.Method

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", proxy.Protocol)
	}

	return outbound, nil
}

// UpdateSingBoxConfig updates the SingBox configuration with new proxy configs
func UpdateSingBoxConfig(runner *runner.SingBoxRunner, proxyConfigs []*models.ProxyConfig) error {
	// Convert proxy configs to SingBox outbounds
	outbounds, err := ConvertToSingBoxOutbounds(proxyConfigs, config.CLIConfig.SingBox.StartPort)
	if err != nil {
		return fmt.Errorf("failed to convert to SingBox outbounds: %v", err)
	}

	// Get current config
	config, err := runner.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %v", err)
	}

	// Update outbounds
	config.Outbounds = outbounds

	// Update the config
	if err := runner.UpdateConfig(config); err != nil {
		return fmt.Errorf("failed to update SingBox config: %v", err)
	}

	return nil
}
