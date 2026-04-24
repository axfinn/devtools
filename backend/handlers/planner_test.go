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
	if board.Focus.Primary == nil || board.Focus.Primary.ID != "a1" {
		t.Fatalf("expected primary focus a1, got %#v", board.Focus.Primary)
	}
}

func TestBuildPlannerICSTimedEvent(t *testing.T) {
	remindAt := time.Date(2026, 4, 23, 19, 30, 0, 0, time.Local)
	repeatUntil := time.Date(2026, 5, 23, 19, 30, 0, 0, time.Local)
	task := &models.PlannerTask{
		ID:          "ics123",
		Kind:        "life",
		EntryType:   models.PlannerEntryEvent,
		Bucket:      models.PlannerBucketPlanned,
		Title:       "预约复诊",
		Detail:      "去医院复诊",
		Notes:       "带医保卡",
		PlannedFor:  "2026-04-24",
		RemindAt:    &remindAt,
		RepeatType:  "weekly",
		RepeatUntil: &repeatUntil,
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
	if !strings.Contains(content, "RRULE:FREQ=WEEKLY") {
		t.Fatalf("expected repeat rule in calendar content, got %s", content)
	}
}

func TestPlannerNextReminderAfterDaily(t *testing.T) {
	base := time.Date(2026, 4, 24, 9, 30, 0, 0, time.Local)
	task := &models.PlannerTask{
		RemindAt:       &base,
		RepeatType:     "daily",
		RepeatInterval: 1,
	}

	next := plannerNextReminderAfter(task, time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local))
	if next == nil {
		t.Fatalf("expected next reminder")
	}
	if got := next.Format("2006-01-02 15:04"); got != "2026-04-25 09:30" {
		t.Fatalf("expected 2026-04-25 09:30, got %s", got)
	}
}

func TestPlannerNextReminderAfterWeekdays(t *testing.T) {
	friday := time.Date(2026, 4, 24, 18, 0, 0, 0, time.Local)
	task := &models.PlannerTask{
		RemindAt:       &friday,
		RepeatType:     "weekdays",
		RepeatInterval: 1,
	}

	next := plannerNextReminderAfter(task, time.Date(2026, 4, 24, 18, 5, 0, 0, time.Local))
	if next == nil {
		t.Fatalf("expected next reminder")
	}
	if got := next.Format("2006-01-02 15:04"); got != "2026-04-27 18:00" {
		t.Fatalf("expected 2026-04-27 18:00, got %s", got)
	}
}

func TestPlannerNextReminderAfterMonthlySkipsInvalidDays(t *testing.T) {
	january31 := time.Date(2026, 1, 31, 9, 0, 0, 0, time.Local)
	task := &models.PlannerTask{
		RemindAt:       &january31,
		RepeatType:     "monthly",
		RepeatInterval: 1,
	}

	next := plannerNextReminderAfter(task, time.Date(2026, 1, 31, 9, 5, 0, 0, time.Local))
	if next == nil {
		t.Fatalf("expected next reminder")
	}
	if got := next.Format("2006-01-02 15:04"); got != "2026-03-31 09:00" {
		t.Fatalf("expected 2026-03-31 09:00, got %s", got)
	}
}

func TestBuildPlannerBoardSelectsNextEvent(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.Local)
	eventSoon := now.Add(90 * time.Minute)
	eventTomorrow := now.Add(26 * time.Hour)
	tasks := []*models.PlannerTask{
		{
			ID:         "e2",
			Kind:       plannerKindWork,
			EntryType:  models.PlannerEntryEvent,
			Bucket:     models.PlannerBucketPlanned,
			Title:      "明天会议",
			Status:     plannerStatusOpen,
			Priority:   "medium",
			PlannedFor: now.Add(24 * time.Hour).Format("2006-01-02"),
			RemindAt:   &eventTomorrow,
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		{
			ID:         "e1",
			Kind:       plannerKindWork,
			EntryType:  models.PlannerEntryEvent,
			Bucket:     models.PlannerBucketPlanned,
			Title:      "即将开始的会议",
			Status:     plannerStatusOpen,
			Priority:   "high",
			PlannedFor: now.Format("2006-01-02"),
			RemindAt:   &eventSoon,
			CreatedAt:  now.Add(-1 * time.Hour),
		},
	}

	board := buildPlannerBoard(tasks, plannerKindWork, now)
	if board.Focus.NextEvent == nil || board.Focus.NextEvent.ID != "e1" {
		t.Fatalf("expected next event e1, got %#v", board.Focus.NextEvent)
	}
	if board.Focus.NextEvent.EventPhase == "" {
		t.Fatalf("expected next event phase")
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

func TestPlannerProfileUpdateRequiresCreatorKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	if err := db.InitPlanner(); err != nil {
		t.Fatalf("init planner failed: %v", err)
	}

	hashedCreatorKey, err := utils.HashPassword("creator-key")
	if err != nil {
		t.Fatalf("hash creator key failed: %v", err)
	}
	profile := &models.PlannerProfile{
		PasswordIndex: plannerPasswordIndex("secret-1234"),
		CreatorKey:    hashedCreatorKey,
		Name:          "测试档案",
	}
	if err := db.CreatePlannerProfile(profile, nil); err != nil {
		t.Fatalf("create profile failed: %v", err)
	}

	handler := NewPlannerHandler(db, config.DefaultConfig())
	body, _ := json.Marshal(map[string]string{"name": "新名字"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/planner/profile/"+profile.ID, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Password", "secret-1234")
	c.Params = gin.Params{{Key: "id", Value: profile.ID}}

	handler.UpdateProfile(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when using password only, got %d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/planner/profile/"+profile.ID, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Creator-Key", "creator-key")
	c.Params = gin.Params{{Key: "id", Value: profile.ID}}

	handler.UpdateProfile(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when using creator key, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestPlannerParseFallsBackWhenMiniMaxFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
	}))
	defer server.Close()

	originalURL := plannerMiniMaxAPIURL
	plannerMiniMaxAPIURL = server.URL
	defer func() {
		plannerMiniMaxAPIURL = originalURL
	}()

	cfg := config.DefaultConfig()
	cfg.MiniMax.APIKey = "test-key"
	handler := NewPlannerHandler(nil, cfg)

	tasks, provider, err := handler.parsePlannerText("今天开会确认方案\n下班后买水果", plannerKindWork)
	if err != nil {
		t.Fatalf("expected fallback without error, got %v", err)
	}
	if provider != "fallback" {
		t.Fatalf("expected fallback provider, got %s", provider)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected fallback tasks, got %d", len(tasks))
	}
}

func TestFallbackParsePlannerTextExtractsEventReminder(t *testing.T) {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	tasks := fallbackParsePlannerText("明天下午3点开会确认方案", plannerKindWork)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	task := tasks[0]
	if task.EntryType != models.PlannerEntryEvent {
		t.Fatalf("expected event, got %s", task.EntryType)
	}
	if task.PlannedFor != tomorrow {
		t.Fatalf("expected planned_for %s, got %s", tomorrow, task.PlannedFor)
	}
	if task.RemindAt != tomorrow+"T15:00" {
		t.Fatalf("expected remind_at %sT15:00, got %s", tomorrow, task.RemindAt)
	}
}

func TestNormalizePlannerAITaskInfersReminderFromText(t *testing.T) {
	task := normalizePlannerAITask(plannerAITask{
		EntryType: models.PlannerEntryEvent,
		Title:     "周五晚上8点和客户开会",
		Detail:    "确认方案",
	}, plannerKindWork)

	if task.EntryType != models.PlannerEntryEvent {
		t.Fatalf("expected event, got %s", task.EntryType)
	}
	if task.RemindAt == "" {
		t.Fatalf("expected remind_at to be inferred")
	}
	if !strings.Contains(task.RemindAt, "T20:00") {
		t.Fatalf("expected inferred 20:00 reminder, got %s", task.RemindAt)
	}
}

func TestPlannerAdviseFallsBackWhenMiniMaxFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`timeout`))
	}))
	defer server.Close()

	originalURL := plannerMiniMaxAPIURL
	plannerMiniMaxAPIURL = server.URL
	defer func() {
		plannerMiniMaxAPIURL = originalURL
	}()

	cfg := config.DefaultConfig()
	cfg.MiniMax.APIKey = "test-key"
	handler := NewPlannerHandler(nil, cfg)
	profile := &models.PlannerProfile{Name: "测试档案"}
	board := createEmptyPlannerBoardForTest()
	board.Kind = plannerKindWork
	board.Focus.TodayPrimaryCount = 1
	board.Focus.TodayPrimaryLimit = 3
	board.Focus.Message = "今天主推进 1 件，节奏还比较舒服。"
	board.Recovery.Message = "收件箱里还有 1 条想法。"
	board.Recovery.InboxOpen = 1
	board.Counts[plannerStatusOpen] = 2

	advice, provider, err := handler.advisePlanner(profile, board, "plan", "")
	if err != nil {
		t.Fatalf("expected fallback without error, got %v", err)
	}
	if provider != "fallback" {
		t.Fatalf("expected fallback provider, got %s", provider)
	}
	if strings.TrimSpace(advice.Summary) == "" {
		t.Fatalf("expected fallback advice summary")
	}
}

func TestBuildPlannerReview(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.Local)
	doneAt := now.Add(-24 * time.Hour)
	tasks := []*models.PlannerTask{
		{
			ID:          "done1",
			Kind:        plannerKindWork,
			EntryType:   models.PlannerEntryTask,
			Bucket:      models.PlannerBucketPlanned,
			Title:       "完成事项",
			Status:      plannerStatusDone,
			CreatedAt:   now.Add(-48 * time.Hour),
			CompletedAt: &doneAt,
		},
		{
			ID:         "event1",
			Kind:       plannerKindWork,
			EntryType:  models.PlannerEntryEvent,
			Bucket:     models.PlannerBucketPlanned,
			Title:      "待收尾事件",
			Status:     plannerStatusOpen,
			PlannedFor: now.Add(-24 * time.Hour).Format("2006-01-02"),
			CreatedAt:  now.Add(-36 * time.Hour),
		},
	}

	review := buildPlannerReview(&models.PlannerProfile{Name: "档案"}, tasks, plannerKindWork, "week", now)
	if review.Stats["done"] != 1 {
		t.Fatalf("expected done 1, got %d", review.Stats["done"])
	}
	if review.Stats["events"] != 1 {
		t.Fatalf("expected events 1, got %d", review.Stats["events"])
	}
	if len(review.Suggestions) == 0 {
		t.Fatalf("expected review suggestions")
	}
}

func createEmptyPlannerBoardForTest() plannerBoardResponse {
	return plannerBoardResponse{
		Kind:         plannerKindLife,
		Groups:       []*plannerTimelineGroup{},
		EventGroups:  []*plannerTimelineGroup{},
		InboxItems:   []*plannerTimelineItem{},
		SomedayItems: []*plannerTimelineItem{},
		RecentItems:  []*plannerTimelineItem{},
		Counts: map[string]int{
			plannerStatusOpen:       0,
			plannerStatusInProgress: 0,
			plannerStatusDone:       0,
			plannerStatusCancelled:  0,
			"rolled_over":           0,
			"inbox_open":            0,
			"event_open":            0,
			"someday_open":          0,
		},
		Focus: plannerFocusSummary{
			TodayPrimaryLimit: 3,
		},
		Recovery: plannerRecoverySummary{},
	}
}
