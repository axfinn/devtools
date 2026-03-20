package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	maxImageSize = 5 * 1024 * 1024  // 5MB
	maxVideoSize = 50 * 1024 * 1024 // 50MB
	maxAudioSize = 10 * 1024 * 1024 // 10MB
	maxFileSize  = 20 * 1024 * 1024 // 20MB 通用文件
	uploadDir    = "./data/uploads"
)

// 允许的文件扩展名白名单
var allowedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".mp4": true, ".webm": true, ".mov": true,
	".mp3": true, ".wav": true, ".ogg": true,
	".pdf": true, ".zip": true,
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// BotConfig 机器人配置
type BotConfig struct {
	Enabled      bool   `json:"enabled"`
	Nickname     string `json:"nickname"`
	Role         string `json:"role"`
	SystemPrompt string `json:"system_prompt"`
	EnableTTS    bool   `json:"enable_tts"`   // 是否开启语音朗读
	TTSVoice     string `json:"tts_voice"`    // 语音名，留空用角色默认
	TTSEngine    string `json:"tts_engine"`   // "auto" | "kokoro" | "edge-tts"
}

// 角色默认 edge-tts 语音映射
var botRoleVoices = map[string]string{
	"general":      "zh-CN-XiaoxiaoNeural",
	"code":         "zh-CN-YunxiNeural",
	"translate":    "zh-CN-XiaoyiNeural",
	"customer":     "zh-CN-XiaoxiaoNeural",
	"psychology":   "zh-CN-XiaomouNeural",
	"student_girl": "zh-CN-XiaoyiNeural",
	"college":      "zh-CN-YunxiNeural",
	"girlfriend":   "zh-CN-XiaoxiaoNeural",
	"uncle":        "zh-CN-YunyangNeural",
	"kid":          "zh-CN-YunxiNeural",
}

// BotRoleTemplate 预设角色模板
type BotRoleTemplate struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Nickname     string `json:"nickname"`
	SystemPrompt string `json:"system_prompt"`
}

var botRoleTemplates = []BotRoleTemplate{
	{Key: "general", Name: "通用助手", Nickname: "🤖 小助手", SystemPrompt: "你是一个友好的通用助手，用简洁清晰的语言回答用户问题。"},
	{Key: "code", Name: "代码助手", Nickname: "🤖 码农", SystemPrompt: "你是一个专业的编程助手，帮助用户解决代码问题，回复中适当使用代码块。"},
	{Key: "translate", Name: "翻译助手", Nickname: "🤖 译者", SystemPrompt: "你是一个翻译助手。用户发中文时翻译成英文，发英文时翻译成中文，其他语言翻译成中文，只输出译文。"},
	{Key: "customer", Name: "客服助手", Nickname: "🤖 客服", SystemPrompt: "你是一个礼貌专业的客服助手，耐心回答用户问题，语气温和友善。"},
	{Key: "psychology", Name: "心理咨询师", Nickname: "🧠 心理咨询师", SystemPrompt: "你是一位专业的心理咨询师，擅长倾听、共情，用温暖支持的语言帮助用户疏导情绪，不做诊断，鼓励专业就医。"},
	{Key: "student_girl", Name: "学生妹", Nickname: "🎀 学生妹", SystemPrompt: "你是一个活泼可爱的女大学生，说话带点俏皮和学生气息，喜欢用emoji，对什么都充满好奇。"},
	{Key: "college", Name: "大学生", Nickname: "🎓 大学生", SystemPrompt: "你是一个普通的男大学生，说话随意接地气，偶尔夹带网络用语，聊天风格轻松幽默。"},
	{Key: "girlfriend", Name: "电子女朋友", Nickname: "💕 小甜甜", SystemPrompt: "你是用户的电子女朋友，温柔体贴、善解人意，会撒娇，关心用户的日常生活和情绪，语气甜蜜可爱。"},
	{Key: "uncle", Name: "大叔", Nickname: "🧔 大叔", SystemPrompt: "你是一个阅历丰富的大叔，说话沉稳有见地，偶尔唠叨，但都是真心话，会用生活经验给年轻人一些建议。"},
	{Key: "kid", Name: "小屁孩", Nickname: "👦 小屁孩", SystemPrompt: "你是一个七八岁的小孩，说话天真烂漫，喜欢问为什么，喜欢玩，会用儿童视角看世界，偶尔冒出令人捧腹的童言童语。"},
}

// botMessage MiniMax 对话历史条目
type botMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatHandler struct {
	db            *models.DB
	rooms         map[string]*Room
	mu            sync.RWMutex
	adminPassword string
	minimax       config.MiniMaxConfig
	ttsServiceURL string
}

type Room struct {
	ID      string
	clients map[*Client]bool
	mu      sync.RWMutex
	bot     *BotConfig
	history []botMessage // 最近 20 条对话上下文
}

type Client struct {
	conn     *websocket.Conn
	room     *Room
	nickname string
	send     chan []byte
}

type WSMessage struct {
	Type         string `json:"type"`
	Content      string `json:"content,omitempty"`
	Nickname     string `json:"nickname,omitempty"`
	ID           int64  `json:"id,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	MsgType      string `json:"msg_type,omitempty"`
	OriginalName string `json:"original_name,omitempty"`
	AudioURL     string `json:"audio_url,omitempty"` // TTS 音频 URL（仅 bot 消息）
}

func NewChatHandler(db *models.DB, adminPassword string, minimax config.MiniMaxConfig, ttsServiceURL string) *ChatHandler {
	return &ChatHandler{
		db:            db,
		rooms:         make(map[string]*Room),
		adminPassword: adminPassword,
		minimax:       minimax,
		ttsServiceURL: ttsServiceURL,
	}
}

type CreateRoomRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password"`
}

type JoinRoomRequest struct {
	Password string `json:"password"`
	Nickname string `json:"nickname" binding:"required"`
}

// CreateRoom 创建聊天室
func (h *ChatHandler) CreateRoom(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "房间名称不能为空", "code": 400})
		return
	}

	if len(req.Name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "房间名称不能超过50个字符", "code": 400})
		return
	}

	room := &models.ChatRoom{
		Name:      req.Name,
		CreatorIP: c.ClientIP(),
	}

	if req.Password != "" {
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
			return
		}
		room.Password = hashedPassword
	}

	if err := h.db.CreateRoom(room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建房间失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           room.ID,
		"name":         room.Name,
		"has_password": room.Password != "",
		"created_at":   room.CreatedAt,
	})
}

// GetRoom 获取房间信息
func (h *ChatHandler) GetRoom(c *gin.Context) {
	id := c.Param("id")

	room, err := h.db.GetRoom(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             room.ID,
		"name":           room.Name,
		"has_password":   room.HasPassword,
		"last_active_at": room.LastActiveAt,
		"created_at":     room.CreatedAt,
	})
}

// GetRooms 获取房间列表
func (h *ChatHandler) GetRooms(c *gin.Context) {
	rooms, err := h.db.GetRooms(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取房间列表失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

// GetRoomMessages 获取聊天室历史消息
func (h *ChatHandler) GetRoomMessages(c *gin.Context) {
	id := c.Param("id")

	// 检查房间是否存在
	if !h.db.RoomExists(id) {
		log.Printf("房间不存在: %s", id)
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}

	// 获取最近100条消息
	messages, err := h.db.GetMessages(id, 100)
	if err != nil {
		log.Printf("获取消息失败 (room_id=%s): %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败", "code": 500})
		return
	}

	// 转换消息格式为前端需要的格式
	var result []gin.H
	for _, msg := range messages {
		result = append(result, gin.H{
			"id":            msg.ID,
			"nickname":      msg.Nickname,
			"content":       msg.Content,
			"msg_type":      msg.MsgType,
			"original_name": msg.OriginalName,
			"created_at":    msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	log.Printf("成功获取 %d 条历史消息 (room_id=%s)", len(result), id)
	c.JSON(http.StatusOK, gin.H{"messages": result})
}

// JoinRoom 验证密码加入房间
func (h *ChatHandler) JoinRoom(c *gin.Context) {
	id := c.Param("id")
	var req JoinRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "昵称不能为空", "code": 400})
		return
	}

	if len(req.Nickname) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "昵称不能超过20个字符", "code": 400})
		return
	}

	room, err := h.db.GetRoom(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}

	// 验证密码
	if room.Password != "" {
		if req.Password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要密码", "code": 401, "has_password": true})
			return
		}
		if !utils.VerifyPassword(req.Password, room.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
			return
		}
	}

	// 获取历史消息
	messages, _ := h.db.GetMessages(id, 100)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"room":     room,
		"messages": messages,
	})
}

// WebSocket 处理
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	roomID := c.Param("id")
	nickname := c.Query("nickname")

	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "昵称不能为空"})
		return
	}

	// 检查房间是否存在
	if !h.db.RoomExists(roomID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// 获取或创建房间
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	if !exists {
		room = &Room{
			ID:      roomID,
			clients: make(map[*Client]bool),
		}
		// 从数据库恢复机器人配置
		if botJSON, err := h.db.LoadBotConfig(roomID); err == nil && botJSON != "" {
			var bc BotConfig
			if json.Unmarshal([]byte(botJSON), &bc) == nil {
				room.bot = &bc
			}
		}
		h.rooms[roomID] = room
	}
	h.mu.Unlock()

	client := &Client{
		conn:     conn,
		room:     room,
		nickname: nickname,
		send:     make(chan []byte, 1024),
	}

	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	// 更新房间活跃时间
	h.db.UpdateRoomActivity(roomID)

	// 广播加入消息
	joinMsg := WSMessage{
		Type:    "system",
		Content: nickname + " 加入了房间",
	}
	h.broadcast(room, joinMsg, nil)

	go h.writePump(client)
	go h.readPump(client, roomID)
}

func (h *ChatHandler) readPump(client *Client, roomID string) {
	defer func() {
		h.removeClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(65536) // 64KB
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "message" && msg.Content != "" {
			// 限制消息长度
			if len(msg.Content) > 5000 {
				msg.Content = msg.Content[:5000]
			}

			// 保存消息到数据库
			chatMsg := &models.ChatMessage{
				RoomID:   roomID,
				Nickname: client.nickname,
				Content:  msg.Content,
				MsgType:  "text",
			}
			if msg.MsgType != "" {
				chatMsg.MsgType = msg.MsgType
			}
			if msg.OriginalName != "" {
				chatMsg.OriginalName = msg.OriginalName
			}
			h.db.CreateMessage(chatMsg)

			// 更新房间活跃时间
			h.db.UpdateRoomActivity(roomID)

			// 广播消息
			broadcastMsg := WSMessage{
				Type:         "message",
				ID:           chatMsg.ID,
				Nickname:     client.nickname,
				Content:      msg.Content,
				MsgType:      chatMsg.MsgType,
				OriginalName: chatMsg.OriginalName,
				CreatedAt:    chatMsg.CreatedAt.Format(time.RFC3339),
			}
			h.broadcast(client.room, broadcastMsg, nil)

			// 异步触发机器人回复（仅文本消息）
			if chatMsg.MsgType == "text" || chatMsg.MsgType == "" {
				go h.triggerBotReply(client.room, roomID, client.nickname, msg.Content)
			}
		}
	}
}

func (h *ChatHandler) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *ChatHandler) broadcast(room *Room, msg WSMessage, exclude *Client) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	// 收集发送失败的客户端（避免在持有读锁时修改 map）
	var failedClients []*Client

	room.mu.RLock()
	for client := range room.clients {
		if client != exclude {
			select {
			case client.send <- data:
			default:
				failedClients = append(failedClients, client)
			}
		}
	}
	room.mu.RUnlock()

	// 使用写锁移除失败的客户端
	if len(failedClients) > 0 {
		room.mu.Lock()
		for _, client := range failedClients {
			if _, ok := room.clients[client]; ok {
				close(client.send)
				delete(room.clients, client)
			}
		}
		room.mu.Unlock()
	}
}

func (h *ChatHandler) removeClient(client *Client) {
	room := client.room
	room.mu.Lock()
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
		close(client.send)
	}
	room.mu.Unlock()

	// 广播离开消息
	leaveMsg := WSMessage{
		Type:    "system",
		Content: client.nickname + " 离开了房间",
	}
	h.broadcast(room, leaveMsg, nil)

	// 如果房间没有人了，从内存中移除
	h.mu.Lock()
	if len(room.clients) == 0 {
		delete(h.rooms, room.ID)
	}
	h.mu.Unlock()
}

// UploadFile 上传文件（图片、视频、音频、文档）
func (h *ChatHandler) UploadFile(c *gin.Context) {
	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		// 兼容旧的 "image" 字段名
		file, header, err = c.Request.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件", "code": 400})
			return
		}
	}
	defer file.Close()

	// 读取文件头用于魔数检测
	magicBytes := make([]byte, 16)
	n, _ := file.Read(magicBytes)
	magicBytes = magicBytes[:n]

	// 重置文件指针
	file.Seek(0, 0)

	// 检测真实文件类型
	detectedType := detectFileType(magicBytes)
	declaredType := header.Header.Get("Content-Type")

	// 确定文件类别
	var fileCategory string
	var maxSize int64
	var sizeLabel string

	if strings.HasPrefix(detectedType, "image/") || strings.HasPrefix(declaredType, "image/") {
		fileCategory = "image"
		maxSize = maxImageSize
		sizeLabel = "5MB"
	} else if strings.HasPrefix(detectedType, "video/") || strings.HasPrefix(declaredType, "video/") {
		fileCategory = "video"
		maxSize = maxVideoSize
		sizeLabel = "50MB"
	} else if strings.HasPrefix(detectedType, "audio/") || strings.HasPrefix(declaredType, "audio/") {
		fileCategory = "audio"
		maxSize = maxAudioSize
		sizeLabel = "10MB"
	} else {
		fileCategory = "file"
		maxSize = maxFileSize
		sizeLabel = "20MB"
	}

	// 对于图片、视频、音频，验证魔数匹配
	if fileCategory == "image" || fileCategory == "video" || fileCategory == "audio" {
		if detectedType == "" {
			// 无法检测到有效魔数，但声明是媒体文件，允许但记录警告
			log.Printf("Warning: Cannot detect magic number for file %s (declared: %s)", header.Filename, declaredType)
		} else if fileCategory == "image" && !strings.HasPrefix(detectedType, "image/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件类型与内容不匹配", "code": 400})
			return
		} else if fileCategory == "video" && !strings.HasPrefix(detectedType, "video/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件类型与内容不匹配", "code": 400})
			return
		} else if fileCategory == "audio" && !strings.HasPrefix(detectedType, "audio/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件类型与内容不匹配", "code": 400})
			return
		}
	}

	// 检查文件大小
	if header.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("文件大小不能超过%s", sizeLabel),
			"code":  400,
		})
		return
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = getExtFromMimeType(detectedType)
		if ext == "" {
			ext = getExtFromMimeType(declaredType)
		}
	}

	// 扩展名白名单验证
	if ext != "" && !allowedExtensions[ext] {
		ext = "" // 移除不安全的扩展名
	}

	// 生成随机文件名
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}
	filename := fmt.Sprintf("%s%s", hex.EncodeToString(randomBytes), ext)

	// 确保上传目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}

	// 保存文件
	filePath := filepath.Join(uploadDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
		return
	}

	// 返回文件URL和类型
	fileURL := "/api/chat/uploads/" + filename
	c.JSON(http.StatusOK, gin.H{
		"url":           fileURL,
		"filename":      filename,
		"original_name": header.Filename,
		"type":          fileCategory,
		"size":          header.Size,
	})
}

// UploadImage 保持向后兼容
func (h *ChatHandler) UploadImage(c *gin.Context) {
	h.UploadFile(c)
}

// AdminListRooms 管理员列出所有房间
func (h *ChatHandler) AdminListRooms(c *gin.Context) {
	adminPassword := c.Query("admin_password")
	if adminPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要管理员密码", "code": 400})
		return
	}

	if h.adminPassword == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员功能未启用", "code": 403})
		return
	}

	if adminPassword != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	// 获取所有房间（增加限制）
	rooms, err := h.db.GetRooms(1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取房间列表失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rooms": rooms, "total": len(rooms)})
}

// GetBotConfig 获取房间机器人配置及所有预设模板
func (h *ChatHandler) GetBotConfig(c *gin.Context) {
	roomID := c.Param("id")
	if !h.db.RoomExists(roomID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}

	h.mu.RLock()
	room := h.rooms[roomID]
	h.mu.RUnlock()

	var bot *BotConfig
	if room != nil {
		room.mu.RLock()
		bot = room.bot
		room.mu.RUnlock()
	}

	c.JSON(http.StatusOK, gin.H{
		"bot":       bot,
		"templates": botRoleTemplates,
		"has_key":   h.minimax.APIKey != "",
	})
}

type AddBotRequest struct {
	Role         string `json:"role" binding:"required"`
	Nickname     string `json:"nickname"`
	SystemPrompt string `json:"system_prompt"`
	EnableTTS    bool   `json:"enable_tts"`
	TTSVoice     string `json:"tts_voice"`
	TTSEngine    string `json:"tts_engine"` // "auto" | "kokoro" | "edge-tts"
}

// AddBot 添加/更新房间机器人
func (h *ChatHandler) AddBot(c *gin.Context) {
	roomID := c.Param("id")
	if !h.db.RoomExists(roomID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}
	if h.minimax.APIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MiniMax API Key 未配置", "code": 503})
		return
	}

	var req AddBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色不能为空", "code": 400})
		return
	}

	// 查找预设模板
	var tmpl *BotRoleTemplate
	for i := range botRoleTemplates {
		if botRoleTemplates[i].Key == req.Role {
			tmpl = &botRoleTemplates[i]
			break
		}
	}
	if tmpl == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未知的角色模板", "code": 400})
		return
	}

	bc := &BotConfig{
		Enabled:      true,
		Role:         req.Role,
		Nickname:     tmpl.Nickname,
		SystemPrompt: tmpl.SystemPrompt,
		EnableTTS:    req.EnableTTS,
		TTSVoice:     botRoleVoices[req.Role],
		TTSEngine:    "auto",
	}
	if req.Nickname != "" {
		bc.Nickname = req.Nickname
	}
	if req.SystemPrompt != "" {
		bc.SystemPrompt = req.SystemPrompt
	}
	if req.TTSVoice != "" {
		bc.TTSVoice = req.TTSVoice
	}
	if req.TTSEngine != "" {
		bc.TTSEngine = req.TTSEngine
	}

	// 更新内存中的房间
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	if !exists {
		room = &Room{ID: roomID, clients: make(map[*Client]bool)}
		h.rooms[roomID] = room
	}
	h.mu.Unlock()

	room.mu.Lock()
	room.bot = bc
	room.history = nil
	room.mu.Unlock()

	// 持久化到数据库
	if data, err := json.Marshal(bc); err == nil {
		h.db.SaveBotConfig(roomID, string(data))
	}

	// 广播机器人加入消息
	h.broadcast(room, WSMessage{Type: "system", Content: bc.Nickname + " 加入了房间"}, nil)

	c.JSON(http.StatusOK, gin.H{"bot": bc})
}

// RemoveBot 移除房间机器人
func (h *ChatHandler) RemoveBot(c *gin.Context) {
	roomID := c.Param("id")
	if !h.db.RoomExists(roomID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}

	h.mu.RLock()
	room := h.rooms[roomID]
	h.mu.RUnlock()

	var nickname string
	if room != nil {
		room.mu.Lock()
		if room.bot != nil {
			nickname = room.bot.Nickname
			room.bot = nil
			room.history = nil
		}
		room.mu.Unlock()

		if nickname != "" {
			h.broadcast(room, WSMessage{Type: "system", Content: nickname + " 离开了房间"}, nil)
		}
	}

	h.db.SaveBotConfig(roomID, "")
	c.JSON(http.StatusOK, gin.H{"message": "机器人已移除"})
}

// triggerBotReply 异步生成机器人回复并广播
func (h *ChatHandler) triggerBotReply(room *Room, roomID, userNickname, userMsg string) {
	room.mu.RLock()
	bot := room.bot
	room.mu.RUnlock()
	if bot == nil || !bot.Enabled {
		return
	}

	// 更新对话历史
	room.mu.Lock()
	room.history = append(room.history, botMessage{Role: "user", Content: userNickname + ": " + userMsg})
	if len(room.history) > 20 {
		room.history = room.history[len(room.history)-20:]
	}
	historyCopy := make([]botMessage, len(room.history))
	copy(historyCopy, room.history)
	room.mu.Unlock()

	reply, err := h.callMiniMax(bot.SystemPrompt, historyCopy)
	if err != nil {
		log.Printf("Bot MiniMax error (room=%s): %v", roomID, err)
		return
	}

	// 保存并广播机器人回复
	botMsg := &models.ChatMessage{
		RoomID:   roomID,
		Nickname: bot.Nickname,
		Content:  reply,
		MsgType:  "text",
	}
	h.db.CreateMessage(botMsg)

	// 把机器人回复追加到历史
	room.mu.Lock()
	room.history = append(room.history, botMessage{Role: "assistant", Content: reply})
	if len(room.history) > 20 {
		room.history = room.history[len(room.history)-20:]
	}
	room.mu.Unlock()

	// 可选：TTS 生成音频
	var audioURL string
	if bot.EnableTTS {
		voice := bot.TTSVoice
		if voice == "" {
			voice = botRoleVoices[bot.Role]
		}
		if voice == "" {
			voice = "zh-CN-XiaoxiaoNeural"
		}
		engine := bot.TTSEngine
		if engine == "" {
			engine = "auto"
		}
		if url, ttsErr := h.callEdgeTTS(reply, voice, engine); ttsErr != nil {
			log.Printf("TTS error (room=%s): %v", roomID, ttsErr)
		} else {
			audioURL = url
		}
	}

	h.broadcast(room, WSMessage{
		Type:      "message",
		ID:        botMsg.ID,
		Nickname:  bot.Nickname,
		Content:   reply,
		MsgType:   "text",
		AudioURL:  audioURL,
		CreatedAt: botMsg.CreatedAt.Format(time.RFC3339),
	}, nil)
}

// cleanTextForTTS 清洗文本，移除 Markdown 和特殊字符，避免 edge-tts NoAudioReceived
func cleanTextForTTS(text string) string {
	// 移除代码块
	for strings.Contains(text, "```") {
		start := strings.Index(text, "```")
		end := strings.Index(text[start+3:], "```")
		if end == -1 {
			text = text[:start]
			break
		}
		text = text[:start] + " 代码块 " + text[start+3+end+3:]
	}
	// 移除行内代码
	for strings.Contains(text, "`") {
		s := strings.Index(text, "`")
		e := strings.Index(text[s+1:], "`")
		if e == -1 {
			text = strings.ReplaceAll(text, "`", "")
			break
		}
		text = text[:s] + text[s+1:s+1+e] + text[s+1+e+1:]
	}
	// 移除 URL
	for _, prefix := range []string{"https://", "http://"} {
		for strings.Contains(text, prefix) {
			s := strings.Index(text, prefix)
			e := s
			for e < len(text) && text[e] != ' ' && text[e] != '\n' && text[e] != ')' {
				e++
			}
			text = text[:s] + "链接" + text[e:]
		}
	}
	// 移除 Markdown 格式符
	for _, ch := range []string{"**", "__", "~~", "##", "#", ">", "- ", "* "} {
		text = strings.ReplaceAll(text, ch, "")
	}
	// 限制长度（edge-tts 超长会失败）
	runes := []rune(text)
	if len(runes) > 300 {
		text = string(runes[:300]) + "……"
	}
	return strings.TrimSpace(text)
}

// callEdgeTTS 通过 TTS HTTP 服务合成语音，返回可访问的 URL
func (h *ChatHandler) callEdgeTTS(text, voice, engine string) (string, error) {
	ttsText := cleanTextForTTS(text)
	if ttsText == "" {
		return "", fmt.Errorf("empty text after clean")
	}

	if h.ttsServiceURL == "" {
		return "", fmt.Errorf("TTS service not configured")
	}

	reqBody, _ := json.Marshal(map[string]string{
		"text":   ttsText,
		"voice":  voice,
		"engine": engine,
	})

	resp, err := http.Post(h.ttsServiceURL+"/tts", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("TTS service unavailable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("TTS service error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("TTS response decode error: %v", err)
	}
	return result.URL, nil
}

// callMiniMax 调用 MiniMax API（兼容 Anthropic 格式）
func (h *ChatHandler) callMiniMax(systemPrompt string, history []botMessage) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var msgs []message
	msgs = append(msgs, message{Role: "system", Content: systemPrompt})
	for _, m := range history {
		msgs = append(msgs, message{Role: m.Role, Content: m.Content})
	}

	model := h.minimax.Model
	if model == "" {
		model = "abab6.5s-chat"
	}
	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 1024,
		"messages":   msgs,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	apiURL := "https://api.minimaxi.com/anthropic/v1/messages"
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.minimax.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("MiniMax API error: %s", result.Error.Message)
	}
	for _, c := range result.Content {
		if c.Type == "text" {
			return c.Text, nil
		}
	}
	return "", fmt.Errorf("empty response from MiniMax")
}

// AdminDeleteRoom 管理员删除房间
func (h *ChatHandler) AdminDeleteRoom(c *gin.Context) {
	roomID := c.Param("id")
	adminPassword := c.Query("admin_password")

	if adminPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要管理员密码", "code": 400})
		return
	}

	if h.adminPassword == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员功能未启用", "code": 403})
		return
	}

	if adminPassword != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	// 检查房间是否存在
	if !h.db.RoomExists(roomID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "房间不存在", "code": 404})
		return
	}

	// 从内存中移除房间
	h.mu.Lock()
	if room, exists := h.rooms[roomID]; exists {
		// 关闭所有连接
		room.mu.Lock()
		for client := range room.clients {
			close(client.send)
			client.conn.Close()
		}
		room.mu.Unlock()
		delete(h.rooms, roomID)
	}
	h.mu.Unlock()

	// 从数据库中删除房间
	if err := h.db.DeleteRoom(roomID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除房间失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "房间删除成功"})
}
