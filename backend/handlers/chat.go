package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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
		send:     make(chan []byte, 256),
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
	data, _ := json.Marshal(msg)
	room.mu.RLock()
	defer room.mu.RUnlock()

	for client := range room.clients {
		if client != exclude {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(room.clients, client)
			}
		}
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
