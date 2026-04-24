package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func init() {
	RegisterInit("MiniMax结果分享(minimax_result_shares)", (*DB).InitMiniMaxResultShares)
}

type MiniMaxResultShare struct {
	ID         string    `json:"id"`
	APIKeyID   string    `json:"api_key_id"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	ResultType string    `json:"result_type"`
	Model      string    `json:"model"`
	Status     string    `json:"status"`
	Payload    string    `json:"payload"`
	AssetsJSON string    `json:"assets_json"`
	CreatorIP  string    `json:"creator_ip"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type MiniMaxResultShareAsset struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	OriginalURL string `json:"original_url,omitempty"`
}

func (db *DB) InitMiniMaxResultShares() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS minimax_result_shares (
			id TEXT PRIMARY KEY,
			api_key_id TEXT DEFAULT '',
			title TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			result_type TEXT NOT NULL,
			model TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			payload TEXT DEFAULT '{}',
			assets_json TEXT DEFAULT '[]',
			creator_ip TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_minimax_result_shares_created_at ON minimax_result_shares(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_minimax_result_shares_status ON minimax_result_shares(status, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_minimax_result_shares_type ON minimax_result_shares(result_type, created_at DESC);
	`)
	return err
}

func (db *DB) CreateMiniMaxResultShare(share *MiniMaxResultShare) error {
	if share == nil {
		return errors.New("share is nil")
	}
	if share.ID == "" {
		share.ID = "mrs_" + generateID(10)
	}
	now := time.Now()
	share.CreatedAt = now
	share.UpdatedAt = now
	if share.Status == "" {
		share.Status = "active"
	}
	if share.Payload == "" {
		share.Payload = "{}"
	}
	if share.AssetsJSON == "" {
		share.AssetsJSON = "[]"
	}
	_, err := db.conn.Exec(`
		INSERT INTO minimax_result_shares (
			id, api_key_id, title, summary, result_type, model, status, payload, assets_json, creator_ip, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, share.ID, share.APIKeyID, share.Title, share.Summary, share.ResultType, share.Model, share.Status, share.Payload, share.AssetsJSON, share.CreatorIP, share.CreatedAt, share.UpdatedAt)
	return err
}

func (db *DB) GetMiniMaxResultShare(id string) (*MiniMaxResultShare, error) {
	item := &MiniMaxResultShare{}
	err := db.conn.QueryRow(`
		SELECT id, api_key_id, title, summary, result_type, model, status, payload, assets_json, creator_ip, created_at, updated_at
		FROM minimax_result_shares
		WHERE id = ?
	`, id).Scan(&item.ID, &item.APIKeyID, &item.Title, &item.Summary, &item.ResultType, &item.Model, &item.Status, &item.Payload, &item.AssetsJSON, &item.CreatorIP, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (db *DB) ListMiniMaxResultShares(status, keyword string, limit, offset int) ([]*MiniMaxResultShare, error) {
	query := `
		SELECT id, api_key_id, title, summary, result_type, model, status, payload, assets_json, creator_ip, created_at, updated_at
		FROM minimax_result_shares
		WHERE 1 = 1`
	args := make([]interface{}, 0, 4)
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if keyword != "" {
		query += ` AND (title LIKE ? OR summary LIKE ? OR model LIKE ? OR result_type LIKE ?)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*MiniMaxResultShare, 0)
	for rows.Next() {
		item := &MiniMaxResultShare{}
		if err := rows.Scan(&item.ID, &item.APIKeyID, &item.Title, &item.Summary, &item.ResultType, &item.Model, &item.Status, &item.Payload, &item.AssetsJSON, &item.CreatorIP, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (db *DB) CountMiniMaxResultShares(status, keyword string) (int, error) {
	query := `SELECT COUNT(*) FROM minimax_result_shares WHERE 1 = 1`
	args := make([]interface{}, 0, 3)
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if keyword != "" {
		query += ` AND (title LIKE ? OR summary LIKE ? OR model LIKE ? OR result_type LIKE ?)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}

	var total int
	err := db.conn.QueryRow(query, args...).Scan(&total)
	return total, err
}

func (db *DB) UpdateMiniMaxResultShare(share *MiniMaxResultShare) error {
	if share == nil {
		return errors.New("share is nil")
	}
	share.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE minimax_result_shares
		SET title = ?, summary = ?, status = ?, payload = ?, assets_json = ?, updated_at = ?
		WHERE id = ?
	`, share.Title, share.Summary, share.Status, share.Payload, share.AssetsJSON, share.UpdatedAt, share.ID)
	return err
}

func (db *DB) DeleteMiniMaxResultShare(id string) error {
	_, err := db.conn.Exec(`DELETE FROM minimax_result_shares WHERE id = ?`, id)
	return err
}

func ParseMiniMaxResultShareAssets(raw string) ([]MiniMaxResultShareAsset, error) {
	if raw == "" {
		return []MiniMaxResultShareAsset{}, nil
	}
	items := make([]MiniMaxResultShareAsset, 0)
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func MustMarshalMiniMaxResultShareAssets(items []MiniMaxResultShareAsset) string {
	if len(items) == 0 {
		return "[]"
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func scanNullString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
