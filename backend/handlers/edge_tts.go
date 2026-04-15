package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// EdgeTTSHandler Edge TTS 处理器，代理到 tts-service
type EdgeTTSHandler struct {
	ttsServiceURL string // e.g. http://127.0.0.1:8083
	httpClient    *http.Client
}

// NewEdgeTTSHandler 创建 EdgeTTSHandler
func NewEdgeTTSHandler(ttsServiceURL string) *EdgeTTSHandler {
	return &EdgeTTSHandler{
		ttsServiceURL: ttsServiceURL,
		httpClient:    &http.Client{},
	}
}

// EdgeTTSVoice 音色结构
type EdgeTTSVoice struct {
	ID     string `json:"id"`     // e.g. "zh-CN-XiaoxiaoNeural"
	Name   string `json:"name"`   // e.g. "晓晓"
	Gender string `json:"gender"` // "female" or "male"
	Style  string `json:"style"`  // e.g. "温柔"
}

// EdgeTTSVoicesResponse 音色列表响应
type EdgeTTSVoicesResponse struct {
	Voices []EdgeTTSVoice `json:"voices"`
}

// EdgeTTSTTSRequest TTS 请求
type EdgeTTSTTSRequest struct {
	Text        string `json:"text" binding:"required"`
	Voice       string `json:"voice"`         // 音色 ID，默认 zh-CN-XiaoxiaoNeural
	AudioFormat string `json:"audio_format"`  // mp3 或 wav
}

// EdgeTTSTTSResponse TTS 响应
type EdgeTTSTTSResponse struct {
	URL      string `json:"url"`       // 音频文件 URL
	Filename string `json:"filename"`  // 文件名
}

// ListVoices 获取可用音色列表
// GET /api/edge-tts/voices
func (h *EdgeTTSHandler) ListVoices(c *gin.Context) {
	resp, err := h.httpClient.Get(h.ttsServiceURL + "/voices")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法连接到 TTS 服务: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadGateway, gin.H{"error": "TTS 服务返回错误: " + string(body)})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取响应失败: " + err.Error()})
		return
	}

	// 直接透传响应
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(body))
}

// Synthesize 合成语音
// POST /api/edge-tts/tts
func (h *EdgeTTSHandler) Synthesize(c *gin.Context) {
	var req EdgeTTSTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求: " + err.Error()})
		return
	}

	if req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文本不能为空"})
		return
	}

	// 默认音色
	if req.Voice == "" {
		req.Voice = "zh-CN-XiaoxiaoNeural"
	}

	// 转发到 tts-service
	ttsReq := map[string]string{
		"text":  req.Text,
		"voice": req.Voice,
	}

	reqBody, _ := json.Marshal(ttsReq)
	resp, err := h.httpClient.Post(h.ttsServiceURL+"/tts", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TTS 服务调用失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadGateway, gin.H{"error": "TTS 服务返回错误: " + string(body)})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取响应失败: " + err.Error()})
		return
	}

	// 解析 tts-service 的响应 {"url": "/api/chat/uploads/xxx.mp3"}
	var ttsResp map[string]string
	if err := json.Unmarshal(body, &ttsResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析 TTS 响应失败: " + err.Error()})
		return
	}

	// 返回给前端的 URL
	audioURL := ttsResp["url"]
	filename := filepath.Base(audioURL)

	c.JSON(http.StatusOK, EdgeTTSTTSResponse{
		URL:      audioURL,
		Filename: filename,
	})
}

// Health 健康检查
// GET /api/edge-tts/health
func (h *EdgeTTSHandler) Health(c *gin.Context) {
	resp, err := h.httpClient.Get(h.ttsServiceURL + "/health")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "error",
			"error":    "无法连接到 TTS 服务",
			"detail":   err.Error(),
			"edge_tts": false,
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	result["url"] = h.ttsServiceURL
	c.JSON(http.StatusOK, result)
}

// ServeAudioFile 提供音频文件访问（代理到 uploads 目录）
// GET /api/edge-tts/audio/:filename
func (h *EdgeTTSHandler) ServeAudioFile(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件名"})
		return
	}

	// 安全检查：只允许字母数字和常见音频扩展名
	if !isValidAudioFilename(filename) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件名"})
		return
	}

	// 音频文件存储在 ./data/uploads 目录
	audioDir := "./data/uploads"
	filePath := filepath.Join(audioDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.File(filePath)
}

// isValidAudioFilename 检查文件名是否安全
func isValidAudioFilename(filename string) bool {
	allowedExt := map[string]bool{
		".mp3":  true,
		".wav":  true,
		".m4a":  true,
		".ogg":  true,
		".webm": true,
	}
	ext := filepath.Ext(filename)
	return allowedExt[ext] && filename == filepath.Base(filename)
}

// ConvertAudioFormat 转换音频格式（使用 ffmpeg）
// POST /api/edge-tts/convert
func (h *EdgeTTSHandler) ConvertAudioFormat(c *gin.Context) {
	var req struct {
		SourceURL string `json:"source_url" binding:"required"` // 源文件 URL 或路径
		Format    string `json:"format" binding:"required"`      // 目标格式 wav/mp3
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求: " + err.Error()})
		return
	}

	// 处理相对 URL：拼接当前服务的 host
	sourceURL := req.SourceURL
	if strings.HasPrefix(sourceURL, "/") {
		// 相对路径，拼接当前请求的 host
		host := c.Request.Host
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		sourceURL = fmt.Sprintf("%s://%s%s", scheme, host, sourceURL)
	}

	// 获取源文件
	resp, err := h.httpClient.Get(sourceURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "下载源文件失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "源文件下载失败"})
		return
	}

	// 读取源文件
	sourceData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取源文件失败: " + err.Error()})
		return
	}

	// 创建临时文件
	tmpDir := os.TempDir()
	sourcePath := filepath.Join(tmpDir, fmt.Sprintf("source_%d.mp3", os.Getpid()))
	outputPath := filepath.Join(tmpDir, fmt.Sprintf("output_%d.%s", os.Getpid(), req.Format))

	if err := os.WriteFile(sourcePath, sourceData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时文件失败: " + err.Error()})
		return
	}
	defer os.Remove(sourcePath)

	// 调用 ffmpeg 转换
	var ffmpegCmd []string
	if req.Format == "wav" {
		ffmpegCmd = []string{"ffmpeg", "-y", "-i", sourcePath, outputPath}
	} else {
		ffmpegCmd = []string{"ffmpeg", "-y", "-i", sourcePath, "-acodec", "libmp3lame", "-q:a", "2", outputPath}
	}

	cmd := exec.Command(ffmpegCmd[0], ffmpegCmd[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(outputPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("音频转换失败: %s, 错误: %v", string(output), err)})
		return
	}
	defer os.Remove(outputPath)

	// 读取转换后的文件
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取转换后的文件失败: " + err.Error()})
		return
	}

	// 设置响应头
	if req.Format == "wav" {
		c.Header("Content-Type", "audio/wav")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=voice.%s", req.Format))
	} else {
		c.Header("Content-Type", "audio/mpeg")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=voice.%s", req.Format))
	}

	c.Data(http.StatusOK, "", outputData)
}