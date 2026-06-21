package geoip

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"sharelink/internal/config"
	"sharelink/internal/db"
	"sharelink/internal/security"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var (
	searcher *xdb.Searcher
	cache    sync.Map // IP -> GeoInfo
)

type GeoInfo struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`
}

func Init() {
	dbPath := config.AppConfig.IPDBPath
	if dbPath == "" {
		log.Fatal("GeoIP IPDBPath is not set. Refusing to start.")
	}

	// Read database file into memory buffer
	dbBuff, err := os.ReadFile(dbPath)
	if err != nil {
		log.Fatalf("failed to read GeoIP database file from %s: %v. Refusing to start.", dbPath, err)
	}

	// Load header from the buffer
	header, err := xdb.LoadHeaderFromBuff(dbBuff)
	if err != nil {
		log.Fatalf("failed to load GeoIP header from buffer: %v", err)
	}

	// Extract the Version from the header
	version, err := xdb.VersionFromHeader(header)
	if err != nil {
		log.Fatalf("failed to extract GeoIP version from header: %v", err)
	}

	// Create searcher with buffer
	searcher, err = xdb.NewWithBuffer(version, dbBuff)
	if err != nil {
		log.Fatalf("failed to initialize GeoIP searcher from buffer: %v", err)
	}

	log.Printf("GeoIP database loaded successfully from %s", dbPath)
}

func ResolveIP(ipStr string) GeoInfo {
	if !IsEnabled() {
		return GeoInfo{Country: "未知", Region: "未知", City: "未知"}
	}

	// Clean IP string (remove port if exists)
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}
	ipStr = strings.TrimSpace(ipStr)

	// Check cache
	if val, ok := cache.Load(ipStr); ok {
		return val.(GeoInfo)
	}

	// Parse IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoInfo{Country: "未知", Region: "未知", City: "未知"}
	}

	// Check if internal IP
	if security.IsPrivateIP(ip) {
		info := GeoInfo{Country: "内网/本地", Region: "内网/本地", City: "内网/本地"}
		cache.Store(ipStr, info)
		return info
	}

	if searcher == nil {
		return GeoInfo{Country: "未知", Region: "未知", City: "未知"}
	}

	// Search in database
	regionStr, err := searcher.Search(ipStr)
	if err != nil {
		info := GeoInfo{Country: "未知", Region: "未知", City: "未知"}
		cache.Store(ipStr, info)
		return info
	}

	// Region string format in the bundled xdb: 国家|省/州|城市|ISP|国家码
	parts := strings.Split(regionStr, "|")
	if len(parts) < 5 {
		info := GeoInfo{Country: "未知", Region: "未知", City: "未知"}
		cache.Store(ipStr, info)
		return info
	}

	country := parts[0]
	region := parts[1]
	city := parts[2]

	if country == "0" {
		country = "未知"
	}
	if region == "0" {
		region = ""
	}
	if city == "0" {
		city = ""
	}

	info := GeoInfo{
		Country: country,
		Region:  region,
		City:    city,
	}

	cache.Store(ipStr, info)
	return info
}

func IsEnabled() bool {
	if db.DB == nil {
		return true
	}
	setting, found, err := db.FindGlobalSetting("geoip_enabled")
	if err != nil || !found {
		return true
	}
	return setting.Value == "true" || setting.Value == "1"
}
