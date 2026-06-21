package links

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"sharelink/internal/cache"
	"sharelink/internal/db"

	"github.com/gin-gonic/gin"
)

var ReservedPrefixes = map[string]bool{
	"/admin":       true,
	"/api":         true,
	"/assets":      true,
	"/static":      true,
	"/login":       true,
	"/logout":      true,
	"/favicon.ico": true,
	"/favicon.svg": true,
	"/health":      true,
	"/icons.svg":   true,
}

var (
	prefixPattern = regexp.MustCompile(`^/[A-Za-z0-9][A-Za-z0-9._-]*$`)
	slugPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// GenerateRandomSlug creates a 10-character random slug
func GenerateRandomSlug() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	var sb strings.Builder
	for _, val := range b {
		sb.WriteByte(charset[int(val)%len(charset)])
	}
	return sb.String()
}

func normalizeLinkTimes(link *db.Link) {
	if link.StartTime != nil {
		t := link.StartTime.UTC()
		link.StartTime = &t
	}
	if link.ExpireTime != nil {
		t := link.ExpireTime.UTC()
		link.ExpireTime = &t
	}
}

// ListLinks handles GET /api/admin/links
func ListLinks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	mode := c.Query("mode")
	enabledStr := c.Query("enabled")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := db.DB.Model(&db.Link{})

	if keyword != "" {
		keywordPattern := "%" + keyword + "%"
		query = query.Where("prefix LIKE ? OR slug LIKE ? OR target_url LIKE ? OR note LIKE ?", keywordPattern, keywordPattern, keywordPattern, keywordPattern)
	}

	if mode != "" {
		query = query.Where("mode = ?", mode)
	}

	if enabledStr != "" {
		enabled, err := strconv.ParseBool(enabledStr)
		if err == nil {
			query = query.Where("enabled = ?", enabled)
		}
	}

	var total int64
	query.Count(&total)

	var list []db.Link
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&list)

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

// GetLink handles GET /api/admin/links/:id
func GetLink(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    link,
		"error":   nil,
	})
}

// CreateLink handles POST /api/admin/links
func CreateLink(c *gin.Context) {
	var link db.Link
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
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
	if err := json.Unmarshal(body, &link); err != nil {
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

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
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

	if _, ok := raw["enabled"]; !ok {
		link.Enabled = true
	}
	if link.Mode == "" {
		link.Mode = "proxy"
	}
	if link.CacheTTL == 0 {
		link.CacheTTL = 600
	}
	if link.CacheMaxObjectSizeMB == 0 {
		link.CacheMaxObjectSizeMB = 5
	}
	if link.FilenameMode == "" {
		link.FilenameMode = "inherit"
	}
	normalizeLinkTimes(&link)

	// Validate Prefix
	link.Prefix = strings.TrimSpace(link.Prefix)
	if link.Prefix == "" || !strings.HasPrefix(link.Prefix, "/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PREFIX",
				"message": "Prefix must start with '/'",
			},
		})
		return
	}
	if !prefixPattern.MatchString(link.Prefix) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PREFIX",
				"message": "Prefix may only contain letters, numbers, dot, underscore, and hyphen",
			},
		})
		return
	}
	if strings.Contains(link.Prefix[1:], "/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_PREFIX",
				"message": "Prefix must be a single-level path (e.g. /export)",
			},
		})
		return
	}
	if ReservedPrefixes[link.Prefix] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "RESERVED_PREFIX",
				"message": "Prefix is reserved for system use",
			},
		})
		return
	}

	// Validate Slug
	link.Slug = strings.TrimSpace(link.Slug)
	if link.Slug == "" {
		// Auto generate slug
		link.Slug = GenerateRandomSlug()
	}
	if strings.Contains(link.Slug, "/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_SLUG",
				"message": "Slug cannot contain '/'",
			},
		})
		return
	}
	if !slugPattern.MatchString(link.Slug) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_SLUG",
				"message": "Slug may only contain letters, numbers, dot, underscore, and hyphen",
			},
		})
		return
	}

	if !isValidMode(link.Mode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_MODE",
				"message": "Mode must be proxy or redirect",
			},
		})
		return
	}
	if !isValidFilenameMode(link.FilenameMode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_FILENAME_MODE",
				"message": "Filename mode must be inherit, custom, or auto",
			},
		})
		return
	}
	if link.CacheTTL < 0 || link.CacheMaxObjectSizeMB < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_CACHE_CONFIG",
				"message": "Cache TTL and max object size must not be negative",
			},
		})
		return
	}

	// Validate Target URL
	link.TargetURL = strings.TrimSpace(link.TargetURL)
	targetURL, err := url.Parse(link.TargetURL)
	if err != nil || targetURL.Scheme == "" || targetURL.Hostname() == "" || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_TARGET_URL",
				"message": "Target URL must be a valid HTTP or HTTPS link",
			},
		})
		return
	}

	// Construct public_path
	link.PublicPath = link.Prefix + "/" + link.Slug

	// Check duplicates
	var count int64
	db.DB.Model(&db.Link{}).Where("prefix = ? AND slug = ?", link.Prefix, link.Slug).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "CONFLICT",
				"message": "The combination of prefix and slug already exists",
			},
		})
		return
	}

	// Save to DB
	if err := db.DB.Select("*").Create(&link).Error; err != nil {
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
		"data":    link,
		"error":   nil,
	})
}

// UpdateLink handles PUT /api/admin/links/:id
func UpdateLink(c *gin.Context) {
	id := c.Param("id")
	var existing db.Link
	if err := db.DB.First(&existing, id).Error; err != nil {
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

	var req db.Link
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

	// Target URL validation
	req.TargetURL = strings.TrimSpace(req.TargetURL)
	targetURL, err := url.Parse(req.TargetURL)
	if err != nil || targetURL.Scheme == "" || targetURL.Hostname() == "" || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_TARGET_URL",
				"message": "Target URL must be a valid HTTP or HTTPS link",
			},
		})
		return
	}

	if !isValidMode(req.Mode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_MODE",
				"message": "Mode must be proxy or redirect",
			},
		})
		return
	}
	if !isValidFilenameMode(req.FilenameMode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_FILENAME_MODE",
				"message": "Filename mode must be inherit, custom, or auto",
			},
		})
		return
	}
	if req.CacheTTL < 0 || req.CacheMaxObjectSizeMB < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_CACHE_CONFIG",
				"message": "Cache TTL and max object size must not be negative",
			},
		})
		return
	}

	// Update allowed fields (we do not allow modifying prefix and slug after creation to prevent routing breaks)
	existing.TargetURL = req.TargetURL
	existing.Mode = req.Mode
	existing.Enabled = req.Enabled
	existing.StartTime = req.StartTime
	existing.ExpireTime = req.ExpireTime
	normalizeLinkTimes(&existing)
	existing.CacheEnabled = req.CacheEnabled
	existing.CacheTTL = req.CacheTTL
	existing.CacheMaxObjectSizeMB = req.CacheMaxObjectSizeMB
	existing.FilenameMode = req.FilenameMode
	existing.CustomFilename = req.CustomFilename
	existing.UAPolicyID = req.UAPolicyID
	existing.Note = req.Note

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

	// Clear cache for this link on modification
	cache.GetGlobalCache().ClearLinkCache(existing.Prefix, existing.Slug)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    existing,
		"error":   nil,
	})
}

func isValidMode(mode string) bool {
	return mode == "proxy" || mode == "redirect"
}

func isValidFilenameMode(mode string) bool {
	return mode == "inherit" || mode == "custom" || mode == "auto"
}

// DeleteLink handles DELETE /api/admin/links/:id
func DeleteLink(c *gin.Context) {
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

	// Clear cache for this link on deletion
	cache.GetGlobalCache().ClearLinkCache(link.Prefix, link.Slug)

	if err := db.DB.Delete(&link).Error; err != nil {
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
		"data":    gin.H{"message": "Link successfully deleted"},
		"error":   nil,
	})
}

// EnableLink handles POST /api/admin/links/:id/enable
func EnableLink(c *gin.Context) {
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

	link.Enabled = true
	db.DB.Save(&link)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    link,
		"error":   nil,
	})
}

// DisableLink handles POST /api/admin/links/:id/disable
func DisableLink(c *gin.Context) {
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

	link.Enabled = false
	db.DB.Save(&link)

	// Clear cache on disable
	cache.GetGlobalCache().ClearLinkCache(link.Prefix, link.Slug)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    link,
		"error":   nil,
	})
}

// GenerateSlug handles POST /api/admin/links/generate-slug
func GenerateSlug(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"slug": GenerateRandomSlug()},
		"error":   nil,
	})
}
