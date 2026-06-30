package models

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPlannerSearchCoversCommentsAndRecordings(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer db.Close()
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}

	profile := &PlannerProfile{Name: "t", PasswordIndex: "x"}
	if err := db.CreatePlannerProfile(profile, nil); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// 任务 A: 标题/详情/笔记 都不含 "lawyer"
	taskA := &PlannerTask{
		ProfileID: profile.ID,
		Title:     "Q3 收入结算",
		Detail:    "与财务对接",
		Notes:     "",
		EntryType: PlannerEntryTask,
		Bucket:    PlannerBucketInbox,
		Status:    "open",
		Priority:  "medium",
		Kind:      "work",
	}
	if err := db.CreatePlannerTask(taskA); err != nil {
		t.Fatalf("create taskA: %v", err)
	}

	// 任务 B: 标题本身没有 "lawyer",但评论里有
	taskB := &PlannerTask{
		ProfileID: profile.ID,
		Title:     "续约与法务",
		Detail:    "准备续签材料",
		EntryType: PlannerEntryTask,
		Bucket:    PlannerBucketPlanned,
		Status:    "open",
		Priority:  "high",
		Kind:      "work",
	}
	if err := db.CreatePlannerTask(taskB); err != nil {
		t.Fatalf("create taskB: %v", err)
	}
	comment := &PlannerTaskComment{
		TaskID:    taskB.ID,
		ProfileID: profile.ID,
		Author:    "me",
		Content:   "已与外部 lawyer 沟通,律师建议在 7 月前完成主体变更",
	}
	if err := db.CreatePlannerTaskComment(comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	// 任务 C: 标题含 "lawyer"
	taskC := &PlannerTask{
		ProfileID: profile.ID,
		Title:     "找一家靠谱的 lawyer 咨询股权架构",
		EntryType: PlannerEntryTask,
		Bucket:    PlannerBucketSomeday,
		Status:    "open",
		Priority:  "low",
		Kind:      "work",
	}
	if err := db.CreatePlannerTask(taskC); err != nil {
		t.Fatalf("create taskC: %v", err)
	}

	// 任务 D: 录音转写里含 "lawyer"
	taskD := &PlannerTask{
		ProfileID: profile.ID,
		Title:     "客户拜访录音",
		EntryType: PlannerEntryTask,
		Bucket:    PlannerBucketPlanned,
		Status:    "done",
		Priority:  "medium",
		Kind:      "work",
	}
	if err := db.CreatePlannerTask(taskD); err != nil {
		t.Fatalf("create taskD: %v", err)
	}
	_, err = db.conn.Exec(`INSERT INTO voice_memos
		(id, title, transcript, status, planner_profile_id, planner_task_id, created_at)
		VALUES (?, ?, ?, 'ready', ?, ?, datetime('now'))`,
		"vm-1", "客户沟通", "今天和客户的 lawyer 谈了一小时,主要聊了合作条款", profile.ID, taskD.ID)
	if err != nil {
		t.Fatalf("insert vm: %v", err)
	}

	hits, err := db.PlannerSearch(profile.ID, "lawyer", 30, 3)
	if err != nil {
		t.Fatalf("PlannerSearch: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected at least one hit for 'lawyer'")
	}

	// 至少应覆盖到 3 个来源
	kinds := map[string]bool{}
	for _, h := range hits {
		kinds[h.MatchKind] = true
	}
	for _, k := range []string{PlannerSearchKindTask, PlannerSearchKindComment, PlannerSearchKindRecording} {
		if !kinds[k] {
			t.Errorf("expected match_kind %q in hits, got kinds=%v", k, kinds)
		}
	}

	// 评论命中应排在录音命中和任务命中之前
	idxOfKind := map[string]int{}
	for i, h := range hits {
		if _, ok := idxOfKind[h.MatchKind]; !ok {
			idxOfKind[h.MatchKind] = i
		}
	}
	if idxOfKind[PlannerSearchKindComment] > idxOfKind[PlannerSearchKindRecording] {
		t.Errorf("comment hits should rank before recording hits, order=%v", idxOfKind)
	}
	if idxOfKind[PlannerSearchKindRecording] > idxOfKind[PlannerSearchKindTask] {
		t.Errorf("recording hits should rank before task hits, order=%v", idxOfKind)
	}

	// Snippet 必须包含查询词
	for _, h := range hits {
		if !strings.Contains(strings.ToLower(h.Snippet), "lawyer") {
			t.Errorf("snippet should contain query, got %q", h.Snippet)
		}
	}
}

func TestPlannerSearchEmptyQueryReturnsEmpty(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer db.Close()
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	hits, err := db.PlannerSearch("any", "  ", 30, 3)
	if err != nil {
		t.Fatalf("PlannerSearch: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("empty query should return no hits, got %d", len(hits))
	}
}

func TestPlannerSearchDedupesPerTask(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer db.Close()
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	profile := &PlannerProfile{Name: "t", PasswordIndex: "x"}
	if err := db.CreatePlannerProfile(profile, nil); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	task := &PlannerTask{ProfileID: profile.ID, Title: "alpha alpha alpha", EntryType: PlannerEntryTask, Bucket: PlannerBucketInbox, Status: "open", Priority: "medium", Kind: "work"}
	if err := db.CreatePlannerTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// 5 条评论都含 "alpha"
	for i := 0; i < 5; i++ {
		if err := db.CreatePlannerTaskComment(&PlannerTaskComment{TaskID: task.ID, ProfileID: profile.ID, Author: "me", Content: "alpha " + string(rune('a'+i))}); err != nil {
			t.Fatalf("create comment: %v", err)
		}
	}
	hits, err := db.PlannerSearch(profile.ID, "alpha", 30, 2)
	if err != nil {
		t.Fatalf("PlannerSearch: %v", err)
	}
	// maxPerTask=2 限制了同一任务最多 2 条(1 task + 1 comment 实际不会冲突,但任务自身一条)
	if len(hits) > 2 {
		t.Errorf("maxPerTask=2 should cap hits, got %d", len(hits))
	}
}
