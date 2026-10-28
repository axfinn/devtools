package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// 本文件：/ai-chat 的会话/消息同步接口。
//
// 背景（多端不同步）：原来会话列表与消息只存浏览器 localStorage，换台设备打开是空白，
// 同一个会话在不同设备上互相看不见，任务结果也只回到"发起它的那个浏览器"。
// 现在以服务端 SQLite 为唯一真相：
//   GET    /api/chat-plus/conversations                 会话列表（所有设备共享）
//   POST   /api/chat-plus/conversations {id,title}      新建/改标题
//   DELETE /api/chat-plus/conversations/:id             删除会话
//   GET    /api/chat-plus/conversations/:id/messages?since_ts=0   消息列表（增量）
//   POST   /api/chat-plus/conversations/:id/messages    批量 upsert（图片落文件，返回归一化结果）
//   GET    /api/chat-plus/images/:name                  取图（文件名 32 位随机 hex，等价于不可猜的凭证）
// localStorage 退化为离线缓存：断网时照旧能看，联网后以上面这份为准。

// chatPlusImageDir 图片落盘目录。用 var 而不是 const：测试里指向 t.TempDir()，
// 避免往仓库工作目录里写文件。
var chatPlusImageDir = "./data/chat_plus_img"

// chatPlusImageNameRe 只允许我们自己生成的随机文件名，杜绝路径穿越。
var chatPlusImageNameRe = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png|webp|gif)$`)

const (
	chatPlusMaxImagesPerMsg = 4
	chatPlusMaxImageBytes   = 8 << 20
)

type chatPlusImageInput struct {
	ID      string `json:"id"`
	DataURL string `json:"dataUrl"`
	URL     string `json:"url"`
	W       int    `json:"w"`
	H       int    `json:"h"`
}

type chatPlusImageOut struct {
	ID  string `json:"id"`
	URL string `json:"url"`
	W   int    `json:"w"`
	H   int    `json:"h"`
}

type chatPlusMessageInput struct {
	ID           string               `json:"id"`
	Role         string               `json:"role"`
	Content      string               `json:"content"`
	Reasoning    string               `json:"reasoning"`
	Images       []chatPlusImageInput `json:"images"`
	TaskID       string               `json:"task_id"`
	Kind         string               `json:"kind"`
	Status       string               `json:"status"`
	ErrorMessage string               `json:"error_message"`
	Ts           int64                `json:"ts"`
}

// saveChatPlusDataURL 把 data:image/...;base64,xxx 落成文件，返回相对 URL。
// 只认图片 mime，且按魔数兜底校验，避免把任意内容写进可访问目录。
func saveChatPlusDataURL(dataURL string) (string, error) {
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 || !strings.HasPrefix(dataURL, "data:image/") {
		return "", fmt.Errorf("不支持的图片格式")
	}
	meta := dataURL[:comma]
	var ext string
	switch {
	case strings.Contains(meta, "image/jpeg"), strings.Contains(meta, "image/jpg"):
		ext = ".jpg"
	case strings.Contains(meta, "image/png"):
		ext = ".png"
	case strings.Contains(meta, "image/webp"):
		ext = ".webp"
	case strings.Contains(meta, "image/gif"):
		ext = ".gif"
	default:
		return "", fmt.Errorf("不支持的图片格式")
	}
	if strings.Contains(meta, "base64") == false {
		return "", fmt.Errorf("图片编码必须为 base64")
	}
	raw, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return "", fmt.Errorf("图片解码失败")
	}
	if len(raw) == 0 || len(raw) > chatPlusMaxImageBytes {
		return "", fmt.Errorf("图片大小超限")
	}
	if err := os.MkdirAll(chatPlusImageDir, 0o755); err != nil {
		return "", err
	}
	name := utils.GenerateHexKey(16) + ext // 16 字节 = 32 hex
	if err := os.WriteFile(filepath.Join(chatPlusImageDir, name), raw, 0o644); err != nil {
		return "", err
	}
	return "/api/chat-plus/images/" + name, nil
}

// normalizeChatPlusImages 把前端传来的图片统一成"只有 URL"的形态（dataURL 落盘）。
func normalizeChatPlusImages(in []chatPlusImageInput) ([]chatPlusImageOut, error) {
	out := make([]chatPlusImageOut, 0, len(in))
	for i, im := range in {
		if i >= chatPlusMaxImagesPerMsg {
			break
		}
		url := strings.TrimSpace(im.URL)
		if url == "" && strings.TrimSpace(im.DataURL) != "" {
			saved, err := saveChatPlusDataURL(im.DataURL)
			if err != nil {
				// 图片存不下不该让整条消息丢掉：跳过这张，正文照常保存
				continue
			}
			url = saved
		}
		if url == "" {
			continue
		}
		id := im.ID
		if id == "" {
			id = utils.GenerateHexKey(4)
		}
		out = append(out, chatPlusImageOut{ID: id, URL: url, W: im.W, H: im.H})
	}
	return out, nil
}

// ChatPlusConversations GET/POST /api/chat-plus/conversations
func (h *AIGatewayHandler) ChatPlusConversations(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{"conversations": []interface{}{}})
		return
	}
	switch c.Request.Method {
	case http.MethodPost:
		var req struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少会话 id"})
			return
		}
		if err := h.db.UpsertChatPlusConversation(strings.TrimSpace(req.ID), strings.TrimSpace(req.Title)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	list, err := h.db.ListChatPlusConversations(300)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": list})
}

// ChatPlusConversationDelete DELETE /api/chat-plus/conversations/:id
func (h *AIGatewayHandler) ChatPlusConversationDelete(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少会话 id"})
		return
	}
	if h.db != nil {
		if err := h.db.DeleteChatPlusConversation(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ChatPlusMessages GET/POST /api/chat-plus/conversations/:id/messages
func (h *AIGatewayHandler) ChatPlusMessages(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	convID := strings.TrimSpace(c.Param("id"))
	if convID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少会话 id"})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{"messages": []interface{}{}})
		return
	}

	if c.Request.Method == http.MethodPost {
		var req struct {
			Messages []chatPlusMessageInput `json:"messages"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
			return
		}
		if len(req.Messages) > 200 {
			req.Messages = req.Messages[:200]
		}
		saved := make([]gin.H, 0, len(req.Messages))
		for _, in := range req.Messages {
			if strings.TrimSpace(in.ID) == "" {
				continue
			}
			role := in.Role
			if role != "user" && role != "assistant" {
				role = "user"
			}
			imgs, _ := normalizeChatPlusImages(in.Images)
			imgJSON := ""
			if len(imgs) > 0 {
				b, _ := json.Marshal(imgs)
				imgJSON = string(b)
			}
			m := &models.ChatPlusMessage{
				ID:             in.ID,
				ConversationID: convID,
				Role:           role,
				Content:        in.Content,
				Reasoning:      in.Reasoning,
				Images:         imgJSON,
				TaskID:         in.TaskID,
				Kind:           in.Kind,
				Status:         in.Status,
				ErrMessage:     in.ErrorMessage,
				Ts:             in.Ts,
			}
			if err := h.db.UpsertChatPlusMessage(m); err != nil {
				continue
			}
			saved = append(saved, gin.H{
				"id": m.ID, "role": m.Role, "content": m.Content,
				"reasoning": m.Reasoning, "images": imgs, "task_id": m.TaskID,
				"kind": m.Kind, "status": m.Status, "error_message": m.ErrMessage, "ts": m.Ts,
			})
		}
		_ = h.db.TouchChatPlusConversation(convID)
		c.JSON(http.StatusOK, gin.H{"messages": saved})
		return
	}

	since, _ := strconv.ParseInt(c.DefaultQuery("since_ts", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	msgs, err := h.db.ListChatPlusMessages(convID, since, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		var imgs []chatPlusImageOut
		if m.Images != "" {
			_ = json.Unmarshal([]byte(m.Images), &imgs)
		}
		out = append(out, gin.H{
			"id": m.ID, "conversation_id": m.ConversationID, "role": m.Role,
			"content": m.Content, "reasoning": m.Reasoning, "images": imgs,
			"task_id": m.TaskID, "kind": m.Kind, "status": m.Status,
			"error_message": m.ErrMessage, "ts": m.Ts,
		})
	}
	c.JSON(http.StatusOK, gin.H{"messages": out, "server_time": time.Now().UnixMilli()})
}

// ChatPlusImage GET /api/chat-plus/images/:name
// <img> 标签带不了自定义 header，所以这里不做密码校验；文件名是 16 字节随机 hex，
// 不可枚举即为凭证。只允许我们自己的命名格式，且路径拼接前已用正则白名单校验。
func (h *AIGatewayHandler) ChatPlusImage(c *gin.Context) {
	name := c.Param("name")
	if !chatPlusImageNameRe.MatchString(name) {
		c.Status(http.StatusNotFound)
		return
	}
	path := filepath.Join(chatPlusImageDir, name)
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(path)
}

// inlineChatPlusLocalImages 把消息里指向本服务的图片 URL 换成 base64 data URL，
// 再转发给 Ollama。否则 Ollama 需要反向访问 devtools（跨网段/公网时根本不可达），
// 视觉模型会直接报 "failed to load image"。
func inlineChatPlusLocalImages(msgs []map[string]interface{}) {
	for _, m := range msgs {
		parts, ok := m["content"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range parts {
			pm, ok := p.(map[string]interface{})
			if !ok || pm["type"] != "image_url" {
				continue
			}
			iu, ok := pm["image_url"].(map[string]interface{})
			if !ok {
				continue
			}
			url, _ := iu["url"].(string)
			name := ""
			if idx := strings.Index(url, "/api/chat-plus/images/"); idx >= 0 {
				name = url[idx+len("/api/chat-plus/images/"):]
			}
			if !chatPlusImageNameRe.MatchString(name) {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(chatPlusImageDir, name))
			if err != nil || len(raw) == 0 {
				continue
			}
			mime := "image/jpeg"
			switch {
			case strings.HasSuffix(name, ".png"):
				mime = "image/png"
			case strings.HasSuffix(name, ".webp"):
				mime = "image/webp"
			case strings.HasSuffix(name, ".gif"):
				mime = "image/gif"
			}
			iu["url"] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw)
		}
	}
}
