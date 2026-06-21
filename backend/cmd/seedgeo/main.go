package main

import (
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type VisitLog struct {
	ID                 uint      `gorm:"primaryKey"`
	LinkID             *uint     `gorm:"index:idx_visit_logs_link_id"`
	Prefix             string
	Slug               string
	PublicPath         string `gorm:"index:idx_visit_logs_public_path"`
	IP                 string
	IPHash             string `gorm:"index:idx_visit_logs_ip_hash"`
	VisitorHash        string
	UserAgent          string
	Referer            string
	Country            string
	Region             string
	City               string
	AccessTime         time.Time `gorm:"index:idx_visit_logs_access_time"`
	Mode               string
	Status             string
	BlockedReason      string
	ResponseStatusCode int
	UpstreamStatusCode int
	ResponseSize       int64
	CacheStatus        string
}

type GlobalSetting struct {
	Key       string    `gorm:"primaryKey"`
	Value     string    `gorm:"not null"`
	UpdatedAt time.Time
}

type seedGeo struct {
	Country string
	Region  string
	City    string
	PV      int
}

func main() {
	db, err := gorm.Open(sqlite.Open("sharelink.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	if err := db.Where("public_path = ?", "/__seed/geo-dashboard").Delete(&VisitLog{}).Error; err != nil {
		log.Fatalf("clear old seed rows: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash test password: %v", err)
	}
	if err := db.Save(&GlobalSetting{
		Key:       "admin_password_hash",
		Value:     string(hash),
		UpdatedAt: time.Now().UTC(),
	}).Error; err != nil {
		log.Fatalf("reset test password: %v", err)
	}

	rows := []seedGeo{
		{Country: "中国", Region: "广东", City: "广州", PV: 60},
		{Country: "中国", Region: "浙江", City: "杭州", PV: 42},
		{Country: "中国", Region: "北京", City: "北京", PV: 36},
		{Country: "中国", Region: "上海", City: "上海", PV: 31},
		{Country: "中国", Region: "四川", City: "成都", PV: 24},
		{Country: "中国", Region: "湖北", City: "武汉", PV: 18},
		{Country: "中国", Region: "新疆", City: "乌鲁木齐", PV: 12},
		{Country: "美国", Region: "California", City: "Los Angeles", PV: 28},
		{Country: "日本", Region: "Tokyo", City: "Tokyo", PV: 20},
		{Country: "德国", Region: "Hesse", City: "Frankfurt", PV: 15},
		{Country: "新加坡", Region: "Singapore", City: "Singapore", PV: 10},
	}

	now := time.Now().UTC()
	var logs []VisitLog
	for rowIndex, row := range rows {
		for i := 0; i < row.PV; i++ {
			logs = append(logs, VisitLog{
				Prefix:             "seed",
				Slug:               "geo-dashboard",
				PublicPath:         "/__seed/geo-dashboard",
				IP:                 "203.0.113.10",
				IPHash:             "seed-ip",
				VisitorHash:        "seed-visitor",
				UserAgent:          "ShareLink Geo Dashboard Seed",
				Referer:            "local-seed",
				Country:            row.Country,
				Region:             row.Region,
				City:               row.City,
				AccessTime:         now.Add(-time.Duration(rowIndex*row.PV+i) * time.Minute),
				Mode:               "redirect",
				Status:             "success",
				ResponseStatusCode: 200,
				UpstreamStatusCode: 200,
				ResponseSize:       1024,
				CacheStatus:        "disabled",
			})
		}
	}

	if err := db.CreateInBatches(logs, 100).Error; err != nil {
		log.Fatalf("insert seed rows: %v", err)
	}

	log.Printf("reset admin password to test-password and inserted %d geo dashboard seed visit logs", len(logs))
}
