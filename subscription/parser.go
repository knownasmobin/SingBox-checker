package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"xray-checker/logger"
	"xray-checker/models"

	libXray "github.com/xtls/libxray"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

type libXrayResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type libXrayOutbound struct {
	Protocol       string                 `json:"protocol"`
	SendThrough    string                 `json:"sendThrough"`
	Tag            string                 `json:"tag"`
	Settings       *libXraySettings       `json:"settings"`
	StreamSettings *libXrayStreamSettings `json:"streamSettings"`
}

type libXraySettings struct {
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Level      int    `json:"level"`
	ID         string `json:"id"`
	Flow       string `json:"flow"`
	Encryption string `json:"encryption"`
	AlterId    int    `json:"alterId"`
	Security   string `json:"security"`
	Password   string `json:"password"`
	Method     string `json:"method"`
}

type libXrayStreamSettings struct {
	Network             string                      `json:"network"`
	Security            string                      `json:"security"`
	TlsSettings         *libXrayTlsSettings         `json:"tlsSettings"`
	RealitySettings     *libXrayRealitySettings     `json:"realitySettings"`
	RawSettings         *libXrayRawSettings         `json:"rawSettings"`
	WsSettings          *libXrayWsSettings          `json:"wsSettings"`
	GrpcSettings        *libXrayGrpcSettings        `json:"grpcSettings"`
	HttpSettings        *libXrayHttpSettings        `json:"httpSettings"`
	HttpupgradeSettings *libXrayHttpupgradeSettings `json:"httpupgradeSettings"`
	XhttpSettings       *libXrayXhttpSettings       `json:"xhttpSettings"`
	SplithttpSettings   *libXraySplithttpSettings   `json:"splithttpSettings"`
}

type libXrayTlsSettings struct {
	ServerName    string   `json:"serverName"`
	AllowInsecure bool     `json:"allowInsecure"`
	Fingerprint   string   `json:"fingerprint"`
	Alpn          []string `json:"alpn"`
}

type libXrayRealitySettings struct {
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	ShortId     string `json:"shortId"`
}

type libXrayRawSettings struct {
	Header *struct {
		Type    string `json:"type"`
		Request *struct {
			Path    []string `json:"path"`
			Headers *struct {
				Host []string `json:"Host"`
			} `json:"headers"`
		} `json:"request"`
	} `json:"header"`
}

type libXrayWsSettings struct {
	Path    string `json:"path"`
	Headers *struct {
		Host string `json:"Host"`
	} `json:"headers"`
	Host string `json:"host"`
}

type libXrayGrpcSettings struct {
	ServiceName string `json:"serviceName"`
	MultiMode   bool   `json:"multiMode"`
}

type libXrayHttpSettings struct {
	Path string   `json:"path"`
	Host []string `json:"host"`
}

type libXrayHttpupgradeSettings struct {
	Path string `json:"path"`
	Host string `json:"host"`
}

type libXrayXhttpSettings struct {
	Path string `json:"path"`
	Host string `json:"host"`
}

type libXraySplithttpSettings struct {
	Path string `json:"path"`
	Host string `json:"host"`
}

type originalLinkData struct {
	Name          string
	Encryption    string
	Type          string
	Path          string
	Host          string
	AllowInsecure bool
}

type parsedLink struct {
	Server        string
	Port          int
	Name          string
	Encryption    string
	Type          string
	Path          string
	Host          string
	AllowInsecure bool
}

type xrayStandardSettings struct {
	Vnext []struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
		Users   []struct {
			ID         string `json:"id"`
			Flow       string `json:"flow"`
			Encryption string `json:"encryption"`
			AlterId    int    `json:"alterId"`
			Security   string `json:"security"`
			Level      int    `json:"level"`
		} `json:"users"`
	} `json:"vnext"`
	Servers []struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Password string `json:"password"`
		Method   string `json:"method"`
		Flow     string `json:"flow"`
	} `json:"servers"`
}

func (p *Parser) Parse(subscriptionData string) ([]*models.ProxyConfig, error) {
	sourceType := p.detectSourceType(subscriptionData)
	logger.Debug("Detected source type: %s", sourceType)

	var rawData []byte
	var err error

	switch sourceType {
	case "url":
		rawData, err = p.fetchURLContent(subscriptionData)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL content: %v", err)
		}
	case "folder":
		folderPath := strings.TrimPrefix(subscriptionData, "folder://")
		return p.parseFolder(folderPath)
	case "file":
		filePath := strings.TrimPrefix(subscriptionData, "file://")
		rawData, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
	case "base64":
		rawData = []byte(strings.TrimPrefix(subscriptionData, "base64://"))
		rawData = []byte(strings.TrimPrefix(string(rawData), "data:text/plain;base64,"))
	default:
		rawData = []byte(subscriptionData)
	}

	trimmedData := strings.TrimSpace(string(rawData))
	if strings.HasPrefix(trimmedData, "[") {
		logger.Debug("Detected JSON array format")
		return p.parseJSONConfigs(rawData)
	}

	originalData := p.parseOriginalLinks(rawData)

	cleanedData := p.cleanEmptyLines(rawData)

	base64Data := base64.StdEncoding.EncodeToString(cleanedData)

	resultBase64 := libXray.ConvertShareLinksToXrayJson(base64Data)

	resultBytes, err := base64.StdEncoding.DecodeString(resultBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode libXray response: %v", err)
	}

	var response libXrayResponse
	if err := json.Unmarshal(resultBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse libXray response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("libXray parsing failed")
	}

	var xrayConfig struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(response.Data, &xrayConfig); err != nil {
		return nil, fmt.Errorf("failed to parse libXray config data: %v", err)
	}

	logger.Debug("Parsed %d outbounds", len(xrayConfig.Outbounds))

	var proxyConfigs []*models.ProxyConfig
	configIndex := 0
	for _, outboundRaw := range xrayConfig.Outbounds {
		proxyConfig, err := p.convertOutbound(outboundRaw, configIndex, originalData)
		if err != nil {
			logger.Debug("Skipping outbound: %v", err)
			continue
		}
		if proxyConfig != nil {
			proxyConfigs = append(proxyConfigs, proxyConfig)
			configIndex++
		}
	}

	if len(proxyConfigs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found")
	}

	return proxyConfigs, nil
}

func (p *Parser) parseJSONConfigs(data []byte) ([]*models.ProxyConfig, error) {
	var configs []struct {
		Remarks   string            `json:"remarks"`
		Outbounds []json.RawMessage `json:"outbounds"`
	}

	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON configs: %v", err)
	}

	logger.Debug("Parsed %d JSON configs", len(configs))

	var proxyConfigs []*models.ProxyConfig
	configIndex := 0

	for _, config := range configs {
		for _, outboundRaw := range config.Outbounds {
			proxyConfig, err := p.convertOutbound(outboundRaw, configIndex, nil)
			if err != nil {
				continue
			}
			if proxyConfig != nil {
				if config.Remarks != "" {
					proxyConfig.Name = config.Remarks
				}
				proxyConfigs = append(proxyConfigs, proxyConfig)
				configIndex++
			}
		}
	}

	if len(proxyConfigs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found in JSON")
	}

	return proxyConfigs, nil
}

func (p *Parser) cleanEmptyLines(data []byte) []byte {
	decoded := p.tryDecodeBase64(data)

	lines := strings.Split(string(decoded), "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return []byte(strings.Join(cleanLines, "\n"))
}

func (p *Parser) detectSourceType(source string) string {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return "url"
	}
	if strings.HasPrefix(source, "folder://") {
		return "folder"
	}
	if strings.HasPrefix(source, "file://") {
		return "file"
	}
	if strings.HasPrefix(source, "base64://") || strings.HasPrefix(source, "data:text/plain;base64,") {
		return "base64"
	}
	return "raw"
}

func (p *Parser) fetchURLContent(source string) ([]byte, error) {
	req, err := http.NewRequest("GET", source, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Xray-Checker")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (p *Parser) parseOriginalLinks(rawData []byte) map[string]*originalLinkData {
	result := make(map[string]*originalLinkData)

	decoded := p.tryDecodeBase64(rawData)

	lines := strings.Split(string(decoded), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		data := p.parseShareLink(line)
		if data != nil {
			key := fmt.Sprintf("%s:%d", data.Server, data.Port)
			result[key] = &originalLinkData{
				Name:          data.Name,
				Encryption:    data.Encryption,
				Type:          data.Type,
				Path:          data.Path,
				Host:          data.Host,
				AllowInsecure: data.AllowInsecure,
			}
		}
	}

	return result
}

func (p *Parser) parseShareLink(link string) *parsedLink {
	if strings.HasPrefix(link, "vmess://") {
		return p.parseVMessLink(link)
	}

	u, err := url.Parse(link)
	if err != nil {
		return nil
	}

	result := &parsedLink{
		Name: u.Fragment,
	}

	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		return nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port == 0 {
		return nil
	}
	result.Server = host
	result.Port = port

	query := u.Query()
	result.Type = query.Get("type")
	result.Encryption = query.Get("encryption")
	result.Path = query.Get("path")
	result.Host = query.Get("host")
	result.AllowInsecure = query.Get("allowInsecure") == "1" || query.Get("allowInsecure") == "true"

	return result
}

func (p *Parser) parseVMessLink(link string) *parsedLink {
	encoded := strings.TrimPrefix(link, "vmess://")
	decoded, err := p.decodeBase64(encoded)
	if err != nil {
		return nil
	}

	var vmess map[string]interface{}
	if err := json.Unmarshal(decoded, &vmess); err != nil {
		return nil
	}

	result := &parsedLink{}

	if ps, ok := vmess["ps"].(string); ok {
		result.Name = ps
	}
	if add, ok := vmess["add"].(string); ok {
		result.Server = add
	}

	switch port := vmess["port"].(type) {
	case float64:
		result.Port = int(port)
	case string:
		if p, err := strconv.Atoi(port); err == nil {
			result.Port = p
		}
	}

	if result.Port == 0 {
		return nil
	}

	if net, ok := vmess["net"].(string); ok {
		result.Type = net
	}
	if host, ok := vmess["host"].(string); ok {
		result.Host = host
	}
	if path, ok := vmess["path"].(string); ok {
		result.Path = path
	}

	return result
}

func (p *Parser) convertOutbound(raw json.RawMessage, index int, originalData map[string]*originalLinkData) (*models.ProxyConfig, error) {
	var baseOutbound struct {
		Protocol       string                 `json:"protocol"`
		Tag            string                 `json:"tag"`
		SendThrough    string                 `json:"sendThrough"`
		Settings       json.RawMessage        `json:"settings"`
		StreamSettings *libXrayStreamSettings `json:"streamSettings"`
	}
	if err := json.Unmarshal(raw, &baseOutbound); err != nil {
		return nil, err
	}

	if baseOutbound.Protocol == "freedom" || baseOutbound.Protocol == "blackhole" || baseOutbound.Protocol == "dns" {
		return nil, nil
	}

	pc := &models.ProxyConfig{
		Index:    index,
		Name:     baseOutbound.SendThrough,
		Protocol: baseOutbound.Protocol,
	}

	if pc.Name == "" {
		pc.Name = baseOutbound.Tag
	}

	var flatSettings libXraySettings
	if err := json.Unmarshal(baseOutbound.Settings, &flatSettings); err == nil && flatSettings.Address != "" {
		pc.Server = flatSettings.Address
		pc.Port = flatSettings.Port

		switch baseOutbound.Protocol {
		case "vless":
			pc.UUID = flatSettings.ID
			pc.Flow = flatSettings.Flow
			pc.Encryption = flatSettings.Encryption
			pc.Level = flatSettings.Level
		case "vmess":
			pc.UUID = flatSettings.ID
			pc.AlterId = flatSettings.AlterId
			pc.Security = flatSettings.Security
			pc.Level = flatSettings.Level
		case "trojan":
			pc.Password = flatSettings.Password
		case "shadowsocks":
			pc.Password = flatSettings.Password
			pc.Method = flatSettings.Method
		}
	} else {
		var stdSettings xrayStandardSettings
		if err := json.Unmarshal(baseOutbound.Settings, &stdSettings); err != nil {
			return nil, fmt.Errorf("failed to parse settings: %v", err)
		}

		switch baseOutbound.Protocol {
		case "vless", "vmess":
			if len(stdSettings.Vnext) == 0 || len(stdSettings.Vnext[0].Users) == 0 {
				return nil, fmt.Errorf("no vnext/users found")
			}
			pc.Server = stdSettings.Vnext[0].Address
			pc.Port = stdSettings.Vnext[0].Port
			user := stdSettings.Vnext[0].Users[0]
			pc.UUID = user.ID
			pc.Flow = user.Flow
			pc.Encryption = user.Encryption
			pc.AlterId = user.AlterId
			pc.Level = user.Level
			if baseOutbound.Protocol == "vmess" {
				pc.Security = user.Security
			}
		case "trojan", "shadowsocks":
			if len(stdSettings.Servers) == 0 {
				return nil, fmt.Errorf("no servers found")
			}
			srv := stdSettings.Servers[0]
			pc.Server = srv.Address
			pc.Port = srv.Port
			pc.Password = srv.Password
			pc.Method = srv.Method
			pc.Flow = srv.Flow
		default:
			return nil, fmt.Errorf("unsupported protocol: %s", baseOutbound.Protocol)
		}
	}

	if pc.Server == "" || pc.Port == 0 {
		return nil, fmt.Errorf("failed to parse server/port")
	}

	if pc.Port == 0 || pc.Port == 1 {
		return nil, nil
	}

	if baseOutbound.StreamSettings != nil {
		ss := baseOutbound.StreamSettings
		pc.Type = ss.Network
		pc.Security = ss.Security

		if ss.TlsSettings != nil {
			pc.SNI = ss.TlsSettings.ServerName
			pc.AllowInsecure = ss.TlsSettings.AllowInsecure
			pc.Fingerprint = ss.TlsSettings.Fingerprint
			pc.ALPN = ss.TlsSettings.Alpn
		}

		if ss.RealitySettings != nil {
			pc.SNI = ss.RealitySettings.ServerName
			pc.Fingerprint = ss.RealitySettings.Fingerprint
			pc.PublicKey = ss.RealitySettings.PublicKey
			pc.ShortID = ss.RealitySettings.ShortId
		}

		if ss.Network == "raw" {
			pc.Type = "tcp"
		}

		if ss.RawSettings != nil && ss.RawSettings.Header != nil {
			pc.HeaderType = ss.RawSettings.Header.Type
			if ss.RawSettings.Header.Request != nil {
				if len(ss.RawSettings.Header.Request.Path) > 0 {
					pc.Path = ss.RawSettings.Header.Request.Path[0]
				}
				if ss.RawSettings.Header.Request.Headers != nil && len(ss.RawSettings.Header.Request.Headers.Host) > 0 {
					pc.Host = ss.RawSettings.Header.Request.Headers.Host[0]
				}
			}
		}

		if ss.WsSettings != nil {
			pc.Path = ss.WsSettings.Path
			if ss.WsSettings.Headers != nil {
				pc.Host = ss.WsSettings.Headers.Host
			}
			if pc.Host == "" {
				pc.Host = ss.WsSettings.Host
			}
		}

		if ss.GrpcSettings != nil {
			pc.ServiceName = ss.GrpcSettings.ServiceName
			pc.MultiMode = ss.GrpcSettings.MultiMode
		}

		if ss.HttpSettings != nil {
			pc.Path = ss.HttpSettings.Path
			if len(ss.HttpSettings.Host) > 0 {
				pc.Host = strings.Join(ss.HttpSettings.Host, ",")
			}
		}

		if ss.HttpupgradeSettings != nil {
			pc.Type = "httpupgrade"
			pc.Path = ss.HttpupgradeSettings.Path
			pc.Host = ss.HttpupgradeSettings.Host
		}

		if ss.XhttpSettings != nil {
			pc.Type = "xhttp"
			pc.Path = ss.XhttpSettings.Path
			pc.Host = ss.XhttpSettings.Host
		}

		if ss.SplithttpSettings != nil {
			pc.Type = "splithttp"
			pc.Path = ss.SplithttpSettings.Path
			pc.Host = ss.SplithttpSettings.Host
		}
	}

	key := fmt.Sprintf("%s:%d", pc.Server, pc.Port)
	if orig, ok := originalData[key]; ok {
		if pc.Encryption == "" || pc.Encryption == "none" {
			if orig.Encryption != "" {
				pc.Encryption = orig.Encryption
			}
		}
		if orig.AllowInsecure {
			pc.AllowInsecure = true
		}
	}

	if err := pc.Validate(); err != nil {
		return nil, err
	}

	pc.StableID = pc.GenerateStableID()

	return pc, nil
}

func (p *Parser) tryDecodeBase64(data []byte) []byte {
	text := strings.TrimSpace(string(data))

	if strings.HasPrefix(text, "vless://") || strings.HasPrefix(text, "vmess://") ||
		strings.HasPrefix(text, "trojan://") || strings.HasPrefix(text, "ss://") ||
		strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return data
	}

	decoded, err := p.decodeBase64(text)
	if err != nil {
		return data
	}

	return decoded
}

func (p *Parser) decodeBase64(text string) ([]byte, error) {
	text = strings.ReplaceAll(text, "-", "+")
	text = strings.ReplaceAll(text, "_", "/")

	if m := len(text) % 4; m != 0 {
		text += strings.Repeat("=", 4-m)
	}

	return base64.StdEncoding.DecodeString(text)
}

func (p *Parser) parseFolder(folderPath string) ([]*models.ProxyConfig, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read folder: %v", err)
	}

	var allConfigs []*models.ProxyConfig
	configIndex := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".json" {
			continue
		}

		filePath := filepath.Join(folderPath, fileName)
		data, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warn("Failed to read file %s: %v", fileName, err)
			continue
		}

		configs, err := p.parseSingleConfigFile(data, configIndex)
		if err != nil {
			logger.Warn("Failed to parse file %s: %v", fileName, err)
			continue
		}

		for _, cfg := range configs {
			cfg.Index = configIndex
			allConfigs = append(allConfigs, cfg)
			configIndex++
		}

		logger.Debug("Parsed %d configs from %s", len(configs), fileName)
	}

	if len(allConfigs) == 0 {
		return nil, fmt.Errorf("no valid proxy configurations found in folder")
	}

	logger.Debug("Total configs from folder: %d", len(allConfigs))
	return allConfigs, nil
}

func (p *Parser) parseSingleConfigFile(data []byte, startIndex int) ([]*models.ProxyConfig, error) {
	trimmedData := strings.TrimSpace(string(data))

	if strings.HasPrefix(trimmedData, "[") {
		return p.parseJSONConfigs(data)
	}

	if strings.HasPrefix(trimmedData, "{") {
		var config struct {
			Remarks   string            `json:"remarks"`
			Outbounds []json.RawMessage `json:"outbounds"`
		}

		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %v", err)
		}

		var proxyConfigs []*models.ProxyConfig
		for _, outboundRaw := range config.Outbounds {
			proxyConfig, err := p.convertOutbound(outboundRaw, startIndex, nil)
			if err != nil {
				continue
			}
			if proxyConfig != nil {
				if config.Remarks != "" {
					proxyConfig.Name = config.Remarks
				}
				proxyConfigs = append(proxyConfigs, proxyConfig)
			}
		}

		if len(proxyConfigs) == 0 {
			return nil, fmt.Errorf("no valid proxy configurations found")
		}

		return proxyConfigs, nil
	}

	return nil, fmt.Errorf("unsupported config format")
}
