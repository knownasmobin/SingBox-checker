package models

// SingBoxConfig represents the root configuration for SingBox
type SingBoxConfig struct {
	Log         *LogConfig         `json:"log,omitempty"`
	DNS         *DNSConfig         `json:"dns,omitempty"`
	Inbounds    []Inbound          `json:"inbounds,omitempty"`
	Outbounds   []Outbound         `json:"outbounds,omitempty"`
	Route       *RouteConfig       `json:"route,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
}

// LogConfig represents the logging configuration
type LogConfig struct {
	Level      string `json:"level,omitempty"`
	Timestamp  bool   `json:"timestamp,omitempty"`
	Output     string `json:"output,omitempty"`
}

// DNSConfig represents the DNS configuration
type DNSConfig struct {
	Servers []DNSServer `json:"servers,omitempty"`
}

// DNSServer represents a DNS server configuration
type DNSServer struct {
	Address     string   `json:"address,omitempty"`
	Domains     []string `json:"domains,omitempty"`
	ExpectIPs   []string `json:"expect_ips,omitempty"`
}

// Inbound represents an inbound proxy configuration
type Inbound struct {
	Type          string                 `json:"type"`
	Tag           string                 `json:"tag,omitempty"`
	Listen        string                 `json:"listen,omitempty"`
	ListenPort    int                    `json:"listen_port,omitempty"`
	Users         []User                 `json:"users,omitempty"`
	TLS           *TLSConfig             `json:"tls,omitempty"`
	Transport     *TransportConfig       `json:"transport,omitempty"`
	Sniff         bool                   `json:"sniff,omitempty"`
	SniffOverride bool                   `json:"sniff_override,omitempty"`
	Settings      map[string]interface{} `json:"settings,omitempty"`
}

// Outbound represents an outbound proxy configuration
type Outbound struct {
	Type        string                 `json:"type"`
	Tag         string                 `json:"tag,omitempty"`
	Server      string                 `json:"server,omitempty"`
	ServerPort  int                    `json:"server_port,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Password    string                 `json:"password,omitempty"`
	UUID        string                 `json:"uuid,omitempty"`
	Flow        string                 `json:"flow,omitempty"`
	Security    string                 `json:"security,omitempty"`
	TLS         *TLSConfig             `json:"tls,omitempty"`
	Transport   *TransportConfig       `json:"transport,omitempty"`
	Plugin      string                 `json:"plugin,omitempty"`
	PluginOpts  map[string]interface{} `json:"plugin_opts,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// User represents a user configuration
type User struct {
	Name     string `json:"name,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled             bool     `json:"enabled,omitempty"`
	ServerName         string   `json:"server_name,omitempty"`
	Insecure           bool     `json:"insecure,omitempty"`
	ALPN               []string `json:"alpn,omitempty"`
	MinVersion         string   `json:"min_version,omitempty"`
	MaxVersion         string   `json:"max_version,omitempty"`
	CipherSuites       []string `json:"cipher_suites,omitempty"`
	Certificate        string   `json:"certificate,omitempty"`
	CertificatePath    string   `json:"certificate_path,omitempty"`
	Key                string   `json:"key,omitempty"`
	KeyPath            string   `json:"key_path,omitempty"`
	Fingerprint        string   `json:"fingerprint,omitempty"`
}

// TransportConfig represents transport layer configuration
type TransportConfig struct {
	Type        string                 `json:"type"`
	Path        string                 `json:"path,omitempty"`
	Host        string                 `json:"host,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	ServiceName string                 `json:"service_name,omitempty"`
	IdleTimeout string                 `json:"idle_timeout,omitempty"`
	PingTimeout string                 `json:"ping_timeout,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// RouteConfig represents routing configuration
type RouteConfig struct {
	Rules    []Rule   `json:"rules,omitempty"`
	AutoDetectInterface bool     `json:"auto_detect_interface,omitempty"`
	DefaultInterface   string   `json:"default_interface,omitempty"`
}

// Rule represents a routing rule
type Rule struct {
	Type        string   `json:"type,omitempty"`
	OutboundTag string   `json:"outbound_tag,omitempty"`
	InboundTag  []string `json:"inbound_tag,omitempty"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	Protocol    []string `json:"protocol,omitempty"`
}

// ExperimentalConfig represents experimental features
type ExperimentalConfig struct {
	CacheFile string `json:"cache_file,omitempty"`
}

// NewDefaultSingBoxConfig creates a new SingBoxConfig with default values
func NewDefaultSingBoxConfig() *SingBoxConfig {
	return &SingBoxConfig{
		Log: &LogConfig{
			Level:      "info",
			Timestamp:  true,
			Output:     "",
		},
		Route: &RouteConfig{
			Rules: []Rule{
				{
					Type:        "field",
					OutboundTag: "direct",
					Domain:      []string{"geosite:cn"},
				},
				{
					Type:        "field",
					OutboundTag: "proxy",
					Domain:      []string{"geosite:geolocation-!cn"},
				},
			},
		},
	}
}
