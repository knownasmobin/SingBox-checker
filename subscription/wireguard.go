package subscription

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"xray-checker/logger"
	"xray-checker/models"
)

type WireGuardConfig struct {
	// Interface section
	PrivateKey string
	Address    []string
	DNS        []string
	MTU        int

	// Peer section
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int
}

func ParseWireGuardConfig(filePath string) (*models.ProxyConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open WireGuard config: %v", err)
	}
	defer file.Close()

	wgConfig := &WireGuardConfig{}
	currentSection := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch currentSection {
		case "interface":
			switch strings.ToLower(key) {
			case "privatekey":
				wgConfig.PrivateKey = value
			case "address":
				wgConfig.Address = parseCommaSeparated(value)
			case "dns":
				wgConfig.DNS = parseCommaSeparated(value)
			case "mtu":
				if mtu, err := strconv.Atoi(value); err == nil {
					wgConfig.MTU = mtu
				}
			}
		case "peer":
			switch strings.ToLower(key) {
			case "publickey":
				wgConfig.PublicKey = value
			case "presharedkey":
				wgConfig.PresharedKey = value
			case "endpoint":
				wgConfig.Endpoint = value
			case "allowedips":
				wgConfig.AllowedIPs = parseCommaSeparated(value)
			case "persistentkeepalive":
				if keepalive, err := strconv.Atoi(value); err == nil {
					wgConfig.PersistentKeepalive = keepalive
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading WireGuard config: %v", err)
	}

	if wgConfig.PrivateKey == "" {
		return nil, fmt.Errorf("WireGuard config missing PrivateKey")
	}
	if wgConfig.PublicKey == "" {
		return nil, fmt.Errorf("WireGuard config missing Peer PublicKey")
	}
	if wgConfig.Endpoint == "" {
		return nil, fmt.Errorf("WireGuard config missing Peer Endpoint")
	}

	server, port, err := parseEndpoint(wgConfig.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %v", err)
	}

	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	proxyConfig := &models.ProxyConfig{
		Protocol:              "wireguard",
		Name:                  name,
		Server:                server,
		Port:                  port,
		WGPrivateKey:          wgConfig.PrivateKey,
		WGPublicKey:           wgConfig.PublicKey,
		WGPresharedKey:        wgConfig.PresharedKey,
		WGLocalAddress:        wgConfig.Address,
		WGDNS:                 wgConfig.DNS,
		WGMTU:                 wgConfig.MTU,
		WGPersistentKeepalive: wgConfig.PersistentKeepalive,
		WGAllowedIPs:          wgConfig.AllowedIPs,
	}

	return proxyConfig, nil
}

func ParseWireGuardConfigs(paths []string) ([]*models.ProxyConfig, error) {
	var configs []*models.ProxyConfig

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			logger.Warn("Failed to stat WireGuard config path %s: %v", path, err)
			continue
		}

		if info.IsDir() {
			dirConfigs, err := parseWireGuardDir(path)
			if err != nil {
				logger.Warn("Failed to parse WireGuard directory %s: %v", path, err)
				continue
			}
			configs = append(configs, dirConfigs...)
		} else {
			config, err := ParseWireGuardConfig(path)
			if err != nil {
				logger.Warn("Failed to parse WireGuard config %s: %v", path, err)
				continue
			}
			configs = append(configs, config)
		}
	}

	return configs, nil
}

func parseWireGuardDir(dirPath string) ([]*models.ProxyConfig, error) {
	var configs []*models.ProxyConfig

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".conf") {
			continue
		}

		configPath := filepath.Join(dirPath, name)
		config, err := ParseWireGuardConfig(configPath)
		if err != nil {
			logger.Warn("Failed to parse WireGuard config %s: %v", configPath, err)
			continue
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func parseCommaSeparated(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func parseEndpoint(endpoint string) (string, int, error) {
	lastColon := strings.LastIndex(endpoint, ":")
	if lastColon == -1 {
		return "", 0, fmt.Errorf("endpoint missing port: %s", endpoint)
	}

	server := endpoint[:lastColon]
	portStr := endpoint[lastColon+1:]

	if strings.HasPrefix(server, "[") && strings.HasSuffix(server, "]") {
		server = server[1 : len(server)-1]
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in endpoint: %s", endpoint)
	}

	return server, port, nil
}
