package models

import (
	"database/sql"
	"strings"
	"time"
)

type PlannerProfile struct {
	ID            string     `json:"id"`
	PasswordIndex string     `json:"-"`
	CreatorKey    string     `json:"creator_key"`
	Name          string     `json:"name"`
	NotifyEmail   string     `json:"notify_email"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PlannerTask struct {
	ID             string     `json:"id"`
	ProfileID      string     `json:"profile_id"`
	Kind           string     `json:"kind"`
	Title          string     `json:"title"`
	Detail         string     `json:"detail"`
	Notes          string     `json:"notes"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	PlannedFor     string     `json:"planned_for"`
	RemindAt       *time.Time `json:"remind_at"`
	NotifyEmail    string     `json:"notify_email"`
	LastNotifiedAt *time.Time `json:"last_notified_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PlannerTaskComment struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	ProfileID string    `json:"profile_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type PlannerProfileSummary struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	NotifyEmail string     `json:"notify_email"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	TaskCount   int        `json:"task_count"`
	OpenCount   int        `json:"open_count"`
}

func (db *DB) InitPlanner() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS planner_profiles (
			id TEXT PRIMARY KEY,
			password_index TEXT NOT NULL UNIQUE,
			creator_key TEXT NOT NULL,
			name TEXT NOT NULL,
			notify_email TEXT DEFAULT '',
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS planner_tasks (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			detail TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
			planned_for TEXT NOT NULL,
			remind_at DATETIME,
			notify_email TEXT DEFAULT '',
			last_notified_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES planner_profiles(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS planner_task_comments (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			profile_id TEXT NOT NULL,
			author TEXT NOT NULL DEFAULT '用户',
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES planner_tasks(id) ON DELETE CASCADE,
			FOREIGN KEY (profile_id) REFERENCES planner_profiles(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_planner_profiles_expires_at ON planner_profiles(expires_at);
		CREATE INDEX IF NOT EXISTS idx_planner_tasks_profile_kind_status ON planner_tasks(profile_id, kind, status, planned_for);
		CREATE INDEX IF NOT EXISTS idx_planner_tasks_remind_at ON planner_tasks(status, remind_at, last_notified_at);
		CREATE INDEX IF NOT EXISTS idx_planner_comments_task_created_at ON planner_task_comments(task_id, created_at);
	`)
	if err != nil {
		return err
	}

	db.conn.Exec("ALTER TABLE planner_profiles ADD COLUMN notify_email TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE planner_tasks ADD COLUMN notes TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE planner_tasks ADD COLUMN notify_email TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE planner_tasks ADD COLUMN last_notified_at DATETIME")
	db.conn.Exec("ALTER TABLE planner_tasks ADD COLUMN completed_at DATETIME")

	return nil
}

func (db *DB) CreatePlannerProfile(profile *PlannerProfile, expiresAt *time.Time) error {
	profile.ID = generateID(8)
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	profile.ExpiresAt = expiresAt

	_, err := db.conn.Exec(`
		INSERT INTO planner_profiles (id, password_index, creator_key, name, notify_email, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.PasswordIndex, profile.CreatorKey, profile.Name, profile.NotifyEmail, profile.ExpiresAt, profile.CreatedAt, profile.UpdatedAt)
	return err
}

func (db *DB) GetPlannerProfile(id string) (*PlannerProfile, error) {
	row := db.conn.QueryRow(`
		SELECT id, password_index, creator_key, name, notify_email, expires_at, created_at, updated_at
		FROM planner_profiles
		WHERE id = ?
	`, id)
	return scanPlannerProfile(row)
}

func (db *DB) GetPlannerProfileByPasswordIndex(passwordIndex string) (*PlannerProfile, error) {
	row := db.conn.QueryRow(`
		SELECT id, password_index, creator_key, name, notify_email, expires_at, created_at, updated_at
		FROM planner_profiles
		WHERE password_index = ?
	`, passwordIndex)
	return scanPlannerProfile(row)
}

func scanPlannerProfile(row *sql.Row) (*PlannerProfile, error) {
	profile := &PlannerProfile{}
	var expiresAt sql.NullTime
	if err := row.Scan(
		&profile.ID,
		&profile.PasswordIndex,
		&profile.CreatorKey,
		&profile.Name,
		&profile.NotifyEmail,
		&expiresAt,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}
	return profile, nil
}

func (db *DB) UpdatePlannerProfile(profile *PlannerProfile) error {
	profile.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE planner_profiles
		SET name = ?, notify_email = ?, expires_at = ?, updated_at = ?
		WHERE id = ?
	`, profile.Name, profile.NotifyEmail, profile.ExpiresAt, profile.UpdatedAt, profile.ID)
	return err
}

func (db *DB) DeletePlannerProfile(id string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM planner_task_comments WHERE profile_id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`DELETE FROM planner_tasks WHERE profile_id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`DELETE FROM planner_profiles WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (db *DB) PlannerProfileExists(id string) (bool, error) {
	var count int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM planner_profiles WHERE id = ?`, id).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) ListPlannerTasksByProfile(profileID string) ([]*PlannerTask, error) {
	return db.ListPlannerTasks(profileID, "", "")
}

func (db *DB) DeletePlannerTask(id string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM planner_task_comments WHERE task_id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err = tx.Exec(`DELETE FROM planner_tasks WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (db *DB) CountPlannerProfiles() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM planner_profiles`).Scan(&count)
	return count, err
}

func (db *DB) ListPlannerProfiles(limit, offset int) ([]*PlannerProfileSummary, error) {
	rows, err := db.conn.Query(`
		SELECT
			p.id,
			p.name,
			p.notify_email,
			p.expires_at,
			p.created_at,
			p.updated_at,
			COUNT(t.id) AS task_count,
			SUM(CASE WHEN t.status = 'open' THEN 1 ELSE 0 END) AS open_count
		FROM planner_profiles p
		LEFT JOIN planner_tasks t ON t.profile_id = p.id
		GROUP BY p.id
		ORDER BY p.updated_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerProfileSummary, 0)
	for rows.Next() {
		item := &PlannerProfileSummary{}
		var expiresAt sql.NullTime
		var openCount sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.NotifyEmail,
			&expiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.TaskCount,
			&openCount,
		); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.Time
		}
		if openCount.Valid {
			item.OpenCount = int(openCount.Int64)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (db *DB) CreatePlannerTask(task *PlannerTask) error {
	task.ID = generateID(8)
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	_, err := db.conn.Exec(`
		INSERT INTO planner_tasks (
			id, profile_id, kind, title, detail, notes, status, priority, planned_for,
			remind_at, notify_email, last_notified_at, completed_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.ProfileID, task.Kind, task.Title, task.Detail, task.Notes, task.Status,
		task.Priority, task.PlannedFor, task.RemindAt, task.NotifyEmail, task.LastNotifiedAt,
		task.CompletedAt, task.CreatedAt, task.UpdatedAt)
	return err
}

func (db *DB) GetPlannerTask(id string) (*PlannerTask, error) {
	row := db.conn.QueryRow(`
		SELECT id, profile_id, kind, title, detail, notes, status, priority, planned_for,
		       remind_at, notify_email, last_notified_at, completed_at, created_at, updated_at
		FROM planner_tasks
		WHERE id = ?
	`, id)
	return scanPlannerTask(row)
}

func scanPlannerTask(row *sql.Row) (*PlannerTask, error) {
	task := &PlannerTask{}
	var remindAt sql.NullTime
	var lastNotifiedAt sql.NullTime
	var completedAt sql.NullTime
	if err := row.Scan(
		&task.ID,
		&task.ProfileID,
		&task.Kind,
		&task.Title,
		&task.Detail,
		&task.Notes,
		&task.Status,
		&task.Priority,
		&task.PlannedFor,
		&remindAt,
		&task.NotifyEmail,
		&lastNotifiedAt,
		&completedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if remindAt.Valid {
		task.RemindAt = &remindAt.Time
	}
	if lastNotifiedAt.Valid {
		task.LastNotifiedAt = &lastNotifiedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return task, nil
}

func (db *DB) ListPlannerTasks(profileID, kind, status string) ([]*PlannerTask, error) {
	query := `
		SELECT id, profile_id, kind, title, detail, notes, status, priority, planned_for,
		       remind_at, notify_email, last_notified_at, completed_at, created_at, updated_at
		FROM planner_tasks
		WHERE profile_id = ?
	`
	args := []interface{}{profileID}
	if kind != "" {
		query += " AND kind = ?"
		args = append(args, kind)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY planned_for ASC, created_at ASC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerTask, 0)
	for rows.Next() {
		task := &PlannerTask{}
		var remindAt sql.NullTime
		var lastNotifiedAt sql.NullTime
		var completedAt sql.NullTime
		if err := rows.Scan(
			&task.ID,
			&task.ProfileID,
			&task.Kind,
			&task.Title,
			&task.Detail,
			&task.Notes,
			&task.Status,
			&task.Priority,
			&task.PlannedFor,
			&remindAt,
			&task.NotifyEmail,
			&lastNotifiedAt,
			&completedAt,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if remindAt.Valid {
			task.RemindAt = &remindAt.Time
		}
		if lastNotifiedAt.Valid {
			task.LastNotifiedAt = &lastNotifiedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		result = append(result, task)
	}
	return result, rows.Err()
}

func (db *DB) UpdatePlannerTask(task *PlannerTask) error {
	task.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE planner_tasks
		SET kind = ?, title = ?, detail = ?, notes = ?, status = ?, priority = ?,
		    planned_for = ?, remind_at = ?, notify_email = ?, last_notified_at = ?,
		    completed_at = ?, updated_at = ?
		WHERE id = ?
	`, task.Kind, task.Title, task.Detail, task.Notes, task.Status, task.Priority,
		task.PlannedFor, task.RemindAt, task.NotifyEmail, task.LastNotifiedAt, task.CompletedAt,
		task.UpdatedAt, task.ID)
	return err
}

func (db *DB) CreatePlannerTaskComment(comment *PlannerTaskComment) error {
	comment.ID = generateID(8)
	comment.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO planner_task_comments (id, task_id, profile_id, author, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, comment.ID, comment.TaskID, comment.ProfileID, comment.Author, comment.Content, comment.CreatedAt)
	return err
}

func (db *DB) ListPlannerTaskComments(taskID string) ([]*PlannerTaskComment, error) {
	rows, err := db.conn.Query(`
		SELECT id, task_id, profile_id, author, content, created_at
		FROM planner_task_comments
		WHERE task_id = ?
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerTaskComment, 0)
	for rows.Next() {
		item := &PlannerTaskComment{}
		if err := rows.Scan(&item.ID, &item.TaskID, &item.ProfileID, &item.Author, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (db *DB) CountPlannerTasks(profileID string) (map[string]int, error) {
	rows, err := db.conn.Query(`
		SELECT kind || ':' || status AS bucket, COUNT(*)
		FROM planner_tasks
		WHERE profile_id = ?
		GROUP BY kind, status
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		stats[key] = count
	}
	return stats, rows.Err()
}

func (db *DB) ListPlannerTasksDueForReminder(now time.Time, limit int) ([]*PlannerTask, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(`
		SELECT id, profile_id, kind, title, detail, notes, status, priority, planned_for,
		       remind_at, notify_email, last_notified_at, completed_at, created_at, updated_at
		FROM planner_tasks
		WHERE status IN ('open', 'in_progress')
		  AND remind_at IS NOT NULL
		  AND remind_at <= ?
		  AND (last_notified_at IS NULL OR last_notified_at < remind_at)
		ORDER BY remind_at ASC
		LIMIT ?
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerTask, 0)
	for rows.Next() {
		task := &PlannerTask{}
		var remindAt sql.NullTime
		var lastNotifiedAt sql.NullTime
		var completedAt sql.NullTime
		if err := rows.Scan(
			&task.ID,
			&task.ProfileID,
			&task.Kind,
			&task.Title,
			&task.Detail,
			&task.Notes,
			&task.Status,
			&task.Priority,
			&task.PlannedFor,
			&remindAt,
			&task.NotifyEmail,
			&lastNotifiedAt,
			&completedAt,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if remindAt.Valid {
			task.RemindAt = &remindAt.Time
		}
		if lastNotifiedAt.Valid {
			task.LastNotifiedAt = &lastNotifiedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		result = append(result, task)
	}
	return result, rows.Err()
}

func (db *DB) MarkPlannerTaskReminderSent(taskID string, sentAt time.Time) error {
	_, err := db.conn.Exec(`
		UPDATE planner_tasks
		SET last_notified_at = ?, updated_at = ?
		WHERE id = ?
	`, sentAt, sentAt, taskID)
	return err
}

func (db *DB) CleanExpiredPlannerProfiles() (int64, error) {
	rows, err := db.conn.Query(`SELECT id FROM planner_profiles WHERE expires_at IS NOT NULL AND expires_at < ?`, time.Now())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var deleted int64
	for _, id := range ids {
		if err := db.DeletePlannerProfile(id); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func (db *DB) CountPlannerProfilesByNotifyEmail(email string) (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM planner_profiles WHERE notify_email = ?`, email).Scan(&count)
	return count, err
}

func (db *DB) SearchPlannerProfiles(keyword string, limit int) ([]*PlannerProfileSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	like := "%" + strings.TrimSpace(keyword) + "%"
	rows, err := db.conn.Query(`
		SELECT
			p.id,
			p.name,
			p.notify_email,
			p.expires_at,
			p.created_at,
			p.updated_at,
			COUNT(t.id) AS task_count,
			SUM(CASE WHEN t.status = 'open' THEN 1 ELSE 0 END) AS open_count
		FROM planner_profiles p
		LEFT JOIN planner_tasks t ON t.profile_id = p.id
		WHERE p.id LIKE ? OR p.name LIKE ? OR p.notify_email LIKE ?
		GROUP BY p.id
		ORDER BY p.updated_at DESC
		LIMIT ?
	`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerProfileSummary, 0)
	for rows.Next() {
		item := &PlannerProfileSummary{}
		var expiresAt sql.NullTime
		var openCount sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.NotifyEmail,
			&expiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.TaskCount,
			&openCount,
		); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.Time
		}
		if openCount.Valid {
			item.OpenCount = int(openCount.Int64)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
