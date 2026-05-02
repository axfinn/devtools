package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

func (h *PlannerHandler) AIParse(c *gin.Context) {
	if _, ok := h.loadProfileByAccess(c, c.Param("id")); !ok {
		return
	}
	var req plannerAIParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入要整理的内容", "code": 400})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入要整理的内容", "code": 400})
		return
	}
	defaultKind := normalizePlannerKind(req.DefaultKind)
	tasks, provider, err := h.parsePlannerText(text, defaultKind)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "provider": provider, "tasks": tasks})
}

func (h *PlannerHandler) AIAdvise(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	var req plannerAICoachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}
	kind := normalizePlannerKind(plannerFirstNonEmpty(req.Kind, plannerModeDefault(time.Now())))
	tasks, err := h.db.ListPlannerTasksByProfile(profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取事项失败", "code": 500})
		return
	}
	board := buildPlannerBoard(tasks, kind, time.Now())
	advice, provider, err := h.advisePlanner(profile, board, strings.TrimSpace(req.Mode), strings.TrimSpace(req.Text))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "provider": provider, "advice": advice})
}

func (h *PlannerHandler) parsePlannerText(text, defaultKind string) ([]plannerAITask, string, error) {
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	prompt := fmt.Sprintf(`请把下面的中文内容整理成 JSON 数组，每个元素包含字段：
kind(work/life)、entry_type(task/event)、bucket(inbox/planned/someday)、title、detail、notes、priority(low/medium/high)、status(open/in_progress/done/cancelled)、planned_for(YYYY-MM-DD)、remind_at(YYYY-MM-DDTHH:MM，可为空)、cancel_reason(可为空)、raw_text(原文)。

规则：
1. 只返回 JSON 数组，不要解释。
2. 默认 kind 为 %s。
3. title 要整理成简洁、非口述的表达，去掉"然后"、"就是"、"那个"、"记得提醒我"、"的时候"等口语词。title 长度不超过 30 字。
4. raw_text 必须填写原文对应段落，不做任何修改。
5. 如果是明确时间发生的安排，如会议、复诊、预约、航班、出发，请优先标为 event。
6. 如果只是先记下来、回头处理、尚未排期，请放到 inbox。
7. 如果是以后再说、周末、有空再做，请优先放到 someday。
8. 用户说了明确日期就要写入 planned_for；用户说了明确钟点时间就必须写入 remind_at。
9. 对 event 来说，只要原文出现具体时间点，remind_at 不要留空。
10. 如果没有明确日期，planned_for 可以用今天。
11. 不要编造不存在的信息。

原始内容：
%s`, defaultKind, text)

	reqBody := map[string]interface{}{
		"model": plannerFirstNonEmpty(h.cfg.MiniMax.Model, "abab6.5s-chat"),
		"messages": []map[string]string{
			{"role": "system", "content": "你是智能整理助手。当前日期是2026年4月25日，日期必须使用2026年，不要使用训练数据中的旧日期。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", plannerMiniMaxAPIURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("planner ai parse: build request failed, fallback enabled: %v", err)
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.MiniMax.APIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("planner ai parse: minimax request failed, fallback enabled: %v", err)
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("planner ai parse: minimax returned %d, fallback enabled", resp.StatusCode)
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		log.Printf("planner ai parse: invalid minimax response, fallback enabled: %v", err)
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	choices, _ := raw["choices"].([]interface{})
	if len(choices) == 0 {
		log.Printf("planner ai parse: minimax returned no choices, fallback enabled")
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	first, _ := choices[0].(map[string]interface{})
	msg, _ := first["message"].(map[string]interface{})
	content, _ := msg["content"].(string)
	payload, ok := extractJSONPayload(content)
	if !ok {
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	var tasks []plannerAITask
	if err := json.Unmarshal([]byte(payload), &tasks); err != nil {
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	for i := range tasks {
		tasks[i] = normalizePlannerAITask(tasks[i], defaultKind)
	}
	return tasks, "minimax", nil
}

func (h *PlannerHandler) advisePlanner(profile *models.PlannerProfile, board plannerBoardResponse, mode, text string) (plannerAICoachResponse, string, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode != "summary" {
		mode = "plan"
	}
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}

	prompt := fmt.Sprintf(`你是一个只做建议、不做操作的事项管理顾问。请基于下面的信息输出一个 JSON 对象，字段固定为：
summary(string)
insights(string[])
suggestions(string[])

约束：
1. 只返回 JSON，不要附带解释。
2. 不要替用户执行操作，不要假装已完成。
3. 总结模式更强调复盘和观察；规划模式更强调下一步建议。
4. suggestions 最多 5 条，每条要具体可执行。
5. 不要重复同义句。

当前模式: %s
档案名: %s
事项分区: %s
用户补充: %s

当前看板摘要：
%s
`, mode, profile.Name, map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[board.Kind], plannerFirstNonEmpty(text, "无"), plannerBoardSnapshot(board))

	reqBody := map[string]interface{}{
		"model": plannerFirstNonEmpty(h.cfg.MiniMax.Model, "abab6.5s-chat"),
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个事项管理顾问，只提供总结、取舍建议和规划建议。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", plannerMiniMaxAPIURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("planner ai advise: build request failed, fallback enabled: %v", err)
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.MiniMax.APIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("planner ai advise: minimax request failed, fallback enabled: %v", err)
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("planner ai advise: minimax returned %d, fallback enabled", resp.StatusCode)
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		log.Printf("planner ai advise: invalid minimax response, fallback enabled: %v", err)
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	choices, _ := raw["choices"].([]interface{})
	if len(choices) == 0 {
		log.Printf("planner ai advise: minimax returned no choices, fallback enabled")
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	first, _ := choices[0].(map[string]interface{})
	msg, _ := first["message"].(map[string]interface{})
	content, _ := msg["content"].(string)
	payload, ok := extractJSONPayload(content)
	if !ok {
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	var advice plannerAICoachResponse
	if err := json.Unmarshal([]byte(payload), &advice); err != nil {
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	if strings.TrimSpace(advice.Summary) == "" {
		return fallbackPlannerAdvice(profile, board, mode, text), "fallback", nil
	}
	if advice.Insights == nil {
		advice.Insights = []string{}
	}
	if advice.Suggestions == nil {
		advice.Suggestions = []string{}
	}
	return advice, "minimax", nil
}

func fallbackPlannerAdvice(profile *models.PlannerProfile, board plannerBoardResponse, mode, text string) plannerAICoachResponse {
	label := map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[board.Kind]
	openCount := board.Counts[plannerStatusOpen] + board.Counts[plannerStatusInProgress]
	summary := fmt.Sprintf("%s档案「%s」当前还有 %d 件未完成事项，今天主线 %d 件。", label, profile.Name, openCount, board.Focus.TodayPrimaryCount)
	if mode == "summary" {
		summary = fmt.Sprintf("%s档案「%s」的记录显示：未完成 %d 件，收件箱 %d 条，已完成 %d 件。", label, profile.Name, openCount, board.Recovery.InboxOpen, board.Recovery.DoneToday)
	}
	insights := []string{
		board.Focus.Message,
		board.Recovery.Message,
	}
	if board.Counts["event_open"] > 0 {
		insights = append(insights, fmt.Sprintf("还有 %d 个待发生事件，建议先检查是否都带了明确时间。", board.Counts["event_open"]))
	}
	if board.Counts["rolled_over"] > 0 {
		insights = append(insights, fmt.Sprintf("有 %d 件事项已经顺延到今天，说明计划密度偏高。", board.Counts["rolled_over"]))
	}
	suggestions := make([]string, 0, 5)
	if board.Focus.NeedsTrim {
		suggestions = append(suggestions, fmt.Sprintf("先把今天时间线压到 %d 件主事项以内，其余移到收件箱或改到明天。", board.Focus.TodayPrimaryLimit))
	}
	if len(board.InboxItems) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("先处理收件箱里的前 %d 条，把它们分成计划中或放一放。", minPlannerInt(len(board.InboxItems), 3)))
	}
	if len(board.EventGroups) > 0 {
		suggestions = append(suggestions, "检查最近事件是否都带提醒时间，避免事件存在但没有触发点。")
	}
	if len(board.Groups) > 0 {
		suggestions = append(suggestions, "从今天时间线里只选一件最高优先级事项推进到完成，先建立正反馈。")
	}
	if strings.TrimSpace(text) != "" {
		suggestions = append(suggestions, fmt.Sprintf("结合你的补充“%s”，优先把它归到今天唯一最重要的下一步里。", text))
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "当前事项不多，可以先补充未来两天最重要的安排，让节奏更稳定。")
	}
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	return plannerAICoachResponse{
		Summary:     summary,
		Insights:    insights,
		Suggestions: suggestions,
	}
}

func plannerBoardSnapshot(board plannerBoardResponse) string {
	lines := []string{
		fmt.Sprintf("未完成: %d", board.Counts[plannerStatusOpen]+board.Counts[plannerStatusInProgress]),
		fmt.Sprintf("已完成: %d", board.Counts[plannerStatusDone]),
		fmt.Sprintf("已取消: %d", board.Counts[plannerStatusCancelled]),
		fmt.Sprintf("收件箱: %d", len(board.InboxItems)),
		fmt.Sprintf("放一放: %d", len(board.SomedayItems)),
		fmt.Sprintf("事件: %d", board.Counts["event_open"]),
		fmt.Sprintf("顺延到今天: %d", board.Counts["rolled_over"]),
		fmt.Sprintf("聚焦提示: %s", board.Focus.Message),
		fmt.Sprintf("恢复提示: %s", board.Recovery.Message),
	}
	appendItems := func(title string, items []*plannerTimelineItem) {
		if len(items) == 0 {
			return
		}
		limit := minPlannerInt(len(items), 4)
		parts := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			parts = append(parts, items[i].Title)
		}
		lines = append(lines, fmt.Sprintf("%s: %s", title, strings.Join(parts, " / ")))
	}
	for _, group := range board.Groups {
		appendItems("时间线-"+group.Label, group.Items)
		if len(lines) > 14 {
			break
		}
	}
	for _, group := range board.EventGroups {
		appendItems("事件-"+group.Label, group.Items)
		if len(lines) > 18 {
			break
		}
	}
	appendItems("收件箱样本", board.InboxItems)
	appendItems("最近完成", board.RecentItems)
	return strings.Join(lines, "\n")
}

func minPlannerInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizePlannerAITask(task plannerAITask, defaultKind string) plannerAITask {
	rawPlannedFor := strings.TrimSpace(task.PlannedFor)
	rawRemindAt := strings.TrimSpace(task.RemindAt)
	scheduleSource := strings.Join([]string{
		strings.TrimSpace(task.Title),
		strings.TrimSpace(task.Detail),
		strings.TrimSpace(task.Notes),
	}, " ")
	inferred := plannerExtractScheduleFromText(scheduleSource, time.Now())
	task.Kind = normalizePlannerKind(plannerFirstNonEmpty(task.Kind, defaultKind))
	task.EntryType = normalizePlannerEntryType(task.EntryType)
	task.Bucket = normalizePlannerBucket(task.Bucket, task.EntryType)
	task.Priority = normalizePlannerPriority(task.Priority)
	task.Status = normalizePlannerStatus(task.Status)
	if rawPlannedFor != "" {
		task.PlannedFor = parsePlannerDate(rawPlannedFor)
	} else if inferred.HasDate || inferred.HasTime {
		task.PlannedFor = inferred.PlannedFor
	} else {
		task.PlannedFor = plannerToday()
	}
	if parsedRemindAt := parsePlannerDateTime(rawRemindAt); parsedRemindAt != nil {
		task.RemindAt = plannerFormatDateTimeValue(*parsedRemindAt)
	} else if inferred.HasTime {
		task.RemindAt = inferred.RemindAt
	} else {
		task.RemindAt = ""
	}
	if task.EntryType == models.PlannerEntryEvent && task.Bucket != models.PlannerBucketPlanned {
		task.Bucket = models.PlannerBucketPlanned
	}
	// Truncate fields to prevent frontend overflow
	task.Title = truncatePlannerText(task.Title, 30)
	task.Detail = truncatePlannerText(task.Detail, 200)
	task.Notes = truncatePlannerText(task.Notes, 500)
	task.RawText = truncatePlannerText(task.RawText, 1000)

	if strings.TrimSpace(task.Title) == "" {
		task.Title = "待整理事项"
	}
	if strings.TrimSpace(task.Intent) == "" {
		task.Intent = strings.TrimSpace(task.Notes)
	}
	if strings.TrimSpace(task.EnergyLevel) == "" {
		task.EnergyLevel = guessEnergyLevel(strings.Join([]string{task.Title, task.Detail, task.Notes}, " "))
	}
	return task
}

func fallbackParsePlannerText(text, defaultKind string) []plannerAITask {
	segments := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '。' || r == ';' || r == '；'
	})
	result := make([]plannerAITask, 0)
	for _, segment := range segments {
		item := strings.TrimSpace(segment)
		if item == "" {
			continue
		}
		schedule := plannerExtractScheduleFromText(item, time.Now())
		lower := strings.ToLower(item)
		kind := defaultKind
		if strings.Contains(lower, "生活") || strings.Contains(item, "买菜") || strings.Contains(item, "买水果") || strings.Contains(item, "家里") || strings.Contains(item, "散步") {
			kind = plannerKindLife
		}
		if strings.Contains(lower, "工作") || strings.Contains(item, "开会") || strings.Contains(item, "需求") || strings.Contains(item, "上线") || strings.Contains(item, "联调") {
			kind = plannerKindWork
		}

		entryType := models.PlannerEntryTask
		bucket := models.PlannerBucketPlanned
		if looksLikeEvent(item) || schedule.HasTime {
			entryType = models.PlannerEntryEvent
			bucket = models.PlannerBucketPlanned
		} else if strings.Contains(item, "回头") || strings.Contains(item, "记一下") || strings.Contains(item, "先记") {
			bucket = models.PlannerBucketInbox
		} else if strings.Contains(item, "以后") || strings.Contains(item, "有空") || strings.Contains(item, "周末") {
			bucket = models.PlannerBucketSomeday
		}

		priority := "medium"
		if strings.Contains(item, "尽快") || strings.Contains(item, "今天") || strings.Contains(item, "马上") {
			priority = "high"
		}
		result = append(result, plannerAITask{
			Kind:        normalizePlannerKind(kind),
			EntryType:   entryType,
			Bucket:      bucket,
			Title:       item,
			Detail:      item,
			RawText:     item,
			Priority:    priority,
			Status:      plannerStatusOpen,
			PlannedFor:  plannerFirstNonEmpty(schedule.PlannedFor, plannerToday()),
			RemindAt:    schedule.RemindAt,
			EnergyLevel: guessEnergyLevel(item),
		})
	}
	if len(result) == 0 {
		result = append(result, plannerAITask{
			Kind:       normalizePlannerKind(defaultKind),
			EntryType:  models.PlannerEntryTask,
			Bucket:     models.PlannerBucketInbox,
			Title:      strings.TrimSpace(text),
			Detail:     strings.TrimSpace(text),
			RawText:    strings.TrimSpace(text),
			Priority:   "medium",
			Status:     plannerStatusOpen,
			PlannedFor: plannerToday(),
		})
	}
	return result
}

func looksLikeEvent(text string) bool {
	eventKeywords := []string{"会议", "开会", "复诊", "预约", "出发", "航班", "车票", "电影", "聚餐", "面试", "回诊", "课程", "直播", "电话"}
	timePatterns := []string{`\d{1,2}:\d{2}`, `\d{1,2}点`, `周[一二三四五六日天]`, `\d{1,2}月\d{1,2}日`, `\d{1,2}号`}
	for _, keyword := range eventKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	for _, pattern := range timePatterns {
		if regexp.MustCompile(pattern).MatchString(text) {
			return true
		}
	}
	return false
}

func plannerFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncatePlannerText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= maxLen {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxLen])) + "…"
}

func guessEnergyLevel(text string) string {
	lower := strings.ToLower(text)
	deepWords := []string{"写", "方案", "设计", "代码", "分析", "报告", "研究", "规划", "文档", "架构", "算法", "review", "评审"}
	shallowWords := []string{"回", "邮件", "发票", "报销", "填表", "签到", "打卡", "审批", "整理", "同步", "沟通", "周报"}
	errandWords := []string{"买", "取", "快递", "超市", "药店", "医院", "银行", "接", "送", "寄", "搬", "修", "理发", "加油"}
	creativeWords := []string{"想", "创意", "brainstorm", "头脑风暴", "构思", "探索", "灵感", "idea", "点子", "方向"}

	score := map[string]int{}
	for _, w := range deepWords {
		if strings.Contains(lower, w) { score["deep"]++ }
	}
	for _, w := range shallowWords {
		if strings.Contains(lower, w) { score["shallow"]++ }
	}
	for _, w := range errandWords {
		if strings.Contains(lower, w) { score["errand"]++ }
	}
	for _, w := range creativeWords {
		if strings.Contains(lower, w) { score["creative"]++ }
	}

	best := ""
	bestScore := 0
	for _, level := range []string{"deep", "shallow", "errand", "creative"} {
		if score[level] > bestScore {
			best = level
			bestScore = score[level]
		}
	}
	return best
}
