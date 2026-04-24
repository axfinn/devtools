package models

import (
	"database/sql"
	"fmt"
	"time"
)

func init() {
	RegisterInit("记账(expenses)", (*DB).InitExpense)
}

// ExpenseProfile 档案主表
type ExpenseProfile struct {
	ID            string     `json:"id"`
	Password      string     `json:"-"`
	PasswordIndex string     `json:"-"`
	CreatorKey    string     `json:"-"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CreatorIP     string     `json:"-"`
}

// ExpenseAccount 账户表
type ExpenseAccount struct {
	ID           string    `json:"id"`
	ProfileID    string    `json:"profile_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // cash, bank, alipay, wechat, credit
	Balance      float64   `json:"balance"`
	Color        string    `json:"color"`
	Icon         string    `json:"icon"`
	Sort         int       `json:"sort"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ExpenseCategory 分类表
type ExpenseCategory struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // income, expense
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExpenseTransaction 记账记录表
type ExpenseTransaction struct {
	ID         string     `json:"id"`
	ProfileID  string     `json:"profile_id"`
	AccountID  string     `json:"account_id"`
	CategoryID string     `json:"category_id"`
	Amount     float64    `json:"amount"`
	Type       string     `json:"type"` // income, expense
	Date       string     `json:"date"` // YYYY-MM-DD
	Remark     string     `json:"remark"`
	Tags       string     `json:"tags"` // comma-separated
	VoiceText  string     `json:"voice_text"` // 原始语音文字
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (db *DB) InitExpense() error {
	// 档案主表
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS expense_profiles (
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

	// 账户表
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS expense_accounts (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'cash',
			balance REAL NOT NULL DEFAULT 0,
			color TEXT DEFAULT '#409EFF',
			icon TEXT DEFAULT 'Wallet',
			sort INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES expense_profiles(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 分类表
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS expense_categories (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			icon TEXT DEFAULT 'Folder',
			color TEXT DEFAULT '#909399',
			sort INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES expense_profiles(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 记账记录表
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS expense_transactions (
			id TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			account_id TEXT NOT NULL,
			category_id TEXT NOT NULL,
			amount REAL NOT NULL,
			type TEXT NOT NULL,
			date TEXT NOT NULL,
			remark TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (profile_id) REFERENCES expense_profiles(id) ON DELETE CASCADE,
			FOREIGN KEY (account_id) REFERENCES expense_accounts(id) ON DELETE CASCADE,
			FOREIGN KEY (category_id) REFERENCES expense_categories(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_expense_profiles_expires_at ON expense_profiles(expires_at)")
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_expense_accounts_profile ON expense_accounts(profile_id)")
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_expense_categories_profile ON expense_categories(profile_id)")
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_expense_transactions_profile ON expense_transactions(profile_id)")
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_expense_transactions_date ON expense_transactions(date)")

	// 添加 voice_text 列（如果不存在）
	db.conn.Exec("ALTER TABLE expense_transactions ADD COLUMN voice_text TEXT DEFAULT ''")

	return nil
}

// Profile CRUD

func (db *DB) CreateExpenseProfile(profile *ExpenseProfile) error {
	profile.ID = generateID(8)
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO expense_profiles (id, password, password_index, creator_key, expires_at, created_at, updated_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Password, profile.PasswordIndex, profile.CreatorKey,
		profile.ExpiresAt, profile.CreatedAt, profile.UpdatedAt, profile.CreatorIP)

	return err
}

func (db *DB) GetExpenseProfile(id string) (*ExpenseProfile, error) {
	profile := &ExpenseProfile{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, password, password_index, creator_key, expires_at, created_at, updated_at, creator_ip
		FROM expense_profiles WHERE id = ?
	`, id).Scan(
		&profile.ID, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey,
		&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}

	return profile, nil
}

func (db *DB) GetExpenseProfileByPasswordIndex(passwordIndex string) (*ExpenseProfile, error) {
	profile := &ExpenseProfile{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, password, password_index, creator_key, expires_at, created_at, updated_at, creator_ip
		FROM expense_profiles WHERE password_index = ?
	`, passwordIndex).Scan(
		&profile.ID, &profile.Password, &profile.PasswordIndex, &profile.CreatorKey,
		&expiresAt, &profile.CreatedAt, &profile.UpdatedAt, &profile.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		profile.ExpiresAt = &expiresAt.Time
	}

	return profile, nil
}

func (db *DB) DeleteExpenseProfile(id string) error {
	_, err := db.conn.Exec("DELETE FROM expense_profiles WHERE id = ?", id)
	return err
}

func (db *DB) ExtendExpenseProfile(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(
		"UPDATE expense_profiles SET expires_at = ?, updated_at = ? WHERE id = ?",
		expiresAt, time.Now(), id)
	return err
}

func (db *DB) CountExpenseProfilesByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM expense_profiles WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

// Account CRUD

func (db *DB) CreateExpenseAccount(account *ExpenseAccount) error {
	account.ID = generateID(8)
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO expense_accounts (id, profile_id, name, type, balance, color, icon, sort, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, account.ID, account.ProfileID, account.Name, account.Type, account.Balance,
		account.Color, account.Icon, account.Sort, account.CreatedAt, account.UpdatedAt)

	return err
}

func (db *DB) GetExpenseAccounts(profileID string) ([]*ExpenseAccount, error) {
	rows, err := db.conn.Query(`
		SELECT id, profile_id, name, type, balance, color, icon, sort, created_at, updated_at
		FROM expense_accounts WHERE profile_id = ? ORDER BY sort ASC, created_at ASC
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*ExpenseAccount
	for rows.Next() {
		a := &ExpenseAccount{}
		if err := rows.Scan(&a.ID, &a.ProfileID, &a.Name, &a.Type, &a.Balance,
			&a.Color, &a.Icon, &a.Sort, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (db *DB) GetExpenseAccount(id string) (*ExpenseAccount, error) {
	a := &ExpenseAccount{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, name, type, balance, color, icon, sort, created_at, updated_at
		FROM expense_accounts WHERE id = ?
	`, id).Scan(&a.ID, &a.ProfileID, &a.Name, &a.Type, &a.Balance,
		&a.Color, &a.Icon, &a.Sort, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (db *DB) UpdateExpenseAccount(account *ExpenseAccount) error {
	account.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE expense_accounts SET name = ?, type = ?, balance = ?, color = ?, icon = ?, sort = ?, updated_at = ?
		WHERE id = ?
	`, account.Name, account.Type, account.Balance, account.Color, account.Icon,
		account.Sort, account.UpdatedAt, account.ID)
	return err
}

func (db *DB) DeleteExpenseAccount(id string) error {
	_, err := db.conn.Exec("DELETE FROM expense_accounts WHERE id = ?", id)
	return err
}

func (db *DB) UpdateExpenseAccountBalance(id string, balance float64) error {
	_, err := db.conn.Exec(
		"UPDATE expense_accounts SET balance = ?, updated_at = ? WHERE id = ?",
		balance, time.Now(), id)
	return err
}

// Category CRUD

func (db *DB) CreateExpenseCategory(category *ExpenseCategory) error {
	category.ID = generateID(8)
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO expense_categories (id, profile_id, name, type, icon, color, sort, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, category.ID, category.ProfileID, category.Name, category.Type,
		category.Icon, category.Color, category.Sort, category.CreatedAt, category.UpdatedAt)

	return err
}

func (db *DB) GetExpenseCategories(profileID string) ([]*ExpenseCategory, error) {
	rows, err := db.conn.Query(`
		SELECT id, profile_id, name, type, icon, color, sort, created_at, updated_at
		FROM expense_categories WHERE profile_id = ? ORDER BY sort ASC, created_at ASC
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*ExpenseCategory
	for rows.Next() {
		c := &ExpenseCategory{}
		if err := rows.Scan(&c.ID, &c.ProfileID, &c.Name, &c.Type,
			&c.Icon, &c.Color, &c.Sort, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (db *DB) GetExpenseCategory(id string) (*ExpenseCategory, error) {
	c := &ExpenseCategory{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, name, type, icon, color, sort, created_at, updated_at
		FROM expense_categories WHERE id = ?
	`, id).Scan(&c.ID, &c.ProfileID, &c.Name, &c.Type,
		&c.Icon, &c.Color, &c.Sort, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (db *DB) UpdateExpenseCategory(category *ExpenseCategory) error {
	category.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE expense_categories SET name = ?, type = ?, icon = ?, color = ?, sort = ?, updated_at = ?
		WHERE id = ?
	`, category.Name, category.Type, category.Icon, category.Color,
		category.Sort, category.UpdatedAt, category.ID)
	return err
}

func (db *DB) DeleteExpenseCategory(id string) error {
	_, err := db.conn.Exec("DELETE FROM expense_categories WHERE id = ?", id)
	return err
}

// Transaction CRUD

func (db *DB) CreateExpenseTransaction(tx *ExpenseTransaction) error {
	tx.ID = generateID(8)
	tx.CreatedAt = time.Now()
	tx.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO expense_transactions (id, profile_id, account_id, category_id, amount, type, date, remark, tags, voice_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tx.ID, tx.ProfileID, tx.AccountID, tx.CategoryID, tx.Amount, tx.Type,
		tx.Date, tx.Remark, tx.Tags, tx.VoiceText, tx.CreatedAt, tx.UpdatedAt)

	return err
}

func (db *DB) GetExpenseTransactions(profileID string, startDate, endDate string) ([]*ExpenseTransaction, error) {
	var rows *sql.Rows
	var err error

	if startDate != "" && endDate != "" {
		rows, err = db.conn.Query(`
			SELECT id, profile_id, account_id, category_id, amount, type, date, remark, tags, COALESCE(voice_text,'') as voice_text, created_at, updated_at
			FROM expense_transactions WHERE profile_id = ? AND date >= ? AND date <= ? ORDER BY date DESC, created_at DESC
		`, profileID, startDate, endDate)
	} else {
		rows, err = db.conn.Query(`
			SELECT id, profile_id, account_id, category_id, amount, type, date, remark, tags, COALESCE(voice_text,'') as voice_text, created_at, updated_at
			FROM expense_transactions WHERE profile_id = ? ORDER BY date DESC, created_at DESC
		`, profileID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*ExpenseTransaction
	for rows.Next() {
		t := &ExpenseTransaction{}
		if err := rows.Scan(&t.ID, &t.ProfileID, &t.AccountID, &t.CategoryID, &t.Amount, &t.Type,
			&t.Date, &t.Remark, &t.Tags, &t.VoiceText, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		txs = append(txs, t)
	}
	return txs, nil
}

func (db *DB) GetExpenseTransaction(id string) (*ExpenseTransaction, error) {
	t := &ExpenseTransaction{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, account_id, category_id, amount, type, date, remark, tags, created_at, updated_at
		FROM expense_transactions WHERE id = ?
	`, id).Scan(&t.ID, &t.ProfileID, &t.AccountID, &t.CategoryID, &t.Amount, &t.Type,
		&t.Date, &t.Remark, &t.Tags, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (db *DB) UpdateExpenseTransaction(tx *ExpenseTransaction) error {
	tx.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE expense_transactions SET account_id = ?, category_id = ?, amount = ?, type = ?, date = ?, remark = ?, tags = ?, updated_at = ?
		WHERE id = ?
	`, tx.AccountID, tx.CategoryID, tx.Amount, tx.Type, tx.Date, tx.Remark,
		tx.Tags, tx.UpdatedAt, tx.ID)
	return err
}

func (db *DB) DeleteExpenseTransaction(id string) error {
	_, err := db.conn.Exec("DELETE FROM expense_transactions WHERE id = ?", id)
	return err
}

// Statistics

type ExpenseStats struct {
	TotalIncome   float64                `json:"total_income"`
	TotalExpense  float64                `json:"total_expense"`
	Balance       float64                `json:"balance"`
	ByCategory    map[string]float64     `json:"by_category"`
	ByAccount     map[string]float64    `json:"by_account"`
	ByMonth       map[string]float64    `json:"by_month"`
	TransactionCount int                `json:"transaction_count"`
}

func (db *DB) GetExpenseStats(profileID, period string) (*ExpenseStats, error) {
	// Calculate date range based on period
	now := time.Now()
	today := now.Format("2006-01-02")
	var startDate string

	switch period {
	case "today":
		// Today only
		startDate = today
	case "week":
		// Get start of current week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")
	case "month":
		// Start of this month (1st)
		startDate = fmt.Sprintf("%d-%02d-01", now.Year(), now.Month())
	case "year":
		// Start of this year (Jan 1st)
		startDate = fmt.Sprintf("%d-01-01", now.Year())
	default:
		// Default to this month
		startDate = fmt.Sprintf("%d-%02d-01", now.Year(), now.Month())
	}

	endDate := today

	stats := &ExpenseStats{
		ByCategory: make(map[string]float64),
		ByAccount:  make(map[string]float64),
		ByMonth:    make(map[string]float64),
	}

	// Total income
	err := db.conn.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM expense_transactions
		WHERE profile_id = ? AND type = 'income' AND date >= ? AND date <= ?
	`, profileID, startDate, endDate).Scan(&stats.TotalIncome)
	if err != nil {
		return nil, err
	}

	// Total expense
	err = db.conn.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM expense_transactions
		WHERE profile_id = ? AND type = 'expense' AND date >= ? AND date <= ?
	`, profileID, startDate, endDate).Scan(&stats.TotalExpense)
	if err != nil {
		return nil, err
	}

	stats.Balance = stats.TotalIncome - stats.TotalExpense

	// By category (include both income and expense)
	rows, err := db.conn.Query(`
		SELECT COALESCE(c.name, '未分类'), COALESCE(SUM(t.amount), 0) as total
		FROM expense_transactions t
		LEFT JOIN expense_categories c ON t.category_id = c.id
		WHERE t.profile_id = ? AND t.date >= ? AND t.date <= ?
		GROUP BY t.category_id, c.name
	`, profileID, startDate, endDate)
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var total float64
			if err := rows.Scan(&name, &total); err == nil && total > 0 {
				stats.ByCategory[name] = total
			}
		}
	}

	// By account
	rows, err = db.conn.Query(`
		SELECT COALESCE(a.name, '未分类'), COALESCE(SUM(t.amount), 0) as total
		FROM expense_transactions t
		LEFT JOIN expense_accounts a ON t.account_id = a.id
		WHERE t.profile_id = ? AND t.date >= ? AND t.date <= ?
		GROUP BY t.account_id, a.name
	`, profileID, startDate, endDate)
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var total float64
			if err := rows.Scan(&name, &total); err == nil && total > 0 {
				stats.ByAccount[name] = total
			}
		}
	}

	// By month (last 12 months of expenses only)
	rows, err = db.conn.Query(`
		SELECT substr(date, 1, 7) as month, COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total
		FROM expense_transactions
		WHERE profile_id = ? AND date >= date('now', '-12 months')
		GROUP BY substr(date, 1, 7)
		ORDER BY month ASC
		LIMIT 12
	`, profileID)
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var month string
			var total float64
			if err := rows.Scan(&month, &total); err == nil {
				stats.ByMonth[month] = total
			}
		}
	}

	// Transaction count
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM expense_transactions
		WHERE profile_id = ? AND date >= ? AND date <= ?
	`, profileID, startDate, endDate).Scan(&stats.TransactionCount)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// CleanExpiredExpenseProfiles 清理过期的档案
func (db *DB) CleanExpiredExpenseProfiles() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM expense_profiles
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
