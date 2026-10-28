package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// 本文件覆盖 /ai-chat 会话与消息同步（多端一致）的关键行为：
//  1. 会话 upsert 幂等、列表按 updated_at 倒序；
//  2. 消息 upsert 幂等（流式过程中会反复覆盖同一条），增量拉取只返回新消息；
//  3. dataURL 图片落盘后只回 URL（避免 base64 撑爆 localStorage / 数据库）；
//  4. 删除会话连带删消息。

func newChatSyncTestRouter(t *testing.T) (*gin.Engine, *models.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitChatPlusTasks(); err != nil {
		t.Fatalf("init chat_plus_tasks: %v", err)
	}
	if err := db.InitChatPlusConvs(); err != nil {
		t.Fatalf("init chat_plus_convs: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ChatPlus.Password = testChatPwd
	h := NewAIGatewayHandler(db, cfg, nil, testEncEncryptionService)

	// 图片写到临时目录，别污染工作目录
	prevDir := chatPlusImageDir
	chatPlusImageDir = t.TempDir()
	t.Cleanup(func() { chatPlusImageDir = prevDir })

	r := gin.New()
	r.GET("/api/chat-plus/conversations", h.ChatPlusConversations)
	r.POST("/api/chat-plus/conversations", h.ChatPlusConversations)
	r.DELETE("/api/chat-plus/conversations/:id", h.ChatPlusConversationDelete)
	r.GET("/api/chat-plus/conversations/:id/messages", h.ChatPlusMessages)
	r.POST("/api/chat-plus/conversations/:id/messages", h.ChatPlusMessages)
	return r, db
}

func TestChatPlusConversationUpsertAndList(t *testing.T) {
	router, _ := newChatSyncTestRouter(t)

	for _, title := range []string{"第一个会话", ""} {
		body, _ := json.Marshal(map[string]string{"id": "conv-a", "title": title})
		w := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/conversations", string(body))
		if w.Code != http.StatusOK {
			t.Fatalf("upsert status = %d body=%s", w.Code, w.Body.String())
		}
	}

	// 第二次 title="" 不能把已有标题清空
	w := chatTaskRequest(t, router, http.MethodGet, "/api/chat-plus/conversations", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d", w.Code)
	}
	var resp struct {
		Conversations []models.ChatPlusConversation `json:"conversations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Conversations) != 1 {
		t.Fatalf("会话数 = %d, want 1", len(resp.Conversations))
	}
	if resp.Conversations[0].Title != "第一个会话" {
		t.Errorf("标题 = %q, want 第一个会话（空标题不应覆盖）", resp.Conversations[0].Title)
	}
}

func TestChatPlusMessagesUpsertIsIdempotent(t *testing.T) {
	router, db := newChatSyncTestRouter(t)

	// 流式过程中同一条 assistant 消息会被覆盖多次：最终只应留一条，正文是最后一次的
	for _, content := range []string{"你", "你好", "你好，世界"} {
		body, _ := json.Marshal(map[string]interface{}{
			"messages": []map[string]interface{}{{
				"id": "msg-1", "role": "assistant", "content": content,
				"task_id": "ctask_1", "ts": 1000,
			}},
		})
		w := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/conversations/conv-a/messages", string(body))
		if w.Code != http.StatusOK {
			t.Fatalf("upsert status = %d body=%s", w.Code, w.Body.String())
		}
	}

	msgs, err := db.ListChatPlusMessages("conv-a", 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("消息数 = %d, want 1（upsert 幂等）", len(msgs))
	}
	if msgs[0].Content != "你好，世界" {
		t.Errorf("正文 = %q, want 你好，世界", msgs[0].Content)
	}
	if msgs[0].TaskID != "ctask_1" {
		t.Errorf("task_id = %q", msgs[0].TaskID)
	}
}

func TestChatPlusMessagesSinceTsIsIncremental(t *testing.T) {
	router, _ := newChatSyncTestRouter(t)
	body, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]interface{}{
			{"id": "m1", "role": "user", "content": "旧", "ts": 1000},
			{"id": "m2", "role": "assistant", "content": "新", "ts": 2000},
		},
	})
	if w := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/conversations/c1/messages", string(body)); w.Code != http.StatusOK {
		t.Fatalf("upsert status = %d", w.Code)
	}

	w := chatTaskRequest(t, router, http.MethodGet, "/api/chat-plus/conversations/c1/messages?since_ts=1500", "")
	var resp struct {
		Messages []map[string]interface{} `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Messages) != 1 || resp.Messages[0]["id"] != "m2" {
		t.Fatalf("增量拉取结果 = %v, want 只有 m2", resp.Messages)
	}
}

func TestChatPlusImageDataURLStoredAsURL(t *testing.T) {
	router, _ := newChatSyncTestRouter(t)
	// 1x1 透明 PNG
	const png = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	body, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]interface{}{{
			"id": "m-img", "role": "user", "content": "看图", "ts": 10,
			"images": []map[string]interface{}{{"id": "i1", "dataUrl": png, "w": 1, "h": 1}},
		}},
	})
	w := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/conversations/c-img/messages", string(body))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Messages []struct {
			Images []chatPlusImageOut `json:"images"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Messages) != 1 || len(resp.Messages[0].Images) != 1 {
		t.Fatalf("图片未保存: %s", w.Body.String())
	}
	url := resp.Messages[0].Images[0].URL
	if len(url) == 0 || url[:len("/api/chat-plus/images/")] != "/api/chat-plus/images/" {
		t.Fatalf("图片 URL = %q, want 本地图片路径", url)
	}
	name := url[len("/api/chat-plus/images/"):]
	if !chatPlusImageNameRe.MatchString(name) {
		t.Fatalf("文件名 %q 不符合白名单格式", name)
	}
}

func TestChatPlusDeleteConversationDropsMessages(t *testing.T) {
	router, db := newChatSyncTestRouter(t)
	body, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]interface{}{{"id": "m1", "role": "user", "content": "hi", "ts": 1}},
	})
	if w := chatTaskRequest(t, router, http.MethodPost, "/api/chat-plus/conversations/c-del/messages", string(body)); w.Code != http.StatusOK {
		t.Fatalf("upsert status = %d", w.Code)
	}
	if w := chatTaskRequest(t, router, http.MethodDelete, "/api/chat-plus/conversations/c-del", ""); w.Code != http.StatusOK {
		t.Fatalf("delete status = %d", w.Code)
	}
	msgs, err := db.ListChatPlusMessages("c-del", 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("删除会话后仍有 %d 条消息", len(msgs))
	}
	convs, err := db.ListChatPlusConversations(10)
	if err != nil {
		t.Fatalf("list convs: %v", err)
	}
	if len(convs) != 0 {
		t.Fatalf("删除后仍有 %d 个会话", len(convs))
	}
}
