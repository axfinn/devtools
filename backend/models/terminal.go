package models

import (
	"database/sql"
	"fmt"
	"time"
)

// SSHSession SSH 会话结构体
type SSHSession struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Host         string     `json:"host"`
	Port         int        `json:"port"`
	Username     string     `json:"username"`

	// 加密存储的敏感信息（不返回给前端）
	PasswordEncrypted    string `json:"-"` // AES加密的密码
	PrivateKeyEncrypted  string `json:"-"` // AES加密的私钥

	// 主机密钥指纹（用于验证）
	HostKeyFingerprint   string `json:"host_key_fingerprint,omitempty"`

	// 用户标识
	UserToken    string `json:"-"` // 用户令牌（隔离会话）
	CreatorKey   string `json:"-"` // 创建者密钥（管理权限）
	CreatorIP    string `json:"-"` // 创建者IP

	// 终端配置
	Width        int    `json:"width"`
	Height       int    `json:"height"`

	// 会话状态
	Status       string    `json:"status"` // active（连接中）、idle（已断开）、expired（已过期）
	KeepAlive    bool      `json:"keep_alive"` // 是否保持会话
	LastActiveAt time.Time `json:"last_active_at"` // 最后活跃时间

	// 时间信息
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SSHHistory SSH 历史记录
type SSHHistory struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"` // 'input' or 'output'
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SSHHostKey SSH 主机密钥
type SSHHostKey struct {
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	KeyType     string    `json:"key_type"`
	Fingerprint string    `json:"fingerprint"`
	PublicKey   string    `json:"public_key"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// InitSSH 初始化 SSH 相关表
func (db *DB) InitSSH() error {
	// 1. 创建 ssh_sessions 表
	createSessionsTable := `
	CREATE TABLE IF NOT EXISTS ssh_sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT 'SSH 终端',
		host TEXT NOT NULL,
		port INTEGER NOT NULL DEFAULT 22,
		username TEXT NOT NULL,
		password_encrypted TEXT,
		private_key_encrypted TEXT,
		host_key_fingerprint TEXT,

		user_token TEXT NOT NULL,
		creator_key TEXT NOT NULL,
		creator_ip TEXT,

		width INTEGER NOT NULL DEFAULT 80,
		height INTEGER NOT NULL DEFAULT 24,

		status TEXT NOT NULL DEFAULT 'idle',
		keep_alive INTEGER NOT NULL DEFAULT 1,
		last_active_at DATETIME DEFAULT CURRENT_TIMESTAMP,

		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// 2. 创建 ssh_history 表
	createHistoryTable := `
	CREATE TABLE IF NOT EXISTS ssh_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,

		FOREIGN KEY (session_id) REFERENCES ssh_sessions(id) ON DELETE CASCADE
	);
	`

	// 3. 创建 ssh_host_keys 表
	createHostKeysTable := `
	CREATE TABLE IF NOT EXISTS ssh_host_keys (
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		key_type TEXT NOT NULL,
		fingerprint TEXT NOT NULL,
		public_key TEXT,
		first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,

		PRIMARY KEY (host, port)
	);
	`

	// 执行创建表
	if _, err := db.conn.Exec(createSessionsTable); err != nil {
		return fmt.Errorf("failed to create ssh_sessions table: %w", err)
	}

	if _, err := db.conn.Exec(createHistoryTable); err != nil {
		return fmt.Errorf("failed to create ssh_history table: %w", err)
	}

	if _, err := db.conn.Exec(createHostKeysTable); err != nil {
		return fmt.Errorf("failed to create ssh_host_keys table: %w", err)
	}

	// 4. 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_ssh_user_token ON ssh_sessions(user_token)",
		"CREATE INDEX IF NOT EXISTS idx_ssh_status ON ssh_sessions(status)",
		"CREATE INDEX IF NOT EXISTS idx_ssh_expires_at ON ssh_sessions(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_ssh_last_active ON ssh_sessions(last_active_at)",
		"CREATE INDEX IF NOT EXISTS idx_ssh_creator_ip ON ssh_sessions(creator_ip)",
		"CREATE INDEX IF NOT EXISTS idx_history_session ON ssh_history(session_id, timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_history_timestamp ON ssh_history(timestamp)",
	}

	for _, idx := range indexes {
		if _, err := db.conn.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// ====================== SSH Session CRUD ======================

// CreateSSHSession 创建 SSH 会话
func (db *DB) CreateSSHSession(session *SSHSession) error {
	session.ID = generateID(8)
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	session.LastActiveAt = time.Now()

	if session.Status == "" {
		session.Status = "idle"
	}

	_, err := db.conn.Exec(`
		INSERT INTO ssh_sessions (
			id, name, host, port, username,
			password_encrypted, private_key_encrypted, host_key_fingerprint,
			user_token, creator_key, creator_ip,
			width, height,
			status, keep_alive, last_active_at,
			expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.ID, session.Name, session.Host, session.Port, session.Username,
		session.PasswordEncrypted, session.PrivateKeyEncrypted, session.HostKeyFingerprint,
		session.UserToken, session.CreatorKey, session.CreatorIP,
		session.Width, session.Height,
		session.Status, session.KeepAlive, session.LastActiveAt,
		session.ExpiresAt, session.CreatedAt, session.UpdatedAt,
	)

	return err
}

// GetSSHSession 获取 SSH 会话
func (db *DB) GetSSHSession(id string) (*SSHSession, error) {
	session := &SSHSession{}
	var expiresAt sql.NullTime
	var passwordEncrypted, privateKeyEncrypted, hostKeyFingerprint sql.NullString
	var keepAlive int

	err := db.conn.QueryRow(`
		SELECT id, name, host, port, username,
			password_encrypted, private_key_encrypted, host_key_fingerprint,
			user_token, creator_key, creator_ip,
			width, height,
			status, keep_alive, last_active_at,
			expires_at, created_at, updated_at
		FROM ssh_sessions WHERE id = ?
	`, id).Scan(
		&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
		&passwordEncrypted, &privateKeyEncrypted, &hostKeyFingerprint,
		&session.UserToken, &session.CreatorKey, &session.CreatorIP,
		&session.Width, &session.Height,
		&session.Status, &keepAlive, &session.LastActiveAt,
		&expiresAt, &session.CreatedAt, &session.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	session.PasswordEncrypted = passwordEncrypted.String
	session.PrivateKeyEncrypted = privateKeyEncrypted.String
	session.HostKeyFingerprint = hostKeyFingerprint.String
	session.KeepAlive = keepAlive != 0

	if expiresAt.Valid {
		session.ExpiresAt = &expiresAt.Time
	}

	return session, nil
}

// GetSSHSessionsByUserToken 获取用户的所有 SSH 会话
func (db *DB) GetSSHSessionsByUserToken(userToken string) ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username,
			width, height,
			status, keep_alive, last_active_at,
			expires_at, created_at, updated_at
		FROM ssh_sessions
		WHERE user_token = ?
		ORDER BY last_active_at DESC, created_at DESC
	`, userToken)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		var keepAlive int

		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height,
			&session.Status, &keepAlive, &session.LastActiveAt,
			&expiresAt, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		session.KeepAlive = keepAlive != 0
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetSSHSessionsByCreatorKey 通过创建者密钥获取会话列表
func (db *DB) GetSSHSessionsByCreatorKey(creatorKey string) ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username,
			width, height,
			status, keep_alive, last_active_at,
			expires_at, created_at, updated_at
		FROM ssh_sessions
		WHERE creator_key = ?
		ORDER BY last_active_at DESC, created_at DESC
	`, creatorKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		var keepAlive int

		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height,
			&session.Status, &keepAlive, &session.LastActiveAt,
			&expiresAt, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		session.KeepAlive = keepAlive != 0
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetAllSSHSessions 获取所有 SSH 会话（管理员）
func (db *DB) GetAllSSHSessions() ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username,
			width, height,
			status, keep_alive, last_active_at,
			expires_at, created_at, updated_at, creator_ip
		FROM ssh_sessions
		ORDER BY last_active_at DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		var keepAlive int

		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height,
			&session.Status, &keepAlive, &session.LastActiveAt,
			&expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP,
		)
		if err != nil {
			return nil, err
		}

		session.KeepAlive = keepAlive != 0
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateSSHSessionName 更新会话名称
func (db *DB) UpdateSSHSessionName(id, name string) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET name = ?, updated_at = ? WHERE id = ?",
		name, time.Now(), id)
	return err
}

// UpdateSSHSessionSize 更新终端大小
func (db *DB) UpdateSSHSessionSize(id string, width, height int) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET width = ?, height = ?, updated_at = ? WHERE id = ?",
		width, height, time.Now(), id)
	return err
}

// UpdateSSHSessionStatus 更新会话状态
func (db *DB) UpdateSSHSessionStatus(id, status string) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET status = ?, last_active_at = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), time.Now(), id)
	return err
}

// UpdateSSHSessionLastActive 更新最后活跃时间
func (db *DB) UpdateSSHSessionLastActive(id string) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET last_active_at = ?, updated_at = ? WHERE id = ?",
		time.Now(), time.Now(), id)
	return err
}

// ExtendSSHSession 延长会话过期时间
func (db *DB) ExtendSSHSession(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET expires_at = ?, updated_at = ? WHERE id = ?",
		expiresAt, time.Now(), id)
	return err
}

// DeleteSSHSession 删除 SSH 会话
func (db *DB) DeleteSSHSession(id string) error {
	// 由于外键约束，删除会话会自动删除关联的历史记录
	_, err := db.conn.Exec("DELETE FROM ssh_sessions WHERE id = ?", id)
	return err
}

// ====================== History ======================

// SaveSSHHistory 保存 SSH 历史记录
func (db *DB) SaveSSHHistory(sessionID, msgType, content string) error {
	_, err := db.conn.Exec(`
		INSERT INTO ssh_history (session_id, type, content, timestamp)
		VALUES (?, ?, ?, ?)
	`, sessionID, msgType, content, time.Now())
	return err
}

// GetSSHHistory 获取会话历史记录
func (db *DB) GetSSHHistory(sessionID string, limit, offset int) ([]*SSHHistory, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT id, session_id, type, content, timestamp
		FROM ssh_history
		WHERE session_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*SSHHistory
	for rows.Next() {
		h := &SSHHistory{}
		if err := rows.Scan(&h.ID, &h.SessionID, &h.Type, &h.Content, &h.Timestamp); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	// 反转顺序（最早的在前）
	for i := 0; i < len(history)/2; i++ {
		history[i], history[len(history)-1-i] = history[len(history)-1-i], history[i]
	}

	return history, nil
}

// GetSSHHistoryCount 获取历史记录数量
func (db *DB) GetSSHHistoryCount(sessionID string) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM ssh_history WHERE session_id = ?",
		sessionID,
	).Scan(&count)
	return count, err
}

// ====================== Host Keys ======================

// SaveSSHHostKey 保存或更新主机密钥
func (db *DB) SaveSSHHostKey(hostKey *SSHHostKey) error {
	_, err := db.conn.Exec(`
		INSERT INTO ssh_host_keys (host, port, key_type, fingerprint, public_key, first_seen, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(host, port) DO UPDATE SET
			last_seen = ?,
			fingerprint = ?,
			key_type = ?,
			public_key = ?
	`, hostKey.Host, hostKey.Port, hostKey.KeyType, hostKey.Fingerprint, hostKey.PublicKey,
		hostKey.FirstSeen, hostKey.LastSeen,
		hostKey.LastSeen, hostKey.Fingerprint, hostKey.KeyType, hostKey.PublicKey)
	return err
}

// GetSSHHostKey 获取主机密钥
func (db *DB) GetSSHHostKey(host string, port int) (*SSHHostKey, error) {
	hostKey := &SSHHostKey{}
	err := db.conn.QueryRow(`
		SELECT host, port, key_type, fingerprint, public_key, first_seen, last_seen
		FROM ssh_host_keys
		WHERE host = ? AND port = ?
	`, host, port).Scan(
		&hostKey.Host, &hostKey.Port, &hostKey.KeyType,
		&hostKey.Fingerprint, &hostKey.PublicKey,
		&hostKey.FirstSeen, &hostKey.LastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return hostKey, nil
}

// ====================== Cleanup ======================

// CleanExpiredSSHSessions 清理过期的 SSH 会话
func (db *DB) CleanExpiredSSHSessions() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM ssh_sessions
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanInactiveSSHSessions 清理长时间未活跃的会话
func (db *DB) CleanInactiveSSHSessions(inactiveDays int) (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM ssh_sessions
		WHERE last_active_at < ? AND status = 'idle'
	`, time.Now().Add(-time.Duration(inactiveDays)*24*time.Hour))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanOldSSHHistory 清理旧的历史记录
func (db *DB) CleanOldSSHHistory(maxAgeDays int) (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM ssh_history
		WHERE timestamp < ?
	`, time.Now().Add(-time.Duration(maxAgeDays)*24*time.Hour))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ====================== Statistics ======================

// CountSSHSessionsByIP 统计IP创建的会话数
func (db *DB) CountSSHSessionsByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM ssh_sessions WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}
