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

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

func (h *AIGatewayHandler) ChatCompletions(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "chat")
	if !ok {
		return
	}

	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.ensureModelAllowed(c, key, req.Model) {
		return
	}

	start := time.Now()
	statusCode := http.StatusOK
	rawRequest := sanitizeJSON(req)
	var responseBody string
	var success bool
	var errMessage string
	provider := h.resolveChatProvider(req.Model)

	result, rawMap, err := h.executeChatRequest(req)
	usage := h.buildChatUsage(req, rawMap, provider)
	if err != nil {
		statusCode = http.StatusBadGateway
		errMessage = err.Error()
		c.JSON(statusCode, gin.H{"error": errMessage, "code": statusCode})
	} else {
		success = true
		responseBody = sanitizeJSON(rawMap)
		result["usage_summary"] = gin.H{
			"input_tokens":   usage.InputTokens,
			"output_tokens":  usage.OutputTokens,
			"total_tokens":   usage.TotalTokens,
			"estimated_cost": usage.Cost,
			"currency":       usage.Currency,
		}
		c.JSON(http.StatusOK, result)
	}

	h.logAPIRequest(key, req.Model, provider, "/api/ai-gateway/v1/chat/completions", "chat", statusCode, success, errMessage, rawRequest, responseBody, c.ClientIP(), time.Since(start), usage)
}

func (h *AIGatewayHandler) AsyncChatCompletions(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "chat")
	if !ok {
		return
	}
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.ensureModelAllowed(c, key, req.Model) {
		return
	}

	provider := h.resolveChatProvider(req.Model)
	taskID := "ltask_" + utils.GenerateHexKey(12)
	reqJSON, _ := json.Marshal(req)

	task := &models.LLMTask{
		ID:          taskID,
		APIKeyID:    key.ID,
		Model:       req.Model,
		Provider:    provider,
		Status:      "pending",
		RequestBody: truncateString(string(reqJSON), 50000),
		ClientIP:    c.ClientIP(),
	}
	if err := h.db.CreateLLMTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败", "code": 500})
		return
	}

	go h.runAsyncChatTask(taskID, req)

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"model":   req.Model,
	})
}

// GetChatTask 查询异步聊天任务状态
// GET /api/ai-gateway/v1/chat/tasks/:id

func (h *AIGatewayHandler) GetChatTask(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "chat")
	if !ok {
		return
	}
	task, err := h.db.GetLLMTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if task.APIKeyID != key.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该任务", "code": 403})
		return
	}

	resp := gin.H{
		"task_id":    task.ID,
		"status":     task.Status,
		"model":      task.Model,
		"provider":   task.Provider,
		"created_at": task.CreatedAt,
	}
	if task.CompletedAt != nil {
		resp["completed_at"] = task.CompletedAt
	}
	if task.Status == "succeeded" && task.ResultJSON != "" {
		var result interface{}
		if json.Unmarshal([]byte(task.ResultJSON), &result) == nil {
			resp["result"] = result
		}
	}
	if task.Status == "failed" {
		resp["error"] = task.ErrorMessage
	}
	c.JSON(http.StatusOK, resp)
}

// runAsyncChatTask 后台执行 LLM 调用，更新任务状态

func (h *AIGatewayHandler) runAsyncChatTask(taskID string, req ChatCompletionRequest) {
	task, err := h.db.GetLLMTask(taskID)
	if err != nil {
		return
	}
	task.Status = "running"
	_ = h.db.UpdateLLMTask(task)

	result, _, err := h.executeChatRequest(req)
	now := time.Now()
	task.CompletedAt = &now
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
	} else {
		task.Status = "succeeded"
		if data, jsonErr := json.Marshal(result); jsonErr == nil {
			task.ResultJSON = string(data)
		}
	}
	_ = h.db.UpdateLLMTask(task)
}

const internalQwenVisionKeyID = "internal:qwen-vision"
const internalChatKeyID = "internal:chat"

// InternalChat 内部免 API Key 聊天接口，仅限同域浏览器调用
// POST /api/internal/chat

func (h *AIGatewayHandler) InternalChat(c *gin.Context) {
	if !requireSameOrigin(c) {
		return
	}
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数解析失败"})
		return
	}
	if req.Model == "" {
		req.Model = fallbackString(h.cfg.MiniMax.Model, "MiniMax-M2.5")
	}

	start := time.Now()
	result, _, err := h.executeChatRequest(req)
	if err != nil {
		_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
			APIKeyID: internalChatKeyID, Model: req.Model, Provider: "minimax",
			Endpoint: "/api/internal/chat", RequestType: "chat",
			StatusCode: http.StatusBadGateway, Success: false, ErrorMessage: err.Error(),
			ClientIP: c.ClientIP(), LatencyMS: time.Since(start).Milliseconds(),
		})
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_ = h.db.CreateAIAPIRequestLog(&models.AIAPIRequestLog{
		APIKeyID: internalChatKeyID, Model: req.Model, Provider: "minimax",
		Endpoint: "/api/internal/chat", RequestType: "chat",
		StatusCode: http.StatusOK, Success: true,
		ClientIP: c.ClientIP(), LatencyMS: time.Since(start).Milliseconds(),
	})
	c.JSON(http.StatusOK, result)
}

// requireSameOrigin 校验请求来源是否来自当前服务器（同域浏览器请求）
// 非浏览器直接调用（无 Origin/Referer）会被拒绝，防止外部直接访问内部免密钥接口

func requireSameOrigin(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	referer := strings.TrimSpace(c.GetHeader("Referer"))
	source := origin
	if source == "" {
		source = referer
	}
	if source == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "此接口仅限项目内部调用，不允许外部直接请求", "code": 403})
		return false
	}
	parsed, err := url.Parse(source)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "无效的请求来源", "code": 403})
		return false
	}
	sourceHost := parsed.Hostname()
	// 允许 localhost / 127.0.0.1 / ::1（开发模式）
	if sourceHost == "localhost" || sourceHost == "127.0.0.1" || sourceHost == "::1" {
		return true
	}
	// 允许与当前请求 host 相同的来源
	reqHost := c.Request.Host
	if idx := strings.LastIndex(reqHost, ":"); idx >= 0 {
		reqHost = reqHost[:idx]
	}
	if sourceHost == reqHost {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "此接口仅限项目内部调用，不允许外部直接请求", "code": 403})
	return false
}

// InternalQwenVision 内部免 API Key 图像理解接口（qwen3.5-plus 视觉能力）
// POST /api/image-understanding/qwen-vision

func (h *AIGatewayHandler) executeChatRequest(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	provider := h.resolveChatProvider(req.Model)
	switch provider {
	case "dashscope":
		return h.callDashScope(req)
	case "deepseek":
		return h.callDeepSeek(req)
	case "minimax":
		return h.callMiniMax(req)
	case "proxy":
		return h.callProxyChat(req)
	default:
		return nil, nil, fmt.Errorf("不支持的文本模型: %s", req.Model)
	}
}

func (h *AIGatewayHandler) callDashScope(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.DashScope.APIKey) == "" {
		return nil, nil, fmt.Errorf("未配置 dashscope.api_key 或 DASHSCOPE_API_KEY")
	}
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	baseURL := fallbackString(h.cfg.DashScope.BaseURL, "https://coding.dashscope.aliyuncs.com/v1")
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	raw, err := h.doJSONRequest(endpoint, h.cfg.DashScope.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractString(raw, "choices", "0", "message", "content")
	if content == "" {
		content = extractMessageContentFromChoices(raw["choices"])
	}
	result := gin.H{
		"id":           fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":       "chat.completion",
		"created":      time.Now().Unix(),
		"model":        req.Model,
		"provider":     "dashscope",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
	}
	return result, raw, nil
}

func (h *AIGatewayHandler) callDeepSeek(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	bodyMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}
	if req.Stop != nil {
		bodyMap["stop"] = req.Stop
	}
	if len(req.ResponseFormat) > 0 {
		bodyMap["response_format"] = req.ResponseFormat
	}
	if len(req.Tools) > 0 {
		bodyMap["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		bodyMap["tool_choice"] = req.ToolChoice
	}
	if strings.TrimSpace(req.ReasoningEffort) != "" {
		bodyMap["reasoning_effort"] = strings.TrimSpace(req.ReasoningEffort)
	}
	for key, value := range req.ExtraBody {
		bodyMap[key] = value
	}

	raw, err := h.doJSONRequest(deepseekChatCompletionURL(req.Messages), h.cfg.DeepSeek.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractString(raw, "choices", "0", "message", "content")
	if content == "" {
		content = extractMessageContentFromChoices(raw["choices"])
	}
	reasoningContent := extractReasoningContentFromChoices(raw["choices"])
	result := gin.H{
		"id":           fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":       "chat.completion",
		"created":      time.Now().Unix(),
		"model":        req.Model,
		"provider":     "deepseek",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
	}
	if reasoningContent != "" {
		result["reasoning_content"] = reasoningContent
	}
	return result, raw, nil
}

func (h *AIGatewayHandler) callMiniMax(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	bodyMap := map[string]interface{}{
		"model":      req.Model,
		"messages":   req.Messages,
		"max_tokens": valueOrDefaultInt(req.MaxTokens, 1024),
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	raw, err := h.doJSONRequest("https://api.minimaxi.com/anthropic/v1/messages", h.cfg.MiniMax.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractMiniMaxText(raw)
	result := gin.H{
		"id":       fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":   "chat.completion",
		"created":  time.Now().Unix(),
		"model":    req.Model,
		"provider": "minimax",
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"content":      content,
		"raw_response": raw,
	}
	return result, raw, nil
}

// ProxyMinimaxAnthropic 转发 Anthropic 协议格式的请求到 MiniMax Anthropic 兼容端点
// POST /api/minimax/anthropic/v1/messages

func (h *AIGatewayHandler) callProxyChat(req ChatCompletionRequest) (gin.H, map[string]interface{}, error) {
	if strings.TrimSpace(h.cfg.AIGateway.Proxy.APIURL) == "" || strings.TrimSpace(h.cfg.AIGateway.Proxy.APIKey) == "" {
		return nil, nil, fmt.Errorf("未配置 ai_gateway.proxy.api_url 或 ai_gateway.proxy.api_key")
	}

	bodyMap := map[string]interface{}{
		"model":    h.proxyUpstreamModel(req.Model),
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		bodyMap["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}

	endpoint := strings.TrimRight(h.cfg.AIGateway.Proxy.APIURL, "/") + "/chat/completions"
	raw, err := h.doJSONRequest(endpoint, h.cfg.AIGateway.Proxy.APIKey, bodyMap)
	if err != nil {
		return nil, raw, err
	}
	content := extractString(raw, "choices", "0", "message", "content")
	if content == "" {
		content = extractMessageContentFromChoices(raw["choices"])
	}
	result := gin.H{
		"id":           fallbackString(extractString(raw, "id"), "chatcmpl-"+utils.GenerateHexKey(4)),
		"object":       "chat.completion",
		"created":      time.Now().Unix(),
		"model":        req.Model,
		"provider":     "proxy",
		"choices":      raw["choices"],
		"usage":        raw["usage"],
		"content":      content,
		"raw_response": raw,
	}
	return result, raw, nil
}

func (h *AIGatewayHandler) doJSONRequest(url, apiKey string, bodyMap map[string]interface{}) (map[string]interface{}, error) {
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var payload map[string]interface{}
	_ = json.Unmarshal(respBody, &payload)
	if resp.StatusCode >= 400 {
		return payload, fmt.Errorf("上游返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 400))
	}
	return payload, nil
}

// doRawRequest 转发原始请求到上游

func (h *AIGatewayHandler) doRawRequest(url, apiKey, method string, body []byte, headers http.Header) ([]byte, error) {
	respBody, _, err := h.doRawRequestWithResp(url, apiKey, method, body, headers)
	return respBody, err
}

// doRawRequestWithResp 转发原始请求到上游，返回响应体、Content-Type 和错误

func (h *AIGatewayHandler) doRawRequestWithResp(url, apiKey, method string, body []byte, headers http.Header) ([]byte, string, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	// 透传必要的 headers（过滤掉代理相关和可能导致 HTTP/2 问题的 headers）
	skipHeaders := map[string]bool{
		"Content-Type":      true,
		"Authorization":     true,
		"Accept":            true,
		"Connection":        true,
		"Proxy-Connection":  true,
		"Upgrade":           true,
		"Keep-Alive":        true,
		"TE":                true,
		"Trailer":           true,
		"Transfer-Encoding": true,
		"Host":              true,
		"Content-Length":    true,
	}
	for key, values := range headers {
		if skipHeaders[http.CanonicalHeaderKey(key)] {
			continue
		}
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}
	resp, err := h.noProxyClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode >= 400 {
		return respBody, contentType, fmt.Errorf("上游返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 400))
	}
	return respBody, contentType, nil
}

func (h *AIGatewayHandler) resolveChatProvider(model string) string {
	// DashScope 优先（若配置了 API Key）
	if h.cfg.DashScope.APIKey != "" && dashscopeModels[model] {
		return "dashscope"
	}
	switch model {
	case fallbackString(h.cfg.DeepSeek.Model, "deepseek-chat"), "deepseek-chat", "deepseek-reasoner", "deepseek-v4-pro", "deepseek-v4-flash", "deepseek-coder":
		return "deepseek"
	case fallbackString(h.cfg.MiniMax.Model, "abab6.5s-chat"), "abab6.5s-chat", "MiniMax-M2.5":
		return "minimax"
	default:
		if h.hasProxyConfig() && h.isProxyModel(model) {
			return "proxy"
		}
		return ""
	}
}

func (h *AIGatewayHandler) proxyUpstreamModel(requestModel string) string {
	for _, model := range h.cfg.AIGateway.Proxy.Models {
		if strings.TrimSpace(model.Model) == requestModel && strings.TrimSpace(model.UpstreamModel) != "" {
			return model.UpstreamModel
		}
	}
	if strings.TrimSpace(h.cfg.AIGateway.Proxy.UpstreamModel) != "" {
		return h.cfg.AIGateway.Proxy.UpstreamModel
	}
	return requestModel
}

func (h *AIGatewayHandler) hasProxyConfig() bool {
	return strings.TrimSpace(h.cfg.AIGateway.Proxy.APIURL) != "" && strings.TrimSpace(h.cfg.AIGateway.Proxy.APIKey) != ""
}

func (h *AIGatewayHandler) isProxyModel(model string) bool {
	for _, item := range h.getProxyModels() {
		if strings.TrimSpace(item.Model) == model {
			return true
		}
	}
	return false
}

func (h *AIGatewayHandler) getProxyModels() []config.AIGatewayProxyModelConfig {
	if len(h.cfg.AIGateway.Proxy.Models) > 0 {
		models := make([]config.AIGatewayProxyModelConfig, 0, len(h.cfg.AIGateway.Proxy.Models))
		for _, item := range h.cfg.AIGateway.Proxy.Models {
			if strings.TrimSpace(item.Model) == "" {
				continue
			}
			models = append(models, item)
		}
		if len(models) > 0 {
			return models
		}
	}
	return []config.AIGatewayProxyModelConfig{
		{
			Model:         fallbackString(h.cfg.AIGateway.Proxy.Model, "proxy-chat"),
			UpstreamModel: h.cfg.AIGateway.Proxy.UpstreamModel,
			Description:   "OpenAI 兼容代理文本模型",
		},
	}
}

func extractMiniMaxText(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	contentArr, ok := raw["content"].([]interface{})
	if !ok {
		return ""
	}
	for _, item := range contentArr {
		if m, ok := item.(map[string]interface{}); ok {
			if m["type"] == "text" {
				if text, ok := m["text"].(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func extractMessageContentFromChoices(choices interface{}) string {
	items, ok := choices.([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return ""
	}
	message, ok := first["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	content, _ := message["content"].(string)
	return content
}

func extractReasoningContentFromChoices(choices interface{}) string {
	items, ok := choices.([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return ""
	}
	message, ok := first["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	content, _ := message["reasoning_content"].(string)
	return content
}

func jsonStringContains(raw, target string) bool {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return false
	}
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func deepseekChatCompletionURL(messages []map[string]interface{}) string {
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if strings.EqualFold(fmt.Sprint(last["role"]), "assistant") {
			if prefix, ok := last["prefix"].(bool); ok && prefix {
				return "https://api.deepseek.com/beta/chat/completions"
			}
		}
	}
	return "https://api.deepseek.com/chat/completions"
}

func boundedInt(raw string, fallback, min, max int) int {
	value := parseInt(raw, fallback)
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func valueOrDefaultInt(value *int, fallback int) int {
	if value == nil || *value <= 0 {
		return fallback
	}
	return *value
}

func (h *AIGatewayHandler) buildChatUsage(req ChatCompletionRequest, raw map[string]interface{}, provider string) usageSummary {
	inputTokens, outputTokens, totalTokens := extractUsageFromRaw(raw)
	if inputTokens == 0 {
		inputTokens = estimateMessagesTokens(req.Messages)
	}
	if outputTokens == 0 {
		outputTokens = estimateTextTokens(extractMessageContentFromChoices(raw["choices"]) + extractMiniMaxText(raw))
	}
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	cost, currency := h.calculateCost(req.Model, provider, inputTokens, outputTokens)
	return usageSummary{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		Cost:         cost,
		Currency:     currency,
	}
}
