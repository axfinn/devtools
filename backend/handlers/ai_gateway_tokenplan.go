package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

func (h *AIGatewayHandler) MediaGenerations(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}

	var req MediaGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.ensureModelAllowed(c, key, req.Model) {
		return
	}

	start := time.Now()
	statusCode := http.StatusOK
	responseBody := ""
	success := false
	errMessage := ""

	result, sc, err := h.bailian.ExecuteTask(CreateBailianTaskRequest{
		Model:           req.Model,
		Prompt:          req.Prompt,
		NegativePrompt:  req.NegativePrompt,
		Image:           req.Image,
		Images:          req.Images,
		Size:            req.Size,
		Count:           req.Count,
		Seed:            req.Seed,
		Watermark:       req.Watermark,
		Duration:        req.Duration,
		Resolution:      req.Resolution,
		FPS:             req.FPS,
		AutoPoll:        req.AutoPoll,
		WaitSeconds:     req.WaitSeconds,
		ClientName:      "api-key:" + key.ID + ":" + req.ClientName,
		ClientRequestID: req.ClientRequestID,
		Parameters:      req.Parameters,
	}, "ai-gateway", c.ClientIP())
	statusCode = sc
	usage := h.buildMediaUsage(req.Model)
	if result != nil {
		responseBody = sanitizeJSON(result)
	}
	if err != nil {
		errMessage = err.Error()
		c.JSON(statusCode, gin.H{"error": errMessage, "code": statusCode, "result": result})
	} else {
		success = true
		c.JSON(http.StatusOK, result)
	}

	h.logAPIRequest(key, req.Model, "bailian", "/api/ai-gateway/v1/media/generations", "media", statusCode, success, errMessage, sanitizeJSON(req), responseBody, c.ClientIP(), time.Since(start), usage)
}

func (h *AIGatewayHandler) ListMediaTasks(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	tasks, err := h.db.ListBailianTasksByClientPrefix(
		boundedInt(c.Query("limit"), 20, 1, 100),
		boundedInt(c.Query("offset"), 0, 0, 100000),
		"api-key:"+key.ID+":",
		c.Query("model"),
		c.Query("status"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *AIGatewayHandler) GetMediaTask(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	task, err := h.db.GetBailianTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if !strings.HasPrefix(task.ClientName, "api-key:"+key.ID+":") {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该任务", "code": 403})
		return
	}
	events, _ := h.db.ListBailianTaskEvents(task.ID)
	c.JSON(http.StatusOK, gin.H{"task": task, "events": events})
}

// AsyncChatCompletions 异步聊天接口：立即返回 task_id，后台调用 LLM，避免 Cloudflare 524
// POST /api/ai-gateway/v1/chat/tasks

func resolveTokenPlanModelEndpoint(model string) string {
	switch model {
	// TTS - 正确端点: /v1/t2a_v2
	case "speech-01-hd", "speech-01-turbo",
		"speech-02-hd", "speech-02-turbo",
		"speech-2.6-hd", "speech-2.6-turbo",
		"speech-2.8-hd", "speech-2.8-turbo":
		return "/v1/t2a_v2"
	// Hailuo 视频 - 正确端点: /v1/video_generation
	// 官方模型名: MiniMax-Hailuo-2.3-Fast, MiniMax-Hailuo-2.3, MiniMax-Hailuo-02, T2V-01-Director, T2V-01
	case "MiniMax-Hailuo-2.3-Fast", "MiniMax-Hailuo-2.3", "MiniMax-Hailuo-02",
		"T2V-01-Director", "T2V-01":
		return "/v1/video_generation"
	// Music - 正确端点: /v1/music_generation
	case "music-2.5", "music-2.6", "music-cover":
		return "/v1/music_generation"
	// Image - 正确端点: /v1/image_generation
	case "image-01", "image-01-live":
		return "/v1/image_generation"
	default:
		return ""
	}
}

// ProxyMinimaxTokenPlan 转发 Token Plan 请求到 MiniMax Token Plan 端点
// POST /api/minimax/token-plan/v1/generations

func (h *AIGatewayHandler) ProxyMinimaxTokenPlan(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	// 直接透传请求体到上游
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}

	// 解析 model 字段用于校验和日志
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体 JSON 格式错误"})
		return
	}

	model, _ := bodyMap["model"].(string)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 字段"})
		return
	}

	if model == "image-01" {
		if _, exists := bodyMap["style"]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image-01 不支持 style 参数，请改用 response_format / subject_reference 等官方字段", "code": 400})
			return
		}
		if _, exists := bodyMap["response_format"]; !exists {
			bodyMap["response_format"] = "url"
		}
		if count, exists := bodyMap["count"]; exists {
			if _, hasN := bodyMap["n"]; !hasN {
				bodyMap["n"] = count
			}
			delete(bodyMap, "count")
		}
		if size, ok := bodyMap["size"].(string); ok && size != "" {
			if _, hasAspect := bodyMap["aspect_ratio"]; !hasAspect {
				aspectRatio, supported := minimaxImageAspectRatioFromSize(size)
				if !supported {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("image-01 不支持尺寸 %s，请使用官方支持的尺寸", size), "code": 400})
					return
				}
				bodyMap["aspect_ratio"] = aspectRatio
			}
			delete(bodyMap, "size")
		}
		bodyBytes, _ = json.Marshal(bodyMap)
	}

	// MiniMax-Hailuo-2.3-Fast 是图生视频模型，兼容前端已有的 image 字段。
	if model == "MiniMax-Hailuo-2.3-Fast" {
		if firstFrame := interfaceToString(bodyMap["first_frame_image"]); strings.TrimSpace(firstFrame) == "" {
			if image := strings.TrimSpace(interfaceToString(bodyMap["image"])); image != "" {
				bodyMap["first_frame_image"] = image
				delete(bodyMap, "image")
				bodyBytes, _ = json.Marshal(bodyMap)
			}
		}
	}

	// 音乐模型默认要求 URL 输出，避免返回不可直接播放/分享的十六进制音频内容。
	if strings.HasPrefix(model, "music-") {
		if _, exists := bodyMap["output_format"]; !exists {
			bodyMap["output_format"] = "url"
			bodyBytes, _ = json.Marshal(bodyMap)
		}
	}

	// 校验模型是否在允许列表中
	if !isModelAllowed(model, TokenPlanAllowedModels) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该端点不支持模型 %s，允许的模型: %v", model, TokenPlanAllowedModels)})
		return
	}

	if key != nil && !h.ensureModelAllowed(c, key, model) {
		return
	}

	apiKey := h.cfg.MiniMaxTokenPlan.APIKey
	if apiKey == "" {
		apiKey = h.cfg.MiniMax.APIKey // fallback to MiniMax APIKey
	}

	baseURL := h.cfg.MiniMaxTokenPlan.BaseURL
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax Token Plan API Key"})
		return
	}

	// 根据模型确定上游路径
	upstreamPath := resolveTokenPlanModelEndpoint(model)
	if upstreamPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("模型 %s 未知的 API 路径", model)})
		return
	}

	start := time.Now()
	upstreamURL := strings.TrimRight(baseURL, "/") + upstreamPath

	// 判断是否为异步模型（视频、音乐、图片生成需要轮询）
	if isTokenPlanAsyncModel(model) {
		h.handleAsyncTokenPlanRequest(c, key, model, apiKey, baseURL, upstreamURL, bodyBytes, bodyMap, start)
		return
	}

	// 同步模型（TTS）：直接透传请求
	h.handleSyncTokenPlanRequest(c, key, model, apiKey, upstreamURL, bodyBytes, bodyMap, start)
}

// handleSyncTokenPlanRequest 处理同步 Token Plan 请求（TTS）

func (h *AIGatewayHandler) handleSyncTokenPlanRequest(c *gin.Context, key *models.AIAPIKey, model, apiKey, upstreamURL string, bodyBytes []byte, bodyMap map[string]interface{}, start time.Time) {
	respBody, respContentType, err := h.doRawRequestWithResp(upstreamURL, apiKey, "POST", bodyBytes, c.Request.Header)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	usage := h.buildMediaUsage(model)
	logRespBody := ""
	if !strings.HasPrefix(respContentType, "audio/") {
		logRespBody = string(respBody)
	}
	h.logAPIRequest(key, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), logRespBody, c.ClientIP(), time.Since(start), usage)

	if strings.HasPrefix(respContentType, "audio/") {
		c.DataFromReader(http.StatusOK, int64(len(respBody)), respContentType, bytes.NewReader(respBody), nil)
	} else {
		c.Data(http.StatusOK, respContentType, respBody)
	}
}

// handleAsyncTokenPlanRequest 处理异步 Token Plan 请求（视频/音乐/图片生成）

func (h *AIGatewayHandler) handleAsyncTokenPlanRequest(c *gin.Context, key *models.AIAPIKey, model, apiKey, baseURL, upstreamURL string, bodyBytes []byte, bodyMap map[string]interface{}, start time.Time) {
	// 创建本地任务记录
	taskID := "mmt_" + utils.GenerateHexKey(12)
	task := &models.MiniMaxMediaTask{
		ID:          taskID,
		APIKeyID:    firstAPIKeyID(key),
		Model:       model,
		Provider:    "minimax",
		Status:      "pending",
		RequestBody: truncateString(string(bodyBytes), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateMiniMaxMediaTask(task); err != nil {
		h.logAPIRequest(key, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusInternalServerError, false, err.Error(), string(bodyBytes), "", c.ClientIP(), time.Since(start), usageSummary{})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	// 立即返回任务ID，让客户端可以轮询
	c.JSON(http.StatusOK, gin.H{
		"task_id":    taskID,
		"model":      model,
		"status":     "pending",
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"message":    "任务已提交，请通过 GET /api/minimax/token-plan/tasks/" + taskID + " 查询状态",
	})

	// 异步提交到 MiniMax API 并轮询结果
	go h.runAsyncMinimaxMediaTask(taskID, apiKey, baseURL, upstreamURL, bodyBytes, model, firstAPIKeyID(key))
}

// runAsyncMinimaxMediaTask 后台异步执行 MiniMax 媒体任务并轮询结果

func (h *AIGatewayHandler) runAsyncMinimaxMediaTask(taskID, apiKey, baseURL, upstreamURL string, bodyBytes []byte, model, apiKeyID string) {
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		return
	}

	// 1. 提交任务到 MiniMax
	task.Status = "running"
	_ = h.db.UpdateMiniMaxMediaTask(task)

	start := time.Now()
	respBody, respContentType, err := h.doRawRequestWithResp(upstreamURL, apiKey, "POST", bodyBytes, nil)

	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		h.logAPIRequestByID(apiKeyID, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusBadGateway, false, err.Error(), string(bodyBytes), "", task.ClientIP, time.Since(start), usageSummary{})
		return
	}

	// 解析响应获取 task_id 和检查是否有同步结果
	var respMap map[string]interface{}
	hasSyncResult := false
	baseErr := ""
	if err := json.Unmarshal(respBody, &respMap); err == nil {
		// 尝试从响应中提取 MiniMax 的任务 ID
		externalTaskID := extractMinimaxTaskID(respMap)
		if externalTaskID != "" {
			task.ExternalTaskID = externalTaskID
			_ = h.db.UpdateMiniMaxMediaTask(task)
		}

		baseErr = minimaxBaseRespError(respMap)
		hasSyncResult = minimaxHasInlineMediaResult(respMap, model, respContentType)
	}

	// 同步返回了媒体内容
	if hasSyncResult {
		task.Status = "succeeded"
		task.ResultJSON = string(respBody)
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		h.logAPIRequestByID(apiKeyID, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), string(respBody), task.ClientIP, time.Since(start), usageSummary{})
		return
	}

	// 异步模式：需要轮询 external_task_id
	if task.ExternalTaskID == "" {
		task.Status = "failed"
		task.ResultJSON = string(respBody)
		if baseErr != "" {
			task.ErrorMessage = baseErr
		} else {
			task.ErrorMessage = "无法获取 MiniMax 任务 ID"
		}
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateMiniMaxMediaTask(task)
		return
	}

	// 轮询 MiniMax 任务状态
	deadline := start.Add(minimaxAsyncTimeout(model))
	for {
		if time.Now().After(deadline) {
			task.Status = "failed"
			task.ErrorMessage = "任务超时未完成"
			now := time.Now()
			task.CompletedAt = &now
			_ = h.db.UpdateMiniMaxMediaTask(task)
			return
		}

		time.Sleep(3 * time.Second)

		pollResp, err := h.pollMinimaxTaskStatus(task.ExternalTaskID, apiKey, baseURL, model)
		if err != nil {
			continue
		}

		task.Status = pollResp.Status
		if pollResp.ResultJSON != "" {
			task.ResultJSON = pollResp.ResultJSON
		}
		if pollResp.ErrorMessage != "" {
			task.ErrorMessage = pollResp.ErrorMessage
		}

		if pollResp.Status == "succeeded" || pollResp.Status == "failed" {
			now := time.Now()
			task.CompletedAt = &now
			_ = h.db.UpdateMiniMaxMediaTask(task)
			h.logAPIRequestByID(apiKeyID, model, "minimax-token-plan", "/api/minimax/token-plan/v1/generations", "media", http.StatusOK, true, "", string(bodyBytes), task.ResultJSON, task.ClientIP, time.Since(start), usageSummary{})
			return
		}
	}
}

// minimaxPollResponse MiniMax 轮询响应
type minimaxPollResponse struct {
	Status       string
	ResultJSON   string
	ErrorMessage string
}

// pollMinimaxTaskStatus 轮询 MiniMax 任务状态

func (h *AIGatewayHandler) pollMinimaxTaskStatus(externalTaskID, apiKey, baseURL, model string) (*minimaxPollResponse, error) {
	pollURL := resolveMinimaxPollURL(baseURL, externalTaskID, model)

	req, err := http.NewRequest(http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		msg := extractString(payload, "message")
		if msg == "" {
			msg = fmt.Sprintf("轮询失败，状态码 %d", resp.StatusCode)
		}
		return &minimaxPollResponse{Status: "failed", ErrorMessage: msg}, nil
	}

	// 解析状态
	status := extractMinimaxTaskStatus(payload)
	resultPayload := payload
	errorMessage := firstNonEmpty(minimaxBaseRespError(payload), extractMinimaxTaskError(payload))

	if mapMinimaxAsyncStatus(status) == "succeeded" {
		if fileID := extractString(payload, "file_id"); fileID != "" {
			if filePayload, err := h.retrieveMinimaxFile(apiKey, baseURL, fileID); err == nil {
				resultPayload = mergeMaps(payload, filePayload)
				if fileObj, ok := filePayload["file"].(map[string]interface{}); ok {
					if downloadURL := extractString(fileObj, "download_url"); downloadURL != "" {
						resultPayload["download_url"] = downloadURL
						if isMiniMaxVideoModel(model) {
							resultPayload["video_url"] = downloadURL
						}
					}
				}
			}
		}
	}
	resultJSON, _ := json.Marshal(resultPayload)

	return &minimaxPollResponse{
		Status:       mapMinimaxAsyncStatus(status),
		ResultJSON:   string(resultJSON),
		ErrorMessage: errorMessage,
	}, nil
}

func minimaxAsyncTimeout(model string) time.Duration {
	switch {
	case isMiniMaxVideoModel(model):
		return 15 * time.Minute
	case strings.HasPrefix(model, "music-"):
		return 8 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func minimaxImageAspectRatioFromSize(size string) (string, bool) {
	switch strings.TrimSpace(size) {
	case "1024x1024":
		return "1:1", true
	case "1280x720":
		return "16:9", true
	case "720x1280":
		return "9:16", true
	case "1152x864":
		return "4:3", true
	case "864x1152":
		return "3:4", true
	case "1248x832":
		return "3:2", true
	case "832x1248":
		return "2:3", true
	case "1344x576":
		return "21:9", true
	case "576x1344":
		return "9:21", true
	default:
		return "", false
	}
}

func resolveMinimaxPollURL(baseURL, externalTaskID, model string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if isMiniMaxVideoModel(model) {
		return baseURL + "/v1/query/video_generation?" + url.Values{"task_id": []string{externalTaskID}}.Encode()
	}
	return baseURL + "/v1/tasks/" + externalTaskID
}

func isMiniMaxVideoModel(model string) bool {
	return strings.HasPrefix(model, "MiniMax-Hailuo-") || strings.HasPrefix(model, "T2V-")
}

func (h *AIGatewayHandler) retrieveMinimaxFile(apiKey, baseURL, fileID string) (map[string]interface{}, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/files/retrieve?" + url.Values{"file_id": []string{fileID}}.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf(firstNonEmpty(minimaxBaseRespError(payload), fmt.Sprintf("文件查询失败，状态码 %d", resp.StatusCode)))
	}
	return payload, nil
}

func mergeMaps(primary, secondary map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(primary)+len(secondary))
	for key, value := range primary {
		merged[key] = value
	}
	for key, value := range secondary {
		if _, exists := merged[key]; exists && key == "file" {
			continue
		}
		merged[key] = value
	}
	return merged
}

// extractMinimaxTaskID 从响应中提取 MiniMax 任务 ID

func extractMinimaxTaskID(payload map[string]interface{}) string {
	for _, keyPath := range [][]string{
		{"task_id"},
		{"taskId"},
		{"id"},
		{"data", "task_id"},
		{"data", "taskId"},
		{"data", "id"},
		{"output", "task_id"},
		{"output", "taskId"},
		{"output", "id"},
		{"data", "task", "task_id"},
		{"data", "task", "taskId"},
		{"task", "task_id"},
		{"task", "taskId"},
	} {
		if id := extractString(payload, keyPath...); id != "" {
			return id
		}
	}
	return ""
}

func minimaxHasInlineMediaResult(payload map[string]interface{}, model, respContentType string) bool {
	if len(extractMediaURLs(payload)) > 0 {
		return true
	}
	if strings.HasPrefix(respContentType, "audio/") || strings.HasPrefix(respContentType, "image/") || strings.HasPrefix(respContentType, "video/") {
		return true
	}
	for _, candidate := range []map[string]interface{}{
		payload,
		mapAt(payload, "data"),
		mapAt(payload, "output"),
		mapAt(mapAt(payload, "data"), "task"),
	} {
		if candidate == nil {
			continue
		}
		for _, key := range []string{
			"audio", "audio_url", "audio_urls",
			"video", "video_url", "video_urls",
			"image", "image_url", "image_urls",
			"file_url", "file_urls", "url", "urls",
		} {
			if hasNonEmptyValue(candidate[key]) {
				return true
			}
		}
	}
	// 音乐 / TTS 同步接口即便没有 URL，也可能直接内嵌音频内容。
	if strings.HasPrefix(model, "music-") || strings.HasPrefix(model, "speech-") {
		if hasNonEmptyValue(extractAny(payload, "data", "audio")) || hasNonEmptyValue(extractAny(payload, "audio")) {
			return true
		}
	}
	return false
}

func mapAt(payload map[string]interface{}, key string) map[string]interface{} {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok {
		return nil
	}
	result, _ := value.(map[string]interface{})
	return result
}

func extractAny(payload map[string]interface{}, path ...string) interface{} {
	var current interface{} = payload
	for _, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = obj[key]
	}
	return current
}

func hasNonEmptyValue(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// extractMinimaxTaskStatus 从响应中提取任务状态

func extractMinimaxTaskStatus(payload map[string]interface{}) string {
	// MiniMax 可能的状态：pending, processing, succeeded, failed
	if status := extractString(payload, "status"); status != "" {
		return status
	}
	if output, ok := payload["output"].(map[string]interface{}); ok {
		if status := extractString(output, "status"); status != "" {
			return status
		}
	}
	return ""
}

// extractMinimaxTaskError 从响应中提取错误信息

func extractMinimaxTaskError(payload map[string]interface{}) string {
	if msg := extractString(payload, "message"); msg != "" {
		return msg
	}
	if msg := extractString(payload, "error", "message"); msg != "" {
		return msg
	}
	if output, ok := payload["output"].(map[string]interface{}); ok {
		if msg := extractString(output, "error_message"); msg != "" {
			return msg
		}
	}
	return ""
}

// mapMinimaxAsyncStatus 将 MiniMax 状态映射为内部状态

func mapMinimaxAsyncStatus(status string) string {
	switch strings.ToLower(status) {
	case "pending", "queued":
		return "pending"
	case "processing", "running":
		return "running"
	case "succeeded", "success":
		return "succeeded"
	case "failed", "fail":
		return "failed"
	default:
		return "pending"
	}
}

// ListMinimaxTokenPlanTasks 获取当前 API Key 的 MiniMax 媒体任务列表
// GET /api/minimax/token-plan/tasks

func (h *AIGatewayHandler) ListMinimaxTokenPlanTasks(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	limit := 20
	offset := 0
	if l := parseInt(c.Query("limit"), 0); l > 0 && l <= 100 {
		limit = l
	}
	if o := parseInt(c.Query("offset"), 0); o >= 0 {
		offset = o
	}

	apiKeyID := firstAPIKeyID(key)
	tasks, err := h.db.ListMiniMaxMediaTasks(apiKeyID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败", "code": 500})
		return
	}

	total, _ := h.db.CountMiniMaxMediaTasks(apiKeyID)

	// 转换输出格式
	taskList := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		item := gin.H{
			"task_id":    t.ID,
			"model":      t.Model,
			"provider":   t.Provider,
			"status":     t.Status,
			"error":      t.ErrorMessage,
			"created_at": t.CreatedAt.Format(time.RFC3339),
		}
		if t.CompletedAt != nil {
			item["completed_at"] = t.CompletedAt.Format(time.RFC3339)
		}
		if t.ExternalTaskID != "" {
			item["external_task_id"] = t.ExternalTaskID
		}
		// 如果有结果，解析并提取 URL
		if t.ResultJSON != "" && t.Status == "succeeded" {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(t.ResultJSON), &result); err == nil {
				if output, ok := result["output"].(map[string]interface{}); ok {
					urls := extractMediaURLs(output)
					if len(urls) > 0 {
						item["result_urls"] = urls
					}
				}
			}
		}
		taskList = append(taskList, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  taskList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetMinimaxTokenPlanTask 获取 MiniMax 媒体任务详情
// GET /api/minimax/token-plan/tasks/:id

func (h *AIGatewayHandler) GetMinimaxTokenPlanTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	taskID := c.Param("id")
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}

	// 验证任务属于当前 API Key
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	result := gin.H{
		"task_id":    task.ID,
		"model":      task.Model,
		"provider":   task.Provider,
		"status":     task.Status,
		"error":      task.ErrorMessage,
		"request":    task.RequestBody,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}

	if task.CompletedAt != nil {
		result["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}
	if task.ExternalTaskID != "" {
		result["external_task_id"] = task.ExternalTaskID
	}
	if task.ResultJSON != "" {
		var resultMap map[string]interface{}
		if err := json.Unmarshal([]byte(task.ResultJSON), &resultMap); err == nil {
			result["result"] = resultMap
			// 提取媒体 URL 列表
			if urls := extractMediaURLs(resultMap); len(urls) > 0 {
				result["result_urls"] = urls
			}
			// 提取内容类型
			if contentType := extractContentType(resultMap); contentType != "" {
				result["content_type"] = contentType
			}
		} else {
			result["result"] = task.ResultJSON
		}
	}

	c.JSON(http.StatusOK, result)
}

// extractMediaURLs 从结果中提取媒体 URL

func extractMediaURLs(output map[string]interface{}) []string {
	urls := make([]string, 0)
	seen := make(map[string]struct{})
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case string:
			if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
				if _, exists := seen[val]; !exists {
					seen[val] = struct{}{}
					urls = append(urls, val)
				}
			}
		case map[string]interface{}:
			for _, v := range val {
				walk(v)
			}
		case []interface{}:
			for _, v := range val {
				walk(v)
			}
		}
	}
	walk(output)
	return urls
}

// extractContentType 从结果中提取内容类型

func extractContentType(output map[string]interface{}) string {
	// 优先从顶层 content_type 字段获取
	if ct := extractString(output, "content_type"); ct != "" {
		return ct
	}
	// 尝试从 output 对象中获取
	if outputObj, ok := output["output"].(map[string]interface{}); ok {
		if ct := extractString(outputObj, "content_type"); ct != "" {
			return ct
		}
		// 如果 output 是 URL 字符串，根据扩展名推断
		if urlStr := extractString(outputObj, "url"); urlStr != "" {
			return inferContentType(urlStr)
		}
		if urlStr := extractString(outputObj, "audio_url"); urlStr != "" {
			return inferContentType(urlStr)
		}
		if urlStr := extractString(outputObj, "video_url"); urlStr != "" {
			return inferContentType(urlStr)
		}
		if urlStr := extractString(outputObj, "image_url"); urlStr != "" {
			return inferContentType(urlStr)
		}
	}
	// 根据模型推断
	if model := extractString(output, "model"); model != "" {
		return inferContentTypeByModel(model)
	}
	return ""
}

// inferContentType 根据 URL 扩展名推断内容类型

func inferContentType(url string) string {
	if strings.HasSuffix(url, ".mp3") || strings.HasSuffix(url, ".mpeg") {
		return "audio/mpeg"
	}
	if strings.HasSuffix(url, ".wav") {
		return "audio/wav"
	}
	if strings.HasSuffix(url, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(url, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(url, ".jpg") || strings.HasSuffix(url, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(url, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(url, ".webp") {
		return "image/webp"
	}
	return "application/octet-stream"
}

// inferContentTypeByModel 根据模型名称推断内容类型

func inferContentTypeByModel(model string) string {
	switch {
	case strings.HasPrefix(model, "speech-"):
		return "audio/mpeg"
	case strings.HasPrefix(model, "MiniMax-Hailuo-") || strings.HasPrefix(model, "T2V-"):
		return "video/mp4"
	case strings.HasPrefix(model, "music-"):
		return "audio/mpeg"
	case model == "image-01" || model == "image-01-live":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

// GetTokenPlanDocs 返回 Token Plan 端点的 API 文档
// GET /api/minimax/token-plan/docs

func (h *AIGatewayHandler) GetTokenPlanDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax Token Plan 接口文档",
		"summary": "通过 AI Gateway 调用 MiniMax Token Plan 媒体生成模型，支持 TTS HD、Hailuo 视频、Music 音乐、Image 生成。",
		"auth": gin.H{
			"api_key": "Authorization: Bearer dtk_ai_xxx",
			"scope":   "media",
		},
		"base_url": "/api/minimax/token-plan",
		"upstream": "https://api.minimaxi.com",
		"models":   TokenPlanAllowedModels,
		"routes": []gin.H{
			{"method": "GET", "path": "/api/minimax/token-plan/docs", "description": "获取本文档"},
			{"method": "POST", "path": "/api/minimax/token-plan/v1/generations", "description": "MiniMax Token Plan 媒体生成接口"},
			{"method": "GET", "path": "/api/minimax/token-plan/tasks", "description": "获取当前 API Key 的任务列表"},
			{"method": "GET", "path": "/api/minimax/token-plan/tasks/:id", "description": "获取任务详情（含 result_urls）"},
			{"method": "GET", "path": "/api/minimax/token-plan/tasks/:id/download", "description": "下载任务产物（代理 MiniMax 媒体文件）"},
		},
		"examples": gin.H{
			"tts_hd_request": gin.H{
				"model": "speech-2.8-hd",
				"text":  "你好，这是语音合成测试",
				"voice_setting": gin.H{
					"voice_id": "male-qn",
					"speed":    1.0,
				},
				"audio_setting": gin.H{
					"audio_format": "mp3",
					"sample_rate":  32000,
				},
			},
			"hailuo_request": gin.H{
				"model":      "MiniMax-Hailuo-2.3-Fast",
				"prompt":     "一只猫在草地上玩耍",
				"duration":   6,
				"resolution": "768P",
			},
			"music_request": gin.H{
				"model":    "music-2.6",
				"prompt":   "轻松的爵士音乐，适合咖啡厅背景",
				"duration": 300,
			},
			"image_request": gin.H{
				"model":  "image-01",
				"prompt": "一个穿着汉服的少女在樱花树下",
				"size":   "1024x1024",
			},
			"curl": gin.H{
				"language": "cURL",
				"code": `curl -X POST https://your-devtools:8080/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是语音合成测试"
  }'`,
			},
		},
		"model_descriptions": gin.H{
			// TTS HD / Turbo 系列
			"speech-01-hd":     "高清语音合成（speech-01 系列）",
			"speech-01-turbo":  "标准语音合成（speech-01 系列 turbo 版）",
			"speech-02-hd":     "高清语音合成（speech-02 系列）",
			"speech-02-turbo":  "标准语音合成（speech-02 系列 turbo 版）",
			"speech-2.6-hd":    "高清语音合成（speech-2.6 系列）",
			"speech-2.6-turbo": "标准语音合成（speech-2.6 系列 turbo 版）",
			"speech-2.8-hd":    "高清语音合成（speech-2.8 系列，推荐）",
			"speech-2.8-turbo": "标准语音合成（speech-2.8 系列 turbo 版）",
			// Hailuo 视频（官方模型名）
			"MiniMax-Hailuo-2.3-Fast": "Hailuo 视频生成（Fast 6s / 768P）",
			"MiniMax-Hailuo-2.3":      "Hailuo 视频生成（MiniMax-Hailuo-2.3）",
			"MiniMax-Hailuo-02":       "Hailuo 视频生成（MiniMax-Hailuo-02）",
			"T2V-01-Director":         "视频生成（T2V-01-Director）",
			"T2V-01":                  "视频生成（T2V-01）",
			// Music
			"music-2.5":   "音乐生成（2.5）",
			"music-2.6":   "音乐生成（2.6）",
			"music-cover": "翻唱生成（需先拿到 cover_feature_id）",
			// Image
			"image-01":      "图像生成",
			"image-01-live": "图像生成（live 版）",
		},
	})
}

// proxyAnthropic 转发 Anthropic 协议请求到指定上游

func (h *AIGatewayHandler) buildMediaUsage(model string) usageSummary {
	cost, currency := h.calculateCost(model, "bailian", 0, 0)
	return usageSummary{Cost: cost, Currency: currency}
}

func extractUsageFromRaw(raw map[string]interface{}) (int, int, int) {
	if raw == nil {
		return 0, 0, 0
	}
	if usage, ok := raw["usage"].(map[string]interface{}); ok {
		input := interfaceToInt(usage["prompt_tokens"])
		if input == 0 {
			input = interfaceToInt(usage["input_tokens"])
		}
		output := interfaceToInt(usage["completion_tokens"])
		if output == 0 {
			output = interfaceToInt(usage["output_tokens"])
		}
		total := interfaceToInt(usage["total_tokens"])
		return input, output, total
	}
	return 0, 0, 0
}

func estimateMessagesTokens(messages []map[string]interface{}) int {
	total := 0
	for _, msg := range messages {
		for _, key := range []string{"content", "role"} {
			if value, ok := msg[key]; ok {
				total += estimateTextTokens(interfaceToString(value))
			}
		}
	}
	return total
}

func estimateTextTokens(text string) int {
	runes := len([]rune(strings.TrimSpace(text)))
	if runes == 0 {
		return 0
	}
	estimated := runes / 4
	if estimated < 1 {
		return 1
	}
	return estimated
}

func (h *AIGatewayHandler) DownloadMinimaxTokenPlanTask(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	taskID := c.Param("id")
	task, err := h.db.GetMiniMaxMediaTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}

	// 验证任务属于当前 API Key
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	// 检查任务状态
	if task.Status != "succeeded" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("任务未完成，当前状态: %s", task.Status), "code": 400})
		return
	}

	if task.ResultJSON == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务结果为空", "code": 400})
		return
	}

	// 解析 ResultJSON 获取媒体 URL
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(task.ResultJSON), &resultMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析任务结果失败", "code": 500})
		return
	}

	// 提取媒体 URL
	urls := extractMediaURLs(resultMap)
	if len(urls) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到媒体文件 URL", "code": 400})
		return
	}

	mediaURL := urls[0] // 取第一个 URL

	// 提取 content type
	contentType := extractContentType(resultMap)
	if contentType == "" {
		contentType = inferContentType(mediaURL)
	}

	// 从 MiniMax URL 下载媒体内容
	req, err := http.NewRequest(http.MethodGet, mediaURL, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "创建请求失败", "code": 502})
		return
	}

	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("下载媒体失败: %s", err.Error()), "code": 502})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("媒体文件返回 HTTP %d", resp.StatusCode), "code": 502})
		return
	}

	// 从响应头获取真实 Content-Type
	if realCT := resp.Header.Get("Content-Type"); realCT != "" {
		contentType = realCT
	}

	// 设置文件名后缀
	filename := ""
	switch {
	case strings.HasPrefix(contentType, "audio/"):
		filename = "audio.mp3"
	case strings.HasPrefix(contentType, "video/"):
		filename = "video.mp4"
	case strings.HasPrefix(contentType, "image/"):
		ext := strings.TrimPrefix(contentType, "image/")
		filename = "image." + ext
	}
	if filename == "" {
		filename = "download"
	}

	// 设置下载 header
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", contentType)
	if resp.ContentLength > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}

	// 透传媒体内容
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// extractImageMeta 从 data URL 中提取 mime 类型和估算大小（KB）
// data URL 格式: data:<mime>;base64,<data>
func extractImageMeta(dataURL string) (mime string, sizeKB int) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "unknown", len(dataURL) / 1024
	}
	// 取 "data:" 和 ";" 之间的部分
	rest := dataURL[5:]
	semi := strings.Index(rest, ";")
	if semi < 0 {
		return "unknown", len(dataURL) / 1024
	}
	mime = rest[:semi]
	// base64 数据长度估算实际字节数
	comma := strings.Index(dataURL, ",")
	if comma >= 0 {
		b64Len := len(dataURL) - comma - 1
		sizeKB = b64Len * 3 / 4 / 1024
	}
	return mime, sizeKB
}

// ==================== Voice Cloning Handlers ====================

// UploadVoiceClone 上传音频复刻音色
// POST /api/minimax/voice-cloning/upload
