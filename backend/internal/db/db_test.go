package db

import (
	"os"
	"sharelink/internal/config"
	"testing"
	"time"
)

func TestDBInit(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	os.Setenv("DB_DSN", "file::memory:?cache=shared")
	os.Setenv("APP_TIMEZONE", "Asia/Shanghai")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")
	defer os.Unsetenv("APP_TIMEZONE")

	config.Load()
	Init()

	if DB == nil {
		t.Fatal("expected DB connection to be initialized, got nil")
	}

	// Verify that password setting was created
	var setting GlobalSetting
	err := DB.First(&setting, "key = ?", "admin_password_hash").Error
	if err != nil {
		t.Fatalf("failed to find admin_password_hash setting: %v", err)
	}

	if setting.Value == "" {
		t.Fatal("expected admin_password_hash to be non-empty")
	}

	if setting.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected admin password updated_at to use UTC, got %s", setting.UpdatedAt.Location())
	}
}

func TestGormAutoTimestampsUseUTC(t *testing.T) {
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	os.Setenv("DB_DSN", "file:utc-timestamps?mode=memory&cache=shared")
	os.Setenv("APP_TIMEZONE", "Asia/Shanghai")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")
	defer os.Unsetenv("APP_TIMEZONE")

	config.Load()
	Init()

	link := Link{
		Prefix:       "/utc",
		Slug:         "stamp",
		PublicPath:   "/utc/stamp",
		TargetURL:    "https://example.com/file",
		Mode:         "proxy",
		Enabled:      true,
		FilenameMode: "inherit",
	}

	if err := DB.Create(&link).Error; err != nil {
		t.Fatalf("failed to create link: %v", err)
	}

	if link.CreatedAt.Location() != time.UTC {
		t.Fatalf("expected created_at to use UTC, got %s", link.CreatedAt.Location())
	}
	if link.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected updated_at to use UTC, got %s", link.UpdatedAt.Location())
	}
}
