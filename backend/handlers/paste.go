package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

type PasteHandler struct {
	db         *models.DB
	maxTotal   int
	maxPerIP   int
	ipWindow   time.Duration
}

func NewPasteHandler(db *models.DB) *PasteHandler {
	return &PasteHandler{
		db:       db,
		maxTotal: 10000,       // 最多存储 10000 条
		maxPerIP: 5,           // 每 IP 每分钟最多 5 条
		ipWindow: time.Minute,
	}
}

type CreatePasteRequest struct {
	Content   string `json:"content" binding:"required"`
	Title     string `json:"title"`
	Language  string `json:"language"`
	Password  string `json:"password"`
	ExpiresIn int    `json:"expires_in"` // 过期时间（小时）
	MaxViews  int    `json:"max_views"`
}

type PasteResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Language  string    `json:"language"`
	Content   string    `json:"content,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	MaxViews  int       `json:"max_views"`
	Views     int       `json:"views"`
	CreatedAt time.Time `json:"created_at"`
	HasPassword bool    `json:"has_password"`
}

func (h *PasteHandler) Create(c *gin.Context) {
	var req CreatePasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// 检查内容大小（100KB 限制）
	if len(req.Content) > 100*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容大小超过 100KB 限制", "code": 400})
		return
	}

	ip := c.ClientIP()

	// 检查 IP 限流
	count, err := h.db.CountByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	// 检查总量限制
	total, err := h.db.TotalCount()
	if err == nil && total >= h.maxTotal {
		// 清理过期数据
		h.db.CleanExpired()
		// 再次检查
		total, _ = h.db.TotalCount()
		if total >= h.maxTotal {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "存储已满，请稍后再试", "code": 503})
			return
		}
	}

	// 设置默认值
	if req.Language == "" {
		req.Language = "text"
	}
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 24 // 默认 24 小时
	}
	if req.ExpiresIn > 168 { // 最长 7 天
		req.ExpiresIn = 168
	}
	if req.MaxViews <= 0 {
		req.MaxViews = 100
	}
	if req.MaxViews > 1000 {
		req.MaxViews = 1000
	}

	paste := &models.Paste{
		Content:   req.Content,
		Title:     req.Title,
		Language:  req.Language,
		ExpiresAt: time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour),
		MaxViews:  req.MaxViews,
		CreatorIP: ip,
	}

	// 密码加密存储
	if req.Password != "" {
		hash := sha256.Sum256([]byte(req.Password))
		paste.Password = hex.EncodeToString(hash[:])
	}

	if err := h.db.CreatePaste(paste); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         paste.ID,
		"expires_at": paste.ExpiresAt,
		"max_views":  paste.MaxViews,
	})
}

func (h *PasteHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := c.Query("password")

	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 检查是否过期
	if time.Now().After(paste.ExpiresAt) {
		h.db.DeletePaste(id)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已过期", "code": 410})
		return
	}

	// 检查访问次数
	if paste.Views >= paste.MaxViews {
		h.db.DeletePaste(id)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已达到最大访问次数", "code": 410})
		return
	}

	// 检查密码
	if paste.Password != "" {
		if password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":        "需要密码",
				"code":         401,
				"has_password": true,
			})
			return
		}
		hash := sha256.Sum256([]byte(password))
		if hex.EncodeToString(hash[:]) != paste.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
			return
		}
	}

	// 增加访问次数
	h.db.IncrementViews(id)
	paste.Views++

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
		Content:     paste.Content,
		ExpiresAt:   paste.ExpiresAt,
		MaxViews:    paste.MaxViews,
		Views:       paste.Views,
		CreatedAt:   paste.CreatedAt,
		HasPassword: paste.Password != "",
	})
}

func (h *PasteHandler) GetInfo(c *gin.Context) {
	id := c.Param("id")

	paste, err := h.db.GetPaste(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// 检查是否过期
	if time.Now().After(paste.ExpiresAt) {
		h.db.DeletePaste(id)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已过期", "code": 410})
		return
	}

	c.JSON(http.StatusOK, PasteResponse{
		ID:          paste.ID,
		Title:       paste.Title,
		Language:    paste.Language,
		ExpiresAt:   paste.ExpiresAt,
		MaxViews:    paste.MaxViews,
		Views:       paste.Views,
		CreatedAt:   paste.CreatedAt,
		HasPassword: paste.Password != "",
	})
}
