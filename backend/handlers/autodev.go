package handlers

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/models"

	"github.com/gin-gonic/gin"
)

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
		Description string `json:"description" binding:"required"`
		Password    string `json:"password" binding:"required"`
		Publish     bool   `json:"publish"`
		Build       bool   `json:"build"`
		Push        bool   `json:"push"`
		// Resume support: if set, --from <phase> is passed and WorkDir is used
		ResumeFrom  int    `json:"resume_from"` // phase number to resume from (1-based, 0 = new task)
		WorkDir     string `json:"work_dir"`    // existing work dir to resume
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if h.adminPassword == "" || req.Password != h.adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	if _, err := os.Stat(h.autodevPath); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "autodev 未安装，请检查 Docker 配置"})
		return
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
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务目录失败"})
			return
		}
	}

	opts := models.AutoDevOptions{Publish: req.Publish, Build: req.Build, Push: req.Push}
	task, err := h.db.CreateAutoDevTask(req.Description, models.MarshalAutoDevOptions(opts), taskDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	resumeFrom := 0
	if isResume {
		resumeFrom = req.ResumeFrom
	}
	go h.runTask(task.ID, req.Description, taskDir, req.Publish, req.Build, req.Push, resumeFrom)

	c.JSON(http.StatusOK, task)
}

// List handles GET /api/autodev/tasks?password=xxx
func (h *AutoDevHandler) List(c *gin.Context) {
	if !h.checkPassword(c) {
		return
	}
	tasks, err := h.db.ListAutoDevTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
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
	c.JSON(http.StatusOK, gin.H{"files": files})
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
	os.MkdirAll(logDir, 0755)
	logFile, _ := os.Create(filepath.Join(logDir, "driver.log"))
	if logFile != nil {
		defer logFile.Close()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	// pass Claude env vars from system environment
	cmd.Env = os.Environ()

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

// listTaskFiles returns key files in a task directory
func listTaskFiles(workDir string) []map[string]any {
	var files []map[string]any

	patterns := []string{
		filepath.Join(workDir, "RESULT.md"),
		filepath.Join(workDir, "process", "*.md"),
		filepath.Join(workDir, ".autodev", "logs", "*.log"),
		filepath.Join(workDir, ".autodev", "state.json"),
		filepath.Join(workDir, "mkdocs.yml"),
	}

	seen := map[string]bool{}
	for _, pat := range patterns {
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			rel, _ := filepath.Rel(workDir, m)
			if seen[rel] {
				continue
			}
			seen[rel] = true
			info, err := os.Stat(m)
			if err != nil {
				continue
			}
			files = append(files, map[string]any{
				"path":  rel,
				"name":  filepath.Base(rel),
				"size":  info.Size(),
				"mtime": info.ModTime().Format(time.RFC3339),
			})
		}
	}
	return files
}

// taskToMap converts a task to a map for JSON serialization with extra fields
func taskToMap(t *models.AutoDevTask) map[string]any {
	m := map[string]any{
		"id":           t.ID,
		"description":  t.Description,
		"options":      t.Options,
		"status":       t.Status,
		"exit_code":    t.ExitCode,
		"work_dir":     t.WorkDir,
		"pid":          t.PID,
		"created_at":   t.CreatedAt,
		"started_at":   t.StartedAt,
		"completed_at": t.CompletedAt,
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
