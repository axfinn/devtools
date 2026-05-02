package models

import (
	"database/sql"
	"time"
)

// VoiceMemo represents a voice recording with transcription.
type VoiceMemo struct {
	ID            string     `json:"id"`
	DeviceID      string     `json:"device_id"`
	ProfileID     string     `json:"profile_id,omitempty"`
	Title         string     `json:"title"`
	AudioURL      string     `json:"audio_url"`
	Transcript    string     `json:"transcript"`
	Language      string     `json:"language"`
	DurationSec   float64    `json:"duration_sec"`
	FileSize      int64      `json:"file_size"`
	Status        string     `json:"status"` // "draft", "transcribing", "completed", "saved", "failed"
	ErrorMessage  string     `json:"error_message,omitempty"`
	PlannerTaskID string     `json:"planner_task_id,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

const voicememoSchema = `
CREATE TABLE IF NOT EXISTS voice_memos (
	id TEXT PRIMARY KEY,
	device_id TEXT NOT NULL DEFAULT '',
	title TEXT DEFAULT '',
	audio_url TEXT DEFAULT '',
	transcript TEXT DEFAULT '',
	language TEXT DEFAULT 'zh',
	duration_sec REAL DEFAULT 0,
	file_size INTEGER DEFAULT 0,
	status TEXT DEFAULT 'draft',
	error_message TEXT DEFAULT '',
	planner_profile_id TEXT DEFAULT '',
	planner_task_id TEXT DEFAULT '',
	expires_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	deleted_at DATETIME
);
`

var voicememoIndexes = []string{
	"CREATE INDEX IF NOT EXISTS idx_voice_memos_device ON voice_memos(device_id, created_at DESC)",
	"CREATE INDEX IF NOT EXISTS idx_voice_memos_profile ON voice_memos(planner_profile_id, created_at DESC)",
	"CREATE INDEX IF NOT EXISTS idx_voice_memos_deleted ON voice_memos(deleted_at)",
	"CREATE INDEX IF NOT EXISTS idx_voice_memos_expires ON voice_memos(expires_at)",
	"CREATE INDEX IF NOT EXISTS idx_voice_memos_planner_task ON voice_memos(planner_task_id)",
}

func init() {
	RegisterInit("语音备忘录(voice_memos)", func(db *DB) error {
		return db.initVoiceMemo()
	})
}

func (db *DB) initVoiceMemo() error {
	_, err := db.conn.Exec(voicememoSchema)
	if err != nil {
		return err
	}

	var expiresAtColumnCount int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('voice_memos') WHERE name='expires_at'").Scan(&expiresAtColumnCount); err != nil {
		return err
	}
	if expiresAtColumnCount == 0 {
		if _, err := db.conn.Exec("ALTER TABLE voice_memos ADD COLUMN expires_at DATETIME"); err != nil {
			return err
		}
	}
	var plannerProfileColumnCount int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('voice_memos') WHERE name='planner_profile_id'").Scan(&plannerProfileColumnCount); err != nil {
		return err
	}
	if plannerProfileColumnCount == 0 {
		if _, err := db.conn.Exec("ALTER TABLE voice_memos ADD COLUMN planner_profile_id TEXT DEFAULT ''"); err != nil {
			return err
		}
	}
	var plannerTaskColumnCount int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('voice_memos') WHERE name='planner_task_id'").Scan(&plannerTaskColumnCount); err != nil {
		return err
	}
	if plannerTaskColumnCount == 0 {
		if _, err := db.conn.Exec("ALTER TABLE voice_memos ADD COLUMN planner_task_id TEXT DEFAULT ''"); err != nil {
			return err
		}
	}

	for _, stmt := range voicememoIndexes {
		if _, err := db.conn.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

// CreateVoiceMemo inserts a new voice memo draft (default 14-day expiry).
func (db *DB) CreateVoiceMemo(m *VoiceMemo) error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	if m.Status == "" {
		m.Status = "draft"
	}
	if m.ExpiresAt == nil {
		exp := time.Now().Add(14 * 24 * time.Hour)
		m.ExpiresAt = &exp
	}
	_, err := db.conn.Exec(
		`INSERT INTO voice_memos (id, device_id, title, audio_url, transcript, language, duration_sec, file_size, status, error_message, planner_profile_id, planner_task_id, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.DeviceID, m.Title, m.AudioURL, m.Transcript, m.Language,
		m.DurationSec, m.FileSize, m.Status, m.ErrorMessage, m.ProfileID, m.PlannerTaskID, m.ExpiresAt, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// GetVoiceMemo retrieves a voice memo by ID (not soft-deleted, not expired).
func (db *DB) GetVoiceMemo(id string) (*VoiceMemo, error) {
	m := &VoiceMemo{}
	var errorMsg sql.NullString
	var expiresAt sql.NullTime
	err := db.conn.QueryRow(
		`SELECT id, device_id, title, audio_url, transcript, language, duration_sec, file_size, status, error_message, planner_profile_id, planner_task_id, expires_at, created_at, updated_at
		 FROM voice_memos WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(&m.ID, &m.DeviceID, &m.Title, &m.AudioURL, &m.Transcript, &m.Language,
		&m.DurationSec, &m.FileSize, &m.Status, &errorMsg, &m.ProfileID, &m.PlannerTaskID, &expiresAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.ErrorMessage = errorMsg.String
	if expiresAt.Valid {
		m.ExpiresAt = &expiresAt.Time
	}
	return m, nil
}

// ListVoiceMemos returns voice memos for a profile, while still exposing legacy device-only memos for migration.
func (db *DB) ListVoiceMemos(profileID, deviceID string, limit, offset int) ([]VoiceMemo, int, error) {
	var total int
	if profileID != "" {
		if err := db.conn.QueryRow(
			`SELECT COUNT(*) FROM voice_memos
			 WHERE deleted_at IS NULL
			   AND (planner_profile_id = ? OR (planner_profile_id = '' AND device_id = ?))`,
			profileID, deviceID,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		if err := db.conn.QueryRow(
			`SELECT COUNT(*) FROM voice_memos WHERE device_id = ? AND deleted_at IS NULL`, deviceID,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	var rows *sql.Rows
	var err error
	if profileID != "" {
		rows, err = db.conn.Query(
			`SELECT id, device_id, title, audio_url, transcript, language, duration_sec, file_size, status, error_message, planner_profile_id, planner_task_id, expires_at, created_at, updated_at
			 FROM voice_memos
			 WHERE deleted_at IS NULL
			   AND (planner_profile_id = ? OR (planner_profile_id = '' AND device_id = ?))
			 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			profileID, deviceID, limit, offset,
		)
	} else {
		rows, err = db.conn.Query(
			`SELECT id, device_id, title, audio_url, transcript, language, duration_sec, file_size, status, error_message, planner_profile_id, planner_task_id, expires_at, created_at, updated_at
			 FROM voice_memos WHERE device_id = ? AND deleted_at IS NULL
			 ORDER BY created_at DESC LIMIT ? OFFSET ?`, deviceID, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var memos []VoiceMemo
	for rows.Next() {
		var m VoiceMemo
		var errorMsg sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.DeviceID, &m.Title, &m.AudioURL, &m.Transcript, &m.Language,
			&m.DurationSec, &m.FileSize, &m.Status, &errorMsg, &m.ProfileID, &m.PlannerTaskID, &expiresAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, err
		}
		m.ErrorMessage = errorMsg.String
		if expiresAt.Valid {
			m.ExpiresAt = &expiresAt.Time
		}
		memos = append(memos, m)
	}
	if memos == nil {
		memos = []VoiceMemo{}
	}
	return memos, total, nil
}

// UpdateVoiceMemoTranscript updates transcript after ASR completes.
func (db *DB) UpdateVoiceMemoTranscript(id, transcript, language, status, errorMessage string) error {
	_, err := db.conn.Exec(
		`UPDATE voice_memos SET transcript = ?, language = ?, status = ?, error_message = ?, updated_at = ? WHERE id = ?`,
		transcript, language, status, errorMessage, time.Now(), id,
	)
	return err
}

// UpdateVoiceMemo updates title, transcript, status, and optionally clears expiration.
func (db *DB) UpdateVoiceMemo(id, title, transcript, status string) error {
	var expiresAt *time.Time
	if status == "saved" {
		expiresAt = nil // saved memos never expire
	}
	_, err := db.conn.Exec(
		`UPDATE voice_memos SET title = ?, transcript = ?, status = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
		title, transcript, status, expiresAt, time.Now(), id,
	)
	return err
}

// LinkVoiceMemoToPlanner links a memo to a planner task and saves it permanently.
func (db *DB) LinkVoiceMemoToPlanner(id, title, transcript, profileID, taskID string) error {
	_, err := db.conn.Exec(
		`UPDATE voice_memos
		 SET title = ?, transcript = ?, status = 'saved', planner_profile_id = ?, planner_task_id = ?, expires_at = NULL, updated_at = ?
		 WHERE id = ?`,
		title, transcript, profileID, taskID, time.Now(), id,
	)
	return err
}

func (db *DB) BindVoiceMemoProfile(id, profileID string) error {
	_, err := db.conn.Exec(
		`UPDATE voice_memos
		 SET planner_profile_id = CASE WHEN planner_profile_id = '' THEN ? ELSE planner_profile_id END,
		     updated_at = ?
		 WHERE id = ?`,
		profileID, time.Now(), id,
	)
	return err
}

// DeleteVoiceMemo soft-deletes a voice memo.
func (db *DB) DeleteVoiceMemo(id string) error {
	now := time.Now()
	_, err := db.conn.Exec(`UPDATE voice_memos SET deleted_at = ? WHERE id = ?`, now, id)
	return err
}

// CleanExpiredVoiceMemos hard-deletes:
// 1. Draft memos past their expires_at (14 days of inactivity)
// 2. Soft-deleted memos older than 7 days
func (db *DB) CleanExpiredVoiceMemos() (int64, error) {
	result, err := db.conn.Exec(
		`DELETE FROM voice_memos WHERE
			(deleted_at IS NOT NULL AND deleted_at < ?)
			OR
			(expires_at IS NOT NULL AND expires_at < ? AND status = 'draft' AND deleted_at IS NULL)`,
		time.Now().Add(-7*24*time.Hour),
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
