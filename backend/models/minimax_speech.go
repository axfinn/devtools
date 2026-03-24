package models

import "time"

// MiniMaxSpeechFile 记录通过网关上传或生成的语音文件，便于按 API Key 做资源隔离。
type MiniMaxSpeechFile struct {
	ID        int64     `json:"id"`
	APIKeyID  string    `json:"api_key_id"`
	FileID    string    `json:"file_id"`
	Purpose   string    `json:"purpose"`
	Filename  string    `json:"filename"`
	Bytes     int64     `json:"bytes"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MiniMaxSpeechTask 记录异步长文本语音任务与所属 API Key 的映射关系。
type MiniMaxSpeechTask struct {
	TaskID       string    `json:"task_id"`
	APIKeyID     string    `json:"api_key_id"`
	Model        string    `json:"model"`
	Status       string    `json:"status"`
	OutputFileID string    `json:"output_file_id"`
	RequestBody  string    `json:"request_body"`
	ResultBody   string    `json:"result_body"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (db *DB) InitMiniMaxSpeechFiles() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS minimax_speech_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id TEXT NOT NULL,
			file_id TEXT NOT NULL UNIQUE,
			purpose TEXT NOT NULL,
			filename TEXT DEFAULT '',
			bytes INTEGER DEFAULT 0,
			status TEXT DEFAULT 'available',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_minimax_speech_files_api_key ON minimax_speech_files(api_key_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_minimax_speech_files_purpose ON minimax_speech_files(purpose, created_at DESC);
	`)
	return err
}

func (db *DB) UpsertMiniMaxSpeechFile(file *MiniMaxSpeechFile) error {
	now := time.Now()
	if file.CreatedAt.IsZero() {
		file.CreatedAt = now
	}
	file.UpdatedAt = now
	_, err := db.conn.Exec(`
		INSERT INTO minimax_speech_files (api_key_id, file_id, purpose, filename, bytes, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_id) DO UPDATE SET
			api_key_id = excluded.api_key_id,
			purpose = excluded.purpose,
			filename = excluded.filename,
			bytes = excluded.bytes,
			status = excluded.status,
			updated_at = excluded.updated_at
	`, file.APIKeyID, file.FileID, file.Purpose, file.Filename, file.Bytes, file.Status, file.CreatedAt, file.UpdatedAt)
	return err
}

func (db *DB) GetMiniMaxSpeechFileByFileID(fileID string) (*MiniMaxSpeechFile, error) {
	file := &MiniMaxSpeechFile{}
	err := db.conn.QueryRow(`
		SELECT id, api_key_id, file_id, purpose, filename, bytes, status, created_at, updated_at
		FROM minimax_speech_files
		WHERE file_id = ?
	`, fileID).Scan(&file.ID, &file.APIKeyID, &file.FileID, &file.Purpose, &file.Filename, &file.Bytes, &file.Status, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (db *DB) ListMiniMaxSpeechFiles(apiKeyID, purpose string, limit, offset int) ([]*MiniMaxSpeechFile, error) {
	query := `
		SELECT id, api_key_id, file_id, purpose, filename, bytes, status, created_at, updated_at
		FROM minimax_speech_files
		WHERE 1=1`
	args := make([]interface{}, 0, 4)

	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}
	if purpose != "" {
		query += ` AND purpose = ?`
		args = append(args, purpose)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*MiniMaxSpeechFile, 0)
	for rows.Next() {
		file := &MiniMaxSpeechFile{}
		if err := rows.Scan(&file.ID, &file.APIKeyID, &file.FileID, &file.Purpose, &file.Filename, &file.Bytes, &file.Status, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (db *DB) DeleteMiniMaxSpeechFile(fileID, apiKeyID string) error {
	query := `DELETE FROM minimax_speech_files WHERE file_id = ?`
	args := []interface{}{fileID}
	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}
	_, err := db.conn.Exec(query, args...)
	return err
}

func (db *DB) InitMiniMaxSpeechTasks() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS minimax_speech_tasks (
			task_id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			model TEXT NOT NULL,
			status TEXT DEFAULT 'submitted',
			output_file_id TEXT DEFAULT '',
			request_body TEXT DEFAULT '',
			result_body TEXT DEFAULT '',
			error_message TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_minimax_speech_tasks_api_key ON minimax_speech_tasks(api_key_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_minimax_speech_tasks_status ON minimax_speech_tasks(status, updated_at DESC);
	`)
	return err
}

func (db *DB) UpsertMiniMaxSpeechTask(task *MiniMaxSpeechTask) error {
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	_, err := db.conn.Exec(`
		INSERT INTO minimax_speech_tasks (task_id, api_key_id, model, status, output_file_id, request_body, result_body, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET
			api_key_id = excluded.api_key_id,
			model = excluded.model,
			status = excluded.status,
			output_file_id = excluded.output_file_id,
			request_body = excluded.request_body,
			result_body = excluded.result_body,
			error_message = excluded.error_message,
			updated_at = excluded.updated_at
	`, task.TaskID, task.APIKeyID, task.Model, task.Status, task.OutputFileID, task.RequestBody, task.ResultBody, task.ErrorMessage, task.CreatedAt, task.UpdatedAt)
	return err
}

func (db *DB) GetMiniMaxSpeechTask(taskID string) (*MiniMaxSpeechTask, error) {
	task := &MiniMaxSpeechTask{}
	err := db.conn.QueryRow(`
		SELECT task_id, api_key_id, model, status, output_file_id, request_body, result_body, error_message, created_at, updated_at
		FROM minimax_speech_tasks
		WHERE task_id = ?
	`, taskID).Scan(&task.TaskID, &task.APIKeyID, &task.Model, &task.Status, &task.OutputFileID, &task.RequestBody, &task.ResultBody, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (db *DB) ListMiniMaxSpeechTasks(apiKeyID string, limit, offset int) ([]*MiniMaxSpeechTask, error) {
	query := `
		SELECT task_id, api_key_id, model, status, output_file_id, request_body, result_body, error_message, created_at, updated_at
		FROM minimax_speech_tasks
		WHERE 1=1`
	args := make([]interface{}, 0, 3)
	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}
	query += ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*MiniMaxSpeechTask, 0)
	for rows.Next() {
		task := &MiniMaxSpeechTask{}
		if err := rows.Scan(&task.TaskID, &task.APIKeyID, &task.Model, &task.Status, &task.OutputFileID, &task.RequestBody, &task.ResultBody, &task.ErrorMessage, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (db *DB) DeleteVoiceCloneAny(id uint) error {
	_, err := db.conn.Exec(`DELETE FROM voice_clones WHERE id = ?`, id)
	return err
}
