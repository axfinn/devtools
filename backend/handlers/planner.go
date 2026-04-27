package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"regexp"
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
var plannerASRServiceURL = "http://asr-service:9000"

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
	asrServiceURL     string
	serviceHTTPClient *http.Client
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
	RawText        string `json:"raw_text"`
	PostponeReason string `json:"postpone_reason"`
	Intent         string `json:"intent"`
	EnergyLevel    string `json:"energy_level"`
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
	RawText        *string `json:"raw_text"`
	PostponeReason *string `json:"postpone_reason"`
	Intent         *string `json:"intent"`
	EnergyLevel    *string `json:"energy_level"`
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
	RawText      string `json:"raw_text"`
	Intent       string `json:"intent"`
	EnergyLevel  string `json:"energy_level"`
}

type plannerAICoachResponse struct {
	Summary     string   `json:"summary"`
	Insights    []string `json:"insights"`
	Suggestions []string `json:"suggestions"`
}

type plannerParsedSchedule struct {
	PlannedFor string
	RemindAt   string
	HasDate    bool
	HasTime    bool
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
	asrServiceURL := strings.TrimSpace(os.Getenv("ASR_SERVICE_URL"))
	if asrServiceURL == "" {
		asrServiceURL = plannerASRServiceURL
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
		asrServiceURL:     asrServiceURL,
		serviceHTTPClient: &http.Client{Timeout: 5 * time.Minute, Transport: &http.Transport{Proxy: nil}},
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

func normalizePlannerEnergyLevel(level string) string {
	switch strings.TrimSpace(strings.ToLower(level)) {
	case "deep":
		return "deep"
	case "shallow":
		return "shallow"
	case "errand":
		return "errand"
	case "creative":
		return "creative"
	default:
		return ""
	}
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

func plannerFormatDateTimeValue(t time.Time) string {
	return t.In(time.Local).Format("2006-01-02T15:04")
}

func plannerNextWeekday(base time.Time, weekday time.Weekday, forceNextWeek bool) time.Time {
	base = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.Local)
	diff := int(weekday - base.Weekday())
	if diff < 0 {
		diff += 7
	}
	if forceNextWeek || diff == 0 {
		diff += 7
	}
	return base.AddDate(0, 0, diff)
}

func plannerResolveDateFromText(text string, base time.Time) (time.Time, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	date := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.Local)
	switch {
	case strings.Contains(lower, "大后天"):
		return date.AddDate(0, 0, 3), true
	case strings.Contains(lower, "后天"):
		return date.AddDate(0, 0, 2), true
	case strings.Contains(lower, "明天"), strings.Contains(lower, "明早"), strings.Contains(lower, "明晚"), strings.Contains(lower, "明晨"):
		return date.AddDate(0, 0, 1), true
	case strings.Contains(lower, "今天"), strings.Contains(lower, "今晚"), strings.Contains(lower, "今早"), strings.Contains(lower, "今日"), strings.Contains(lower, "今天晚上"):
		return date, true
	}

	weekdayMap := map[string]time.Weekday{
		"日": time.Sunday,
		"天": time.Sunday,
		"一": time.Monday,
		"二": time.Tuesday,
		"三": time.Wednesday,
		"四": time.Thursday,
		"五": time.Friday,
		"六": time.Saturday,
	}
	if matches := regexp.MustCompile(`(下周|这周|本周|周|星期)([一二三四五六日天])`).FindStringSubmatch(text); len(matches) == 3 {
		prefix := matches[1]
		target := weekdayMap[matches[2]]
		return plannerNextWeekday(date, target, prefix == "下周"), true
	}
	if matches := regexp.MustCompile(`(\d{1,2})月(\d{1,2})[日号]?`).FindStringSubmatch(text); len(matches) == 3 {
		month, _ := strconv.Atoi(matches[1])
		day, _ := strconv.Atoi(matches[2])
		year := date.Year()
		candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		if candidate.Month() == time.Month(month) && candidate.Day() == day {
			if candidate.Before(date) {
				candidate = time.Date(year+1, time.Month(month), day, 0, 0, 0, 0, time.Local)
			}
			return candidate, true
		}
	}
	if matches := regexp.MustCompile(`(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})`).FindStringSubmatch(text); len(matches) == 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		if candidate.Year() == year && candidate.Month() == time.Month(month) && candidate.Day() == day {
			return candidate, true
		}
	}
	return date, false
}

func plannerResolveTimeFromText(text string) (int, int, bool) {
	if matches := regexp.MustCompile(`(\d{1,2})[:：](\d{2})`).FindStringSubmatch(text); len(matches) == 3 {
		hour, _ := strconv.Atoi(matches[1])
		minute, _ := strconv.Atoi(matches[2])
		if hour >= 0 && hour < 24 && minute >= 0 && minute < 60 {
			return hour, minute, true
		}
	}
	if matches := regexp.MustCompile(`(凌晨|早上|上午|中午|下午|傍晚|晚上|今晚|半夜)?\s*(\d{1,2})点(?:(\d{1,2})分?|半)?`).FindStringSubmatch(text); len(matches) >= 3 {
		period := matches[1]
		hour, _ := strconv.Atoi(matches[2])
		minute := 0
		switch matches[3] {
		case "半":
			minute = 30
		case "":
			minute = 0
		default:
			minute, _ = strconv.Atoi(matches[3])
		}
		if minute < 0 || minute >= 60 || hour < 0 || hour > 23 {
			return 0, 0, false
		}
		switch period {
		case "下午", "傍晚", "晚上", "今晚":
			if hour < 12 {
				hour += 12
			}
		case "中午":
			if hour < 11 {
				hour += 12
			}
		case "凌晨", "半夜":
			if hour == 12 {
				hour = 0
			}
		}
		if hour >= 0 && hour < 24 {
			return hour, minute, true
		}
	}
	return 0, 0, false
}

func plannerExtractScheduleFromText(text string, base time.Time) plannerParsedSchedule {
	date, hasDate := plannerResolveDateFromText(text, base)
	hour, minute, hasTime := plannerResolveTimeFromText(text)
	schedule := plannerParsedSchedule{}
	if hasDate {
		schedule.HasDate = true
		schedule.PlannedFor = date.Format("2006-01-02")
	}
	if hasTime {
		schedule.HasTime = true
		if !hasDate {
			date = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.Local)
			schedule.PlannedFor = date.Format("2006-01-02")
		}
		schedule.RemindAt = plannerFormatDateTimeValue(time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local))
	}
	return schedule
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
