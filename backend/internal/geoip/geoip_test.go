package geoip

import (
	"os"
	"sharelink/internal/config"
	"testing"
)

func TestGeoIP(t *testing.T) {
	// Set database path for test
	os.Setenv("IP_DB_PATH", "../../data/ip2region.xdb")
	defer os.Unsetenv("IP_DB_PATH")

	config.Load()
	Init()

	// Test resolving internal IP
	info := ResolveIP("127.0.0.1")
	if info.Country != "内网/本地" {
		t.Errorf("expected Country for 127.0.0.1 to be '内网/本地', got %s", info.Country)
	}

	// Test resolving a public IP (e.g. 8.8.8.8)
	infoPublic := ResolveIP("8.8.8.8")
	if infoPublic.Country == "未知" || infoPublic.Country == "" {
		t.Errorf("expected Country for 8.8.8.8 to be resolved, got %s", infoPublic.Country)
	}
	t.Logf("8.8.8.8 resolved to: Country=%s, Region=%s, City=%s", infoPublic.Country, infoPublic.Region, infoPublic.City)
}

func TestGeoIPFieldMapping(t *testing.T) {
	os.Setenv("IP_DB_PATH", "../../data/ip2region.xdb")
	defer os.Unsetenv("IP_DB_PATH")

	config.Load()
	Init()

	tests := []struct {
		ip      string
		country string
		region  string
		city    string
	}{
		{ip: "14.156.44.53", country: "中国", region: "广东省", city: "东莞市"},
		{ip: "36.27.39.163", country: "中国", region: "浙江省", city: "杭州市"},
		{ip: "141.11.42.161", country: "Netherlands", region: "Utrecht", city: ""},
	}

	for _, tt := range tests {
		info := ResolveIP(tt.ip)
		if info.Country != tt.country || info.Region != tt.region || info.City != tt.city {
			t.Errorf(
				"%s resolved to Country=%q Region=%q City=%q, want Country=%q Region=%q City=%q",
				tt.ip,
				info.Country,
				info.Region,
				info.City,
				tt.country,
				tt.region,
				tt.city,
			)
		}
	}
}
