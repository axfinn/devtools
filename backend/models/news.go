package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func init() {
	RegisterInit("新闻(news_digests)", (*DB).InitNews)
}

// NewsItem 单条新闻条目(序列化进 NewsDigest.Items 的 JSON 数组)。
type NewsItem struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Source    string `json:"source"`
	Summary   string `json:"summary"`
	Category  string `json:"category"`
	Published string `json:"published"` // RFC3339 或空
}

// NewsDigest 每日一份新闻归档(按 date 唯一)。
type NewsDigest struct {
	Date        string    `json:"date"`       // YYYY-MM-DD
	DateLabel   string    `json:"date_label"` // 如 2026年10月25日
	Summary     string    `json:"summary"`    // AI 精编摘要(Markdown)
	Items       string    `json:"items"`      // JSON 数组字符串 []NewsItem
	SourceCount int       `json:"source_count"`
	ItemCount   int       `json:"item_count"`
	AIModel     string    `json:"ai_model"`
	Status      string    `json:"status"` // ok | partial | empty
	ErrorMsg    string    `json:"error_msg"`
	GeneratedAt time.Time `json:"generated_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (db *DB) InitNews() error {
	query := `
	CREATE TABLE IF NOT EXISTS news_digests (
		date         TEXT PRIMARY KEY,
		date_label   TEXT NOT NULL DEFAULT '',
		summary      TEXT NOT NULL DEFAULT '',
		items        TEXT NOT NULL DEFAULT '[]',
		source_count INTEGER NOT NULL DEFAULT 0,
		item_count   INTEGER NOT NULL DEFAULT 0,
		ai_model     TEXT NOT NULL DEFAULT '',
		status       TEXT NOT NULL DEFAULT 'ok',
		error_msg    TEXT NOT NULL DEFAULT '',
		generated_at DATETIME,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_news_digests_date ON news_digests(date DESC);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) UpsertNewsDigest(d *NewsDigest) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	if d.GeneratedAt.IsZero() {
		d.GeneratedAt = now
	}
	_, err := db.conn.Exec(`
		INSERT INTO news_digests
			(date, date_label, summary, items, source_count, item_count, ai_model, status, error_msg, generated_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			date_label   = excluded.date_label,
			summary      = excluded.summary,
			items        = excluded.items,
			source_count = excluded.source_count,
			item_count   = excluded.item_count,
			ai_model     = excluded.ai_model,
			status       = excluded.status,
			error_msg    = excluded.error_msg,
			generated_at = excluded.generated_at,
			updated_at   = excluded.updated_at
	`, d.Date, d.DateLabel, d.Summary, d.Items, d.SourceCount, d.ItemCount, d.AIModel, d.Status, d.ErrorMsg,
		d.GeneratedAt, d.CreatedAt, d.UpdatedAt)
	return err
}

func scanNewsDigest(row interface {
	Scan(...interface{}) error
}) (*NewsDigest, error) {
	d := &NewsDigest{}
	var generatedAt, createdAt, updatedAt sql.NullTime
	err := row.Scan(
		&d.Date, &d.DateLabel, &d.Summary, &d.Items, &d.SourceCount, &d.ItemCount,
		&d.AIModel, &d.Status, &d.ErrorMsg, &generatedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if generatedAt.Valid {
		d.GeneratedAt = generatedAt.Time
	}
	if createdAt.Valid {
		d.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		d.UpdatedAt = updatedAt.Time
	}
	return d, nil
}

func (db *DB) GetNewsDigest(date string) (*NewsDigest, error) {
	row := db.conn.QueryRow(`
		SELECT date, date_label, summary, items, source_count, item_count, ai_model, status, error_msg, generated_at, created_at, updated_at
		FROM news_digests WHERE date = ?
	`, date)
	return scanNewsDigest(row)
}

func (db *DB) GetLatestNewsDigest() (*NewsDigest, error) {
	row := db.conn.QueryRow(`
		SELECT date, date_label, summary, items, source_count, item_count, ai_model, status, error_msg, generated_at, created_at, updated_at
		FROM news_digests ORDER BY date DESC LIMIT 1
	`)
	return scanNewsDigest(row)
}

// ListNewsDates 返回归档索引(不含 items 大字段,仅为列表展示用)。
func (db *DB) ListNewsDates(limit, offset int) ([]NewsDigest, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := db.conn.Query(`
		SELECT date, date_label, source_count, item_count, ai_model, status, generated_at
		FROM news_digests ORDER BY date DESC LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NewsDigest
	for rows.Next() {
		d := NewsDigest{}
		var generatedAt sql.NullTime
		if err := rows.Scan(&d.Date, &d.DateLabel, &d.SourceCount, &d.ItemCount, &d.AIModel, &d.Status, &generatedAt); err != nil {
			return nil, err
		}
		if generatedAt.Valid {
			d.GeneratedAt = generatedAt.Time
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (db *DB) CountNewsDigests() (int, error) {
	var n int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM news_digests`).Scan(&n)
	return n, err
}

// DeleteNewsDigestsBefore 清理 date 早于 before(按 YYYY-MM-DD 字符串比较)的归档。retention_days>0 时调用。
func (db *DB) DeleteNewsDigestsBefore(before time.Time) (int64, error) {
	res, err := db.conn.Exec(`DELETE FROM news_digests WHERE date < ?`, before.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// MarshalNewsItems 序列化条目列表(供 handler 使用,统一容错)。
func MarshalNewsItems(items []NewsItem) string {
	if len(items) == 0 {
		return "[]"
	}
	b, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// UnmarshalNewsItems 反序列化条目列表(供 handler/前端二次处理,统一容错)。
func UnmarshalNewsItems(s string) ([]NewsItem, error) {
	if s == "" {
		return []NewsItem{}, nil
	}
	var items []NewsItem
	if err := json.Unmarshal([]byte(s), &items); err != nil {
		return nil, fmt.Errorf("解析新闻条目失败: %w", err)
	}
	return items, nil
}
