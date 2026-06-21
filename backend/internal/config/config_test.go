package config

import (
	"os"
	"testing"
	"time"
)

func TestBusinessDateUsesConfiguredTimezone(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Shanghai")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	Load()

	instant := time.Date(2026, 6, 20, 16, 30, 0, 0, time.UTC)
	if got := BusinessDate(instant); got != "2026-06-21" {
		t.Fatalf("expected Shanghai business date 2026-06-21, got %s", got)
	}

	start := BusinessDayStartUTC(instant)
	want := time.Date(2026, 6, 20, 16, 0, 0, 0, time.UTC)
	if !start.Equal(want) {
		t.Fatalf("expected Shanghai day start %s UTC, got %s", want, start)
	}
}

func TestInvalidTimezoneFallsBackToUTC(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Invalid/Zone")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	Load()

	if AppConfig.AppTimezone != "UTC" {
		t.Fatalf("expected invalid timezone to fall back to UTC, got %s", AppConfig.AppTimezone)
	}
	if Location() != time.UTC {
		t.Fatal("expected UTC location after invalid timezone fallback")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	AppConfig = nil
	os.Exit(code)
}
