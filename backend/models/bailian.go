package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func init() {
	RegisterInit("百炼(bailian_tasks)", (*DB).InitBailian)
}

type BailianImageTask struct {
	ID             string     `json:"id"`
	Model          string     `json:"model"`
	TaskType       string     `json:"task_type"`
	Source         string     `json:"source"`
	ClientName     string     `json:"client_name"`
	Prompt         string     `json:"prompt"`
	NegativePrompt string     `json:"negative_prompt"`
	InputImages    string     `json:"input_images"`
	ParamsJSON     string     `json:"params_json"`
	Status         string     `json:"status"`
	VendorStatus   string     `json:"vendor_status"`
	ExternalTaskID string     `json:"external_task_id"`
	ResultJSON     string     `json:"result_json"`
	ErrorMessage   string     `json:"error_message"`
	RequestBody    string     `json:"request_body"`
	ResponseBody   string     `json:"response_body"`
	QuotaTotal     int        `json:"quota_total"`
	QuotaUsed      int        `json:"quota_used"`
	QuotaExpiresAt *time.Time `json:"quota_expires_at"`
	QuotaCounted   bool       `json:"quota_counted"`
	CreatorIP      string     `json:"creator_ip"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

type BailianTaskEvent struct {
	ID        int64     `json:"id"`
	TaskID    string    `json:"task_id"`
	Stage     string    `json:"stage"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) InitBailian() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS bailian_image_tasks (
			id TEXT PRIMARY KEY,
			model TEXT NOT NULL,
			task_type TEXT NOT NULL,
			source TEXT DEFAULT 'debug',
			client_name TEXT DEFAULT '',
			prompt TEXT DEFAULT '',
			negative_prompt TEXT DEFAULT '',
			input_images TEXT DEFAULT '[]',
			params_json TEXT DEFAULT '{}',
			status TEXT DEFAULT 'created',
			vendor_status TEXT DEFAULT '',
			external_task_id TEXT DEFAULT '',
			result_json TEXT DEFAULT '{}',
			error_message TEXT DEFAULT '',
			request_body TEXT DEFAULT '',
			response_body TEXT DEFAULT '',
			quota_total INTEGER DEFAULT 0,
			quota_used INTEGER DEFAULT 0,
			quota_expires_at DATETIME,
			quota_counted INTEGER DEFAULT 1,
			creator_ip TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS bailian_task_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			stage TEXT NOT NULL,
			status TEXT DEFAULT '',
			message TEXT DEFAULT '',
			payload TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES bailian_image_tasks(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_bailian_tasks_model ON bailian_image_tasks(model)`,
		`CREATE INDEX IF NOT EXISTS idx_bailian_tasks_status ON bailian_image_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bailian_tasks_created_at ON bailian_image_tasks(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_bailian_tasks_external_task_id ON bailian_image_tasks(external_task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bailian_events_task_id ON bailian_task_events(task_id, created_at DESC)`,
	}
	for _, idx := range indexes {
		if _, err := db.conn.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) CreateBailianTask(task *BailianImageTask) error {
	if task == nil {
		return errors.New("task is nil")
	}
	task.ID = generateID(8)
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.InputImages == "" {
		task.InputImages = "[]"
	}
	if task.ParamsJSON == "" {
		task.ParamsJSON = "{}"
	}
	if task.ResultJSON == "" {
		task.ResultJSON = "{}"
	}
	quotaCounted := 0
	if task.QuotaCounted {
		quotaCounted = 1
	}

	_, err := db.conn.Exec(`
		INSERT INTO bailian_image_tasks (
			id, model, task_type, source, client_name, prompt, negative_prompt, input_images,
			params_json, status, vendor_status, external_task_id, result_json, error_message,
			request_body, response_body, quota_total, quota_used, quota_expires_at, quota_counted,
			creator_ip, created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Model, task.TaskType, task.Source, task.ClientName, task.Prompt,
		task.NegativePrompt, task.InputImages, task.ParamsJSON, task.Status, task.VendorStatus,
		task.ExternalTaskID, task.ResultJSON, task.ErrorMessage, task.RequestBody, task.ResponseBody,
		task.QuotaTotal, task.QuotaUsed, task.QuotaExpiresAt, quotaCounted, task.CreatorIP,
		task.CreatedAt, task.UpdatedAt, task.CompletedAt,
	)
	return err
}

func (db *DB) UpdateBailianTask(task *BailianImageTask) error {
	if task == nil {
		return errors.New("task is nil")
	}
	task.UpdatedAt = time.Now()
	quotaCounted := 0
	if task.QuotaCounted {
		quotaCounted = 1
	}

	_, err := db.conn.Exec(`
		UPDATE bailian_image_tasks
		SET model = ?, task_type = ?, source = ?, client_name = ?, prompt = ?, negative_prompt = ?,
			input_images = ?, params_json = ?, status = ?, vendor_status = ?, external_task_id = ?,
			result_json = ?, error_message = ?, request_body = ?, response_body = ?, quota_total = ?,
			quota_used = ?, quota_expires_at = ?, quota_counted = ?, creator_ip = ?, updated_at = ?,
			completed_at = ?
		WHERE id = ?`,
		task.Model, task.TaskType, task.Source, task.ClientName, task.Prompt, task.NegativePrompt,
		task.InputImages, task.ParamsJSON, task.Status, task.VendorStatus, task.ExternalTaskID,
		task.ResultJSON, task.ErrorMessage, task.RequestBody, task.ResponseBody, task.QuotaTotal,
		task.QuotaUsed, task.QuotaExpiresAt, quotaCounted, task.CreatorIP, task.UpdatedAt,
		task.CompletedAt, task.ID,
	)
	return err
}

func (db *DB) GetBailianTask(id string) (*BailianImageTask, error) {
	task := &BailianImageTask{}
	var quotaExpiresAt sql.NullTime
	var completedAt sql.NullTime
	var quotaCounted int
	err := db.conn.QueryRow(`
		SELECT id, model, task_type, source, client_name, prompt, negative_prompt, input_images,
			params_json, status, vendor_status, external_task_id, result_json, error_message,
			request_body, response_body, quota_total, quota_used, quota_expires_at, quota_counted,
			creator_ip, created_at, updated_at, completed_at
		FROM bailian_image_tasks
		WHERE id = ?`, id).Scan(
		&task.ID, &task.Model, &task.TaskType, &task.Source, &task.ClientName, &task.Prompt,
		&task.NegativePrompt, &task.InputImages, &task.ParamsJSON, &task.Status, &task.VendorStatus,
		&task.ExternalTaskID, &task.ResultJSON, &task.ErrorMessage, &task.RequestBody, &task.ResponseBody,
		&task.QuotaTotal, &task.QuotaUsed, &quotaExpiresAt, &quotaCounted, &task.CreatorIP,
		&task.CreatedAt, &task.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if quotaExpiresAt.Valid {
		task.QuotaExpiresAt = &quotaExpiresAt.Time
	}
	task.QuotaCounted = quotaCounted == 1
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return task, nil
}

func (db *DB) ListBailianTasks(limit, offset int, model, status string) ([]*BailianImageTask, error) {
	query := `
		SELECT id, model, task_type, source, client_name, prompt, negative_prompt, input_images,
			params_json, status, vendor_status, external_task_id, result_json, error_message,
			request_body, response_body, quota_total, quota_used, quota_expires_at, quota_counted,
			creator_ip, created_at, updated_at, completed_at
		FROM bailian_image_tasks
		WHERE 1 = 1`
	var args []interface{}
	if model != "" {
		query += ` AND model = ?`
		args = append(args, model)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*BailianImageTask, 0)
	for rows.Next() {
		task := &BailianImageTask{}
		var quotaExpiresAt sql.NullTime
		var completedAt sql.NullTime
		var quotaCounted int
		if err := rows.Scan(
			&task.ID, &task.Model, &task.TaskType, &task.Source, &task.ClientName, &task.Prompt,
			&task.NegativePrompt, &task.InputImages, &task.ParamsJSON, &task.Status, &task.VendorStatus,
			&task.ExternalTaskID, &task.ResultJSON, &task.ErrorMessage, &task.RequestBody, &task.ResponseBody,
			&task.QuotaTotal, &task.QuotaUsed, &quotaExpiresAt, &quotaCounted, &task.CreatorIP,
			&task.CreatedAt, &task.UpdatedAt, &completedAt,
		); err != nil {
			continue
		}
		if quotaExpiresAt.Valid {
			task.QuotaExpiresAt = &quotaExpiresAt.Time
		}
		task.QuotaCounted = quotaCounted == 1
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (db *DB) CountBailianTasks(model, status string) (int, error) {
	query := `SELECT COUNT(*) FROM bailian_image_tasks WHERE 1 = 1`
	args := make([]interface{}, 0, 2)
	if model != "" {
		query += ` AND model = ?`
		args = append(args, model)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	var count int
	err := db.conn.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (db *DB) ListBailianTasksByClientPrefix(limit, offset int, clientPrefix, model, status string) ([]*BailianImageTask, error) {
	query := `
		SELECT id, model, task_type, source, client_name, prompt, negative_prompt, input_images,
			params_json, status, vendor_status, external_task_id, result_json, error_message,
			request_body, response_body, quota_total, quota_used, quota_expires_at, quota_counted,
			creator_ip, created_at, updated_at, completed_at
		FROM bailian_image_tasks
		WHERE client_name LIKE ?`
	args := []interface{}{clientPrefix + "%"}
	if model != "" {
		query += ` AND model = ?`
		args = append(args, model)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*BailianImageTask, 0)
	for rows.Next() {
		task := &BailianImageTask{}
		var quotaExpiresAt sql.NullTime
		var completedAt sql.NullTime
		var quotaCounted int
		if err := rows.Scan(
			&task.ID, &task.Model, &task.TaskType, &task.Source, &task.ClientName, &task.Prompt,
			&task.NegativePrompt, &task.InputImages, &task.ParamsJSON, &task.Status, &task.VendorStatus,
			&task.ExternalTaskID, &task.ResultJSON, &task.ErrorMessage, &task.RequestBody, &task.ResponseBody,
			&task.QuotaTotal, &task.QuotaUsed, &quotaExpiresAt, &quotaCounted, &task.CreatorIP,
			&task.CreatedAt, &task.UpdatedAt, &completedAt,
		); err != nil {
			continue
		}
		if quotaExpiresAt.Valid {
			task.QuotaExpiresAt = &quotaExpiresAt.Time
		}
		task.QuotaCounted = quotaCounted == 1
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (db *DB) CountBailianQuotaUsage(model string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM bailian_image_tasks
		WHERE model = ? AND quota_counted = 1`,
		model,
	).Scan(&count)
	return count, err
}

func (db *DB) CreateBailianTaskEvent(event *BailianTaskEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}
	if event.Payload == "" {
		event.Payload = "{}"
	}
	event.CreatedAt = time.Now()
	result, err := db.conn.Exec(`
		INSERT INTO bailian_task_events (task_id, stage, status, message, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		event.TaskID, event.Stage, event.Status, event.Message, event.Payload, event.CreatedAt,
	)
	if err != nil {
		return err
	}
	event.ID, _ = result.LastInsertId()
	return nil
}

func (db *DB) ListBailianTaskEvents(taskID string) ([]*BailianTaskEvent, error) {
	rows, err := db.conn.Query(`
		SELECT id, task_id, stage, status, message, payload, created_at
		FROM bailian_task_events
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*BailianTaskEvent, 0)
	for rows.Next() {
		event := &BailianTaskEvent{}
		if err := rows.Scan(&event.ID, &event.TaskID, &event.Stage, &event.Status, &event.Message, &event.Payload, &event.CreatedAt); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func (db *DB) CleanOldBailianTasks(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	result, err := db.conn.Exec(`
		DELETE FROM bailian_image_tasks
		WHERE created_at < ? AND status IN ('succeeded', 'failed', 'canceled', 'expired')`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func MustJSONString(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
