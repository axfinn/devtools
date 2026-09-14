package trip

import (
	"context"
	"errors"
	"fmt"
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