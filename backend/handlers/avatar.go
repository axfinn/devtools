package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

const (
	avatarModelDir        = "./data/avatar/models"
	avatarClipDir         = "./data/avatar/clips"
	avatarMaxModelSize    = 20 * 1024 * 1024 // 20MB
	avatarMaxClipSize     = 4 * 1024 * 1024  // 4MB
	avatarDefaultModelTTL = 30 * 24 * time.Hour
	avatarDefaultClipTTL  = 30 * 24 * time.Hour
)

// AvatarHandler 虚拟形象模块处理器
//
// 阶段:覆盖 §1.2 的"模型库 / 我的资产 / 姿态片段"四类最小子集;
// P0 不实现完整 12 个端点 —— 只交付核心 CRUD(GET 列表 / POST 上传 / DELETE 删除),
// GET /:id/file 与分享链接留到 P1(前端 fetch 当前用 blob URL 或本地路径也能跑)。
type AvatarHandler struct {
	db *models.DB
}

func NewAvatarHandler(db *models.DB) *AvatarHandler {
	return &AvatarHandler{db: db}
}

// avatarRandID 与 models.generateID 等价的对外包装 —— handlers 包无法直接调用
// models 包内的未导出函数,所以在 handlers 这里重新生成一个 10 字节 hex id。
// 这里只是用于落盘文件名,不参与 DB 主键(DB 会自己 generateID)。
func avatarRandID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("a%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)[:n]
}

// UploadModel 上传模型(multipart, field=file) —— P0 收齐一个二进制 blob 入库,
// 前端先用 GLTFLoader 解析再回填 bone_count / poly_count / has_skeleton。
func (h *AvatarHandler) UploadModel(c *gin.Context) {
	if err := os.MkdirAll(avatarModelDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建模型目录"})
		return
	}
	ownerID := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if ownerID == "" {
		ownerID = strings.TrimSpace(c.PostForm("creator_key"))
	}

	title := strings.TrimSpace(c.PostForm("title"))
	format := strings.ToLower(strings.TrimSpace(c.PostForm("format")))
	if format == "" {
		format = "glb"
	}
	if format != "glb" && format != "gltf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format 仅支持 glb / gltf"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供 file 字段"})
		return
	}
	if fileHeader.Size > avatarMaxModelSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": fmt.Sprintf("模型超过 %d MB 上限", avatarMaxModelSize/(1024*1024))})
		return
	}

	dst := filepath.Join(avatarModelDir, avatarRandID(10)+"."+format)
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开上传文件"})
		return
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法写入磁盘"})
		return
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}
	out.Close()

	boneCount, _ := strconv.Atoi(c.PostForm("bone_count"))
	polyCount, _ := strconv.Atoi(c.PostForm("poly_count"))
	hasSkeleton := strings.EqualFold(c.PostForm("has_skeleton"), "1") || strings.EqualFold(c.PostForm("has_skeleton"), "true")

	expires := time.Now().Add(avatarDefaultModelTTL)
	m := &models.AvatarModel{
		OwnerID:     ownerID,
		Title:       title,
		Filename:    dst,
		Original:    fileHeader.Filename,
		Format:      format,
		Size:        fileHeader.Size,
		BoneCount:   boneCount,
		PolyCount:   polyCount,
		HasSkeleton: hasSkeleton,
		ExpiresAt:   &expires,
	}
	if err := h.db.CreateAvatarModel(m); err != nil {
		os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入库失败"})
		return
	}
	c.JSON(http.StatusCreated, m)
}

// ListModels 分页列出 —— 默认包含系统共享(owner_id 空) + 调用者自己的。
func (h *AvatarHandler) ListModels(c *gin.Context) {
	ownerID := strings.TrimSpace(c.Query("owner_id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	list, err := h.db.ListAvatarModels(ownerID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":  list,
		"limit":  limit,
		"offset": offset,
		"count":  len(list),
	})
}

// GetModel 详情
func (h *AvatarHandler) GetModel(c *gin.Context) {
	id := c.Param("id")
	m, err := h.db.GetAvatarModel(id)
	if err != nil {
		if errors.Is(err, models.ErrAvatarModelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "模型不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, m)
}

// DeleteModel 删除记录 + 磁盘文件(仅作者或共享)。
func (h *AvatarHandler) DeleteModel(c *gin.Context) {
	id := c.Param("id")
	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.Query("owner_id"))
	}

	m, err := h.db.GetAvatarModel(id)
	if err != nil {
		if errors.Is(err, models.ErrAvatarModelNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "模型不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	// 系统共享(owner_id 为空)允许删除;用户上传的需匹配 owner
	if m.OwnerID != "" && m.OwnerID != callerOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "非作者无权删除"})
		return
	}
	if err := h.db.DeleteAvatarModel(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if m.Filename != "" {
		_ = os.Remove(m.Filename)
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
}

// RegisterMeAsset 把模型标记进"我的资产"(去重 upsert by owner+model)。
func (h *AvatarHandler) RegisterMeAsset(c *gin.Context) {
	var req struct {
		OwnerID string `json:"owner_id" binding:"required"`
		ModelID string `json:"model_id" binding:"required"`
		Title   string `json:"title"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	a := &models.AvatarMeAsset{
		OwnerID: req.OwnerID,
		ModelID: req.ModelID,
		Title:   req.Title,
		Note:    req.Note,
	}
	if err := h.db.UpsertAvatarMeAsset(a); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入库失败"})
		return
	}
	c.JSON(http.StatusOK, a)
}

// ListMeAssets 列出我的资产。
func (h *AvatarHandler) ListMeAssets(c *gin.Context) {
	ownerID := strings.TrimSpace(c.Query("owner_id"))
	if ownerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 owner_id"})
		return
	}
	list, err := h.db.ListAvatarMeAssets(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "count": len(list)})
}

// DeleteMeAsset
func (h *AvatarHandler) DeleteMeAsset(c *gin.Context) {
	id := c.Param("id")
	ownerID := strings.TrimSpace(c.Query("owner_id"))
	if ownerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 owner_id"})
		return
	}
	if err := h.db.DeleteAvatarMeAsset(id, ownerID); err != nil {
		if errors.Is(err, models.ErrAvatarMeAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
}

// UploadClip 上传姿态片段 —— body 是 JSON(title/model_id/fps/bone_names/frames),
// 后端落 .json 文件入 avatarClipDir 并登记一条 AvatarClip。
func (h *AvatarHandler) UploadClip(c *gin.Context) {
	if err := os.MkdirAll(avatarClipDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建片段目录"})
		return
	}
	ownerID := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if ownerID == "" {
		ownerID = strings.TrimSpace(c.PostForm("creator_key"))
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, avatarMaxClipSize+1024))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}
	if int64(len(body)) > avatarMaxClipSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "姿态片段过大"})
		return
	}

	var payload struct {
		Title     string            `json:"title"`
		ModelID   string            `json:"model_id"`
		FPS       int               `json:"fps"`
		BoneNames []string          `json:"bone_names"`
		Frames    []json.RawMessage `json:"frames"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON 解析失败: " + err.Error()})
		return
	}

	dst := filepath.Join(avatarClipDir, avatarRandID(10)+".json")
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}

	boneNamesJSON, _ := json.Marshal(payload.BoneNames)
	expires := time.Now().Add(avatarDefaultClipTTL)
	clip := &models.AvatarClip{
		ModelID:    payload.ModelID,
		OwnerID:    ownerID,
		Title:      payload.Title,
		FPS:        payload.FPS,
		FrameCount: len(payload.Frames),
		BoneNames:  string(boneNamesJSON),
		FilePath:   dst,
		Format:     models.AvatarFormatPose,
		ExpiresAt:  &expires,
	}
	if err := h.db.CreateAvatarClip(clip); err != nil {
		os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入库失败"})
		return
	}
	c.JSON(http.StatusCreated, clip)
}

// GetClip 详情 —— 直接返回原始 JSON 内容(供前端 fetch 后直接喂 poseData.js)。
func (h *AvatarHandler) GetClip(c *gin.Context) {
	id := c.Param("id")
	clip, err := h.db.GetAvatarClip(id)
	if err != nil {
		if errors.Is(err, models.ErrAvatarClipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "片段不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if clip.FilePath == "" {
		c.JSON(http.StatusOK, clip)
		return
	}
	data, err := os.ReadFile(clip.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件读取失败"})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// DeleteClip
func (h *AvatarHandler) DeleteClip(c *gin.Context) {
	id := c.Param("id")
	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.Query("owner_id"))
	}
	clip, err := h.db.GetAvatarClip(id)
	if err != nil {
		if errors.Is(err, models.ErrAvatarClipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "片段不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if clip.OwnerID != "" && clip.OwnerID != callerOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "非作者无权删除"})
		return
	}
	if err := h.db.DeleteAvatarClip(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if clip.FilePath != "" {
		_ = os.Remove(clip.FilePath)
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
}

// ServeModelFile 直接返回 glb/gltf 二进制 —— 前端 GLTFLoader 直接吃。
func (h *AvatarHandler) ServeModelFile(c *gin.Context) {
	id := c.Param("id")
	m, err := h.db.GetAvatarModel(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模型不存在"})
		return
	}
	if m.Filename == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件缺失"})
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(m.Filename)
}