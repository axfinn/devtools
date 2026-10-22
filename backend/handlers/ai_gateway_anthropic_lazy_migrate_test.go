package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// 本文件覆盖 fix(ai-gateway) 的"旧明文懒迁移"修复:
//
// UpdateAnthropicProvider 是全列 UPDATE,旧代码在懒迁移写回时只塞了
// ID / APIKeyEncrypted / APIKey 三个字段,把 header_timeout_sec / default_model /
// models / enabled 等列一并清零 —— 例如把 Ollama 的 600 秒首字节窗口打成 0。
// 修复后的写法是把整行快照(snapshot := *p)带过去再写回。
//
// 断言锚点:
//   - 写回在 dbAnthropicProvidersToConfig 内部异步发生(goroutine),需要轮询等待
//   - SQLite :memory: 必须 SetMaxOpenConns(1),否则后台 goroutine 拿到不同连接,
//     读不到刚写的 row(feedback_sqlite_memory_pool.md)
//   - setupAIGatewayEncryptDB / newEncryptEnc 已处理好上面两件事,直接复用

// legacyLazyProviderFixture 建一条"升级前"的 provider 行:
// legacy 明文 api_key + 非零 header_timeout_sec + 非空 models / default_model / enabled。
func legacyLazyProviderFixture(t *testing.T, db *models.DB) *models.AnthropicProvider {
	t.Helper()
	p := &models.AnthropicProvider{
		Name:             "LegacyLazyProvider",
		APIURL:           "https://legacy.example.com/anthropic",
		APIKey:           "sk-legacy-lazy-plain-key",
		Models:           `["legacy-m1","legacy-m2"]`,
		Aliases:          `[]`,
		Enabled:          true,
		IsDefault:        true,
		DefaultModel:     "legacy-m1",
		HeaderTimeoutSec: 600,
	}
	if err := db.CreateAnthropicProvider(p); err != nil {
		t.Fatalf("CreateAnthropicProvider (legacy 明文行): %v", err)
	}
	return p
}

// waitLazyMigrated 轮询等待懒迁移写回完成(明文被清空 + 密文落库)。
func waitLazyMigrated(t *testing.T, db *models.DB, id int64) *models.AnthropicProvider {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var row *models.AnthropicProvider
	for time.Now().Before(deadline) {
		got, err := db.GetAnthropicProviderByID(id)
		if err != nil {
			t.Fatalf("GetAnthropicProviderByID: %v", err)
		}
		row = got
		if row.APIKeyEncrypted != "" && row.APIKey == "" {
			return row
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("懒迁移写回超时(3s): row=%+v; 期望 api_key_encrypted 非空且 api_key 已清空", row)
	return nil
}

// TestAnthropicLegacyLazyMigratePreservesColumns 走真实 HTTP 路径
// (GET /api/anthropic/v1/models → ProxyAnthropicModels → allAnthropicProviders
// → dbAnthropicProvidersToConfig)触发懒迁移,断言写回后其它列没被清零。
func TestAnthropicLegacyLazyMigratePreservesColumns(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)
	h := NewAIGatewayHandler(db, config.DefaultConfig(), nil, enc)

	created := legacyLazyProviderFixture(t, db)
	id := created.ID

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/anthropic/v1/models", h.ProxyAnthropicModels)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/anthropic/v1/models", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/anthropic/v1/models 状态码 = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "legacy-m1") {
		t.Fatalf("响应体未包含 provider 的模型列表, 期望含 \"legacy-m1\"; body=%s", rec.Body.String())
	}

	// 懒迁移写回(异步)完成后,其它列必须原样保留
	row := waitLazyMigrated(t, db, id)

	if row.HeaderTimeoutSec != 600 {
		t.Errorf("懒迁移把 header_timeout_sec 清零了 = %d, want 600", row.HeaderTimeoutSec)
	}
	if row.Models != `["legacy-m1","legacy-m2"]` {
		t.Errorf("懒迁移把 models 清零了 = %q", row.Models)
	}
	if !row.Enabled {
		t.Errorf("懒迁移把 enabled 清零了")
	}
	if !row.IsDefault {
		t.Errorf("懒迁移把 is_default 清零了")
	}
	if row.DefaultModel != "legacy-m1" {
		t.Errorf("懒迁移把 default_model 清零了 = %q, want \"legacy-m1\"", row.DefaultModel)
	}
	if row.APIURL != "https://legacy.example.com/anthropic" {
		t.Errorf("懒迁移把 api_url 改了 = %q", row.APIURL)
	}
	if row.Name != "LegacyLazyProvider" {
		t.Errorf("懒迁移把 name 改了 = %q", row.Name)
	}
	if row.Aliases != `[]` {
		t.Errorf("懒迁移把 aliases 改了 = %q", row.Aliases)
	}
	// 密文本身也要能解回明文(证明写回的是"整行 + 新密文")
	plain, err := enc.Decrypt(row.APIKeyEncrypted)
	if err != nil {
		t.Fatalf("Decrypt 懒迁移密文: %v", err)
	}
	if plain != "sk-legacy-lazy-plain-key" {
		t.Errorf("解密结果 = %q, want %q", plain, "sk-legacy-lazy-plain-key")
	}
}

// TestAnthropicLegacyLazyMigrateKeepsOllamaHeaderTimeout 内置 Ollama 线路
// (header_timeout_sec=600 / models=["qwen3.8:27b-mlx"] / api_key=明文 "ollama")
// 同样走懒迁移路径,写回后这些值必须保留 —— 这正是旧 bug 受害最重的一行。
func TestAnthropicLegacyLazyMigrateKeepsOllamaHeaderTimeout(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)
	h := NewAIGatewayHandler(db, config.DefaultConfig(), nil, enc)

	rows, err := db.ListAnthropicProviders()
	if err != nil {
		t.Fatalf("ListAnthropicProviders: %v", err)
	}
	var ollama *models.AnthropicProvider
	for _, p := range rows {
		if p.Name == "Ollama" {
			ollama = p
		}
	}
	if ollama == nil {
		t.Skipf("本环境没有 seed Ollama 行, 跳过")
	}
	if ollama.HeaderTimeoutSec != 600 {
		t.Fatalf("前置条件不满足: Ollama header_timeout_sec = %d, want 600", ollama.HeaderTimeoutSec)
	}
	if ollama.APIKey == "" || ollama.APIKeyEncrypted != "" {
		t.Fatalf("前置条件不满足: Ollama 行应是 legacy 明文(未加密)状态")
	}

	cfgs := h.dbAnthropicProvidersToConfig([]*models.AnthropicProvider{ollama})
	if len(cfgs) != 1 {
		t.Fatalf("len(cfgs) = %d, want 1", len(cfgs))
	}
	if cfgs[0].HeaderTimeoutSec != 600 {
		t.Errorf("cfg.HeaderTimeoutSec = %d, want 600", cfgs[0].HeaderTimeoutSec)
	}

	row := waitLazyMigrated(t, db, ollama.ID)
	if row.HeaderTimeoutSec != 600 {
		t.Errorf("Ollama 懒迁移后 header_timeout_sec = %d, want 600", row.HeaderTimeoutSec)
	}
	if row.Models != `["qwen3.8:27b-mlx"]` {
		t.Errorf("Ollama 懒迁移后 models = %q", row.Models)
	}
	if row.DefaultModel != "qwen3.8:27b-mlx" {
		t.Errorf("Ollama 懒迁移后 default_model = %q", row.DefaultModel)
	}
	if !row.Enabled {
		t.Errorf("Ollama 懒迁移后 enabled 被清零")
	}
}

// TestAnthropicLegacyLazyMigrateConfigKeepsTimeout 配置层断言:
// dbAnthropicProvidersToConfig 返回的 cfg 必须带上 header_timeout_sec / default_model,
// 否则按 provider 派生的流式客户端会退回 60s 默认值(慢上游首字节被掐死)。
func TestAnthropicLegacyLazyMigrateConfigKeepsTimeout(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)
	h := NewAIGatewayHandler(db, config.DefaultConfig(), nil, enc)

	created := legacyLazyProviderFixture(t, db)
	row, err := db.GetAnthropicProviderByID(created.ID)
	if err != nil {
		t.Fatalf("GetAnthropicProviderByID: %v", err)
	}
	cfgs := h.dbAnthropicProvidersToConfig([]*models.AnthropicProvider{row})
	if len(cfgs) != 1 {
		t.Fatalf("len(cfgs) = %d, want 1", len(cfgs))
	}
	if cfgs[0].HeaderTimeoutSec != 600 {
		t.Errorf("cfg.HeaderTimeoutSec = %d, want 600", cfgs[0].HeaderTimeoutSec)
	}
	if cfgs[0].DefaultModel != "legacy-m1" {
		t.Errorf("cfg.DefaultModel = %q, want \"legacy-m1\"", cfgs[0].DefaultModel)
	}
	if len(cfgs[0].Models) != 2 {
		t.Errorf("cfg.Models = %v, want 2 个模型", cfgs[0].Models)
	}
}

// TestAnthropicLegacyLazyMigrateDetectsPartialWrite 反事实校验(可证伪性保障):
// 证明上面三个用例的断言不是恒真闸门 —— 只要写回时漏传字段(旧 bug 的写法:
// 只填 ID / APIKeyEncrypted / APIKey),header_timeout_sec / models / enabled /
// default_model 确实会被清零。
//
// 本用例刻意复现"坏写法",因此它断言的是"坏写法会清零";一旦生产代码退化成坏写法,
// 主用例会失败,而本用例会失败并提示断言失去区分力。
func TestAnthropicLegacyLazyMigrateDetectsPartialWrite(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()

	created := legacyLazyProviderFixture(t, db)

	// 旧 bug 的写法:只塞三个字段
	if err := db.UpdateAnthropicProvider(&models.AnthropicProvider{
		ID:              created.ID,
		APIKeyEncrypted: "partial-write-placeholder",
		APIKey:          "",
	}); err != nil {
		t.Fatalf("UpdateAnthropicProvider (部分行): %v", err)
	}
	got, err := db.GetAnthropicProviderByID(created.ID)
	if err != nil {
		t.Fatalf("GetAnthropicProviderByID: %v", err)
	}
	if got.HeaderTimeoutSec == 600 {
		t.Errorf("反事实失败: 部分行写回没有清零 header_timeout_sec —— 主用例的断言已失去区分力")
	}
	if got.Models == `["legacy-m1","legacy-m2"]` {
		t.Errorf("反事实失败: 部分行写回没有清零 models —— 主用例的断言已失去区分力")
	}
	if got.Enabled {
		t.Errorf("反事实失败: 部分行写回没有清零 enabled —— 主用例的断言已失去区分力")
	}
	if got.DefaultModel == "legacy-m1" {
		t.Errorf("反事实失败: 部分行写回没有清零 default_model —— 主用例的断言已失去区分力")
	}
}
