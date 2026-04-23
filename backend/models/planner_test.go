package models

import (
	"testing"
)

func TestInitPlannerMigratesLegacySchema(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db failed: %v", err)
	}
	defer db.Close()

	_, err = db.conn.Exec(`
		DROP TABLE IF EXISTS planner_task_comments;
		DROP TABLE IF EXISTS planner_tasks;
		DROP TABLE IF EXISTS planner_profiles;

		CREATE TABLE planner_profiles (
			id TEXT PRIMARY KEY,
			password_index TEXT NOT NULL UNIQUE,
			creator_key TEXT NOT NULL,
			name TEXT NOT NULL,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE planner_tasks (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			detail TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
			planned_for TEXT NOT NULL,
			remind_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE planner_task_comments (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			profile_id TEXT NOT NULL,
			author TEXT NOT NULL DEFAULT '用户',
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("prepare legacy schema failed: %v", err)
	}

	if err := db.InitPlanner(); err != nil {
		t.Fatalf("InitPlanner should migrate legacy schema, got error: %v", err)
	}

	rows, err := db.conn.Query(`PRAGMA table_info(planner_tasks)`)
	if err != nil {
		t.Fatalf("inspect planner_tasks failed: %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var (
			cid       int
			name      string
			dataType  string
			notNull   int
			defaultV  interface{}
			primaryPK int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultV, &primaryPK); err != nil {
			t.Fatalf("scan column failed: %v", err)
		}
		columns[name] = true
	}

	required := []string{
		"entry_type",
		"bucket",
		"notes",
		"notify_email",
		"last_notified_at",
		"completed_at",
		"original_planned_for",
		"rollover_count",
		"last_postpone_reason",
		"last_postponed_at",
		"cancel_reason",
	}
	for _, column := range required {
		if !columns[column] {
			t.Fatalf("expected migrated column %q to exist", column)
		}
	}
}
