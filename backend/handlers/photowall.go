package handlers

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const photoWallFileDir = "./data/photowall"

type PhotoWallHandler struct {
	db                 *models.DB
	adminPassword      string
	defaultExpiresDays int
	maxExpiresDays     int
	maxPhotoSize       int64
	maxPerIP           int
	ipWindow           time.Duration
}

func NewPhotoWallHandler(db *models.DB, cfg *config.Config) *PhotoWallHandler {
	wallCfg := cfg.PhotoWall
	if wallCfg.DefaultExpiresDays <= 0 {
		wallCfg.DefaultExpiresDays = 90
	}
	if wallCfg.MaxExpiresDays <= 0 {
		wallCfg.MaxExpiresDays = 180
	}
	if wallCfg.MaxPhotoSizeMB <= 0 {
		wallCfg.MaxPhotoSizeMB = 10
	}
	return &PhotoWallHandler{
		db:                 db,
		adminPassword:      strings.TrimSpace(wallCfg.AdminPassword),
		defaultExpiresDays: wallCfg.DefaultExpiresDays,
		maxExpiresDays:     wallCfg.MaxExpiresDays,
		maxPhotoSize:       wallCfg.MaxPhotoSizeMB * 1024 * 1024,
		maxPerIP:           5,
		ipWindow:           time.Hour,
	}
}

type CreatePhotoWallProfileRequest struct {
	Title         string `json:"title"`
	Password      string `json:"password" binding:"required"`
	ExpiresIn     int    `json:"expires_in"`
	AdminPassword string `json:"admin_password"`
}

type LoginPhotoWallProfileRequest struct {
	Password string `json:"password" binding:"required"`
}

type CreatePhotoWallProfileResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	CreatorKey  string     `json:"creator_key"`
	AccessKey   string     `json:"access_key"`
	ShortCode   string     `json:"short_code"`
	ShareURL    string     `json:"share_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsPermanent bool       `json:"is_permanent"`
}

type PhotoWallProfileSummaryResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ItemCount   int        `json:"item_count"`
	ShortCode   string     `json:"short_code"`
	ShareURL    string     `json:"share_url,omitempty"`
	IsPermanent bool       `json:"is_permanent"`
}

type PhotoWallItemPayload struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	TakenAt     *time.Time `json:"taken_at"`
	ImageURL    string     `json:"image_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PhotoWallProfilePayload struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	ExpiresAt   *time.Time             `json:"expires_at"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ShortCode   string                 `json:"short_code"`
	ShareURL    string                 `json:"share_url,omitempty"`
	IsPermanent bool                   `json:"is_permanent"`
	ItemCount   int                    `json:"item_count"`
	Categories  []string               `json:"categories"`
	Timeline    []map[string]any       `json:"timeline"`
	Items       []PhotoWallItemPayload `json:"items"`
}

type UpdatePhotoWallProfileRequest struct {
	Action        string `json:"action"`
	Title         string `json:"title"`
	ExpiresIn     int    `json:"expires_in"`
	CreatorKey    string `json:"creator_key"`
	Password      string `json:"password"`
	AdminPassword string `json:"admin_password"`
}

type UpdatePhotoWallItemRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	TakenAt     string `json:"taken_at"`
	CreatorKey  string `json:"creator_key"`
	Password    string `json:"password"`
}

type AdminUpdatePhotoWallProfileRequest struct {
	Action        string `json:"action"`
	ExpiresIn     int    `json:"expires_in"`
	AdminPassword string `json:"admin_password"`
}

func photoWallPasswordIndex(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

func (h *PhotoWallHandler) CreateProfile(c *gin.Context) {
	var req CreatePhotoWallProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		req.Title = "我的照片墙"
	}
	if len(req.Title) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能超过50个字符", "code": 400})
		return
	}
	if len(req.Password) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要4个字符", "code": 400})
		return
	}

	pwIndex := photoWallPasswordIndex(req.Password)
	existing, _ := h.db.GetPhotoWallProfileByPasswordIndex(pwIndex)
	if existing != nil {
		if h.isExpired(existing) {
			_ = h.db.DeletePhotoWallProfile(existing.ID)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "该密码已被使用，请直接登录或更换密码", "code": 409})
			return
		}
	}

	ip := c.ClientIP()
	count, err := h.db.CountPhotoWallProfilesByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	isPermanent := false
	var expiresAt *time.Time
	if req.ExpiresIn == -1 {
		if h.adminPassword == "" || req.AdminPassword != h.adminPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "永久档案需要超管密码", "code": 401})
			return
		}
		isPermanent = true
	} else {
		days := req.ExpiresIn
		if days <= 0 {
			days = h.defaultExpiresDays
		}
		if days > h.maxExpiresDays {
			days = h.maxExpiresDays
		}
		exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		expiresAt = &exp
	}

	creatorKey := utils.GenerateHexKey(16)
	accessKey := utils.GenerateHexKey(16)
	hashedCreatorKey, _ := utils.HashPassword(creatorKey)
	hashedAccessKey, _ := utils.HashPassword(accessKey)
	hashedPassword, _ := utils.HashPassword(req.Password)

	profile := &models.PhotoWallProfile{
		Title:         req.Title,
		Password:      hashedPassword,
		PasswordIndex: pwIndex,
		CreatorKey:    hashedCreatorKey,
		AccessKey:     hashedAccessKey,
		ExpiresAt:     expiresAt,
		CreatorIP:     ip,
		IsPermanent:   isPermanent,
	}
	if err := h.db.CreatePhotoWallProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建档案失败", "code": 500})
		return
	}

	sharePath := fmt.Sprintf("/wall/%s?key=%s", profile.ID, accessKey)
	shortURL := &models.ShortURL{
		OriginalURL: sharePath,
		ExpiresAt:   expiresAt,
		MaxClicks:   100000,
		CreatorIP:   ip,
	}
	if err := h.db.CreateShortURLFromStruct(shortURL); err == nil {
		profile.ShortCode = shortURL.ID
		_ = h.db.UpdatePhotoWallProfile(profile)
	}

	c.JSON(http.StatusCreated, CreatePhotoWallProfileResponse{
		ID:          profile.ID,
		Title:       profile.Title,
		CreatorKey:  creatorKey,
		AccessKey:   accessKey,
		ShortCode:   profile.ShortCode,
		ShareURL:    h.buildShareURL(profile.ID, accessKey),
		ExpiresAt:   profile.ExpiresAt,
		IsPermanent: profile.IsPermanent,
	})
}

func (h *PhotoWallHandler) LoginProfile(c *gin.Context) {
	var req LoginPhotoWallProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码", "code": 400})
		return
	}

	profile, err := h.db.GetPhotoWallProfileByPasswordIndex(photoWallPasswordIndex(req.Password))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的档案", "code": 404})
		return
	}
	if h.isExpired(profile) {
		_ = h.db.DeletePhotoWallProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	c.JSON(http.StatusOK, PhotoWallProfileSummaryResponse{
		ID:          profile.ID,
		Title:       profile.Title,
		ExpiresAt:   profile.ExpiresAt,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
		ItemCount:   profile.ItemCount,
		ShortCode:   profile.ShortCode,
		IsPermanent: profile.IsPermanent,
	})
}

func (h *PhotoWallHandler) GetProfile(c *gin.Context) {
	profile, items, ok := h.loadProfileAndItems(c, false)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, h.buildProfilePayload(profile, items, ""))
}

func (h *PhotoWallHandler) GetShare(c *gin.Context) {
	profile, items, ok := h.loadProfileAndItems(c, true)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, h.buildProfilePayload(profile, items, c.Query("key")))
}

func (h *PhotoWallHandler) UpdateProfile(c *gin.Context) {
	id := c.Param("id")
	var req UpdatePhotoWallProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	if h.isExpired(profile) {
		_ = h.db.DeletePhotoWallProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "档案已过期", "code": 410})
		return
	}
	if !h.canManageProfile(profile, req.CreatorKey, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	switch req.Action {
	case "rename":
		title := strings.TrimSpace(req.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空", "code": 400})
			return
		}
		if len(title) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能超过50个字符", "code": 400})
			return
		}
		profile.Title = title
	case "extend":
		if profile.IsPermanent {
			c.JSON(http.StatusBadRequest, gin.H{"error": "永久档案无需延期", "code": 400})
			return
		}
		days := req.ExpiresIn
		if days <= 0 {
			days = h.defaultExpiresDays
		}
		if days > h.maxExpiresDays {
			days = h.maxExpiresDays
		}
		exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		profile.ExpiresAt = &exp
		if profile.ShortCode != "" {
			_ = h.db.UpdateShortURLExpires(profile.ShortCode, profile.ExpiresAt)
		}
	case "reshare":
		accessKey := utils.GenerateHexKey(16)
		hashedAccessKey, _ := utils.HashPassword(accessKey)
		profile.AccessKey = hashedAccessKey
		if profile.ShortCode != "" {
			_ = h.db.DeleteShortURL(profile.ShortCode)
			profile.ShortCode = ""
		}
		shortURL := &models.ShortURL{
			OriginalURL: fmt.Sprintf("/wall/%s?key=%s", profile.ID, accessKey),
			ExpiresAt:   profile.ExpiresAt,
			MaxClicks:   100000,
			CreatorIP:   profile.CreatorIP,
		}
		if err := h.db.CreateShortURLFromStruct(shortURL); err == nil {
			profile.ShortCode = shortURL.ID
		}
		if err := h.db.UpdatePhotoWallProfile(profile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"short_code": profile.ShortCode,
			"share_url":  h.buildShareURL(profile.ID, accessKey),
		})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的操作", "code": 400})
		return
	}

	if err := h.db.UpdatePhotoWallProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"title":        profile.Title,
		"expires_at":   profile.ExpiresAt,
		"short_code":   profile.ShortCode,
		"is_permanent": profile.IsPermanent,
	})
}

func (h *PhotoWallHandler) DeleteProfile(c *gin.Context) {
	id := c.Param("id")
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	creatorKey := c.Query("creator_key")
	password := c.Query("password")
	if !h.canManageProfile(profile, creatorKey, password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}
	if err := h.db.DeletePhotoWallProfile(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *PhotoWallHandler) UploadItem(c *gin.Context) {
	id := c.Param("id")
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	if h.isExpired(profile) {
		_ = h.db.DeletePhotoWallProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "档案已过期", "code": 410})
		return
	}

	creatorKey := c.PostForm("creator_key")
	password := c.PostForm("password")
	if !h.canManageProfile(profile, creatorKey, password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择照片", "code": 400})
		return
	}
	defer file.Close()

	if header.Size <= 0 || header.Size > h.maxPhotoSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("单张照片不能超过 %dMB", h.maxPhotoSize/1024/1024), "code": 400})
		return
	}

	magicBytes := make([]byte, 16)
	n, _ := file.Read(magicBytes)
	magicBytes = magicBytes[:n]
	_, _ = file.Seek(0, 0)

	detectedType := detectFileType(magicBytes)
	declaredType := header.Header.Get("Content-Type")
	if !(strings.HasPrefix(detectedType, "image/") || strings.HasPrefix(declaredType, "image/")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持图片文件", "code": 400})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = getExtFromMimeType(detectedType)
	}
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".avif":
	default:
		ext = ".jpg"
	}

	filename := h.randomFilename(ext)
	if err := os.MkdirAll(photoWallFileDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败", "code": 500})
		return
	}
	path := filepath.Join(photoWallFileDir, filename)
	out, err := os.Create(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存图片失败", "code": 500})
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存图片失败", "code": 500})
		return
	}
	out.Close()

	takenAt, err := parseOptionalDateTime(c.PostForm("taken_at"))
	if err != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusBadRequest, gin.H{"error": "拍摄时间格式无效", "code": 400})
		return
	}

	item := &models.PhotoWallItem{
		ProfileID:   profile.ID,
		Title:       strings.TrimSpace(c.PostForm("title")),
		Description: strings.TrimSpace(c.PostForm("description")),
		Category:    strings.TrimSpace(c.PostForm("category")),
		TakenAt:     takenAt,
		ImageURL:    "/api/photowall/files/" + filename,
		Filename:    filename,
	}
	if err := h.db.CreatePhotoWallItem(item); err != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存照片记录失败", "code": 500})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"item": h.toItemPayload(item),
	})
}

func (h *PhotoWallHandler) UpdateItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("itemId")

	var req UpdatePhotoWallItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	if !h.canManageProfile(profile, req.CreatorKey, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	item, err := h.db.GetPhotoWallItem(id, itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "照片不存在", "code": 404})
		return
	}
	item.Title = strings.TrimSpace(req.Title)
	item.Description = strings.TrimSpace(req.Description)
	item.Category = strings.TrimSpace(req.Category)
	takenAt, err := parseOptionalDateTime(req.TakenAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "拍摄时间格式无效", "code": 400})
		return
	}
	item.TakenAt = takenAt

	if err := h.db.UpdatePhotoWallItem(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": h.toItemPayload(item)})
}

func (h *PhotoWallHandler) DeleteItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("itemId")
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	creatorKey := c.Query("creator_key")
	password := c.Query("password")
	if !h.canManageProfile(profile, creatorKey, password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}
	filename, err := h.db.DeletePhotoWallItem(id, itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "照片不存在", "code": 404})
		return
	}
	if filename != "" {
		_ = os.Remove(filepath.Join(photoWallFileDir, filename))
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *PhotoWallHandler) DownloadSelection(c *gin.Context) {
	id := c.Param("id")
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	password := c.Query("password")
	creatorKey := c.Query("creator_key")
	if !h.canManageProfile(profile, creatorKey, password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限下载", "code": 401})
		return
	}

	items, err := h.db.ListPhotoWallItems(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取照片失败", "code": 500})
		return
	}

	selectedIDs := map[string]bool{}
	if raw := strings.TrimSpace(c.Query("item_ids")); raw != "" {
		for _, itemID := range strings.Split(raw, ",") {
			itemID = strings.TrimSpace(itemID)
			if itemID != "" {
				selectedIDs[itemID] = true
			}
		}
	}

	var selected []*models.PhotoWallItem
	if len(selectedIDs) == 0 {
		selected = items
	} else {
		for _, item := range items {
			if selectedIDs[item.ID] {
				selected = append(selected, item)
			}
		}
	}
	if len(selected) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可下载的照片", "code": 400})
		return
	}

	if len(selected) == 1 {
		item := selected[0]
		path := filepath.Join(photoWallFileDir, item.Filename)
		if _, err := os.Stat(path); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在", "code": 404})
			return
		}
		c.FileAttachment(path, h.buildDownloadName(profile.Title, item, filepath.Ext(item.Filename)))
		return
	}

	zipName := fmt.Sprintf("%s_%s.zip", h.safeName(profile.Title), time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))

	writer := zip.NewWriter(c.Writer)
	defer writer.Close()

	for _, item := range selected {
		path := filepath.Join(photoWallFileDir, item.Filename)
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		entry, err := writer.Create(h.buildDownloadName(profile.Title, item, filepath.Ext(item.Filename)))
		if err != nil {
			file.Close()
			continue
		}
		_, _ = io.Copy(entry, file)
		file.Close()
	}
}

func (h *PhotoWallHandler) ServeFile(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	if filename == "" || filename == "." || filename == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件名", "code": 400})
		return
	}
	path := filepath.Join(photoWallFileDir, filename)
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在", "code": 404})
		return
	}
	c.File(path)
}

func (h *PhotoWallHandler) AdminList(c *gin.Context) {
	if !h.requireAdmin(c, c.Query("admin_password")) {
		return
	}
	profiles, err := h.db.ListAllPhotoWallProfiles(200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败", "code": 500})
		return
	}
	list := make([]PhotoWallProfileSummaryResponse, 0, len(profiles))
	for _, profile := range profiles {
		list = append(list, PhotoWallProfileSummaryResponse{
			ID:          profile.ID,
			Title:       profile.Title,
			ExpiresAt:   profile.ExpiresAt,
			CreatedAt:   profile.CreatedAt,
			UpdatedAt:   profile.UpdatedAt,
			ItemCount:   profile.ItemCount,
			ShortCode:   profile.ShortCode,
			IsPermanent: profile.IsPermanent,
		})
	}
	c.JSON(http.StatusOK, gin.H{"profiles": list, "total": len(list)})
}

func (h *PhotoWallHandler) AdminGet(c *gin.Context) {
	if !h.requireAdmin(c, c.Query("admin_password")) {
		return
	}
	profile, items, ok := h.loadProfileAndItemsByID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	c.JSON(http.StatusOK, h.buildProfilePayload(profile, items, ""))
}

func (h *PhotoWallHandler) AdminUpdate(c *gin.Context) {
	id := c.Param("id")
	var req AdminUpdatePhotoWallProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}
	if !h.requireAdmin(c, req.AdminPassword) {
		return
	}
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	switch req.Action {
	case "set_permanent":
		profile.IsPermanent = true
		profile.ExpiresAt = nil
		if profile.ShortCode != "" {
			_ = h.db.UpdateShortURLExpires(profile.ShortCode, nil)
		}
	case "extend":
		days := req.ExpiresIn
		if days <= 0 {
			days = h.maxExpiresDays
		}
		exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		profile.IsPermanent = false
		profile.ExpiresAt = &exp
		if profile.ShortCode != "" {
			_ = h.db.UpdateShortURLExpires(profile.ShortCode, profile.ExpiresAt)
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的操作", "code": 400})
		return
	}
	if err := h.db.UpdatePhotoWallProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "expires_at": profile.ExpiresAt, "is_permanent": profile.IsPermanent})
}

func (h *PhotoWallHandler) AdminDelete(c *gin.Context) {
	if !h.requireAdmin(c, c.Query("admin_password")) {
		return
	}
	if err := h.db.DeletePhotoWallProfile(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *PhotoWallHandler) loadProfileAndItems(c *gin.Context, shareMode bool) (*models.PhotoWallProfile, []*models.PhotoWallItem, bool) {
	return h.loadProfileAndItemsByIDAuth(c.Param("id"), c.Query("password"), c.Query("creator_key"), c.Query("key"), shareMode, c)
}

func (h *PhotoWallHandler) loadProfileAndItemsByID(id string) (*models.PhotoWallProfile, []*models.PhotoWallItem, bool) {
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		return nil, nil, false
	}
	items, err := h.db.ListPhotoWallItems(id)
	if err != nil {
		return nil, nil, false
	}
	return profile, items, true
}

func (h *PhotoWallHandler) loadProfileAndItemsByIDAuth(id, password, creatorKey, shareKey string, shareMode bool, c *gin.Context) (*models.PhotoWallProfile, []*models.PhotoWallItem, bool) {
	profile, err := h.db.GetPhotoWallProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return nil, nil, false
	}
	if h.isExpired(profile) {
		_ = h.db.DeletePhotoWallProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "档案已过期", "code": 410})
		return nil, nil, false
	}
	switch {
	case shareMode:
		if shareKey == "" || !utils.VerifyPassword(shareKey, profile.AccessKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "分享密钥无效", "code": 401})
			return nil, nil, false
		}
	default:
		if !h.canManageProfile(profile, creatorKey, password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码或创建者密钥无效", "code": 401})
			return nil, nil, false
		}
	}
	items, err := h.db.ListPhotoWallItems(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取照片失败", "code": 500})
		return nil, nil, false
	}
	return profile, items, true
}

func (h *PhotoWallHandler) buildProfilePayload(profile *models.PhotoWallProfile, items []*models.PhotoWallItem, accessKey string) PhotoWallProfilePayload {
	categorySet := map[string]struct{}{}
	timelineMap := map[string]int{}
	payloadItems := make([]PhotoWallItemPayload, 0, len(items))
	for _, item := range items {
		payloadItems = append(payloadItems, h.toItemPayload(item))
		category := strings.TrimSpace(item.Category)
		if category != "" {
			categorySet[category] = struct{}{}
		}
		marker := item.CreatedAt
		if item.TakenAt != nil {
			marker = *item.TakenAt
		}
		key := marker.Format("2006-01")
		timelineMap[key]++
	}

	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}
	if len(categories) > 1 {
		sortStrings(categories)
	}

	timelineKeys := make([]string, 0, len(timelineMap))
	for key := range timelineMap {
		timelineKeys = append(timelineKeys, key)
	}
	sortStringsDesc(timelineKeys)
	timeline := make([]map[string]any, 0, len(timelineKeys))
	for _, key := range timelineKeys {
		timeline = append(timeline, map[string]any{"month": key, "count": timelineMap[key]})
	}

	shareURL := ""
	if accessKey != "" {
		shareURL = h.buildShareURL(profile.ID, accessKey)
	}

	return PhotoWallProfilePayload{
		ID:          profile.ID,
		Title:       profile.Title,
		ExpiresAt:   profile.ExpiresAt,
		CreatedAt:   profile.CreatedAt,
		UpdatedAt:   profile.UpdatedAt,
		ShortCode:   profile.ShortCode,
		ShareURL:    shareURL,
		IsPermanent: profile.IsPermanent,
		ItemCount:   len(items),
		Categories:  categories,
		Timeline:    timeline,
		Items:       payloadItems,
	}
}

func (h *PhotoWallHandler) toItemPayload(item *models.PhotoWallItem) PhotoWallItemPayload {
	return PhotoWallItemPayload{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Category:    item.Category,
		TakenAt:     item.TakenAt,
		ImageURL:    item.ImageURL,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func (h *PhotoWallHandler) canManageProfile(profile *models.PhotoWallProfile, creatorKey, password string) bool {
	if creatorKey != "" && utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		return true
	}
	if password != "" && utils.VerifyPassword(password, profile.Password) {
		return true
	}
	return false
}

func (h *PhotoWallHandler) requireAdmin(c *gin.Context, password string) bool {
	if h.adminPassword == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "未配置超管密码", "code": 403})
		return false
	}
	if password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "超管密码错误", "code": 401})
		return false
	}
	return true
}

func (h *PhotoWallHandler) isExpired(profile *models.PhotoWallProfile) bool {
	return !profile.IsPermanent && profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt)
}

func (h *PhotoWallHandler) buildShareURL(id, accessKey string) string {
	return fmt.Sprintf("/wall/%s?key=%s", id, accessKey)
}

func (h *PhotoWallHandler) randomFilename(ext string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf) + ext
}

func parseOptionalDateTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid time")
}

func (h *PhotoWallHandler) safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "photowall"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "", "\"", "", "<", "", ">", "", "|", "-", " ", "_")
	return replacer.Replace(name)
}

func (h *PhotoWallHandler) buildDownloadName(profileTitle string, item *models.PhotoWallItem, ext string) string {
	parts := []string{h.safeName(profileTitle)}
	if category := h.safeName(item.Category); category != "" {
		parts = append(parts, category)
	}
	if title := h.safeName(item.Title); title != "" {
		parts = append(parts, title)
	}
	if item.TakenAt != nil {
		parts = append(parts, item.TakenAt.Format("20060102_150405"))
	} else {
		parts = append(parts, item.CreatedAt.Format("20060102_150405"))
	}
	return strings.Join(parts, "_") + ext
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func sortStringsDesc(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] > values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
