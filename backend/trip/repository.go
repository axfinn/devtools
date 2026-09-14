package trip

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound 仓储层统一"未找到"错误,Service 层翻译为业务错误。
var ErrNotFound = errors.New("trip: not found")

// Repository 是聚合根的仓储接口;供 Service 调用,也方便单测用 mock。
//
// 实现:SQLiteRepository,基于 *sql.DB,与 models.DB 共享同一连接池
// (即调用方传 db.Conn() 进来即可)。这意味着 trip 表与 pastes / shorturls /
// planner 等其他表共存于 ./data/paste.db,无独立 db 文件。
type Repository interface {
	EnsureSchema(ctx context.Context) error

	// Trip
	CreateTrip(ctx context.Context, t *Trip) error
	GetTrip(ctx context.Context, id string) (*Trip, error)
	GetTripWithDays(ctx context.Context, id string) (*Trip, error)
	ListTrips(ctx context.Context) ([]*Trip, error)
	UpdateTrip(ctx context.Context, t *Trip) error
	DeleteTrip(ctx context.Context, id string) error

	// DayPlan
	AddDay(ctx context.Context, d *DayPlan) error
	GetDay(ctx context.Context, id string) (*DayPlan, error)
	ListDaysByTrip(ctx context.Context, tripID string) ([]*DayPlan, error)
	DeleteDay(ctx context.Context, id string) error

	// Activity
	AddActivity(ctx context.Context, a *Activity) error
	ListActivitiesByDay(ctx context.Context, dayID string) ([]*Activity, error)
	UpdateActivityReminder(ctx context.Context, activityID string, remindBeforeMinutes int, remindEmails []string) error
	DeleteActivity(ctx context.Context, id string) error

	// Reminder scheduling:扫所有 remind_before_minutes > 0 的活动,
	// 返回触发时刻在 [from, to) 区间内、且尚未发送过提醒的活动。
	// sent_at 非空的活动不再返回,实现"幂等发一次"。
	ListDueReminders(ctx context.Context, from, to time.Time, limit int) ([]*ActivityReminderItem, error)
	MarkReminderSent(ctx context.Context, activityID string, sentAt time.Time) error
	ResetReminderSent(ctx context.Context, activityID string) (int64, error)
}

// ActivityReminderItem 把 Activity + 所属 Day + Trip 拼到一起,
// 提醒发送协程一次性拿到所有上下文,避免 N+1 查询。
type ActivityReminderItem struct {
	Activity *Activity
	Day      *DayPlan
	Trip     *Trip
}

// SQLiteRepository 是 Repository 的 SQLite 实现。
// 表名一律 trip_ 前缀,与 models 包已有表无冲突。
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository 把 *sql.DB 包装为 Repository;不做 schema 初始化。
// 调用方(Handler / CLI 启动期)需调 EnsureSchema 一次。
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// EnsureSchema 幂等建表;启动期调用一次即可。
func (r *SQLiteRepository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trip_trips (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '[]',
			cover_cities TEXT NOT NULL DEFAULT '[]',
			notify_email TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trip_days (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			date TEXT NOT NULL,
			destination TEXT NOT NULL DEFAULT '{}',
			"order" INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (trip_id) REFERENCES trip_trips(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_days_trip ON trip_days(trip_id)`,
		`CREATE TABLE IF NOT EXISTS trip_activities (
			id TEXT PRIMARY KEY,
			day_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			location TEXT NOT NULL DEFAULT '',
			destination TEXT NOT NULL DEFAULT '{}',
			start_time TEXT NOT NULL DEFAULT '',
			duration_min INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			"order" INTEGER NOT NULL DEFAULT 0,
			remind_before_minutes INTEGER NOT NULL DEFAULT 0,
			remind_emails TEXT NOT NULL DEFAULT '[]',
			remind_sent_at DATETIME NOT NULL DEFAULT '',
			FOREIGN KEY (day_id) REFERENCES trip_days(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_activities_day ON trip_activities(day_id)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_activities_reminder
			ON trip_activities(remind_before_minutes)
			WHERE remind_before_minutes > 0`,
	}
	for _, s := range stmts {
		if _, err := r.db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("trip: ensure schema (%s): %w", firstLine(s), err)
		}
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i > 0 {
		return s[:i]
	}
	return s
}

// ---- Trip CRUD ----

func (r *SQLiteRepository) CreateTrip(ctx context.Context, t *Trip) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	tags := mustJSON(t.Tags)
	cities := mustJSON(t.CoverCities)
	// go-sqlite3 在 _parse_time=true 下只解析 "2006-01-02 15:04:05.999999999-07:00" 格式,
	// 不接受 RFC3339 的 T 分隔符。复刻 go-sqlite3 内部解析格式以保证 Scan 成功。
	createdAt := formatSQLiteTime(t.CreatedAt)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trip_trips (id, name, description, start_date, end_date, tags, cover_cities, notify_email, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Description, t.StartDate, t.EndDate, tags, cities, t.NotifyEmail,
		createdAt, createdAt,
	)
	return err
}

func (r *SQLiteRepository) GetTrip(ctx context.Context, id string) (*Trip, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, start_date, end_date, tags, cover_cities, notify_email, created_at, updated_at
		 FROM trip_trips WHERE id = ?`, id)
	return scanTrip(row)
}

func (r *SQLiteRepository) GetTripWithDays(ctx context.Context, id string) (*Trip, error) {
	t, err := r.GetTrip(ctx, id)
	if err != nil {
		return nil, err
	}
	days, err := r.ListDaysByTrip(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, d := range days {
		acts, err := r.ListActivitiesByDay(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		d.Activities = acts
	}
	t.Days = days
	return t, nil
}

func (r *SQLiteRepository) ListTrips(ctx context.Context) ([]*Trip, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, start_date, end_date, tags, cover_cities, notify_email, created_at, updated_at
		 FROM trip_trips ORDER BY start_date DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) UpdateTrip(ctx context.Context, t *Trip) error {
	t.UpdatedAt = time.Now()
	res, err := r.db.ExecContext(ctx,
		`UPDATE trip_trips
		 SET name = ?, description = ?, start_date = ?, end_date = ?,
		     tags = ?, cover_cities = ?, notify_email = ?, updated_at = ?
		 WHERE id = ?`,
		t.Name, t.Description, t.StartDate, t.EndDate,
		mustJSON(t.Tags), mustJSON(t.CoverCities), t.NotifyEmail,
		formatSQLiteTime(t.UpdatedAt), t.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) DeleteTrip(ctx context.Context, id string) error {
	// ON DELETE CASCADE 处理 days + activities;先禁掉外键检查防御性提升。
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_trips WHERE id = ?`, id)
	return err
}

// ---- DayPlan CRUD ----

func (r *SQLiteRepository) AddDay(ctx context.Context, d *DayPlan) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	destJSON, _ := json.Marshal(d.Destination)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trip_days (id, trip_id, date, destination, "order")
		 VALUES (?, ?, ?, ?, ?)`,
		d.ID, d.TripID, d.Date, string(destJSON), d.Order,
	)
	return err
}

func (r *SQLiteRepository) GetDay(ctx context.Context, id string) (*DayPlan, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, trip_id, date, destination, "order" FROM trip_days WHERE id = ?`, id)
	return scanDay(row)
}

func (r *SQLiteRepository) ListDaysByTrip(ctx context.Context, tripID string) ([]*DayPlan, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, trip_id, date, destination, "order" FROM trip_days
		 WHERE trip_id = ? ORDER BY date ASC, "order" ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*DayPlan
	for rows.Next() {
		d, err := scanDay(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) DeleteDay(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_days WHERE id = ?`, id)
	return err
}

// ---- Activity CRUD ----

func (r *SQLiteRepository) AddActivity(ctx context.Context, a *Activity) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	destJSON, _ := json.Marshal(a.Destination)
	emails := mustJSON(a.RemindEmails)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trip_activities
		 (id, day_id, kind, title, location, destination, start_time, duration_min, note, "order", remind_before_minutes, remind_emails)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.DayID, string(a.Kind), a.Title, a.Location, string(destJSON),
		a.StartTime, a.DurationMin, a.Note, a.Order,
		a.RemindBeforeMinutes, emails,
	)
	return err
}

func (r *SQLiteRepository) ListActivitiesByDay(ctx context.Context, dayID string) ([]*Activity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, day_id, kind, title, location, destination, start_time, duration_min, note, "order",
		        remind_before_minutes, remind_emails
		 FROM trip_activities WHERE day_id = ? ORDER BY "order" ASC, start_time ASC, title ASC`, dayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) UpdateActivityReminder(ctx context.Context, activityID string, remindBeforeMinutes int, remindEmails []string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trip_activities
		 SET remind_before_minutes = ?, remind_emails = ?
		 WHERE id = ?`,
		remindBeforeMinutes, mustJSON(remindEmails), activityID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) DeleteActivity(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_activities WHERE id = ?`, id)
	return err
}

// ---- Reminder scheduling ----

func (r *SQLiteRepository) ListDueReminders(ctx context.Context, from, to time.Time, limit int) ([]*ActivityReminderItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	// 取所有 remind_before_minutes > 0 且未发送的活动,加载 DayPlan + Trip,
	// 在 Go 侧按 ActivityReminderTime 计算精确的触发时刻,落在 [from, to) 区间即返回。
	rows, err := r.db.QueryContext(ctx,
		`SELECT a.id, a.day_id, a.kind, a.title, a.location, a.destination, a.start_time, a.duration_min, a.note, a."order",
		        a.remind_before_minutes, a.remind_emails,
		        d.id, d.trip_id, d.date, d.destination, d."order",
		        t.id, t.name, t.description, t.start_date, t.end_date, t.tags, t.cover_cities, t.notify_email, t.created_at, t.updated_at
		 FROM trip_activities a
		 JOIN trip_days d ON d.id = a.day_id
		 JOIN trip_trips t ON t.id = d.trip_id
		 WHERE a.remind_before_minutes > 0 AND a.remind_sent_at = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ActivityReminderItem
	for rows.Next() {
		var (
			a     Activity
			d     DayPlan
			t     Trip
			destA string
			destD string
			tags  string
			cities string
			emails string
		)
		if err := rows.Scan(
			&a.ID, &a.DayID, &a.Kind, &a.Title, &a.Location, &destA, &a.StartTime, &a.DurationMin, &a.Note, &a.Order,
			&a.RemindBeforeMinutes, &emails,
			&d.ID, &d.TripID, &d.Date, &destD, &d.Order,
			&t.ID, &t.Name, &t.Description, &t.StartDate, &t.EndDate, &tags, &cities, &t.NotifyEmail, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(destA), &a.Destination)
		_ = json.Unmarshal([]byte(destD), &d.Destination)
		_ = json.Unmarshal([]byte(tags), &t.Tags)
		_ = json.Unmarshal([]byte(cities), &t.CoverCities)
		_ = json.Unmarshal([]byte(emails), &a.RemindEmails)

		trig, ok := ActivityReminderTime(&d, &a)
		if !ok {
			continue
		}
		// 触发时刻需落在 [from, to) 区间内:trig < from 或 trig >= to 都跳过。
		if trig.Before(from) || !trig.Before(to) {
			continue
		}
		d.Activities = nil
		out = append(out, &ActivityReminderItem{Activity: &a, Day: &d, Trip: &t})
		if len(out) >= limit {
			break
		}
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) MarkReminderSent(ctx context.Context, activityID string, sentAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE trip_activities SET remind_sent_at = ? WHERE id = ?`,
		formatSQLiteTime(sentAt), activityID,
	)
	return err
}

// ResetReminderSent 把活动的 remind_sent_at 清零,允许重发。
func (r *SQLiteRepository) ResetReminderSent(ctx context.Context, activityID string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trip_activities SET remind_sent_at = 0 WHERE id = ?`,
		activityID,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ---- helpers ----

// rowScanner 抽象 *sql.Row 和 *sql.Rows,scanTrip / scanDay / scanActivity 共用。
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(s rowScanner) (*Trip, error) {
	var (
		t      Trip
		tags   string
		cities string
	)
	err := s.Scan(&t.ID, &t.Name, &t.Description, &t.StartDate, &t.EndDate,
		&tags, &cities, &t.NotifyEmail, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(tags), &t.Tags)
	_ = json.Unmarshal([]byte(cities), &t.CoverCities)
	return &t, nil
}

func scanDay(s rowScanner) (*DayPlan, error) {
	var (
		d   DayPlan
		dst string
	)
	err := s.Scan(&d.ID, &d.TripID, &d.Date, &dst, &d.Order)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(dst), &d.Destination)
	return &d, nil
}

func scanActivity(s rowScanner) (*Activity, error) {
	var (
		a      Activity
		dest   string
		emails string
	)
	err := s.Scan(&a.ID, &a.DayID, &a.Kind, &a.Title, &a.Location, &dest,
		&a.StartTime, &a.DurationMin, &a.Note, &a.Order,
		&a.RemindBeforeMinutes, &emails)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(dest), &a.Destination)
	_ = json.Unmarshal([]byte(emails), &a.RemindEmails)
	return &a, nil
}

func mustJSON(v any) string {
	if v == nil {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// formatSQLiteTime 用 go-sqlite3 内部 SQLITE_TIME_FORMAT 写入 time.Time,
// 这样 driver 在 Scan 回 *time.Time 时不会报 "string into *time.Time"。
// 留空字符串等同于 "零值",保持 reminder 未发状态。
func formatSQLiteTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	// go-sqlite3 internal format: "2006-01-02 15:04:05.999999999-07:00"
	// 简化版:秒级精度即可;毫秒/纳秒为可选字段。
	return t.UTC().Format("2006-01-02 15:04:05.999999999-07:00")
}