package handlers

// 集成测试：从环境变量读 MINIMAX_API_KEY，没设就 skip；设了就真实调用一次 /v1/music_generation 端到端验证。
// 默认 `go test ./handlers/` 不会跑；要跑集成测试用 `MINIMAX_API_KEY=xxx go test ./handlers/ -run MusicIntegration -v`。
// 本文件不会包含任何明文密钥，全部从环境变量读取。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const musicIntegrationBaseURL = "https://api.minimaxi.com"

// TestMusicIntegrationLiveAPI 对每个支持的 music 模型跑一次真实端到端生成。
// 默认 skip，只有在 MINIMAX_API_KEY 设置时才会真跑。
// 每个模型跑最短时长（10s 音频），验证：
//  1. 请求能完成（不被 90s 超时杀掉）
//  2. 响应里有 base_resp.status_code == 0
//  3. data.audio 或 data.audio_url / data.status == 2 存在
func TestMusicIntegrationLiveAPI(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY 未设置，跳过真实 API 集成测试")
	}

	models := []string{
		"music-3.0",
		"music-3.0-free",
		"music-2.6",
		"music-2.6-free",
		"music-cover",
		"music-cover-free",
	}

	submitClient := &http.Client{Timeout: 5 * time.Minute}
	downloadClient := &http.Client{Timeout: 60 * time.Second}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			body := buildMusicRequestBody(t, model)
			req, err := http.NewRequest(http.MethodPost, musicIntegrationBaseURL+"/v1/music_generation", bytes.NewReader(body))
			if err != nil {
				t.Fatalf("构造请求失败: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("Content-Type", "application/json")

			start := time.Now()
			resp, err := submitClient.Do(req)
			if err != nil {
				t.Fatalf("请求失败（怀疑 5 分钟超时不够 / 网络问题）: %v", err)
			}
			defer resp.Body.Close()
			elapsed := time.Since(start)

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取响应失败: %v", err)
			}

			t.Logf("[%s] 生成耗时 %s，HTTP %d，响应长度 %d 字节", model, elapsed.Round(time.Second), resp.StatusCode, len(respBody))

			var payload map[string]interface{}
			if err := json.Unmarshal(respBody, &payload); err != nil {
				t.Fatalf("响应不是 JSON: %v\nbody: %s", err, truncateForLog(respBody))
			}

			baseResp, _ := payload["base_resp"].(map[string]interface{})
			statusCode, _ := baseResp["status_code"].(float64)
			if int(statusCode) != 0 {
				statusMsg, _ := baseResp["status_msg"].(string)
				t.Fatalf("base_resp.status_code=%v, msg=%q\nbody: %s", statusCode, statusMsg, truncateForLog(respBody))
			}

			data, _ := payload["data"].(map[string]interface{})
			if data == nil {
				t.Fatalf("响应缺少 data 字段\nbody: %s", truncateForLog(respBody))
			}
			status, _ := data["status"].(float64)
			if status != 2 {
				t.Fatalf("data.status=%v, want 2 (已完成)", status)
			}
			audioRaw, _ := data["audio"].(string)
			if audioRaw == "" {
				t.Fatalf("data.audio 为空\nbody: %s", truncateForLog(respBody))
			}

			// output_format=url 时 data.audio 是音频 URL，下载验证 magic bytes。
			// output_format=hex 时 data.audio 是 hex 编码音频数据，解码后验证 magic bytes。
			if strings.HasPrefix(audioRaw, "http://") || strings.HasPrefix(audioRaw, "https://") {
				verifyAudioURL(t, downloadClient, audioRaw, model)
			} else {
				verifyAudioHex(t, audioRaw, model)
			}

			if elapsed > 4*time.Minute {
				t.Logf("[%s] 警告：耗时 %s 接近 5 分钟超时，建议上调 musicSubmitClient 超时", model, elapsed)
			}
		})
	}
}

// verifyAudioURL 下载音频 URL，校验返回是合法 mp3/wav 文件（magic bytes + 合理大小）。
func verifyAudioURL(t *testing.T, client *http.Client, url, model string) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("[%s] 下载音频失败: %v", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("[%s] 下载音频 HTTP %d", model, resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	// 读前 16 字节用于 magic bytes 校验
	head := make([]byte, 16)
	n, _ := io.ReadFull(resp.Body, head)
	head = head[:n]
	// 全部读取用于大小校验（限制 50MB 防止 OOM）
	limited := io.LimitReader(resp.Body, 50*1024*1024)
	rest, _ := io.ReadAll(limited)
	full := append(head, rest...)
	if len(full) < 1024 {
		t.Fatalf("[%s] 音频文件过小: %d 字节", model, len(full))
	}
	if !looksLikeAudio(head, contentType) {
		t.Fatalf("[%s] 文件不像合法音频: magic=%x, content-type=%s, size=%d", model, head, contentType, len(full))
	}
	t.Logf("[%s] 音频 URL 下载成功: %d 字节, content-type=%s", model, len(full), contentType)
}

// verifyAudioHex 解码 hex 音频数据，校验 magic bytes 和大小。
func verifyAudioHex(t *testing.T, hexData, model string) {
	t.Helper()
	decoded, err := decodeHex(hexData)
	if err != nil {
		t.Fatalf("[%s] hex 解码失败: %v", model, err)
	}
	if len(decoded) < 1024 {
		t.Fatalf("[%s] hex 解码后音频过小: %d 字节", model, len(decoded))
	}
	head := decoded
	if len(head) > 16 {
		head = decoded[:16]
	}
	if !looksLikeAudio(head, "") {
		t.Fatalf("[%s] 解码后不像合法音频: magic=%x, size=%d", model, head, len(decoded))
	}
	t.Logf("[%s] hex 音频解码成功: %d 字节", model, len(decoded))
}

// looksLikeAudio 校验文件 magic bytes 是不是常见音频格式。
func looksLikeAudio(head []byte, contentType string) bool {
	ct := strings.ToLower(contentType)
	if strings.HasPrefix(ct, "audio/") {
		return true
	}
	// MP3: "ID3" 标签 或 0xFF 0xFB/0xFA/0xF3/0xF2 frame sync
	if len(head) >= 3 && string(head[:3]) == "ID3" {
		return true
	}
	if len(head) >= 2 && head[0] == 0xFF && (head[1]&0xE0) == 0xE0 {
		return true
	}
	// WAV: "RIFF....WAVE"
	if len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WAVE" {
		return true
	}
	// OGG: "OggS"
	if len(head) >= 4 && string(head[:4]) == "OggS" {
		return true
	}
	// FLAC: "fLaC"
	if len(head) >= 4 && string(head[:4]) == "fLaC" {
		return true
	}
	// M4A/MP4/AAC: "ftyp" at offset 4
	if len(head) >= 8 && string(head[4:8]) == "ftyp" {
		return true
	}
	return false
}

func decodeHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errOddLength
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		hi, ok1 := hexVal(s[i])
		lo, ok2 := hexVal(s[i+1])
		if !ok1 || !ok2 {
			return nil, errInvalidHex
		}
		out[i/2] = hi<<4 | lo
	}
	return out, nil
}

func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

var (
	errOddLength  = hexErr("hex 字符串长度必须为偶数")
	errInvalidHex = hexErr("hex 字符串包含非法字符")
)

type hexErr string

func (e hexErr) Error() string { return string(e) }

// buildMusicRequestBody 为不同模型构造最小可生成请求体。
// music-cover 需要 cover_feature_id，但跑集成测试拿不到真实 feature，
// 所以只校验前 5 个文本生成模型；cover 系列跳过端到端（保留路由测试在前面的单测）。
func buildMusicRequestBody(t *testing.T, model string) []byte {
	t.Helper()
	if strings.HasPrefix(model, "music-cover") {
		t.Skip("music-cover 需要 cover_feature_id，跑不通端到端；路由/允许列表/超时已在前面的单测覆盖")
	}
	body := map[string]interface{}{
		"model":         model,
		"prompt":        "轻轻的钢琴前奏，温暖治愈",
		"lyrics":        "[Verse]\n测试歌词\n[Chorus]\n副歌歌词",
		"output_format": "url",
		"audio_setting": map[string]interface{}{
			"sample_rate": 44100,
			"bitrate":     256000,
			"format":      "mp3",
		},
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	return b
}

func truncateForLog(b []byte) string {
	const maxLen = 500
	if len(b) <= maxLen {
		return string(b)
	}
	return string(b[:maxLen]) + "...(truncated)"
}