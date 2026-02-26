package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Recipe struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Password    string     `json:"-"`
	PasswordIndex string   `json:"-"`
	CreatorKey  string     `json:"-"`
	Data        string     `json:"data"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatorIP   string     `json:"-"`
}

func (db *DB) InitRecipe() error {
	query := `
	CREATE TABLE IF NOT EXISTS recipes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		password TEXT NOT NULL,
		password_index TEXT NOT NULL DEFAULT '',
		creator_key TEXT NOT NULL,
		data TEXT NOT NULL DEFAULT '{}',
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		creator_ip TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_recipes_expires_at ON recipes(expires_at);
	CREATE INDEX IF NOT EXISTS idx_recipes_creator_ip ON recipes(creator_ip);
	CREATE INDEX IF NOT EXISTS idx_recipes_password_index ON recipes(password_index) WHERE password_index != '';
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) CreateRecipe(recipe *Recipe) error {
	recipe.ID = generateID(8)
	recipe.CreatedAt = time.Now()
	recipe.UpdatedAt = time.Now()

	_, err := db.conn.Exec(`
		INSERT INTO recipes (id, name, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, recipe.ID, recipe.Name, recipe.Password, recipe.PasswordIndex, recipe.CreatorKey, recipe.Data,
		recipe.ExpiresAt, recipe.CreatedAt, recipe.UpdatedAt, recipe.CreatorIP)

	return err
}

func (db *DB) GetRecipe(id string) (*Recipe, error) {
	recipe := &Recipe{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, name, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip
		FROM recipes WHERE id = ?
	`, id).Scan(
		&recipe.ID, &recipe.Name, &recipe.Password, &recipe.PasswordIndex, &recipe.CreatorKey, &recipe.Data,
		&expiresAt, &recipe.CreatedAt, &recipe.UpdatedAt, &recipe.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		recipe.ExpiresAt = &expiresAt.Time
	}

	return recipe, nil
}

func (db *DB) GetRecipeByPasswordIndex(passwordIndex string) (*Recipe, error) {
	recipe := &Recipe{}
	var expiresAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT id, name, password, password_index, creator_key, data, expires_at, created_at, updated_at, creator_ip
		FROM recipes WHERE password_index = ?
	`, passwordIndex).Scan(
		&recipe.ID, &recipe.Name, &recipe.Password, &recipe.PasswordIndex, &recipe.CreatorKey, &recipe.Data,
		&expiresAt, &recipe.CreatedAt, &recipe.UpdatedAt, &recipe.CreatorIP)

	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		recipe.ExpiresAt = &expiresAt.Time
	}

	return recipe, nil
}

func (db *DB) UpdateRecipeData(id, data string) error {
	_, err := db.conn.Exec(
		"UPDATE recipes SET data = ?, updated_at = ? WHERE id = ?",
		data, time.Now(), id)
	return err
}

func (db *DB) UpdateRecipeName(id, name string) error {
	_, err := db.conn.Exec(
		"UPDATE recipes SET name = ?, updated_at = ? WHERE id = ?",
		name, time.Now(), id)
	return err
}

func (db *DB) ExtendRecipe(id string, expiresAt *time.Time) error {
	_, err := db.conn.Exec(
		"UPDATE recipes SET expires_at = ?, updated_at = ? WHERE id = ?",
		expiresAt, time.Now(), id)
	return err
}

func (db *DB) DeleteRecipe(id string) error {
	_, err := db.conn.Exec("DELETE FROM recipes WHERE id = ?", id)
	return err
}

func (db *DB) CleanExpiredRecipes() (int64, error) {
	result, err := db.conn.Exec(`
		DELETE FROM recipes
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) CountRecipeByIP(ip string, since time.Time) (int, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM recipes WHERE creator_ip = ? AND created_at > ?",
		ip, since,
	).Scan(&count)
	return count, err
}

// GetDefaultRecipeData 返回默认菜谱数据（JSON 格式）
func GetDefaultRecipeData() map[string]interface{} {
	jsonData := `{
		"weeklyRecipes": [
			[
				{"day": "周一", "breakfast": "燕麦粥+水煮蛋+小番茄", "breakfastTip": "燕麦低GI", "lunch": "清蒸鱼+糙米饭+西兰花", "lunchTip": "优质蛋白", "dinner": "冬瓜排骨汤+清炒油麦菜", "dinnerTip": "利尿排石", "suggestion": "多喝水，午休后散步"},
				{"day": "周二", "breakfast": "全麦面包+牛奶+香蕉", "breakfastTip": "香蕉补钾", "lunch": "番茄炒蛋+糙米饭+紫菜汤", "lunchTip": "低草酸", "dinner": "清炒鸡肉+凉拌黄瓜", "dinnerTip": "少油", "suggestion": "晚餐不宜太晚"},
				{"day": "周三", "breakfast": "玉米糊+鸡蛋羹+苹果", "breakfastTip": "苹果低草酸", "lunch": "红烧肉（瘦）+米饭+炒青菜", "lunchTip": "适量瘦肉", "dinner": "鲫鱼豆腐汤+蒸南瓜", "dinnerTip": "豆腐补钙", "suggestion": "睡前少喝水"},
				{"day": "周四", "breakfast": "小米粥+咸鸭蛋（少）+坚果", "breakfastTip": "坚果少量", "lunch": "炒鸡胸肉+糙米饭+胡萝卜", "lunchTip": "高纤维", "dinner": "清蒸虾+凉拌木耳+米饭", "dinnerTip": "低嘌呤", "suggestion": "每天运动30分钟"},
				{"day": "周五", "breakfast": "豆浆+油条（少）+水果", "breakfastTip": "自制豆浆", "lunch": "土豆烧牛肉+米饭+青菜", "lunchTip": "牛肉适量", "dinner": "冬瓜虾皮汤+炒生菜", "dinnerTip": "补钙", "suggestion": "周末去公园散步"},
				{"day": "周六", "breakfast": "牛奶+鸡蛋+全麦饼干", "breakfastTip": "加餐点心", "lunch": "清蒸排骨+糙米饭+炒白菜", "lunchTip": "排骨少盐", "dinner": "番茄鱼片汤+凉拌黄瓜", "dinnerTip": "清淡", "suggestion": "家人一起用餐"},
				{"day": "周日", "breakfast": "八宝粥+鸡蛋+坚果", "breakfastTip": "杂粮粥", "lunch": "红烧鸡翅+米饭+炒菠菜", "lunchTip": "菠菜焯水", "dinner": "紫菜蛋花汤+玉米+水果", "dinnerTip": "轻食", "suggestion": "准备下周食材"}
			],
			[
				{"day": "周一", "breakfast": "燕麦牛奶+水煮蛋+葡萄", "breakfastTip": "低GI水果", "lunch": "清蒸鲈鱼+糙米饭+炒油麦菜", "lunchTip": "低草酸蔬菜", "dinner": "玉米排骨汤+凉拌黄瓜", "dinnerTip": "利尿", "suggestion": "多喝水促进排石"},
				{"day": "周二", "breakfast": "全麦吐司+酸奶+蓝莓", "breakfastTip": "酸奶益生菌", "lunch": "炒牛肉丝+米饭+胡萝卜丝", "lunchTip": "适量补铁", "dinner": "冬瓜肉丸汤+清炒生菜", "dinnerTip": "清淡", "suggestion": "饭后散步"},
				{"day": "周三", "breakfast": "玉米糊+荷包蛋+苹果", "breakfastTip": "粗粮低GI", "lunch": "白灼虾+糙米饭+炒西葫芦", "lunchTip": "低嘌呤", "dinner": "番茄炒蛋+米饭+紫菜汤", "dinnerTip": "简单营养", "suggestion": "控制盐量"},
				{"day": "周四", "breakfast": "豆浆+粗粮饼+香蕉", "breakfastTip": "香蕉补钾", "lunch": "炖鸡腿+米饭+炒青菜", "lunchTip": "去皮鸡肉", "dinner": "豆腐鱼头汤+凉拌木耳", "dinnerTip": "豆腐补钙", "suggestion": "睡前2小时不喝水"},
				{"day": "周五", "breakfast": "小米粥+鸡蛋+坚果", "breakfastTip": "每天一小把", "lunch": "清蒸鳕鱼+糙米饭+西兰花", "lunchTip": "优质蛋白", "dinner": "丝瓜蛋花汤+炒白菜", "dinnerTip": "清热", "suggestion": "午休后活动"},
				{"day": "周六", "breakfast": "牛奶+鸡蛋+全麦面包", "breakfastTip": "营养早餐", "lunch": "红烧肉（瘦）+米饭+炒菠菜", "lunchTip": "菠菜焯水", "dinner": "冬瓜虾皮汤+玉米", "dinnerTip": "补钙", "suggestion": "周末家庭日"},
				{"day": "周日", "breakfast": "八宝粥+咸鸭蛋+水果", "breakfastTip": "杂粮均衡", "lunch": "炒鸡胸肉+糙米饭+胡萝卜", "lunchTip": "高纤维", "dinner": "鲫鱼豆腐汤+凉拌黄瓜", "dinnerTip": "利尿", "suggestion": "准备下周计划"}
			],
			[
				{"day": "周一", "breakfast": "燕麦粥+鸡蛋+小番茄", "breakfastTip": "抗氧化物", "lunch": "清蒸鲈鱼+糙米饭+炒油麦", "lunchTip": "低草酸", "dinner": "冬瓜排骨汤+凉拌黄瓜", "dinnerTip": "利尿排石", "suggestion": "每天8杯水"},
				{"day": "周二", "breakfast": "全麦面包+牛奶+苹果", "breakfastTip": "低GI", "lunch": "番茄牛腩+米饭+炒青菜", "lunchTip": "适量", "dinner": "紫菜蛋花汤+蒸南瓜", "dinnerTip": "清淡", "suggestion": "餐后活动"},
				{"day": "周三", "breakfast": "玉米糊+鸡蛋羹+香蕉", "breakfastTip": "易消化", "lunch": "白灼虾+糙米饭+西兰花", "lunchTip": "高维C", "dinner": "豆腐肉末汤+炒生菜", "dinnerTip": "补钙", "suggestion": "少盐少油"},
				{"day": "周四", "breakfast": "豆浆+油条（少）+坚果", "breakfastTip": "自制豆浆", "lunch": "炒鸡胸肉+米饭+胡萝卜", "lunchTip": "低脂", "dinner": "冬瓜虾皮汤+凉拌木耳", "dinnerTip": "补钙", "suggestion": "适度运动"},
				{"day": "周五", "breakfast": "小米粥+鸡蛋+水果", "breakfastTip": "养胃", "lunch": "清蒸排骨+糙米饭+炒白菜", "lunchTip": "少盐", "dinner": "番茄鱼片汤+玉米", "dinnerTip": "高蛋白", "suggestion": "周末采购"},
				{"day": "周六", "breakfast": "牛奶+全麦饼干+蓝莓", "breakfastTip": "抗氧化", "lunch": "红烧肉（瘦）+米饭+菠菜", "lunchTip": "焯水去草酸", "dinner": "丝瓜蛋花汤+炒青菜", "dinnerTip": "清淡", "suggestion": "家庭聚餐"},
				{"day": "周日", "breakfast": "八宝粥+鸡蛋+坚果", "breakfastTip": "杂粮", "lunch": "炖鸡翅+糙米饭+炒西葫芦", "lunchTip": "去皮", "dinner": "紫菜汤+蒸南瓜+水果", "dinnerTip": "轻食", "suggestion": "总结下周"}
			],
			[
				{"day": "周一", "breakfast": "燕麦牛奶+水煮蛋+葡萄", "breakfastTip": "低GI水果", "lunch": "清蒸鲈鱼+糙米饭+西兰花", "lunchTip": "高纤维", "dinner": "冬瓜排骨汤+凉拌黄瓜", "dinnerTip": "利尿", "suggestion": "多喝水"},
				{"day": "周二", "breakfast": "全麦吐司+酸奶+苹果", "breakfastTip": "益生菌", "lunch": "炒牛肉+米饭+炒油麦", "lunchTip": "适量补铁", "dinner": "豆腐鲫鱼汤+炒青菜", "dinnerTip": "优质蛋白", "suggestion": "饭后散步"},
				{"day": "周三", "breakfast": "玉米糊+荷包蛋+香蕉", "breakfastTip": "补钾", "lunch": "白灼虾+糙米饭+胡萝卜", "lunchTip": "低嘌呤", "dinner": "番茄炒蛋+米饭+紫菜汤", "dinnerTip": "简单", "suggestion": "控制糖分"},
				{"day": "周四", "breakfast": "豆浆+粗粮饼+坚果", "breakfastTip": "植物蛋白", "lunch": "炖鸡腿+米饭+炒菠菜", "lunchTip": "焯水", "dinner": "冬瓜虾皮汤+蒸南瓜", "dinnerTip": "补钙", "suggestion": "少油"},
				{"day": "周五", "breakfast": "小米粥+鸡蛋+水果", "breakfastTip": "养胃", "lunch": "清蒸鳕鱼+糙米饭+炒白菜", "lunchTip": "低脂", "dinner": "丝瓜肉片汤+凉拌木耳", "dinnerTip": "清淡", "suggestion": "周末计划"},
				{"day": "周六", "breakfast": "牛奶+鸡蛋+全麦面包", "breakfastTip": "营养全面", "lunch": "红烧肉（瘦）+米饭+青菜", "lunchTip": "控制量", "dinner": "紫菜蛋花汤+玉米+水果", "dinnerTip": "轻食", "suggestion": "家庭时光"},
				{"day": "周日", "breakfast": "八宝粥+咸鸭蛋+坚果", "breakfastTip": "杂粮", "lunch": "炒鸡胸肉+糙米饭+胡萝卜", "lunchTip": "高蛋白", "dinner": "鲫鱼豆腐汤+黄瓜", "dinnerTip": "利尿", "suggestion": "下周准备"}
			]
		],
		"shoppingCategories": [
			{"name": "肉类", "items": ["鸡胸肉 500g", "猪排骨 500g", "鲈鱼/鲫鱼 2条", "虾 500g", "牛肉 300g", "鸡蛋 20个"]},
			{"name": "蔬菜", "items": ["西兰花 1颗", "油麦菜 2把", "菠菜 1把", "黄瓜 4根", "胡萝卜 3根", "番茄 6个", "冬瓜 1块", "白菜 1颗", "南瓜 1个", "木耳 适量"]},
			{"name": "主食", "items": ["糙米 2kg", "燕麦片 500g", "全麦面包 1袋", "小米 500g", "玉米 4根", "杂粮粥料 1包"]},
			{"name": "水果/坚果", "items": ["苹果 6个", "香蕉 6根", "葡萄 1斤", "小番茄 1盒", "坚果（核桃/杏仁）300g", "蓝莓 1盒"]},
			{"name": "奶/豆制品", "items": ["牛奶 2箱", "酸奶 1箱", "豆浆 1箱", "豆腐 2块", "紫菜 1包"]},
			{"name": "其他", "items": ["鸡蛋 20个", "粗粮饼 1袋", "八宝粥料 1包", "排骨汤料 适量"]}
		]
	}`

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}

// GetDetailedRecipeSteps 返回带详细步骤的菜谱数据
func GetDetailedRecipeSteps() map[string]interface{} {
	jsonData := `{
		"weeklyRecipes": [
			[
				{"day": "周一", "breakfast": "燕麦粥+水煮蛋+小番茄", "breakfastTip": "燕麦低GI", "breakfastSteps": ["燕麦片洗净，加水煮至软烂约10分钟", "水煮蛋：冷水下锅，大火煮沸后转小火8分钟", "小番茄洗净即可食用"], "lunch": "清蒸鱼+糙米饭+西兰花", "lunchTip": "优质蛋白", "lunchSteps": ["清蒸鱼：鱼身划刀，姜片铺底，葱段放鱼上，蒸锅水开后放入鱼蒸8-10分钟", "出锅后淋入蒸鱼豉油，浇热油", "糙米提前浸泡2小时，电饭煲煮熟", "西兰花洗净切小朵，沸水焯2分钟捞出，蒜蓉清炒"], "dinner": "冬瓜排骨汤+清炒油麦菜", "dinnerTip": "利尿排石", "dinnerSteps": ["排骨洗净冷水下锅，加姜片料酒焯水去腥，捞出洗净", "冬瓜去皮切块，与排骨一起放入砂锅", "加足量清水，大火烧开转小火炖1小时", "油麦菜洗净，蒜蓉爆香后放入翻炒1分钟出锅"]},
				{"day": "周二", "breakfast": "全麦面包+牛奶+香蕉", "breakfastTip": "香蕉补钾", "breakfastSteps": ["全麦面包切片，烤箱或平底锅略微烘烤", "牛奶加热至温热即可", "香蕉剥皮直接食用"], "lunch": "番茄炒蛋+糙米饭+紫菜汤", "lunchTip": "低草酸", "lunchSteps": ["番茄洗净切块，鸡蛋打散加少许盐", "热锅热油先炒鸡蛋至凝固盛出", "再炒番茄出汁，倒入鸡蛋翻炒均匀", "糙米提前浸泡煮熟", "紫菜撕碎，沸水冲泡，加盐和香油"], "dinner": "清炒鸡肉+凉拌黄瓜", "dinnerTip": "少油", "dinnerSteps": ["鸡胸肉切丁，加料酒、淀粉腌制10分钟", "热锅少油，先炒鸡肉至变色", "加少许酱油调味，快速翻炒出锅", "黄瓜拍碎切块，加蒜泥、醋、盐、香油拌匀"]},
				{"day": "周三", "breakfast": "玉米糊+鸡蛋羹+苹果", "breakfastTip": "苹果低草酸", "breakbreakSteps": ["玉米面加水搅匀，沸水边倒边搅，煮至粘稠", "鸡蛋打散，加1:1温水，盐少许", "蒸锅水开，放入鸡蛋碗，盖保鲜膜扎孔，蒸8分钟", "苹果洗净切块或直接食用"], "lunch": "红烧肉（瘦）+米饭+炒青菜", "lunchTip": "适量瘦肉", "lunchSteps": ["五花肉切块，冷水下锅焯水", "热锅炒糖色，放入肉块翻炒上色", "加生抽、老抽、姜片、八角、桂皮", "加水没过肉块，小火炖40分钟", "米饭用电饭煲煮熟", "青菜洗净，蒜蓉爆香后快速翻炒"], "dinner": "鲫鱼豆腐汤+蒸南瓜", "dinnerTip": "豆腐补钙", "dinnerSteps": ["鲫鱼去鳞去内脏，洗净，两面划刀", "热锅热油，将鱼煎至两面金黄", "加开水、姜片，大火煮10分钟", "放入豆腐块，小火再煮10分钟", "南瓜切块，蒸锅蒸15分钟"]},
				{"day": "周四", "breakfast": "小米粥+咸鸭蛋（少）+坚果", "breakfastTip": "坚果少量", "breakfastSteps": ["小米洗净，水开后下锅，小火煮20分钟", "咸鸭蛋煮熟，取1/4个蛋黄食用", "坚果一小把（约15克）"], "lunch": "炒鸡胸肉+糙米饭+胡萝卜", "lunchTip": "高纤维", "lunchSteps": ["鸡胸肉切丝，加料酒、淀粉腌制", "胡萝卜切丝", "热锅少油，先炒鸡胸肉变色盛出", "再炒胡萝卜丝，放入鸡胸肉一起翻炒", "糙米提前浸泡煮熟"], "dinner": "清蒸虾+凉拌木耳+米饭", "dinnerTip": "低嘌呤", "dinnerSteps": ["鲜虾洗净，去虾线，摆在盘子上", "放姜片葱段，蒸锅蒸5分钟", "木耳提前泡发，沸水焯2分钟", "加蒜泥、醋、盐、香油拌匀", "米饭煮熟"]},
				{"day": "周五", "breakfast": "豆浆+油条（少）+水果", "breakfastTip": "自制豆浆", "breakfastSteps": ["黄豆提前浸泡8小时，放入豆浆机打成豆浆", "豆浆煮沸后转小火煮5分钟", "油条切段，锅中复炸至金黄酥脆（少量）", "水果洗净切块"], "lunch": "土豆烧牛肉+米饭+青菜", "lunchTip": "牛肉适量", "lunchSteps": ["牛肉切块，冷水焯水去腥", "土豆切块备用", "热锅炒香姜蒜，放入牛肉翻炒", "加生抽、老抽、料酒，加水没过牛肉", "放入土豆，小火炖30分钟", "米饭煮熟，青菜清炒"], "dinner": "冬瓜虾皮汤+炒生菜", "dinnerTip": "补钙", "dinnerSteps": ["冬瓜去皮切片，虾皮洗净", "锅中加水烧开，放入冬瓜煮5分钟", "放入虾皮，再煮2分钟", "生菜洗净，蒜蓉爆香后快速翻炒1分钟"]},
				{"day": "周六", "breakfast": "牛奶+鸡蛋+全麦饼干", "breakfastTip": "加餐点心", "breakfastSteps": ["牛奶加热", "水煮蛋：冷水下锅煮8分钟", "全麦饼干直接食用"], "lunch": "清蒸排骨+糙米饭+炒白菜", "lunchTip": "排骨少盐", "lunchSteps": ["排骨洗净，加姜片、葱段、料酒", "蒸锅蒸30分钟至软烂", "糙米煮熟", "白菜切块，蒜蓉爆香后翻炒"], "dinner": "番茄鱼片汤+凉拌黄瓜", "dinnerTip": "清淡", "dinnerSteps": ["番茄切块，鱼片用淀粉腌制", "锅中加水煮番茄，放入鱼片煮熟", "黄瓜拍碎，加蒜泥、醋、盐、香油拌匀"]},
				{"day": "周日", "breakfast": "八宝粥+鸡蛋+坚果", "breakfastTip": "杂粮粥", "breakfastSteps": ["八宝粥料提前浸泡4小时", "加水煮至软烂约40分钟", "水煮蛋1个", "坚果15克"], "lunch": "红烧鸡翅+米饭+炒菠菜", "lunchTip": "菠菜焯水", "lunchSteps": ["鸡翅洗净，两面划刀", "热锅炒糖色，放入鸡翅翻炒", "加生抽、老抽、料酒、姜片", "加水没过鸡翅，小火焖20分钟", "菠菜沸水焯1分钟去草酸，捞出凉拌或清炒", "米饭煮熟"], "dinner": "紫菜蛋花汤+玉米+水果", "dinnerTip": "轻食", "dinnerSteps": ["紫菜撕碎，沸水冲泡", "鸡蛋打散，慢慢倒入形成蛋花", "加盐、香油", "玉米煮熟或蒸熟", "水果洗净"]}
			],
			[
				{"day": "周一", "breakfast": "燕麦牛奶+水煮蛋+葡萄", "breakfastTip": "低GI水果", "breakfastSteps": ["燕麦片加牛奶煮5分钟至软烂", "水煮蛋冷水下锅煮8分钟", "葡萄洗净"], "lunch": "清蒸鲈鱼+糙米饭+炒油麦菜", "lunchTip": "低草酸蔬菜", "lunchSteps": ["鲈鱼洗净，两面划刀", "盘底铺姜片葱段，放上鱼", "蒸锅水开后蒸8-10分钟", "淋蒸鱼豉油，浇热油", "油麦菜洗净，蒜蓉清炒"], "dinner": "玉米排骨汤+凉拌黄瓜", "dinnerTip": "利尿", "dinnerSteps": ["排骨焯水，玉米切段", "砂锅加水，放入排骨和玉米", "小火炖1小时", "黄瓜拍碎凉拌"]},
				{"day": "周二", "breakfast": "全麦吐司+酸奶+蓝莓", "breakfastTip": "酸奶益生菌", "breakfastSteps": ["全麦吐司烤一下", "酸奶直接从冰箱取出食用", "蓝莓洗净"], "lunch": "炒牛肉丝+米饭+胡萝卜丝", "lunchTip": "适量补铁", "lunchSteps": ["牛肉切丝，加料酒、淀粉腌制", "胡萝卜切丝", "热锅少油，先炒牛肉丝变色盛出", "再炒胡萝卜丝，放入牛肉一起翻炒", "米饭煮熟"], "dinner": "冬瓜肉丸汤+清炒生菜", "dinnerTip": "清淡", "dinnerSteps": ["冬瓜去皮切片，肉末加盐、淀粉搅拌上劲", "水烧开后放入肉丸", "放入冬瓜煮5分钟", "生菜洗净清炒"]},
				{"day": "周三", "breakfast": "玉米糊+荷包蛋+苹果", "breakfastTip": "粗粮低GI", "breakbreakSteps": ["玉米面加水搅匀，沸水边倒边搅煮至粘稠", "热锅热油，打入鸡蛋煎至蛋白凝固", "苹果洗净切块"], "lunch": "白灼虾+糙米饭+炒西葫芦", "lunchTip": "低嘌呤", "lunchSteps": ["鲜虾洗净，沸水加姜片、料酒", "放入虾煮至变色捞出", "西葫芦切片，蒜蓉清炒", "糙米煮熟"], "dinner": "番茄炒蛋+米饭+紫菜汤", "dinnerTip": "简单营养", "dinnerSteps": ["番茄切块，鸡蛋打散", "炒鸡蛋盛出，再炒番茄", "放入鸡蛋一起翻炒", "米饭煮熟，紫菜沸水冲泡"]},
				{"day": "周四", "breakfast": "豆浆+粗粮饼+香蕉", "breakfastTip": "香蕉补钾", "breakfastSteps": ["黄豆打豆浆煮沸", "粗粮饼煎至两面金黄", "香蕉剥皮食用"], "lunch": "炖鸡腿+米饭+炒青菜", "lunchTip": "去皮鸡肉", "lunchSteps": ["鸡腿去皮洗净，加姜片、料酒", "加水没过鸡腿，小火炖30分钟", "米饭煮熟，青菜清炒"], "dinner": "豆腐鱼头汤+凉拌木耳", "dinnerTip": "豆腐补钙", "dinnerSteps": ["鱼头洗净煎至两面金黄", "加开水、姜片，大火煮10分钟", "放入豆腐块，小火煮10分钟", "木耳泡发焯水，凉拌"]},
				{"day": "周五", "breakfast": "小米粥+鸡蛋+坚果", "breakfastTip": "每天一小把", "breakfastSteps": ["小米洗净煮粥20分钟", "水煮蛋1个", "坚果15克"], "lunch": "清蒸鳕鱼+糙米饭+西兰花", "lunchTip": "优质蛋白", "lunchSteps": ["鳕鱼洗净摆盘，放姜片葱段", "蒸锅蒸8分钟", "西兰花切小朵，沸水焯2分钟，蒜蓉清炒", "糙米煮熟"], "dinner": "丝瓜蛋花汤+炒白菜", "dinnerTip": "清热", "dinnerSteps": ["丝瓜切片，鸡蛋打散", "加水烧开，放入丝瓜", "倒入蛋花", "白菜切块清炒"]},
				{"day": "周六", "breakfast": "牛奶+鸡蛋+全麦面包", "breakfastTip": "营养早餐", "breakfastSteps": ["牛奶加热", "水煮蛋", "全麦面包烤一下"], "lunch": "红烧肉（瘦）+米饭+炒菠菜", "lunchTip": "菠菜焯水", "lunchSteps": ["瘦肉切块焯水", "炒糖色，放肉翻炒", "加调料和水，小火炖40分钟", "菠菜沸水焯1分钟", "米饭煮熟"], "dinner": "冬瓜虾皮汤+玉米", "dinnerTip": "补钙", "dinnerSteps": ["冬瓜切片，虾皮洗净", "加水煮冬瓜5分钟", "放入虾皮煮2分钟", "玉米蒸熟"]},
				{"day": "周日", "breakfast": "八宝粥+咸鸭蛋+水果", "breakfastTip": "杂粮均衡", "breakbreakSteps": ["八宝粥料提前浸泡，煮40分钟", "咸鸭蛋煮熟取1/4", "水果洗净"]}
			],
			[
				{"day": "周一", "breakfast": "燕麦粥+鸡蛋+小番茄", "breakfastTip": "抗氧化物", "breakfastSteps": ["燕麦片加水煮软烂约10分钟", "水煮蛋", "小番茄洗净"], "lunch": "清蒸鲈鱼+糙米饭+炒油麦", "lunchTip": "低草酸", "lunchSteps": ["鲈鱼洗净蒸8-10分钟", "油麦菜洗净蒜蓉清炒", "糙米煮熟"], "dinner": "冬瓜排骨汤+凉拌黄瓜", "dinnerTip": "利尿排石", "dinnerSteps": ["排骨焯水，冬瓜切块", "一起炖1小时", "黄瓜拍碎凉拌"]},
				{"day": "周二", "breakfast": "全麦面包+牛奶+苹果", "breakfastTip": "低GI", "breakfastSteps": ["全麦面包烤一下", "牛奶加热", "苹果洗净切块"], "lunch": "番茄牛腩+米饭+炒青菜", "lunchTip": "适量", "lunchSteps": ["牛腩切块焯水", "番茄切块", "一起炖1小时", "米饭煮熟，青菜清炒"], "dinner": "紫菜蛋花汤+蒸南瓜", "dinnerTip": "清淡", "dinnerSteps": ["紫菜沸水冲泡，鸡蛋打散倒入", "南瓜切块蒸15分钟"]},
				{"day": "周三", "breakfast": "玉米糊+鸡蛋羹+香蕉", "breakfastTip": "易消化", "breakbreakSteps": ["玉米面加水搅匀煮沸", "鸡蛋羹蒸8分钟", "香蕉直接食用"], "lunch": "白灼虾+糙米饭+西兰花", "lunchTip": "高维C", "lunchSteps": ["虾沸水煮至变色", "西兰花焯水后清炒", "糙米煮熟"], "dinner": "豆腐肉末汤+炒生菜", "dinnerTip": "补钙", "dinnerSteps": ["豆腐切块，肉末炒香加水", "放入豆腐煮5分钟", "生菜清炒"]},
				{"day": "周四", "breakfast": "豆浆+油条（少）+坚果", "breakfastTip": "自制豆浆", "breakfastSteps": ["黄豆打浆煮沸", "油条少量复炸", "坚果15克"], "lunch": "炒鸡胸肉+米饭+胡萝卜", "lunchTip": "低脂", "lunchSteps": ["鸡胸肉切丁腌制", "胡萝卜切丁", "一起翻炒", "米饭煮熟"], "dinner": "冬瓜虾皮汤+凉拌木耳", "dinnerTip": "补钙", "dinnerSteps": ["冬瓜切片煮5分钟", "放入虾皮煮2分钟", "木耳凉拌"]},
				{"day": "周五", "breakfast": "小米粥+鸡蛋+水果", "breakfastTip": "养胃", "breakfastSteps": ["小米煮粥20分钟", "水煮蛋", "水果洗净"], "lunch": "清蒸排骨+糙米饭+炒白菜", "lunchTip": "少盐", "lunchSteps": ["排骨洗净蒸30分钟", "白菜切块清炒", "糙米煮熟"], "dinner": "番茄鱼片汤+玉米", "dinnerTip": "高蛋白", "dinnerSteps": ["番茄煮汤，放入鱼片煮熟", "玉米蒸熟"]},
				{"day": "周六", "breakfast": "牛奶+全麦饼干+蓝莓", "breakfastTip": "抗氧化", "breakfastSteps": ["牛奶加热", "全麦饼干", "蓝莓洗净"], "lunch": "红烧肉（瘦）+米饭+菠菜", "lunchTip": "焯水去草酸", "lunchSteps": ["瘦肉红烧肉做法", "菠菜焯水", "米饭煮熟"], "dinner": "丝瓜蛋花汤+炒青菜", "dinnerTip": "清淡", "dinnerSteps": ["丝瓜煮汤", "青菜清炒"]},
				{"day": "周日", "breakfast": "八宝粥+鸡蛋+坚果", "breakfastTip": "杂粮", "breakbreakSteps": ["八宝粥煮40分钟", "水煮蛋", "坚果15克"], "lunch": "炖鸡翅+糙米饭+炒西葫芦", "lunchTip": "去皮", "lunchSteps": ["鸡翅去皮炖20分钟", "西葫芦切片清炒", "糙米煮熟"], "dinner": "紫菜汤+蒸南瓜+水果", "dinnerTip": "轻食", "dinnerSteps": ["紫菜沸水冲泡", "南瓜蒸15分钟", "水果洗净"]}
			],
			[
				{"day": "周一", "breakfast": "燕麦牛奶+水煮蛋+葡萄", "breakfastTip": "低GI水果", "breakfastSteps": ["燕麦加牛奶煮5分钟", "水煮蛋", "葡萄洗净"], "lunch": "清蒸鲈鱼+糙米饭+西兰花", "lunchTip": "高纤维", "lunchSteps": ["鲈鱼蒸8-10分钟", "西兰花焯水清炒", "糙米煮熟"], "dinner": "冬瓜排骨汤+凉拌黄瓜", "dinnerTip": "利尿", "dinnerSteps": ["排骨炖1小时加冬瓜", "黄瓜凉拌"]},
				{"day": "周二", "breakfast": "全麦吐司+酸奶+苹果", "breakfastTip": "益生菌", "breakfastSteps": ["吐司烤一下", "酸奶", "苹果切块"], "lunch": "炒牛肉+米饭+炒油麦", "lunchTip": "适量补铁", "lunchSteps": ["牛肉切片炒熟", "油麦菜清炒", "米饭煮熟"], "dinner": "豆腐鲫鱼汤+炒青菜", "dinnerTip": "优质蛋白", "dinnerSteps": ["鲫鱼煎一下加水", "放入豆腐煮10分钟", "青菜清炒"]},
				{"day": "周三", "breakfast": "玉米糊+荷包蛋+香蕉", "breakfastTip": "补钾", "breakbreakSteps": ["玉米面煮糊", "煎荷包蛋", "香蕉食用"], "lunch": "白灼虾+糙米饭+胡萝卜", "lunchTip": "低嘌呤", "lunchSteps": ["虾沸水煮", "胡萝卜切片清炒", "糙米煮熟"], "dinner": "番茄炒蛋+米饭+紫菜汤", "dinnerTip": "简单", "dinnerSteps": ["番茄炒蛋", "米饭煮熟", "紫菜汤"]},
				{"day": "周四", "breakfast": "豆浆+粗粮饼+坚果", "breakfastTip": "植物蛋白", "breakfastSteps": ["豆浆煮沸", "粗粮饼煎一下", "坚果15克"], "lunch": "炖鸡腿+米饭+炒菠菜", "lunchTip": "焯水", "lunchSteps": ["鸡腿炖30分钟", "菠菜焯水", "米饭煮熟"], "dinner": "冬瓜虾皮汤+蒸南瓜", "dinnerTip": "补钙", "dinnerSteps": ["冬瓜虾皮煮汤", "南瓜蒸熟"]},
				{"day": "周五", "breakfast": "小米粥+鸡蛋+水果", "breakfastTip": "养胃", "breakfastSteps": ["小米粥煮20分钟", "水煮蛋", "水果"], "lunch": "清蒸鳕鱼+糙米饭+炒白菜", "lunchTip": "低脂", "lunchSteps": ["鳕鱼蒸8分钟", "白菜清炒", "糙米煮熟"], "dinner": "丝瓜肉片汤+凉拌木耳", "dinnerTip": "清淡", "dinnerSteps": ["丝瓜肉片煮汤", "木耳凉拌"]},
				{"day": "周六", "breakfast": "牛奶+鸡蛋+全麦面包", "breakfastTip": "营养全面", "breakfastSteps": ["牛奶加热", "水煮蛋", "全麦面包"], "lunch": "红烧肉（瘦）+米饭+青菜", "lunchTip": "控制量", "lunchSteps": ["红烧肉做法", "米饭煮熟", "青菜清炒"], "dinner": "紫菜蛋花汤+玉米+水果", "dinnerTip": "轻食", "dinnerSteps": ["紫菜蛋花汤", "玉米蒸熟", "水果"]},
				{"day": "周日", "breakfast": "八宝粥+咸鸭蛋+坚果", "breakfastTip": "杂粮", "breakbreakSteps": ["八宝粥煮40分钟", "咸鸭蛋1/4", "坚果15克"], "lunch": "炒鸡胸肉+糙米饭+胡萝卜", "lunchTip": "高蛋白", "lunchSteps": ["鸡胸肉炒熟", "胡萝卜切片", "糙米煮熟"], "dinner": "鲫鱼豆腐汤+黄瓜", "dinnerTip": "利尿", "dinnerSteps": ["鲫鱼豆腐汤", "黄瓜凉拌"]}
			}
		],
		"shoppingCategories": [
			{"name": "肉类", "items": ["鸡胸肉 500g", "猪排骨 500g", "鲈鱼/鲫鱼 2条", "虾 500g", "牛肉 300g", "鸡蛋 20个"]},
			{"name": "蔬菜", "items": ["西兰花 1颗", "油麦菜 2把", "菠菜 1把", "黄瓜 4根", "胡萝卜 3根", "番茄 6个", "冬瓜 1块", "白菜 1颗", "南瓜 1个", "木耳 适量"]},
			{"name": "主食", "items": ["糙米 2kg", "燕麦片 500g", "全麦面包 1袋", "小米 500g", "玉米 4根", "杂粮粥料 1包"]},
			{"name": "水果/坚果", "items": ["苹果 6个", "香蕉 6根", "葡萄 1斤", "小番茄 1盒", "坚果（核桃/杏仁）300g", "蓝莓 1盒"]},
			{"name": "奶/豆制品", "items": ["牛奶 2箱", "酸奶 1箱", "豆浆 1箱", "豆腐 2块", "紫菜 1包"]},
			{"name": "调料", "items": ["生抽", "老抽", "料酒", "盐", "糖", "姜", "蒜", "葱", "蒸鱼豉油", "香油"]}
		]
	}`

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}
