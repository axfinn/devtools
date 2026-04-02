package handlers

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

// autodevUID/GID is the non-root user created in the Dockerfile for running
// autodev tasks. Claude Code refuses --dangerously-skip-permissions as root.
const autodevUID = 1001
const autodevGID = 1001
const autodevHome = "/home/autodev"

const (
	defaultFilePreviewBytes = 256 * 1024
	defaultFilePreviewLines = 400
	defaultLogPreviewBytes  = 192 * 1024
	defaultLogPreviewLines  = 300

	maxPreviewBytes = 1024 * 1024
	maxPreviewLines = 2000

	maxListedTextFileSize   = 5 * 1024 * 1024
	maxListedBinaryFileSize = 20 * 1024 * 1024
	maxListedMediaFileSize  = 100 * 1024 * 1024

	fileSniffBytes = 1024
)

// setSysProcCredential sets the credential on cmd only when running as root.
// When not root (e.g. local dev), setuid is not permitted and must be skipped.
func setSysProcCredential(cmd *exec.Cmd) {
	if os.Getuid() == 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: autodevUID, Gid: autodevGID},
		}
	}
}

func resolveHome() string {
	if os.Getuid() != 0 {
		if home := os.Getenv("HOME"); home != "" {
			return home
		}
	}
	return autodevHome
}

// AutoDevHandler handles autodev task operations
type AutoDevHandler struct {
	db             *models.DB
	adminPassword  string
	autodevPath    string
	stopScriptPath string
	dataDir        string
	mu             sync.RWMutex
	processes      map[string]*exec.Cmd
}

// NewAutoDevHandler creates a new AutoDevHandler
func NewAutoDevHandler(db *models.DB, adminPassword, autodevPath, dataDir string) *AutoDevHandler {
	stopScript := filepath.Join(filepath.Dir(autodevPath), "autodev-stop")
	if _, err := os.Stat(stopScript); err != nil {
		// try /opt/clawtest/autodev/autodev-stop
		stopScript = "/opt/clawtest/autodev/autodev-stop"
	}
	h := &AutoDevHandler{
		db:             db,
		adminPassword:  adminPassword,
		autodevPath:    autodevPath,
		stopScriptPath: stopScript,
		dataDir:        dataDir,
		processes:      make(map[string]*exec.Cmd),
	}
	os.MkdirAll(dataDir, 0755)
	return h
}

// Ask handles POST /api/autodev/ask — submit a quick Q&A request using `autodev ask`
func (h *AutoDevHandler) Ask(c *gin.Context) {
	var req struct {
		Description string `json:"description" binding:"required"`
		Password    string `json:"password" binding:"required"`
		WorkDir     string `json:"work_dir"` // existing project directory for context
		Module      string `json:"module"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// Verify autodev CLI is available
	if _, err := os.Stat(h.autodevPath); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "autodev 未安装，请检查 Docker 配置"})
		return
	}

	// WorkDir is required for ask (needs existing project context)
	if req.WorkDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ask 模式需要指定 work_dir"})
		return
	}

	// Verify work_dir exists
	if _, err := os.Stat(req.WorkDir); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作目录不存在: " + req.WorkDir})
		return
	}

	module := models.NormalizeAutoDevModule(req.Module)

	// Create task record
	opts := models.AutoDevOptions{Module: module}
	task, err := h.db.CreateAutoDevTask(models.TaskTypeAsk, req.Description, models.MarshalAutoDevOptions(opts), req.WorkDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	go h.runAskTask(task.ID, req.Description, req.WorkDir, module)
	c.JSON(http.StatusOK, task)
}

// GetAskResult handles GET /api/autodev/ask/:id?password=xxx
func (h *AutoDevHandler) GetAskResult(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	// Only allow ask type tasks
	if task.Type != models.TaskTypeAsk {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该任务不是问答任务"})
		return
	}

	result := gin.H{
		"id":          task.ID,
		"description": task.Description,
		"status":      task.Status,
		"work_dir":    task.WorkDir,
	}

	// If completed, try to read qa.md
	if task.Status == "completed" && task.ResultFile != "" {
		if content, err := os.ReadFile(task.ResultFile); err == nil {
			result["qa_content"] = string(content)
			result["qa_file"] = task.ResultFile
		}
	}

	// Also check for log file
	logFile := filepath.Join(task.WorkDir, ".autodev", "logs", "ask.log")
	if content, err := os.ReadFile(logFile); err == nil {
		result["logs"] = string(content)
	}

	c.JSON(http.StatusOK, result)
}

// Extend handles POST /api/autodev/extend — extend an existing project with new requirements
func (h *AutoDevHandler) Extend(c *gin.Context) {
	var req struct {
		Description string `json:"description" binding:"required"`
		Password    string `json:"password" binding:"required"`
		WorkDir     string `json:"work_dir" binding:"required"` // existing project directory
		Module      string `json:"module"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// Verify work_dir exists
	if _, err := os.Stat(req.WorkDir); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作目录不存在: " + req.WorkDir})
		return
	}

	// Verify autodev CLI is available
	if _, err := os.Stat(h.autodevPath); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "autodev 未安装，请检查 Docker 配置"})
		return
	}

	module := models.NormalizeAutoDevModule(req.Module)

	// Create task record
	opts := models.AutoDevOptions{Module: module}
	task, err := h.db.CreateAutoDevTask(models.TaskTypeExtend, req.Description, models.MarshalAutoDevOptions(opts), req.WorkDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	go h.runExtendTask(task.ID, req.Description, req.WorkDir, module)
	c.JSON(http.StatusOK, task)
}

// InitProject handles GET /api/autodev/init/stream?password=xxx&work_dir=xxx
// Streams `autodev init --path <work_dir>` output via SSE.
// init_project is pure Python (no claude), so no UID switch needed.
func (h *AutoDevHandler) InitProject(c *gin.Context) {
	password := c.Query("password")
	workDir := c.Query("work_dir")

	if h.adminPassword == "" || password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	if _, err := os.Stat(h.autodevPath); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "autodev 未安装"})
		return
	}
	if workDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_dir 不能为空"})
		return
	}
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作目录不存在: " + workDir})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	sendSSE := func(event, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		c.Writer.Flush()
	}
	sendLog := func(line string) {
		b, _ := json.Marshal(map[string]string{"line": line})
		sendSSE("log", string(b))
	}

	// init_project is pure Python — run as current process user (no UID switch)
	cmd := exec.Command(h.autodevPath, "init", "--path", workDir)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		sendSSE("done", string(b))
		return
	}

	// Stream both stdout and stderr line by line
	scanLines := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			sendLog(sc.Text())
		}
	}
	go scanLines(stderr)
	scanLines(stdout)

	err := cmd.Wait()
	claudeMdPath := filepath.Join(workDir, "CLAUDE.md")
	_, statErr := os.Stat(claudeMdPath)
	claudeMdExists := statErr == nil

	if err != nil {
		b, _ := json.Marshal(map[string]any{
			"error":            "init 执行失败: " + err.Error(),
			"claude_md_exists": claudeMdExists,
		})
		sendSSE("done", string(b))
		return
	}

	b, _ := json.Marshal(map[string]any{
		"ok":               true,
		"claude_md_exists": claudeMdExists,
		"claude_md_path":   claudeMdPath,
	})
	sendSSE("done", string(b))
}

// VerifyPassword handles POST /api/autodev/verify
func (h *AutoDevHandler) VerifyPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetCapabilities handles GET /api/autodev/capabilities.
func (h *AutoDevHandler) GetCapabilities(c *gin.Context) {
	c.JSON(http.StatusOK, models.GetCapabilities())
}

// Submit handles POST /api/autodev/tasks — start a new task or resume from breakpoint
func (h *AutoDevHandler) Submit(c *gin.Context) {
	var req struct {
		Type        string `json:"type"` // develop, loop, ask, export (default: develop)
		Description string `json:"description" binding:"required"`
		Password    string `json:"password" binding:"required"`
		Publish     bool   `json:"publish"`
		Build       bool   `json:"build"`
		Push        bool   `json:"push"`
		Loop        int    `json:"loop"`     // 0=不循环, -1=无限, N=最多N次迭代
		// Resume support: if set, --from <phase> is passed and WorkDir is used
		ResumeFrom int    `json:"resume_from"` // phase number to resume from (1-based, 0 = new task)
		WorkDir    string `json:"work_dir"`    // existing work dir to resume
		// Export options
		ExportFormat string `json:"export_format"` // zip, tar (default: zip)
		Module       string `json:"module"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// Default to develop type
	taskType := req.Type
	if taskType == "" {
		taskType = models.TaskTypeDevelop
	}

	// Check autodev CLI availability for develop/export tasks
	if taskType != models.TaskTypeAsk {
		if _, err := os.Stat(h.autodevPath); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "autodev 未安装，请检查 Docker 配置"})
			return
		}
	}

	var taskDir string
	var isResume bool

	if req.ResumeFrom > 0 && req.WorkDir != "" {
		// Resume mode: reuse existing work dir
		taskDir = req.WorkDir
		isResume = true
		// clear STOP file if present so task can resume
		stopFile := filepath.Join(taskDir, ".autodev", "STOP")
		os.Remove(stopFile)
	} else {
		// New task: create work directory
		taskName := sanitizeTaskName(req.Description)
		taskDir = filepath.Join(h.dataDir, fmt.Sprintf("%s-%d", taskName, time.Now().Unix()))
		if err := os.MkdirAll(taskDir, 0777); err != nil {
			log.Printf("[AutoDev] Create task directory error: dataDir=%s, taskDir=%s, error=%v", h.dataDir, taskDir, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务目录失败: " + err.Error()})
			return
		}
		// Transfer ownership to non-root autodev user so it can create files without permission issues
		os.Chown(taskDir, autodevUID, autodevGID)
	}

	module := models.NormalizeAutoDevModule(req.Module)
	opts := models.AutoDevOptions{Publish: req.Publish, Build: req.Build, Push: req.Push, Module: module, Loop: req.Loop}
	task, err := h.db.CreateAutoDevTask(taskType, req.Description, models.MarshalAutoDevOptions(opts), taskDir)
	if err != nil {
		log.Printf("[AutoDev] CreateAutoDevTask error: type=%s, description=%s, workDir=%s, error=%v", taskType, req.Description, taskDir, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败: " + err.Error()})
		return
	}

	// Execute task based on type
	switch taskType {
	case models.TaskTypeAsk:
		go h.runAskTask(task.ID, req.Description, taskDir, module)
	case models.TaskTypeExtend:
		go h.runExtendTask(task.ID, req.Description, taskDir, module)
	case models.TaskTypeExport:
		go h.runExportTask(task.ID, req.Description, taskDir, req.ExportFormat)
	case models.TaskTypeLoop:
		go h.runLoopTask(task.ID, req.Description, taskDir, req.Publish, req.Build, req.Push, req.Loop, module)
	default:
		resumeFrom := 0
		if isResume {
			resumeFrom = req.ResumeFrom
		}
		go h.runTask(task.ID, req.Description, taskDir, req.Publish, req.Build, req.Push, resumeFrom, module)
	}

	c.JSON(http.StatusOK, task)
}

// List handles GET /api/autodev/tasks?password=xxx&limit=20&offset=0&status=&type=
// Only running tasks get their autodev_state included to avoid expensive disk reads for every task.
func (h *AutoDevHandler) List(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	statusFilter := c.Query("status")
	typeFilter := c.Query("type")
	if limit <= 0 || limit > 200 {
		limit = 20
	}

	tasks, err := h.db.ListAutoDevTasks(limit, offset, statusFilter, typeFilter)
	if err != nil {
		log.Printf("[AutoDev] ListAutoDevTasks error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败: " + err.Error()})
		return
	}
	if tasks == nil {
		tasks = []*models.AutoDevTask{}
	}

	total, _ := h.db.CountAutoDevTasks(statusFilter, typeFilter)

	// Only include autodev_state for running tasks — avoids reading state.json for every task
	result := make([]any, 0, len(tasks))
	for _, t := range tasks {
		m := taskToMap(t)
		if t.Status == "running" || t.Status == "failed" || t.Status == "paused" || t.Status == "stopped" {
			if state := readAutoDevState(t.WorkDir); state != nil {
				m["autodev_state"] = state
			}
		}
		result = append(result, m)
	}
	c.JSON(http.StatusOK, gin.H{"tasks": result, "total": total, "limit": limit, "offset": offset})
}

// ListProjects handles GET /api/autodev/projects?password=xxx
// Returns a list of unique project directories from task history
func (h *AutoDevHandler) ListProjects(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	projects, err := h.db.ListAutoDevProjects()
	if err != nil {
		log.Printf("[AutoDev] ListAutoDevProjects error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取项目列表失败: " + err.Error()})
		return
	}
	if projects == nil {
		projects = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// GetTask handles GET /api/autodev/tasks/:id?password=xxx
func (h *AutoDevHandler) GetTask(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}
	m := taskToMap(task)
	if state := readAutoDevState(task.WorkDir); state != nil {
		m["autodev_state"] = state
	}
	c.JSON(http.StatusOK, m)
}

// GetState handles GET /api/autodev/tasks/:id/state?password=xxx
// Returns the detailed autodev state.json (phase progress, current phase, etc.)
func (h *AutoDevHandler) GetState(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}
	state := readAutoDevState(task.WorkDir)
	if state == nil {
		state = map[string]any{"status": task.Status}
	}
	c.JSON(http.StatusOK, state)
}

// GetFiles handles GET /api/autodev/tasks/:id/files?password=xxx
func (h *AutoDevHandler) GetFiles(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}
	files := listTaskFiles(task.WorkDir)
	siteDir := filepath.Join(task.WorkDir, "_site")
	_, siteErr := os.Stat(siteDir)
	c.JSON(http.StatusOK, gin.H{"files": files, "has_site": siteErr == nil})
}

// GetFile handles GET /api/autodev/tasks/:id/file?password=xxx&path=RESULT.md
func (h *AutoDevHandler) GetFile(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	filePath := c.Query("path")
	if filePath == "" {
		filePath = "RESULT.md"
	}
	absPath, relPath, err := resolveTaskPath(task.WorkDir, filePath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "路径不合法"})
		return
	}

	maxBytes := parsePositiveInt(c.Query("max_bytes"), defaultFilePreviewBytes, 8*1024, maxPreviewBytes)
	maxLines := parsePositiveInt(c.Query("max_lines"), defaultFilePreviewLines, 20, maxPreviewLines)
	mode := normalizePreviewMode(c.Query("mode"))

	preview, err := buildTaskFilePreview(absPath, relPath, mode, maxBytes, maxLines)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.JSON(http.StatusOK, preview)
}

// GetRawFile handles GET /api/autodev/tasks/:id/raw?password=xxx&path=foo.png
// Streams the original file so the browser can preview images/audio/video/PDF natively.
func (h *AutoDevHandler) GetRawFile(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path 不能为空"})
		return
	}
	absPath, relPath, err := resolveTaskPath(task.WorkDir, filePath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "路径不合法"})
		return
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(relPath)))
	c.File(absPath)
}

// GetLogs handles GET /api/autodev/tasks/:id/logs?password=xxx&phase=driver
// phase: driver (default), session, or any phase name like "01-discover"
func (h *AutoDevHandler) GetLogs(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	phase := c.Query("phase")
	if phase == "" {
		phase = "driver"
	}
	maxBytes := parsePositiveInt(c.Query("max_bytes"), defaultLogPreviewBytes, 8*1024, maxPreviewBytes)
	maxLines := parsePositiveInt(c.Query("max_lines"), defaultLogPreviewLines, 20, maxPreviewLines)

	logDir := filepath.Join(task.WorkDir, ".autodev", "logs")
	// try phase.log first, then fallback to driver.log
	candidates := []string{
		filepath.Join(logDir, phase+".log"),
		filepath.Join(logDir, "driver.log"),
		filepath.Join(logDir, "session.log"),
		filepath.Join(task.WorkDir, "autodev.log"),
	}
	availablePhases := listAvailableLogPhases(logDir)
	for _, lp := range candidates {
		info, err := os.Stat(lp)
		if err == nil && !info.IsDir() {
			preview, err := readTextPreview(lp, "tail", maxBytes, maxLines)
			if err != nil {
				continue
			}
			c.JSON(http.StatusOK, gin.H{
				"logs":                 preview.Content,
				"available_phases":     availablePhases,
				"log_path":             filepath.Base(lp),
				"preview_mode":         preview.Mode,
				"total_bytes":          info.Size(),
				"display_bytes":        preview.DisplayBytes,
				"displayed_line_count": preview.DisplayedLineCount,
				"truncated":            preview.Truncated,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"logs":                 "",
		"available_phases":     availablePhases,
		"log_path":             "",
		"preview_mode":         "tail",
		"total_bytes":          0,
		"display_bytes":        0,
		"displayed_line_count": 0,
		"truncated":            false,
	})
}

// Download handles GET /api/autodev/tasks/:id/download?password=xxx
// Returns a ZIP archive containing all result files, process docs and logs
func (h *AutoDevHandler) Download(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	safeName := sanitizeTaskName(task.Description)
	zipName := fmt.Sprintf("autodev-%s-%s.zip", safeName, task.ID)

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipName))

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	filepath.WalkDir(task.WorkDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(task.WorkDir, path)
		// skip .git and other deep hidden dirs except .autodev
		parts := strings.SplitN(rel, string(os.PathSeparator), 2)
		if strings.HasPrefix(parts[0], ".") && parts[0] != ".autodev" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		w, err := zw.Create(rel)
		if err != nil {
			return nil
		}
		io.Copy(w, f)
		return nil
	})

	c.Status(http.StatusOK)
}

// GetSite handles GET /api/autodev/tasks/:id/site/*filepath?password=xxx
// Serves the static _site directory generated by MkDocs build
func (h *AutoDevHandler) GetSite(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	siteDir := filepath.Join(task.WorkDir, "_site")
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "_site 目录不存在，任务可能尚未完成或未执行 build"})
		return
	}

	filePath := c.Param("filepath")
	if filePath == "" || filePath == "/" {
		filePath = "/index.html"
	}

	absPath := filepath.Join(siteDir, filepath.Clean(filePath))
	if !strings.HasPrefix(absPath, filepath.Clean(siteDir)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "路径不合法"})
		return
	}

	c.File(absPath)
}

func controlStatus(action string) string {
	if action == "terminate" {
		return "stopped"
	}
	return "paused"
}

func resolveTaskStatus(workDir, fallback string) string {
	state := readAutoDevState(workDir)
	if state == nil {
		return fallback
	}
	if status, ok := state["status"].(string); ok {
		switch status {
		case "finished", "completed":
			return "completed"
		case "failed", "paused", "stopped":
			return status
		}
	}
	if action, ok := state["control_action"].(string); ok && action != "" {
		return controlStatus(action)
	}
	return fallback
}

func (h *AutoDevHandler) requestTaskControl(id string, task *models.AutoDevTask, action string) int {
	status := controlStatus(action)
	writeAutoDevState(task.WorkDir, map[string]any{
		"control_action":    action,
		"stop_requested_at": time.Now().Format(time.RFC3339),
		"status":            status,
	})

	if _, err := os.Stat(h.stopScriptPath); err == nil {
		cmd := exec.Command("python3", h.stopScriptPath, "--path", task.WorkDir)
		_ = cmd.Run()
	} else {
		stopFile := filepath.Join(task.WorkDir, ".autodev", "STOP")
		os.MkdirAll(filepath.Dir(stopFile), 0755)
		_ = os.WriteFile(stopFile, []byte(time.Now().Format(time.RFC3339)), 0644)
	}

	h.mu.Lock()
	if cmd, ok := h.processes[id]; ok && cmd.Process != nil {
		if action == "terminate" {
			_ = cmd.Process.Kill()
		} else {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	}
	h.mu.Unlock()

	resumeFrom := 1
	state := readAutoDevState(task.WorkDir)
	if state != nil {
		if last, ok := state["last_completed"]; ok {
			switch v := last.(type) {
			case float64:
				resumeFrom = int(v) + 2
			}
		}
	}
	if resumeFrom < 1 {
		resumeFrom = 1
	}
	return resumeFrom
}

// StopTask handles POST /api/autodev/tasks/:id/stop
// stop is treated as a pause: keep the worktree and allow later resume.
func (h *AutoDevHandler) StopTask(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	id := c.Param("id")
	task, err := h.db.GetAutoDevTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	resumeFrom := h.requestTaskControl(id, task, "pause")
	h.db.UpdateAutoDevTaskStatus(id, "paused", -1, 0)
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"status":      "paused",
		"resume_from": resumeFrom,
		"work_dir":    task.WorkDir,
	})
}

// TerminateTask handles POST /api/autodev/tasks/:id/terminate
// terminate stops the current execution more aggressively, but preserves files for later inspection or resume.
func (h *AutoDevHandler) TerminateTask(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	id := c.Param("id")
	task, err := h.db.GetAutoDevTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	resumeFrom := h.requestTaskControl(id, task, "terminate")
	h.db.UpdateAutoDevTaskStatus(id, "stopped", -1, 0)
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"status":      "stopped",
		"resume_from": resumeFrom,
		"work_dir":    task.WorkDir,
	})
}

// DeleteTask handles DELETE /api/autodev/tasks/:id?password=xxx
func (h *AutoDevHandler) DeleteTask(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	task, ok := h.loadTask(c)
	if !ok {
		return
	}

	h.mu.Lock()
	if cmd, ok2 := h.processes[task.ID]; ok2 && cmd.Process != nil {
		cmd.Process.Kill()
	}
	delete(h.processes, task.ID)
	h.mu.Unlock()

	os.RemoveAll(task.WorkDir)
	h.db.DeleteAutoDevTask(task.ID)
	c.JSON(http.StatusNoContent, nil)
}

// runTask executes autodev in a background goroutine
func (h *AutoDevHandler) runTask(id, description, workDir string, publish, build, push bool, resumeFrom int, module string) {
	module = models.NormalizeAutoDevModule(module)
	args := []string{description, "--path", workDir, "--module", module}
	if publish {
		args = append(args, "--publish")
	}
	if build {
		args = append(args, "--build")
	}
	if push {
		args = append(args, "--push")
	}
	if resumeFrom > 0 {
		args = append(args, "--from", strconv.Itoa(resumeFrom))
	}

	cmd := exec.Command(h.autodevPath, args...)
	cmd.Dir = workDir

	// Create log directory and transfer ownership to non-root autodev user (uid 1001).
	// Chown is more reliable than chmod for cross-user file access.
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0755)
	os.Chown(workDir, autodevUID, autodevGID)
	os.Chown(filepath.Join(workDir, ".autodev"), autodevUID, autodevGID)
	os.Chown(logDir, autodevUID, autodevGID)

	setSysProcCredential(cmd)
	cmd.Env = h.buildEnv()

	h.mu.Lock()
	h.processes[id] = cmd
	h.mu.Unlock()

	if err := cmd.Start(); err != nil {
		h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
		return
	}
	h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)

	err := cmd.Wait()
	exitCode := 0
	status := "completed"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	status = resolveTaskStatus(workDir, status)
	if status == "completed" {
		exitCode = 0
	}
	if status == "paused" || status == "stopped" {
		exitCode = -1
	}

	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	h.mu.Lock()
	delete(h.processes, id)
	h.mu.Unlock()
}

// runAskTask executes `autodev ask "description" --path workDir` in a background goroutine.
// The new clawtest supports the `ask` subcommand which appends Q&A to process/qa.md.
func (h *AutoDevHandler) runAskTask(id, description, workDir, module string) {
	// Ensure directories exist with correct ownership for non-root autodev user (uid 1001)
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(filepath.Join(workDir, "process"), 0755)
	os.Chown(workDir, autodevUID, autodevGID)
	os.Chown(filepath.Join(workDir, ".autodev"), autodevUID, autodevGID)
	os.Chown(logDir, autodevUID, autodevGID)
	os.Chown(filepath.Join(workDir, "process"), autodevUID, autodevGID)

	// Execute: autodev ask "description" --path workDir --module <cc|codex>
	// Matches the new clawtest driver.py ask subcommand.
	module = models.NormalizeAutoDevModule(module)
	cmd := exec.Command(h.autodevPath, "ask", description, "--path", workDir, "--module", module)
	cmd.Dir = workDir
	setSysProcCredential(cmd)
	cmd.Env = h.buildEnv()
	// Do NOT redirect stdout/stderr — runner.py writes all logs (including claude output)
	// directly to .autodev/logs/driver.log and ask-N.log, which GetLogs already reads.

	h.mu.Lock()
	h.processes[id] = cmd
	h.mu.Unlock()

	if err := cmd.Start(); err != nil {
		log.Printf("[AutoDev] runAskTask start error: %v", err)
		h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
		h.mu.Lock()
		delete(h.processes, id)
		h.mu.Unlock()
		return
	}
	h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)

	// ask_project does NOT call mark_phase_start. Write initial state manually
	// so the frontend progress bar shows something during execution.
	writeAutoDevState(workDir, map[string]any{
		"status":      "running",
		"phase_label": "ASK 问答",
	})

	waitErr := cmd.Wait()
	exitCode := 0
	status := "completed"
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	status = resolveTaskStatus(workDir, status)
	if status == "completed" {
		writeAutoDevState(workDir, map[string]any{
			"status":      "completed",
			"phase_label": "ASK 问答",
		})
	} else if status == "failed" {
		writeAutoDevState(workDir, map[string]any{
			"status":      "failed",
			"phase_label": "ASK 问答",
		})
	}

	// ask_project writes answers to process/qa.md.
	// Verify it was created; if not, mark as failed even when exit code is 0.
	qaFile := filepath.Join(workDir, "process", "qa.md")
	if status == "completed" {
		if _, err := os.Stat(qaFile); err != nil {
			log.Printf("[AutoDev] runAskTask: qa.md not found after completion, marking failed")
			status = "failed"
		}
	}
	if status == "completed" {
		exitCode = 0
	}
	if status == "paused" || status == "stopped" {
		exitCode = -1
	}

	h.db.UpdateAutoDevTaskResult(id, qaFile)
	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	h.mu.Lock()
	delete(h.processes, id)
	h.mu.Unlock()
}

// runExtendTask executes `autodev extend "description" --path workDir` in a background goroutine.
// The new clawtest supports the `extend` subcommand which adds new requirements to existing projects.
// Each iteration writes to process/iter-N/ and updates RESULT.md.
func (h *AutoDevHandler) runExtendTask(id, description, workDir, module string) {
	// Ensure directories exist with correct ownership for non-root autodev user (uid 1001)
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0755)
	os.Chown(workDir, autodevUID, autodevGID)
	os.Chown(filepath.Join(workDir, ".autodev"), autodevUID, autodevGID)
	os.Chown(logDir, autodevUID, autodevGID)

	// Execute: autodev extend "description" --path workDir --module <cc|codex>
	// Matches the new clawtest driver.py extend subcommand.
	module = models.NormalizeAutoDevModule(module)
	cmd := exec.Command(h.autodevPath, "extend", description, "--path", workDir, "--module", module)
	cmd.Dir = workDir
	setSysProcCredential(cmd)
	cmd.Env = h.buildEnv()
	// Do NOT redirect stdout/stderr — runner.py writes all logs (including claude output)
	// directly to .autodev/logs/driver.log and extend-iter-N.log, which GetLogs already reads.

	h.mu.Lock()
	h.processes[id] = cmd
	h.mu.Unlock()

	if err := cmd.Start(); err != nil {
		log.Printf("[AutoDev] runExtendTask start error: %v", err)
		h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
		h.mu.Lock()
		delete(h.processes, id)
		h.mu.Unlock()
		return
	}
	h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)

	// extend_project does NOT call mark_phase_start/mark_finished, so state.json
	// won't have status/phase_label updates. Write initial state manually so the
	// frontend progress bar shows something during execution.
	writeAutoDevState(workDir, map[string]any{
		"status":      "running",
		"phase_label": "EXTEND 扩展",
	})

	waitErr := cmd.Wait()
	exitCode := 0
	status := "completed"
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	status = resolveTaskStatus(workDir, status)
	if status == "completed" {
		writeAutoDevState(workDir, map[string]any{
			"status":      "completed",
			"phase_label": "EXTEND 扩展",
		})
	} else if status == "failed" {
		writeAutoDevState(workDir, map[string]any{
			"status":      "failed",
			"phase_label": "EXTEND 扩展",
		})
	}
	if status == "completed" {
		exitCode = 0
	}
	if status == "paused" || status == "stopped" {
		exitCode = -1
	}

	// extend_project writes result to RESULT.md and process/iter-N/result.md.
	resultFile := filepath.Join(workDir, "RESULT.md")
	h.db.UpdateAutoDevTaskResult(id, resultFile)
	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	h.mu.Lock()
	delete(h.processes, id)
	h.mu.Unlock()
}

// runLoopTask executes `autodev --loop [N] "description" --path workDir` in a background goroutine.
// loop=0 means no loop (same as develop), loop=-1 means infinite, loop=N means at most N iterations.
func (h *AutoDevHandler) runLoopTask(id, description, workDir string, publish, build, push bool, loop int, module string) {
	module = models.NormalizeAutoDevModule(module)

	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0755)
	os.Chown(workDir, autodevUID, autodevGID)
	os.Chown(filepath.Join(workDir, ".autodev"), autodevUID, autodevGID)
	os.Chown(logDir, autodevUID, autodevGID)

	args := []string{description, "--path", workDir, "--module", module}
	if publish {
		args = append(args, "--publish")
	}
	if build {
		args = append(args, "--build")
	}
	if push {
		args = append(args, "--push")
	}
	// --loop [N]: -1=无限(不传N), N>0=最多N次
	if loop < 0 {
		args = append(args, "--loop")
	} else if loop > 0 {
		args = append(args, "--loop", strconv.Itoa(loop))
	}

	cmd := exec.Command(h.autodevPath, args...)
	cmd.Dir = workDir
	setSysProcCredential(cmd)
	cmd.Env = h.buildEnv()

	h.mu.Lock()
	h.processes[id] = cmd
	h.mu.Unlock()

	if err := cmd.Start(); err != nil {
		log.Printf("[AutoDev] runLoopTask start error: %v", err)
		h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
		h.mu.Lock()
		delete(h.processes, id)
		h.mu.Unlock()
		return
	}
	h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)

	waitErr := cmd.Wait()
	exitCode := 0
	status := "completed"
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	status = resolveTaskStatus(workDir, status)
	if status == "completed" {
		exitCode = 0
	}
	if status == "paused" || status == "stopped" {
		exitCode = -1
	}

	resultFile := filepath.Join(workDir, "RESULT.md")
	h.db.UpdateAutoDevTaskResult(id, resultFile)
	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	h.mu.Lock()
	delete(h.processes, id)
	h.mu.Unlock()
}

// buildEnv builds the environment variables for running autodev tasks
func (h *AutoDevHandler) buildEnv() []string {
	env := make([]string, 0, len(os.Environ())+3)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "UV_CACHE_DIR=") {
			env = append(env, e)
		}
	}
	env = append(env, "HOME="+resolveHome())
	env = append(env, "UV_CACHE_DIR=/tmp/uv-cache-autodev")
	// Inject SSH key for git operations
	if gitSSHCmd := h.gitSSHCommand(); gitSSHCmd != "" {
		filtered := env[:0]
		for _, e := range env {
			if !strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
				filtered = append(filtered, e)
			}
		}
		env = append(filtered, "GIT_SSH_COMMAND="+gitSSHCmd)
	}
	return env
}

// runExportTask executes autodev export in a background goroutine
func (h *AutoDevHandler) runExportTask(id, description, workDir, exportFormat string) {
	// Default format is zip
	if exportFormat == "" {
		exportFormat = "zip"
	}

	args := []string{description, "--path", workDir, "--export", exportFormat}

	cmd := exec.Command(h.autodevPath, args...)
	cmd.Dir = workDir

	// Create log directory and transfer ownership to non-root autodev user (uid 1001)
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0755)
	os.Chown(workDir, autodevUID, autodevGID)
	os.Chown(filepath.Join(workDir, ".autodev"), autodevUID, autodevGID)
	os.Chown(logDir, autodevUID, autodevGID)

	// Set up environment
	setSysProcCredential(cmd)
	cmd.Env = h.buildEnv()

	h.mu.Lock()
	h.processes[id] = cmd
	h.mu.Unlock()

	if err := cmd.Start(); err != nil {
		h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
		return
	}
	h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)

	err := cmd.Wait()
	exitCode := 0
	status := "completed"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	status = resolveTaskStatus(workDir, status)
	if status == "completed" {
		exitCode = 0
	}
	if status == "paused" || status == "stopped" {
		exitCode = -1
	}

	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	h.mu.Lock()
	delete(h.processes, id)
	h.mu.Unlock()
}

// ---- helpers ----

func (h *AutoDevHandler) checkPassword(c *gin.Context) bool {
	password := c.Query("password")
	if h.adminPassword == "" || password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return false
	}
	return true
}

func (h *AutoDevHandler) loadTask(c *gin.Context) (*models.AutoDevTask, bool) {
	id := c.Param("id")
	task, err := h.db.GetAutoDevTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return nil, false
	}
	return task, true
}

type textPreview struct {
	Content            string
	Mode               string
	Truncated          bool
	DisplayBytes       int
	DisplayedLineCount int
}

func resolveTaskPath(rootDir, relativePath string) (string, string, error) {
	rootDir = filepath.Clean(rootDir)
	relativePath = strings.TrimSpace(relativePath)
	relativePath = strings.TrimPrefix(relativePath, "/")
	relativePath = strings.TrimPrefix(relativePath, "\\")
	if relativePath == "" || relativePath == "." {
		return "", "", fmt.Errorf("empty path")
	}

	absPath := filepath.Join(rootDir, filepath.Clean(relativePath))
	rel, err := filepath.Rel(rootDir, absPath)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path escapes root")
	}
	return absPath, filepath.ToSlash(rel), nil
}

func parsePositiveInt(raw string, fallback, minValue, maxValue int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func normalizePreviewMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "head", "tail", "full":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "auto"
	}
}

func buildTaskFilePreview(absPath, relPath, mode string, maxBytes, maxLines int) (gin.H, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, os.ErrNotExist
	}

	sample, err := readFileSample(absPath, fileSniffBytes)
	if err != nil {
		return nil, err
	}
	kind, mimeType, isText := detectPreviewKind(filepath.Base(absPath), sample)
	preview := gin.H{
		"path":      relPath,
		"kind":      kind,
		"mime_type": mimeType,
		"size":      info.Size(),
		"is_text":   isText,
	}
	if !isText {
		preview["previewable"] = kind == "image" || kind == "audio" || kind == "video" || kind == "pdf"
		return preview, nil
	}

	resolvedMode := mode
	if resolvedMode == "auto" {
		if classifyFile(relPath, filepath.Base(absPath)) == "log" {
			resolvedMode = "tail"
		} else if info.Size() > int64(maxBytes) {
			resolvedMode = "head"
		} else {
			resolvedMode = "full"
		}
	}

	text, err := readTextPreview(absPath, resolvedMode, maxBytes, maxLines)
	if err != nil {
		return nil, err
	}

	preview["previewable"] = true
	preview["preview_mode"] = text.Mode
	preview["content"] = text.Content
	preview["truncated"] = text.Truncated
	preview["total_bytes"] = info.Size()
	preview["display_bytes"] = text.DisplayBytes
	preview["displayed_line_count"] = text.DisplayedLineCount
	return preview, nil
}

func readFileSample(path string, maxBytes int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, int64(maxBytes)))
}

func readTextPreview(path, mode string, maxBytes, maxLines int) (textPreview, error) {
	f, err := os.Open(path)
	if err != nil {
		return textPreview{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return textPreview{}, err
	}

	var (
		raw       []byte
		truncated bool
	)

	switch mode {
	case "tail":
		raw, truncated, err = readTailBytes(f, info.Size(), int64(maxBytes))
	default:
		limit := info.Size()
		if limit > int64(maxBytes) {
			limit = int64(maxBytes)
			truncated = true
		}
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return textPreview{}, err
		}
		raw, err = io.ReadAll(io.LimitReader(f, limit))
	}
	if err != nil {
		return textPreview{}, err
	}

	if mode == "tail" && truncated {
		if idx := bytes.IndexByte(raw, '\n'); idx >= 0 {
			raw = raw[idx+1:]
		}
	}

	var lineTrimmed bool
	if mode == "tail" {
		raw, lineTrimmed = trimLastLines(raw, maxLines)
	} else {
		raw, lineTrimmed = trimFirstLines(raw, maxLines)
	}
	truncated = truncated || lineTrimmed

	content := string(bytes.ToValidUTF8(raw, []byte("�")))
	return textPreview{
		Content:            content,
		Mode:               mode,
		Truncated:          truncated,
		DisplayBytes:       len(raw),
		DisplayedLineCount: countLines(content),
	}, nil
}

func readTailBytes(f *os.File, fileSize, maxBytes int64) ([]byte, bool, error) {
	if maxBytes <= 0 || fileSize <= maxBytes {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, false, err
		}
		data, err := io.ReadAll(f)
		return data, false, err
	}

	start := fileSize - maxBytes
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, false, err
	}
	data, err := io.ReadAll(f)
	return data, true, err
}

func trimFirstLines(data []byte, maxLines int) ([]byte, bool) {
	if maxLines <= 0 || len(data) == 0 {
		return data, false
	}
	lines := bytes.SplitAfter(data, []byte("\n"))
	if len(lines) <= maxLines {
		return data, false
	}
	return bytes.Join(lines[:maxLines], nil), true
}

func trimLastLines(data []byte, maxLines int) ([]byte, bool) {
	if maxLines <= 0 || len(data) == 0 {
		return data, false
	}
	lines := bytes.SplitAfter(data, []byte("\n"))
	if len(lines) <= maxLines {
		return data, false
	}
	return bytes.Join(lines[len(lines)-maxLines:], nil), true
}

func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func listAvailableLogPhases(logDir string) []string {
	logFiles, _ := filepath.Glob(filepath.Join(logDir, "*.log"))
	phases := make([]string, 0, len(logFiles))
	for _, lf := range logFiles {
		phases = append(phases, strings.TrimSuffix(filepath.Base(lf), ".log"))
	}
	sort.Slice(phases, func(i, j int) bool {
		priority := map[string]int{"driver": 0, "session": 1}
		pi, okI := priority[phases[i]]
		pj, okJ := priority[phases[j]]
		switch {
		case okI && okJ:
			return pi < pj
		case okI:
			return true
		case okJ:
			return false
		default:
			return phases[i] < phases[j]
		}
	})
	return phases
}

func detectPreviewKind(name string, sample []byte) (string, string, bool) {
	kind := inferPreviewKindByName(name)
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if mimeType == "" && len(sample) > 0 {
		mimeType = http.DetectContentType(sample)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image", mimeType, false
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio", mimeType, false
	case strings.HasPrefix(mimeType, "video/"):
		return "video", mimeType, false
	case mimeType == "application/pdf":
		return "pdf", mimeType, false
	case strings.HasPrefix(mimeType, "text/"):
		if kind == "markdown" {
			return kind, mimeType, true
		}
		return "text", mimeType, true
	case looksLikeText(sample):
		if kind == "markdown" {
			return kind, "text/markdown; charset=utf-8", true
		}
		return "text", "text/plain; charset=utf-8", true
	}

	switch kind {
	case "image", "audio", "video", "pdf", "archive":
		return kind, mimeType, false
	case "markdown":
		return kind, "text/markdown; charset=utf-8", true
	case "text":
		return kind, "text/plain; charset=utf-8", true
	default:
		return "binary", mimeType, false
	}
}

func inferPreviewKindByName(name string) string {
	lower := strings.ToLower(name)
	ext := strings.ToLower(filepath.Ext(lower))

	switch lower {
	case "dockerfile", "makefile", "readme", "readme.md", ".gitignore", ".env", ".env.example":
		return "text"
	}

	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".avif":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".aac", ".flac", ".opus":
		return "audio"
	case ".mp4", ".webm", ".mov", ".mkv", ".avi", ".m4v":
		return "video"
	case ".pdf":
		return "pdf"
	case ".zip", ".tar", ".gz", ".tgz", ".bz2", ".xz", ".7z", ".rar", ".jar", ".war":
		return "archive"
	case ".txt", ".log", ".json", ".yaml", ".yml", ".toml", ".xml", ".csv", ".ini", ".conf",
		".sql", ".sh", ".bash", ".zsh", ".fish", ".py", ".rb", ".php", ".go", ".rs", ".java",
		".kt", ".js", ".jsx", ".ts", ".tsx", ".vue", ".css", ".scss", ".less", ".html", ".htm",
		".c", ".cc", ".cpp", ".cxx", ".h", ".hpp", ".hh":
		return "text"
	default:
		return "binary"
	}
}

func looksLikeText(sample []byte) bool {
	if len(sample) == 0 {
		return true
	}
	if !utf8.Valid(sample) {
		return false
	}
	var controlCount int
	for _, b := range sample {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			controlCount++
		}
	}
	return controlCount*20 < len(sample)
}

// readAutoDevState reads .autodev/state.json from a task work directory
func readAutoDevState(workDir string) map[string]any {
	p := filepath.Join(workDir, ".autodev", "state.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return state
}

// writeAutoDevState merges the given fields into state.json, preserving existing keys.
func writeAutoDevState(workDir string, fields map[string]any) {
	p := filepath.Join(workDir, ".autodev", "state.json")
	os.MkdirAll(filepath.Dir(p), 0755)

	state := map[string]any{}
	if data, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(data, &state)
	}
	for k, v := range fields {
		state[k] = v
	}
	if data, err := json.Marshal(state); err == nil {
		os.WriteFile(p, data, 0644)
	}
}

// listTaskFiles recursively walks workDir and returns all relevant files with categories.
//
// Categories:
//
//	"result"  – RESULT.md (the final deliverable)
//	"code"    – project source files generated by autodev (*.cpp, *.go, *.py, …)
//	"process" – per-phase process docs in process/
//	"log"     – execution logs in .autodev/logs/
//	"docs"    – mkdocs.yml and docs/ directory
//	"state"   – .autodev/state.json
//
// Skipped entirely: .git/, node_modules/, __pycache__/, _site/ (too many files),
// and files deeper than 6 levels. Large media files are still listed with
// metadata so the frontend can offer native previews instead of raw binary text.
func listTaskFiles(workDir string) []map[string]any {
	var files []map[string]any

	filepath.WalkDir(workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(workDir, path)
		if rel == "." {
			return nil
		}

		// Skip entire subtrees we don't want
		if d.IsDir() {
			base := d.Name()
			switch base {
			case ".git", "node_modules", "__pycache__", "_site", ".cache", "vendor":
				return filepath.SkipDir
			}
			return nil
		}

		// Depth limit: skip files more than 6 levels deep
		if strings.Count(rel, string(os.PathSeparator)) >= 6 {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		kind := inferPreviewKindByName(d.Name())
		sizeLimit := int64(maxListedTextFileSize)
		switch kind {
		case "image", "audio", "video", "pdf":
			sizeLimit = maxListedMediaFileSize
		case "archive", "binary":
			sizeLimit = maxListedBinaryFileSize
		}
		if info.Size() > sizeLimit {
			return nil
		}

		category := classifyFile(rel, d.Name())
		if category == "" {
			return nil // skip
		}

		files = append(files, map[string]any{
			"path":     rel,
			"name":     d.Name(),
			"size":     info.Size(),
			"mtime":    info.ModTime().Format(time.RFC3339),
			"category": category,
			"kind":     kind,
		})
		return nil
	})

	return files
}

// classifyFile returns the display category for a file, or "" to skip it.
func classifyFile(rel, name string) string {
	parts := strings.Split(rel, string(os.PathSeparator))
	topDir := parts[0]

	// .autodev/ subtree
	if topDir == ".autodev" {
		if len(parts) >= 2 && parts[1] == "logs" {
			return "log"
		}
		if name == "state.json" {
			return "state"
		}
		return "" // skip STOP file and other internal files
	}

	// _site/ – skip individual files; the frontend shows a separate "preview" button
	if topDir == "_site" {
		return ""
	}

	// Top-level result document
	if name == "RESULT.md" && len(parts) == 1 {
		return "result"
	}

	// Process docs
	if topDir == "process" {
		return "process"
	}

	// MkDocs config and docs/ source
	if name == "mkdocs.yml" || topDir == "docs" {
		return "docs"
	}

	// Everything else at root or in subdirs = project code output
	return "code"
}

// taskToMap converts a task to a map for JSON serialization with extra fields
func taskToMap(t *models.AutoDevTask) map[string]any {
	m := map[string]any{
		"id":           t.ID,
		"type":         t.Type,
		"description":  t.Description,
		"options":      t.Options,
		"status":       t.Status,
		"exit_code":    t.ExitCode,
		"work_dir":     t.WorkDir,
		"pid":          t.PID,
		"created_at":   t.CreatedAt,
		"started_at":   t.StartedAt,
		"completed_at": t.CompletedAt,
		"result_file":  t.ResultFile,
		"module":       t.Module,
	}
	return m
}

func sanitizeTaskName(s string) string {
	var result []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result = append(result, r)
		} else if r == ' ' || r == '-' || r == '_' {
			result = append(result, '-')
		}
	}
	name := string(result)
	if len(name) > 30 {
		name = name[:30]
	}
	if name == "" {
		name = "task"
	}
	return strings.Trim(name, "-")
}

// ============================================================
// CLI 版本管理
// ============================================================

// CLIInfo holds CLI version info
type CLIInfo struct {
	Version     string `json:"version"`
	Path        string `json:"path"`
	NpmVersion  string `json:"npm_version"`
	NodeVersion string `json:"node_version"`
	Available   bool   `json:"available"`
}

// ClaudeHealth holds Claude API connectivity test result
type ClaudeHealth struct {
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	HasToken  bool   `json:"has_token"`
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
	Response  string `json:"response,omitempty"`
}

// CLITest holds a smoke-test result for an installed CLI runtime.
type CLITest struct {
	Version   string `json:"version"`
	Path      string `json:"path"`
	Home      string `json:"home"`
	Available bool   `json:"available"`
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
	Response  string `json:"response,omitempty"`
}

func trimCommandOutput(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max]) + "...<truncated>"
}

func stripEnvVar(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func extractCodexCLIResponse(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lastMessage := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if eventType, _ := event["type"].(string); eventType != "item.completed" {
			continue
		}

		item, _ := event["item"].(map[string]any)
		if item == nil {
			continue
		}
		if itemType, _ := item["type"].(string); itemType != "agent_message" {
			continue
		}
		if text, _ := item["text"].(string); strings.TrimSpace(text) != "" {
			lastMessage = strings.TrimSpace(text)
		}
	}

	if lastMessage != "" {
		return lastMessage
	}
	return trimCommandOutput(output, 2000)
}

// TestClaudeCLI handles GET /api/autodev/claude/cli/test?password=xxx
// It runs a minimal non-interactive Claude Code request with the same HOME/user
// strategy as task execution, so the result matches actual runtime behavior.
func (h *AutoDevHandler) TestClaudeCLI(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}

	result := CLITest{Home: resolveHome(), ExitCode: 0}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		result.Error = "Claude Code CLI 未安装"
		c.JSON(http.StatusOK, result)
		return
	}

	result.Available = true
	result.Path = claudePath
	if out, err := exec.Command(claudePath, "--version").Output(); err == nil {
		result.Version = strings.TrimSpace(string(out))
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, claudePath, "--print", "--dangerously-skip-permissions", "-p", "Reply with exactly pong and nothing else.")
	cmd.Dir = h.dataDir
	setSysProcCredential(cmd)
	cmd.Env = stripEnvVar(h.buildEnv(), "CLAUDECODE")

	start := time.Now()
	out, err := cmd.CombinedOutput()
	result.LatencyMs = time.Since(start).Milliseconds()
	output := trimCommandOutput(string(out), 2000)

	if ctx.Err() == context.DeadlineExceeded {
		result.Error = "CLI 测试超时（45s）"
		if output != "" {
			result.Error += ": " + output
		}
		c.JSON(http.StatusOK, result)
		return
	}

	if err != nil {
		result.ExitCode = -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		if output != "" {
			result.Error = output
		} else {
			result.Error = err.Error()
		}
		c.JSON(http.StatusOK, result)
		return
	}

	result.OK = true
	result.Response = output
	c.JSON(http.StatusOK, result)
}

// TestCodexCLI handles GET /api/autodev/codex/cli/test?password=xxx
// It runs a minimal Codex exec request with the same HOME/user strategy as task execution.
func (h *AutoDevHandler) TestCodexCLI(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}

	result := CLITest{Home: resolveHome(), ExitCode: 0}

	codexPath, err := exec.LookPath("codex")
	if err != nil {
		result.Error = "Codex CLI 未安装"
		c.JSON(http.StatusOK, result)
		return
	}

	result.Available = true
	result.Path = codexPath
	if out, err := exec.Command(codexPath, "--version").Output(); err == nil {
		result.Version = strings.TrimSpace(string(out))
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, codexPath, "exec", "--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox", "-C", h.dataDir, "Reply with exactly pong and nothing else.")
	cmd.Dir = h.dataDir
	setSysProcCredential(cmd)
	cmd.Env = h.buildEnv()

	start := time.Now()
	out, err := cmd.CombinedOutput()
	result.LatencyMs = time.Since(start).Milliseconds()
	rawOutput := string(out)
	output := trimCommandOutput(rawOutput, 2000)

	if ctx.Err() == context.DeadlineExceeded {
		result.Error = "CLI 测试超时（45s）"
		if output != "" {
			result.Error += ": " + output
		}
		c.JSON(http.StatusOK, result)
		return
	}

	if err != nil {
		result.ExitCode = -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		if output != "" {
			result.Error = output
		} else {
			result.Error = err.Error()
		}
		c.JSON(http.StatusOK, result)
		return
	}

	result.OK = true
	result.Response = extractCodexCLIResponse(rawOutput)
	c.JSON(http.StatusOK, result)
}

// TestModel handles GET /api/autodev/claude/test?password=xxx
// Sends a minimal message to the configured Anthropic endpoint and checks connectivity
func (h *AutoDevHandler) TestModel(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}

	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-3-haiku-20240307"
	}
	authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")

	result := ClaudeHealth{BaseURL: baseURL, Model: model, HasToken: authToken != ""}

	if authToken == "" {
		result.Error = "ANTHROPIC_AUTH_TOKEN 未配置"
		c.JSON(http.StatusOK, result)
		return
	}

	reqBody := map[string]any{
		"model":      model,
		"max_tokens": 16,
		"messages":   []map[string]any{{"role": "user", "content": "reply with: pong"}},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/messages"
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(bodyBytes)))
	if err != nil {
		result.Error = "构造请求失败: " + err.Error()
		c.JSON(http.StatusOK, result)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", authToken)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = "请求失败: " + err.Error()
		c.JSON(http.StatusOK, result)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		c.JSON(http.StatusOK, result)
		return
	}

	var respJSON map[string]any
	if err := json.Unmarshal(respBody, &respJSON); err == nil {
		if content, ok := respJSON["content"].([]any); ok && len(content) > 0 {
			if block, ok := content[0].(map[string]any); ok {
				if text, ok := block["text"].(string); ok {
					result.Response = strings.TrimSpace(text)
				}
			}
		}
	}

	result.OK = true
	c.JSON(http.StatusOK, result)
}

func getCLIInfo(binary string) CLIInfo {
	info := CLIInfo{}
	if path, err := exec.LookPath(binary); err == nil {
		info.Path = path
		info.Available = true
	}
	if out, err := exec.Command(binary, "--version").Output(); err == nil {
		info.Version = strings.TrimSpace(string(out))
		info.Available = true
	}
	if out, err := exec.Command("npm", "--version").Output(); err == nil {
		info.NpmVersion = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("node", "--version").Output(); err == nil {
		info.NodeVersion = strings.TrimSpace(string(out))
	}
	return info
}

// GetClaudeVersion handles GET /api/autodev/claude/version?password=xxx
func (h *AutoDevHandler) GetClaudeVersion(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	c.JSON(http.StatusOK, getCLIInfo("claude"))
}

// GetCodexVersion handles GET /api/autodev/codex/version?password=xxx
func (h *AutoDevHandler) GetCodexVersion(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	c.JSON(http.StatusOK, getCLIInfo("codex"))
}

func (h *AutoDevHandler) streamNpmCLIUpdate(c *gin.Context, displayName, binary, npmPackage string) {
	if !h.checkPasswordQuery(c) {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式输出"})
		return
	}

	sendEvent := func(event, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}
	sendLine := func(line string) {
		b, _ := json.Marshal(map[string]string{"line": line})
		sendEvent("log", string(b))
	}

	installCommand := fmt.Sprintf("npm install -g %s", npmPackage)
	sendLine(fmt.Sprintf("🚀 开始更新 %s...", displayName))
	sendLine("执行: " + installCommand)

	oldVersion := ""
	if out, err := exec.Command(binary, "--version").Output(); err == nil {
		oldVersion = strings.TrimSpace(string(out))
		sendLine(fmt.Sprintf("当前版本: %s", oldVersion))
	} else {
		sendLine(fmt.Sprintf("⚠️  当前未安装 %s", displayName))
	}

	sendLine("")

	cmd := exec.Command("npm", "install", "-g", npmPackage, "--unsafe-perm")
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendLine("❌ 启动更新失败: " + err.Error())
		sendEvent("done", `{"success":false}`)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sendLine("❌ 启动更新失败: " + err.Error())
		sendEvent("done", `{"success":false}`)
		return
	}
	if err := cmd.Start(); err != nil {
		sendLine("❌ 启动 npm 失败: " + err.Error())
		sendEvent("done", `{"success":false}`)
		return
	}

	done := make(chan struct{}, 2)
	stream := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			sendLine(scanner.Text())
		}
		done <- struct{}{}
	}
	go stream(stdout)
	go stream(stderr)
	<-done
	<-done

	err = cmd.Wait()
	sendLine("")
	if err != nil {
		sendLine("❌ 更新失败: " + err.Error())
		sendEvent("done", `{"success":false,"error":"update failed"}`)
		return
	}

	newVersion := ""
	if out, err2 := exec.Command(binary, "--version").Output(); err2 == nil {
		newVersion = strings.TrimSpace(string(out))
	}
	if newVersion != "" && newVersion != oldVersion {
		sendLine(fmt.Sprintf("✅ 更新成功！%s → %s", oldVersion, newVersion))
	} else if newVersion != "" {
		sendLine(fmt.Sprintf("✅ 已是最新版本: %s", newVersion))
	} else {
		sendLine("✅ 更新完成")
	}

	b, _ := json.Marshal(map[string]string{"old_version": oldVersion, "new_version": newVersion})
	sendEvent("done", string(b))
}

// UpdateClaude handles GET /api/autodev/claude/update/stream?password=xxx
// Streams npm install output via SSE
func (h *AutoDevHandler) UpdateClaude(c *gin.Context) {
	h.streamNpmCLIUpdate(c, "Claude Code CLI", "claude", "@anthropic-ai/claude-code@latest")
}

// UpdateCodex handles GET /api/autodev/codex/update/stream?password=xxx
// Streams npm install output via SSE
func (h *AutoDevHandler) UpdateCodex(c *gin.Context) {
	h.streamNpmCLIUpdate(c, "Codex CLI", "codex", "@openai/codex@latest")
}

// ClawtestInfo holds clawtest repo version info
type ClawtestInfo struct {
	Commit      string `json:"commit"`
	CommitShort string `json:"commit_short"`
	CommitDate  string `json:"commit_date"`
	Branch      string `json:"branch"`
	Path        string `json:"path"`
	Available   bool   `json:"available"`
}

// GetClawtestVersion handles GET /api/autodev/clawtest/version?password=xxx
func (h *AutoDevHandler) GetClawtestVersion(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}

	repoDir := filepath.Dir(filepath.Dir(h.autodevPath)) // /opt/clawtest
	info := ClawtestInfo{Path: repoDir}

	if _, err := os.Stat(repoDir); err != nil {
		c.JSON(http.StatusOK, info)
		return
	}
	info.Available = true

	if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output(); err == nil {
		info.Commit = strings.TrimSpace(string(out))
		if len(info.Commit) >= 7 {
			info.CommitShort = info.Commit[:7]
		}
	}
	if out, err := exec.Command("git", "-C", repoDir, "log", "-1", "--format=%ci").Output(); err == nil {
		info.CommitDate = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("git", "-C", repoDir, "branch", "--show-current").Output(); err == nil {
		info.Branch = strings.TrimSpace(string(out))
	}

	c.JSON(http.StatusOK, info)
}

// UpdateClawtest handles GET /api/autodev/clawtest/update/stream?password=xxx
// Streams `git pull` output via SSE to update clawtest in-place.
func (h *AutoDevHandler) UpdateClawtest(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式输出"})
		return
	}

	sendEvent := func(event, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}
	sendLine := func(line string) {
		b, _ := json.Marshal(map[string]string{"line": line})
		sendEvent("log", string(b))
	}

	repoDir := filepath.Dir(filepath.Dir(h.autodevPath)) // /opt/clawtest

	if _, err := os.Stat(repoDir); err != nil {
		sendLine("❌ clawtest 目录不存在: " + repoDir)
		sendEvent("done", `{"success":false}`)
		return
	}

	sendLine("🚀 开始更新 clawtest...")
	sendLine("执行: git pull (目录: " + repoDir + ")")

	// record commit before update
	oldCommit := ""
	if out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--short", "HEAD").Output(); err == nil {
		oldCommit = strings.TrimSpace(string(out))
		sendLine("当前版本: " + oldCommit)
	}
	sendLine("")

	// Use fetch + reset --hard to avoid divergent-branch conflicts.
	// This always brings the local copy in sync with remote HEAD.
	cmd := exec.Command("git", "-C", repoDir, "fetch", "origin")
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendLine("❌ 启动更新失败: " + err.Error())
		sendEvent("done", `{"success":false}`)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sendLine("❌ 启动更新失败: " + err.Error())
		sendEvent("done", `{"success":false}`)
		return
	}

	if err := cmd.Start(); err != nil {
		sendLine("❌ 启动 git pull 失败: " + err.Error())
		sendEvent("done", `{"success":false}`)
		return
	}

	done := make(chan struct{}, 2)
	stream := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			sendLine(scanner.Text())
		}
		done <- struct{}{}
	}
	go stream(stdout)
	go stream(stderr)
	<-done
	<-done

	err = cmd.Wait()
	sendLine("")

	if err != nil {
		sendLine("❌ fetch 失败: " + err.Error())
		sendEvent("done", `{"success":false,"error":"git fetch failed"}`)
		return
	}

	// Hard reset to origin/HEAD to discard any local divergence
	sendLine("执行: git reset --hard origin/HEAD")
	resetCmd := exec.Command("git", "-C", repoDir, "reset", "--hard", "origin/HEAD")
	resetCmd.Env = os.Environ()
	if out, resetErr := resetCmd.CombinedOutput(); resetErr != nil {
		sendLine("❌ reset 失败: " + string(out))
		sendEvent("done", `{"success":false,"error":"git reset failed"}`)
		return
	} else {
		sendLine(strings.TrimSpace(string(out)))
	}
	sendLine("")

	newCommit := ""
	if out, err2 := exec.Command("git", "-C", repoDir, "rev-parse", "--short", "HEAD").Output(); err2 == nil {
		newCommit = strings.TrimSpace(string(out))
	}

	if newCommit != "" && newCommit != oldCommit {
		sendLine("✅ 更新成功！" + oldCommit + " → " + newCommit)
	} else {
		sendLine("✅ 已是最新版本: " + newCommit)
	}

	b, _ := json.Marshal(map[string]string{
		"old_commit": oldCommit,
		"new_commit": newCommit,
	})
	sendEvent("done", string(b))
}

// checkPasswordQuery checks password from query param (for GET/SSE endpoints)
func (h *AutoDevHandler) checkPasswordQuery(c *gin.Context) bool {
	password := c.Query("password")
	if h.adminPassword == "" || password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return false
	}
	return true
}

// ─── SSH Key Management ────────────────────────────────────────────────────

// ensureSSHKey generates an ed25519 SSH keypair (if not yet created) inside
// the persistent data volume at <dataDir>/ssh/ and returns the public key.
// The private key is chowned to the autodev user so it can be read during tasks.
func (h *AutoDevHandler) ensureSSHKey() (pubKey string, err error) {
	sshDir := filepath.Join(h.dataDir, "ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")
	pubKeyPath := keyPath + ".pub"
	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	if err = os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("创建 SSH 目录失败: %v", err)
	}

	if _, statErr := os.Stat(keyPath); os.IsNotExist(statErr) {
		// Generate ed25519 keypair
		out, cmdErr := exec.Command(
			"ssh-keygen", "-t", "ed25519",
			"-f", keyPath,
			"-N", "",
			"-C", "autodev@devtools",
		).CombinedOutput()
		if cmdErr != nil {
			return "", fmt.Errorf("生成 SSH 密钥失败: %v\n%s", cmdErr, out)
		}
		_ = os.Chmod(keyPath, 0600)
		_ = os.Chmod(pubKeyPath, 0644)

		// Pre-populate GitHub host keys so git doesn't prompt
		if scanOut, scanErr := exec.Command("ssh-keyscan", "-H", "github.com").Output(); scanErr == nil && len(scanOut) > 0 {
			_ = os.WriteFile(knownHostsPath, scanOut, 0644)
		}
	}

	// Always ensure the autodev user (uid 1001) owns the key files so that
	// tasks running as that user can actually read the private key.
	_ = os.Chown(sshDir, autodevUID, autodevGID)
	_ = os.Chown(keyPath, autodevUID, autodevGID)
	_ = os.Chown(pubKeyPath, autodevUID, autodevGID)
	if _, statErr := os.Stat(knownHostsPath); statErr == nil {
		_ = os.Chown(knownHostsPath, autodevUID, autodevGID)
	}

	raw, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("读取公钥失败: %v", err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// gitSSHCommand returns a GIT_SSH_COMMAND value that forces git to use the
// persistent autodev SSH key. Returns "" if the key is not available yet
// (e.g. first startup before any call to GetSSHKey).
func (h *AutoDevHandler) gitSSHCommand() string {
	sshDir := filepath.Join(h.dataDir, "ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")
	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	if _, err := os.Stat(keyPath); err != nil {
		return ""
	}
	base := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=accept-new", keyPath)
	if _, err := os.Stat(knownHostsPath); err == nil {
		base += fmt.Sprintf(" -o UserKnownHostsFile=%s", knownHostsPath)
	}
	return base
}

// GetSSHKey handles GET /api/autodev/sshkey — returns (and generates if needed)
// the public SSH key that should be added to GitHub as a deploy key.
func (h *AutoDevHandler) GetSSHKey(c *gin.Context) {
	if !h.checkPasswordQuery(c) {
		return
	}
	pubKey, err := h.ensureSSHKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	keyPath := filepath.Join(h.dataDir, "ssh", "id_ed25519")
	c.JSON(http.StatusOK, gin.H{
		"public_key": pubKey,
		"key_type":   "ed25519",
		"key_path":   keyPath,
		"comment":    "autodev@devtools",
		"github_tip": "将上面的 public_key 添加到 GitHub → Settings → SSH and GPG keys",
	})
}

// RegenerateSSHKey handles POST /api/autodev/sshkey/regenerate — deletes the
// existing keypair and generates a new one.
func (h *AutoDevHandler) RegenerateSSHKey(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	sshDir := filepath.Join(h.dataDir, "ssh")
	keyPath := filepath.Join(sshDir, "id_ed25519")

	// Remove old keys
	_ = os.Remove(keyPath)
	_ = os.Remove(keyPath + ".pub")
	_ = os.Remove(filepath.Join(sshDir, "known_hosts"))

	pubKey, err := h.ensureSSHKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"public_key":  pubKey,
		"key_type":    "ed25519",
		"regenerated": true,
		"github_tip":  "新密钥已生成，请将 public_key 重新添加到 GitHub → Settings → SSH and GPG keys",
	})
}
