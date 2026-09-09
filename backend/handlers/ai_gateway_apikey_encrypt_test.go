package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/utils"
)

// setupAIGatewayEncryptDB 构造一个完整表初始化的 :memory: DB,
// SetMaxOpenConns(1) 防止懒迁移 goroutine 落到不同连接上查不到刚写的 row。
//
// 触发场景:dbAnthropicProvidersToConfig 在 legacy APIKey 非空时会
//
//	go func() { h.db.UpdateAnthropicProvider(...) }
//
// 异步把明文加密写回 DB。这是 feedback_sqlite_memory_pool.md 的标准触发模式。
func setupAIGatewayEncryptDB(t *testing.T) *models.DB {
	t.Helper()
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitAll(); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	return db
}

// newEncryptEnc 创建一个 AES-256-GCM 测试用加密服务(32 字节主密钥)。
func newEncryptEnc(t *testing.T) *utils.EncryptionService {
	t.Helper()
	enc, err := utils.NewEncryptionService("devtools-test-master-key-32-bytes!")
	if err != nil {
		t.Fatalf("NewEncryptionService: %v", err)
	}
	return enc
}

// TestEncryptionService_RoundTrip 单元测 Encrypt/Decrypt 往返,
// 与 utils/encryption_test.go 互补(这里从 handler 视角用,验证集成链路)。
func TestEncryptionService_RoundTrip(t *testing.T) {
	enc := newEncryptEnc(t)
	cases := []string{
		"plain-api-key-1",
		"sk-ant-abc123def456",
		"contains space and 中文",
		strings.Repeat("x", 1024),
	}
	for _, plain := range cases {
		t.Run(plain[:min(20, len(plain))], func(t *testing.T) {
			cipher, err := enc.Encrypt(plain)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			if cipher == "" {
				t.Fatalf("Encrypt 返回空密文")
			}
			if cipher == plain {
				t.Fatalf("密文与明文相同,加密未生效")
			}
			got, err := enc.Decrypt(cipher)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if got != plain {
				t.Errorf("Decrypt round-trip 失配: got %q, want %q", got, plain)
			}
		})
	}
}

// TestAIGatewayProvider_CreateStoresEncrypted 验证 CreateAnthropicProvider
// 写入时若 APIKeyEncrypted 已设置,DB 里读到的是密文、APIKey 字段为空。
func TestAIGatewayProvider_CreateStoresEncrypted(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)

	plain := "sk-test-plain-api-key-do-not-log"
	cipher, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	p := &models.AnthropicProvider{
		Name:            "TestProvider",
		APIURL:          "https://api.example.com/anthropic",
		APIKey:          "", // legacy 明文不写,只走加密通道
		APIKeyEncrypted: cipher,
		Models:          "[]",
		Aliases:         "[]",
		Enabled:         true,
		IsDefault:       false,
		DefaultModel:    "claude-sonnet-4-6",
	}
	if err := db.CreateAnthropicProvider(p); err != nil {
		t.Fatalf("CreateAnthropicProvider: %v", err)
	}

	got, err := db.GetAnthropicProviderByID(p.ID)
	if err != nil {
		t.Fatalf("GetAnthropicProviderByID: %v", err)
	}
	if got.APIKey != "" {
		t.Errorf("legacy APIKey 应当为空, 实际: %q", got.APIKey)
	}
	if got.APIKeyEncrypted == "" {
		t.Fatalf("APIKeyEncrypted 不应为空")
	}
	if got.APIKeyEncrypted != cipher {
		t.Errorf("APIKeyEncrypted 与写入密文不一致")
	}
	// 反向:解密密文必须等于明文
	plainGot, err := enc.Decrypt(got.APIKeyEncrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if plainGot != plain {
		t.Errorf("decrypt = %q, want %q", plainGot, plain)
	}
}

// TestAIGatewayProvider_UpdateReEncrypted 验证 UpdateAnthropicProvider
// 后 DB 里存的密文被新密文替换,且 legacy 明文仍为空。
func TestAIGatewayProvider_UpdateReEncrypted(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)

	// 1) 先建一条带 cipherA 的 provider
	cipherA, _ := enc.Encrypt("old-key")
	p := &models.AnthropicProvider{
		Name:            "RotateKey",
		APIURL:          "https://api.example.com/anthropic",
		APIKeyEncrypted: cipherA,
		Models:          "[]",
		Aliases:         "[]",
		Enabled:         true,
		DefaultModel:    "claude-sonnet-4-6",
	}
	if err := db.CreateAnthropicProvider(p); err != nil {
		t.Fatalf("CreateAnthropicProvider: %v", err)
	}

	// 2) 加密新 key 并 Update
	cipherB, err := enc.Encrypt("rotated-key")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	p.APIKeyEncrypted = cipherB
	if err := db.UpdateAnthropicProvider(p); err != nil {
		t.Fatalf("UpdateAnthropicProvider: %v", err)
	}

	// 3) 读回来:应当是 cipherB
	got, err := db.GetAnthropicProviderByID(p.ID)
	if err != nil {
		t.Fatalf("GetAnthropicProviderByID: %v", err)
	}
	if got.APIKey != "" {
		t.Errorf("legacy APIKey 应当仍为空, 实际: %q", got.APIKey)
	}
	if got.APIKeyEncrypted != cipherB {
		t.Errorf("APIKeyEncrypted 应更新为 cipherB, 实际: %q", got.APIKeyEncrypted)
	}
	// 4) 哈希校验:确保密文变化被持久化
	hashA := sha256.Sum256([]byte(cipherA))
	hashB := sha256.Sum256([]byte(got.APIKeyEncrypted))
	if hex.EncodeToString(hashA[:]) == hex.EncodeToString(hashB[:]) {
		t.Errorf("更新前后密文哈希相同,疑似未真正写入")
	}
}

// TestAIGatewayProvider_LegacyPlaintextLazyMigration 是核心安全回归测试:
//
// 场景:升级前的 DB 行只有 legacy api_key=明文,api_key_encrypted="";
//
//	dbAnthropicProvidersToConfig 应当:
//	  a) 把明文回填到 cfg.APIKey(不报错,继续工作)
//	  b) 加密后异步 go func() 写回 DB(懒迁移)
//	  c) 写回后 legacy api_key 被清空,api_key_encrypted 是新密文
//
// 这是 feedback 提到的"加密迁移"路径,防止老 DB 里的明文 key 被前端序列化泄漏。
func TestAIGatewayProvider_LegacyPlaintextLazyMigration(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)
	h := NewAIGatewayHandler(db, config.DefaultConfig(), nil, enc)

	// 1) 直插一行 legacy 明文(模拟升级前数据)
	legacyPlain := "sk-legacy-plain-from-old-db"
	p := &models.AnthropicProvider{
		Name:         "LegacyProvider",
		APIURL:       "https://api.example.com/anthropic",
		APIKey:       legacyPlain,
		Models:       "[]",
		Aliases:      "[]",
		Enabled:      true,
		IsDefault:    false,
		DefaultModel: "claude-sonnet-4-6",
	}
	if err := db.CreateAnthropicProvider(p); err != nil {
		t.Fatalf("CreateAnthropicProvider (legacy): %v", err)
	}
	id := p.ID

	// 2) 通过 handler 的转换函数读出 cfg
	dbRows, err := db.GetAnthropicProviderByID(id)
	if err != nil {
		t.Fatalf("GetAnthropicProviderByID: %v", err)
	}
	cfgs := h.dbAnthropicProvidersToConfig([]*models.AnthropicProvider{dbRows})
	if len(cfgs) != 1 {
		t.Fatalf("len(cfgs) = %d, want 1", len(cfgs))
	}
	cfg := cfgs[0]
	// a) cfg.APIKey 应当等于明文(继续工作,这是设计行为)
	if cfg.APIKey != legacyPlain {
		t.Errorf("cfg.APIKey = %q, want %q(懒迁移应保留明文,直到异步写回完成)", cfg.APIKey, legacyPlain)
	}
	// b) dbRows 的 APIKeyEncrypted 应当已经被赋值(enc.Encrypt 后 p.APIKeyEncrypted = enc)
	if dbRows.APIKeyEncrypted == "" {
		t.Errorf("调用 dbAnthropicProvidersToConfig 后 p.APIKeyEncrypted 应已被赋值为密文(同步部分)")
	}

	// c) 等异步 goroutine 写回 DB(异步 UpdateAnthropicProvider)
	//    :memory: + SetMaxOpenConns(1) 保证 goroutine 一定能看到 row
	deadline := time.Now().Add(2 * time.Second)
	var finalRow *models.AnthropicProvider
	for time.Now().Before(deadline) {
		finalRow, err = db.GetAnthropicProviderByID(id)
		if err != nil {
			t.Fatalf("GetAnthropicProviderByID: %v", err)
		}
		if finalRow.APIKeyEncrypted != "" && finalRow.APIKey == "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if finalRow == nil {
		t.Fatalf("无法读到 final row")
	}
	if finalRow.APIKey != "" {
		t.Errorf("懒迁移写回后 legacy APIKey 应被清空, 实际: %q", finalRow.APIKey)
	}
	if finalRow.APIKeyEncrypted == "" {
		t.Errorf("懒迁移写回后 APIKeyEncrypted 应非空")
	}
	// d) 解密验证:异步写回的密文能解出明文
	dec, err := enc.Decrypt(finalRow.APIKeyEncrypted)
	if err != nil {
		t.Fatalf("Decrypt lazy cipher: %v", err)
	}
	if dec != legacyPlain {
		t.Errorf("decrypted lazy cipher = %q, want %q", dec, legacyPlain)
	}
}

// TestAIGatewayProvider_NormalEncryptedPath 走正常加密路径,确保:
//
//	dbAnthropicProvidersToConfig 返回的 cfg.APIKey 是明文(供后续代理调用),
//	而 DB 行只存密文 + 空明文。
func TestAIGatewayProvider_NormalEncryptedPath(t *testing.T) {
	db := setupAIGatewayEncryptDB(t)
	defer db.Close()
	enc := newEncryptEnc(t)
	h := NewAIGatewayHandler(db, config.DefaultConfig(), nil, enc)

	plain := "sk-normal-encrypted-key"
	cipher, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	p := &models.AnthropicProvider{
		Name:            "NormalEnc",
		APIURL:          "https://api.example.com/anthropic",
		APIKeyEncrypted: cipher,
		Models:          "[]",
		Aliases:         "[]",
		Enabled:         true,
		IsDefault:       false,
		DefaultModel:    "claude-sonnet-4-6",
	}
	if err := db.CreateAnthropicProvider(p); err != nil {
		t.Fatalf("CreateAnthropicProvider: %v", err)
	}

	row, err := db.GetAnthropicProviderByID(p.ID)
	if err != nil {
		t.Fatalf("GetAnthropicProviderByID: %v", err)
	}
	cfgs := h.dbAnthropicProvidersToConfig([]*models.AnthropicProvider{row})
	if len(cfgs) != 1 {
		t.Fatalf("len(cfgs) = %d, want 1", len(cfgs))
	}
	cfg := cfgs[0]
	if cfg.APIKey != plain {
		t.Errorf("cfg.APIKey = %q, want %q", cfg.APIKey, plain)
	}
	if row.APIKey != "" {
		t.Errorf("legacy APIKey 应当为空, 实际: %q", row.APIKey)
	}
	if row.APIKeyEncrypted == "" {
		t.Errorf("APIKeyEncrypted 应当非空")
	}
}
