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
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const (
	avatarModelDir        = "./data/avatar/models"
	avatarClipDir         = "./data/avatar/clips"
	avatarMaxModelSize    = 20 * 1024 * 1024 // 20MB
	avatarMaxClipSize     = 4 * 1024 * 1024  // 4MB
	avatarDefaultModelTTL = 30 * 24 * time.Hour
	avatarDefaultClipTTL  = 30 * 24 * time.Hour
	avatarMaxFrameCount   = 60000             // R10:30 分钟 @ 30fps 的安全上限
	avatarMinPasswordLen  = 4                 // 与 excalidraw 一致(见 excalidraw.go:94)
	avatarShareCodeBytes  = 8                 // 64-bit hex(16 chars) —— 实际可用空间远大于短链池
)

// AvatarHandler 虚拟形象模块处理器
//
// P1 阶段:补齐 §1.2 完整 12+ 端点(含 share)、creator_key + password 双因子鉴权(R5)、
// UploadClip 帧数上限(R10);前端 useAssets.js 已就位,等待 P2 接摄像头/Timeline。
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

// readOwnerAndPassword —— 从 header / form / query 取 owner_id + password,
// 双因子都给齐才走"可删/可改"路径;owner_id 必填,password 可选(空模型允许无密码)。
// 复用模式:上传鉴权也走它,避免每个 handler 重复。
func readOwnerAndPassword(c *gin.Context) (ownerID, password string, ok bool) {
	ownerID = strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if ownerID == "" {
		ownerID = strings.TrimSpace(c.PostForm("creator_key"))
		if ownerID == "" {
			ownerID = strings.TrimSpace(c.Query("owner_id"))
		}
	}
	password = c.PostForm("password")
	if password == "" {
		password = c.Query("password")
	}
	return ownerID, password, ownerID != ""
}

// hashPasswordIfPresent —— 空密码返回 ""(无密码,后续鉴权跳过);非空密码走 bcrypt。
func hashPasswordIfPresent(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	if len(password) < avatarMinPasswordLen {
		return "", fmt.Errorf("密码至少 %d 个字符", avatarMinPasswordLen)
	}
	return utils.HashPassword(password)
}

// UploadModel 上传模型(multipart, field=file) —— P1 起接收可选 password(R5)对齐 excalidraw;
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
	// owner_id 可选(允许匿名上传作系统共享),但 password 仅在 owner_id 非空时才有意义。
	password := c.PostForm("password")
	passwordHash, err := hashPasswordIfPresent(password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
		OwnerID:      ownerID,
		Title:        title,
		Filename:     dst,
		Original:     fileHeader.Filename,
		Format:       format,
		Size:         fileHeader.Size,
		BoneCount:    boneCount,
		PolyCount:    polyCount,
		HasSkeleton:  hasSkeleton,
		PasswordHash: passwordHash,
		ExpiresAt:    &expires,
	}
	if err := h.db.CreateAvatarModel(m); err != nil {
		os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入库失败"})
		return
	}
	c.JSON(http.StatusCreated, m)
}

// ListModels 分页列出 —— 三种模式:
//   - 不带 owner_id:返回系统共享 + 所有用户的(全库浏览,前端 useAssets 默认走这路)
//   - 带 owner_id="" 显式:同上(显式声明)
//   - 带 owner_id=alice:返回共享 + alice 自己的(个人视图,前端"我的模型"列表)
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
	// 不传 owner_id 或 owner_id="" 都视为"全库浏览"——前端 useAssets 默认行为,
	// 否则模型库里别人的作品永远搜不到(原 §1.2 设计就是混合列表)。
	list, err := h.db.ListAvatarModels(ownerID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    list,
		"limit":    limit,
		"offset":   offset,
		"count":    len(list),
		"scope":    ownerScope(ownerID),
	})
}

// ownerScope —— 给前端响应里加一个明示字段,便于 UI 区分"全库"和"我的"。
func ownerScope(ownerID string) string {
	if ownerID == "" {
		return "all"
	}
	return "owner:" + ownerID
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

// DeleteModel 删除记录 + 磁盘文件 —— R1 + R5:
//   必须提供 creator_key;只能删自己上传的(系统共享不允许);
//   若模型设了 password_hash,password 也必须对(双因子对齐 excalidraw)。
func (h *AvatarHandler) DeleteModel(c *gin.Context) {
	id := c.Param("id")
	callerOwner, password, ok := readOwnerAndPassword(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 X-Creator-Key 或 owner_id"})
		return
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
	if m.OwnerID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统共享模型不允许删除"})
		return
	}
	if m.OwnerID != callerOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "非作者无权删除"})
		return
	}
	// R5:password 校验。空 hash 表示无密码,跳过;非空则 caller 必须给出正确密码。
	if m.PasswordHash != "" && !utils.VerifyPassword(password, m.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
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
// R6 修正:要求 caller 提供 X-Creator-Key(或 form/query owner_id),且与 body.owner_id 一致。
func (h *AvatarHandler) RegisterMeAsset(c *gin.Context) {
	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.PostForm("owner_id"))
		if callerOwner == "" {
			callerOwner = strings.TrimSpace(c.Query("owner_id"))
		}
	}
	if callerOwner == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 X-Creator-Key"})
		return
	}

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
	// 横向越权防护:caller 必须等于 req.OwnerID。
	if req.OwnerID != callerOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "owner_id 与 X-Creator-Key 不一致"})
		return
	}
	a := &models.AvatarMeAsset{
		OwnerID: callerOwner,
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

// ListMeAssets 列出我的资产 —— R6 修正:要求 X-Creator-Key 鉴权,不接受任意 owner_id query。
func (h *AvatarHandler) ListMeAssets(c *gin.Context) {
	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.Query("owner_id"))
	}
	if callerOwner == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 X-Creator-Key 或 owner_id"})
		return
	}
	list, err := h.db.ListAvatarMeAssets(callerOwner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "count": len(list)})
}

// DeleteMeAsset —— R7 修正:要求 X-Creator-Key 鉴权,不接受任意 owner_id query。
func (h *AvatarHandler) DeleteMeAsset(c *gin.Context) {
	id := c.Param("id")
	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.Query("owner_id"))
	}
	if callerOwner == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 X-Creator-Key 或 owner_id"})
		return
	}
	if err := h.db.DeleteAvatarMeAsset(id, callerOwner); err != nil {
		if errors.Is(err, models.ErrAvatarMeAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "ok": true})
}

// UploadClip 上传姿态片段 —— body 是 JSON(title/model_id/fps/bone_names/frames/password)。
// P1 起:R5 password 可选 + R10 帧数上限 ≤ 60000(30 分钟 @30fps),超过直接 400。
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
		Password  string            `json:"password"` // R5:可选
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON 解析失败: " + err.Error()})
		return
	}

	// R10:帧数硬上限 —— 防止有人塞 100k 帧把磁盘打爆。
	if len(payload.Frames) > avatarMaxFrameCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("帧数超过上限 %d(当前 %d)", avatarMaxFrameCount, len(payload.Frames))})
		return
	}
	passwordHash, err := hashPasswordIfPresent(payload.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		ModelID:      payload.ModelID,
		OwnerID:      ownerID,
		Title:        payload.Title,
		FPS:          payload.FPS,
		FrameCount:   len(payload.Frames),
		BoneNames:    string(boneNamesJSON),
		FilePath:     dst,
		Format:       models.AvatarFormatPose,
		PasswordHash: passwordHash,
		ExpiresAt:    &expires,
	}
	if err := h.db.CreateAvatarClip(clip); err != nil {
		os.Remove(dst)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入库失败"})
		return
	}
	c.JSON(http.StatusCreated, clip)
}

// GetClip 详情 —— 直接返回原始 JSON 内容(供前端 fetch 后直接喂 poseData.js)。
// 如果 clip 设了 password 且 caller 没给正确密码,401。
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
	if clip.PasswordHash != "" {
		password := c.Query("password")
		if !utils.VerifyPassword(password, clip.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 password 或密码错误"})
			return
		}
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

// DeleteClip —— 必须提供 creator_key;只允许作者删除自己的片段;R5 password 双因子。
func (h *AvatarHandler) DeleteClip(c *gin.Context) {
	id := c.Param("id")
	callerOwner, password, ok := readOwnerAndPassword(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 X-Creator-Key 或 owner_id"})
		return
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
	if clip.OwnerID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统共享片段不允许删除"})
		return
	}
	if clip.OwnerID != callerOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "非作者无权删除"})
		return
	}
	if clip.PasswordHash != "" && !utils.VerifyPassword(password, clip.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
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
// 模型过期(ExpiresAt 非空且 < now)返回 410 Gone 并清理磁盘文件,不让 TTL 名存实亡。
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
	if m.ExpiresAt != nil && !m.ExpiresAt.IsZero() && time.Now().After(*m.ExpiresAt) {
		// 过期:清理磁盘 + DB 行,避免后续再次命中。
		_ = os.Remove(m.Filename)
		_ = h.db.DeleteAvatarModel(id)
		c.JSON(http.StatusGone, gin.H{"error": "模型已过期"})
		return
	}
	// 根据格式给出准确的 Content-Type(R11),GLTFLoader 接受但浏览器下载更体面。
	contentType := "application/octet-stream"
	switch strings.ToLower(m.Format) {
	case "glb":
		contentType = "model/gltf-binary"
	case "gltf":
		contentType = "model/gltf+json"
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Content-Type", contentType)
	c.File(m.Filename)
}

// --- 分享链接(P1 起) ---

// CreateShare 把一个 model 或 clip 包成只读 code。
// body: {target_type:"model"|"clip", target_id, password?, expires_in_days?}
// 仅允许把"自己上传的"目标打成 share —— 系统共享目标也可由任何 caller 分享。
func (h *AvatarHandler) CreateShare(c *gin.Context) {
	var req struct {
		TargetType    string `json:"target_type" binding:"required"`
		TargetID      string `json:"target_id" binding:"required"`
		Password      string `json:"password"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.TargetType != "model" && req.TargetType != "clip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_type 仅支持 model / clip"})
		return
	}
	if req.ExpiresInDays <= 0 || req.ExpiresInDays > 365 {
		req.ExpiresInDays = 30 // 默认 30 天
	}

	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.PostForm("creator_key"))
	}

	// 验证 target 存在
	switch req.TargetType {
	case "model":
		if _, err := h.db.GetAvatarModel(req.TargetID); err != nil {
			if errors.Is(err, models.ErrAvatarModelNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "模型不存在"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
	case "clip":
		if _, err := h.db.GetAvatarClip(req.TargetID); err != nil {
			if errors.Is(err, models.ErrAvatarClipNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "片段不存在"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
	}

	passwordHash, err := hashPasswordIfPresent(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	code := avatarRandID(avatarShareCodeBytes)
	expires := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
	s := &models.AvatarShare{
		Code:         code,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		OwnerID:      callerOwner, // 可空 —— 匿名分享
		PasswordHash: passwordHash,
		ExpiresAt:    &expires,
	}
	if err := h.db.CreateAvatarShare(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "入库失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"code":        code,
		"target_type": s.TargetType,
		"target_id":   s.TargetID,
		"expires_at":  s.ExpiresAt,
		"has_password": passwordHash != "",
		"share_url":   "/api/avatar/share/" + code,
	})
}

// GetShare 按 code 取元数据;过期 → 410;密码不对 → 401。
// 命中后 +1 hit_count(异步)。返回 target_type/target_id 让前端决定怎么取内容。
func (h *AvatarHandler) GetShare(c *gin.Context) {
	code := c.Param("code")
	s, err := h.db.GetAvatarShare(code)
	if err != nil {
		if errors.Is(err, models.ErrAvatarShareNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.IsZero() && time.Now().After(*s.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "分享已过期"})
		return
	}
	if s.PasswordHash != "" {
		password := c.Query("password")
		if !utils.VerifyPassword(password, s.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 password 或密码错误"})
			return
		}
	}
	// 异步计数
	go h.db.IncrementAvatarShareHit(code)
	c.JSON(http.StatusOK, gin.H{
		"code":        s.Code,
		"target_type": s.TargetType,
		"target_id":   s.TargetID,
		"hit_count":   s.HitCount + 1,
		"created_at":  s.CreatedAt,
		"expires_at":  s.ExpiresAt,
	})
}

// DeleteShare —— 创建者可删(用 owner_id 校验);匿名 share 不可删。
func (h *AvatarHandler) DeleteShare(c *gin.Context) {
	code := c.Param("code")
	callerOwner := strings.TrimSpace(c.GetHeader("X-Creator-Key"))
	if callerOwner == "" {
		callerOwner = strings.TrimSpace(c.Query("owner_id"))
	}
	if callerOwner == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要 X-Creator-Key 或 owner_id"})
		return
	}
	if err := h.db.DeleteAvatarShare(code, callerOwner); err != nil {
		if errors.Is(err, models.ErrAvatarShareNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或非作者"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": code, "ok": true})
}