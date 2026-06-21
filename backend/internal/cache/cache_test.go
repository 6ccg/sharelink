package cache

import (
	"net/http"
	"os"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"testing"
	"time"
)

func TestRAMCache(t *testing.T) {
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	os.Setenv("DB_DSN", "file::memory:?cache=shared")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")

	config.Load()
	db.Init()

	c := GetGlobalCache()
	c.ClearAll()

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	// Test 1: Put and Get
	c.Put("key1", 200, headers, []byte("body1"), 5)
	item, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to be found")
	}
	if string(item.Body) != "body1" {
		t.Errorf("expected 'body1', got '%s'", string(item.Body))
	}

	// Test 2: Expire
	c.Put("key2", 200, headers, []byte("body2"), 1)
	time.Sleep(1100 * time.Millisecond)
	_, ok = c.Get("key2")
	if ok {
		t.Error("expected key2 to have expired")
	}

	// Test 3: Eviction
	// Set dynamic cache limit to 1MB
	setting := db.GlobalSetting{Key: "global_cache_max_memory_mb", Value: "1", UpdatedAt: time.Now()}
	db.DB.Save(&setting)

	// Put an item that fits
	c.Put("key_fit", 200, headers, []byte("small"), 10)
	// Put a huge item that forces eviction
	hugeBody := make([]byte, 1024*1024) // 1MB
	c.Put("key_huge", 200, headers, hugeBody, 10)

	// key_fit should be evicted because it exceeds 1MB limit
	_, ok = c.Get("key_fit")
	if ok {
		t.Error("expected key_fit to be evicted")
	}

	// Test 4: Clear link cache
	c.Put("/export|file1|GET|a=1", 200, headers, []byte("data"), 10)
	c.Put("/export|file1|GET|b=2", 200, headers, []byte("data"), 10)
	c.Put("/export|file2|GET|a=1", 200, headers, []byte("data"), 10)

	c.ClearLinkCache("/export", "file1")

	_, ok = c.Get("/export|file1|GET|a=1")
	if ok {
		t.Error("expected /export|file1|GET|a=1 to be cleared")
	}
	_, ok = c.Get("/export|file1|GET|b=2")
	if ok {
		t.Error("expected /export|file1|GET|b=2 to be cleared")
	}
	_, ok = c.Get("/export|file2|GET|a=1")
	if !ok {
		t.Error("expected /export|file2|GET|a=1 to remain cached")
	}
}
