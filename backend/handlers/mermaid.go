package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

type MermaidHandler struct {
	db     *models.DB
	cfg    *config.Config
	client *http.Client
}

func NewMermaidHandler(db *models.DB, cfg *config.Config) *MermaidHandler {
	return &MermaidHandler{
		db:  db,
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{Proxy: nil},
		},
	}
}

// ─── 档案 ────────────────────────────────────────────────────

func (h *MermaidHandler) CreateProject(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.db.CreateMermaidProject(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *MermaidHandler) ListProjects(c *gin.Context) {
	list, err := h.db.ListMermaidProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []models.MermaidProject{}
	}
	c.JSON(http.StatusOK, gin.H{"projects": list})
}

func (h *MermaidHandler) DeleteProject(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteMermaidProject(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── 版本 ────────────────────────────────────────────────────

func (h *MermaidHandler) SaveVersion(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.db.AddMermaidVersion(id, req.Code, "manual")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *MermaidHandler) ListVersions(c *gin.Context) {
	id := c.Param("id")
	list, err := h.db.ListMermaidVersions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []models.MermaidVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"versions": list})
}

// ─── 对话历史 ─────────────────────────────────────────────────

func (h *MermaidHandler) ListMessages(c *gin.Context) {
	id := c.Param("id")
	list, err := h.db.ListMermaidMessages(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []models.MermaidMessage{}
	}
	c.JSON(http.StatusOK, gin.H{"messages": list})
}

func (h *MermaidHandler) ClearMessages(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.ClearMermaidMessages(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── AI 生成 / 微调 ───────────────────────────────────────────

var mermaidCodeRe = regexp.MustCompile("(?s)```(?:mermaid)?\\s*\\n?(.*?)```")

func extractMermaidCode(text string) string {
	if m := mermaidCodeRe.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// 没有代码块，直接返回去掉首尾空白的内容
	return strings.TrimSpace(text)
}

func (h *MermaidHandler) AIGenerate(c *gin.Context) {
	projectID := c.Param("id")

	var req struct {
		Prompt      string `json:"prompt" binding:"required"`
		CurrentCode string `json:"current_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.cfg.MiniMax.APIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 MiniMax API Key"})
		return
	}

	// 读取历史对话
	history, err := h.db.ListMermaidMessages(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构造 messages
	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var msgs []anthropicMsg
	for _, m := range history {
		msgs = append(msgs, anthropicMsg{Role: m.Role, Content: m.Content})
	}

	// 本次用户消息：如果有当前代码，附在 prompt 里
	userContent := req.Prompt
	if req.CurrentCode != "" {
		userContent = fmt.Sprintf("%s\n\n当前图表代码：\n```mermaid\n%s\n```", req.Prompt, req.CurrentCode)
	}
	msgs = append(msgs, anthropicMsg{Role: "user", Content: userContent})

	systemPrompt := `你是一个专业的 Mermaid 图表生成助手。
用户会描述他们想要的图表，或者要求修改现有图表。
你必须只返回 Mermaid 代码，用 ` + "```mermaid" + ` 和 ` + "```" + ` 包裹，不要有任何其他解释文字。
支持的图表类型：flowchart、sequenceDiagram、classDiagram、stateDiagram-v2、erDiagram、gantt、pie、mindmap、timeline、gitGraph、quadrantChart、sankey-beta。`

	body := map[string]interface{}{
		"model":      "MiniMax-M2.7",
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages":   msgs,
	}
	bodyBytes, _ := json.Marshal(body)

	httpReq, err := http.NewRequest("POST", "https://api.minimaxi.com/anthropic/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", h.cfg.MiniMax.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "调用 MiniMax 失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "读取响应失败"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "MiniMax 返回错误: " + strconv.Itoa(resp.StatusCode) + " " + string(raw)})
		return
	}

	// 解析 Anthropic 响应
	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &anthropicResp); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "解析响应失败: " + err.Error()})
		return
	}

	var fullText string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			fullText += block.Text
		}
	}

	code := extractMermaidCode(fullText)
	if code == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI 未返回有效的 Mermaid 代码"})
		return
	}

	// 保存对话历史
	h.db.AddMermaidMessage(projectID, "user", userContent)
	h.db.AddMermaidMessage(projectID, "assistant", "```mermaid\n"+code+"\n```")

	// 保存版本
	v, err := h.db.AddMermaidVersion(projectID, code, "ai")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code, "version_id": v.ID})
}
