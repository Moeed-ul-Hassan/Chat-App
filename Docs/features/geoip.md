# Feature Specification: Geolocation Service (GeoIP)

**Status**: ✅ Implemented
**Location**: `backend/core/geoip/geoip.go`

## Overview
The GeoIP service is responsible for identifying the user's country code based on their IP address. This information is used during the World Chat flow to assign users to country-specific channels (Milestone 2).

## Implementation Details

### Provider
- **Service**: [ip-api.com](http://ip-api.com) (JSON API)
- **Endpoint**: `http://ip-api.com/json/{query}`
- **Rate Limit**: 45 requests per minute (Free Tier).

### Logic & Fallbacks
- **Local/Loopback handling**: If the detected IP is `127.0.0.1` or `::1` (standard for development), the service returns `"PK"` as the default country code for testing.
- **Failures**: In case of a failure (API down or invalid response), the service defaults to `"US"`.
- **Inbound validation**: Cleans the IP address string to remove ports (e.g., `127.0.0.1:54321` → `127.0.0.1`).

## Data Structure

```go
type Info struct {
	Status      string  `json:"status"`      // "success" or "fail"
	Country     string  `json:"country"`     // Full name: e.g., "United States"
	CountryCode string  `json:"countryCode"` // alpha-2: e.g., "US"
	Lat/Lon     float64 `json:"lat"/"lon"`   // Coords for potential advanced use
	AS          string  `json:"as"`          // Autonomous System
	Query       string  `json:"query"`       // The IP searched
}
```

## Security Implications
1. **Privacy**: We do *not* store the full GeoIP response for every request. We only extract the `CountryCode` for channel assignment.
2. **Reliability**: The 5-second timeout on the `httpClient` ensures that if the third-party GeoIP service is slow, it won't block our WebSocket or API processes for more than 5 seconds.
3. **Accuracy**: VPNs and Proxies will report the IP of the proxy server. This is expected behavior and will result in the user seeing the channel of the VPN's country.
