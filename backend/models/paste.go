package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Paste struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Title       string    `json:"title"`
	Language    string    `json:"language"`
	Password    string    `json:"-"`
	ExpiresAt   time.Time `json:"expires_at"`
	MaxViews    int       `json:"max_views"`
	Views       int       `json:"views"`
	CreatedAt   time.Time `json:"created_at"`
	CreatorIP   string    `json:"-"`
}

type DB struct {
	conn *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS pastes (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		title TEXT DEFAULT '',
		language TEXT DEFAULT 'text',
		password TEXT DEFAULT '',
		expires_at DATETIME,
		max_views INTEGER DEFAULT 100,
		views INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON pastes(expires_at);
	CREATE INDEX IF NOT EXISTS idx_creator_ip ON pastes(creator_ip);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreatePaste(paste *Paste) error {
	paste.ID = generateID(8)
	paste.CreatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO pastes (id, content, title, language, password, expires_at, max_views, views, created_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, paste.ID, paste.Content, paste.Title, paste.Language, paste.Password,
		paste.ExpiresAt, paste.MaxViews, paste.Views, paste.CreatedAt, paste.CreatorIP)

	return err
}

func (db *DB) GetPaste(id string) (*Paste, error) {
	paste := &Paste{}
	err := db.conn.QueryRow(`
		SELECT id, content, title, language, password, expires_at, max_views, views, created_at, creator_ip
		FROM pastes WHERE id = ?
	`, id).Scan(
		&paste.ID, &paste.Content, &paste.Title, &paste.Language, &paste.Password,
		&paste.ExpiresAt, &paste.MaxViews, &paste.Views, &paste.CreatedAt, &paste.CreatorIP)

	if err != nil {
		return nil, err
	}

	return paste, nil
}

func (db *DB) IncrementViews(id string) error {
	_, err := db.conn.Exec("UPDATE pastes SET views = views + 1 WHERE id = ?", id)
	return err
}

func (db *DB) DeletePaste(id string) error {
	_, err := db.conn.Exec("DELETE FROM pastes WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpired() (int64, error) {
	result, err := db.conn.Exec("DELETE FROM pastes WHERE expires_at < ?", time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) CountByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM pastes WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

func (db *DB) TotalCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM pastes").Scan(&count)
	return count, err
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func generateID(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
