package handlers

// 本文件：把局域网 ComfyUI（自建扩散模型，如 Qwen-Image-2.1 GGUF）接进 /ai-chat。
//
// 为什么不能只靠 Ollama：
//   Ollama 底层是 llama.cpp，只封装语言/视觉语言模型的推理，没有 diffusion
//   transformer 与 VAE 解码链路，因此跑不了扩散模型的 .gguf 权重。
//   出图必须由 ComfyUI 完成，本文件负责把它的 HTTP API 包装成三种对外形态：
//
//     1) 聊天页「画图」模式：POST /api/chat-plus/image/generate（立即返回 task_id）
//        + GET  /api/chat-plus/image/tasks/:id/stream（SSE 进度，边说边出）
//     2) 轮询：GET /api/chat-plus/image/tasks/:id
//     3) OpenAI 兼容：POST /api/ai-gateway/v1/images/generations（API Key / 超管）
//
// 一次 1024² 出图在本机 M4 Max 上约 3~5 分钟（25 步），所以全链路都是
// 「提交 → 轮询/推送」，绝不在一个 HTTP 请求里同步等（会被反代 524 或浏览器超时掐断）。
//
// 进度来源有两路：主链路轮询 /history（权威、必达），WebSocket /ws 只用来把
// 采样步数变成百分比（锦上添花，连不上也不影响正确性）。

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ── 常量 ────────────────────────────────────────────────────────────────────

const (
	comfyImageSubDir  = "ai"                    // 生成图落盘子目录：./data/uploads/ai/YYYYMMDD
	comfyImageURLBase = "/api/chat/uploads/ai"  // 对外访问路径（复用 chat 的静态路由）
	comfyMaxPromptLen = 2000                    // 提示词长度上限，防超长文本把编码器打爆
	comfyMaxRefImage  = 10 << 20                // 参考图上限 10MB
	comfyPollInterval = 1500 * time.Millisecond // 主链路轮询间隔
	comfyStreamTick   = 400 * time.Millisecond  // SSE 推送间隔
	comfyTaskTTL      = 2 * time.Hour           // 任务内存保留时长
	// SSE 单连接最长存活。不能设得太短：这个到期事件只影响「这条 SSE」，
	// ComfyUI 那边任务照跑，如果我们在这里写 failed，前端就会看到假失败。
	comfyStreamMaxIdle = 60 * time.Minute
	// 主链路等待窗口（cfg.timeout_sec）到了之后的宽限续等时长。
	// ComfyUI 不受我们 ctx 约束，首帧加载模型/排队时经常在超时后才出图。
	comfyGraceWait     = 30 * time.Minute
	comfySamplerNodeID = "8"              // 工作流里 KSampler 的节点 id（进度百分比据此换算）
	comfySaveNodeID    = "10"             // 工作流里 SaveImage 的节点 id
	comfyImageModel    = "qwen-image-2.1" // 对外暴露的模型名（OpenAI 兼容端点用）
	comfyImageScope    = "media"          // API Key 需要的权限域，与图片理解一致
	comfyMaxBatch      = 1                // ComfyUI 一次一张，n>1 不支持
	comfyViewTimeout   = 60 * time.Second // 拉取成品图的超时
	comfyControlTimout = 30 * time.Second // 提交/查询等控制类请求的超时
	// comfyClientID 提交 /prompt 与订阅 /ws 必须用同一个 client_id。
	// ComfyUI 把执行事件（execution_start / executing / progress）按 client_id 定向投递给
	// 提交者；两边不一致时事件会发给一个没人监听的 sid，表现为「进度永远停在排队中」。
	comfyClientID = "devtools-aichat"
)

// ── 请求 / 任务模型 ─────────────────────────────────────────────────────────

// comfyImageRequest 出图请求。聊天页与 OpenAI 兼容端点共用同一结构，
// 字段做了 OpenAI 风格（size / response_format）与直连风格（width / height）的双向兼容。
type comfyImageRequest struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	Model          string  `json:"model"`           // OpenAI 兼容端点忽略此字段（只暴露一个本地模型）
	Size           string  `json:"size"`            // "1024x1024"（OpenAI 风格）
	Width          int     `json:"width"`           // 直连风格，优先于 size
	Height         int     `json:"height"`          // 直连风格，优先于 size
	Resolution     int     `json:"resolution"`      // 正方形边长，优先于 width/height
	AspectRatio    string  `json:"aspect_ratio"`    // 云端出图专用比例（1:1 / 16:9 / ...），本地 ComfyUI 忽略
	Steps          int     `json:"steps"`           // 0 = 用服务端配置
	CFG            float64 `json:"cfg"`             // 0 = 用服务端配置
	Seed           *int64  `json:"seed"`            // nil = 随机
	Image          string  `json:"image"`           // 参考图（data URL / http URL）；单图模式（edit/inpaint/outpaint/restore）的底图
	ResponseFormat string  `json:"response_format"` // url（默认）| b64_json
	// 新模式 / 多图 / 遮罩
	Mode   string   `json:"mode"`   // t2i | edit | inpaint | outpaint | restore | multi_ref | composite（空则按有无 Image 推断）
	Images []string `json:"images"` // 多图输入（multi_ref / composite）；为空时单图模式退回 Image
	Mask   string   `json:"mask"`   // 遮罩图（data URL / http URL）；inpaint / outpaint 用，白色=要重绘区域
	N      int      `json:"n"`      // 仅支持 1
	// ConversationID / MessageID 只为「任务终态落库」服务：带上它，出图结果（或失败原因）
	// 会由服务端直接写进 chat_plus_messages。前端掉线/页面关掉也能在下次进会话时看到，
	// 不会再出现「ComfyUI 图都出来了、前端却说失败且拿不到」的情况。
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

// comfyImage 一张生成结果。
type comfyImage struct {
	URL      string `json:"url,omitempty"`
	B64      string `json:"b64_json,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Filename string `json:"filename,omitempty"`
	// RelPath 相对 uploadDir 的落盘路径，不回传前端；b64_json 输出时用它反查文件，
	// 避免用 time.Now() 重新拼日期——跨零点出图会拼到错误的目录。
	RelPath string `json:"-"`
}

// comfyTask 一次出图任务的内存快照。所有字段的读写都必须经 comfyTaskStore 加锁。
type comfyTask struct {
	ID        string       `json:"task_id"`
	Status    string       `json:"status"`   // queued | running | succeeded | failed
	Phase     string       `json:"phase"`    // 人类可读的阶段说明
	Progress  int          `json:"progress"` // 0~100，采样阶段由 WebSocket 驱动
	Mode      string       `json:"mode"`     // t2i | edit
	Model     string       `json:"model"`    // 出图模型预设 id（前端下拉框选的那个）
	Prompt    string       `json:"prompt"`
	ImageURL  string       `json:"image_url,omitempty"`
	Images    []comfyImage `json:"images,omitempty"`
	Error     string       `json:"error,omitempty"`
	ElapsedMS int64        `json:"elapsed_ms"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`

	// 以下字段不出现在 JSON 里，仅内部使用。
	promptID string
	started  time.Time
}

// comfyTaskStore 任务内存表。出图是短生命周期操作，结果图本身已落盘，
// 所以任务元数据不需要进 SQLite——重启丢失可接受（前端会收到"任务不存在"）。
type comfyTaskStore struct {
	mu    sync.RWMutex
	items map[string]*comfyTask
}

func newComfyTaskStore() *comfyTaskStore {
	return &comfyTaskStore{items: make(map[string]*comfyTask)}
}

func (s *comfyTaskStore) put(t *comfyTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[t.ID] = t
	s.gcLocked()
}

// mutate 在锁内修改任务并返回一份拷贝，避免调用方读到写一半的结构。
func (s *comfyTaskStore) mutate(id string, fn func(t *comfyTask)) (comfyTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return comfyTask{}, false
	}
	fn(t)
	t.UpdatedAt = time.Now()
	if !t.started.IsZero() && t.Status != "queued" {
		t.ElapsedMS = t.UpdatedAt.Sub(t.started).Milliseconds()
	}
	return *t, true
}

func (s *comfyTaskStore) get(id string) (comfyTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.items[id]
	if !ok {
		return comfyTask{}, false
	}
	snap := *t
	// 排队/上传参考图/首次加载模型这几个阶段可能几分钟没有 mutate，
	// elapsed 在这里实时算，前端「已等待 Ns」才会真的在走。
	if !t.started.IsZero() && t.Status != "queued" && t.Status != "succeeded" && t.Status != "failed" {
		snap.ElapsedMS = time.Since(t.started).Milliseconds()
	}
	return snap, true
}

// gcLocked 清理过期任务。调用方必须已持有写锁。
func (s *comfyTaskStore) gcLocked() {
	if len(s.items) < 32 {
		return
	}
	cutoff := time.Now().Add(-comfyTaskTTL)
	for id, t := range s.items {
		if t.UpdatedAt.Before(cutoff) {
			delete(s.items, id)
		}
	}
}

// comfyTasks 进程级任务表（AIGatewayHandler 是单例，但任务表独立便于单测）。
var comfyTasks = newComfyTaskStore()

// ── ComfyUI HTTP 客户端 ─────────────────────────────────────────────────────

// comfyClient 直连局域网的 ComfyUI。显式禁用代理：容器里常配了 HTTP_PROXY，
// 局域网地址走代理会直接连不上。
// comfyNoProxyClient 下载参考图用：显式禁用代理，只服务于局域网/内网地址。
var comfyNoProxyClient = &http.Client{
	Timeout: comfyControlTimout,
	Transport: &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: comfyControlTimout,
	},
}

type comfyClient struct {
	baseURL string
	http    *http.Client
}

func newComfyClient(cfg config.ComfyUIConfig) *comfyClient {
	transport := &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          16,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: comfyControlTimout,
	}
	return &comfyClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		http:    &http.Client{Transport: transport},
	}
}

// ping 探活：/system_stats 是 ComfyUI 最轻的只读接口。
func (c *comfyClient) ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/system_stats", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ComfyUI 返回 HTTP %d", resp.StatusCode)
	}
	return nil
}

// uploadImage 把参考图塞进 ComfyUI 的 input 目录，返回 LoadImage 可用的名字。
func (c *comfyClient) uploadImage(ctx context.Context, filename string, data []byte, subdir string) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("image", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	_ = mw.WriteField("overwrite", "true")
	if subdir != "" {
		_ = mw.WriteField("subfolder", subdir)
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/upload/image", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("上传参考图失败 HTTP %d: %s", resp.StatusCode, truncateString(string(raw), 200))
	}
	var out struct {
		Name      string `json:"name"`
		Subfolder string `json:"subfolder"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("解析上传响应失败: %w", err)
	}
	if out.Name == "" {
		return "", fmt.Errorf("上传参考图失败: 响应缺少 name")
	}
	if out.Subfolder != "" {
		return out.Subfolder + "/" + out.Name, nil
	}
	return out.Name, nil
}

// queuePrompt 提交工作流，返回 prompt_id。
func (c *comfyClient) queuePrompt(ctx context.Context, workflow map[string]interface{}) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"prompt":    workflow,
		"client_id": comfyClientID,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/prompt", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if resp.StatusCode >= 400 {
		// ComfyUI 会在 body 里给出 node_errors，原样回传对排查工作流最有用
		return "", fmt.Errorf("提交工作流失败 HTTP %d: %s", resp.StatusCode, comfyErrorDetail(raw))
	}
	var out struct {
		PromptID   string          `json:"prompt_id"`
		NodeErrors json.RawMessage `json:"node_errors"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("解析 /prompt 响应失败: %w", err)
	}
	if out.PromptID == "" {
		return "", fmt.Errorf("提交工作流失败: 响应缺少 prompt_id")
	}
	return out.PromptID, nil
}

// history 查询一次执行结果。第二个返回值为 false 表示还没进历史（仍在排队/执行）。
func (c *comfyClient) history(ctx context.Context, promptID string) (*comfyHistoryEntry, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/history/"+url.PathEscape(promptID), nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, false, fmt.Errorf("查询执行结果失败 HTTP %d", resp.StatusCode)
	}
	var all map[string]*comfyHistoryEntry
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&all); err != nil {
		return nil, false, err
	}
	entry, ok := all[promptID]
	if !ok || entry == nil {
		return nil, false, nil
	}
	return entry, true, nil
}

// viewImage 取成品图字节。
func (c *comfyClient) viewImage(ctx context.Context, ref comfyImageRef) ([]byte, error) {
	q := url.Values{}
	q.Set("filename", ref.Filename)
	q.Set("subfolder", ref.Subfolder)
	q.Set("type", comfyOr(ref.Type, "output"))
	ctx, cancel := context.WithTimeout(ctx, comfyViewTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/view?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("下载成品图失败 HTTP %d", resp.StatusCode)
	}
	// 单张 1024² PNG 约 1~2MB，给 32MB 上限足够
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// comfyHistoryEntry /history 里单次执行的结构（只声明用到的字段）。
type comfyHistoryEntry struct {
	Outputs map[string]struct {
		Images []comfyImageRef `json:"images"`
	} `json:"outputs"`
	Status struct {
		StatusStr string            `json:"status_str"`
		Completed bool              `json:"completed"`
		Messages  []json.RawMessage `json:"messages"`
	} `json:"status"`
}

// comfyImageRef ComfyUI 输出的图片引用。
type comfyImageRef struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

// watchProgress 用 ComfyUI 的 /ws 把采样进度换算成百分比。
// 纯增强：连不上、解析失败、中途断开都只是少个百分比，不影响任务正确性，因此统一吞错。
func (c *comfyClient) watchProgress(ctx context.Context, promptID string, report func(progress int, phase string)) {
	if c.baseURL == "" {
		return
	}
	wsURL := c.baseURL
	if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}
	wsURL += "/ws?clientId=" + comfyClientID

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		// 显式禁用代理，与 HTTP 客户端保持一致
		Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		log.Printf("[comfy] 连接进度通道失败（不影响出图）: %v", err)
		return
	}
	defer conn.Close()

	// ctx 结束（任务完成/超时/取消）时主动断开，让下面的 ReadMessage 立刻返回
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()

	for {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType != websocket.TextMessage {
			continue // 预览图等二进制帧直接跳过
		}
		var msg struct {
			Type string `json:"type"`
			Data struct {
				PromptID string `json:"prompt_id"`
				Node     string `json:"node"`
				Value    int    `json:"value"`
				Max      int    `json:"max"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		// 只认自己的任务：同一个 ComfyUI 可能同时在跑别人的活
		if msg.Data.PromptID != "" && msg.Data.PromptID != promptID {
			continue
		}
		switch msg.Type {
		case "execution_start":
			report(2, "开始执行")
		case "executing":
			if msg.Data.Node == comfySamplerNodeID {
				report(5, "采样中")
			} else {
				report(3, "准备中")
			}
		case "progress":
			if msg.Data.Max > 0 {
				// 采样占整体耗时的绝大部分，映射到 5~95 区间
				pct := 5 + int(float64(msg.Data.Value)/float64(msg.Data.Max)*90)
				if pct < 5 {
					pct = 5
				}
				if pct > 95 {
					pct = 95
				}
				report(pct, fmt.Sprintf("采样中 %d/%d", msg.Data.Value, msg.Data.Max))
			}
		case "execution_error":
			// 具体错误由主链路从 /history 里取，这里只更新阶段文案
			report(95, "执行出错")
			return
		case "execution_success":
			report(98, "解码出图")
			return
		}
	}
}

// comfyOr 取非空值，或回退到默认值。
func comfyOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// comfyErrorDetail 从 ComfyUI 的错误响应里抽出可读信息。
func comfyErrorDetail(raw []byte) string {
	var out struct {
		Error      json.RawMessage            `json:"error"`
		NodeErrors map[string]json.RawMessage `json:"node_errors"`
	}
	if json.Unmarshal(raw, &out) == nil {
		var parts []string
		if len(out.Error) > 0 {
			var e struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}
			if json.Unmarshal(out.Error, &e) == nil && e.Message != "" {
				parts = append(parts, e.Message)
			} else {
				parts = append(parts, truncateString(string(out.Error), 300))
			}
		}
		for node, detail := range out.NodeErrors {
			parts = append(parts, fmt.Sprintf("节点 %s: %s", node, truncateString(string(detail), 300)))
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}
	return truncateString(strings.TrimSpace(string(raw)), 400)
}

// comfyExecutionError 从已完成的 /history 条目里抽执行期异常。
func comfyExecutionError(entry *comfyHistoryEntry) string {
	interrupted := false
	for _, raw := range entry.Status.Messages {
		var pair []json.RawMessage
		if json.Unmarshal(raw, &pair) != nil || len(pair) < 2 {
			continue
		}
		var kind string
		if json.Unmarshal(pair[0], &kind) != nil {
			continue
		}
		// 被取消（ComfyUI 侧 interrupt）不会有 exception_message，只有这条
		if kind == "execution_interrupted" {
			interrupted = true
			continue
		}
		if kind != "execution_error" {
			continue
		}
		var detail struct {
			NodeID           string `json:"node_id"`
			NodeType         string `json:"node_type"`
			ExceptionMessage string `json:"exception_message"`
		}
		if json.Unmarshal(pair[1], &detail) == nil && detail.ExceptionMessage != "" {
			return fmt.Sprintf("节点 %s(%s) 执行失败: %s",
				comfyOr(detail.NodeID, "?"), comfyOr(detail.NodeType, "?"),
				truncateString(detail.ExceptionMessage, 300))
		}
	}
	if interrupted {
		return "任务被中断：ComfyUI 侧被取消（可能是手动 interrupt 或 ComfyUI 重启）"
	}
	return "ComfyUI 执行失败（未返回详细原因）"
}

// ── 工作流构造（纯函数，便于单测） ──────────────────────────────────────────

// comfyGenOptions 一次生成的可调参数（已由请求 + 服务端配置合并）。
type comfyGenOptions struct {
	Steps      int
	CFG        float64
	Seed       int64
	Resolution int
	ImageName  string // 非空 → 编辑模式，需要 LoadImage 节点
	// 新模式
	Mode          string   // 出图模式，决定走哪条工作流分支
	ImageNames    []string // 多图 / 底图上传后在 ComfyUI 里的文件名（multi_ref / composite / 单图底图）
	MaskName      string   // 遮罩上传后的文件名（inpaint / outpaint）
	CompositeCell int      // 拼合每格边长（composite 用，0 = 默认 512）
}

// buildComfyWorkflow 生成 ComfyUI API 格式工作流。
//
// 节点布局沿用官方 Qwen-Image-2.1 模板：
//
//	1 UnetLoaderGGUF  2 CLIPLoader  3 VAELoader
//	5 TextEncodeQwenImage21（同时产出正向、负向、latent 三路）
//	8 KSampler  9 VAEDecode  10 SaveImage
//	4 LoadImage（仅编辑模式）
//
// 注意：编辑模式的参考图必须写成扁平键名 "images.image_1"。唐 该节点的 images 输入是
// Autogrow 类型，写成嵌套 dict 会被执行器静默丢弃，表现为"图没接上、模型凭空生成"。
func buildComfyWorkflow(cfg config.ComfyUIConfig, prompt, negative string, o comfyGenOptions) map[string]interface{} {
	// 新模式按 Mode 优先分发（与家族无关或跨家族）：
	//   composite   —— 纯图像拼合，不需要加载扩散模型；
	//   multi_ref   —— 多图内容结合，走 qwen21 编码器的多图 autogrow 输入（cfg 已被切成 qwen21 预设）；
	//   inpaint/outpaint/restore —— qwen_image_edit 家族的遮罩重绘 / 修复（cfg 已被切成 qwen_edit 预设）。
	switch o.Mode {
	case "composite":
		return buildCompositeWorkflow(o)
	case "multi_ref":
		return buildMultiRefWorkflow(cfg, prompt, negative, o)
	case "inpaint", "outpaint", "restore":
		return buildQwenEditVariantWorkflow(cfg, prompt, negative, o)
	}
	switch cfg.Family {
	case comfyFamilyFlux2:
		return buildFlux2Workflow(cfg, prompt, negative, o)
	case comfyFamilyQwenEdit:
		return buildQwenEditWorkflow(cfg, prompt, negative, o)
	case comfyFamilyZImage:
		return buildZImageWorkflow(cfg, prompt, negative, o)
	}
	// TextEncodeQwenImage21 只吃一个 resolution，内部自行推导 latent 尺寸，
	// 所以这里不单独传宽高（非正方形需求见 comfyResolveParams 的短边收敛）。
	encoder := map[string]interface{}{
		"clip":            []interface{}{"2", 0},
		"vae":             []interface{}{"3", 0},
		"prompt":          prompt,
		"negative_prompt": negative,
		"resolution":      o.Resolution,
	}
	workflow := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "UnetLoaderGGUF",
			"inputs":     map[string]interface{}{"unet_name": cfg.Model},
		},
		"2": comfyClipLoaderNode(cfg),
		"3": map[string]interface{}{
			"class_type": "VAELoader",
			"inputs":     map[string]interface{}{"vae_name": cfg.Vae},
		},
		"5": map[string]interface{}{
			"class_type": "TextEncodeQwenImage21",
			"inputs":     encoder,
		},
		"8": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"model":        []interface{}{comfyModelNodeID(cfg), 0},
				"positive":     []interface{}{"5", 0},
				"negative":     []interface{}{"5", 1},
				"latent_image": []interface{}{"5", 2},
				"seed":         o.Seed,
				"steps":        o.Steps,
				"cfg":          o.CFG,
				"sampler_name": cfg.Sampler,
				"scheduler":    cfg.Scheduler,
				"denoise":      1.0,
			},
		},
		"9": map[string]interface{}{
			"class_type": "VAEDecode",
			"inputs": map[string]interface{}{
				"samples": []interface{}{"8", 0},
				"vae":     []interface{}{"3", 0},
			},
		},
		"10": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs": map[string]interface{}{
				"images":          []interface{}{"9", 0},
				"filename_prefix": "aichat",
			},
		},
	}
	comfyAttachLora(workflow, cfg)
	if o.ImageName != "" {
		workflow["4"] = map[string]interface{}{
			"class_type": "LoadImage",
			"inputs":     map[string]interface{}{"image": o.ImageName},
		}
		// Autogrow 输入的扁平键名，见函数注释
		encoder["images.image_1"] = []interface{}{"4", 0}
	}
	return workflow
}

// comfyModelNodeID 决定 KSampler 的 model 从哪个节点取：
// 挂了蒸馏 LoRA 就先过 LoRA 节点（11），否则直接用 unet（1）。
func comfyModelNodeID(cfg config.ComfyUIConfig) string {
	if strings.TrimSpace(cfg.Lora) != "" {
		return "11"
	}
	return "1"
}

// comfyAttachLora 把 LoRA 节点补进工作流。
// 只用 LoraLoaderModelOnly：CLIP/VAE 不需要 LoRA 改动（turbo LoRA 只训练了 unet），
// 用 LoraLoader 反而会去改文本编码器，可能让中文理解变差。
func comfyAttachLora(workflow map[string]interface{}, cfg config.ComfyUIConfig) {
	if comfyModelNodeID(cfg) != "11" {
		return
	}
	strength := cfg.LoraStrength
	if strength == 0 {
		strength = 1.0
	}
	workflow["11"] = map[string]interface{}{
		"class_type": "LoraLoaderModelOnly",
		"inputs": map[string]interface{}{
			"model":          []interface{}{"1", 0},
			"lora_name":      cfg.Lora,
			"strength_model": strength,
		},
	}
}

// comfyClipLoaderNode 按配置选 CLIPLoader（safetensors）或 CLIPLoaderGGUF（.gguf）。
// 两者的输入名一致，只有 device 这个可选字段仅前者支持。
func comfyClipLoaderNode(cfg config.ComfyUIConfig) map[string]interface{} {
	inputs := map[string]interface{}{
		"clip_name": cfg.Clip,
		"type":      cfg.ClipType,
	}
	if cfg.ClipLoader == "CLIPLoaderGGUF" {
		return map[string]interface{}{"class_type": "CLIPLoaderGGUF", "inputs": inputs}
	}
	inputs["device"] = "default"
	return map[string]interface{}{"class_type": "CLIPLoader", "inputs": inputs}
}

// buildFlux2Workflow 生成 FLUX.2（Klein 等）的工作流。
//
//	1 UnetLoaderGGUF  2 CLIPLoaderGGUF  3 VAELoader
//	4 CLIPTextEncode（正向） 6 CLIPTextEncode（负向） 5 EmptyFlux2LatentImage
//	8 KSampler  9 VAEDecode  10 SaveImage
//
// 与 qwen21 的差别：正向/负向分开两个节点，latent 由 EmptyFlux2LatentImage 直接给
// 宽高（不走 resolution 收敛）。节点号沿用 8=KSampler / 10=SaveImage 的约定，
// watchProgress 的进度百分比映射才继续有效。
func buildFlux2Workflow(cfg config.ComfyUIConfig, prompt, negative string, o comfyGenOptions) map[string]interface{} {
	side := o.Resolution
	if side < 256 {
		side = 256
	}
	workflow := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "UnetLoaderGGUF",
			"inputs":     map[string]interface{}{"unet_name": cfg.Model},
		},
		"2": comfyClipLoaderNode(cfg),
		"3": map[string]interface{}{
			"class_type": "VAELoader",
			"inputs":     map[string]interface{}{"vae_name": cfg.Vae},
		},
		"4": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]interface{}{"clip": []interface{}{"2", 0}, "text": prompt},
		},
		"6": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]interface{}{"clip": []interface{}{"2", 0}, "text": negative},
		},
		"5": map[string]interface{}{
			"class_type": "EmptyFlux2LatentImage",
			"inputs": map[string]interface{}{
				"width": side, "height": side, "batch_size": 1,
			},
		},
		"8": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"model":        []interface{}{comfyModelNodeID(cfg), 0},
				"positive":     []interface{}{"4", 0},
				"negative":     []interface{}{"6", 0},
				"latent_image": []interface{}{"5", 0},
				"seed":         o.Seed,
				"steps":        o.Steps,
				"cfg":          o.CFG,
				"sampler_name": cfg.Sampler,
				"scheduler":    cfg.Scheduler,
				"denoise":      1.0,
			},
		},
		"9": map[string]interface{}{
			"class_type": "VAEDecode",
			"inputs":     map[string]interface{}{"samples": []interface{}{"8", 0}, "vae": []interface{}{"3", 0}},
		},
		"10": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{"images": []interface{}{"9", 0}, "filename_prefix": "aichat"},
		},
	}
	comfyAttachLora(workflow, cfg)
	return workflow
}

// buildZImageWorkflow 生成 Z-Image Turbo 的工作流。
//
//	1 UnetLoaderGGUF  2 CLIPLoader(lumina2)  3 VAELoader(Flux 的 ae)
//	7 ModelSamplingAuraFlow(shift=3)  4/6 CLIPTextEncode 正负
//	5 EmptySD3LatentImage  8 KSampler  9 VAEDecode  10 SaveImage
//
// 两个必须遵守的点（都是实测踩出来的）：
//   - **ClipLoader 必须是 CLIPLoader**：同一份 qwen_3_4b safetensors 用 CLIPLoaderGGUF
//     加载会在 CLIPTextEncode 里报 "mat1 and mat2 shapes cannot be multiplied
//     (8x2560 and 1280x4096)"，GGUF loader 走 lumina2 分支时维度解析不对。
//   - 模型必须过一次 ModelSamplingAuraFlow(shift=3)，这是 Z-Image 的采样轨迹要求，
//     少了要么糊要么报错。cfg 固定 1.0（蒸馏模型，负向提示词基本无效）。
func buildZImageWorkflow(cfg config.ComfyUIConfig, prompt, negative string, o comfyGenOptions) map[string]interface{} {
	side := o.Resolution
	if side < 256 {
		side = 256
	}
	shift := cfg.Shift
	if shift <= 0 {
		shift = 3.0
	}
	workflow := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "UnetLoaderGGUF",
			"inputs":     map[string]interface{}{"unet_name": cfg.Model},
		},
		"2": comfyClipLoaderNode(cfg),
		"3": map[string]interface{}{
			"class_type": "VAELoader",
			"inputs":     map[string]interface{}{"vae_name": cfg.Vae},
		},
		"4": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]interface{}{"clip": []interface{}{"2", 0}, "text": prompt},
		},
		"6": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]interface{}{"clip": []interface{}{"2", 0}, "text": negative},
		},
		"7": map[string]interface{}{
			"class_type": "ModelSamplingAuraFlow",
			"inputs": map[string]interface{}{
				"model": []interface{}{comfyModelNodeID(cfg), 0},
				"shift": shift,
			},
		},
		"5": map[string]interface{}{
			"class_type": "EmptySD3LatentImage",
			"inputs": map[string]interface{}{
				"width": side, "height": side, "batch_size": 1,
			},
		},
		"8": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"model":        []interface{}{"7", 0},
				"positive":     []interface{}{"4", 0},
				"negative":     []interface{}{"6", 0},
				"latent_image": []interface{}{"5", 0},
				"seed":         o.Seed,
				"steps":        o.Steps,
				"cfg":          o.CFG,
				"sampler_name": cfg.Sampler,
				"scheduler":    cfg.Scheduler,
				"denoise":      1.0,
			},
		},
		"9": map[string]interface{}{
			"class_type": "VAEDecode",
			"inputs":     map[string]interface{}{"samples": []interface{}{"8", 0}, "vae": []interface{}{"3", 0}},
		},
		"10": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{"images": []interface{}{"9", 0}, "filename_prefix": "aichat"},
		},
	}
	comfyAttachLora(workflow, cfg)
	return workflow
}

// buildQwenEditWorkflow 生成 Qwen-Image-Edit（如 2511）专用的编辑工作流。
//
//	1 UnetLoaderGGUF  2 CLIPLoader(GGUF)  3 VAELoader  4 LoadImage
//	5 TextEncodeQwenImageEditPlus（吃参考图出条件） 6 CLIPTextEncode（负向）
//	7 VAEEncode（把参考图编码成 latent，编辑的起点） 8 KSampler  9 VAEDecode  10 SaveImage
//
// 与 qwen21 的差别：Edit 家族没有 TextEncodeQwenImage21 这种「一个节点出正负+latent」的
// 编码器，latent 必须自己 VAEEncode 源图；且文本编码器维度必须匹配（Qwen2.5-VL 3584 维），
// 用错会直接报 "normalized_shape" 不匹配。
func buildQwenEditWorkflow(cfg config.ComfyUIConfig, prompt, negative string, o comfyGenOptions) map[string]interface{} {
	workflow := map[string]interface{}{
		"1": map[string]interface{}{
			"class_type": "UnetLoaderGGUF",
			"inputs":     map[string]interface{}{"unet_name": cfg.Model},
		},
		"2": comfyClipLoaderNode(cfg),
		"3": map[string]interface{}{
			"class_type": "VAELoader",
			"inputs":     map[string]interface{}{"vae_name": cfg.Vae},
		},
		"6": map[string]interface{}{
			"class_type": "CLIPTextEncode",
			"inputs":     map[string]interface{}{"clip": []interface{}{"2", 0}, "text": negative},
		},
		"8": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"model":        []interface{}{comfyModelNodeID(cfg), 0},
				"positive":     []interface{}{"5", 0},
				"negative":     []interface{}{"6", 0},
				"latent_image": []interface{}{"7", 0},
				"seed":         o.Seed,
				"steps":        o.Steps,
				"cfg":          o.CFG,
				"sampler_name": cfg.Sampler,
				"scheduler":    cfg.Scheduler,
				"denoise":      1.0,
			},
		},
		"9": map[string]interface{}{
			"class_type": "VAEDecode",
			"inputs":     map[string]interface{}{"samples": []interface{}{"8", 0}, "vae": []interface{}{"3", 0}},
		},
		"10": map[string]interface{}{
			"class_type": "SaveImage",
			"inputs":     map[string]interface{}{"images": []interface{}{"9", 0}, "filename_prefix": "aichat"},
		},
	}
	comfyAttachLora(workflow, cfg)
	if o.ImageName == "" {
		return workflow // 缺参考图时由调用方拦在前面，这里保持图结构不残缺即可
	}
	workflow["4"] = map[string]interface{}{
		"class_type": "LoadImage",
		"inputs":     map[string]interface{}{"image": o.ImageName},
	}
	workflow["5"] = map[string]interface{}{
		"class_type": "TextEncodeQwenImageEditPlus",
		"inputs": map[string]interface{}{
			"clip":   []interface{}{"2", 0},
			"vae":    []interface{}{"3", 0},
			"prompt": prompt,
			"image1": []interface{}{"4", 0},
		},
	}
	workflow["7"] = map[string]interface{}{
		"class_type": "VAEEncode",
		"inputs":     map[string]interface{}{"pixels": []interface{}{"4", 0}, "vae": []interface{}{"3", 0}},
	}
	return workflow
}

// buildQwenEditVariantWorkflow 在 Qwen-Image-Edit 家族基础上，按模式叠加遮罩重绘或纯修复。
//
//	1 UnetLoaderGGUF  2 CLIPLoader(GGUF)  3 VAELoader  4 LoadImage(底图)  [4m] LoadImage(遮罩)
//	5 TextEncodeQwenImageEditPlus  6 CLIPTextEncode(负向)
//	7 VAEEncode（restore）或 VAEEncodeForInpaint（inpaint/outpaint，吃遮罩 + grow_mask_by）
//	8 KSampler  9 VAEDecode  10 SaveImage
//
// outpaint 在后端与 inpaint 同构：都靠遮罩圈出要重绘的区域（外补时遮罩是四周边框），
// 区别只在前端给的遮罩形状与提示词。restore 则不传遮罩、对整个源图做提示词驱动的重绘/增强。
func buildQwenEditVariantWorkflow(cfg config.ComfyUIConfig, prompt, negative string, o comfyGenOptions) map[string]interface{} {
	workflow := map[string]interface{}{
		"1": map[string]interface{}{"class_type": "UnetLoaderGGUF", "inputs": map[string]interface{}{"unet_name": cfg.Model}},
		"2": comfyClipLoaderNode(cfg),
		"3": map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": cfg.Vae}},
		"6": map[string]interface{}{"class_type": "CLIPTextEncode", "inputs": map[string]interface{}{"clip": []interface{}{"2", 0}, "text": negative}},
		"8": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"model":        []interface{}{comfyModelNodeID(cfg), 0},
				"positive":     []interface{}{"5", 0},
				"negative":     []interface{}{"6", 0},
				"latent_image": []interface{}{"7", 0},
				"seed":         o.Seed,
				"steps":        o.Steps,
				"cfg":          o.CFG,
				"sampler_name": cfg.Sampler,
				"scheduler":    cfg.Scheduler,
				"denoise":      1.0,
			},
		},
		"9":  map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"8", 0}, "vae": []interface{}{"3", 0}}},
		"10": map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"images": []interface{}{"9", 0}, "filename_prefix": "aichat"}},
	}
	comfyAttachLora(workflow, cfg)
	if o.ImageName == "" {
		return workflow // 缺底图由调用方拦在前面
	}
	workflow["4"] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": o.ImageName}}
	workflow["5"] = map[string]interface{}{
		"class_type": "TextEncodeQwenImageEditPlus",
		"inputs": map[string]interface{}{
			"clip":   []interface{}{"2", 0},
			"vae":    []interface{}{"3", 0},
			"prompt": prompt,
			"image1": []interface{}{"4", 0},
		},
	}
	if o.Mode == "inpaint" || o.Mode == "outpaint" {
		// 遮罩重绘：VAEEncodeForInpaint 把遮罩区域标记为待重绘（grow_mask_by 让边缘过渡自然）。
		// 实测该节点强制要求 grow_mask_by 输入，缺了会 400 校验失败。
		workflow["4m"] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": o.MaskName}}
		workflow["7"] = map[string]interface{}{
			"class_type": "VAEEncodeForInpaint",
			"inputs": map[string]interface{}{
				"pixels":       []interface{}{"4", 0},
				"vae":          []interface{}{"3", 0},
				"mask":         []interface{}{"4m", 1},
				"grow_mask_by": 6,
			},
		}
	} else {
		// restore：对整个源图 VAEEncode，由提示词驱动增强/修复。
		workflow["7"] = map[string]interface{}{
			"class_type": "VAEEncode",
			"inputs":     map[string]interface{}{"pixels": []interface{}{"4", 0}, "vae": []interface{}{"3", 0}},
		}
	}
	return workflow
}

// buildMultiRefWorkflow 多图内容结合：走 Qwen-Image-2.1 编码器的多图 autogrow 输入
// （images.image_1 / images.image_2 / ...）。该编码器原生支持多张参考图，模型据此把多张图
// 的内容/风格融合成一张图。cfg 必须已是 qwen21 家族预设（调用方已按模式切好）。
func buildMultiRefWorkflow(cfg config.ComfyUIConfig, prompt, negative string, o comfyGenOptions) map[string]interface{} {
	side := o.Resolution
	if side < 256 {
		side = 256
	}
	encoder := map[string]interface{}{
		"clip":            []interface{}{"2", 0},
		"vae":             []interface{}{"3", 0},
		"prompt":          prompt,
		"negative_prompt": negative,
		"resolution":      side,
	}
	workflow := map[string]interface{}{
		"1": map[string]interface{}{"class_type": "UnetLoaderGGUF", "inputs": map[string]interface{}{"unet_name": cfg.Model}},
		"2": comfyClipLoaderNode(cfg),
		"3": map[string]interface{}{"class_type": "VAELoader", "inputs": map[string]interface{}{"vae_name": cfg.Vae}},
		"8": map[string]interface{}{
			"class_type": "KSampler",
			"inputs": map[string]interface{}{
				"model":        []interface{}{comfyModelNodeID(cfg), 0},
				"positive":     []interface{}{"5", 0},
				"negative":     []interface{}{"5", 1},
				"latent_image": []interface{}{"5", 2},
				"seed":         o.Seed,
				"steps":        o.Steps,
				"cfg":          o.CFG,
				"sampler_name": cfg.Sampler,
				"scheduler":    cfg.Scheduler,
				"denoise":      1.0,
			},
		},
		"9":  map[string]interface{}{"class_type": "VAEDecode", "inputs": map[string]interface{}{"samples": []interface{}{"8", 0}, "vae": []interface{}{"3", 0}}},
		"10": map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"images": []interface{}{"9", 0}, "filename_prefix": "aichat"}},
	}
	comfyAttachLora(workflow, cfg)
	// 多张参考图：每个 LoadImage 一个节点，编码器用扁平键名 images.image_N 接入（Autogrow 不能写嵌套 dict）。
	for i, name := range o.ImageNames {
		id := fmt.Sprintf("L%d", i)
		workflow[id] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": name}}
		encoder[fmt.Sprintf("images.image_%d", i+1)] = []interface{}{id, 0}
	}
	workflow["5"] = map[string]interface{}{"class_type": "TextEncodeQwenImage21", "inputs": encoder}
	return workflow
}

// buildCompositeWorkflow 多图拼合 / 图层：把多张图缩放成统一格子，按网格贴到一张画布上。
// 纯图像运算，不加载扩散模型（秒级出图）。先以第 0 张铺满画布作底（会被后续逐格覆盖，不留缝），
// 再用 SolidMask + ImageCompositeMasked 把每张图贴到各自格子。
func buildCompositeWorkflow(o comfyGenOptions) map[string]interface{} {
	names := o.ImageNames
	if len(names) == 0 && o.ImageName != "" {
		names = []string{o.ImageName}
	}
	if len(names) == 0 {
		return map[string]interface{}{}
	}
	cell := o.CompositeCell
	if cell < 64 {
		cell = 512
	}
	n := len(names)
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	if cols < 1 {
		cols = 1
	}
	rows := (n + cols - 1) / cols
	canvasW := cols * cell
	canvasH := rows * cell

	wf := map[string]interface{}{}
	for i, name := range names {
		li := fmt.Sprintf("L%d", i)
		si := fmt.Sprintf("S%d", i)
		mi := fmt.Sprintf("M%d", i)
		wf[li] = map[string]interface{}{"class_type": "LoadImage", "inputs": map[string]interface{}{"image": name}}
		wf[si] = map[string]interface{}{
			"class_type": "ImageScale",
			"inputs":     map[string]interface{}{"image": []interface{}{li, 0}, "width": cell, "height": cell, "upscale_method": "bilinear", "crop": "center"},
		}
		wf[mi] = map[string]interface{}{"class_type": "SolidMask", "inputs": map[string]interface{}{"width": cell, "height": cell, "value": 1.0}}
	}
	// 底图：第 0 张铺满整张画布（拉伸，仅作背景，随后被逐格覆盖）
	wf["B0"] = map[string]interface{}{
		"class_type": "ImageScale",
		"inputs":     map[string]interface{}{"image": []interface{}{"L0", 0}, "width": canvasW, "height": canvasH, "upscale_method": "bilinear", "crop": "center"},
	}
	prev := "B0"
	for i := 0; i < n; i++ {
		col := i % cols
		row := i / cols
		ci := fmt.Sprintf("C%d", i)
		wf[ci] = map[string]interface{}{
			"class_type": "ImageCompositeMasked",
			"inputs": map[string]interface{}{
				"destination":   []interface{}{prev, 0},
				"source":        []interface{}{fmt.Sprintf("S%d", i), 0},
				"mask":          []interface{}{fmt.Sprintf("M%d", i), 0},
				"x":             col * cell,
				"y":             row * cell,
				"resize_source": false,
			},
		}
		prev = ci
	}
	wf["10"] = map[string]interface{}{"class_type": "SaveImage", "inputs": map[string]interface{}{"images": []interface{}{prev, 0}, "filename_prefix": "aichat_comp"}}
	return wf
}

// ── 任务执行 ────────────────────────────────────────────────────────────────

// comfyResolveParams 把请求参数与服务端配置合并，并对所有取值做边界收敛。
func comfyResolveParams(cfg config.ComfyUIConfig, req comfyImageRequest) (comfyGenOptions, error) {
	opts := comfyGenOptions{
		Steps:      cfg.Steps,
		CFG:        cfg.CFG,
		Resolution: cfg.Resolution,
	}
	if req.Steps > 0 {
		opts.Steps = req.Steps
	}
	if req.Steps > 150 {
		opts.Steps = 150
	}
	if req.CFG > 0 {
		opts.CFG = req.CFG
	}
	// 这些模式不依赖正方形分辨率：遮罩重绘/外补/修复跟随源图尺寸，拼合用网格格子，
	// 不需要把整图收敛成正方形。
	resolutionDriven := !(req.Mode == "inpaint" || req.Mode == "outpaint" ||
		req.Mode == "restore" || req.Mode == "composite")
	if resolutionDriven {
		if req.Resolution > 0 {
			opts.Resolution = req.Resolution
		} else if req.Width > 0 && req.Height > 0 {
			// 非正方形请求按短边收敛：Qwen-Image 的编码节点只吃一个 resolution，
			// 强行按宽高比出图会得到被拉伸的画面，这里统一按短边走正方形。
			opts.Resolution = min(req.Width, req.Height)
		} else if req.Size != "" {
			w, h, ok := parseImageSize(req.Size)
			if !ok {
				return opts, fmt.Errorf("size 格式应为 宽x高，例如 1024x1024")
			}
			opts.Resolution = min(w, h)
		}
		if opts.Resolution < 256 || opts.Resolution > 2048 {
			return opts, fmt.Errorf("分辨率需在 256~2048 之间，当前 %d", opts.Resolution)
		}
		// 必须能被 64 整除，否则 latent 尺寸取整后画面会出现错位接缝
		opts.Resolution = opts.Resolution / 64 * 64
	}

	if req.Seed != nil {
		opts.Seed = *req.Seed
	} else {
		opts.Seed = randomSeed()
	}
	return opts, nil
}

// parseImageSize 解析 OpenAI 风格的 "1024x1024"。
func parseImageSize(size string) (int, int, bool) {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(size)), func(r rune) bool {
		return r == 'x' || r == '*' || r == '×'
	})
	if len(parts) != 2 {
		return 0, 0, false
	}
	var w, h int
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil {
		return 0, 0, false
	}
	if w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

// randomSeed 生成一个随机种子（ComfyUI 的种子是 int64）。
func randomSeed() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UnixNano()
	}
	return int64(binary.BigEndian.Uint64(b[:]) & 0x7FFFFFFFFFFF)
}

// startComfyTask 建任务并起后台 goroutine 执行，立即返回。
// 必须用独立 context（不继承请求 context）——POST 一返回，请求 context 就取消了。
func (h *AIGatewayHandler) startComfyTask(cfg config.ComfyUIConfig, req comfyImageRequest, prompt, negative string, opts comfyGenOptions, mode, modelID string) comfyTask {
	task := &comfyTask{
		ID:        "img_" + comfyRandomHex(8),
		Status:    "queued",
		Phase:     "排队中",
		Prompt:    prompt,
		Mode:      mode,
		Model:     modelID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	comfyTasks.put(task)
	// 快照：goroutine 会并发改写任务对象，调用方只能拿到启动瞬间的拷贝
	snapshot := *task
	taskID := task.ID

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in comfy task %s: %v", taskID, r)
				comfyTasks.mutate(taskID, func(t *comfyTask) {
					t.Status = "failed"
					t.Phase = "内部错误"
					t.Error = fmt.Sprintf("内部错误: %v", r)
				})
			}
		}()
		if cfg.Family == comfyFamilyMiniMax {
			// 云端通道：MiniMax image-01（文生图 + subject_reference 参考主体）
			h.runMiniMaxImageTask(cfg, taskID, req, prompt, negative, opts, mode)
		} else {
			h.runComfyTask(cfg, taskID, req, prompt, negative, opts, mode)
		}
		// 云端通道（MiniMax image-01）与本地 ComfyUI 共用同一套任务状态机：
		// 前面按家族分流，后面的「终态落库」对两条链路完全通用。
		h.persistComfyTaskOutcome(taskID, req, prompt)
	}()
	return snapshot
}

// persistComfyTaskOutcome 把出图任务的终态写成一条 assistant 消息（幂等，id 用前端给的 message_id）。
// 只有请求带了 conversation_id + message_id 才写；缺字段的老调用方（OpenAI 兼容端点等）直接跳过。
func (h *AIGatewayHandler) persistComfyTaskOutcome(taskID string, req comfyImageRequest, prompt string) {
	if h.db == nil || req.ConversationID == "" || req.MessageID == "" {
		return
	}
	task, ok := comfyTasks.get(taskID)
	if !ok {
		return
	}
	if task.Status != "succeeded" && task.Status != "failed" {
		return
	}
	// 防串扰：用户点「重新生成」时同一条消息会换成新 task_id。
	// 老任务如果还在 ComfyUI 里跑，晚到的终态不能把新任务的结果覆盖掉。
	if old, err := h.db.GetChatPlusMessage(req.MessageID); err == nil && old != nil {
		if old.TaskID != "" && old.TaskID != taskID {
			log.Printf("[comfy] 任务 %s 已不是消息 %s 的当前任务（现为 %s），跳过落库",
				taskID, req.MessageID, old.TaskID)
			return
		}
	}
	m := &models.ChatPlusMessage{
		ID:             req.MessageID,
		ConversationID: req.ConversationID,
		Role:           "assistant",
		Content:        prompt,
		TaskID:         taskID,
		Kind:           "image",
		Status:         task.Status,
		Ts:             task.CreatedAt.UnixMilli(),
	}
	if task.Status == "succeeded" {
		imgs := make([]map[string]interface{}, 0, len(task.Images))
		for _, img := range task.Images {
			imgs = append(imgs, map[string]interface{}{
				"id": img.URL, "url": img.URL, "w": img.Width, "h": img.Height,
				// 带上模型 id，换设备/刷新后也能看出这张图是哪个模型出的
				"model": task.Model,
			})
		}
		raw, err := json.Marshal(imgs)
		if err != nil {
			log.Printf("[comfy] 任务 %s 图片序列化失败: %v", taskID, err)
			return
		}
		m.Images = string(raw)
	} else {
		m.ErrMessage = task.Error
	}
	if err := h.db.UpsertChatPlusMessage(m); err != nil {
		log.Printf("[comfy] 任务 %s 消息落库失败: %v", taskID, err)
		return
	}
	_ = h.db.TouchChatPlusConversation(req.ConversationID)
	log.Printf("[comfy] 任务 %s 终态已落库: status=%s images=%d", taskID, m.Status, len(task.Images))
}

// runComfyTask 执行完整出图链路：上传参考图（可选）→ 提交 → 轮询 → 下载 → 落盘。
func (h *AIGatewayHandler) runComfyTask(cfg config.ComfyUIConfig, taskID string, req comfyImageRequest, prompt, negative string, opts comfyGenOptions, mode string) {
	client := newComfyClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Status = "running"
		t.Phase = "准备中"
		t.started = time.Now()
	})

	// 1) 按模式把底图 / 多图 / 遮罩送进 ComfyUI
	uploadOne := func(tag, raw string) (string, error) {
		data, err := decodeImagePayload(raw)
		if err != nil {
			return "", err
		}
		return client.uploadImage(ctx, tag+comfyRandomHex(6)+".png", data, cfg.UploadDir)
	}
	switch mode {
	case "edit":
		raw := req.Image
		if raw == "" && len(req.Images) > 0 {
			raw = req.Images[0]
		}
		comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "上传参考图" })
		name, err := uploadOne("aichat_ref_", raw)
		if err != nil {
			h.failComfyTask(taskID, err.Error())
			return
		}
		opts.ImageName = name
	case "inpaint", "outpaint", "restore":
		raw := req.Image
		if raw == "" && len(req.Images) > 0 {
			raw = req.Images[0]
		}
		if strings.TrimSpace(raw) == "" {
			h.failComfyTask(taskID, "该模式需要一张底图")
			return
		}
		comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "上传底图" })
		baseName, err := uploadOne("aichat_base_", raw)
		if err != nil {
			h.failComfyTask(taskID, err.Error())
			return
		}
		opts.ImageName = baseName
		if mode == "inpaint" || mode == "outpaint" {
			if strings.TrimSpace(req.Mask) == "" {
				h.failComfyTask(taskID, "遮罩重绘/外补模式需要一张遮罩图")
				return
			}
			comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "上传遮罩" })
			maskName, err := uploadOne("aichat_mask_", req.Mask)
			if err != nil {
				h.failComfyTask(taskID, err.Error())
				return
			}
			opts.MaskName = maskName
		}
	case "multi_ref", "composite":
		imgs := req.Images
		if len(imgs) == 0 && req.Image != "" {
			imgs = []string{req.Image}
		}
		if len(imgs) == 0 {
			h.failComfyTask(taskID, "该模式需要至少一张图片")
			return
		}
		comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "上传多张图片" })
		for _, ref := range imgs {
			name, err := uploadOne("aichat_ref_", ref)
			if err != nil {
				h.failComfyTask(taskID, err.Error())
				return
			}
			opts.ImageNames = append(opts.ImageNames, name)
		}
	}

	// 2) 提交工作流
	comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "提交任务" })
	workflow := buildComfyWorkflow(cfg, prompt, negative, opts)
	promptID, err := client.queuePrompt(ctx, workflow)
	if err != nil {
		h.failComfyTask(taskID, err.Error())
		return
	}
	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.promptID = promptID
		t.Phase = "排队中"
		t.Progress = 1
	})

	// 3) 进度增强（不阻塞主链路）
	progressCtx, stopProgress := context.WithCancel(ctx)
	defer stopProgress()
	go client.watchProgress(progressCtx, promptID, func(pct int, phase string) {
		comfyTasks.mutate(taskID, func(t *comfyTask) {
			if pct > t.Progress {
				t.Progress = pct
			}
			t.Phase = phase
		})
	})

	// 4) 主链路：轮询 /history 直到完成
	entry, err := waitComfyHistory(ctx, client, promptID)
	if err != nil {
		// 等待窗口到了，但 ComfyUI 完全不受我们 ctx 约束（它在自己进程里跑）。
		// 实测图生图首帧要加载模型，常常比 timeout_sec 更久才出图——此时判失败就是
		// 「图已经生成、前端却收到失败」。所以不判死，换独立宽限窗继续等。
		log.Printf("[comfy] 任务 %s 主等待窗口结束（%v），进入宽限续等", taskID, err)
		comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "等待时间较长（ComfyUI 仍在执行）" })
		graceCtx, cancelGrace := context.WithTimeout(context.Background(), comfyGraceWait)
		// 宽限期内进度照旧要更新（模型加载完成后 KSampler 还会推 progress）
		go client.watchProgress(graceCtx, promptID, func(pct int, phase string) {
			comfyTasks.mutate(taskID, func(t *comfyTask) {
				if pct > t.Progress {
					t.Progress = pct
				}
				t.Phase = phase
			})
		})
		entry, err = waitComfyHistory(graceCtx, client, promptID)
		cancelGrace()
		if err != nil {
			h.failComfyTask(taskID, err.Error())
			return
		}
	}
	stopProgress()

	if entry.Status.StatusStr == "error" {
		h.failComfyTask(taskID, comfyExecutionError(entry))
		return
	}

	// 5) 下载成品图并落盘
	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Phase = "保存图片"
		t.Progress = 99
	})
	refs := collectComfyImages(entry)
	if len(refs) == 0 {
		h.failComfyTask(taskID, "ComfyUI 执行完成但没有产出图片（检查工作流是否包含 SaveImage 节点）")
		return
	}

	images := make([]comfyImage, 0, len(refs))
	var firstErr error
	for _, ref := range refs {
		data, err := client.viewImage(ctx, ref)
		if err != nil {
			firstErr = err
			continue
		}
		img, err := saveComfyImage(cfg, data)
		if err != nil {
			firstErr = err
			continue
		}
		images = append(images, img)
	}
	if len(images) == 0 {
		h.failComfyTask(taskID, fmt.Sprintf("成品图保存失败: %v", firstErr))
		return
	}

	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Status = "succeeded"
		t.Phase = "完成"
		t.Progress = 100
		t.Images = images
		t.ImageURL = images[0].URL
	})
}

// waitComfyHistory 轮询 /history 直到拿到执行结果或超时。
func waitComfyHistory(ctx context.Context, client *comfyClient, promptID string) (*comfyHistoryEntry, error) {
	ticker := time.NewTicker(comfyPollInterval)
	defer ticker.Stop()
	for {
		entry, ok, err := client.history(ctx, promptID)
		if err != nil {
			// 单次查询失败（网络抖动）不立刻判死，继续等
			log.Printf("[comfy] 查询 %s 失败: %v", promptID, err)
		} else if ok {
			return entry, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("出图超时或已取消（%v）：ComfyUI 可能仍在执行，稍后可在其队列里看到", ctx.Err())
		case <-ticker.C:
		}
	}
}

// collectComfyImages 汇总所有输出节点里的图片。
func collectComfyImages(entry *comfyHistoryEntry) []comfyImageRef {
	if saveNode, ok := entry.Outputs[comfySaveNodeID]; ok && len(saveNode.Images) > 0 {
		return saveNode.Images
	}
	var out []comfyImageRef
	for _, node := range entry.Outputs {
		out = append(out, node.Images...)
	}
	return out
}

// failComfyTask 统一标记失败。
func (h *AIGatewayHandler) failComfyTask(taskID, msg string) {
	log.Printf("[comfy] 任务 %s 失败: %s", taskID, msg)
	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Status = "failed"
		t.Error = msg
	})
}

// decodeImagePayload 把 data URL / http(s) URL 统一解码成图片字节。
func decodeImagePayload(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("参考图为空")
	}
	if strings.HasPrefix(raw, "data:") {
		_, b64 := splitDataURL(raw)
		if b64 == "" {
			return nil, fmt.Errorf("参考图 data URL 格式不正确")
		}
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("参考图 base64 解码失败: %w", err)
		}
		if len(data) > comfyMaxRefImage {
			return nil, fmt.Errorf("参考图不能超过 10MB")
		}
		return data, nil
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		// 走专用 no-proxy 客户端：容器里常配了 HTTP_PROXY，
		// 局域网/内网的参考图 URL 一旦过代理会直接连不通。
		resp, err := comfyNoProxyClient.Get(raw)
		if err != nil {
			return nil, fmt.Errorf("下载参考图失败: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("下载参考图失败 HTTP %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, comfyMaxRefImage+1))
		if err != nil {
			return nil, fmt.Errorf("下载参考图失败: %w", err)
		}
		if len(data) > comfyMaxRefImage {
			return nil, fmt.Errorf("参考图不能超过 10MB")
		}
		return data, nil
	}
	return nil, fmt.Errorf("参考图需为 data URL 或 http(s) 链接")
}

// saveComfyImage 把生成的图片写到 ./data/uploads/ai/YYYYMMDD/，
// 并通过已有的 /api/chat/uploads 静态路由对外（无需新增静态目录）。
func saveComfyImage(cfg config.ComfyUIConfig, data []byte) (comfyImage, error) {
	day := time.Now().Format("20060102")
	dir := filepath.Join(uploadDir, comfyImageSubDir, day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return comfyImage{}, fmt.Errorf("创建图片目录失败: %w", err)
	}
	name := comfyRandomHex(12) + imageExt(data)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return comfyImage{}, fmt.Errorf("写入图片失败: %w", err)
	}
	w, h := imageDimensions(data)
	urlPath := comfyImageURLBase + "/" + day + "/" + name
	return comfyImage{
		URL:      cfg.PublicBaseURL + urlPath,
		Width:    w,
		Height:   h,
		Filename: name,
		RelPath:  filepath.Join(comfyImageSubDir, day, name),
	}, nil
}

// imageExt 按魔数判断扩展名，避免把 PNG 存成 .jpg 让浏览器 MIME 判断出错。
func imageExt(data []byte) string {
	switch {
	case len(data) > 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return ".png"
	case len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8:
		return ".jpg"
	case len(data) > 12 && string(data[8:12]) == "WEBP":
		return ".webp"
	default:
		return ".png"
	}
}

// imageDimensions 读 PNG / JPEG 头部的宽高（不引第三方库，够用且无额外依赖）。
// 解析失败返回 0,0——前端只把它当提示信息，不影响使用。
func imageDimensions(data []byte) (int, int) {
	// PNG: 8 字节签名 + 4 字节长度 + "IHDR" + 宽(4) + 高(4)
	if len(data) >= 24 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")) && string(data[12:16]) == "IHDR" {
		w := int(binary.BigEndian.Uint32(data[16:20]))
		h := int(binary.BigEndian.Uint32(data[20:24]))
		return w, h
	}
	// JPEG: 逐段扫描 SOFn
	if len(data) > 4 && data[0] == 0xFF && data[1] == 0xD8 {
		i := 2
		for i+9 < len(data) {
			if data[i] != 0xFF {
				i++
				continue
			}
			marker := data[i+1]
			// SOF0/1/2/3/5/6/7/9/10/11/13/14/15 都带尺寸
			if marker >= 0xC0 && marker <= 0xCF &&
				marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
				h := int(binary.BigEndian.Uint16(data[i+5 : i+7]))
				w := int(binary.BigEndian.Uint16(data[i+7 : i+9]))
				return w, h
			}
			if i+3 >= len(data) {
				break
			}
			segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
			if segLen < 2 {
				break
			}
			i += 2 + segLen
		}
	}
	return 0, 0
}

// randomHex 生成 n 字节随机数的十六进制串。
func comfyRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// ── HTTP handler：聊天页 ────────────────────────────────────────────────────

// comfyRuntime 取归一化后的 ComfyUI 配置，并在未配置时统一回错。
func (h *AIGatewayHandler) comfyRuntime(c *gin.Context) (config.ComfyUIConfig, bool) {
	cfg := h.cfg.ChatPlus.ComfyUI.Normalized()
	if !cfg.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "未配置 chat_plus.comfyui.base_url，本地出图功能已禁用",
		})
		return cfg, false
	}
	return cfg, true
}

// ChatPlusImageStatus GET /api/chat-plus/image/status —— 出图能力探活
//
// available 只代表「本地 ComfyUI 可用」；是否整体能出图要看
// available || minimax_available（云端通道不依赖局域网机器）。
func (h *AIGatewayHandler) ChatPlusImageStatus(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	cfg := h.cfg.ChatPlus.ComfyUI.Normalized()
	mm := h.cfg.ChatPlus.MiniMaxImage.Normalized(h.cfg.MiniMax.APIKey)
	resp := gin.H{
		"available":   false,
		"base_url":    cfg.BaseURL,
		"model":       cfg.Model,
		"steps":       cfg.Steps,
		"cfg":         cfg.CFG,
		"sampler":     cfg.Sampler,
		"scheduler":   cfg.Scheduler,
		"resolution":  cfg.Resolution,
		"timeout_sec": cfg.TimeoutSec,
		"modes":       []string{"t2i", "edit", "inpaint", "outpaint", "restore", "multi_ref", "composite"},
		"image_model": comfyImageModel,
		"max_batch":   comfyMaxBatch,
		// 多模型：默认预设 + 可选预设 id 列表。模型详情（耗时/能力/可用性）走
		// GET /api/chat-plus/image/models，那里会额外探活 ComfyUI 上真实存在的文件。
		"default_model": comfyDefaultPresetID(cfg),
		"models":        comfyPresetIDs(cfg),
		// 云端通道（MiniMax image-01）
		"minimax_available": mm.Enabled(),
		"minimax_base_url":  mm.BaseURL,
		"minimax_model":     mm.Model,
		"minimax_max_n":     mm.MaxImages,
		"minimax_modes":     []string{"t2i", "edit"},
	}
	if !cfg.Enabled() {
		resp["error"] = "未配置 comfyui.base_url"
		c.JSON(http.StatusOK, resp)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := newComfyClient(cfg).ping(ctx); err != nil {
		resp["error"] = err.Error()
		c.JSON(http.StatusOK, resp)
		return
	}
	resp["available"] = true
	c.JSON(http.StatusOK, resp)
}

// ChatPlusImageGenerate POST /api/chat-plus/image/generate
// 立即返回 task_id，进度通过 /tasks/:id/stream（SSE）或轮询获取。
func (h *AIGatewayHandler) ChatPlusImageGenerate(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	// 注意：这里不能一上来就要求 ComfyUI 可用。云端（MiniMax）与本地 ComfyUI 是
	// 并列的两个供应商，配了任意一个就能出图，谁可用由最终选中的预设家族决定。
	cfg := h.cfg.ChatPlus.ComfyUI.Normalized()

	var req comfyImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入画面描述"})
		return
	}
	if len([]rune(prompt)) > comfyMaxPromptLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("描述过长（上限 %d 字）", comfyMaxPromptLen)})
		return
	}

	negative := strings.TrimSpace(req.NegativePrompt)
	if negative == "" {
		negative = cfg.NegativePrompt
	}
	// 出图模式：优先用前端显式传的 mode；没传则按有无图片推断（兼容老调用）。
	mode := strings.TrimSpace(req.Mode)
	switch mode {
	case "":
		if len(req.Images) > 0 || strings.TrimSpace(req.Image) != "" {
			mode = "edit"
		} else {
			mode = "t2i"
		}
	case "t2i", "edit", "inpaint", "outpaint", "restore", "multi_ref", "composite":
		// 合法取值，放行
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("不支持的模式: %s", mode)})
		return
	}

	// 模型切换：前端传预设 id（空 = 用默认预设），后端把整条工作流参数换掉。
	preset, ok := comfyPresetByID(cfg, req.Model)
	if !ok {
		ids := make([]string, 0, len(comfyPresets(cfg)))
		for _, p := range comfyPresets(cfg) {
			ids = append(ids, p.ID)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("不支持的模型: %s（可选：%s）", req.Model, strings.Join(ids, "、")),
		})
		return
	}

	// ── 云端通道（MiniMax image-01）──
	// 只有文生图与「参考主体」图生图两种能力：遮罩重绘 / 扩图 / 修复 / 多图融合
	// 都依赖本地扩散工作流，云端做不到，直接给明确提示而不是静默降级。
	if comfyIsOnlineFamily(preset.Family) {
		if mode != "t2i" && mode != "edit" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("在线模型「%s」只支持文生图与图生图，%s 请换成本地模型", preset.Label, mode),
			})
			return
		}
		mm := h.cfg.ChatPlus.MiniMaxImage.Normalized(h.cfg.MiniMax.APIKey)
		if !mm.Enabled() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "未配置 chat_plus.minimax_image.api_key（或 minimax.api_key），在线出图不可用",
			})
			return
		}
		if req.N < 0 || req.N > mm.MaxImages {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("在线出图一次最多 %d 张", mm.MaxImages),
			})
			return
		}
		// 分辨率先按预设取值，再让 comfyResolveParams 用请求里的 size/width 覆盖；
		// 步数/采样器对云端无意义，保持原样传下去也不会被使用。
		cfg = comfyApplyPreset(cfg, preset)
		opts, err := comfyResolveParams(cfg, req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		opts.Mode = mode
		want := req.N
		if want <= 0 {
			want = 1
		}
		task := h.startComfyTask(cfg, req, prompt, negative, opts, mode, preset.ID)
		c.JSON(http.StatusOK, gin.H{
			"task_id":  task.ID,
			"status":   task.Status,
			"mode":     task.Mode,
			"model":    task.Model,
			"seed":     opts.Seed,
			"steps":    opts.Steps,
			"provider": "minimax",
			"count":    want,
		})
		return
	}

	// ── 本地 ComfyUI 通道 ──
	if !cfg.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "未配置 chat_plus.comfyui.base_url，本地出图功能已禁用",
		})
		return
	}
	if req.N > comfyMaxBatch {
		c.JSON(http.StatusBadRequest, gin.H{"error": "本地出图一次只生成 1 张，请分多次请求"})
		return
	}
	// 部分模式强依赖特定家族，自动切到对应预设，省得前端还要手动换模型。
	switch mode {
	case "inpaint", "outpaint", "restore":
		if preset.Family != comfyFamilyQwenEdit {
			if p, ok2 := comfyPresetByFamily(cfg, comfyFamilyQwenEdit); ok2 {
				preset = p
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "当前没有配置 Qwen-Image-Edit 模型，无法做遮罩重绘/外补/修复"})
				return
			}
		}
	case "multi_ref":
		if preset.Family != comfyFamilyQwen21 {
			if p, ok2 := comfyPresetByFamily(cfg, comfyFamilyQwen21); ok2 {
				preset = p
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "当前没有配置 Qwen-Image-2.1 模型，无法做多图内容结合"})
				return
			}
		}
	}
	// 基础能力校验（t2i/edit 仍按家族约束）。
	if mode == "t2i" && !comfySupportsT2I(preset) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("模型「%s」只能改图，请先附一张参考图", preset.Label)})
		return
	}
	if mode == "edit" && !comfySupportsEdit(preset) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("模型「%s」不支持参考图编辑，请换成支持改图的模型", preset.Label)})
		return
	}
	// 输入校验
	switch mode {
	case "inpaint", "outpaint":
		if strings.TrimSpace(req.Mask) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "遮罩重绘/外补模式需要一张遮罩图（白色=要重绘区域）"})
			return
		}
		if strings.TrimSpace(req.Image) == "" && len(req.Images) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "遮罩重绘/外补模式需要一张底图"})
			return
		}
	case "restore":
		if strings.TrimSpace(req.Image) == "" && len(req.Images) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "修复模式需要一张底图"})
			return
		}
	case "multi_ref", "composite":
		if len(req.Images) == 0 && strings.TrimSpace(req.Image) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该模式需要至少一张图片（多图请传 images 数组）"})
			return
		}
	}
	// 先把预设叠进配置再算参数：这样 flux2 的 4 步、各家族的分辨率都能生效，
	// 而请求里显式给的 steps/cfg/size 仍然优先（comfyResolveParams 内部就是这个优先级）。
	cfg = comfyApplyPreset(cfg, preset)
	opts, err := comfyResolveParams(cfg, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	opts.Mode = mode

	task := h.startComfyTask(cfg, req, prompt, negative, opts, mode, preset.ID)
	c.JSON(http.StatusOK, gin.H{
		"task_id": task.ID,
		"status":  task.Status,
		"mode":    task.Mode,
		"model":   task.Model,
		"seed":    opts.Seed,
		"steps":   opts.Steps,
	})
}

// ChatPlusImageTask GET /api/chat-plus/image/tasks/:id —— 轮询单任务
func (h *AIGatewayHandler) ChatPlusImageTask(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	task, ok := comfyTasks.get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// ChatPlusImageTaskStream GET /api/chat-plus/image/tasks/:id/stream
// SSE 推送任务状态，直到成功/失败为止。事件体统一是任务 JSON，
// 前端只需按 status 字段判断，省掉一套多事件类型的解析逻辑。
func (h *AIGatewayHandler) ChatPlusImageTaskStream(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	taskID := c.Param("id")
	if _, ok := comfyTasks.get(taskID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}

	// 先确认能流式写，再落响应头——一旦 WriteHeader，后面的 JSON 错误就写不进去了
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "当前服务器不支持流式响应"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 关掉 nginx/反代缓冲，否则进度会被攒着一起发
	c.Writer.WriteHeader(http.StatusOK)

	writer := newSSEWriter(c.Writer, flusher)

	send := func(task comfyTask) bool {
		payload, err := json.Marshal(task)
		if err != nil {
			return false
		}
		if err := writer.write([]byte("data: " + string(payload) + "\n\n")); err != nil {
			return false
		}
		return true
	}

	ctx := c.Request.Context()
	ticker := time.NewTicker(comfyStreamTick)
	// 心跳要密：出图在「上传参考图/排队/加载模型」阶段可能几分钟没有进度变化，
	// 而移动网络 NAT 与反代的空闲超时常见只有 15~30s，20s 一跳会被掐断，
	// 前端就会误报「请求未到达服务器」。这里 4s 一跳，远低于任何空闲超时。
	heartbeat := time.NewTicker(4 * time.Second)
	deadline := time.After(comfyStreamMaxIdle)
	defer ticker.Stop()
	defer heartbeat.Stop()

	last := ""
	lastSend := time.Now()
	for {
		select {
		case <-ctx.Done():
			return // 客户端断开
		case <-deadline:
			// 到期只结束这条 SSE（让前端转轮询），绝不写 failed：
			// 任务可能仍在 ComfyUI 里跑着，写 failed 会让前端把已生成的图判成失败。
			_ = writer.write([]byte(": stream-deadline\n\n"))
			return
		case <-heartbeat.C:
			if err := writer.write([]byte(": ping\n\n")); err != nil {
				return
			}
		case <-ticker.C:
			task, found := comfyTasks.get(taskID)
			if !found {
				_ = writer.write([]byte("data: {\"status\":\"failed\",\"error\":\"任务不存在或已过期\"}\n\n"))
				return
			}
			// 只在内容变化时推送，减少无意义流量（elapsed_ms 单独看，避免每秒都变）；
			// 但无变化超过 3s 也要补一帧，让 elapse 前进、链路保持活跃
			sig := fmt.Sprintf("%s|%d|%s|%d", task.Status, task.Progress, task.Phase, len(task.Images))
			if sig != last || time.Since(lastSend) >= 3*time.Second {
				last = sig
				if !send(task) {
					return
				}
				lastSend = time.Now()
			}
			if task.Status == "succeeded" || task.Status == "failed" {
				return
			}
		}
	}
}

// ── HTTP handler：OpenAI 兼容 ───────────────────────────────────────────────

// OpenAIImagesGenerations POST /api/ai-gateway/v1/images/generations
//
// OpenAI 图片接口是同步语义，这里也同步等待（受 timeout_sec 约束）。
// 鉴权与计费口径沿用 AI Gateway：API Key 需带 media 权限域 + 模型白名单。
func (h *AIGatewayHandler) OpenAIImagesGenerations(c *gin.Context) {
	key, ok := h.authenticateAdminOrAPIKey(c, comfyImageScope)
	if !ok {
		return
	}
	// 先把配置取出来做预设解析：云端预设（minimax）不需要 ComfyUI 可用。
	cfg := h.cfg.ChatPlus.ComfyUI.Normalized()

	var req comfyImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
		return
	}
	// 与聊天页共用同一套预设：调用方传的 model 就是预设 id，空则用默认预设。
	preset, ok := comfyPresetByID(cfg, req.Model)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("不支持的模型: %s（本地可选 %s）", strings.TrimSpace(req.Model), strings.Join(comfyPresetIDs(cfg), "、")),
		})
		return
	}
	// 供应商可用性：云端预设看 key，本地预设看 base_url。
	if comfyIsOnlineFamily(preset.Family) {
		mm := h.cfg.ChatPlus.MiniMaxImage.Normalized(h.cfg.MiniMax.APIKey)
		if !mm.Enabled() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 chat_plus.minimax_image.api_key，在线出图不可用"})
			return
		}
	} else if !cfg.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "未配置 chat_plus.comfyui.base_url，本地出图功能已禁用"})
		return
	}
	// 模型白名单按「实际要用的预设」校验，而不是写死默认模型：
	// 否则限制了 qwen-image-2.1 的 key 也能偷偷用 flux-2-klein。
	// 超管认证时 key 为 nil（不限制），此时跳过校验。
	if key != nil && !h.ensureModelAllowed(c, key, preset.ID) {
		return
	}
	cfg = comfyApplyPreset(cfg, preset)
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt 不能为空"})
		return
	}
	if len([]rune(prompt)) > comfyMaxPromptLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("prompt 过长（上限 %d 字）", comfyMaxPromptLen)})
		return
	}
	if req.N > comfyMaxBatch {
		c.JSON(http.StatusBadRequest, gin.H{"error": "n 仅支持 1（本地单卡串行出图）"})
		return
	}

	opts, err := comfyResolveParams(cfg, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	negative := strings.TrimSpace(req.NegativePrompt)
	if negative == "" {
		negative = cfg.NegativePrompt
	}
	mode := "t2i"
	if strings.TrimSpace(req.Image) != "" {
		mode = "edit"
	}

	start := time.Now()
	task := h.startComfyTask(cfg, req, prompt, negative, opts, mode, preset.ID)
	// 同步等结果：直接复用任务表，避免两套执行路径
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()
	final, err := waitComfyTask(ctx, task.ID)
	latency := time.Since(start)
	if err != nil {
		h.logAPIRequest(key, preset.ID, "comfyui", "/api/ai-gateway/v1/images/generations",
			"image_generation", http.StatusGatewayTimeout, false, err.Error(),
			truncateString(prompt, 500), "", c.ClientIP(), latency, usageSummary{})
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	if final.Status != "succeeded" {
		h.logAPIRequest(key, preset.ID, "comfyui", "/api/ai-gateway/v1/images/generations",
			"image_generation", http.StatusBadGateway, false, final.Error,
			truncateString(prompt, 500), truncateString(final.Error, 500), c.ClientIP(), latency, usageSummary{})
		c.JSON(http.StatusBadGateway, gin.H{"error": final.Error})
		return
	}

	wantB64 := req.ResponseFormat == "b64_json"
	data := make([]gin.H, 0, len(final.Images))
	for _, img := range final.Images {
		item := gin.H{"revised_prompt": prompt}
		if wantB64 {
			raw, readErr := readSavedImage(img)
			if readErr != nil {
				continue
			}
			item["b64_json"] = base64.StdEncoding.EncodeToString(raw)
		} else {
			item["url"] = img.URL
		}
		if img.Width > 0 && img.Height > 0 {
			item["width"], item["height"] = img.Width, img.Height
		}
		data = append(data, item)
	}
	if len(data) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "结果图读取失败"})
		return
	}

	h.logAPIRequest(key, preset.ID, "comfyui", "/api/ai-gateway/v1/images/generations",
		"image_generation", http.StatusOK, true, "",
		truncateString(prompt, 500), truncateString(fmt.Sprintf("%+v", data), 500),
		c.ClientIP(), latency, usageSummary{})

	c.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"model":   preset.ID,
		"seed":    opts.Seed,
		"data":    data,
	})
}

// waitComfyTask 等待任务进入终态。
func waitComfyTask(ctx context.Context, taskID string) (comfyTask, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		task, ok := comfyTasks.get(taskID)
		if !ok {
			return comfyTask{}, fmt.Errorf("任务不存在或已过期")
		}
		if task.Status == "succeeded" || task.Status == "failed" {
			return task, nil
		}
		select {
		case <-ctx.Done():
			return task, fmt.Errorf("等待出图超时: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// readSavedImage 按落盘相对路径反查文件，供 response_format=b64_json 输出。
func readSavedImage(img comfyImage) ([]byte, error) {
	if img.RelPath == "" {
		return nil, fmt.Errorf("缺少落盘路径")
	}
	// 路径由服务端生成，仍然做一次越界校验，避免将来调用方传入外部值
	clean := filepath.Clean(img.RelPath)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return nil, fmt.Errorf("非法文件路径")
	}
	return os.ReadFile(filepath.Join(uploadDir, clean))
}
