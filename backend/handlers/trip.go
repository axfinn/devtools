package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devtools/models"
	"devtools/notif"
	"devtools/trip"

	"github.com/gin-gonic/gin"
)

// TripHandler 暴露 /api/trips/* HTTP 接口。
// 全部走 trip.Service,handler 只做参数解析 + 状态码翻译。
// 提醒处理用 trip.ReminderProcessor,cleanup tick 通过 ProcessWindow 复用。
type TripHandler struct {
	service   *trip.Service
	processor *trip.ReminderProcessor
}

// NewTripHandler 构造 handler。db.Conn() 给 trip 包复用主 SQLite 连接,
// notifCfg 留空也能跑,只是提醒不发邮件(SMTP 未配齐时静默跳过)。
func NewTripHandler(db *models.DB, notifCfg notif.Config) *TripHandler {
	repo := trip.NewSQLiteRepository(db.Conn())
	svc := trip.NewService(repo)
	if err := svc.EnsureSchema(context.Background()); err != nil {
		// 不 panic;启动期不影响主路径
		// 真正使用时 list / create 也会再返错
		// (主路径完全无依赖 trip 表,启动错也不阻塞)
		_ = err
	}
	proc := trip.NewReminderProcessor(repo, notifCfg)
	return &TripHandler{service: svc, processor: proc}
}

// ProcessDueReminders 给 backend/cleanup.go 每小时调一次。
// 复用 planner 的 hook 命名习惯。
func (h *TripHandler) ProcessDueReminders() {
	defer func() {
		if r := recover(); r != nil {
			// 单条 panic 不应阻断 tick,与 planner 处理同思路。
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	now := time.Now()
	from := now.Add(-30 * time.Minute)
	to := now.Add(30 * time.Minute)
	h.processor.ProcessWindow(ctx, from, to)
}

// QuickLog 启动期一次性提示 SMTP 状态,避免用户配错一脸懵。
func (h *TripHandler) QuickLog(scope string) {
	h.processor.QuickLog(scope)
}

// ---- 路由 handler ----

func (h *TripHandler) List(c *gin.Context) {
	trips, err := h.service.ListTrips(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list trips failed", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trips": trips, "count": len(trips)})
}

func (h *TripHandler) Get(c *gin.Context) {
	id := c.Param("id")
	t, err := h.service.GetTrip(c.Request.Context(), id)
	if err == trip.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "trip not found", "code": 404})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *TripHandler) Markdown(c *gin.Context) {
	id := c.Param("id")
	md, err := h.service.RenderMarkdown(c.Request.Context(), id)
	if err == trip.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "trip not found", "code": 404})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.Header("Content-Type", "text/markdown; charset=UTF-8")
	c.String(http.StatusOK, md)
}

type tripCreateRequest struct {
	Name        string   `form:"name" json:"name"`
	Description string   `form:"description" json:"description"`
	StartDate   string   `form:"start_date" json:"start_date"`
	EndDate     string   `form:"end_date" json:"end_date"`
	Tags        []string `form:"tags" json:"tags"`
	CoverCities []string `form:"cover_cities" json:"cover_cities"`
	NotifyEmail string   `form:"notify_email" json:"notify_email"`
}

func (h *TripHandler) Create(c *gin.Context) {
	var req tripCreateRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	t := &trip.Trip{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		StartDate:   strings.TrimSpace(req.StartDate),
		EndDate:     strings.TrimSpace(req.EndDate),
		Tags:        req.Tags,
		CoverCities: req.CoverCities,
		NotifyEmail: strings.TrimSpace(req.NotifyEmail),
	}
	if err := h.service.CreateTrip(c.Request.Context(), t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trip": t})
}

func (h *TripHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteTrip(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}

type tripAddDayRequest struct {
	Date       string `form:"date" json:"date"`
	City       string `form:"city" json:"city"`
	Region     string `form:"region" json:"region"`
	Country    string `form:"country" json:"country"`
}

func (h *TripHandler) AddDay(c *gin.Context) {
	tripID := c.Param("id")
	var req tripAddDayRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	dp := &trip.DayPlan{
		TripID: tripID,
		Date:   strings.TrimSpace(req.Date),
		Destination: trip.Destination{
			City:    strings.TrimSpace(req.City),
			Region:  strings.TrimSpace(req.Region),
			Country: strings.TrimSpace(req.Country),
		},
	}
	if err := h.service.AddDay(c.Request.Context(), dp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": dp})
}

type tripAddActivityRequest struct {
	DayID               string   `form:"day_id" json:"day_id"`
	Kind                string   `form:"kind" json:"kind"`
	Title               string   `form:"title" json:"title"`
	Location            string   `form:"location" json:"location"`
	City                string   `form:"city" json:"city"`
	Region              string   `form:"region" json:"region"`
	Country             string   `form:"country" json:"country"`
	StartTime           string   `form:"start_time" json:"start_time"`
	DurationMin         int      `form:"duration_min" json:"duration_min"`
	Note                string   `form:"note" json:"note"`
	RemindBeforeMinutes int      `form:"remind_before_minutes" json:"remind_before_minutes"`
	RemindEmails        []string `form:"remind_emails" json:"remind_emails"`
}

func (h *TripHandler) AddActivity(c *gin.Context) {
	// URL 既支持 /trips/days/:dayId/activities, 也兼容 /trips/:id/days/:dayId/activities
	dayID := c.Param("dayId")
	if dayID == "" {
		dayID = c.Param("day_id")
	}
	if dayID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing day_id", "code": 400})
		return
	}
	var req tripAddActivityRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	// 兼容 path 优先; body 内 day_id 是 fallback
	if req.DayID != "" && dayID == "" {
		dayID = req.DayID
	}
	a := &trip.Activity{
		DayID:               dayID,
		Kind:                trip.ActivityKind(strings.TrimSpace(req.Kind)),
		Title:               strings.TrimSpace(req.Title),
		Location:            strings.TrimSpace(req.Location),
		StartTime:           strings.TrimSpace(req.StartTime),
		DurationMin:         req.DurationMin,
		Note:                strings.TrimSpace(req.Note),
		RemindBeforeMinutes: req.RemindBeforeMinutes,
		RemindEmails:        req.RemindEmails,
		Destination: trip.Destination{
			City:    strings.TrimSpace(req.City),
			Region:  strings.TrimSpace(req.Region),
			Country: strings.TrimSpace(req.Country),
		},
	}
	if err := h.service.AddActivity(c.Request.Context(), a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activity": a})
}

type tripReminderRequest struct {
	RemindBeforeMinutes int      `form:"remind_before_minutes" json:"remind_before_minutes"`
	RemindEmails        []string `form:"remind_emails" json:"remind_emails"`
	Reset               bool     `form:"reset" json:"reset"` // 重发提醒:清掉 remind_sent_at
}

// SetReminder 单独更新活动的提醒配置,前端 UI 改时间不必重发整条活动。
// reset=true 时把 remind_sent_at 清零,允许重发已发过的提醒(用于"测试提醒"按钮)。
func (h *TripHandler) SetReminder(c *gin.Context) {
	activityID := c.Param("activityId")
	if activityID == "" {
		activityID = c.Param("activity_id")
	}
	if activityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing activity_id", "code": 400})
		return
	}
	var req tripReminderRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	if err := h.service.SetReminder(c.Request.Context(), activityID, req.RemindBeforeMinutes, req.RemindEmails); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	if req.Reset {
		// 清零 remind_sent_at:直接 db 写;Service 没暴露此操作,因为只有 UI 显式触发
		if _, err := h.processor.ResetSentAt(c.Request.Context(), activityID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "reminder updated"})
}

func (h *TripHandler) Seed(c *gin.Context) {
	force, _ := strconv.ParseBool(c.DefaultQuery("force", "false"))
	id, created, err := h.service.SeedRealTrip(c.Request.Context(), force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trip_id": id, "created": created})
}

// ProcessNow 立即扫描一次,前端"测试提醒"按钮用。返回发了多少条。
func (h *TripHandler) ProcessNow(c *gin.Context) {
	sent, skipped, err := h.processor.ProcessNow(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "sent": sent, "skipped": skipped})
}

// ---- 预算 / 费用 / 总结 / 计划调整 ----

type tripBudgetRequest struct {
	// BudgetTotal 用「元」传入(前端友好);服务端转 cents 入库。
	// BudgetTotalCents 可选,直接接「分」精度更高,优先使用。
	BudgetTotal      int64   `form:"budget_total" json:"budget_total"`
	BudgetTotalCents int64   `form:"budget_total_cents" json:"budget_total_cents"`
	BudgetCurrency   string  `form:"budget_currency" json:"budget_currency"`
	// 兼容字段:金额也能按"元"传,带 2 位小数
	BudgetYuan       float64 `form:"budget_yuan" json:"budget_yuan"`
}

// UpdateBudget 处理 PUT /api/trips/:id/budget。
// 接受「元」或「分」两种精度输入;BudgetTotalCents > 0 时按 cents 入库,否则按 yuan * 100。
func (h *TripHandler) UpdateBudget(c *gin.Context) {
	tripID := c.Param("id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing trip_id", "code": 400})
		return
	}
	var req tripBudgetRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	cents := req.BudgetTotalCents
	if cents == 0 && req.BudgetYuan != 0 {
		cents = int64(req.BudgetYuan * 100)
	}
	if cents == 0 && req.BudgetTotal != 0 {
		cents = req.BudgetTotal * 100 // 兼容旧字段:把整数当元
	}
	currency := strings.ToUpper(strings.TrimSpace(req.BudgetCurrency))
	if err := h.service.UpdateBudget(c.Request.Context(), tripID, cents, currency); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":             0,
		"trip_id":          tripID,
		"budget_total":     cents,
		"budget_currency":  firstNonEmptyCurrency(currency, "CNY"),
	})
}

func firstNonEmptyCurrency(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "CNY"
}

type tripExpenseRequest struct {
	Date          string  `form:"date" json:"date"`
	Category      string  `form:"category" json:"category"`
	Amount        float64 `form:"amount" json:"amount"`             // 元
	AmountCents   int64   `form:"amount_cents" json:"amount_cents"` // 优先
	Currency      string  `form:"currency" json:"currency"`
	Note          string  `form:"note" json:"note"`
	PaymentMethod string  `form:"payment_method" json:"payment_method"`
	ActivityID    string  `form:"activity_id" json:"activity_id"`
}

func (h *TripHandler) AddExpense(c *gin.Context) {
	tripID := c.Param("id")
	if tripID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing trip_id", "code": 400})
		return
	}
	var req tripExpenseRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	cents := req.AmountCents
	if cents == 0 && req.Amount != 0 {
		cents = int64(req.Amount * 100)
	}
	if cents == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount or amount_cents is required", "code": 400})
		return
	}
	e := &trip.Expense{
		TripID:        tripID,
		Date:          strings.TrimSpace(req.Date),
		Category:      trip.ExpenseKind(strings.TrimSpace(req.Category)),
		AmountCents:   cents,
		Currency:      strings.ToUpper(strings.TrimSpace(req.Currency)),
		Note:          strings.TrimSpace(req.Note),
		PaymentMethod: strings.TrimSpace(req.PaymentMethod),
		ActivityID:    strings.TrimSpace(req.ActivityID),
	}
	if err := h.service.AddExpense(c.Request.Context(), e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expense": e})
}

func (h *TripHandler) ListExpenses(c *gin.Context) {
	tripID := c.Param("id")
	expenses, err := h.service.ListExpenses(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses, "count": len(expenses)})
}

// UpdateExpense 与 AddExpense 字段一致,但要求 path 里有 expenseId。
func (h *TripHandler) UpdateExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	if expenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing expense_id", "code": 400})
		return
	}
	var req tripExpenseRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	cents := req.AmountCents
	if cents == 0 && req.Amount != 0 {
		cents = int64(req.Amount * 100)
	}
	if cents == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount or amount_cents is required", "code": 400})
		return
	}
	e := &trip.Expense{
		ID:            expenseID,
		Date:          strings.TrimSpace(req.Date),
		Category:      trip.ExpenseKind(strings.TrimSpace(req.Category)),
		AmountCents:   cents,
		Currency:      strings.ToUpper(strings.TrimSpace(req.Currency)),
		Note:          strings.TrimSpace(req.Note),
		PaymentMethod: strings.TrimSpace(req.PaymentMethod),
		ActivityID:    strings.TrimSpace(req.ActivityID),
	}
	if err := h.service.UpdateExpense(c.Request.Context(), e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expense": e})
}

func (h *TripHandler) DeleteExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	if expenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing expense_id", "code": 400})
		return
	}
	if err := h.service.DeleteExpense(c.Request.Context(), expenseID); err == trip.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "expense not found", "code": 404})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted"})
}

// Summary 返回 JSON,SummaryMarkdown 返回可贴 Notion 的 Markdown。
func (h *TripHandler) Summary(c *gin.Context) {
	tripID := c.Param("id")
	sum, err := h.service.TripSummary(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.JSON(http.StatusOK, sum)
}

func (h *TripHandler) SummaryMarkdown(c *gin.Context) {
	tripID := c.Param("id")
	md, err := h.service.RenderSummaryMarkdown(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}
	c.Header("Content-Type", "text/markdown; charset=UTF-8")
	c.String(http.StatusOK, md)
}

// ---- 计划调整(出行中 inline edit) ----

type tripUpdateActivityRequest struct {
	Title       string `form:"title" json:"title"`
	Kind        string `form:"kind" json:"kind"`
	Location    string `form:"location" json:"location"`
	StartTime   string `form:"start_time" json:"start_time"`
	DurationMin int    `form:"duration_min" json:"duration_min"`
	Note        string `form:"note" json:"note"`
	City        string `form:"city" json:"city"`
	Region      string `form:"region" json:"region"`
	Country     string `form:"country" json:"country"`
}

// UpdateActivity 用于出行中临时调整某项活动(改时间 / 标题 / 时长 / 备注)。
// 不改 remind_* 字段(由 SetReminder 单独维护),不改 order(避免误改排序)。
func (h *TripHandler) UpdateActivity(c *gin.Context) {
	activityID := c.Param("activityId")
	if activityID == "" {
		activityID = c.Param("id")
	}
	if activityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing activity_id", "code": 400})
		return
	}
	var req tripUpdateActivityRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	a := &trip.Activity{
		ID:          activityID,
		Kind:        trip.ActivityKind(strings.TrimSpace(req.Kind)),
		Title:       strings.TrimSpace(req.Title),
		Location:    strings.TrimSpace(req.Location),
		StartTime:   strings.TrimSpace(req.StartTime),
		DurationMin: req.DurationMin,
		Note:        strings.TrimSpace(req.Note),
		Destination: trip.Destination{
			City:    strings.TrimSpace(req.City),
			Region:  strings.TrimSpace(req.Region),
			Country: strings.TrimSpace(req.Country),
		},
	}
	if err := h.service.UpdateActivity(c.Request.Context(), a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activity": a})
}

type tripUpdateDayRequest struct {
	Date    string `form:"date" json:"date"`
	City    string `form:"city" json:"city"`
	Region  string `form:"region" json:"region"`
	Country string `form:"country" json:"country"`
}

// UpdateDay 出行中临时调整某天。
func (h *TripHandler) UpdateDay(c *gin.Context) {
	dayID := c.Param("dayId")
	if dayID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing day_id", "code": 400})
		return
	}
	var req tripUpdateDayRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "code": 400})
		return
	}
	d := &trip.DayPlan{
		ID:  dayID,
		Date: strings.TrimSpace(req.Date),
		Destination: trip.Destination{
			City:    strings.TrimSpace(req.City),
			Region:  strings.TrimSpace(req.Region),
			Country: strings.TrimSpace(req.Country),
		},
	}
	if err := h.service.UpdateDay(c.Request.Context(), d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": 400})
		return
	}
	c.JSON(http.StatusOK, gin.H{"day": d})
}