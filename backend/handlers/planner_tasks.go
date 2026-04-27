package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

func (h *PlannerHandler) ListTimeline(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	kind := normalizePlannerKind(c.DefaultQuery("kind", plannerModeDefault(time.Now())))
	tasks, err := h.db.ListPlannerTasksByProfile(profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取事项失败", "code": 500})
		return
	}
	commentSummaries, err := h.db.ListPlannerTaskCommentSummaries(profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论摘要失败", "code": 500})
		return
	}
	now := time.Now()
	board := buildPlannerBoard(tasks, kind, now)
	for _, group := range board.Groups {
		for _, item := range group.Items {
			applyPlannerCommentSummary(item, commentSummaries)
		}
	}
	for _, group := range board.EventGroups {
		for _, item := range group.Items {
			applyPlannerCommentSummary(item, commentSummaries)
		}
	}
	for _, item := range board.InboxItems {
		applyPlannerCommentSummary(item, commentSummaries)
	}
	for _, item := range board.SomedayItems {
		applyPlannerCommentSummary(item, commentSummaries)
	}
	for _, item := range board.RecentItems {
		applyPlannerCommentSummary(item, commentSummaries)
	}
	board.ProfileName = profile.Name
	board.ModeDefault = plannerModeDefault(now)
	board.ModeHint = plannerModeHint(now)
	c.JSON(http.StatusOK, gin.H{"code": 0, "board": board})
}

func (h *PlannerHandler) Review(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	kind := normalizePlannerKind(c.DefaultQuery("kind", plannerModeDefault(time.Now())))
	period := strings.TrimSpace(strings.ToLower(c.DefaultQuery("period", "week")))
	tasks, err := h.db.ListPlannerTasksByProfile(profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取事项失败", "code": 500})
		return
	}
	review := buildPlannerReview(profile, tasks, kind, period, time.Now())
	c.JSON(http.StatusOK, gin.H{"code": 0, "review": review})
}

func buildPlannerReview(profile *models.PlannerProfile, tasks []*models.PlannerTask, kind, period string, now time.Time) plannerReviewResponse {
	days := 7
	label := "周回顾"
	switch period {
	case "month":
		days = 30
		label = "月回顾"
	case "year":
		days = 365
		label = "年回顾"
	default:
		period = "week"
	}
	start := now.AddDate(0, 0, -(days - 1))
	stats := map[string]int{
		"created":       0,
		"done":          0,
		"cancelled":     0,
		"open":          0,
		"events":        0,
		"rollovers":     0,
		"commented":     0,
		"needs_closure": 0,
	}
	wins := make([]string, 0, 4)
	drifts := make([]string, 0, 4)
	suggestions := make([]string, 0, 4)
	highlights := make([]*plannerTimelineItem, 0, 5)
	highlightSeen := map[string]bool{}

	for _, task := range tasks {
		if task.Kind != kind {
			continue
		}
		if task.CreatedAt.After(start) {
			stats["created"]++
		}
		if task.EntryType == models.PlannerEntryEvent {
			stats["events"]++
		}
		if task.RolloverCount > 0 && task.UpdatedAt.After(start) {
			stats["rollovers"]++
		}
		if task.CompletedAt != nil && task.CompletedAt.After(start) {
			switch normalizePlannerStatus(task.Status) {
			case plannerStatusDone:
				stats["done"]++
			case plannerStatusCancelled:
				stats["cancelled"]++
			}
			if len(highlights) < 5 && !highlightSeen[task.ID] {
				item := &plannerTimelineItem{
					PlannerTask:  task,
					DisplayDate:  itemCompletionDate(task, now.Format("2006-01-02")),
					DisplayLabel: plannerDateLabel(itemCompletionDate(task, now.Format("2006-01-02")), now.Format("2006-01-02")),
				}
				applyPlannerTimeContext(item, now)
				highlights = append(highlights, item)
				highlightSeen[task.ID] = true
			}
		}
		if normalizePlannerStatus(task.Status) == plannerStatusOpen || normalizePlannerStatus(task.Status) == plannerStatusInProgress {
			stats["open"]++
			if task.EntryType == models.PlannerEntryEvent {
				item := &plannerTimelineItem{PlannerTask: task, DisplayDate: task.PlannedFor}
				applyPlannerTimeContext(item, now)
				if item.NeedsClosure {
					stats["needs_closure"]++
				}
			}
		}
		if strings.TrimSpace(task.LastPostponeReason) != "" && task.LastPostponedAt != nil && task.LastPostponedAt.After(start) {
			stats["commented"]++
		}
	}

	switch {
	case stats["done"] > 0:
		wins = append(wins, fmt.Sprintf("这段时间完成了 %d 件%s事项。", stats["done"], map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[kind]))
	case stats["created"] > 0:
		wins = append(wins, fmt.Sprintf("至少把 %d 个念头接住了，没有让它们继续悬在脑子里。", stats["created"]))
	default:
		wins = append(wins, "这段时间记录不多，说明节奏相对稳定。")
	}
	if stats["events"] > 0 {
		wins = append(wins, fmt.Sprintf("管理了 %d 个事件型安排。", stats["events"]))
	}

	if stats["rollovers"] > 0 {
		drifts = append(drifts, fmt.Sprintf("有 %d 次顺延，说明计划密度偏高。", stats["rollovers"]))
	}
	if stats["needs_closure"] > 0 {
		drifts = append(drifts, fmt.Sprintf("还有 %d 个事件过时未收尾，档案感会被削弱。", stats["needs_closure"]))
	}
	if stats["open"] > stats["done"]+2 {
		drifts = append(drifts, "未完成量高于收尾量，下一阶段要收而不是继续铺。")
	}

	if stats["needs_closure"] > 0 {
		suggestions = append(suggestions, "先把已发生但未收尾的事件逐个完成或取消，保持事件档案干净。")
	}
	if stats["rollovers"] > 0 {
		suggestions = append(suggestions, "下一阶段给今天只留一个主事项，其余转收件箱或明确改期。")
	}
	if stats["open"] > 0 {
		suggestions = append(suggestions, "从未完成事项里只挑一件最有杠杆的，作为新的阶段起点。")
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "这段时间节奏还算稳，可以补一两个真正重要的下一步，而不是继续堆列表。")
	}

	summary := fmt.Sprintf("%s里新增 %d 条记录，完成 %d 条，仍有 %d 条未收尾。", label, stats["created"], stats["done"], stats["open"])
	return plannerReviewResponse{
		Period:      period,
		Label:       label,
		Summary:     fmt.Sprintf("%s「%s」%s", map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[kind], profile.Name, summary),
		Stats:       stats,
		Wins:        wins,
		Drifts:      drifts,
		Suggestions: suggestions,
		Highlights:  highlights,
	}
}

func (h *PlannerHandler) buildPlannerTask(profile *models.PlannerProfile, req createPlannerTaskRequest) (*models.PlannerTask, error) {
	task := &models.PlannerTask{
		ProfileID:          profile.ID,
		Kind:               normalizePlannerKind(req.Kind),
		EntryType:          normalizePlannerEntryType(req.EntryType),
		Bucket:             normalizePlannerBucket(req.Bucket, req.EntryType),
		Title:              strings.TrimSpace(req.Title),
		Detail:             strings.TrimSpace(req.Detail),
		Notes:              strings.TrimSpace(req.Notes),
		Status:             normalizePlannerStatus(req.Status),
		Priority:           normalizePlannerPriority(req.Priority),
		PlannedFor:         parsePlannerDate(req.PlannedFor),
		OriginalPlannedFor: parsePlannerDate(req.PlannedFor),
		RemindAt:           parsePlannerDateTime(req.RemindAt),
		RepeatType:         normalizePlannerRepeatType(req.RepeatType),
		RepeatInterval:     normalizePlannerRepeatInterval(req.RepeatInterval),
		RepeatUntil:        parsePlannerDateTime(req.RepeatUntil),
		NotifyEmail:        strings.TrimSpace(req.NotifyEmail),
		RawText:            strings.TrimSpace(req.RawText),
		CancelReason:       strings.TrimSpace(req.CancelReason),
		Intent:             strings.TrimSpace(req.Intent),
		EnergyLevel:        normalizePlannerEnergyLevel(req.EnergyLevel),
	}
	if task.Title == "" {
		return nil, fmt.Errorf("标题不能为空")
	}
	if task.NotifyEmail == "" {
		task.NotifyEmail = profile.NotifyEmail
	}
	if task.RepeatType == "none" {
		task.RepeatInterval = 1
		task.RepeatUntil = nil
	}
	if task.RemindAt == nil {
		task.RepeatType = "none"
		task.RepeatInterval = 1
		task.RepeatUntil = nil
	}
	if task.RepeatType != "none" && task.RemindAt == nil {
		return nil, fmt.Errorf("重复提醒需要先设置提醒时间")
	}
	if task.RepeatUntil != nil && task.RemindAt != nil && task.RepeatUntil.Before(*task.RemindAt) {
		return nil, fmt.Errorf("重复提醒结束时间不能早于首次提醒")
	}
	if task.EntryType == models.PlannerEntryEvent {
		task.Bucket = models.PlannerBucketPlanned
	}
	if task.Bucket == models.PlannerBucketInbox && strings.TrimSpace(req.PlannedFor) == "" {
		task.PlannedFor = plannerToday()
		task.OriginalPlannedFor = task.PlannedFor
	}
	if task.Status == plannerStatusDone || task.Status == plannerStatusCancelled {
		now := time.Now()
		task.CompletedAt = &now
	}
	if task.Status == plannerStatusCancelled && task.CancelReason == "" {
		task.CancelReason = "用户取消"
	}
	return task, nil
}

func (h *PlannerHandler) CreateTask(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	var req createPlannerTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空", "code": 400})
		return
	}
	task, err := h.buildPlannerTask(profile, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	if err := h.db.CreatePlannerTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}
	_ = h.db.CreatePlannerTaskActivity(&models.PlannerTaskActivity{
		TaskID:       task.ID,
		ProfileID:    task.ProfileID,
		ActivityType: "created",
		Title:        plannerTaskLifecycleTitle(task),
		Content:      plannerTaskLifecycleContent(task),
	})
	c.JSON(http.StatusOK, gin.H{"code": 0, "task": task})
}

func (h *PlannerHandler) CreateTaskBatch(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	var req createPlannerTaskBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Tasks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供至少一条事项", "code": 400})
		return
	}
	if len(req.Tasks) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "一次最多写入 50 条事项", "code": 400})
		return
	}
	tasks := make([]*models.PlannerTask, 0, len(req.Tasks))
	for index, item := range req.Tasks {
		task, err := h.buildPlannerTask(profile, item)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":        fmt.Sprintf("第 %d 条事项无效: %s", index+1, err.Error()),
				"code":         400,
				"failed_index": index,
			})
			return
		}
		tasks = append(tasks, task)
	}
	if err := h.db.CreatePlannerTasks(tasks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量写入失败", "code": 500})
		return
	}
	for _, task := range tasks {
		_ = h.db.CreatePlannerTaskActivity(&models.PlannerTaskActivity{
			TaskID:       task.ID,
			ProfileID:    task.ProfileID,
			ActivityType: "created",
			Title:        plannerTaskLifecycleTitle(task),
			Content:      plannerTaskLifecycleContent(task),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "created_count": len(tasks), "tasks": tasks})
}

func (h *PlannerHandler) UpdateTask(c *gin.Context) {
	_, task, ok := h.loadTask(c)
	if !ok {
		return
	}
	before := *task
	originalPlanned := task.PlannedFor
	var req updatePlannerTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}
	if req.Kind != nil {
		task.Kind = normalizePlannerKind(*req.Kind)
	}
	if req.EntryType != nil {
		task.EntryType = normalizePlannerEntryType(*req.EntryType)
	}
	if req.Bucket != nil {
		task.Bucket = normalizePlannerBucket(*req.Bucket, task.EntryType)
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空", "code": 400})
			return
		}
		task.Title = title
	}
	if req.Detail != nil {
		task.Detail = strings.TrimSpace(*req.Detail)
	}
	if req.Notes != nil {
		task.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.Status != nil {
		task.Status = normalizePlannerStatus(*req.Status)
	}
	if req.Priority != nil {
		task.Priority = normalizePlannerPriority(*req.Priority)
	}
	if req.NotifyEmail != nil {
		task.NotifyEmail = strings.TrimSpace(*req.NotifyEmail)
	}
	if req.RawText != nil {
		task.RawText = strings.TrimSpace(*req.RawText)
	}
	if req.RemindAt != nil {
		task.RemindAt = parsePlannerDateTime(*req.RemindAt)
		task.LastNotifiedAt = nil
	}
	if req.RepeatType != nil {
		task.RepeatType = normalizePlannerRepeatType(*req.RepeatType)
		task.LastNotifiedAt = nil
	}
	if req.RepeatInterval != nil {
		task.RepeatInterval = normalizePlannerRepeatInterval(*req.RepeatInterval)
		task.LastNotifiedAt = nil
	}
	if req.RepeatUntil != nil {
		task.RepeatUntil = parsePlannerDateTime(*req.RepeatUntil)
		task.LastNotifiedAt = nil
	}
	if task.RepeatType == "none" {
		task.RepeatInterval = 1
		task.RepeatUntil = nil
	}
	if req.PlannedFor != nil {
		task.PlannedFor = parsePlannerDate(*req.PlannedFor)
	}
	if task.RemindAt == nil {
		task.RepeatType = "none"
		task.RepeatInterval = 1
		task.RepeatUntil = nil
		task.LastNotifiedAt = nil
	}
	if task.RepeatType != "none" && task.RemindAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重复提醒需要先设置提醒时间", "code": 400})
		return
	}
	if task.RepeatUntil != nil && task.RemindAt != nil && task.RepeatUntil.Before(*task.RemindAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重复提醒结束时间不能早于首次提醒", "code": 400})
		return
	}
	if task.EntryType == models.PlannerEntryEvent {
		task.Bucket = models.PlannerBucketPlanned
	}
	if task.OriginalPlannedFor == "" {
		task.OriginalPlannedFor = originalPlanned
	}
	if task.OriginalPlannedFor == "" {
		task.OriginalPlannedFor = task.PlannedFor
	}
	postponeReason := ""
	if req.PostponeReason != nil {
		postponeReason = strings.TrimSpace(*req.PostponeReason)
	}
	if task.PlannedFor != "" && task.PlannedFor != originalPlanned {
		if task.PlannedFor > originalPlanned && originalPlanned != "" {
			task.RolloverCount++
			task.LastPostponeReason = plannerFirstNonEmpty(postponeReason, task.LastPostponeReason, "手动顺延")
			now := time.Now()
			task.LastPostponedAt = &now
		}
	}
	switch task.Status {
	case plannerStatusDone:
		now := time.Now()
		task.CompletedAt = &now
		task.CancelReason = ""
	case plannerStatusCancelled:
		now := time.Now()
		task.CompletedAt = &now
		if req.CancelReason != nil {
			task.CancelReason = plannerFirstNonEmpty(strings.TrimSpace(*req.CancelReason), task.CancelReason, "用户取消")
		} else {
			task.CancelReason = plannerFirstNonEmpty(task.CancelReason, "用户取消")
		}
	default:
		task.CompletedAt = nil
		task.CancelReason = ""
	}
	if err := h.db.UpdatePlannerTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	for _, activity := range plannerTaskActivitiesFromUpdate(&before, task) {
		_ = h.db.CreatePlannerTaskActivity(activity)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "task": task})
}

func (h *PlannerHandler) DeleteTask(c *gin.Context) {
	if _, _, ok := h.loadTask(c); !ok {
		return
	}
	if err := h.db.DeletePlannerTask(c.Param("taskId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *PlannerHandler) ListTaskComments(c *gin.Context) {
	_, task, ok := h.loadTask(c)
	if !ok {
		return
	}
	comments, err := h.db.ListPlannerTaskComments(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "comments": comments})
}

func (h *PlannerHandler) CreateTaskComment(c *gin.Context) {
	profile, task, ok := h.loadTask(c)
	if !ok {
		return
	}
	var req plannerCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论内容不能为空", "code": 400})
		return
	}
	comment := &models.PlannerTaskComment{
		TaskID:    task.ID,
		ProfileID: profile.ID,
		Author:    plannerFirstNonEmpty(strings.TrimSpace(req.Author), "我"),
		Content:   strings.TrimSpace(req.Content),
	}
	if comment.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论内容不能为空", "code": 400})
		return
	}
	if err := h.db.CreatePlannerTaskComment(comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "comment": comment})
}

func plannerTaskLifecycleTitle(task *models.PlannerTask) string {
	if task.EntryType == models.PlannerEntryEvent {
		return "创建了一个事件"
	}
	return "创建了一个事项"
}

func plannerTaskLifecycleContent(task *models.PlannerTask) string {
	parts := []string{task.Title}
	if strings.TrimSpace(task.PlannedFor) != "" {
		parts = append(parts, "日期 "+task.PlannedFor)
	}
	if task.RemindAt != nil {
		parts = append(parts, "提醒 "+task.RemindAt.In(time.Local).Format("2006-01-02 15:04"))
	}
	if repeatSummary := plannerRepeatSummary(task); repeatSummary != "" {
		parts = append(parts, repeatSummary)
	}
	return strings.Join(parts, " · ")
}

func plannerTaskActivitiesFromUpdate(before, after *models.PlannerTask) []*models.PlannerTaskActivity {
	items := make([]*models.PlannerTaskActivity, 0)
	appendActivity := func(activityType, title, content string) {
		items = append(items, &models.PlannerTaskActivity{
			TaskID:       after.ID,
			ProfileID:    after.ProfileID,
			ActivityType: activityType,
			Title:        title,
			Content:      strings.TrimSpace(content),
		})
	}

	if before.PlannedFor != after.PlannedFor {
		appendActivity("schedule_changed", "调整了日期", fmt.Sprintf("%s -> %s", plannerFirstNonEmpty(before.PlannedFor, "未设置"), plannerFirstNonEmpty(after.PlannedFor, "未设置")))
	}
	beforeRemind := ""
	if before.RemindAt != nil {
		beforeRemind = before.RemindAt.In(time.Local).Format("2006-01-02 15:04")
	}
	afterRemind := ""
	if after.RemindAt != nil {
		afterRemind = after.RemindAt.In(time.Local).Format("2006-01-02 15:04")
	}
	if beforeRemind != afterRemind {
		content := fmt.Sprintf("%s -> %s", plannerFirstNonEmpty(beforeRemind, "未设置"), plannerFirstNonEmpty(afterRemind, "未设置"))
		appendActivity("reminder_changed", "调整了提醒时间", content)
	}
	beforeRepeat := plannerRepeatSummary(before)
	afterRepeat := plannerRepeatSummary(after)
	if beforeRepeat != afterRepeat {
		content := fmt.Sprintf("%s -> %s", plannerFirstNonEmpty(beforeRepeat, "不重复"), plannerFirstNonEmpty(afterRepeat, "不重复"))
		appendActivity("repeat_changed", "调整了重复提醒", content)
	}
	if before.Status != after.Status {
		title := "更新了状态"
		switch after.Status {
		case plannerStatusDone:
			title = "完成了这条记录"
		case plannerStatusCancelled:
			title = "取消了这条记录"
		case plannerStatusInProgress:
			title = "开始推进这条记录"
		case plannerStatusOpen:
			title = "重新打开这条记录"
		}
		content := fmt.Sprintf("%s -> %s", statusLabelForActivity(before.Status), statusLabelForActivity(after.Status))
		if after.Status == plannerStatusCancelled && strings.TrimSpace(after.CancelReason) != "" {
			content = plannerFirstNonEmpty(content, "") + " · 原因：" + strings.TrimSpace(after.CancelReason)
		}
		appendActivity("status_changed", title, strings.Trim(content, " ·"))
	}
	if before.Bucket != after.Bucket && after.EntryType != models.PlannerEntryEvent {
		appendActivity("bucket_changed", "调整了分区", fmt.Sprintf("%s -> %s", bucketLabelForActivity(before.Bucket), bucketLabelForActivity(after.Bucket)))
	}
	if before.Title != after.Title || before.Detail != after.Detail || before.Notes != after.Notes {
		summary := after.Title
		if before.Title != after.Title {
			summary = fmt.Sprintf("%s -> %s", plannerFirstNonEmpty(before.Title, "未命名"), plannerFirstNonEmpty(after.Title, "未命名"))
		}
		appendActivity("content_updated", "更新了内容", summary)
	}
	return items
}

func statusLabelForActivity(status string) string {
	switch normalizePlannerStatus(status) {
	case plannerStatusInProgress:
		return "进行中"
	case plannerStatusDone:
		return "已完成"
	case plannerStatusCancelled:
		return "已取消"
	default:
		return "未开始"
	}
}

func bucketLabelForActivity(bucket string) string {
	switch bucket {
	case models.PlannerBucketInbox:
		return "收件箱"
	case models.PlannerBucketSomeday:
		return "放一放"
	default:
		return "计划中"
	}
}

func plannerTaskSyntheticActivities(task *models.PlannerTask) []*plannerTaskActivityItem {
	items := []*plannerTaskActivityItem{
		{
			ID:           "synthetic-created",
			Source:       "system",
			ActivityType: "created",
			Title:        plannerTaskLifecycleTitle(task),
			Content:      plannerTaskLifecycleContent(task),
			CreatedAt:    task.CreatedAt,
		},
	}
	if task.LastPostponedAt != nil {
		items = append(items, &plannerTaskActivityItem{
			ID:           "synthetic-postpone",
			Source:       "system",
			ActivityType: "schedule_changed",
			Title:        "最近一次顺延",
			Content:      plannerFirstNonEmpty(task.LastPostponeReason, "已调整日期"),
			CreatedAt:    *task.LastPostponedAt,
		})
	}
	if task.CompletedAt != nil {
		title := "完成了这条记录"
		content := statusLabelForActivity(task.Status)
		if normalizePlannerStatus(task.Status) == plannerStatusCancelled {
			title = "取消了这条记录"
			content = plannerFirstNonEmpty(strings.TrimSpace(task.CancelReason), content)
		}
		items = append(items, &plannerTaskActivityItem{
			ID:           "synthetic-completed",
			Source:       "system",
			ActivityType: "status_changed",
			Title:        title,
			Content:      content,
			CreatedAt:    *task.CompletedAt,
		})
	}
	return items
}

func (h *PlannerHandler) ListTaskActivities(c *gin.Context) {
	_, task, ok := h.loadTask(c)
	if !ok {
		return
	}
	activities, err := h.db.ListPlannerTaskActivities(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取生命周期失败", "code": 500})
		return
	}
	comments, err := h.db.ListPlannerTaskComments(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论失败", "code": 500})
		return
	}
	items := make([]*plannerTaskActivityItem, 0, len(activities)+len(comments)+3)
	if len(activities) == 0 {
		items = append(items, plannerTaskSyntheticActivities(task)...)
	}
	for _, activity := range activities {
		items = append(items, &plannerTaskActivityItem{
			ID:           activity.ID,
			Source:       "system",
			ActivityType: activity.ActivityType,
			Title:        activity.Title,
			Content:      activity.Content,
			CreatedAt:    activity.CreatedAt,
		})
	}
	for _, comment := range comments {
		items = append(items, &plannerTaskActivityItem{
			ID:           "comment-" + comment.ID,
			Source:       "comment",
			ActivityType: "comment",
			Title:        plannerFirstNonEmpty(comment.Author, "我") + " 留下了一条进展",
			Content:      comment.Content,
			CreatedAt:    comment.CreatedAt,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	c.JSON(http.StatusOK, gin.H{"code": 0, "activities": items})
}

func buildPlannerICS(task *models.PlannerTask) []byte {
	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//DevTools//Planner//CN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:planner-%s@devtools\r\n", task.ID))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", time.Now().UTC().Format("20060102T150405Z")))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICSLine(task.Title)))
	description := strings.TrimSpace(task.Detail)
	if strings.TrimSpace(task.Notes) != "" {
		if description != "" {
			description += "\\n\\n"
		}
		description += "备注: " + strings.TrimSpace(task.Notes)
	}
	if strings.TrimSpace(task.CancelReason) != "" {
		if description != "" {
			description += "\\n\\n"
		}
		description += "取消原因: " + strings.TrimSpace(task.CancelReason)
	}
	if description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICSLine(description)))
	}
	if task.RemindAt != nil {
		start := task.RemindAt.In(time.Local)
		end := start.Add(30 * time.Minute)
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", start.Format("20060102T150405")))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", end.Format("20060102T150405")))
		if rrule := plannerICSRRULE(task); rrule != "" {
			sb.WriteString(fmt.Sprintf("RRULE:%s\r\n", rrule))
		}
		sb.WriteString("BEGIN:VALARM\r\n")
		sb.WriteString("TRIGGER:PT0M\r\n")
		sb.WriteString("ACTION:DISPLAY\r\n")
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICSLine(task.Title)))
		sb.WriteString("END:VALARM\r\n")
	} else {
		planned := parsePlannerDate(task.PlannedFor)
		sb.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", strings.ReplaceAll(planned, "-", "")))
	}
	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")
	return []byte(sb.String())
}

func escapeICSLine(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, ";", "\\;")
	return value
}

func plannerCalendarFilename(task *models.PlannerTask) string {
	title := regexp.MustCompile(`[^a-zA-Z0-9\p{Han}_-]+`).ReplaceAllString(task.Title, "-")
	title = strings.Trim(title, "-")
	if title == "" {
		title = "planner"
	}
	return fmt.Sprintf("%s-%s.ics", task.Kind, title)
}

func plannerRepeatSummary(task *models.PlannerTask) string {
	if task == nil || task.RemindAt == nil {
		return ""
	}
	timePart := task.RemindAt.In(time.Local).Format("15:04")
	untilPart := ""
	if task.RepeatUntil != nil {
		untilPart = "，截止 " + task.RepeatUntil.In(time.Local).Format("2006-01-02 15:04")
	}
	switch normalizePlannerRepeatType(task.RepeatType) {
	case "daily":
		interval := max(1, task.RepeatInterval)
		if interval == 1 {
			return "每天 " + timePart + untilPart
		}
		return fmt.Sprintf("每 %d 天 %s%s", interval, timePart, untilPart)
	case "weekdays":
		return "工作日 " + timePart + untilPart
	case "weekly":
		interval := max(1, task.RepeatInterval)
		weekday := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[task.RemindAt.In(time.Local).Weekday()]
		if interval == 1 {
			return weekday + " " + timePart + untilPart
		}
		return fmt.Sprintf("每 %d 周%s %s%s", interval, weekday, timePart, untilPart)
	case "monthly":
		interval := max(1, task.RepeatInterval)
		day := task.RemindAt.In(time.Local).Day()
		if interval == 1 {
			return fmt.Sprintf("每月 %d 日 %s%s", day, timePart, untilPart)
		}
		return fmt.Sprintf("每 %d 月 %d 日 %s%s", interval, day, timePart, untilPart)
	default:
		return ""
	}
}

func plannerICSRRULE(task *models.PlannerTask) string {
	if task == nil || task.RemindAt == nil {
		return ""
	}
	repeatType := normalizePlannerRepeatType(task.RepeatType)
	if repeatType == "none" {
		return ""
	}
	interval := normalizePlannerRepeatInterval(task.RepeatInterval)
	parts := []string{}
	switch repeatType {
	case "daily":
		parts = append(parts, "FREQ=DAILY")
	case "weekdays":
		parts = append(parts, "FREQ=WEEKLY", "BYDAY=MO,TU,WE,TH,FR")
	case "weekly":
		weekdayMap := []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}
		parts = append(parts, "FREQ=WEEKLY", "BYDAY="+weekdayMap[task.RemindAt.In(time.Local).Weekday()])
	case "monthly":
		parts = append(parts, "FREQ=MONTHLY", fmt.Sprintf("BYMONTHDAY=%d", task.RemindAt.In(time.Local).Day()))
	default:
		return ""
	}
	if interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", interval))
	}
	if task.RepeatUntil != nil {
		parts = append(parts, "UNTIL="+task.RepeatUntil.In(time.Local).UTC().Format("20060102T150405Z"))
	}
	return strings.Join(parts, ";")
}

func plannerNextMonthlyReminder(current time.Time, interval int) time.Time {
	anchor := current.In(time.Local)
	anchorDay := anchor.Day()
	cursor := time.Date(anchor.Year(), anchor.Month(), 1, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), time.Local)
	for i := 0; i < 1024; i++ {
		cursor = cursor.AddDate(0, interval, 0)
		candidate := time.Date(cursor.Year(), cursor.Month(), anchorDay, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), time.Local)
		if candidate.Month() == cursor.Month() {
			return candidate
		}
	}
	return anchor
}

func plannerNextReminderAfter(task *models.PlannerTask, now time.Time) *time.Time {
	if task == nil || task.RemindAt == nil {
		return nil
	}
	repeatType := normalizePlannerRepeatType(task.RepeatType)
	if repeatType == "none" {
		return nil
	}
	interval := normalizePlannerRepeatInterval(task.RepeatInterval)
	current := task.RemindAt.In(time.Local)
	until := task.RepeatUntil
	for i := 0; i < 1024; i++ {
		switch repeatType {
		case "daily":
			current = current.AddDate(0, 0, interval)
		case "weekdays":
			for {
				current = current.AddDate(0, 0, 1)
				weekday := current.Weekday()
				if weekday >= time.Monday && weekday <= time.Friday {
					break
				}
			}
		case "weekly":
			current = current.AddDate(0, 0, 7*interval)
		case "monthly":
			current = plannerNextMonthlyReminder(current, interval)
		default:
			return nil
		}
		if until != nil && current.After(until.In(time.Local)) {
			return nil
		}
		if current.After(now) {
			next := current
			return &next
		}
	}
	return nil
}

func (h *PlannerHandler) DownloadCalendar(c *gin.Context) {
	_, task, ok := h.loadTask(c)
	if !ok {
		return
	}
	content := buildPlannerICS(task)
	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", plannerCalendarFilename(task)))
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", content)
}
