package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

// NFSShare NFS 文件分享记录
type NFSShare struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	FilePath  string     `json:"file_path"`
	FileSize  int64      `json:"file_size"`
	MimeType  string     `json:"mime_type"`
	MaxViews  int        `json:"max_views"`
	Views     int        `json:"views"`
	Password  string     `json:"-"` // bcrypt hash，不对外暴露
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	CreatorIP string     `json:"creator_ip"`
}

// NFSShareLog NFS 分享访问日志
type NFSShareLog struct {
	ID         int64     `json:"id"`
	ShareID    string    `json:"share_id"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	Status     string    `json:"status"`     // success / denied_views / denied_expired / file_missing / error
	BytesSent  int64     `json:"bytes_sent"`
	AccessedAt time.Time `json:"accessed_at"`
}

// InitNFSShare 初始化 NFS 分享数据库表
func (db *DB) InitNFSShare() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS nfs_shares (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			file_path  TEXT NOT NULL,
			file_size  INTEGER NOT NULL DEFAULT 0,
			mime_type  TEXT NOT NULL DEFAULT 'application/octet-stream',
			max_views  INTEGER NOT NULL DEFAULT 1,
			views      INTEGER NOT NULL DEFAULT 0,
			password   TEXT NOT NULL DEFAULT '',
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			creator_ip TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_nfs_shares_expires_at ON nfs_shares(expires_at);
		CREATE TABLE IF NOT EXISTS nfs_share_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			share_id    TEXT NOT NULL,
			client_ip   TEXT,
			user_agent  TEXT,
			status      TEXT NOT NULL,
			bytes_sent  INTEGER NOT NULL DEFAULT 0,
			accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_nfs_share_logs_share_id   ON nfs_share_logs(share_id);
		CREATE INDEX IF NOT EXISTS idx_nfs_share_logs_accessed_at ON nfs_share_logs(accessed_at);
	`)
	if err != nil {
		return err
	}
	// 迁移：为旧数据库添加 password 列
	db.conn.Exec(`ALTER TABLE nfs_shares ADD COLUMN password TEXT NOT NULL DEFAULT ''`)
	return nil
}

// CreateNFSShare 创建 NFS 分享
func (db *DB) CreateNFSShare(name, filePath, mimeType, password string, fileSize int64, maxViews int, expiresAt *time.Time, creatorIP string) (*NFSShare, error) {
	b := make([]byte, 4)
	rand.Read(b)
	id := hex.EncodeToString(b)

	var expiresAtVal interface{}
	if expiresAt != nil {
		expiresAtVal = *expiresAt
	}

	_, err := db.conn.Exec(
		`INSERT INTO nfs_shares (id, name, file_path, file_size, mime_type, max_views, views, password, expires_at, creator_ip)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?)`,
		id, name, filePath, fileSize, mimeType, maxViews, password, expiresAtVal, creatorIP,
	)
	if err != nil {
		return nil, err
	}
	return db.GetNFSShare(id)
}

// GetNFSShare 根据 ID 获取分享记录
func (db *DB) GetNFSShare(id string) (*NFSShare, error) {
	s := &NFSShare{}
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(
		`SELECT id, name, file_path, file_size, mime_type, max_views, views, password, expires_at, created_at, creator_ip
		 FROM nfs_shares WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.FilePath, &s.FileSize, &s.MimeType, &s.MaxViews, &s.Views, &s.Password, &expiresAt, &s.CreatedAt, &s.CreatorIP)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		s.ExpiresAt = &expiresAt.Time
	}
	return s, nil
}

// IncrementNFSShareViews 原子自增访问次数
func (db *DB) IncrementNFSShareViews(id string) error {
	_, err := db.conn.Exec(`UPDATE nfs_shares SET views = views + 1 WHERE id = ?`, id)
	return err
}

// AddNFSShareLog 记录一条访问日志
func (db *DB) AddNFSShareLog(shareID, clientIP, userAgent, status string, bytesSent int64) error {
	_, err := db.conn.Exec(
		`INSERT INTO nfs_share_logs (share_id, client_ip, user_agent, status, bytes_sent) VALUES (?, ?, ?, ?, ?)`,
		shareID, clientIP, userAgent, status, bytesSent,
	)
	return err
}

// GetAllNFSShares 分页获取所有分享（管理员用）
func (db *DB) GetAllNFSShares(page, pageSize int) ([]NFSShare, int, error) {
	offset := (page - 1) * pageSize
	rows, err := db.conn.Query(
		`SELECT id, name, file_path, file_size, mime_type, max_views, views, expires_at, created_at, creator_ip
		 FROM nfs_shares ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shares []NFSShare
	for rows.Next() {
		var s NFSShare
		var expiresAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.Name, &s.FilePath, &s.FileSize, &s.MimeType, &s.MaxViews, &s.Views, &expiresAt, &s.CreatedAt, &s.CreatorIP); err != nil {
			continue
		}
		if expiresAt.Valid {
			s.ExpiresAt = &expiresAt.Time
		}
		shares = append(shares, s)
	}

	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM nfs_shares`).Scan(&total)
	return shares, total, nil
}

// GetNFSShareLogs 分页获取分享访问日志
func (db *DB) GetNFSShareLogs(shareID string, page, pageSize int) ([]NFSShareLog, int, error) {
	offset := (page - 1) * pageSize
	rows, err := db.conn.Query(
		`SELECT id, share_id, client_ip, user_agent, status, bytes_sent, accessed_at
		 FROM nfs_share_logs WHERE share_id = ? ORDER BY accessed_at DESC LIMIT ? OFFSET ?`,
		shareID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []NFSShareLog
	for rows.Next() {
		var l NFSShareLog
		if err := rows.Scan(&l.ID, &l.ShareID, &l.ClientIP, &l.UserAgent, &l.Status, &l.BytesSent, &l.AccessedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM nfs_share_logs WHERE share_id = ?`, shareID).Scan(&total)
	return logs, total, nil
}

// DeleteNFSShare 删除分享及其日志
func (db *DB) DeleteNFSShare(id string) error {
	if _, err := db.conn.Exec(`DELETE FROM nfs_share_logs WHERE share_id = ?`, id); err != nil {
		return err
	}
	_, err := db.conn.Exec(`DELETE FROM nfs_shares WHERE id = ?`, id)
	return err
}

// UpdateNFSShare 更新分享的访问次数上限与过期时间
func (db *DB) UpdateNFSShare(id string, maxViews int, expiresAt *time.Time) error {
	var expiresAtVal interface{}
	if expiresAt != nil {
		expiresAtVal = *expiresAt
	}
	_, err := db.conn.Exec(
		`UPDATE nfs_shares SET max_views = ?, expires_at = ? WHERE id = ?`,
		maxViews, expiresAtVal, id,
	)
	return err
}

// CleanExpiredNFSShares 清理已过期或已耗尽访问次数的分享
func (db *DB) CleanExpiredNFSShares() (int, error) {
	rows, err := db.conn.Query(`
		SELECT id FROM nfs_shares
		WHERE (expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP)
		   OR (max_views > 0 AND views >= max_views)
	`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		db.conn.Exec(`DELETE FROM nfs_share_logs WHERE share_id = ?`, id)
		db.conn.Exec(`DELETE FROM nfs_shares WHERE id = ?`, id)
	}
	return len(ids), nil
}
