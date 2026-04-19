package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"devtools/models"
	"devtools/state"

	"github.com/gin-gonic/gin"
)

const defaultCPAGatewayBaseURL = "http://192.168.31.200:8317/v1"

type APIGatewayHandler struct {
	cpaProxy *httputil.ReverseProxy
	ai       *AIGatewayHandler
	image    *ImageUnderstandingHandler
}

func NewAPIGatewayHandler(ai *AIGatewayHandler, image *ImageUnderstandingHandler) *APIGatewayHandler {
	target, err := url.Parse(defaultCPAGatewayBaseURL)
	if err != nil {
		panic("invalid CPA gateway target URL: " + err.Error())
	}

	const cpaPrefix = "/api/api-gateway/cpa/v1"
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		proxyPath := strings.TrimPrefix(req.URL.Path, cpaPrefix)
		if proxyPath == "/" {
			proxyPath = ""
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = joinURLPath(target.Path, proxyPath)
		req.URL.RawPath = req.URL.Path
		req.Host = target.Host

		// Mirror the standard reverse proxy behavior for User-Agent.
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
	}
	proxy.FlushInterval = -1
	proxy.Transport = &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp == nil {
			return nil
		}
		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(contentType, "text/event-stream") || resp.ContentLength < 0 {
			resp.Header.Del("Content-Length")
			resp.Header.Set("X-Accel-Buffering", "no")
			resp.Header.Set("Cache-Control", "no-cache, no-transform")
			resp.Header.Set("Connection", "keep-alive")
		}
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("cpa reverse proxy error: method=%s path=%s err=%v", req.Method, req.URL.Path, err)
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(`{"error":"下游 CPA 服务不可用"}`))
	}

	return &APIGatewayHandler{cpaProxy: proxy, ai: ai, image: image}
}

func (h *APIGatewayHandler) ProxyCPA(c *gin.Context) {
	h.cpaProxy.ServeHTTP(c.Writer, c.Request)
}

type apiGatewayImageRequest struct {
	Prompt string                 `json:"prompt"`
	Image  string                 `json:"image" binding:"required"`
	Tool   string                 `json:"tool"`
	Args   map[string]interface{} `json:"args"`
}

func (h *APIGatewayHandler) ImageUnderstanding(c *gin.Context) {
	key, ok := h.ai.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ai.ensureModelAllowed(c, key, model) {
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

	// Detach from request context to avoid client/proxy cancellation mid-MCP.
	ctx, cancel := context.WithTimeout(context.Background(), h.image.cfg.Timeout())
	defer cancel()

	start := time.Now()
	toolName, text, result, payload, err := h.image.ExecuteWithPath(ctx, req.Tool, req.Prompt, req.Args, imagePath)
	latency := time.Since(start)
	if err != nil {
		h.logImageUsage(key, model, toolName, latency, false, err.Error(), req, nil, c.ClientIP())
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	h.logImageUsage(key, model, toolName, latency, true, "", req, result, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"tool":         toolName,
		"model":        model,
		"text":         text,
		"result":       result,
		"args_preview": sanitizeArgs(payload),
	})
}

func (h *APIGatewayHandler) ImageUnderstandingFile(c *gin.Context) {
	key, ok := h.ai.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ai.ensureModelAllowed(c, key, model) {
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

	imagePath, err := saveMultipartToTemp(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	// Detach from request context to avoid client/proxy cancellation mid-MCP.
	ctx, cancel := context.WithTimeout(context.Background(), h.image.cfg.Timeout())
	defer cancel()

	start := time.Now()
	toolName, text, result, payload, err := h.image.ExecuteWithPath(ctx, tool, prompt, args, imagePath)
	latency := time.Since(start)
	if err != nil {
		h.logImageUsage(key, model, toolName, latency, false, err.Error(), nil, nil, c.ClientIP())
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	h.logImageUsage(key, model, toolName, latency, true, "", nil, result, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"tool":         toolName,
		"model":        model,
		"text":         text,
		"result":       result,
		"args_preview": sanitizeArgs(payload),
	})
}

// ImageUnderstandingSSE 创建 SSE 任务
func (h *APIGatewayHandler) ImageUnderstandingSSE(c *gin.Context) {
	key, ok := h.ai.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ai.ensureModelAllowed(c, key, model) {
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

	// 创建任务
	task := newImageTask(req.Args)
	task.Tool = req.Tool
	task.Status = "processing"
	if err := h.image.saveTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存任务失败"})
		return
	}

	// 后台执行
	go func(task *state.ImageTask) {
		ctx, cancel := context.WithTimeout(context.Background(), h.image.cfg.Timeout())
		defer cancel()

		start := time.Now()
		toolName, text, result, payload, err := h.image.ExecuteWithPath(ctx, req.Tool, req.Prompt, req.Args, imagePath)
		latency := time.Since(start)

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			h.logImageUsage(key, model, toolName, latency, false, err.Error(), req, nil, c.ClientIP())
		} else {
			task.Status = "completed"
			task.Tool = toolName
			task.Text = text
			task.Result, _ = json.Marshal(result)
			h.logImageUsage(key, model, toolName, latency, true, "", req, result, c.ClientIP())
		}
		if saveErr := h.image.saveTask(task); saveErr != nil {
			return
		}
		_ = payload
	}(task)

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.ID,
		"status":  "processing",
	})
}

// ImageUnderstandingSSEFile 从文件创建 SSE 任务
func (h *APIGatewayHandler) ImageUnderstandingSSEFile(c *gin.Context) {
	key, ok := h.ai.authenticateAPIKey(c, "media")
	if !ok {
		return
	}
	model := "minimax-mcp-understand-image"
	if !h.ai.ensureModelAllowed(c, key, model) {
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

	imagePath, err := saveMultipartToTemp(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	// 创建任务
	task := newImageTask(args)
	task.Tool = tool
	task.Status = "processing"
	if err := h.image.saveTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存任务失败"})
		return
	}

	// 后台执行
	go func(task *state.ImageTask) {
		ctx, cancel := context.WithTimeout(context.Background(), h.image.cfg.Timeout())
		defer cancel()

		start := time.Now()
		toolName, text, result, payload, err := h.image.ExecuteWithPath(ctx, tool, prompt, args, imagePath)
		latency := time.Since(start)

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			h.logImageUsage(key, model, toolName, latency, false, err.Error(), nil, nil, c.ClientIP())
		} else {
			task.Status = "completed"
			task.Tool = toolName
			task.Text = text
			task.Result, _ = json.Marshal(result)
			h.logImageUsage(key, model, toolName, latency, true, "", nil, result, c.ClientIP())
		}
		if saveErr := h.image.saveTask(task); saveErr != nil {
			return
		}
		_ = payload
	}(task)

	c.JSON(http.StatusOK, gin.H{
		"task_id": task.ID,
		"status":  "processing",
	})
}

// ImageUnderstandingStream SSE 事件流
func (h *APIGatewayHandler) ImageUnderstandingStream(c *gin.Context) {
	taskID := c.Param("id")

	task, ok := h.image.getTask(taskID)

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

	// 如果任务已完成，直接返回结果
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
			t, found := h.image.getTask(taskID)
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

func (h *APIGatewayHandler) logImageUsage(key *models.AIAPIKey, model, tool string, latency time.Duration, success bool, errMsg string, req interface{}, resp interface{}, clientIP string) {
	if h.ai == nil {
		return
	}
	requestBody, _ := json.Marshal(req)
	responseBody, _ := json.Marshal(resp)
	h.ai.logAPIRequest(
		key,
		model,
		"minimax_mcp",
		"/api/api-gateway/v1/image/understanding",
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

func saveMultipartToTemp(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	return writeMultipartToTemp(fileHeader.Filename, file)
}

func joinURLPath(basePath, appendPath string) string {
	if appendPath == "" {
		if basePath == "" {
			return "/"
		}
		return basePath
	}

	baseHasSlash := strings.HasSuffix(basePath, "/")
	appendHasSlash := strings.HasPrefix(appendPath, "/")

	switch {
	case baseHasSlash && appendHasSlash:
		return basePath + strings.TrimPrefix(appendPath, "/")
	case !baseHasSlash && !appendHasSlash:
		return basePath + "/" + appendPath
	default:
		return basePath + appendPath
	}
}
