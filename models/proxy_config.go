package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type ProxyConfig struct {
	Protocol      string
	Server        string
	Port          int
	Name          string
	Security      string
	Type          string
	UUID          string
	Flow          string
	HeaderType    string
	Path          string
	Host          string
	SNI           string
	Fingerprint   string
	PublicKey     string
	ShortID       string
	Mode          string
	ExtraXhttp    string
	Password      string
	Method        string
	Level         int
	AlterId       int
	VMessAid      int
	MultiMode     bool
	ServiceName   string
	IdleTimeout   int
	WindowsSize   int
	AllowInsecure bool
	ALPN          []string
	Index         int
	Settings      map[string]string
	StableID      string
}

func (pc *ProxyConfig) Validate() error {
	if pc.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if pc.Server == "" {
		return fmt.Errorf("server is required")
	}
	if pc.Port <= 0 || pc.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", pc.Port)
	}

	switch pc.Protocol {
	case "vless", "vmess":
		if pc.UUID == "" {
			return fmt.Errorf("UUID is required for %s", pc.Protocol)
		}
	case "trojan":
		if pc.Password == "" {
			return fmt.Errorf("password is required for Trojan")
		}
	case "shadowsocks":
		if pc.Password == "" || pc.Method == "" {
			return fmt.Errorf("password and method are required for Shadowsocks")
		}
	default:
		return fmt.Errorf("unsupported protocol: %s", pc.Protocol)
	}

	return nil
}

func (pc *ProxyConfig) GenerateStableID() string {
	var idComponents []string

	idComponents = append(idComponents, pc.Protocol)

	idComponents = append(idComponents, pc.Server)
	idComponents = append(idComponents, fmt.Sprintf("%d", pc.Port))

	switch pc.Protocol {
	case "vless", "vmess":
		if pc.UUID != "" {
			idComponents = append(idComponents, pc.UUID)
		}
	case "trojan", "shadowsocks":
		if pc.Password != "" {
			idComponents = append(idComponents, pc.Password)
		}
		if pc.Protocol == "shadowsocks" && pc.Method != "" {
			idComponents = append(idComponents, pc.Method)
		}
	}

	if pc.SNI != "" {
		idComponents = append(idComponents, pc.SNI)
	}

	if pc.Type != "" {
		idComponents = append(idComponents, pc.Type)
	}

	if pc.Security != "" {
		idComponents = append(idComponents, pc.Security)
	}

	if pc.PublicKey != "" {
		idComponents = append(idComponents, pc.PublicKey)
	}

	idString := strings.Join(idComponents, "|")

	hash := sha256.Sum256([]byte(idString))

	return hex.EncodeToString(hash[:])[:16]
}

func (pc *ProxyConfig) GetEndpointPath() string {
	return fmt.Sprintf("%s-%s-%d", pc.Protocol, pc.Server, pc.Port)
}

func (pc *ProxyConfig) GetTransportType() string {
	if pc.Type == "" {
		return "tcp"
	}
	return pc.Type
}

func (pc *ProxyConfig) GetSecurityType() string {
	if pc.Security == "" {
		return "none"
	}
	return pc.Security
}

func (pc *ProxyConfig) GetAlterId() int {
	if pc.AlterId == 0 {
		return pc.VMessAid
	}
	return pc.AlterId
}

func (pc *ProxyConfig) GetVMessSecurity() string {
	if pc.Security == "" {
		return "auto"
	}
	return pc.Security
}

func (pc *ProxyConfig) GetUserLevel() int {
	if pc.Level == 0 {
		return 0
	}
	return pc.Level
}

func (pc *ProxyConfig) HasGRPCSettings() bool {
	return pc.Type == "grpc" && pc.ServiceName != ""
}

func (pc *ProxyConfig) HasHTTPUpgradeSettings() bool {
	return pc.Type == "httpupgrade"
}

func (pc *ProxyConfig) GetServiceName() string {
	if pc.ServiceName == "" {
		return "GunService"
	}
	return pc.ServiceName
}

func (pc *ProxyConfig) GetALPNSettings() []string {
	if len(pc.ALPN) == 0 {
		return []string{"h2", "http/1.1"}
	}
	return pc.ALPN
}
