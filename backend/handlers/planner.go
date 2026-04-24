package handlers

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

const (
	plannerKindWork = "work"
	plannerKindLife = "life"

	plannerStatusOpen       = "open"
	plannerStatusInProgress = "in_progress"
	plannerStatusDone       = "done"
	plannerStatusCancelled  = "cancelled"
)

var plannerMiniMaxAPIURL = "https://api.minimax.chat/v1/text/chatcompletion_v2"

type PlannerHandler struct {
	db                *models.DB
	cfg               *config.Config
	adminPassword     string
	defaultExpiresDay int
	maxExpiresDay     int
	smtpHost          string
	smtpPort          int
	smtpUser          string
	smtpPass          string
}

type plannerProfileResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	NotifyEmail string                 `json:"notify_email"`
	ExpiresAt   *time.Time             `json:"expires_at"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Stats       map[string]int         `json:"stats,omitempty"`
	ModeDefault string                 `json:"mode_default,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

type plannerTimelineItem struct {
	*models.PlannerTask
	DisplayDate        string     `json:"display_date"`
	DisplayLabel       string     `json:"display_label"`
	IsRolledOver       bool       `json:"is_rolled_over"`
	IsToday            bool       `json:"is_today"`
	OverdueDays        int        `json:"overdue_days"`
	CalendarAvailable  bool       `json:"calendar_available"`
	CommentCount       int        `json:"comment_count"`
	LastCommentPreview string     `json:"last_comment_preview"`
	LastCommentAt      *time.Time `json:"last_comment_at,omitempty"`
	TimeHint           string     `json:"time_hint"`
	EventPhase         string     `json:"event_phase"`
	NeedsClosure       bool       `json:"needs_closure"`
}

type plannerTimelineGroup struct {
	Date  string                 `json:"date"`
	Label string                 `json:"label"`
	Items []*plannerTimelineItem `json:"items"`
}

type plannerFocusSummary struct {
	TodayPrimaryLimit int                    `json:"today_primary_limit"`
	TodayPrimaryCount int                    `json:"today_primary_count"`
	NeedsTrim         bool                   `json:"needs_trim"`
	Message           string                 `json:"message"`
	Primary           *plannerTimelineItem   `json:"primary,omitempty"`
	Secondary         []*plannerTimelineItem `json:"secondary,omitempty"`
	NextEvent         *plannerTimelineItem   `json:"next_event,omitempty"`
}

type plannerRecoverySummary struct {
	DoneToday      int    `json:"done_today"`
	CancelledToday int    `json:"cancelled_today"`
	InboxOpen      int    `json:"inbox_open"`
	Message        string `json:"message"`
}

type plannerBoardResponse struct {
	Kind         string                  `json:"kind"`
	ProfileName  string                  `json:"profile_name"`
	Groups       []*plannerTimelineGroup `json:"groups"`
	EventGroups  []*plannerTimelineGroup `json:"event_groups"`
	InboxItems   []*plannerTimelineItem  `json:"inbox_items"`
	SomedayItems []*plannerTimelineItem  `json:"someday_items"`
	RecentItems  []*plannerTimelineItem  `json:"recent_items"`
	Counts       map[string]int          `json:"counts"`
	Focus        plannerFocusSummary     `json:"focus"`
	Recovery     plannerRecoverySummary  `json:"recovery"`
	ModeDefault  string                  `json:"mode_default"`
	ModeHint     string                  `json:"mode_hint"`
}

type plannerReviewResponse struct {
	Period      string                 `json:"period"`
	Label       string                 `json:"label"`
	Summary     string                 `json:"summary"`
	Stats       map[string]int         `json:"stats"`
	Wins        []string               `json:"wins"`
	Drifts      []string               `json:"drifts"`
	Suggestions []string               `json:"suggestions"`
	Highlights  []*plannerTimelineItem `json:"highlights"`
}

type createPlannerProfileRequest struct {
	Password    string `json:"password" binding:"required,min=4"`
	Name        string `json:"name"`
	NotifyEmail string `json:"notify_email"`
	ExpiresIn   int    `json:"expires_in"`
}

type loginPlannerProfileRequest struct {
	Password string `json:"password" binding:"required,min=4"`
}

type updatePlannerProfileRequest struct {
	Name        string `json:"name"`
	NotifyEmail string `json:"notify_email"`
	ExpiresIn   int    `json:"expires_in"`
}

type createPlannerTaskRequest struct {
	Kind           string `json:"kind"`
	EntryType      string `json:"entry_type"`
	Bucket         string `json:"bucket"`
	Title          string `json:"title" binding:"required"`
	Detail         string `json:"detail"`
	Notes          string `json:"notes"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	PlannedFor     string `json:"planned_for"`
	RemindAt       string `json:"remind_at"`
	RepeatType     string `json:"repeat_type"`
	RepeatInterval int    `json:"repeat_interval"`
	RepeatUntil    string `json:"repeat_until"`
	NotifyEmail    string `json:"notify_email"`
	CancelReason   string `json:"cancel_reason"`
	PostponeReason string `json:"postpone_reason"`
}

type createPlannerTaskBatchRequest struct {
	Tasks []createPlannerTaskRequest `json:"tasks"`
}

type updatePlannerTaskRequest struct {
	Kind           *string `json:"kind"`
	EntryType      *string `json:"entry_type"`
	Bucket         *string `json:"bucket"`
	Title          *string `json:"title"`
	Detail         *string `json:"detail"`
	Notes          *string `json:"notes"`
	Status         *string `json:"status"`
	Priority       *string `json:"priority"`
	PlannedFor     *string `json:"planned_for"`
	RemindAt       *string `json:"remind_at"`
	RepeatType     *string `json:"repeat_type"`
	RepeatInterval *int    `json:"repeat_interval"`
	RepeatUntil    *string `json:"repeat_until"`
	NotifyEmail    *string `json:"notify_email"`
	CancelReason   *string `json:"cancel_reason"`
	PostponeReason *string `json:"postpone_reason"`
}

type plannerCommentRequest struct {
	Author  string `json:"author"`
	Content string `json:"content" binding:"required"`
}

type plannerTaskActivityItem struct {
	ID           string    `json:"id"`
	Source       string    `json:"source"`
	ActivityType string    `json:"activity_type"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

type plannerAIParseRequest struct {
	Text        string `json:"text" binding:"required"`
	DefaultKind string `json:"default_kind"`
}

type plannerAICoachRequest struct {
	Kind string `json:"kind"`
	Mode string `json:"mode"`
	Text string `json:"text"`
}

type plannerAITask struct {
	Kind         string `json:"kind"`
	EntryType    string `json:"entry_type"`
	Bucket       string `json:"bucket"`
	Title        string `json:"title"`
	Detail       string `json:"detail"`
	Notes        string `json:"notes"`
	Priority     string `json:"priority"`
	Status       string `json:"status"`
	PlannedFor   string `json:"planned_for"`
	RemindAt     string `json:"remind_at"`
	CancelReason string `json:"cancel_reason"`
}

type plannerAICoachResponse struct {
	Summary     string   `json:"summary"`
	Insights    []string `json:"insights"`
	Suggestions []string `json:"suggestions"`
}

func NewPlannerHandler(db *models.DB, cfg *config.Config) *PlannerHandler {
	plannerCfg := cfg.Planner
	if plannerCfg.DefaultExpiresDays <= 0 {
		plannerCfg.DefaultExpiresDays = 365
	}
	if plannerCfg.MaxExpiresDays <= 0 {
		plannerCfg.MaxExpiresDays = 3650
	}
	if plannerCfg.SMTPPort == 0 {
		plannerCfg.SMTPPort = 465
	}
	return &PlannerHandler{
		db:                db,
		cfg:               cfg,
		adminPassword:     plannerCfg.AdminPassword,
		defaultExpiresDay: plannerCfg.DefaultExpiresDays,
		maxExpiresDay:     plannerCfg.MaxExpiresDays,
		smtpHost:          plannerCfg.SMTPHost,
		smtpPort:          plannerCfg.SMTPPort,
		smtpUser:          plannerCfg.SMTPUser,
		smtpPass:          plannerCfg.SMTPPass,
	}
}

func plannerPasswordIndex(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func plannerCreatorKey(c *gin.Context) string {
	key := strings.TrimSpace(c.Query("creator_key"))
	if key != "" {
		return key
	}
	return strings.TrimSpace(c.GetHeader("X-Creator-Key"))
}

func normalizePlannerKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case plannerKindLife:
		return plannerKindLife
	default:
		return plannerKindWork
	}
}

func normalizePlannerStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case plannerStatusInProgress:
		return plannerStatusInProgress
	case plannerStatusDone:
		return plannerStatusDone
	case plannerStatusCancelled:
		return plannerStatusCancelled
	default:
		return plannerStatusOpen
	}
}

func normalizePlannerPriority(priority string) string {
	switch strings.TrimSpace(strings.ToLower(priority)) {
	case "low":
		return "low"
	case "high":
		return "high"
	default:
		return "medium"
	}
}

func normalizePlannerEntryType(entryType string) string {
	switch strings.TrimSpace(strings.ToLower(entryType)) {
	case models.PlannerEntryEvent:
		return models.PlannerEntryEvent
	default:
		return models.PlannerEntryTask
	}
}

func normalizePlannerBucket(bucket, entryType string) string {
	switch strings.TrimSpace(strings.ToLower(bucket)) {
	case models.PlannerBucketInbox:
		return models.PlannerBucketInbox
	case models.PlannerBucketSomeday:
		return models.PlannerBucketSomeday
	case models.PlannerBucketPlanned:
		return models.PlannerBucketPlanned
	}
	if normalizePlannerEntryType(entryType) == models.PlannerEntryEvent {
		return models.PlannerBucketPlanned
	}
	return models.PlannerBucketPlanned
}

func normalizePlannerRepeatType(repeatType string) string {
	switch strings.TrimSpace(strings.ToLower(repeatType)) {
	case "daily":
		return "daily"
	case "weekdays":
		return "weekdays"
	case "weekly":
		return "weekly"
	case "monthly":
		return "monthly"
	default:
		return "none"
	}
}

func normalizePlannerRepeatInterval(interval int) int {
	switch {
	case interval <= 0:
		return 1
	case interval > 365:
		return 365
	default:
		return interval
	}
}

func plannerToday() string {
	return time.Now().Format("2006-01-02")
}

func parsePlannerDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return plannerToday()
	}
	layouts := []string{"2006-01-02", "2006/01/02", "2006.01.02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return plannerToday()
}

func parsePlannerDateTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			tt := t
			return &tt
		}
	}
	return nil
}

func plannerModeDefault(now time.Time) string {
	weekday := now.Weekday()
	if weekday >= time.Monday && weekday <= time.Friday {
		hour := now.Hour()
		if hour >= 9 && hour < 18 {
			return plannerKindWork
		}
	}
	return plannerKindLife
}

func plannerModeHint(now time.Time) string {
	if plannerModeDefault(now) == plannerKindWork {
		return "当前是工作时段，默认进入工作模式"
	}
	return "当前是下班或休息时段，默认进入生活模式"
}

func (h *PlannerHandler) requireAdmin(c *gin.Context) bool {
	password := strings.TrimSpace(c.Query("admin_password"))
	if password == "" {
		password = strings.TrimSpace(c.GetHeader("X-Admin-Password"))
	}
	if h.adminPassword == "" || password != h.adminPassword {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限", "code": 403})
		return false
	}
	return true
}

func (h *PlannerHandler) loadProfileByAccess(c *gin.Context, profileID string) (*models.PlannerProfile, bool) {
	profile, err := h.db.GetPlannerProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return nil, false
	}
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "档案已过期", "code": 403})
		return nil, false
	}
	if key := plannerCreatorKey(c); key != "" && utils.VerifyPassword(key, profile.CreatorKey) {
		return profile, true
	}
	if pwd := strings.TrimSpace(getPassword(c)); pwd != "" && plannerPasswordIndex(pwd) == profile.PasswordIndex {
		return profile, true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "无权限", "code": 403})
	return nil, false
}

func (h *PlannerHandler) loadProfileByCreator(c *gin.Context, profileID string) (*models.PlannerProfile, bool) {
	profile, err := h.db.GetPlannerProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return nil, false
	}
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "档案已过期", "code": 403})
		return nil, false
	}
	key := plannerCreatorKey(c)
	if key == "" || !utils.VerifyPassword(key, profile.CreatorKey) {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要创建者密钥", "code": 403, "requires_creator_key": true})
		return nil, false
	}
	return profile, true
}

func (h *PlannerHandler) loadTask(c *gin.Context) (*models.PlannerProfile, *models.PlannerTask, bool) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return nil, nil, false
	}
	task, err := h.db.GetPlannerTask(c.Param("taskId"))
	if err != nil || task.ProfileID != profile.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "事项不存在", "code": 404})
		return nil, nil, false
	}
	return profile, task, true
}

func (h *PlannerHandler) CreateProfile(c *gin.Context) {
	var req createPlannerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供至少 4 位密码", "code": 400})
		return
	}
	passwordIndex := plannerPasswordIndex(req.Password)
	existing, _ := h.db.GetPlannerProfileByPasswordIndex(passwordIndex)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该密码已存在档案，请直接登录", "code": 409})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "我的事项档案"
	}
	creatorKey := utils.GenerateHexKey(16)
	hashedCreatorKey, _ := utils.HashPassword(creatorKey)
	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = h.defaultExpiresDay
	}
	if expiresIn > h.maxExpiresDay {
		expiresIn = h.maxExpiresDay
	}
	expiresAt := time.Now().AddDate(0, 0, expiresIn)

	profile := &models.PlannerProfile{
		PasswordIndex: passwordIndex,
		CreatorKey:    hashedCreatorKey,
		Name:          name,
		NotifyEmail:   strings.TrimSpace(req.NotifyEmail),
	}
	if err := h.db.CreatePlannerProfile(profile, &expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"id":           profile.ID,
		"creator_key":  creatorKey,
		"name":         profile.Name,
		"notify_email": profile.NotifyEmail,
		"expires_at":   profile.ExpiresAt,
		"created_at":   profile.CreatedAt,
	})
}

func (h *PlannerHandler) LoginProfile(c *gin.Context) {
	var req loginPlannerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码", "code": 400})
		return
	}
	profile, err := h.db.GetPlannerProfileByPasswordIndex(plannerPasswordIndex(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "档案已过期", "code": 403})
		return
	}
	stats, _ := h.db.CountPlannerTasks(profile.ID)
	now := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"id":           profile.ID,
		"name":         profile.Name,
		"notify_email": profile.NotifyEmail,
		"expires_at":   profile.ExpiresAt,
		"created_at":   profile.CreatedAt,
		"updated_at":   profile.UpdatedAt,
		"stats":        stats,
		"mode_default": plannerModeDefault(now),
		"mode_hint":    plannerModeHint(now),
	})
}

func (h *PlannerHandler) GetProfile(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	stats, _ := h.db.CountPlannerTasks(profile.ID)
	now := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"profile": plannerProfileResponse{
			ID:          profile.ID,
			Name:        profile.Name,
			NotifyEmail: profile.NotifyEmail,
			ExpiresAt:   profile.ExpiresAt,
			CreatedAt:   profile.CreatedAt,
			UpdatedAt:   profile.UpdatedAt,
			Stats:       stats,
			ModeDefault: plannerModeDefault(now),
			Meta: map[string]interface{}{
				"mode_hint": plannerModeHint(now),
			},
		},
	})
}

func (h *PlannerHandler) UpdateProfile(c *gin.Context) {
	profile, ok := h.loadProfileByCreator(c, c.Param("id"))
	if !ok {
		return
	}
	var req updatePlannerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		profile.Name = name
	}
	profile.NotifyEmail = strings.TrimSpace(req.NotifyEmail)
	if req.ExpiresIn > 0 {
		days := req.ExpiresIn
		if days > h.maxExpiresDay {
			days = h.maxExpiresDay
		}
		expiresAt := time.Now().AddDate(0, 0, days)
		profile.ExpiresAt = &expiresAt
	}
	if err := h.db.UpdatePlannerProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	stats, _ := h.db.CountPlannerTasks(profile.ID)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"profile": plannerProfileResponse{
			ID:          profile.ID,
			Name:        profile.Name,
			NotifyEmail: profile.NotifyEmail,
			ExpiresAt:   profile.ExpiresAt,
			CreatedAt:   profile.CreatedAt,
			UpdatedAt:   profile.UpdatedAt,
			Stats:       stats,
		},
	})
}

func (h *PlannerHandler) DeleteProfile(c *gin.Context) {
	if _, ok := h.loadProfileByCreator(c, c.Param("id")); !ok {
		return
	}
	if err := h.db.DeletePlannerProfile(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

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
	if len(value) <= 48 {
		return value
	}
	return value[:48] + "..."
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
		CancelReason:       strings.TrimSpace(req.CancelReason),
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
kind(work/life)、entry_type(task/event)、bucket(inbox/planned/someday)、title、detail、notes、priority(low/medium/high)、status(open/in_progress/done/cancelled)、planned_for(YYYY-MM-DD)、remind_at(YYYY-MM-DDTHH:MM，可为空)、cancel_reason(可为空)。

规则：
1. 只返回 JSON 数组，不要解释。
2. 默认 kind 为 %s。
3. 如果是明确时间发生的安排，如会议、复诊、预约、航班、出发，请优先标为 event。
4. 如果只是先记下来、回头处理、尚未排期，请放到 inbox。
5. 如果是以后再说、周末、有空再做，请优先放到 someday。
6. 如果没有明确日期，planned_for 可以用今天。
7. 不要编造不存在的信息。

原始内容：
%s`, defaultKind, text)

	reqBody := map[string]interface{}{
		"model": plannerFirstNonEmpty(h.cfg.MiniMax.Model, "abab6.5s-chat"),
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个只负责管理工作事项、生活事项、收件箱和事件的智能整理助手。"},
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
	task.Kind = normalizePlannerKind(plannerFirstNonEmpty(task.Kind, defaultKind))
	task.EntryType = normalizePlannerEntryType(task.EntryType)
	task.Bucket = normalizePlannerBucket(task.Bucket, task.EntryType)
	task.Priority = normalizePlannerPriority(task.Priority)
	task.Status = normalizePlannerStatus(task.Status)
	task.PlannedFor = parsePlannerDate(task.PlannedFor)
	if strings.TrimSpace(task.Title) == "" {
		task.Title = "待整理事项"
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
		if looksLikeEvent(item) {
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
			Kind:       normalizePlannerKind(kind),
			EntryType:  entryType,
			Bucket:     bucket,
			Title:      item,
			Detail:     item,
			Priority:   priority,
			Status:     plannerStatusOpen,
			PlannedFor: plannerToday(),
		})
	}
	if len(result) == 0 {
		result = append(result, plannerAITask{
			Kind:       normalizePlannerKind(defaultKind),
			EntryType:  models.PlannerEntryTask,
			Bucket:     models.PlannerBucketInbox,
			Title:      strings.TrimSpace(text),
			Detail:     strings.TrimSpace(text),
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

func (h *PlannerHandler) AdminList(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	var (
		items []*models.PlannerProfileSummary
		err   error
	)
	if keyword != "" {
		items, err = h.db.SearchPlannerProfiles(keyword, 50)
	} else {
		items, err = h.db.ListPlannerProfiles(100, 0)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败", "code": 500})
		return
	}
	total, _ := h.db.CountPlannerProfiles()
	c.JSON(http.StatusOK, gin.H{"code": 0, "total": total, "items": items})
}

func (h *PlannerHandler) AdminGet(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	profile, err := h.db.GetPlannerProfile(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	tasks, _ := h.db.ListPlannerTasksByProfile(profile.ID)
	stats, _ := h.db.CountPlannerTasks(profile.ID)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"profile": plannerProfileResponse{
			ID:          profile.ID,
			Name:        profile.Name,
			NotifyEmail: profile.NotifyEmail,
			ExpiresAt:   profile.ExpiresAt,
			CreatedAt:   profile.CreatedAt,
			UpdatedAt:   profile.UpdatedAt,
			Stats:       stats,
		},
		"tasks": tasks,
	})
}

func (h *PlannerHandler) AdminUpdate(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	profile, err := h.db.GetPlannerProfile(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "档案不存在", "code": 404})
		return
	}
	var req updatePlannerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		profile.Name = name
	}
	profile.NotifyEmail = strings.TrimSpace(req.NotifyEmail)
	if req.ExpiresIn > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresIn)
		profile.ExpiresAt = &expiresAt
	}
	if err := h.db.UpdatePlannerProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *PlannerHandler) AdminDelete(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}
	if err := h.db.DeletePlannerProfile(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *PlannerHandler) ProcessDueReminders() {
	tasks, err := h.db.ListPlannerTasksDueForReminder(time.Now(), 100)
	if err != nil || len(tasks) == 0 {
		return
	}
	for _, task := range tasks {
		profile, err := h.db.GetPlannerProfile(task.ProfileID)
		if err != nil {
			continue
		}
		recipients := plannerFirstNonEmpty(strings.TrimSpace(task.NotifyEmail), strings.TrimSpace(profile.NotifyEmail))
		if recipients == "" {
			continue
		}
		subject := fmt.Sprintf("%s提醒 [%s] %s", map[string]string{models.PlannerEntryTask: "事项", models.PlannerEntryEvent: "事件"}[task.EntryType], map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[task.Kind], task.Title)
		bodyLines := []string{
			fmt.Sprintf("档案: %s", profile.Name),
			fmt.Sprintf("类型: %s", map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[task.Kind]),
			fmt.Sprintf("条目: %s", map[string]string{models.PlannerEntryTask: "任务", models.PlannerEntryEvent: "事件"}[task.EntryType]),
			fmt.Sprintf("阶段: %s", map[string]string{models.PlannerBucketInbox: "收件箱", models.PlannerBucketPlanned: "计划中", models.PlannerBucketSomeday: "放一放"}[task.Bucket]),
			fmt.Sprintf("标题: %s", task.Title),
			fmt.Sprintf("状态: %s", task.Status),
			fmt.Sprintf("优先级: %s", task.Priority),
			fmt.Sprintf("计划日期: %s", task.PlannedFor),
		}
		if task.RemindAt != nil {
			bodyLines = append(bodyLines, fmt.Sprintf("提醒时间: %s", task.RemindAt.In(time.Local).Format("2006-01-02 15:04")))
		}
		if repeatSummary := plannerRepeatSummary(task); repeatSummary != "" {
			bodyLines = append(bodyLines, fmt.Sprintf("重复提醒: %s", repeatSummary))
		}
		if strings.TrimSpace(task.Detail) != "" {
			bodyLines = append(bodyLines, "", "详情:", task.Detail)
		}
		if strings.TrimSpace(task.LastPostponeReason) != "" {
			bodyLines = append(bodyLines, "", "最近一次顺延原因:", task.LastPostponeReason)
		}
		if strings.TrimSpace(task.CancelReason) != "" {
			bodyLines = append(bodyLines, "", "取消原因:", task.CancelReason)
		}
		attachment := &mailAttachment{
			Filename:    plannerCalendarFilename(task),
			ContentType: "text/calendar; charset=UTF-8",
			Content:     buildPlannerICS(task),
		}
		if err := h.sendReminderMail(recipients, subject, strings.Join(bodyLines, "\n"), attachment); err != nil {
			log.Printf("planner reminder: send failed for task %s: %v", task.ID, err)
			continue
		}
		sentAt := time.Now()
		nextRemindAt := plannerNextReminderAfter(task, sentAt)
		if nextRemindAt != nil {
			_ = h.db.UpdatePlannerTaskReminderState(task.ID, nextRemindAt, task.RepeatUntil, &sentAt)
			continue
		}
		_ = h.db.MarkPlannerTaskReminderSent(task.ID, sentAt)
	}
}

func (h *PlannerHandler) sendReminderMail(recipientRaw, subject, body string, attachment *mailAttachment) error {
	if h.smtpHost == "" || h.smtpUser == "" || h.smtpPass == "" {
		return fmt.Errorf("planner smtp not configured")
	}
	recipients := splitAlertRecipients(recipientRaw)
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients")
	}
	port := h.smtpPort
	if port == 0 {
		port = 465
	}
	addr := net.JoinHostPort(h.smtpHost, strconv.Itoa(port))
	msg := buildSMTPMessage(h.smtpUser, recipients, subject, body, attachment)

	var (
		conn   net.Conn
		client *smtp.Client
		err    error
	)
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	tlsCfg := &tls.Config{ServerName: h.smtpHost}
	if port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, h.smtpHost)
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, h.smtpHost)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err = client.StartTLS(tlsCfg); err != nil {
					client.Close()
					return err
				}
			}
		}
	}
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", h.smtpUser, h.smtpPass, h.smtpHost)
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(h.smtpUser); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write([]byte(msg)); err != nil {
		writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
