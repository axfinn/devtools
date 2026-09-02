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

const defaultMiniMaxH3BaseURL = "https://api.minimax.cn"

// GetMiniMaxH3VideoDocs 文档入口,展示 H3 视频生成端点与异步任务管理方式。
func (h *AIGatewayHandler) GetMiniMaxH3VideoDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title":   "MiniMax H3 视频生成(Hailuo-03)",
		"summary": "MiniMax-H3 异步视频生成/H3-Context-IR/视频再生成网关入口。submit→task_id→轮询,避免 Cloudflare 524。",
		"auth": gin.H{
			"api_key":      "Authorization: Bearer dtk_ai_xxx",
			"admin_header": "X-Super-Admin-Password",
			"scope":        "media",
		},
		"base_url": "/api/minimax/h3",
		"upstream": defaultMiniMaxH3BaseURL,
		"models":   []string{"MiniMax-H3", "MiniMax-H3-Max"},
		"routes": []gin.H{
			{"method": "POST", "path": "/api/minimax/h3/v2/video_generation", "description": "异步创建视频生成/H3-Context-IR 任务,立即返回 task_id,后台轮询状态"},
			{"method": "GET", "path": "/api/minimax/h3/v2/query/video_generation", "description": "列出最近 7 天内任务(支持 filter.status / filter.task_ids / filter.model / filter.task_type)"},
			{"method": "GET", "path": "/api/minimax/h3/v2/query/video_generation/:task_id", "description": "查询单个任务状态(succeeded 时 content.url 为视频产物)"},
			{"method": "DELETE", "path": "/api/minimax/h3/v2/video_generation/:task_id", "description": "取消 queued 任务或删除 succeeded/failed 记录(running/cancelled 不可操作)"},
		},
		"examples": gin.H{
			"text_to_video": gin.H{
				"model":      "MiniMax-H3",
				"prompt":     "一个男孩在海边打篮球",
				"resolution": "2K",
				"duration":   5,
				"ratio":      "16:9",
			},
			"image_to_video": gin.H{
				"model":             "MiniMax-H3",
				"prompt":            "Pull focus to the people in the background",
				"first_frame_image": "https://cdn.example.com/first_frame.png",
				"resolution":        "2K",
				"duration":          5,
				"ratio":             "adaptive",
			},
		},
	})
}

// MiniMaxH3VideoCreate POST /api/minimax/h3/v2/video_generation
// 上游 MiniMax-H3 视频生成/H3-Context-IR 异步创建:立即返回 task_id,
// 后台 goroutine 轮询上游直到 succeeded/failed/cancelled,把结果落进 minimax_media_tasks。
//
// 上游 endpoint: POST {baseURL}/v2/video_generation
// 返回: {"task_id": "..."}
//
// 客户端轮询: GET /api/minimax/h3/v2/query/video_generation/:task_id
func (h *AIGatewayHandler) MiniMaxH3VideoCreate(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败", "code": 400})
		return
	}
	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空", "code": 400})
		return
	}

	apiKey, baseURL := h.resolveH3VideoConfig()
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax H3 API Key", "code": 502})
		return
	}

	// 解析 model 用于任务记录 & 日志;非必填时仍允许(上游默认 MiniMax-H3)。
	var bodyMap map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &bodyMap)
	model := "MiniMax-H3"
	if m, ok := bodyMap["model"].(string); ok && strings.TrimSpace(m) != "" {
		model = m
	}
	taskType := inferH3TaskType(bodyMap)

	upstreamURL := strings.TrimRight(baseURL, "/") + "/v2/video_generation"
	endpoint := "/api/minimax/h3/v2/video_generation"

	start := time.Now()
	respBody, statusCode, err := h.performH3Request(http.MethodPost, upstreamURL, apiKey, bodyBytes)
	if err != nil {
		h.logAPIRequest(key, model, "minimax-h3-video", endpoint, "media", http.StatusBadGateway, false, err.Error(), truncateString(string(bodyBytes), 10000), "", c.ClientIP(), time.Since(start), h.buildMediaUsage(model))
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}

	success := statusCode < 400
	baseErr := ""
	var payload map[string]interface{}
	if err := json.Unmarshal(respBody, &payload); err == nil {
		baseErr = minimaxBaseRespError(payload)
		if baseErr != "" {
			success = false
		}
	}
	if !success {
		logStatus := statusCode
		if logStatus < 400 {
			logStatus = http.StatusBadGateway
		}
		h.logAPIRequest(key, model, "minimax-h3-video", endpoint, "media", logStatus, false, baseErr, truncateString(string(bodyBytes), 10000), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage(model))
		// 上游 4xx 透传,5xx 包装
		writeMiniMaxResponse(c, statusCode, nil, respBody)
		return
	}

	// 解析上游 task_id,失败则当成整体失败响应(200 但 base_resp 非 0)。
	externalTaskID := extractMinimaxTaskID(payload)
	if externalTaskID == "" {
		h.logAPIRequest(key, model, "minimax-h3-video", endpoint, "media", http.StatusBadGateway, false, "上游响应缺少 task_id", truncateString(string(bodyBytes), 10000), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage(model))
		c.JSON(http.StatusBadGateway, gin.H{"error": "上游响应缺少 task_id", "code": 502, "raw": json.RawMessage(respBody)})
		return
	}

	// 本地任务记录(本地 ID 形如 mmh3_ + 12 hex,与上游 18 位数字 ID 区分;external_task_id 存上游 ID)。
	localTaskID := "mmh3_" + utils.GenerateHexKey(12)
	task := &models.MiniMaxMediaTask{
		ID:             localTaskID,
		APIKeyID:       firstAPIKeyID(key),
		Model:          model,
		Provider:       "minimax-h3-video",
		Status:         "pending",
		RequestBody:    truncateString(string(bodyBytes), 50000),
		ExternalTaskID: externalTaskID,
		ClientIP:       c.ClientIP(),
	}
	if err := h.db.CreateMiniMaxMediaTask(task); err != nil {
		h.logAPIRequest(key, model, "minimax-h3-video", endpoint, "media", http.StatusInternalServerError, false, err.Error(), truncateString(string(bodyBytes), 10000), "", c.ClientIP(), time.Since(start), h.buildMediaUsage(model))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	h.logAPIRequest(key, model, "minimax-h3-video", endpoint, "media", http.StatusAccepted, true, "", truncateString(string(bodyBytes), 10000), truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage(model))

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":          localTaskID,
		"external_task_id": externalTaskID,
		"model":            model,
		"task_type":        taskType,
		"status":           "pending",
		"created_at":       task.CreatedAt.Format(time.RFC3339),
		"poll_url":         "/api/minimax/h3/v2/query/video_generation/" + localTaskID,
		"message":          "H3 任务已提交,请通过 GET /api/minimax/h3/v2/query/video_generation/" + localTaskID + " 轮询结果",
	})

	// 后台轮询;失败/超时/成功都更新本地任务记录。
	go h.runAsyncH3VideoTask(localTaskID, externalTaskID, model, apiKey, baseURL, taskType, firstAPIKeyID(key))
}

// runAsyncH3VideoTask 在 goroutine 中轮询 H3 任务状态,直到 succeeded/failed/cancelled。
// 轮询间隔 4s,超时 30 分钟(2K 视频常见 1-3 分钟,但用户可能排长队)。
// H3-Context-IR (modality=text) 不会超过 1 分钟,留同样窗口足够。
// 连续 5 次网络/解析错误就认栽(避免上游故障时无限压测)。
func (h *AIGatewayHandler) runAsyncH3VideoTask(localTaskID, externalTaskID, model, apiKey, baseURL, taskType, apiKeyID string) {
	deadline := time.Now().Add(30 * time.Minute)
	pollInterval := 4 * time.Second
	endpoint := "/api/minimax/h3/v2/query/video_generation"
	start := time.Now()
	const maxConsecutiveErrors = 5
	consecutiveErrors := 0

	for {
		if time.Now().After(deadline) {
			h.finalizeH3Task(localTaskID, apiKeyID, model, endpoint, "failed", "任务轮询超时(30 分钟未完成)", start, "")
			return
		}
		time.Sleep(pollInterval)

		pollURL := strings.TrimRight(baseURL, "/") + "/v2/query/video_generation/" + externalTaskID
		respBody, _, err := h.performH3Request(http.MethodGet, pollURL, apiKey, nil)
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				h.finalizeH3Task(localTaskID, apiKeyID, model, endpoint, "failed", fmt.Sprintf("连续 %d 次轮询失败: %s", consecutiveErrors, err.Error()), start, "")
				return
			}
			continue
		}

		var payload map[string]interface{}
		_ = json.Unmarshal(respBody, &payload)
		taskObj, _ := payload["task"].(map[string]interface{})
		if taskObj == nil {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				h.finalizeH3Task(localTaskID, apiKeyID, model, endpoint, "failed", "上游响应缺少 task 结构,连续异常", start, "")
				return
			}
			continue
		}
		consecutiveErrors = 0 // 拿到合法 payload,清零

		upstreamStatus := strings.ToLower(extractString(taskObj, "status"))
		mappedStatus := mapH3Status(upstreamStatus)

		// 终态 → 落库
		if mappedStatus == "succeeded" || mappedStatus == "failed" || mappedStatus == "cancelled" {
			resultJSON := string(respBody)
			errMsg := extractH3ErrorMessage(taskObj)
			if mappedStatus == "failed" && errMsg == "" {
				errMsg = "上游任务失败"
			}
			h.finalizeH3Task(localTaskID, apiKeyID, model, endpoint, mappedStatus, errMsg, start, resultJSON)
			return
		}

		// 运行中 → 更新本地状态(轻量,不写 result_json 避免覆盖)
		if task, err := h.db.GetMiniMaxMediaTask(localTaskID); err == nil && task != nil {
			task.Status = mappedStatus
			_ = h.db.UpdateMiniMaxMediaTask(task)
		}
	}
}

// finalizeH3Task 把本地任务收尾:状态/result_json/error_message/completed_at,再写一行 API 日志。
func (h *AIGatewayHandler) finalizeH3Task(localTaskID, apiKeyID, model, endpoint, status, errMsg string, start time.Time, resultJSON string) {
	task, err := h.db.GetMiniMaxMediaTask(localTaskID)
	if err != nil {
		return
	}
	task.Status = status
	if resultJSON != "" {
		task.ResultJSON = resultJSON
	}
	if errMsg != "" {
		task.ErrorMessage = errMsg
	}
	now := time.Now()
	task.CompletedAt = &now
	_ = h.db.UpdateMiniMaxMediaTask(task)

	success := status == "succeeded"
	logStatus := http.StatusOK
	if !success {
		logStatus = http.StatusBadGateway
	}
	h.logAPIRequestByID(apiKeyID, model, "minimax-h3-video", endpoint, "media", logStatus, success, errMsg, task.RequestBody, resultJSON, task.ClientIP, time.Since(start), h.buildMediaUsage(model))
}

// MiniMaxH3VideoQuery GET /api/minimax/h3/v2/query/video_generation/:task_id
// 支持 local_task_id (mmh3_ 前缀) 与 external_task_id (纯数字) 两种入参。
// local_task_id 时优先返回本地记录 + 顺手刷新一次上游;external_task_id 时直透上游。
func (h *AIGatewayHandler) MiniMaxH3VideoQuery(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	rawID := strings.TrimSpace(c.Param("task_id"))
	if rawID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 task_id", "code": 400})
		return
	}

	if strings.HasPrefix(rawID, "mmh3_") {
		h.queryH3ByLocalID(c, key, rawID)
		return
	}
	h.queryH3ByExternalID(c, key, rawID)
}

func (h *AIGatewayHandler) queryH3ByLocalID(c *gin.Context, key *models.AIAPIKey, localID string) {
	task, err := h.db.GetMiniMaxMediaTask(localID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if task.Provider != "minimax-h3-video" {
		c.JSON(http.StatusNotFound, gin.H{"error": "不是 H3 视频任务", "code": 404})
		return
	}
	if key != nil && task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
		return
	}

	result := gin.H{
		"task_id":          task.ID,
		"external_task_id": task.ExternalTaskID,
		"model":            task.Model,
		"provider":         task.Provider,
		"status":           task.Status,
		"error":            task.ErrorMessage,
		"created_at":       task.CreatedAt.Format(time.RFC3339),
		"request_body":     task.RequestBody,
	}
	if task.CompletedAt != nil {
		result["completed_at"] = task.CompletedAt.Format(time.RFC3339)
	}
	if task.ResultJSON != "" {
		var r map[string]interface{}
		if err := json.Unmarshal([]byte(task.ResultJSON), &r); err == nil {
			result["result"] = r
		} else {
			result["result_raw"] = task.ResultJSON
		}
	}
	c.JSON(http.StatusOK, result)
}

func (h *AIGatewayHandler) queryH3ByExternalID(c *gin.Context, key *models.AIAPIKey, externalID string) {
	apiKey, baseURL := h.resolveH3VideoConfig()
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax H3 API Key", "code": 502})
		return
	}
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v2/query/video_generation/" + externalID
	start := time.Now()
	respBody, statusCode, err := h.performH3Request(http.MethodGet, upstreamURL, apiKey, nil)
	if err != nil {
		h.logAPIRequest(key, "MiniMax-H3", "minimax-h3-video", "/api/minimax/h3/v2/query/video_generation/"+externalID, "media", http.StatusBadGateway, false, err.Error(), "", "", c.ClientIP(), time.Since(start), h.buildMediaUsage("MiniMax-H3"))
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	success := statusCode < 400
	h.logAPIRequest(key, "MiniMax-H3", "minimax-h3-video", "/api/minimax/h3/v2/query/video_generation/"+externalID, "media", statusCode, success, "", "", truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage("MiniMax-H3"))
	writeMiniMaxResponse(c, statusCode, nil, respBody)
}

// MiniMaxH3VideoList GET /api/minimax/h3/v2/query/video_generation
// 支持 query: filter.status / filter.task_ids / filter.model / filter.task_type / page_num / page_size
// 透传到上游 GET,7 天窗口内的任务列表 + total。
func (h *AIGatewayHandler) MiniMaxH3VideoList(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	apiKey, baseURL := h.resolveH3VideoConfig()
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax H3 API Key", "code": 502})
		return
	}

	// 上游 list 接口要求 query 参数走 filter.X 命名空间,这里直接透传客户端 query。
	upstreamURL := strings.TrimRight(baseURL, "/") + "/v2/query/video_generation"
	if raw := c.Request.URL.RawQuery; raw != "" {
		upstreamURL += "?" + raw
	}
	start := time.Now()
	respBody, statusCode, err := h.performH3Request(http.MethodGet, upstreamURL, apiKey, nil)
	if err != nil {
		h.logAPIRequest(key, "MiniMax-H3", "minimax-h3-video", "/api/minimax/h3/v2/query/video_generation", "media", http.StatusBadGateway, false, err.Error(), "", "", c.ClientIP(), time.Since(start), h.buildMediaUsage("MiniMax-H3"))
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	success := statusCode < 400
	h.logAPIRequest(key, "MiniMax-H3", "minimax-h3-video", "/api/minimax/h3/v2/query/video_generation", "media", statusCode, success, "", "", truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage("MiniMax-H3"))
	writeMiniMaxResponse(c, statusCode, nil, respBody)
}

// MiniMaxH3VideoDelete DELETE /api/minimax/h3/v2/video_generation/:task_id
// local_task_id 时仅删本地记录 + 通知上游;external_task_id 时直透上游。
func (h *AIGatewayHandler) MiniMaxH3VideoDelete(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, "media")
	if !ok {
		return
	}

	rawID := strings.TrimSpace(c.Param("task_id"))
	if rawID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 task_id", "code": 400})
		return
	}

	apiKey, baseURL := h.resolveH3VideoConfig()
	if apiKey == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "未配置 MiniMax H3 API Key", "code": 502})
		return
	}

	// 解析要传给上游的 external_task_id。
	externalID := rawID
	if strings.HasPrefix(rawID, "mmh3_") {
		task, err := h.db.GetMiniMaxMediaTask(rawID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
			return
		}
		if key != nil && task.APIKeyID != key.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此任务", "code": 403})
			return
		}
		if task.ExternalTaskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "本地任务尚未拿到上游 task_id,无法取消", "code": 400})
			return
		}
		externalID = task.ExternalTaskID
	}

	upstreamURL := strings.TrimRight(baseURL, "/") + "/v2/video_generation/" + url.PathEscape(externalID)
	start := time.Now()
	respBody, statusCode, err := h.performH3Request(http.MethodDelete, upstreamURL, apiKey, nil)
	if err != nil {
		h.logAPIRequest(key, "MiniMax-H3", "minimax-h3-video", "/api/minimax/h3/v2/video_generation/"+externalID, "media", http.StatusBadGateway, false, err.Error(), "", "", c.ClientIP(), time.Since(start), h.buildMediaUsage("MiniMax-H3"))
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	success := statusCode < 400
	h.logAPIRequest(key, "MiniMax-H3", "minimax-h3-video", "/api/minimax/h3/v2/video_generation/"+externalID, "media", statusCode, success, "", "", truncateString(string(respBody), 10000), c.ClientIP(), time.Since(start), h.buildMediaUsage("MiniMax-H3"))

	// 如果是本地任务,且上游操作成功,把本地状态标记为 cancelled/deleted(状态由上游 action 字段决定)。
	if success && strings.HasPrefix(rawID, "mmh3_") {
		if task, err := h.db.GetMiniMaxMediaTask(rawID); err == nil && task != nil {
			var payload map[string]interface{}
			_ = json.Unmarshal(respBody, &payload)
			if action, _ := payload["action"].(string); action != "" {
				task.Status = mapH3Status(action)
			} else if status, _ := payload["status"].(string); status != "" {
				task.Status = mapH3Status(status)
			}
			now := time.Now()
			task.CompletedAt = &now
			_ = h.db.UpdateMiniMaxMediaTask(task)
		}
	}
	writeMiniMaxResponse(c, statusCode, nil, respBody)
}

// resolveH3VideoConfig 返回 (apiKey, baseURL),按优先级:
// 1) minimax_h3_video.api_key / base_url(独立配置);
// 2) minimax.api_key 兜底;
// 3) base_url 默认 https://api.minimax.cn。
func (h *AIGatewayHandler) resolveH3VideoConfig() (string, string) {
	apiKey := strings.TrimSpace(h.cfg.MiniMaxH3Video.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(h.cfg.MiniMax.APIKey)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(h.cfg.MiniMaxH3Video.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultMiniMaxH3BaseURL
	}
	return apiKey, baseURL
}

// performH3Request 通用 H3 上游调用:走 mediaClient(90s,启用压缩,与 token-plan 媒体链路一致)。
// 创建 POST 异步返回 task_id,本身是毫秒级响应;真正耗时的视频生成发生在上游,90s 足够。
// 不走 noProxyClient:那是给 Chat/Anthropic 长推理的,且禁用压缩,容易与 H3 端点的 Accept-Encoding 期望冲突。
func (h *AIGatewayHandler) performH3Request(method, upstreamURL, apiKey string, body []byte) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, upstreamURL, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	client := h.mediaClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	// mediaTransport 启用压缩;部分上游链路(CF / 中转)可能返回 gzip 但漏 Content-Encoding,这里兜底解压。
	respBody = maybeGunzip(respBody)
	return respBody, resp.StatusCode, nil
}

// mapH3Status 把上游状态字符串映射为内部状态(queued→pending, running→running, ...)。
func mapH3Status(status string) string {
	switch strings.ToLower(status) {
	case "queued":
		return "pending"
	case "running":
		return "running"
	case "succeeded":
		return "succeeded"
	case "failed":
		return "failed"
	case "cancelled", "deleted":
		return status
	default:
		return "pending"
	}
}

// extractH3ErrorMessage 从上游 task 对象里抠错误信息(支持 task.error.code+message 与 task.message 两种形态)。
func extractH3ErrorMessage(task map[string]interface{}) string {
	if msg := extractString(task, "error", "message"); msg != "" {
		return msg
	}
	if msg := extractString(task, "message"); msg != "" {
		return msg
	}
	if code := extractString(task, "error", "code"); code != "" {
		return fmt.Sprintf("上游错误码 %s", code)
	}
	return ""
}

// inferH3TaskType 根据 content role 推断 H3 任务类型(仅用于本地 task_type 字段,不影响上游)。
// - content 含 reference_* role → h3_context_ir
// - content 含 base_video → regeneration
// - content 含 first_frame/last_frame → image_to_video
// - 默认 → text_to_video
func inferH3TaskType(bodyMap map[string]interface{}) string {
	rawContent, ok := bodyMap["content"].([]interface{})
	if !ok {
		return "text_to_video"
	}
	hasRefImage, hasRefVideo, hasRefAudio := false, false, false
	hasBaseVideo := false
	hasFirstFrame, hasLastFrame := false, false
	for _, raw := range rawContent {
		item, _ := raw.(map[string]interface{})
		if item == nil {
			continue
		}
		switch extractString(item, "role") {
		case "reference_image":
			hasRefImage = true
		case "reference_video":
			hasRefVideo = true
		case "reference_audio":
			hasRefAudio = true
		case "base_video":
			hasBaseVideo = true
		case "first_frame":
			hasFirstFrame = true
		case "last_frame":
			hasLastFrame = true
		}
	}
	switch {
	case hasBaseVideo:
		return "regeneration"
	case hasRefImage || hasRefVideo || hasRefAudio:
		return "h3_context_ir"
	case hasFirstFrame || hasLastFrame:
		return "image_to_video"
	default:
		return "text_to_video"
	}
}
