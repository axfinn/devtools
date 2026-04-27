package models

import (
	"strings"
	"time"
)

func init() {
	RegisterInit("会议纪要(planner_meeting_minutes)", (*DB).InitPlannerMeetings)
}

type PlannerMeetingMinutes struct {
	ID              string     `json:"id"`
	ProfileID       string     `json:"profile_id"`
	Title           string     `json:"title"`
	Content         string     `json:"content"`
	Summary         string     `json:"summary"`
	ActionItems     string     `json:"action_items"`
	Participants    string     `json:"participants"`
	RecordingURL    string     `json:"recording_url"`
	DurationMinutes int        `json:"duration_minutes"`
	MeetingDate     string     `json:"meeting_date"`
	MeetingTime     string     `json:"meeting_time"`
	Tags            string     `json:"tags"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (db *DB) InitPlannerMeetings() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS planner_meeting_minutes (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			action_items TEXT DEFAULT '[]',
			participants TEXT DEFAULT '[]',
			recording_url TEXT DEFAULT '',
			duration_minutes INTEGER DEFAULT 0,
			meeting_date TEXT NOT NULL,
			meeting_time TEXT DEFAULT '',
			tags TEXT DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES planner_profiles(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	migrations := []string{
		"ALTER TABLE planner_meeting_minutes ADD COLUMN tags TEXT DEFAULT '[]'",
	}
	for _, stmt := range migrations {
		if _, err := db.conn.Exec(stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_planner_meetings_profile_date ON planner_meeting_minutes(profile_id, meeting_date)",
	}
	for _, stmt := range indexes {
		if _, err := db.conn.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) CreatePlannerMeeting(m *PlannerMeetingMinutes) error {
	m.ID = generateID(8)
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	_, err := db.conn.Exec(`
		INSERT INTO planner_meeting_minutes
			(id, profile_id, title, content, summary, action_items, participants,
			 recording_url, duration_minutes, meeting_date, meeting_time, tags, status,
			 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.ProfileID, m.Title, m.Content, m.Summary, m.ActionItems, m.Participants,
		m.RecordingURL, m.DurationMinutes, m.MeetingDate, m.MeetingTime, m.Tags, m.Status,
		m.CreatedAt, m.UpdatedAt)
	return err
}

func scanPlannerMeeting(scanner interface{ Scan(dest ...interface{}) error }) (*PlannerMeetingMinutes, error) {
	m := &PlannerMeetingMinutes{}
	if err := scanner.Scan(
		&m.ID, &m.ProfileID, &m.Title, &m.Content, &m.Summary,
		&m.ActionItems, &m.Participants, &m.RecordingURL, &m.DurationMinutes,
		&m.MeetingDate, &m.MeetingTime, &m.Tags, &m.Status,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return m, nil
}

func (db *DB) GetPlannerMeeting(id string) (*PlannerMeetingMinutes, error) {
	row := db.conn.QueryRow(`
		SELECT id, profile_id, title, content, summary, action_items, participants,
		       recording_url, duration_minutes, meeting_date, meeting_time, tags, status,
		       created_at, updated_at
		FROM planner_meeting_minutes WHERE id = ?
	`, id)
	return scanPlannerMeeting(row)
}

func (db *DB) ListPlannerMeetings(profileID string, q, tag, dateFrom, dateTo string) ([]*PlannerMeetingMinutes, error) {
	query := `SELECT id, profile_id, title, content, summary, action_items, participants,
	       recording_url, duration_minutes, meeting_date, meeting_time, tags, status,
	       created_at, updated_at
	FROM planner_meeting_minutes
	WHERE profile_id = ?`
	args := []interface{}{profileID}

	if strings.TrimSpace(q) != "" {
		query += ` AND (title LIKE ? OR content LIKE ?)`
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like)
	}
	if strings.TrimSpace(tag) != "" {
		query += ` AND tags LIKE ?`
		args = append(args, `%`+strings.TrimSpace(tag)+`%`)
	}
	if strings.TrimSpace(dateFrom) != "" {
		query += ` AND meeting_date >= ?`
		args = append(args, strings.TrimSpace(dateFrom))
	}
	if strings.TrimSpace(dateTo) != "" {
		query += ` AND meeting_date <= ?`
		args = append(args, strings.TrimSpace(dateTo))
	}
	query += ` ORDER BY meeting_date DESC, created_at DESC`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerMeetingMinutes, 0)
	for rows.Next() {
		m, err := scanPlannerMeeting(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (db *DB) UpdatePlannerMeeting(m *PlannerMeetingMinutes) error {
	m.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE planner_meeting_minutes
		SET title = ?, content = ?, summary = ?, action_items = ?, participants = ?,
		    recording_url = ?, duration_minutes = ?, meeting_date = ?, meeting_time = ?,
		    tags = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, m.Title, m.Content, m.Summary, m.ActionItems, m.Participants,
		m.RecordingURL, m.DurationMinutes, m.MeetingDate, m.MeetingTime,
		m.Tags, m.Status, m.UpdatedAt, m.ID)
	return err
}

func (db *DB) DeletePlannerMeeting(id string) error {
	_, err := db.conn.Exec(`DELETE FROM planner_meeting_minutes WHERE id = ?`, id)
	return err
}
