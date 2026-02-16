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

// decodeGeoIPResponse parses an ip-api.com response into GeoIPInfo.
func decodeGeoIPResponse(resp *http.Response) (*GeoIPInfo, error) {
	defer resp.Body.Close()

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode geoip response: %v", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("geoip lookup failed for IP %s", result.Query)
	}

	return &GeoIPInfo{
		IP:          result.Query,
		CountryCode: result.CountryCode,
		Country:     result.Country,
		LastChecked: time.Now(),
	}, nil
}

// LookupGeoIP looks up the country for an IP address using ip-api.com.
func LookupGeoIP(ip string, client *http.Client) (*GeoIPInfo, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil, fmt.Errorf("empty IP address")
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,query", ip)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("geoip lookup failed: %v", err)
	}
	return decodeGeoIPResponse(resp)
}

// LookupGeoIPViaClient looks up GeoIP by calling ip-api.com through the provided HTTP client.
// When the client routes through a proxy, ip-api.com sees the proxy's exit IP and returns its location.
func LookupGeoIPViaClient(client *http.Client) (*GeoIPInfo, error) {
	resp, err := client.Get("http://ip-api.com/json/?fields=status,country,countryCode,query")
	if err != nil {
		return nil, fmt.Errorf("geoip lookup failed: %v", err)
	}
	return decodeGeoIPResponse(resp)
}

// CountryCodeToFlag converts a 2-letter country code to a flag emoji
func CountryCodeToFlag(code string) string {
	if len(code) != 2 {
		return ""
	}
	code = strings.ToUpper(code)
	first := rune(code[0]) - 'A' + 0x1F1E6
	second := rune(code[1]) - 'A' + 0x1F1E6
	return string([]rune{first, second})
}
