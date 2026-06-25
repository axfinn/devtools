package models

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// plannerRecordingDir 会议录音存放目录，与 handlers/planner_meeting.go 中的常量保持一致
const plannerRecordingDir = "./data/planner_recordings"

// plannerSafeRecordingFilename 从 RecordingURL 中提取安全的文件名（防目录穿越）
// 仅允许形如 "meeting_xxx_<digits>.<ext>" 的文件名；其他情况返回空字符串，避免误删。
func plannerSafeRecordingFilename(recordingURL string) string {
	recordingURL = strings.TrimSpace(recordingURL)
	if recordingURL == "" {
		return ""
	}
	name := filepath.Base(recordingURL)
	if name == "." || name == "/" || name == ".." {
		return ""
	}
	if !strings.HasPrefix(name, "meeting_") {
		return ""
	}
	return name
}

// deletePlannerRecordingFile 删除单个录音文件，失败仅记日志不返回错误
func (db *DB) deletePlannerRecordingFile(recordingURL string) {
	filename := plannerSafeRecordingFilename(recordingURL)
	if filename == "" {
		return
	}
	_ = os.Remove(filepath.Join(plannerRecordingDir, filename))
}

// DeletePlannerMeeting 删除会议纪要并清理关联的录音文件
func (db *DB) DeletePlannerMeeting(id string) error {
	if meeting, err := db.GetPlannerMeeting(id); err != nil {
		log.Printf("planner: 查询会议 %s 用于清理录音失败: %v", id, err)
	} else if meeting != nil {
		db.deletePlannerRecordingFile(meeting.RecordingURL)
	}
	_, err := db.conn.Exec(`DELETE FROM planner_meeting_minutes WHERE id = ?`, id)
	return err
}

// DeletePlannerMeetingsRecordingFiles 批量删除指定档案下所有会议的录音文件（档案删除前的清理）
func (db *DB) DeletePlannerMeetingsRecordingFiles(profileID string) {
	rows, err := db.conn.Query(`SELECT recording_url FROM planner_meeting_minutes WHERE profile_id = ?`, profileID)
	if err != nil {
		log.Printf("planner: 查询档案 %s 录音URL失败: %v", profileID, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var url string
		if scanErr := rows.Scan(&url); scanErr == nil {
			db.deletePlannerRecordingFile(url)
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("planner: 遍历档案 %s 录音URL时出错: %v", profileID, err)
	}
}

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
