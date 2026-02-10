package models

import (
	"database/sql"
	"time"
)

type PregnancyProfile struct {
	ID            string     `json:"id"`
	EDD           string     `json:"edd"`
	Password      string     `json:"-"`
	PasswordIndex string     `json:"-"` // SHA256 hash for fast lookup
	CreatorKey    string     `json:"-"`
	Data          string     `json:"data"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatorIP     string     `json:"-"`
}

func (db *DB) InitPregnancy() error {
	query := `
	CREATE TABLE IF NOT EXISTS pregnancy_profiles (
		id TEXT PRIMARY KEY,
		edd TEXT NOT NULL,
		password TEXT NOT NULL,
		password_index TEXT NOT NULL DEFAULT '',
		creator_key TEXT NOT NULL,
		data TEXT NOT NULL DEFAULT '{}',
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_pregnancy_expires_at ON pregnancy_profiles(expires_at);
	CREATE INDEX IF NOT EXISTS idx_pregnancy_creator_ip ON pregnancy_profiles(creator_ip);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_pregnancy_password_index ON pregnancy_profiles(password_index) WHERE password_index != '';
	`
	// Auto-migrate: add password_index column if missing
	db.conn.Exec("ALTER TABLE pregnancy_profiles ADD COLUMN password_index TEXT NOT NULL DEFAULT ''")
	db.conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_pregnancy_password_index ON pregnancy_profiles(password_index) WHERE password_index != ''")
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreatePregnancyProfile(profile *PregnancyProfile) error {
	profile.ID = generateID(8)
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO pregnancy_profiles (id, edd, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.EDD, profile.Password, profile.PasswordIndex, profile.CreatorKey, profile.Data,
		profile.ExpiresAt, profile.CreatedAt, profile.UpdatedAt, profile.CreatorIP)

	return err
}

func (db *DB) GetPregnancyProfile(id string) (*PregnancyProfile, error) {
	profile := &PregnancyProfile{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, edd, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip
		FROM pregnancy_profiles WHERE id = ?
	`, id).Scan(
		&profile.ID, &profile.EDD, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey, &profile.Data,
		&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}

	return profile, nil
}

func (db *DB) GetPregnancyProfileByPasswordIndex(passwordIndex string) (*PregnancyProfile, error) {
	profile := &PregnancyProfile{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, edd, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip
		FROM pregnancy_profiles WHERE password_index = ?
	`, passwordIndex).Scan(
		&profile.ID, &profile.EDD, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey, &profile.Data,
		&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}

	return profile, nil
}

func (db *DB) UpdatePregnancyProfileData(id, data string) error {
	_, err := db.conn.Exec(
		"UPDATE pregnancy_profiles SET data = ?, updated_at = ? WHERE id = ?",
		data, time.Now(), id)
	return err
}

func (db *DB) UpdatePregnancyProfileEDD(id, edd string) error {
	_, err := db.conn.Exec(
		"UPDATE pregnancy_profiles SET edd = ?, updated_at = ? WHERE id = ?",
		edd, time.Now(), id)
	return err
}

func (db *DB) ExtendPregnancyProfile(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(
		"UPDATE pregnancy_profiles SET expires_at = ?, updated_at = ? WHERE id = ?",
		expiresAt, time.Now(), id)
	return err
}

func (db *DB) DeletePregnancyProfile(id string) error {
	_, err := db.conn.Exec("DELETE FROM pregnancy_profiles WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpiredPregnancyProfiles() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM pregnancy_profiles
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) GetPregnancyProfilesWithoutIndex() ([]*PregnancyProfile, error) {
	rows, err := db.conn.Query(`
		SELECT id, edd, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip
		FROM pregnancy_profiles WHERE password_index = '' OR password_index IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*PregnancyProfile
	for rows.Next() {
		p := &PregnancyProfile{}
		var expiresAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.EDD, &p.Password, &p.PasswordIndex, &p.CreatorKey, &p.Data,
			&expiresAt, &p.CreatedAt, &p.UpdatedAt, &p.CreatorIP); err != nil {
			continue
		}
		if expiresAt.Valid {
			p.ExpiresAt = &expiresAt.Time
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (db *DB) UpdatePregnancyProfilePasswordIndex(id, passwordIndex string) error {
	_, err := db.conn.Exec(
		"UPDATE pregnancy_profiles SET password_index = ? WHERE id = ?",
		passwordIndex, id)
	return err
}

func (db *DB) CountPregnancyProfilesByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM pregnancy_profiles WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}
