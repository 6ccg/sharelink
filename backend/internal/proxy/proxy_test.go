package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"sharelink/internal/cache"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"sharelink/internal/geoip"
	"sharelink/internal/logs"
	"sharelink/internal/security"

	"github.com/gin-gonic/gin"
)

func TestProxyRedirect(t *testing.T) {
	security.AllowLoopback = true
	t.Cleanup(func() { security.AllowLoopback = false })
	// Init DB and config
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass")
	os.Setenv("DB_DSN", "file::memory:?cache=shared")
	os.Setenv("IP_DB_PATH", "../../data/ip2region.xdb")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")
	defer os.Unsetenv("IP_DB_PATH")

	config.Load()
	db.Init()
	geoip.Init()
	logs.Init()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/:prefix/:slug", ProxyHandler)

	// Create a redirect link
	link := db.Link{
		Prefix:     "/go",
		Slug:       "test-redirect",
		PublicPath: "/go/test-redirect",
		TargetURL:  "http://127.0.0.1/public?q=sharelink",
		Mode:       "redirect",
		Enabled:    true,
	}
	db.DB.Create(&link)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/go/test-redirect", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got status %d", w.Code)
	}

	if w.Header().Get("Location") != "http://127.0.0.1/public?q=sharelink" {
		t.Errorf("expected redirect location to be google search, got %s", w.Header().Get("Location"))
	}
}

func TestProxyRedirectFromNoRouteFallback(t *testing.T) {
	security.AllowLoopback = true
	t.Cleanup(func() { security.AllowLoopback = false })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.NoRoute(ProxyHandler)

	link := db.Link{
		Prefix:     "/go",
		Slug:       "fallback-redirect",
		PublicPath: "/go/fallback-redirect",
		TargetURL:  "http://127.0.0.1/fallback",
		Mode:       "redirect",
		Enabled:    true,
	}
	db.DB.Create(&link)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/go/fallback-redirect", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect through NoRoute fallback, got status %d", w.Code)
	}

	if w.Header().Get("Location") != "http://127.0.0.1/fallback" {
		t.Errorf("expected redirect location to be example fallback, got %s", w.Header().Get("Location"))
	}
}

func TestProxyRequestFlow(t *testing.T) {
	security.AllowLoopback = true
	t.Cleanup(func() { security.AllowLoopback = false })

	// Create mock upstream server
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that cookies were stripped
		if r.Header.Get("Cookie") != "" {
			t.Error("expected Cookie header to be stripped by proxy")
		}

		// Check that host header was rewritten
		if r.Host == "" {
			t.Error("expected Host header to be rewritten")
		}

		// Set custom response headers
		w.Header().Set("Set-Cookie", "session=123")
		w.Header().Set("X-Upstream-Header", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock upstream body"))
	}))
	defer mockUpstream.Close()

	// Setup routing
	r := gin.New()
	r.Any("/:prefix/:slug", ProxyHandler)

	// Create proxy link
	link := db.Link{
		Prefix:       "/export",
		Slug:         "test-proxy",
		PublicPath:   "/export/test-proxy",
		TargetURL:    mockUpstream.URL,
		Mode:         "proxy",
		Enabled:      true,
		CacheEnabled: false,
	}
	db.DB.Create(&link)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/export/test-proxy", nil)
	req.Header.Set("Cookie", "session=client_session")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Set-Cookie") != "" {
		t.Error("expected Set-Cookie to be stripped from upstream response")
	}

	if w.Header().Get("X-Upstream-Header") != "yes" {
		t.Error("expected other headers to be pass through")
	}

	if w.Body.String() != "mock upstream body" {
		t.Errorf("expected 'mock upstream body', got '%s'", w.Body.String())
	}
}

func TestProxyDoesNotAppendPublicPathToTarget(t *testing.T) {
	security.AllowLoopback = true
	t.Cleanup(func() { security.AllowLoopback = false })

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/data.json" {
			t.Errorf("expected upstream path /api/data.json, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "token=abc&page=2" {
			t.Errorf("expected merged query token=abc&page=2, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer mockUpstream.Close()

	r := gin.New()
	r.Any("/:prefix/:slug", ProxyHandler)

	link := db.Link{
		Prefix:     "/export",
		Slug:       "mapped",
		PublicPath: "/export/mapped",
		TargetURL:  mockUpstream.URL + "/api/data.json?token=abc",
		Mode:       "proxy",
		Enabled:    true,
	}
	db.DB.Create(&link)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/export/mapped?page=2", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestProxyCacheAndFilename(t *testing.T) {
	security.AllowLoopback = true
	t.Cleanup(func() { security.AllowLoopback = false })

	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file contents"))
	}))
	defer mockUpstream.Close()

	// Clear cache
	c := cache.GetGlobalCache()
	c.ClearAll()

	r := gin.New()
	r.Any("/:prefix/:slug", ProxyHandler)

	// Create proxy link with custom filename and cache enabled
	filename := "report.txt"
	link := db.Link{
		Prefix:         "/s",
		Slug:           "download",
		PublicPath:     "/s/download",
		TargetURL:      mockUpstream.URL,
		Mode:           "proxy",
		Enabled:        true,
		CacheEnabled:   true,
		CacheTTL:       60,
		FilenameMode:   "custom",
		CustomFilename: &filename,
	}
	db.DB.Create(&link)

	// Enable global cache in settings
	db.DB.Save(&db.GlobalSetting{Key: "global_cache_enabled", Value: "true", UpdatedAt: time.Now()})

	// First Request (Cache Miss)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/s/download", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("first request failed: status %d", w.Code)
	}

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") || !strings.Contains(cd, "report.txt") {
		t.Errorf("expected Content-Disposition with report.txt, got %s", cd)
	}

	// Verify Cache Status
	time.Sleep(100 * time.Millisecond) // wait for async log or cache write if any
	_, exists := c.Get("/s|download|GET|")
	if !exists {
		t.Fatal("expected item to be in cache")
	}

	// Second Request (Cache Hit)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/s/download", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("second request failed: status %d", w2.Code)
	}

	if w2.Body.String() != "file contents" {
		t.Errorf("expected cached body 'file contents', got '%s'", w2.Body.String())
	}
}

func TestProxyRedirectInterception(t *testing.T) {
	security.AllowLoopback = true
	t.Cleanup(func() { security.AllowLoopback = false })

	// Redirect target upstream server
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://127.0.0.1/private")
		w.WriteHeader(http.StatusFound)
	}))
	defer mockUpstream.Close()

	r := gin.New()
	r.Any("/:prefix/:slug", ProxyHandler)

	link := db.Link{
		Prefix:     "/s",
		Slug:       "redirect-upstream",
		PublicPath: "/s/redirect-upstream",
		TargetURL:  mockUpstream.URL,
		Mode:       "proxy",
		Enabled:    true,
	}
	db.DB.Create(&link)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/s/redirect-upstream", nil)
	r.ServeHTTP(w, req)

	// Must fail with 502 Bad Gateway
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status 502 for upstream redirect interception, got %d", w.Code)
	}

	assertNoStoreHeaders(t, w)
}

func TestExpiredLinkErrorIsNotCacheable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/:prefix/:slug", ProxyHandler)

	expiredAt := time.Now().Add(-time.Hour)
	link := db.Link{
		Prefix:     "/s",
		Slug:       "expired",
		PublicPath: "/s/expired",
		TargetURL:  "http://example.com/file",
		Mode:       "proxy",
		Enabled:    true,
		ExpireTime: &expiredAt,
	}
	db.DB.Create(&link)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/s/expired", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("expected status 410 for expired link, got %d", w.Code)
	}

	assertNoStoreHeaders(t, w)
}

func assertNoStoreHeaders(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	if got := w.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate, max-age=0" {
		t.Errorf("expected no-store Cache-Control, got %q", got)
	}
	if got := w.Header().Get("Pragma"); got != "no-cache" {
		t.Errorf("expected no-cache Pragma, got %q", got)
	}
	if got := w.Header().Get("Expires"); got != "0" {
		t.Errorf("expected Expires 0, got %q", got)
	}
}
