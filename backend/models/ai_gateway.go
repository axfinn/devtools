package models

import (
	"database/sql"
	"errors"
	"time"
)

func init() {
	RegisterInit("AI Gateway(llm_tasks)", (*DB).InitLLMTasks)
	RegisterInit("AI Gateway(ai_api_keys)", (*DB).InitAIGateway)
	RegisterInit("AI Gateway(anthropic_providers)", (*DB).InitAnthropicProviders)
	RegisterInit("MiniMax Media(media_tasks)", (*DB).InitMiniMaxMediaTasks)
	RegisterInit("语音克隆(voice_clones)", (*DB).InitVoiceClones)
}

type AIAPIKey struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	KeyPrefix         string     `json:"key_prefix"`
	KeyHash           string     `json:"-"`
	Status            string     `json:"status"`
	AllowedModels     string     `json:"allowed_models"`
	AllowedScopes     string     `json:"allowed_scopes"`
	RateLimitPerHour  int        `json:"rate_limit_per_hour"`
	TotalRequests     int        `json:"total_requests"`
	TotalInputTokens  int        `json:"total_input_tokens"`
	TotalOutputTokens int        `json:"total_output_tokens"`
	TotalTokens       int        `json:"total_tokens"`
	TotalCost         float64    `json:"total_cost"`
	BillingCurrency   string     `json:"billing_currency"`
	BudgetLimit       float64    `json:"budget_limit"`
	AlertThreshold    float64    `json:"alert_threshold"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CreatorIP         string     `json:"creator_ip"`
	Notes             string     `json:"notes"`
}

type AIAPIRequestLog struct {
	ID            int64     `json:"id"`
	APIKeyID      string    `json:"api_key_id"`
	Model         string    `json:"model"`
	Provider      string    `json:"provider"`
	Endpoint      string    `json:"endpoint"`
	RequestType   string    `json:"request_type"`
	StatusCode    int       `json:"status_code"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message"`
	RequestBody   string    `json:"request_body"`
	ResponseBody  string    `json:"response_body"`
	ClientIP      string    `json:"client_ip"`
	LatencyMS     int64     `json:"latency_ms"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	EstimatedCost float64   `json:"estimated_cost"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
}

type AIUsageReportRow struct {
	Period       string  `json:"period"`
	APIKeyID     string  `json:"api_key_id"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	RequestCount int     `json:"request_count"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	Currency     string  `json:"currency"`
}

// LLMTask 异步 LLM 任务（用于避免 Cloudflare 524 长连接超时）
type LLMTask struct {
	ID           string     `json:"id"`
	APIKeyID     string     `json:"api_key_id"`
	Model        string     `json:"model"`
	Provider     string     `json:"provider"`
	Status       string     `json:"status"` // pending/running/succeeded/failed
	RequestBody  string     `json:"request_body"`
	ResultJSON   string     `json:"result_json,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ClientIP     string     `json:"client_ip"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (db *DB) InitLLMTasks() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS llm_tasks (
			id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			request_body TEXT DEFAULT '',
			result_json TEXT DEFAULT '',
			error_message TEXT DEFAULT '',
			client_ip TEXT DEFAULT '',
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_llm_tasks_status ON llm_tasks(status, created_at DESC);
	`)
	return err
}

func (db *DB) CreateLLMTask(task *LLMTask) error {
	task.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO llm_tasks (id, api_key_id, model, provider, status, request_body, result_json, error_message, client_ip, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.APIKeyID, task.Model, task.Provider, task.Status, task.RequestBody, task.ResultJSON, task.ErrorMessage, task.ClientIP, task.CreatedAt)
	return err
}

func (db *DB) GetLLMTask(id string) (*LLMTask, error) {
	task := &LLMTask{}
	err := db.conn.QueryRow(`
		SELECT id, api_key_id, model, provider, status, request_body, result_json, error_message, client_ip, completed_at, created_at
		FROM llm_tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.APIKeyID, &task.Model, &task.Provider, &task.Status, &task.RequestBody, &task.ResultJSON, &task.ErrorMessage, &task.ClientIP, &task.CompletedAt, &task.CreatedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (db *DB) UpdateLLMTask(task *LLMTask) error {
	_, err := db.conn.Exec(`
		UPDATE llm_tasks SET status=?, result_json=?, error_message=?, completed_at=? WHERE id=?
	`, task.Status, task.ResultJSON, task.ErrorMessage, task.CompletedAt, task.ID)
	return err
}

func (db *DB) InitAIGateway() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS ai_api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			allowed_models TEXT DEFAULT '["*"]',
			allowed_scopes TEXT DEFAULT '["chat","media"]',
			rate_limit_per_hour INTEGER DEFAULT 1000,
			total_requests INTEGER DEFAULT 0,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
			billing_currency TEXT DEFAULT 'CNY',
			budget_limit REAL DEFAULT 0,
			alert_threshold REAL DEFAULT 0.8,
			last_used_at DATETIME,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			creator_ip TEXT DEFAULT '',
			notes TEXT DEFAULT ''
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS ai_api_request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			request_type TEXT NOT NULL,
			status_code INTEGER DEFAULT 0,
			success INTEGER DEFAULT 0,
			error_message TEXT DEFAULT '',
			request_body TEXT DEFAULT '',
			response_body TEXT DEFAULT '',
			client_ip TEXT DEFAULT '',
			latency_ms INTEGER DEFAULT 0,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			estimated_cost REAL DEFAULT 0,
			currency TEXT DEFAULT 'CNY',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (api_key_id) REFERENCES ai_api_keys(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_ai_api_keys_prefix ON ai_api_keys(key_prefix)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_api_keys_status ON ai_api_keys(status)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_api_logs_key_time ON ai_api_request_logs(api_key_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_api_logs_model ON ai_api_request_logs(model, created_at DESC)`,
	}
	for _, idx := range indexes {
		if _, err := db.conn.Exec(idx); err != nil {
			return err
		}
	}

	alterStatements := []string{
		`ALTER TABLE ai_api_keys ADD COLUMN total_input_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_api_keys ADD COLUMN total_output_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_api_keys ADD COLUMN total_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_api_keys ADD COLUMN total_cost REAL DEFAULT 0`,
		`ALTER TABLE ai_api_keys ADD COLUMN billing_currency TEXT DEFAULT 'CNY'`,
		`ALTER TABLE ai_api_keys ADD COLUMN budget_limit REAL DEFAULT 0`,
		`ALTER TABLE ai_api_keys ADD COLUMN alert_threshold REAL DEFAULT 0.8`,
		`ALTER TABLE ai_api_request_logs ADD COLUMN input_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_api_request_logs ADD COLUMN output_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_api_request_logs ADD COLUMN total_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_api_request_logs ADD COLUMN estimated_cost REAL DEFAULT 0`,
		`ALTER TABLE ai_api_request_logs ADD COLUMN currency TEXT DEFAULT 'CNY'`,
	}
	for _, stmt := range alterStatements {
		_, _ = db.conn.Exec(stmt)
	}

	return nil
}

func (db *DB) CreateAIAPIKey(key *AIAPIKey) error {
	if key == nil {
		return errors.New("key is nil")
	}
	key.ID = generateID(8)
	now := time.Now()
	key.CreatedAt = now
	key.UpdatedAt = now
	if key.BillingCurrency == "" {
		key.BillingCurrency = "CNY"
	}
	_, err := db.conn.Exec(`
		INSERT INTO ai_api_keys (
			id, name, key_prefix, key_hash, status, allowed_models, allowed_scopes,
			rate_limit_per_hour, total_requests, total_input_tokens, total_output_tokens,
			total_tokens, total_cost, billing_currency, budget_limit, alert_threshold,
			last_used_at, expires_at, created_at, updated_at, creator_ip, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.Name, key.KeyPrefix, key.KeyHash, key.Status, key.AllowedModels,
		key.AllowedScopes, key.RateLimitPerHour, key.TotalRequests, key.TotalInputTokens,
		key.TotalOutputTokens, key.TotalTokens, key.TotalCost, key.BillingCurrency, key.BudgetLimit, key.AlertThreshold,
		key.LastUsedAt, key.ExpiresAt, key.CreatedAt, key.UpdatedAt, key.CreatorIP, key.Notes,
	)
	return err
}

func (db *DB) GetAIAPIKeyByID(id string) (*AIAPIKey, error) {
	return db.scanAIAPIKey(`
		SELECT id, name, key_prefix, key_hash, status, allowed_models, allowed_scopes,
			rate_limit_per_hour, total_requests, total_input_tokens, total_output_tokens,
			total_tokens, total_cost, billing_currency, budget_limit, alert_threshold,
			last_used_at, expires_at, created_at, updated_at, creator_ip, notes
		FROM ai_api_keys
		WHERE id = ?`, id)
}

func (db *DB) GetAIAPIKeysByPrefix(prefix string) ([]*AIAPIKey, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, key_prefix, key_hash, status, allowed_models, allowed_scopes,
			rate_limit_per_hour, total_requests, total_input_tokens, total_output_tokens,
			total_tokens, total_cost, billing_currency, budget_limit, alert_threshold,
			last_used_at, expires_at, created_at, updated_at, creator_ip, notes
		FROM ai_api_keys
		WHERE key_prefix = ?
		ORDER BY created_at DESC`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*AIAPIKey, 0)
	for rows.Next() {
		key, err := scanAIAPIKeyRow(rows)
		if err == nil {
			items = append(items, key)
		}
	}
	return items, nil
}

func (db *DB) ListAIAPIKeys(limit, offset int) ([]*AIAPIKey, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, key_prefix, key_hash, status, allowed_models, allowed_scopes,
			rate_limit_per_hour, total_requests, total_input_tokens, total_output_tokens,
			total_tokens, total_cost, billing_currency, budget_limit, alert_threshold,
			last_used_at, expires_at, created_at, updated_at, creator_ip, notes
		FROM ai_api_keys
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*AIAPIKey, 0)
	for rows.Next() {
		key, err := scanAIAPIKeyRow(rows)
		if err == nil {
			items = append(items, key)
		}
	}
	return items, nil
}

func (db *DB) CountAIAPIKeys() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM ai_api_keys`).Scan(&count)
	return count, err
}

func (db *DB) UpdateAIAPIKey(key *AIAPIKey) error {
	if key == nil {
		return errors.New("key is nil")
	}
	key.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE ai_api_keys
		SET name = ?, status = ?, allowed_models = ?, allowed_scopes = ?, rate_limit_per_hour = ?,
			total_requests = ?, total_input_tokens = ?, total_output_tokens = ?, total_tokens = ?,
			total_cost = ?, billing_currency = ?, budget_limit = ?, alert_threshold = ?,
			last_used_at = ?, expires_at = ?, updated_at = ?, notes = ?
		WHERE id = ?`,
		key.Name, key.Status, key.AllowedModels, key.AllowedScopes, key.RateLimitPerHour,
		key.TotalRequests, key.TotalInputTokens, key.TotalOutputTokens, key.TotalTokens,
		key.TotalCost, key.BillingCurrency, key.BudgetLimit, key.AlertThreshold, key.LastUsedAt, key.ExpiresAt, key.UpdatedAt, key.Notes, key.ID,
	)
	return err
}

func (db *DB) TouchAIAPIKeyUsage(id string, usedAt time.Time, inputTokens, outputTokens, totalTokens int, cost float64, currency string) error {
	if currency == "" {
		currency = "CNY"
	}
	_, err := db.conn.Exec(`
		UPDATE ai_api_keys
		SET total_requests = total_requests + 1,
			total_input_tokens = total_input_tokens + ?,
			total_output_tokens = total_output_tokens + ?,
			total_tokens = total_tokens + ?,
			total_cost = total_cost + ?,
			billing_currency = ?,
			last_used_at = ?, updated_at = ?
		WHERE id = ?`,
		inputTokens, outputTokens, totalTokens, cost, currency, usedAt, usedAt, id,
	)
	return err
}

func (db *DB) CreateAIAPIRequestLog(log *AIAPIRequestLog) error {
	if log == nil {
		return errors.New("log is nil")
	}
	log.CreatedAt = time.Now()
	if log.Currency == "" {
		log.Currency = "CNY"
	}
	success := 0
	if log.Success {
		success = 1
	}
	result, err := db.conn.Exec(`
		INSERT INTO ai_api_request_logs (
			api_key_id, model, provider, endpoint, request_type, status_code, success,
			error_message, request_body, response_body, client_ip, latency_ms, input_tokens,
			output_tokens, total_tokens, estimated_cost, currency, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.APIKeyID, log.Model, log.Provider, log.Endpoint, log.RequestType, log.StatusCode,
		success, log.ErrorMessage, log.RequestBody, log.ResponseBody, log.ClientIP,
		log.LatencyMS, log.InputTokens, log.OutputTokens, log.TotalTokens, log.EstimatedCost, log.Currency, log.CreatedAt,
	)
	if err != nil {
		return err
	}
	log.ID, _ = result.LastInsertId()
	return nil
}

func (db *DB) ListAIAPIRequestLogs(apiKeyID string, limit, offset int) ([]*AIAPIRequestLog, error) {
	query := `
		SELECT id, api_key_id, model, provider, endpoint, request_type, status_code, success,
			error_message, request_body, response_body, client_ip, latency_ms, input_tokens,
			output_tokens, total_tokens, estimated_cost, currency, created_at
		FROM ai_api_request_logs
		WHERE 1 = 1`
	args := make([]interface{}, 0, 3)
	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*AIAPIRequestLog, 0)
	for rows.Next() {
		item := &AIAPIRequestLog{}
		var success int
		if err := rows.Scan(
			&item.ID, &item.APIKeyID, &item.Model, &item.Provider, &item.Endpoint, &item.RequestType,
			&item.StatusCode, &success, &item.ErrorMessage, &item.RequestBody, &item.ResponseBody,
			&item.ClientIP, &item.LatencyMS, &item.InputTokens, &item.OutputTokens,
			&item.TotalTokens, &item.EstimatedCost, &item.Currency, &item.CreatedAt,
		); err != nil {
			continue
		}
		item.Success = success == 1
		logs = append(logs, item)
	}
	return logs, nil
}

func (db *DB) CountAIAPIRequestsSince(apiKeyID string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM ai_api_request_logs
		WHERE api_key_id = ? AND created_at >= ?`,
		apiKeyID, since,
	).Scan(&count)
	return count, err
}

func (db *DB) CleanOldAIAPIRequestLogs(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	// 排除 internal:* 记录（这些记录永不过期）
	result, err := db.conn.Exec(
		`DELETE FROM ai_api_request_logs WHERE created_at < ? AND api_key_id NOT LIKE 'internal:%'`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ListInternalRequestLogs 查询内部（免 API Key）请求流水，按时间倒序
func (db *DB) ListInternalRequestLogs(keyID string, limit, offset int) ([]*AIAPIRequestLog, error) {
	rows, err := db.conn.Query(`
		SELECT id, api_key_id, model, provider, endpoint, request_type, status_code, success,
			error_message, request_body, response_body, client_ip, latency_ms, input_tokens,
			output_tokens, total_tokens, estimated_cost, currency, created_at
		FROM ai_api_request_logs
		WHERE api_key_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		keyID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := make([]*AIAPIRequestLog, 0)
	for rows.Next() {
		item := &AIAPIRequestLog{}
		var success int
		if err := rows.Scan(
			&item.ID, &item.APIKeyID, &item.Model, &item.Provider, &item.Endpoint, &item.RequestType,
			&item.StatusCode, &success, &item.ErrorMessage, &item.RequestBody, &item.ResponseBody,
			&item.ClientIP, &item.LatencyMS, &item.InputTokens, &item.OutputTokens,
			&item.TotalTokens, &item.EstimatedCost, &item.Currency, &item.CreatedAt,
		); err != nil {
			continue
		}
		item.Success = success == 1
		logs = append(logs, item)
	}
	return logs, nil
}

// CountInternalRequestLogs 统计内部请求总数
func (db *DB) CountInternalRequestLogs(keyID string) (int, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM ai_api_request_logs WHERE api_key_id = ?`, keyID,
	).Scan(&count)
	return count, err
}

func (db *DB) scanAIAPIKey(query string, args ...interface{}) (*AIAPIKey, error) {
	row := db.conn.QueryRow(query, args...)
	return scanAIAPIKeyScanner(row)
}

func scanAIAPIKeyRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*AIAPIKey, error) {
	return scanAIAPIKeyScanner(scanner)
}

func scanAIAPIKeyScanner(scanner interface {
	Scan(dest ...interface{}) error
}) (*AIAPIKey, error) {
	item := &AIAPIKey{}
	var lastUsedAt sql.NullTime
	var expiresAt sql.NullTime
	err := scanner.Scan(
		&item.ID, &item.Name, &item.KeyPrefix, &item.KeyHash, &item.Status, &item.AllowedModels,
		&item.AllowedScopes, &item.RateLimitPerHour, &item.TotalRequests, &item.TotalInputTokens,
		&item.TotalOutputTokens, &item.TotalTokens, &item.TotalCost, &item.BillingCurrency, &item.BudgetLimit, &item.AlertThreshold,
		&lastUsedAt, &expiresAt, &item.CreatedAt, &item.UpdatedAt, &item.CreatorIP, &item.Notes,
	)
	if err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		item.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	return item, nil
}

func (db *DB) GetAIUsageReport(groupBy, apiKeyID string, since time.Time) ([]*AIUsageReportRow, error) {
	periodExpr := "strftime('%Y-%m-%d', created_at)"
	if groupBy == "month" {
		periodExpr = "strftime('%Y-%m', created_at)"
	}
	query := `
		SELECT ` + periodExpr + ` AS period, api_key_id, provider, model,
			COUNT(*) AS request_count,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(estimated_cost), 0),
			COALESCE(MAX(currency), 'CNY')
		FROM ai_api_request_logs
		WHERE created_at >= ?`
	args := []interface{}{since}
	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}
	query += ` GROUP BY period, api_key_id, provider, model ORDER BY period DESC, request_count DESC`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*AIUsageReportRow, 0)
	for rows.Next() {
		item := &AIUsageReportRow{}
		if err := rows.Scan(&item.Period, &item.APIKeyID, &item.Provider, &item.Model, &item.RequestCount, &item.InputTokens, &item.OutputTokens, &item.TotalTokens, &item.TotalCost, &item.Currency); err != nil {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

// MiniMaxMediaTask MiniMax 媒体生成异步任务
type MiniMaxMediaTask struct {
	ID              string     `json:"id"`
	APIKeyID        string     `json:"api_key_id"`
	Model           string     `json:"model"`
	Provider        string     `json:"provider"` // minimax
	Status          string     `json:"status"`   // pending/running/succeeded/failed
	RequestBody     string     `json:"request_body"`
	ResultJSON      string     `json:"result_json,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	ExternalTaskID  string     `json:"external_task_id"` // MiniMax 返回的任务ID
	ClientIP        string     `json:"client_ip"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// InitMiniMaxMediaTasks 初始化 MiniMax 媒体任务表
func (db *DB) InitMiniMaxMediaTasks() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS minimax_media_tasks (
			id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			model TEXT NOT NULL,
			provider TEXT DEFAULT 'minimax',
			status TEXT DEFAULT 'pending',
			request_body TEXT DEFAULT '',
			result_json TEXT DEFAULT '',
			error_message TEXT DEFAULT '',
			external_task_id TEXT DEFAULT '',
			client_ip TEXT DEFAULT '',
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_minimax_tasks_api_key ON minimax_media_tasks(api_key_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_minimax_tasks_status ON minimax_media_tasks(status, created_at DESC);
	`)
	return err
}

// CreateMiniMaxMediaTask 创建 MiniMax 媒体任务
func (db *DB) CreateMiniMaxMediaTask(task *MiniMaxMediaTask) error {
	task.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO minimax_media_tasks (id, api_key_id, model, provider, status, request_body, result_json, error_message, external_task_id, client_ip, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.APIKeyID, task.Model, task.Provider, task.Status, task.RequestBody, task.ResultJSON, task.ErrorMessage, task.ExternalTaskID, task.ClientIP, task.CreatedAt)
	return err
}

// GetMiniMaxMediaTask 获取 MiniMax 媒体任务
func (db *DB) GetMiniMaxMediaTask(id string) (*MiniMaxMediaTask, error) {
	task := &MiniMaxMediaTask{}
	err := db.conn.QueryRow(`
		SELECT id, api_key_id, model, provider, status, request_body, result_json, error_message, external_task_id, client_ip, completed_at, created_at
		FROM minimax_media_tasks WHERE id = ?
	`, id).Scan(&task.ID, &task.APIKeyID, &task.Model, &task.Provider, &task.Status, &task.RequestBody, &task.ResultJSON, &task.ErrorMessage, &task.ExternalTaskID, &task.ClientIP, &task.CompletedAt, &task.CreatedAt)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// GetMiniMaxMediaTaskByAPIKey 获取指定 API Key 的 MiniMax 媒体任务
func (db *DB) GetMiniMaxMediaTaskByAPIKey(id string) (*MiniMaxMediaTask, error) {
	return db.GetMiniMaxMediaTask(id)
}

// UpdateMiniMaxMediaTask 更新 MiniMax 媒体任务
func (db *DB) UpdateMiniMaxMediaTask(task *MiniMaxMediaTask) error {
	_, err := db.conn.Exec(`
		UPDATE minimax_media_tasks SET status=?, result_json=?, error_message=?, external_task_id=?, completed_at=? WHERE id=?
	`, task.Status, task.ResultJSON, task.ErrorMessage, task.ExternalTaskID, task.CompletedAt, task.ID)
	return err
}

// ListMiniMaxMediaTasks 列出 MiniMax 媒体任务
func (db *DB) ListMiniMaxMediaTasks(apiKeyID string, limit, offset int) ([]*MiniMaxMediaTask, error) {
	query := `
		SELECT id, api_key_id, model, provider, status, request_body, result_json, error_message, external_task_id, client_ip, completed_at, created_at
		FROM minimax_media_tasks WHERE 1=1`
	args := make([]interface{}, 0)

	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]*MiniMaxMediaTask, 0)
	for rows.Next() {
		task := &MiniMaxMediaTask{}
		if err := rows.Scan(&task.ID, &task.APIKeyID, &task.Model, &task.Provider, &task.Status, &task.RequestBody, &task.ResultJSON, &task.ErrorMessage, &task.ExternalTaskID, &task.ClientIP, &task.CompletedAt, &task.CreatedAt); err != nil {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// CountMiniMaxMediaTasks 统计 MiniMax 媒体任务数量
func (db *DB) CountMiniMaxMediaTasks(apiKeyID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM minimax_media_tasks`
	args := []interface{}{}
	if apiKeyID != "" {
		query += ` WHERE api_key_id = ?`
		args = append(args, apiKeyID)
	}
	err := db.conn.QueryRow(query, args...).Scan(&count)
	return count, err
}

// VoiceClone 音色克隆记录
type VoiceClone struct {
	ID        uint   `json:"id"`
	APIKeyID  string `json:"api_key_id"`
	VoiceID   string `json:"voice_id"`   // MiniMax 返回的 voice_id
	Name      string `json:"name"`      // 用户定义的音色名称
	Status    string `json:"status"`     // pending/active/failed
	CreatedAt string `json:"created_at"`
}

// InitVoiceClones 初始化音色克隆表
func (db *DB) InitVoiceClones() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS voice_clones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id TEXT NOT NULL,
			voice_id TEXT NOT NULL,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_voice_clones_api_key ON voice_clones(api_key_id, created_at DESC);
	`)
	return err
}

// CreateVoiceClone 创建音色克隆记录
func (db *DB) CreateVoiceClone(clone *VoiceClone) error {
	_, err := db.conn.Exec(`
		INSERT INTO voice_clones (api_key_id, voice_id, name, status, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
	`, clone.APIKeyID, clone.VoiceID, clone.Name, clone.Status)
	return err
}

// GetVoiceClone 获取音色克隆记录
func (db *DB) GetVoiceClone(id uint) (*VoiceClone, error) {
	clone := &VoiceClone{}
	err := db.conn.QueryRow(`
		SELECT id, api_key_id, voice_id, name, status, created_at
		FROM voice_clones WHERE id = ?
	`, id).Scan(&clone.ID, &clone.APIKeyID, &clone.VoiceID, &clone.Name, &clone.Status, &clone.CreatedAt)
	if err != nil {
		return nil, err
	}
	return clone, nil
}

// GetVoiceCloneByVoiceID 根据 voice_id 获取音色克隆记录
func (db *DB) GetVoiceCloneByVoiceID(voiceID string) (*VoiceClone, error) {
	clone := &VoiceClone{}
	err := db.conn.QueryRow(`
		SELECT id, api_key_id, voice_id, name, status, created_at
		FROM voice_clones WHERE voice_id = ?
	`, voiceID).Scan(&clone.ID, &clone.APIKeyID, &clone.VoiceID, &clone.Name, &clone.Status, &clone.CreatedAt)
	if err != nil {
		return nil, err
	}
	return clone, nil
}

// DeleteVoiceClone 删除音色克隆记录
func (db *DB) DeleteVoiceClone(id uint, apiKeyID string) error {
	_, err := db.conn.Exec(`DELETE FROM voice_clones WHERE id = ? AND api_key_id = ?`, id, apiKeyID)
	return err
}

// ListVoiceClones 列出音色克隆记录
func (db *DB) ListVoiceClones(apiKeyID string, limit, offset int) ([]*VoiceClone, error) {
	query := `
		SELECT id, api_key_id, voice_id, name, status, created_at
		FROM voice_clones WHERE 1=1`
	args := make([]interface{}, 0)

	if apiKeyID != "" {
		query += ` AND api_key_id = ?`
		args = append(args, apiKeyID)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clones := make([]*VoiceClone, 0)
	for rows.Next() {
		clone := &VoiceClone{}
		if err := rows.Scan(&clone.ID, &clone.APIKeyID, &clone.VoiceID, &clone.Name, &clone.Status, &clone.CreatedAt); err != nil {
			return nil, err
		}
		clones = append(clones, clone)
	}
	return clones, nil
}

// UpdateVoiceCloneStatus 更新音色克隆状态
func (db *DB) UpdateVoiceCloneStatus(id uint, status string) error {
	_, err := db.conn.Exec(`UPDATE voice_clones SET status = ? WHERE id = ?`, status, id)
	return err
}

// AnthropicProvider DB 存储的 Anthropic 下游提供商
type AnthropicProvider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	APIURL    string `json:"api_url"`
	APIKey    string `json:"api_key"`
	Models    string `json:"models"`   // JSON array
	Aliases   string `json:"aliases"`  // JSON array of {model, upstream_model}
	Enabled     bool   `json:"enabled"`
	IsDefault   bool   `json:"is_default"`
	DefaultModel string `json:"default_model"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (db *DB) InitAnthropicProviders() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS anthropic_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			api_url TEXT NOT NULL,
			api_key TEXT NOT NULL DEFAULT '',
			models TEXT NOT NULL DEFAULT '[]',
			aliases TEXT NOT NULL DEFAULT '[]',
			enabled INTEGER NOT NULL DEFAULT 1,
			is_default INTEGER NOT NULL DEFAULT 0,
			default_model TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_anthropic_providers_name ON anthropic_providers(name);
	`)
	if err != nil {
		return err
	}
	// Migration: add default_model column for existing tables
	db.conn.Exec(`ALTER TABLE anthropic_providers ADD COLUMN default_model TEXT NOT NULL DEFAULT ''`)
	// Seed builtin providers if table is empty
	var count int
	db.conn.QueryRow(`SELECT COUNT(*) FROM anthropic_providers`).Scan(&count)
	if count == 0 {
		builtins := []struct {
			Name         string
			APIURL       string
			APIKey       string
			Models       string
			Aliases      string
			IsDefault    bool
			DefaultModel string
		}{
			{"MiniMax", "https://api.minimaxi.com/anthropic", "", `["MiniMax-M2.5","MiniMax-M2.5-highspeed","MiniMax-M2.1","MiniMax-M2.1-highspeed","MiniMax-M2","MiniMax-M2.7"]`, "[]", false, "MiniMax-M2.5"},
			{"DashScope", "https://coding.dashscope.aliyuncs.com/apps/anthropic", "", `["qwen3.5-plus","qwen3-max-2026-01-23","qwen3-coder-next","qwen3-coder-plus","glm-5","glm-4.7","kimi-k2.5","MiniMax-M2.5"]`, "[]", false, "qwen3.5-plus"},
			{"DeepSeek", "https://api.deepseek.com/anthropic", "", `["deepseek-chat","deepseek-reasoner","deepseek-v4-flash","deepseek-v4-pro"]`, "[]", true, "deepseek-v4-pro"},
			{"PackyAPI", "https://www.packyapi.com", "", `["claude-opus-4-7","claude-sonnet-4-6","claude-haiku-4-5-20251001","claude-sonnet-4-5"]`, "[]", false, "claude-sonnet-4-6"},
			{"OpenClaudeCode", "https://www.openclaudecode.cn", "", `["claude-opus-4-7","claude-sonnet-4-6","claude-haiku-4-5-20251001","claude-sonnet-4-5"]`, "[]", false, "claude-sonnet-4-6"},
		}
		for _, b := range builtins {
			db.conn.Exec(`INSERT OR IGNORE INTO anthropic_providers (name, api_url, api_key, models, aliases, is_default, default_model) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				b.Name, b.APIURL, b.APIKey, b.Models, b.Aliases, b.IsDefault, b.DefaultModel)
		}
	}
	return nil
}

// ListAnthropicProviders 列出所有 providers
func (db *DB) ListAnthropicProviders() ([]*AnthropicProvider, error) {
	return db.listAnthropicProviders("")
}

// ListEnabledAnthropicProviders 列出启用的 providers
func (db *DB) ListEnabledAnthropicProviders() ([]*AnthropicProvider, error) {
	return db.listAnthropicProviders("WHERE enabled = 1")
}

// GetDefaultAnthropicProvider 获取默认线路
func (db *DB) GetDefaultAnthropicProvider() (*AnthropicProvider, error) {
	return scanAnthropicProvider(db.conn.QueryRow(`SELECT `+anthropicProviderColumns+` FROM anthropic_providers WHERE enabled = 1 AND is_default = 1 LIMIT 1`))
}

func (db *DB) listAnthropicProviders(where string) ([]*AnthropicProvider, error) {
	query := `SELECT id, name, api_url, api_key, models, aliases, enabled, is_default, created_at, updated_at FROM anthropic_providers `
	if where != "" {
		query += where
	}
	query += ` ORDER BY id ASC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := make([]*AnthropicProvider, 0)
	for rows.Next() {
		p := &AnthropicProvider{}
		if err := rows.Scan(&p.ID, &p.Name, &p.APIURL, &p.APIKey, &p.Models, &p.Aliases, &p.Enabled, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, nil
}

var anthropicProviderColumns = `id, name, api_url, api_key, models, aliases, enabled, is_default, default_model, created_at, updated_at`

func scanAnthropicProvider(scanner interface{ Scan(dest ...interface{}) error }) (*AnthropicProvider, error) {
	p := &AnthropicProvider{}
	err := scanner.Scan(&p.ID, &p.Name, &p.APIURL, &p.APIKey, &p.Models, &p.Aliases, &p.Enabled, &p.IsDefault, &p.DefaultModel, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// GetAnthropicProviderByID 按 ID 获取
func (db *DB) GetAnthropicProviderByID(id int64) (*AnthropicProvider, error) {
	return scanAnthropicProvider(db.conn.QueryRow(`SELECT `+anthropicProviderColumns+` FROM anthropic_providers WHERE id = ?`, id))
}

// GetAnthropicProviderByName 按名称获取（用于路由查找）
func (db *DB) GetAnthropicProviderByName(name string) (*AnthropicProvider, error) {
	return scanAnthropicProvider(db.conn.QueryRow(`SELECT `+anthropicProviderColumns+` FROM anthropic_providers WHERE name = ? AND enabled = 1`, name))
}

// CreateAnthropicProvider 新增
func (db *DB) CreateAnthropicProvider(p *AnthropicProvider) error {
	res, err := db.conn.Exec(`INSERT INTO anthropic_providers (name, api_url, api_key, models, aliases, enabled, is_default, default_model) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.APIURL, p.APIKey, p.Models, p.Aliases, p.Enabled, p.IsDefault, p.DefaultModel)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	p.ID = id
	return nil
}

// UpdateAnthropicProvider 更新
func (db *DB) UpdateAnthropicProvider(p *AnthropicProvider) error {
	_, err := db.conn.Exec(`UPDATE anthropic_providers SET name=?, api_url=?, api_key=?, models=?, aliases=?, enabled=?, is_default=?, default_model=?, updated_at=datetime('now') WHERE id=?`,
		p.Name, p.APIURL, p.APIKey, p.Models, p.Aliases, p.Enabled, p.IsDefault, p.DefaultModel, p.ID)
	return err
}

// SetDefaultAnthropicProvider 设置默认线路（先清除所有，再设置指定）
func (db *DB) SetDefaultAnthropicProvider(id int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE anthropic_providers SET is_default = 0`); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE anthropic_providers SET is_default = 1, updated_at = datetime('now') WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteAnthropicProvider 删除
func (db *DB) DeleteAnthropicProvider(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM anthropic_providers WHERE id = ?`, id)
	return err
}
