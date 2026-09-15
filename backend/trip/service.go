package trip

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Service 用例层:封装跨仓储的复合操作(Markdown 渲染、级联删除提醒重建等)。
// HTTP handler / CLI 都从这里调。
type Service struct {
	repo Repository
}

// NewService 构造一个 Service;调用方负责传一个已 EnsureSchema 的 Repository。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// EnsureSchema 把建表这一动作代理到 Service,启动期一次调用即可。
func (s *Service) EnsureSchema(ctx context.Context) error {
	return s.repo.EnsureSchema(ctx)
}

// ---- Trip ----

func (s *Service) CreateTrip(ctx context.Context, t *Trip) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.CoverCities = dedupNonEmpty(t.CoverCities)
	t.Tags = dedupNonEmpty(t.Tags)
	if t.BudgetCurrency == "" {
		t.BudgetCurrency = "CNY"
	}
	return s.repo.CreateTrip(ctx, t)
}

func (s *Service) GetTrip(ctx context.Context, id string) (*Trip, error) {
	if id == "" {
		return nil, errors.New("trip: id is required")
	}
	return s.repo.GetTripWithDays(ctx, id)
}

func (s *Service) ListTrips(ctx context.Context) ([]*Trip, error) {
	return s.repo.ListTrips(ctx)
}

func (s *Service) UpdateTrip(ctx context.Context, t *Trip) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateTrip(ctx, t)
}

// UpdateBudget 单独接口:前端"预算"Tab 只动这两个字段,
// 不会把 name / dates 重置(避免误操作)。
func (s *Service) UpdateBudget(ctx context.Context, tripID string, total int64, currency string) error {
	if tripID == "" {
		return errors.New("trip: trip_id is required")
	}
	if total < 0 {
		return errors.New("budget_total must be >= 0")
	}
	if currency == "" {
		currency = "CNY"
	}
	if len(currency) != 3 {
		return fmt.Errorf("currency must be ISO 4217 (3 letters), got %q", currency)
	}
	return s.repo.UpdateTripBudget(ctx, tripID, total, currency)
}

func (s *Service) DeleteTrip(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("trip: id is required")
	}
	return s.repo.DeleteTrip(ctx, id)
}

// ---- DayPlan ----

func (s *Service) AddDay(ctx context.Context, d *DayPlan) error {
	if err := d.Validate(); err != nil {
		return err
	}
	// Order 默认 = 当前 trip 已有的天数 + 1,便于连续追加。
	if d.Order == 0 {
		existing, err := s.repo.ListDaysByTrip(ctx, d.TripID)
		if err != nil {
			return err
		}
		d.Order = len(existing) + 1
	}
	return s.repo.AddDay(ctx, d)
}

// UpdateDay 服务层包装:校验后调仓储。
func (s *Service) UpdateDay(ctx context.Context, d *DayPlan) error {
	if err := d.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateDay(ctx, d)
}

// ---- Activity ----

func (s *Service) AddActivity(ctx context.Context, a *Activity) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Order == 0 {
		existing, err := s.repo.ListActivitiesByDay(ctx, a.DayID)
		if err != nil {
			return err
		}
		a.Order = len(existing) + 1
	}
	a.RemindEmails = dedupNonEmpty(a.RemindEmails)
	return s.repo.AddActivity(ctx, a)
}

// UpdateActivity 用于"出行中临时调整"按钮:改 title / location / start_time /
// duration_min / note(以及 kind)。先 GetActivity 拿到当前完整记录再调仓储,
// 这样 remind_* 字段不会被覆盖。
func (s *Service) UpdateActivity(ctx context.Context, a *Activity) error {
	if a.ID == "" {
		return errors.New("trip: activity_id is required")
	}
	if err := a.Validate(); err != nil {
		return err
	}
	// 校验 ID 真实存在,避免 ErrNotFound 漂到仓储层时丢上下文
	if _, err := s.repo.GetActivity(ctx, a.ID); err != nil {
		return err
	}
	return s.repo.UpdateActivity(ctx, a)
}

// SetReminder 单独接口,前端 UI 改提醒时间不必重发整个 Activity。
func (s *Service) SetReminder(ctx context.Context, activityID string, remindBeforeMinutes int, remindEmails []string) error {
	if activityID == "" {
		return errors.New("trip: activity_id is required")
	}
	if remindBeforeMinutes < 0 {
		return errors.New("trip: remind_before_minutes must be >= 0")
	}
	return s.repo.UpdateActivityReminder(ctx, activityID, remindBeforeMinutes, dedupNonEmpty(remindEmails))
}

// ---- Expense ----

func (s *Service) AddExpense(ctx context.Context, e *Expense) error {
	if err := e.Validate(); err != nil {
		return err
	}
	// 货币缺省沿用 trip 的预算货币,保证汇总时口径一致
	if e.Currency == "" {
		t, err := s.repo.GetTrip(ctx, e.TripID)
		if err == nil && t != nil && t.BudgetCurrency != "" {
			e.Currency = t.BudgetCurrency
		} else {
			e.Currency = "CNY"
		}
	}
	return s.repo.AddExpense(ctx, e)
}

// UpdateExpense 编辑支出:校验后调仓储。
// 注意:必须先 GetExpense 拿到原始记录,再覆盖待改字段(避免漏写)。
func (s *Service) UpdateExpense(ctx context.Context, e *Expense) error {
	if e.ID == "" {
		return errors.New("trip: expense_id is required")
	}
	if err := e.Validate(); err != nil {
		return err
	}
	orig, err := s.repo.GetExpense(ctx, e.ID)
	if err != nil {
		return err
	}
	e.TripID = orig.TripID
	e.CreatedAt = orig.CreatedAt
	return s.repo.UpdateExpense(ctx, e)
}

func (s *Service) DeleteExpense(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("trip: expense_id is required")
	}
	return s.repo.DeleteExpense(ctx, id)
}

func (s *Service) ListExpenses(ctx context.Context, tripID string) ([]*Expense, error) {
	if tripID == "" {
		return nil, errors.New("trip: trip_id is required")
	}
	return s.repo.ListExpensesByTrip(ctx, tripID)
}

// ---- Summary ----

// Summary 行程级汇总,前端"总结 Tab"和 Markdown 都从这里取数。
// 多币种情况下,不同 currency 的金额直接相加不做汇率折算
// (汇总期用户可一眼看出"哪几天用的是别的币种")。
type Summary struct {
	Trip              *Trip                       `json:"trip"`
	TotalSpentCents   int64                       `json:"total_spent_cents"`
	Currency          string                      `json:"currency"` // 默认用 trip 的预算币种
	ByCategory        map[ExpenseKind]int64       `json:"by_category_cents"`
	ByDay             map[string]int64            `json:"by_day_cents"` // date -> cents
	TopCategories     []CategorySpend             `json:"top_categories"`
	ExpenseCount      int                         `json:"expense_count"`
	OtherCurrencies   []string                    `json:"other_currencies,omitempty"`
	RemainingCents    int64                       `json:"remaining_cents,omitempty"` // BudgetTotal - TotalSpentCents;无预算则为 0
	OverBudget        bool                        `json:"over_budget"`
	DaysCovered       int                         `json:"days_covered"`
	AveragePerDayCents int64                      `json:"average_per_day_cents,omitempty"`
}

type CategorySpend struct {
	Category ExpenseKind `json:"category"`
	Cents    int64       `json:"cents"`
	Percent  float64     `json:"percent"`
}

// TripSummary 计算一份 trip 的所有费用汇总。
func (s *Service) TripSummary(ctx context.Context, tripID string) (*Summary, error) {
	if tripID == "" {
		return nil, errors.New("trip: trip_id is required")
	}
	t, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	expenses, err := s.repo.ListExpensesByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}

	currency := t.BudgetCurrency
	if currency == "" {
		currency = "CNY"
	}
	out := &Summary{
		Trip:       t,
		Currency:   currency,
		ByCategory: make(map[ExpenseKind]int64, 6),
		ByDay:      make(map[string]int64),
	}
	otherCurrencies := make(map[string]struct{})

	for _, e := range expenses {
		if e.Currency == currency {
			out.TotalSpentCents += e.AmountCents
		} else if e.Currency != "" {
			otherCurrencies[e.Currency] = struct{}{}
		} else {
			// 老数据兜底:等同 trip currency
			out.TotalSpentCents += e.AmountCents
		}
		out.ByCategory[e.Category] += e.AmountCents
		out.ByDay[e.Date] += e.AmountCents
	}
	out.ExpenseCount = len(expenses)

	for c := range otherCurrencies {
		out.OtherCurrencies = append(out.OtherCurrencies, c)
	}
	sort.Strings(out.OtherCurrencies)

	// 排序:TopCategories 按金额降序
	for cat, c := range out.ByCategory {
		pct := 0.0
		if out.TotalSpentCents > 0 {
			pct = float64(c) / float64(out.TotalSpentCents) * 100
		}
		out.TopCategories = append(out.TopCategories, CategorySpend{
			Category: cat, Cents: c, Percent: pct,
		})
	}
	sort.Slice(out.TopCategories, func(i, j int) bool {
		return out.TopCategories[i].Cents > out.TopCategories[j].Cents
	})

	if t.BudgetTotal > 0 {
		out.RemainingCents = t.BudgetTotal - out.TotalSpentCents
		out.OverBudget = out.TotalSpentCents > t.BudgetTotal
	}
	// 已发生的日期(在 trip 日期范围内)
	start, _ := time.Parse("2006-01-02", t.StartDate)
	end, _ := time.Parse("2006-01-02", t.EndDate)
	daysCount := 0
	if !start.IsZero() && !end.IsZero() {
		daysCount = int(end.Sub(start).Hours()/24) + 1
		if daysCount < 0 {
			daysCount = 0
		}
	}
	out.DaysCovered = daysCount
	if daysCount > 0 && out.TotalSpentCents > 0 {
		out.AveragePerDayCents = out.TotalSpentCents / int64(daysCount)
	}
	return out, nil
}

// RenderSummaryMarkdown 把 TripSummary 渲染成可贴 Notion 的总结 Markdown。
// 给"回来后我可以帮你整个做一个总结"那个按钮用。
func (s *Service) RenderSummaryMarkdown(ctx context.Context, tripID string) (string, error) {
	sum, err := s.TripSummary(ctx, tripID)
	if err != nil {
		return "", err
	}
	t := sum.Trip
	var b strings.Builder

	fmt.Fprintf(&b, "# %s — 行程总结\n\n", t.Name)
	fmt.Fprintf(&b, "> %s → %s", t.StartDate, t.EndDate)
	if len(t.CoverCities) > 0 {
		fmt.Fprintf(&b, " · %s", strings.Join(t.CoverCities, " / "))
	}
	b.WriteString("\n\n")

	// 预算概要
	b.WriteString("## 预算概要\n\n")
	b.WriteString("| 项目 | 数值 |\n|------|------|\n")
	fmt.Fprintf(&b, "| 预算 | %s |\n", formatMoney(t.BudgetTotal, t.BudgetCurrency))
	fmt.Fprintf(&b, "| 实际花费 | **%s** |\n", formatMoney(sum.TotalSpentCents, sum.Currency))
	if t.BudgetTotal > 0 {
		fmt.Fprintf(&b, "| 剩余/超支 | %s |\n", formatMoney(sum.RemainingCents, t.BudgetCurrency))
		if sum.OverBudget {
			fmt.Fprintf(&b, "> ⚠️ 实际花费超过预算 %.1f%%\n", float64(sum.TotalSpentCents-t.BudgetTotal)/float64(t.BudgetTotal)*100)
		}
	}
	fmt.Fprintf(&b, "| 笔数 | %d 笔 |\n", sum.ExpenseCount)
	if sum.DaysCovered > 0 {
		fmt.Fprintf(&b, "| 日均 | %s |\n", formatMoney(sum.AveragePerDayCents, sum.Currency))
	}
	if len(sum.OtherCurrencies) > 0 {
		fmt.Fprintf(&b, "> ⚠️ 检测到其他币种的支出 (%s),未折算入总花费(避免汇率漂移)。\n",
			strings.Join(sum.OtherCurrencies, ", "))
	}
	b.WriteString("\n")

	// 按类别
	if len(sum.ByCategory) > 0 {
		b.WriteString("## 按类别\n\n")
		b.WriteString("| 类别 | 金额 | 占比 |\n|------|------|------|\n")
		for _, c := range sum.TopCategories {
			fmt.Fprintf(&b, "| %s | %s | %.1f%% |\n",
				expenseKindCN(c.Category), formatMoney(c.Cents, sum.Currency), c.Percent)
		}
		b.WriteString("\n")
	}

	// 按天(列出每天花费,有则按降序)
	if len(sum.ByDay) > 0 {
		b.WriteString("## 按天\n\n")
		dates := make([]string, 0, len(sum.ByDay))
		for d := range sum.ByDay {
			dates = append(dates, d)
		}
		sort.Strings(dates)
		b.WriteString("| 日期 | 花费 | 星期 |\n|------|------|------|\n")
		for _, d := range dates {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", d, formatMoney(sum.ByDay[d], sum.Currency), weekdayCN(d))
		}
		b.WriteString("\n")
	}

	// 明细列表
	if sum.ExpenseCount > 0 {
		expenses, err := s.repo.ListExpensesByTrip(ctx, tripID)
		if err == nil {
			b.WriteString("## 明细\n\n")
			b.WriteString("| 日期 | 类别 | 金额 | 备注 |\n|------|------|------|------|\n")
			for _, e := range expenses {
				note := strings.TrimSpace(e.Note)
				if len(note) > 40 {
					note = note[:39] + "…"
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
					e.Date, expenseKindCN(e.Category),
					formatMoney(e.AmountCents, firstOrDefault(e.Currency, sum.Currency)),
					note)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("---\n\n")
	b.WriteString("_本总结由 DevTools / Trip 工具自动生成,可贴 Notion / Markdown 阅读器。_\n")
	return b.String(), nil
}

func firstOrDefault(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func formatMoney(cents int64, currency string) string {
	if currency == "" {
		currency = "CNY"
	}
	// CNY 用 ¥,其它 ISO 4217 走 "¥1,234.56" 仍可读,只前缀货币代码
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	whole := cents / 100
	frac := cents % 100
	wholeStr := groupThousands(whole)
	switch currency {
	case "CNY":
		return fmt.Sprintf("%s¥%s.%02d", sign, wholeStr, frac)
	default:
		return fmt.Sprintf("%s%s %s.%02d", sign, currency, wholeStr, frac)
	}
}

func groupThousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func expenseKindCN(k ExpenseKind) string {
	switch k {
	case ExpenseTransport:
		return "交通"
	case ExpenseLodging:
		return "住宿"
	case ExpenseFood:
		return "餐饮"
	case ExpenseSight:
		return "门票"
	case ExpenseShopping:
		return "购物"
	case ExpenseMisc:
		return "杂项"
	default:
		return string(k)
	}
}

// ---- Markdown 渲染 ----

// RenderMarkdown 把 Trip 渲染成一份可读的行程单。
//
// 形如:
//
//	# 国庆 2026 粤港澳潮汕 14 日深度游
//
//	**日期**:2026-09-24 → 2026-10-07(14 天)
//	**覆盖城市**:深圳、香港、澳门、珠海、广州、潮汕
//	**通知邮箱**:foo@example.com
//
//	---
//
//	## D1 · 2026-09-24(周三) · 深圳
//
//	- 09:00  🚌 交通:抵达深圳宝安机场
//	  - 备注:入住福田/罗湖酒店
//	- 14:00  🏛️ 景点:深圳湾公园骑行
//	  - 地点:深圳湾公园
//
// 输出不含 HTML,适合直接贴 Notion / Markdown 阅读器。
func (s *Service) RenderMarkdown(ctx context.Context, tripID string) (string, error) {
	t, err := s.repo.GetTripWithDays(ctx, tripID)
	if err != nil {
		return "", err
	}
	return s.renderMarkdown(t), nil
}

func (s *Service) renderMarkdown(t *Trip) string {
	var b strings.Builder
	start, _ := time.Parse("2006-01-02", t.StartDate)
	end, _ := time.Parse("2006-01-02", t.EndDate)
	days := int(end.Sub(start).Hours()/24) + 1
	if days <= 0 {
		days = len(t.Days)
	}

	fmt.Fprintf(&b, "# %s\n\n", t.Name)
	if t.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", t.Description)
	}
	fmt.Fprintf(&b, "**日期**:%s → %s(%d 天)\n", t.StartDate, t.EndDate, days)
	if len(t.CoverCities) > 0 {
		fmt.Fprintf(&b, "**覆盖城市**:%s\n", strings.Join(t.CoverCities, " / "))
	}
	if len(t.Tags) > 0 {
		fmt.Fprintf(&b, "**标签**:%s\n", strings.Join(t.Tags, ", "))
	}
	if t.NotifyEmail != "" {
		fmt.Fprintf(&b, "**默认通知邮箱**:%s\n", t.NotifyEmail)
	}
	b.WriteString("\n---\n\n")

	for i, d := range t.Days {
		fmt.Fprintf(&b, "## D%d · %s(%s)", i+1, d.Date, weekdayCN(d.Date))
		if dest := destinationLabel(d.Destination); dest != "" {
			fmt.Fprintf(&b, " · %s", dest)
		}
		b.WriteString("\n\n")

		if len(d.Activities) == 0 {
			b.WriteString("- (本日暂无活动)\n\n")
			continue
		}
		for _, a := range d.Activities {
			icon := kindIcon(a.Kind)
			prefix := "- "
			if a.StartTime != "" {
				prefix = fmt.Sprintf("- %s  ", a.StartTime)
			}
			fmt.Fprintf(&b, "%s%s **%s**:%s", prefix, icon, kindCN(a.Kind), a.Title)
			if a.Location != "" {
				fmt.Fprintf(&b, " _(地点:%s)_", a.Location)
			}
			if a.DurationMin > 0 {
				fmt.Fprintf(&b, " _(时长:%d 分钟)_", a.DurationMin)
			}
			if a.RemindBeforeMinutes > 0 {
				fmt.Fprintf(&b, " ⏰提前 %d 分钟提醒", a.RemindBeforeMinutes)
			}
			b.WriteString("\n")
			if a.Note != "" {
				fmt.Fprintf(&b, "  - %s\n", a.Note)
			}
		}
		b.WriteString("\n")
	}

	if len(t.Days) == 0 {
		b.WriteString("> 本行程暂无 DayPlan,用 `multica trip add-day` 添加。\n")
	}
	return b.String()
}

func kindIcon(k ActivityKind) string {
	switch k {
	case ActivityTransit:
		return "🚌"
	case ActivitySight:
		return "🏛️"
	case ActivityFood:
		return "🍜"
	case ActivityLodging:
		return "🏨"
	case ActivityShopping:
		return "🛍️"
	case ActivityLeisure:
		return "🎯"
	default:
		return "•"
	}
}

func kindCN(k ActivityKind) string {
	switch k {
	case ActivityTransit:
		return "交通"
	case ActivitySight:
		return "景点"
	case ActivityFood:
		return "餐饮"
	case ActivityLodging:
		return "住宿"
	case ActivityShopping:
		return "购物"
	case ActivityLeisure:
		return "自由"
	default:
		return string(k)
	}
}

func weekdayCN(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[int(t.Weekday())]
}

func destinationLabel(d Destination) string {
	parts := []string{}
	if d.City != "" {
		parts = append(parts, d.City)
	}
	if d.Region != "" {
		parts = append(parts, d.Region)
	}
	if d.Country != "" && (len(parts) == 0 || d.Country != parts[0]) {
		parts = append(parts, d.Country)
	}
	return strings.Join(parts, " · ")
}

func dedupNonEmpty(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}