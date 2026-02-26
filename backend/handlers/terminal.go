package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"devtools/models"
	"devtools/utils"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var sshUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// SSH Session 管理器
type SSHSessionManager struct {
	sessions map[string]*ActiveSSHSession
	mu       sync.RWMutex
}

type ActiveSSHSession struct {
	ID          string
	Client      *ssh.Client
	Session     *ssh.Session
	Stdin       io.Writer
	Stdout      io.Reader
	Stderr      io.Reader
	Width       int
	Height      int
	mu          sync.Mutex
	OutputBuffer []byte
}

var sshManager = &SSHSessionManager{
	sessions: make(map[string]*ActiveSSHSession),
}

type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
}

func (m *SSHSessionManager) Create(id string, config *SSHConfig, width, height int) (*ActiveSSHSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已有活跃会话
	if existing, ok := m.sessions[id]; ok {
		if existing.Client != nil {
			return existing, nil
		}
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var authMethods []ssh.AuthMethod
	if config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(config.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("私钥解析失败: %v", err)
		}
		authMethods = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else {
		authMethods = []ssh.AuthMethod{ssh.Password(config.Password)}
	}

	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("创建 SSH 会话失败: %v", err)
	}

	if err := session.RequestPty("xterm-256color", width, height, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("请求终端失败: %v", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("获取 stdin 失败: %v", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("获取 stdout 失败: %v", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("获取 stderr 失败: %v", err)
	}

	// 启动 shell
	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("启动 shell 失败: %v", err)
	}

	log.Printf("SSH shell 已启动: %s@%s", config.Username, addr)

	// 启动输出读取协程
	go m.readOutput(id, stdout)

	activeSession := &ActiveSSHSession{
		ID:           id,
		Client:       client,
		Session:      session,
		Stdin:        stdin,
		Stdout:       stdout,
		Stderr:       stderr,
		Width:        width,
		Height:       height,
		OutputBuffer: make([]byte, 0, 8192),
	}

	m.sessions[id] = activeSession

	return activeSession, nil
}

func (m *SSHSessionManager) readOutput(id string, stdout io.Reader) {
	buf := make([]byte, 8192)
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			log.Printf("SSH stdout 读取结束: %v", err)
			return
		}

		if n > 0 {
			m.mu.RLock()
			if s, exists := m.sessions[id]; exists {
				s.mu.Lock()
				s.OutputBuffer = append(s.OutputBuffer, buf[:n]...)
				s.mu.Unlock()
			}
			m.mu.RUnlock()
		}
	}
}

func (m *SSHSessionManager) Get(id string) (*ActiveSSHSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *SSHSessionManager) Write(id string, data []byte) error {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok || session.Stdin == nil {
		return fmt.Errorf("session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	_, err := session.Stdin.Write(data)
	return err
}

func (m *SSHSessionManager) Close(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		if session.Session != nil {
			session.Session.Close()
		}
		if session.Client != nil {
			session.Client.Close()
		}
		delete(m.sessions, id)
	}
}

func (m *SSHSessionManager) Resize(id string, width, height int) error {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found")
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	session.Width = width
	session.Height = height
	session.Session.WindowChange(width, height)
	return nil
}

func (m *SSHSessionManager) GetOutput(id string) []byte {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if len(session.OutputBuffer) == 0 {
		return nil
	}

	output := make([]byte, len(session.OutputBuffer))
	copy(output, session.OutputBuffer)
	session.OutputBuffer = session.OutputBuffer[:0]

	return output
}

type SSHHandler struct {
	db             *models.DB
	defaultExpDays int
	adminPassword  string
}

func NewSSHHandler(db *models.DB, defaultExpDays int, adminPassword string) *SSHHandler {
	if defaultExpDays <= 0 {
		defaultExpDays = 365
	}
	return &SSHHandler{
		db:             db,
		defaultExpDays: defaultExpDays,
		adminPassword:  adminPassword,
	}
}

func sshConfigIndex(host string, port int, username string) string {
	data := fmt.Sprintf("%s:%d:%s", host, port, username)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

type CreateSSHRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port"`
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ExpiresIn   int    `json:"expires_in"`
	KeepSession bool   `json:"keep_session"`
}

type CreateSSHResponse struct {
	ID         string     `json:"id"`
	CreatorKey string     `json:"creator_key"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func (h *SSHHandler) Create(c *gin.Context) {
	var req CreateSSHRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if req.Host == "" || req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入服务器地址和用户名", "code": 400})
		return
	}

	if req.Password == "" && req.PrivateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码或私钥", "code": 400})
		return
	}

	if req.Name == "" {
		req.Name = "SSH 终端"
	}

	if req.Port <= 0 {
		req.Port = 22
	}

	if req.Width <= 0 {
		req.Width = 80
	}
	if req.Height <= 0 {
		req.Height = 24
	}

	ip := c.ClientIP()

	// IP 限流
	count, err := h.db.CountSSHSessionsByIP(ip, time.Now().Add(-time.Hour))
	if err == nil && count >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	creatorKey := generateKey()
	hashedCreatorKey, _ := utils.HashPassword(creatorKey)

	configIndex := sshConfigIndex(req.Host, req.Port, req.Username)
	userToken := c.Query("user_token")

	expDays := req.ExpiresIn
	if expDays <= 0 {
		expDays = h.defaultExpDays
	}
	exp := time.Now().Add(time.Duration(expDays) * 24 * time.Hour)

	session := &models.SSHSession{
		Name:        req.Name,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		ConfigIndex: configIndex,
		CreatorKey:  hashedCreatorKey,
		UserToken:   userToken,
		Width:       req.Width,
		Height:      req.Height,
		KeepSession: req.KeepSession,
		ExpiresAt:   &exp,
		CreatorIP:   ip,
	}

	if err := h.db.CreateSSHSession(session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, CreateSSHResponse{
		ID:         session.ID,
		CreatorKey: creatorKey,
		ExpiresAt:  session.ExpiresAt,
	})
}

// List 用户所有会话
func (h *SSHHandler) List(c *gin.Context) {
	userToken := c.Query("user_token")
	if userToken != "" {
		sessions, err := h.db.GetSSHSessionsByUserToken(userToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败", "code": 500})
			return
		}

		var result []gin.H
		for _, session := range sessions {
			active, _ := sshManager.Get(session.ID)
			connected := active != nil && active.Client != nil

			result = append(result, gin.H{
				"id":         session.ID,
				"name":       session.Name,
				"host":       session.Host,
				"port":       session.Port,
				"username":   session.Username,
				"connected":  connected,
				"expires_at": session.ExpiresAt,
				"created_at": session.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"sessions": result,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_token", "code": 400})
}

// 获取会话详情
func (h *SSHHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		h.db.DeleteSSHSession(id)
		sshManager.Close(id)
		c.JSON(http.StatusGone, gin.H{"error": "该 SSH 配置已过期", "code": 410})
		return
	}

	if session.UserToken != "" && userToken != session.UserToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权访问", "code": 401})
		return
	}

	active, _ := sshManager.Get(id)
	connected := active != nil && active.Client != nil

	c.JSON(http.StatusOK, gin.H{
		"id":         session.ID,
		"name":       session.Name,
		"host":       session.Host,
		"port":       session.Port,
		"username":   session.Username,
		"width":      session.Width,
		"height":     session.Height,
		"connected":  connected,
		"expires_at": session.ExpiresAt,
		"created_at": session.CreatedAt,
	})
}

// 发送输入
type InputRequest struct {
	Data string `json:"data"`
}

func (h *SSHHandler) Input(c *gin.Context) {
	id := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if session.UserToken != "" && userToken != session.UserToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权访问", "code": 401})
		return
	}

	var req InputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求", "code": 400})
		return
	}

	// 确保 SSH 会话存在
	activeSession, ok := sshManager.Get(id)
	if !ok || activeSession.Client == nil {
		config := &SSHConfig{
			Host:       session.Host,
			Port:       session.Port,
			Username:   session.Username,
			Password:   session.Password,
			PrivateKey: session.PrivateKey,
		}
		width, height := session.Width, session.Height
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}

		activeSession, err = sshManager.Create(id, config, width, height)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("连接失败: %v", err), "code": 500})
			return
		}
	}

	// 发送输入
	if err := sshManager.Write(id, []byte(req.Data)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("输入失败: %v", err), "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// 长轮询获取输出
func (h *SSHHandler) Output(c *gin.Context) {
	id := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if session.UserToken != "" && userToken != session.UserToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权访问", "code": 401})
		return
	}

	// 检查 SSH 会话是否存在
	activeSession, ok := sshManager.Get(id)
	if !ok || activeSession.Client == nil {
		c.JSON(http.StatusOK, gin.H{
			"output":    "",
			"connected": false,
		})
		return
	}

	// 长轮询：最多等待 3 秒
	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			output := sshManager.GetOutput(id)
			if len(output) > 0 {
				c.JSON(http.StatusOK, gin.H{
					"output":    output,
					"connected": true,
				})
				return
			}
		case <-timeout:
			c.JSON(http.StatusOK, gin.H{
				"output":    "",
				"connected": true,
			})
			return
		}
	}
}

// 调整终端大小
type ResizeRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (h *SSHHandler) Resize(c *gin.Context) {
	id := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if session.UserToken != "" && userToken != session.UserToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权访问", "code": 401})
		return
	}

	var req ResizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求", "code": 400})
		return
	}

	if req.Width > 0 && req.Height > 0 {
		h.db.UpdateSSHSessionSize(id, req.Width, req.Height)
		sshManager.Resize(id, req.Width, req.Height)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// 断开连接
func (h *SSHHandler) Disconnect(c *gin.Context) {
	id := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if session.UserToken != "" && userToken != session.UserToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权访问", "code": 401})
		return
	}

	sshManager.Close(id)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminList 管理员查看所有会话
func (h *SSHHandler) AdminList(c *gin.Context) {
	adminPassword := c.Query("admin_password")

	if adminPassword != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	sessions, err := h.db.GetAllSSHSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败", "code": 500})
		return
	}

	var result []gin.H
	for _, session := range sessions {
		active, _ := sshManager.Get(session.ID)
		connected := active != nil && active.Client != nil

		result = append(result, gin.H{
			"id":         session.ID,
			"name":       session.Name,
			"host":       session.Host,
			"port":       session.Port,
			"username":   session.Username,
			"connected":  connected,
			"expires_at": session.ExpiresAt,
			"created_at": session.CreatedAt,
			"creator_ip": session.CreatorIP,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": result,
	})
}

// AdminDelete 管理员删除会话
func (h *SSHHandler) AdminDelete(c *gin.Context) {
	adminPassword := c.Query("admin_password")
	id := c.Param("id")

	if adminPassword != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误", "code": 401})
		return
	}

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	sshManager.Close(id)
	h.db.DeleteSSHSession(id)

	c.JSON(http.StatusOK, gin.H{
		"id":   session.ID,
		"name": session.Name,
		"host": session.Host,
	})
}

// Delete 删除会话
func (h *SSHHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, session.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	sshManager.Close(id)
	h.db.DeleteSSHSession(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// WebSocket 处理（保留兼容）
func (h *SSHHandler) HandleWebSocket(c *gin.Context) {
	id := c.Param("id")
	userToken := c.Query("user_token")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数", "code": 400})
		return
	}

	session, err := h.db.GetSSHSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该 SSH 配置", "code": 404})
		return
	}

	if session.UserToken != "" && userToken != session.UserToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权访问", "code": 401})
		return
	}

	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		h.db.DeleteSSHSession(id)
		sshManager.Close(id)
		c.JSON(http.StatusGone, gin.H{"error": "该 SSH 配置已过期", "code": 410})
		return
	}

	config := &SSHConfig{
		Host:       session.Host,
		Port:       session.Port,
		Username:   session.Username,
		Password:   session.Password,
		PrivateKey: session.PrivateKey,
	}

	conn, err := sshUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	activeSession, sessionExists := sshManager.Get(id)

	if !sessionExists || activeSession.Client == nil {
		log.Printf("HandleWebSocket: 开始创建 SSH 会话: %s@%s:%d", session.Username, session.Host, session.Port)
		activeSession, err = sshManager.Create(id, config, session.Width, session.Height)
		if err != nil {
			log.Printf("SSH 会话创建失败: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n连接失败: %v\r\n", err)))
			return
		}
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Connected to %s@%s:%d\r\n", session.Username, session.Host, session.Port)))
		log.Printf("SSH 会话已建立: %s@%s:%d", session.Username, session.Host, session.Port)
	}

	// 处理输入
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var wsMsg map[string]interface{}
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				continue
			}

			msgType, _ := wsMsg["type"].(string)

			switch msgType {
			case "input":
				if data, ok := wsMsg["data"].(string); ok {
					if err := sshManager.Write(id, []byte(data)); err != nil {
						log.Printf("SSH Write 错误: %v", err)
					}
				}
			case "resize":
				if width, ok := wsMsg["width"].(float64); ok {
					if height, ok := wsMsg["height"].(float64); ok {
						sshManager.Resize(id, int(width), int(height))
					}
				}
			}
		}
	}()

	// 处理输出
	go func() {
		buf := make([]byte, 4096)
		for {
			s, ok := sshManager.Get(id)
			if !ok || s.Client == nil {
				conn.WriteMessage(websocket.TextMessage, []byte("\r\nSession closed\r\n"))
				return
			}

			s.mu.Lock()
			n, err := s.Stdout.Read(buf)
			s.mu.Unlock()

			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("\r\nSession ended\r\n"))
				return
			}

			if n > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}

	sshManager.Close(id)
}
