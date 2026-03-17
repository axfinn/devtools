package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// AutoDevTask represents an autodev execution task
type AutoDevTask struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Options     string     `json:"options"` // JSON: {publish,build,push}
	Status      string     `json:"status"`  // pending, running, completed, failed
	ExitCode    int        `json:"exit_code"`
	WorkDir     string     `json:"work_dir"`
	PID         int        `json:"pid"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// AutoDevOptions task execution options
type AutoDevOptions struct {
	Publish bool `json:"publish"`
	Build   bool `json:"build"`
	Push    bool `json:"push"`
}

// InitAutoDevTasks initializes the autodev_tasks table
func (db *DB) InitAutoDevTasks() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS autodev_tasks (
			id TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			options TEXT DEFAULT '{}',
			status TEXT DEFAULT 'pending',
			exit_code INTEGER DEFAULT 0,
			work_dir TEXT,
			pid INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_autodev_tasks_status ON autodev_tasks(status);
		CREATE INDEX IF NOT EXISTS idx_autodev_tasks_created_at ON autodev_tasks(created_at);
	`)
	return err
}

// CreateAutoDevTask creates a new task record
func (db *DB) CreateAutoDevTask(description, options, workDir string) (*AutoDevTask, error) {
	id := generateID(8)
	_, err := db.conn.Exec(
		`INSERT INTO autodev_tasks (id, description, options, work_dir, status) VALUES (?, ?, ?, ?, 'pending')`,
		id, description, options, workDir,
	)
	if err != nil {
		return nil, err
	}
	return db.GetAutoDevTask(id)
}

// GetAutoDevTask retrieves a task by ID
func (db *DB) GetAutoDevTask(id string) (*AutoDevTask, error) {
	var task AutoDevTask
	var startedAt, completedAt sql.NullTime
	err := db.conn.QueryRow(
		`SELECT id, description, options, status, exit_code, work_dir, pid, created_at, started_at, completed_at
		 FROM autodev_tasks WHERE id = ?`, id,
	).Scan(
		&task.ID, &task.Description, &task.Options, &task.Status,
		&task.ExitCode, &task.WorkDir, &task.PID, &task.CreatedAt,
		&startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return &task, nil
}

// ListAutoDevTasks lists all tasks ordered by creation time desc
func (db *DB) ListAutoDevTasks() ([]*AutoDevTask, error) {
	rows, err := db.conn.Query(
		`SELECT id, description, options, status, exit_code, work_dir, pid, created_at, started_at, completed_at
		 FROM autodev_tasks ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*AutoDevTask
	for rows.Next() {
		var task AutoDevTask
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(
			&task.ID, &task.Description, &task.Options, &task.Status,
			&task.ExitCode, &task.WorkDir, &task.PID, &task.CreatedAt,
			&startedAt, &completedAt,
		); err != nil {
			continue
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

// UpdateAutoDevTaskStatus updates task status and related fields
func (db *DB) UpdateAutoDevTaskStatus(id, status string, exitCode, pid int) error {
	now := time.Now()
	switch status {
	case "running":
		_, err := db.conn.Exec(
			`UPDATE autodev_tasks SET status = ?, pid = ?, started_at = ? WHERE id = ?`,
			status, pid, now, id,
		)
		return err
	case "completed", "failed":
		_, err := db.conn.Exec(
			`UPDATE autodev_tasks SET status = ?, exit_code = ?, completed_at = ? WHERE id = ?`,
			status, exitCode, now, id,
		)
		return err
	default:
		_, err := db.conn.Exec(
			`UPDATE autodev_tasks SET status = ? WHERE id = ?`,
			status, id,
		)
		return err
	}
}

// DeleteAutoDevTask deletes a task record
func (db *DB) DeleteAutoDevTask(id string) error {
	_, err := db.conn.Exec(`DELETE FROM autodev_tasks WHERE id = ?`, id)
	return err
}

// MarshalAutoDevOptions encodes options to JSON
func MarshalAutoDevOptions(opts AutoDevOptions) string {
	b, _ := json.Marshal(opts)
	return string(b)
}
