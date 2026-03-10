package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

const imageUnderstandingMaxSize = 10 * 1024 * 1024 // 10MB

// 图像理解任务状态
type imageTask struct {
	ID        string
	Status    string // pending, processing, completed, failed
	Tool      string
	Text      string
	Result    []byte
	Args      map[string]interface{}
	Error     string
	CreatedAt time.Time
}

// imageTaskStore 任务存储（内存）
var imageTaskStore = make(map[string]*imageTask)
var imageTaskMu sync.RWMutex

func newImageTask() *imageTask {
	return &imageTask{
		ID:        generateTaskID(),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
}

func generateTaskID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

type ImageUnderstandingHandler struct {
	cfg config.MiniMaxMCPConfig
}

type imageUnderstandingRequest struct {
	Image  string                 `json:"image" binding:"required"`
	Prompt string                 `json:"prompt"`
	Tool   string                 `json:"tool"`
	Args   map[string]interface{} `json:"args"`
}

type imageUnderstandingUploadRequest struct {
	Prompt string `form:"prompt"`
	Tool   string `form:"tool"`
	Args   string `form:"args"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func NewImageUnderstandingHandler(cfg *config.Config) *ImageUnderstandingHandler {
	return &ImageUnderstandingHandler{cfg: cfg.MiniMaxMCP}
}

func (h *ImageUnderstandingHandler) ListTools(c *gin.Context) {
	if strings.TrimSpace(h.cfg.APIKey) == "" {
		c.JSON(503, gin.H{"error": "未配置 minimax_mcp.api_key"})
		return
	}
	proc, err := startMCPProcess(h.cfg)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer proc.Close()

	// Detach from request context to avoid client/proxy cancellation mid-MCP.
	ctx, cancel := context.WithTimeout(context.Background(), h.cfg.Timeout())
	defer cancel()

	if err := proc.Initialize(ctx); err != nil {
		msg := enrichMCPError(err, proc)
		logMCPError("initialize", msg)
		c.JSON(502, gin.H{"error": msg})
		return
	}
	tools, err := proc.ListTools(ctx)
	if err != nil {
		msg := enrichMCPError(err, proc)
		logMCPError("tools/list", msg)
		c.JSON(502, gin.H{"error": msg})
		return
	}
	c.JSON(200, gin.H{"tools": tools})
}

func (h *ImageUnderstandingHandler) Describe(c *gin.Context) {
	var req imageUnderstandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "缺少 image 字段"})
		return
	}
	if strings.TrimSpace(h.cfg.APIKey) == "" {
		c.JSON(503, gin.H{"error": "未配置 minimax_mcp.api_key"})
		return
	}

	if err := validateImageSize(req.Image); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	proc, err := startMCPProcess(h.cfg)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer proc.Close()

	// Detach from request context to avoid client/proxy cancellation mid-MCP.
	ctx, cancel := context.WithTimeout(context.Background(), h.cfg.Timeout())
	defer cancel()

	if err := proc.Initialize(ctx); err != nil {
		msg := enrichMCPError(err, proc)
		logMCPError("initialize", msg)
		c.JSON(502, gin.H{"error": msg})
		return
	}
	tools, err := proc.ListTools(ctx)
	if err != nil {
		msg := enrichMCPError(err, proc)
		logMCPError("tools/list", msg)
		c.JSON(502, gin.H{"error": msg})
		return
	}

	tool, ok := resolveTool(req.Tool, tools)
	if !ok {
		c.JSON(400, gin.H{"error": "未找到可用的图像理解工具"})
		return
	}

	args, cleanup, err := prepareToolArgs(req, tool)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if cleanup != nil {
		defer cleanup()
	}
	result, err := proc.CallTool(ctx, tool.Name, args)
	if err != nil {
		msg := enrichMCPError(err, proc)
		logMCPArgs(tool.Name, args)
		logMCPError("tools/call", msg)
		c.JSON(502, gin.H{"error": msg})
		return
	}

	text := extractToolText(result)
	c.JSON(200, gin.H{
		"tool":       tool.Name,
		"text":       text,
		"result":     result,
		"args_preview": sanitizeArgs(args),
	})
}

func (h *ImageUnderstandingHandler) DescribeFromUpload(c *gin.Context) {
	if strings.TrimSpace(h.cfg.APIKey) == "" {
		c.JSON(503, gin.H{"error": "未配置 minimax_mcp.api_key"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "缺少 file"})
		return
	}
	if file.Size > imageUnderstandingMaxSize {
		c.JSON(400, gin.H{"error": "图片大小不能超过 10MB"})
		return
	}

	var req imageUnderstandingUploadRequest
	_ = c.ShouldBind(&req)

	args := map[string]interface{}{}
	if strings.TrimSpace(req.Args) != "" {
		if err := json.Unmarshal([]byte(req.Args), &args); err != nil {
			c.JSON(400, gin.H{"error": "args JSON 解析失败"})
			return
		}
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".png"
	}
	tempFile, err := os.CreateTemp("", "minimax-upload-*"+ext)
	if err != nil {
		c.JSON(500, gin.H{"error": "创建临时文件失败"})
		return
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()

	src, err := file.Open()
	if err != nil {
		_ = os.Remove(tempPath)
		c.JSON(500, gin.H{"error": "读取上传文件失败"})
		return
	}
	defer src.Close()

	dst, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		_ = os.Remove(tempPath)
		c.JSON(500, gin.H{"error": "写入临时文件失败"})
		return
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(tempPath)
		c.JSON(500, gin.H{"error": "保存上传文件失败"})
		return
	}
	_ = dst.Close()

	scheduleDelete(tempPath, 10*time.Minute)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.cfg.Timeout())
	defer cancel()

	toolName, text, result, payload, err := h.ExecuteWithPath(ctx, req.Tool, req.Prompt, args, tempPath)
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"tool":       toolName,
		"text":       text,
		"result":     result,
		"args_preview": sanitizeArgs(payload),
	})
}

func validateImageSize(image string) error {
	imageData := image
	if strings.Contains(imageData, ",") {
		imageData = strings.Split(imageData, ",")[1]
	}
	decodedLen := base64.StdEncoding.DecodedLen(len(imageData))
	if decodedLen > imageUnderstandingMaxSize {
		return fmt.Errorf("图片大小不能超过 10MB")
	}
	return nil
}

func resolveTool(requested string, tools []mcpTool) (mcpTool, bool) {
	if requested != "" {
		for _, tool := range tools {
			if tool.Name == requested {
				return tool, true
			}
		}
		return mcpTool{}, false
	}
	for _, tool := range tools {
		if tool.Name == "understand_image" {
			return tool, true
		}
	}
	for _, keyword := range []string{"image", "vision", "understanding", "describe", "caption"} {
		for _, tool := range tools {
			if strings.Contains(strings.ToLower(tool.Name), keyword) {
				return tool, true
			}
		}
	}
	if len(tools) > 0 {
		return tools[0], true
	}
	return mcpTool{}, false
}

func prepareToolArgs(req imageUnderstandingRequest, tool mcpTool) (map[string]interface{}, func(), error) {
	args := map[string]interface{}{}
	for key, value := range req.Args {
		args[key] = value
	}
	schemaProps := extractSchemaProps(tool.InputSchema)
	cleanup := func() {}

	if !hasAnyKey(args, "image", "images", "image_url", "imageUrl", "image_path", "imagePath", "path", "file", "image_file", "imageFile", "image_paths", "imagePaths", "paths", "files", "image_files", "input_images", "image_source") {
		needsTemp := isDataImage(req.Image) || isLikelyBase64(req.Image)
		imagePath := ""
		if needsTemp {
			var err error
			imagePath, err = writeTempImage(req.Image)
			if err != nil {
				return nil, nil, err
			}
			cleanup = func() { _ = os.Remove(imagePath) }
		}

		pathKeys := []string{"image_path", "imagePath", "path", "file", "image_file", "imageFile"}
		pathsKeys := []string{"image_paths", "imagePaths", "paths", "files", "image_files", "input_images"}
		sourceKey := "image_source"
		if schemaProps[sourceKey] {
			if imagePath == "" {
				var err error
				imagePath, err = writeTempImage(req.Image)
				if err != nil {
					return nil, nil, err
				}
				cleanup = func() { _ = os.Remove(imagePath) }
			}
			args[sourceKey] = strings.TrimPrefix(imagePath, "@")
		} else
		if hasAnySchemaKey(schemaProps, pathKeys...) || hasAnySchemaKey(schemaProps, pathsKeys...) {
			if imagePath == "" {
				var err error
				imagePath, err = writeTempImage(req.Image)
				if err != nil {
					return nil, nil, err
				}
				cleanup = func() { _ = os.Remove(imagePath) }
			}
			if hasAnySchemaKey(schemaProps, pathsKeys...) {
				key := firstSchemaKey(schemaProps, pathsKeys...)
				args[key] = []string{imagePath}
			} else {
				key := firstSchemaKey(schemaProps, pathKeys...)
				args[key] = imagePath
			}
		} else if imagePath != "" {
			switch {
			case schemaProps["images"]:
				args["images"] = []string{"file://" + imagePath}
			case schemaProps["image_url"]:
				args["image_url"] = "file://" + imagePath
			case schemaProps["imageUrl"]:
				args["imageUrl"] = "file://" + imagePath
			default:
				args["image"] = imagePath
			}
		} else {
			switch {
			case schemaProps["images"]:
				args["images"] = []string{req.Image}
			case schemaProps["image_url"]:
				args["image_url"] = req.Image
			case schemaProps["imageUrl"]:
				args["imageUrl"] = req.Image
			default:
				args["image"] = req.Image
			}
		}
	}

	if req.Prompt != "" && !hasAnyKey(args, "prompt", "text", "instruction", "query") {
		switch {
		case schemaProps["prompt"]:
			args["prompt"] = req.Prompt
		case schemaProps["text"]:
			args["text"] = req.Prompt
		case schemaProps["instruction"]:
			args["instruction"] = req.Prompt
		case schemaProps["query"]:
			args["query"] = req.Prompt
		default:
			args["prompt"] = req.Prompt
		}
	} else if schemaProps["prompt"] && !hasAnyKey(args, "prompt") {
		args["prompt"] = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	return args, cleanup, nil
}

func prepareToolArgsWithPath(prompt string, tool mcpTool, overrides map[string]interface{}, path string) map[string]interface{} {
	args := map[string]interface{}{}
	for key, value := range overrides {
		args[key] = value
	}
	schemaProps := extractSchemaProps(tool.InputSchema)
	if !hasAnyKey(args, "image", "images", "image_url", "imageUrl", "image_path", "imagePath", "path", "file", "image_file", "imageFile", "image_paths", "imagePaths", "paths", "files", "image_files", "input_images", "image_source") {
		if schemaProps["image_source"] {
			args["image_source"] = strings.TrimPrefix(path, "@")
		} else if hasAnySchemaKey(schemaProps, "image_path", "imagePath", "path", "file", "image_file", "imageFile") {
			key := firstSchemaKey(schemaProps, "image_path", "imagePath", "path", "file", "image_file", "imageFile")
			args[key] = path
		} else if hasAnySchemaKey(schemaProps, "image_paths", "imagePaths", "paths", "files", "image_files", "input_images") {
			key := firstSchemaKey(schemaProps, "image_paths", "imagePaths", "paths", "files", "image_files", "input_images")
			args[key] = []string{path}
		} else if schemaProps["image_url"] {
			args["image_url"] = "file://" + path
		} else if schemaProps["imageUrl"] {
			args["imageUrl"] = "file://" + path
		} else {
			args["image"] = path
		}
	}

	if prompt != "" && !hasAnyKey(args, "prompt", "text", "instruction", "query") {
		switch {
		case schemaProps["prompt"]:
			args["prompt"] = prompt
		case schemaProps["text"]:
			args["text"] = prompt
		case schemaProps["instruction"]:
			args["instruction"] = prompt
		case schemaProps["query"]:
			args["query"] = prompt
		default:
			args["prompt"] = prompt
		}
	} else if schemaProps["prompt"] && !hasAnyKey(args, "prompt") {
		args["prompt"] = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}
	return args
}

func extractSchemaProps(schema map[string]interface{}) map[string]bool {
	props := map[string]bool{}
	if schema == nil {
		return props
	}
	if raw, ok := schema["properties"].(map[string]interface{}); ok {
		for key := range raw {
			props[key] = true
		}
	}
	return props
}

func hasAnySchemaKey(schemaProps map[string]bool, keys ...string) bool {
	for _, key := range keys {
		if schemaProps[key] {
			return true
		}
	}
	return false
}

func firstSchemaKey(schemaProps map[string]bool, keys ...string) string {
	for _, key := range keys {
		if schemaProps[key] {
			return key
		}
	}
	return keys[0]
}

func writeTempImage(data string) (string, error) {
	raw := data
	if strings.Contains(raw, ",") {
		raw = strings.SplitN(raw, ",", 2)[1]
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("图片解码失败")
	}
	file, err := os.CreateTemp("", "minimax-image-*.png")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败")
	}
	if _, err := file.Write(decoded); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("写入临时文件失败")
	}
	_ = file.Close()
	return file.Name(), nil
}

func isDataImage(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "data:image")
}

func isLikelyBase64(value string) bool {
	raw := strings.TrimSpace(value)
	if raw == "" || strings.Contains(raw, "http://") || strings.Contains(raw, "https://") {
		return false
	}
	if strings.HasPrefix(raw, "data:") {
		return false
	}
	if len(raw) < 32 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(trimBase64(raw))
	return err == nil
}

func trimBase64(raw string) string {
	if strings.Contains(raw, ",") {
		return strings.SplitN(raw, ",", 2)[1]
	}
	return raw
}

func scheduleDelete(path string, delay time.Duration) {
	if path == "" {
		return
	}
	go func() {
		<-time.After(delay)
		_ = os.Remove(path)
	}()
}

func writeMultipartToTemp(filename string, src io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".png"
	}
	tempFile, err := os.CreateTemp("", "minimax-upload-*"+ext)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败")
	}
	path := tempFile.Name()
	if _, err := io.Copy(tempFile, src); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("保存上传文件失败")
	}
	_ = tempFile.Close()
	return path, nil
}

func (h *ImageUnderstandingHandler) ExecuteWithPath(ctx context.Context, toolName, prompt string, args map[string]interface{}, path string) (string, string, map[string]interface{}, map[string]interface{}, error) {
	proc, err := startMCPProcess(h.cfg)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer proc.Close()

	if err := proc.Initialize(ctx); err != nil {
		msg := enrichMCPError(err, proc)
		logMCPError("initialize", msg)
		return "", "", nil, nil, fmt.Errorf(msg)
	}
	tools, err := proc.ListTools(ctx)
	if err != nil {
		msg := enrichMCPError(err, proc)
		logMCPError("tools/list", msg)
		return "", "", nil, nil, fmt.Errorf(msg)
	}

	tool, ok := resolveTool(toolName, tools)
	if !ok {
		return "", "", nil, nil, fmt.Errorf("未找到可用的图像理解工具")
	}
	payload := prepareToolArgsWithPath(prompt, tool, args, path)

	result, err := proc.CallTool(ctx, tool.Name, payload)
	if err != nil {
		msg := enrichMCPError(err, proc)
		logMCPArgs(tool.Name, payload)
		logMCPError("tools/call", msg)
		return tool.Name, "", nil, payload, fmt.Errorf(msg)
	}
	text := extractToolText(result)
	return tool.Name, text, result, payload, nil
}

func hasAnyKey(args map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := args[key]; ok {
			return true
		}
	}
	return false
}

func sanitizeArgs(args map[string]interface{}) map[string]interface{} {
	clean := map[string]interface{}{}
	for key, value := range args {
		if key == "image" || key == "images" || key == "image_url" || key == "imageUrl" {
			clean[key] = "[image]"
			continue
		}
		clean[key] = value
	}
	return clean
}

func extractToolText(result map[string]interface{}) string {
	if result == nil {
		return ""
	}
	if content, ok := result["content"].([]interface{}); ok {
		var parts []string
		for _, item := range content {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	if text, ok := result["text"].(string); ok {
		return text
	}
	if text, ok := result["answer"].(string); ok {
		return text
	}
	return ""
}

type mcpProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *bytes.Buffer
	nextID int
	mode   string
}

func startMCPProcess(cfg config.MiniMaxMCPConfig) (*mcpProcess, error) {
	command := cfg.Command
	args := cfg.Args
	if strings.TrimSpace(command) == "" {
		command = "uvx"
	}
	if len(args) == 0 {
		args = []string{"minimax-coding-plan-mcp", "-y"}
	}
	resolved, err := resolveCommandPath(command)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(resolved, args...)
	cmd.Env = append(os.Environ(), "MINIMAX_API_KEY="+cfg.APIKey)
	if strings.TrimSpace(cfg.APIHost) != "" {
		cmd.Env = append(cmd.Env, "MINIMAX_API_HOST="+cfg.APIHost)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcpProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		stderr: stderr,
		nextID: 1,
		mode:   normalizeTransport(cfg.Transport),
	}, nil
}

func (p *mcpProcess) Close() {
	if p == nil {
		return
	}
	_ = p.stdin.Close()
	waitErrCh := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		waitErrCh <- p.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		if err := <-waitErrCh; err != nil {
			logMCPError("process_exit", err.Error())
		}
	case <-time.After(2 * time.Second):
		if p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
	}
}

func (p *mcpProcess) Initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "devtools",
			"version": "1.0",
		},
	}
	if _, err := p.sendRequest(ctx, "initialize", params); err != nil {
		return err
	}
	_ = p.sendNotification("notifications/initialized", map[string]interface{}{})
	return nil
}

func (p *mcpProcess) ListTools(ctx context.Context) ([]mcpTool, error) {
	resp, err := p.sendRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("工具列表响应格式错误")
	}
	rawTools, ok := result["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("工具列表为空")
	}
	tools := make([]mcpTool, 0, len(rawTools))
	for _, item := range rawTools {
		raw, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tool := mcpTool{
			Name:        toString(raw["name"]),
			Description: toString(raw["description"]),
		}
		if schema, ok := raw["inputSchema"].(map[string]interface{}); ok {
			tool.InputSchema = schema
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func (p *mcpProcess) CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	resp, err := p.sendRequest(ctx, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		if p.stderr != nil {
			logMCPError("tools/call.stderr", strings.TrimSpace(p.stderr.String()))
		}
		return nil, err
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("工具调用返回格式错误")
	}
	if isError, ok := result["isError"].(bool); ok && isError {
		return result, fmt.Errorf("工具调用失败")
	}
	return result, nil
}

func (p *mcpProcess) sendRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	id := p.nextID
	p.nextID++
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	if err := p.writeMessage(payload); err != nil {
		return nil, err
	}
	for {
		raw, err := p.readMessage(ctx)
		if err != nil {
			return nil, err
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(raw, &resp); err != nil {
			logMCPRaw("json_unmarshal_failed", raw)
			return nil, fmt.Errorf("mcp 响应解析失败: %w", err)
		}
		if respID, ok := parseID(resp["id"]); ok && respID == id {
			if errObj, ok := resp["error"].(map[string]interface{}); ok {
				if raw, err := json.Marshal(errObj); err == nil {
					logMCPRaw("rpc_error_"+method, raw)
				}
				return nil, errors.New(toString(errObj["message"]))
			}
			return resp, nil
		}
	}
}

func (p *mcpProcess) sendNotification(method string, params map[string]interface{}) error {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	return p.writeMessage(payload)
}

func (p *mcpProcess) writeMessage(payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if p.mode == "line" {
		_, err = p.stdin.Write(append(body, '\n'))
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := p.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err = p.stdin.Write(body)
	return err
}

func (p *mcpProcess) readMessage(ctx context.Context) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := p.readMessageBlocking()
		ch <- result{data: data, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.data, res.err
	}
}

func (p *mcpProcess) readMessageBlocking() ([]byte, error) {
	if p.mode == "line" {
		line, err := p.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		return []byte(strings.TrimSpace(line)), nil
	}
	contentLength := 0
	for {
		line, err := p.stdout.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &contentLength)
			}
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("响应格式错误")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(p.stdout, body); err != nil {
		return nil, err
	}
	return body, nil
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

func parseID(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}

func resolveCommandPath(command string) (string, error) {
	if strings.Contains(command, "/") {
		if _, err := os.Stat(command); err == nil {
			return command, nil
		}
		return "", fmt.Errorf("mcp command 不存在: %s", command)
	}
	path, err := exec.LookPath(command)
	if err == nil {
		return path, nil
	}
	if command == "uvx" {
		candidates := []string{"/root/.local/bin/uvx", "/usr/local/bin/uvx"}
		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("mcp command 找不到: %s", command)
}

func normalizeTransport(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "header", "headers", "content-length":
		return "header"
	default:
		return "line"
	}
}

func enrichMCPError(err error, proc *mcpProcess) string {
	if proc == nil || proc.stderr == nil {
		return err.Error()
	}
	raw := strings.TrimSpace(proc.stderr.String())
	if raw == "" {
		return err.Error()
	}
	if len(raw) > 800 {
		raw = raw[:800] + "..."
	}
	return fmt.Sprintf("%s | stderr: %s", err.Error(), raw)
}

func logMCPError(stage, message string) {
	fmt.Printf("MCP error (%s): %s\n", stage, message)
}

func logMCPRaw(stage string, raw []byte) {
	if len(raw) == 0 {
		return
	}
	fmt.Printf("MCP raw (%s): %s\n", stage, string(raw))
}

func logMCPArgs(tool string, args map[string]interface{}) {
	if args == nil {
		return
	}
	fmt.Printf("MCP args (%s): %s\n", tool, sanitizeArgs(args))
}

// ===== SSE 图像理解接口 =====

// CreateSseTask 创建任务并返回 task_id
func (h *ImageUnderstandingHandler) CreateSseTask(c *gin.Context) {
	if strings.TrimSpace(h.cfg.APIKey) == "" {
		c.JSON(503, gin.H{"error": "未配置 minimax_mcp.api_key"})
		return
	}

	var req imageUnderstandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "缺少 image 字段"})
		return
	}

	if err := validateImageSize(req.Image); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	imagePath, err := writeTempImage(req.Image)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	// 创建任务
	task := newImageTask()
	task.Tool = req.Tool
	task.Args = req.Args
	task.Status = "processing"

	imageTaskMu.Lock()
	imageTaskStore[task.ID] = task
	imageTaskMu.Unlock()

	// 后台执行
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), h.cfg.Timeout())
		defer cancel()

		toolName, text, result, payload, err := h.ExecuteWithPath(ctx, req.Tool, req.Prompt, req.Args, imagePath)

		imageTaskMu.Lock()
		defer imageTaskMu.Unlock()

		task := imageTaskStore[task.ID]
		if task == nil {
			return
		}

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
		} else {
			task.Status = "completed"
			task.Tool = toolName
			task.Text = text
			task.Result, _ = json.Marshal(result)
		}
		_ = payload // 忽略
	}()

	c.JSON(200, gin.H{
		"task_id": task.ID,
		"status":  "processing",
	})
}

// CreateSseTaskFromFile 从文件创建 SSE 任务
func (h *ImageUnderstandingHandler) CreateSseTaskFromFile(c *gin.Context) {
	if strings.TrimSpace(h.cfg.APIKey) == "" {
		c.JSON(503, gin.H{"error": "未配置 minimax_mcp.api_key"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "缺少 file"})
		return
	}
	if file.Size > imageUnderstandingMaxSize {
		c.JSON(400, gin.H{"error": "图片大小不能超过 10MB"})
		return
	}

	prompt := strings.TrimSpace(c.PostForm("prompt"))
	tool := strings.TrimSpace(c.PostForm("tool"))
	argsText := strings.TrimSpace(c.PostForm("args"))
	args := map[string]interface{}{}
	if argsText != "" {
		if err := json.Unmarshal([]byte(argsText), &args); err != nil {
			c.JSON(400, gin.H{"error": "args JSON 解析失败"})
			return
		}
	}
	if prompt == "" {
		prompt = "请简洁描述图片内容，提取关键对象、场景和文字信息。"
	}

	imagePath, err := saveMultipartToTempFile(file)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	scheduleDelete(imagePath, 10*time.Minute)

	// 创建任务
	task := newImageTask()
	task.Tool = tool
	task.Args = args
	task.Status = "processing"

	imageTaskMu.Lock()
	imageTaskStore[task.ID] = task
	imageTaskMu.Unlock()

	// 后台执行
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), h.cfg.Timeout())
		defer cancel()

		toolName, text, result, payload, err := h.ExecuteWithPath(ctx, tool, prompt, args, imagePath)

		imageTaskMu.Lock()
		defer imageTaskMu.Unlock()

		task := imageTaskStore[task.ID]
		if task == nil {
			return
		}

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
		} else {
			task.Status = "completed"
			task.Tool = toolName
			task.Text = text
			task.Result, _ = json.Marshal(result)
		}
		_ = payload
	}()

	c.JSON(200, gin.H{
		"task_id": task.ID,
		"status":  "processing",
	})
}

// GetSseTask 获取任务状态（用于 SSE 轮询）
func (h *ImageUnderstandingHandler) GetSseTask(c *gin.Context) {
	taskID := c.Param("id")

	imageTaskMu.RLock()
	task, ok := imageTaskStore[taskID]
	imageTaskMu.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	if task.Status == "completed" {
		c.JSON(200, gin.H{
			"task_id": task.ID,
			"status":  task.Status,
			"tool":    task.Tool,
			"text":    task.Text,
			"result":  task.Result,
		})
		return
	}

	if task.Status == "failed" {
		c.JSON(200, gin.H{
			"task_id": task.ID,
			"status":  task.Status,
			"error":   task.Error,
		})
		return
	}

	// processing 或 pending
	c.JSON(200, gin.H{
		"task_id": task.ID,
		"status":  task.Status,
	})
}

// StreamSseTask SSE 事件流
func (h *ImageUnderstandingHandler) StreamSseTask(c *gin.Context) {
	taskID := c.Param("id")

	imageTaskMu.RLock()
	task, ok := imageTaskStore[taskID]
	imageTaskMu.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(500, gin.H{"error": "SSE 不支持"})
		return
	}

	sendEvent := func(event, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	// 立即发送初始状态
	sendEvent("status", `{"task_id":"`+taskID+`","status":"`+task.Status+`"}`)

	// 轮询任务状态
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			imageTaskMu.RLock()
			t := imageTaskStore[taskID]
			imageTaskMu.RUnlock()

			if t == nil {
				sendEvent("error", `{"error":"任务不存在"}`)
				return
			}

			if t.Status == "completed" {
				sendEvent("completed", fmt.Sprintf(`{"task_id":"%s","tool":"%s","text":%s,"result":%s}`,
					t.ID, t.Tool, jsonStr(t.Text), jsonStr(t.Result)))
				return
			}

			if t.Status == "failed" {
				sendEvent("error", fmt.Sprintf(`{"task_id":"%s","error":%s}`, t.ID, jsonStr(t.Error)))
				return
			}

			// 继续等待
			sendEvent("status", `{"task_id":"`+taskID+`","status":"`+t.Status+`"}`)
		}
	}
}

func jsonStr(v interface{}) string {
	if v == nil {
		return `""`
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return `""`
		}
		b, _ := json.Marshal(val)
		return string(b)
	case []byte:
		if len(val) == 0 {
			return `""`
		}
		return string(val)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func saveMultipartToTempFile(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	return writeMultipartToTemp(fileHeader.Filename, file)
}
