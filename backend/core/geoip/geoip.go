package geoip

import (
	"encoding/json" //Json Working.
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Info represents the response from the ip-api.com
type Info struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
	Message     string  `json:"message"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

// GetCountryCodeFromIP detects the country code (ISO 3166-1 alpha-2) from an IP address.
func GetCountryCodeFromIP(ipStr string) (string, error) {
	// Clean the IP string (remove port if present)
	host, _, err := net.SplitHostPort(ipStr)
	if err == nil {
		ipStr = host
	}

	// Handle local/loopback IPs
	if ipStr == "127.0.0.1" || ipStr == "::1" || ipStr == "localhost" {
		fmt.Printf("📍 Local IP detected (%s). Defaulting to PK\n", ipStr)
		return "PK", nil
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s", ipStr)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "US", fmt.Errorf("failed to call geoip api: %w", err)
	}
	defer resp.Body.Close()

	var info Info
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "US", fmt.Errorf("failed to decode geoip response: %w", err)
	}

	if info.Status == "fail" {
		fmt.Printf("⚠️ GeoIP lookup failed for %s: %s. Defaulting to US\n", ipStr, info.Message)
		return "US", nil
	}

	return strings.ToUpper(info.CountryCode), nil
}
