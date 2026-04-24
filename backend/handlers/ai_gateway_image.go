package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"devtools/models"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

type apiGatewayImageRequest struct {
	Prompt string                 `json:"prompt"`
	Image  string                 `json:"image" binding:"required"`
	Tool   string                 `json:"tool"`
	Args   map[string]interface{} `json:"args"`
}

func (h *AIGatewayHandler) ImageUnderstanding(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ensureModelAllowed(c, key, model) {
		return
	}

	var req apiGatewayImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整"})
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		req.Prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}
	if err := validateImageSize(req.Image); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	imagePath, err := writeTempImage(req.Image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), h.imageHandler.cfg.Timeout())
	defer cancel()

	start := time.Now()
	toolName, text, result, payload, err := h.imageHandler.ExecuteWithPath(ctx, req.Tool, req.Prompt, req.Args, imagePath)
	latency := time.Since(start)
	if err != nil {
		h.logImageUsageGateway(key, model, toolName, latency, false, err.Error(), req, nil, c.ClientIP())
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	h.logImageUsageGateway(key, model, toolName, latency, true, "", req, result, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"tool":         toolName,
		"model":        model,
		"text":         text,
		"result":       result,
		"args_preview": sanitizeArgs(payload),
	})
}

func (h *AIGatewayHandler) ImageUnderstandingFile(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ensureModelAllowed(c, key, model) {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file"})
		return
	}
	if file.Size > imageUnderstandingMaxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片大小不能超过 10MB"})
		return
	}

	prompt := strings.TrimSpace(c.PostForm("prompt"))
	tool := strings.TrimSpace(c.PostForm("tool"))
	argsText := strings.TrimSpace(c.PostForm("args"))
	args := map[string]interface{}{}
	if argsText != "" {
		if err := json.Unmarshal([]byte(argsText), &args); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "args JSON 解析失败"})
			return
		}
	}
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	imagePath, err := saveMultipartToTempFile(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), h.imageHandler.cfg.Timeout())
	defer cancel()

	start := time.Now()
	toolName, text, result, payload, err := h.imageHandler.ExecuteWithPath(ctx, tool, prompt, args, imagePath)
	latency := time.Since(start)
	if err != nil {
		h.logImageUsageGateway(key, model, toolName, latency, false, err.Error(), nil, nil, c.ClientIP())
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	h.logImageUsageGateway(key, model, toolName, latency, true, "", nil, result, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"tool":         toolName,
		"model":        model,
		"text":         text,
		"result":       result,
		"args_preview": sanitizeArgs(payload),
	})
}

func (h *AIGatewayHandler) ImageUnderstandingSSE(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ensureModelAllowed(c, key, model) {
		return
	}

	var req apiGatewayImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整"})
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		req.Prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}
	if err := validateImageSize(req.Image); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	imagePath, err := writeTempImage(req.Image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	task := newImageTask(req.Args)
	task.Tool = req.Tool
	task.Status = "processing"
	if err := h.imageHandler.saveTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存任务失败"})
		return
	}

	go func(task *state.ImageTask) {
		ctx, cancel := context.WithTimeout(context.Background(), h.imageHandler.cfg.Timeout())
		defer cancel()

		start := time.Now()
		toolName, text, result, payload, err := h.imageHandler.ExecuteWithPath(ctx, req.Tool, req.Prompt, req.Args, imagePath)
		latency := time.Since(start)

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			h.logImageUsageGateway(key, model, toolName, latency, false, err.Error(), req, nil, c.ClientIP())
		} else {
			task.Status = "completed"
			task.Tool = toolName
			task.Text = text
			task.Result, _ = json.Marshal(result)
			h.logImageUsageGateway(key, model, toolName, latency, true, "", req, result, c.ClientIP())
		}
		if saveErr := h.imageHandler.saveTask(task); saveErr != nil {
			return
		}
		_ = payload
	}(task)

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.ID,
		"status":  "processing",
	})
}

func (h *AIGatewayHandler) ImageUnderstandingSSEFile(c *gin.Context) {
	key, ok := h.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ensureModelAllowed(c, key, model) {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 file"})
		return
	}
	if file.Size > imageUnderstandingMaxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片大小不能超过 10MB"})
		return
	}

	prompt := strings.TrimSpace(c.PostForm("prompt"))
	tool := strings.TrimSpace(c.PostForm("tool"))
	argsText := strings.TrimSpace(c.PostForm("args"))
	args := map[string]interface{}{}
	if argsText != "" {
		if err := json.Unmarshal([]byte(argsText), &args); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "args JSON 解析失败"})
			return
		}
	}
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	imagePath, err := saveMultipartToTempFile(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	task := newImageTask(args)
	task.Tool = tool
	task.Status = "processing"
	if err := h.imageHandler.saveTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存任务失败"})
		return
	}

	go func(task *state.ImageTask) {
		ctx, cancel := context.WithTimeout(context.Background(), h.imageHandler.cfg.Timeout())
		defer cancel()

		start := time.Now()
		toolName, text, result, payload, err := h.imageHandler.ExecuteWithPath(ctx, tool, prompt, args, imagePath)
		latency := time.Since(start)

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			h.logImageUsageGateway(key, model, toolName, latency, false, err.Error(), nil, nil, c.ClientIP())
		} else {
			task.Status = "completed"
			task.Tool = toolName
			task.Text = text
			task.Result, _ = json.Marshal(result)
			h.logImageUsageGateway(key, model, toolName, latency, true, "", nil, result, c.ClientIP())
		}
		if saveErr := h.imageHandler.saveTask(task); saveErr != nil {
			return
		}
		_ = payload
	}(task)

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.ID,
		"status":  "processing",
	})
}

func (h *AIGatewayHandler) ImageUnderstandingStream(c *gin.Context) {
	taskID := c.Param("id")

	task, ok := h.imageHandler.getTask(taskID)

	if !ok {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Status(http.StatusOK)
		flusher, _ := c.Writer.(http.Flusher)
		if flusher != nil {
			fmt.Fprintf(c.Writer, "event: error\ndata: {\"error\":\"任务不存在\"}\n\n")
			flusher.Flush()
		}
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Content-Type")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if c.Request.Method == "OPTIONS" {
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	sendEvent := func(event, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	if task.Status == "completed" {
		sendEvent("completed", fmt.Sprintf(`{"task_id":"%s","tool":"%s","text":%s,"result":%s}`,
			task.ID, task.Tool, jsonStr(task.Text), jsonStr(task.Result)))
		return
	}
	if task.Status == "failed" {
		sendEvent("error", fmt.Sprintf(`{"task_id":"%s","error":%s}`, task.ID, jsonStr(task.Error)))
		return
	}

	sendEvent("status", fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, taskID, task.Status))

	ticker := time.NewTicker(500 * time.Millisecond)
	pingTicker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	defer pingTicker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-pingTicker.C:
			sendEvent("ping", `{"time":"`+time.Now().Format(time.RFC3339)+`"}`)
		case <-ticker.C:
			t, found := h.imageHandler.getTask(taskID)
			if !found {
				sendEvent("error", `{"error":"任务不存在"}`)
				return
			}

			if t.Status == "completed" {
				sendEvent("completed", fmt.Sprintf(`{"task_id":"%s","tool":"%s","text":%s,"result":%s}`,
					t.ID, t.Tool, jsonStr(t.Text), jsonStr(t.Result)))
				return
			}

			if t.Status == "failed" {
				sendEvent("error", fmt.Sprintf(`{"task_id":"%s","error":%s}`, t.ID, jsonStr(t.Error)))
				return
			}

			sendEvent("status", fmt.Sprintf(`{"task_id":"%s","status":"%s"}`, taskID, t.Status))
		}
	}
}

func (h *AIGatewayHandler) logImageUsageGateway(key *models.AIAPIKey, model, tool string, latency time.Duration, success bool, errMsg string, req interface{}, resp interface{}, clientIP string) {
	if h == nil {
		return
	}
	requestBody, _ := json.Marshal(req)
	responseBody, _ := json.Marshal(resp)
	h.logAPIRequest(
		key,
		model,
		"minimax_mcp",
		"/api/ai-gateway/v1/image/understanding",
		"image_understanding",
		http.StatusOK,
		success,
		errMsg,
		string(requestBody),
		string(responseBody),
		clientIP,
		latency,
		usageSummary{},
	)
}

