package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

type GlucoseHandler struct {
	db             *models.DB
	defaultExpDays int
	maxPerIP       int
	ipWindow       time.Duration
	cfg            *config.Config
}

func NewGlucoseHandler(db *models.DB, cfg *config.Config) *GlucoseHandler {
	glucoseCfg := cfg.Glucose
	if glucoseCfg.DefaultExpiresDays <= 0 {
		glucoseCfg.DefaultExpiresDays = 365
	}
	return &GlucoseHandler{
		db:             db,
		defaultExpDays: glucoseCfg.DefaultExpiresDays,
		maxPerIP:       5,
		ipWindow:       time.Hour,
		cfg:            cfg,
	}
}

func glucosePasswordIndex(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

// getPassword 获取密码，支持从 Query 或 Header 获取（复用 expense.go 中的函数）

// Request/Response types

type CreateGlucoseRequest struct {
	Password string `json:"password" binding:"required"`
}

type CreateGlucoseResponse struct {
	ID         string     `json:"id"`
	CreatorKey string    `json:"creator_key"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

type LoginGlucoseRequest struct {
	Password string `json:"password" binding:"required"`
}

type GlucoseProfileResponse struct {
	ID        string     `json:"id"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateRecordRequest struct {
	Value       float64 `json:"value" binding:"required"`
	MeasureType string  `json:"measure_type"`
	Time        string  `json:"time"`
	Note        string  `json:"note"`
	VoiceText   string  `json:"voice_text"`
	Tags        string  `json:"tags"`
	Food        string  `json:"food"`
	Exercise    string  `json:"exercise"`
	Medication  string  `json:"medication"`
	Sleep       string  `json:"sleep"`
	Mood        string  `json:"mood"`
}

type UpdateRecordRequest struct {
	Value       float64 `json:"value"`
	MeasureType string  `json:"measure_type"`
	Time        string  `json:"time"`
	Note        string  `json:"note"`
	VoiceText   string  `json:"voice_text"`
	Tags        string  `json:"tags"`
	Food        string  `json:"food"`
	Exercise    string  `json:"exercise"`
	Medication  string  `json:"medication"`
	Sleep       string  `json:"sleep"`
	Mood        string  `json:"mood"`
}

type ImportRecordsRequest struct {
	Records []CreateRecordRequest `json:"records" binding:"required"`
}

type GlucoseVoiceParseRequest struct {
	Text string `json:"text" binding:"required"`
}

type GlucoseVoiceParseResponse struct {
	Value       float64 `json:"value"`
	MeasureType string  `json:"measure_type"`
	Time        string  `json:"time"`
	Food        string  `json:"food"`
	Exercise    string  `json:"exercise"`
	Medication  string  `json:"medication"`
	Note        string  `json:"note"`
	Confidence  float64 `json:"confidence"`
}

// Handlers

func (h *GlucoseHandler) Create(c *gin.Context) {
	var req CreateGlucoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if len(req.Password) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要4个字符", "code": 400})
		return
	}

	// Check if password already in use
	pwIndex := glucosePasswordIndex(req.Password)
	existing, _ := h.db.GetGlucoseProfileByPasswordIndex(pwIndex)

	if existing != nil {
		if existing.ExpiresAt != nil && time.Now().After(*existing.ExpiresAt) {
			h.db.DeleteGlucoseProfile(existing.ID)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "该密码已被使用，请直接登录或使用其他密码", "code": 409})
			return
		}
	}

	ip := c.ClientIP()

	// IP rate limiting
	count, err := h.db.CountGlucoseProfilesByIP(ip, time.Now().Add(-h.ipWindow))
	if err == nil && count >= h.maxPerIP {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "创建过于频繁，请稍后再试", "code": 429})
		return
	}

	creatorKey := utils.GenerateHexKey(16)
	hashedCreatorKey, _ := utils.HashPassword(creatorKey)
	hashedPassword, _ := utils.HashPassword(req.Password)

	exp := time.Now().Add(time.Duration(h.defaultExpDays) * 24 * time.Hour)

	profile := &models.GlucoseProfile{
		Password:      hashedPassword,
		PasswordIndex: pwIndex,
		CreatorKey:    hashedCreatorKey,
		ExpiresAt:     &exp,
		CreatorIP:     ip,
	}

	if err := h.db.CreateGlucoseProfile(profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, CreateGlucoseResponse{
		ID:         profile.ID,
		CreatorKey: creatorKey,
		ExpiresAt:  profile.ExpiresAt,
	})
}

func (h *GlucoseHandler) Login(c *gin.Context) {
	var req LoginGlucoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入密码", "code": 400})
		return
	}

	pwIndex := glucosePasswordIndex(req.Password)
	profile, err := h.db.GetGlucoseProfileByPasswordIndex(pwIndex)

	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的档案", "code": 404})
		return
	}

	// Check expiration
	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		h.db.DeleteGlucoseProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	c.JSON(http.StatusOK, GlucoseProfileResponse{
		ID:        profile.ID,
		ExpiresAt: profile.ExpiresAt,
		CreatedAt: profile.CreatedAt,
	})
}

func (h *GlucoseHandler) Get(c *gin.Context) {
	id := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetGlucoseProfile(id)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if profile.ExpiresAt != nil && time.Now().After(*profile.ExpiresAt) {
		h.db.DeleteGlucoseProfile(profile.ID)
		c.JSON(http.StatusGone, gin.H{"error": "该档案已过期", "code": 410})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	c.JSON(http.StatusOK, GlucoseProfileResponse{
		ID:        profile.ID,
		ExpiresAt: profile.ExpiresAt,
		CreatedAt: profile.CreatedAt,
	})
}

func (h *GlucoseHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")

	profile, err := h.db.GetGlucoseProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	h.db.DeleteGlucoseProfile(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *GlucoseHandler) Extend(c *gin.Context) {
	id := c.Param("id")
	creatorKey := c.Query("creator_key")
	days := c.DefaultQuery("days", "365")

	profile, err := h.db.GetGlucoseProfile(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if creatorKey == "" || !utils.VerifyPassword(creatorKey, profile.CreatorKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无权限操作", "code": 401})
		return
	}

	daysInt := 365
	fmt.Sscanf(days, "%d", &daysInt)
	if daysInt > 730 {
		daysInt = 730
	}

	newExpires := time.Now().Add(time.Duration(daysInt) * 24 * time.Hour)
	if err := h.db.ExtendGlucoseProfile(id, &newExpires); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "延期失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "expires_at": newExpires})
}

// Record handlers

func (h *GlucoseHandler) GetRecords(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	records, err := h.db.GetGlucoseRecords(profileID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, records)
}

func (h *GlucoseHandler) CreateRecord(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	// Validate value
	if req.Value < 1 || req.Value > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "血糖值应在 1-50 mmol/L 之间", "code": 400})
		return
	}

	record := &models.GlucoseRecord{
		ProfileID:   profileID,
		Value:       req.Value,
		MeasureType: req.MeasureType,
		Note:        req.Note,
		VoiceText:   req.VoiceText,
		Tags:        req.Tags,
		Food:        req.Food,
		Exercise:    req.Exercise,
		Medication:  req.Medication,
		Sleep:       req.Sleep,
		Mood:        req.Mood,
	}

	// Parse time
	if req.Time != "" {
		t, err := time.Parse(time.RFC3339, req.Time)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "时间格式错误，应为 ISO 8601 格式", "code": 400})
			return
		}
		record.Time = t
	} else {
		record.Time = time.Now()
	}

	if record.MeasureType == "" {
		record.MeasureType = "random"
	}

	if err := h.db.CreateGlucoseRecord(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}

	// 记录创建历史
	ip := c.ClientIP()
	h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
		RecordID:   record.ID,
		ProfileID:  profileID,
		Action:     "create",
		FieldName:  "all",
		OldValue:   "",
		NewValue:   fmt.Sprintf("血糖 %.1f mmol/L, %s", record.Value, record.MeasureType),
		ChangeDesc: fmt.Sprintf("创建血糖记录：%.1f mmol/L (%s)", record.Value, record.MeasureType),
		IPAddress:  ip,
	})

	c.JSON(http.StatusCreated, record)
}

func (h *GlucoseHandler) UpdateRecord(c *gin.Context) {
	profileID := c.Param("id")
	recordID := c.Param("recordId")
	password := getPassword(c)

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	oldRecord, err := h.db.GetGlucoseRecord(recordID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该记录", "code": 404})
		return
	}

	var req UpdateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	record := &models.GlucoseRecord{
		ID:          recordID,
		ProfileID:   profileID,
		Value:       oldRecord.Value,
		MeasureType: oldRecord.MeasureType,
		Time:        oldRecord.Time,
		Note:        oldRecord.Note,
		Tags:        oldRecord.Tags,
		Food:        oldRecord.Food,
		Exercise:    oldRecord.Exercise,
		Medication:  oldRecord.Medication,
		Sleep:       oldRecord.Sleep,
		Mood:        oldRecord.Mood,
	}

	ip := c.ClientIP()
	changes := []string{}

	// Apply updates and record history
	if req.Value > 0 && req.Value != oldRecord.Value {
		oldVal := oldRecord.Value
		record.Value = req.Value
		changes = append(changes, fmt.Sprintf("血糖值: %.1f → %.1f", oldVal, req.Value))
		// 创建历史记录
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "value",
			OldValue:   fmt.Sprintf("%.1f", oldVal),
			NewValue:   fmt.Sprintf("%.1f", req.Value),
			ChangeDesc: fmt.Sprintf("血糖值从 %.1f 改为 %.1f", oldVal, req.Value),
			IPAddress:  ip,
		})
	}
	if req.MeasureType != "" && req.MeasureType != oldRecord.MeasureType {
		oldVal := oldRecord.MeasureType
		record.MeasureType = req.MeasureType
		changes = append(changes, fmt.Sprintf("测量类型: %s → %s", oldVal, req.MeasureType))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "measure_type",
			OldValue:   oldVal,
			NewValue:   req.MeasureType,
			ChangeDesc: fmt.Sprintf("测量类型从 %s 改为 %s", oldVal, req.MeasureType),
			IPAddress:  ip,
		})
	}
	if req.Time != "" {
		t, err := time.Parse(time.RFC3339, req.Time)
		if err == nil && !t.Equal(oldRecord.Time) {
			oldVal := oldRecord.Time.Format("2006-01-02 15:04")
			record.Time = t
			newVal := t.Format("2006-01-02 15:04")
			changes = append(changes, fmt.Sprintf("时间: %s → %s", oldVal, newVal))
			h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
				RecordID:   recordID,
				ProfileID:  profileID,
				Action:     "update",
				FieldName:  "time",
				OldValue:   oldVal,
				NewValue:   newVal,
				ChangeDesc: fmt.Sprintf("时间从 %s 改为 %s", oldVal, newVal),
				IPAddress:  ip,
			})
		}
	}
	if req.Note != "" && req.Note != oldRecord.Note {
		oldVal := oldRecord.Note
		record.Note = req.Note
		changes = append(changes, fmt.Sprintf("备注: %s → %s", oldVal, req.Note))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "note",
			OldValue:   oldVal,
			NewValue:   req.Note,
			ChangeDesc: fmt.Sprintf("备注从 '%s' 改为 '%s'", oldVal, req.Note),
			IPAddress:  ip,
		})
	}
	if req.Tags != "" && req.Tags != oldRecord.Tags {
		oldVal := oldRecord.Tags
		record.Tags = req.Tags
		changes = append(changes, fmt.Sprintf("标签: %s → %s", oldVal, req.Tags))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "tags",
			OldValue:   oldVal,
			NewValue:   req.Tags,
			ChangeDesc: fmt.Sprintf("标签从 '%s' 改为 '%s'", oldVal, req.Tags),
			IPAddress:  ip,
		})
	}
	if req.Food != "" && req.Food != oldRecord.Food {
		oldVal := oldRecord.Food
		record.Food = req.Food
		changes = append(changes, fmt.Sprintf("饮食: %s → %s", oldVal, req.Food))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "food",
			OldValue:   oldVal,
			NewValue:   req.Food,
			ChangeDesc: fmt.Sprintf("饮食从 '%s' 改为 '%s'", oldVal, req.Food),
			IPAddress:  ip,
		})
	}
	if req.Exercise != "" && req.Exercise != oldRecord.Exercise {
		oldVal := oldRecord.Exercise
		record.Exercise = req.Exercise
		changes = append(changes, fmt.Sprintf("运动: %s → %s", oldVal, req.Exercise))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "exercise",
			OldValue:   oldVal,
			NewValue:   req.Exercise,
			ChangeDesc: fmt.Sprintf("运动从 '%s' 改为 '%s'", oldVal, req.Exercise),
			IPAddress:  ip,
		})
	}
	if req.Medication != "" && req.Medication != oldRecord.Medication {
		oldVal := oldRecord.Medication
		record.Medication = req.Medication
		changes = append(changes, fmt.Sprintf("药物: %s → %s", oldVal, req.Medication))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "medication",
			OldValue:   oldVal,
			NewValue:   req.Medication,
			ChangeDesc: fmt.Sprintf("药物从 '%s' 改为 '%s'", oldVal, req.Medication),
			IPAddress:  ip,
		})
	}
	if req.Sleep != "" && req.Sleep != oldRecord.Sleep {
		oldVal := oldRecord.Sleep
		record.Sleep = req.Sleep
		changes = append(changes, fmt.Sprintf("睡眠: %s → %s", oldVal, req.Sleep))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "sleep",
			OldValue:   oldVal,
			NewValue:   req.Sleep,
			ChangeDesc: fmt.Sprintf("睡眠从 '%s' 改为 '%s'", oldVal, req.Sleep),
			IPAddress:  ip,
		})
	}
	if req.Mood != "" && req.Mood != oldRecord.Mood {
		oldVal := oldRecord.Mood
		record.Mood = req.Mood
		changes = append(changes, fmt.Sprintf("情绪: %s → %s", oldVal, req.Mood))
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "update",
			FieldName:  "mood",
			OldValue:   oldVal,
			NewValue:   req.Mood,
			ChangeDesc: fmt.Sprintf("情绪从 '%s' 改为 '%s'", oldVal, req.Mood),
			IPAddress:  ip,
		})
	}

	if len(changes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的内容", "code": 400})
		return
	}

	if err := h.db.UpdateGlucoseRecord(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"record":  record,
		"changes":  changes,
		"message":  "更新成功",
	})
}

func (h *GlucoseHandler) DeleteRecord(c *gin.Context) {
	profileID := c.Param("id")
	recordID := c.Param("recordId")
	password := getPassword(c)

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	// 获取记录信息用于记录历史
	record, _ := h.db.GetGlucoseRecord(recordID)

	if err := h.db.DeleteGlucoseRecord(recordID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}

	// 记录删除历史
	if record != nil {
		ip := c.ClientIP()
		h.db.CreateGlucoseRecordHistory(&models.GlucoseRecordHistory{
			RecordID:   recordID,
			ProfileID:  profileID,
			Action:     "delete",
			FieldName:  "all",
			OldValue:   fmt.Sprintf("血糖 %.1f mmol/L, %s", record.Value, record.MeasureType),
			NewValue:   "",
			ChangeDesc: fmt.Sprintf("删除血糖记录：%.1f mmol/L (%s)", record.Value, record.MeasureType),
			IPAddress:  ip,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Import records
func (h *GlucoseHandler) ImportRecords(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req ImportRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据", "code": 400})
		return
	}

	if len(req.Records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有要导入的记录", "code": 400})
		return
	}

	// Limit import count
	if len(req.Records) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次导入最多500条记录", "code": 400})
		return
	}

	records := make([]*models.GlucoseRecord, 0, len(req.Records))
	for _, r := range req.Records {
		if r.Value < 1 || r.Value > 50 {
			continue
		}
		record := &models.GlucoseRecord{
			ProfileID:   profileID,
			Value:       r.Value,
			MeasureType: r.MeasureType,
			Note:        r.Note,
			Tags:        r.Tags,
			Food:        r.Food,
			Exercise:    r.Exercise,
			Medication:  r.Medication,
			Sleep:       r.Sleep,
			Mood:        r.Mood,
		}

		if r.Time != "" {
			t, err := time.Parse(time.RFC3339, r.Time)
			if err == nil {
				record.Time = t
			} else {
				record.Time = time.Now()
			}
		} else {
			record.Time = time.Now()
		}

		if record.MeasureType == "" {
			record.MeasureType = "random"
		}

		records = append(records, record)
	}

	if err := h.db.BatchCreateGlucoseRecords(records); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导入失败", "code": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"imported":     len(records),
		"total":        len(req.Records),
	})
}

// Stats
func (h *GlucoseHandler) GetStats(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	// Default to last 30 days if not specified
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	stats, err := h.db.GetGlucoseStats(profileID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GlucoseVoiceParse 智能解析
func (h *GlucoseHandler) VoiceParse(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)
	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	var req GlucoseVoiceParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文本内容", "code": 400})
		return
	}

	// 优先使用 MiniMax，其次 DeepSeek
	apiKey := h.cfg.MiniMax.APIKey
	if apiKey == "" {
		apiKey = h.cfg.DeepSeek.APIKey
	}

	if apiKey == "" {
		// Fallback to basic parsing
		result := basicGlucoseParse(req.Text)
		c.JSON(http.StatusOK, result)
		return
	}

	// Use MiniMax for intelligent parsing
	result, err := h.callMiniMaxParse(req.Text)
	if err != nil {
		log.Printf("MiniMax parse failed: %v, using basic parse", err)
		result = basicGlucoseParse(req.Text)
		result.Confidence = 0.3
		c.JSON(http.StatusOK, result)
		return
	}

	log.Printf("VoiceParse returning: %+v", result)
	c.JSON(http.StatusOK, result)
}

func (h *GlucoseHandler) callMiniMaxParse(text string) (*GlucoseVoiceParseResponse, error) {
	// 优先使用 MiniMax，其次 DeepSeek
	apiKey := h.cfg.MiniMax.APIKey
	model := h.cfg.MiniMax.Model
	if apiKey == "" {
		apiKey = h.cfg.DeepSeek.APIKey
		model = h.cfg.DeepSeek.Model
	}
	if model == "" {
		model = "abab6.5s-chat"
	}

	prompt := fmt.Sprintf(`从以下文本提取血糖记录。只返回一个血糖值。

文本：%s

规则：
1. 血糖值单位是mmol/L
2. 优先提取餐后2小时血糖，如果没有就提取空腹血糖
3. 只返回JSON：{"value":数字,"measure_type":"fasting或2h或random","food":"食物(无则空)","confidence":0-1}

示例输入：今天空腹血糖5.6
示例输出：{"value":5.6,"measure_type":"fasting","food":"","confidence":0.95}

直接输出JSON，不要解释。`, text)

	url := "https://api.minimaxi.com/anthropic/v1/messages"
	reqBody := map[string]interface{}{
		"model":       model,
		"max_tokens":  1024,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个专业的血糖数据解析助手，擅长从自然语言中提取结构化的血糖检测记录。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误: %s", string(respBody))
	}

	// Debug: log response
	log.Printf("MiniMax Response: %s\n", string(respBody))

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	var content string

	// MiniMax returns content as an array: [{type: "thinking", ...}, {type: "text", text: "..."}]
	// We need to find the element with type "text"
	log.Printf("Result keys: %v", reflect.ValueOf(result).MapKeys())

	if contentArr, ok := result["content"].([]interface{}); ok && len(contentArr) > 0 {
		log.Printf("Content array length: %d", len(contentArr))
		for i, item := range contentArr {
			log.Printf("Content[%d] type: %T", i, item)
			if c, ok := item.(map[string]interface{}); ok {
				log.Printf("Content[%d] map keys: %v", i, c)
				if ctype, ok := c["type"].(string); ok {
					log.Printf("Content[%d] type field: %s", i, ctype)
					if ctype == "text" {
						if ctext, ok := c["text"].(string); ok {
							content = ctext
							log.Printf("Found text content: %s", content)
							break
						}
					}
				}
			}
		}
	}

	log.Printf("Extracted content: %s\n", content)

	if content == "" {
		return nil, fmt.Errorf("API响应格式错误，无法提取内容")
	}

	// Extract JSON from response
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Find JSON in content - use simple string search
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("未找到JSON, content: %s", content)
	}
	jsonMatch := content[start : end+1]

	log.Printf("JSON match: %s\n", jsonMatch)

	var parseResult GlucoseVoiceParseResponse
	if err := json.Unmarshal([]byte(jsonMatch), &parseResult); err != nil {
		return nil, fmt.Errorf("解析结果失败: %v, json: %s", err, jsonMatch)
	}

	log.Printf("Parse result: %+v\n", parseResult)
	return &parseResult, nil
}

// GetRecordHistory 获取单条记录的历史
func (h *GlucoseHandler) GetRecordHistory(c *gin.Context) {
	profileID := c.Param("id")
	recordID := c.Param("recordId")
	password := getPassword(c)

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	// Verify record belongs to profile
	record, err := h.db.GetGlucoseRecord(recordID)
	if err != nil || record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该记录", "code": 404})
		return
	}

	if record.ProfileID != profileID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限查看", "code": 403})
		return
	}

	histories, err := h.db.GetGlucoseRecordHistory(recordID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取历史失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, histories)
}

// GetProfileHistory 获取档案的所有历史记录
func (h *GlucoseHandler) GetProfileHistory(c *gin.Context) {
	profileID := c.Param("id")
	password := getPassword(c)
	limit := c.DefaultQuery("limit", "100")

	profile, err := h.db.GetGlucoseProfile(profileID)
	if err != nil || profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该档案", "code": 404})
		return
	}

	if password == "" || !utils.VerifyPassword(password, profile.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误", "code": 401})
		return
	}

	limitInt := 100
	fmt.Sscanf(limit, "%d", &limitInt)

	histories, err := h.db.GetGlucoseProfileHistory(profileID, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取历史失败", "code": 500})
		return
	}

	c.JSON(http.StatusOK, histories)
}

func basicGlucoseParse(text string) *GlucoseVoiceParseResponse {
	result := &GlucoseVoiceParseResponse{
		Value:       0,
		MeasureType: "random",
		Time:        "",
		Food:        "",
		Exercise:    "",
		Medication:  "",
		Note:        "",
		Confidence:  0.5,
	}

	lowerText := strings.ToLower(text)

	// Parse glucose value
	patterns := []string{
		`血糖[:：]?\s*(\d+\.?\d*)`,
		`血糖值[:：]?\s*(\d+\.?\d*)`,
		`血糖是[:：]?\s*(\d+\.?\d*)`,
		`(\d+\.?\d*)\s*mmol`,
		`[:：，,]\s*(\d+\.?\d*)\s*(?:mmol|L)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			value, _ := strconv.ParseFloat(matches[1], 64)
			if value >= 1 && value <= 50 {
				result.Value = value
				break
			}
		}
	}

	// Detect measure type
	if strings.Contains(lowerText, "空腹") || strings.Contains(lowerText, "早上") || strings.Contains(lowerText, "早餐") {
		result.MeasureType = "fasting"
	} else if strings.Contains(lowerText, "餐后1") || strings.Contains(lowerText, "1小时") {
		result.MeasureType = "1h"
	} else if strings.Contains(lowerText, "餐后2") || strings.Contains(lowerText, "2小时") || strings.Contains(lowerText, "饭后") {
		result.MeasureType = "2h"
	} else if strings.Contains(lowerText, "睡前") || strings.Contains(lowerText, "晚上") {
		result.MeasureType = "bedtime"
	} else if strings.Contains(lowerText, "凌晨") {
		result.MeasureType = "dawn"
	}

	// Detect time
	if strings.Contains(lowerText, "昨天") {
		yesterday := time.Now().AddDate(0, 0, -1)
		result.Time = yesterday.Format(time.RFC3339)
	} else if strings.Contains(lowerText, "前天") {
		dayBefore := time.Now().AddDate(0, 0, -2)
		result.Time = dayBefore.Format(time.RFC3339)
	} else if strings.Contains(lowerText, "今天") {
		result.Time = time.Now().Format(time.RFC3339)
	}

	// Detect exercise
	exerciseKeywords := map[string]string{
		"散步": "walk",
		"慢跑": "jog",
		"快走": "fast_walk",
		"瑜伽": "yoga",
		"游泳": "swim",
		"骑行": "cycling",
		"打球": "sports",
	}
	for k, v := range exerciseKeywords {
		if strings.Contains(lowerText, k) {
			result.Exercise = v
			break
		}
	}

	// Detect medication
	medKeywords := []string{"二甲双胍", "胰岛素", "降糖", "药物", "服药", "吃药"}
	for _, k := range medKeywords {
		if strings.Contains(lowerText, k) {
			result.Medication = k
			break
		}
	}

	return result
}
