package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

const monitoringTimeLayout = "2006-01-02 15:04:05"

type HTTPRequestLog struct {
	ID            int64     `json:"id"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	Route         string    `json:"route"`
	StatusCode    int       `json:"status_code"`
	LatencyMS     int64     `json:"latency_ms"`
	ClientIP      string    `json:"client_ip"`
	VisitorID     string    `json:"-"`
	UserAgent     string    `json:"user_agent"`
	RequestBytes  int64     `json:"request_bytes"`
	ResponseBytes int64     `json:"response_bytes"`
	ErrorMessage  string    `json:"error_message"`
	Archived      bool      `json:"archived"`
	CreatedAt     time.Time `json:"created_at"`
}

type MonitoringOverview struct {
	RequestCount  int64                     `json:"request_count"`
	UniqueIPs     int64                     `json:"unique_ips"`
	DAU           int64                     `json:"dau"`
	WAU           int64                     `json:"wau"`
	MAU           int64                     `json:"mau"`
	Status2xx     int64                     `json:"status_2xx"`
	Status3xx     int64                     `json:"status_3xx"`
	Status4xx     int64                     `json:"status_4xx"`
	Status5xx     int64                     `json:"status_5xx"`
	AverageMS     float64                   `json:"average_latency_ms"`
	MaxLatencyMS  int64                     `json:"max_latency_ms"`
	RequestBytes  int64                     `json:"request_bytes"`
	ResponseBytes int64                     `json:"response_bytes"`
	Timeline      []MonitoringTimelinePoint `json:"timeline"`
}

type MonitoringTimelinePoint struct {
	Bucket       string `json:"bucket"`
	Requests     int64  `json:"requests"`
	Status2xx    int64  `json:"status_2xx"`
	Status3xx    int64  `json:"status_3xx"`
	Status4xx    int64  `json:"status_4xx"`
	Status5xx    int64  `json:"status_5xx"`
	AverageMS    int64  `json:"average_latency_ms"`
	ResponseSize int64  `json:"response_bytes"`
}

type MonitoringEndpointStat struct {
	Route        string  `json:"route"`
	Method       string  `json:"method"`
	Requests     int64   `json:"requests"`
	Errors       int64   `json:"errors"`
	ErrorRate    float64 `json:"error_rate"`
	AverageMS    float64 `json:"average_latency_ms"`
	MaxLatencyMS int64   `json:"max_latency_ms"`
	ResponseSize int64   `json:"response_bytes"`
}

type MonitoringArchiveResult struct {
	ArchivedDetails int64 `json:"archived_details"`
	DeletedDetails  int64 `json:"deleted_details"`
	DeletedHourly   int64 `json:"deleted_hourly"`
	DeletedVisitors int64 `json:"deleted_visitors"`
}

type MonitoringStorageStats struct {
	DetailRows         int64      `json:"detail_rows"`
	ArchivedDetailRows int64      `json:"archived_detail_rows"`
	HourlyRows         int64      `json:"hourly_rows"`
	DailyVisitorRows   int64      `json:"daily_visitor_rows"`
	DatabaseBytes      int64      `json:"database_bytes"`
	OldestDetailAt     *time.Time `json:"oldest_detail_at,omitempty"`
	NewestDetailAt     *time.Time `json:"newest_detail_at,omitempty"`
	LastArchiveAt      *time.Time `json:"last_archive_at,omitempty"`
}

type AIMonitoringSummary struct {
	RequestCount int64                   `json:"request_count"`
	ErrorCount   int64                   `json:"error_count"`
	AverageMS    float64                 `json:"average_latency_ms"`
	InputTokens  int64                   `json:"input_tokens"`
	OutputTokens int64                   `json:"output_tokens"`
	TotalTokens  int64                   `json:"total_tokens"`
	TotalCost    float64                 `json:"total_cost"`
	Models       []AIMonitoringModelStat `json:"models"`
}

type AIMonitoringModelStat struct {
	Model       string  `json:"model"`
	Provider    string  `json:"provider"`
	Requests    int64   `json:"requests"`
	Errors      int64   `json:"errors"`
	TotalTokens int64   `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
	AverageMS   float64 `json:"average_latency_ms"`
}

var monitoringArchiveMu sync.Mutex

func init() {
	RegisterInit("监控统计(http_request_logs)", func(db *DB) error { return db.InitMonitoring() })
}

func (db *DB) InitMonitoring() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS http_request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			route TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			client_ip TEXT NOT NULL DEFAULT '',
			visitor_id TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			request_bytes INTEGER NOT NULL DEFAULT 0,
			response_bytes INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			archived INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_http_logs_created ON http_request_logs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_http_logs_route_created ON http_request_logs(route, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_http_logs_status_created ON http_request_logs(status_code, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_http_logs_visitor_created ON http_request_logs(visitor_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_http_logs_archive_created ON http_request_logs(archived, created_at);

		CREATE TABLE IF NOT EXISTS http_request_hourly (
			bucket_start DATETIME NOT NULL,
			method TEXT NOT NULL,
			route TEXT NOT NULL,
			request_count INTEGER NOT NULL DEFAULT 0,
			status_2xx INTEGER NOT NULL DEFAULT 0,
			status_3xx INTEGER NOT NULL DEFAULT 0,
			status_4xx INTEGER NOT NULL DEFAULT 0,
			status_5xx INTEGER NOT NULL DEFAULT 0,
			latency_total_ms INTEGER NOT NULL DEFAULT 0,
			latency_max_ms INTEGER NOT NULL DEFAULT 0,
			request_bytes INTEGER NOT NULL DEFAULT 0,
			response_bytes INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket_start, method, route)
		);
		CREATE INDEX IF NOT EXISTS idx_http_hourly_bucket ON http_request_hourly(bucket_start DESC);
		CREATE INDEX IF NOT EXISTS idx_http_hourly_route ON http_request_hourly(route, bucket_start DESC);

		CREATE TABLE IF NOT EXISTS http_daily_visitors (
			day TEXT NOT NULL,
			visitor_id TEXT NOT NULL,
			first_seen_at DATETIME NOT NULL,
			last_seen_at DATETIME NOT NULL,
			request_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (day, visitor_id)
		);
		CREATE INDEX IF NOT EXISTS idx_http_visitors_day ON http_daily_visitors(day DESC);

		CREATE TABLE IF NOT EXISTS proxy_session_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_type TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT '',
			route TEXT NOT NULL DEFAULT '',
			target TEXT NOT NULL DEFAULT '',
			client_ip TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			status_code INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			bytes_in INTEGER NOT NULL DEFAULT 0,
			bytes_out INTEGER NOT NULL DEFAULT 0,
			frames_in INTEGER NOT NULL DEFAULT 0,
			frames_out INTEGER NOT NULL DEFAULT 0,
			close_reason TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_proxy_sessions_created ON proxy_session_logs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_proxy_sessions_type_created ON proxy_session_logs(session_type, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_proxy_sessions_route_created ON proxy_session_logs(route, created_at DESC);
	`)
	return err
}

func HashMonitoringVisitor(clientIP string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(clientIP)))
	return hex.EncodeToString(sum[:])
}

func (db *DB) CreateHTTPRequestLog(item *HTTPRequestLog) error {
	if item == nil {
		return fmt.Errorf("request log is nil")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	if item.VisitorID == "" {
		item.VisitorID = HashMonitoringVisitor(item.ClientIP)
	}
	archived := 0
	if item.Archived {
		archived = 1
	}
	result, err := db.conn.Exec(`
		INSERT INTO http_request_logs (
			method, path, route, status_code, latency_ms, client_ip, visitor_id,
			user_agent, request_bytes, response_bytes, error_message, archived, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Method, item.Path, item.Route, item.StatusCode, item.LatencyMS, item.ClientIP,
		item.VisitorID, item.UserAgent, item.RequestBytes, item.ResponseBytes,
		item.ErrorMessage, archived, item.CreatedAt.UTC(),
	)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	return nil
}

func (db *DB) ArchiveHTTPRequestLogs(archiveBefore time.Time, detailRetentionDays, archiveRetentionDays int) (*MonitoringArchiveResult, error) {
	monitoringArchiveMu.Lock()
	defer monitoringArchiveMu.Unlock()

	result := &MonitoringArchiveResult{}
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.QueryRow(`SELECT COUNT(*) FROM http_request_logs WHERE archived = 0 AND created_at < ?`, archiveBefore.UTC()).Scan(&result.ArchivedDetails); err != nil {
		return nil, err
	}

	if result.ArchivedDetails > 0 {
		if _, err := tx.Exec(`
			INSERT INTO http_daily_visitors (day, visitor_id, first_seen_at, last_seen_at, request_count)
			SELECT strftime('%Y-%m-%d', created_at), visitor_id, MIN(created_at), MAX(created_at), COUNT(*)
			FROM http_request_logs
			WHERE archived = 0 AND created_at < ?
			GROUP BY strftime('%Y-%m-%d', created_at), visitor_id
			ON CONFLICT(day, visitor_id) DO UPDATE SET
				first_seen_at = MIN(first_seen_at, excluded.first_seen_at),
				last_seen_at = MAX(last_seen_at, excluded.last_seen_at),
				request_count = request_count + excluded.request_count`, archiveBefore.UTC()); err != nil {
			return nil, err
		}

		if _, err := tx.Exec(`
			INSERT INTO http_request_hourly (
				bucket_start, method, route, request_count, status_2xx, status_3xx,
				status_4xx, status_5xx, latency_total_ms, latency_max_ms, request_bytes, response_bytes
			)
			SELECT strftime('%Y-%m-%d %H:00:00', created_at), method, route, COUNT(*),
				SUM(CASE WHEN status_code BETWEEN 200 AND 299 THEN 1 ELSE 0 END),
				SUM(CASE WHEN status_code BETWEEN 300 AND 399 THEN 1 ELSE 0 END),
				SUM(CASE WHEN status_code BETWEEN 400 AND 499 THEN 1 ELSE 0 END),
				SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END),
				SUM(latency_ms), MAX(latency_ms), SUM(request_bytes), SUM(response_bytes)
			FROM http_request_logs
			WHERE archived = 0 AND created_at < ?
			GROUP BY strftime('%Y-%m-%d %H:00:00', created_at), method, route
			ON CONFLICT(bucket_start, method, route) DO UPDATE SET
				request_count = request_count + excluded.request_count,
				status_2xx = status_2xx + excluded.status_2xx,
				status_3xx = status_3xx + excluded.status_3xx,
				status_4xx = status_4xx + excluded.status_4xx,
				status_5xx = status_5xx + excluded.status_5xx,
				latency_total_ms = latency_total_ms + excluded.latency_total_ms,
				latency_max_ms = MAX(latency_max_ms, excluded.latency_max_ms),
				request_bytes = request_bytes + excluded.request_bytes,
				response_bytes = response_bytes + excluded.response_bytes`, archiveBefore.UTC()); err != nil {
			return nil, err
		}

		if _, err := tx.Exec(`UPDATE http_request_logs SET archived = 1 WHERE archived = 0 AND created_at < ?`, archiveBefore.UTC()); err != nil {
			return nil, err
		}
	}

	if detailRetentionDays > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(detailRetentionDays) * 24 * time.Hour)
		deleteResult, err := tx.Exec(`DELETE FROM http_request_logs WHERE archived = 1 AND created_at < ?`, cutoff)
		if err != nil {
			return nil, err
		}
		result.DeletedDetails, _ = deleteResult.RowsAffected()
	}

	if archiveRetentionDays > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(archiveRetentionDays) * 24 * time.Hour)
		hourlyResult, err := tx.Exec(`DELETE FROM http_request_hourly WHERE bucket_start < ?`, cutoff)
		if err != nil {
			return nil, err
		}
		result.DeletedHourly, _ = hourlyResult.RowsAffected()
		visitorResult, err := tx.Exec(`DELETE FROM http_daily_visitors WHERE day < strftime('%Y-%m-%d', ?)`, cutoff)
		if err != nil {
			return nil, err
		}
		result.DeletedVisitors, _ = visitorResult.RowsAffected()
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (db *DB) DeleteHTTPRequestDetails(before time.Time, scope string) (int64, error) {
	query := `DELETE FROM http_request_logs WHERE archived = 1 AND created_at < ?`
	switch scope {
	case "errors":
		query += ` AND status_code >= 400`
	case "success":
		query += ` AND status_code < 400`
	case "all", "":
	default:
		return 0, fmt.Errorf("invalid delete scope")
	}
	result, err := db.conn.Exec(query, before.UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) GetMonitoringOverview(since time.Time) (*MonitoringOverview, error) {
	overview := &MonitoringOverview{Timeline: make([]MonitoringTimelinePoint, 0)}
	combined := `
		WITH combined AS (
			SELECT bucket_start AS bucket, request_count, status_2xx, status_3xx, status_4xx, status_5xx,
				latency_total_ms, latency_max_ms, request_bytes, response_bytes
			FROM http_request_hourly WHERE bucket_start >= ?
			UNION ALL
			SELECT strftime('%Y-%m-%d %H:00:00', created_at), COUNT(*),
				SUM(CASE WHEN status_code BETWEEN 200 AND 299 THEN 1 ELSE 0 END),
				SUM(CASE WHEN status_code BETWEEN 300 AND 399 THEN 1 ELSE 0 END),
				SUM(CASE WHEN status_code BETWEEN 400 AND 499 THEN 1 ELSE 0 END),
				SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END),
				SUM(latency_ms), MAX(latency_ms), SUM(request_bytes), SUM(response_bytes)
			FROM http_request_logs WHERE archived = 0 AND created_at >= ?
			GROUP BY strftime('%Y-%m-%d %H:00:00', created_at)
		)`
	if err := db.conn.QueryRow(combined+`
		SELECT COALESCE(SUM(request_count), 0), COALESCE(SUM(status_2xx), 0),
			COALESCE(SUM(status_3xx), 0), COALESCE(SUM(status_4xx), 0), COALESCE(SUM(status_5xx), 0),
			CASE WHEN COALESCE(SUM(request_count), 0) = 0 THEN 0 ELSE CAST(SUM(latency_total_ms) AS REAL) / SUM(request_count) END,
			COALESCE(MAX(latency_max_ms), 0), COALESCE(SUM(request_bytes), 0), COALESCE(SUM(response_bytes), 0)
		FROM combined`, since.UTC(), since.UTC()).Scan(
		&overview.RequestCount, &overview.Status2xx, &overview.Status3xx, &overview.Status4xx,
		&overview.Status5xx, &overview.AverageMS, &overview.MaxLatencyMS,
		&overview.RequestBytes, &overview.ResponseBytes,
	); err != nil {
		return nil, err
	}

	unique, err := db.countMonitoringVisitors(since)
	if err != nil {
		return nil, err
	}
	overview.UniqueIPs = unique
	now := time.Now().UTC()
	overview.DAU, err = db.countMonitoringVisitors(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
	if err != nil {
		return nil, err
	}
	overview.WAU, err = db.countMonitoringVisitors(now.AddDate(0, 0, -6))
	if err != nil {
		return nil, err
	}
	overview.MAU, err = db.countMonitoringVisitors(now.AddDate(0, 0, -29))
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.Query(combined+`
		SELECT bucket, SUM(request_count), SUM(status_2xx), SUM(status_3xx), SUM(status_4xx), SUM(status_5xx),
			CASE WHEN SUM(request_count) = 0 THEN 0 ELSE SUM(latency_total_ms) / SUM(request_count) END,
			SUM(response_bytes)
		FROM combined GROUP BY bucket ORDER BY bucket`, since.UTC(), since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var point MonitoringTimelinePoint
		if err := rows.Scan(&point.Bucket, &point.Requests, &point.Status2xx, &point.Status3xx,
			&point.Status4xx, &point.Status5xx, &point.AverageMS, &point.ResponseSize); err != nil {
			return nil, err
		}
		overview.Timeline = append(overview.Timeline, point)
	}
	return overview, rows.Err()
}

func (db *DB) countMonitoringVisitors(since time.Time) (int64, error) {
	var count int64
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT visitor_id FROM http_daily_visitors WHERE day >= strftime('%Y-%m-%d', ?)
			UNION
			SELECT visitor_id FROM http_request_logs WHERE archived = 0 AND created_at >= ?
		)`, since.UTC(), since.UTC()).Scan(&count)
	return count, err
}

func (db *DB) GetMonitoringTopEndpoints(since time.Time, limit int) ([]MonitoringEndpointStat, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.conn.Query(`
		WITH combined AS (
			SELECT route, method, request_count,
				status_4xx + status_5xx AS errors, latency_total_ms, latency_max_ms, response_bytes
			FROM http_request_hourly WHERE bucket_start >= ?
			UNION ALL
			SELECT route, method, COUNT(*),
				SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), SUM(latency_ms), MAX(latency_ms), SUM(response_bytes)
			FROM http_request_logs WHERE archived = 0 AND created_at >= ? GROUP BY route, method
		)
		SELECT route, method, SUM(request_count), SUM(errors),
			CASE WHEN SUM(request_count) = 0 THEN 0 ELSE CAST(SUM(errors) AS REAL) * 100 / SUM(request_count) END,
			CASE WHEN SUM(request_count) = 0 THEN 0 ELSE CAST(SUM(latency_total_ms) AS REAL) / SUM(request_count) END,
			MAX(latency_max_ms), SUM(response_bytes)
		FROM combined GROUP BY route, method ORDER BY SUM(request_count) DESC LIMIT ?`, since.UTC(), since.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]MonitoringEndpointStat, 0)
	for rows.Next() {
		var item MonitoringEndpointStat
		if err := rows.Scan(&item.Route, &item.Method, &item.Requests, &item.Errors, &item.ErrorRate,
			&item.AverageMS, &item.MaxLatencyMS, &item.ResponseSize); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (db *DB) ListHTTPRequestLogs(since time.Time, statusGroup, keyword string, limit, offset int) ([]HTTPRequestLog, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where := ` WHERE created_at >= ?`
	args := []interface{}{since.UTC()}
	switch statusGroup {
	case "success":
		where += ` AND status_code < 400`
	case "errors":
		where += ` AND status_code >= 400`
	case "2xx":
		where += ` AND status_code BETWEEN 200 AND 299`
	case "3xx":
		where += ` AND status_code BETWEEN 300 AND 399`
	case "4xx":
		where += ` AND status_code BETWEEN 400 AND 499`
	case "5xx":
		where += ` AND status_code >= 500`
	case "all", "":
	default:
		return nil, 0, fmt.Errorf("invalid status group")
	}
	if strings.TrimSpace(keyword) != "" {
		where += ` AND (path LIKE ? OR route LIKE ? OR client_ip LIKE ?)`
		pattern := "%" + strings.TrimSpace(keyword) + "%"
		args = append(args, pattern, pattern, pattern)
	}
	var total int64
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM http_request_logs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := db.conn.Query(`
		SELECT id, method, path, route, status_code, latency_ms, client_ip, visitor_id, user_agent,
			request_bytes, response_bytes, error_message, archived, created_at
		FROM http_request_logs`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := make([]HTTPRequestLog, 0)
	for rows.Next() {
		var item HTTPRequestLog
		var archived int
		if err := rows.Scan(&item.ID, &item.Method, &item.Path, &item.Route, &item.StatusCode,
			&item.LatencyMS, &item.ClientIP, &item.VisitorID, &item.UserAgent, &item.RequestBytes,
			&item.ResponseBytes, &item.ErrorMessage, &archived, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		item.Archived = archived == 1
		result = append(result, item)
	}
	return result, total, rows.Err()
}

func parseMonitoringTime(value string) *time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		monitoringTimeLayout,
	} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func (db *DB) GetMonitoringStorageStats() (*MonitoringStorageStats, error) {
	stats := &MonitoringStorageStats{}
	var oldest, newest, lastArchive sql.NullString
	if err := db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN archived = 1 THEN 1 ELSE 0 END), 0), MIN(created_at), MAX(created_at)
		FROM http_request_logs`).Scan(&stats.DetailRows, &stats.ArchivedDetailRows, &oldest, &newest); err != nil {
		return nil, err
	}
	if err := db.conn.QueryRow(`SELECT COUNT(*), MAX(bucket_start) FROM http_request_hourly`).Scan(&stats.HourlyRows, &lastArchive); err != nil {
		return nil, err
	}
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM http_daily_visitors`).Scan(&stats.DailyVisitorRows); err != nil {
		return nil, err
	}
	var pageCount, pageSize int64
	if err := db.conn.QueryRow(`PRAGMA page_count`).Scan(&pageCount); err == nil {
		_ = db.conn.QueryRow(`PRAGMA page_size`).Scan(&pageSize)
		stats.DatabaseBytes = pageCount * pageSize
	}
	if oldest.Valid {
		stats.OldestDetailAt = parseMonitoringTime(oldest.String)
	}
	if newest.Valid {
		stats.NewestDetailAt = parseMonitoringTime(newest.String)
	}
	if lastArchive.Valid {
		stats.LastArchiveAt = parseMonitoringTime(lastArchive.String)
	}
	return stats, nil
}

type ProxySessionLog struct {
	ID          int64     `json:"id"`
	SessionType string    `json:"session_type"`
	Method      string    `json:"method"`
	Route       string    `json:"route"`
	Target      string    `json:"target"`
	ClientIP    string    `json:"client_ip"`
	UserAgent   string    `json:"user_agent"`
	StatusCode  int       `json:"status_code"`
	DurationMS  int64     `json:"duration_ms"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	FramesIn    int64     `json:"frames_in"`
	FramesOut   int64     `json:"frames_out"`
	CloseReason string    `json:"close_reason"`
	CreatedAt   time.Time `json:"created_at"`
}

func (db *DB) CreateProxySessionLog(item *ProxySessionLog) error {
	if item == nil {
		return fmt.Errorf("proxy session log is nil")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	result, err := db.conn.Exec(`
		INSERT INTO proxy_session_logs (
			session_type, method, route, target, client_ip, user_agent,
			status_code, duration_ms, bytes_in, bytes_out, frames_in, frames_out,
			close_reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.SessionType, item.Method, item.Route, item.Target, item.ClientIP, item.UserAgent,
		item.StatusCode, item.DurationMS, item.BytesIn, item.BytesOut, item.FramesIn, item.FramesOut,
		item.CloseReason, item.CreatedAt.UTC(),
	)
	if err != nil {
		return err
	}
	item.ID, _ = result.LastInsertId()
	return nil
}

type ProxySessionSummary struct {
	WSSessions     int64 `json:"ws_sessions"`
	WSFramesIn     int64 `json:"ws_frames_in"`
	WSFramesOut    int64 `json:"ws_frames_out"`
	WSBytesIn      int64 `json:"ws_bytes_in"`
	WSBytesOut     int64 `json:"ws_bytes_out"`
	TunnelSessions int64 `json:"tunnel_sessions"`
	TunnelBytesIn  int64 `json:"tunnel_bytes_in"`
	TunnelBytesOut int64 `json:"tunnel_bytes_out"`
}

func (db *DB) GetProxySessionSummary(since time.Time) (*ProxySessionSummary, error) {
	summary := &ProxySessionSummary{}
	err := db.conn.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN session_type = 'websocket' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'websocket' THEN frames_in ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'websocket' THEN frames_out ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'websocket' THEN bytes_in ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'websocket' THEN bytes_out ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'connect_tunnel' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'connect_tunnel' THEN bytes_in ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN session_type = 'connect_tunnel' THEN bytes_out ELSE 0 END), 0)
		FROM proxy_session_logs
		WHERE created_at >= ?`, since.UTC(),
	).Scan(
		&summary.WSSessions,
		&summary.WSFramesIn, &summary.WSFramesOut,
		&summary.WSBytesIn, &summary.WSBytesOut,
		&summary.TunnelSessions,
		&summary.TunnelBytesIn, &summary.TunnelBytesOut,
	)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func (db *DB) ListProxySessionLogs(sessionType string, limit, offset int) ([]ProxySessionLog, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where := ""
	args := []interface{}{}
	if sessionType = strings.TrimSpace(sessionType); sessionType != "" {
		where = ` WHERE session_type = ?`
		args = append(args, sessionType)
	}
	var total int64
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM proxy_session_logs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := db.conn.Query(`
		SELECT id, session_type, method, route, target, client_ip, user_agent,
			status_code, duration_ms, bytes_in, bytes_out, frames_in, frames_out,
			close_reason, created_at
		FROM proxy_session_logs`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := make([]ProxySessionLog, 0)
	for rows.Next() {
		var item ProxySessionLog
		if err := rows.Scan(&item.ID, &item.SessionType, &item.Method, &item.Route, &item.Target,
			&item.ClientIP, &item.UserAgent, &item.StatusCode, &item.DurationMS,
			&item.BytesIn, &item.BytesOut, &item.FramesIn, &item.FramesOut,
			&item.CloseReason, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, item)
	}
	return result, total, rows.Err()
}

func (db *DB) GetAIMonitoringSummary(since time.Time) (*AIMonitoringSummary, error) {
	summary := &AIMonitoringSummary{Models: make([]AIMonitoringModelStat, 0)}
	if err := db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(AVG(latency_ms), 0), COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(estimated_cost), 0)
		FROM ai_api_request_logs WHERE created_at >= ?`, since.UTC()).Scan(
		&summary.RequestCount, &summary.ErrorCount, &summary.AverageMS, &summary.InputTokens,
		&summary.OutputTokens, &summary.TotalTokens, &summary.TotalCost,
	); err != nil {
		return nil, err
	}
	rows, err := db.conn.Query(`
		SELECT model, provider, COUNT(*), SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END),
			COALESCE(SUM(total_tokens), 0), COALESCE(SUM(estimated_cost), 0), COALESCE(AVG(latency_ms), 0)
		FROM ai_api_request_logs WHERE created_at >= ?
		GROUP BY model, provider ORDER BY COUNT(*) DESC LIMIT 20`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item AIMonitoringModelStat
		if err := rows.Scan(&item.Model, &item.Provider, &item.Requests, &item.Errors,
			&item.TotalTokens, &item.TotalCost, &item.AverageMS); err != nil {
			return nil, err
		}
		summary.Models = append(summary.Models, item)
	}
	return summary, rows.Err()
}
