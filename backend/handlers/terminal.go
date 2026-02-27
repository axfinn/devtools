package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// WebSocket upgrader
var sshUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
}

// ====================== SSH Session Manager ======================

// ActiveSSHSession 活跃的 SSH 会话
type ActiveSSHSession struct {
	ID          string
	Client      *ssh.Client
	Session     *ssh.Session
	Stdin       io.WriteCloser
	Stdout      io.Reader

	// 通道管理
	InputChan   chan []byte
	OutputChan  chan []byte
	ControlChan chan ControlMessage

	// 状态
	Status      string
	LastActive  time.Time
	Width       int
	Height      int

	// 上下文和取消
	ctx         context.Context
	cancel      context.CancelFunc

	// 同步
	mu          sync.RWMutex
}

// ControlMessage 控制消息
type ControlMessage struct {
	Type  string // resize, close, ping
	Data  interface{}
}

// ResizeData 调整大小数据
type ResizeData struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SSHSessionManager SSH 会话管理器
type SSHSessionManager struct {
	sessions   map[string]*ActiveSSHSession
	encryption *utils.EncryptionService
	db         *models.DB
	mu         sync.RWMutex
	config     *SSHHandlerConfig
}

// NewSSHSessionManager 创建 SSH 会话管理器
func NewSSHSessionManager(db *models.DB, encryption *utils.EncryptionService, config *SSHHandlerConfig) *SSHSessionManager {
	return &SSHSessionManager{
		sessions:   make(map[string]*ActiveSSHSession),
		encryption: encryption,
		db:         db,
		config:     config,
	}
}

// CreateSession 创建新的 SSH 会话
func (m *SSHSessionManager) CreateSession(sessionID string, config *SSHConnectConfig, width, height int) (*ActiveSSHSession, error) {
	// 构建 SSH 连接地址
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// 配置认证方法
	var authMethods []ssh.AuthMethod
	if config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(config.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("私钥解析失败: %v", err)
		}
		authMethods = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if config.Password != "" {
		authMethods = []ssh.AuthMethod{ssh.Password(config.Password)}
	} else {
		return nil, fmt.Errorf("必须提供密码或私钥")
	}

	// 配置主机密钥验证回调
	hostKeyCallback := m.createHostKeyCallback(config.Host, config.Port)

	// 创建 SSH 客户端配置
	clientConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	// 连接 SSH 服务器
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %v", err)
	}

	// 创建 SSH 会话
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("创建 SSH 会话失败: %v", err)
	}

	// 获取标准输入输出
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

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// 请求伪终端
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("请求 PTY 失败: %v", err)
	}

	// 启动 shell
	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("启动 shell 失败: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建活跃会话对象
	activeSession := &ActiveSSHSession{
		ID:          sessionID,
		Client:      client,
		Session:     session,
		Stdin:       stdin,
		Stdout:      stdout,
		InputChan:   make(chan []byte, 100),
		OutputChan:  make(chan []byte, 1000),
		ControlChan: make(chan ControlMessage, 10),
		Status:      "active",
		LastActive:  time.Now(),
		Width:       width,
		Height:      height,
		ctx:         ctx,
		cancel:      cancel,
	}

	// 保存到管理器
	m.mu.Lock()
	m.sessions[sessionID] = activeSession
	m.mu.Unlock()

	// 启动输入输出协程
	go m.handleInput(activeSession)
	go m.handleOutput(activeSession)
	go m.handleControl(activeSession)

	// 更新数据库状态
	if err := m.db.UpdateSSHSessionStatus(sessionID, "active"); err != nil {
		log.Printf("Failed to update session status: %v", err)
	}

	return activeSession, nil
}

// GetSession 获取会话
func (m *SSHSessionManager) GetSession(sessionID string) (*ActiveSSHSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[sessionID]
	return session, ok
}

// ResumeSession 恢复会话（如果SSH已断开则重新连接）
func (m *SSHSessionManager) ResumeSession(sessionID, userToken string) (*ActiveSSHSession, error) {
	// 检查是否已有活跃会话
	if session, ok := m.GetSession(sessionID); ok {
		// 更新最后活跃时间
		session.mu.Lock()
		session.LastActive = time.Now()
		session.mu.Unlock()

		m.db.UpdateSSHSessionLastActive(sessionID)
		return session, nil
	}

	// 从数据库获取会话信息
	dbSession, err := m.db.GetSSHSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %v", err)
	}

	// 验证用户令牌
	if dbSession.UserToken != userToken {
		return nil, fmt.Errorf("无权访问此会话")
	}

	// 解密密码和私钥
	password, err := m.encryption.DecryptIfNotEmpty(dbSession.PasswordEncrypted)
	if err != nil {
		return nil, fmt.Errorf("解密密码失败: %v", err)
	}

	privateKey, err := m.encryption.DecryptIfNotEmpty(dbSession.PrivateKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("解密私钥失败: %v", err)
	}

	// 重新建立 SSH 连接
	config := &SSHConnectConfig{
		Host:       dbSession.Host,
		Port:       dbSession.Port,
		Username:   dbSession.Username,
		Password:   password,
		PrivateKey: privateKey,
	}

	return m.CreateSession(sessionID, config, dbSession.Width, dbSession.Height)
}

// CloseSession 关闭会话
func (m *SSHSessionManager) CloseSession(sessionID string) error {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("会话不存在")
	}
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	// 取消上下文
	session.cancel()

	// 关闭 SSH 连接
	if session.Session != nil {
		session.Session.Close()
	}
	if session.Client != nil {
		session.Client.Close()
	}

	// 关闭通道
	close(session.InputChan)
	close(session.OutputChan)
	close(session.ControlChan)

	// 更新数据库状态
	if err := m.db.UpdateSSHSessionStatus(sessionID, "idle"); err != nil {
		log.Printf("Failed to update session status: %v", err)
	}

	return nil
}

// ====================== Input/Output/Control Handlers ======================

// handleInput 处理输入（InputChan → SSH Stdin）
func (m *SSHSessionManager) handleInput(session *ActiveSSHSession) {
	for {
		select {
		case <-session.ctx.Done():
			return
		case data := <-session.InputChan:
			if data == nil {
				return
			}

			// 写入 SSH stdin
			if _, err := session.Stdin.Write(data); err != nil {
				log.Printf("Failed to write to stdin: %v", err)
				return
			}

			// 保存到历史
			if len(data) > 0 {
				m.db.SaveSSHHistory(session.ID, "input", string(data))
			}

			// 更新最后活跃时间
			session.mu.Lock()
			session.LastActive = time.Now()
			session.mu.Unlock()
		}
	}
}

// handleOutput 处理输出（SSH Stdout → OutputChan）
func (m *SSHSessionManager) handleOutput(session *ActiveSSHSession) {
	buffer := make([]byte, 8192)

	for {
		select {
		case <-session.ctx.Done():
			return
		default:
			n, err := session.Stdout.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Failed to read from stdout: %v", err)
				}
				return
			}

			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				// 发送到输出通道
				select {
				case session.OutputChan <- data:
				case <-session.ctx.Done():
					return
				case <-time.After(5 * time.Second):
					log.Printf("Output channel blocked, dropping data")
				}

				// 保存到历史（限制大小）
				if n < 10000 {
					m.db.SaveSSHHistory(session.ID, "output", string(data))
				}

				// 更新最后活跃时间
				session.mu.Lock()
				session.LastActive = time.Now()
				session.mu.Unlock()
			}
		}
	}
}

// handleControl 处理控制消息
func (m *SSHSessionManager) handleControl(session *ActiveSSHSession) {
	for {
		select {
		case <-session.ctx.Done():
			return
		case msg := <-session.ControlChan:
			switch msg.Type {
			case "resize":
				if resizeData, ok := msg.Data.(ResizeData); ok {
					if err := session.Session.WindowChange(resizeData.Height, resizeData.Width); err != nil {
						log.Printf("Failed to resize window: %v", err)
					} else {
						session.mu.Lock()
						session.Width = resizeData.Width
						session.Height = resizeData.Height
						session.mu.Unlock()

						m.db.UpdateSSHSessionSize(session.ID, resizeData.Width, resizeData.Height)
					}
				}
			case "close":
				m.CloseSession(session.ID)
				return
			}
		}
	}
}

// createHostKeyCallback 创建主机密钥验证回调
func (m *SSHSessionManager) createHostKeyCallback(host string, port int) ssh.HostKeyCallback {
	if m.config != nil && !m.config.HostKeyVerification {
		// 如果配置禁用了验证，使用不安全的回调
		log.Printf("WARNING: Host key verification disabled for %s:%d", host, port)
		return ssh.InsecureIgnoreHostKey()
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// 计算指纹
		fingerprint := ssh.FingerprintSHA256(key)
		keyType := key.Type()

		// 从数据库查询已知主机密钥
		knownKey, err := m.db.GetSSHHostKey(host, port)
		if err != nil {
			log.Printf("Failed to get host key from db: %v", err)
		}

		if knownKey == nil {
			// 首次连接，保存主机密钥
			log.Printf("First time connecting to %s:%d, saving host key %s", host, port, fingerprint)

			newKey := &models.SSHHostKey{
				Host:        host,
				Port:        port,
				KeyType:     keyType,
				Fingerprint: fingerprint,
				PublicKey:   base64.StdEncoding.EncodeToString(key.Marshal()),
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
			}

			if err := m.db.SaveSSHHostKey(newKey); err != nil {
				log.Printf("Failed to save host key: %v", err)
			}

			return nil
		}

		// 验证指纹是否匹配
		if knownKey.Fingerprint != fingerprint {
			return fmt.Errorf("主机密钥不匹配！可能遭受中间人攻击\n期望: %s\n实际: %s",
				knownKey.Fingerprint, fingerprint)
		}

		// 更新最后见时间
		knownKey.LastSeen = time.Now()
		m.db.SaveSSHHostKey(knownKey)

		return nil
	}
}

// ====================== Handler Configuration ======================

// SSHHandlerConfig SSH 处理器配置
type SSHHandlerConfig struct {
	AdminPassword       string
	HostKeyVerification bool
	MaxSessionsPerUser  int
	SessionIdleTimeout  time.Duration
}

// SSHHandler SSH HTTP 处理器
type SSHHandler struct {
	db         *models.DB
	manager    *SSHSessionManager
	config     *SSHHandlerConfig
	encryption *utils.EncryptionService
}

// NewSSHHandler 创建 SSH 处理器
func NewSSHHandler(db *models.DB, encryption *utils.EncryptionService, config *SSHHandlerConfig) *SSHHandler {
	if config == nil {
		config = &SSHHandlerConfig{
			HostKeyVerification: true,
			MaxSessionsPerUser:  10,
			SessionIdleTimeout:  5 * time.Minute,
		}
	}

	return &SSHHandler{
		db:         db,
		manager:    NewSSHSessionManager(db, encryption, config),
		config:     config,
		encryption: encryption,
	}
}

// ====================== HTTP API Handlers ======================

// SSHConnectConfig SSH 连接配置
type SSHConnectConfig struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port"`
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	UserToken  string `json:"user_token" binding:"required"`
	KeepAlive  bool   `json:"keep_alive"`
	ExpiresIn  int    `json:"expires_in"` // 过期天数
}

// Create 创建新的 SSH 会话
func (h *SSHHandler) Create(c *gin.Context) {
	var config SSHConnectConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 默认值
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Width == 0 {
		config.Width = 80
	}
	if config.Height == 0 {
		config.Height = 24
	}
	if config.Name == "" {
		config.Name = fmt.Sprintf("%s@%s", config.Username, config.Host)
	}

	// 如果没有提供密码或私钥，尝试从历史会话中查找
	if config.Password == "" && config.PrivateKey == "" {
		log.Printf("[SSH] 尝试查找历史密码: userToken=%s, host=%s, port=%d, username=%s",
			config.UserToken, config.Host, config.Port, config.Username)

		// 查找该用户最近一次连接这个主机的会话
		historySession, err := h.db.GetLatestSSHSessionByHost(config.UserToken, config.Host, config.Port, config.Username)
		if err != nil {
			log.Printf("[SSH] 查找历史会话失败: %v", err)
		}

		if historySession != nil {
			log.Printf("[SSH] 找到历史会话: id=%s, hasPassword=%v, hasPrivateKey=%v",
				historySession.ID, historySession.PasswordEncrypted != "", historySession.PrivateKeyEncrypted != "")

			// 尝试解密历史密码
			if historySession.PasswordEncrypted != "" {
				password, err := h.encryption.DecryptIfNotEmpty(historySession.PasswordEncrypted)
				if err != nil {
					log.Printf("[SSH] 解密密码失败: %v", err)
				} else {
					log.Printf("[SSH] 密码解密成功, length=%d", len(password))
					config.Password = password
				}
			}
			// 尝试解密历史私钥
			if config.Password == "" && historySession.PrivateKeyEncrypted != "" {
				privateKey, err := h.encryption.DecryptIfNotEmpty(historySession.PrivateKeyEncrypted)
				if err != nil {
					log.Printf("[SSH] 解密私钥失败: %v", err)
				} else {
					log.Printf("[SSH] 私钥解密成功, length=%d", len(privateKey))
					config.PrivateKey = privateKey
				}
			}
		} else {
			log.Printf("[SSH] 未找到历史会话")
		}

		// 如果还是没有找到密码或私钥，返回需要密码的错误
		if config.Password == "" && config.PrivateKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "need_password", "message": "请输入密码或私钥"})
			return
		}
	}

	// IP 速率限制
	clientIP := c.ClientIP()
	count, _ := h.db.CountSSHSessionsByIP(clientIP, time.Now().Add(-1*time.Hour))
	if count >= 20 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建会话过于频繁，请稍后再试"})
		return
	}

	// 检查用户会话数量限制
	userSessions, _ := h.db.GetSSHSessionsByUserToken(config.UserToken)
	if len(userSessions) >= h.config.MaxSessionsPerUser {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("会话数量已达上限 (%d)", h.config.MaxSessionsPerUser)})
		return
	}

	// 加密密码和私钥
	passwordEncrypted, err := h.encryption.EncryptIfNotEmpty(config.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密密码失败"})
		return
	}

	privateKeyEncrypted, err := h.encryption.EncryptIfNotEmpty(config.PrivateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密私钥失败"})
		return
	}

	// 生成创建者密钥
	creatorKey := generateRandomKey()

	// 计算过期时间
	var expiresAt *time.Time
	if config.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(config.ExpiresIn) * 24 * time.Hour)
		expiresAt = &expiry
	}

	// 创建数据库会话记录
	dbSession := &models.SSHSession{
		Name:                config.Name,
		Host:                config.Host,
		Port:                config.Port,
		Username:            config.Username,
		PasswordEncrypted:   passwordEncrypted,
		PrivateKeyEncrypted: privateKeyEncrypted,
		UserToken:           config.UserToken,
		CreatorKey:          creatorKey,
		CreatorIP:           clientIP,
		Width:               config.Width,
		Height:              config.Height,
		Status:              "idle",
		KeepAlive:           config.KeepAlive,
		ExpiresAt:           expiresAt,
	}

	if err := h.db.CreateSSHSession(dbSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败: " + err.Error()})
		return
	}

	// 返回会话信息（不包含敏感信息）
	c.JSON(http.StatusOK, gin.H{
		"id":          dbSession.ID,
		"name":        dbSession.Name,
		"host":        dbSession.Host,
		"port":        dbSession.Port,
		"username":    dbSession.Username,
		"status":      dbSession.Status,
		"creator_key": creatorKey,
		"created_at":  dbSession.CreatedAt,
		"expires_at":  dbSession.ExpiresAt,
	})
}

// generateRandomKey 生成随机密钥
func generateRandomKey() string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), randomInt())))
	return hex.EncodeToString(hash.Sum(nil))[:32]
}

// randomInt 生成随机整数（简单实现）
func randomInt() int {
	return int(time.Now().UnixNano() % 1000000)
}

// List 获取用户的会话列表
func (h *SSHHandler) List(c *gin.Context) {
	userToken := c.Query("user_token")
	if userToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 user_token"})
		return
	}

	sessions, err := h.db.GetSSHSessionsByUserToken(userToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// Get 获取单个会话信息
func (h *SSHHandler) Get(c *gin.Context) {
	sessionID := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 验证权限
	if session.UserToken != userToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
		return
	}

	// 检查是否有活跃连接
	_, isActive := h.manager.GetSession(sessionID)
	if isActive {
		session.Status = "active"
	}

	c.JSON(http.StatusOK, session)
}

// GetCredentials 获取会话凭证（解密后的密码或私钥）
func (h *SSHHandler) GetCredentials(c *gin.Context) {
	sessionID := c.Param("id")
	userToken := c.Query("user_token")

	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 验证权限
	if session.UserToken != userToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
		return
	}

	// 解密密码和私钥
	hasPassword := session.PasswordEncrypted != ""
	hasPrivateKey := session.PrivateKeyEncrypted != ""

	response := gin.H{
		"id":           session.ID,
		"has_password":   hasPassword,
		"has_private_key": hasPrivateKey,
	}

	// 尝试解密密码
	if hasPassword {
		password, err := h.encryption.DecryptIfNotEmpty(session.PasswordEncrypted)
		if err == nil {
			response["password"] = password
		}
	}

	// 尝试解密私钥
	if hasPrivateKey {
		privateKey, err := h.encryption.DecryptIfNotEmpty(session.PrivateKeyEncrypted)
		if err == nil {
			response["private_key"] = privateKey
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetByCreator 通过创建者密钥获取会话
func (h *SSHHandler) GetByCreator(c *gin.Context) {
	sessionID := c.Param("id")
	creatorKey := c.Query("creator_key")

	if creatorKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 creator_key"})
		return
	}

	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 验证创建者密钥
	if session.CreatorKey != creatorKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
		return
	}

	// 检查是否有活跃连接
	_, isActive := h.manager.GetSession(sessionID)
	if isActive {
		session.Status = "active"
	}

	c.JSON(http.StatusOK, session)
}

// GetHistory 获取会话历史记录
func (h *SSHHandler) GetHistory(c *gin.Context) {
	sessionID := c.Param("id")
	userToken := c.Query("user_token")

	// 验证权限
	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if session.UserToken != userToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
		return
	}

	// 分页参数
	limit := 100
	offset := 0

	history, err := h.db.GetSSHHistory(sessionID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取历史记录失败"})
		return
	}

	count, _ := h.db.GetSSHHistoryCount(sessionID)

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   count,
		"limit":   limit,
		"offset":  offset,
	})
}

// Update 更新会话（重命名、调整大小等）
func (h *SSHHandler) Update(c *gin.Context) {
	sessionID := c.Param("id")

	var req struct {
		Action     string `json:"action"`
		Name       string `json:"name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		ExpiresIn  int    `json:"expires_in"`
		CreatorKey string `json:"creator_key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证创建者密钥
	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if session.CreatorKey != req.CreatorKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此会话"})
		return
	}

	// 根据 action 执行不同操作
	switch req.Action {
	case "rename":
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "会话名称不能为空"})
			return
		}
		if err := h.db.UpdateSSHSessionName(sessionID, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}

	case "resize":
		if req.Width <= 0 || req.Height <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的终端大小"})
			return
		}
		if err := h.db.UpdateSSHSessionSize(sessionID, req.Width, req.Height); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}

		// 如果会话活跃，同时调整SSH终端大小
		if activeSession, ok := h.manager.GetSession(sessionID); ok {
			activeSession.ControlChan <- ControlMessage{
				Type: "resize",
				Data: ResizeData{Width: req.Width, Height: req.Height},
			}
		}

	case "extend":
		if req.ExpiresIn <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的过期时间"})
			return
		}
		expiry := time.Now().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour)
		if err := h.db.ExtendSSHSession(sessionID, &expiry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "未知的操作: " + req.Action})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// Resume 恢复会话（重新连接SSH）
func (h *SSHHandler) Resume(c *gin.Context) {
	sessionID := c.Param("id")

	var req struct {
		UserToken string `json:"user_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 尝试恢复会话
	activeSession, err := h.manager.ResumeSession(sessionID, req.UserToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复会话失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "会话已恢复",
		"status":  activeSession.Status,
	})
}

// Disconnect 断开 SSH 连接但保留会话记录
func (h *SSHHandler) Disconnect(c *gin.Context) {
	sessionID := c.Param("id")
	userToken := c.Query("user_token")

	// 验证权限
	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if session.UserToken != userToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此会话"})
		return
	}

	// 关闭活跃会话
	if err := h.manager.CloseSession(sessionID); err != nil {
		// 会话可能已经关闭，不算错误
		log.Printf("Close session warning: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "已断开连接"})
}

// Delete 删除会话
func (h *SSHHandler) Delete(c *gin.Context) {
	sessionID := c.Param("id")
	creatorKey := c.Query("creator_key")
	userToken := c.Query("user_token")

	// 验证权限：支持 creator_key 或 user_token
	session, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 优先使用 creator_key，其次使用 user_token
	hasPermission := false
	if creatorKey != "" && session.CreatorKey == creatorKey {
		hasPermission = true
	} else if userToken != "" && session.UserToken == userToken {
		hasPermission = true
	}

	if !hasPermission {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此会话"})
		return
	}

	// 先关闭活跃会话
	h.manager.CloseSession(sessionID)

	// 删除数据库记录（会自动删除历史记录）
	if err := h.db.DeleteSSHSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "会话已删除"})
}

// AdminList 管理员获取所有会话
func (h *SSHHandler) AdminList(c *gin.Context) {
	adminPassword := c.Query("admin_password")

	if adminPassword != h.config.AdminPassword {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员密码错误"})
		return
	}

	sessions, err := h.db.GetAllSSHSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// AdminDelete 管理员删除会话
func (h *SSHHandler) AdminDelete(c *gin.Context) {
	sessionID := c.Param("id")
	adminPassword := c.Query("admin_password")

	if adminPassword != h.config.AdminPassword {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员密码错误"})
		return
	}

	// 先关闭活跃会话
	h.manager.CloseSession(sessionID)

	// 删除数据库记录
	if err := h.db.DeleteSSHSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "会话已删除"})
}

// Login 用户登录（生成或验证用户令牌）
func (h *SSHHandler) Login(c *gin.Context) {
	// 简单实现：生成一个随机的用户令牌
	// 实际应用中可以集成真正的用户认证系统

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供用户名密码，直接生成匿名令牌
		token := generateRandomKey()
		c.JSON(http.StatusOK, gin.H{
			"user_token": token,
			"username":   "anonymous",
		})
		return
	}

	// TODO: 实现真正的用户认证
	// 现在只是生成一个基于用户名的令牌
	hash := sha256.New()
	hash.Write([]byte(req.Username + req.Password + time.Now().String()))
	token := hex.EncodeToString(hash.Sum(nil))

	c.JSON(http.StatusOK, gin.H{
		"user_token": token,
		"username":   req.Username,
	})
}

// ====================== WebSocket Handler ======================

// HandleWebSocket WebSocket 连接处理
func (h *SSHHandler) HandleWebSocket(c *gin.Context) {
	sessionID := c.Param("id")
	userToken := c.Query("user_token")

	// 验证权限
	dbSession, err := h.db.GetSSHSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if dbSession.UserToken != userToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
		return
	}

	// 升级到 WebSocket
	conn, err := sshUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 恢复或创建 SSH 会话
	activeSession, err := h.manager.ResumeSession(sessionID, userToken)
	if err != nil {
		// 检查是否是解密失败
		errMsg := err.Error()
		isDecryptError := strings.Contains(errMsg, "解密密码失败") ||
			strings.Contains(errMsg, "failed to decrypt") ||
			strings.Contains(errMsg, "message authentication failed")

		if isDecryptError {
			conn.WriteJSON(map[string]interface{}{
				"type":         "error",
				"message":      "无法连接 SSH: 密码解密失败，可能是服务器重启导致密钥变化",
				"decryptError": true,
				"sessionId":    sessionID,
			})
		} else {
			conn.WriteJSON(map[string]interface{}{
				"type":    "error",
				"message": "无法连接 SSH: " + errMsg,
			})
		}
		return
	}

	// 发送连接成功消息
	conn.WriteJSON(map[string]interface{}{
		"type":    "status",
		"status":  "connected",
		"message": "SSH 连接成功",
	})

	// 发送历史记录
	history, _ := h.db.GetSSHHistory(sessionID, 100, 0)
	if len(history) > 0 {
		var historyData bytes.Buffer
		for _, h := range history {
			if h.Type == "output" {
				historyData.WriteString(h.Content)
			}
		}
		conn.WriteJSON(map[string]interface{}{
			"type": "history",
			"data": historyData.String(),
		})
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动输出协程（OutputChan → WebSocket）
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-activeSession.OutputChan:
				if data == nil {
					return
				}

				err := conn.WriteJSON(map[string]interface{}{
					"type": "output",
					"data": string(data),
				})

				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}()

	// 启动心跳协程
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.WriteJSON(map[string]interface{}{
					"type": "ping",
				}); err != nil {
					return
				}
			}
		}
	}()

	// 主循环：处理来自 WebSocket 的消息
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "input":
			if data, ok := msg["data"].(string); ok {
				// 发送输入到 SSH
				select {
				case activeSession.InputChan <- []byte(data):
				case <-time.After(5 * time.Second):
					log.Printf("Input channel blocked")
				}
			}

		case "resize":
			if cols, ok := msg["cols"].(float64); ok {
				if rows, ok := msg["rows"].(float64); ok {
					activeSession.ControlChan <- ControlMessage{
						Type: "resize",
						Data: ResizeData{Width: int(cols), Height: int(rows)},
					}
				}
			}

		case "pong":
			// 心跳响应
			activeSession.mu.Lock()
			activeSession.LastActive = time.Now()
			activeSession.mu.Unlock()
		}
	}

	// WebSocket 关闭，但不立即关闭 SSH（支持重连）
	log.Printf("WebSocket closed for session %s", sessionID)
}

// ====================== Utility Functions ======================

// StartCleanupRoutine 启动定期清理协程
func (h *SSHHandler) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(h.config.SessionIdleTimeout)
		defer ticker.Stop()

		for range ticker.C {
			// 关闭长时间空闲的活跃会话
			h.manager.mu.RLock()
			for sessionID, session := range h.manager.sessions {
				session.mu.RLock()
				idleTime := time.Since(session.LastActive)
				session.mu.RUnlock()

				if idleTime > h.config.SessionIdleTimeout {
					log.Printf("Closing idle session %s (idle for %v)", sessionID, idleTime)
					go h.manager.CloseSession(sessionID)
				}
			}
			h.manager.mu.RUnlock()
		}
	}()
}
