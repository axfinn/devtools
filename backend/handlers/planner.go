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
	DisplayDate       string `json:"display_date"`
	DisplayLabel      string `json:"display_label"`
	IsRolledOver      bool   `json:"is_rolled_over"`
	IsToday           bool   `json:"is_today"`
	CalendarAvailable bool   `json:"calendar_available"`
}

type plannerTimelineGroup struct {
	Date  string                 `json:"date"`
	Label string                 `json:"label"`
	Items []*plannerTimelineItem `json:"items"`
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
	CreatorKey  string `json:"creator_key"`
}

type createPlannerTaskRequest struct {
	Kind        string `json:"kind"`
	Title       string `json:"title" binding:"required"`
	Detail      string `json:"detail"`
	Notes       string `json:"notes"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	PlannedFor  string `json:"planned_for"`
	RemindAt    string `json:"remind_at"`
	NotifyEmail string `json:"notify_email"`
}

type updatePlannerTaskRequest struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Notes       string `json:"notes"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	PlannedFor  string `json:"planned_for"`
	RemindAt    string `json:"remind_at"`
	NotifyEmail string `json:"notify_email"`
}

type plannerCommentRequest struct {
	Author  string `json:"author"`
	Content string `json:"content" binding:"required"`
}

type plannerAIParseRequest struct {
	Text        string `json:"text" binding:"required"`
	DefaultKind string `json:"default_kind"`
}

type plannerAITask struct {
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Notes      string `json:"notes"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	PlannedFor string `json:"planned_for"`
	RemindAt   string `json:"remind_at"`
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

func plannerToday() string {
	return time.Now().Format("2006-01-02")
}

func parsePlannerDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return plannerToday()
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t.Format("2006-01-02")
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
	key := plannerCreatorKey(c)
	if key != "" && utils.VerifyPassword(key, profile.CreatorKey) {
		return profile, true
	}
	password := strings.TrimSpace(getPassword(c))
	if password != "" && plannerPasswordIndex(password) == profile.PasswordIndex {
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
	key := plannerCreatorKey(c)
	password := strings.TrimSpace(getPassword(c))
	if key != "" && utils.VerifyPassword(key, profile.CreatorKey) {
		return profile, true
	}
	if password != "" && plannerPasswordIndex(password) == profile.PasswordIndex {
		return profile, true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "无权限", "code": 403})
	return nil, false
}

func (h *PlannerHandler) loadTask(c *gin.Context) (*models.PlannerProfile, *models.PlannerTask, bool) {
	profileID := c.Param("id")
	profile, ok := h.loadProfileByAccess(c, profileID)
	if !ok {
		return nil, nil, false
	}
	task, err := h.db.GetPlannerTask(c.Param("taskId"))
	if err != nil || task.ProfileID != profileID {
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
	if req.NotifyEmail != "" || strings.TrimSpace(req.NotifyEmail) == "" {
		profile.NotifyEmail = strings.TrimSpace(req.NotifyEmail)
	}
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

func (h *PlannerHandler) ListTimeline(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	kind := normalizePlannerKind(c.DefaultQuery("kind", plannerModeDefault(time.Now())))
	tasks, err := h.db.ListPlannerTasks(profile.ID, kind, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取时间线失败", "code": 500})
		return
	}
	groups, counts := buildPlannerTimeline(tasks, time.Now())
	c.JSON(http.StatusOK, gin.H{
		"code":         0,
		"kind":         kind,
		"profile_name": profile.Name,
		"groups":       groups,
		"counts":       counts,
		"mode_default": plannerModeDefault(time.Now()),
	})
}

func buildPlannerTimeline(tasks []*models.PlannerTask, now time.Time) ([]*plannerTimelineGroup, map[string]int) {
	today := now.Format("2006-01-02")
	groupMap := make(map[string]*plannerTimelineGroup)
	order := make([]string, 0)
	counts := map[string]int{
		plannerStatusOpen:       0,
		plannerStatusInProgress: 0,
		plannerStatusDone:       0,
		plannerStatusCancelled:  0,
		"rolled_over":           0,
	}

	for _, task := range tasks {
		status := normalizePlannerStatus(task.Status)
		counts[status]++
		displayDate := task.PlannedFor
		rolledOver := false
		if displayDate == "" {
			displayDate = today
		}
		if status == plannerStatusOpen || status == plannerStatusInProgress {
			if displayDate < today {
				displayDate = today
				rolledOver = true
				counts["rolled_over"]++
			}
		} else if task.CompletedAt != nil {
			displayDate = task.CompletedAt.Format("2006-01-02")
		}
		group, ok := groupMap[displayDate]
		if !ok {
			group = &plannerTimelineGroup{
				Date:  displayDate,
				Label: plannerDateLabel(displayDate, today),
				Items: make([]*plannerTimelineItem, 0),
			}
			groupMap[displayDate] = group
			order = append(order, displayDate)
		}
		group.Items = append(group.Items, &plannerTimelineItem{
			PlannerTask:       task,
			DisplayDate:       displayDate,
			DisplayLabel:      group.Label,
			IsRolledOver:      rolledOver,
			IsToday:           displayDate == today,
			CalendarAvailable: true,
		})
	}

	sort.Strings(order)
	result := make([]*plannerTimelineGroup, 0, len(order))
	for _, key := range order {
		group := groupMap[key]
		sort.SliceStable(group.Items, func(i, j int) bool {
			a := group.Items[i]
			b := group.Items[j]
			if a.Status != b.Status {
				return plannerStatusRank(a.Status) < plannerStatusRank(b.Status)
			}
			if a.Priority != b.Priority {
				return plannerPriorityRank(a.Priority) < plannerPriorityRank(b.Priority)
			}
			return a.CreatedAt.Before(b.CreatedAt)
		})
		result = append(result, group)
	}
	return result, counts
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
	task := &models.PlannerTask{
		ProfileID:   profile.ID,
		Kind:        normalizePlannerKind(req.Kind),
		Title:       strings.TrimSpace(req.Title),
		Detail:      strings.TrimSpace(req.Detail),
		Notes:       strings.TrimSpace(req.Notes),
		Status:      normalizePlannerStatus(req.Status),
		Priority:    normalizePlannerPriority(req.Priority),
		PlannedFor:  parsePlannerDate(req.PlannedFor),
		RemindAt:    parsePlannerDateTime(req.RemindAt),
		NotifyEmail: strings.TrimSpace(req.NotifyEmail),
	}
	if task.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空", "code": 400})
		return
	}
	if task.NotifyEmail == "" {
		task.NotifyEmail = profile.NotifyEmail
	}
	if task.Status == plannerStatusDone {
		now := time.Now()
		task.CompletedAt = &now
	}
	if err := h.db.CreatePlannerTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "task": task})
}

func (h *PlannerHandler) UpdateTask(c *gin.Context) {
	_, task, ok := h.loadTask(c)
	if !ok {
		return
	}
	var req updatePlannerTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}
	if strings.TrimSpace(req.Kind) != "" {
		task.Kind = normalizePlannerKind(req.Kind)
	}
	if strings.TrimSpace(req.Title) != "" {
		task.Title = strings.TrimSpace(req.Title)
	}
	if req.Detail != "" || strings.TrimSpace(req.Detail) == "" {
		task.Detail = strings.TrimSpace(req.Detail)
	}
	if req.Notes != "" || strings.TrimSpace(req.Notes) == "" {
		task.Notes = strings.TrimSpace(req.Notes)
	}
	if strings.TrimSpace(req.Status) != "" {
		task.Status = normalizePlannerStatus(req.Status)
		if task.Status == plannerStatusDone {
			now := time.Now()
			task.CompletedAt = &now
		} else {
			task.CompletedAt = nil
		}
	}
	if strings.TrimSpace(req.Priority) != "" {
		task.Priority = normalizePlannerPriority(req.Priority)
	}
	if strings.TrimSpace(req.PlannedFor) != "" {
		task.PlannedFor = parsePlannerDate(req.PlannedFor)
	}
	if req.RemindAt != "" || strings.TrimSpace(req.RemindAt) == "" {
		task.RemindAt = parsePlannerDateTime(req.RemindAt)
	}
	if req.NotifyEmail != "" || strings.TrimSpace(req.NotifyEmail) == "" {
		task.NotifyEmail = strings.TrimSpace(req.NotifyEmail)
	}
	if err := h.db.UpdatePlannerTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
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
		Author:    strings.TrimSpace(req.Author),
		Content:   strings.TrimSpace(req.Content),
	}
	if comment.Author == "" {
		comment.Author = "我"
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
	if description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICSLine(description)))
	}
	if task.RemindAt != nil {
		start := task.RemindAt.In(time.Local)
		end := start.Add(30 * time.Minute)
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", start.Format("20060102T150405")))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", end.Format("20060102T150405")))
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
		title = "task"
	}
	return fmt.Sprintf("%s-%s.ics", task.Kind, title)
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
	_, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
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

func (h *PlannerHandler) parsePlannerText(text, defaultKind string) ([]plannerAITask, string, error) {
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		return fallbackParsePlannerText(text, defaultKind), "fallback", nil
	}
	prompt := fmt.Sprintf(`请把下面的中文事项整理成 JSON 数组，每个元素包含字段：
kind(work/life)、title、detail、notes、priority(low/medium/high)、status(open/in_progress/done/cancelled)、planned_for(YYYY-MM-DD)、remind_at(YYYY-MM-DDTHH:MM，可为空)。
要求：
1. 仅返回 JSON 数组，不要解释。
2. 默认 kind 为 %s。
3. 如果没有明确日期，planned_for 用今天。
4. 标题要短，detail 保留上下文。
5. 不要杜撰不存在的信息。

原始内容：
%s`, defaultKind, text)

	reqBody := map[string]interface{}{
		"model": h.cfg.MiniMax.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个只负责管理工作事项和生活事项的智能助手。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	if h.cfg.MiniMax.Model == "" {
		reqBody["model"] = "abab6.5s-chat"
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://api.minimax.chat/v1/text/chatcompletion_v2", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.MiniMax.APIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("MiniMax 请求失败: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("MiniMax 返回异常(%d): %s", resp.StatusCode, string(respBody))
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, "", fmt.Errorf("解析 AI 响应失败: %v", err)
	}
	choices, _ := raw["choices"].([]interface{})
	if len(choices) == 0 {
		return nil, "", fmt.Errorf("AI 未返回结果")
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
		tasks[i].Kind = normalizePlannerKind(plannerFirstNonEmpty(tasks[i].Kind, defaultKind))
		tasks[i].Priority = normalizePlannerPriority(tasks[i].Priority)
		tasks[i].Status = normalizePlannerStatus(tasks[i].Status)
		tasks[i].PlannedFor = parsePlannerDate(tasks[i].PlannedFor)
		if strings.TrimSpace(tasks[i].Title) == "" {
			tasks[i].Title = "待整理事项"
		}
	}
	return tasks, "minimax", nil
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
		kind := defaultKind
		lower := strings.ToLower(item)
		if strings.Contains(lower, "生活") || strings.Contains(item, "买菜") || strings.Contains(item, "买水果") || strings.Contains(item, "家里") || strings.Contains(item, "散步") {
			kind = plannerKindLife
		}
		if strings.Contains(lower, "工作") || strings.Contains(item, "开会") || strings.Contains(item, "需求") || strings.Contains(item, "上线") {
			kind = plannerKindWork
		}
		priority := "medium"
		if strings.Contains(item, "尽快") || strings.Contains(item, "今天") || strings.Contains(item, "马上") {
			priority = "high"
		}
		result = append(result, plannerAITask{
			Kind:       normalizePlannerKind(kind),
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
			Title:      strings.TrimSpace(text),
			Detail:     strings.TrimSpace(text),
			Priority:   "medium",
			Status:     plannerStatusOpen,
			PlannedFor: plannerToday(),
		})
	}
	return result
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
	if req.NotifyEmail != "" || strings.TrimSpace(req.NotifyEmail) == "" {
		profile.NotifyEmail = strings.TrimSpace(req.NotifyEmail)
	}
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
		recipients := strings.TrimSpace(task.NotifyEmail)
		if recipients == "" {
			recipients = strings.TrimSpace(profile.NotifyEmail)
		}
		if recipients == "" {
			continue
		}
		subject := fmt.Sprintf("事项提醒 [%s] %s", map[string]string{plannerKindWork: "工作", plannerKindLife: "生活"}[task.Kind], task.Title)
		bodyLines := []string{
			fmt.Sprintf("档案: %s", profile.Name),
			fmt.Sprintf("类型: %s", map[string]string{plannerKindWork: "工作事项", plannerKindLife: "生活事项"}[task.Kind]),
			fmt.Sprintf("标题: %s", task.Title),
			fmt.Sprintf("状态: %s", task.Status),
			fmt.Sprintf("优先级: %s", task.Priority),
			fmt.Sprintf("计划日期: %s", task.PlannedFor),
		}
		if task.RemindAt != nil {
			bodyLines = append(bodyLines, fmt.Sprintf("提醒时间: %s", task.RemindAt.In(time.Local).Format("2006-01-02 15:04")))
		}
		if strings.TrimSpace(task.Detail) != "" {
			bodyLines = append(bodyLines, "", "详情:", task.Detail)
		}
		if strings.TrimSpace(task.Notes) != "" {
			bodyLines = append(bodyLines, "", "备注:", task.Notes)
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
		_ = h.db.MarkPlannerTaskReminderSent(task.ID, time.Now())
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
