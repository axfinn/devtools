package trip

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Service 行程领域服务 —— 编排 Repository,提供业务用例与 Markdown 渲染。
//
// CLI 和未来 HTTP handler 都通过 Service 调用,不直接碰 Repository。
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService 构造行程服务;now 默认 time.Now,测试时可注入。
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// SetClock 注入时钟(用于测试)。
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// --- 用例 ---

// CreateTripInput 创建行程入参。
type CreateTripInput struct {
	Name      string
	StartDate string // YYYY-MM-DD
	EndDate   string // YYYY-MM-DD
	Summary   string
	Cities    string // 逗号分隔,如 "深圳,香港,澳门"
	Tags      string // 逗号分隔
}

// CreateTrip 创建一个空行程(没有 days)。
func (s *Service) CreateTrip(ctx context.Context, in CreateTripInput) (*Trip, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errors.New("trip name is required")
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return nil, fmt.Errorf("invalid start_date %q (want YYYY-MM-DD): %w", in.StartDate, err)
	}
	if _, err := time.Parse("2006-01-02", in.EndDate); err != nil {
		return nil, fmt.Errorf("invalid end_date %q (want YYYY-MM-DD): %w", in.EndDate, err)
	}
	t := &Trip{
		Name:      strings.TrimSpace(in.Name),
		StartDate: in.StartDate,
		EndDate:   in.EndDate,
		Summary:   in.Summary,
		Cities:    in.Cities,
		Tags:      in.Tags,
	}
	if err := s.repo.CreateTrip(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// AddDayInput 添加一天入参。
type AddDayInput struct {
	TripID   string
	Date     string // YYYY-MM-DD
	City     string
	Title    string
	Summary  string
}

// AddDay 给一个 trip 加一天。DayIndex 自动按 (date ASC, 现有最大 index+1) 计算,
// 同一 trip 内唯一。
func (s *Service) AddDay(ctx context.Context, in AddDayInput) (*DayPlan, error) {
	if _, err := s.repo.GetTrip(ctx, in.TripID); err != nil {
		return nil, err
	}
	if _, err := time.Parse("2006-01-02", in.Date); err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", in.Date, err)
	}
	existing, err := s.repo.ListDaysByTrip(ctx, in.TripID)
	if err != nil {
		return nil, err
	}
	nextIdx := 1
	for _, d := range existing {
		if d.DayIndex >= nextIdx {
			nextIdx = d.DayIndex + 1
		}
	}
	// 同日期合并为同一天的 index —— 简单策略:同 trip_id+date 视为同一天。
	// 但当前 schema 用 (trip_id, day_index) 唯一,允许同日期不同 index,保留为不同行,
	// 这对后期扩展(同一日多段)友好。这里默认追加到末尾。
	d := &DayPlan{
		TripID:   in.TripID,
		DayIndex: nextIdx,
		Date:     in.Date,
		City:     in.City,
		Title:    in.Title,
		Summary:  in.Summary,
	}
	if err := s.repo.CreateDayPlan(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// AddActivityInput 添加一个活动入参。
type AddActivityInput struct {
	DayPlanID string
	Kind      ActivityKind
	Time      string // "上午"/"下午"/"晚上"/"全天"/具体时刻
	Title     string
	Location  string
	Notes     string
	// DestinationName/Region 可选 —— 若提供则 upsert 到 destinations 表。
	DestinationName   string
	DestinationRegion string
}

// AddActivity 给 day_plan 加一项活动。Seq 自动追加到末尾。
func (s *Service) AddActivity(ctx context.Context, in AddActivityInput) (*Activity, error) {
	if in.DayPlanID == "" {
		return nil, errors.New("day_plan_id is required")
	}
	if in.Title == "" {
		return nil, errors.New("activity title is required")
	}
	if !ValidActivityKinds[in.Kind] {
		return nil, ErrInvalidActivityKind
	}
	existing, err := s.repo.ListActivitiesByDay(ctx, in.DayPlanID)
	if err != nil {
		return nil, err
	}
	nextSeq := 1
	for _, a := range existing {
		if a.Seq >= nextSeq {
			nextSeq = a.Seq + 1
		}
	}
	a := &Activity{
		DayPlanID: in.DayPlanID,
		Seq:       nextSeq,
		Kind:      in.Kind,
		Time:      in.Time,
		Title:     in.Title,
		Location:  in.Location,
		Notes:     in.Notes,
	}
	if in.DestinationName != "" {
		d, err := s.repo.UpsertDestination(ctx, &Destination{
			Name:   in.DestinationName,
			Region: in.DestinationRegion,
		})
		if err != nil {
			return nil, err
		}
		a.DestinationID = d.ID
	}
	if err := s.ValidateActivity(a); err != nil {
		return nil, err
	}
	if err := s.repo.CreateActivity(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// ValidateActivity 校验 activity。
func (s *Service) ValidateActivity(a *Activity) error {
	return a.Validate()
}

// Show 加载 trip 的完整聚合(trip + days + activities),用于渲染 Markdown。
func (s *Service) Show(ctx context.Context, tripID string) (*Trip, error) {
	t, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	days, err := s.repo.ListDaysByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	for _, d := range days {
		acts, err := s.repo.ListActivitiesByDay(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		// 防御性排序 —— 仓储层已 ORDER BY,但 list 之前已 sort 过会更稳。
		sort.SliceStable(acts, func(i, j int) bool { return acts[i].Seq < acts[j].Seq })
		d.Activities = acts
	}
	// Days 二次按 date 排序,避免入库顺序乱序时渲染跳天。
	sort.SliceStable(days, func(i, j int) bool {
		if days[i].Date == days[j].Date {
			return days[i].DayIndex < days[j].DayIndex
		}
		return days[i].Date < days[j].Date
	})
	t.Days = days
	return t, nil
}

// List 列出所有 trip(不展开 days,用于 `trip list`)。
func (s *Service) List(ctx context.Context) ([]*Trip, error) {
	return s.repo.ListTrips(ctx)
}

// DeleteTrip 删除 trip 级联 days+activities。
func (s *Service) DeleteTrip(ctx context.Context, tripID string) error {
	return s.repo.DeleteTrip(ctx, tripID)
}

// --- Markdown 渲染 ---

// RenderMarkdown 把 Trip + Days + Activities 渲染为可读的 Markdown 行程单。
//
// 风格:
//   - H1 行程名 + 概览表
//   - 按 day_index 顺序 H2 每个 DayPlan,日期 + 城市 + 当天标题
//   - 每个 Activity 一行 bullet,前缀 kind 图标 + 时刻
//   - 同一天按 kind 粗分组(transport → sight → food → lodging → shopping → note)
//     以保证"先交通再景点再吃饭"的阅读节奏
func (s *Service) RenderMarkdown(t *Trip) string {
	if t == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(t.Name)
	b.WriteString("\n\n")

	if t.Summary != "" {
		b.WriteString("> ")
		b.WriteString(t.Summary)
		b.WriteString("\n\n")
	}

	b.WriteString("| 项目 | 内容 |\n|------|------|\n")
	b.WriteString("| 日期 | ")
	b.WriteString(t.StartDate)
	b.WriteString(" → ")
	b.WriteString(t.EndDate)
	b.WriteString(" |\n")
	if t.Cities != "" {
		b.WriteString("| 覆盖城市 | ")
		b.WriteString(t.Cities)
		b.WriteString(" |\n")
	}
	if t.Tags != "" {
		b.WriteString("| 标签 | `")
		b.WriteString(t.Tags)
		b.WriteString("` |\n")
	}
	b.WriteString("| 总天数 | ")
	b.WriteString(fmt.Sprintf("%d", len(t.Days)))
	b.WriteString(" |\n")
	b.WriteString("| ID | `")
	b.WriteString(t.ID)
	b.WriteString("` |\n\n")

	// 按 day 渲染
	for _, d := range t.Days {
		b.WriteString("## ")
		if d.Title != "" {
			b.WriteString(d.Title)
		} else {
			b.WriteString(fmt.Sprintf("Day %d", d.DayIndex))
		}
		b.WriteString("\n\n")
		b.WriteString("**")
		b.WriteString(d.Date)
		b.WriteString("**")
		if d.City != "" {
			b.WriteString(" · ")
			b.WriteString(d.City)
		}
		b.WriteString("\n\n")
		if d.Summary != "" {
			b.WriteString("> ")
			b.WriteString(d.Summary)
			b.WriteString("\n\n")
		}
		if len(d.Activities) == 0 {
			b.WriteString("_(无活动)_\n\n")
			continue
		}
		// 按 kind 排序 —— 顺序:transport, sight, food, lodging, shopping, note
		order := map[ActivityKind]int{
			ActivityTransport: 0,
			ActivitySight:     1,
			ActivityFood:      2,
			ActivityLodging:   3,
			ActivityShopping:  4,
			ActivityNote:      5,
		}
		acts := make([]*Activity, len(d.Activities))
		copy(acts, d.Activities)
		sort.SliceStable(acts, func(i, j int) bool {
			oi, oj := order[acts[i].Kind], order[acts[j].Kind]
			if oi != oj {
				return oi < oj
			}
			return acts[i].Seq < acts[j].Seq
		})
		for _, a := range acts {
			b.WriteString("- ")
			b.WriteString(iconForKind(a.Kind))
			b.WriteString(" **")
			b.WriteString(a.Title)
			b.WriteString("**")
			if a.Time != "" {
				b.WriteString(" _(时刻:")
				b.WriteString(a.Time)
				b.WriteString(")_")
			}
			b.WriteString("\n")
			if a.Location != "" {
				b.WriteString("  - 地点: ")
				b.WriteString(a.Location)
				b.WriteString("\n")
			}
			if a.Notes != "" {
				b.WriteString("  - ")
				b.WriteString(a.Notes)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func iconForKind(k ActivityKind) string {
	switch k {
	case ActivityTransport:
		return "🚆"
	case ActivitySight:
		return "🗺️"
	case ActivityFood:
		return "🍜"
	case ActivityLodging:
		return "🏨"
	case ActivityShopping:
		return "🛍️"
	default:
		return "📝"
	}
}
