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
