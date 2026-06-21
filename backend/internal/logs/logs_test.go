package logs

import (
	"os"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"testing"
	"time"
)

func TestLogsQueue(t *testing.T) {
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	os.Setenv("DB_DSN", "file::memory:?cache=shared")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")

	config.Load()
	db.Init()
	Init()

	// Queue some logs
	for i := 0; i < 5; i++ {
		QueueLog(&db.VisitLog{
			Prefix:      "/export",
			Slug:        "file",
			PublicPath:  "/export/file",
			IP:          "1.2.3.4",
			AccessTime:  time.Now(),
			Status:      "success",
			CacheStatus: "miss",
		})
	}

	// Give the background worker a bit of time to flush (ticker is 500ms)
	time.Sleep(600 * time.Millisecond)

	// Query DB
	var count int64
	err := db.DB.Model(&db.VisitLog{}).Count(&count).Error
	if err != nil {
		t.Fatalf("failed to query visit logs count: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 logs, got %d", count)
	}

	// Test cleanup
	// Create an expired log entry (91 days ago)
	expiredTime := time.Now().AddDate(0, 0, -91)
	db.DB.Create(&db.VisitLog{
		Prefix:      "/export",
		Slug:        "file",
		PublicPath:  "/export/file",
		IP:          "1.2.3.4",
		AccessTime:  expiredTime,
		Status:      "success",
		CacheStatus: "miss",
	})

	// Run cleanup manually
	runCleanup()

	// Query count again - should still be 5 because the expired one was cleaned
	_ = db.DB.Model(&db.VisitLog{}).Count(&count).Error
	if count != 5 {
		t.Errorf("expected 5 logs after cleanup, got %d", count)
	}
}
