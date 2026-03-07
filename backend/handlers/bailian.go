package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

type BailianHandler struct {
	db       *models.DB
	cfg      config.BailianConfig
	client   *http.Client
	modelMap map[string]config.BailianModelConfig
}

type CreateBailianTaskRequest struct {
	AdminPassword   string                 `json:"admin_password"`
	Model           string                 `json:"model" binding:"required"`
	Prompt          string                 `json:"prompt" binding:"required"`
	NegativePrompt  string                 `json:"negative_prompt"`
	Image           string                 `json:"image"`
	Images          []string               `json:"images"`
	Size            string                 `json:"size"`
	Count           int                    `json:"count"`
	Seed            *int                   `json:"seed"`
	Watermark       *bool                  `json:"watermark"`
	Duration        int                    `json:"duration"`
	Resolution      string                 `json:"resolution"`
	FPS             int                    `json:"fps"`
	AutoPoll        bool                   `json:"auto_poll"`
	WaitSeconds     int                    `json:"wait_seconds"`
	ClientName      string                 `json:"client_name"`
	ClientRequestID string                 `json:"client_request_id"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type PollTaskRequest struct {
	AdminPassword string `json:"admin_password"`
	WaitSeconds   int    `json:"wait_seconds"`
}

type bailianCreateResult struct {
	Task      *models.BailianImageTask   `json:"task"`
	Events    []*models.BailianTaskEvent `json:"events,omitempty"`
	Waited    bool                       `json:"waited"`
	Completed bool                       `json:"completed"`
	Assets    []map[string]interface{}   `json:"assets,omitempty"`
	RawOutput interface{}                `json:"raw_output,omitempty"`
}

func NewBailianHandler(db *models.DB, cfg *config.Config) *BailianHandler {
	modelMap := make(map[string]config.BailianModelConfig)
	for _, model := range cfg.Bailian.Models {
		modelMap[model.Name] = model
	}
	return &BailianHandler{
		db:  db,
		cfg: cfg.Bailian,
		client: &http.Client{
			Timeout: 180 * time.Second,
		},
		modelMap: modelMap,
	}
}

func (h *BailianHandler) GetModels(c *gin.Context) {
	if !h.requireAdmin(c, "") {
		return
	}

	now := time.Now()
	modelsResp := make([]gin.H, 0, len(h.cfg.Models))
	for _, model := range h.cfg.Models {
		used, _ := h.db.CountBailianQuotaUsage(model.Name)
		expiresAt, expired := h.parseModelExpiry(model)
		modelsResp = append(modelsResp, gin.H{
			"name":            model.Name,
			"type":            model.Type,
			"enabled":         model.Enabled,
			"description":     model.Description,
			"default_size":    model.DefaultSize,
			"total_quota":     model.TotalQuota,
			"used_quota":      used,
			"remaining_quota": maxInt(model.TotalQuota-used, 0),
			"expires_at":      timeToString(expiresAt),
			"is_expired":      expired || (!expiresAt.IsZero() && now.After(expiresAt)),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": modelsResp,
		"config": gin.H{
			"default_wait_seconds": h.cfg.DefaultWaitSeconds,
			"task_retention_days":  h.cfg.TaskRetentionDays,
		},
	})
}

func (h *BailianHandler) CreateTask(c *gin.Context) {
	h.handleCreateTask(c, "debug")
}

func (h *BailianHandler) OpenAPICreateTask(c *gin.Context) {
	h.handleCreateTask(c, "openapi")
}

func (h *BailianHandler) handleCreateTask(c *gin.Context, source string) {
	var req CreateBailianTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不完整", "code": 400})
		return
	}
	if !h.requireAdmin(c, req.AdminPassword) {
		return
	}
	result, statusCode, err := h.ExecuteTask(req, source, c.ClientIP())
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error(), "code": statusCode, "task": result.Task, "events": result.Events})
		return
	}
	c.JSON(statusCode, result)
}

func (h *BailianHandler) ExecuteTask(req CreateBailianTaskRequest, source, clientIP string) (*bailianCreateResult, int, error) {
	modelCfg, ok := h.modelMap[req.Model]
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("不支持的模型")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("prompt 不能为空")
	}

	images := normalizeImages(req.Image, req.Images)
	if modelCfg.Type == "image2video" && len(images) == 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("图生视频至少需要 1 张图片")
	}
	if modelCfg.Type == "multimodal" && len(images) > 3 {
		return nil, http.StatusBadRequest, fmt.Errorf("图像编辑最多支持 3 张图片")
	}

	quotaTotal, quotaUsed, quotaExpiresAt, err := h.checkQuota(modelCfg)
	if err != nil {
		return nil, http.StatusForbidden, err
	}

	task := &models.BailianImageTask{
		Model:          req.Model,
		TaskType:       modelCfg.Type,
		Source:         source,
		ClientName:     strings.TrimSpace(req.ClientName),
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
		InputImages:    models.MustJSONString(summarizeImages(images)),
		ParamsJSON: models.MustJSONString(gin.H{
			"size":              normalizeBailianSize(nonEmpty(req.Size, modelCfg.DefaultSize)),
			"count":             defaultInt(req.Count, 1),
			"seed":              req.Seed,
			"watermark":         req.Watermark,
			"duration":          req.Duration,
			"resolution":        req.Resolution,
			"fps":               req.FPS,
			"wait_seconds":      req.WaitSeconds,
			"client_request_id": req.ClientRequestID,
			"parameters":        req.Parameters,
		}),
		Status:         "created",
		QuotaTotal:     quotaTotal,
		QuotaUsed:      quotaUsed + 1,
		QuotaExpiresAt: quotaExpiresAt,
		QuotaCounted:   true,
		CreatorIP:      clientIP,
	}
	if err := h.db.CreateBailianTask(task); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("任务创建失败")
	}
	h.addEvent(task.ID, "task.created", "created", "任务已创建，准备调用百炼接口", gin.H{
		"model":       req.Model,
		"task_type":   modelCfg.Type,
		"image_count": len(images),
		"source":      source,
	})

	body, endpoint, async, err := h.buildVendorRequest(req, modelCfg, images)
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateBailianTask(task)
		h.addEvent(task.ID, "task.validation_failed", "failed", err.Error(), nil)
		events, _ := h.db.ListBailianTaskEvents(task.ID)
		return &bailianCreateResult{Task: task, Events: events}, http.StatusBadRequest, err
	}
	task.RequestBody = sanitizeJSON(body)
	_ = h.db.UpdateBailianTask(task)

	responseMap, responseBody, err := h.callBailianAPI(endpoint, body, async)
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		task.ResponseBody = truncateString(responseBody, 20000)
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateBailianTask(task)
		h.addEvent(task.ID, "vendor.request_failed", "failed", err.Error(), gin.H{
			"endpoint": endpoint,
			"response": tryParseJSON(responseBody),
		})
		events, _ := h.db.ListBailianTaskEvents(task.ID)
		return &bailianCreateResult{Task: task, Events: events}, http.StatusBadGateway, err
	}

	task.ResponseBody = truncateString(responseBody, 50000)
	task.VendorStatus = h.extractVendorStatus(responseMap)
	if async {
		task.ExternalTaskID = extractString(responseMap, "output", "task_id")
		if task.ExternalTaskID == "" {
			task.ExternalTaskID = extractString(responseMap, "task_id")
		}
		task.Status = mapAsyncStatus(task.VendorStatus)
		if task.Status == "created" {
			task.Status = "submitted"
		}
	} else {
		task.Status = "succeeded"
		now := time.Now()
		task.CompletedAt = &now
	}

	assets := extractAssets(responseMap)
	task.ResultJSON = models.MustJSONString(gin.H{
		"assets":     assets,
		"raw_output": responseMap["output"],
	})

	if errMsg := h.extractVendorError(responseMap); errMsg != "" {
		task.ErrorMessage = errMsg
		if task.Status != "succeeded" {
			now := time.Now()
			task.CompletedAt = &now
			task.Status = "failed"
		}
	}

	_ = h.db.UpdateBailianTask(task)
	h.addEvent(task.ID, "vendor.response", task.Status, "百炼接口已返回", gin.H{
		"endpoint":      endpoint,
		"async":         async,
		"task_id":       task.ExternalTaskID,
		"vendor_status": task.VendorStatus,
		"assets":        assets,
		"raw_response":  tryParseJSON(responseBody),
	})

	waitSeconds := req.WaitSeconds
	if waitSeconds <= 0 {
		waitSeconds = h.cfg.DefaultWaitSeconds
	}
	completed := task.Status == "succeeded" || task.Status == "failed"
	waited := false
	taskID := task.ID
	if async && req.AutoPoll && !completed && waitSeconds > 0 {
		waited = true
		task, _ = h.waitForTask(taskID, waitSeconds)
		completed = task != nil && (task.Status == "succeeded" || task.Status == "failed")
	}

	if task == nil {
		task, _ = h.db.GetBailianTask(taskID)
	}
	events, _ := h.db.ListBailianTaskEvents(taskID)
	result := bailianCreateResult{
		Task:      task,
		Events:    events,
		Waited:    waited,
		Completed: completed,
	}
	if task != nil {
		var payload map[string]interface{}
		_ = json.Unmarshal([]byte(task.ResultJSON), &payload)
		if payload != nil {
			if assetsValue, ok := payload["assets"].([]interface{}); ok {
				result.Assets = normalizeAssetList(assetsValue)
			}
			result.RawOutput = payload["raw_output"]
		}
	}
	return &result, http.StatusOK, nil
}

func (h *BailianHandler) ListTasks(c *gin.Context) {
	if !h.requireAdmin(c, "") {
		return
	}
	limit := clampInt(parseInt(c.Query("limit"), 20), 1, 100)
	offset := maxInt(parseInt(c.Query("offset"), 0), 0)
	model := strings.TrimSpace(c.Query("model"))
	status := strings.TrimSpace(c.Query("status"))

	tasks, err := h.db.ListBailianTasks(limit, offset, model, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败", "code": 500})
		return
	}
	total, _ := h.db.CountBailianTasks(model, status)
	c.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *BailianHandler) GetTask(c *gin.Context) {
	if !h.requireAdmin(c, "") {
		return
	}
	task, err := h.db.GetBailianTask(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	events, _ := h.db.ListBailianTaskEvents(task.ID)
	c.JSON(http.StatusOK, gin.H{"task": task, "events": events})
}

func (h *BailianHandler) GetTaskEvents(c *gin.Context) {
	if !h.requireAdmin(c, "") {
		return
	}
	events, err := h.db.ListBailianTaskEvents(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取流水失败", "code": 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *BailianHandler) PollTask(c *gin.Context) {
	var req PollTaskRequest
	_ = c.ShouldBindJSON(&req)
	if !h.requireAdmin(c, req.AdminPassword) {
		return
	}
	taskID := c.Param("id")
	task, err := h.db.GetBailianTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在", "code": 404})
		return
	}
	if task.ExternalTaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该任务没有外部 task_id，无需轮询", "code": 400})
		return
	}
	waitSeconds := req.WaitSeconds
	if waitSeconds <= 0 {
		waitSeconds = 1
	}
	if waitSeconds > 1 {
		task, err = h.waitForTask(taskID, waitSeconds)
	} else {
		task, err = h.pollOnce(task)
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": 502})
		return
	}
	events, _ := h.db.ListBailianTaskEvents(task.ID)
	c.JSON(http.StatusOK, gin.H{"task": task, "events": events})
}

func (h *BailianHandler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"title": "百炼图片模型 API 文档",
		"auth": gin.H{
			"header": "X-Admin-Password",
			"query":  "admin_password",
			"body":   "admin_password",
		},
		"routes": []gin.H{
			{"method": "GET", "path": "/api/bailian/docs", "description": "获取接口文档"},
			{"method": "GET", "path": "/api/bailian/models", "description": "获取模型列表、剩余额度、到期时间"},
			{"method": "POST", "path": "/api/bailian/tasks", "description": "调试台创建任务"},
			{"method": "GET", "path": "/api/bailian/tasks", "description": "任务列表，支持 model/status/limit/offset"},
			{"method": "GET", "path": "/api/bailian/tasks/:id", "description": "任务详情，包含当前状态与结果"},
			{"method": "GET", "path": "/api/bailian/tasks/:id/events", "description": "任务全流程流水"},
			{"method": "POST", "path": "/api/bailian/tasks/:id/poll", "description": "轮询异步任务状态"},
			{"method": "POST", "path": "/api/bailian/generate", "description": "开放给其他业务的通用创建接口"},
		},
		"request_example": gin.H{
			"admin_password":  "your-password",
			"model":           "qwen-image-2.0-pro",
			"prompt":          "一只橘猫穿着宇航服，电影级光影，超清海报",
			"negative_prompt": "模糊、低质量、变形的手",
			"images":          []string{"data:image/png;base64,..."},
			"size":            "1328x1328",
			"count":           1,
			"auto_poll":       true,
			"wait_seconds":    30,
			"client_name":     "marketing-site",
			"parameters": gin.H{
				"style": "photorealistic",
			},
		},
		"notes": []string{
			"所有生成接口必须提供管理密码，防止额度被滥用。",
			"本地会按模型维度记录累计调用次数，并在配置的 expires_at 到期后自动拒绝继续调用。",
			"任务流水会记录创建、外部请求、轮询、成功/失败等关键阶段，方便回溯。",
			"异步任务返回的媒体 URL 可能是临时地址，建议及时下载并在业务侧落盘。",
		},
	})
}

func (h *BailianHandler) buildVendorRequest(req CreateBailianTaskRequest, modelCfg config.BailianModelConfig, images []string) (map[string]interface{}, string, bool, error) {
	switch modelCfg.Type {
	case "multimodal":
		content := make([]map[string]string, 0, len(images)+1)
		for _, image := range images {
			content = append(content, map[string]string{"image": image})
		}
		content = append(content, map[string]string{"text": req.Prompt})

		parameters := map[string]interface{}{
			"size": normalizeBailianSize(nonEmpty(req.Size, modelCfg.DefaultSize)),
			"n":    defaultInt(req.Count, 1),
		}
		if req.NegativePrompt != "" {
			parameters["negative_prompt"] = req.NegativePrompt
		}
		if req.Seed != nil {
			parameters["seed"] = *req.Seed
		}
		if req.Watermark != nil {
			parameters["watermark"] = *req.Watermark
		}
		mergeParameters(parameters, req.Parameters)

		return map[string]interface{}{
			"model": modelCfg.Name,
			"input": map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"role": "system",
						"content": []map[string]string{
							{"text": "You are a professional image generation assistant."},
						},
					},
					{
						"role":    "user",
						"content": content,
					},
				},
			},
			"parameters": parameters,
		}, h.cfg.BaseURL + "/api/v1/services/aigc/multimodal-generation/generation", false, nil
	case "text2image":
		parameters := map[string]interface{}{
			"size": normalizeBailianSize(nonEmpty(req.Size, modelCfg.DefaultSize)),
			"n":    defaultInt(req.Count, 1),
		}
		if req.NegativePrompt != "" {
			parameters["negative_prompt"] = req.NegativePrompt
		}
		if req.Seed != nil {
			parameters["seed"] = *req.Seed
		}
		if req.Watermark != nil {
			parameters["watermark"] = *req.Watermark
		}
		mergeParameters(parameters, req.Parameters)
		return map[string]interface{}{
			"model": modelCfg.Name,
			"input": map[string]interface{}{
				"prompt": req.Prompt,
			},
			"parameters": parameters,
		}, h.cfg.BaseURL + "/api/v1/services/aigc/text2image/image-synthesis", true, nil
	case "image2video":
		if len(images) == 0 {
			return nil, "", false, fmt.Errorf("图生视频至少需要 1 张图片")
		}
		parameters := map[string]interface{}{
			"resolution": nonEmpty(req.Resolution, modelCfg.DefaultSize),
		}
		if req.Duration > 0 {
			parameters["duration"] = req.Duration
		}
		if req.FPS > 0 {
			parameters["fps"] = req.FPS
		}
		if req.Watermark != nil {
			parameters["watermark"] = *req.Watermark
		}
		mergeParameters(parameters, req.Parameters)

		return map[string]interface{}{
			"model": modelCfg.Name,
			"input": map[string]interface{}{
				"prompt":  req.Prompt,
				"img_url": images[0],
			},
			"parameters": parameters,
		}, h.cfg.BaseURL + "/api/v1/services/aigc/video-generation/video-synthesis", true, nil
	default:
		return nil, "", false, fmt.Errorf("未知模型类型: %s", modelCfg.Type)
	}
}

func (h *BailianHandler) callBailianAPI(endpoint string, body map[string]interface{}, async bool) (map[string]interface{}, string, error) {
	if strings.TrimSpace(h.cfg.APIKey) == "" {
		return nil, "", fmt.Errorf("未配置 bailian.api_key 或 BAILIAN_API_KEY")
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if async {
		req.Header.Set("X-DashScope-Async", "enable")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	respText := string(respBody)
	var payload map[string]interface{}
	_ = json.Unmarshal(respBody, &payload)

	if resp.StatusCode >= 400 {
		if msg := h.extractVendorError(payload); msg != "" {
			return payload, respText, fmt.Errorf("百炼接口错误: %s", msg)
		}
		return payload, respText, fmt.Errorf("百炼接口返回状态码 %d", resp.StatusCode)
	}
	return payload, respText, nil
}

func (h *BailianHandler) waitForTask(taskID string, waitSeconds int) (*models.BailianImageTask, error) {
	deadline := time.Now().Add(time.Duration(waitSeconds) * time.Second)
	for {
		task, err := h.db.GetBailianTask(taskID)
		if err != nil {
			return nil, err
		}
		if task.Status == "succeeded" || task.Status == "failed" || task.Status == "canceled" {
			return task, nil
		}
		if time.Now().After(deadline) {
			return task, nil
		}
		task, err = h.pollOnce(task)
		if err != nil {
			return task, err
		}
		if task.Status == "succeeded" || task.Status == "failed" || task.Status == "canceled" {
			return task, nil
		}
		time.Sleep(3 * time.Second)
	}
}

func (h *BailianHandler) pollOnce(task *models.BailianImageTask) (*models.BailianImageTask, error) {
	if task == nil || task.ExternalTaskID == "" {
		return task, fmt.Errorf("缺少外部任务 ID")
	}
	endpoint := h.cfg.BaseURL + "/api/v1/tasks/" + task.ExternalTaskID
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return task, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.APIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return task, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	respText := string(body)

	var payload map[string]interface{}
	_ = json.Unmarshal(body, &payload)

	if resp.StatusCode >= 400 {
		msg := h.extractVendorError(payload)
		if msg == "" {
			msg = fmt.Sprintf("轮询失败，状态码 %d", resp.StatusCode)
		}
		task.ErrorMessage = msg
		task.ResponseBody = truncateString(respText, 50000)
		task.Status = "failed"
		now := time.Now()
		task.CompletedAt = &now
		_ = h.db.UpdateBailianTask(task)
		h.addEvent(task.ID, "vendor.poll_failed", "failed", msg, tryParseJSON(respText))
		return task, fmt.Errorf(msg)
	}

	task.VendorStatus = h.extractVendorStatus(payload)
	task.Status = mapAsyncStatus(task.VendorStatus)
	task.ResponseBody = truncateString(respText, 50000)
	task.ResultJSON = models.MustJSONString(gin.H{
		"assets":     extractAssets(payload),
		"raw_output": payload["output"],
	})
	if msg := h.extractVendorError(payload); msg != "" {
		task.ErrorMessage = msg
		if task.Status != "succeeded" {
			task.Status = "failed"
		}
	}
	if task.Status == "succeeded" || task.Status == "failed" || task.Status == "canceled" {
		now := time.Now()
		task.CompletedAt = &now
	}
	_ = h.db.UpdateBailianTask(task)
	h.addEvent(task.ID, "vendor.poll", task.Status, "任务状态已刷新", gin.H{
		"vendor_status": task.VendorStatus,
		"response":      tryParseJSON(respText),
	})
	return h.db.GetBailianTask(task.ID)
}

func (h *BailianHandler) checkQuota(model config.BailianModelConfig) (int, int, *time.Time, error) {
	if !model.Enabled {
		return 0, 0, nil, fmt.Errorf("模型 %s 当前未开启", model.Name)
	}
	expiresAt, expired := h.parseModelExpiry(model)
	if expired {
		return model.TotalQuota, 0, timePtrOrNil(expiresAt), fmt.Errorf("模型 %s 已超过本地截止时间 %s，已停止调用防止计费", model.Name, expiresAt.Format("2006-01-02"))
	}
	used, err := h.db.CountBailianQuotaUsage(model.Name)
	if err != nil {
		return model.TotalQuota, 0, timePtrOrNil(expiresAt), err
	}
	if model.TotalQuota <= 0 {
		return model.TotalQuota, used, timePtrOrNil(expiresAt), fmt.Errorf("模型 %s 未配置免费额度，默认禁止调用", model.Name)
	}
	if used >= model.TotalQuota {
		return model.TotalQuota, used, timePtrOrNil(expiresAt), fmt.Errorf("模型 %s 本地额度已用完（%d/%d），已停止调用防止收费", model.Name, used, model.TotalQuota)
	}
	return model.TotalQuota, used, timePtrOrNil(expiresAt), nil
}

func (h *BailianHandler) parseModelExpiry(model config.BailianModelConfig) (time.Time, bool) {
	if strings.TrimSpace(model.ExpiresAt) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", model.ExpiresAt)
	if err != nil {
		return time.Time{}, false
	}
	expireAt := t.Add(24*time.Hour - time.Second)
	return expireAt, time.Now().After(expireAt)
}

func (h *BailianHandler) requireAdmin(c *gin.Context, bodyPassword string) bool {
	password := bodyPassword
	if password == "" {
		password = c.GetHeader("X-Admin-Password")
	}
	if password == "" {
		password = c.Query("admin_password")
	}
	if strings.TrimSpace(h.cfg.AdminPassword) == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统未设置 bailian.admin_password 或 BAILIAN_ADMIN_PASSWORD", "code": 403})
		return false
	}
	if password != h.cfg.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理密码错误", "code": 401})
		return false
	}
	return true
}

func (h *BailianHandler) addEvent(taskID, stage, status, message string, payload interface{}) {
	_ = h.db.CreateBailianTaskEvent(&models.BailianTaskEvent{
		TaskID:  taskID,
		Stage:   stage,
		Status:  status,
		Message: message,
		Payload: sanitizeJSON(payload),
	})
}

func (h *BailianHandler) extractVendorStatus(payload map[string]interface{}) string {
	status := extractString(payload, "output", "task_status")
	if status == "" {
		status = extractString(payload, "output", "status")
	}
	if status == "" {
		status = extractString(payload, "status")
	}
	return strings.ToUpper(status)
}

func (h *BailianHandler) extractVendorError(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	candidates := []string{
		extractString(payload, "message"),
		extractString(payload, "error", "message"),
		extractString(payload, "output", "message"),
		extractString(payload, "output", "error_message"),
	}
	for _, item := range candidates {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}

func normalizeImages(single string, many []string) []string {
	images := make([]string, 0, len(many)+1)
	if strings.TrimSpace(single) != "" {
		images = append(images, strings.TrimSpace(single))
	}
	for _, image := range many {
		if strings.TrimSpace(image) != "" {
			images = append(images, strings.TrimSpace(image))
		}
	}
	return images
}

func summarizeImages(images []string) []gin.H {
	summaries := make([]gin.H, 0, len(images))
	for _, image := range images {
		summaries = append(summaries, gin.H{
			"kind":    detectImageKind(image),
			"preview": truncateString(image, 160),
			"length":  len(image),
		})
	}
	return summaries
}

func detectImageKind(image string) string {
	switch {
	case strings.HasPrefix(image, "data:image"):
		return "data-url"
	case strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://"):
		return "url"
	default:
		return "raw"
	}
}

func extractAssets(payload map[string]interface{}) []gin.H {
	assets := make([]gin.H, 0)
	seen := make(map[string]bool)
	var walk func(interface{})
	walk = func(node interface{}) {
		switch value := node.(type) {
		case map[string]interface{}:
			for key, item := range value {
				lower := strings.ToLower(key)
				if str, ok := item.(string); ok {
					if looksLikeAsset(lower, str) && !seen[str] {
						seen[str] = true
						assets = append(assets, gin.H{
							"type":  inferAssetType(lower, str),
							"value": str,
						})
					}
				}
				walk(item)
			}
		case []interface{}:
			for _, item := range value {
				walk(item)
			}
		}
	}
	walk(payload)
	return assets
}

func looksLikeAsset(key, value string) bool {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "data:image") || strings.HasPrefix(value, "data:video") {
		return true
	}
	return strings.Contains(key, "url") || strings.Contains(key, "image") || strings.Contains(key, "video")
}

func inferAssetType(key, value string) string {
	if strings.Contains(key, "video") || strings.HasPrefix(value, "data:video") || strings.Contains(strings.ToLower(value), ".mp4") {
		return "video"
	}
	return "image"
}

func mergeParameters(dst map[string]interface{}, extra map[string]interface{}) {
	for key, value := range extra {
		if _, exists := dst[key]; !exists || value != nil {
			if key == "size" {
				if str, ok := value.(string); ok {
					dst[key] = normalizeBailianSize(str)
					continue
				}
			}
			dst[key] = value
		}
	}
}

func normalizeBailianSize(value string) string {
	size := strings.TrimSpace(value)
	if size == "" {
		return ""
	}
	size = strings.ReplaceAll(size, "×", "*")
	size = strings.ReplaceAll(size, "X", "*")
	size = strings.ReplaceAll(size, "x", "*")
	return size
}

func tryParseJSON(raw string) interface{} {
	if raw == "" {
		return gin.H{}
	}
	var payload interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		return payload
	}
	return gin.H{"raw": truncateString(raw, 1000)}
}

func sanitizeJSON(payload interface{}) string {
	return models.MustJSONString(sanitizeValue(payload))
}

func sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return truncateString(v, 2000)
	case []string:
		items := make([]interface{}, 0, len(v))
		for _, item := range v {
			items = append(items, truncateString(item, 500))
		}
		return items
	case []interface{}:
		items := make([]interface{}, 0, len(v))
		for _, item := range v {
			items = append(items, sanitizeValue(item))
		}
		return items
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, item := range v {
			result[key] = sanitizeValue(item)
		}
		return result
	case gin.H:
		result := make(map[string]interface{}, len(v))
		for key, item := range v {
			result[key] = sanitizeValue(item)
		}
		return result
	default:
		return value
	}
}

func extractString(payload map[string]interface{}, path ...string) string {
	var current interface{} = payload
	for _, key := range path {
		node, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = node[key]
		if !ok {
			return ""
		}
	}
	switch value := current.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return ""
	}
}

func normalizeAssetList(items []interface{}) []map[string]interface{} {
	assets := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			assets = append(assets, m)
		}
	}
	return assets
}

func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func truncateString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "...(truncated)"
}

func mapAsyncStatus(vendorStatus string) string {
	switch strings.ToUpper(strings.TrimSpace(vendorStatus)) {
	case "PENDING", "QUEUED":
		return "queued"
	case "RUNNING", "PROCESSING":
		return "running"
	case "SUCCEEDED", "SUCCESS":
		return "succeeded"
	case "FAILED", "FAIL":
		return "failed"
	case "CANCELED", "CANCELLED":
		return "canceled"
	case "":
		return "submitted"
	default:
		return "submitted"
	}
}

func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func timeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
