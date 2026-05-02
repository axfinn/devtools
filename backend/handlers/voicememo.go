package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

type VoiceMemoHandler struct {
	db             *models.DB
	plannerHandler *PlannerHandler
	asrServiceURL  string
	uploadDir      string
}

func NewVoiceMemoHandler(db *models.DB, plannerHandler *PlannerHandler, asrServiceURL string) *VoiceMemoHandler {
	uploadDir := "./data/voicememos"
	os.MkdirAll(uploadDir, 0755)
	return &VoiceMemoHandler{
		db:             db,
		plannerHandler: plannerHandler,
		asrServiceURL:  strings.TrimSpace(asrServiceURL),
		uploadDir:      uploadDir,
	}
}

type createPlannerTaskFromMemoRequest struct {
	ProfileID   string `json:"profile_id" binding:"required"`
	Kind        string `json:"kind"`
	EntryType   string `json:"entry_type"`
	Bucket      string `json:"bucket"`
	Title       string `json:"title"`
	Transcript  string `json:"transcript"`
	Detail      string `json:"detail"`
	Notes       string `json:"notes"`
	PlannedFor  string `json:"planned_for"`
	Priority    string `json:"priority"`
	Intent      string `json:"intent"`
	EnergyLevel string `json:"energy_level"`
}

func currentVoiceMemoDeviceID(c *gin.Context) string {
	deviceID := strings.TrimSpace(c.Query("device_id"))
	if deviceID == "" {
		deviceID = strings.TrimSpace(c.GetHeader("X-Device-ID"))
	}
	if deviceID == "" {
		deviceID = c.ClientIP()
	}
	return deviceID
}

func (h *VoiceMemoHandler) requirePlannerProfile(c *gin.Context, profileID string) (*models.PlannerProfile, bool) {
	if strings.TrimSpace(profileID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先进入事项档案"})
		return nil, false
	}
	if h.plannerHandler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "事项管理模块未启用"})
		return nil, false
	}
	return h.plannerHandler.loadProfileByAccess(c, profileID)
}

func (h *VoiceMemoHandler) canAccessMemo(c *gin.Context, memo *models.VoiceMemo, profileID string) bool {
	if memo.ProfileID != "" {
		profile, ok := h.requirePlannerProfile(c, memo.ProfileID)
		return ok && profile.ID == memo.ProfileID
	}
	if strings.TrimSpace(profileID) != "" {
		profile, ok := h.requirePlannerProfile(c, profileID)
		if !ok {
			return false
		}
		if currentVoiceMemoDeviceID(c) != memo.DeviceID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
			return false
		}
		_ = h.db.BindVoiceMemoProfile(memo.ID, profile.ID)
		memo.ProfileID = profile.ID
		return true
	}
	if memo.DeviceID != currentVoiceMemoDeviceID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return false
	}
	return true
}

func generateVoiceMemoID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *VoiceMemoHandler) publicMemoAudioURL(memoID string) string {
	return "/api/voicememo/" + memoID + "/audio"
}

func (h *VoiceMemoHandler) serializeMemo(memo *models.VoiceMemo) *models.VoiceMemo {
	if memo == nil {
		return nil
	}
	cloned := *memo
	cloned.AudioURL = h.publicMemoAudioURL(memo.ID)
	return &cloned
}

func (h *VoiceMemoHandler) serializeMemoList(items []models.VoiceMemo) []models.VoiceMemo {
	if len(items) == 0 {
		return []models.VoiceMemo{}
	}
	cloned := make([]models.VoiceMemo, 0, len(items))
	for _, item := range items {
		item.AudioURL = h.publicMemoAudioURL(item.ID)
		cloned = append(cloned, item)
	}
	return cloned
}

// Upload creates a voice memo as draft (no auto-transcribe).
func (h *VoiceMemoHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传音频文件"})
		return
	}

	if file.Size > 50*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过 50MB"})
		return
	}

	profileID := strings.TrimSpace(c.PostForm("profile_id"))
	if _, ok := h.requirePlannerProfile(c, profileID); !ok {
		return
	}
	deviceID := currentVoiceMemoDeviceID(c)

	title := c.PostForm("title")
	if title == "" {
		title = fmt.Sprintf("语音备忘 %s", time.Now().Format("01-02 15:04"))
	}

	durationSec := 0.0
	if ds := c.PostForm("duration"); ds != "" {
		fmt.Sscanf(ds, "%f", &durationSec)
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".webm"
	}
	filename := fmt.Sprintf("memo_%s_%d%s", generateVoiceMemoID(), time.Now().UnixNano(), ext)
	savePath := filepath.Join(h.uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
		return
	}

	exp := time.Now().Add(14 * 24 * time.Hour)
	memoID := generateVoiceMemoID()
	memo := &models.VoiceMemo{
		ID:          memoID,
		DeviceID:    deviceID,
		ProfileID:   profileID,
		Title:       title,
		AudioURL:    "/api/voicememo/audio/" + filename,
		DurationSec: durationSec,
		FileSize:    file.Size,
		Status:      "draft",
		ExpiresAt:   &exp,
	}

	if err := h.db.CreateVoiceMemo(memo); err != nil {
		os.Remove(savePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存记录失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           memo.ID,
		"title":        memo.Title,
		"audio_url":    h.publicMemoAudioURL(memo.ID),
		"duration_sec": memo.DurationSec,
		"file_size":    memo.FileSize,
		"status":       memo.Status,
		"profile_id":   memo.ProfileID,
		"expires_at":   memo.ExpiresAt,
		"created_at":   memo.CreatedAt,
	})
}

// Transcribe triggers ASR for a draft memo.
func (h *VoiceMemoHandler) Transcribe(c *gin.Context) {
	memo, err := h.db.GetVoiceMemo(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	if !h.canAccessMemo(c, memo, c.Query("profile_id")) {
		return
	}
	if memo.Status != "draft" && memo.Status != "failed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只有草稿或失败的备忘才能转写"})
		return
	}

	// Extract filename from audio_url
	filename := filepath.Base(memo.AudioURL)
	filePath := filepath.Join(h.uploadDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频文件不存在，可能已过期"})
		return
	}

	// Set status to transcribing
	h.db.UpdateVoiceMemoTranscript(memo.ID, "", "", "transcribing", "")

	// Run ASR in background
	go h.transcribeMemo(memo.ID, filePath, filename)

	c.JSON(http.StatusOK, gin.H{
		"id":     memo.ID,
		"status": "transcribing",
	})
}

// Update edits a memo's title, transcript, or saves it.
func (h *VoiceMemoHandler) Update(c *gin.Context) {
	memo, err := h.db.GetVoiceMemo(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	if !h.canAccessMemo(c, memo, c.Query("profile_id")) {
		return
	}

	var req struct {
		Title      string `json:"title"`
		Transcript string `json:"transcript"`
		Status     string `json:"status"` // "saved" to confirm, or "draft" to keep
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	title := memo.Title
	if req.Title != "" {
		title = req.Title
	}
	transcript := memo.Transcript
	if req.Transcript != "" {
		transcript = req.Transcript
	}
	status := memo.Status
	if req.Status == "saved" || req.Status == "draft" {
		status = req.Status
	}

	if err := h.db.UpdateVoiceMemo(memo.ID, title, transcript, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	memo.Title = title
	memo.Transcript = transcript
	memo.Status = status
	if status == "saved" {
		memo.ExpiresAt = nil
	}
	c.JSON(http.StatusOK, h.serializeMemo(memo))
}

// List returns voice memos for the current planner profile.
func (h *VoiceMemoHandler) List(c *gin.Context) {
	profileID := strings.TrimSpace(c.Query("profile_id"))
	if _, ok := h.requirePlannerProfile(c, profileID); !ok {
		return
	}
	deviceID := currentVoiceMemoDeviceID(c)

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit > 100 {
			limit = 100
		}
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	memos, total, err := h.db.ListVoiceMemos(profileID, deviceID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": h.serializeMemoList(memos),
		"total": total,
	})
}

// Get returns a single voice memo.
func (h *VoiceMemoHandler) Get(c *gin.Context) {
	memo, err := h.db.GetVoiceMemo(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	if !h.canAccessMemo(c, memo, c.Query("profile_id")) {
		return
	}
	c.JSON(http.StatusOK, h.serializeMemo(memo))
}

// Delete soft-deletes a voice memo.
func (h *VoiceMemoHandler) Delete(c *gin.Context) {
	memo, err := h.db.GetVoiceMemo(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	if !h.canAccessMemo(c, memo, c.Query("profile_id")) {
		return
	}

	if err := h.db.DeleteVoiceMemo(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// CreatePlannerTask turns a voice memo into a planner task using the active planner profile.
func (h *VoiceMemoHandler) CreatePlannerTask(c *gin.Context) {
	memo, err := h.db.GetVoiceMemo(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	if memo.Status != "completed" && memo.Status != "saved" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先完成语音转写后再转成事项"})
		return
	}
	if h.plannerHandler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "事项管理模块未启用"})
		return
	}

	var req createPlannerTaskFromMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "事项档案参数不完整"})
		return
	}
	if !h.canAccessMemo(c, memo, req.ProfileID) {
		return
	}

	profile, ok := h.plannerHandler.loadProfileByAccess(c, req.ProfileID)
	if !ok {
		return
	}

	if memo.ProfileID == "" {
		if err := h.db.BindVoiceMemoProfile(memo.ID, profile.ID); err == nil {
			memo.ProfileID = profile.ID
		}
	}
	if memo.PlannerTaskID != "" && memo.ProfileID == profile.ID {
		task, taskErr := h.db.GetPlannerTask(memo.PlannerTaskID)
		if taskErr == nil && task.ProfileID == profile.ID {
			c.JSON(http.StatusConflict, gin.H{
				"error": "该语音已同步到事项管理",
				"memo":  h.serializeMemo(memo),
				"task":  task,
			})
			return
		}
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(memo.Title)
	}
	if title == "" {
		title = "语音事项"
	}

	transcript := strings.TrimSpace(req.Transcript)
	if transcript == "" {
		transcript = strings.TrimSpace(memo.Transcript)
	}
	if transcript == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "转写内容为空，无法创建事项"})
		return
	}

	detail := strings.TrimSpace(req.Detail)
	if detail == "" {
		detail = transcript
	}

	notes := strings.TrimSpace(req.Notes)
	defaultNotes := []string{
		"来源：语音收件箱",
		"录音时间：" + memo.CreatedAt.In(time.Local).Format("2006-01-02 15:04"),
	}
	if memo.DurationSec > 0 {
		defaultNotes = append(defaultNotes, fmt.Sprintf("录音时长：%.0f 秒", memo.DurationSec))
	}
	if notes == "" {
		notes = strings.Join(defaultNotes, "\n")
	} else {
		notes = notes + "\n" + strings.Join(defaultNotes, "\n")
	}

	taskReq := createPlannerTaskRequest{
		Kind:        req.Kind,
		EntryType:   plannerFirstNonEmpty(strings.TrimSpace(req.EntryType), models.PlannerEntryTask),
		Bucket:      plannerFirstNonEmpty(strings.TrimSpace(req.Bucket), models.PlannerBucketInbox),
		Title:       title,
		Detail:      detail,
		Notes:       notes,
		PlannedFor:  strings.TrimSpace(req.PlannedFor),
		Priority:    strings.TrimSpace(req.Priority),
		RawText:     transcript,
		Intent:      strings.TrimSpace(req.Intent),
		EnergyLevel: strings.TrimSpace(req.EnergyLevel),
	}

	task, err := h.plannerHandler.buildPlannerTask(profile, taskReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.CreatePlannerTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建事项失败"})
		return
	}
	_ = h.db.CreatePlannerTaskActivity(&models.PlannerTaskActivity{
		TaskID:       task.ID,
		ProfileID:    task.ProfileID,
		ActivityType: "created",
		Title:        plannerTaskLifecycleTitle(task),
		Content:      plannerTaskLifecycleContent(task),
	})
	_ = h.db.CreatePlannerTaskActivity(&models.PlannerTaskActivity{
		TaskID:       task.ID,
		ProfileID:    task.ProfileID,
		ActivityType: "voice_imported",
		Title:        "从语音收件箱导入",
		Content:      memo.Title,
	})
	_ = h.db.CreatePlannerTaskComment(&models.PlannerTaskComment{
		TaskID:    task.ID,
		ProfileID: task.ProfileID,
		Author:    "语音收件箱",
		Content:   transcript,
	})
	if err := h.db.LinkVoiceMemoToPlanner(memo.ID, title, transcript, profile.ID, task.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "回写语音事项关联失败"})
		return
	}

	updatedMemo, err := h.db.GetVoiceMemo(memo.ID)
	if err != nil {
		updatedMemo = memo
		updatedMemo.Title = title
		updatedMemo.Transcript = transcript
		updatedMemo.Status = "saved"
		updatedMemo.ProfileID = profile.ID
		updatedMemo.PlannerTaskID = task.ID
	}

	c.JSON(http.StatusOK, gin.H{
		"memo": h.serializeMemo(updatedMemo),
		"task": task,
	})
}

// ServeMemoAudio serves a stored audio file for an authorized memo.
func (h *VoiceMemoHandler) ServeMemoAudio(c *gin.Context) {
	memo, err := h.db.GetVoiceMemo(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	if !h.canAccessMemo(c, memo, c.Query("profile_id")) {
		return
	}

	filename := filepath.Base(memo.AudioURL)
	filePath := filepath.Join(h.uploadDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.File(filePath)
}

// ── ASR Integration ──

type asrResponse struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Status   string `json:"status"`
	Error    string `json:"error"`
}

func (h *VoiceMemoHandler) transcribeMemo(memoID, filePath, originalName string) {
	if h.asrServiceURL == "" {
		h.db.UpdateVoiceMemoTranscript(memoID, "", "", "failed", "未配置 ASR 服务")
		return
	}

	result, err := h.callASR(filePath, originalName)
	if err != nil {
		log.Printf("voicememo: ASR failed for %s: %v", memoID, err)
		h.db.UpdateVoiceMemoTranscript(memoID, "", "", "failed", truncateString(err.Error(), 500))
		return
	}

	status := "completed"
	if result.Status == "failed" {
		status = "failed"
	}
	errMsg := ""
	if status == "failed" {
		errMsg = result.Error
	}

	if err := h.db.UpdateVoiceMemoTranscript(memoID, result.Text, result.Language, status, errMsg); err != nil {
		log.Printf("voicememo: failed to update transcript for %s: %v", memoID, err)
	}
}

func (h *VoiceMemoHandler) callASR(filePath, originalName string) (*asrResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(originalName))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.WriteField("language", "zh"); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(h.asrServiceURL, "/")+"/transcribe", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("ASR 服务返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 300))
	}

	result := &asrResponse{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	if result.Status == "" {
		result.Status = "completed"
	}
	return result, nil
}
