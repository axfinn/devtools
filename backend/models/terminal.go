package models

import (
	"database/sql"
	"time"
)

type TerminalSession struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Password    string     `json:"-"`
	PasswordIndex string   `json:"-"`
	CreatorKey  string     `json:"-"`
	Shell       string     `json:"shell"` // shell 类型: bash, zsh, sh
	Width       int        `json:"width"`  // 终端宽度
	Height      int        `json:"height"` // 终端高度
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatorIP   string     `json:"-"`
}

func (db *DB) InitTerminal() error {
	query := `
	CREATE TABLE IF NOT EXISTS terminal_sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '终端',
		password TEXT NOT NULL,
		password_index TEXT NOT NULL DEFAULT '',
		creator_key TEXT NOT NULL,
		shell TEXT NOT NULL DEFAULT '/bin/bash',
		width INTEGER NOT NULL DEFAULT 80,
		height INTEGER NOT NULL DEFAULT 24,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_terminal_expires_at ON terminal_sessions(expires_at);
	CREATE INDEX IF NOT EXISTS idx_terminal_creator_ip ON terminal_sessions(creator_ip);
	CREATE INDEX IF NOT EXISTS idx_terminal_password_index ON terminal_sessions(password_index) WHERE password_index != '';
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreateTerminalSession(session *TerminalSession) error {
	session.ID = generateID(8)
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO terminal_sessions (id, name, password, password_index, creator_key, shell, width, height, expires_at, created_at, updated_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.Name, session.Password, session.PasswordIndex, session.CreatorKey,
		session.Shell, session.Width, session.Height, session.ExpiresAt, session.CreatedAt, session.UpdatedAt, session.CreatorIP)

	return err
}

func (db *DB) GetTerminalSession(id string) (*TerminalSession, error) {
	session := &TerminalSession{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, name, password, password_index, creator_key, shell, width, height, expires_at, created_at, updated_at, creator_ip
		FROM terminal_sessions WHERE id = ?
	`, id).Scan(
		&session.ID, &session.Name, &session.Password, &session.PasswordIndex, &session.CreatorKey,
		&session.Shell, &session.Width, &session.Height, &expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		session.ExpiresAt = &expiresAt.Time
	}

	return session, nil
}

func (db *DB) GetTerminalSessionByPasswordIndex(passwordIndex string) (*TerminalSession, error) {
	session := &TerminalSession{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, name, password, password_index, creator_key, shell, width, height, expires_at, created_at, updated_at, creator_ip
		FROM terminal_sessions WHERE password_index = ?
	`, passwordIndex).Scan(
		&session.ID, &session.Name, &session.Password, &session.PasswordIndex, &session.CreatorKey,
		&session.Shell, &session.Width, &session.Height, &expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		session.ExpiresAt = &expiresAt.Time
	}

	return session, nil
}

func (db *DB) UpdateTerminalSession(id string, width, height int) error {
	_, err := db.conn.Exec(
		"UPDATE terminal_sessions SET width = ?, height = ?, updated_at = ? WHERE id = ?",
		width, height, time.Now(), id)
	return err
}

func (db *DB) UpdateTerminalName(id, name string) error {
	_, err := db.conn.Exec(
		"UPDATE terminal_sessions SET name = ?, updated_at = ? WHERE id = ?",
		name, time.Now(), id)
	return err
}

func (db *DB) ExtendTerminalSession(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(
		"UPDATE terminal_sessions SET expires_at = ?, updated_at = ? WHERE id = ?",
		expiresAt, time.Now(), id)
	return err
}

func (db *DB) DeleteTerminalSession(id string) error {
	_, err := db.conn.Exec("DELETE FROM terminal_sessions WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpiredTerminalSessions() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM terminal_sessions
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) CountTerminalSessionsByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM terminal_sessions WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

// SSH Session 结构体
type SSHSession struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Host         string     `json:"host"`
	Port         int        `json:"port"`
	Username     string     `json:"username"`
	Password     string     `json:"-"`
	PrivateKey   string     `json:"-"`
	ConfigIndex  string     `json:"-"`
	CreatorKey   string     `json:"-"`
	UserToken    string     `json:"-"` // 用户口令（用于标识用户）
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	KeepSession  bool       `json:"keep_session"` // 是否保持会话
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatorIP    string     `json:"-"`
}

func (db *DB) InitSSH() error {
	// 先创建表
	query := `
	CREATE TABLE IF NOT EXISTS ssh_sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT 'SSH 终端',
		host TEXT NOT NULL,
		port INTEGER NOT NULL DEFAULT 22,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		private_key TEXT,
		config_index TEXT NOT NULL DEFAULT '',
		creator_key TEXT NOT NULL,
		user_token TEXT NOT NULL DEFAULT '',
		width INTEGER NOT NULL DEFAULT 80,
		height INTEGER NOT NULL DEFAULT 24,
		keep_session INTEGER NOT NULL DEFAULT 0,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	`
	_, err := db.conn.Exec(query)
	if err != nil {
		return err
	}

	// 迁移数据：添加 keep_session 列（如果不存在）
	_, err = db.conn.Exec("ALTER TABLE ssh_sessions ADD COLUMN keep_session INTEGER NOT NULL DEFAULT 0")
	// 忽略错误，因为列可能已存在

	// 迁移数据：添加 user_token 列（如果不存在）
	_, err = db.conn.Exec("ALTER TABLE ssh_sessions ADD COLUMN user_token TEXT NOT NULL DEFAULT ''")
	// 忽略错误，因为列可能已存在

	// 创建索引
	_, err = db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_ssh_expires_at ON ssh_sessions(expires_at)")
	_, err = db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_ssh_creator_ip ON ssh_sessions(creator_ip)")
	_, err = db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_ssh_config_index ON ssh_sessions(config_index)")
	_, err = db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_ssh_user_token ON ssh_sessions(user_token)")

	return nil
}

func (db *DB) CreateSSHSession(session *SSHSession) error {
	session.ID = generateID(8)
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO ssh_sessions (id, name, host, port, username, password, private_key, config_index, creator_key, user_token, width, height, expires_at, created_at, updated_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.Name, session.Host, session.Port, session.Username, session.Password, session.PrivateKey, session.ConfigIndex, session.CreatorKey, session.UserToken,
		session.Width, session.Height, session.ExpiresAt, session.CreatedAt, session.UpdatedAt, session.CreatorIP)

	return err
}

func (db *DB) GetSSHSession(id string) (*SSHSession, error) {
	session := &SSHSession{}
	var expiresAt sql.NullTime
	var privateKey sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, host, port, username, password, private_key, config_index, creator_key, width, height, expires_at, created_at, updated_at, creator_ip
		FROM ssh_sessions WHERE id = ?
	`, id).Scan(
		&session.ID, &session.Name, &session.Host, &session.Port, &session.Username, &session.Password, &privateKey, &session.ConfigIndex, &session.CreatorKey,
		&session.Width, &session.Height, &expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		session.ExpiresAt = &expiresAt.Time
	}
	if privateKey.Valid {
		session.PrivateKey = privateKey.String
	}

	return session, nil
}

func (db *DB) GetSSHSessionByConfigIndex(configIndex string) (*SSHSession, error) {
	session := &SSHSession{}
	var expiresAt sql.NullTime
	var privateKey sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, host, port, username, password, private_key, config_index, creator_key, width, height, expires_at, created_at, updated_at, creator_ip
		FROM ssh_sessions WHERE config_index = ?
	`, configIndex).Scan(
		&session.ID, &session.Name, &session.Host, &session.Port, &session.Username, &session.Password, &privateKey, &session.ConfigIndex, &session.CreatorKey,
		&session.Width, &session.Height, &expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		session.ExpiresAt = &expiresAt.Time
	}
	if privateKey.Valid {
		session.PrivateKey = privateKey.String
	}

	return session, nil
}

func (db *DB) GetSSHSessionsByConfigIndex(configIndex string) ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username, width, height, keep_session, expires_at, created_at, updated_at, creator_ip
		FROM ssh_sessions WHERE config_index = ?
		ORDER BY created_at DESC
	`, configIndex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height, &session.KeepSession, &expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP,
		)
		if err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (db *DB) GetSSHSessionsByUserToken(userToken string) ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username, width, height, keep_session, expires_at, created_at, updated_at
		FROM ssh_sessions WHERE user_token = ?
		ORDER BY created_at DESC
	`, userToken)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height, &session.KeepSession, &expiresAt, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (db *DB) UpdateSSHName(id, name string) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET name = ?, updated_at = ? WHERE id = ?",
		name, time.Now(), id)
	return err
}

func (db *DB) ExtendSSHSession(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(
		"UPDATE ssh_sessions SET expires_at = ?, updated_at = ? WHERE id = ?",
		expiresAt, time.Now(), id)
	return err
}

func (db *DB) DeleteSSHSession(id string) error {
	_, err := db.conn.Exec("DELETE FROM ssh_sessions WHERE id = ?", id)
	return err
}

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

func (db *DB) CountSSHSessionsByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM ssh_sessions WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

func (db *DB) GetAllSSHSessions() ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username, width, height, keep_session, expires_at, created_at, updated_at, creator_ip
		FROM ssh_sessions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height, &session.KeepSession, &expiresAt, &session.CreatedAt, &session.UpdatedAt, &session.CreatorIP,
		)
		if err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (db *DB) GetSSHSessionsByCreatorKey(creatorKey string) ([]*SSHSession, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, host, port, username, width, height, keep_session, expires_at, created_at, updated_at
		FROM ssh_sessions
		WHERE creator_key = ?
		ORDER BY created_at DESC
	`, creatorKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SSHSession
	for rows.Next() {
		session := &SSHSession{}
		var expiresAt sql.NullTime
		err := rows.Scan(
			&session.ID, &session.Name, &session.Host, &session.Port, &session.Username,
			&session.Width, &session.Height, &session.KeepSession, &expiresAt, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			session.ExpiresAt = &expiresAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}
