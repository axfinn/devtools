package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
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
	Files       string    `json:"files"` // JSON array of file metadata [{filename, type, size, url}]
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
		content TEXT DEFAULT '',
		title TEXT DEFAULT '',
		language TEXT DEFAULT 'text',
		password TEXT DEFAULT '',
		expires_at DATETIME,
		max_views INTEGER DEFAULT 100,
		views INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT,
		files TEXT DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_expires_at ON pastes(expires_at);
	CREATE INDEX IF NOT EXISTS idx_creator_ip ON pastes(creator_ip);
	`
	_, err := db.conn.Exec(query)
	if err != nil {
		return err
	}
	// 添加 files 列（如果不存在）
	db.conn.Exec("ALTER TABLE pastes ADD COLUMN files TEXT DEFAULT ''")
	return nil
}

func (db *DB) CreatePaste(paste *Paste) error {
	paste.ID = generateID(8)
	paste.CreatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO pastes (id, content, title, language, password, expires_at, max_views, views, created_at, creator_ip, files)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, paste.ID, paste.Content, paste.Title, paste.Language, paste.Password,
		paste.ExpiresAt, paste.MaxViews, paste.Views, paste.CreatedAt, paste.CreatorIP, paste.Files)

	return err
}

func (db *DB) GetPaste(id string) (*Paste, error) {
	paste := &Paste{}
	var files sql.NullString
	err := db.conn.QueryRow(`
		SELECT id, content, title, language, password, expires_at, max_views, views, created_at, creator_ip, COALESCE(files, '')
		FROM pastes WHERE id = ?
	`, id).Scan(
		&paste.ID, &paste.Content, &paste.Title, &paste.Language, &paste.Password,
		&paste.ExpiresAt, &paste.MaxViews, &paste.Views, &paste.CreatedAt, &paste.CreatorIP, &files)

	if err != nil {
		return nil, err
	}
	paste.Files = files.String

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
	// 先获取过期的 paste 及其文件列表
	rows, err := db.conn.Query("SELECT id, files FROM pastes WHERE expires_at < ?", time.Now())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type fileMetadata struct {
		Filename string `json:"filename"`
	}

	var filesToDelete []string
	var idsToDelete []string

	for rows.Next() {
		var id string
		var filesJSON sql.NullString
		if err := rows.Scan(&id, &filesJSON); err != nil {
			continue
		}

		idsToDelete = append(idsToDelete, id)

		// 解析文件列表
		if filesJSON.Valid && filesJSON.String != "" {
			var files []fileMetadata
			if err := json.Unmarshal([]byte(filesJSON.String), &files); err == nil {
				for _, f := range files {
					filesToDelete = append(filesToDelete, f.Filename)
				}
			}
		}
	}

	// 删除数据库记录
	result, err := db.conn.Exec("DELETE FROM pastes WHERE expires_at < ?", time.Now())
	if err != nil {
		return 0, err
	}

	// 删除文件
	for _, filename := range filesToDelete {
		filePath := filepath.Join("./data/paste_files", filename)
		os.Remove(filePath) // 忽略错误，文件可能已不存在
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
