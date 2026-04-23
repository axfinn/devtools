package handlers

import (
	"strings"
	"testing"
	"time"

	"devtools/models"
)

func TestBuildPlannerTimelineRollover(t *testing.T) {
	now := time.Date(2026, 4, 23, 10, 0, 0, 0, time.Local)
	tasks := []*models.PlannerTask{
		{
			ID:         "a1",
			Kind:       "work",
			Title:      "昨天没做完",
			Status:     "open",
			Priority:   "high",
			PlannedFor: "2026-04-22",
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		{
			ID:         "a2",
			Kind:       "work",
			Title:      "今天要做",
			Status:     "in_progress",
			Priority:   "medium",
			PlannedFor: "2026-04-23",
			CreatedAt:  now.Add(-1 * time.Hour),
		},
	}

	groups, counts := buildPlannerTimeline(tasks, now)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Date != "2026-04-23" {
		t.Fatalf("expected group date 2026-04-23, got %s", groups[0].Date)
	}
	if !groups[0].Items[0].IsRolledOver {
		t.Fatalf("expected first task to be rolled over")
	}
	if counts["rolled_over"] != 1 {
		t.Fatalf("expected rolled_over count 1, got %d", counts["rolled_over"])
	}
}

func TestBuildPlannerICSTimedEvent(t *testing.T) {
	remindAt := time.Date(2026, 4, 23, 19, 30, 0, 0, time.Local)
	task := &models.PlannerTask{
		ID:         "ics123",
		Kind:       "life",
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
	if items[1].Kind != "life" {
		t.Fatalf("expected second item life, got %s", items[1].Kind)
	}
}
