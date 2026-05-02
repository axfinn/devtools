package handlers

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"devtools/models"
)

func buildPlannerBoard(tasks []*models.PlannerTask, kind string, now time.Time) plannerBoardResponse {
	today := now.Format("2006-01-02")
	board := plannerBoardResponse{
		Kind:         kind,
		Groups:       make([]*plannerTimelineGroup, 0),
		EventGroups:  make([]*plannerTimelineGroup, 0),
		InboxItems:   make([]*plannerTimelineItem, 0),
		SomedayItems: make([]*plannerTimelineItem, 0),
		RecentItems:  make([]*plannerTimelineItem, 0),
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
	}

	timelineGroups := map[string]*plannerTimelineGroup{}
	eventGroups := map[string]*plannerTimelineGroup{}
	timelineOrder := make([]string, 0)
	eventOrder := make([]string, 0)
	todayPrimaryCount := 0
	doneToday := 0
	cancelledToday := 0
	focusCandidates := make([]*plannerTimelineItem, 0)
	upcomingEvents := make([]*plannerTimelineItem, 0)

	for _, task := range tasks {
		status := normalizePlannerStatus(task.Status)
		if task.Kind != kind {
			continue
		}
		board.Counts[status]++

		item := &plannerTimelineItem{
			PlannerTask:       task,
			DisplayDate:       task.PlannedFor,
			DisplayLabel:      plannerDateLabel(task.PlannedFor, today),
			IsToday:           task.PlannedFor == today,
			CalendarAvailable: true,
			OverdueDays:       0,
		}
		applyPlannerTimeContext(item, now)
		if task.OriginalPlannedFor == "" {
			task.OriginalPlannedFor = task.PlannedFor
		}

		if status == plannerStatusDone && task.CompletedAt != nil && task.CompletedAt.Format("2006-01-02") == today {
			doneToday++
		}
		if status == plannerStatusCancelled && task.CompletedAt != nil && task.CompletedAt.Format("2006-01-02") == today {
			cancelledToday++
		}

		if status == plannerStatusDone || status == plannerStatusCancelled {
			item.DisplayDate = itemCompletionDate(task, today)
			item.DisplayLabel = plannerDateLabel(item.DisplayDate, today)
			board.RecentItems = append(board.RecentItems, item)
			continue
		}

		switch task.Bucket {
		case models.PlannerBucketInbox:
			board.Counts["inbox_open"]++
			board.InboxItems = append(board.InboxItems, item)
			continue
		case models.PlannerBucketSomeday:
			board.Counts["someday_open"]++
			board.SomedayItems = append(board.SomedayItems, item)
			continue
		}

		if task.EntryType == models.PlannerEntryEvent {
			board.Counts["event_open"]++
			group := ensurePlannerGroup(eventGroups, &eventOrder, task.PlannedFor, today)
			item.IsToday = item.DisplayDate == today
			if item.IsToday {
				todayPrimaryCount++
			}
			group.Items = append(group.Items, item)
			upcomingEvents = append(upcomingEvents, item)
			continue
		}

		displayDate := task.PlannedFor
		if displayDate == "" {
			displayDate = today
		}
		if displayDate < today {
			item.IsRolledOver = true
			item.DisplayDate = today
			item.DisplayLabel = "今天"
			item.IsToday = true
			board.Counts["rolled_over"]++
			if plannedTime, err := time.Parse("2006-01-02", displayDate); err == nil {
				item.OverdueDays = int(now.Sub(plannedTime).Hours() / 24)
				if item.OverdueDays < 0 {
					item.OverdueDays = 0
				}
			}
		}
		group := ensurePlannerGroup(timelineGroups, &timelineOrder, item.DisplayDate, today)
		group.Items = append(group.Items, item)
		if item.DisplayDate == today {
			todayPrimaryCount++
			focusCandidates = append(focusCandidates, item)
		}
	}

	board.Groups = finalizePlannerGroups(timelineGroups, timelineOrder)
	board.EventGroups = finalizePlannerGroups(eventGroups, eventOrder)

	sortPlannerItems(board.InboxItems)
	sortPlannerItems(board.SomedayItems)
	sort.SliceStable(board.RecentItems, func(i, j int) bool {
		return board.RecentItems[i].UpdatedAt.After(board.RecentItems[j].UpdatedAt)
	})
	if len(board.RecentItems) > 8 {
		board.RecentItems = board.RecentItems[:8]
	}

	limit := 3
	if kind == plannerKindLife {
		limit = 2
	}
	board.Focus = plannerFocusSummary{
		TodayPrimaryLimit: limit,
		TodayPrimaryCount: todayPrimaryCount,
		NeedsTrim:         todayPrimaryCount > limit,
		Secondary:         make([]*plannerTimelineItem, 0),
	}
	sortPlannerItems(focusCandidates)
	if len(focusCandidates) > 0 {
		board.Focus.Primary = focusCandidates[0]
		if len(focusCandidates) > 1 {
			maxSecondary := minPlannerInt(2, len(focusCandidates)-1)
			board.Focus.Secondary = append(board.Focus.Secondary, focusCandidates[1:1+maxSecondary]...)
		}
	}
	sortPlannerEventItems(upcomingEvents, now)
	if len(upcomingEvents) > 0 {
		board.Focus.NextEvent = upcomingEvents[0]
	}
	if board.Focus.NeedsTrim {
		board.Focus.Message = fmt.Sprintf("今天挂在时间线上的%s事项有 %d 件，建议先收敛到 %d 件以内。", map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[kind], todayPrimaryCount, limit)
	} else if board.Focus.Primary != nil {
		board.Focus.Message = fmt.Sprintf("今天最该先推进的是「%s」。", board.Focus.Primary.Title)
	} else {
		board.Focus.Message = fmt.Sprintf("今天主推进 %d 件，节奏还比较舒服。", todayPrimaryCount)
	}

	board.Recovery = plannerRecoverySummary{
		DoneToday:      doneToday,
		CancelledToday: cancelledToday,
		InboxOpen:      len(board.InboxItems),
	}
	switch {
	case doneToday > 0:
		board.Recovery.Message = fmt.Sprintf("今天已经完成 %d 件事，别只盯着没做完的部分。", doneToday)
	case len(board.InboxItems) > 0:
		board.Recovery.Message = fmt.Sprintf("收件箱里还有 %d 条想法，先分类，不急着都立刻做。", len(board.InboxItems))
	default:
		board.Recovery.Message = "今天的记录已经比较清爽，可以给自己留一点空。"
	}

	return board
}

func applyPlannerTimeContext(item *plannerTimelineItem, now time.Time) {
	if item == nil {
		return
	}
	if item.EntryType == models.PlannerEntryEvent {
		eventTime := plannerEventBaseTime(item)
		if eventTime != nil {
			diff := eventTime.Sub(now)
			switch {
			case diff < -2*time.Hour:
				item.EventPhase = "awaiting_closure"
				item.NeedsClosure = normalizePlannerStatus(item.Status) != plannerStatusDone && normalizePlannerStatus(item.Status) != plannerStatusCancelled
				item.TimeHint = "事件时间已过，建议收尾"
			case diff < 0:
				item.EventPhase = "in_window"
				item.TimeHint = "事件进行中"
			case diff <= 2*time.Hour:
				item.EventPhase = "soon"
				item.TimeHint = fmt.Sprintf("%d 分钟后开始", int(diff.Minutes()))
			case diff <= 24*time.Hour:
				item.EventPhase = "today"
				item.TimeHint = fmt.Sprintf("%d 小时后开始", int(diff.Hours()+0.5))
			default:
				item.EventPhase = "planned"
				item.TimeHint = plannerDateLabel(item.DisplayDate, now.Format("2006-01-02"))
			}
			return
		}
	}
	if item.IsRolledOver && item.OverdueDays > 0 {
		item.TimeHint = fmt.Sprintf("已晚 %d 天", item.OverdueDays)
		return
	}
	if item.IsToday {
		item.TimeHint = "今天"
		if item.EnergyLevel != "" {
			item.TimeHint = "今天 · " + energyLabel(item.EnergyLevel)
		}
		return
	}
	if item.DisplayLabel != "" {
		item.TimeHint = item.DisplayLabel
	}
}

func plannerEventBaseTime(item *plannerTimelineItem) *time.Time {
	if item.RemindAt != nil {
		return item.RemindAt
	}
	if strings.TrimSpace(item.PlannedFor) == "" {
		return nil
	}
	if t, err := time.ParseInLocation("2006-01-02", item.PlannedFor, time.Local); err == nil {
		return &t
	}
	return nil
}

func sortPlannerEventItems(items []*plannerTimelineItem, now time.Time) {
	sort.SliceStable(items, func(i, j int) bool {
		a := plannerEventBaseTime(items[i])
		b := plannerEventBaseTime(items[j])
		if a == nil && b == nil {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		da := a.Sub(now)
		db := b.Sub(now)
		if da < 0 && db >= 0 {
			return false
		}
		if da >= 0 && db < 0 {
			return true
		}
		return a.Before(*b)
	})
}

func plannerCommentPreview(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	runes := []rune(value)
	if len(runes) <= 48 {
		return value
	}
	return string(runes[:48]) + "..."
}

func applyPlannerCommentSummary(item *plannerTimelineItem, summaries map[string]*models.PlannerTaskCommentSummary) {
	if item == nil || summaries == nil {
		return
	}
	summary, ok := summaries[item.ID]
	if !ok || summary == nil {
		return
	}
	item.CommentCount = summary.CommentCount
	item.LastCommentPreview = plannerCommentPreview(summary.LastContent)
	item.LastCommentAt = summary.LastCreatedAt
}

func ensurePlannerGroup(groups map[string]*plannerTimelineGroup, order *[]string, date, today string) *plannerTimelineGroup {
	group, ok := groups[date]
	if !ok {
		group = &plannerTimelineGroup{
			Date:  date,
			Label: plannerDateLabel(date, today),
			Items: make([]*plannerTimelineItem, 0),
		}
		groups[date] = group
		*order = append(*order, date)
	}
	return group
}

func finalizePlannerGroups(groups map[string]*plannerTimelineGroup, order []string) []*plannerTimelineGroup {
	sort.Strings(order)
	result := make([]*plannerTimelineGroup, 0, len(order))
	for _, date := range order {
		group := groups[date]
		sortPlannerItems(group.Items)
		result = append(result, group)
	}
	return result
}

func sortPlannerItems(items []*plannerTimelineItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a := items[i]
		b := items[j]
		if a.Status != b.Status {
			return plannerStatusRank(a.Status) < plannerStatusRank(b.Status)
		}
		if a.Priority != b.Priority {
			return plannerPriorityRank(a.Priority) < plannerPriorityRank(b.Priority)
		}
		if a.EntryType != b.EntryType {
			return a.EntryType < b.EntryType
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
}

func plannerStatusRank(status string) int {
	switch normalizePlannerStatus(status) {
	case plannerStatusOpen:
		return 0
	case plannerStatusInProgress:
		return 1
	case plannerStatusDone:
		return 2
	default:
		return 3
	}
}

func plannerPriorityRank(priority string) int {
	switch normalizePlannerPriority(priority) {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}

func plannerDateLabel(date, today string) string {
	if date == today {
		return "今天"
	}
	if t, err := time.Parse("2006-01-02", today); err == nil {
		if date == t.AddDate(0, 0, 1).Format("2006-01-02") {
			return "明天"
		}
		if date == t.AddDate(0, 0, -1).Format("2006-01-02") {
			return "昨天"
		}
	}
	return date
}

func itemCompletionDate(task *models.PlannerTask, fallback string) string {
	if task.CompletedAt != nil {
		return task.CompletedAt.Format("2006-01-02")
	}
	if task.UpdatedAt.IsZero() {
		return fallback
	}
	return task.UpdatedAt.Format("2006-01-02")
}

func energyLabel(level string) string {
	switch level {
	case "deep":
		return "需专注"
	case "shallow":
		return "事务性"
	case "errand":
		return "需外出"
	case "creative":
		return "创意型"
	default:
		return ""
	}
}
