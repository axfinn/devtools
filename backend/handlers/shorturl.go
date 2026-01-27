package handlers

import (
	"devtools/models"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ShortURLHandler struct {
	db       *models.DB
	password string // 管理密码
}

func NewShortURLHandler(db *models.DB, password string) *ShortURLHandler {
	return &ShortURLHandler{db: db, password: password}
}

type CreateShortURLRequest struct {
	OriginalURL string `json:"original_url" binding:"required"`
	ExpiresIn   int    `json:"expires_in"` // hours, default 720 (30 days)
	MaxClicks   int    `json:"max_clicks"` // default 1000
	CustomID    string `json:"custom_id"`  // 自定义短链ID，如 1, 2, 3
	Password    string `json:"password"`   // 管理密码
}

type CreateShortURLResponse struct {
	ID        string     `json:"id"`
	ShortURL  string     `json:"short_url"`
	ExpiresAt *time.Time `json:"expires_at"`
	MaxClicks int        `json:"max_clicks"`
}

type ShortURLStatsResponse struct {
	ID          string     `json:"id"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxClicks   int        `json:"max_clicks"`
	Clicks      int        `json:"clicks"`
	CreatedAt   time.Time  `json:"created_at"`
}

// validateURL validates the URL format and protocol
func validateURL(rawURL string) error {
	// Check length
	if len(rawURL) > 2048 {
		return errors.New("URL length exceeds 2048 characters")
	}

	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return errors.New("URL cannot be empty")
	}

	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Check scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("only http and https protocols are allowed")
	}

	// Check host
	if parsed.Host == "" {
		return errors.New("URL must have a valid host")
	}

	return nil
}

// Create handles POST /api/shorturl
func (h *ShortURLHandler) Create(c *gin.Context) {
	var req CreateShortURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// 检查是否配置了管理密码
	hasAdminPassword := h.password != ""
	hasCustomID := req.CustomID != ""
	passwordCorrect := req.Password != "" && req.Password == h.password

	// 使用自定义ID时需要验证密码
	if hasCustomID {
		if !hasAdminPassword {
			c.JSON(http.StatusForbidden, gin.H{"error": "自定义ID需要配置管理密码"})
			return
		}
		if !passwordCorrect {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}
	}

	// 验证自定义ID格式（仅允许字母、数字、下划线、短横线）
	if hasCustomID {
		if len(req.CustomID) > 32 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "自定义ID长度不能超过32个字符"})
			return
		}
		for _, r := range req.CustomID {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
				c.JSON(http.StatusBadRequest, gin.H{"error": "自定义ID只能包含字母、数字、下划线和短横线"})
				return
			}
		}
	}

	// Validate URL
	if err := validateURL(req.OriginalURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get client IP
	clientIP := c.ClientIP()

	// 密码正确时跳过限流检查
	if !passwordCorrect {
		// Check rate limit per IP: 10 per hour
		count, err := h.db.CountShortURLsByIP(clientIP, time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
			return
		}
		if count >= 10 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded: maximum 10 short URLs per hour per IP",
			})
			return
		}
	}

	// Set defaults
	if req.ExpiresIn == 0 {
		req.ExpiresIn = 720 // 30 days
	}
	if req.MaxClicks == 0 {
		req.MaxClicks = 1000
	}

	// Create short URL
	shortURL, err := h.db.CreateShortURLWithCustomID(req.OriginalURL, req.CustomID, req.ExpiresIn, req.MaxClicks, clientIP)
	if err != nil {
		if err.Error() == "storage limit reached" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "ID already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "该短链ID已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short URL"})
		return
	}

	// Build short URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	fullShortURL := fmt.Sprintf("%s://%s/s/%s", scheme, c.Request.Host, shortURL.ID)

	c.JSON(http.StatusCreated, CreateShortURLResponse{
		ID:        shortURL.ID,
		ShortURL:  fullShortURL,
		ExpiresAt: shortURL.ExpiresAt,
		MaxClicks: shortURL.MaxClicks,
	})
}

// Redirect handles GET /s/:id
func (h *ShortURLHandler) Redirect(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Short URL ID is required"})
		return
	}

	// Get short URL
	shortURL, err := h.db.GetShortURL(id)
	if err != nil {
		if err.Error() == "short URL not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve short URL"})
		return
	}

	// Check if expired
	if shortURL.ExpiresAt != nil && time.Now().After(*shortURL.ExpiresAt) {
		// Delete expired short URL
		h.db.DeleteShortURL(id)
		c.JSON(http.StatusGone, gin.H{"error": "Short URL has expired"})
		return
	}

	// Check if max clicks reached
	if shortURL.Clicks >= shortURL.MaxClicks {
		// Delete short URL that reached max clicks
		h.db.DeleteShortURL(id)
		c.JSON(http.StatusGone, gin.H{"error": "Short URL has reached its maximum click limit"})
		return
	}

	// Increment clicks atomically
	if err := h.db.IncrementClicks(id); err != nil {
		// Even if increment fails, we should still redirect
		// This is a best-effort tracking
	}

	// Redirect to original URL
	c.Redirect(http.StatusFound, shortURL.OriginalURL)
}

// GetStats handles GET /api/shorturl/:id/stats
func (h *ShortURLHandler) GetStats(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Short URL ID is required"})
		return
	}

	// Get short URL
	shortURL, err := h.db.GetShortURL(id)
	if err != nil {
		if err.Error() == "short URL not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve short URL"})
		return
	}

	c.JSON(http.StatusOK, ShortURLStatsResponse{
		ID:          shortURL.ID,
		OriginalURL: shortURL.OriginalURL,
		ExpiresAt:   shortURL.ExpiresAt,
		MaxClicks:   shortURL.MaxClicks,
		Clicks:      shortURL.Clicks,
		CreatedAt:   shortURL.CreatedAt,
	})
}

// List handles GET /api/shorturl/list?password=xxx
func (h *ShortURLHandler) List(c *gin.Context) {
	// 需要密码验证
	if h.password == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "未配置管理密码"})
		return
	}

	password := c.Query("password")
	if password != h.password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	urls, err := h.db.ListShortURLs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"urls": urls})
}
