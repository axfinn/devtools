package models

import (
	"time"
)

type ChatRoom struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Password     string    `json:"-"`
	HasPassword  bool      `json:"has_password"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
	CreatorIP    string    `json:"-"`
}

type ChatMessage struct {
	ID        int64     `json:"id"`
	RoomID    string    `json:"room_id"`
	Nickname  string    `json:"nickname"`
	Content   string    `json:"content"`
	MsgType   string    `json:"msg_type"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) InitChat() error {
	query := `
	CREATE TABLE IF NOT EXISTS chat_rooms (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		password TEXT DEFAULT '',
		last_active_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_rooms_last_active ON chat_rooms(last_active_at);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id TEXT NOT NULL,
		nickname TEXT NOT NULL,
		content TEXT NOT NULL,
		msg_type TEXT DEFAULT 'text',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (room_id) REFERENCES chat_rooms(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_messages_room ON chat_messages(room_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created ON chat_messages(created_at);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreateRoom(room *ChatRoom) error {
	room.ID = generateID(8)
	room.CreatedAt = time.Now()
	room.LastActiveAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO chat_rooms (id, name, password, last_active_at, created_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?)
	`, room.ID, room.Name, room.Password, room.LastActiveAt, room.CreatedAt, room.CreatorIP)

	return err
}

func (db *DB) GetRoom(id string) (*ChatRoom, error) {
	room := &ChatRoom{}
	var password string
	err := db.conn.QueryRow(`
		SELECT id, name, password, last_active_at, created_at, creator_ip
		FROM chat_rooms WHERE id = ?
	`, id).Scan(&room.ID, &room.Name, &password, &room.LastActiveAt, &room.CreatedAt, &room.CreatorIP)

	if err != nil {
		return nil, err
	}
	room.Password = password
	room.HasPassword = password != ""
	return room, nil
}

func (db *DB) GetRooms(limit int) ([]*ChatRoom, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, password, last_active_at, created_at
		FROM chat_rooms
		ORDER BY last_active_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*ChatRoom
	for rows.Next() {
		room := &ChatRoom{}
		var password string
		if err := rows.Scan(&room.ID, &room.Name, &password, &room.LastActiveAt, &room.CreatedAt); err != nil {
			return nil, err
		}
		room.HasPassword = password != ""
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (db *DB) UpdateRoomActivity(roomID string) error {
	_, err := db.conn.Exec("UPDATE chat_rooms SET last_active_at = ? WHERE id = ?", time.Now(), roomID)
	return err
}

func (db *DB) DeleteRoom(id string) error {
	_, err := db.conn.Exec("DELETE FROM chat_rooms WHERE id = ?", id)
	return err
}

func (db *DB) CreateMessage(msg *ChatMessage) error {
	msg.CreatedAt = time.Now()
	result, err := db.conn.Exec(`
		INSERT INTO chat_messages (room_id, nickname, content, msg_type, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, msg.RoomID, msg.Nickname, msg.Content, msg.MsgType, msg.CreatedAt)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	msg.ID = id
	return nil
}

func (db *DB) GetMessages(roomID string, limit int) ([]*ChatMessage, error) {
	rows, err := db.conn.Query(`
		SELECT id, room_id, nickname, content, msg_type, created_at
		FROM chat_messages
		WHERE room_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*ChatMessage
	for rows.Next() {
		msg := &ChatMessage{}
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.Nickname, &msg.Content, &msg.MsgType, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	// 反转顺序，使最早的消息在前
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (db *DB) CleanExpiredRooms(days int) (int64, error) {
	expireTime := time.Now().AddDate(0, 0, -days)
	result, err := db.conn.Exec("DELETE FROM chat_rooms WHERE last_active_at < ?", expireTime)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) CleanExpiredMessages(days int) (int64, error) {
	expireTime := time.Now().AddDate(0, 0, -days)
	result, err := db.conn.Exec("DELETE FROM chat_messages WHERE created_at < ?", expireTime)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) RoomExists(id string) bool {
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM chat_rooms WHERE id = ?", id).Scan(&count)
	return count > 0
}
