package handlers

import (
	"crypto/rand"
	"crypto/sha256"
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

	"devtools/models"

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

// 文件魔数签名
var fileMagicNumbers = map[string][]byte{
	"image/jpeg":      {0xFF, 0xD8, 0xFF},
	"image/png":       {0x89, 0x50, 0x4E, 0x47},
	"image/gif":       {0x47, 0x49, 0x46},
	"image/webp":      {0x52, 0x49, 0x46, 0x46}, // RIFF
	"video/mp4":       {0x00, 0x00, 0x00},        // ftyp
	"video/webm":      {0x1A, 0x45, 0xDF, 0xA3},
	"audio/mpeg":      {0xFF, 0xFB},              // MP3
	"audio/wav":       {0x52, 0x49, 0x46, 0x46},  // RIFF
	"audio/ogg":       {0x4F, 0x67, 0x67, 0x53},
	"audio/webm":      {0x1A, 0x45, 0xDF, 0xA3},
	"application/pdf": {0x25, 0x50, 0x44, 0x46},
	"application/zip": {0x50, 0x4B, 0x03, 0x04},
}

// 检测文件真实类型
func detectFileType(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif"
	}
	// WebP (RIFF....WEBP)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}
	// MP4/MOV (ftyp)
	if len(data) >= 8 && data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		return "video/mp4"
	}
	// WebM/MKV
	if data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3 {
		return "video/webm"
	}
	// MP3 (ID3 or sync)
	if (data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33) || // ID3
		(data[0] == 0xFF && (data[1]&0xE0) == 0xE0) { // sync
		return "audio/mpeg"
	}
	// WAV (RIFF....WAVE)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "audio/wav"
	}
	// OGG
	if data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		return "audio/ogg"
	}
	// PDF
	if data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
		return "application/pdf"
	}
	// ZIP
	if data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
		return "application/zip"
	}

	return ""
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type ChatHandler struct {
	db    *models.DB
	rooms map[string]*Room
	mu    sync.RWMutex
}

type Room struct {
	ID      string
	clients map[*Client]bool
	mu      sync.RWMutex
}

type Client struct {
	conn     *websocket.Conn
	room     *Room
	nickname string
	send     chan []byte
}

type WSMessage struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	ID        int64  `json:"id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	MsgType   string `json:"msg_type,omitempty"`
}

func NewChatHandler(db *models.DB) *ChatHandler {
	return &ChatHandler{
		db:    db,
		rooms: make(map[string]*Room),
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
		hash := sha256.Sum256([]byte(req.Password))
		room.Password = hex.EncodeToString(hash[:])
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
		hash := sha256.Sum256([]byte(req.Password))
		if hex.EncodeToString(hash[:]) != room.Password {
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
			h.db.CreateMessage(chatMsg)

			// 更新房间活跃时间
			h.db.UpdateRoomActivity(roomID)

			// 广播消息
			broadcastMsg := WSMessage{
				Type:      "message",
				ID:        chatMsg.ID,
				Nickname:  client.nickname,
				Content:   msg.Content,
				MsgType:   chatMsg.MsgType,
				CreatedAt: chatMsg.CreatedAt.Format(time.RFC3339),
			}
			h.broadcast(client.room, broadcastMsg, nil)
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

	// 生成随机文件名
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
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

// getExtFromMimeType 根据 MIME 类型获取扩展名
func getExtFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4", "video/quicktime":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	default:
		return ""
	}
}

// UploadImage 保持向后兼容
func (h *ChatHandler) UploadImage(c *gin.Context) {
	h.UploadFile(c)
}
