package handlers

import (
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
