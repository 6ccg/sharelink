package proxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"sharelink/internal/cache"
	"sharelink/internal/config"
	"sharelink/internal/db"
	"sharelink/internal/geoip"
	"sharelink/internal/logs"
	"sharelink/internal/requestip"
	"sharelink/internal/security"
	"sharelink/internal/ua"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProxyHandler handles both redirection and reverse proxying for public paths
func ProxyHandler(c *gin.Context) {
	prefix, slug := resolvePublicPath(c)

	// 1. Query the Link
	var link db.Link
	err := db.DB.First(&link, "prefix = ? AND slug = ?", prefix, slug).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			serveFriendlyError(c, http.StatusNotFound, "Link Not Found", "The requested link does not exist.")
			return
		}
		serveFriendlyError(c, http.StatusInternalServerError, "Database Error", "Failed to retrieve link configuration.")
		return
	}

	// 2. Validate availability
	if !link.Enabled {
		logAccessFailure(c, &link, "blocked", "link_disabled", http.StatusForbidden)
		serveFriendlyError(c, http.StatusForbidden, "Link Disabled", "This link has been disabled by the administrator.")
		return
	}

	now := config.NowUTC()
	if link.StartTime != nil && now.Before(*link.StartTime) {
		logAccessFailure(c, &link, "blocked", "not_active", http.StatusForbidden)
		serveFriendlyError(c, http.StatusForbidden, "Link Not Active Yet", "This link is not yet active.")
		return
	}

	if link.ExpireTime != nil && now.After(*link.ExpireTime) {
		logAccessFailure(c, &link, "expired", "expired", http.StatusGone)
		serveFriendlyError(c, http.StatusGone, "Link Expired", "This link has expired and is no longer available.")
		return
	}

	// 3. User-Agent Validation
	var policy *db.UAPolicy
	if link.UAPolicyID != nil {
		var p db.UAPolicy
		if err := db.DB.First(&p, "id = ?", *link.UAPolicyID).Error; err == nil {
			policy = &p
		}
	}
	if policy == nil {
		policy = ua.GetGlobalUAPolicy()
	}

	clientUA := c.GetHeader("User-Agent")
	if policy != nil {
		allowed, reason := ua.ValidateUA(clientUA, policy)
		if !allowed {
			// Log the blocked attempt
			logAccessFailure(c, &link, "blocked", reason, http.StatusForbidden)
			serveFriendlyError(c, http.StatusForbidden, "Access Denied", "Your client is not allowed to access this link.")
			return
		}
	}

	// 4. Construct base visit log
	remoteIP := requestip.ClientIP(c)
	geo := geoip.ResolveIP(remoteIP)
	ipHash := sha256Sum(remoteIP)
	visitorHash := sha256Sum(remoteIP + clientUA + config.BusinessDate(now))

	vlog := &db.VisitLog{
		LinkID:      &link.ID,
		Prefix:      link.Prefix,
		Slug:        link.Slug,
		PublicPath:  link.PublicPath,
		IP:          remoteIP,
		IPHash:      ipHash,
		VisitorHash: visitorHash,
		UserAgent:   clientUA,
		Referer:     c.GetHeader("Referer"),
		Country:     geo.Country,
		Region:      geo.Region,
		City:        geo.City,
		AccessTime:  now,
		Mode:        link.Mode,
		Status:      "success",
		CacheStatus: "disabled",
	}

	// 5. Handle Redirect Mode
	if link.Mode == "redirect" {
		targetURL, err := url.Parse(link.TargetURL)
		if err != nil || targetURL.Hostname() == "" {
			vlog.Status = "failed"
			vlog.BlockedReason = "invalid_target_url"
			vlog.ResponseStatusCode = http.StatusBadRequest
			logs.QueueLog(vlog)
			serveFriendlyError(c, http.StatusBadRequest, "Invalid Target URL", "The configured destination URL is invalid.")
			return
		}
		if _, err := security.CheckSSRF(c.Request.Context(), targetURL.Hostname()); err != nil {
			vlog.Status = "blocked"
			vlog.BlockedReason = "ssrf_blocked"
			vlog.ResponseStatusCode = http.StatusForbidden
			logs.QueueLog(vlog)
			serveFriendlyError(c, http.StatusForbidden, "Access Denied", "The configured destination is not allowed.")
			return
		}
		vlog.ResponseStatusCode = http.StatusFound
		logs.QueueLog(vlog)
		c.Redirect(http.StatusFound, link.TargetURL)
		return
	}

	// 6. Handle Proxy Mode
	if link.Mode == "proxy" {
		// Verify dynamic global settings
		maxRespSizeMB := getGlobalSettingInt("max_proxy_response_size_mb", 5)
		if link.CacheMaxObjectSizeMB == 0 {
			link.CacheMaxObjectSizeMB = int(maxRespSizeMB)
		}

		// Check cache if GET request
		cacheKey := generateCacheKey(c.Request, &link)
		isGET := c.Request.Method == http.MethodGet

		if isGET && link.CacheEnabled && cache.IsGlobalCacheEnabled() {
			vlog.CacheStatus = "miss"
			ramCache := cache.GetGlobalCache()
			if cachedItem, found := ramCache.Get(cacheKey); found {
				vlog.CacheStatus = "hit"
				vlog.ResponseStatusCode = cachedItem.StatusCode
				vlog.ResponseSize = cachedItem.Size
				logs.QueueLog(vlog)

				// Serve from cache
				for k, vals := range cachedItem.Headers {
					for _, v := range vals {
						c.Writer.Header().Add(k, v)
					}
				}
				c.Writer.WriteHeader(cachedItem.StatusCode)
				_, _ = c.Writer.Write(cachedItem.Body)
				return
			}
		} else if isGET && link.CacheEnabled {
			vlog.CacheStatus = "bypass"
		}

		// Run Reverse Proxy
		runReverseProxy(c, &link, vlog, cacheKey)
		return
	}

	serveFriendlyError(c, http.StatusBadRequest, "Invalid Mode", "Unknown link routing mode.")
}

func resolvePublicPath(c *gin.Context) (string, string) {
	prefixParam := c.Param("prefix")
	slugParam := c.Param("slug")
	if prefixParam != "" || slugParam != "" {
		return "/" + strings.TrimPrefix(prefixParam, "/"), slugParam
	}

	parts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	if len(parts) != 2 {
		return "/", ""
	}
	return "/" + parts[0], parts[1]
}

func runReverseProxy(c *gin.Context, link *db.Link, vlog *db.VisitLog, cacheKey string) {
	targetURL, err := url.Parse(link.TargetURL)
	if err != nil {
		vlog.Status = "failed"
		vlog.BlockedReason = "invalid_target_url"
		vlog.ResponseStatusCode = http.StatusBadRequest
		logs.QueueLog(vlog)
		serveFriendlyError(c, http.StatusBadRequest, "Invalid Target URL", "The configured destination URL is invalid.")
		return
	}

	// Enforce HTTP/HTTPS only
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		vlog.Status = "failed"
		vlog.BlockedReason = "unsupported_scheme"
		vlog.ResponseStatusCode = http.StatusBadRequest
		logs.QueueLog(vlog)
		serveFriendlyError(c, http.StatusBadRequest, "Protocol Blocked", "Only HTTP and HTTPS proxy targets are allowed.")
		return
	}

	// Retrieve timeouts
	connTimeout := time.Duration(getGlobalSettingInt("upstream_connect_timeout", 10)) * time.Second
	respHeaderTimeout := time.Duration(getGlobalSettingInt("upstream_response_header_timeout", 15)) * time.Second
	totalTimeout := time.Duration(getGlobalSettingInt("proxy_total_timeout", 60)) * time.Second

	// Setup custom dialer to prevent SSRF and DNS Rebinding
	dialer := &net.Dialer{
		Timeout:   connTimeout,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Validate IP ranges through DNS resolution
			ips, err := security.CheckSSRF(ctx, host)
			if err != nil {
				return nil, err
			}

			// Connect directly to the checked IP to prevent DNS rebinding
			targetIP := ips[0]
			targetAddr := net.JoinHostPort(targetIP.String(), port)
			return dialer.DialContext(ctx, network, targetAddr)
		},
		ResponseHeaderTimeout: respHeaderTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Strict certificate check
		},
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			incomingQuery := req.URL.RawQuery
			targetPath := targetURL.Path
			if targetPath == "" {
				targetPath = "/"
			}
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = targetPath
			req.URL.RawPath = targetURL.RawPath
			if targetURL.RawQuery != "" && incomingQuery != "" {
				req.URL.RawQuery = targetURL.RawQuery + "&" + incomingQuery
			} else if targetURL.RawQuery != "" {
				req.URL.RawQuery = targetURL.RawQuery
			} else {
				req.URL.RawQuery = incomingQuery
			}

			// Rewriting request headers
			req.Host = targetURL.Host
			req.Header.Del("Cookie") // Strip cookies to isolate session

			// Clear hop-by-hop headers from client request
			removeHopByHop(req.Header)

			// X-Forwarded-For configuration
			clientIP := requestip.ClientIP(c)
			if clientIP != "" {
				req.Header.Set("X-Forwarded-For", clientIP)
			}
		},
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			// 1. Intercept 3xx redirect codes to prevent redirect SSRF
			vlog.UpstreamStatusCode = resp.StatusCode
			if resp.StatusCode >= 300 && resp.StatusCode < 400 {
				return errors.New("upstream redirect intercepted")
			}

			// 2. Strip Set-Cookie headers
			resp.Header.Del("Set-Cookie")

			// 3. Remove hop-by-hop headers
			removeHopByHop(resp.Header)

			// 4. File Download Name control
			applyFilenameMode(resp, link)

			// 5. Size check
			maxSizeMB := getGlobalSettingInt("max_proxy_response_size_mb", 5)
			maxSizeBytes := int64(maxSizeMB) * 1024 * 1024

			if resp.ContentLength > maxSizeBytes {
				resp.StatusCode = http.StatusRequestEntityTooLarge
				return errors.New("response size exceeded limit")
			}

			// 6. Handle caching and stream reading
			isGET := c.Request.Method == http.MethodGet
			isCacheable := isGET && link.CacheEnabled && cache.IsGlobalCacheEnabled() && resp.StatusCode >= 200 && resp.StatusCode < 300

			var bodyBytes []byte
			if isCacheable {
				// Read full body up to max cache object size (or max response size)
				cacheMaxMB := link.CacheMaxObjectSizeMB
				if cacheMaxMB <= 0 {
					cacheMaxMB = int(maxSizeMB)
				}
				cacheMaxBytes := int64(cacheMaxMB) * 1024 * 1024

				limitBytes := maxSizeBytes + 1
				limitReader := io.LimitReader(resp.Body, limitBytes)
				buf := &bytes.Buffer{}
				written, err := io.Copy(buf, limitReader)
				if err != nil {
					return err
				}

				if written > maxSizeBytes {
					return errors.New("response size exceeded limit")
				}

				if written > cacheMaxBytes {
					// Exceeded object size, don't cache, but still return the bounded body.
					resp.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
					vlog.CacheStatus = "bypass"
					vlog.ResponseSize = int64(len(buf.Bytes()))
				} else {
					bodyBytes = buf.Bytes()
					vlog.CacheStatus = "miss"
					vlog.ResponseSize = int64(len(bodyBytes))

					// Save cache
					cache.GetGlobalCache().Put(cacheKey, resp.StatusCode, resp.Header, bodyBytes, link.CacheTTL)

					// Replace body reader
					resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			} else {
				// If not cached, wrap body in a limit check reader
				resp.Body = &limitReadCloser{
					rc:    resp.Body,
					limit: maxSizeBytes,
				}
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			vlog.Status = "failed"
			if err.Error() == "upstream redirect intercepted" {
				vlog.BlockedReason = "upstream_redirect_blocked"
				vlog.ResponseStatusCode = http.StatusBadGateway
				writeErrorHeaderAndBody(w, http.StatusBadGateway, "SSRF Blocked", "The destination URL redirected, which is forbidden.")
			} else if err.Error() == "response size exceeded limit" || strings.Contains(err.Error(), "response_size_exceeded") {
				vlog.BlockedReason = "response_size_exceeded"
				vlog.ResponseStatusCode = http.StatusRequestEntityTooLarge
				writeErrorHeaderAndBody(w, http.StatusRequestEntityTooLarge, "Payload Too Large", "The requested content exceeds the allowed size limit.")
			} else {
				vlog.BlockedReason = "upstream_connection_failed"
				vlog.ResponseStatusCode = http.StatusBadGateway
				writeErrorHeaderAndBody(w, http.StatusBadGateway, "Bad Gateway", "Failed to connect to the target upstream server.")
			}
			logs.QueueLog(vlog)
		},
	}

	// Run timeout context
	ctx, cancel := context.WithTimeout(c.Request.Context(), totalTimeout)
	defer cancel()

	// Update request context
	c.Request = c.Request.WithContext(ctx)

	// Custom response writer wrapper to record size and status
	rw := &proxyResponseWriter{
		ResponseWriter: c.Writer,
		vlog:           vlog,
	}

	proxy.ServeHTTP(rw, c.Request)

	// Ensure final log is recorded
	if vlog.Status == "success" {
		vlog.ResponseStatusCode = rw.statusCode
		if vlog.ResponseSize == 0 {
			vlog.ResponseSize = rw.writtenBytes
		}
		logs.QueueLog(vlog)
	}
}

type limitReadCloser struct {
	rc        io.ReadCloser
	limit     int64
	readBytes int64
}

func (l *limitReadCloser) Read(p []byte) (n int, err error) {
	n, err = l.rc.Read(p)
	l.readBytes += int64(n)
	if l.readBytes > l.limit {
		return n, fmt.Errorf("response_size_exceeded")
	}
	return n, err
}

func (l *limitReadCloser) Close() error {
	return l.rc.Close()
}

type proxyResponseWriter struct {
	gin.ResponseWriter
	vlog         *db.VisitLog
	statusCode   int
	writtenBytes int64
}

func (p *proxyResponseWriter) WriteHeader(code int) {
	p.statusCode = code
	p.ResponseWriter.WriteHeader(code)
}

func (p *proxyResponseWriter) Write(b []byte) (int, error) {
	if p.statusCode == 0 {
		p.statusCode = http.StatusOK
	}
	n, err := p.ResponseWriter.Write(b)
	p.writtenBytes += int64(n)
	return n, err
}

func sha256Sum(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func generateCacheKey(req *http.Request, link *db.Link) string {
	return fmt.Sprintf("%s|%s|%s|%s", link.Prefix, link.Slug, req.Method, req.URL.RawQuery)
}

func getGlobalSettingInt(key string, defaultValue int) int {
	if setting, found, err := db.FindGlobalSetting(key); err == nil && found {
		if val, err := strconv.Atoi(setting.Value); err == nil {
			return val
		}
	}
	return defaultValue
}

func applyFilenameMode(resp *http.Response, link *db.Link) {
	if link.FilenameMode == "inherit" {
		return
	}

	var filename string
	if link.FilenameMode == "custom" && link.CustomFilename != nil && *link.CustomFilename != "" {
		filename = *link.CustomFilename
	} else if link.FilenameMode == "auto" {
		u, err := url.Parse(link.TargetURL)
		if err == nil {
			filename = path.Base(u.Path)
		}
	}

	// Filter dangerous characters
	filename = sanitizeFilename(filename)

	if filename != "" && filename != "." && filename != "/" {
		// Standard RFC 5987 header encoding to support Chinese/UTF-8 filenames
		encoded := url.PathEscape(filename)
		resp.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", filename, encoded))
	}
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\\", "")
	return s
}

func removeHopByHop(header http.Header) {
	for _, h := range []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		header.Del(h)
	}
}

func logAccessFailure(c *gin.Context, link *db.Link, status, reason string, responseCode int) {
	remoteIP := requestip.ClientIP(c)
	geo := geoip.ResolveIP(remoteIP)
	now := config.NowUTC()

	logs.QueueLog(&db.VisitLog{
		LinkID:             &link.ID,
		Prefix:             link.Prefix,
		Slug:               link.Slug,
		PublicPath:         link.PublicPath,
		IP:                 remoteIP,
		IPHash:             sha256Sum(remoteIP),
		VisitorHash:        sha256Sum(remoteIP + c.GetHeader("User-Agent") + config.BusinessDate(now)),
		UserAgent:          c.GetHeader("User-Agent"),
		Referer:            c.GetHeader("Referer"),
		Country:            geo.Country,
		Region:             geo.Region,
		City:               geo.City,
		AccessTime:         now,
		Mode:               link.Mode,
		Status:             status,
		BlockedReason:      reason,
		ResponseStatusCode: responseCode,
		CacheStatus:        "disabled",
	})
}

// RenderPublicErrorHTML generates a styled HTML error page matching the ShareLink frontend design system.
// The HTML template is embedded via go:embed and rendered through html/template.
func RenderPublicErrorHTML(statusCode int, title, message string) string {
	return renderErrorPage(errorPageData{
		Title:      title,
		StatusCode: statusCode,
		Message:    message,
		IconSVG:    template.HTML(errorIconSVG(statusCode)),
	})
}

// errorIconSVG returns an SVG icon string based on the HTTP error category.
func errorIconSVG(statusCode int) string {
	switch {
	case statusCode == 404:
		// Search / not found
		return `<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#f87171" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="8" y1="11" x2="14" y2="11"/></svg>`
	case statusCode == 403:
		// Shield / blocked
		return `<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#f87171" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><line x1="9" y1="9" x2="15" y2="15"/><line x1="15" y1="9" x2="9" y2="15"/></svg>`
	case statusCode == 410:
		// Clock / expired
		return `<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#fbbf24" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`
	case statusCode == 413:
		// Arrow up / too large
		return `<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#fb923c" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"/><line x1="12" y1="9" x2="12" y2="21"/><path d="M2 3h20"/></svg>`
	case statusCode == 400:
		// Alert circle / bad request
		return `<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#f87171" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`
	default:
		// Alert triangle / server error (500, 502, 503, etc.)
		return `<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="#fb923c" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`
	}
}

func serveFriendlyError(c *gin.Context, statusCode int, title, message string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	setNoStoreHeaders(c.Writer.Header())
	c.String(statusCode, RenderPublicErrorHTML(statusCode, title, message))
}

func writeErrorHeaderAndBody(w http.ResponseWriter, statusCode int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	setNoStoreHeaders(w.Header())
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(RenderPublicErrorHTML(statusCode, title, message)))
}

func setNoStoreHeaders(header http.Header) {
	header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Expires", "0")
}
