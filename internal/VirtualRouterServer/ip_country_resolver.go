package VirtualRouterServer

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipCountryResolver struct {
	mu     sync.RWMutex
	cache  map[string]IPGeoInfo
	client *http.Client
}

type IPGeoInfo struct {
	Country     string
	CountryCode string
	RegionName  string
	City        string
	ISP         string
	Org         string
	AS          string
}

var globalIPCountryResolver = &ipCountryResolver{
	cache: make(map[string]IPGeoInfo),
	client: &http.Client{
		Timeout: 1500 * time.Millisecond,
	},
}

func resolveIPCountry(ip string) string {
	geo := resolveIPGeo(ip)
	if geo.Country == "" {
		return "未知"
	}
	if geo.CountryCode != "" {
		return geo.Country + " (" + geo.CountryCode + ")"
	}
	return geo.Country
}

func resolveIPGeo(ip string) IPGeoInfo {
	cleaned := strings.TrimSpace(ip)
	if cleaned == "" {
		return IPGeoInfo{Country: "-"}
	}

	parsed := net.ParseIP(cleaned)
	if parsed == nil {
		return IPGeoInfo{Country: "未知"}
	}
	if parsed.IsLoopback() {
		return IPGeoInfo{Country: "本机", CountryCode: "LOCAL"}
	}
	if parsed.IsPrivate() {
		return IPGeoInfo{Country: "内网", CountryCode: "LAN"}
	}

	if cached, ok := globalIPCountryResolver.get(cleaned); ok {
		return cached
	}

	geo := globalIPCountryResolver.fetch(cleaned)
	globalIPCountryResolver.set(cleaned, geo)
	return geo
}

func (r *ipCountryResolver) get(ip string) (IPGeoInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.cache[ip]
	return v, ok
}

func (r *ipCountryResolver) set(ip string, geo IPGeoInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[ip] = geo
}

func (r *ipCountryResolver) fetch(ip string) IPGeoInfo {
	url := "http://ip-api.com/json/" + ip + "?fields=status,country,countryCode,regionName,city,isp,org,as"
	resp, err := r.client.Get(url)
	if err != nil {
		return IPGeoInfo{Country: "未知"}
	}
	defer resp.Body.Close()

	var result struct {
		Status      string `json:"status"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		RegionName  string `json:"regionName"`
		City        string `json:"city"`
		ISP         string `json:"isp"`
		Org         string `json:"org"`
		AS          string `json:"as"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return IPGeoInfo{Country: "未知"}
	}
	if strings.ToLower(result.Status) != "success" {
		return IPGeoInfo{Country: "未知"}
	}
	if result.Country == "" {
		result.Country = "未知"
	}
	return IPGeoInfo{
		Country:     result.Country,
		CountryCode: result.CountryCode,
		RegionName:  result.RegionName,
		City:        result.City,
		ISP:         result.ISP,
		Org:         result.Org,
		AS:          result.AS,
	}
}
