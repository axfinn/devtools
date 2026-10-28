package models

import (
	"database/sql"
	"time"
)

func init() {
	RegisterInit("AI Chat 会话/消息(chat_plus_conversations)", (*DB).InitChatPlusConvs)
}

// 本文件：/ai-chat 页的会话与消息持久化。
//
// 为什么要有这张表：
//   之前会话列表和消息全部只存在浏览器 localStorage 里，等于"每个浏览器一份私有历史"。
//   换台设备（手机/另一台电脑）打开就是空白，同一个会话在 A 端发出的提问在 B 端永远看不到，
//   任务表（chat_plus_tasks）里虽然有结果，但 reconciliation 只按 conversation_id 找，
//   而 conversation_id 本身也是每个浏览器各自生成的 —— 于是"多端不同步"。
//
// 现在把会话和消息搬进 SQLite 作为唯一真相，localStorage 退化成离线缓存。

// ChatPlusConversation 一条会话（列表项）。
type ChatPlusConversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	MsgCount  int       `json:"msg_count"`
}

// ChatPlusMessage 一条消息。Images 存 JSON 数组（图片落文件，只存相对 URL）。
type ChatPlusMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"` // user | assistant
	Content        string    `json:"content"`
	Reasoning      string    `json:"reasoning,omitempty"`
	Images         string    `json:"images"`  // JSON: [{"id","url","w","h"}]
	TaskID         string    `json:"task_id"` // 关联的异步任务（assistant 消息）
	Kind           string    `json:"kind"`    // text | image
	Status         string    `json:"status"`  // ok | failed | canceled | running（前端展示失败原因用）
	ErrMessage     string    `json:"error_message,omitempty"`
	Ts             int64     `json:"ts"` // 毫秒时间戳，排序 + 增量同步游标
	UpdatedAt      time.Time `json:"updated_at"`
}

func (db *DB) InitChatPlusConvs() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS chat_plus_conversations (
			id TEXT PRIMARY KEY,
			title TEXT DEFAULT '',
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS chat_plus_messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			content TEXT DEFAULT '',
			reasoning TEXT DEFAULT '',
			images TEXT DEFAULT '',
			task_id TEXT DEFAULT '',
			kind TEXT DEFAULT 'text',
			status TEXT DEFAULT 'ok',
			error_message TEXT DEFAULT '',
			ts INTEGER DEFAULT 0,
			updated_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_chat_plus_conv_updated
			ON chat_plus_conversations(updated_at DESC);
		CREATE INDEX IF NOT EXISTS idx_chat_plus_msg_conv
			ON chat_plus_messages(conversation_id, ts);
	`)
	return err
}

// UpsertChatPlusConversation 新建或更新会话标题。已存在时不覆盖 created_at，
// 标题为空时保留原标题（避免前端草稿态把标题冲掉）。
func (db *DB) UpsertChatPlusConversation(id, title string) error {
	if id == "" {
		return nil
	}
	now := time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO chat_plus_conversations (id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = CASE WHEN excluded.title <> '' THEN excluded.title ELSE chat_plus_conversations.title END,
			updated_at = excluded.updated_at
	`, id, title, now, now)
	return err
}

// TouchChatPlusConversation 只更新 updated_at（有新消息时把会话顶到列表前面）。
func (db *DB) TouchChatPlusConversation(id string) error {
	if id == "" {
		return nil
	}
	now := time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO chat_plus_conversations (id, title, created_at, updated_at)
		VALUES (?, '', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, id, now, now)
	return err
}

func (db *DB) ListChatPlusConversations(limit int) ([]*ChatPlusConversation, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := db.conn.Query(`
		SELECT c.id, c.title, c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM chat_plus_messages m WHERE m.conversation_id = c.id) AS cnt
		  FROM chat_plus_conversations c
		 ORDER BY c.updated_at DESC
		 LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*ChatPlusConversation, 0, 16)
	for rows.Next() {
		var c ChatPlusConversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt, &c.MsgCount); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (db *DB) DeleteChatPlusConversation(id string) error {
	if id == "" {
		return nil
	}
	if _, err := db.conn.Exec(`DELETE FROM chat_plus_messages WHERE conversation_id = ?`, id); err != nil {
		return err
	}
	_, err := db.conn.Exec(`DELETE FROM chat_plus_conversations WHERE id = ?`, id)
	return err
}

// UpsertChatPlusMessage 写入/覆盖一条消息（流式过程中会反复覆盖同一条 assistant 消息）。
func (db *DB) UpsertChatPlusMessage(m *ChatPlusMessage) error {
	now := time.Now()
	if m.Ts <= 0 {
		m.Ts = now.UnixMilli()
	}
	if m.Kind == "" {
		m.Kind = "text"
	}
	if m.Status == "" {
		m.Status = "ok"
	}
	_, err := db.conn.Exec(`
		INSERT INTO chat_plus_messages
			(id, conversation_id, role, content, reasoning, images, task_id, kind, status, error_message, ts, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			reasoning = excluded.reasoning,
			images = excluded.images,
			task_id = excluded.task_id,
			kind = excluded.kind,
			status = excluded.status,
			error_message = excluded.error_message,
			updated_at = excluded.updated_at
	`, m.ID, m.ConversationID, m.Role, m.Content, m.Reasoning, m.Images, m.TaskID,
		m.Kind, m.Status, m.ErrMessage, m.Ts, now)
	return err
}

// ListChatPlusMessages sinceTs > 0 时只返回该时间戳之后的消息（多端增量拉取）。
func (db *DB) ListChatPlusMessages(convID string, sinceTs int64, limit int) ([]*ChatPlusMessage, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	rows, err := db.conn.Query(`
		SELECT id, conversation_id, role, content, reasoning, images, task_id, kind, status, error_message, ts, updated_at
		  FROM chat_plus_messages
		 WHERE conversation_id = ? AND ts >= ?
		 ORDER BY ts ASC
		 LIMIT ?
	`, convID, sinceTs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*ChatPlusMessage, 0, 32)
	for rows.Next() {
		var m ChatPlusMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.Reasoning,
			&m.Images, &m.TaskID, &m.Kind, &m.Status, &m.ErrMessage, &m.Ts, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (db *DB) GetChatPlusMessage(id string) (*ChatPlusMessage, error) {
	var m ChatPlusMessage
	err := db.conn.QueryRow(`
		SELECT id, conversation_id, role, content, reasoning, images, task_id, kind, status, error_message, ts, updated_at
		  FROM chat_plus_messages WHERE id = ?
	`, id).Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.Reasoning,
		&m.Images, &m.TaskID, &m.Kind, &m.Status, &m.ErrMessage, &m.Ts, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
