package models

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitPlannerMigratesLegacySchema(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db failed: %v", err)
	}
	defer db.Close()

	_, err = db.conn.Exec(`
		DROP TABLE IF EXISTS planner_task_comments;
		DROP TABLE IF EXISTS planner_tasks;
		DROP TABLE IF EXISTS planner_profiles;

		CREATE TABLE planner_profiles (
			id TEXT PRIMARY KEY,
			password_index TEXT NOT NULL UNIQUE,
			creator_key TEXT NOT NULL,
			name TEXT NOT NULL,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE planner_tasks (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			detail TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
			planned_for TEXT NOT NULL,
			remind_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE planner_task_comments (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			profile_id TEXT NOT NULL,
			author TEXT NOT NULL DEFAULT '用户',
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("prepare legacy schema failed: %v", err)
	}

	if err := db.InitPlanner(); err != nil {
		t.Fatalf("InitPlanner should migrate legacy schema, got error: %v", err)
	}

	rows, err := db.conn.Query(`PRAGMA table_info(planner_tasks)`)
	if err != nil {
		t.Fatalf("inspect planner_tasks failed: %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var (
			cid       int
			name      string
			dataType  string
			notNull   int
			defaultV  interface{}
			primaryPK int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultV, &primaryPK); err != nil {
			t.Fatalf("scan column failed: %v", err)
		}
		columns[name] = true
	}

	required := []string{
		"entry_type",
		"bucket",
		"notes",
		"notify_email",
		"last_notified_at",
		"completed_at",
		"original_planned_for",
		"rollover_count",
		"last_postpone_reason",
		"last_postponed_at",
		"cancel_reason",
	}
	for _, column := range required {
		if !columns[column] {
			t.Fatalf("expected migrated column %q to exist", column)
		}
	}
}

func TestPlannerSafeRecordingFilename(t *testing.T) {
	cases := map[string]string{
		"":                                                  "",
		"/api/planner/recordings/meeting_abc123_1710000000.webm": "meeting_abc123_1710000000.webm",
		"meeting_xyz_1710000000000.mp3":                       "meeting_xyz_1710000000000.mp3",
		"/api/planner/recordings/../etc/passwd":               "",
		"/api/planner/recordings/secret.txt":                  "",
		"/api/planner/recordings/.hidden":                      "",
		"/api/planner/recordings/..":                          "",
		"   ":                                                "",
	}
	for input, want := range cases {
		if got := plannerSafeRecordingFilename(input); got != want {
			t.Errorf("plannerSafeRecordingFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDeletePlannerMeetingRemovesRecordingFile(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db failed: %v", err)
	}
	defer db.Close()

	if err := db.InitPlannerMeetings(); err != nil {
		t.Fatalf("init meetings failed: %v", err)
	}

	if err := os.MkdirAll(plannerRecordingDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	defer os.RemoveAll(plannerRecordingDir)

	profileID := "profile_test_1"
	meeting := &PlannerMeetingMinutes{
		ProfileID:    profileID,
		Title:        "测试会议",
		Content:      "测试内容",
		MeetingDate:  "2026-06-24",
		RecordingURL: "/api/planner/recordings/meeting_" + profileID + "_1710000000000.webm",
	}
	if err := db.CreatePlannerMeeting(meeting); err != nil {
		t.Fatalf("create meeting failed: %v", err)
	}

	filename := plannerSafeRecordingFilename(meeting.RecordingURL)
	filePath := filepath.Join(plannerRecordingDir, filename)
	if err := os.WriteFile(filePath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("write fake recording failed: %v", err)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("fake recording should exist before delete: %v", err)
	}

	if err := db.DeletePlannerMeeting(meeting.ID); err != nil {
		t.Fatalf("delete meeting failed: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("recording file should be removed, stat err = %v", err)
	}
}

func TestVoicememoSafeFilename(t *testing.T) {
	cases := map[string]string{
		"":                                                    "",
		"/api/voicememo/audio/memo_abc123_1710000000000000000.webm": "memo_abc123_1710000000000000000.webm",
		"memo_xyz_123.mp3":                                     "memo_xyz_123.mp3",
		"/api/voicememo/audio/../etc/passwd":                   "",
		"/api/voicememo/audio/secret.txt":                      "",
		"/api/voicememo/audio/.hidden":                         "",
		"   ":                                                  "",
	}
	for input, want := range cases {
		if got := voicememoSafeFilename(input); got != want {
			t.Errorf("voicememoSafeFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCleanExpiredVoiceMemosRemovesAudio(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db failed: %v", err)
	}
	defer db.Close()

	if err := db.InitAll(); err != nil {
		t.Fatalf("init all failed: %v", err)
	}

	if err := os.MkdirAll(VoicememoUploadDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	defer os.RemoveAll(VoicememoUploadDir)

	// 创建一个 draft 且已过期的备忘
	memoID := "memtest01"
	expired := time.Now().Add(-1 * time.Hour)
	memo := &VoiceMemo{
		ID:        memoID,
		DeviceID:  "dev1",
		Title:     "test",
		AudioURL:  "/api/voicememo/audio/memo_memtest01_1710000000000000000.webm",
		Status:    "draft",
		ExpiresAt: &expired,
	}
	if err := db.CreateVoiceMemo(memo); err != nil {
		t.Fatalf("create memo failed: %v", err)
	}

	filename := voicememoSafeFilename(memo.AudioURL)
	filePath := filepath.Join(VoicememoUploadDir, filename)
	if err := os.WriteFile(filePath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("write fake audio failed: %v", err)
	}

	deleted, err := db.CleanExpiredVoiceMemos()
	if err != nil {
		t.Fatalf("clean expired failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deletion, got %d", deleted)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("audio file should be removed, stat err = %v", err)
	}
}
