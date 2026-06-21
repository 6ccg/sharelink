package config

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Port                 string
	DBType               string
	DBDSN                string
	InitialAdminPassword string
	DataDir              string
	IPDBPath             string
	LogLevel             string
	JWTSecret            string
	CORSAllowedOrigins   string
	AppTimezone          string
	AppLocation          *time.Location
}

var AppConfig *Config

func Load() {
	port := getEnv("PORT", "8080")
	dbType := getEnv("DB_TYPE", "sqlite")
	dataDir := getEnv("DATA_DIR", ".")

	// Ensure data directory exists
	if dataDir != "." && dataDir != "" {
		_ = os.MkdirAll(dataDir, 0755)
	}

	defaultDSN := filepath.Join(dataDir, "sharelink.db")
	dbDSN := getEnv("DB_DSN", defaultDSN)

	// If IP_DB_PATH is not set, default to dataDir/ip2region.xdb or data/ip2region.xdb
	defaultIPDBPath := filepath.Join(dataDir, "data", "ip2region.xdb")
	if _, err := os.Stat(defaultIPDBPath); os.IsNotExist(err) {
		// fallback to local data/ip2region.xdb
		defaultIPDBPath = filepath.Join("data", "ip2region.xdb")
	}
	ipDBPath := getEnv("IP_DB_PATH", defaultIPDBPath)

	logLevel := getEnv("LOG_LEVEL", "info")
	initialAdminPassword := os.Getenv("INITIAL_ADMIN_PASSWORD")
	jwtSecret := getEnv("JWT_SECRET", "sharelink-session-secret-key-32-chars")
	corsAllowedOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	appTimezone := getEnv("APP_TIMEZONE", "Asia/Shanghai")
	appLocation, err := time.LoadLocation(appTimezone)
	if err != nil {
		log.Printf("invalid APP_TIMEZONE %q, falling back to UTC: %v", appTimezone, err)
		appTimezone = "UTC"
		appLocation = time.UTC
	}

	AppConfig = &Config{
		Port:                 port,
		DBType:               dbType,
		DBDSN:                dbDSN,
		InitialAdminPassword: initialAdminPassword,
		DataDir:              dataDir,
		IPDBPath:             ipDBPath,
		LogLevel:             logLevel,
		JWTSecret:            jwtSecret,
		CORSAllowedOrigins:   corsAllowedOrigins,
		AppTimezone:          appTimezone,
		AppLocation:          appLocation,
	}
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func Location() *time.Location {
	if AppConfig == nil || AppConfig.AppLocation == nil {
		return time.UTC
	}
	return AppConfig.AppLocation
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func BusinessDate(t time.Time) string {
	return t.In(Location()).Format("2006-01-02")
}

func BusinessDayStartUTC(t time.Time) time.Time {
	local := t.In(Location())
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, Location())
	return start.UTC()
}
