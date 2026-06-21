package db

import (
	"log"
	"time"

	"sharelink/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Link struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	Prefix               string     `gorm:"not null;index:idx_links_prefix_slug,unique" json:"prefix"`
	Slug                 string     `gorm:"not null;index:idx_links_prefix_slug,unique" json:"slug"`
	PublicPath           string     `gorm:"not null;unique" json:"public_path"`
	TargetURL            string     `gorm:"not null" json:"target_url"`
	Mode                 string     `gorm:"not null;default:'proxy'" json:"mode"` // proxy or redirect
	Enabled              bool       `gorm:"not null;index:idx_links_enabled" json:"enabled"`
	StartTime            *time.Time `json:"start_time"`
	ExpireTime           *time.Time `json:"expire_time"`
	CacheEnabled         bool       `gorm:"not null;default:false" json:"cache_enabled"`
	CacheTTL             int        `gorm:"not null;default:600" json:"cache_ttl"`
	CacheMaxObjectSizeMB int        `gorm:"not null;default:5" json:"cache_max_object_size_mb"`
	FilenameMode         string     `gorm:"not null;default:'inherit'" json:"filename_mode"` // inherit, custom, auto
	CustomFilename       *string    `json:"custom_filename"`
	UAPolicyID           *uint      `json:"ua_policy_id"`
	Note                 *string    `json:"note"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type VisitLog struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	LinkID             *uint     `gorm:"index:idx_visit_logs_link_id" json:"link_id"`
	Prefix             string    `json:"prefix"`
	Slug               string    `json:"slug"`
	PublicPath         string    `gorm:"index:idx_visit_logs_public_path" json:"public_path"`
	IP                 string    `json:"ip"`
	IPHash             string    `gorm:"index:idx_visit_logs_ip_hash" json:"ip_hash"`
	VisitorHash        string    `json:"visitor_hash"`
	UserAgent          string    `json:"user_agent"`
	Referer            string    `json:"referer"`
	Country            string    `json:"country"`
	Region             string    `json:"region"`
	City               string    `json:"city"`
	AccessTime         time.Time `gorm:"index:idx_visit_logs_access_time" json:"access_time"`
	Mode               string    `json:"mode"`           // proxy, redirect, blocked
	Status             string    `json:"status"`         // success, expired, blocked, failed
	BlockedReason      string    `json:"blocked_reason"` // ua_blocked, ip_blocked, etc
	ResponseStatusCode int       `json:"response_status_code"`
	UpstreamStatusCode int       `json:"upstream_status_code"`
	ResponseSize       int64     `json:"response_size"`
	CacheStatus        string    `json:"cache_status"` // hit, miss, bypass, disabled
}

type UAPolicy struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	Mode          string    `gorm:"not null;default:'disabled'" json:"mode"` // disabled, whitelist, blacklist, mixed
	AllowKeywords string    `json:"allow_keywords"`                          // JSON array of strings
	BlockKeywords string    `json:"block_keywords"`                          // JSON array of strings
	AllowEmptyUA  bool      `gorm:"not null" json:"allow_empty_ua"`
	CaseSensitive bool      `gorm:"not null;default:false" json:"case_sensitive"`
	MatchType     string    `gorm:"not null;default:'contains'" json:"match_type"` // contains, regex
	Enabled       bool      `gorm:"not null" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GlobalSetting struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"not null" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

var DB *gorm.DB

func Init() {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBDSN), &gorm.Config{
		NowFunc: config.NowUTC,
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Run auto migrations
	err = DB.AutoMigrate(&Link{}, &VisitLog{}, &UAPolicy{}, &GlobalSetting{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Initialize admin password
	initAdminPassword()
}

func FindGlobalSetting(key string) (GlobalSetting, bool, error) {
	var setting GlobalSetting
	result := DB.Limit(1).Find(&setting, "key = ?", key)
	if result.Error != nil {
		return setting, false, result.Error
	}
	return setting, result.RowsAffected > 0, nil
}

func initAdminPassword() {
	if _, found, err := FindGlobalSetting("admin_password_hash"); err != nil {
		log.Fatalf("failed to query global settings: %v", err)
	} else if !found {
		// Password doesn't exist, check initial admin password env variable
		initialPassword := config.AppConfig.InitialAdminPassword
		if initialPassword == "" {
			log.Fatal("database contains no admin password and INITIAL_ADMIN_PASSWORD environment variable is empty. Refusing to start.")
		}

		// Generate bcrypt hash
		hash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("failed to hash initial admin password: %v", err)
		}

		newSetting := GlobalSetting{
			Key:       "admin_password_hash",
			Value:     string(hash),
			UpdatedAt: config.NowUTC(),
		}

		if err := DB.Create(&newSetting).Error; err != nil {
			log.Fatalf("failed to save admin password hash to database: %v", err)
		}
		log.Println("successfully initialized admin password hash in database")
	}
}
