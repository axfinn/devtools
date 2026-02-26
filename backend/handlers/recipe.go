package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

type RecipeHandler struct {
	db             *models.DB
	defaultExpDays int
	maxDataSize    int
	maxPerIP       int
	ipWindow       time.Duration
}

func NewRecipeHandler(db *models.DB, defaultExpDays, maxDataSize int) *RecipeHandler {
	if defaultExpDays <= 0 {
		defaultExpDays = 365
	}
	if maxDataSize <= 0 {
		maxDataSize = 1024 * 1024 // 1MB
	}
	return &RecipeHandler{
		db:             db,
		defaultExpDays: defaultExpDays,
		maxDataSize:    maxDataSize,
		maxPerIP:       5,
		ipWindow:       time.Hour,
	}
}

func recipePasswordIndex(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

type CreateRecipeRequest struct {
	Name      string `json:"name" binding:"required"`
	Password  string `json:"password" binding:"required"`
	ExpiresIn int    `json:"expires_in"`
}

type CreateRecipeResponse struct {
	ID         string     `json:"id"`
	CreatorKey string     `json:"creator_key"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type UpdateRecipeRequest struct {
	Action    string          `json:"action"` // "update_data", "update_name", "extend"
	CreatorKey string        `json:"creator_key"`
	Data     json.RawMessage `json:"data"`
	Name     string          `json:"name"`
	ExpiresIn int            `json:"expires_in"` // days
}

func (h *RecipeHandler) Create(c *gin.Context) {
	var req CreateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if len(req.Password) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要4个字符", "code": 400})
		return
	}

	if req.Name == "" {
		req.Name = "我的菜谱"
	}

	// Check if password already in use
	pwIndex := recipePasswordIndex(req.Password)
	existing, _ := h.db.GetRecipeByPasswordIndex(pwIndex)

	if existing != nil {
		if existing.ExpiresAt != nil && time.Now().After(*existing.ExpiresAt) {
			h.db.DeleteRecipe(existing.ID)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "该密码已被使用，请直接登录或使用其他密码", "code": 409})
			return
		}
	}

	ip := c.ClientIP()

	// IP rate limiting
	count, err := h.db.CountRecipeByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

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

	// Initialize default data
	defaultData := models.GetDefaultRecipeData()
	dataBytes, _ := json.Marshal(defaultData)

	expDays := req.ExpiresIn
	if expDays <= 0 {
		expDays = h.defaultExpDays
	}
	exp := time.Now().Add(time.Duration(expDays) * 24 * time.Hour)

	recipe := &models.Recipe{
		Name:         req.Name,
		Password:     hashedPassword,
		PasswordIndex: pwIndex,
		CreatorKey:  hashedCreatorKey,
		Data:         string(dataBytes),
		ExpiresAt:    &exp,
		CreatorIP:    ip,
	}

	if err := h.db.CreateRecipe(recipe); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, CreateRecipeResponse{
		ID:         recipe.ID,
		CreatorKey: creatorKey,
		ExpiresAt:  recipe.ExpiresAt,
	})
}

func (h *RecipeHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := c.Query("password")

	recipe, err := h.db.GetRecipe(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该菜谱", "code": 404})
		return
	}

	// Check expiration
	if recipe.ExpiresAt != nil && time.Now().After(*recipe.ExpiresAt) {
		h.db.DeleteRecipe(recipe.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该菜谱已过期", "code": 410})
		return
	}

	if password == "" || !utils.VerifyPassword(password, recipe.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         recipe.ID,
		"name":       recipe.Name,
		"data":       json.RawMessage(recipe.Data),
		"expires_at": recipe.ExpiresAt,
		"created_at": recipe.CreatedAt,
		"updated_at": recipe.UpdatedAt,
	})
}

func (h *RecipeHandler) GetByCreator(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	recipe, err := h.db.GetRecipe(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该菜谱", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, recipe.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "创建者密钥无效", "code": 401})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         recipe.ID,
		"name":       recipe.Name,
		"data":       json.RawMessage(recipe.Data),
		"expires_at": recipe.ExpiresAt,
		"created_at": recipe.CreatedAt,
		"updated_at": recipe.UpdatedAt,
	})
}

func (h *RecipeHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	recipe, err := h.db.GetRecipe(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该菜谱", "code": 404})
		return
	}

	// Auth check
	if req.CreatorKey == "" || !utils.VerifyPassword(req.CreatorKey, recipe.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	switch req.Action {
	case "update_data":
		if req.Data == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少数据", "code": 400})
			return
		}
		dataStr := string(req.Data)
		if len(dataStr) > h.maxDataSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "数据超过大小限制", "code": 400})
			return
		}
		if err := h.db.UpdateRecipeData(id, dataStr); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
			return
		}

	case "update_name":
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空", "code": 400})
			return
		}
		if err := h.db.UpdateRecipeName(id, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
			return
		}

	case "extend":
		days := req.ExpiresIn
		if days <= 0 {
			days = h.defaultExpDays
		}
		if days > 730 {
			days = 730
		}
		newExpires := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		if err := h.db.ExtendRecipe(id, &newExpires); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "延期失败", "code": 500})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "expires_at": newExpires})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的操作", "code": 400})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *RecipeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	recipe, err := h.db.GetRecipe(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该菜谱", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, recipe.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	h.db.DeleteRecipe(recipe.ID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type LoginRecipeRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *RecipeHandler) Login(c *gin.Context) {
	var req LoginRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码", "code": 400})
		return
	}

	pwIndex := recipePasswordIndex(req.Password)
	recipe, err := h.db.GetRecipeByPasswordIndex(pwIndex)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的菜谱", "code": 404})
		return
	}

	// Check expiration
	if recipe.ExpiresAt != nil && time.Now().After(*recipe.ExpiresAt) {
		h.db.DeleteRecipe(recipe.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该菜谱已过期", "code": 410})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         recipe.ID,
		"name":       recipe.Name,
		"data":       json.RawMessage(recipe.Data),
		"expires_at": recipe.ExpiresAt,
		"created_at": recipe.CreatedAt,
		"updated_at": recipe.UpdatedAt,
	})
}

// GetDefault 返回默认菜谱数据（无需登录）
func (h *RecipeHandler) GetDefault(c *gin.Context) {
	defaultData := models.GetDefaultRecipeData()
	c.JSON(http.StatusOK, gin.H{
		"data": defaultData,
	})
}

// GetDetailed 返回带详细步骤的菜谱数据（无需登录）
func (h *RecipeHandler) GetDetailed(c *gin.Context) {
	detailedData := models.GetDetailedRecipeSteps()
	c.JSON(http.StatusOK, gin.H{
		"data": detailedData,
	})
}
