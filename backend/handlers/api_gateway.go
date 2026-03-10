package handlers

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"devtools/models"

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

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		proxyPath := strings.TrimPrefix(req.URL.Path, "/api/api-gateway/cpa/v1")
		if proxyPath == "" {
			proxyPath = "/"
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
	proxy.Transport = &http.Transport{
		Proxy: nil,
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
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
