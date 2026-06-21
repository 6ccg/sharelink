package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuth(t *testing.T) {
	// Initialize test database
	os.Setenv("INITIAL_ADMIN_PASSWORD", "testpass123")
	os.Setenv("DB_DSN", "file::memory:?cache=shared")
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")
	defer os.Unsetenv("DB_DSN")

	config.Load()
	db.Init()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Authentication API endpoints
	r.POST("/api/auth/login", LoginHandler)
	r.POST("/api/auth/logout", LogoutHandler)

	// Protected endpoint group
	protected := r.Group("/api/admin")
	protected.Use(Middleware())
	protected.GET("/me", MeHandler)

	// Test 1: Unauthorized access to /api/admin/me
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// Test 2: Login with wrong password
	loginReqWrong := LoginRequest{
		Username: "url",
		Password: "wrongpassword",
	}
	bodyWrong, _ := json.Marshal(loginReqWrong)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(bodyWrong))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d on wrong password, got %d", http.StatusUnauthorized, w.Code)
	}

	// Test 3: Login with correct password
	loginReqCorrect := LoginRequest{
		Username: "url",
		Password: "testpass123",
	}
	bodyCorrect, _ := json.Marshal(loginReqCorrect)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(bodyCorrect))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d on correct password, got %d", http.StatusOK, w.Code)
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	if !loginResp.Success || loginResp.Data.Token == "" {
		t.Fatalf("expected successful login with non-empty token")
	}

	// Test 4: Access protected endpoint with valid token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/admin/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Data.Token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d on valid token, got %d", http.StatusOK, w.Code)
	}

	var meResp struct {
		Success bool `json:"success"`
		Data    struct {
			Username string `json:"username"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &meResp)
	if meResp.Data.Username != "url" {
		t.Errorf("expected username 'url', got '%s'", meResp.Data.Username)
	}
}
