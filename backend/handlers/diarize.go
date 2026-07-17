package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiarizeTurn 对应 diarize-service /diarize 响应 turns 数组里的一项。
type DiarizeTurn struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
}

// DiarizeResponse /diarize 完整响应。
type DiarizeResponse struct {
	Turns  []DiarizeTurn `json:"turns"`
	Status string        `json:"status"`
	Error  string        `json:"error"`
}

// DiarizeSegment 是合并阶段的最小单元(Start/End/Text/Speaker)。
// 不耦合具体 ASR 段类型,各 handler 在边界转一次。
type DiarizeSegment struct {
	Start   float64
	End     float64
	Text    string
	Speaker string
}

// DiarizeClient 调外部 diarize-service。
// URL 为空时表示未启用,NewDiarizeClient 直接返回 nil。
type DiarizeClient struct {
	URL        string
	HTTPClient *http.Client
}

// NewDiarizeClient 构造客户端;url 为空时返回 nil(调用方按 nil 处理)。
func NewDiarizeClient(url string) *DiarizeClient {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	return &DiarizeClient{
		URL:        url,
		HTTPClient: &http.Client{Timeout: 5 * time.Minute, Transport: &http.Transport{Proxy: nil}},
	}
}

// Enabled 返回是否启用(URL 非空)。
func (c *DiarizeClient) Enabled() bool { return c != nil && c.URL != "" }

// Call 上传音频到 /diarize 并返回说话人分段。
// 失败/超时/服务不可用都返回 error,调用方应降级为只转写,不要把错透给用户。
func (c *DiarizeClient) Call(filePath, originalName string) (*DiarizeResponse, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("diarize client not enabled")
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
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(c.URL, "/")+"/diarize", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("diarize 服务返回错误(%d): %s", resp.StatusCode, truncateString(string(respBody), 300))
	}

	result := &DiarizeResponse{}
	if err := json.Unmarshal(respBody, result); err != nil {
		return nil, err
	}
	return result, nil
}

// AssignSpeakers 按段中点时间把说话人标签贴到转写段上。
// turns 为空时原样返回(Speaker 留空)。
func AssignSpeakers(segs []DiarizeSegment, turns []DiarizeTurn) []DiarizeSegment {
	if len(turns) == 0 {
		return segs
	}
	out := make([]DiarizeSegment, len(segs))
	for i, s := range segs {
		mid := (s.Start + s.End) / 2.0
		speaker := ""
		for _, t := range turns {
			if t.Start <= mid && mid <= t.End {
				speaker = t.Speaker
				break
			}
		}
		s.Speaker = speaker
		out[i] = s
	}
	return out
}

// FormatTranscriptWithSpeakers 把段拼成多行文本,带说话人前缀。
// 没有标签的段保持原样,空段跳过。
func FormatTranscriptWithSpeakers(segs []DiarizeSegment) string {
	var b strings.Builder
	for _, s := range segs {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		if s.Speaker != "" {
			b.WriteString("[")
			b.WriteString(s.Speaker)
			b.WriteString("] ")
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
