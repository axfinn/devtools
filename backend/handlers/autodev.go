package handlers

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

// autodevUID/GID is the non-root user created in the Dockerfile for running
// autodev tasks. Claude Code refuses --dangerously-skip-permissions as root.
const autodevUID = 1001
const autodevGID = 1001
const autodevHome = "/home/autodev"

// AutoDevHandler handles autodev task operations
type AutoDevHandler struct {
	db            *models.DB
	adminPassword string
	autodevPath   string
	stopScriptPath string
	dataDir       string
	mu            sync.RWMutex
	processes     map[string]*exec.Cmd
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

// Ask handles POST /api/autodev/ask — submit a quick Q&A request using driver.py ask
func (h *AutoDevHandler) Ask(c *gin.Context) {
	var req struct {
		Description string `json:"description" binding:"required"`
		Password    string `json:"password" binding:"required"`
		WorkDir     string `json:"work_dir"` // optional: specify working directory for context
		Background  bool   `json:"background"` // run in background
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// WorkDir is required for ask
	if req.WorkDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ask 模式需要指定 work_dir"})
		return
	}

	// Verify work_dir exists
	if _, err := os.Stat(req.WorkDir); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作目录不存在: " + req.WorkDir})
		return
	}

	// Create task record
	task, err := h.db.CreateAutoDevTask(models.TaskTypeAsk, req.Description, "{}", req.WorkDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// Run ask task (synchronously or in background)
	if req.Background {
		go h.runAskTask(task.ID, req.Description, req.WorkDir, true)
		c.JSON(http.StatusOK, gin.H{
			"id":          task.ID,
			"description": task.Description,
			"status":      "running",
			"work_dir":    task.WorkDir,
			"message":     "task running in background",
		})
	} else {
		go h.runAskTask(task.ID, req.Description, req.WorkDir, false)
		c.JSON(http.StatusOK, gin.H{
			"id":          task.ID,
			"description": task.Description,
			"status":      task.Status,
			"work_dir":    task.WorkDir,
		})
	}
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
		Background  bool   `json:"background"`                  // run in background
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

	// Create task record
	task, err := h.db.CreateAutoDevTask(models.TaskTypeExtend, req.Description, "{}", req.WorkDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// Run extend task
	if req.Background {
		go h.runExtendTask(task.ID, req.Description, req.WorkDir, true)
		c.JSON(http.StatusOK, gin.H{
			"id":          task.ID,
			"description": task.Description,
			"status":      "running",
			"work_dir":    task.WorkDir,
			"message":     "extend task running in background",
		})
	} else {
		go h.runExtendTask(task.ID, req.Description, req.WorkDir, false)
		c.JSON(http.StatusOK, gin.H{
			"id":          task.ID,
			"description": task.Description,
			"status":      task.Status,
			"work_dir":    task.WorkDir,
		})
	}
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

// Submit handles POST /api/autodev/tasks — start a new task or resume from breakpoint
func (h *AutoDevHandler) Submit(c *gin.Context) {
	var req struct {
		Type        string `json:"type"` // develop, ask, export (default: develop)
		Description string `json:"description" binding:"required"`
		Password    string `json:"password" binding:"required"`
		Publish     bool   `json:"publish"`
		Build       bool   `json:"build"`
		Push        bool   `json:"push"`
		// Resume support: if set, --from <phase> is passed and WorkDir is used
		ResumeFrom  int    `json:"resume_from"` // phase number to resume from (1-based, 0 = new task)
		WorkDir     string `json:"work_dir"`    // existing work dir to resume
		// Export options
		ExportFormat string `json:"export_format"` // zip, tar (default: zip)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务目录失败"})
			return
		}
		// Allow non-root autodev user to write
		os.Chmod(taskDir, 0777)
	}

	opts := models.AutoDevOptions{Publish: req.Publish, Build: req.Build, Push: req.Push}
	task, err := h.db.CreateAutoDevTask(taskType, req.Description, models.MarshalAutoDevOptions(opts), taskDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	// Execute task based on type
	switch taskType {
	case models.TaskTypeAsk:
		go h.runAskTask(task.ID, req.Description, taskDir, false)
	case models.TaskTypeExport:
		go h.runExportTask(task.ID, req.Description, taskDir, req.ExportFormat)
	default:
		resumeFrom := 0
		if isResume {
			resumeFrom = req.ResumeFrom
		}
		go h.runTask(task.ID, req.Description, taskDir, req.Publish, req.Build, req.Push, resumeFrom)
	}

	c.JSON(http.StatusOK, task)
}

// List handles GET /api/autodev/tasks?password=xxx
func (h *AutoDevHandler) List(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	tasks, err := h.db.ListAutoDevTasks()
	if err != nil {
		log.Printf("[AutoDev] ListAutoDevTasks error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败: " + err.Error()})
		return
	}
	if tasks == nil {
		tasks = []*models.AutoDevTask{}
	}
	// enrich each task with state.json info
	result := make([]any, 0, len(tasks))
	for _, t := range tasks {
		m := taskToMap(t)
		if state := readAutoDevState(t.WorkDir); state != nil {
			m["autodev_state"] = state
		}
		result = append(result, m)
	}
	c.JSON(http.StatusOK, gin.H{"tasks": result})
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

	absPath := filepath.Join(task.WorkDir, filePath)
	if !strings.HasPrefix(filepath.Clean(absPath), filepath.Clean(task.WorkDir)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "路径不合法"})
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(content), "path": filePath})
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

	logDir := filepath.Join(task.WorkDir, ".autodev", "logs")
	// try phase.log first, then fallback to driver.log
	candidates := []string{
		filepath.Join(logDir, phase+".log"),
		filepath.Join(logDir, "driver.log"),
		filepath.Join(logDir, "session.log"),
		filepath.Join(task.WorkDir, "autodev.log"),
	}
	for _, lp := range candidates {
		if content, err := os.ReadFile(lp); err == nil {
			// list available log files
			logFiles, _ := filepath.Glob(filepath.Join(logDir, "*.log"))
			var phases []string
			for _, lf := range logFiles {
				phases = append(phases, strings.TrimSuffix(filepath.Base(lf), ".log"))
			}
			c.JSON(http.StatusOK, gin.H{
				"logs":            string(content),
				"available_phases": phases,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"logs": "", "available_phases": []string{}})
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

// StopTask handles POST /api/autodev/tasks/:id/stop
// Uses autodev-stop script to gracefully stop (writes STOP file + SIGTERM)
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

	// Try graceful stop via autodev-stop script
	if _, err := os.Stat(h.stopScriptPath); err == nil {
		cmd := exec.Command("python3", h.stopScriptPath, "--path", task.WorkDir)
		cmd.Run()
	} else {
		// Fallback: write STOP file directly
		stopFile := filepath.Join(task.WorkDir, ".autodev", "STOP")
		os.MkdirAll(filepath.Dir(stopFile), 0755)
		os.WriteFile(stopFile, []byte(time.Now().Format(time.RFC3339)), 0644)
		// also kill the process
		h.mu.Lock()
		if cmd, ok := h.processes[id]; ok && cmd.Process != nil {
			cmd.Process.Kill()
		}
		h.mu.Unlock()
	}

	// Read resume suggestion from state
	state := readAutoDevState(task.WorkDir)
	resumeFrom := 0
	if state != nil {
		if last, ok := state["last_completed"]; ok {
			switch v := last.(type) {
			case float64:
				resumeFrom = int(v) + 2 // next phase (1-indexed)
			}
		}
	}

	h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
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
func (h *AutoDevHandler) runTask(id, description, workDir string, publish, build, push bool, resumeFrom int) {
	args := []string{description, "--path", workDir}
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

	// write stdout+stderr to driver.log
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0777)
	// chmod so non-root autodev user can also write here
	os.Chmod(workDir, 0777)
	os.Chmod(filepath.Join(workDir, ".autodev"), 0777)
	os.Chmod(logDir, 0777)

	// Pre-create driver.log and chmod 0666 so the non-root autodev user can open it for append.
	// The Python script opens this file directly, not through stdout/stderr redirection.
	logFilePath := filepath.Join(logDir, "driver.log")
	if f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		f.Close()
	}
	os.Chmod(logFilePath, 0666) // explicitly bypass umask so autodev user (uid 1001) can write

	// Run as non-root user (uid 1001) so Claude Code allows --dangerously-skip-permissions.
	// Replace HOME so Claude Code finds the non-root user's config.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: autodevUID, Gid: autodevGID},
	}
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "UV_CACHE_DIR=") {
			env = append(env, e)
		}
	}
	env = append(env, "HOME="+autodevHome)
	env = append(env, "UV_CACHE_DIR=/tmp/uv-cache-autodev")
	// Inject SSH key for git operations (e.g. cloning private GitHub repos)
	if gitSSHCmd := h.gitSSHCommand(); gitSSHCmd != "" {
		// Filter out any existing GIT_SSH_COMMAND
		filtered := env[:0]
		for _, e := range env {
			if !strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
				filtered = append(filtered, e)
			}
		}
		env = append(filtered, "GIT_SSH_COMMAND="+gitSSHCmd)
	}
	cmd.Env = env

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

	// check autodev state for actual completion status
	if state := readAutoDevState(workDir); state != nil {
		if s, ok := state["status"].(string); ok && s == "finished" {
			status = "completed"
			exitCode = 0
		}
	}

	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	h.mu.Lock()
	delete(h.processes, id)
	h.mu.Unlock()
}

// runAskTask executes claude ask in a background goroutine
func (h *AutoDevHandler) runAskTask(id, description, workDir string, background bool) {
	// Create log directory
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0777)
	os.Chmod(workDir, 0777)
	os.Chmod(filepath.Join(workDir, ".autodev"), 0777)
	os.Chmod(logDir, 0777)

	// Pre-create ask.log
	logFilePath := filepath.Join(logDir, "ask.log")
	if f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		f.Close()
	}
	os.Chmod(logFilePath, 0666)

	// Ensure process directory exists for qa.md
	os.MkdirAll(filepath.Join(workDir, "process"), 0777)

	// Run driver.py ask command
	cmd := exec.Command("python3", h.autodevPath, "ask", description, "--path", workDir)
	cmd.Dir = workDir

	// Set up environment for non-root user
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: autodevUID, Gid: autodevGID},
	}
	env := h.buildEnv()
	cmd.Env = env

	// If background mode, start the process and return immediately
	if background {
		h.mu.Lock()
		h.processes[id] = cmd
		h.mu.Unlock()

		if err := cmd.Start(); err != nil {
			h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
			return
		}
		h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)
		return
	}

	// Synchronous mode: wait for completion
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Update task status
	exitCode := 0
	status := "completed"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	// Find the qa.md file (in process/qa.md)
	qaFilePath := filepath.Join(workDir, "process", "qa.md")

	// Store result file path
	h.db.UpdateAutoDevTaskResult(id, qaFilePath)
	h.db.UpdateAutoDevTaskStatus(id, status, exitCode, 0)

	// Append logs to ask.log
	logContent := fmt.Sprintf("[%s] Ask completed with status: %s\nOutput: %s\n", time.Now().Format(time.RFC3339), status, outputStr)
	os.WriteFile(logFilePath, []byte(logContent), 0666)
}

// runExtendTask executes autodev extend in a background goroutine
func (h *AutoDevHandler) runExtendTask(id, description, workDir string, background bool) {
	// Create log directory
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0777)
	os.Chmod(workDir, 0777)
	os.Chmod(filepath.Join(workDir, ".autodev"), 0777)
	os.Chmod(logDir, 0777)

	// Pre-create extend.log
	logFilePath := filepath.Join(logDir, "extend.log")
	if f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		f.Close()
	}
	os.Chmod(logFilePath, 0666)

	// Run driver.py extend command
	cmd := exec.Command("python3", h.autodevPath, "extend", description, "--path", workDir)
	cmd.Dir = workDir

	// Set up environment for non-root user
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: autodevUID, Gid: autodevGID},
	}
	env := h.buildEnv()
	cmd.Env = env

	// If background mode, start the process and return immediately
	if background {
		h.mu.Lock()
		h.processes[id] = cmd
		h.mu.Unlock()

		if err := cmd.Start(); err != nil {
			h.db.UpdateAutoDevTaskStatus(id, "failed", -1, 0)
			return
		}
		h.db.UpdateAutoDevTaskStatus(id, "running", 0, cmd.Process.Pid)
		return
	}

	// Synchronous mode: wait for completion
	err := cmd.Wait()
	exitCode := 0
	status := "completed"
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = "failed"
	}

	// Check autodev state for actual completion status
	if state := readAutoDevState(workDir); state != nil {
		if s, ok := state["status"].(string); ok && s == "finished" {
			status = "completed"
			exitCode = 0
		}
	}

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
	env = append(env, "HOME="+autodevHome)
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

	// Create log directory
	logDir := filepath.Join(workDir, ".autodev", "logs")
	os.MkdirAll(logDir, 0777)
	os.Chmod(workDir, 0777)
	os.Chmod(filepath.Join(workDir, ".autodev"), 0777)
	os.Chmod(logDir, 0777)

	logFilePath := filepath.Join(logDir, "export.log")
	if f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		f.Close()
	}
	os.Chmod(logFilePath, 0666)

	// Set up environment
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: autodevUID, Gid: autodevGID},
	}
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "UV_CACHE_DIR=") {
			env = append(env, e)
		}
	}
	env = append(env, "HOME="+autodevHome)
	env = append(env, "UV_CACHE_DIR=/tmp/uv-cache-autodev")
	if gitSSHCmd := h.gitSSHCommand(); gitSSHCmd != "" {
		filtered := env[:0]
		for _, e := range env {
			if !strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
				filtered = append(filtered, e)
			}
		}
		env = append(filtered, "GIT_SSH_COMMAND="+gitSSHCmd)
	}
	cmd.Env = env

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

// listTaskFiles recursively walks workDir and returns all relevant files with categories.
//
// Categories:
//   "result"  – RESULT.md (the final deliverable)
//   "code"    – project source files generated by autodev (*.cpp, *.go, *.py, …)
//   "process" – per-phase process docs in process/
//   "log"     – execution logs in .autodev/logs/
//   "docs"    – mkdocs.yml and docs/ directory
//   "state"   – .autodev/state.json
//
// Skipped entirely: .git/, node_modules/, __pycache__/, _site/ (too many files),
// binary/large files (>5MB), and files deeper than 6 levels.
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
		// Skip large files (>5MB) – not useful to display in browser
		if info.Size() > 5*1024*1024 {
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
// Claude CLI 版本管理
// ============================================================

// ClaudeVersion holds claude CLI version info
type ClaudeVersion struct {
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

	result := ClaudeHealth{
		BaseURL:  baseURL,
		Model:    model,
		HasToken: authToken != "",
	}

	if authToken == "" {
		result.Error = "ANTHROPIC_AUTH_TOKEN 未配置"
		c.JSON(http.StatusOK, result)
		return
	}

	// Build request body
	reqBody := map[string]any{
		"model":      model,
		"max_tokens": 16,
		"messages": []map[string]any{
			{"role": "user", "content": "reply with: pong"},
		},
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
	// some compatible endpoints use Authorization Bearer
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

	// Extract text from response
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

// GetClaudeVersion handles GET /api/autodev/claude/version?password=xxx
func (h *AutoDevHandler) GetClaudeVersion(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}

	info := ClaudeVersion{}

	// get claude version
	if out, err := exec.Command("claude", "--version").Output(); err == nil {
		info.Version = strings.TrimSpace(string(out))
		info.Available = true
	} else {
		info.Available = false
	}

	// get claude path
	if out, err := exec.Command("which", "claude").Output(); err == nil {
		info.Path = strings.TrimSpace(string(out))
	}

	// npm version
	if out, err := exec.Command("npm", "--version").Output(); err == nil {
		info.NpmVersion = strings.TrimSpace(string(out))
	}

	// node version
	if out, err := exec.Command("node", "--version").Output(); err == nil {
		info.NodeVersion = strings.TrimSpace(string(out))
	}

	c.JSON(http.StatusOK, info)
}

// UpdateClaude handles GET /api/autodev/claude/update/stream?password=xxx
// Streams npm install output via SSE
func (h *AutoDevHandler) UpdateClaude(c *gin.Context) {
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

	sendLine("🚀 开始更新 Claude Code CLI...")
	sendLine("执行: npm install -g @anthropic-ai/claude-code@latest")

	// record version before update
	oldVersion := ""
	if out, err := exec.Command("claude", "--version").Output(); err == nil {
		oldVersion = strings.TrimSpace(string(out))
		sendLine(fmt.Sprintf("当前版本: %s", oldVersion))
	} else {
		sendLine("⚠️  当前未安装 Claude Code")
	}

	sendLine("")

	// run npm install
	cmd := exec.Command("npm", "install", "-g", "@anthropic-ai/claude-code@latest", "--unsafe-perm")
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

	// stream stdout and stderr concurrently
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

	// get new version
	newVersion := ""
	if out, err2 := exec.Command("claude", "--version").Output(); err2 == nil {
		newVersion = strings.TrimSpace(string(out))
	}

	if newVersion != "" && newVersion != oldVersion {
		sendLine(fmt.Sprintf("✅ 更新成功！%s → %s", oldVersion, newVersion))
	} else if newVersion != "" {
		sendLine(fmt.Sprintf("✅ 已是最新版本: %s", newVersion))
	} else {
		sendLine("✅ 更新完成")
	}

	b, _ := json.Marshal(map[string]string{
		"old_version": oldVersion,
		"new_version": newVersion,
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
		"public_key": pubKey,
		"key_type":   "ed25519",
		"regenerated": true,
		"github_tip": "新密钥已生成，请将 public_key 重新添加到 GitHub → Settings → SSH and GPG keys",
	})
}
