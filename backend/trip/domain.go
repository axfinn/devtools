// Package trip 是「旅游行程助手」的 DDD 限界上下文。
//
// 设计原则:
//   - Trip 是聚合根,DayPlan 是聚合内的实体,Activity 是 DayPlan 内的实体。
//   - Destination 是值对象,不可变,只承载城市/区域/坐标/官方信息。
//   - Reminder 是 Activity 上的提醒配置(单位:分钟),由后台 cleanup 协程扫表发邮件。
//   - 数据访问走 Repository 接口;Service 是用例层,负责增删改查 + Markdown 渲染。
//
// 模块独立:不依赖 handlers / routes;由 backend/handlers/trip.go 做 HTTP 适配,
// 由 backend/cmd/trip 做 CLI 适配。
package trip

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ActivityKind 活动类型。
type ActivityKind string

const (
	ActivityTransit   ActivityKind = "transit"   // 交通(过关 / 高铁 / 大巴)
	ActivitySight     ActivityKind = "sight"     // 景点
	ActivityFood      ActivityKind = "food"      // 餐饮
	ActivityLodging   ActivityKind = "lodging"   // 住宿
	ActivityShopping  ActivityKind = "shopping"  // 购物
	ActivityLeisure   ActivityKind = "leisure"   // 自由活动 / 休息
	ActivityReminder  ActivityKind = "reminder"  // 提醒锚点(非实体活动,只用于在 Markdown 中显示一行提醒说明)
)

// ValidKinds 用于校验外部输入。
var ValidKinds = map[ActivityKind]bool{
	ActivityTransit: true, ActivitySight: true, ActivityFood: true,
	ActivityLodging: true, ActivityShopping: true, ActivityLeisure: true,
}

// ExpenseKind 费用分类。复用了大部分 ActivityKind 语义,
// 但加了 ExpenseMisc(杂项,如停车费/小费)和 ExpenseTicket(门票,
// 跟 lodging 区分以便按类别渲染汇总时不被遗漏)。
type ExpenseKind string

const (
	ExpenseTransport ExpenseKind = "transport" // 交通
	ExpenseLodging   ExpenseKind = "lodging"   // 住宿
	ExpenseFood      ExpenseKind = "food"      // 餐饮
	ExpenseSight     ExpenseKind = "sight"     // 门票/景点
	ExpenseShopping  ExpenseKind = "shopping"  // 购物
	ExpenseMisc      ExpenseKind = "misc"      // 杂项(停车/小费/通讯/换汇等)
)

// ValidExpenseKinds 用于校验外部输入。
var ValidExpenseKinds = map[ExpenseKind]bool{
	ExpenseTransport: true, ExpenseLodging: true, ExpenseFood: true,
	ExpenseSight: true, ExpenseShopping: true, ExpenseMisc: true,
}

// Destination 值对象:城市/区域/经纬度/官方信息。
type Destination struct {
	City      string  `json:"city"`                // 城市(如 "深圳" / "澳门")
	Region    string  `json:"region,omitempty"`    // 区域(如 "氹仔" / "尖沙咀")
	Country   string  `json:"country,omitempty"`   // 国家/地区
	Latitude  float64 `json:"latitude,omitempty"`  // 纬度
	Longitude float64 `json:"longitude,omitempty"` // 经度
	Note      string  `json:"note,omitempty"`      // 备注(签证/天气/海拔...)
}

// Activity DayPlan 内的实体:一个具体事件。
type Activity struct {
	ID         string       `json:"id"`
	DayID      string       `json:"day_id"`
	Kind       ActivityKind `json:"kind"`
	Title      string       `json:"title"`
	Location   string       `json:"location,omitempty"`   // 自由文本,如 "尖沙咀星光大道"
	Destination Destination  `json:"destination"`          // 值对象;空也合法
	StartTime  string       `json:"start_time,omitempty"` // "HH:MM",留空表示整天
	DurationMin int         `json:"duration_min,omitempty"`
	Note       string       `json:"note,omitempty"`
	Order      int          `json:"order"`

	// 提醒配置:Activity 时间 - remind_before_minutes = 提醒时刻。
	// 0 表示不提醒。后台 cleanup 协程扫表,到点调 notif.Send 发邮件。
	RemindBeforeMinutes int      `json:"remind_before_minutes,omitempty"`
	RemindEmails        []string `json:"remind_emails,omitempty"` // 空 = 走 Trip.NotifyEmail
}

// DayPlan Trip 下的实体:一天的所有活动。
type DayPlan struct {
	ID         string     `json:"id"`
	TripID     string     `json:"trip_id"`
	Date       string     `json:"date"` // YYYY-MM-DD(本地日期;不存时区,出行者按本地理解)
	Destination Destination `json:"destination"` // 当天主目的地(冗余方便查询)
	Order      int        `json:"order"`
	Activities []*Activity `json:"activities,omitempty"`
}

// Trip 聚合根。
type Trip struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	StartDate   string    `json:"start_date"` // YYYY-MM-DD
	EndDate     string    `json:"end_date"`   // YYYY-MM-DD
	Tags        []string  `json:"tags,omitempty"`
	CoverCities []string  `json:"cover_cities,omitempty"` // 冗余字段,首日 / 末日方便筛选
	NotifyEmail string    `json:"notify_email,omitempty"` // 行程级默认提醒收件人

	// 预算(整程级)。金额用「分」存储避免浮点漂移,前端按 currency 渲染。
	// BudgetTotal == 0 表示未设置预算。
	BudgetTotal    int64  `json:"budget_total,omitempty"`    // 单位:分
	BudgetCurrency string `json:"budget_currency,omitempty"` // 3 字母 ISO 4217,默认 "CNY"

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Days      []*DayPlan `json:"days,omitempty"`
}

// Expense 一次具体开销。与 Trip + Activity 解耦,允许一笔 expense 不挂在任何 activity 下
// (用户随手记一笔也可),也允许一笔挂到具体 activity(把活动当作 cost bucket)。
type Expense struct {
	ID            string      `json:"id"`
	TripID        string      `json:"trip_id"`
	Date          string      `json:"date"` // YYYY-MM-DD;允许不在 Trip 日期范围内(边角账)
	Category      ExpenseKind `json:"category"`
	AmountCents   int64       `json:"amount_cents"`           // 单位:分
	Currency      string      `json:"currency,omitempty"`     // 默认沿用 Trip.BudgetCurrency
	Note          string      `json:"note,omitempty"`         // 自由文本
	PaymentMethod string      `json:"payment_method,omitempty"` // cash / card / alipay / wechat / other
	ActivityID    string      `json:"activity_id,omitempty"`  // 可选:关联到具体 activity
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// Validate 日期合法性 + 名称非空。
func (t *Trip) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("trip name is required")
	}
	start, err := time.Parse("2006-01-02", t.StartDate)
	if err != nil {
		return fmt.Errorf("invalid start_date %q: %w", t.StartDate, err)
	}
	end, err := time.Parse("2006-01-02", t.EndDate)
	if err != nil {
		return fmt.Errorf("invalid end_date %q: %w", t.EndDate, err)
	}
	if end.Before(start) {
		return fmt.Errorf("end_date %s is before start_date %s", t.EndDate, t.StartDate)
	}
	if t.BudgetTotal < 0 {
		return errors.New("budget_total must be >= 0")
	}
	if t.BudgetCurrency != "" {
		if len(t.BudgetCurrency) != 3 {
			return fmt.Errorf("budget_currency must be ISO 4217 (3 letters), got %q", t.BudgetCurrency)
		}
	}
	return nil
}

// Validate Expense 字段:日期 / 类别 / 金额。
func (e *Expense) Validate() error {
	if e.TripID == "" {
		return errors.New("expense missing trip_id")
	}
	if _, err := time.Parse("2006-01-02", e.Date); err != nil {
		return fmt.Errorf("invalid expense date %q: %w", e.Date, err)
	}
	if !ValidExpenseKinds[e.Category] {
		return fmt.Errorf("invalid expense category %q", e.Category)
	}
	if e.AmountCents <= 0 {
		return errors.New("amount_cents must be > 0")
	}
	if e.Currency != "" && len(e.Currency) != 3 {
		return fmt.Errorf("currency must be ISO 4217 (3 letters), got %q", e.Currency)
	}
	return nil
}

// Validate DayPlan 字段。
func (d *DayPlan) Validate() error {
	if _, err := time.Parse("2006-01-02", d.Date); err != nil {
		return fmt.Errorf("invalid day date %q: %w", d.Date, err)
	}
	if d.TripID == "" {
		return errors.New("day plan missing trip_id")
	}
	return nil
}

// Validate Activity 字段。
func (a *Activity) Validate() error {
	if strings.TrimSpace(a.Title) == "" {
		return errors.New("activity title is required")
	}
	if !ValidKinds[a.Kind] {
		return fmt.Errorf("invalid activity kind %q", a.Kind)
	}
	if a.StartTime != "" {
		if _, err := time.Parse("15:04", a.StartTime); err != nil {
			return fmt.Errorf("invalid start_time %q (want HH:MM): %w", a.StartTime, err)
		}
	}
	if a.RemindBeforeMinutes < 0 {
		return errors.New("remind_before_minutes must be >= 0")
	}
	return nil
}

// ActivityTime 把 Activity.Date + Activity.StartTime 合成一个 time.Time。
// 用于提醒调度;StartTime 为空时取当天 09:00 作为提醒锚点。
func ActivityTime(day *DayPlan, a *Activity) (time.Time, bool) {
	dayDate, err := time.ParseInLocation("2006-01-02", day.Date, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	if a.StartTime == "" {
		return time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(), 9, 0, 0, 0, time.Local), true
	}
	t, err := time.ParseInLocation("15:04", a.StartTime, dayDate.Location())
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), true
}

// ActivityReminderTime 算出提醒真正触发的时刻。
// 公式:activity_at - remind_before_minutes。
// 当 remind_before_minutes == 0 时返回 (zero, false),表示不提醒。
func ActivityReminderTime(day *DayPlan, a *Activity) (time.Time, bool) {
	if a.RemindBeforeMinutes <= 0 {
		return time.Time{}, false
	}
	at, ok := ActivityTime(day, a)
	if !ok {
		return time.Time{}, false
	}
	return at.Add(-time.Duration(a.RemindBeforeMinutes) * time.Minute), true
}