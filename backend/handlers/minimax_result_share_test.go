package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

func setupMiniMaxResultShareHandler(t *testing.T) (*AIGatewayHandler, *models.DB) {
	t.Helper()
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	if err := db.InitMiniMaxResultShares(); err != nil {
		t.Fatalf("init result shares: %v", err)
	}
	h := NewAIGatewayHandler(db, config.DefaultConfig(), nil, nil)
	return h, db
}

func insertMiniMaxResultShare(t *testing.T, db *models.DB, share *models.MiniMaxResultShare, assets []models.MiniMaxResultShareAsset, payload map[string]interface{}) {
	t.Helper()
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	share.Payload = string(rawPayload)
	share.AssetsJSON = models.MustMarshalMiniMaxResultShareAssets(assets)
	if err := db.CreateMiniMaxResultShare(share); err != nil {
		t.Fatalf("create share: %v", err)
	}
}

func TestPublicListMiniMaxResultSharesIncludesUsableListFields(t *testing.T) {
	h, db := setupMiniMaxResultShareHandler(t)
	defer db.Close()

	insertMiniMaxResultShare(t, db, &models.MiniMaxResultShare{
		ID:         "mrs_video",
		Title:      "Video share",
		Summary:    "A generated video",
		ResultType: "media",
		Model:      "MiniMax-Hailuo-2.3",
		Status:     "active",
	}, []models.MiniMaxResultShareAsset{
		{ID: "asset_01", Kind: "video", Filename: "asset_01.mp4", ContentType: "video/mp4", SizeBytes: 2048},
	}, map[string]interface{}{
		"task_id":          "mmt_local",
		"external_task_id": "upstream_123",
		"request":          `{"prompt":"a dog running"}`,
	})

	router := gin.New()
	router.GET("/api/minimax/result-shares/public/list", h.PublicListMiniMaxResultShares)
	req := httptest.NewRequest(http.MethodGet, "/api/minimax/result-shares/public/list?limit=10", nil)
	req.Host = "example.test"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("total/items = %d/%d, want 1/1", resp.Total, len(resp.Items))
	}
	item := resp.Items[0]
	if got := item["display_type"]; got != "video" {
		t.Fatalf("display_type = %v, want video", got)
	}
	if got := item["task_id"]; got != "mmt_local" {
		t.Fatalf("task_id = %v, want mmt_local", got)
	}
	if got := item["external_task_id"]; got != "upstream_123" {
		t.Fatalf("external_task_id = %v, want upstream_123", got)
	}
	if got := item["prompt"]; got != "a dog running" {
		t.Fatalf("prompt = %v, want a dog running", got)
	}
	if got := item["asset_count"]; got != float64(1) {
		t.Fatalf("asset_count = %v, want 1", got)
	}
	primary, ok := item["primary_asset"].(map[string]interface{})
	if !ok {
		t.Fatalf("primary_asset missing or wrong type: %#v", item["primary_asset"])
	}
	if got := primary["kind"]; got != "video" {
		t.Fatalf("primary kind = %v, want video", got)
	}
	if got := primary["asset_url"]; got != "http://example.test/api/minimax/result-shares/mrs_video/assets/asset_01" {
		t.Fatalf("asset_url = %v", got)
	}
}

func TestPublicListMiniMaxResultSharesFiltersByAssetKind(t *testing.T) {
	h, db := setupMiniMaxResultShareHandler(t)
	defer db.Close()

	insertMiniMaxResultShare(t, db, &models.MiniMaxResultShare{
		ID:         "mrs_video",
		Title:      "Video share",
		ResultType: "media",
		Model:      "MiniMax-Hailuo-2.3",
		Status:     "active",
	}, []models.MiniMaxResultShareAsset{
		{ID: "asset_01", Kind: "video", Filename: "asset_01.mp4", ContentType: "video/mp4"},
	}, map[string]interface{}{"task_id": "mmt_video"})
	insertMiniMaxResultShare(t, db, &models.MiniMaxResultShare{
		ID:         "mrs_audio",
		Title:      "Audio share",
		ResultType: "media",
		Model:      "music-2.6",
		Status:     "active",
	}, []models.MiniMaxResultShareAsset{
		{ID: "asset_01", Kind: "audio", Filename: "asset_01.mp3", ContentType: "audio/mpeg"},
	}, map[string]interface{}{"task_id": "mmt_audio"})

	router := gin.New()
	router.GET("/api/minimax/result-shares/public/list", h.PublicListMiniMaxResultShares)
	req := httptest.NewRequest(http.MethodGet, "/api/minimax/result-shares/public/list?type=audio&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Items []map[string]interface{} `json:"items"`
		Total int                      `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("total/items = %d/%d, want 1/1", resp.Total, len(resp.Items))
	}
	if got := resp.Items[0]["id"]; got != "mrs_audio" {
		t.Fatalf("item id = %v, want mrs_audio", got)
	}
	if got := resp.Items[0]["display_type"]; got != "audio" {
		t.Fatalf("display_type = %v, want audio", got)
	}
}
