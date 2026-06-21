package settings

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sharelink/internal/cache"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"sharelink/internal/geoip"
	"sharelink/internal/ua"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var BootTime time.Time

func init() {
	BootTime = config.NowUTC()
}

var defaultSettings = map[string]string{
	"global_cache_enabled":             "true",
	"global_cache_max_memory_mb":       "64",
	"default_cache_ttl":                "600",
	"default_cache_max_object_size_mb": "5",
	"max_proxy_response_size_mb":       "5",
	"upstream_connect_timeout":         "10",
	"upstream_response_header_timeout": "15",
	"proxy_total_timeout":              "60",
	"log_cleanup_enabled":              "true",
	"log_retention_days":               "90",
	"geoip_enabled":                    "true",
	"trust_proxy_headers":              "false",
}

var allowedSettings = map[string]string{
	"global_cache_enabled":             "bool",
	"global_cache_max_memory_mb":       "positive_int",
	"default_cache_ttl":                "positive_int",
	"default_cache_max_object_size_mb": "positive_int",
	"max_proxy_response_size_mb":       "positive_int",
	"upstream_connect_timeout":         "positive_int",
	"upstream_response_header_timeout": "positive_int",
	"proxy_total_timeout":              "positive_int",
	"log_cleanup_enabled":              "bool",
	"log_retention_days":               "positive_int",
	"geoip_enabled":                    "bool",
	"trust_proxy_headers":              "bool",
	"global_ua_policy_id":              "uint",
}

func isValidSettingValue(kind, value string) bool {
	switch kind {
	case "bool":
		return value == "true" || value == "false" || value == "1" || value == "0"
	case "positive_int":
		parsed, err := strconv.Atoi(value)
		return err == nil && parsed > 0
	case "uint":
		if value == "" {
			return true
		}
		parsed, err := strconv.ParseUint(value, 10, 64)
		return err == nil && parsed > 0
	default:
		return false
	}
}

// -------------------------------------------------------------
// Settings Handlers
// -------------------------------------------------------------

// GetSettings handles GET /api/admin/settings
func GetSettings(c *gin.Context) {
	var list []db.GlobalSetting
	db.DB.Find(&list)

	res := make(map[string]string)
	for k, v := range defaultSettings {
		res[k] = v
	}
	for _, s := range list {
		// Do not return password hash
		if s.Key != "admin_password_hash" {
			res[s.Key] = s.Value
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
		"error":   nil,
	})
}

// UpdateSettings handles PUT /api/admin/settings
func UpdateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	// Save all key-values except password hash
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for k, v := range req {
			if k == "admin_password_hash" {
				continue
			}
			if kind, ok := allowedSettings[k]; !ok {
				return gorm.ErrInvalidData
			} else if !isValidSettingValue(kind, v) {
				return gorm.ErrInvalidData
			}
			setting := db.GlobalSetting{
				Key:       k,
				Value:     v,
				UpdatedAt: config.NowUTC(),
			}
			if err := tx.Save(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		status := http.StatusInternalServerError
		code := "DATABASE_ERROR"
		message := err.Error()
		if errors.Is(err, gorm.ErrInvalidData) {
			status = http.StatusBadRequest
			code = "INVALID_SETTING"
			message = "Invalid or unsupported setting value"
		}
		c.JSON(status, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    code,
				"message": message,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Settings updated successfully"},
		"error":   nil,
	})
}

// ChangePassword handles POST /api/admin/settings/password
func ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	// Fetch old hash
	setting, found, err := db.FindGlobalSetting("admin_password_hash")
	if err != nil || !found {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Admin password setting missing",
			},
		})
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(setting.Value), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Incorrect old password",
			},
		})
		return
	}

	// Generate new hash
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SYSTEM_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	setting.Value = string(newHash)
	setting.UpdatedAt = config.NowUTC()
	db.DB.Save(&setting)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Password changed successfully"},
		"error":   nil,
	})
}

// -------------------------------------------------------------
// Cache Handlers
// -------------------------------------------------------------

// GetCacheStatus handles GET /api/admin/cache/status
func GetCacheStatus(c *gin.Context) {
	count, currentBytes, maxBytes := cache.GetGlobalCache().GetStatus()

	// Query hits and misses from visit logs to compute rate
	var hits, misses int64
	db.DB.Model(&db.VisitLog{}).Where("cache_status = ?", "hit").Count(&hits)
	db.DB.Model(&db.VisitLog{}).Where("cache_status = ?", "miss").Count(&misses)

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"count":             count,
			"memory_used_bytes": currentBytes,
			"memory_max_bytes":  maxBytes,
			"hits":              hits,
			"misses":            misses,
			"hit_rate_percent":  hitRate,
			"enabled":           cache.IsGlobalCacheEnabled(),
		},
		"error": nil,
	})
}

// ClearCache handles POST /api/admin/cache/clear
func ClearCache(c *gin.Context) {
	cache.GetGlobalCache().ClearAll()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Cache successfully cleared"},
		"error":   nil,
	})
}

// ClearLinkCache handles POST /api/admin/cache/clear-link/:id
func ClearLinkCache(c *gin.Context) {
	id := c.Param("id")
	var link db.Link
	if err := db.DB.First(&link, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Link not found",
			},
		})
		return
	}

	cache.GetGlobalCache().ClearLinkCache(link.Prefix, link.Slug)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Link cache successfully cleared"},
		"error":   nil,
	})
}

// -------------------------------------------------------------
// User-Agent Policy Handlers
// -------------------------------------------------------------

// ListUAPolicies handles GET /api/admin/ua-policies
func ListUAPolicies(c *gin.Context) {
	var list []db.UAPolicy
	db.DB.Find(&list)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
		"error":   nil,
	})
}

// GetUAPolicy handles GET /api/admin/ua-policies/:id
func GetUAPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy db.UAPolicy
	if err := db.DB.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Policy not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    policy,
		"error":   nil,
	})
}

// CreateUAPolicy handles POST /api/admin/ua-policies
func CreateUAPolicy(c *gin.Context) {
	var policy db.UAPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	normalizeUAPolicy(&policy)
	if err := validateUAPolicy(policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_UA_POLICY",
				"message": err.Error(),
			},
		})
		return
	}

	if err := db.DB.Select("*").Create(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    policy,
		"error":   nil,
	})
}

// UpdateUAPolicy handles PUT /api/admin/ua-policies/:id
func UpdateUAPolicy(c *gin.Context) {
	id := c.Param("id")
	var existing db.UAPolicy
	if err := db.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Policy not found",
			},
		})
		return
	}

	var req db.UAPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	normalizeUAPolicy(&req)
	if err := validateUAPolicy(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_UA_POLICY",
				"message": err.Error(),
			},
		})
		return
	}

	existing.Name = req.Name
	existing.Mode = req.Mode
	existing.AllowKeywords = req.AllowKeywords
	existing.BlockKeywords = req.BlockKeywords
	existing.AllowEmptyUA = req.AllowEmptyUA
	existing.CaseSensitive = req.CaseSensitive
	existing.MatchType = req.MatchType
	existing.Enabled = req.Enabled

	if err := db.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    existing,
		"error":   nil,
	})
}

func normalizeUAPolicy(policy *db.UAPolicy) {
	policy.Name = strings.TrimSpace(policy.Name)
	if policy.Mode == "" {
		policy.Mode = "disabled"
	}
	if policy.MatchType == "" {
		policy.MatchType = "contains"
	}
}

func validateUAPolicy(policy db.UAPolicy) error {
	if policy.Name == "" {
		return errors.New("policy name is required")
	}
	switch policy.Mode {
	case "disabled", "whitelist", "blacklist", "mixed":
	default:
		return errors.New("invalid UA policy mode")
	}
	switch policy.MatchType {
	case "contains", "regex":
	default:
		return errors.New("invalid UA match type")
	}
	if !isJSONArray(policy.AllowKeywords) || !isJSONArray(policy.BlockKeywords) {
		return errors.New("allow_keywords and block_keywords must be JSON arrays")
	}
	return nil
}

func isJSONArray(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	var items []string
	return json.Unmarshal([]byte(value), &items) == nil
}

// DeleteUAPolicy handles DELETE /api/admin/ua-policies/:id
func DeleteUAPolicy(c *gin.Context) {
	id := c.Param("id")
	var policy db.UAPolicy
	if err := db.DB.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Policy not found",
			},
		})
		return
	}

	// Set links referencing this policy to NULL first
	db.DB.Model(&db.Link{}).Where("ua_policy_id = ?", policy.ID).Update("ua_policy_id", nil)

	if err := db.DB.Delete(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Policy successfully deleted"},
		"error":   nil,
	})
}

// TestUAPolicy handles POST /api/admin/ua-policies/test
func TestUAPolicy(c *gin.Context) {
	var req struct {
		PolicyID  uint   `json:"policy_id" binding:"required"`
		UserAgent string `json:"user_agent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	var policy db.UAPolicy
	if err := db.DB.First(&policy, req.PolicyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Policy not found",
			},
		})
		return
	}

	allowed, reason := ua.ValidateUA(req.UserAgent, &policy)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"allowed":        allowed,
			"blocked_reason": reason,
		},
		"error": nil,
	})
}

// -------------------------------------------------------------
// Visit Logs Handlers
// -------------------------------------------------------------

// ListLogs handles GET /api/admin/logs
func ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "15"))
	linkIDStr := c.Query("link_id")
	prefix := c.Query("prefix")
	slug := c.Query("slug")
	ip := c.Query("ip")
	country := c.Query("country")
	status := c.Query("status")
	mode := c.Query("mode")
	cacheStatus := c.Query("cache_status")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 15
	}

	query := db.DB.Model(&db.VisitLog{})

	if linkIDStr != "" {
		if linkID, err := strconv.Atoi(linkIDStr); err == nil {
			query = query.Where("link_id = ?", linkID)
		}
	}
	if prefix != "" {
		query = query.Where("prefix = ?", prefix)
	}
	if slug != "" {
		query = query.Where("slug = ?", slug)
	}
	if ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if country != "" {
		query = query.Where("country = ?", country)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if mode != "" {
		query = query.Where("mode = ?", mode)
	}
	if cacheStatus != "" {
		query = query.Where("cache_status = ?", cacheStatus)
	}

	if startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query = query.Where("access_time >= ?", t.UTC())
		}
	}
	if endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query = query.Where("access_time <= ?", t.UTC())
		}
	}

	if keyword != "" {
		keywordPattern := "%" + keyword + "%"
		query = query.Where("public_path LIKE ? OR ip LIKE ? OR user_agent LIKE ? OR referer LIKE ?", keywordPattern, keywordPattern, keywordPattern, keywordPattern)
	}

	var total int64
	query.Count(&total)

	var list []db.VisitLog
	offset := (page - 1) * pageSize
	query.Order("access_time DESC").Offset(offset).Limit(pageSize).Find(&list)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"items":     list,
		},
		"error": nil,
	})
}

// -------------------------------------------------------------
// Analytics Handlers
// -------------------------------------------------------------

// GetAnalyticsOverview handles GET /api/admin/analytics/overview
func GetAnalyticsOverview(c *gin.Context) {
	// Total Links Count
	var totalLinks, activeLinks int64
	db.DB.Model(&db.Link{}).Count(&totalLinks)
	db.DB.Model(&db.Link{}).Where("enabled = ?", true).Count(&activeLinks)

	// Total PV (all successful visits)
	var totalPV int64
	db.DB.Model(&db.VisitLog{}).Where("status = ?", "success").Count(&totalPV)

	// Total UV
	var totalUV int64
	db.DB.Model(&db.VisitLog{}).Select("COUNT(DISTINCT visitor_hash)").Row().Scan(&totalUV)

	// Today start boundary
	todayStart := config.BusinessDayStartUTC(config.NowUTC())

	// Today PV
	var todayPV int64
	db.DB.Model(&db.VisitLog{}).Where("status = ? AND access_time >= ?", "success", todayStart).Count(&todayPV)

	// Today UV
	var todayUV int64
	db.DB.Model(&db.VisitLog{}).Where("access_time >= ?", todayStart).Select("COUNT(DISTINCT visitor_hash)").Row().Scan(&todayUV)

	// Cache stats
	count, memoryUsedBytes, memoryMaxBytes := cache.GetGlobalCache().GetStatus()
	var hits, misses int64
	db.DB.Model(&db.VisitLog{}).Where("cache_status = ?", "hit").Count(&hits)
	db.DB.Model(&db.VisitLog{}).Where("cache_status = ?", "miss").Count(&misses)

	var hitRate float64
	totalCacheQueries := hits + misses
	if totalCacheQueries > 0 {
		hitRate = float64(hits) / float64(totalCacheQueries) * 100
	}

	// GeoIP status
	geoipEnabled := geoip.IsEnabled()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_links":      totalLinks,
			"active_links":     activeLinks,
			"total_pv":         totalPV,
			"total_uv":         totalUV,
			"today_pv":         todayPV,
			"today_uv":         todayUV,
			"cache_objects":    count,
			"cache_used_bytes": memoryUsedBytes,
			"cache_max_bytes":  memoryMaxBytes,
			"cache_hit_rate":   hitRate,
			"geoip_enabled":    geoipEnabled,
			"geoip_db_path":    config.AppConfig.IPDBPath,
			"uptime_seconds":   int64(time.Since(BootTime).Seconds()),
		},
		"error": nil,
	})
}

// GetAnalyticsTrend handles GET /api/admin/analytics/trend
func GetAnalyticsTrend(c *gin.Context) {
	// Query daily trend for the last 15 days
	loc := config.Location()
	todayLocal := config.NowUTC().In(loc)
	startLocal := time.Date(todayLocal.Year(), todayLocal.Month(), todayLocal.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -14)
	startUTC := startLocal.UTC()

	type TrendItem struct {
		Date string `json:"date"`
		PV   int64  `json:"pv"`
		UV   int64  `json:"uv"`
	}

	var logs []db.VisitLog
	err := db.DB.
		Select("access_time, visitor_hash").
		Where("access_time >= ? AND status = ?", startUTC, "success").
		Order("access_time ASC").
		Find(&logs).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	type trendAgg struct {
		pv int64
		uv map[string]struct{}
	}

	grouped := make(map[string]*trendAgg)
	for _, logItem := range logs {
		date := config.BusinessDate(logItem.AccessTime)
		agg := grouped[date]
		if agg == nil {
			agg = &trendAgg{uv: make(map[string]struct{})}
			grouped[date] = agg
		}
		agg.pv++
		if logItem.VisitorHash != "" {
			agg.uv[logItem.VisitorHash] = struct{}{}
		}
	}

	results := make([]TrendItem, 0, len(grouped))
	for day := startLocal; !day.After(todayLocal); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		agg := grouped[date]
		if agg == nil {
			continue
		}
		results = append(results, TrendItem{
			Date: date,
			PV:   agg.pv,
			UV:   int64(len(agg.uv)),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"error":   nil,
	})
}

// GetAnalyticsGeo handles GET /api/admin/analytics/geo
func GetAnalyticsGeo(c *gin.Context) {
	type GeoItem struct {
		Country  string `json:"country"`
		Region   string `json:"region"`
		City     string `json:"city"`
		Requests int64  `json:"requests"`
		UV       int64  `json:"uv"`
		IPCount  int64  `json:"ip_count"`
	}

	var results []GeoItem
	err := db.DB.Model(&db.VisitLog{}).
		Select("country, region, city, COUNT(*) as requests, COUNT(DISTINCT visitor_hash) as uv, COUNT(DISTINCT ip) as ip_count").
		Where("status = ?", "success").
		Group("country, region, city").
		Order("requests DESC").
		Limit(20).
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"error":   nil,
	})
}

// GetAnalyticsUserAgents handles GET /api/admin/analytics/user-agents
func GetAnalyticsUserAgents(c *gin.Context) {
	type UAItem struct {
		UserAgent string `json:"user_agent"`
		PV        int64  `json:"pv"`
	}

	var results []UAItem
	err := db.DB.Model(&db.VisitLog{}).
		Select("user_agent, COUNT(*) as pv").
		Where("status = ?", "success").
		Group("user_agent").
		Order("pv DESC").
		Limit(15).
		Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"error":   nil,
	})
}
