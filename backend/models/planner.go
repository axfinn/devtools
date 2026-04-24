package models

import (
	"database/sql"
	"strings"
	"time"
)

func init() {
	RegisterInit("事项规划(planner_profiles)", (*DB).InitPlanner)
}

const (
	PlannerEntryTask  = "task"
	PlannerEntryEvent = "event"

	PlannerBucketInbox   = "inbox"
	PlannerBucketPlanned = "planned"
	PlannerBucketSomeday = "someday"
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
	ID                 string     `json:"id"`
	ProfileID          string     `json:"profile_id"`
	Kind               string     `json:"kind"`
	EntryType          string     `json:"entry_type"`
	Bucket             string     `json:"bucket"`
	Title              string     `json:"title"`
	Detail             string     `json:"detail"`
	Notes              string     `json:"notes"`
	Status             string     `json:"status"`
	Priority           string     `json:"priority"`
	PlannedFor         string     `json:"planned_for"`
	OriginalPlannedFor string     `json:"original_planned_for"`
	RolloverCount      int        `json:"rollover_count"`
	LastPostponeReason string     `json:"last_postpone_reason"`
	LastPostponedAt    *time.Time `json:"last_postponed_at"`
	RemindAt           *time.Time `json:"remind_at"`
	RepeatType         string     `json:"repeat_type"`
	RepeatInterval     int        `json:"repeat_interval"`
	RepeatUntil        *time.Time `json:"repeat_until"`
	NotifyEmail        string     `json:"notify_email"`
	LastNotifiedAt     *time.Time `json:"last_notified_at"`
	CancelReason       string     `json:"cancel_reason"`
	CompletedAt        *time.Time `json:"completed_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type PlannerTaskComment struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	ProfileID string    `json:"profile_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type PlannerTaskActivity struct {
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	ProfileID    string    `json:"profile_id"`
	ActivityType string    `json:"activity_type"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

type PlannerTaskCommentSummary struct {
	TaskID        string
	CommentCount  int
	LastContent   string
	LastCreatedAt *time.Time
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
			entry_type TEXT NOT NULL DEFAULT 'task',
			bucket TEXT NOT NULL DEFAULT 'planned',
			title TEXT NOT NULL,
			detail TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
				planned_for TEXT NOT NULL,
				original_planned_for TEXT DEFAULT '',
				rollover_count INTEGER DEFAULT 0,
				last_postpone_reason TEXT DEFAULT '',
				last_postponed_at DATETIME,
				remind_at DATETIME,
				repeat_type TEXT NOT NULL DEFAULT 'none',
				repeat_interval INTEGER NOT NULL DEFAULT 1,
				repeat_until DATETIME,
				notify_email TEXT DEFAULT '',
				last_notified_at DATETIME,
				cancel_reason TEXT DEFAULT '',
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

		CREATE TABLE IF NOT EXISTS planner_task_activities (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			profile_id TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (task_id) REFERENCES planner_tasks(id) ON DELETE CASCADE,
			FOREIGN KEY (profile_id) REFERENCES planner_profiles(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	migrations := []string{
		"ALTER TABLE planner_profiles ADD COLUMN notify_email TEXT DEFAULT ''",
		"ALTER TABLE planner_tasks ADD COLUMN entry_type TEXT NOT NULL DEFAULT 'task'",
		"ALTER TABLE planner_tasks ADD COLUMN bucket TEXT NOT NULL DEFAULT 'planned'",
		"ALTER TABLE planner_tasks ADD COLUMN notes TEXT DEFAULT ''",
		"ALTER TABLE planner_tasks ADD COLUMN notify_email TEXT DEFAULT ''",
		"ALTER TABLE planner_tasks ADD COLUMN last_notified_at DATETIME",
		"ALTER TABLE planner_tasks ADD COLUMN completed_at DATETIME",
		"ALTER TABLE planner_tasks ADD COLUMN original_planned_for TEXT DEFAULT ''",
		"ALTER TABLE planner_tasks ADD COLUMN rollover_count INTEGER DEFAULT 0",
		"ALTER TABLE planner_tasks ADD COLUMN last_postpone_reason TEXT DEFAULT ''",
		"ALTER TABLE planner_tasks ADD COLUMN last_postponed_at DATETIME",
		"ALTER TABLE planner_tasks ADD COLUMN repeat_type TEXT NOT NULL DEFAULT 'none'",
		"ALTER TABLE planner_tasks ADD COLUMN repeat_interval INTEGER NOT NULL DEFAULT 1",
		"ALTER TABLE planner_tasks ADD COLUMN repeat_until DATETIME",
		"ALTER TABLE planner_tasks ADD COLUMN cancel_reason TEXT DEFAULT ''",
	}
	for _, stmt := range migrations {
		if _, err := db.conn.Exec(stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_planner_profiles_expires_at ON planner_profiles(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_planner_tasks_profile_kind_status ON planner_tasks(profile_id, kind, status, planned_for)",
		"CREATE INDEX IF NOT EXISTS idx_planner_tasks_bucket_entry_type ON planner_tasks(profile_id, bucket, entry_type, status)",
		"CREATE INDEX IF NOT EXISTS idx_planner_tasks_remind_at ON planner_tasks(status, remind_at, last_notified_at)",
		"CREATE INDEX IF NOT EXISTS idx_planner_comments_task_created_at ON planner_task_comments(task_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_planner_activities_task_created_at ON planner_task_activities(task_id, created_at)",
	}
	for _, stmt := range indexes {
		if _, err := db.conn.Exec(stmt); err != nil {
			return err
		}
	}

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
	if _, err = tx.Exec(`DELETE FROM planner_task_activities WHERE profile_id = ?`, id); err != nil {
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
	return db.ListPlannerTasksAdvanced(profileID, "", "", "", "")
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
	if _, err = tx.Exec(`DELETE FROM planner_task_activities WHERE task_id = ?`, id); err != nil {
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
			SUM(CASE WHEN t.status IN ('open', 'in_progress') THEN 1 ELSE 0 END) AS open_count
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
			id, profile_id, kind, entry_type, bucket, title, detail, notes, status, priority,
			planned_for, original_planned_for, rollover_count, last_postpone_reason, last_postponed_at,
			remind_at, repeat_type, repeat_interval, repeat_until, notify_email, last_notified_at,
			cancel_reason, completed_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.ProfileID, task.Kind, task.EntryType, task.Bucket, task.Title, task.Detail, task.Notes, task.Status,
		task.Priority, task.PlannedFor, task.OriginalPlannedFor, task.RolloverCount, task.LastPostponeReason,
		task.LastPostponedAt, task.RemindAt, task.RepeatType, task.RepeatInterval, task.RepeatUntil,
		task.NotifyEmail, task.LastNotifiedAt, task.CancelReason, task.CompletedAt, task.CreatedAt, task.UpdatedAt)
	return err
}

func (db *DB) CreatePlannerTasks(tasks []*PlannerTask) error {
	if len(tasks) == 0 {
		return nil
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO planner_tasks (
			id, profile_id, kind, entry_type, bucket, title, detail, notes, status, priority,
			planned_for, original_planned_for, rollover_count, last_postpone_reason, last_postponed_at,
			remind_at, repeat_type, repeat_interval, repeat_until, notify_email, last_notified_at,
			cancel_reason, completed_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, task := range tasks {
		task.ID = generateID(8)
		task.CreatedAt = now
		task.UpdatedAt = now
		if _, err := stmt.Exec(
			task.ID, task.ProfileID, task.Kind, task.EntryType, task.Bucket, task.Title, task.Detail, task.Notes, task.Status,
			task.Priority, task.PlannedFor, task.OriginalPlannedFor, task.RolloverCount, task.LastPostponeReason,
			task.LastPostponedAt, task.RemindAt, task.RepeatType, task.RepeatInterval, task.RepeatUntil,
			task.NotifyEmail, task.LastNotifiedAt, task.CancelReason, task.CompletedAt, task.CreatedAt,
			task.UpdatedAt,
		); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) GetPlannerTask(id string) (*PlannerTask, error) {
	row := db.conn.QueryRow(plannerTaskSelectSQL+` WHERE id = ?`, id)
	return scanPlannerTask(row)
}

const plannerTaskSelectSQL = `
	SELECT id, profile_id, kind, entry_type, bucket, title, detail, notes, status, priority,
	       planned_for, original_planned_for, rollover_count, last_postpone_reason, last_postponed_at,
	       remind_at, repeat_type, repeat_interval, repeat_until, notify_email, last_notified_at,
	       cancel_reason, completed_at, created_at, updated_at
	FROM planner_tasks
`

type plannerTaskScanner interface {
	Scan(dest ...interface{}) error
}

func scanPlannerTask(scanner plannerTaskScanner) (*PlannerTask, error) {
	task := &PlannerTask{}
	var lastPostponedAt sql.NullTime
	var remindAt sql.NullTime
	var repeatUntil sql.NullTime
	var lastNotifiedAt sql.NullTime
	var completedAt sql.NullTime
	if err := scanner.Scan(
		&task.ID,
		&task.ProfileID,
		&task.Kind,
		&task.EntryType,
		&task.Bucket,
		&task.Title,
		&task.Detail,
		&task.Notes,
		&task.Status,
		&task.Priority,
		&task.PlannedFor,
		&task.OriginalPlannedFor,
		&task.RolloverCount,
		&task.LastPostponeReason,
		&lastPostponedAt,
		&remindAt,
		&task.RepeatType,
		&task.RepeatInterval,
		&repeatUntil,
		&task.NotifyEmail,
		&lastNotifiedAt,
		&task.CancelReason,
		&completedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if lastPostponedAt.Valid {
		task.LastPostponedAt = &lastPostponedAt.Time
	}
	if remindAt.Valid {
		task.RemindAt = &remindAt.Time
	}
	if repeatUntil.Valid {
		task.RepeatUntil = &repeatUntil.Time
	}
	if lastNotifiedAt.Valid {
		task.LastNotifiedAt = &lastNotifiedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if strings.TrimSpace(task.EntryType) == "" {
		task.EntryType = PlannerEntryTask
	}
	if strings.TrimSpace(task.Bucket) == "" {
		task.Bucket = PlannerBucketPlanned
	}
	if strings.TrimSpace(task.RepeatType) == "" {
		task.RepeatType = "none"
	}
	if task.RepeatInterval <= 0 {
		task.RepeatInterval = 1
	}
	if strings.TrimSpace(task.OriginalPlannedFor) == "" {
		task.OriginalPlannedFor = task.PlannedFor
	}
	return task, nil
}

func (db *DB) ListPlannerTasks(profileID, kind, status string) ([]*PlannerTask, error) {
	return db.ListPlannerTasksAdvanced(profileID, kind, status, "", "")
}

func (db *DB) ListPlannerTasksAdvanced(profileID, kind, status, bucket, entryType string) ([]*PlannerTask, error) {
	query := plannerTaskSelectSQL + ` WHERE profile_id = ?`
	args := []interface{}{profileID}
	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if bucket != "" {
		query += ` AND bucket = ?`
		args = append(args, bucket)
	}
	if entryType != "" {
		query += ` AND entry_type = ?`
		args = append(args, entryType)
	}
	query += ` ORDER BY planned_for ASC, created_at ASC`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerTask, 0)
	for rows.Next() {
		task, err := scanPlannerTask(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, rows.Err()
}

func (db *DB) UpdatePlannerTask(task *PlannerTask) error {
	task.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
			UPDATE planner_tasks
			SET kind = ?, entry_type = ?, bucket = ?, title = ?, detail = ?, notes = ?, status = ?, priority = ?,
			    planned_for = ?, original_planned_for = ?, rollover_count = ?, last_postpone_reason = ?, last_postponed_at = ?,
			    remind_at = ?, repeat_type = ?, repeat_interval = ?, repeat_until = ?, notify_email = ?,
			    last_notified_at = ?, cancel_reason = ?, completed_at = ?, updated_at = ?
			WHERE id = ?
		`, task.Kind, task.EntryType, task.Bucket, task.Title, task.Detail, task.Notes, task.Status, task.Priority,
		task.PlannedFor, task.OriginalPlannedFor, task.RolloverCount, task.LastPostponeReason, task.LastPostponedAt,
		task.RemindAt, task.RepeatType, task.RepeatInterval, task.RepeatUntil, task.NotifyEmail,
		task.LastNotifiedAt, task.CancelReason, task.CompletedAt, task.UpdatedAt,
		task.ID)
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

func (db *DB) CreatePlannerTaskActivity(activity *PlannerTaskActivity) error {
	activity.ID = generateID(8)
	activity.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO planner_task_activities (id, task_id, profile_id, activity_type, title, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, activity.ID, activity.TaskID, activity.ProfileID, activity.ActivityType, activity.Title, activity.Content, activity.CreatedAt)
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

func (db *DB) ListPlannerTaskActivities(taskID string) ([]*PlannerTaskActivity, error) {
	rows, err := db.conn.Query(`
		SELECT id, task_id, profile_id, activity_type, title, content, created_at
		FROM planner_task_activities
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*PlannerTaskActivity, 0)
	for rows.Next() {
		item := &PlannerTaskActivity{}
		if err := rows.Scan(&item.ID, &item.TaskID, &item.ProfileID, &item.ActivityType, &item.Title, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (db *DB) ListPlannerTaskCommentSummaries(profileID string) (map[string]*PlannerTaskCommentSummary, error) {
	rows, err := db.conn.Query(`
		SELECT
			t.id,
			(SELECT COUNT(*) FROM planner_task_comments c WHERE c.task_id = t.id) AS comment_count,
			COALESCE((
				SELECT c.content
				FROM planner_task_comments c
				WHERE c.task_id = t.id
				ORDER BY c.created_at DESC, c.id DESC
				LIMIT 1
			), '') AS last_content,
			(
				SELECT c.created_at
				FROM planner_task_comments c
				WHERE c.task_id = t.id
				ORDER BY c.created_at DESC, c.id DESC
				LIMIT 1
			) AS last_created_at
		FROM planner_tasks t
		WHERE t.profile_id = ?
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*PlannerTaskCommentSummary)
	for rows.Next() {
		item := &PlannerTaskCommentSummary{}
		var lastCreatedAt sql.NullTime
		if err := rows.Scan(&item.TaskID, &item.CommentCount, &item.LastContent, &lastCreatedAt); err != nil {
			return nil, err
		}
		if lastCreatedAt.Valid {
			item.LastCreatedAt = &lastCreatedAt.Time
		}
		result[item.TaskID] = item
	}
	return result, rows.Err()
}

func (db *DB) CountPlannerTasks(profileID string) (map[string]int, error) {
	rows, err := db.conn.Query(`
		SELECT kind || ':' || entry_type || ':' || bucket || ':' || status AS bucket_key, COUNT(*)
		FROM planner_tasks
		WHERE profile_id = ?
		GROUP BY kind, entry_type, bucket, status
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
	rows, err := db.conn.Query(plannerTaskSelectSQL+`
		WHERE status IN ('open', 'in_progress')
		  AND bucket != 'inbox'
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
		task, err := scanPlannerTask(rows)
		if err != nil {
			return nil, err
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

func (db *DB) UpdatePlannerTaskReminderState(taskID string, remindAt, repeatUntil, lastNotifiedAt *time.Time) error {
	now := time.Now()
	_, err := db.conn.Exec(`
		UPDATE planner_tasks
		SET remind_at = ?, repeat_until = ?, last_notified_at = ?, updated_at = ?
		WHERE id = ?
	`, remindAt, repeatUntil, lastNotifiedAt, now, taskID)
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
			SUM(CASE WHEN t.status IN ('open', 'in_progress') THEN 1 ELSE 0 END) AS open_count
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
