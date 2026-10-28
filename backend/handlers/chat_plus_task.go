package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"devtools/models"
	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// 本文件：/ai-chat 的「异步任务化」聊天通道。
//
// 动机（弱网）：同步 SSE 的问题是"客户端连接就是任务本身"——切 Wi-Fi、锁屏、刷新、
// 地铁进隧道，连接一断，上游还在生成的内容就再也拿不回来，token 已经花掉了。
// 这里把任务搬到服务端后台跑：
//   POST /chat-plus/tasks            → 立刻返回 task_id（202）
//   GET  /chat-plus/tasks/:id/stream → SSE 订阅，?from=<seq> 从断点续传
//   GET  /chat-plus/tasks/:id        → 终态快照（成功内容 / 失败原因）
//   GET  /chat-plus/tasks?conversation_id=… → 页面刷新后对齐消息
//   POST /chat-plus/tasks/:id/stop   → 取消
//
// 原来的同步 /chat/stream 保留不动（兼容 + 兜底）。
// 任务终态落 SQLite（models.ChatPlusTask），内存只保留最近任务的 SSE 片段用于续传。

const (
	chatTaskBufferTTL   = 30 * time.Minute  // 内存片段保留时长（终态以 DB 为准）
	chatTaskMaxBuffers  = 64                // 内存中同时保留的任务数上限
	chatTaskMaxBytes    = 1 << 20           // 单任务缓存片段上限（1MB，聊天文本远小于此）
	chatTaskPersistGap  = 2 * time.Second   // 进度落库间隔（不能每个片段写一次库）
	chatTaskStreamIdle  = 10 * time.Minute  // 订阅端"无新片段"最长等待
	chatTaskHeartbeat   = 20 * time.Second  // SSE 心跳
	chatTaskMaxRunning  = 8                 // 同时进行的任务上限
	chatTaskStaleReason = "服务重启导致任务中断，请重试"
)

type chatTaskChunk struct {
	seq  int
	data string
}

// chatTaskBuffer 单任务的内存态：SSE 片段（供续传）+ 累积正文（供落库）。
type chatTaskBuffer struct {
	id string

	mu        sync.Mutex
	seq       int
	chunks    []chatTaskChunk
	bytes     int
	truncated bool
	done      bool
	status    string
	errMsg    string
	content   strings.Builder
	reasoning strings.Builder
	pending   string // 跨 Write 的不完整 SSE 行
	cancel    context.CancelFunc
	subs      map[chan struct{}]struct{}
	updatedAt time.Time
}

func newChatTaskBuffer(id string) *chatTaskBuffer {
	return &chatTaskBuffer{
		id:        id,
		status:    "running",
		subs:      make(map[chan struct{}]struct{}),
		updatedAt: time.Now(),
	}
}

// appendChunk 记录一段 SSE 原文，并同步累积正文（跨片段的半行会被拼起来再解析）。
func (b *chatTaskBuffer) appendChunk(data string) {
	b.mu.Lock()
	if !b.truncated {
		b.seq++
		b.chunks = append(b.chunks, chatTaskChunk{seq: b.seq, data: data})
		b.bytes += len(data)
		if b.bytes > chatTaskMaxBytes {
			// 超出上限后不再缓存原文（续传只能拿到前 1MB），但正文继续累积，终态不丢
			b.truncated = true
		}
	}
	b.feedLocked(data)
	b.updatedAt = time.Now()
	subs := b.subsLocked()
	b.mu.Unlock()
	notifySubs(subs)
}

func (b *chatTaskBuffer) feedLocked(data string) {
	b.pending += data
	for {
		idx := strings.IndexByte(b.pending, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimSpace(b.pending[:idx])
		b.pending = b.pending[idx+1:]
		accumulateChatSSELine(line, &b.content, &b.reasoning)
	}
	// 防止上游一直不发换行把内存撑爆
	if len(b.pending) > 1<<20 {
		b.pending = ""
	}
}

// accumulateChatSSELine 从一行 SSE 里抽出增量文本。只认 OpenAI 兼容 chunk。
func accumulateChatSSELine(line string, content, reasoning *strings.Builder) {
	if !strings.HasPrefix(line, "data:") {
		return
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "" || payload == "[DONE]" {
		return
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content          string `json:"content"`
				Reasoning        string `json:"reasoning"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(payload), &chunk) != nil {
		return
	}
	for _, ch := range chunk.Choices {
		content.WriteString(ch.Delta.Content)
		reasoning.WriteString(ch.Delta.Reasoning)
		reasoning.WriteString(ch.Delta.ReasoningContent)
	}
}

func (b *chatTaskBuffer) subsLocked() []chan struct{} {
	out := make([]chan struct{}, 0, len(b.subs))
	for ch := range b.subs {
		out = append(out, ch)
	}
	return out
}

func notifySubs(subs []chan struct{}) {
	for _, ch := range subs {
		select {
		case ch <- struct{}{}:
		default: // 已经有一个待处理信号，无需堆积
		}
	}
}

func (b *chatTaskBuffer) chunksSince(from int) []chatTaskChunk {
	b.mu.Lock()
	defer b.mu.Unlock()
	if from >= b.seq || len(b.chunks) == 0 {
		return nil
	}
	// chunks 的 seq 单调递增，二分找到第一个 > from
	lo, hi := 0, len(b.chunks)
	for lo < hi {
		mid := (lo + hi) / 2
		if b.chunks[mid].seq <= from {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	out := make([]chatTaskChunk, len(b.chunks)-lo)
	copy(out, b.chunks[lo:])
	return out
}

type chatTaskSnapshot struct {
	Status    string
	Seq       int
	Content   string
	Reasoning string
	ErrMsg    string
	Done      bool
}

func (b *chatTaskBuffer) snapshot() chatTaskSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return chatTaskSnapshot{
		Status:    b.status,
		Seq:       b.seq,
		Content:   b.content.String(),
		Reasoning: b.reasoning.String(),
		ErrMsg:    b.errMsg,
		Done:      b.done,
	}
}

func (b *chatTaskBuffer) finish(status, errMsg string) chatTaskSnapshot {
	b.mu.Lock()
	if !b.done {
		b.done = true
		b.status = status
		b.errMsg = errMsg
	}
	b.updatedAt = time.Now()
	subs := b.subsLocked()
	snap := chatTaskSnapshot{
		Status:    b.status,
		Seq:       b.seq,
		Content:   b.content.String(),
		Reasoning: b.reasoning.String(),
		ErrMsg:    b.errMsg,
		Done:      true,
	}
	b.mu.Unlock()
	notifySubs(subs)
	return snap
}

func (b *chatTaskBuffer) subscribe() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
	}
}

func (b *chatTaskBuffer) setCancel(cancel context.CancelFunc) {
	b.mu.Lock()
	b.cancel = cancel
	b.mu.Unlock()
}

func (b *chatTaskBuffer) requestCancel() bool {
	b.mu.Lock()
	cancel := b.cancel
	done := b.done
	b.mu.Unlock()
	if done || cancel == nil {
		return false
	}
	cancel()
	return true
}

// chatTaskStore 进程级任务表。
type chatTaskStore struct {
	mu    sync.Mutex
	items map[string]*chatTaskBuffer
}

var chatTasks = &chatTaskStore{items: make(map[string]*chatTaskBuffer)}

func (s *chatTaskStore) create(id string) *chatTaskBuffer {
	buf := newChatTaskBuffer(id)
	s.mu.Lock()
	s.items[id] = buf
	s.gcLocked()
	s.mu.Unlock()
	return buf
}

func (s *chatTaskStore) get(id string) (*chatTaskBuffer, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buf, ok := s.items[id]
	return buf, ok
}

// runningCount 统计仍在跑的任务数，用于限制并发（防止弱网重试风暴把上游/内存打满）。
func (s *chatTaskStore) runningCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, buf := range s.items {
		buf.mu.Lock()
		if !buf.done {
			n++
		}
		buf.mu.Unlock()
	}
	return n
}

// gcLocked 清理"已完成且超过 TTL"或过多的任务缓冲。调用方需持锁。
func (s *chatTaskStore) gcLocked() {
	cutoff := time.Now().Add(-chatTaskBufferTTL)
	for id, buf := range s.items {
		buf.mu.Lock()
		done, updated := buf.done, buf.updatedAt
		buf.mu.Unlock()
		if done && updated.Before(cutoff) {
			delete(s.items, id)
		}
	}
	if len(s.items) <= chatTaskMaxBuffers {
		return
	}
	// 仍然超限：按更新时间淘汰最老的已完成任务
	type pair struct {
		id string
		at time.Time
	}
	finished := make([]pair, 0, len(s.items))
	for id, buf := range s.items {
		buf.mu.Lock()
		done, updated := buf.done, buf.updatedAt
		buf.mu.Unlock()
		if done {
			finished = append(finished, pair{id, updated})
		}
	}
	for len(s.items) > chatTaskMaxBuffers && len(finished) > 0 {
		oldest := 0
		for i := range finished {
			if finished[i].at.Before(finished[oldest].at) {
				oldest = i
			}
		}
		delete(s.items, finished[oldest].id)
		finished = append(finished[:oldest], finished[oldest+1:]...)
	}
}

// ── 任务写入通道（把流式 handler 的输出接到内存缓冲）─────────────────────────

// chatTaskSink 实现 gin.ResponseWriter，把流式内容"写"进任务缓冲而不是网络连接。
// 这样 streamOpenAICompatibleWithClient / streamMiniMaxChat 等既有逻辑一行都不用改。
type chatTaskSink struct {
	buf *chatTaskBuffer

	mu      sync.Mutex
	head    http.Header
	status  int
	size    int
	wrote   bool
	errBody string
	closed  chan bool
}

func newChatTaskSink(buf *chatTaskBuffer) *chatTaskSink {
	return &chatTaskSink{buf: buf, head: make(http.Header), closed: make(chan bool)}
}

func (s *chatTaskSink) Header() http.Header { return s.head }

func (s *chatTaskSink) WriteHeader(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
}

func (s *chatTaskSink) WriteHeaderNow() { s.WriteHeader(http.StatusOK) }

func (s *chatTaskSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	status := s.status
	s.mu.Unlock()

	if status >= 400 {
		// 头阶段就出错的场景：c.JSON 写进来的是错误 JSON，不该当成 SSE 片段回放
		s.mu.Lock()
		if len(s.errBody) < 4096 {
			s.errBody += string(p)
		}
		s.size += len(p)
		s.mu.Unlock()
		return len(p), nil
	}

	s.mu.Lock()
	s.size += len(p)
	s.mu.Unlock()
	s.buf.appendChunk(string(p))
	return len(p), nil
}

func (s *chatTaskSink) WriteString(str string) (int, error) { return s.Write([]byte(str)) }

func (s *chatTaskSink) Status() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.wrote {
		return http.StatusOK
	}
	return s.status
}

func (s *chatTaskSink) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size
}

func (s *chatTaskSink) Written() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wrote
}

func (s *chatTaskSink) Flush() {}

func (s *chatTaskSink) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("任务通道不支持连接劫持")
}

func (s *chatTaskSink) CloseNotify() <-chan bool { return s.closed }

func (s *chatTaskSink) Pusher() http.Pusher { return nil }

func (s *chatTaskSink) errText() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(s.errBody)
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

// chatTaskCreateRequest 在 ChatCompletionRequest 基础上多带一个会话 id（用于刷新后对齐）。
type chatTaskCreateRequest struct {
	ChatCompletionRequest
	ConversationID string `json:"conversation_id"`
}

// ChatPlusTaskCreate 创建异步聊天任务并立刻返回 task_id。
// POST /api/chat-plus/tasks
func (h *AIGatewayHandler) ChatPlusTaskCreate(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	var req chatTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model"})
		return
	}
	if err := h.validateChatPlusModel(req.Model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 并发上限：单机 Ollama 同时跑多个大模型只会互相拖慢，且弱网重试容易堆任务
	if chatTasks.runningCount() >= chatTaskMaxRunning {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "同时进行的生成任务过多，请稍后再试或先停止当前任务"})
		return
	}

	taskID := "ctask_" + utils.GenerateHexKey(12)
	if h.db != nil {
		task := &models.ChatPlusTask{
			ID:             taskID,
			ConversationID: req.ConversationID,
			Model:          req.Model,
			Status:         "running",
		}
		if err := h.db.CreateChatPlusTask(task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败: " + err.Error()})
			return
		}
	}

	buf := chatTasks.create(taskID)
	go h.runChatPlusTask(buf, req.ChatCompletionRequest, req.ConversationID)

	c.JSON(http.StatusAccepted, gin.H{
		"task_id":         taskID,
		"status":          "running",
		"seq":             0,
		"model":           req.Model,
		"conversation_id": req.ConversationID,
	})
}

// runChatPlusTask 在后台跑一次聊天：复用同步流的实现，把输出写进任务缓冲。
func (h *AIGatewayHandler) runChatPlusTask(buf *chatTaskBuffer, req ChatCompletionRequest, convID string) {
	ctx, cancel := context.WithCancel(context.Background())
	buf.setCancel(cancel)
	defer cancel()

	sink := newChatTaskSink(buf)
	c, _ := gin.CreateTestContext(sink)
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "/api/chat-plus/tasks", bytes.NewReader(body))
	if err != nil {
		snap := buf.finish("failed", "任务初始化失败: "+err.Error())
		h.persistChatTaskFinal(buf, snap)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.Request = httpReq

	// 进度落库：每 2s 一次，保证"页面刷新/进程重启"也能看到已生成的部分
	stopPersist := make(chan struct{})
	var persistWG sync.WaitGroup
	persistWG.Add(1)
	go func() {
		defer persistWG.Done()
		ticker := time.NewTicker(chatTaskPersistGap)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				h.persistChatTaskProgress(buf)
			case <-stopPersist:
				return
			}
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[chat-task] %s panic: %v", buf.id, r)
			snap := buf.finish("failed", fmt.Sprintf("服务内部错误: %v", r))
			h.persistChatTaskFinal(buf, snap)
		}
		close(stopPersist)
		persistWG.Wait()
	}()

	h.runChatPlusStream(c, req)

	status := "succeeded"
	errMsg := ""
	if sink.Status() >= 400 {
		status = "failed"
		errMsg = sink.errText()
		if errMsg == "" {
			errMsg = fmt.Sprintf("上游返回错误（HTTP %d）", sink.Status())
		}
	}
	if ctx.Err() != nil && status == "succeeded" {
		// 用户主动停止：内容可能不完整，但要保留已生成部分
		status = "canceled"
		errMsg = "已停止生成"
	}

	snap := buf.finish(status, errMsg)
	h.persistChatTaskFinal(buf, snap)
	log.Printf("[chat-task] %s 结束: status=%s seq=%d 正文 %d 字 %s",
		buf.id, snap.Status, snap.Seq, len([]rune(snap.Content)), errMsg)
}

func (h *AIGatewayHandler) persistChatTaskProgress(buf *chatTaskBuffer) {
	if h.db == nil {
		return
	}
	snap := buf.snapshot()
	if err := h.db.UpdateChatPlusTaskProgress(buf.id, snap.Content, snap.Reasoning, snap.Seq); err != nil {
		log.Printf("[chat-task] %s 进度落库失败: %v", buf.id, err)
	}
}

func (h *AIGatewayHandler) persistChatTaskFinal(buf *chatTaskBuffer, snap chatTaskSnapshot) {
	if h.db == nil {
		return
	}
	if err := h.db.FinishChatPlusTask(buf.id, snap.Status, snap.Content, snap.Reasoning, snap.ErrMsg, snap.Seq); err != nil {
		log.Printf("[chat-task] %s 终态落库失败: %v", buf.id, err)
	}
}

// ChatPlusTaskGet 返回任务当前快照（内存优先，DB 兜底）。
// GET /api/chat-plus/tasks/:id
func (h *AIGatewayHandler) ChatPlusTaskGet(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	taskID := c.Param("id")
	if buf, ok := chatTasks.get(taskID); ok {
		snap := buf.snapshot()
		c.JSON(http.StatusOK, gin.H{
			"task_id":       taskID,
			"status":        snap.Status,
			"seq":           snap.Seq,
			"content":       snap.Content,
			"reasoning":     snap.Reasoning,
			"error_message": snap.ErrMsg,
			"done":          snap.Done,
		})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	task, err := h.db.GetChatPlusTask(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"task_id":         task.ID,
		"conversation_id": task.ConversationID,
		"model":           task.Model,
		"status":          task.Status,
		"seq":             task.Seq,
		"content":         task.Content,
		"reasoning":       task.Reasoning,
		"error_message":   task.ErrorMessage,
		"done":            task.Status != "running",
		"created_at":      task.CreatedAt,
	})
}

// ChatPlusTaskList 按会话列出最近任务，供页面刷新后补全"已完成/失败"的消息。
// GET /api/chat-plus/tasks?conversation_id=xxx&limit=20
func (h *AIGatewayHandler) ChatPlusTaskList(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	convID := strings.TrimSpace(c.Query("conversation_id"))
	if convID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 conversation_id"})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	tasks, err := h.db.ListChatPlusTasks(convID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		// 内存里有更新的快照就用内存的（进度落库有 2s 延迟）
		if buf, ok := chatTasks.get(t.ID); ok {
			snap := buf.snapshot()
			t.Status, t.Seq = snap.Status, snap.Seq
			if snap.Content != "" {
				t.Content = snap.Content
			}
			if snap.Reasoning != "" {
				t.Reasoning = snap.Reasoning
			}
			if snap.ErrMsg != "" {
				t.ErrorMessage = snap.ErrMsg
			}
		}
		out = append(out, gin.H{
			"task_id":         t.ID,
			"conversation_id": t.ConversationID,
			"model":           t.Model,
			"status":          t.Status,
			"seq":             t.Seq,
			"content":         t.Content,
			"reasoning":       t.Reasoning,
			"error_message":   t.ErrorMessage,
			"created_at":      t.CreatedAt,
			"updated_at":      t.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"tasks": out})
}

// ChatPlusTaskStop 取消任务。
// POST /api/chat-plus/tasks/:id/stop
func (h *AIGatewayHandler) ChatPlusTaskStop(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	taskID := c.Param("id")
	buf, ok := chatTasks.get(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	if !buf.requestCancel() {
		c.JSON(http.StatusConflict, gin.H{"error": "任务已结束"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task_id": taskID, "status": "stopping"})
}

// ChatPlusTaskStream 订阅任务输出。?from=<seq> 表示"我已经有到 seq 为止的内容"，
// 服务端从 seq+1 开始回放，再续上实时片段 —— 这就是弱网断线重连不丢字的关键。
// GET /api/chat-plus/tasks/:id/stream?from=0
func (h *AIGatewayHandler) ChatPlusTaskStream(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	taskID := c.Param("id")
	from, _ := strconv.Atoi(c.DefaultQuery("from", "0"))
	if from < 0 {
		from = 0
	}

	buf, ok := chatTasks.get(taskID)
	if !ok {
		// 内存缓冲已清理（或服务重启）：用 DB 快照一次性答复
		h.streamChatTaskFromDB(c, taskID, from)
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "当前服务器不支持流式响应"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	writer := newSSEWriter(c.Writer, flusher)
	ctx := c.Request.Context()

	// 先订阅再回放：否则"回放完 → 订阅前"这段窗口里新增的片段会漏掉
	// （弱网下窗口恰好撞上任务收尾的概率不低，表现为回复缺尾巴却不报错）。
	sub, unsubscribe := buf.subscribe()
	defer unsubscribe()

	cursor := from
	for _, ch := range buf.chunksSince(from) {
		if err := writer.write([]byte("id: " + strconv.Itoa(ch.seq) + "\n" + ch.data)); err != nil {
			return
		}
		cursor = ch.seq
	}

	snap := buf.snapshot()
	if snap.Done {
		h.finishChatTaskStream(writer, snap)
		return
	}

	heartbeat := time.NewTicker(chatTaskHeartbeat)
	defer heartbeat.Stop()
	idle := time.NewTimer(chatTaskStreamIdle)
	defer idle.Stop()

	for {
		select {
		case <-ctx.Done():
			return // 客户端断开：任务照常在后台跑，重连后凭 from 续传
		case <-heartbeat.C:
			if err := writer.write([]byte(": ping\n\n")); err != nil {
				return
			}
		case <-idle.C:
			_ = writer.write([]byte(`data: {"error":{"message":"等待上游超时，任务仍在后台执行，可稍后重连","type":"stream_idle"}}` + "\n\n"))
			return
		case <-sub:
			for _, ch := range buf.chunksSince(cursor) {
				if err := writer.write([]byte("id: " + strconv.Itoa(ch.seq) + "\n" + ch.data)); err != nil {
					return
				}
				cursor = ch.seq
			}
			if s := buf.snapshot(); s.Done {
				h.finishChatTaskStream(writer, s)
				return
			}
			idle.Reset(chatTaskStreamIdle)
		}
	}
}

// finishChatTaskStream 给订阅端收尾：成功补 [DONE]，失败发明确的错误帧。
func (h *AIGatewayHandler) finishChatTaskStream(writer *sseWriter, snap chatTaskSnapshot) {
	switch snap.Status {
	case "failed":
		writeOpenAISSEError(writer, fallbackString(snap.ErrMsg, "生成失败"), "task_failed")
	case "canceled":
		writeOpenAISSEError(writer, fallbackString(snap.ErrMsg, "已停止生成"), "task_canceled")
	}
	_ = writer.write([]byte("data: [DONE]\n\n"))
}

// streamChatTaskFromDB 内存缓冲不存在时的兜底：把 DB 里的终态补成一条完整回复。
func (h *AIGatewayHandler) streamChatTaskFromDB(c *gin.Context, taskID string, from int) {
	if h.db == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	task, err := h.db.GetChatPlusTask(taskID)
	if err != nil || task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "当前服务器不支持流式响应"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	writer := newSSEWriter(c.Writer, flusher)

	if task.Status == "running" {
		// 进程重启后 DB 里残留 running：后台清理会把它标失败，这里先如实告知
		writeOpenAISSEError(writer, chatTaskStaleReason, "task_interrupted")
		_ = writer.write([]byte("data: [DONE]\n\n"))
		return
	}
	if from == 0 && task.Content != "" {
		payload, _ := json.Marshal(gin.H{
			"id":     task.ID,
			"object": "chat.completion.chunk",
			"model":  task.Model,
			"choices": []gin.H{{
				"index": 0,
				"delta": gin.H{"content": task.Content, "reasoning": task.Reasoning},
			}},
		})
		_ = writer.write([]byte("id: 0\ndata: " + string(payload) + "\n\n"))
	}
	if task.Status == "failed" || task.Status == "canceled" {
		writeOpenAISSEError(writer, fallbackString(task.ErrorMessage, "生成失败"), "task_"+task.Status)
	}
	_ = writer.write([]byte("data: [DONE]\n\n"))
}

