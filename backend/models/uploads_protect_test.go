package models

import (
	"reflect"
	"sort"
	"testing"
)

func TestExtractUploadFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/api/chat/uploads/abc.png", "abc.png"},
		{"/api/chat/uploads/abc-123_456.mp4", "abc-123_456.mp4"},
		{"/api/paste/files/foo.jpg?token=xxx", "foo.jpg"},
		{"/api/paste/files/foo.jpg#anchor", "foo.jpg"},
		{"__uploads__/bar.png", "bar.png"},
		{"  /api/chat/uploads/spaced.png  ", "spaced.png"},
		{"", ""},
		{"/api/chat/uploads/", ""},
		{"/api/chat/uploads/../escape.txt", ""}, // 防穿越
	}
	for _, c := range cases {
		got := extractUploadFilename(c.in)
		if got != c.want {
			t.Errorf("extractUploadFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestReferencedUploadFilenamesAggregatesSources 验证只覆盖"事件相关"来源:
//   - planner_task_comments.image_urls   (事项评论贴图)
//   - voice_memos.audio_url              (语音备忘/事项录音)
//   - photowall_items.filename           (照片墙)
//   - nfs_shares.file_path               (NFS 长期分享)
//
// 故意不保护的范围(paste / chat / md / excalidraw)由各自的过期机制处理。
func TestReferencedUploadFilenamesAggregatesSources(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer db.Close()

	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}

	// 1) planner_task_comments.image_urls
	profile := &PlannerProfile{Name: "t", PasswordIndex: "x"}
	if err := db.CreatePlannerProfile(profile, nil); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	task := &PlannerTask{ProfileID: profile.ID, Title: "t", EntryType: PlannerEntryTask, Bucket: PlannerBucketInbox, Status: "open", Priority: "medium", Kind: "work"}
	if err := db.CreatePlannerTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	comment := &PlannerTaskComment{
		TaskID:    task.ID,
		ProfileID: profile.ID,
		Author:    "me",
		Content:   "看图",
		ImageURLs: `["/api/paste/files/comment-pic.png"]`,
	}
	if err := db.CreatePlannerTaskComment(comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	// 2) voice_memos.audio_url
	_, err = db.conn.Exec(`INSERT INTO voice_memos (id, audio_url, status, created_at) VALUES (?, ?, ?, datetime('now'))`,
		"vm1", "/api/chat/uploads/voice-memo.mp3", "ready")
	if err != nil {
		t.Fatalf("insert voicememo: %v", err)
	}

	// 3) photowall_items.filename
	_, err = db.conn.Exec(`INSERT INTO photowall_profiles (id, password, password_index, creator_key, access_key, title, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		"pw1", "x", "pi1", "ck1", "ak1", "相册")
	if err != nil {
		t.Fatalf("insert pw profile: %v", err)
	}
	_, err = db.conn.Exec(`INSERT INTO photowall_items (id, profile_id, image_url, filename, created_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		"pwi1", "pw1", "/api/photowall/files/photo-wall.png", "photo-wall.png")
	if err != nil {
		t.Fatalf("insert pw item: %v", err)
	}

	// 4) nfs_shares.file_path
	_, err = db.conn.Exec(`INSERT INTO nfs_shares (id, name, file_path, created_at) VALUES (?, ?, ?, datetime('now'))`,
		"nfs1", "share", "__uploads__/nfs-file.bin")
	if err != nil {
		t.Fatalf("insert nfs: %v", err)
	}

	// 反面用例(故意不保护,验证确实没被收集)
	_, err = db.conn.Exec(`INSERT INTO pastes (id, content, files, created_at) VALUES (?, ?, ?, datetime('now'))`,
		"p1", "hello", `[{"filename":"paste-attach.png","size":1}]`)
	if err != nil {
		t.Fatalf("insert paste: %v", err)
	}
	_, err = db.conn.Exec(`INSERT INTO chat_rooms (id, name, created_at) VALUES (?, ?, datetime('now'))`, "cr1", "room")
	if err != nil {
		t.Fatalf("insert room: %v", err)
	}
	_, err = db.conn.Exec(`INSERT INTO chat_messages (room_id, nickname, content, msg_type, created_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		"cr1", "alice", "/api/chat/uploads/chat-pic.jpg", "image")
	if err != nil {
		t.Fatalf("insert chat msg: %v", err)
	}

	got, err := db.ReferencedUploadFilenames()
	if err != nil {
		t.Fatalf("ReferencedUploadFilenames: %v", err)
	}

	want := map[string]struct{}{
		"comment-pic.png": {},
		"voice-memo.mp3":  {},
		"photo-wall.png":  {},
		"nfs-file.bin":    {},
	}
	// 验证必须的 4 个都在
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q from result set %v", k, got)
		}
	}
	// 验证故意不保护的范围没被收集
	notWanted := []string{"paste-attach.png", "chat-pic.jpg"}
	for _, k := range notWanted {
		if _, ok := got[k]; ok {
			t.Errorf("paste/chat 文件不应被保护, 但 %q 出现在结果集 %v", k, got)
		}
	}
	// 验证总数对得上(4 个,不多不少)
	if len(got) != len(want) {
		t.Errorf("len mismatch: got %d, want %d, got=%v", len(got), len(want), got)
	}
}

// 保留一个 sort 用例,确保反面对照不依赖具体顺序。
func TestSortHelper(t *testing.T) {
	got := []string{"b", "a"}
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("sort broken")
	}
}
