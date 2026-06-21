package ua

import (
	"os"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"testing"
)

func TestValidateUA(t *testing.T) {
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	os.Setenv("DB_DSN", "file::memory:?cache=shared")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")

	config.Load()
	db.Init()

	policy := &db.UAPolicy{
		Name:          "Test Policy",
		Enabled:       true,
		Mode:          "blacklist",
		BlockKeywords: `["curl", "wget", "spider"]`,
		AllowEmptyUA:  false,
		CaseSensitive: false,
		MatchType:     "contains",
	}

	// Test 1: Empty UA when not allowed
	allowed, reason := ValidateUA("", policy)
	if allowed {
		t.Error("expected empty UA to be blocked")
	}
	if reason != "empty_ua_blocked" {
		t.Errorf("expected reason 'empty_ua_blocked', got '%s'", reason)
	}

	// Test 2: Standard browser UA (allowed)
	allowed, _ = ValidateUA("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0", policy)
	if !allowed {
		t.Error("expected Chrome UA to be allowed")
	}

	// Test 3: Blocked UA (curl)
	allowed, reason = ValidateUA("curl/7.68.0", policy)
	if allowed {
		t.Error("expected curl UA to be blocked")
	}
	if reason != "ua_blocked" {
		t.Errorf("expected reason 'ua_blocked', got '%s'", reason)
	}

	// Test 4: Whitelist mode
	whitelistPolicy := &db.UAPolicy{
		Name:          "Whitelist Policy",
		Enabled:       true,
		Mode:          "whitelist",
		AllowKeywords: `["Chrome", "Safari"]`,
		AllowEmptyUA:  true,
		CaseSensitive: false,
		MatchType:     "contains",
	}

	allowed, _ = ValidateUA("Mozilla/5.0 Chrome/120", whitelistPolicy)
	if !allowed {
		t.Error("expected Chrome to be allowed in whitelist")
	}

	allowed, _ = ValidateUA("curl/7.68.0", whitelistPolicy)
	if allowed {
		t.Error("expected curl to be blocked in whitelist")
	}

	// Test 5: Regex Match
	regexPolicy := &db.UAPolicy{
		Name:          "Regex Policy",
		Enabled:       true,
		Mode:          "blacklist",
		BlockKeywords: `["^curl/\\d+"]`,
		AllowEmptyUA:  true,
		CaseSensitive: false,
		MatchType:     "regex",
	}
	allowed, _ = ValidateUA("curl/7", regexPolicy)
	if allowed {
		t.Error("expected curl/7 to match regex blacklist")
	}
	allowed, _ = ValidateUA("mycurl/7", regexPolicy)
	if !allowed {
		t.Error("expected mycurl/7 to not match regex blacklist")
	}
}
