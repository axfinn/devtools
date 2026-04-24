package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func init() {
	RegisterInit("Mermaid(mermaid_shares)", (*DB).InitMermaid)
}

type MermaidProject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MermaidVersion struct {
	ID        int64     `json:"id"`
	ProjectID string    `json:"project_id"`
	Code      string    `json:"code"`
	Source    string    `json:"source"` // "ai" | "manual"
	CreatedAt time.Time `json:"created_at"`
}

type MermaidMessage struct {
	ID        int64     `json:"id"`
	ProjectID string    `json:"project_id"`
	Role      string    `json:"role"` // "user" | "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) InitMermaid() error {
	_, err := db.conn.Exec(`
	CREATE TABLE IF NOT EXISTS mermaid_projects (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS mermaid_versions (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id TEXT NOT NULL,
		code       TEXT NOT NULL,
		source     TEXT NOT NULL DEFAULT 'manual',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES mermaid_projects(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_mermaid_versions_project ON mermaid_versions(project_id);
	CREATE TABLE IF NOT EXISTS mermaid_messages (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id TEXT NOT NULL,
		role       TEXT NOT NULL,
		content    TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES mermaid_projects(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_mermaid_messages_project ON mermaid_messages(project_id);
	`)
	return err
}

func (db *DB) CreateMermaidProject(name string) (*MermaidProject, error) {
	b := make([]byte, 8)
	rand.Read(b)
	id := hex.EncodeToString(b)
	now := time.Now()
	_, err := db.conn.Exec(`INSERT INTO mermaid_projects (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		id, name, now, now)
	if err != nil {
		return nil, err
	}
	return &MermaidProject{ID: id, Name: name, CreatedAt: now, UpdatedAt: now}, nil
}

func (db *DB) ListMermaidProjects() ([]MermaidProject, error) {
	rows, err := db.conn.Query(`SELECT id, name, created_at, updated_at FROM mermaid_projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []MermaidProject
	for rows.Next() {
		var p MermaidProject
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

func (db *DB) GetMermaidProject(id string) (*MermaidProject, error) {
	var p MermaidProject
	err := db.conn.QueryRow(`SELECT id, name, created_at, updated_at FROM mermaid_projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) DeleteMermaidProject(id string) error {
	_, err := db.conn.Exec(`DELETE FROM mermaid_projects WHERE id = ?`, id)
	return err
}

func (db *DB) AddMermaidVersion(projectID, code, source string) (*MermaidVersion, error) {
	now := time.Now()
	res, err := db.conn.Exec(`INSERT INTO mermaid_versions (project_id, code, source, created_at) VALUES (?, ?, ?, ?)`,
		projectID, code, source, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	db.conn.Exec(`UPDATE mermaid_projects SET updated_at = ? WHERE id = ?`, now, projectID)
	return &MermaidVersion{ID: id, ProjectID: projectID, Code: code, Source: source, CreatedAt: now}, nil
}

func (db *DB) ListMermaidVersions(projectID string) ([]MermaidVersion, error) {
	rows, err := db.conn.Query(`SELECT id, project_id, code, source, created_at FROM mermaid_versions WHERE project_id = ? ORDER BY id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []MermaidVersion
	for rows.Next() {
		var v MermaidVersion
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.Code, &v.Source, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

func (db *DB) AddMermaidMessage(projectID, role, content string) error {
	_, err := db.conn.Exec(`INSERT INTO mermaid_messages (project_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		projectID, role, content, time.Now())
	return err
}

func (db *DB) ListMermaidMessages(projectID string) ([]MermaidMessage, error) {
	rows, err := db.conn.Query(`SELECT id, project_id, role, content, created_at FROM mermaid_messages WHERE project_id = ? ORDER BY id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []MermaidMessage
	for rows.Next() {
		var m MermaidMessage
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}

func (db *DB) ClearMermaidMessages(projectID string) error {
	_, err := db.conn.Exec(`DELETE FROM mermaid_messages WHERE project_id = ?`, projectID)
	return err
}
