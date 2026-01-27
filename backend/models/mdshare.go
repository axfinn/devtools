package models

import (
	"database/sql"
	"time"
)

type MDShare struct {
	ID         string     `json:"id"`
	Content    string     `json:"content"`
	Title      string     `json:"title"`
	CreatorKey string     `json:"-"`
	AccessKey  string     `json:"-"`
	MaxViews   int        `json:"max_views"`
	Views      int        `json:"views"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	CreatorIP  string     `json:"-"`
	ShortCode  string     `json:"short_code"`
}

func (db *DB) InitMDShare() error {
	// Create table without short_code first (for compatibility)
	query := `
	CREATE TABLE IF NOT EXISTS markdown_shares (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		title TEXT DEFAULT '',
		creator_key TEXT NOT NULL,
		access_key TEXT NOT NULL,
		max_views INTEGER DEFAULT 5,
		views INTEGER DEFAULT 0,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_mdshare_expires_at ON markdown_shares(expires_at);
	CREATE INDEX IF NOT EXISTS idx_mdshare_creator_ip ON markdown_shares(creator_ip);
	`
	_, err := db.conn.Exec(query)
	if err != nil {
		return err
	}

	// Add short_code column if not exists (for existing tables)
	db.conn.Exec("ALTER TABLE markdown_shares ADD COLUMN short_code TEXT DEFAULT ''")

	// Create index on short_code after column exists
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_mdshare_short_code ON markdown_shares(short_code)")

	return nil
}

func (db *DB) CreateMDShare(share *MDShare) error {
	share.ID = generateID(8)
	share.CreatedAt = time.Now()
	share.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO markdown_shares (id, content, title, creator_key, access_key, max_views, views, expires_at, created_at, updated_at, creator_ip, short_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, share.ID, share.Content, share.Title, share.CreatorKey, share.AccessKey,
		share.MaxViews, share.Views, share.ExpiresAt,
		share.CreatedAt, share.UpdatedAt, share.CreatorIP, share.ShortCode)

	return err
}

func (db *DB) GetMDShare(id string) (*MDShare, error) {
	share := &MDShare{}
	var expiresAt sql.NullTime
	var shortCode sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, content, title, creator_key, access_key, max_views, views, expires_at, created_at, updated_at, creator_ip, COALESCE(short_code, '')
		FROM markdown_shares WHERE id = ?
	`, id).Scan(
		&share.ID, &share.Content, &share.Title, &share.CreatorKey, &share.AccessKey,
		&share.MaxViews, &share.Views, &expiresAt,
		&share.CreatedAt, &share.UpdatedAt, &share.CreatorIP, &shortCode)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		share.ExpiresAt = &expiresAt.Time
	}
	share.ShortCode = shortCode.String

	return share, nil
}

func (db *DB) UpdateMDShare(share *MDShare) error {
	share.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		UPDATE markdown_shares
		SET content = ?, title = ?, access_key = ?, max_views = ?, views = ?,
		    expires_at = ?, updated_at = ?, short_code = ?
		WHERE id = ?
	`, share.Content, share.Title, share.AccessKey, share.MaxViews, share.Views,
		share.ExpiresAt, share.UpdatedAt, share.ShortCode, share.ID)

	return err
}

func (db *DB) UpdateMDShareShortCode(id, shortCode string) error {
	_, err := db.conn.Exec("UPDATE markdown_shares SET short_code = ?, updated_at = ? WHERE id = ?",
		shortCode, time.Now(), id)
	return err
}

func (db *DB) IncrementMDShareViews(id string) error {
	_, err := db.conn.Exec("UPDATE markdown_shares SET views = views + 1, updated_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (db *DB) DeleteMDShare(id string) error {
	_, err := db.conn.Exec("DELETE FROM markdown_shares WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpiredMDShares() (int64, error) {
	// Get shares to delete (for short URL cleanup)
	rows, err := db.conn.Query(`
		SELECT id, short_code FROM markdown_shares
		WHERE (expires_at IS NOT NULL AND expires_at < ?)
		   OR (views >= max_views)
	`, time.Now())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []string
	var shortCodes []string
	for rows.Next() {
		var id, shortCode string
		if err := rows.Scan(&id, &shortCode); err == nil {
			ids = append(ids, id)
			if shortCode != "" {
				shortCodes = append(shortCodes, shortCode)
			}
		}
	}

	// Delete associated short URLs
	for _, code := range shortCodes {
		db.DeleteShortURL(code)
	}

	// Delete expired shares
	result, err := db.conn.Exec(`
		DELETE FROM markdown_shares
		WHERE (expires_at IS NOT NULL AND expires_at < ?)
		   OR (views >= max_views)
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) CountMDSharesByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM markdown_shares WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

func (db *DB) ListAllMDShares() ([]*MDShare, error) {
	rows, err := db.conn.Query(`
		SELECT id, content, title, creator_key, access_key, max_views, views, expires_at, created_at, updated_at, creator_ip, COALESCE(short_code, '')
		FROM markdown_shares ORDER BY created_at DESC LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*MDShare
	for rows.Next() {
		share := &MDShare{}
		var expiresAt sql.NullTime
		var shortCode sql.NullString

		err := rows.Scan(
			&share.ID, &share.Content, &share.Title, &share.CreatorKey, &share.AccessKey,
			&share.MaxViews, &share.Views, &expiresAt,
			&share.CreatedAt, &share.UpdatedAt, &share.CreatorIP, &shortCode)
		if err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			share.ExpiresAt = &expiresAt.Time
		}
		share.ShortCode = shortCode.String
		shares = append(shares, share)
	}

	return shares, nil
}
