package trip

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Repository 行程仓储接口(DDD 风格:抽象聚合持久化)。
//
// 真实实现是 SQLiteTripRepository。任何 *sql.DB 实例都可注入,使得 CLI
// 和未来的 HTTP handler 共用同一套语义。
type Repository interface {
	InitSchema(ctx context.Context) error

	// Trip
	CreateTrip(ctx context.Context, t *Trip) error
	GetTrip(ctx context.Context, id string) (*Trip, error)
	ListTrips(ctx context.Context) ([]*Trip, error)
	UpdateTripSummary(ctx context.Context, id, summary, cities, tags string) error
	DeleteTrip(ctx context.Context, id string) error

	// DayPlan
	CreateDayPlan(ctx context.Context, d *DayPlan) error
	ListDaysByTrip(ctx context.Context, tripID string) ([]*DayPlan, error)
	DeleteDayPlan(ctx context.Context, id string) error

	// Activity
	CreateActivity(ctx context.Context, a *Activity) error
	ListActivitiesByDay(ctx context.Context, dayPlanID string) ([]*Activity, error)

	// Destination(值对象)
	UpsertDestination(ctx context.Context, d *Destination) (*Destination, error)
}

// SQLiteRepository 是 Repository 的 SQLite 实现。
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository 用一个 *sql.DB 构造仓储。
// 调用方负责连接生命周期(关闭、conn pool 等)。
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// OpenSQLite 打开一个 SQLite 连接并启用 _parse_time=true。
// 这是 cmd/trip CLI 单独使用的便捷构造。
func OpenSQLite(path string) (*SQLiteRepository, error) {
	conn, err := sql.Open("sqlite3", path+"?_parse_time=true")
	if err != nil {
		return nil, err
	}
	// 单进程 CLI 顺序访问,保持 1 连接避免 :memory: 并发坑。
	conn.SetMaxOpenConns(1)
	if err := conn.PingContext(context.Background()); err != nil {
		conn.Close()
		return nil, err
	}
	return &SQLiteRepository{db: conn}, nil
}

// Close 关闭底层连接。
func (r *SQLiteRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// generateID 与 models.generateID 等价的 hex id 生成 —— 16 字节 = 32 字符,
// 比 8 字符更长以容纳更多 fixture(避免和 paste/shorturl 等 8 字符 id 撞库)。
func generateID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)[:n]
}

// --- schema ---

const schema = `
CREATE TABLE IF NOT EXISTS trip_trips (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	start_date TEXT NOT NULL,
	end_date TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	cities TEXT NOT NULL DEFAULT '',
	tags TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_trip_trips_start ON trip_trips(start_date);

CREATE TABLE IF NOT EXISTS trip_day_plans (
	id TEXT PRIMARY KEY,
	trip_id TEXT NOT NULL,
	day_index INTEGER NOT NULL,
	date TEXT NOT NULL,
	city TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_trip_day_plans_trip ON trip_day_plans(trip_id, day_index);
CREATE UNIQUE INDEX IF NOT EXISTS idx_trip_day_plans_unique ON trip_day_plans(trip_id, day_index);

CREATE TABLE IF NOT EXISTS trip_activities (
	id TEXT PRIMARY KEY,
	day_plan_id TEXT NOT NULL,
	seq INTEGER NOT NULL,
	kind TEXT NOT NULL,
	time TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL,
	location TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	destination_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_trip_activities_day ON trip_activities(day_plan_id, seq);

CREATE TABLE IF NOT EXISTS trip_destinations (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	region TEXT NOT NULL DEFAULT '',
	info TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_trip_destinations_name ON trip_destinations(name, region);
`

// InitSchema 建表;幂等。
func (r *SQLiteRepository) InitSchema(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("trip: init schema: %w", err)
	}
	return nil
}

// --- Trip ---

func (r *SQLiteRepository) CreateTrip(ctx context.Context, t *Trip) error {
	if t.ID == "" {
		t.ID = generateID(12)
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_trips (id, name, start_date, end_date, summary, cities, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.Name, t.StartDate, t.EndDate, t.Summary, t.Cities, t.Tags, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("trip: create: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetTrip(ctx context.Context, id string) (*Trip, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, start_date, end_date, summary, cities, tags, created_at, updated_at
		FROM trip_trips WHERE id = ?
	`, id)
	var t Trip
	if err := row.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.Summary, &t.Cities, &t.Tags, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTripNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *SQLiteRepository) ListTrips(ctx context.Context) ([]*Trip, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, start_date, end_date, summary, cities, tags, created_at, updated_at
		FROM trip_trips ORDER BY start_date DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Trip
	for rows.Next() {
		var t Trip
		if err := rows.Scan(&t.ID, &t.Name, &t.StartDate, &t.EndDate, &t.Summary, &t.Cities, &t.Tags, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) UpdateTripSummary(ctx context.Context, id, summary, cities, tags string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE trip_trips SET summary = ?, cities = ?, tags = ?, updated_at = ? WHERE id = ?
	`, summary, cities, tags, time.Now(), id)
	return err
}

func (r *SQLiteRepository) DeleteTrip(ctx context.Context, id string) error {
	// 先删 day_plans,再删 activities,最后删 trip —— 顺序保证无悬挂引用
	if _, err := r.db.ExecContext(ctx, `DELETE FROM trip_activities WHERE day_plan_id IN (SELECT id FROM trip_day_plans WHERE trip_id = ?)`, id); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM trip_day_plans WHERE trip_id = ?`, id); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_trips WHERE id = ?`, id)
	return err
}

// --- DayPlan ---

func (r *SQLiteRepository) CreateDayPlan(ctx context.Context, d *DayPlan) error {
	if d.ID == "" {
		d.ID = generateID(12)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_day_plans (id, trip_id, day_index, date, city, title, summary)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.TripID, d.DayIndex, d.Date, d.City, d.Title, d.Summary)
	if err != nil {
		return fmt.Errorf("trip: create day: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListDaysByTrip(ctx context.Context, tripID string) ([]*DayPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, day_index, date, city, title, summary
		FROM trip_day_plans WHERE trip_id = ? ORDER BY day_index ASC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*DayPlan
	for rows.Next() {
		var d DayPlan
		if err := rows.Scan(&d.ID, &d.TripID, &d.DayIndex, &d.Date, &d.City, &d.Title, &d.Summary); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) DeleteDayPlan(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM trip_activities WHERE day_plan_id = ?;
		DELETE FROM trip_day_plans WHERE id = ?;
	`, id, id)
	return err
}

// --- Activity ---

func (r *SQLiteRepository) CreateActivity(ctx context.Context, a *Activity) error {
	if a.ID == "" {
		a.ID = generateID(12)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_activities (id, day_plan_id, seq, kind, time, title, location, notes, destination_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.DayPlanID, a.Seq, string(a.Kind), a.Time, a.Title, a.Location, a.Notes, a.DestinationID)
	if err != nil {
		return fmt.Errorf("trip: create activity: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListActivitiesByDay(ctx context.Context, dayPlanID string) ([]*Activity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, day_plan_id, seq, kind, time, title, location, notes, destination_id
		FROM trip_activities WHERE day_plan_id = ? ORDER BY seq ASC
	`, dayPlanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Activity
	for rows.Next() {
		var a Activity
		var kind string
		if err := rows.Scan(&a.ID, &a.DayPlanID, &a.Seq, &kind, &a.Time, &a.Title, &a.Location, &a.Notes, &a.DestinationID); err != nil {
			return nil, err
		}
		a.Kind = ActivityKind(kind)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// --- Destination ---

func (r *SQLiteRepository) UpsertDestination(ctx context.Context, d *Destination) (*Destination, error) {
	if d.Name == "" {
		return nil, errors.New("destination name is required")
	}
	// 先按 name+region 查;找到就返回,没找到就建一条。
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, region, info FROM trip_destinations
		WHERE name = ? AND region = ? LIMIT 1
	`, d.Name, d.Region)
	var existing Destination
	if err := row.Scan(&existing.ID, &existing.Name, &existing.Region, &existing.Info); err == nil {
		return &existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if d.ID == "" {
		d.ID = generateID(10)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_destinations (id, name, region, info) VALUES (?, ?, ?, ?)
	`, d.ID, d.Name, d.Region, d.Info)
	if err != nil {
		return nil, err
	}
	return d, nil
}
