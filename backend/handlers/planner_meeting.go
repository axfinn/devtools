package handlers

import (
	"bytes"
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

var plannerRecordingDir = "./data/planner_recordings"

type createMeetingRequest struct {
	Title           string `json:"title" binding:"required"`
	Content         string `json:"content"`
	Summary         string `json:"summary"`
	ActionItems     string `json:"action_items"`
	Participants    string `json:"participants"`
	RecordingURL    string `json:"recording_url"`
	DurationMinutes int    `json:"duration_minutes"`
	MeetingDate     string `json:"meeting_date"`
	MeetingTime     string `json:"meeting_time"`
	Tags            string `json:"tags"`
	Status          string `json:"status"`
}

type updateMeetingRequest struct {
	Title           *string `json:"title"`
	Content         *string `json:"content"`
	Summary         *string `json:"summary"`
	ActionItems     *string `json:"action_items"`
	Participants    *string `json:"participants"`
	RecordingURL    *string `json:"recording_url"`
	DurationMinutes *int    `json:"duration_minutes"`
	MeetingDate     *string `json:"meeting_date"`
	MeetingTime     *string `json:"meeting_time"`
	Tags            *string `json:"tags"`
	Status          *string `json:"status"`
}

func (h *PlannerHandler) ListMeetingMinutes(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	meetings, err := h.db.ListPlannerMeetings(profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会议纪要失败", "code": 500})
		return
	}
	if meetings == nil {
		meetings = []*models.PlannerMeetingMinutes{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "meetings": meetings})
}

func (h *PlannerHandler) CreateMeetingMinutes(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	var req createMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空", "code": 400})
		return
	}
	meetingDate := strings.TrimSpace(req.MeetingDate)
	if meetingDate == "" {
		meetingDate = time.Now().Format("2006-01-02")
	}
	meeting := &models.PlannerMeetingMinutes{
		ProfileID:       profile.ID,
		Title:           strings.TrimSpace(req.Title),
		Content:         strings.TrimSpace(req.Content),
		Summary:         strings.TrimSpace(req.Summary),
		ActionItems:     plannerFirstNonEmpty(strings.TrimSpace(req.ActionItems), "[]"),
		Participants:    plannerFirstNonEmpty(strings.TrimSpace(req.Participants), "[]"),
		RecordingURL:    strings.TrimSpace(req.RecordingURL),
		DurationMinutes: req.DurationMinutes,
		MeetingDate:     meetingDate,
		MeetingTime:     strings.TrimSpace(req.MeetingTime),
		Tags:            plannerFirstNonEmpty(strings.TrimSpace(req.Tags), "[]"),
		Status:          plannerFirstNonEmpty(strings.TrimSpace(req.Status), "draft"),
	}
	if err := h.db.CreatePlannerMeeting(meeting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败", "code": 500})
		return
	}
	h.logPlannerActivity(profile.ID, meeting.ID, "create", "创建会议纪要", meeting.Title)
	c.JSON(http.StatusOK, gin.H{"code": 0, "meeting": meeting})
}

func (h *PlannerHandler) GetMeetingMinutes(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	meetingID := c.Param("meetingId")
	meeting, err := h.db.GetPlannerMeeting(meetingID)
	if err != nil || meeting.ProfileID != profile.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会议纪要不存在", "code": 404})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "meeting": meeting})
}

func (h *PlannerHandler) UpdateMeetingMinutes(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	meetingID := c.Param("meetingId")
	meeting, err := h.db.GetPlannerMeeting(meetingID)
	if err != nil || meeting.ProfileID != profile.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会议纪要不存在", "code": 404})
		return
	}
	var req updateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": 400})
		return
	}
	if req.Title != nil {
		meeting.Title = strings.TrimSpace(*req.Title)
	}
	if req.Content != nil {
		meeting.Content = strings.TrimSpace(*req.Content)
	}
	if req.Summary != nil {
		meeting.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.ActionItems != nil {
		meeting.ActionItems = strings.TrimSpace(*req.ActionItems)
	}
	if req.Participants != nil {
		meeting.Participants = strings.TrimSpace(*req.Participants)
	}
	if req.RecordingURL != nil {
		meeting.RecordingURL = strings.TrimSpace(*req.RecordingURL)
	}
	if req.DurationMinutes != nil {
		meeting.DurationMinutes = *req.DurationMinutes
	}
	if req.MeetingDate != nil {
		meeting.MeetingDate = strings.TrimSpace(*req.MeetingDate)
	}
	if req.MeetingTime != nil {
		meeting.MeetingTime = strings.TrimSpace(*req.MeetingTime)
	}
	if req.Tags != nil {
		meeting.Tags = strings.TrimSpace(*req.Tags)
	}
	if req.Status != nil {
		meeting.Status = strings.TrimSpace(*req.Status)
	}
	if err := h.db.UpdatePlannerMeeting(meeting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败", "code": 500})
		return
	}
	h.logPlannerActivity(profile.ID, meeting.ID, "update", "更新会议纪要", meeting.Title)
	c.JSON(http.StatusOK, gin.H{"code": 0, "meeting": meeting})
}

func (h *PlannerHandler) DeleteMeetingMinutes(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	meetingID := c.Param("meetingId")
	meeting, err := h.db.GetPlannerMeeting(meetingID)
	if err != nil || meeting.ProfileID != profile.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会议纪要不存在", "code": 404})
		return
	}
	if err := h.db.DeletePlannerMeeting(meetingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败", "code": 500})
		return
	}
	h.logPlannerActivity(profile.ID, meetingID, "delete", "删除会议纪要", meeting.Title)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}

func (h *PlannerHandler) SummarizeMeetingMinutes(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}
	meetingID := c.Param("meetingId")
	meeting, err := h.db.GetPlannerMeeting(meetingID)
	if err != nil || meeting.ProfileID != profile.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会议纪要不存在", "code": 404})
		return
	}

	summary, err := h.aiSummarizeMeeting(meeting)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": 500})
		return
	}

	meeting.Summary = summary.Summary
	meeting.ActionItems = summary.ActionItems
	if err := h.db.UpdatePlannerMeeting(meeting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存摘要失败", "code": 500})
		return
	}
	h.logPlannerActivity(profile.ID, meeting.ID, "summarize", "AI 总结会议纪要", meeting.Title)
	c.JSON(http.StatusOK, gin.H{"code": 0, "summary": summary.Summary, "action_items": summary.ActionItems})
}

type meetingSummaryResult struct {
	Summary     string `json:"summary"`
	ActionItems string `json:"action_items"`
}

type plannerASRSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type plannerASRResponse struct {
	Text     string              `json:"text"`
	Language string              `json:"language"`
	Segments []plannerASRSegment `json:"segments"`
	Status   string              `json:"status"`
	Error    string              `json:"error"`
}

func (h *PlannerHandler) aiSummarizeMeeting(meeting *models.PlannerMeetingMinutes) (*meetingSummaryResult, error) {
	result := &meetingSummaryResult{Summary: "暂无 AI 摘要", ActionItems: "[]"}

	content := strings.TrimSpace(meeting.Content)
	if content == "" {
		return result, nil
	}
	if strings.TrimSpace(h.cfg.MiniMax.APIKey) == "" {
		return h.meetingSummaryFallback(meeting), nil
	}

	participantStr := meeting.Participants
	if participantStr == "[]" || participantStr == "" {
		participantStr = "未记录"
	}

	prompt := fmt.Sprintf(`你是一个会议纪要助手。请基于下面的会议内容输出 JSON，字段固定为：
summary(string) — 会议总结，200 字以内
action_items(string) — JSON 数组字符串，每条包含 task(string) 和 assignee(string)

规则：
1. 只返回 JSON，不要附带解释。
2. 如果原文没有明确待办事项，action_items 返回空数组 []。
3. 用中文总结。

会议标题：%s
参与人：%s
日期：%s

会议内容：
%s`, meeting.Title, participantStr, meeting.MeetingDate, content)

	reqBody := map[string]interface{}{
		"model": plannerFirstNonEmpty(h.cfg.MiniMax.Model, "abab6.5s-chat"),
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个专业的会议纪要助手，擅长总结和提取待办事项。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", plannerMiniMaxAPIURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("meeting summarize: build request failed, fallback: %v", err)
		return h.meetingSummaryFallback(meeting), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.MiniMax.APIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("meeting summarize: request failed, fallback: %v", err)
		return h.meetingSummaryFallback(meeting), nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("meeting summarize: API returned %d, fallback", resp.StatusCode)
		return h.meetingSummaryFallback(meeting), nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return h.meetingSummaryFallback(meeting), nil
	}
	choices, _ := raw["choices"].([]interface{})
	if len(choices) == 0 {
		return h.meetingSummaryFallback(meeting), nil
	}
	first, _ := choices[0].(map[string]interface{})
	msg, _ := first["message"].(map[string]interface{})
	contentResp, _ := msg["content"].(string)
	payload, ok := extractJSONPayload(contentResp)
	if !ok {
		return h.meetingSummaryFallback(meeting), nil
	}
	var ms meetingSummaryResult
	if err := json.Unmarshal([]byte(payload), &ms); err != nil {
		return h.meetingSummaryFallback(meeting), nil
	}
	if strings.TrimSpace(ms.Summary) == "" {
		return h.meetingSummaryFallback(meeting), nil
	}
	if strings.TrimSpace(ms.ActionItems) == "" {
		ms.ActionItems = "[]"
	}
	return &ms, nil
}

func (h *PlannerHandler) meetingSummaryFallback(meeting *models.PlannerMeetingMinutes) *meetingSummaryResult {
	content := strings.TrimSpace(meeting.Content)
	if content == "" {
		return &meetingSummaryResult{Summary: "暂无内容可总结", ActionItems: "[]"}
	}

	parts := make([]string, 0, 3)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts = append(parts, line)
		if len(parts) >= 3 {
			break
		}
	}
	preview := strings.Join(parts, "；")
	runes := []rune(preview)
	if len(runes) == 0 {
		runes = []rune(content)
	}
	if len(runes) > 140 {
		runes = runes[:140]
	}

	actionItems := make([]map[string]string, 0, 5)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		isAction := strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "*") ||
			strings.HasPrefix(trimmed, "1.") ||
			strings.HasPrefix(trimmed, "1、") ||
			strings.Contains(trimmed, "待办") ||
			strings.Contains(trimmed, "行动项") ||
			strings.Contains(trimmed, "跟进") ||
			strings.Contains(lower, "todo")
		if !isAction {
			continue
		}
		task := strings.TrimSpace(strings.TrimLeft(trimmed, "-*0123456789.、[] "))
		if task == "" {
			continue
		}
		actionItems = append(actionItems, map[string]string{
			"task":     task,
			"assignee": "",
		})
		if len(actionItems) >= 5 {
			break
		}
	}
	actionItemsJSON := "[]"
	if len(actionItems) > 0 {
		if body, err := json.Marshal(actionItems); err == nil {
			actionItemsJSON = string(body)
		}
	}

	return &meetingSummaryResult{
		Summary:     fmt.Sprintf("会议「%s」已生成本地摘要：%s", meeting.Title, string(runes)),
		ActionItems: actionItemsJSON,
	}
}

func (h *PlannerHandler) UploadMeetingRecording(c *gin.Context) {
	profile, ok := h.loadProfileByAccess(c, c.Param("id"))
	if !ok {
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件", "code": 400})
		return
	}
	defer file.Close()

	// 限制 50MB
	if header.Size > 50*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "录音文件不能超过 50MB", "code": 400})
		return
	}

	if err := os.MkdirAll(plannerRecordingDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误", "code": 500})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".webm"
	}
	filename := fmt.Sprintf("meeting_%s_%d%s", profile.ID, time.Now().UnixMilli(), ext)
	filePath := filepath.Join(plannerRecordingDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": 500})
		return
	}

	recordingURL := "/api/planner/recordings/" + filename
	asrResult, asrErr := h.transcribeMeetingRecording(filePath, header.Filename)
	if asrErr != nil {
		log.Printf("planner meeting transcription failed: %v", asrErr)
	}

	transcriptStatus := "skipped"
	transcript := ""
	transcriptLanguage := ""
	transcriptError := ""
	transcriptSegments := []plannerASRSegment{}
	if asrResult != nil {
		transcriptStatus = plannerFirstNonEmpty(strings.TrimSpace(asrResult.Status), "completed")
		transcript = strings.TrimSpace(asrResult.Text)
		transcriptLanguage = strings.TrimSpace(asrResult.Language)
		transcriptError = strings.TrimSpace(asrResult.Error)
		transcriptSegments = asrResult.Segments
	}
	if asrErr != nil && transcriptError == "" {
		transcriptStatus = "failed"
		transcriptError = "录音已保存，但语音转文字服务暂不可用"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":                0,
		"recording_url":       recordingURL,
		"filename":            filename,
		"transcript":          transcript,
		"transcript_status":   transcriptStatus,
		"transcript_language": transcriptLanguage,
		"transcript_error":    transcriptError,
		"transcript_segments": transcriptSegments,
	})
}

func (h *PlannerHandler) ServeRecording(c *gin.Context) {
	filename := c.Param("filename")
	if strings.Contains(filename, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件名", "code": 400})
		return
	}
	filePath := filepath.Join(plannerRecordingDir, filepath.Base(filename))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在", "code": 404})
		return
	}
	c.File(filePath)
}

func (h *PlannerHandler) logPlannerActivity(profileID, taskID, activityType, title, content string) {
	activity := &models.PlannerTaskActivity{
		ProfileID:    profileID,
		TaskID:       taskID,
		ActivityType: activityType,
		Title:        title,
		Content:      content,
	}
	if err := h.db.CreatePlannerTaskActivity(activity); err != nil {
		log.Printf("planner: failed to log activity: %v", err)
	}
}

func (h *PlannerHandler) transcribeMeetingRecording(filePath, originalName string) (*plannerASRResponse, error) {
	if strings.TrimSpace(h.asrServiceURL) == "" {
		return nil, nil
	}

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

	resp, err := h.serviceHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("ASR 服务返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 300))
	}

	result := &plannerASRResponse{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	if result.Status == "" {
		result.Status = "completed"
	}
	return result, nil
}
