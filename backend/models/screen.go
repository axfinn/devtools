package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

func init() {
	RegisterInit("屏幕共享(screen_sessions)", (*DB).InitScreenSessions)
}

// ScreenSession 屏幕共享会话。
// host_user_id 关联 askit_users.id — 创建会话要求 askit 登录,保证可追溯、可吊销。
// password 留空表示 viewer 匿名可加入(任意人凭链接即可看);非空则 viewer 必须提交明文密码校验。
// status: active=正在共享;stopped=主机主动结束;expired=到期(cleanup 标记)。
type ScreenSession struct {
	ID                 string     `json:"id"`
	HostUserID         string     `json:"host_user_id"`
	Title              string     `json:"title"`
	Password           string     `json:"-"` // SHA-256 hex,对外不暴露
	AllowRemoteControl bool       `json:"allow_remote_control"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	StoppedAt          *time.Time `json:"stopped_at,omitempty"`
	HasPassword        bool       `json:"has_password"` // 公开视图填充,前端用于决定是否走密码框
}

// InitScreenSessions 建表。
func (db *DB) InitScreenSessions() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS screen_sessions (
			id                    TEXT PRIMARY KEY,
			host_user_id          TEXT NOT NULL,
			title                 TEXT NOT NULL DEFAULT '',
			password              TEXT NOT NULL DEFAULT '',
			allow_remote_control  INTEGER NOT NULL DEFAULT 0,
			status                TEXT NOT NULL DEFAULT 'active',
			created_at            DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at            DATETIME,
			stopped_at            DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_screen_sessions_host     ON screen_sessions(host_user_id);
		CREATE INDEX IF NOT EXISTS idx_screen_sessions_expires  ON screen_sessions(expires_at);
		CREATE INDEX IF NOT EXISTS idx_screen_sessions_status   ON screen_sessions(status);
	`)
	return err
}

func genScreenID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateScreenSession 创建会话。host_user_id 必填,作为创建者归属。
func (db *DB) CreateScreenSession(hostUserID, title, password string, allowRemoteControl bool, expiresAt *time.Time) (*ScreenSession, error) {
	if hostUserID == "" {
		return nil, errors.New("host_user_id required")
	}
	id := genScreenID()
	var exp interface{}
	if expiresAt != nil {
		exp = *expiresAt
	}
	allow := 0
	if allowRemoteControl {
		allow = 1
	}
	_, err := db.conn.Exec(
		`INSERT INTO screen_sessions (id, host_user_id, title, password, allow_remote_control, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, hostUserID, title, password, allow, exp,
	)
	if err != nil {
		return nil, err
	}
	return db.GetScreenSession(id)
}

func scanScreenSession(row interface{ Scan(...interface{}) error }) (*ScreenSession, error) {
	s := &ScreenSession{}
	var (
		allow     int
		expiresAt sql.NullTime
		stoppedAt sql.NullTime
	)
	if err := row.Scan(&s.ID, &s.HostUserID, &s.Title, &s.Password, &allow, &s.Status, &s.CreatedAt, &expiresAt, &stoppedAt); err != nil {
		return nil, err
	}
	s.AllowRemoteControl = allow == 1
	if expiresAt.Valid {
		t := expiresAt.Time
		s.ExpiresAt = &t
	}
	if stoppedAt.Valid {
		t := stoppedAt.Time
		s.StoppedAt = &t
	}
	return s, nil
}

// GetScreenSession 取完整会话(含 password 哈希,内部用)。
func (db *DB) GetScreenSession(id string) (*ScreenSession, error) {
	row := db.conn.QueryRow(
		`SELECT id, host_user_id, title, password, allow_remote_control, status, created_at, expires_at, stopped_at
		   FROM screen_sessions WHERE id = ?`, id)
	return scanScreenSession(row)
}

// GetScreenSessionPublic 取公开视图(隐藏 password 哈希)。
func (db *DB) GetScreenSessionPublic(id string) (*ScreenSession, error) {
	s, err := db.GetScreenSession(id)
	if err != nil {
		return nil, err
	}
	s.HasPassword = s.Password != "" // 必须在清空 Password 之前算
	s.Password = ""                  // 公开视图永不带哈希
	return s, nil
}

// StopScreenSession 标记会话为 stopped。仅在当前为 active 时生效。
func (db *DB) StopScreenSession(id, hostUserID string) error {
	res, err := db.conn.Exec(
		`UPDATE screen_sessions
		    SET status = 'stopped', stopped_at = CURRENT_TIMESTAMP
		  WHERE id = ? AND host_user_id = ? AND status = 'active'`,
		id, hostUserID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrScreenSessionNotStoppable
	}
	return nil
}

// ErrScreenSessionNotStoppable 表示当前状态不允许 stop(已停止 / 已过期 / 不归该用户所有)。
var ErrScreenSessionNotStoppable = errors.New("screen_session_not_stoppable")

// CleanExpiredScreenSessions 把过期 active 会话置为 expired。返回受影响行数。
func (db *DB) CleanExpiredScreenSessions() (int64, error) {
	res, err := db.conn.Exec(
		`UPDATE screen_sessions
		    SET status = 'expired'
		  WHERE status = 'active'
		    AND expires_at IS NOT NULL
		    AND expires_at < CURRENT_TIMESTAMP`,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListActiveScreenSessionsByHost 列某用户仍 active 的会话(供 host 端"我的会话"使用)。
func (db *DB) ListActiveScreenSessionsByHost(hostUserID string) ([]*ScreenSession, error) {
	rows, err := db.conn.Query(
		`SELECT id, host_user_id, title, password, allow_remote_control, status, created_at, expires_at, stopped_at
		   FROM screen_sessions
		  WHERE host_user_id = ? AND status = 'active'
		  ORDER BY created_at DESC
		  LIMIT 50`,
		hostUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ScreenSession{}
	for rows.Next() {
		s, err := scanScreenSession(rows)
		if err != nil {
			return nil, err
		}
		s.Password = ""
		out = append(out, s)
	}
	return out, rows.Err()
}
