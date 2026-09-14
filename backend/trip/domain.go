// Package trip 实现"旅游行程助手"限界上下文。
//
// 领域模型(DDD):
//   - Trip         聚合根 —— 一次完整旅行
//   - DayPlan      Trip 下的实体 —— 某一天的计划
//   - Activity     DayPlan 下的实体 —— 某天的活动/景点/餐饮/住宿...
//   - Destination  值对象 —— 城市/区域的目的地参考信息
//
// 仓储接口在 repository.go,业务编排与 Markdown 渲染在 service.go,
// 14 天国庆粤港澳潮汕真实行程的 fixture 在 seed.go。
package trip

import (
	"errors"
	"time"
)

// ActivityKind 活动类型 —— 值对象,枚举。
type ActivityKind string

const (
	ActivityTransport  ActivityKind = "transport"  // 交通(过关/高铁/打车/船)
	ActivitySight      ActivityKind = "sight"      // 景点/观光
	ActivityFood       ActivityKind = "food"       // 餐饮
	ActivityLodging    ActivityKind = "lodging"    // 住宿
	ActivityShopping   ActivityKind = "shopping"   // 购物
	ActivityNote       ActivityKind = "note"       // 自由活动 / 备注
)

// ValidActivityKinds 用于校验入参。
var ValidActivityKinds = map[ActivityKind]bool{
	ActivityTransport: true,
	ActivitySight:     true,
	ActivityFood:      true,
	ActivityLodging:   true,
	ActivityShopping:  true,
	ActivityNote:      true,
}

// KindLabel 中文标签 —— 渲染用。
var KindLabel = map[ActivityKind]string{
	ActivityTransport: "交通",
	ActivitySight:     "景点",
	ActivityFood:      "餐饮",
	ActivityLodging:   "住宿",
	ActivityShopping:  "购物",
	ActivityNote:      "备注",
}

// Trip 行程聚合根。
type Trip struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"` // YYYY-MM-DD,出发日期
	EndDate   string    `json:"end_date"`   // YYYY-MM-DD,返程日期
	Summary   string    `json:"summary"`    // 一句话描述
	Cities    string    `json:"cities"`     // 覆盖城市列表(逗号分隔),冗余便于检索
	Tags      string    `json:"tags"`       // 标签(逗号分隔),便于检索
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联对象(由 Repository 在 Load 时填充,不持久化到 trips 表)
	Days []*DayPlan `json:"days,omitempty"`
}

// DayPlan 每日计划 —— Trip 下的实体。
type DayPlan struct {
	ID        string `json:"id"`
	TripID    string `json:"trip_id"`
	DayIndex  int    `json:"day_index"` // 1-based,在 trip 内的顺序
	Date      string `json:"date"`      // YYYY-MM-DD
	City      string `json:"city"`      // 当天主城市(用于分组与检索)
	Title     string `json:"title"`     // 当天一句话,如 "D1 深圳集合日"
	Summary   string `json:"summary"`

	Activities []*Activity `json:"activities,omitempty"`
}

// Activity 单个活动/景点 —— DayPlan 下的实体。
type Activity struct {
	ID            string       `json:"id"`
	DayPlanID     string       `json:"day_plan_id"`
	Seq           int          `json:"seq"`         // 一天内的顺序,1-based
	Kind          ActivityKind `json:"kind"`        // transport/sight/food/...
	Time          string       `json:"time"`        // "上午"/"下午"/"晚上"/"全天"/"09:30"
	Title         string       `json:"title"`       // 活动名,如 "维多利亚港星光大道"
	Location      string       `json:"location"`    // 具体地点
	Notes         string       `json:"notes"`       // 备注/推荐
	DestinationID string       `json:"destination_id,omitempty"`
}

// Destination 目的地值对象 —— 城市/区域参考信息。
type Destination struct {
	ID     string `json:"id"`
	Name   string `json:"name"`   // 城市/区域名,如 "深圳"/"氹仔"
	Region string `json:"region"` // 大区/省/国家,如 "广东"/"澳门"
	Info   string `json:"info"`   // 简短介绍
}

// Domain-level 错误。
var (
	ErrTripNotFound       = errors.New("trip not found")
	ErrDayNotFound        = errors.New("day plan not found")
	ErrInvalidActivityKind = errors.New("invalid activity kind")
)

// Validate Activity 入参合法性。
func (a *Activity) Validate() error {
	if a.Title == "" {
		return errors.New("activity title is required")
	}
	if !ValidActivityKinds[a.Kind] {
		return ErrInvalidActivityKind
	}
	return nil
}
