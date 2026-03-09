package models

import (
	"strconv"
	"time"
)

// HouseholdItem 家庭物品
type HouseholdItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	Quantity     int       `json:"quantity"`
	Unit         string    `json:"unit"`
	MinQuantity  int       `json:"min_quantity"`
	ExpiryDate   string    `json:"expiry_date"`
	ExpiryDays   int       `json:"expiry_days"`
	Location     string    `json:"location"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ItemTemplate 物品模板
type ItemTemplate struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Category             string `json:"category"`
	Unit                 string `json:"unit"`
	DefaultMinQuantity   int    `json:"default_min_quantity"`
	DefaultExpiryDays    int    `json:"default_expiry_days"`
	IsDefault            bool   `json:"is_default"`
}

// Notification 提醒通知
type Notification struct {
	ID        string    `json:"id"`
	ItemID    string    `json:"item_id"`
	ItemName  string    `json:"item_name"`
	Type      string    `json:"type"` // low_stock, expiring, expired
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// HouseholdTodo 待购买/提醒任务
type HouseholdTodo struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"` // open, done
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HouseholdLocation 档案位置库
type HouseholdLocation struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// HouseholdSpaceLayout 3D 空间布局
type HouseholdSpaceLayout struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// HouseholdSpaceShare 3D 空间分享
type HouseholdSpaceShare struct {
	ID        string    `json:"id"`
	ProfileID string    `json:"profile_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// HouseholdProfile 家庭物品档案
type HouseholdProfile struct {
	ID              string     `json:"id"`
	PasswordIndex   string     `json:"-"`
	CreatorKey      string     `json:"creator_key"`
	Name            string     `json:"name"`
	ExpiresAt       *time.Time `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// InitHousehold 初始化家庭物品数据库表
func (db *DB) InitHousehold() error {
	query := `
	CREATE TABLE IF NOT EXISTS household_items (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		quantity INTEGER DEFAULT 1,
		unit TEXT DEFAULT '个',
		min_quantity INTEGER DEFAULT 1,
		expiry_date TEXT,
		expiry_days INTEGER,
		location TEXT,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_household_category ON household_items(category);

	CREATE TABLE IF NOT EXISTS item_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		unit TEXT DEFAULT '个',
		default_min_quantity INTEGER DEFAULT 1,
		default_expiry_days INTEGER,
		is_default INTEGER DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS household_notifications (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		type TEXT NOT NULL,
		message TEXT NOT NULL,
		is_read INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (item_id) REFERENCES household_items(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_notifications_unread ON household_notifications(is_read, created_at);

	-- 家庭物品档案
	CREATE TABLE IF NOT EXISTS household_profiles (
		id TEXT PRIMARY KEY,
		password_index TEXT NOT NULL UNIQUE,
		creator_key TEXT NOT NULL,
		name TEXT NOT NULL,
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_household_profiles_password ON household_profiles(password_index);

	-- 档案物品（关联档案）
	CREATE TABLE IF NOT EXISTS household_profile_items (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		quantity INTEGER DEFAULT 1,
		unit TEXT DEFAULT '个',
		min_quantity INTEGER DEFAULT 1,
		expiry_date TEXT,
		expiry_days INTEGER,
		location TEXT,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES household_profiles(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_profile_items_category ON household_profile_items(profile_id, category);

	-- 档案通知
	CREATE TABLE IF NOT EXISTS household_profile_notifications (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		item_id TEXT NOT NULL,
		type TEXT NOT NULL,
		message TEXT NOT NULL,
		is_read INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES household_profiles(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS household_todos (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		reason TEXT,
		status TEXT DEFAULT 'open',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES household_profiles(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_household_todos_profile ON household_todos(profile_id, status, created_at);

	CREATE TABLE IF NOT EXISTS household_locations (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES household_profiles(id) ON DELETE CASCADE
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_household_locations_unique ON household_locations(profile_id, name);

	CREATE TABLE IF NOT EXISTS household_space_layouts (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL UNIQUE,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES household_profiles(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS household_space_shares (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES household_profiles(id) ON DELETE CASCADE
	);
	`
	_, err := db.conn.Exec(query)
	if err != nil {
		return err
	}

	// 初始化默认模板
	db.initDefaultTemplates()

	return nil
}

// initDefaultTemplates 初始化默认物品模板
func (db *DB) initDefaultTemplates() error {
	// 检查是否已有模板
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM item_templates").Scan(&count)
	if count > 0 {
		return nil
	}

	templates := []ItemTemplate{
		// 厨房
		{Name: "洗洁精", Category: "厨房", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 730},
		{Name: "洗衣液", Category: "厨房", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 1095},
		{Name: "垃圾袋", Category: "厨房", Unit: "卷", DefaultMinQuantity: 1},
		{Name: "保鲜膜", Category: "厨房", Unit: "卷", DefaultMinQuantity: 1, DefaultExpiryDays: 1825},
		{Name: "保鲜袋", Category: "厨房", Unit: "卷", DefaultMinQuantity: 1},
		{Name: "盐", Category: "厨房", Unit: "袋", DefaultMinQuantity: 1, DefaultExpiryDays: 1825},
		{Name: "糖", Category: "厨房", Unit: "袋", DefaultMinQuantity: 1, DefaultExpiryDays: 1825},
		{Name: "酱油", Category: "厨房", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 730},
		{Name: "醋", Category: "厨房", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 730},
		{Name: "食用油", Category: "厨房", Unit: "桶", DefaultMinQuantity: 1, DefaultExpiryDays: 545},

		// 卫生间
		{Name: "洗发水", Category: "卫生间", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 1095},
		{Name: "沐浴露", Category: "卫生间", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 1095},
		{Name: "牙膏", Category: "卫生间", Unit: "支", DefaultMinQuantity: 1, DefaultExpiryDays: 730},
		{Name: "牙刷", Category: "卫生间", Unit: "支", DefaultMinQuantity: 2, DefaultExpiryDays: 90},
		{Name: "毛巾", Category: "卫生间", Unit: "条", DefaultMinQuantity: 2},
		{Name: "纸巾", Category: "卫生间", Unit: "包", DefaultMinQuantity: 2},
		{Name: "卫生巾", Category: "卫生间", Unit: "包", DefaultMinQuantity: 1, DefaultExpiryDays: 1095},
		{Name: "洗手液", Category: "卫生间", Unit: "瓶", DefaultMinQuantity: 1, DefaultExpiryDays: 730},

		// 卧室
		{Name: "床单", Category: "卧室", Unit: "条", DefaultMinQuantity: 2},
		{Name: "被套", Category: "卧室", Unit: "条", DefaultMinQuantity: 1},
		{Name: "枕头", Category: "卧室", Unit: "个", DefaultMinQuantity: 2},
		{Name: "衣架", Category: "卧室", Unit: "个", DefaultMinQuantity: 10},

		// 客厅
		{Name: "茶叶", Category: "客厅", Unit: "盒", DefaultMinQuantity: 1, DefaultExpiryDays: 730},
		{Name: "咖啡", Category: "客厅", Unit: "盒", DefaultMinQuantity: 1, DefaultExpiryDays: 545},
		{Name: "零食", Category: "客厅", Unit: "包", DefaultMinQuantity: 3},
		{Name: "饮料", Category: "客厅", Unit: "瓶", DefaultMinQuantity: 4},

		// 玄关
		{Name: "鞋套", Category: "玄关", Unit: "包", DefaultMinQuantity: 1},
		{Name: "拖鞋", Category: "玄关", Unit: "双", DefaultMinQuantity: 2},

		// 阳台
		{Name: "花盆", Category: "阳台", Unit: "个", DefaultMinQuantity: 1},
		{Name: "园艺土", Category: "阳台", Unit: "袋", DefaultMinQuantity: 1},

		// 通用
		{Name: "创可贴", Category: "其他", Unit: "片", DefaultMinQuantity: 10, DefaultExpiryDays: 1825},
		{Name: "药品", Category: "其他", Unit: "盒", DefaultMinQuantity: 1},
		{Name: "电池", Category: "其他", Unit: "节", DefaultMinQuantity: 4},
		{Name: "灯泡", Category: "其他", Unit: "个", DefaultMinQuantity: 2},
		{Name: "抹布", Category: "厨房", Unit: "条", DefaultMinQuantity: 3},
	}

	for _, t := range templates {
		t.ID = generateID(8)
		t.IsDefault = true
		_, err := db.conn.Exec(`
			INSERT INTO item_templates (id, name, category, unit, default_min_quantity, default_expiry_days, is_default)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, t.ID, t.Name, t.Category, t.Unit, t.DefaultMinQuantity, t.DefaultExpiryDays, 1)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateHouseholdItem 创建物品
func (db *DB) CreateHouseholdItem(item *HouseholdItem) error {
	item.ID = generateID(8)
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	if item.Quantity == 0 {
		item.Quantity = 1
	}
	if item.Unit == "" {
		item.Unit = "个"
	}
	if item.MinQuantity == 0 {
		item.MinQuantity = 1
	}

	_, err := db.conn.Exec(`
		INSERT INTO household_items (id, name, category, quantity, unit, min_quantity, expiry_date, expiry_days, location, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Name, item.Category, item.Quantity, item.Unit, item.MinQuantity,
		item.ExpiryDate, item.ExpiryDays, item.Location, item.Notes, item.CreatedAt, item.UpdatedAt)

	return err
}

// GetHouseholdItem 获取物品详情
func (db *DB) GetHouseholdItem(id string) (*HouseholdItem, error) {
	item := &HouseholdItem{}
	err := db.conn.QueryRow(`
		SELECT id, name, category, quantity, unit, min_quantity, COALESCE(expiry_date, ''), COALESCE(expiry_days, 0), COALESCE(location, ''), COALESCE(notes, ''), created_at, updated_at
		FROM household_items WHERE id = ?
	`, id).Scan(&item.ID, &item.Name, &item.Category, &item.Quantity, &item.Unit,
		&item.MinQuantity, &item.ExpiryDate, &item.ExpiryDays, &item.Location,
		&item.Notes, &item.CreatedAt, &item.UpdatedAt)

	return item, err
}

// UpdateHouseholdItem 更新物品
func (db *DB) UpdateHouseholdItem(item *HouseholdItem) error {
	item.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE household_items SET name = ?, category = ?, quantity = ?, unit = ?, min_quantity = ?, expiry_date = ?, expiry_days = ?, location = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, item.Name, item.Category, item.Quantity, item.Unit, item.MinQuantity,
		item.ExpiryDate, item.ExpiryDays, item.Location, item.Notes, item.UpdatedAt, item.ID)

	return err
}

// DeleteHouseholdItem 删除物品
func (db *DB) DeleteHouseholdItem(id string) error {
	// 先删除相关通知
	db.conn.Exec("DELETE FROM household_notifications WHERE item_id = ?", id)
	_, err := db.conn.Exec("DELETE FROM household_items WHERE id = ?", id)
	return err
}

// GetAllHouseholdItems 获取所有物品
func (db *DB) GetAllHouseholdItems() ([]*HouseholdItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, category, quantity, unit, min_quantity, COALESCE(expiry_date, ''), COALESCE(expiry_days, 0), COALESCE(location, ''), COALESCE(notes, ''), created_at, updated_at
		FROM household_items
		ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*HouseholdItem
	for rows.Next() {
		item := &HouseholdItem{}
		err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Quantity, &item.Unit,
			&item.MinQuantity, &item.ExpiryDate, &item.ExpiryDays, &item.Location,
			&item.Notes, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// UseHouseholdItem 使用物品（减少数量）
func (db *DB) UseHouseholdItem(id string, amount int) error {
	item, err := db.GetHouseholdItem(id)
	if err != nil {
		return err
	}

	newQuantity := item.Quantity - amount
	if newQuantity < 0 {
		newQuantity = 0
	}

	_, err = db.conn.Exec("UPDATE household_items SET quantity = ?, updated_at = ? WHERE id = ?", newQuantity, time.Now(), id)
	return err
}

// RestockHouseholdItem 补充物品（增加数量）
func (db *DB) RestockHouseholdItem(id string, amount int) error {
	item, err := db.GetHouseholdItem(id)
	if err != nil {
		return err
	}

	newQuantity := item.Quantity + amount

	_, err = db.conn.Exec("UPDATE household_items SET quantity = ?, updated_at = ? WHERE id = ?", newQuantity, time.Now(), id)
	return err
}

// OpenHouseholdItem 重新开封（重置保质期）
func (db *DB) OpenHouseholdItem(id string) error {
	expiryDate := time.Now().Format("2006-01-02")
	_, err := db.conn.Exec("UPDATE household_items SET expiry_date = ?, updated_at = ? WHERE id = ?", expiryDate, time.Now(), id)
	return err
}

// GetItemTemplates 获取物品模板
func (db *DB) GetItemTemplates() ([]*ItemTemplate, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, category, unit, default_min_quantity, default_expiry_days, is_default
		FROM item_templates
		ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*ItemTemplate
	for rows.Next() {
		t := &ItemTemplate{}
		var isDefault int
		err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Unit, &t.DefaultMinQuantity, &t.DefaultExpiryDays, &isDefault)
		if err != nil {
			continue
		}
		t.IsDefault = isDefault == 1
		templates = append(templates, t)
	}

	return templates, nil
}

// GetHouseholdAlerts 获取需要关注的物品（库存不足、即将过期、已过期）
func (db *DB) GetHouseholdAlerts() ([]*HouseholdItem, error) {
	items, err := db.GetAllHouseholdItems()
	if err != nil {
		return nil, err
	}

	var alerts []*HouseholdItem
	now := time.Now()

	for _, item := range items {
		isAlert := false

		// 检查库存不足
		if item.Quantity <= item.MinQuantity {
			isAlert = true
		}

		// 检查保质期
		if item.ExpiryDate != "" && item.ExpiryDays > 0 {
			parsedDate, err := time.Parse("2006-01-02", item.ExpiryDate)
			if err == nil {
				expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
				daysUntilExpiry := int(expiryTime.Sub(now).Hours() / 24)

				if daysUntilExpiry <= 0 {
					// 已过期
					isAlert = true
				} else if daysUntilExpiry <= 7 {
					// 即将过期
					isAlert = true
				}
			}
		}

		if isAlert {
			alerts = append(alerts, item)
		}
	}

	return alerts, nil
}

// GenerateHouseholdNotifications 生成提醒
func (db *DB) GenerateHouseholdNotifications() error {
	// 先删除旧的通知
	db.conn.Exec("DELETE FROM household_notifications")

	items, err := db.GetAllHouseholdItems()
	if err != nil {
		return err
	}

	now := time.Now()

	for _, item := range items {
		// 库存不足提醒
		if item.Quantity <= item.MinQuantity {
			notif := &Notification{
				ID:        generateID(8),
				ItemID:    item.ID,
				ItemName:  item.Name,
				Type:      "low_stock",
				Message:   item.Name + " 库存不足，请补充",
				IsRead:    false,
				CreatedAt: now,
			}
			db.conn.Exec(`
				INSERT INTO household_notifications (id, item_id, type, message, is_read, created_at)
				VALUES (?, ?, ?, ?, ?, ?)
			`, notif.ID, notif.ItemID, notif.Type, notif.Message, 0, notif.CreatedAt)
		}

		// 保质期提醒
		if item.ExpiryDate != "" && item.ExpiryDays > 0 {
			parsedDate, err := time.Parse("2006-01-02", item.ExpiryDate)
			if err == nil {
				expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
				daysUntilExpiry := int(expiryTime.Sub(now).Hours() / 24)

				if daysUntilExpiry <= 0 {
					// 已过期
					notif := &Notification{
						ID:        generateID(8),
						ItemID:    item.ID,
						ItemName:  item.Name,
						Type:      "expired",
						Message:   item.Name + " 已过期，请及时处理",
						IsRead:    false,
						CreatedAt: now,
					}
					db.conn.Exec(`
						INSERT INTO household_notifications (id, item_id, type, message, is_read, created_at)
						VALUES (?, ?, ?, ?, ?, ?)
					`, notif.ID, notif.ItemID, notif.Type, notif.Message, 0, notif.CreatedAt)
				} else if daysUntilExpiry <= 7 {
					// 即将过期
					notif := &Notification{
						ID:        generateID(8),
						ItemID:    item.ID,
						ItemName:  item.Name,
						Type:      "expiring",
						Message:   item.Name + " 即将过期（还剩 " + strconv.Itoa(daysUntilExpiry) + " 天）",
						IsRead:    false,
						CreatedAt: now,
					}
					db.conn.Exec(`
						INSERT INTO household_notifications (id, item_id, type, message, is_read, created_at)
						VALUES (?, ?, ?, ?, ?, ?)
					`, notif.ID, notif.ItemID, notif.Type, notif.Message, 0, notif.CreatedAt)
				}
			}
		}
	}

	return nil
}

// GetNotifications 获取通知列表
func (db *DB) GetNotifications(unreadOnly bool) ([]*Notification, error) {
	query := `
		SELECT id, item_id, type, message, is_read, created_at
		FROM household_notifications
	`
	if unreadOnly {
		query += " WHERE is_read = 0"
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []*Notification
	for rows.Next() {
		n := &Notification{}
		var isRead int
		err := rows.Scan(&n.ID, &n.ItemID, &n.Type, &n.Message, &isRead, &n.CreatedAt)
		if err != nil {
			continue
		}
		n.IsRead = isRead == 1

		// 获取物品名称
		var name string
		db.conn.QueryRow("SELECT name FROM household_items WHERE id = ?", n.ItemID).Scan(&name)
		n.ItemName = name

		notifs = append(notifs, n)
	}

	return notifs, nil
}

// MarkNotificationAsRead 标记通知为已读
func (db *DB) MarkNotificationAsRead(id string) error {
	_, err := db.conn.Exec("UPDATE household_notifications SET is_read = 1 WHERE id = ?", id)
	return err
}

// GetUnreadNotificationCount 获取未读通知数量
func (db *DB) GetUnreadNotificationCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM household_notifications WHERE is_read = 0").Scan(&count)
	return count, err
}

// GetHouseholdCategories 获取所有物品分类
func (db *DB) GetHouseholdCategories() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT category FROM household_items ORDER BY category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			continue
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

// GetHouseholdLocations 获取所有物品位置
func (db *DB) GetHouseholdLocations() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT location FROM household_items WHERE location != '' ORDER BY location")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []string
	for rows.Next() {
		var loc string
		if err := rows.Scan(&loc); err != nil {
			continue
		}
		locations = append(locations, loc)
	}
	return locations, nil
}

// GetProfileLocations 获取档案物品位置
func (db *DB) GetProfileLocations(profileID string) ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT location
		FROM household_profile_items
		WHERE profile_id = ? AND location != ''
		ORDER BY location
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []string
	for rows.Next() {
		var loc string
		if err := rows.Scan(&loc); err != nil {
			continue
		}
		locations = append(locations, loc)
	}
	return locations, nil
}

// CreateHouseholdLocation 添加位置
func (db *DB) CreateHouseholdLocation(loc *HouseholdLocation) error {
	loc.ID = generateID(8)
	loc.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT OR IGNORE INTO household_locations (id, profile_id, name, created_at)
		VALUES (?, ?, ?, ?)
	`, loc.ID, loc.ProfileID, loc.Name, loc.CreatedAt)
	return err
}

// GetHouseholdLocationsLibrary 获取位置库
func (db *DB) GetHouseholdLocationsLibrary(profileID string) ([]*HouseholdLocation, error) {
	rows, err := db.conn.Query(`
		SELECT id, profile_id, name, created_at
		FROM household_locations
		WHERE profile_id = ?
		ORDER BY name
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []*HouseholdLocation
	for rows.Next() {
		loc := &HouseholdLocation{}
		if err := rows.Scan(&loc.ID, &loc.ProfileID, &loc.Name, &loc.CreatedAt); err != nil {
			continue
		}
		locations = append(locations, loc)
	}
	return locations, nil
}

// UpdateHouseholdLocation 更新位置
func (db *DB) UpdateHouseholdLocation(id, name string) error {
	_, err := db.conn.Exec(`
		UPDATE household_locations SET name = ?
		WHERE id = ?
	`, name, id)
	return err
}

// DeleteHouseholdLocation 删除位置
func (db *DB) DeleteHouseholdLocation(id string) error {
	_, err := db.conn.Exec("DELETE FROM household_locations WHERE id = ?", id)
	return err
}

// GetHouseholdSpaceLayout 获取空间布局
func (db *DB) GetHouseholdSpaceLayout(profileID string) (*HouseholdSpaceLayout, error) {
	layout := &HouseholdSpaceLayout{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, content, created_at, updated_at
		FROM household_space_layouts WHERE profile_id = ?
	`, profileID).Scan(&layout.ID, &layout.ProfileID, &layout.Content, &layout.CreatedAt, &layout.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return layout, nil
}

// UpsertHouseholdSpaceLayout 保存空间布局
func (db *DB) UpsertHouseholdSpaceLayout(layout *HouseholdSpaceLayout) error {
	if layout.ID == "" {
		layout.ID = generateID(8)
	}
	now := time.Now()
	if layout.CreatedAt.IsZero() {
		layout.CreatedAt = now
	}
	layout.UpdatedAt = now
	_, err := db.conn.Exec(`
		INSERT INTO household_space_layouts (id, profile_id, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(profile_id) DO UPDATE SET
		  content = excluded.content,
		  updated_at = excluded.updated_at
	`, layout.ID, layout.ProfileID, layout.Content, layout.CreatedAt, layout.UpdatedAt)
	return err
}

// CreateHouseholdSpaceShare 创建空间分享
func (db *DB) CreateHouseholdSpaceShare(share *HouseholdSpaceShare) error {
	if share.ID == "" {
		share.ID = generateID(8)
	}
	share.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO household_space_shares (id, profile_id, content, created_at)
		VALUES (?, ?, ?, ?)
	`, share.ID, share.ProfileID, share.Content, share.CreatedAt)
	return err
}

// GetHouseholdSpaceShare 获取空间分享
func (db *DB) GetHouseholdSpaceShare(id string) (*HouseholdSpaceShare, error) {
	share := &HouseholdSpaceShare{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, content, created_at
		FROM household_space_shares WHERE id = ?
	`, id).Scan(&share.ID, &share.ProfileID, &share.Content, &share.CreatedAt)
	if err != nil {
		return nil, err
	}
	return share, nil
}

// CreateItemTemplate 创建物品模板
func (db *DB) CreateItemTemplate(tpl *ItemTemplate) error {
	tpl.ID = generateID(8)

	_, err := db.conn.Exec(`
		INSERT INTO item_templates (id, name, category, unit, default_min_quantity, default_expiry_days, is_default)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, tpl.ID, tpl.Name, tpl.Category, tpl.Unit, tpl.DefaultMinQuantity, tpl.DefaultExpiryDays, 0)

	return err
}

// DeleteItemTemplate 删除物品模板
func (db *DB) DeleteItemTemplate(id string) error {
	_, err := db.conn.Exec("DELETE FROM item_templates WHERE id = ?", id)
	return err
}

// MarkAllNotificationsAsRead 标记所有通知为已读
func (db *DB) MarkAllNotificationsAsRead() error {
	_, err := db.conn.Exec("UPDATE household_notifications SET is_read = 1")
	return err
}

// ========== 档案相关操作 ==========

// CreateHouseholdProfile 创建档案
func (db *DB) CreateHouseholdProfile(profile *HouseholdProfile, expiresInDays int) error {
	profile.ID = generateID(8)
	profile.CreatorKey = generateID(8)
	profile.CreatedAt = time.Now()
	if expiresInDays > 0 {
		profile.ExpiresAt = &time.Time{}
		*profile.ExpiresAt = time.Now().AddDate(0, 0, expiresInDays)
	}

	_, err := db.conn.Exec(`
		INSERT INTO household_profiles (id, password_index, creator_key, name, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.PasswordIndex, profile.CreatorKey, profile.Name, profile.ExpiresAt, profile.CreatedAt)

	return err
}

// GetHouseholdProfile 获取档案
func (db *DB) GetHouseholdProfile(id string) (*HouseholdProfile, error) {
	profile := &HouseholdProfile{}
	err := db.conn.QueryRow(`
		SELECT id, password_index, creator_key, name, expires_at, created_at
		FROM household_profiles WHERE id = ?
	`, id).Scan(&profile.ID, &profile.PasswordIndex, &profile.CreatorKey, &profile.Name, &profile.ExpiresAt, &profile.CreatedAt)

	return profile, err
}

// GetHouseholdProfileByPasswordIndex 通过密码索引获取档案
func (db *DB) GetHouseholdProfileByPasswordIndex(passwordIndex string) (*HouseholdProfile, error) {
	profile := &HouseholdProfile{}
	err := db.conn.QueryRow(`
		SELECT id, password_index, creator_key, name, expires_at, created_at
		FROM household_profiles WHERE password_index = ?
	`, passwordIndex).Scan(&profile.ID, &profile.PasswordIndex, &profile.CreatorKey, &profile.Name, &profile.ExpiresAt, &profile.CreatedAt)

	if err != nil {
		return nil, err
	}
	return profile, nil
}

// DeleteHouseholdProfile 删除档案
func (db *DB) DeleteHouseholdProfile(id string) error {
	_, err := db.conn.Exec("DELETE FROM household_profiles WHERE id = ?", id)
	return err
}

// ExtendHouseholdProfile 延长档案过期时间
func (db *DB) ExtendHouseholdProfile(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec("UPDATE household_profiles SET expires_at = ? WHERE id = ?", expiresAt, id)
	return err
}

// CleanExpiredHouseholdProfiles 清理过期的档案
func (db *DB) CleanExpiredHouseholdProfiles() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM household_profiles
		WHERE expires_at IS NOT NULL AND expires_at < datetime('now')
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ========== 待购买/提醒任务 ==========

// CreateHouseholdTodo 创建待办
func (db *DB) CreateHouseholdTodo(todo *HouseholdTodo) error {
	if todo.ID == "" {
		todo.ID = generateID(8)
	}
	if todo.Status == "" {
		todo.Status = "open"
	}
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	_, err := db.conn.Exec(`
		INSERT INTO household_todos (id, profile_id, name, category, reason, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, todo.ID, todo.ProfileID, todo.Name, todo.Category, todo.Reason, todo.Status, todo.CreatedAt, todo.UpdatedAt)
	return err
}

// GetHouseholdTodo 获取单个待办
func (db *DB) GetHouseholdTodo(id string) (*HouseholdTodo, error) {
	todo := &HouseholdTodo{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, name, category, reason, status, created_at, updated_at
		FROM household_todos WHERE id = ?
	`, id).Scan(&todo.ID, &todo.ProfileID, &todo.Name, &todo.Category, &todo.Reason, &todo.Status, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return todo, nil
}

// GetHouseholdTodos 获取待办列表
func (db *DB) GetHouseholdTodos(profileID, status string) ([]*HouseholdTodo, error) {
	args := []interface{}{profileID}
	query := `
		SELECT id, profile_id, name, category, reason, status, created_at, updated_at
		FROM household_todos
		WHERE profile_id = ?
	`
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []*HouseholdTodo
	for rows.Next() {
		todo := &HouseholdTodo{}
		if err := rows.Scan(&todo.ID, &todo.ProfileID, &todo.Name, &todo.Category, &todo.Reason, &todo.Status, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			continue
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

// UpdateHouseholdTodoStatus 更新待办状态
func (db *DB) UpdateHouseholdTodoStatus(id, status string) error {
	_, err := db.conn.Exec(`
		UPDATE household_todos SET status = ?, updated_at = ?
		WHERE id = ?
	`, status, time.Now(), id)
	return err
}

// UpdateHouseholdTodo 更新待办内容
func (db *DB) UpdateHouseholdTodo(todo *HouseholdTodo) error {
	_, err := db.conn.Exec(`
		UPDATE household_todos SET name = ?, category = ?, reason = ?, updated_at = ?
		WHERE id = ?
	`, todo.Name, todo.Category, todo.Reason, time.Now(), todo.ID)
	return err
}

// DeleteHouseholdTodo 删除待办
func (db *DB) DeleteHouseholdTodo(id string) error {
	_, err := db.conn.Exec("DELETE FROM household_todos WHERE id = ?", id)
	return err
}

// ========== 档案物品操作 ==========

// ProfileItem 档案物品
type ProfileItem struct {
	ID          string    `json:"id"`
	ProfileID   string    `json:"profile_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Quantity    int       `json:"quantity"`
	Unit        string    `json:"unit"`
	MinQuantity int       `json:"min_quantity"`
	ExpiryDate  string    `json:"expiry_date"`
	ExpiryDays  int       `json:"expiry_days"`
	Location    string    `json:"location"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateProfileItem 创建档案物品
func (db *DB) CreateProfileItem(item *ProfileItem) error {
	item.ID = generateID(8)
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	if item.Quantity == 0 {
		item.Quantity = 1
	}
	if item.Unit == "" {
		item.Unit = "个"
	}
	if item.MinQuantity == 0 {
		item.MinQuantity = 1
	}

	_, err := db.conn.Exec(`
		INSERT INTO household_profile_items (id, profile_id, name, category, quantity, unit, min_quantity, expiry_date, expiry_days, location, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ProfileID, item.Name, item.Category, item.Quantity, item.Unit, item.MinQuantity,
		item.ExpiryDate, item.ExpiryDays, item.Location, item.Notes, item.CreatedAt, item.UpdatedAt)

	return err
}

// GetProfileItems 获取档案物品列表
func (db *DB) GetProfileItems(profileID string) ([]*ProfileItem, error) {
	rows, err := db.conn.Query(`
		SELECT id, profile_id, name, category, quantity, unit, min_quantity, COALESCE(expiry_date, ''), COALESCE(expiry_days, 0), COALESCE(location, ''), COALESCE(notes, ''), created_at, updated_at
		FROM household_profile_items
		WHERE profile_id = ?
		ORDER BY category, name
	`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*ProfileItem
	for rows.Next() {
		item := &ProfileItem{}
		err := rows.Scan(&item.ID, &item.ProfileID, &item.Name, &item.Category, &item.Quantity, &item.Unit,
			&item.MinQuantity, &item.ExpiryDate, &item.ExpiryDays, &item.Location,
			&item.Notes, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// GetProfileItem 获取档案物品
func (db *DB) GetProfileItem(id string) (*ProfileItem, error) {
	item := &ProfileItem{}
	err := db.conn.QueryRow(`
		SELECT id, profile_id, name, category, quantity, unit, min_quantity, COALESCE(expiry_date, ''), COALESCE(expiry_days, 0), COALESCE(location, ''), COALESCE(notes, ''), created_at, updated_at
		FROM household_profile_items WHERE id = ?
	`, id).Scan(&item.ID, &item.ProfileID, &item.Name, &item.Category, &item.Quantity, &item.Unit,
		&item.MinQuantity, &item.ExpiryDate, &item.ExpiryDays, &item.Location,
		&item.Notes, &item.CreatedAt, &item.UpdatedAt)

	return item, err
}

// UpdateProfileItem 更新档案物品
func (db *DB) UpdateProfileItem(item *ProfileItem) error {
	item.UpdatedAt = time.Now()
	_, err := db.conn.Exec(`
		UPDATE household_profile_items SET name = ?, category = ?, quantity = ?, unit = ?, min_quantity = ?, expiry_date = ?, expiry_days = ?, location = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, item.Name, item.Category, item.Quantity, item.Unit, item.MinQuantity,
		item.ExpiryDate, item.ExpiryDays, item.Location, item.Notes, item.UpdatedAt, item.ID)

	return err
}

// DeleteProfileItem 删除档案物品
func (db *DB) DeleteProfileItem(id string) error {
	_, err := db.conn.Exec("DELETE FROM household_profile_items WHERE id = ?", id)
	return err
}

// UseProfileItem 使用物品
func (db *DB) UseProfileItem(id string, amount int) error {
	item, err := db.GetProfileItem(id)
	if err != nil {
		return err
	}

	newQuantity := item.Quantity - amount
	if newQuantity < 0 {
		newQuantity = 0
	}

	_, err = db.conn.Exec("UPDATE household_profile_items SET quantity = ?, updated_at = ? WHERE id = ?", newQuantity, time.Now(), id)
	return err
}

// RestockProfileItem 补充物品
func (db *DB) RestockProfileItem(id string, amount int) error {
	item, err := db.GetProfileItem(id)
	if err != nil {
		return err
	}

	newQuantity := item.Quantity + amount
	_, err = db.conn.Exec("UPDATE household_profile_items SET quantity = ?, updated_at = ? WHERE id = ?", newQuantity, time.Now(), id)
	return err
}

// OpenProfileItem 重新开封
func (db *DB) OpenProfileItem(id string) error {
	expiryDate := time.Now().Format("2006-01-02")
	_, err := db.conn.Exec("UPDATE household_profile_items SET expiry_date = ?, updated_at = ? WHERE id = ?", expiryDate, time.Now(), id)
	return err
}

// GetProfileAlerts 获取档案告警物品
func (db *DB) GetProfileAlerts(profileID string) ([]*ProfileItem, error) {
	items, err := db.GetProfileItems(profileID)
	if err != nil {
		return nil, err
	}

	var alerts []*ProfileItem
	now := time.Now()

	for _, item := range items {
		isAlert := false

		if item.Quantity <= item.MinQuantity {
			isAlert = true
		}

		if item.ExpiryDate != "" && item.ExpiryDays > 0 {
			parsedDate, err := time.Parse("2006-01-02", item.ExpiryDate)
			if err == nil {
				expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
				daysUntilExpiry := int(expiryTime.Sub(now).Hours() / 24)

				if daysUntilExpiry <= 0 || daysUntilExpiry <= 7 {
					isAlert = true
				}
			}
		}

		if isAlert {
			alerts = append(alerts, item)
		}
	}

	return alerts, nil
}

// GetProfileStats 获取档案统计
func (db *DB) GetProfileStats(profileID string) (map[string]interface{}, error) {
	items, _ := db.GetProfileItems(profileID)

	total := len(items)
	lowStock := 0
	expiring := 0
	expired := 0

	now := time.Now()

	for _, item := range items {
		if item.Quantity <= item.MinQuantity {
			lowStock++
		}
		if item.ExpiryDate != "" && item.ExpiryDays > 0 {
			parsedDate, err := time.Parse("2006-01-02", item.ExpiryDate)
			if err == nil {
				expiryTime := parsedDate.AddDate(0, 0, item.ExpiryDays)
				daysUntil := int(expiryTime.Sub(now).Hours() / 24)
				if daysUntil <= 0 {
					expired++
				} else if daysUntil <= 7 {
					expiring++
				}
			}
		}
	}

	return map[string]interface{}{
		"total":      total,
		"low_stock":  lowStock,
		"expiring":   expiring,
		"expired":    expired,
	}, nil
}

// ========== 对话相关操作 ==========

// HouseholdConversation 对话记录
type HouseholdConversation struct {
	ID          string    `json:"id"`
	ProfileID   string    `json:"profile_id"`
	UserID     string    `json:"user_id"`
	Role       string    `json:"role"` // user, assistant
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// InitHouseholdConversations 初始化对话表
func (db *DB) InitHouseholdConversations() error {
	query := `
	CREATE TABLE IF NOT EXISTS household_conversations (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_conversations_profile ON household_conversations(profile_id, user_id, created_at);
	`
	_, err := db.conn.Exec(query)
	return err
}

// SaveConversation 保存对话消息
func (db *DB) SaveConversation(conv *HouseholdConversation) error {
	conv.ID = generateID(8)
	conv.CreatedAt = time.Now()
	_, err := db.conn.Exec(`
		INSERT INTO household_conversations (id, profile_id, user_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, conv.ID, conv.ProfileID, conv.UserID, conv.Role, conv.Content, conv.CreatedAt)
	return err
}

// GetConversations 获取对话历史
func (db *DB) GetConversations(profileID, userID string, limit int) ([]*HouseholdConversation, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(`
		SELECT id, profile_id, user_id, role, content, created_at
		FROM household_conversations
		WHERE profile_id = ? AND user_id = ?
		ORDER BY created_at ASC
		LIMIT ?
	`, profileID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []*HouseholdConversation
	for rows.Next() {
		c := &HouseholdConversation{}
		if err := rows.Scan(&c.ID, &c.ProfileID, &c.UserID, &c.Role, &c.Content, &c.CreatedAt); err != nil {
			continue
		}
		convs = append(convs, c)
	}
	return convs, nil
}

// ClearConversations 清除对话历史
func (db *DB) ClearConversations(profileID, userID string) error {
	_, err := db.conn.Exec("DELETE FROM household_conversations WHERE profile_id = ? AND user_id = ?", profileID, userID)
	return err
}

func init() {
	// strconv 引用需要
}
