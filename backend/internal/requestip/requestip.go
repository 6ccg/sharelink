package requestip

import (
	"net"
	"strings"

	"sharelink/internal/db"

	"github.com/gin-gonic/gin"
)

func ClientIP(c *gin.Context) string {
	if trustProxyHeaders() {
		if ip := firstForwardedIP(c.GetHeader("X-Forwarded-For")); ip != "" {
			return ip
		}
		if ip := validIP(c.GetHeader("X-Real-IP")); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}

func trustProxyHeaders() bool {
	if db.DB == nil {
		return false
	}
	setting, found, err := db.FindGlobalSetting("trust_proxy_headers")
	if err != nil || !found {
		return false
	}
	return setting.Value == "true" || setting.Value == "1"
}

func firstForwardedIP(header string) string {
	for _, part := range strings.Split(header, ",") {
		if ip := validIP(part); ip != "" {
			return ip
		}
	}
	return ""
}

func validIP(value string) string {
	ip := strings.TrimSpace(value)
	if ip == "" {
		return ""
	}
	if parsed := net.ParseIP(ip); parsed == nil {
		return ""
	}
	return ip
}
