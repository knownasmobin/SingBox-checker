package checker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GeoIPInfo holds the geolocation information for an IP
type GeoIPInfo struct {
	IP          string
	CountryCode string
	Country     string
	LastChecked time.Time
}

// ipAPIResponse is the response from ip-api.com
type ipAPIResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Query       string `json:"query"`
}

// LookupGeoIP looks up the country for an IP address using ip-api.com
func LookupGeoIP(ip string) (*GeoIPInfo, error) {
	// Clean IP (remove newlines, spaces)
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil, fmt.Errorf("empty IP address")
	}

	// Use ip-api.com free API (no key required, 45 requests/minute limit)
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,query", ip)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("geoip lookup failed: %v", err)
	}
	defer resp.Body.Close()

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode geoip response: %v", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("geoip lookup failed for IP %s", ip)
	}

	return &GeoIPInfo{
		IP:          result.Query,
		CountryCode: result.CountryCode,
		Country:     result.Country,
		LastChecked: time.Now(),
	}, nil
}

// CountryCodeToFlag converts a 2-letter country code to a flag emoji
func CountryCodeToFlag(code string) string {
	if len(code) != 2 {
		return ""
	}
	code = strings.ToUpper(code)
	// Convert each letter to a regional indicator symbol
	// A = U+1F1E6, B = U+1F1E7, etc.
	first := rune(code[0]) - 'A' + 0x1F1E6
	second := rune(code[1]) - 'A' + 0x1F1E6
	return string([]rune{first, second})
}
