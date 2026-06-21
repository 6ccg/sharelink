package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"sharelink/internal/auth"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"sharelink/internal/geoip"
	"sharelink/internal/links"
	"sharelink/internal/logs"
	"sharelink/internal/proxy"
	"sharelink/internal/settings"

	"github.com/gin-gonic/gin"
)

// devMode404HTML returns a minimal styled HTML 404 for dev-mode non-shortlink routes.
func devMode404HTML() string {
	return proxy.RenderPublicErrorHTML(404, "Not Found", "The requested resource does not exist.")
}

func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := parseAllowedOrigins(config.AppConfig.CORSAllowedOrigins)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Writer.Header().Add("Vary", "Origin")
			if allowedOrigins["*"] {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else if allowedOrigins[origin] {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func parseAllowedOrigins(raw string) map[string]bool {
	origins := make(map[string]bool)
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = true
		}
	}
	return origins
}

func main() {
	log.Println("Starting ShareLink Backend Server...")

	// 1. Load Configurations
	config.Load()

	// 2. Initialize Database & Run Migrations
	db.Init()

	// 3. Initialize GeoIP ip2region
	geoip.Init()

	// 4. Initialize Logs Workers
	logs.Init()

	// 5. Initialize Router
	if config.AppConfig.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())

	// 6. Setup Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth APIs
	r.POST("/api/auth/login", auth.LoginHandler)
	r.POST("/api/auth/logout", auth.LogoutHandler)

	// Admin APIs (Protected by JWT)
	admin := r.Group("/api/admin")
	admin.Use(auth.Middleware())
	{
		admin.GET("/auth/me", auth.MeHandler) // Me check

		// Links CRUD
		admin.GET("/links", links.ListLinks)
		admin.POST("/links", links.CreateLink)
		admin.GET("/links/:id", links.GetLink)
		admin.PUT("/links/:id", links.UpdateLink)
		admin.DELETE("/links/:id", links.DeleteLink)
		admin.POST("/links/:id/enable", links.EnableLink)
		admin.POST("/links/:id/disable", links.DisableLink)
		admin.POST("/links/generate-slug", links.GenerateSlug)

		// Visit Logs
		admin.GET("/logs", settings.ListLogs)

		// Analytics
		admin.GET("/analytics/overview", settings.GetAnalyticsOverview)
		admin.GET("/analytics/trend", settings.GetAnalyticsTrend)
		admin.GET("/analytics/geo", settings.GetAnalyticsGeo)
		admin.GET("/analytics/user-agents", settings.GetAnalyticsUserAgents)

		// UA Policies CRUD
		admin.GET("/ua-policies", settings.ListUAPolicies)
		admin.POST("/ua-policies", settings.CreateUAPolicy)
		admin.GET("/ua-policies/:id", settings.GetUAPolicy)
		admin.PUT("/ua-policies/:id", settings.UpdateUAPolicy)
		admin.DELETE("/ua-policies/:id", settings.DeleteUAPolicy)
		admin.POST("/ua-policies/test", settings.TestUAPolicy)

		// Cache Management
		admin.GET("/cache/status", settings.GetCacheStatus)
		admin.POST("/cache/clear", settings.ClearCache)
		admin.POST("/cache/clear-link/:id", settings.ClearLinkCache)

		// Dynamic Settings
		admin.GET("/settings", settings.GetSettings)
		admin.PUT("/settings", settings.UpdateSettings)
		admin.POST("/settings/password", settings.ChangePassword)
	}

	// 7. Serve Static Files (Vite frontend production build integration)
	// We check if static dist folders exist and mount them
	distDir := "./frontend/dist"
	if _, err := os.Stat(distDir); err != nil {
		if _, err := os.Stat("../frontend/dist"); err == nil {
			distDir = "../frontend/dist"
		}
	}
	if _, err := os.Stat(distDir); err == nil {
		r.StaticFS("/assets", http.Dir(distDir+"/assets"))
		serveStaticFileIfExists(r, "/favicon.ico", distDir+"/favicon.ico")
		serveStaticFileIfExists(r, "/favicon.svg", distDir+"/favicon.svg")
		serveStaticFileIfExists(r, "/icons.svg", distDir+"/icons.svg")

		// SPA Fallback for browser routes in frontend
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// If it's a 2-level path (prefix/slug), let the ProxyHandler try to resolve it first!
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) == 2 && !links.ReservedPrefixes["/"+parts[0]] {
				proxy.ProxyHandler(c)
				return
			}
			// Otherwise serve index.html for React SPA
			c.File(distDir + "/index.html")
		})
	} else {
		// Wildcard route for shortlinks (development mode fallback)
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) == 2 && !links.ReservedPrefixes["/"+parts[0]] {
				proxy.ProxyHandler(c)
				return
			}
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusNotFound, devMode404HTML())
		})
	}

	// 8. Run Server
	addr := fmt.Sprintf(":%s", config.AppConfig.Port)
	log.Printf("Server listening on http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func serveStaticFileIfExists(r *gin.Engine, relativePath, filePath string) {
	if _, err := os.Stat(filePath); err == nil {
		r.StaticFile(relativePath, filePath)
	}
}
