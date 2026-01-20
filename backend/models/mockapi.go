package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

type MockAPI struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Method         string    `json:"method"`
	ResponseBody   string    `json:"response_body"`
	ResponseStatus int       `json:"response_status"`
	ResponseHeaders string   `json:"response_headers"` // JSON string
	ResponseDelay  int       `json:"response_delay"`   // seconds
	ExpiresAt      time.Time `json:"expires_at"`
	MaxCalls       int       `json:"max_calls"`
	CallCount      int       `json:"call_count"`
	CreatedAt      time.Time `json:"created_at"`
	CreatorIP      string    `json:"-"`
	Password       string    `json:"-"`
}

type MockAPILog struct {
	ID          int64     `json:"id"`
	MockAPIID   string    `json:"mock_api_id"`
	Method      string    `json:"method"`
	Headers     string    `json:"headers"`      // JSON string
	QueryParams string    `json:"query_params"` // JSON string
	Body        string    `json:"body"`
	ClientIP    string    `json:"client_ip"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

// InitMockAPI creates the mock_apis and mock_api_logs tables
func (db *DB) InitMockAPI() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS mock_apis (
			id TEXT PRIMARY KEY,
			name TEXT DEFAULT '',
			method TEXT NOT NULL,
			response_body TEXT DEFAULT '',
			response_status INTEGER DEFAULT 200,
			response_headers TEXT DEFAULT '{}',
			response_delay INTEGER DEFAULT 0,
			expires_at DATETIME,
			max_calls INTEGER DEFAULT 1000,
			call_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			creator_ip TEXT,
			password TEXT DEFAULT ''
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS mock_api_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mock_api_id TEXT NOT NULL,
			method TEXT NOT NULL,
			headers TEXT DEFAULT '{}',
			query_params TEXT DEFAULT '{}',
			body TEXT DEFAULT '',
			client_ip TEXT,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (mock_api_id) REFERENCES mock_apis(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_mockapis_expires_at ON mock_apis(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_mockapis_creator_ip ON mock_apis(creator_ip)`,
		`CREATE INDEX IF NOT EXISTS idx_mockapis_created_at ON mock_apis(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_mockapi_logs_mock_api_id ON mock_api_logs(mock_api_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mockapi_logs_created_at ON mock_api_logs(created_at)`,
	}

	for _, idx := range indexes {
		if _, err := db.conn.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

// generateMockAPIID generates a random 8-character hex ID
func generateMockAPIID() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// validateCustomPath validates custom path format
func validateCustomPath(path string) error {
	if len(path) == 0 || len(path) > 32 {
		return errors.New("custom path must be 1-32 characters")
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-_]+$`, path)
	if !matched {
		return errors.New("custom path can only contain letters, numbers, hyphens, and underscores")
	}
	return nil
}

// CreateMockAPI creates a new mock API with random ID
func (db *DB) CreateMockAPI(mockAPI *MockAPI) error {
	return db.CreateMockAPIWithCustomID(mockAPI, "")
}

// CreateMockAPIWithCustomID creates a new mock API with optional custom ID
func (db *DB) CreateMockAPIWithCustomID(mockAPI *MockAPI, customID string) error {
	// Validate response delay
	if mockAPI.ResponseDelay < 0 {
		mockAPI.ResponseDelay = 0
	}
	if mockAPI.ResponseDelay > 30 {
		mockAPI.ResponseDelay = 30
	}

	// Validate response status
	if mockAPI.ResponseStatus < 100 || mockAPI.ResponseStatus > 599 {
		return errors.New("response status must be between 100 and 599")
	}

	// Validate max calls
	if mockAPI.MaxCalls < 1 {
		mockAPI.MaxCalls = 1000
	}
	if mockAPI.MaxCalls > 100000 {
		mockAPI.MaxCalls = 100000
	}

	// Validate response body size (100KB)
	if len(mockAPI.ResponseBody) > 100*1024 {
		return errors.New("response body exceeds 100KB limit")
	}

	// Check storage limit
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM mock_apis").Scan(&count)
	if err != nil {
		return err
	}

	if count >= 1000 {
		// Try to clean expired entries first
		db.CleanExpiredMockAPIs()

		// Check again
		err = db.conn.QueryRow("SELECT COUNT(*) FROM mock_apis").Scan(&count)
		if err != nil {
			return err
		}
		if count >= 1000 {
			return errors.New("storage limit reached (max 1000 mock APIs)")
		}
	}

	var id string
	if customID != "" {
		// Validate custom ID
		if err := validateCustomPath(customID); err != nil {
			return err
		}

		// Check if ID already exists
		var exists bool
		err = db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM mock_apis WHERE id = ?)", customID).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("custom ID already exists")
		}
		id = customID
	} else {
		// Generate unique random ID
		for i := 0; i < 10; i++ {
			id, err = generateMockAPIID()
			if err != nil {
				return err
			}

			// Check if ID already exists
			var exists bool
			err = db.conn.QueryRow("SELECT EXISTS(SELECT 1 FROM mock_apis WHERE id = ?)", id).Scan(&exists)
			if err != nil {
				return err
			}
			if !exists {
				break
			}
		}
	}

	mockAPI.ID = id
	mockAPI.CreatedAt = time.Now()
	mockAPI.CallCount = 0

	// Ensure response_headers is valid JSON
	if mockAPI.ResponseHeaders == "" {
		mockAPI.ResponseHeaders = "{}"
	}

	_, err = db.conn.Exec(`
		INSERT INTO mock_apis (id, name, method, response_body, response_status, response_headers,
			response_delay, expires_at, max_calls, call_count, created_at, creator_ip, password)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mockAPI.ID, mockAPI.Name, mockAPI.Method, mockAPI.ResponseBody, mockAPI.ResponseStatus,
		mockAPI.ResponseHeaders, mockAPI.ResponseDelay, mockAPI.ExpiresAt, mockAPI.MaxCalls,
		mockAPI.CallCount, mockAPI.CreatedAt, mockAPI.CreatorIP, mockAPI.Password)

	return err
}

// GetMockAPI retrieves a mock API by ID
func (db *DB) GetMockAPI(id string) (*MockAPI, error) {
	mockAPI := &MockAPI{}
	err := db.conn.QueryRow(`
		SELECT id, name, method, response_body, response_status, response_headers, response_delay,
			expires_at, max_calls, call_count, created_at, creator_ip, password
		FROM mock_apis
		WHERE id = ?`, id).Scan(
		&mockAPI.ID, &mockAPI.Name, &mockAPI.Method, &mockAPI.ResponseBody, &mockAPI.ResponseStatus,
		&mockAPI.ResponseHeaders, &mockAPI.ResponseDelay, &mockAPI.ExpiresAt, &mockAPI.MaxCalls,
		&mockAPI.CallCount, &mockAPI.CreatedAt, &mockAPI.CreatorIP, &mockAPI.Password)

	if err == sql.ErrNoRows {
		return nil, errors.New("mock API not found")
	}
	if err != nil {
		return nil, err
	}

	return mockAPI, nil
}

// UpdateMockAPI updates a mock API configuration
func (db *DB) UpdateMockAPI(id string, updates map[string]interface{}) error {
	// Build dynamic update query
	validFields := map[string]bool{
		"response_body":    true,
		"response_status":  true,
		"response_headers": true,
		"response_delay":   true,
	}

	query := "UPDATE mock_apis SET "
	args := []interface{}{}
	first := true

	for field, value := range updates {
		if !validFields[field] {
			continue
		}
		if !first {
			query += ", "
		}
		query += field + " = ?"
		args = append(args, value)
		first = false
	}

	if len(args) == 0 {
		return errors.New("no valid fields to update")
	}

	query += " WHERE id = ?"
	args = append(args, id)

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("mock API not found")
	}

	return nil
}

// DeleteMockAPI deletes a mock API and its logs
func (db *DB) DeleteMockAPI(id string) error {
	result, err := db.conn.Exec("DELETE FROM mock_apis WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("mock API not found")
	}

	return nil
}

// IncrementCallCount atomically increments the call count
func (db *DB) IncrementCallCount(id string) error {
	result, err := db.conn.Exec("UPDATE mock_apis SET call_count = call_count + 1 WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("mock API not found")
	}

	return nil
}

// LogMockAPIRequest logs a request to a mock API
func (db *DB) LogMockAPIRequest(log *MockAPILog) error {
	// Truncate body to 10KB
	if len(log.Body) > 10*1024 {
		log.Body = log.Body[:10*1024]
	}

	log.CreatedAt = time.Now()

	// Ensure JSON fields are valid
	if log.Headers == "" {
		log.Headers = "{}"
	}
	if log.QueryParams == "" {
		log.QueryParams = "{}"
	}

	result, err := db.conn.Exec(`
		INSERT INTO mock_api_logs (mock_api_id, method, headers, query_params, body, client_ip, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.MockAPIID, log.Method, log.Headers, log.QueryParams, log.Body, log.ClientIP, log.UserAgent, log.CreatedAt)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

// GetMockAPILogs retrieves logs for a mock API (last 100)
func (db *DB) GetMockAPILogs(mockAPIID string, limit int) ([]*MockAPILog, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT id, mock_api_id, method, headers, query_params, body, client_ip, user_agent, created_at
		FROM mock_api_logs
		WHERE mock_api_id = ?
		ORDER BY created_at DESC
		LIMIT ?`, mockAPIID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*MockAPILog
	for rows.Next() {
		log := &MockAPILog{}
		err := rows.Scan(&log.ID, &log.MockAPIID, &log.Method, &log.Headers, &log.QueryParams,
			&log.Body, &log.ClientIP, &log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// CleanOldLogs removes logs beyond the 100 most recent for each mock API
func (db *DB) CleanOldLogs(mockAPIID string) error {
	_, err := db.conn.Exec(`
		DELETE FROM mock_api_logs
		WHERE mock_api_id = ?
		AND id NOT IN (
			SELECT id FROM mock_api_logs
			WHERE mock_api_id = ?
			ORDER BY created_at DESC
			LIMIT 100
		)`, mockAPIID, mockAPIID)
	return err
}

// CleanExpiredMockAPIs removes expired mock APIs
func (db *DB) CleanExpiredMockAPIs() error {
	_, err := db.conn.Exec("DELETE FROM mock_apis WHERE expires_at < ?", time.Now())
	return err
}

// CountMockAPIsByIP counts mock APIs created by IP in the given duration
func (db *DB) CountMockAPIsByIP(ip string, duration time.Duration) (int, error) {
	var count int
	since := time.Now().Add(-duration)
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM mock_apis WHERE creator_ip = ? AND created_at > ?",
		ip, since).Scan(&count)
	return count, err
}

// TotalMockAPIsCount returns total count of mock APIs
func (db *DB) TotalMockAPIsCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM mock_apis").Scan(&count)
	return count, err
}

// ParseHeadersJSON parses the response_headers JSON string into a map
func ParseHeadersJSON(headersJSON string) (map[string]string, error) {
	headers := make(map[string]string)
	if headersJSON == "" || headersJSON == "{}" {
		return headers, nil
	}
	err := json.Unmarshal([]byte(headersJSON), &headers)
	return headers, err
}

// SerializeHeadersJSON serializes headers map to JSON string
func SerializeHeadersJSON(headers map[string]string) (string, error) {
	if len(headers) == 0 {
		return "{}", nil
	}
	bytes, err := json.Marshal(headers)
	if err != nil {
		return "{}", err
	}
	return string(bytes), nil
}
