package models

import "encoding/json"

type XrayConfig struct {
	Log struct {
		LogLevel string `json:"loglevel"`
	} `json:"log"`
	Inbounds  []XrayInbound  `json:"inbounds"`
	Outbounds []XrayOutbound `json:"outbounds"`
	Routing   XrayRouting    `json:"routing"`
}

type XrayInbound struct {
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
	Sniffing struct {
		Enabled      bool     `json:"enabled"`
		DestOverride []string `json:"destOverride"`
		RouteOnly    bool     `json:"routeOnly"`
	} `json:"sniffing"`
}

type XrayOutbound struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
}

type StreamSettings struct {
	Network             string               `json:"network,omitempty"`
	Security            string               `json:"security,omitempty"`
	TLSSettings         *TLSSettings         `json:"tlsSettings,omitempty"`
	RealitySettings     *RealitySettings     `json:"realitySettings,omitempty"`
	WSSettings          *WebSocketSettings   `json:"wsSettings,omitempty"`
	HTTPUpgradeSettings *HTTPUpgradeSettings `json:"httpupgradeSettings,omitempty"`
	SockOpt             json.RawMessage      `json:"sockopt"`
}

type TLSSettings struct {
	AllowInsecure bool `json:"allowInsecure"`
}

type RealitySettings struct {
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	ShortID     string `json:"shortId"`
}

type WebSocketSettings struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

type HTTPUpgradeSettings struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
}

type XrayRouting struct {
	Rules []XrayRoutingRule `json:"rules"`
}

type XrayRoutingRule struct {
	Type        string   `json:"type"`
	InboundTag  []string `json:"inboundTag"`
	OutboundTag string   `json:"outboundTag"`
}
