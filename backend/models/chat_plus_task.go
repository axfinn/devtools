package models

import (
	"database/sql"
	"time"
)

func init() {
	RegisterInit("AI Chat(chat_plus_tasks)", (*DB).InitChatPlusTasks)
}

// ChatPlusTask /ai-chat 页的异步聊天任务。
//
// 为什么要落 SQLite 而不是只留内存：弱网下客户端会断线/刷新/切后台，
// 任务本身必须在服务端跑完并留下结果；用户重进页面时要能拿到"成功的内容"或"失败的原因"，
// 而不是只剩一条空白对话。内存里的 SSE 片段只负责"续传"，最终结果以这张表为准。
type ChatPlusTask struct {
	ID             string    `json:"task_id"`
	ConversationID string    `json:"conversation_id"`
	Model          string    `json:"model"`
	Status         string    `json:"status"` // running | succeeded | failed | canceled
	Content        string    `json:"content"`
	Reasoning      string    `json:"reasoning,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	Seq            int       `json:"seq"` // 已产出的 SSE 片段序号（用于断线续传）
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (db *DB) InitChatPlusTasks() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS chat_plus_tasks (
			id TEXT PRIMARY KEY,
			conversation_id TEXT DEFAULT '',
			model TEXT DEFAULT '',
			status TEXT DEFAULT 'running',
			content TEXT DEFAULT '',
			reasoning TEXT DEFAULT '',
			error_message TEXT DEFAULT '',
			seq INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_chat_plus_tasks_conv
			ON chat_plus_tasks(conversation_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_chat_plus_tasks_status
			ON chat_plus_tasks(status, updated_at DESC);
	`)
	return err
}

func (db *DB) CreateChatPlusTask(task *ChatPlusTask) error {
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	_, err := db.conn.Exec(`
		INSERT INTO chat_plus_tasks
			(id, conversation_id, model, status, content, reasoning, error_message, seq, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.ConversationID, task.Model, task.Status, task.Content, task.Reasoning,
		task.ErrorMessage, task.Seq, task.CreatedAt, task.UpdatedAt)
	return err
}

// UpdateChatPlusTaskProgress 记录进度快照（内容 + 已产出片段序号）。
// 调用方必须限流（秒级），否则每个 SSE 片段一次写库会把 SQLite 打满。
func (db *DB) UpdateChatPlusTaskProgress(id, content, reasoning string, seq int) error {
	_, err := db.conn.Exec(`
		UPDATE chat_plus_tasks
		   SET content = ?, reasoning = ?, seq = ?, updated_at = ?
		 WHERE id = ?
	`, content, reasoning, seq, time.Now(), id)
	return err
}

// FinishChatPlusTask 写入终态。失败时 errorMessage 必须非空，前端靠它告诉用户"为什么失败"。
func (db *DB) FinishChatPlusTask(id, status, content, reasoning, errMessage string, seq int) error {
	_, err := db.conn.Exec(`
		UPDATE chat_plus_tasks
		   SET status = ?, content = ?, reasoning = ?, error_message = ?, seq = ?, updated_at = ?
		 WHERE id = ?
	`, status, content, reasoning, errMessage, seq, time.Now(), id)
	return err
}

func (db *DB) GetChatPlusTask(id string) (*ChatPlusTask, error) {
	row := db.conn.QueryRow(`
		SELECT id, conversation_id, model, status, content, reasoning, error_message, seq, created_at, updated_at
		  FROM chat_plus_tasks WHERE id = ?
	`, id)
	return scanChatPlusTask(row)
}

// ListChatPlusTasks 按会话倒序返回最近任务，供页面刷新后对齐消息。
func (db *DB) ListChatPlusTasks(conversationID string, limit int) ([]*ChatPlusTask, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.conn.Query(`
		SELECT id, conversation_id, model, status, content, reasoning, error_message, seq, created_at, updated_at
		  FROM chat_plus_tasks
		 WHERE conversation_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*ChatPlusTask, 0, limit)
	for rows.Next() {
		t, err := scanChatPlusTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// FailStaleChatPlusTasks 把上次进程异常退出时遗留的 running 任务标为失败，
// 否则用户重进页面会看到一个永远"进行中"的任务。
func (db *DB) FailStaleChatPlusTasks(reason string) (int64, error) {
	res, err := db.conn.Exec(`
		UPDATE chat_plus_tasks
		   SET status = 'failed', error_message = ?, updated_at = ?
		 WHERE status = 'running'
	`, reason, time.Now())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DB) CleanOldChatPlusTasks(before time.Time) (int64, error) {
	res, err := db.conn.Exec(`DELETE FROM chat_plus_tasks WHERE updated_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type chatPlusTaskScanner interface {
	Scan(dest ...interface{}) error
}

func scanChatPlusTask(row chatPlusTaskScanner) (*ChatPlusTask, error) {
	var t ChatPlusTask
	err := row.Scan(&t.ID, &t.ConversationID, &t.Model, &t.Status, &t.Content,
		&t.Reasoning, &t.ErrorMessage, &t.Seq, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
