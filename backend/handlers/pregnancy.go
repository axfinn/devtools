package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

func passwordIndex(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

type PregnancyHandler struct {
	db             *models.DB
	defaultExpDays int
	maxDataSize    int
	maxPerIP       int
	ipWindow       time.Duration
}

func NewPregnancyHandler(db *models.DB, defaultExpDays, maxDataSize int) *PregnancyHandler {
	if defaultExpDays <= 0 {
		defaultExpDays = 365
	}
	if maxDataSize <= 0 {
		maxDataSize = 1024 * 1024 // 1MB
	}
	return &PregnancyHandler{
		db:             db,
		defaultExpDays: defaultExpDays,
		maxDataSize:    maxDataSize,
		maxPerIP:       5,
		ipWindow:       time.Hour,
	}
}

type CreatePregnancyRequest struct {
	EDD      string `json:"edd" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreatePregnancyResponse struct {
	ID         string     `json:"id"`
	CreatorKey string     `json:"creator_key"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type UpdatePregnancyRequest struct {
	Action     string          `json:"action"` // "update_data", "update_edd", "extend"
	CreatorKey string          `json:"creator_key"`
	Data       json.RawMessage `json:"data"`
	EDD        string          `json:"edd"`
	ExpiresIn  int             `json:"expires_in"` // days
}

func (h *PregnancyHandler) Create(c *gin.Context) {
	var req CreatePregnancyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if len(req.Password) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要4个字符", "code": 400})
		return
	}

	// Validate EDD format
	if _, err := time.Parse("2006-01-02", req.EDD); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "预产期格式无效，请使用 YYYY-MM-DD", "code": 400})
		return
	}

	// Check if password already in use
	pwIndex := passwordIndex(req.Password)
	existing, _ := h.db.GetPregnancyProfileByPasswordIndex(pwIndex)

	// Fallback: check legacy profiles without password_index
	if existing == nil {
		legacyProfiles, _ := h.db.GetPregnancyProfilesWithoutIndex()
		for _, p := range legacyProfiles {
			if utils.VerifyPassword(req.Password, p.Password) {
				existing = p
				h.db.UpdatePregnancyProfilePasswordIndex(p.ID, pwIndex)
				break
			}
		}
	}

	if existing != nil {
		// Check if expired
		if existing.ExpiresAt != nil && time.Now().After(*existing.ExpiresAt) {
			h.db.DeletePregnancyProfile(existing.ID)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "该密码已被使用，请直接登录或使用其他密码", "code": 409})
			return
		}
	}

	ip := c.ClientIP()

	// IP rate limiting
	count, err := h.db.CountPregnancyProfilesByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	creatorKey := generateKey()

	hashedCreatorKey, err := utils.HashPassword(creatorKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密钥生成失败", "code": 500})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败", "code": 500})
		return
	}

	// Initialize default data
	defaultData := getDefaultPregnancyData()
	dataBytes, _ := json.Marshal(defaultData)

	exp := time.Now().Add(time.Duration(h.defaultExpDays) * 24 * time.Hour)

	profile := &models.PregnancyProfile{
		EDD:           req.EDD,
		Password:      hashedPassword,
		PasswordIndex: pwIndex,
		CreatorKey:    hashedCreatorKey,
		Data:          string(dataBytes),
		ExpiresAt:     &exp,
		CreatorIP:     ip,
	}

	if err := h.db.CreatePregnancyProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, CreatePregnancyResponse{
		ID:         profile.ID,
		CreatorKey: creatorKey,
		ExpiresAt:  profile.ExpiresAt,
	})
}

func (h *PregnancyHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := c.Query("password")

	profile, err := h.db.GetPregnancyProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	// Check expiration
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		h.db.DeletePregnancyProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         profile.ID,
		"edd":        profile.EDD,
		"data":       json.RawMessage(profile.Data),
		"expires_at": profile.ExpiresAt,
		"created_at": profile.CreatedAt,
		"updated_at": profile.UpdatedAt,
	})
}

func (h *PregnancyHandler) GetByCreator(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	profile, err := h.db.GetPregnancyProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "创建者密钥无效", "code": 401})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         profile.ID,
		"edd":        profile.EDD,
		"data":       json.RawMessage(profile.Data),
		"expires_at": profile.ExpiresAt,
		"created_at": profile.CreatedAt,
		"updated_at": profile.UpdatedAt,
	})
}

func (h *PregnancyHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdatePregnancyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	profile, err := h.db.GetPregnancyProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	// Auth check
	if req.CreatorKey == "" || !utils.VerifyPassword(req.CreatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	switch req.Action {
	case "update_data":
		if req.Data == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少数据", "code": 400})
			return
		}
		dataStr := string(req.Data)
		if len(dataStr) > h.maxDataSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "数据超过大小限制", "code": 400})
			return
		}
		if err := h.db.UpdatePregnancyProfileData(id, dataStr); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
			return
		}

	case "update_edd":
		if req.EDD == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少预产期", "code": 400})
			return
		}
		if _, err := time.Parse("2006-01-02", req.EDD); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "预产期格式无效", "code": 400})
			return
		}
		if err := h.db.UpdatePregnancyProfileEDD(id, req.EDD); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
			return
		}

	case "extend":
		days := req.ExpiresIn
		if days <= 0 {
			days = h.defaultExpDays
		}
		if days > 730 {
			days = 730
		}
		newExpires := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		if err := h.db.ExtendPregnancyProfile(id, &newExpires); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "延期失败", "code": 500})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "expires_at": newExpires})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的操作", "code": 400})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *PregnancyHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	profile, err := h.db.GetPregnancyProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	h.db.DeletePregnancyProfile(profile.ID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type LoginPregnancyRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *PregnancyHandler) Login(c *gin.Context) {
	var req LoginPregnancyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码", "code": 400})
		return
	}

	pwIndex := passwordIndex(req.Password)
	profile, err := h.db.GetPregnancyProfileByPasswordIndex(pwIndex)

	// Fallback: scan legacy profiles without password_index
	if err != nil {
		legacyProfiles, _ := h.db.GetPregnancyProfilesWithoutIndex()
		for _, p := range legacyProfiles {
			if utils.VerifyPassword(req.Password, p.Password) {
				profile = p
				// Backfill password_index for future fast lookups
				h.db.UpdatePregnancyProfilePasswordIndex(p.ID, pwIndex)
				break
			}
		}
	}

	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的档案", "code": 404})
		return
	}

	// Check expiration
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		h.db.DeletePregnancyProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         profile.ID,
		"edd":        profile.EDD,
		"data":       json.RawMessage(profile.Data),
		"expires_at": profile.ExpiresAt,
		"created_at": profile.CreatedAt,
		"updated_at": profile.UpdatedAt,
	})
}

// getDefaultPregnancyData returns the initial data structure with preset items
func getDefaultPregnancyData() map[string]interface{} {
	return map[string]interface{}{
		"hospital_bag": map[string]interface{}{
			"mom": []map[string]interface{}{
				{"name": "身份证", "checked": false},
				{"name": "医保卡/就诊卡", "checked": false},
				{"name": "母子健康手册", "checked": false},
				{"name": "产检资料", "checked": false},
				{"name": "哺乳内衣（2-3件）", "checked": false},
				{"name": "一次性内裤（多条）", "checked": false},
				{"name": "产褥垫", "checked": false},
				{"name": "卫生巾（产后专用）", "checked": false},
				{"name": "月子帽", "checked": false},
				{"name": "月子鞋/拖鞋", "checked": false},
				{"name": "吸管杯/水杯", "checked": false},
				{"name": "餐具套装", "checked": false},
				{"name": "洗漱用品", "checked": false},
				{"name": "毛巾（2-3条）", "checked": false},
				{"name": "手机充电器", "checked": false},
			},
			"baby": []map[string]interface{}{
				{"name": "新生儿衣服（2-3套）", "checked": false},
				{"name": "包被/抱被", "checked": false},
				{"name": "纸尿裤（NB码）", "checked": false},
				{"name": "湿巾", "checked": false},
				{"name": "小帽子", "checked": false},
				{"name": "小袜子", "checked": false},
				{"name": "奶瓶（1-2个）", "checked": false},
			},
			"documents": []map[string]interface{}{
				{"name": "夫妻双方身份证", "checked": false},
				{"name": "户口本", "checked": false},
				{"name": "结婚证", "checked": false},
				{"name": "医保卡", "checked": false},
				{"name": "准生证/生育服务单", "checked": false},
				{"name": "现金/银行卡", "checked": false},
			},
			"other": []map[string]interface{}{
				{"name": "零食/巧克力", "checked": false},
				{"name": "相机/手机", "checked": false},
				{"name": "纸笔", "checked": false},
			},
		},
		"baby_essentials": map[string]interface{}{
			"feeding": []map[string]interface{}{
				{"name": "奶瓶（宽口径2-3个）", "checked": false},
				{"name": "奶瓶刷", "checked": false},
				{"name": "奶瓶消毒器", "checked": false},
				{"name": "温奶器", "checked": false},
				{"name": "奶粉（备用1罐）", "checked": false},
				{"name": "吸奶器", "checked": false},
				{"name": "储奶袋", "checked": false},
				{"name": "防溢乳垫", "checked": false},
			},
			"diaper": []map[string]interface{}{
				{"name": "纸尿裤NB码（2包）", "checked": false},
				{"name": "纸尿裤S码（1包）", "checked": false},
				{"name": "湿巾（多包）", "checked": false},
				{"name": "护臀膏", "checked": false},
			},
			"clothing": []map[string]interface{}{
				{"name": "连体衣（4-6件）", "checked": false},
				{"name": "和尚服（2-3件）", "checked": false},
				{"name": "帽子（2个）", "checked": false},
				{"name": "袜子（3-4双）", "checked": false},
				{"name": "口水巾/围嘴（5-6条）", "checked": false},
			},
			"bathing": []map[string]interface{}{
				{"name": "婴儿浴盆", "checked": false},
				{"name": "婴儿沐浴露", "checked": false},
				{"name": "婴儿润肤露", "checked": false},
				{"name": "浴巾（2条）", "checked": false},
				{"name": "小方巾（多条）", "checked": false},
				{"name": "棉签", "checked": false},
				{"name": "婴儿指甲剪", "checked": false},
			},
			"bedding": []map[string]interface{}{
				{"name": "婴儿床", "checked": false},
				{"name": "床垫", "checked": false},
				{"name": "隔尿垫（多条）", "checked": false},
				{"name": "小被子", "checked": false},
				{"name": "睡袋", "checked": false},
			},
			"outdoor": []map[string]interface{}{
				{"name": "婴儿推车", "checked": false},
				{"name": "安全座椅", "checked": false},
				{"name": "背带/腰凳", "checked": false},
				{"name": "妈咪包", "checked": false},
			},
		},
		"prenatal_checks": []map[string]interface{}{
			{"week": "6-8", "name": "首次产检", "items": "确认宫内妊娠、胎心胎芽、血常规、尿常规、血型、肝肾功能", "done": false, "notes": ""},
			{"week": "11-13", "name": "NT检查", "items": "NT超声筛查、早期唐筛（可选）", "done": false, "notes": ""},
			{"week": "15-20", "name": "唐氏筛查", "items": "中期唐筛/无创DNA/羊水穿刺", "done": false, "notes": ""},
			{"week": "20-24", "name": "大排畸", "items": "系统B超（三维/四维）、胎儿结构筛查", "done": false, "notes": ""},
			{"week": "24-28", "name": "糖耐量检查", "items": "OGTT糖耐量试验、血常规", "done": false, "notes": ""},
			{"week": "28-30", "name": "常规产检", "items": "血压、体重、宫高腹围、胎心", "done": false, "notes": ""},
			{"week": "30-32", "name": "小排畸", "items": "B超检查胎儿发育、胎位", "done": false, "notes": ""},
			{"week": "32-34", "name": "常规产检", "items": "血压、体重、胎心监护", "done": false, "notes": ""},
			{"week": "34-36", "name": "GBS筛查", "items": "B族链球菌筛查、胎心监护、血常规", "done": false, "notes": ""},
			{"week": "36", "name": "足月产检", "items": "胎心监护、B超评估、骨盆测量", "done": false, "notes": ""},
			{"week": "37", "name": "每周产检", "items": "胎心监护、宫颈检查", "done": false, "notes": ""},
			{"week": "38", "name": "每周产检", "items": "胎心监护、B超评估羊水", "done": false, "notes": ""},
			{"week": "39", "name": "每周产检", "items": "胎心监护、评估分娩方式", "done": false, "notes": ""},
			{"week": "40", "name": "预产期检查", "items": "胎心监护、超声评估、讨论催产计划", "done": false, "notes": ""},
		},
		"weight_records":  []interface{}{},
		"fetal_movements": []interface{}{},
	}
}
