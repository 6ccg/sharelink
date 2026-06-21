package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"sharelink/internal/config"
	"sharelink/internal/db"
	"sharelink/internal/requestip"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username"` // Optional, but if provided, must be "url"
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string    `json:"token"`
	ExpireAt time.Time `json:"expire_at"`
}

var (
	loginAttemptsMu sync.Mutex
	loginAttempts   = make(map[string][]time.Time)
)

const (
	maxLoginFailures = 5
	loginWindow      = 10 * time.Minute
)

func isLoginRateLimited(key string) bool {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	cutoff := config.NowUTC().Add(-loginWindow)
	attempts := pruneLoginAttempts(loginAttempts[key], cutoff)
	loginAttempts[key] = attempts
	return len(attempts) >= maxLoginFailures
}

func recordLoginFailure(key string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	cutoff := config.NowUTC().Add(-loginWindow)
	attempts := pruneLoginAttempts(loginAttempts[key], cutoff)
	attempts = append(attempts, config.NowUTC())
	loginAttempts[key] = attempts
}

func clearLoginFailures(key string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	delete(loginAttempts, key)
}

func pruneLoginAttempts(attempts []time.Time, cutoff time.Time) []time.Time {
	idx := 0
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			attempts[idx] = attempt
			idx++
		}
	}
	return attempts[:idx]
}

// GenerateJWT creates a new JWT token for the admin user
func GenerateJWT(username string) (string, time.Time, error) {
	now := config.NowUTC()
	expireTime := now.Add(24 * time.Hour) // Token valid for 24 hours
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "sharelink",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	return tokenString, expireTime, err
}

// Middleware verifies the JWT token in Authorization header
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Missing Authorization header",
				},
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header must be Bearer token",
				},
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"data":    nil,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}

// LoginHandler handles password authentication
func LoginHandler(c *gin.Context) {
	var req LoginRequest
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

	clientKey := requestip.ClientIP(c)
	if isLoginRateLimited(clientKey) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "RATE_LIMITED",
				"message": "Too many failed login attempts. Please try again later.",
			},
		})
		return
	}

	// Username must be "url" if provided
	if req.Username != "" && req.Username != "url" {
		// To prevent timing attacks, we still run a dummy hash check, but since this is local admin we just fail
		recordLoginFailure(clientKey)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
		return
	}

	// Fetch password hash from DB
	setting, found, err := db.FindGlobalSetting("admin_password_hash")
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
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "SYSTEM_ERROR",
				"message": "Admin password not initialized",
			},
		})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(setting.Value), []byte(req.Password))
	if err != nil {
		recordLoginFailure(clientKey)
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
		return
	}

	clearLoginFailures(clientKey)

	// Generate token
	token, expireAt, err := GenerateJWT("url")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"error": gin.H{
				"code":    "TOKEN_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": LoginResponse{
			Token:    token,
			ExpireAt: expireAt,
		},
		"error": nil,
	})
}

// LogoutHandler handles logout (stateless discard, success returned)
func LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "Successfully logged out"},
		"error":   nil,
	})
}

// MeHandler returns current logged in status
func MeHandler(c *gin.Context) {
	username, _ := c.Get("username")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"username": username,
		},
		"error": nil,
	})
}
