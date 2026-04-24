package handlers

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

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
