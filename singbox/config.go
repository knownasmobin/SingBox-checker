package singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"xray-checker/logger"
	"xray-checker/models"
)

type ConfigGenerator struct{}

func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

func (g *ConfigGenerator) GenerateConfig(proxies []*models.ProxyConfig, startPort int, logLevel string) ([]byte, error) {
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"level": g.mapLogLevel(logLevel),
		},
		"inbounds":  g.generateInbounds(proxies, startPort),
		"outbounds": g.generateOutbounds(proxies),
		"route":     g.generateRoute(proxies),
	}

	return json.MarshalIndent(config, "", "  ")
}

func (g *ConfigGenerator) mapLogLevel(level string) string {
	switch level {
	case "debug":
		return "debug"
	case "info":
		return "info"
	case "warning", "warn":
		return "warn"
	case "error":
		return "error"
	case "none":
		return "silent"
	default:
		return "warn"
	}
}

func (g *ConfigGenerator) GenerateAndSaveConfig(proxies []*models.ProxyConfig, startPort int, filename string, logLevel string) error {
	configBytes, err := g.GenerateConfig(proxies, startPort, logLevel)
	if err != nil {
		return fmt.Errorf("error generating config: %v", err)
	}

	if err := g.ValidateConfig(configBytes); err != nil {
		logger.Warn("Config validation failed: %v", err)
	}

	if err := os.WriteFile(filename, configBytes, 0644); err != nil {
		return fmt.Errorf("error saving config: %v", err)
	}

	return nil
}

func (g *ConfigGenerator) ValidateConfig(configBytes []byte) error {
	var config map[string]interface{}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	required := []string{"inbounds", "outbounds", "route"}
	for _, field := range required {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}

func (g *ConfigGenerator) generateInbounds(proxies []*models.ProxyConfig, startPort int) []map[string]interface{} {
	var inbounds []map[string]interface{}

	for _, proxy := range proxies {
		inbound := map[string]interface{}{
			"type":   "socks",
			"tag":    fmt.Sprintf("%s_%s_%d_Inbound", proxy.Name, proxy.Protocol, proxy.Index),
			"listen": "127.0.0.1",
			"listen_port": startPort + proxy.Index,
			"sniff":           true,
			"sniff_override_destination": false,
		}
		inbounds = append(inbounds, inbound)
	}

	return inbounds
}

func (g *ConfigGenerator) generateOutbounds(proxies []*models.ProxyConfig) []map[string]interface{} {
	var outbounds []map[string]interface{}

	outbounds = append(outbounds, map[string]interface{}{
		"type": "direct",
		"tag":  "direct",
	})

	outbounds = append(outbounds, map[string]interface{}{
		"type": "block",
		"tag":  "block",
	})

	for _, proxy := range proxies {
		outbound := g.generateProxyOutbound(proxy)
		outbounds = append(outbounds, outbound)
	}

	return outbounds
}

func (g *ConfigGenerator) generateProxyOutbound(proxy *models.ProxyConfig) map[string]interface{} {
	outbound := map[string]interface{}{
		"tag":  fmt.Sprintf("%s_%d", proxy.Name, proxy.Index),
		"type": proxy.Protocol,
	}

	// WireGuard has a different structure
	if proxy.Protocol == "wireguard" {
		outbound["private_key"] = proxy.WGPrivateKey
		outbound["peers"] = []map[string]interface{}{
			{
				"public_key": proxy.WGPublicKey,
				"server":     proxy.Server,
				"server_port": proxy.Port,
			},
		}
		if len(proxy.WGLocalAddress) > 0 {
			outbound["local_address"] = proxy.WGLocalAddress
		}
		if proxy.WGPresharedKey != "" {
			outbound["peers"].([]map[string]interface{})[0]["pre_shared_key"] = proxy.WGPresharedKey
		}
		if len(proxy.WGAllowedIPs) > 0 {
			outbound["peers"].([]map[string]interface{})[0]["allowed_ips"] = proxy.WGAllowedIPs
		}
		if proxy.WGPersistentKeepalive > 0 {
			outbound["peers"].([]map[string]interface{})[0]["persistent_keepalive_interval"] = fmt.Sprintf("%ds", proxy.WGPersistentKeepalive)
		}
		if proxy.WGMTU > 0 {
			outbound["mtu"] = proxy.WGMTU
		}
		if len(proxy.WGReserved) > 0 {
			outbound["reserved"] = proxy.WGReserved
		}
		return outbound
	}

	outbound["server"] = proxy.Server
	outbound["server_port"] = proxy.Port

	switch proxy.Protocol {
	case "vless":
		outbound["uuid"] = proxy.UUID
		if proxy.Flow != "" {
			outbound["flow"] = proxy.Flow
		}

	case "vmess":
		outbound["uuid"] = proxy.UUID
		outbound["alter_id"] = proxy.GetAlterId()
		security := proxy.GetVMessSecurity()
		if security != "" && security != "auto" {
			outbound["security"] = security
		}

	case "trojan":
		outbound["password"] = proxy.Password
		if proxy.Flow != "" {
			outbound["flow"] = proxy.Flow
		}

	case "shadowsocks":
		outbound["method"] = proxy.Method
		outbound["password"] = proxy.Password
	}

	if tls := g.generateTLS(proxy); tls != nil {
		outbound["tls"] = tls
	}

	if transport := g.generateTransport(proxy); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

func (g *ConfigGenerator) generateTLS(proxy *models.ProxyConfig) map[string]interface{} {
	security := proxy.Security
	if security == "" {
		security = "none"
	}

	if security == "none" {
		return nil
	}

	tls := map[string]interface{}{
		"enabled": true,
	}

	if security == "tls" {
		if proxy.SNI != "" {
			tls["server_name"] = proxy.SNI
		}
		if proxy.Fingerprint != "" {
			tls["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": proxy.Fingerprint,
			}
		}
		if len(proxy.ALPN) > 0 {
			tls["alpn"] = proxy.ALPN
		}
		if proxy.AllowInsecure {
			tls["insecure"] = true
		}
	}

	if security == "reality" {
		tls["reality"] = map[string]interface{}{
			"enabled":    true,
			"public_key": proxy.PublicKey,
		}
		if proxy.ShortID != "" {
			tls["reality"].(map[string]interface{})["short_id"] = proxy.ShortID
		}
		if proxy.SNI != "" {
			tls["server_name"] = proxy.SNI
		}
		if proxy.Fingerprint != "" {
			tls["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": proxy.Fingerprint,
			}
		}
	}

	return tls
}

func (g *ConfigGenerator) generateTransport(proxy *models.ProxyConfig) map[string]interface{} {
	network := proxy.Type
	if network == "" {
		network = "tcp"
	}

	if network == "tcp" {
		if proxy.HeaderType == "" || proxy.HeaderType == "none" {
			return nil
		}
		if proxy.HeaderType == "http" {
			transport := map[string]interface{}{
				"type":   "http",
				"method": "GET",
			}
			if proxy.Path != "" {
				transport["path"] = proxy.Path
			}
			if proxy.Host != "" {
				transport["host"] = []string{proxy.Host}
			}
			return transport
		}
		return nil
	}

	switch network {
	case "ws":
		transport := map[string]interface{}{
			"type": "ws",
		}
		if proxy.Path != "" {
			transport["path"] = proxy.Path
		}
		if proxy.Host != "" {
			transport["headers"] = map[string]interface{}{
				"Host": proxy.Host,
			}
		}
		return transport

	case "grpc":
		transport := map[string]interface{}{
			"type": "grpc",
		}
		if proxy.GetServiceName() != "" {
			transport["service_name"] = proxy.GetServiceName()
		}
		return transport

	case "http", "h2":
		transport := map[string]interface{}{
			"type": "http",
		}
		if proxy.Path != "" {
			transport["path"] = proxy.Path
		}
		if proxy.Host != "" {
			transport["host"] = strings.Split(proxy.Host, ",")
		}
		return transport

	case "httpupgrade":
		transport := map[string]interface{}{
			"type": "httpupgrade",
		}
		if proxy.Path != "" {
			transport["path"] = proxy.Path
		}
		if proxy.Host != "" {
			transport["host"] = proxy.Host
		}
		return transport

	case "quic":
		return map[string]interface{}{
			"type": "quic",
		}
	}

	return nil
}

func (g *ConfigGenerator) generateRoute(proxies []*models.ProxyConfig) map[string]interface{} {
	var rules []map[string]interface{}

	for _, proxy := range proxies {
		inboundTag := fmt.Sprintf("%s_%s_%d_Inbound", proxy.Name, proxy.Protocol, proxy.Index)
		outboundTag := fmt.Sprintf("%s_%d", proxy.Name, proxy.Index)

		rules = append(rules, map[string]interface{}{
			"inbound":  []string{inboundTag},
			"outbound": outboundTag,
		})
	}

	return map[string]interface{}{
		"rules":          rules,
		"final":          "direct",
		"auto_detect_interface": true,
	}
}
