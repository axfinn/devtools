package handlers

import (
	"net/http"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

type ExcalidrawHandler struct {
	db             *models.DB
	adminPassword  string
	defaultExpDays int
	maxContentSize int
	maxPerIP       int
	ipWindow       time.Duration
}

func NewExcalidrawHandler(db *models.DB, adminPassword string, defaultExpDays, maxContentSize int) *ExcalidrawHandler {
	if defaultExpDays <= 0 {
		defaultExpDays = 30
	}
	if maxContentSize <= 0 {
		maxContentSize = 10 * 1024 * 1024 // 10MB
	}
	return &ExcalidrawHandler{
		db:             db,
		adminPassword:  adminPassword,
		defaultExpDays: defaultExpDays,
		maxContentSize: maxContentSize,
		maxPerIP:       10,
		ipWindow:       time.Hour,
	}
}

type CreateExcalidrawRequest struct {
	Content       string `json:"content" binding:"required"`
	Title         string `json:"title"`
	Password      string `json:"password" binding:"required"`
	ExpiresIn     int    `json:"expires_in"`     // days, 0 = use default, -1 = permanent (admin only)
	AdminPassword string `json:"admin_password"` // for permanent save
}

type CreateExcalidrawResponse struct {
	ID          string     `json:"id"`
	CreatorKey  string     `json:"creator_key"`
	ShortCode   string     `json:"short_code"`
	ShareURL    string     `json:"share_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsPermanent bool       `json:"is_permanent"`
}

type ExcalidrawContentResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	IsPermanent bool       `json:"is_permanent"`
}

type UpdateExcalidrawRequest struct {
	Action        string `json:"action"` // "extend", "edit", "set_permanent"
	Content       string `json:"content"`
	Title         string `json:"title"`
	ExpiresIn     int    `json:"expires_in"`
	CreatorKey    string `json:"creator_key"`
	AdminPassword string `json:"admin_password"` // for set_permanent
}

type UpdateExcalidrawResponse struct {
	Success     bool       `json:"success"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsPermanent bool       `json:"is_permanent,omitempty"`
}

func (h *ExcalidrawHandler) Create(c *gin.Context) {
	var req CreateExcalidrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// Content size limit
	if len(req.Content) > h.maxContentSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过大小限制", "code": 400})
		return
	}

	// Password is required
	if len(req.Password) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要4个字符", "code": 400})
		return
	}

	ip := c.ClientIP()

	// IP rate limiting
	count, err := h.db.CountExcalidrawSharesByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	// Check for permanent save (admin only)
	isPermanent := false
	var expiresAt *time.Time

	if req.ExpiresIn == -1 {
		// Permanent save requires admin password
		if h.adminPassword == "" || req.AdminPassword != h.adminPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "永久保存需要管理员密码", "code": 401})
			return
		}
		isPermanent = true
		// expiresAt stays nil for permanent
	} else {
		// Calculate expiration
		expiresIn := req.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = h.defaultExpDays
		}
		if expiresIn > 365 {
			expiresIn = 365
		}
		exp := time.Now().Add(time.Duration(expiresIn) * 24 * time.Hour)
		expiresAt = &exp
	}

	// Generate creator key
	creatorKey := generateKey()

	hashedCreatorKey, err := utils.HashPassword(creatorKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密钥生成失败", "code": 500})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败", "code": 500})
		return
	}

	share := &models.ExcalidrawShare{
		Content:     req.Content,
		Title:       req.Title,
		CreatorKey:  hashedCreatorKey,
		AccessKey:   hashedPassword,
		ExpiresAt:   expiresAt,
		CreatorIP:   ip,
		IsPermanent: isPermanent,
	}

	if err := h.db.CreateExcalidrawShare(share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	// Create short URL for the share (default 10 clicks)
	shortURL := &models.ShortURL{
		OriginalURL: "/draw/" + share.ID,
		ExpiresAt:   expiresAt,
		MaxClicks:   10, // default 10 clicks
		CreatorIP:   ip,
	}
	if err := h.db.CreateShortURLFromStruct(shortURL); err != nil {
		// Continue without short URL if creation fails
		c.JSON(http.StatusCreated, CreateExcalidrawResponse{
			ID:          share.ID,
			CreatorKey:  creatorKey,
			ExpiresAt:   expiresAt,
			IsPermanent: isPermanent,
		})
		return
	}

	// Update share with short code
	share.ShortCode = shortURL.ID
	h.db.UpdateExcalidrawShortCode(share.ID, shortURL.ID)

	c.JSON(http.StatusCreated, CreateExcalidrawResponse{
		ID:          share.ID,
		CreatorKey:  creatorKey,
		ShortCode:   shortURL.ID,
		ShareURL:    "/s/" + shortURL.ID,
		ExpiresAt:   expiresAt,
		IsPermanent: isPermanent,
	})
}

func (h *ExcalidrawHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := c.Query("password")

	share, err := h.db.GetExcalidrawShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该画图", "code": 404})
		return
	}

	// Check expiration (skip for permanent)
	if !share.IsPermanent && share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		h.cleanupShare(share)
		c.JSON(http.StatusGone, gin.H{"error": "该画图已过期", "code": 410})
		return
	}

	// Validate password
	if password == "" || !utils.VerifyPassword(password, share.AccessKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	c.JSON(http.StatusOK, ExcalidrawContentResponse{
		ID:          share.ID,
		Title:       share.Title,
		Content:     share.Content,
		ExpiresAt:   share.ExpiresAt,
		CreatedAt:   share.CreatedAt,
		IsPermanent: share.IsPermanent,
	})
}

func (h *ExcalidrawHandler) GetByCreator(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	share, err := h.db.GetExcalidrawShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该画图", "code": 404})
		return
	}

	// Validate creator key
	if creatorKey == "" || !utils.VerifyPassword(creatorKey, share.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "创建者密钥无效", "code": 401})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           share.ID,
		"title":        share.Title,
		"content":      share.Content,
		"expires_at":   share.ExpiresAt,
		"created_at":   share.CreatedAt,
		"short_code":   share.ShortCode,
		"is_permanent": share.IsPermanent,
	})
}

func (h *ExcalidrawHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateExcalidrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	share, err := h.db.GetExcalidrawShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该画图", "code": 404})
		return
	}

	// Auth check - creator only (except set_permanent which needs admin)
	if req.Action != "set_permanent" {
		if req.CreatorKey == "" || !utils.VerifyPassword(req.CreatorKey, share.CreatorKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
			return
		}
	}

	var resp UpdateExcalidrawResponse

	switch req.Action {
	case "extend":
		if share.IsPermanent {
			c.JSON(http.StatusBadRequest, gin.H{"error": "永久保存无需延期", "code": 400})
			return
		}
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

	case "edit":
		if req.Content != "" {
			if len(req.Content) > h.maxContentSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": "内容超过大小限制", "code": 400})
				return
			}
			share.Content = req.Content
		}
		if req.Title != "" {
			share.Title = req.Title
		}

	case "set_permanent":
		// Admin only
		if h.adminPassword == "" || req.AdminPassword != h.adminPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
			return
		}
		share.IsPermanent = true
		share.ExpiresAt = nil
		resp.IsPermanent = true

		// Update short URL to not expire
		if share.ShortCode != "" {
			h.db.UpdateShortURLExpires(share.ShortCode, nil)
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的操作", "code": 400})
		return
	}

	if err := h.db.UpdateExcalidrawShare(share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	resp.Success = true
	c.JSON(http.StatusOK, resp)
}

func (h *ExcalidrawHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	share, err := h.db.GetExcalidrawShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该画图", "code": 404})
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

func (h *ExcalidrawHandler) cleanupShare(share *models.ExcalidrawShare) {
	// Delete associated short URL
	if share.ShortCode != "" {
		h.db.DeleteShortURL(share.ShortCode)
	}
	h.db.DeleteExcalidrawShare(share.ID)
}

// Admin list all shares
func (h *ExcalidrawHandler) List(c *gin.Context) {
	adminPwd := c.Query("admin_password")

	if h.adminPassword == "" || adminPwd != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
		return
	}

	shares, err := h.db.ListAllExcalidrawShares()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败", "code": 500})
		return
	}

	var list []gin.H
	for _, share := range shares {
		list = append(list, gin.H{
			"id":           share.ID,
			"title":        share.Title,
			"expires_at":   share.ExpiresAt,
			"created_at":   share.CreatedAt,
			"short_code":   share.ShortCode,
			"is_permanent": share.IsPermanent,
		})
	}

	c.JSON(http.StatusOK, gin.H{"list": list})
}

// Admin get any share content
func (h *ExcalidrawHandler) AdminGet(c *gin.Context) {
	id := c.Param("id")
	adminPwd := c.Query("admin_password")

	if h.adminPassword == "" || adminPwd != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
		return
	}

	share, err := h.db.GetExcalidrawShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该画图", "code": 404})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           share.ID,
		"title":        share.Title,
		"content":      share.Content,
		"expires_at":   share.ExpiresAt,
		"created_at":   share.CreatedAt,
		"short_code":   share.ShortCode,
		"is_permanent": share.IsPermanent,
	})
}

// Admin delete any share
func (h *ExcalidrawHandler) AdminDelete(c *gin.Context) {
	id := c.Param("id")
	adminPwd := c.Query("admin_password")

	if h.adminPassword == "" || adminPwd != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员密码", "code": 401})
		return
	}

	share, err := h.db.GetExcalidrawShare(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该画图", "code": 404})
		return
	}

	h.cleanupShare(share)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
