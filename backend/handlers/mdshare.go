package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

type MDShareHandler struct {
	db              *models.DB
	adminPassword   string
	defaultMaxViews int
	defaultExpDays  int
	maxPerIP        int
	ipWindow        time.Duration
}

func NewMDShareHandler(db *models.DB, adminPassword string, defaultMaxViews, defaultExpDays int) *MDShareHandler {
	if defaultMaxViews <= 0 {
		defaultMaxViews = 5
	}
	if defaultExpDays <= 0 {
		defaultExpDays = 30
	}
	return &MDShareHandler{
		db:              db,
		adminPassword:   adminPassword,
		defaultMaxViews: defaultMaxViews,
		defaultExpDays:  defaultExpDays,
		maxPerIP:        10,
		ipWindow:        time.Hour,
	}
}

type CreateMDShareRequest struct {
	Content   string `json:"content" binding:"required"`
	Title     string `json:"title"`
	MaxViews  int    `json:"max_views"` // 2-10
	ExpiresIn int    `json:"expires_in"` // days, 0 = use default
}

type CreateMDShareResponse struct {
	ID           string    `json:"id"`
	CreatorKey   string    `json:"creator_key"`
	AccessKey    string    `json:"access_key"`
	ShortCode    string    `json:"short_code"`
	ShareURL     string    `json:"share_url"`
	ExpiresAt    time.Time `json:"expires_at"`
	MaxViews     int       `json:"max_views"`
}

type MDShareContentResponse struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	MaxViews      int        `json:"max_views"`
	Views         int        `json:"views"`
	RemainingViews int       `json:"remaining_views"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type UpdateMDShareRequest struct {
	Action       string `json:"action"`        // "extend", "reshare", "edit"
	Content      string `json:"content"`
	Title        string `json:"title"`
	MaxViews     int    `json:"max_views"`
	ExpiresIn    int    `json:"expires_in"`
	CreatorKey   string `json:"creator_key"`
}

type UpdateMDShareResponse struct {
	Success       bool       `json:"success"`
	AccessKey     string     `json:"access_key,omitempty"`
	ShortCode     string     `json:"short_code,omitempty"`
	ShareURL      string     `json:"share_url,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	MaxViews      int        `json:"max_views,omitempty"`
	Views         int        `json:"views,omitempty"`
}

func generateKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (h *MDShareHandler) Create(c *gin.Context) {
	var req CreateMDShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// Content size limit (2MB for images)
	if len(req.Content) > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 2MB 限制", "code": 400})
		return
	}

	ip := c.ClientIP()

	// IP rate limiting
	count, err := h.db.CountMDSharesByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	// Validate max_views (2-10)
	maxViews := req.MaxViews
	if maxViews <= 0 {
		maxViews = h.defaultMaxViews
	}
	if maxViews < 2 {
		maxViews = 2
	}
	if maxViews > 10 {
		maxViews = 10
	}

	// Calculate expiration
	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = h.defaultExpDays
	}
	if expiresIn > 365 {
		expiresIn = 365
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * 24 * time.Hour)

	// Generate keys
	creatorKey := generateKey()
	accessKey := generateKey()

	share := &models.MDShare{
		Content:     req.Content,
		Title:       req.Title,
		CreatorKey:  utils.HashPassword(creatorKey),
		AccessKey:   utils.HashPassword(accessKey),
		MaxViews:    maxViews,
		Views:       0,
		ExpiresAt:   &expiresAt,
		CreatorIP:   ip,
	}

	if err := h.db.CreateMDShare(share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	// Create short URL for the share
	shortURL := &models.ShortURL{
		OriginalURL: "/md/" + share.ID + "?key=" + accessKey,
		ExpiresAt:   &expiresAt,
		MaxClicks:   maxViews + 10, // Allow some buffer
		CreatorIP:   ip,
		MDShareID:   share.ID, // Link to mdshare for auto-cleanup
	}
	if err := h.db.CreateShortURLFromStruct(shortURL); err != nil {
		// Continue without short URL if creation fails
		c.JSON(http.StatusCreated, CreateMDShareResponse{
			ID:         share.ID,
			CreatorKey: creatorKey,
			AccessKey:  accessKey,
			ExpiresAt:  expiresAt,
			MaxViews:   maxViews,
		})
		return
	}

	// Update mdshare with short code
	share.ShortCode = shortURL.ID
	h.db.UpdateMDShareShortCode(share.ID, shortURL.ID)

	c.JSON(http.StatusCreated, CreateMDShareResponse{
		ID:         share.ID,
		CreatorKey: creatorKey,
		AccessKey:  accessKey,
		ShortCode:  shortURL.ID,
		ShareURL:   "/s/" + shortURL.ID,
		ExpiresAt:  expiresAt,
		MaxViews:   maxViews,
	})
}

func (h *MDShareHandler) Get(c *gin.Context) {
	id := c.Param("id")
	key := c.Query("key")

	share, err := h.db.GetMDShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// Check expiration
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		h.cleanupShare(share)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已过期", "code": 410})
		return
	}

	// Validate access key
	if key == "" || !utils.VerifyPassword(key, share.AccessKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "访问密钥无效", "code": 401})
		return
	}

	// Check view limit
	if share.Views >= share.MaxViews {
		h.cleanupShare(share)
		c.JSON(http.StatusGone, gin.H{"error": "该分享已达到最大访问次数", "code": 410})
		return
	}

	// Increment views
	h.db.IncrementMDShareViews(id)
	share.Views++

	c.JSON(http.StatusOK, MDShareContentResponse{
		ID:             share.ID,
		Title:          share.Title,
		Content:        share.Content,
		MaxViews:       share.MaxViews,
		Views:          share.Views,
		RemainingViews: share.MaxViews - share.Views,
		ExpiresAt:      share.ExpiresAt,
		CreatedAt:      share.CreatedAt,
	})
}

func (h *MDShareHandler) GetByCreator(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	share, err := h.db.GetMDShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// Validate creator key
	if creatorKey == "" || !utils.VerifyPassword(creatorKey, share.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "创建者密钥无效", "code": 401})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              share.ID,
		"title":           share.Title,
		"content":         share.Content,
		"max_views":       share.MaxViews,
		"views":           share.Views,
		"remaining_views": share.MaxViews - share.Views,
		"expires_at":      share.ExpiresAt,
		"created_at":      share.CreatedAt,
		"short_code":      share.ShortCode,
	})
}

func (h *MDShareHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateMDShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	share, err := h.db.GetMDShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// Auth check - creator only
	if req.CreatorKey == "" || !utils.VerifyPassword(req.CreatorKey, share.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	var resp UpdateMDShareResponse
	ip := c.ClientIP()

	switch req.Action {
	case "extend":
		days := req.ExpiresIn
		if days <= 0 {
			days = h.defaultExpDays
		}
		if days > 365 {
			days = 365
		}
		newExpires := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		share.ExpiresAt = &newExpires
		resp.ExpiresAt = share.ExpiresAt

		// Update short URL expiration too
		if share.ShortCode != "" {
			h.db.UpdateShortURLExpires(share.ShortCode, &newExpires)
		}

	case "reshare":
		// Generate new access key and reset views
		newAccessKey := generateKey()
		share.AccessKey = utils.HashPassword(newAccessKey)
		share.Views = 0
		if req.MaxViews >= 2 && req.MaxViews <= 10 {
			share.MaxViews = req.MaxViews
		}

		// Delete old short URL and create new one
		if share.ShortCode != "" {
			h.db.DeleteShortURL(share.ShortCode)
		}

		shortURL := &models.ShortURL{
			OriginalURL: "/md/" + share.ID + "?key=" + newAccessKey,
			ExpiresAt:   share.ExpiresAt,
			MaxClicks:   share.MaxViews + 10,
			CreatorIP:   ip,
			MDShareID:   share.ID,
		}
		if err := h.db.CreateShortURLFromStruct(shortURL); err == nil {
			share.ShortCode = shortURL.ID
			resp.ShortCode = shortURL.ID
			resp.ShareURL = "/s/" + shortURL.ID
		}

		resp.AccessKey = newAccessKey
		resp.MaxViews = share.MaxViews
		resp.Views = share.Views

	case "edit":
		if req.Content != "" {
			if len(req.Content) > 2*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过 2MB 限制", "code": 400})
				return
			}
			share.Content = req.Content
		}
		share.Title = req.Title

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的操作", "code": 400})
		return
	}

	if err := h.db.UpdateMDShare(share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	resp.Success = true
	c.JSON(http.StatusOK, resp)
}

func (h *MDShareHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	share, err := h.db.GetMDShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// Auth check
	if creatorKey == "" || !utils.VerifyPassword(creatorKey, share.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	h.cleanupShare(share)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *MDShareHandler) cleanupShare(share *models.MDShare) {
	// Delete associated short URL
	if share.ShortCode != "" {
		h.db.DeleteShortURL(share.ShortCode)
	}
	h.db.DeleteMDShare(share.ID)
}

// Admin list all shares
func (h *MDShareHandler) List(c *gin.Context) {
	adminPwd := c.Query("admin_password")

	if h.adminPassword == "" || adminPwd != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
		return
	}

	shares, err := h.db.ListAllMDShares()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败", "code": 500})
		return
	}

	var list []gin.H
	for _, share := range shares {
		remaining := share.MaxViews - share.Views
		if remaining < 0 {
			remaining = 0
		}
		list = append(list, gin.H{
			"id":              share.ID,
			"title":           share.Title,
			"max_views":       share.MaxViews,
			"views":           share.Views,
			"remaining_views": remaining,
			"expires_at":      share.ExpiresAt,
			"created_at":      share.CreatedAt,
			"short_code":      share.ShortCode,
		})
	}

	c.JSON(http.StatusOK, gin.H{"list": list})
}

// Admin get any share content
func (h *MDShareHandler) AdminGet(c *gin.Context) {
	id := c.Param("id")
	adminPwd := c.Query("admin_password")

	if h.adminPassword == "" || adminPwd != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
		return
	}

	share, err := h.db.GetMDShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	// Admin view doesn't increment views
	c.JSON(http.StatusOK, gin.H{
		"id":              share.ID,
		"title":           share.Title,
		"content":         share.Content,
		"max_views":       share.MaxViews,
		"views":           share.Views,
		"remaining_views": share.MaxViews - share.Views,
		"expires_at":      share.ExpiresAt,
		"created_at":      share.CreatedAt,
		"short_code":      share.ShortCode,
	})
}

// Admin delete any share
func (h *MDShareHandler) AdminDelete(c *gin.Context) {
	id := c.Param("id")
	adminPwd := c.Query("admin_password")

	if h.adminPassword == "" || adminPwd != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
		return
	}

	share, err := h.db.GetMDShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该分享", "code": 404})
		return
	}

	h.cleanupShare(share)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
