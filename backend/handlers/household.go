package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// HouseholdHandler 家庭物品管理处理器
type HouseholdHandler struct {
	db  *models.DB
	cfg *config.Config
}

func NewHouseholdHandler(db *models.DB, cfg *config.Config) *HouseholdHandler {
	return &HouseholdHandler{db: db, cfg: cfg}
}

func (h *HouseholdHandler) profileCreatorKey(c *gin.Context) string {
	key := strings.TrimSpace(c.Query("creator_key"))
	if key != "" {
		return key
	}
	return strings.TrimSpace(c.GetHeader("X-Creator-Key"))
}

func (h *HouseholdHandler) requireProfileAccessWithKey(c *gin.Context, profileID, creatorKey string) (*models.HouseholdProfile, bool) {
	profile, err := h.db.GetHouseholdProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return nil, false
	}

	if creatorKey == "" || creatorKey != profile.CreatorKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限", "code": 403})
		return nil, false
	}

	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "档案已过期", "code": 403})
		return nil, false
	}

	return profile, true
}

func (h *HouseholdHandler) requireProfileAccess(c *gin.Context, profileID string) (*models.HouseholdProfile, bool) {
	return h.requireProfileAccessWithKey(c, profileID, h.profileCreatorKey(c))
}

func (h *HouseholdHandler) requireProfileItemAccess(c *gin.Context) (*models.HouseholdProfile, *models.ProfileItem, bool) {
	profileID := c.Param("id")
	profile, ok := h.requireProfileAccess(c, profileID)
	if !ok {
		return nil, nil, false
	}

	itemID := c.Param("itemId")
	item, err := h.db.GetProfileItem(itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "物品不存在", "code": 404})
		return nil, nil, false
	}
	if item.ProfileID != profileID {
		c.JSON(http.StatusForbidden, gin.H{"error": "物品不属于当前档案", "code": 403})
		return nil, nil, false
	}
	return profile, item, true
}

// ========== 档案管理 API ==========

// CreateProfileRequest 创建档案请求
type CreateProfileRequest struct {
	Password    string `json:"password" binding:"required,min=4"`
	Name        string `json:"name"`
	ExpiresIn   int    `json:"expires_in"`
}

// LoginProfileRequest 登录档案请求
type LoginProfileRequest struct {
	Password string `json:"password" binding:"required,min=4"`
}

// CreateProfile 创建档案
func (h *HouseholdHandler) CreateProfile(c *gin.Context) {
	// 自动初始化数据库表
	_ = h.db.InitHousehold()

	var req CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供密码（至少4位）", "code": 400})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "我的家庭物品"
	}

	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 365 // 默认1年
	}

	// 检查密码是否已存在
	passwordIndex := hashPassword(req.Password)
	existing, _ := h.db.GetHouseholdProfileByPasswordIndex(passwordIndex)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该密码已存在档案，请直接登录或使用其他密码", "code": 409})
		return
	}

	profile := &models.HouseholdProfile{
		PasswordIndex: passwordIndex,
		Name:          name,
	}

	if err := h.db.CreateHouseholdProfile(profile, expiresIn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"id":          profile.ID,
		"creator_key": profile.CreatorKey,
		"name":        profile.Name,
		"expires_at":  profile.ExpiresAt,
	})
}

// LoginProfile 登录档案
func (h *HouseholdHandler) LoginProfile(c *gin.Context) {
	// 自动初始化数据库表
	_ = h.db.InitHousehold()

	var req LoginProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供密码", "code": 400})
		return
	}

	passwordIndex := hashPassword(req.Password)
	profile, err := h.db.GetHouseholdProfileByPasswordIndex(passwordIndex)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	// 检查是否过期
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "档案已过期，请重新创建", "code": 403})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"id":          profile.ID,
		"creator_key": profile.CreatorKey,
		"name":        profile.Name,
		"expires_at":  profile.ExpiresAt,
		"created_at":  profile.CreatedAt,
	})
}

// GetProfile 获取档案信息
func (h *HouseholdHandler) GetProfile(c *gin.Context) {
	profileID := c.Param("id")
	profile, ok := h.requireProfileAccess(c, profileID)
	if !ok {
		return
	}

	items, _ := h.db.GetProfileItems(profileID)
	stats, _ := h.db.GetProfileStats(profileID)

	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"id":          profile.ID,
		"name":        profile.Name,
		"creator_key": profile.CreatorKey,
		"expires_at":  profile.ExpiresAt,
		"created_at":  profile.CreatedAt,
		"items":       items,
		"stats":       stats,
	})
}

// ExtendProfile 延长档案
func (h *HouseholdHandler) ExtendProfile(c *gin.Context) {
	profileID := c.Param("id")

	var req struct {
		CreatorKey string `json:"creator_key"`
		ExpiresIn  int    `json:"expires_in"`
	}
	c.ShouldBindJSON(&req)

	profile, err := h.db.GetHouseholdProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}

	creatorKey := strings.TrimSpace(req.CreatorKey)
	if creatorKey == "" {
		creatorKey = h.profileCreatorKey(c)
	}
	if creatorKey == "" || creatorKey != profile.CreatorKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限", "code": 403})
		return
	}

	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 365
	}

	newExpiresAt := time.Now().AddDate(0, 0, expiresIn)
	if err := h.db.ExtendHouseholdProfile(profileID, &newExpiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"expires_at": newExpiresAt,
	})
}

// DeleteProfile 删除档案
func (h *HouseholdHandler) DeleteProfile(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	if err := h.db.DeleteHouseholdProfile(profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// ========== 档案物品管理 API ==========

// CreateProfileItem 创建档案物品
func (h *HouseholdHandler) CreateProfileItem(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	var item struct {
		Name        string `json:"name" binding:"required"`
		Category    string `json:"category"`
		Quantity    int    `json:"quantity"`
		Unit        string `json:"unit"`
		MinQuantity int    `json:"min_quantity"`
		ExpiryDate  string `json:"expiry_date"`
		ExpiryDays  int    `json:"expiry_days"`
		Location    string `json:"location"`
		Notes       string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供物品名称", "code": 400})
		return
	}

	if item.Category == "" {
		item.Category = "其他"
	}
	if item.Quantity == 0 {
		item.Quantity = 1
	}
	if item.Unit == "" {
		item.Unit = "个"
	}
	if item.MinQuantity == 0 {
		item.MinQuantity = 1
	}

	dbItem := &models.ProfileItem{
		ProfileID:   profileID,
		Name:        item.Name,
		Category:    item.Category,
		Quantity:    item.Quantity,
		Unit:        item.Unit,
		MinQuantity: item.MinQuantity,
		ExpiryDate:  item.ExpiryDate,
		ExpiryDays:  item.ExpiryDays,
		Location:    item.Location,
		Notes:       item.Notes,
	}

	if err := h.db.CreateProfileItem(dbItem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	stats, _ := h.db.GetProfileStats(profileID)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  dbItem,
		"stats": stats,
	})
}

// GetProfileItems 获取档案物品列表
func (h *HouseholdHandler) GetProfileItems(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}
	alert := c.Query("alert") == "true"
	category := strings.TrimSpace(c.Query("category"))

	var items []*models.ProfileItem
	var err error

	if alert {
		items, err = h.db.GetProfileAlerts(profileID)
	} else {
		items, err = h.db.GetProfileItems(profileID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	if category != "" {
		filtered := make([]*models.ProfileItem, 0, len(items))
		for _, item := range items {
			if item.Category == category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	stats, _ := h.db.GetProfileStats(profileID)

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  items,
		"stats": stats,
	})
}

// GetProfileLocations 获取档案位置列表
func (h *HouseholdHandler) GetProfileLocations(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	itemLocations, err := h.db.GetProfileLocations(profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	library, libErr := h.db.GetHouseholdLocationsLibrary(profileID)
	if libErr != nil {
		library = []*models.HouseholdLocation{}
	}

	seen := make(map[string]bool)
	merged := make([]string, 0, len(itemLocations)+len(library))
	for _, loc := range itemLocations {
		if loc == "" || seen[loc] {
			continue
		}
		seen[loc] = true
		merged = append(merged, loc)
	}
	for _, loc := range library {
		if loc.Name == "" || seen[loc.Name] {
			continue
		}
		seen[loc.Name] = true
		merged = append(merged, loc.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": merged,
	})
}

// GetLocationLibrary 获取位置库
func (h *HouseholdHandler) GetLocationLibrary(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	locations, err := h.db.GetHouseholdLocationsLibrary(profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": locations,
	})
}

// CreateLocation 添加位置
func (h *HouseholdHandler) CreateLocation(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供位置名称", "code": 400})
		return
	}

	loc := &models.HouseholdLocation{
		ProfileID: profileID,
		Name:      strings.TrimSpace(req.Name),
	}
	if loc.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供位置名称", "code": 400})
		return
	}

	if err := h.db.CreateHouseholdLocation(loc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": loc,
	})
}

// UpdateLocation 更新位置
func (h *HouseholdHandler) UpdateLocation(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	locID := c.Param("locId")
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供位置名称", "code": 400})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供位置名称", "code": 400})
		return
	}

	if err := h.db.UpdateHouseholdLocation(locID, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

// DeleteLocation 删除位置
func (h *HouseholdHandler) DeleteLocation(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	locID := c.Param("locId")
	if err := h.db.DeleteHouseholdLocation(locID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

// GetSpaceLayout 获取空间布局
func (h *HouseholdHandler) GetSpaceLayout(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	layout, err := h.db.GetHouseholdSpaceLayout(profileID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": layout,
	})
}

// SaveSpaceLayout 保存空间布局
func (h *HouseholdHandler) SaveSpaceLayout(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供布局内容", "code": 400})
		return
	}

	layout := &models.HouseholdSpaceLayout{
		ProfileID: profileID,
		Content:   req.Content,
	}
	if err := h.db.UpsertHouseholdSpaceLayout(layout); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

// CreateSpaceShare 创建空间分享
func (h *HouseholdHandler) CreateSpaceShare(c *gin.Context) {
	profileID := c.Param("id")
	if _, ok := h.requireProfileAccess(c, profileID); !ok {
		return
	}

	layout, err := h.db.GetHouseholdSpaceLayout(profileID)
	if err != nil || layout.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "暂无可分享布局", "code": 400})
		return
	}

	share := &models.HouseholdSpaceShare{
		ProfileID: profileID,
		Content:   layout.Content,
	}
	if err := h.db.CreateHouseholdSpaceShare(share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分享失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"share_id": share.ID,
		"share_url": fmt.Sprintf("/household/space?share=%s", share.ID),
	})
}

// GetSpaceShare 获取空间分享内容
func (h *HouseholdHandler) GetSpaceShare(c *gin.Context) {
	shareID := c.Param("shareId")
	share, err := h.db.GetHouseholdSpaceShare(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在", "code": 404})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": share,
	})
}

// UpdateProfileItem 更新档案物品
func (h *HouseholdHandler) UpdateProfileItem(c *gin.Context) {
	profileID := c.Param("id")
	_, existing, ok := h.requireProfileItemAccess(c)
	if !ok {
		return
	}

	var item struct {
		Name        string `json:"name"`
		Category    string `json:"category"`
		Quantity    int    `json:"quantity"`
		Unit        string `json:"unit"`
		MinQuantity int    `json:"min_quantity"`
		ExpiryDate  string `json:"expiry_date"`
		ExpiryDays  int    `json:"expiry_days"`
		Location    string `json:"location"`
		Notes       string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误", "code": 400})
		return
	}

	if item.Name != "" {
		existing.Name = item.Name
	}
	if item.Category != "" {
		existing.Category = item.Category
	}
	if item.Quantity > 0 {
		existing.Quantity = item.Quantity
	}
	if item.Unit != "" {
		existing.Unit = item.Unit
	}
	if item.MinQuantity > 0 {
		existing.MinQuantity = item.MinQuantity
	}
	if item.ExpiryDate != "" {
		existing.ExpiryDate = item.ExpiryDate
	}
	if item.ExpiryDays > 0 {
		existing.ExpiryDays = item.ExpiryDays
	}
	if item.Location != "" {
		existing.Location = item.Location
	}
	if item.Notes != "" {
		existing.Notes = item.Notes
	}

	if err := h.db.UpdateProfileItem(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	stats, _ := h.db.GetProfileStats(profileID)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  existing,
		"stats": stats,
	})
}

// DeleteProfileItem 删除档案物品
func (h *HouseholdHandler) DeleteProfileItem(c *gin.Context) {
	profileID := c.Param("id")
	_, item, ok := h.requireProfileItemAccess(c)
	if !ok {
		return
	}

	if err := h.db.DeleteProfileItem(item.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	stats, _ := h.db.GetProfileStats(profileID)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"stats": stats,
		"message": "删除成功",
	})
}

// UseProfileItem 使用物品
func (h *HouseholdHandler) UseProfileItem(c *gin.Context) {
	profileID := c.Param("id")
	_, item, ok := h.requireProfileItemAccess(c)
	if !ok {
		return
	}

	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供使用数量", "code": 400})
		return
	}

	if err := h.db.UseProfileItem(item.ID, req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	item, _ = h.db.GetProfileItem(item.ID)
	stats, _ := h.db.GetProfileStats(profileID)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  item,
		"stats": stats,
	})
}

// RestockProfileItem 补充物品
func (h *HouseholdHandler) RestockProfileItem(c *gin.Context) {
	profileID := c.Param("id")
	_, item, ok := h.requireProfileItemAccess(c)
	if !ok {
		return
	}

	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供补充数量", "code": 400})
		return
	}

	if err := h.db.RestockProfileItem(item.ID, req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	item, _ = h.db.GetProfileItem(item.ID)
	stats, _ := h.db.GetProfileStats(profileID)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  item,
		"stats": stats,
	})
}

// OpenProfileItem 重新开封
func (h *HouseholdHandler) OpenProfileItem(c *gin.Context) {
	profileID := c.Param("id")
	_, item, ok := h.requireProfileItemAccess(c)
	if !ok {
		return
	}

	if err := h.db.OpenProfileItem(item.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	item, _ = h.db.GetProfileItem(item.ID)
	stats, _ := h.db.GetProfileStats(profileID)
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  item,
		"stats": stats,
	})
}

// 物品管理 API

// CreateItem 创建物品
func (h *HouseholdHandler) CreateItem(c *gin.Context) {
	var item struct {
		Name        string `json:"name" binding:"required"`
		Category    string `json:"category"`
		Quantity    int    `json:"quantity"`
		Unit        string `json:"unit"`
		MinQuantity int    `json:"min_quantity"`
		ExpiryDate  string `json:"expiry_date"`
		ExpiryDays  int    `json:"expiry_days"`
		Location    string `json:"location"`
		Notes       string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供物品名称", "code": 400})
		return
	}

	if item.Category == "" {
		item.Category = "其他"
	}
	if item.Quantity == 0 {
		item.Quantity = 1
	}
	if item.Unit == "" {
		item.Unit = "个"
	}
	if item.MinQuantity == 0 {
		item.MinQuantity = 1
	}

	dbItem := &models.HouseholdItem{
		Name:        item.Name,
		Category:    item.Category,
		Quantity:    item.Quantity,
		Unit:        item.Unit,
		MinQuantity: item.MinQuantity,
		ExpiryDate:  item.ExpiryDate,
		ExpiryDays:  item.ExpiryDays,
		Location:    item.Location,
		Notes:       item.Notes,
	}

	if err := h.db.CreateHouseholdItem(dbItem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": dbItem,
	})
}

// GetItems 获取物品列表
func (h *HouseholdHandler) GetItems(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))
	location := strings.TrimSpace(c.Query("location"))
	alert := c.Query("alert") == "true"

	var items []*models.HouseholdItem
	var err error

	if alert {
		items, err = h.db.GetHouseholdAlerts()
	} else {
		items, err = h.db.GetAllHouseholdItems()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	// 分类过滤
	if category != "" {
		var filtered []*models.HouseholdItem
		for _, item := range items {
			if item.Category == category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	// 位置过滤
	if location != "" {
		var filtered []*models.HouseholdItem
		for _, item := range items {
			if item.Location == location {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	// 获取分类统计
	categories, _ := h.db.GetHouseholdCategories()
	locations, _ := h.db.GetHouseholdLocations()

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"data":       items,
		"categories": categories,
		"locations":  locations,
	})
}

// GetItem 获取单个物品
func (h *HouseholdHandler) GetItem(c *gin.Context) {
	id := c.Param("id")
	item, err := h.db.GetHouseholdItem(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "物品不存在", "code": 404})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": item,
	})
}

// UpdateItem 更新物品
func (h *HouseholdHandler) UpdateItem(c *gin.Context) {
	id := c.Param("id")

	var item struct {
		Name        string `json:"name"`
		Category    string `json:"category"`
		Quantity    int    `json:"quantity"`
		Unit        string `json:"unit"`
		MinQuantity int    `json:"min_quantity"`
		ExpiryDate  string `json:"expiry_date"`
		ExpiryDays  int    `json:"expiry_days"`
		Location    string `json:"location"`
		Notes       string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误", "code": 400})
		return
	}

	existing, err := h.db.GetHouseholdItem(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "物品不存在", "code": 404})
		return
	}

	if item.Name != "" {
		existing.Name = item.Name
	}
	if item.Category != "" {
		existing.Category = item.Category
	}
	if item.Quantity > 0 {
		existing.Quantity = item.Quantity
	}
	if item.Unit != "" {
		existing.Unit = item.Unit
	}
	if item.MinQuantity > 0 {
		existing.MinQuantity = item.MinQuantity
	}
	if item.ExpiryDate != "" {
		existing.ExpiryDate = item.ExpiryDate
	}
	if item.ExpiryDays > 0 {
		existing.ExpiryDays = item.ExpiryDays
	}
	if item.Location != "" {
		existing.Location = item.Location
	}
	if item.Notes != "" {
		existing.Notes = item.Notes
	}

	if err := h.db.UpdateHouseholdItem(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": existing,
	})
}

// DeleteItem 删除物品
func (h *HouseholdHandler) DeleteItem(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteHouseholdItem(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// UseItem 使用物品（减少数量）
func (h *HouseholdHandler) UseItem(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供使用数量", "code": 400})
		return
	}

	if err := h.db.UseHouseholdItem(id, req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	item, _ := h.db.GetHouseholdItem(id)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": item,
	})
}

// RestockItem 补充物品
func (h *HouseholdHandler) RestockItem(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供补充数量", "code": 400})
		return
	}

	if err := h.db.RestockHouseholdItem(id, req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	item, _ := h.db.GetHouseholdItem(id)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": item,
	})
}

// OpenItem 重新开封
func (h *HouseholdHandler) OpenItem(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.OpenHouseholdItem(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	item, _ := h.db.GetHouseholdItem(id)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": item,
	})
}

// 模板管理 API

// GetTemplates 获取物品模板
func (h *HouseholdHandler) GetTemplates(c *gin.Context) {
	templates, err := h.db.GetItemTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	// 按分类整理
	categoryMap := make(map[string][]*models.ItemTemplate)
	for _, t := range templates {
		categoryMap[t.Category] = append(categoryMap[t.Category], t)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"data":        templates,
		"by_category": categoryMap,
	})
}

// CreateTemplate 创建模板
func (h *HouseholdHandler) CreateTemplate(c *gin.Context) {
	var tpl struct {
		Name               string `json:"name" binding:"required"`
		Category           string `json:"category"`
		Unit               string `json:"unit"`
		DefaultMinQuantity int    `json:"default_min_quantity"`
		DefaultExpiryDays  int    `json:"default_expiry_days"`
	}

	if err := c.ShouldBindJSON(&tpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供模板名称", "code": 400})
		return
	}

	if tpl.Category == "" {
		tpl.Category = "其他"
	}
	if tpl.Unit == "" {
		tpl.Unit = "个"
	}

	template := &models.ItemTemplate{
		Name:               tpl.Name,
		Category:           tpl.Category,
		Unit:               tpl.Unit,
		DefaultMinQuantity: tpl.DefaultMinQuantity,
		DefaultExpiryDays:  tpl.DefaultExpiryDays,
		IsDefault:          false,
	}

	if err := h.db.CreateItemTemplate(template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": template,
	})
}

// DeleteTemplate 删除模板
func (h *HouseholdHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteItemTemplate(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// 通知管理 API

// GetNotifications 获取通知列表
func (h *HouseholdHandler) GetNotifications(c *gin.Context) {
	unreadOnly := c.Query("unread") == "true"

	notifs, err := h.db.GetNotifications(unreadOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	count, _ := h.db.GetUnreadNotificationCount()

	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"data":         notifs,
		"unread_count": count,
	})
}

// MarkNotificationAsRead 标记通知为已读
func (h *HouseholdHandler) MarkNotificationAsRead(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.MarkNotificationAsRead(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记成功",
	})
}

// MarkAllNotificationsAsRead 标记所有通知为已读
func (h *HouseholdHandler) MarkAllNotificationsAsRead(c *gin.Context) {
	if err := h.db.MarkAllNotificationsAsRead(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "标记成功",
	})
}

// GetStats 获取统计信息
func (h *HouseholdHandler) GetStats(c *gin.Context) {
	items, _ := h.db.GetAllHouseholdItems()
	notifs, _ := h.db.GetNotifications(false)

	// 统计各类数量
	total := len(items)
	lowStock := 0
	expiring := 0
	expired := 0

	now := time.Now()

	for _, item := range items {
		if item.Quantity <= item.MinQuantity {
			lowStock++
		}
		if item.ExpiryDate != "" && item.ExpiryDays > 0 {
			parsedDate, err := time.Parse("2006-01-02", item.ExpiryDate)
			if err == nil {
				expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
				daysUntil := int(expiryTime.Sub(now).Hours() / 24)
				if daysUntil <= 0 {
					expired++
				} else if daysUntil <= 7 {
					expiring++
				}
			}
		}
	}

	// 按分类统计
	categoryCount := make(map[string]int)
	for _, item := range items {
		categoryCount[item.Category]++
	}

	// 按位置统计
	locationCount := make(map[string]int)
	for _, item := range items {
		if item.Location != "" {
			locationCount[item.Location]++
		}
	}

	// 未读通知数
	unreadCount, _ := h.db.GetUnreadNotificationCount()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":         total,
			"low_stock":     lowStock,
			"expiring":      expiring,
			"expired":       expired,
			"notifications": len(notifs),
			"unread_count":  unreadCount,
			"by_category":   categoryCount,
			"by_location":   locationCount,
		},
	})
}

// ========== 待购买/提醒任务 API ==========

// CreateTodo 创建待办
func (h *HouseholdHandler) CreateTodo(c *gin.Context) {
	var req struct {
		ProfileID  string `json:"profile_id" binding:"required"`
		CreatorKey string `json:"creator_key"`
		Name       string `json:"name" binding:"required"`
		Category   string `json:"category"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供物品名称", "code": 400})
		return
	}

	profileID := strings.TrimSpace(req.ProfileID)
	creatorKey := strings.TrimSpace(req.CreatorKey)
	if creatorKey == "" {
		creatorKey = h.profileCreatorKey(c)
	}
	if _, ok := h.requireProfileAccessWithKey(c, profileID, creatorKey); !ok {
		return
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "其他"
	}

	todo := &models.HouseholdTodo{
		ProfileID: profileID,
		Name:      strings.TrimSpace(req.Name),
		Category:  category,
		Reason:    strings.TrimSpace(req.Reason),
		Status:    "open",
	}

	if err := h.db.CreateHouseholdTodo(todo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": todo,
	})
}

// GetTodos 获取待办列表
func (h *HouseholdHandler) GetTodos(c *gin.Context) {
	profileID := strings.TrimSpace(c.Query("profile_id"))
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少档案ID", "code": 400})
		return
	}
	creatorKey := strings.TrimSpace(c.Query("creator_key"))
	if creatorKey == "" {
		creatorKey = h.profileCreatorKey(c)
	}
	if _, ok := h.requireProfileAccessWithKey(c, profileID, creatorKey); !ok {
		return
	}

	status := strings.TrimSpace(c.Query("status"))
	todos, err := h.db.GetHouseholdTodos(profileID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": todos,
	})
}

// UpdateTodo 更新待办状态
func (h *HouseholdHandler) UpdateTodo(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		ProfileID  string `json:"profile_id" binding:"required"`
		CreatorKey string `json:"creator_key"`
		Status     string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误", "code": 400})
		return
	}

	profileID := strings.TrimSpace(req.ProfileID)
	creatorKey := strings.TrimSpace(req.CreatorKey)
	if creatorKey == "" {
		creatorKey = h.profileCreatorKey(c)
	}
	if _, ok := h.requireProfileAccessWithKey(c, profileID, creatorKey); !ok {
		return
	}

	todo, err := h.db.GetHouseholdTodo(id)
	if err != nil || todo.ProfileID != profileID {
		c.JSON(http.StatusNotFound, gin.H{"error": "待办不存在", "code": 404})
		return
	}

	status := strings.TrimSpace(req.Status)
	if status != "open" && status != "done" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "状态无效", "code": 400})
		return
	}

	if err := h.db.UpdateHouseholdTodoStatus(id, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

// DeleteTodo 删除待办
func (h *HouseholdHandler) DeleteTodo(c *gin.Context) {
	id := c.Param("id")
	profileID := strings.TrimSpace(c.Query("profile_id"))
	if profileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少档案ID", "code": 400})
		return
	}
	creatorKey := strings.TrimSpace(c.Query("creator_key"))
	if creatorKey == "" {
		creatorKey = h.profileCreatorKey(c)
	}
	if _, ok := h.requireProfileAccessWithKey(c, profileID, creatorKey); !ok {
		return
	}

	todo, err := h.db.GetHouseholdTodo(id)
	if err != nil || todo.ProfileID != profileID {
		c.JSON(http.StatusNotFound, gin.H{"error": "待办不存在", "code": 404})
		return
	}

	if err := h.db.DeleteHouseholdTodo(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

// 初始化 household 数据库
func (h *HouseholdHandler) Init(c *gin.Context) {
	if err := h.db.InitHousehold(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "初始化成功",
	})
}

// AI 智能分析和建议

// AIAnalyzeRequest AI 分析请求
type AIAnalyzeRequest struct {
	Text       string `json:"text"`
	ProfileID  string `json:"profile_id"`
	CreatorKey string `json:"creator_key"`
}

// AIAnalyzeResponse AI 分析响应
type AIAnalyzeResponse struct {
	Suggestions   []AISuggestion `json:"suggestions"`
	Analysis      string         `json:"analysis"`
	ShoppingList  []string       `json:"shopping_list"`
}

// AISuggestion AI 建议
type AISuggestion struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Quantity int    `json:"quantity"`
	Unit     string `json:"unit"`
	Reason   string `json:"reason"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	Message    string `json:"message" binding:"required"`
	ProfileID  string `json:"profile_id"`
	CreatorKey string `json:"creator_key"`
	ClearHistory bool `json:"clear_history"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	Reply      string      `json:"reply"`
	Actions    []ChatAction `json:"actions,omitempty"`
	ItemsAdded []string    `json:"items_added,omitempty"`
}

// ChatAction 对话动作
type ChatAction struct {
	Type   string `json:"type"` // add, restock, delete, query
	ItemID string `json:"item_id,omitempty"`
	Name   string `json:"name,omitempty"`
	Target string `json:"target,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Candidates []string `json:"candidates,omitempty"`
	Quantity    int    `json:"quantity,omitempty"`
	Category    string `json:"category,omitempty"`
	Unit        string `json:"unit,omitempty"`
	MinQuantity int    `json:"min_quantity,omitempty"`
	ExpiryDays  int    `json:"expiry_days,omitempty"`
	Location    string `json:"location,omitempty"`
}

// BarcodeLookupRequest 条码查询请求
type BarcodeLookupRequest struct {
	Barcode string `json:"barcode"`
}

// BarcodeLookupResponse 条码查询响应
type BarcodeLookupResponse struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Unit     string `json:"unit"`
}

// BarcodeLookup 条码查询（本地映射）
func (h *HouseholdHandler) BarcodeLookup(c *gin.Context) {
	var req BarcodeLookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供条码", "code": 400})
		return
	}

	// 本地条码映射表（常见商品条码）
	barcodeMap := map[string]map[string]string{
		"6901234567890": {"name": "农夫山泉", "category": "客厅", "unit": "瓶"},
		"6901234567891": {"name": "可口可乐", "category": "客厅", "unit": "瓶"},
		"6920202888888": {"name": "康师傅方便面", "category": "厨房", "unit": "袋"},
		"6954767420204": {"name": "维他柠檬茶", "category": "客厅", "unit": "瓶"},
		"6925307515058": {"name": "德芙巧克力", "category": "客厅", "unit": "块"},
		"489":           {"name": "可口可乐", "category": "客厅", "unit": "瓶"},
		"500":           {"name": "雀巢咖啡", "category": "客厅", "unit": "盒"},
		"400":           {"name": "雀巢咖啡", "category": "客厅", "unit": "盒"},
		"761":           {"name": "瑞士莲巧克力", "category": "客厅", "unit": "盒"},
	}

	// 尝试精确匹配
	if info, ok := barcodeMap[req.Barcode]; ok {
		c.JSON(http.StatusOK, gin.H{
			"code":     0,
			"name":     info["name"],
			"category": info["category"],
			"unit":     info["unit"],
		})
		return
	}

	// 尝试前缀匹配（常见的 EAN-13 前缀）
	prefix3 := req.Barcode[:min(3, len(req.Barcode))]
	prefixMap := map[string]map[string]string{
		"690": {"name": "食品", "category": "厨房", "unit": "个"},
		"691": {"name": "食品", "category": "厨房", "unit": "个"},
		"692": {"name": "食品", "category": "厨房", "unit": "个"},
		"693": {"name": "食品", "category": "厨房", "unit": "个"},
		"694": {"name": "食品", "category": "厨房", "unit": "个"},
		"695": {"name": "日用品", "category": "卫生间", "unit": "个"},
		"696": {"name": "日用品", "category": "卫生间", "unit": "个"},
		"697": {"name": "日用品", "category": "卫生间", "unit": "个"},
		"698": {"name": "日用品", "category": "卫生间", "unit": "个"},
		"699": {"name": "日用品", "category": "卫生间", "unit": "个"},
		"700": {"name": "食品", "category": "厨房", "unit": "个"},
		"800": {"name": "日用品", "category": "其他", "unit": "个"},
		"900": {"name": "日用品", "category": "其他", "unit": "个"},
	}

	if info, ok := prefixMap[prefix3]; ok {
		c.JSON(http.StatusOK, gin.H{
			"code":     0,
			"name":     info["name"],
			"category": info["category"],
			"unit":     info["unit"],
		})
		return
	}

	// 未找到
	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"name":  "",
	})
}

// ReceiptOCRRequest 小票识别请求
type ReceiptOCRRequest struct {
	Image      string `json:"image" binding:"required"`
	ProfileID  string `json:"profile_id"`
	CreatorKey string `json:"creator_key"`
}

// ReceiptItem 小票商品项
type ReceiptItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    string `json:"price,omitempty"`
	Category string `json:"category,omitempty"`
	Unit     string `json:"unit"`
	Matched  bool   `json:"matched"`
}

// AITodoMergeRequest AI 去重合并待办请求
type AITodoMergeRequest struct {
	ProfileID  string `json:"profile_id" binding:"required"`
	CreatorKey string `json:"creator_key"`
}

// AITodoMergeResponse AI 去重合并待办响应
type AITodoMergeResponse struct {
	Merges []AITodoMerge `json:"merges"`
	Notes  string        `json:"notes"`
}

// AITodoMerge AI 合并项
type AITodoMerge struct {
	KeepID   string   `json:"keep_id"`
	MergeIDs []string `json:"merge_ids"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Reason   string   `json:"reason"`
}

// ReceiptOCR 小票 OCR 识别
func (h *HouseholdHandler) ReceiptOCR(c *gin.Context) {
	var req ReceiptOCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供图片", "code": 400})
		return
	}

	// 验证档案访问权限
	profileID := strings.TrimSpace(req.ProfileID)
	if profileID != "" {
		if _, ok := h.requireProfileAccessWithKey(c, profileID, strings.TrimSpace(req.CreatorKey)); !ok {
			return
		}
	}

	// 调用 OCR 服务识别图片
	ocrURL := os.Getenv("OCR_SERVICE_URL")
	if ocrURL == "" {
		ocrURL = "http://ocr-service:8000"
	}

	// 移除 data:image/xxx;base64, 前缀
	imageData := req.Image
	if strings.Contains(imageData, ",") {
		imageData = strings.Split(imageData, ",")[1]
	}

	ocrReq := map[string]string{"image": imageData}
	ocrBody, _ := json.Marshal(ocrReq)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(ocrURL+"/ocr", "application/json", bytes.NewReader(ocrBody))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OCR 服务不可用", "code": 503})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 解析 OCR 响应
	var ocrResult map[string]interface{}
	if err := json.Unmarshal(respBody, &ocrResult); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCR 解析失败", "code": 500})
		return
	}

	// 提取文字内容
	var textLines []string
	if texts, ok := ocrResult["texts"].([]interface{}); ok {
		for _, t := range texts {
			if line, ok := t.(map[string]interface{}); ok {
				if text, ok := line["text"].(string); ok && strings.TrimSpace(text) != "" {
					textLines = append(textLines, text)
				}
			}
		}
	}

	// 如果没有 texts 字段，尝试其他格式
	if len(textLines) == 0 {
		if resultText, ok := ocrResult["text"].(string); ok {
			textLines = strings.Split(resultText, "\n")
		}
	}

	// 获取现有物品用于匹配
	var existingItems []*models.HouseholdItem
	if profileID != "" {
		profileItems, _ := h.db.GetProfileItems(profileID)
		existingItems = profileItemsToHousehold(profileItems)
	} else {
		existingItems, _ = h.db.GetAllHouseholdItems()
	}

	// 构建物品名称映射用于匹配
	existingNames := make(map[string]bool)
	for _, item := range existingItems {
		existingNames[item.Name] = true
		// 也添加不带数量的变体
		name := strings.TrimSpace(item.Name)
		if name != "" {
			existingNames[name] = true
		}
	}

	// 解析商品列表
	items := h.parseReceiptItems(textLines, existingNames)

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"items": items,
		"raw":   strings.Join(textLines, "\n"),
	})
}

// parseReceiptItems 解析小票文本，提取商品列表
func (h *HouseholdHandler) parseReceiptItems(textLines []string, existingNames map[string]bool) []ReceiptItem {
	var items []ReceiptItem

	// 常见商品关键词（用于过滤）
	keywords := []string{"商品", "产品", "名称", "品名", "项目", "item", "name"}

	// 小票价格模式
	pricePattern := `\d+[.,]?\d*$`

	for _, line := range textLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 跳过标题行和总计行
		isSkip := false
		for _, kw := range keywords {
			if strings.Contains(line, kw) && !strings.Contains(line, "合计") && !strings.Contains(line, "总计") {
				isSkip = true
				break
			}
		}
		if isSkip {
			continue
		}

		// 跳过金额行（包含"合计"、"总计"、"应收"、"实收"、"找零"等）
		if strings.Contains(line, "合计") || strings.Contains(line, "总计") ||
			strings.Contains(line, "应收") || strings.Contains(line, "实收") ||
			strings.Contains(line, "找零") || strings.Contains(line, "收款") ||
			strings.Contains(line, "人民币") || strings.Contains(line, "现金") ||
			strings.Contains(line, "支付") || strings.Contains(line, "时间") ||
			strings.Contains(line, "日期") || strings.Contains(line, "流水号") ||
			strings.Contains(line, "电话") || strings.Contains(line, "地址") ||
			strings.Contains(line, "会员") || strings.Contains(line, "积分") ||
			strings.Contains(line, "优惠") || strings.Contains(line, "折扣") {
			continue
		}

		// 解析商品行
		// 尝试匹配 "商品名称 数量 单价" 或 "商品名称 x数量 价格" 格式

		// 提取价格（通常是行末的数字）
		var quantity int = 1
		var price string
		var name string

		// 尝试匹配 "名称 x数量" 或 "名称 数量" 格式
		xMatch := regexp.MustCompile(`(.+?)\s*[xX×]\s*(\d+)\s*$`).FindStringSubmatch(line)
		if xMatch != nil {
			name = strings.TrimSpace(xMatch[1])
			fmt.Sscanf(xMatch[2], "%d", &quantity)
			// 提取价格（如果有）
			priceMatch := regexp.MustCompile(pricePattern).FindStringSubmatch(line)
			if priceMatch != nil {
				price = priceMatch[0]
			}
		} else {
			// 尝试匹配 "名称 数量" 格式（数量在前）
			spaceParts := strings.Fields(line)
			if len(spaceParts) >= 2 {
				// 检查最后一部分是否是数字
				if lastNum, err := strconv.Atoi(spaceParts[len(spaceParts)-1]); err == nil && lastNum > 0 && lastNum < 1000 {
					quantity = lastNum
					name = strings.TrimSpace(strings.Join(spaceParts[:len(spaceParts)-1], " "))
				} else {
					name = line
				}
			} else {
				name = line
			}
		}

		// 清理名称
		name = strings.TrimSpace(name)
		// 移除可能的金额后缀
		name = regexp.MustCompile(pricePattern).ReplaceAllString(name, "")
		name = strings.TrimSpace(name)

		// 跳过太短的名称
		if len(name) < 2 {
			continue
		}

		// 推断分类和单位
		category := "其他"
		unit := "个"

		// 根据名称关键词推断分类
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "水") || strings.Contains(lowerName, "饮料") || strings.Contains(lowerName, "可乐") || strings.Contains(lowerName, "奶茶") || strings.Contains(lowerName, "果汁") {
			category = "客厅"
			unit = "瓶"
		} else if strings.Contains(lowerName, "方便面") || strings.Contains(lowerName, "面条") || strings.Contains(lowerName, "米") || strings.Contains(lowerName, "油") || strings.Contains(lowerName, "盐") || strings.Contains(lowerName, "糖") || strings.Contains(lowerName, "酱") || strings.Contains(lowerName, "醋") || strings.Contains(lowerName, "酱油") || strings.Contains(lowerName, "味精") {
			category = "厨房"
			unit = "袋"
		} else if strings.Contains(lowerName, "纸巾") || strings.Contains(lowerName, "纸") || strings.Contains(lowerName, "卫生") {
			category = "卫生间"
			unit = "包"
		} else if strings.Contains(lowerName, "洗衣") || strings.Contains(lowerName, "洗洁") || strings.Contains(lowerName, "洗手") || strings.Contains(lowerName, "沐浴") || strings.Contains(lowerName, "洗发") {
			category = "卫生间"
			unit = "瓶"
		} else if strings.Contains(lowerName, "口罩") || strings.Contains(lowerName, "酒精") || strings.Contains(lowerName, "消毒") {
			category = "其他"
			unit = "个"
		}

		// 检查是否与现有物品匹配
		matched := existingNames[name]

		items = append(items, ReceiptItem{
			Name:     name,
			Quantity: quantity,
			Price:    price,
			Category: category,
			Unit:     unit,
			Matched:  matched,
		})
	}

	// 限制返回数量
	if len(items) > 50 {
		items = items[:50]
	}

	return items
}

// AIFeatureCheck 检查 AI 功能是否可用
func (h *HouseholdHandler) AIFeatureCheck(c *gin.Context) {
	enabled := h.cfg.DeepSeek.APIKey != "" || h.cfg.MiniMax.APIKey != ""
	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"enabled":  enabled,
		"provider": getAIProvider(h.cfg),
	})
}

// AIMergeTodos AI 去重合并待购买任务
func (h *HouseholdHandler) AIMergeTodos(c *gin.Context) {
	if h.cfg.DeepSeek.APIKey == "" && h.cfg.MiniMax.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 AI，请配置 DeepSeek 或 MiniMax API Key", "code": 400})
		return
	}

	var req AITodoMergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少档案ID", "code": 400})
		return
	}

	profileID := strings.TrimSpace(req.ProfileID)
	creatorKey := strings.TrimSpace(req.CreatorKey)
	if creatorKey == "" {
		creatorKey = h.profileCreatorKey(c)
	}
	if _, ok := h.requireProfileAccessWithKey(c, profileID, creatorKey); !ok {
		return
	}

	todos, err := h.db.GetHouseholdTodos(profileID, "open")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取待办失败", "code": 500})
		return
	}
	if len(todos) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"merged":  0,
			"message": "无可合并任务",
		})
		return
	}

	var sb strings.Builder
	sb.WriteString("你是一个家庭物品待购买任务去重助手。请识别重复或可合并的任务，并给出合并建议。\n")
	sb.WriteString("只合并高度相似或明显重复的项；不要合并无关物品。\n")
	sb.WriteString("输出 JSON 格式：\n")
	sb.WriteString(`{
  "notes": "简短说明",
  "merges": [
    {"keep_id": "保留的任务ID", "merge_ids": ["需要合并的任务ID1", "任务ID2"], "name": "合并后的名称", "category": "分类", "reason": "合并后的原因"}
  ]
}`)
	sb.WriteString("\n\n任务列表：\n")
	for _, t := range todos {
		sb.WriteString(fmt.Sprintf("- id:%s | name:%s | category:%s | reason:%s\n", t.ID, t.Name, t.Category, t.Reason))
	}

	var result string
	var aiErr error
	if h.cfg.MiniMax.APIKey != "" {
		result, aiErr = h.callMiniMaxAPI(sb.String())
	} else {
		result, aiErr = h.callDeepSeekAPI(sb.String())
	}
	if aiErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 合并失败: " + aiErr.Error(), "code": 500})
		return
	}

	candidate, ok := extractJSONPayload(result)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 返回格式错误", "code": 400, "raw": result})
		return
	}

	var resp AITodoMergeResponse
	if err := json.Unmarshal([]byte(candidate), &resp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 返回格式错误", "code": 400, "raw": result})
		return
	}

	todoMap := make(map[string]*models.HouseholdTodo, len(todos))
	for _, t := range todos {
		todoMap[t.ID] = t
	}

	mergedCount := 0
	updated := 0
	for _, merge := range resp.Merges {
		if merge.KeepID == "" || len(merge.MergeIDs) == 0 {
			continue
		}
		keep, ok := todoMap[merge.KeepID]
		if !ok {
			continue
		}
		if strings.TrimSpace(merge.Name) != "" {
			keep.Name = strings.TrimSpace(merge.Name)
		}
		if strings.TrimSpace(merge.Category) != "" {
			keep.Category = strings.TrimSpace(merge.Category)
		}
		if strings.TrimSpace(merge.Reason) != "" {
			keep.Reason = strings.TrimSpace(merge.Reason)
		}
		if err := h.db.UpdateHouseholdTodo(keep); err == nil {
			updated++
		}

		for _, id := range merge.MergeIDs {
			if id == keep.ID {
				continue
			}
			if todo, exists := todoMap[id]; !exists || todo.ProfileID != profileID {
				continue
			}
			if err := h.db.DeleteHouseholdTodo(id); err == nil {
				mergedCount++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"merged":  mergedCount,
		"updated": updated,
		"notes":   resp.Notes,
	})
}

// AIAnalyze 智能分析库存并给出建议
func (h *HouseholdHandler) AIAnalyze(c *gin.Context) {
	if h.cfg.DeepSeek.APIKey == "" && h.cfg.MiniMax.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 AI，请配置 DeepSeek 或 MiniMax API Key", "code": 400})
		return
	}

	var req AIAnalyzeRequest
	_ = c.ShouldBindJSON(&req)

	var (
		items  []*models.HouseholdItem
		alerts []*models.HouseholdItem
		err    error
	)
	if req.ProfileID != "" {
		if _, ok := h.requireProfileAccessWithKey(c, req.ProfileID, strings.TrimSpace(req.CreatorKey)); !ok {
			return
		}
		profileItems, profileErr := h.db.GetProfileItems(req.ProfileID)
		if profileErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取物品失败", "code": 500})
			return
		}
		profileAlerts, _ := h.db.GetProfileAlerts(req.ProfileID)
		items = profileItemsToHousehold(profileItems)
		alerts = profileItemsToHousehold(profileAlerts)
	} else {
		items, err = h.db.GetAllHouseholdItems()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取物品失败", "code": 500})
			return
		}
		alerts, err = h.db.GetHouseholdAlerts()
		if err != nil {
			alerts = []*models.HouseholdItem{}
		}
	}

	// 构建分析提示词
	prompt := h.buildAnalysisPrompt(items, alerts)

	var result string
	var aiErr error

	if h.cfg.MiniMax.APIKey != "" {
		result, aiErr = h.callMiniMaxAPI(prompt)
	} else {
		result, aiErr = h.callDeepSeekAPI(prompt)
	}

	if aiErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 分析失败: " + aiErr.Error(), "code": 500})
		return
	}

	// 解析 AI 返回的 JSON
	var resp AIAnalyzeResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":     0,
			"analysis": result,
			"ai_error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"suggestions":  resp.Suggestions,
		"analysis":     resp.Analysis,
		"shopping_list": resp.ShoppingList,
	})
}

// AIAddItem 智能添加物品（从自然语言解析）
func (h *HouseholdHandler) AIAddItem(c *gin.Context) {
	if h.cfg.DeepSeek.APIKey == "" && h.cfg.MiniMax.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 AI，请配置 DeepSeek 或 MiniMax API Key", "code": 400})
		return
	}

	var req AIAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供物品描述", "code": 400})
		return
	}

	// 获取模板作为参考
	templates, _ := h.db.GetItemTemplates()

	// 构建解析提示词
	prompt := h.buildParsePrompt(req.Text, templates)

	var result string
	var aiErr error

	if h.cfg.MiniMax.APIKey != "" {
		result, aiErr = h.callMiniMaxAPI(prompt)
	} else {
		result, aiErr = h.callDeepSeekAPI(prompt)
	}

	if aiErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 解析失败: " + aiErr.Error(), "code": 500})
		return
	}

	// 解析 AI 返回的 JSON
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 返回格式错误", "code": 400, "raw": result})
		return
	}

	// 创建物品
	created := make([]interface{}, 0)
	useProfile := strings.TrimSpace(req.ProfileID) != ""
	if useProfile {
		if _, ok := h.requireProfileAccessWithKey(c, req.ProfileID, strings.TrimSpace(req.CreatorKey)); !ok {
			return
		}
	}
	for _, itemData := range items {
		category := getString(itemData, "category")
		if category == "" {
			category = "其他"
		}
		unit := getString(itemData, "unit")
		if unit == "" {
			unit = "个"
		}

		if useProfile {
			item := &models.ProfileItem{
				ProfileID:   req.ProfileID,
				Name:        getString(itemData, "name"),
				Category:    category,
				Quantity:    getInt(itemData, "quantity", 1),
				Unit:        unit,
				MinQuantity: getInt(itemData, "min_quantity", 1),
				ExpiryDays:  getInt(itemData, "expiry_days", 0),
				Location:    getString(itemData, "location"),
			}
			if item.Name != "" {
				if err := h.db.CreateProfileItem(item); err == nil {
					created = append(created, item)
				}
			}
			continue
		}

		item := &models.HouseholdItem{
			Name:         getString(itemData, "name"),
			Category:     category,
			Quantity:     getInt(itemData, "quantity", 1),
			Unit:         unit,
			MinQuantity:  getInt(itemData, "min_quantity", 1),
			ExpiryDays:   getInt(itemData, "expiry_days", 0),
			Location:     getString(itemData, "location"),
		}

		if item.Name != "" {
			if err := h.db.CreateHouseholdItem(item); err == nil {
				created = append(created, item)
			}
		}
	}

	if len(created) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未能识别出有效物品，请重新描述", "code": 400})
		return
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"data":      created,
		"count":     len(created),
		"raw_ai":    result,
	})
}

// AIParseItems AI 解析物品清单（不直接写入）
func (h *HouseholdHandler) AIParseItems(c *gin.Context) {
	if h.cfg.DeepSeek.APIKey == "" && h.cfg.MiniMax.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 AI，请配置 DeepSeek 或 MiniMax API Key", "code": 400})
		return
	}

	var req AIAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供物品描述", "code": 400})
		return
	}

	if strings.TrimSpace(req.ProfileID) != "" {
		if _, ok := h.requireProfileAccessWithKey(c, strings.TrimSpace(req.ProfileID), strings.TrimSpace(req.CreatorKey)); !ok {
			return
		}
	}

	templates, _ := h.db.GetItemTemplates()
	prompt := h.buildParsePrompt(req.Text, templates)

	var result string
	var aiErr error
	if h.cfg.MiniMax.APIKey != "" {
		result, aiErr = h.callMiniMaxAPI(prompt)
	} else {
		result, aiErr = h.callDeepSeekAPI(prompt)
	}
	if aiErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 解析失败: " + aiErr.Error(), "code": 500})
		return
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 返回格式错误", "code": 400, "raw": result})
		return
	}

	normalized := make([]map[string]interface{}, 0, len(items))
	for _, itemData := range items {
		name := getString(itemData, "name")
		if name == "" {
			continue
		}
		category := getString(itemData, "category")
		if category == "" {
			category = "其他"
		}
		unit := getString(itemData, "unit")
		if unit == "" {
			unit = "个"
		}
		normalized = append(normalized, map[string]interface{}{
			"name":         name,
			"category":     category,
			"quantity":     getInt(itemData, "quantity", 1),
			"unit":         unit,
			"min_quantity": getInt(itemData, "min_quantity", 1),
			"expiry_days":  getInt(itemData, "expiry_days", 0),
			"location":     getString(itemData, "location"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"items":  normalized,
		"count":  len(normalized),
		"raw_ai": result,
	})
}

// AISuggestRestock AI 推荐需要补充的物品
func (h *HouseholdHandler) AISuggestRestock(c *gin.Context) {
	profileID := strings.TrimSpace(c.Query("profile_id"))
	var (
		items []*models.HouseholdItem
		err   error
	)
	if profileID != "" {
		if _, ok := h.requireProfileAccess(c, profileID); !ok {
			return
		}
		profileItems, profileErr := h.db.GetProfileAlerts(profileID)
		if profileErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取物品失败", "code": 500})
			return
		}
		items = profileItemsToHousehold(profileItems)
	} else {
		items, err = h.db.GetHouseholdAlerts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取物品失败", "code": 500})
			return
		}
	}

	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":        0,
			"suggestions": []map[string]interface{}{},
			"message":     "当前库存充足，无需补充",
		})
		return
	}

	// 直接返回库存不足的物品
	suggestions := make([]map[string]interface{}, 0)
	for _, item := range items {
		reason := ""
		if item.Quantity <= item.MinQuantity {
			reason = fmt.Sprintf("库存不足（当前 %d%s，需要 %d%s）", item.Quantity, item.Unit, item.MinQuantity, item.Unit)
		}
		if item.ExpiryDate != "" && item.ExpiryDays > 0 {
			parsedDate, _ := time.Parse("2006-01-02", item.ExpiryDate)
			expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
			daysUntil := int(expiryTime.Sub(time.Now()).Hours() / 24)
			if daysUntil <= 0 {
				reason = "已过期"
			} else if daysUntil <= 7 {
				reason = fmt.Sprintf("即将过期（还剩 %d 天）", daysUntil)
			}
		}

		suggestions = append(suggestions, map[string]interface{}{
			"id":       item.ID,
			"name":     item.Name,
			"category": item.Category,
			"quantity": item.Quantity,
			"unit":     item.Unit,
			"reason":   reason,
			"action":   "restock",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

// 构建分析提示词
func (h *HouseholdHandler) buildAnalysisPrompt(items []*models.HouseholdItem, alerts []*models.HouseholdItem) string {
	var sb strings.Builder

	sb.WriteString("你是一个智能家庭物品管理助手。请分析以下家庭物品库存状况，并给出建议。\n\n")

	sb.WriteString("## 当前物品清单：\n")
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("- %s: 数量 %d%s, 最低库存 %d%s, 位置 %s, 过期信息: %s\n",
			item.Name, item.Quantity, item.Unit, item.MinQuantity, item.Unit, item.Location, formatExpiry(item)))
	}

	sb.WriteString("\n## 需要关注的物品（库存不足或即将过期）：\n")
	for _, item := range alerts {
		sb.WriteString(fmt.Sprintf("- %s: 数量 %d%s\n", item.Name, item.Quantity, item.Unit))
	}

	sb.WriteString("\n请以 JSON 格式返回分析结果，格式如下：\n")
	sb.WriteString(`{
  "analysis": "简要分析当前库存状况",
  "shopping_list": ["需要购买的物品1", "需要购买的物品2"],
  "suggestions": [
    {"name": "物品名称", "category": "分类", "quantity": 1, "unit": "个", "reason": "建议原因"}
  ]
}`)

	return sb.String()
}

// 构建解析提示词
func (h *HouseholdHandler) buildParsePrompt(text string, templates []*models.ItemTemplate) string {
	var sb strings.Builder

	sb.WriteString("请从以下自然语言描述中识别出要添加的家庭物品，并返回 JSON 数组格式。\n\n")
	sb.WriteString(fmt.Sprintf("描述文本: %s\n\n", text))

	sb.WriteString("## 可参考的分类和物品模板：\n")
	for _, t := range templates {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", t.Name, t.Category))
	}

	sb.WriteString("\n请返回 JSON 数组格式：\n")
	sb.WriteString(`[
  {"name": "物品名称", "category": "分类", "quantity": 1, "unit": "个", "min_quantity": 1, "expiry_days": 0, "location": "存放位置"}
]`)

	sb.WriteString("\n注意：\n")
	sb.WriteString("- 请根据物品类型推断合适的分类（厨房/卫生间/卧室/客厅/玄关/阳台/其他）\n")
	sb.WriteString("- 请根据物品类型推断合适的单位（瓶/袋/盒/个/包/卷等）\n")
	sb.WriteString("- 请根据物品类型推断合适的最低库存数量\n")
	sb.WriteString("- 请根据物品类型推断合适的保质期天数（食品类通常30-730天，日用品通常365-1095天）\n")
	sb.WriteString("- 请根据物品类型推断合适的存放位置\n")
	sb.WriteString("- 如果无法识别任何物品，返回空数组 []\n")

	return sb.String()
}

// 调用 DeepSeek API
func (h *HouseholdHandler) callDeepSeekAPI(prompt string) (string, error) {
	apiKey := h.cfg.DeepSeek.APIKey
	model := h.cfg.DeepSeek.Model
	if model == "" {
		model = "deepseek-chat"
	}

	url := "https://api.deepseek.com/chat/completions"
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个智能家庭物品管理助手，擅长识别物品信息和分析库存状况。请始终返回有效的 JSON 格式。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("API响应格式错误")
	}

	firstChoice, _ := choices[0].(map[string]interface{})
	message, _ := firstChoice["message"].(map[string]interface{})
	content, _ := message["content"].(string)

	return content, nil
}

// 调用 MiniMax API
func (h *HouseholdHandler) callMiniMaxAPI(prompt string) (string, error) {
	apiKey := h.cfg.MiniMax.APIKey
	model := h.cfg.MiniMax.Model
	if model == "" {
		model = "abab6.5s-chat"
	}

	url := "https://api.minimax.chat/v1/text/chatcompletion_v2"
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个智能家庭物品管理助手，擅长识别物品信息和分析库存状况。请始终返回有效的 JSON 格式。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("API响应格式错误")
	}

	firstChoice, _ := choices[0].(map[string]interface{})
	message, _ := firstChoice["message"].(map[string]interface{})
	content, _ := message["content"].(string)

	return content, nil
}

// 辅助函数
func formatExpiry(item *models.HouseholdItem) string {
	if item.ExpiryDate == "" {
		return "无"
	}
	if item.ExpiryDays == 0 {
		return item.ExpiryDate
	}
	parsedDate, err := time.Parse("2006-01-02", item.ExpiryDate)
	if err != nil {
		return item.ExpiryDate
	}
	expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
	daysUntil := int(expiryTime.Sub(time.Now()).Hours() / 24)
	if daysUntil < 0 {
		return fmt.Sprintf("%s (已过期 %d 天)", item.ExpiryDate, -daysUntil)
	}
	return fmt.Sprintf("%s (还剩 %d 天)", item.ExpiryDate, daysUntil)
}

func getAIProvider(cfg *config.Config) string {
	if cfg.MiniMax.APIKey != "" {
		return "minimax"
	}
	if cfg.DeepSeek.APIKey != "" {
		return "deepseek"
	}
	return ""
}

func profileItemsToHousehold(items []*models.ProfileItem) []*models.HouseholdItem {
	result := make([]*models.HouseholdItem, 0, len(items))
	for _, item := range items {
		result = append(result, &models.HouseholdItem{
			ID:          item.ID,
			Name:        item.Name,
			Category:    item.Category,
			Quantity:    item.Quantity,
			Unit:        item.Unit,
			MinQuantity: item.MinQuantity,
			ExpiryDate:  item.ExpiryDate,
			ExpiryDays:  item.ExpiryDays,
			Location:    item.Location,
			Notes:       item.Notes,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

// hashPassword 使用 SHA256 哈希密码
func hashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

// Chat 对话接口
func (h *HouseholdHandler) Chat(c *gin.Context) {
	if h.cfg.DeepSeek.APIKey == "" && h.cfg.MiniMax.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 AI，请配置 DeepSeek 或 MiniMax API Key", "code": 400})
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供对话内容", "code": 400})
		return
	}

	// 初始化对话表
	_ = h.db.InitHouseholdConversations()

	profileID := strings.TrimSpace(req.ProfileID)
	userID := "default" // 默认用户

	// 验证档案访问权限
	if profileID != "" {
		if _, ok := h.requireProfileAccessWithKey(c, profileID, strings.TrimSpace(req.CreatorKey)); !ok {
			return
		}
	}

	// 清除历史记录
	if req.ClearHistory {
		_ = h.db.ClearConversations(profileID, userID)
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"reply":   "对话历史已清除",
			"actions": []ChatAction{},
		})
		return
	}

	// 获取对话历史
	history, _ := h.db.GetConversations(profileID, userID, 20)

	// 保存用户消息
	userMsg := &models.HouseholdConversation{
		ProfileID: profileID,
		UserID:   userID,
		Role:     "user",
		Content:  req.Message,
	}
	_ = h.db.SaveConversation(userMsg)

	// 获取当前物品列表
	var items []*models.HouseholdItem
	var err error
	if profileID != "" {
		profileItems, _ := h.db.GetProfileItems(profileID)
		items = profileItemsToHousehold(profileItems)
	} else {
		items, err = h.db.GetAllHouseholdItems()
		if err != nil {
			items = []*models.HouseholdItem{}
		}
	}

	// 获取可用位置列表
	var locations []string
	if profileID != "" {
		itemLocations, _ := h.db.GetProfileLocations(profileID)
		library, _ := h.db.GetHouseholdLocationsLibrary(profileID)
		seen := make(map[string]bool)
		for _, loc := range itemLocations {
			if loc == "" || seen[loc] {
				continue
			}
			seen[loc] = true
			locations = append(locations, loc)
		}
		for _, loc := range library {
			if loc.Name == "" || seen[loc.Name] {
				continue
			}
			seen[loc.Name] = true
			locations = append(locations, loc.Name)
		}
	}

	// 构建对话提示词
	prompt := h.buildChatPrompt(req.Message, items, history, locations)

	var result string
	var aiErr error

	if h.cfg.MiniMax.APIKey != "" {
		result, aiErr = h.callMiniMaxChatAPI(prompt, history)
	} else {
		result, aiErr = h.callDeepSeekChatAPI(prompt, history)
	}

	if aiErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 对话失败: " + aiErr.Error(), "code": 500})
		return
	}

	// 解析 AI 返回的 JSON
	resp, ok := parseChatResponse(result)
	if !ok {
		resp = ChatResponse{Reply: result}
	}

	// 保存助手回复
	assistantMsg := &models.HouseholdConversation{
		ProfileID: profileID,
		UserID:   userID,
		Role:     "assistant",
		Content:  resp.Reply,
	}
	_ = h.db.SaveConversation(assistantMsg)

	// 执行动作
	itemsAdded := h.executeChatActions(resp.Actions, profileID)

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"reply":      resp.Reply,
		"actions":    resp.Actions,
		"items_added": itemsAdded,
	})
}

// GetChatHistory 获取对话历史
func (h *HouseholdHandler) GetChatHistory(c *gin.Context) {
	profileID := strings.TrimSpace(c.Query("profile_id"))
	userID := "default"

	_ = h.db.InitHouseholdConversations()

	history, _ := h.db.GetConversations(profileID, userID, 50)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"history": history,
	})
}

// ClearChatHistory 清除对话历史
func (h *HouseholdHandler) ClearChatHistory(c *gin.Context) {
	profileID := strings.TrimSpace(c.Query("profile_id"))
	userID := "default"

	_ = h.db.InitHouseholdConversations()
	_ = h.db.ClearConversations(profileID, userID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "对话历史已清除",
	})
}

// 构建对话提示词
func (h *HouseholdHandler) buildChatPrompt(message string, items []*models.HouseholdItem, history []*models.HouseholdConversation, locations []string) string {
	var sb strings.Builder

	sb.WriteString("你是一个智能家庭物品管理助手。你可以帮用户：\n")
	sb.WriteString("1. 添加物品 - 用户说\"帮我买/添加/家里需要\"时\n")
	sb.WriteString("2. 查询物品 - 用户说\"家里有什么/还有xx吗/看看库存\"时\n")
	sb.WriteString("3. 推荐补货 - 用户说\"有什么要买的/缺什么\"时\n")
	sb.WriteString("4. 删除物品 - 用户说\"不要xx/删除xx\"时\n")
	sb.WriteString("5. 修改物品 - 用户说\"xx改成yy/把xx数量改一下\"时\n")
	sb.WriteString("6. 使用/消耗物品 - 用户说\"用掉/消耗\"时\n")
	sb.WriteString("7. 创建待购买任务 - 用户说\"加入待购/加入清单\"时\n")
	sb.WriteString("8. 日常问答 - 用户问其他问题\n\n")

	sb.WriteString("## 当前物品清单：\n")
	if len(items) == 0 {
		sb.WriteString("暂无物品\n")
	} else {
		for _, item := range items {
			status := ""
			if item.Quantity <= item.MinQuantity {
				status = " [库存不足]"
			}
			sb.WriteString(fmt.Sprintf("- %s: %d%s, 分类:%s, 位置:%s%s\n",
				item.Name, item.Quantity, item.Unit, item.Category, item.Location, status))
		}
	}

	// 添加对话历史
	if len(history) > 0 {
		sb.WriteString("\n## 对话历史：\n")
		for _, h := range history {
			sb.WriteString(fmt.Sprintf("%s: %s\n", h.Role, h.Content))
		}
	}

	if len(locations) > 0 {
		sb.WriteString("\n## 可用位置候选：\n")
		for _, loc := range locations {
			sb.WriteString(fmt.Sprintf("- %s\n", loc))
		}
	}

	sb.WriteString(fmt.Sprintf("\n## 用户消息：%s\n\n", message))

	sb.WriteString("请以 JSON 格式返回，格式如下：\n")
	sb.WriteString(`{
  "reply": "你的回复（友好、简洁）",
  "actions": [
    {"type": "add", "name": "物品名", "quantity": 1, "category": "分类", "unit": "个"},
    {"type": "query", "target": "物品名"},
    {"type": "restock", "item_id": "物品ID", "quantity": 1},
    {"type": "use", "item_id": "物品ID", "quantity": 1},
    {"type": "update", "item_id": "物品ID", "quantity": 1, "min_quantity": 1},
    {"type": "suggest_location", "name": "物品名", "candidates": ["位置1", "位置2"]},
    {"type": "todo", "name": "物品名", "category": "分类", "reason": "原因"},
    {"type": "delete", "item_id": "物品ID"},
    {"type": "delete", "name": "物品名"}
  ]
}`)

	sb.WriteString("\n注意：\n")
	sb.WriteString("- actions 为空数组表示仅回复用户问题\n")
	sb.WriteString("- 如果是添加物品，请提供合适的分类（厨房/卫生间/卧室/客厅/玄关/阳台/其他）、单位（瓶/袋/盒/个/包/卷等）\n")
	sb.WriteString("- 如果是删除/修改，请先查找对应物品\n")
	sb.WriteString("- 如果位置不确定，可返回 suggest_location 动作，候选要简短清晰\n")
	sb.WriteString("- 如果用户要加入待购/清单，请使用 todo 动作\n")
	sb.WriteString("- reply 应该friendly，根据用户意图给出合适回复\n")

	return sb.String()
}

// 执行对话动作
func (h *HouseholdHandler) executeChatActions(actions []ChatAction, profileID string) []string {
	itemsAdded := make([]string, 0)

	for _, action := range actions {
		switch action.Type {
		case "add":
			if action.Name == "" {
				continue
			}

			category := action.Category
			if category == "" {
				category = "其他"
			}
			unit := action.Unit
			if unit == "" {
				unit = "个"
			}
			quantity := action.Quantity
			if quantity <= 0 {
				quantity = 1
			}
			minQuantity := action.MinQuantity
			if minQuantity <= 0 {
				minQuantity = 1
			}

			if profileID != "" {
				profileItem := &models.ProfileItem{
					ProfileID: profileID,
					Name:      action.Name,
					Category:  category,
					Quantity:  quantity,
					Unit:      unit,
					MinQuantity: minQuantity,
					ExpiryDays: action.ExpiryDays,
					Location: action.Location,
				}
				_ = h.db.CreateProfileItem(profileItem)
			} else {
				item := &models.HouseholdItem{
					Name:        action.Name,
					Category:    category,
					Quantity:    quantity,
					Unit:        unit,
					MinQuantity: minQuantity,
					ExpiryDays:  action.ExpiryDays,
					Location:    action.Location,
				}
				_ = h.db.CreateHouseholdItem(item)
			}
			itemsAdded = append(itemsAdded, action.Name)

		case "restock":
			amount := action.Quantity
			if amount <= 0 {
				amount = 1
			}
			h.applyStockChange(profileID, action.ItemID, action.Name, amount, true)

		case "delete":
			if action.ItemID != "" {
				if profileID != "" {
					_ = h.db.DeleteProfileItem(action.ItemID)
				} else {
					_ = h.db.DeleteHouseholdItem(action.ItemID)
				}
			} else if action.Name != "" {
				// 根据名称查找并删除
				if profileID != "" {
					items, _ := h.db.GetProfileItems(profileID)
					for _, item := range items {
						if item.Name == action.Name {
							_ = h.db.DeleteProfileItem(item.ID)
							break
						}
					}
				} else {
					items, _ := h.db.GetAllHouseholdItems()
					for _, item := range items {
						if item.Name == action.Name {
							_ = h.db.DeleteHouseholdItem(item.ID)
							break
						}
					}
				}
			}
		case "use":
			amount := action.Quantity
			if amount <= 0 {
				amount = 1
			}
			h.applyStockChange(profileID, action.ItemID, action.Name, amount, false)
		case "update":
			h.applyItemUpdate(profileID, action)
		case "todo":
			if profileID == "" || action.Name == "" {
				continue
			}
			category := action.Category
			if category == "" {
				category = "其他"
			}
			todo := &models.HouseholdTodo{
				ProfileID: profileID,
				Name:      action.Name,
				Category:  category,
				Reason:    action.Reason,
				Status:    "open",
			}
			_ = h.db.CreateHouseholdTodo(todo)
		case "suggest_location":
			continue
		}
	}

	// 更新通知
	_ = h.db.GenerateHouseholdNotifications()

	return itemsAdded
}

func parseChatResponse(content string) (ChatResponse, bool) {
	candidate, ok := extractJSONPayload(content)
	if !ok {
		return ChatResponse{}, false
	}
	var resp ChatResponse
	if err := json.Unmarshal([]byte(candidate), &resp); err != nil {
		return ChatResponse{}, false
	}
	if strings.TrimSpace(resp.Reply) == "" {
		resp.Reply = strings.TrimSpace(content)
	}
	return resp, true
}

func extractJSONPayload(content string) (string, bool) {
	clean := strings.TrimSpace(content)
	clean = strings.ReplaceAll(clean, "```json", "")
	clean = strings.ReplaceAll(clean, "```", "")
	clean = strings.TrimSpace(clean)

	startObj := strings.Index(clean, "{")
	endObj := strings.LastIndex(clean, "}")
	if startObj != -1 && endObj != -1 && endObj > startObj {
		return clean[startObj : endObj+1], true
	}

	startArr := strings.Index(clean, "[")
	endArr := strings.LastIndex(clean, "]")
	if startArr != -1 && endArr != -1 && endArr > startArr {
		return clean[startArr : endArr+1], true
	}

	return "", false
}

func (h *HouseholdHandler) applyStockChange(profileID, itemID, name string, amount int, restock bool) {
	if itemID == "" && name != "" {
		if profileID != "" {
			items, _ := h.db.GetProfileItems(profileID)
			for _, item := range items {
				if strings.EqualFold(item.Name, name) {
					itemID = item.ID
					break
				}
			}
		} else {
			items, _ := h.db.GetAllHouseholdItems()
			for _, item := range items {
				if strings.EqualFold(item.Name, name) {
					itemID = item.ID
					break
				}
			}
		}
	}

	if itemID == "" {
		return
	}

	if profileID != "" {
		if restock {
			_ = h.db.RestockProfileItem(itemID, amount)
		} else {
			_ = h.db.UseProfileItem(itemID, amount)
		}
		return
	}

	if restock {
		_ = h.db.RestockHouseholdItem(itemID, amount)
	} else {
		_ = h.db.UseHouseholdItem(itemID, amount)
	}
}

func (h *HouseholdHandler) applyItemUpdate(profileID string, action ChatAction) {
	if action.ItemID == "" && action.Name == "" {
		return
	}

	if profileID != "" {
		item, err := h.db.GetProfileItem(action.ItemID)
		if err != nil && action.ItemID == "" {
			items, _ := h.db.GetProfileItems(profileID)
			for _, it := range items {
				if strings.EqualFold(it.Name, action.Name) {
					item = it
					err = nil
					break
				}
			}
		}
		if err != nil || item == nil {
			return
		}
		if action.Name != "" {
			item.Name = action.Name
		}
		if action.Category != "" {
			item.Category = action.Category
		}
		if action.Quantity > 0 {
			item.Quantity = action.Quantity
		}
		if action.Unit != "" {
			item.Unit = action.Unit
		}
		if action.MinQuantity > 0 {
			item.MinQuantity = action.MinQuantity
		}
		if action.ExpiryDays > 0 {
			item.ExpiryDays = action.ExpiryDays
		}
		if action.Location != "" {
			item.Location = action.Location
		}
		_ = h.db.UpdateProfileItem(item)
		return
	}

	item, err := h.db.GetHouseholdItem(action.ItemID)
	if err != nil && action.ItemID == "" {
		items, _ := h.db.GetAllHouseholdItems()
		for _, it := range items {
			if strings.EqualFold(it.Name, action.Name) {
				item = it
				err = nil
				break
			}
		}
	}
	if err != nil || item == nil {
		return
	}
	if action.Name != "" {
		item.Name = action.Name
	}
	if action.Category != "" {
		item.Category = action.Category
	}
	if action.Quantity > 0 {
		item.Quantity = action.Quantity
	}
	if action.Unit != "" {
		item.Unit = action.Unit
	}
	if action.MinQuantity > 0 {
		item.MinQuantity = action.MinQuantity
	}
	if action.ExpiryDays > 0 {
		item.ExpiryDays = action.ExpiryDays
	}
	if action.Location != "" {
		item.Location = action.Location
	}
	_ = h.db.UpdateHouseholdItem(item)
}

// 调用 DeepSeek 聊天 API
func (h *HouseholdHandler) callDeepSeekChatAPI(prompt string, history []*models.HouseholdConversation) (string, error) {
	apiKey := h.cfg.DeepSeek.APIKey
	model := h.cfg.DeepSeek.Model
	if model == "" {
		model = "deepseek-chat"
	}

	url := "https://api.deepseek.com/chat/completions"

	// 构建消息列表
	messages := []map[string]string{
		{"role": "system", "content": "你是一个智能家庭物品管理助手。请始终返回有效的 JSON 格式。"},
	}

	// 添加历史
	for _, h := range history {
		messages = append(messages, map[string]string{
			"role":    h.Role,
			"content": h.Content,
		})
	}

	// 添加当前提示
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	reqBody := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": 0.7,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("API响应格式错误")
	}

	firstChoice, _ := choices[0].(map[string]interface{})
	message, _ := firstChoice["message"].(map[string]interface{})
	content, _ := message["content"].(string)

	return content, nil
}

// 调用 MiniMax 聊天 API
func (h *HouseholdHandler) callMiniMaxChatAPI(prompt string, history []*models.HouseholdConversation) (string, error) {
	apiKey := h.cfg.MiniMax.APIKey
	model := h.cfg.MiniMax.Model
	if model == "" {
		model = "abab6.5s-chat"
	}

	url := "https://api.minimax.chat/v1/text/chatcompletion_v2"

	// 构建消息列表
	messages := []map[string]string{
		{"role": "system", "content": "你是一个智能家庭物品管理助手。请始终返回有效的 JSON 格式。"},
	}

	// 添加历史
	for _, h := range history {
		messages = append(messages, map[string]string{
			"role":    h.Role,
			"content": h.Content,
		})
	}

	// 添加当前提示
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	reqBody := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": 0.7,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误(%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析JSON失败: %v", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("API响应格式错误")
	}

	firstChoice, _ := choices[0].(map[string]interface{})
	message, _ := firstChoice["message"].(map[string]interface{})
	content, _ := message["content"].(string)

	return content, nil
}
