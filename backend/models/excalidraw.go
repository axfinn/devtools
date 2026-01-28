package models

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"io"
	"time"
)

type ExcalidrawShare struct {
	ID          string     `json:"id"`
	Content     string     `json:"content"`
	Title       string     `json:"title"`
	CreatorKey  string     `json:"-"`
	AccessKey   string     `json:"-"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatorIP   string     `json:"-"`
	ShortCode   string     `json:"short_code"`
	IsPermanent bool       `json:"is_permanent"`
}

func (db *DB) InitExcalidraw() error {
	query := `
	CREATE TABLE IF NOT EXISTS excalidraw_shares (
		id TEXT PRIMARY KEY,
		content BLOB NOT NULL,
		title TEXT DEFAULT '',
		creator_key TEXT NOT NULL,
		access_key TEXT NOT NULL,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT,
		short_code TEXT DEFAULT '',
		is_permanent BOOLEAN DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_excalidraw_expires_at ON excalidraw_shares(expires_at);
	CREATE INDEX IF NOT EXISTS idx_excalidraw_creator_ip ON excalidraw_shares(creator_ip);
	CREATE INDEX IF NOT EXISTS idx_excalidraw_short_code ON excalidraw_shares(short_code);
	`
	_, err := db.conn.Exec(query)
	return err
}

func compressContent(content string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressContent(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (db *DB) CreateExcalidrawShare(share *ExcalidrawShare) error {
	share.ID = generateID(8)
	share.CreatedAt = time.Now()
	share.UpdatedAt = time.Now()

	compressed, err := compressContent(share.Content)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`
		INSERT INTO excalidraw_shares (id, content, title, creator_key, access_key, expires_at, created_at, updated_at, creator_ip, short_code, is_permanent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, share.ID, compressed, share.Title, share.CreatorKey, share.AccessKey,
		share.ExpiresAt, share.CreatedAt, share.UpdatedAt, share.CreatorIP, share.ShortCode, share.IsPermanent)

	return err
}

func (db *DB) GetExcalidrawShare(id string) (*ExcalidrawShare, error) {
	share := &ExcalidrawShare{}
	var expiresAt sql.NullTime
	var shortCode sql.NullString
	var compressed []byte

	err := db.conn.QueryRow(`
		SELECT id, content, title, creator_key, access_key, expires_at, created_at, updated_at, creator_ip, COALESCE(short_code, ''), is_permanent
		FROM excalidraw_shares WHERE id = ?
	`, id).Scan(
		&share.ID, &compressed, &share.Title, &share.CreatorKey, &share.AccessKey,
		&expiresAt, &share.CreatedAt, &share.UpdatedAt, &share.CreatorIP, &shortCode, &share.IsPermanent)

	if err != nil {
		return nil, err
	}

	content, err := decompressContent(compressed)
	if err != nil {
		return nil, err
	}
	share.Content = content

	if expiresAt.Valid {
		share.ExpiresAt = &expiresAt.Time
	}
	share.ShortCode = shortCode.String

	return share, nil
}

func (db *DB) UpdateExcalidrawShare(share *ExcalidrawShare) error {
	share.UpdatedAt = time.Now()

	compressed, err := compressContent(share.Content)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`
		UPDATE excalidraw_shares
		SET content = ?, title = ?, expires_at = ?, updated_at = ?, short_code = ?, is_permanent = ?
		WHERE id = ?
	`, compressed, share.Title, share.ExpiresAt, share.UpdatedAt, share.ShortCode, share.IsPermanent, share.ID)

	return err
}

func (db *DB) UpdateExcalidrawShortCode(id, shortCode string) error {
	_, err := db.conn.Exec("UPDATE excalidraw_shares SET short_code = ?, updated_at = ? WHERE id = ?",
		shortCode, time.Now(), id)
	return err
}

func (db *DB) DeleteExcalidrawShare(id string) error {
	_, err := db.conn.Exec("DELETE FROM excalidraw_shares WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpiredExcalidrawShares() (int64, error) {
	// Get shares to delete (for short URL cleanup)
	rows, err := db.conn.Query(`
		SELECT id, short_code FROM excalidraw_shares
		WHERE expires_at IS NOT NULL AND expires_at < ? AND is_permanent = 0
	`, time.Now())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var shortCodes []string
	for rows.Next() {
		var id, shortCode string
		if err := rows.Scan(&id, &shortCode); err == nil {
			if shortCode != "" {
				shortCodes = append(shortCodes, shortCode)
			}
		}
	}

	// Delete associated short URLs
	for _, code := range shortCodes {
		db.DeleteShortURL(code)
	}

	// Delete expired shares (exclude permanent)
	result, err := db.conn.Exec(`
		DELETE FROM excalidraw_shares
		WHERE expires_at IS NOT NULL AND expires_at < ? AND is_permanent = 0
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) CountExcalidrawSharesByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM excalidraw_shares WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

func (db *DB) ListAllExcalidrawShares() ([]*ExcalidrawShare, error) {
	rows, err := db.conn.Query(`
		SELECT id, content, title, creator_key, access_key, expires_at, created_at, updated_at, creator_ip, COALESCE(short_code, ''), is_permanent
		FROM excalidraw_shares ORDER BY created_at DESC LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*ExcalidrawShare
	for rows.Next() {
		share := &ExcalidrawShare{}
		var expiresAt sql.NullTime
		var shortCode sql.NullString
		var compressed []byte

		err := rows.Scan(
			&share.ID, &compressed, &share.Title, &share.CreatorKey, &share.AccessKey,
			&expiresAt, &share.CreatedAt, &share.UpdatedAt, &share.CreatorIP, &shortCode, &share.IsPermanent)
		if err != nil {
			return nil, err
		}

		content, err := decompressContent(compressed)
		if err != nil {
			continue // skip corrupted entries
		}
		share.Content = content

		if expiresAt.Valid {
			share.ExpiresAt = &expiresAt.Time
		}
		share.ShortCode = shortCode.String
		shares = append(shares, share)
	}

	return shares, nil
}
