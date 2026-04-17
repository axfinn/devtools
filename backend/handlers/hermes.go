package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

type HermesHandler struct {
	adminPassword string
	dashboardURL  string
	apiBaseURL    string
	apiKey        string
	model         string
	httpClient    *http.Client
}

func NewHermesHandler(cfg config.HermesConfig) *HermesHandler {
	return &HermesHandler{
		adminPassword: strings.TrimSpace(cfg.AdminPassword),
		dashboardURL:  strings.TrimRight(strings.TrimSpace(cfg.DashboardURL), "/"),
		apiBaseURL:    strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/"),
		apiKey:        strings.TrimSpace(cfg.APIKey),
		model:         strings.TrimSpace(cfg.Model),
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
	}
}

func (h *HermesHandler) checkAdmin(password string) bool {
	return h.adminPassword != "" && password == h.adminPassword
}

func (h *HermesHandler) adminPasswordFromRequest(c *gin.Context) string {
	password := strings.TrimSpace(c.Query("admin_password"))
	if password != "" {
		return password
	}
	return strings.TrimSpace(c.GetHeader("X-Admin-Password"))
}

func (h *HermesHandler) verifyAdmin(c *gin.Context) bool {
	if !h.checkAdmin(h.adminPasswordFromRequest(c)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return false
	}
	return true
}

func (h *HermesHandler) requestJSON(method, url string, body any, withAuth bool) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if withAuth && h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if readErr != nil {
		return resp.StatusCode, nil, readErr
	}
	return resp.StatusCode, data, nil
}

func (h *HermesHandler) dashboardStatus() gin.H {
	if h.dashboardURL == "" {
		return gin.H{"ok": false, "error": "未配置 dashboard_url"}
	}

	statusCode, body, err := h.requestJSON(http.MethodGet, h.dashboardURL+"/", nil, false)
	if err != nil {
		return gin.H{"ok": false, "error": err.Error()}
	}
	return gin.H{
		"ok":          statusCode >= 200 && statusCode < 400,
		"status_code": statusCode,
		"preview":     string(body),
	}
}

func (h *HermesHandler) gatewayStatus() gin.H {
	if h.apiBaseURL == "" {
		return gin.H{"ok": false, "error": "未配置 api_base_url"}
	}

	healthURL := strings.TrimSuffix(h.apiBaseURL, "/v1") + "/health"
	statusCode, body, err := h.requestJSON(http.MethodGet, healthURL, nil, false)
	if err != nil {
		return gin.H{"ok": false, "error": err.Error()}
	}

	var parsed any
	_ = json.Unmarshal(body, &parsed)

	return gin.H{
		"ok":          statusCode >= 200 && statusCode < 400,
		"status_code": statusCode,
		"body":        parsed,
	}
}

// VerifyPassword POST /api/hermes/verify
func (h *HermesHandler) VerifyPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Status GET /api/hermes/status?admin_password=xxx
func (h *HermesHandler) Status(c *gin.Context) {
	if !h.verifyAdmin(c) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dashboard_url": h.dashboardURL,
		"api_base_url":  h.apiBaseURL,
		"api_key_set":   h.apiKey != "",
		"model":         h.model,
		"dashboard":     h.dashboardStatus(),
		"gateway":       h.gatewayStatus(),
	})
}

// Models GET /api/hermes/models?admin_password=xxx
func (h *HermesHandler) Models(c *gin.Context) {
	if !h.verifyAdmin(c) {
		return
	}
	if h.apiBaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 api_base_url"})
		return
	}

	statusCode, body, err := h.requestJSON(http.MethodGet, h.apiBaseURL+"/models", nil, true)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "请求 Hermes 失败: " + err.Error()})
		return
	}
	if statusCode < 200 || statusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       "Hermes 返回异常",
			"status_code": statusCode,
			"body":        string(body),
		})
		return
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "解析模型列表失败"})
		return
	}
	c.JSON(http.StatusOK, payload)
}

// Chat POST /api/hermes/chat
func (h *HermesHandler) Chat(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
		Message  string `json:"message" binding:"required"`
		System   string `json:"system"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if !h.checkAdmin(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	if h.apiBaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置 api_base_url"})
		return
	}

	messages := []map[string]string{}
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": strings.TrimSpace(req.System),
		})
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": strings.TrimSpace(req.Message),
	})

	model := h.model
	if model == "" {
		model = "hermes-agent"
	}

	payload := gin.H{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	statusCode, body, err := h.requestJSON(http.MethodPost, h.apiBaseURL+"/chat/completions", payload, true)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "请求 Hermes 失败: " + err.Error()})
		return
	}
	if statusCode < 200 || statusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":       "Hermes 返回异常",
			"status_code": statusCode,
			"body":        string(body),
		})
		return
	}

	var response struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage any `json:"usage"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "解析 Hermes 响应失败"})
		return
	}

	answer := ""
	if len(response.Choices) > 0 {
		answer = response.Choices[0].Message.Content
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     response.ID,
		"model":  response.Model,
		"answer": answer,
		"finish_reason": func() string {
			if len(response.Choices) == 0 {
				return ""
			}
			return response.Choices[0].FinishReason
		}(),
		"usage": response.Usage,
		"curl_example": fmt.Sprintf(
			"curl %s/chat/completions -H 'Authorization: Bearer %s' -H 'Content-Type: application/json' -d '{\"model\":\"%s\",\"messages\":[{\"role\":\"user\",\"content\":\"%s\"}],\"stream\":false}'",
			h.apiBaseURL,
			h.apiKey,
			model,
			strings.ReplaceAll(strings.TrimSpace(req.Message), "'", "\\'"),
		),
	})
}
