package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

type ShortURL struct {
	ID          string     `json:"id"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxClicks   int        `json:"max_clicks"`
	Clicks      int        `json:"clicks"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatorIP   string     `json:"creator_ip"`
	MDShareID   string     `json:"mdshare_id,omitempty"` // Link to markdown share
}

// InitShortURL creates the short_urls table if it doesn't exist
func (db *DB) InitShortURL() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS short_urls (
			id TEXT PRIMARY KEY,
			original_url TEXT NOT NULL,
			expires_at DATETIME,
			max_clicks INTEGER DEFAULT 1000,
			clicks INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			creator_ip TEXT
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_shorturls_expires_at ON short_urls(expires_at)`)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_shorturls_creator_ip ON short_urls(creator_ip)`)
	return err
}

// generateID generates a random 8-character hex ID
func generateShortURLID() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateShortURL creates a new short URL (for backward compatibility)
func (db *DB) CreateShortURL(originalURL string, expiresIn int, maxClicks int, creatorIP string) (*ShortURL, error) {
	return db.CreateShortURLWithCustomID(originalURL, "", expiresIn, maxClicks, creatorIP)
}

// CreateShortURLWithCustomID creates a new short URL with optional custom ID
func (db *DB) CreateShortURLWithCustomID(originalURL string, customID string, expiresIn int, maxClicks int, creatorIP string) (*ShortURL, error) {
	// Validate expires_in (max 1 year = 8760 hours)
	if expiresIn < 1 {
		expiresIn = 720 // default 30 days
	}
	if expiresIn > 8760 {
		expiresIn = 8760 // max 1 year
	}

	// Validate max_clicks
	if maxClicks < 1 {
		maxClicks = 1000
	}
	if maxClicks > 100000 {
		maxClicks = 100000
	}

	// Check if storage limit reached, clean expired if needed
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM short_urls").Scan(&count)
	if err != nil {
		return nil, err
	}

	if count >= 50000 {
		// Try to clean expired entries first
		db.CleanExpiredShortURLs()

		// Check again
		err = db.conn.QueryRow("SELECT COUNT(*) FROM short_urls").Scan(&count)
		if err != nil {
			return nil, err
		}
		if count >= 50000 {
			return nil, errors.New("storage limit reached")
		}
	}

	var id string
	if customID != "" {
		// 使用自定义ID，检查是否已存在
		var exists bool
		err = db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM short_urls WHERE id = ?)", customID).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("ID already exists")
		}
		id = customID
	} else {
		// Generate unique ID
		for i := 0; i < 10; i++ {
			id, err = generateShortURLID()
			if err != nil {
				return nil, err
			}

			// Check if ID already exists
			var exists bool
			err = db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM short_urls WHERE id = ?)", id).Scan(&exists)
			if err != nil {
				return nil, err
			}
			if !exists {
				break
			}
		}
	}

	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Hour)

	_, err = db.conn.Exec(`
		INSERT INTO short_urls (id, original_url, expires_at, max_clicks, creator_ip)
		VALUES (?, ?, ?, ?, ?)`,
		id, originalURL, expiresAt, maxClicks, creatorIP)

	if err != nil {
		return nil, err
	}

	return &ShortURL{
		ID:          id,
		OriginalURL: originalURL,
		ExpiresAt:   &expiresAt,
		MaxClicks:   maxClicks,
		Clicks:      0,
		CreatedAt:   time.Now(),
		CreatorIP:   creatorIP,
	}, nil
}

// GetShortURL retrieves a short URL by ID
func (db *DB) GetShortURL(id string) (*ShortURL, error) {
	var shortURL ShortURL
	err := db.conn.QueryRow(`
		SELECT id, original_url, expires_at, max_clicks, clicks, created_at, creator_ip
		FROM short_urls
		WHERE id = ?`, id).Scan(
		&shortURL.ID,
		&shortURL.OriginalURL,
		&shortURL.ExpiresAt,
		&shortURL.MaxClicks,
		&shortURL.Clicks,
		&shortURL.CreatedAt,
		&shortURL.CreatorIP,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("short URL not found")
	}
	if err != nil {
		return nil, err
	}

	return &shortURL, nil
}

// IncrementClicks atomically increments the click count
func (db *DB) IncrementClicks(id string) error {
	result, err := db.conn.Exec("UPDATE short_urls SET clicks = clicks + 1 WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("short URL not found")
	}

	return nil
}

// DeleteShortURL deletes a short URL by ID
func (db *DB) DeleteShortURL(id string) error {
	_, err := db.conn.Exec("DELETE FROM short_urls WHERE id = ?", id)
	return err
}

// CleanExpiredShortURLs removes expired short URLs
func (db *DB) CleanExpiredShortURLs() error {
	_, err := db.conn.Exec("DELETE FROM short_urls WHERE expires_at < ?", time.Now())
	return err
}

// ListShortURLs returns all short URLs ordered by created_at desc
func (db *DB) ListShortURLs() ([]ShortURL, error) {
	rows, err := db.conn.Query(`
		SELECT id, original_url, expires_at, max_clicks, clicks, created_at, creator_ip
		FROM short_urls
		ORDER BY created_at DESC
		LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []ShortURL
	for rows.Next() {
		var u ShortURL
		err := rows.Scan(&u.ID, &u.OriginalURL, &u.ExpiresAt, &u.MaxClicks, &u.Clicks, &u.CreatedAt, &u.CreatorIP)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

// CountShortURLsByIP counts short URLs created by IP in the last duration
func (db *DB) CountShortURLsByIP(ip string, duration time.Duration) (int, error) {
	var count int
	since := time.Now().Add(-duration)
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM short_urls WHERE creator_ip = ? AND created_at > ?",
		ip, since).Scan(&count)
	return count, err
}

// CreateShortURLFromStruct creates a new short URL from struct (for MDShare integration)
func (db *DB) CreateShortURLFromStruct(shortURL *ShortURL) error {
	// Generate ID if not provided
	if shortURL.ID == "" {
		var err error
		for i := 0; i < 10; i++ {
			shortURL.ID, err = generateShortURLID()
			if err != nil {
				return err
			}
			var exists bool
			db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM short_urls WHERE id = ?)", shortURL.ID).Scan(&exists)
			if !exists {
				break
			}
		}
	}

	shortURL.CreatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO short_urls (id, original_url, expires_at, max_clicks, clicks, created_at, creator_ip)
		VALUES (?, ?, ?, ?, 0, ?, ?)`,
		shortURL.ID, shortURL.OriginalURL, shortURL.ExpiresAt, shortURL.MaxClicks, shortURL.CreatedAt, shortURL.CreatorIP)

	return err
}

// UpdateShortURLExpires updates the expiration time of a short URL
func (db *DB) UpdateShortURLExpires(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec("UPDATE short_urls SET expires_at = ? WHERE id = ?", expiresAt, id)
	return err
}
