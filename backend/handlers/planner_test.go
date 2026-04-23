package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

func TestBuildPlannerBoardRollover(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.Local)
	tasks := []*models.PlannerTask{
		{
			ID:                 "a1",
			Kind:               "work",
			EntryType:          models.PlannerEntryTask,
			Bucket:             models.PlannerBucketPlanned,
			Title:              "昨天没做完",
			Status:             "open",
			Priority:           "high",
			PlannedFor:         "2026-04-22",
			OriginalPlannedFor: "2026-04-22",
			CreatedAt:          now.Add(-2 * time.Hour),
		},
		{
			ID:                 "a2",
			Kind:               "work",
			EntryType:          models.PlannerEntryTask,
			Bucket:             models.PlannerBucketPlanned,
			Title:              "今天要做",
			Status:             "in_progress",
			Priority:           "medium",
			PlannedFor:         "2026-04-23",
			OriginalPlannedFor: "2026-04-23",
			CreatedAt:          now.Add(-1 * time.Hour),
		},
		{
			ID:                 "life1",
			Kind:               "life",
			EntryType:          models.PlannerEntryTask,
			Bucket:             models.PlannerBucketPlanned,
			Title:              "生活事项不应计入工作看板",
			Status:             "open",
			Priority:           "medium",
			PlannedFor:         "2026-04-23",
			OriginalPlannedFor: "2026-04-23",
			CreatedAt:          now.Add(-30 * time.Minute),
		},
	}

	board := buildPlannerBoard(tasks, plannerKindWork, now)
	if len(board.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(board.Groups))
	}
	if board.Groups[0].Date != "2026-04-23" {
		t.Fatalf("expected group date 2026-04-23, got %s", board.Groups[0].Date)
	}
	if !board.Groups[0].Items[0].IsRolledOver {
		t.Fatalf("expected first task to be rolled over")
	}
	if board.Counts["rolled_over"] != 1 {
		t.Fatalf("expected rolled_over count 1, got %d", board.Counts["rolled_over"])
	}
	if board.Counts["open"] != 1 {
		t.Fatalf("expected work open count 1, got %d", board.Counts["open"])
	}
	if board.Counts["in_progress"] != 1 {
		t.Fatalf("expected work in_progress count 1, got %d", board.Counts["in_progress"])
	}
}

func TestBuildPlannerICSTimedEvent(t *testing.T) {
	remindAt := time.Date(2026, 4, 23, 19, 30, 0, 0, time.Local)
	task := &models.PlannerTask{
		ID:         "ics123",
		Kind:       "life",
		EntryType:  models.PlannerEntryEvent,
		Bucket:     models.PlannerBucketPlanned,
		Title:      "预约复诊",
		Detail:     "去医院复诊",
		Notes:      "带医保卡",
		PlannedFor: "2026-04-24",
		RemindAt:   &remindAt,
	}

	content := string(buildPlannerICS(task))
	if !strings.Contains(content, "BEGIN:VCALENDAR") {
		t.Fatalf("expected calendar header")
	}
	if !strings.Contains(content, "SUMMARY:预约复诊") {
		t.Fatalf("expected summary in calendar content")
	}
	if !strings.Contains(content, "BEGIN:VALARM") {
		t.Fatalf("expected alarm block in calendar content")
	}
}

func TestFallbackParsePlannerText(t *testing.T) {
	items := fallbackParsePlannerText("今天开会确认方案\n下班后买水果", "work")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Kind != "work" {
		t.Fatalf("expected first item work, got %s", items[0].Kind)
	}
	if items[0].EntryType != models.PlannerEntryEvent {
		t.Fatalf("expected first item event, got %s", items[0].EntryType)
	}
	if items[1].Kind != "life" {
		t.Fatalf("expected second item life, got %s", items[1].Kind)
	}
	if items[1].Bucket != models.PlannerBucketPlanned {
		t.Fatalf("expected second item planned bucket, got %s", items[1].Bucket)
	}
}

func TestPlannerUpdateTaskPatchDoesNotClearOtherFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	if err := db.InitPlanner(); err != nil {
		t.Fatalf("init planner failed: %v", err)
	}

	creatorKey, err := utils.HashPassword("creator-key")
	if err != nil {
		t.Fatalf("hash creator key failed: %v", err)
	}
	profile := &models.PlannerProfile{
		PasswordIndex: plannerPasswordIndex("secret-1234"),
		CreatorKey:    creatorKey,
		Name:          "测试档案",
		NotifyEmail:   "owner@example.com",
	}
	if err := db.CreatePlannerProfile(profile, nil); err != nil {
		t.Fatalf("create profile failed: %v", err)
	}

	remindAt := time.Date(2026, 4, 23, 14, 30, 0, 0, time.Local)
	task := &models.PlannerTask{
		ProfileID:          profile.ID,
		Kind:               plannerKindWork,
		EntryType:          models.PlannerEntryTask,
		Bucket:             models.PlannerBucketPlanned,
		Title:              "保留详情",
		Detail:             "原始详情",
		Notes:              "原始备注",
		Status:             plannerStatusOpen,
		Priority:           "high",
		PlannedFor:         "2026-04-23",
		OriginalPlannedFor: "2026-04-23",
		RemindAt:           &remindAt,
		NotifyEmail:        "notify@example.com",
	}
	if err := db.CreatePlannerTask(task); err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	handler := NewPlannerHandler(db, config.DefaultConfig())
	body, _ := json.Marshal(map[string]string{"status": plannerStatusDone})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/planner/profile/"+profile.ID+"/tasks/"+task.ID, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Password", "secret-1234")
	c.Params = gin.Params{
		{Key: "id", Value: profile.ID},
		{Key: "taskId", Value: task.ID},
	}

	handler.UpdateTask(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	updated, err := db.GetPlannerTask(task.ID)
	if err != nil {
		t.Fatalf("get updated task failed: %v", err)
	}
	if updated.Detail != "原始详情" {
		t.Fatalf("expected detail preserved, got %q", updated.Detail)
	}
	if updated.Notes != "原始备注" {
		t.Fatalf("expected notes preserved, got %q", updated.Notes)
	}
	if updated.NotifyEmail != "notify@example.com" {
		t.Fatalf("expected notify email preserved, got %q", updated.NotifyEmail)
	}
	if updated.RemindAt == nil || !updated.RemindAt.Equal(remindAt) {
		t.Fatalf("expected remind_at preserved, got %#v", updated.RemindAt)
	}
	if updated.Status != plannerStatusDone {
		t.Fatalf("expected status done, got %s", updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Fatalf("expected completed_at set")
	}
}
