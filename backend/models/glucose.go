package models

import (
	"database/sql"
	"fmt"
	"time"
)

// GlucoseProfile 血糖档案主表
type GlucoseProfile struct {
	ID            string     `json:"id"`
	Password      string     `json:"-"`
	PasswordIndex string     `json:"-"`
	CreatorKey    string     `json:"-"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatorIP     string     `json:"-"`
}

// GlucoseRecord 血糖记录表
type GlucoseRecord struct {
	ID          string    `json:"id"`
	ProfileID   string    `json:"profile_id"`
	Value       float64   `json:"value"`        // 血糖值 (mmol/L)
	MeasureType string    `json:"measure_type"` // fasting, 1h, 2h, random, bedtime, dawn
	Time        time.Time `json:"time"`         // 测量时间
	Note        string    `json:"note"`
	VoiceText   string    `json:"voice_text"` // 语音识别原文
	Tags        string    `json:"tags"` // comma-separated
	Food        string    `json:"food"` // 饮食记录
	Exercise    string    `json:"exercise"`
	Medication  string    `json:"medication"`
	Sleep       string    `json:"sleep"`
	Mood        string    `json:"mood"`
	CreatedAt   time.Time `json:"created_at"`
}

// GlucoseRecordHistory 血糖记录变更历史表
type GlucoseRecordHistory struct {
	ID           string    `json:"id"`
	RecordID     string    `json:"record_id"`
	ProfileID    string    `json:"profile_id"`
	Action       string    `json:"action"`        // create, update, delete
	FieldName    string    `json:"field_name"`    // 变更的字段名
	OldValue     string    `json:"old_value"`     // 旧值
	NewValue     string    `json:"new_value"`     // 新值
	ChangeDesc   string    `json:"change_desc"`   // 变更描述
	IPAddress    string    `json:"ip_address"`     // 操作者IP
	CreatedAt    time.Time `json:"created_at"`
}

func (db *DB) InitGlucose() error {
	// 档案主表
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS glucose_profiles (
			id TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			password_index TEXT NOT NULL DEFAULT '',
			creator_key TEXT NOT NULL,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			creator_ip TEXT
		)
	`)
	if err != nil {
		return err
	}

	// 血糖记录表
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS glucose_records (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			value REAL NOT NULL,
			measure_type TEXT NOT NULL DEFAULT 'random',
			time DATETIME NOT NULL,
			note TEXT DEFAULT '',
			voice_text TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			food TEXT DEFAULT '',
			exercise TEXT DEFAULT '',
			medication TEXT DEFAULT '',
			sleep TEXT DEFAULT '',
			mood TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES glucose_profiles(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_glucose_records_profile ON glucose_records(profile_id)`)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_glucose_records_time ON glucose_records(time)`)
	if err != nil {
		return err
	}

	// 数据库迁移：添加 voice_text 列
	db.conn.Exec(`ALTER TABLE glucose_records ADD COLUMN voice_text TEXT DEFAULT ''`)

	// 血糖记录变更历史表
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS glucose_record_history (
			id TEXT PRIMARY KEY,
			record_id TEXT NOT NULL,
			profile_id TEXT NOT NULL,
			action TEXT NOT NULL,
			field_name TEXT DEFAULT '',
			old_value TEXT DEFAULT '',
			new_value TEXT DEFAULT '',
			change_desc TEXT DEFAULT '',
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 创建历史记录索引
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_glucose_history_record ON glucose_record_history(record_id)`)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_glucose_history_profile ON glucose_record_history(profile_id)`)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_glucose_history_time ON glucose_record_history(created_at)`)
	if err != nil {
		return err
	}

	return nil
}

// CreateGlucoseProfile 创建血糖档案
func (db *DB) CreateGlucoseProfile(profile *GlucoseProfile) error {
	profile.ID = GenerateHexKey(8)
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO glucose_profiles (id, password, password_index, creator_key, expires_at, created_at, updated_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Password, profile.PasswordIndex, profile.CreatorKey, profile.ExpiresAt, profile.CreatedAt, profile.UpdatedAt, profile.CreatorIP)

	return err
}

// GetGlucoseProfile 获取血糖档案
func (db *DB) GetGlucoseProfile(id string) (*GlucoseProfile, error) {
	var profile GlucoseProfile
	err := db.conn.QueryRow(`
		SELECT id, password, password_index, creator_key, expires_at, created_at, updated_at, creator_ip
		FROM glucose_profiles WHERE id = ?
	`, id).Scan(&profile.ID, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey,
		&profile.ExpiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// GetGlucoseProfileByPasswordIndex 通过密码索引获取档案
func (db *DB) GetGlucoseProfileByPasswordIndex(passwordIndex string) (*GlucoseProfile, error) {
	var profile GlucoseProfile
	err := db.conn.QueryRow(`
		SELECT id, password, password_index, creator_key, expires_at, created_at, updated_at, creator_ip
		FROM glucose_profiles WHERE password_index = ?
	`, passwordIndex).Scan(&profile.ID, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey,
		&profile.ExpiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

// DeleteGlucoseProfile 删除血糖档案
func (db *DB) DeleteGlucoseProfile(id string) error {
	_, err := db.conn.Exec(`DELETE FROM glucose_profiles WHERE id = ?`, id)
	return err
}

// ExtendGlucoseProfile 延长档案过期时间
func (db *DB) ExtendGlucoseProfile(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(`UPDATE glucose_profiles SET expires_at = ?, updated_at = ? WHERE id = ?`,
		expiresAt, time.Now(), id)
	return err
}

// CountGlucoseProfilesByIP 统计IP创建的档案数
func (db *DB) CountGlucoseProfilesByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM glucose_profiles WHERE creator_ip = ? AND created_at > ?`,
		ip, since).Scan(&count)
	return count, err
}

// CreateGlucoseRecord 创建血糖记录
func (db *DB) CreateGlucoseRecord(record *GlucoseRecord) error {
	record.ID = GenerateHexKey(8)
	record.CreatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO glucose_records (id, profile_id, value, measure_type, time, note, voice_text, tags, food, exercise, medication, sleep, mood, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.ProfileID, record.Value, record.MeasureType, record.Time, record.Note, record.VoiceText,
		record.Tags, record.Food, record.Exercise, record.Medication, record.Sleep, record.Mood, record.CreatedAt)

	return err
}

// GetGlucoseRecords 获取血糖记录列表
func (db *DB) GetGlucoseRecords(profileID string, startDate, endDate string) ([]*GlucoseRecord, error) {
	query := `SELECT id, profile_id, value, measure_type, time, note, voice_text, tags, food, exercise, medication, sleep, mood, created_at
		FROM glucose_records WHERE profile_id = ?`
	args := []interface{}{profileID}

	if startDate != "" {
		query += " AND time >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND time <= ?"
		args = append(args, endDate)
	}

	query += " ORDER BY time DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*GlucoseRecord
	for rows.Next() {
		var r GlucoseRecord
		err := rows.Scan(&r.ID, &r.ProfileID, &r.Value, &r.MeasureType, &r.Time, &r.Note, &r.VoiceText,
			&r.Tags, &r.Food, &r.Exercise, &r.Medication, &r.Sleep, &r.Mood, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		records = append(records, &r)
	}

	return records, nil
}

// GetGlucoseRecord 获取单条血糖记录
func (db *DB) GetGlucoseRecord(id string) (*GlucoseRecord, error) {
	var r GlucoseRecord
	err := db.conn.QueryRow(`
		SELECT id, profile_id, value, measure_type, time, note, voice_text, tags, food, exercise, medication, sleep, mood, created_at
		FROM glucose_records WHERE id = ?
	`, id).Scan(&r.ID, &r.ProfileID, &r.Value, &r.MeasureType, &r.Time, &r.Note, &r.VoiceText,
		&r.Tags, &r.Food, &r.Exercise, &r.Medication, &r.Sleep, &r.Mood, &r.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// UpdateGlucoseRecord 更新血糖记录
func (db *DB) UpdateGlucoseRecord(record *GlucoseRecord) error {
	_, err := db.conn.Exec(`
		UPDATE glucose_records SET value = ?, measure_type = ?, time = ?, note = ?, voice_text = ?, tags = ?,
		food = ?, exercise = ?, medication = ?, sleep = ?, mood = ? WHERE id = ?
	`, record.Value, record.MeasureType, record.Time, record.Note, record.VoiceText, record.Tags,
		record.Food, record.Exercise, record.Medication, record.Sleep, record.Mood, record.ID)
	return err
}

// DeleteGlucoseRecord 删除血糖记录
func (db *DB) DeleteGlucoseRecord(id string) error {
	_, err := db.conn.Exec(`DELETE FROM glucose_records WHERE id = ?`, id)
	return err
}

// BatchCreateGlucoseRecords 批量创建血糖记录
func (db *DB) BatchCreateGlucoseRecords(records []*GlucoseRecord) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO glucose_records (id, profile_id, value, measure_type, time, note, voice_text, tags, food, exercise, medication, sleep, mood, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		r.ID = GenerateHexKey(8)
		r.CreatedAt = time.Now()
		_, err := stmt.Exec(r.ID, r.ProfileID, r.Value, r.MeasureType, r.Time, r.Note, r.VoiceText,
			r.Tags, r.Food, r.Exercise, r.Medication, r.Sleep, r.Mood, r.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CleanExpiredGlucoseProfiles 清理过期的血糖档案
func (db *DB) CleanExpiredGlucoseProfiles() (int64, error) {
	result, err := db.conn.Exec(`DELETE FROM glucose_profiles WHERE expires_at IS NOT NULL AND expires_at < ?`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetGlucoseStats 获取血糖统计数据
func (db *DB) GetGlucoseStats(profileID string, startDate, endDate string) (*GlucoseStats, error) {
	query := `
		SELECT
			COUNT(*) as total_count,
			AVG(value) as avg_value,
			MAX(value) as max_value,
			MIN(value) as min_value,
			STDDEV(value) as std_value
		FROM glucose_records WHERE profile_id = ?`
	args := []interface{}{profileID}

	if startDate != "" {
		query += " AND time >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND time <= ?"
		args = append(args, endDate)
	}

	var stats GlucoseStats
	err := db.conn.QueryRow(query, args...).Scan(
		&stats.TotalCount, &stats.AvgValue, &stats.MaxValue, &stats.MinValue, &stats.StdValue,
	)
	if err == sql.ErrNoRows {
		return &stats, nil
	}
	if err != nil {
		return nil, err
	}

	// 计算 TIR (Time In Range)
	tirQuery := `
		SELECT COUNT(*) FROM glucose_records WHERE profile_id = ?`
	tirArgs := []interface{}{profileID}

	if startDate != "" {
		tirQuery += " AND time >= ?"
		tirArgs = append(tirArgs, startDate)
	}
	if endDate != "" {
		tirQuery += " AND time <= ?"
		tirArgs = append(tirArgs, endDate)
	}

	// TIR: 空腹 3.9-6.1, 餐后 3.9-7.8
	var totalInRange, totalCount int
	rows, err := db.conn.Query(tirQuery, tirArgs...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			rows.Scan(&totalCount)
		}
	}

	// 计算各范围的数量
	db.conn.QueryRow(`
		SELECT COUNT(*) FROM glucose_records WHERE profile_id = ? AND time >= ? AND time <= ? AND
		((measure_type = 'fasting' AND value >= 3.9 AND value <= 6.1) OR
		 (measure_type IN ('1h', '2h') AND value >= 3.9 AND value <= 7.8) OR
		 (measure_type NOT IN ('fasting', '1h', '2h') AND value >= 3.9 AND value <= 7.8))
	`, profileID, startDate, endDate).Scan(&totalInRange)

	if totalCount > 0 {
		stats.TIR = float64(totalInRange) / float64(totalCount) * 100
	}

	// 低血糖 (<3.9) 和高血糖 (>10.0) 统计
	db.conn.QueryRow(`
		SELECT COUNT(*) FROM glucose_records WHERE profile_id = ? AND time >= ? AND time <= ? AND value < 3.9
	`, profileID, startDate, endDate).Scan(&stats.HypoglycemiaCount)

	db.conn.QueryRow(`
		SELECT COUNT(*) FROM glucose_records WHERE profile_id = ? AND time >= ? AND time <= ? AND value > 10.0
	`, profileID, startDate, endDate).Scan(&stats.HyperglycemiaCount)

	// 按测量类型统计
	typeStats, _ := db.GetGlucoseStatsByType(profileID, startDate, endDate)
	stats.ByMeasureType = typeStats

	return &stats, nil
}

// GlucoseStats 血糖统计数据
type GlucoseStats struct {
	TotalCount       int     `json:"total_count"`
	AvgValue         float64 `json:"avg_value"`
	MaxValue         float64 `json:"max_value"`
	MinValue         float64 `json:"min_value"`
	StdValue         float64 `json:"std_value"`
	TIR              float64 `json:"tir"`               // Time In Range 目标范围内时间占比
	HypoglycemiaCount int    `json:"hypoglycemia_count"` // 低血糖次数
	HyperglycemiaCount int   `json:"hyperglycemia_count"` // 高血糖次数
	ByMeasureType    map[string]TypeStats `json:"by_measure_type"`
}

// TypeStats 按测量类型的统计
type TypeStats struct {
	Count   int     `json:"count"`
	AvgValue float64 `json:"avg_value"`
	MinValue float64 `json:"min_value"`
	MaxValue float64 `json:"max_value"`
}

// GetGlucoseStatsByType 按测量类型统计
func (db *DB) GetGlucoseStatsByType(profileID string, startDate, endDate string) (map[string]TypeStats, error) {
	query := `
		SELECT measure_type, COUNT(*), AVG(value), MIN(value), MAX(value)
		FROM glucose_records WHERE profile_id = ?`
	args := []interface{}{profileID}

	if startDate != "" {
		query += " AND time >= ?"
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND time <= ?"
		args = append(args, endDate)
	}

	query += " GROUP BY measure_type"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]TypeStats)
	for rows.Next() {
		var ts TypeStats
		var measureType string
		err := rows.Scan(&measureType, &ts.Count, &ts.AvgValue, &ts.MinValue, &ts.MaxValue)
		if err != nil {
			return nil, err
		}
		result[measureType] = ts
	}

	return result, nil
}

// GenerateHexKey 生成随机hex key
func GenerateHexKey(length int) string {
	return fmt.Sprintf("%x", time.Now().UnixNano())[:length]
}

// CreateGlucoseRecordHistory 创建血糖记录变更历史
func (db *DB) CreateGlucoseRecordHistory(history *GlucoseRecordHistory) error {
	history.ID = GenerateHexKey(8)
	history.CreatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO glucose_record_history (id, record_id, profile_id, action, field_name, old_value, new_value, change_desc, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, history.ID, history.RecordID, history.ProfileID, history.Action, history.FieldName,
		history.OldValue, history.NewValue, history.ChangeDesc, history.IPAddress, history.CreatedAt)

	return err
}

// GetGlucoseRecordHistory 获取单条记录的所有变更历史
func (db *DB) GetGlucoseRecordHistory(recordID string) ([]*GlucoseRecordHistory, error) {
	rows, err := db.conn.Query(`
		SELECT id, record_id, profile_id, action, field_name, old_value, new_value, change_desc, ip_address, created_at
		FROM glucose_record_history WHERE record_id = ? ORDER BY created_at DESC
	`, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []*GlucoseRecordHistory
	for rows.Next() {
		var h GlucoseRecordHistory
		err := rows.Scan(&h.ID, &h.RecordID, &h.ProfileID, &h.Action, &h.FieldName,
			&h.OldValue, &h.NewValue, &h.ChangeDesc, &h.IPAddress, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		histories = append(histories, &h)
	}

	return histories, nil
}

// GetGlucoseProfileHistory 获取档案的所有变更历史
func (db *DB) GetGlucoseProfileHistory(profileID string, limit int) ([]*GlucoseRecordHistory, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.conn.Query(`
		SELECT id, record_id, profile_id, action, field_name, old_value, new_value, change_desc, ip_address, created_at
		FROM glucose_record_history WHERE profile_id = ? ORDER BY created_at DESC LIMIT ?
	`, profileID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []*GlucoseRecordHistory
	for rows.Next() {
		var h GlucoseRecordHistory
		err := rows.Scan(&h.ID, &h.RecordID, &h.ProfileID, &h.Action, &h.FieldName,
			&h.OldValue, &h.NewValue, &h.ChangeDesc, &h.IPAddress, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		histories = append(histories, &h)
	}

	return histories, nil
}
