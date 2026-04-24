package models

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"
)

func init() {
	RegisterInit("PhotoWall(photowall_profiles)", (*DB).InitPhotoWall)
}

const photoWallUploadDir = "./data/photowall"

type PhotoWallProfile struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Password      string     `json:"-"`
	PasswordIndex string     `json:"-"`
	CreatorKey    string     `json:"-"`
	AccessKey     string     `json:"-"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatorIP     string     `json:"-"`
	ShortCode     string     `json:"short_code"`
	IsPermanent   bool       `json:"is_permanent"`
	ItemCount     int        `json:"item_count"`
}

type PhotoWallItem struct {
	ID          string     `json:"id"`
	ProfileID   string     `json:"profile_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Category    string     `json:"category"`
	TakenAt     *time.Time `json:"taken_at"`
	ImageURL    string     `json:"image_url"`
	Filename    string     `json:"filename"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (db *DB) InitPhotoWall() error {
	query := `
	CREATE TABLE IF NOT EXISTS photowall_profiles (
		id TEXT PRIMARY KEY,
		title TEXT DEFAULT '',
		password TEXT NOT NULL,
		password_index TEXT NOT NULL DEFAULT '',
		creator_key TEXT NOT NULL,
		access_key TEXT NOT NULL,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT,
		short_code TEXT DEFAULT '',
		is_permanent BOOLEAN DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_photowall_profiles_expires_at ON photowall_profiles(expires_at);
	CREATE INDEX IF NOT EXISTS idx_photowall_profiles_creator_ip ON photowall_profiles(creator_ip);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_photowall_profiles_password_index ON photowall_profiles(password_index) WHERE password_index != '';
	CREATE INDEX IF NOT EXISTS idx_photowall_profiles_short_code ON photowall_profiles(short_code);

	CREATE TABLE IF NOT EXISTS photowall_items (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		title TEXT DEFAULT '',
		description TEXT DEFAULT '',
		category TEXT DEFAULT '',
		taken_at DATETIME,
		image_url TEXT NOT NULL,
		filename TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES photowall_profiles(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_photowall_items_profile_id ON photowall_items(profile_id);
	CREATE INDEX IF NOT EXISTS idx_photowall_items_taken_at ON photowall_items(taken_at);
	CREATE INDEX IF NOT EXISTS idx_photowall_items_category ON photowall_items(category);
	`
	db.conn.Exec("ALTER TABLE photowall_profiles ADD COLUMN short_code TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE photowall_profiles ADD COLUMN is_permanent BOOLEAN DEFAULT 0")
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreatePhotoWallProfile(profile *PhotoWallProfile) error {
	profile.ID = generateID(8)
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = profile.CreatedAt

	_, err := db.conn.Exec(`
		INSERT INTO photowall_profiles (
			id, title, password, password_index, creator_key, access_key, expires_at,
			created_at, updated_at, creator_ip, short_code, is_permanent
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Title, profile.Password, profile.PasswordIndex, profile.CreatorKey, profile.AccessKey,
		profile.ExpiresAt, profile.CreatedAt, profile.UpdatedAt, profile.CreatorIP, profile.ShortCode, profile.IsPermanent)
	return err
}

func (db *DB) GetPhotoWallProfile(id string) (*PhotoWallProfile, error) {
	profile := &PhotoWallProfile{}
	var expiresAt sql.NullTime
	var shortCode sql.NullString

	err := db.conn.QueryRow(`
		SELECT p.id, p.title, p.password, p.password_index, p.creator_key, p.access_key, p.expires_at,
		       p.created_at, p.updated_at, p.creator_ip, COALESCE(p.short_code, ''), p.is_permanent,
		       (SELECT COUNT(*) FROM photowall_items i WHERE i.profile_id = p.id)
		FROM photowall_profiles p
		WHERE p.id = ?
	`, id).Scan(
		&profile.ID, &profile.Title, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey, &profile.AccessKey,
		&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP, &shortCode, &profile.IsPermanent, &profile.ItemCount,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}
	profile.ShortCode = shortCode.String
	return profile, nil
}

func (db *DB) GetPhotoWallProfileByPasswordIndex(passwordIndex string) (*PhotoWallProfile, error) {
	profile := &PhotoWallProfile{}
	var expiresAt sql.NullTime
	var shortCode sql.NullString

	err := db.conn.QueryRow(`
		SELECT p.id, p.title, p.password, p.password_index, p.creator_key, p.access_key, p.expires_at,
		       p.created_at, p.updated_at, p.creator_ip, COALESCE(p.short_code, ''), p.is_permanent,
		       (SELECT COUNT(*) FROM photowall_items i WHERE i.profile_id = p.id)
		FROM photowall_profiles p
		WHERE p.password_index = ?
	`, passwordIndex).Scan(
		&profile.ID, &profile.Title, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey, &profile.AccessKey,
		&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP, &shortCode, &profile.IsPermanent, &profile.ItemCount,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}
	profile.ShortCode = shortCode.String
	return profile, nil
}

func (db *DB) UpdatePhotoWallProfile(profile *PhotoWallProfile) error {
	profile.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE photowall_profiles
		SET title = ?, access_key = ?, expires_at = ?, updated_at = ?, short_code = ?, is_permanent = ?
		WHERE id = ?
	`, profile.Title, profile.AccessKey, profile.ExpiresAt, profile.UpdatedAt, profile.ShortCode, profile.IsPermanent, profile.ID)
	return err
}

func (db *DB) CreatePhotoWallItem(item *PhotoWallItem) error {
	item.ID = generateID(8)
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt

	_, err := db.conn.Exec(`
		INSERT INTO photowall_items (
			id, profile_id, title, description, category, taken_at, image_url, filename, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ProfileID, item.Title, item.Description, item.Category, item.TakenAt, item.ImageURL, item.Filename, item.CreatedAt, item.UpdatedAt)
	return err
}

func (db *DB) ListPhotoWallItems(profileID string) ([]*PhotoWallItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, profile_id, title, description, category, taken_at, image_url, filename, created_at, updated_at
		FROM photowall_items
		WHERE profile_id = ?
		ORDER BY COALESCE(taken_at, created_at) DESC, created_at DESC
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*PhotoWallItem
	for rows.Next() {
		item := &PhotoWallItem{}
		var takenAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.ProfileID, &item.Title, &item.Description, &item.Category, &takenAt,
			&item.ImageURL, &item.Filename, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if takenAt.Valid {
			item.TakenAt = &takenAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) GetPhotoWallItem(profileID, itemID string) (*PhotoWallItem, error) {
	item := &PhotoWallItem{}
	var takenAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT id, profile_id, title, description, category, taken_at, image_url, filename, created_at, updated_at
		FROM photowall_items
		WHERE profile_id = ? AND id = ?
	`, profileID, itemID).Scan(
		&item.ID, &item.ProfileID, &item.Title, &item.Description, &item.Category, &takenAt,
		&item.ImageURL, &item.Filename, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if takenAt.Valid {
		item.TakenAt = &takenAt.Time
	}
	return item, nil
}

func (db *DB) UpdatePhotoWallItem(item *PhotoWallItem) error {
	item.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE photowall_items
		SET title = ?, description = ?, category = ?, taken_at = ?, updated_at = ?
		WHERE profile_id = ? AND id = ?
	`, item.Title, item.Description, item.Category, item.TakenAt, item.UpdatedAt, item.ProfileID, item.ID)
	return err
}

func (db *DB) DeletePhotoWallItem(profileID, itemID string) (string, error) {
	item, err := db.GetPhotoWallItem(profileID, itemID)
	if err != nil {
		return "", err
	}
	_, err = db.conn.Exec("DELETE FROM photowall_items WHERE profile_id = ? AND id = ?", profileID, itemID)
	if err != nil {
		return "", err
	}
	return item.Filename, nil
}

func (db *DB) CountPhotoWallProfilesByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM photowall_profiles WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

func (db *DB) ListAllPhotoWallProfiles(limit int) ([]*PhotoWallProfile, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.conn.Query(`
		SELECT p.id, p.title, p.password, p.password_index, p.creator_key, p.access_key, p.expires_at,
		       p.created_at, p.updated_at, p.creator_ip, COALESCE(p.short_code, ''), p.is_permanent,
		       (SELECT COUNT(*) FROM photowall_items i WHERE i.profile_id = p.id)
		FROM photowall_profiles p
		ORDER BY p.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*PhotoWallProfile
	for rows.Next() {
		profile := &PhotoWallProfile{}
		var expiresAt sql.NullTime
		var shortCode sql.NullString
		if err := rows.Scan(
			&profile.ID, &profile.Title, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey, &profile.AccessKey,
			&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP, &shortCode, &profile.IsPermanent, &profile.ItemCount,
		); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			profile.ExpiresAt = &expiresAt.Time
		}
		profile.ShortCode = shortCode.String
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (db *DB) DeletePhotoWallProfile(id string) error {
	return db.deletePhotoWallProfileWithAssets(id)
}

func (db *DB) deletePhotoWallProfileWithAssets(id string) error {
	rows, err := db.conn.Query("SELECT filename FROM photowall_items WHERE profile_id = ?", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var filename string
			if scanErr := rows.Scan(&filename); scanErr == nil && filename != "" {
				_ = os.Remove(filepath.Join(photoWallUploadDir, filename))
			}
		}
	}

	profile, _ := db.GetPhotoWallProfile(id)
	if profile != nil && profile.ShortCode != "" {
		_ = db.DeleteShortURL(profile.ShortCode)
	}

	_, _ = db.conn.Exec("DELETE FROM photowall_items WHERE profile_id = ?", id)
	_, err = db.conn.Exec("DELETE FROM photowall_profiles WHERE id = ?", id)
	_ = os.RemoveAll(filepath.Join(photoWallUploadDir, id))
	return err
}

func (db *DB) CleanExpiredPhotoWallProfiles() (int64, error) {
	rows, err := db.conn.Query(`
		SELECT id
		FROM photowall_profiles
		WHERE expires_at IS NOT NULL AND expires_at < ? AND is_permanent = 0
	`, time.Now())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var count int64
	for _, id := range ids {
		if err := db.deletePhotoWallProfileWithAssets(id); err == nil {
			count++
		}
	}
	return count, nil
}
