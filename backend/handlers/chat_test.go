package handlers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"
)

func newChatTestHandler(t *testing.T) (*ChatHandler, *Room, string) {
	t.Helper()

	db, err := models.NewDB(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	if err := db.InitChat(); err != nil {
		t.Fatalf("InitChat failed: %v", err)
	}

	roomModel := &models.ChatRoom{Name: "test-room", CreatorIP: "127.0.0.1"}
	if err := db.CreateRoom(roomModel); err != nil {
		t.Fatalf("CreateRoom failed: %v", err)
	}

	handler := NewChatHandler(db, "", config.MiniMaxConfig{}, "")
	room := &Room{
		ID:         roomModel.ID,
		clients:    make(map[*Client]bool),
		bots:       make(map[string]*BotConfig),
		histories:  make(map[string][]botMessage),
		botCancels: make(map[string]context.CancelFunc),
	}
	handler.rooms[roomModel.ID] = room
	handler.botDelayFunc = func(turnIndex int, totalTurns int) time.Duration { return 0 }
	return handler, room, roomModel.ID
}

func waitUntil(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestRunBotConversationContinuesMultipleTurns(t *testing.T) {
	h, room, roomID := newChatTestHandler(t)
	room.bots["BotA"] = &BotConfig{Enabled: true, Nickname: "BotA", SystemPrompt: "A"}
	room.bots["BotB"] = &BotConfig{Enabled: true, Nickname: "BotB", SystemPrompt: "B"}

	h.botReplyFunc = func(ctx context.Context, systemPrompt string, history []botMessage) (string, error) {
		last := history[len(history)-1].Content
		return systemPrompt + " reply to " + last, nil
	}

	if !h.startBotConversation(room, roomID, "用户", "聊聊今天的话题", []*BotConfig{
		room.bots["BotA"],
		room.bots["BotB"],
	}) {
		t.Fatal("expected conversation to start")
	}

	waitUntil(t, 2*time.Second, func() bool {
		messages, err := h.db.GetMessages(roomID, 10)
		if err != nil {
			return false
		}
		return len(messages) == 4
	})

	messages, err := h.db.GetMessages(roomID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(messages))
	}

	wantNicknames := []string{"BotA", "BotB", "BotA", "BotB"}
	for i, want := range wantNicknames {
		if messages[i].Nickname != want {
			t.Fatalf("messages[%d].Nickname = %q, want %q", i, messages[i].Nickname, want)
		}
	}
}

func TestStartBotConversationSmoothlySwitchesTopic(t *testing.T) {
	h, room, roomID := newChatTestHandler(t)
	room.bots["BotA"] = &BotConfig{Enabled: true, Nickname: "BotA", SystemPrompt: "A"}

	firstStarted := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})

	h.botReplyFunc = func(ctx context.Context, systemPrompt string, history []botMessage) (string, error) {
		last := history[len(history)-1].Content
		if last == "用户: 旧话题" {
			select {
			case firstStarted <- struct{}{}:
			default:
			}
			select {
			case <-releaseFirst:
			case <-ctx.Done():
				t.Fatalf("old topic should not be interrupted before finishing")
			}
			return "旧话题收尾回复", nil
		}
		return "新话题回复", nil
	}

	if !h.startBotConversation(room, roomID, "用户", "旧话题", []*BotConfig{room.bots["BotA"]}) {
		t.Fatal("expected first conversation to start")
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first conversation did not start in time")
	}

	if !h.startBotConversation(room, roomID, "用户", "新话题", []*BotConfig{room.bots["BotA"]}) {
		t.Fatal("expected second conversation to start")
	}

	close(releaseFirst)

	waitUntil(t, 2*time.Second, func() bool {
		messages, err := h.db.GetMessages(roomID, 10)
		if err != nil {
			return false
		}
		return len(messages) == 2
	})

	messages, err := h.db.GetMessages(roomID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Content != "旧话题收尾回复" {
		t.Fatalf("messages[0].Content = %q", messages[0].Content)
	}
	if messages[1].Content != "新话题回复" {
		t.Fatalf("messages[1].Content = %q", messages[1].Content)
	}
}

func TestRequestRoomBotSessionStopWaitsCurrentReplyFinish(t *testing.T) {
	h, room, roomID := newChatTestHandler(t)
	room.bots["BotA"] = &BotConfig{Enabled: true, Nickname: "BotA", SystemPrompt: "A"}
	room.bots["BotB"] = &BotConfig{Enabled: true, Nickname: "BotB", SystemPrompt: "B"}

	firstStarted := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})

	h.botReplyFunc = func(ctx context.Context, systemPrompt string, history []botMessage) (string, error) {
		last := history[len(history)-1].Content
		if systemPrompt == "A" && last == "用户: 旧话题" {
			select {
			case firstStarted <- struct{}{}:
			default:
			}
			select {
			case <-releaseFirst:
			case <-ctx.Done():
				t.Fatalf("stop should not cut current reply immediately")
			}
			return "第一条回复", nil
		}
		return systemPrompt + " 后续回复", nil
	}

	if !h.startBotConversation(room, roomID, "用户", "旧话题", []*BotConfig{
		room.bots["BotA"],
		room.bots["BotB"],
	}) {
		t.Fatal("expected conversation to start")
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first conversation did not start in time")
	}

	room.mu.Lock()
	requestRoomBotSessionStopLocked(room)
	room.mu.Unlock()

	close(releaseFirst)

	waitUntil(t, 2*time.Second, func() bool {
		messages, err := h.db.GetMessages(roomID, 10)
		if err != nil {
			return false
		}
		return len(messages) == 1
	})

	messages, err := h.db.GetMessages(roomID, 10)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0].Content != "第一条回复" {
		t.Fatalf("messages[0].Content = %q", messages[0].Content)
	}
}
