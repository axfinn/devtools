package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const maxMiniMaxShareAssetBytes = 120 * 1024 * 1024

type CreateMiniMaxResultShareRequest struct {
	Title      string                         `json:"title"`
	Summary    string                         `json:"summary"`
	ResultType string                         `json:"result_type" binding:"required"`
	Model      string                         `json:"model"`
	Payload    json.RawMessage                `json:"payload"`
	Assets     []MiniMaxResultShareAssetInput `json:"assets"`
}

type MiniMaxResultShareAssetInput struct {
	URL         string `json:"url"`
	DataURL     string `json:"data_url"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Kind        string `json:"kind"`
}

type UpdateMiniMaxResultShareRequest struct {
	Title   *string `json:"title"`
	Summary *string `json:"summary"`
	Status  *string `json:"status"`
}

func (h *AIGatewayHandler) GetMiniMaxResultShareDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Result Shares API",
		"summary": "把 MiniMax Studio 的文本、歌词、图像、音频、视频结果保存为可分享资产。媒体资产在创建分享时会落盘，因此分享页可直接看、直接播。",
		"auth": gin.H{
			"create": "Authorization: Bearer dtk_ai_xxx 或 X-Super-Admin-Password",
			"admin":  "X-Super-Admin-Password",
			"public": "GET 公共分享无需鉴权",
		},
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/result-shares/docs", "description": "查看 API 文档"},
			{"method": "POST", "path": "/api/minimax/result-shares", "description": "创建结果分享"},
			{"method": "GET", "path": "/api/minimax/result-shares/:id", "description": "公共查看结果分享"},
			{"method": "GET", "path": "/api/minimax/result-shares/:id/assets/:assetId", "description": "公共访问分享资产，支持直接播放"},
			{"method": "GET", "path": "/api/minimax/result-shares/admin/list", "description": "超管查看分享列表"},
			{"method": "GET", "path": "/api/minimax/result-shares/admin/:id", "description": "超管查看分享详情"},
			{"method": "PUT", "path": "/api/minimax/result-shares/admin/:id", "description": "超管更新标题、摘要、状态"},
			{"method": "DELETE", "path": "/api/minimax/result-shares/admin/:id", "description": "超管删除分享与本地资产"},
		},
		"examples": gin.H{
			"create_text": gin.H{
				"title":       "MiniMax 文本结果",
				"summary":     "保存当前问答结果",
				"result_type": "text",
				"model":       "MiniMax-M2.5",
				"payload": gin.H{
					"text": "这是最终文本结果",
					"raw":  gin.H{"id": "msg_xxx"},
				},
			},
			"create_media": gin.H{
				"title":       "Hailuo 视频结果",
				"summary":     "保存视频任务产物",
				"result_type": "media",
				"model":       "MiniMax-Hailuo-2.3-Fast",
				"payload": gin.H{
					"task_id": "mmt_xxx",
					"prompt":  "一只猫在草地上玩耍",
				},
				"assets": []gin.H{
					{
						"url":          "https://example.com/output.mp4",
						"kind":         "video",
						"content_type": "video/mp4",
						"filename":     "hailuo-output.mp4",
					},
				},
			},
			"curl": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools/api/minimax/result-shares \
  -H "X-Super-Admin-Password: your_password" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的 MiniMax 结果",
    "result_type": "text",
    "model": "MiniMax-M2.5",
    "payload": {"text": "hello world"}
  }'`,
			},
		},
	})
}

func (h *AIGatewayHandler) CreateMiniMaxResultShare(c *gin.Context) {
	key, ok := h.authenticateAdminOrAnyAPIKey(c)
	if !ok {
		return
	}

	var req CreateMiniMaxResultShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	req.ResultType = strings.TrimSpace(strings.ToLower(req.ResultType))
	if req.ResultType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "result_type 不能为空", "code": 400})
		return
	}
	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage(`{}`)
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(req.Payload, &payloadMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload 不是合法 JSON", "code": 400})
		return
	}

	share := &models.MiniMaxResultShare{
		ID:         "mrs_" + utils.GenerateHexKey(10),
		APIKeyID:   firstAPIKeyID(key),
		Title:      strings.TrimSpace(req.Title),
		Summary:    strings.TrimSpace(req.Summary),
		ResultType: req.ResultType,
		Model:      strings.TrimSpace(req.Model),
		CreatorIP:  c.ClientIP(),
	}

	assets, err := h.persistMiniMaxResultShareAssets(share.ID, req.Assets)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	share.AssetsJSON = models.MustMarshalMiniMaxResultShareAssets(assets)
	share.Payload = string(req.Payload)

	if err := h.db.CreateMiniMaxResultShare(share); err != nil {
		_ = os.RemoveAll(h.minimaxResultShareDir(share.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存分享失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, h.minimaxResultShareResponse(c, share, assets, payloadMap))
}

func (h *AIGatewayHandler) GetMiniMaxResultShare(c *gin.Context) {
	share, payloadMap, assets, ok := h.loadMiniMaxResultShareForPublic(c.Param("id"), true)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已停用", "code": 404})
		return
	}
	c.JSON(http.StatusOK, h.minimaxResultShareResponse(c, share, assets, payloadMap))
}

func (h *AIGatewayHandler) GetMiniMaxResultShareAsset(c *gin.Context) {
	share, _, assets, ok := h.loadMiniMaxResultShareForPublic(c.Param("id"), true)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在或已停用", "code": 404})
		return
	}
	assetID := c.Param("assetId")
	var asset *models.MiniMaxResultShareAsset
	for i := range assets {
		if assets[i].ID == assetID {
			asset = &assets[i]
			break
		}
	}
	if asset == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资产不存在", "code": 404})
		return
	}

	path := filepath.Join(h.minimaxResultShareDir(share.ID), asset.Filename)
	file, err := os.Open(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资产文件不存在", "code": 404})
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取资产失败", "code": 500})
		return
	}
	if asset.ContentType != "" {
		c.Header("Content-Type", asset.ContentType)
	}
	http.ServeContent(c.Writer, c.Request, asset.Filename, info.ModTime(), file)
}

func (h *AIGatewayHandler) AdminListMiniMaxResultShares(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	origin := minimaxPublicOrigin(c)
	limit := boundedInt(c.Query("limit"), 50, 1, 200)
	offset := boundedInt(c.Query("offset"), 0, 0, 100000)
	status := strings.TrimSpace(c.Query("status"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	items, err := h.db.ListMiniMaxResultShares(status, keyword, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询分享列表失败", "code": 500})
		return
	}
	total, _ := h.db.CountMiniMaxResultShares(status, keyword)

	resp := make([]gin.H, 0, len(items))
	for _, item := range items {
		assets, _ := models.ParseMiniMaxResultShareAssets(item.AssetsJSON)
		resp = append(resp, gin.H{
			"id":          item.ID,
			"title":       item.Title,
			"summary":     item.Summary,
			"result_type": item.ResultType,
			"model":       item.Model,
			"status":      item.Status,
			"asset_count": len(assets),
			"created_at":  item.CreatedAt,
			"updated_at":  item.UpdatedAt,
			"share_url":   fmt.Sprintf("%s/minimax/share/%s", origin, item.ID),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  resp,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *AIGatewayHandler) AdminGetMiniMaxResultShare(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	share, payloadMap, assets, ok := h.loadMiniMaxResultShareForPublic(c.Param("id"), false)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在", "code": 404})
		return
	}
	c.JSON(http.StatusOK, h.minimaxResultShareResponse(c, share, assets, payloadMap))
}

func (h *AIGatewayHandler) AdminUpdateMiniMaxResultShare(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	var req UpdateMiniMaxResultShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}

	share, err := h.db.GetMiniMaxResultShare(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在", "code": 404})
		return
	}
	if req.Title != nil {
		share.Title = strings.TrimSpace(*req.Title)
	}
	if req.Summary != nil {
		share.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.Status != nil {
		status := strings.TrimSpace(strings.ToLower(*req.Status))
		if status != "active" && status != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status 仅支持 active / disabled", "code": 400})
			return
		}
		share.Status = status
	}
	if err := h.db.UpdateMiniMaxResultShare(share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	assets, _ := models.ParseMiniMaxResultShareAssets(share.AssetsJSON)
	var payloadMap map[string]interface{}
	_ = json.Unmarshal([]byte(share.Payload), &payloadMap)
	c.JSON(http.StatusOK, h.minimaxResultShareResponse(c, share, assets, payloadMap))
}

func (h *AIGatewayHandler) AdminDeleteMiniMaxResultShare(c *gin.Context) {
	if !h.requireSuperAdmin(c, "") {
		return
	}
	id := c.Param("id")
	if _, err := h.db.GetMiniMaxResultShare(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在", "code": 404})
		return
	}
	if err := h.db.DeleteMiniMaxResultShare(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	_ = os.RemoveAll(h.minimaxResultShareDir(id))
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AIGatewayHandler) authenticateAdminOrAnyAPIKey(c *gin.Context) (*models.AIAPIKey, bool) {
	adminPassword := c.GetHeader("X-Super-Admin-Password")
	if adminPassword == "" {
		adminPassword = c.Query("super_admin_password")
	}
	if adminPassword != "" && strings.TrimSpace(h.cfg.AIGateway.SuperAdminPassword) != "" && adminPassword == h.cfg.AIGateway.SuperAdminPassword {
		return nil, true
	}
	return h.authenticateAnyAPIKey(c)
}

func (h *AIGatewayHandler) authenticateAnyAPIKey(c *gin.Context) (*models.AIAPIKey, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-API-Key"))
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 API Key", "code": 401})
		return nil, false
	}
	if len(token) < 18 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key 格式错误", "code": 401})
		return nil, false
	}
	keys, err := h.db.GetAIAPIKeysByPrefix(token[:18])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "校验 API Key 失败", "code": 500})
		return nil, false
	}
	var matched *models.AIAPIKey
	for _, item := range keys {
		if utils.VerifyPassword(token, item.KeyHash) {
			matched = item
			break
		}
	}
	if matched == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key 无效", "code": 401})
		return nil, false
	}
	if matched.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "API Key 已停用", "code": 403})
		return nil, false
	}
	if matched.ExpiresAt != nil && time.Now().After(*matched.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "API Key 已过期", "code": 403})
		return nil, false
	}
	return matched, true
}

func (h *AIGatewayHandler) loadMiniMaxResultShareForPublic(id string, onlyActive bool) (*models.MiniMaxResultShare, map[string]interface{}, []models.MiniMaxResultShareAsset, bool) {
	share, err := h.db.GetMiniMaxResultShare(id)
	if err != nil {
		return nil, nil, nil, false
	}
	if onlyActive && share.Status != "active" {
		return nil, nil, nil, false
	}
	assets, err := models.ParseMiniMaxResultShareAssets(share.AssetsJSON)
	if err != nil {
		assets = []models.MiniMaxResultShareAsset{}
	}
	payloadMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(share.Payload), &payloadMap)
	return share, payloadMap, assets, true
}

func (h *AIGatewayHandler) minimaxResultShareResponse(c *gin.Context, share *models.MiniMaxResultShare, assets []models.MiniMaxResultShareAsset, payload map[string]interface{}) gin.H {
	origin := minimaxPublicOrigin(c)
	respAssets := make([]gin.H, 0, len(assets))
	for _, asset := range assets {
		assetPath := fmt.Sprintf("/api/minimax/result-shares/%s/assets/%s", share.ID, asset.ID)
		respAssets = append(respAssets, gin.H{
			"id":           asset.ID,
			"kind":         asset.Kind,
			"filename":     asset.Filename,
			"content_type": asset.ContentType,
			"size_bytes":   asset.SizeBytes,
			"asset_path":   assetPath,
			"asset_url":    origin + assetPath,
		})
	}
	return gin.H{
		"id":          share.ID,
		"title":       share.Title,
		"summary":     share.Summary,
		"result_type": share.ResultType,
		"model":       share.Model,
		"status":      share.Status,
		"payload":     payload,
		"assets":      respAssets,
		"created_at":  share.CreatedAt,
		"updated_at":  share.UpdatedAt,
		"share_url":   fmt.Sprintf("%s/minimax/share/%s", origin, share.ID),
	}
}

func (h *AIGatewayHandler) persistMiniMaxResultShareAssets(shareID string, inputs []MiniMaxResultShareAssetInput) ([]models.MiniMaxResultShareAsset, error) {
	if len(inputs) == 0 {
		return []models.MiniMaxResultShareAsset{}, nil
	}
	if len(inputs) > 12 {
		return nil, fmt.Errorf("单次分享最多保存 12 个资产")
	}

	dir := h.minimaxResultShareDir(shareID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建分享目录失败: %w", err)
	}

	items := make([]models.MiniMaxResultShareAsset, 0, len(inputs))
	for index, input := range inputs {
		assetID := fmt.Sprintf("asset_%02d", index+1)
		data, contentType, err := h.readMiniMaxShareAssetContent(input)
		if err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		filename := h.miniMaxShareAssetFilename(assetID, input.Filename, input.URL, contentType)
		if err := os.WriteFile(filepath.Join(dir, filename), data, 0o644); err != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("写入分享资产失败: %w", err)
		}
		kind := strings.TrimSpace(strings.ToLower(input.Kind))
		if kind == "" {
			kind = miniMaxShareAssetKind(contentType, filename)
		}
		items = append(items, models.MiniMaxResultShareAsset{
			ID:          assetID,
			Kind:        kind,
			Filename:    filename,
			ContentType: contentType,
			SizeBytes:   int64(len(data)),
			OriginalURL: strings.TrimSpace(input.URL),
		})
	}
	return items, nil
}

func (h *AIGatewayHandler) readMiniMaxShareAssetContent(input MiniMaxResultShareAssetInput) ([]byte, string, error) {
	if strings.TrimSpace(input.DataURL) != "" {
		return parseMiniMaxShareDataURL(input.DataURL)
	}
	if strings.TrimSpace(input.URL) == "" {
		return nil, "", fmt.Errorf("资产缺少 url 或 data_url")
	}
	req, err := http.NewRequest(http.MethodGet, input.URL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("创建资产下载请求失败: %w", err)
	}
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("下载资产失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("下载资产失败: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMiniMaxShareAssetBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("读取资产失败: %w", err)
	}
	if len(data) > maxMiniMaxShareAssetBytes {
		return nil, "", fmt.Errorf("资产超过 %d MB 限制", maxMiniMaxShareAssetBytes/1024/1024)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return data, contentType, nil
}

func (h *AIGatewayHandler) miniMaxShareAssetFilename(assetID, preferredName, rawURL, contentType string) string {
	name := strings.TrimSpace(preferredName)
	if name == "" && rawURL != "" {
		if parsed, err := neturl.Parse(rawURL); err == nil {
			name = filepath.Base(parsed.Path)
		}
	}
	ext := filepath.Ext(name)
	if ext == "" {
		ext = miniMaxShareAssetExt(contentType, rawURL)
	}
	if ext == "" {
		ext = ".bin"
	}
	return assetID + ext
}

func (h *AIGatewayHandler) minimaxResultShareDir(shareID string) string {
	return filepath.Join("data", "minimax_result_shares", shareID)
}

func parseMiniMaxShareDataURL(raw string) ([]byte, string, error) {
	if !strings.HasPrefix(raw, "data:") {
		return nil, "", fmt.Errorf("data_url 格式错误")
	}
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("data_url 缺少数据体")
	}
	header := parts[0]
	contentType := "application/octet-stream"
	if trimmed := strings.TrimPrefix(header, "data:"); trimmed != "" {
		contentType = strings.TrimSuffix(trimmed, ";base64")
	}
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("data_url 解码失败: %w", err)
	}
	if len(data) > maxMiniMaxShareAssetBytes {
		return nil, "", fmt.Errorf("资产超过 %d MB 限制", maxMiniMaxShareAssetBytes/1024/1024)
	}
	return data, contentType, nil
}

func miniMaxShareAssetExt(contentType, rawURL string) string {
	if contentType != "" {
		if extensions, err := mime.ExtensionsByType(contentType); err == nil && len(extensions) > 0 {
			return extensions[0]
		}
	}
	if rawURL != "" {
		if parsed, err := neturl.Parse(rawURL); err == nil {
			if ext := filepath.Ext(parsed.Path); ext != "" {
				return ext
			}
		}
	}
	return ""
}

func miniMaxShareAssetKind(contentType, filename string) string {
	text := strings.ToLower(contentType + " " + filename)
	switch {
	case strings.Contains(text, "video"):
		return "video"
	case strings.Contains(text, "audio"):
		return "audio"
	case strings.Contains(text, "image"):
		return "image"
	default:
		return "file"
	}
}

func minimaxPublicOrigin(c *gin.Context) string {
	if forwardedProto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwardedProto != "" {
		if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
			return forwardedProto + "://" + forwardedHost
		}
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	return scheme + "://" + host
}
