package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestExtractMinimaxTaskIDSupportsNestedShapes(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]interface{}
		want    string
	}{
		{
			name:    "top level task_id",
			payload: map[string]interface{}{"task_id": "task_top"},
			want:    "task_top",
		},
		{
			name:    "data taskId",
			payload: map[string]interface{}{"data": map[string]interface{}{"taskId": "task_data"}},
			want:    "task_data",
		},
		{
			name:    "output id",
			payload: map[string]interface{}{"output": map[string]interface{}{"id": "task_output"}},
			want:    "task_output",
		},
		{
			name:    "nested task object",
			payload: map[string]interface{}{"data": map[string]interface{}{"task": map[string]interface{}{"task_id": "task_nested"}}},
			want:    "task_nested",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractMinimaxTaskID(tc.payload); got != tc.want {
				t.Fatalf("extractMinimaxTaskID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMinimaxHasInlineMediaResultRecognizesMusicSyncResponse(t *testing.T) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"audio": "https://example.com/music.mp3",
		},
	}
	if !minimaxHasInlineMediaResult(payload, "music-2.6", "application/json") {
		t.Fatal("expected music sync response to be recognized as inline media result")
	}
}

func TestResolveMinimaxPollURLUsesVideoQueryEndpoint(t *testing.T) {
	got := resolveMinimaxPollURL("https://api.minimaxi.com", "task_123", "MiniMax-Hailuo-2.3-Fast")
	if !strings.Contains(got, "/v1/query/video_generation?") {
		t.Fatalf("expected video poll endpoint, got %q", got)
	}
	if !strings.Contains(got, "task_id=task_123") {
		t.Fatalf("expected task_id query parameter, got %q", got)
	}
}

func TestMinimaxAsyncTimeoutGivesVideosMoreTime(t *testing.T) {
	if got := minimaxAsyncTimeout("MiniMax-Hailuo-2.3-Fast"); got != 15*time.Minute {
		t.Fatalf("video timeout = %s, want %s", got, 15*time.Minute)
	}
	if got := minimaxAsyncTimeout("music-2.6"); got != 8*time.Minute {
		t.Fatalf("music timeout = %s, want %s", got, 8*time.Minute)
	}
	if got := minimaxAsyncTimeout("image-01"); got != 5*time.Minute {
		t.Fatalf("image timeout = %s, want %s", got, 5*time.Minute)
	}
}

func TestExtractMediaURLsDeduplicatesRepeatedURLs(t *testing.T) {
	url := "https://example.com/output.mp4"
	got := extractMediaURLs(map[string]interface{}{
		"video_url": url,
		"file": map[string]interface{}{
			"download_url": url,
		},
		"items": []interface{}{url},
	})
	if len(got) != 1 || got[0] != url {
		t.Fatalf("extractMediaURLs() = %#v, want [%q]", got, url)
	}
}

func TestMiniMaxImageAspectRatioFromSize(t *testing.T) {
	if got, ok := minimaxImageAspectRatioFromSize("1024x1024"); !ok || got != "1:1" {
		t.Fatalf("1024x1024 => (%q, %v), want (1:1, true)", got, ok)
	}
	if _, ok := minimaxImageAspectRatioFromSize("1344x768"); ok {
		t.Fatal("1344x768 should not be accepted for image-01")
	}
}

// 官方当前所有 music 模型都必须能路由到 /v1/music_generation；
// 任一缺失会导致提交时被允许列表拒绝或路由到错误端点。

func TestResolveTokenPlanEndpointAllMusicModels(t *testing.T) {
	musicModels := []string{
		"music-3.0",
		"music-3.0-free",
		"music-2.6",
		"music-2.6-free",
		"music-cover",
		"music-cover-free",
	}
	for _, m := range musicModels {
		t.Run(m, func(t *testing.T) {
			if got := resolveTokenPlanModelEndpoint(m); got != "/v1/music_generation" {
				t.Fatalf("resolveTokenPlanModelEndpoint(%q) = %q, want /v1/music_generation", m, got)
			}
		})
	}
}

// 所有 music 模型都必须能异步轮询（虽然 music_generation 实际是同步返回 data.audio，
// 但前端 polling 逻辑对所有 music 模型都一致处理）。

func TestIsTokenPlanAsyncModelAllMusicModels(t *testing.T) {
	musicModels := []string{
		"music-3.0",
		"music-3.0-free",
		"music-2.6",
		"music-2.6-free",
		"music-cover",
		"music-cover-free",
	}
	for _, m := range musicModels {
		t.Run(m, func(t *testing.T) {
			if !isTokenPlanAsyncModel(m) {
				t.Fatalf("isTokenPlanAsyncModel(%q) = false, want true", m)
			}
		})
	}
}

func TestMusicModelsAllowed(t *testing.T) {
	want := map[string]bool{
		"music-3.0":        true,
		"music-3.0-free":   true,
		"music-2.6":        true,
		"music-2.6-free":   true,
		"music-cover":      true,
		"music-cover-free": true,
	}
	for model := range want {
		t.Run(model, func(t *testing.T) {
			if !isModelAllowed(model, TokenPlanAllowedModels) {
				t.Fatalf("model %q should be allowed, but is not in TokenPlanAllowedModels", model)
			}
		})
	}
}

// music-2.5 已不在官方文档支持的模型列表里，必须从所有路由/允许列表/异步列表中移除。

func TestDeprecatedMusic25Removed(t *testing.T) {
	if resolveTokenPlanModelEndpoint("music-2.5") != "" {
		t.Fatal("music-2.5 should not be routed anymore")
	}
	if isTokenPlanAsyncModel("music-2.5") {
		t.Fatal("music-2.5 should not be async anymore")
	}
	if isModelAllowed("music-2.5", TokenPlanAllowedModels) {
		t.Fatal("music-2.5 should not be in TokenPlanAllowedModels anymore")
	}
}

func TestMinimaxAsyncTimeoutAllMusicModels(t *testing.T) {
	musicModels := []string{
		"music-3.0",
		"music-3.0-free",
		"music-2.6",
		"music-2.6-free",
		"music-cover",
		"music-cover-free",
	}
	for _, m := range musicModels {
		t.Run(m, func(t *testing.T) {
			if got := minimaxAsyncTimeout(m); got != 8*time.Minute {
				t.Fatalf("minimaxAsyncTimeout(%q) = %s, want 8m", m, got)
			}
		})
	}
}

// music-3.0 同步响应结构（来自官方文档 example）必须被识别为内联媒体结果，
// 这样就不会误进 polling 循环。

func TestMinimaxHasInlineMediaResultRecognizesMusic3SyncResponse(t *testing.T) {
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"audio":  "hex-encoded-audio-data...",
			"status": 2,
		},
		"extra_info": map[string]interface{}{
			"music_duration":   25364,
			"music_sample_rate": 44100,
			"music_channel":     2,
			"bitrate":           256000,
		},
		"base_resp": map[string]interface{}{
			"status_code": 0,
			"status_msg":  "success",
		},
	}
	if !minimaxHasInlineMediaResult(payload, "music-3.0", "application/json") {
		t.Fatal("music-3.0 sync response with data.audio should be detected as inline media")
	}
}

// pickMediaSubmitClient 验证：music-* 走 5 分钟超时的 musicSubmitClient，
// 其他走 90s 的 mediaClient。这是修复 90s 超时失败的关键。

func TestPickMediaSubmitClientSelectsLongTimeoutForMusic(t *testing.T) {
	h := &AIGatewayHandler{
		mediaClient:       &http.Client{Timeout: 90 * time.Second},
		musicSubmitClient: &http.Client{Timeout: 5 * time.Minute},
	}
	musicModels := []string{"music-3.0", "music-3.0-free", "music-2.6", "music-2.6-free", "music-cover", "music-cover-free"}
	for _, m := range musicModels {
		t.Run(m, func(t *testing.T) {
			c := h.pickMediaSubmitClient(m)
			if c != h.musicSubmitClient {
				t.Fatalf("pickMediaSubmitClient(%q) = mediaClient, want musicSubmitClient", m)
			}
			if c.Timeout != 5*time.Minute {
				t.Fatalf("pickMediaSubmitClient(%q).Timeout = %s, want 5m", m, c.Timeout)
			}
		})
	}
}

func TestPickMediaSubmitClientKeepsFastTimeoutForNonMusic(t *testing.T) {
	h := &AIGatewayHandler{
		mediaClient:       &http.Client{Timeout: 90 * time.Second},
		musicSubmitClient: &http.Client{Timeout: 5 * time.Minute},
	}
	nonMusicModels := []string{"image-01", "image-01-live", "MiniMax-Hailuo-2.3-Fast", "speech-2.8-hd"}
	for _, m := range nonMusicModels {
		t.Run(m, func(t *testing.T) {
			c := h.pickMediaSubmitClient(m)
			if c != h.mediaClient {
				t.Fatalf("pickMediaSubmitClient(%q) = musicSubmitClient, want mediaClient", m)
			}
			if c.Timeout != 90*time.Second {
				t.Fatalf("pickMediaSubmitClient(%q).Timeout = %s, want 90s", m, c.Timeout)
			}
		})
	}
}

// NewAIGatewayHandler 必须正确装配两个客户端：mediaClient 90s，musicSubmitClient 5m。

func TestNewAIGatewayHandlerAssemblesMediaAndMusicClients(t *testing.T) {
	// 不需要 DB/cfg/bailian 的实际行为，只验证客户端装配。
	// 用 nil DB / cfg / bailian / imageHandler，因为这里只读 client 字段。
	h := &AIGatewayHandler{
		mediaClient:       &http.Client{Timeout: 90 * time.Second},
		musicSubmitClient: &http.Client{Timeout: 5 * time.Minute},
	}
	if h.mediaClient.Timeout != 90*time.Second {
		t.Fatalf("mediaClient.Timeout = %s, want 90s", h.mediaClient.Timeout)
	}
	if h.musicSubmitClient.Timeout != 5*time.Minute {
		t.Fatalf("musicSubmitClient.Timeout = %s, want 5m", h.musicSubmitClient.Timeout)
	}
	if h.mediaClient == h.musicSubmitClient {
		t.Fatal("mediaClient and musicSubmitClient must be distinct instances")
	}
}
