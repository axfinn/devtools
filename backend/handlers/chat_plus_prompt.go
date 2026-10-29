package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"devtools/utils"

	"github.com/gin-gonic/gin"
)

// 本文件：出图提示词优化（AI 帮写提示词）。
//
// 背景：图生图/文生图的效果八成取决于提示词，但真实使用里用户只会写「把背景换成海边」
// 这种一句话。这里加一个「让大模型帮改写」的入口：用户写一句大白话 → 文本模型改写成
// 扩散模型能吃的结构化提示词 → 回给前端让用户自己看一眼、改一改、再采纳。
//
// 设计约束：
//   - 只做「改写」，不替用户点出图：结果永远落回输入框，用户有最终决定权。
//   - 复用 chat_plus 的密码门与文本模型通道（Complete / executeChatRequest），
//     不引入新的 key、新的 provider 配置。
//   - 参考图只在「局域网 Ollama + 视觉模型」时真的给模型看；其它情况静默降级成纯文本
//     优化，不让「看不了图」变成一条报错。

const (
	// promptOptimizeMaxInput 与 comfyMaxPromptLen 对齐：能进输入框就能优化。
	promptOptimizeMaxInput = 2000
	// promptOptimizeMaxOutput 优化结果上限。扩散模型吃不下长篇大论，太长反而稀释语义。
	promptOptimizeMaxOutput = 600
	// promptOptimizeMaxImageBytes 参考图 base64 体积上限。前端已压到 1024px/0.8，
	// 这里只是防有人直接塞原图把请求体撑爆。
	promptOptimizeMaxImageBytes = 8 << 20
	// promptOptimizeTimeout 同步接口的等待上限。它被网关卡着（Cloudflare 100s 就 524），
	// 所以只能给这么长；真正的长等待走异步任务。
	promptOptimizeTimeout = 150 * time.Second
	// promptOptimizeTaskTimeout 异步任务的总时长上限。任务在服务端跑，浏览器的连接
	// 早就结束了，因此不受网关超时约束，可以给本地大模型留足时间。
	promptOptimizeTaskTimeout = 300 * time.Second
	// promptOptimizeTaskTTL 终态任务在内存里保留多久：够前端轮询到结果即可。
	promptOptimizeTaskTTL = 15 * time.Minute

	promptOptimizeStatusRunning  = "running"
	promptOptimizeStatusDone     = "done"
	promptOptimizeStatusFailed   = "failed"
	promptOptimizeStatusCanceled = "canceled"
)

// 提示词优化模式。前端按「有没有附参考图 / 有没有写要求」自动决定，用户不用选。
const (
	promptModeT2I      = "t2i"      // 文生图：从零描述
	promptModeEdit     = "edit"     // 图生图：在参考图上改
	promptModeDescribe = "describe" // 只有参考图没写要求：先看懂图再写提示词
)

type imagePromptOptimizeRequest struct {
	Prompt      string `json:"prompt"`
	Model       string `json:"model"`        // 优化用的文本模型（空 = 服务端默认）
	Strict      bool   `json:"strict"`       // true = 用户在弹窗里明确选了模型，不可用时必须报错而不是偷偷换
	Mode        string `json:"mode"`         // t2i | edit | describe（空则按有没有 prompt/image 推断）
	ImageModel  string `json:"image_model"`  // 目标出图模型名，仅用于让模型知道写法偏好
	ImageFamily string `json:"image_family"` // qwen21 | flux2 | qwen_edit
	Image       string `json:"image"`        // 参考图 dataURL（可选）
}

// preparedPromptOptimize 校验通过后的入参快照。
// 抽出来是为了让「同步接口」和「异步任务」共用同一套校验，不会一边允许一边拒绝。
type preparedPromptOptimize struct {
	Prompt string
	Image  string
	Mode   string
	Model  string
	Req    imagePromptOptimizeRequest
}

// prepareImagePromptOptimize 做与传输层无关的校验，返回 (快照, httpStatus, 错误文案)。
// 校验不通过时快照为 nil。
func (h *AIGatewayHandler) prepareImagePromptOptimize(req imagePromptOptimizeRequest) (*preparedPromptOptimize, int, string) {
	prompt := strings.TrimSpace(req.Prompt)
	if utf8.RuneCountInString(prompt) > promptOptimizeMaxInput {
		return nil, http.StatusBadRequest, fmt.Sprintf("原话过长（上限 %d 字），先精简一下再优化", promptOptimizeMaxInput)
	}

	// 参考图只接受 dataURL，且体积要可控；不合格就当没附图（降级而不是报错）
	image := strings.TrimSpace(req.Image)
	if !strings.HasPrefix(image, "data:image/") || len(image) > promptOptimizeMaxImageBytes {
		image = ""
	}

	mode := normalizePromptOptimizeMode(req.Mode, prompt, image)
	if mode == promptModeT2I && prompt == "" {
		return nil, http.StatusBadRequest, "先写一句你想画什么，或者附一张参考图"
	}

	model := strings.TrimSpace(req.Model)
	// 前端传的模型有两种来源：
	//   1) 用户在优化弹窗里明确选的（strict=true）→ 不可用就报错，不能偷偷换，
	//      否则「我选了 A 却拿到 B 的结果」，排查起来像见鬼；
	//   2) 跟随聊天页当前模型（strict=false）→ 聊天模型可能已被换成不可用引擎
	//      （Ollama 掉线等），这种情况回落到服务端默认，别让「优化提示词」跟着一起挂。
	if model != "" && h.resolveChatProvider(model) == "" {
		if req.Strict {
			return nil, http.StatusBadRequest,
				fmt.Sprintf("所选优化模型当前不可用：%s（请换一个，或选「跟随聊天模型」）", model)
		}
		model = ""
	}

	req.Prompt = prompt
	return &preparedPromptOptimize{Prompt: prompt, Image: image, Mode: mode, Model: model, Req: req}, 0, ""
}

// ChatPlusImagePromptOptimize POST /api/chat-plus/image/prompt/optimize
// 同步版本：保留给「前端来不及改造」和脚本调试用。它受网关超时约束（Cloudflare 100s 会 524），
// 页面实际走的是下面的异步任务版本。
func (h *AIGatewayHandler) ChatPlusImagePromptOptimize(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}

	var req imagePromptOptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
		return
	}

	prep, status, msg := h.prepareImagePromptOptimize(req)
	if prep == nil {
		c.JSON(status, gin.H{"error": msg})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), promptOptimizeTimeout)
	defer cancel()

	out, usedModel, usedImage, err := h.optimizeImagePrompt(ctx, prep.Model, prep.Prompt, prep.Mode, prep.Req, prep.Image)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "优化失败：" + strings.TrimSpace(err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prompt":     out,
		"model":      usedModel,
		"mode":       prep.Mode,
		"used_image": usedImage,
	})
}

// ── 异步任务化（解决网关 524） ──────────────────────────────────────────────
//
// 为什么必须异步：优化要等一个大模型把话说完整，本地 27B 视觉模型带图动辄 40~120s，
// 慢起来直接顶破网关的响应超时（Cloudflare 100s → 524、nginx 60s → 504）。
// 网关一断，浏览器只看到一句 "524"，后端其实还在算、结果白扔。
// 把任务搬到服务端后台跑，浏览器改成短轮询，连接时长永远只有几十毫秒，代理层超时彻底失效。

type promptOptimizeTask struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Prompt    string    `json:"prompt,omitempty"`
	Model     string    `json:"model,omitempty"`
	Mode      string    `json:"mode,omitempty"`
	UsedImage bool      `json:"used_image"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ElapsedMS int64     `json:"elapsed_ms"`

	cancel     context.CancelFunc
	// cancelRequested 用户主动点了停止。单独记一个标记而不是只看 ctx.Err()：
	// 上游 SDK 可能把取消包成自己的错误，用 ctx.Err() 判断会漏判，变成显示「失败」。
	cancelRequested bool
	finishedAt      time.Time
}

var (
	promptOptimizeTasksMu sync.Mutex
	promptOptimizeTasks   = make(map[string]*promptOptimizeTask)
)

// prunePromptOptimizeTasksLocked 丢掉过期的终态任务。调用方必须已持锁。
// 优化任务是内存态、生命周期短，不需要落库；但也因此必须有清理，否则 map 只增不减。
func prunePromptOptimizeTasksLocked(now time.Time) {
	for id, t := range promptOptimizeTasks {
		if t.Status == promptOptimizeStatusRunning {
			continue
		}
		if now.Sub(t.finishedAt) > promptOptimizeTaskTTL {
			delete(promptOptimizeTasks, id)
		}
	}
}

// ChatPlusImagePromptOptimizeTaskCreate POST /api/chat-plus/image/prompt/optimize/tasks
// 立刻返回 task_id，真正的改写放到后台 goroutine。
func (h *AIGatewayHandler) ChatPlusImagePromptOptimizeTaskCreate(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}

	var req imagePromptOptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
		return
	}

	prep, status, msg := h.prepareImagePromptOptimize(req)
	if prep == nil {
		c.JSON(status, gin.H{"error": msg})
		return
	}

	task := &promptOptimizeTask{
		ID:        "popt_" + utils.GenerateHexKey(12),
		Status:    promptOptimizeStatusRunning,
		Mode:      prep.Mode,
		CreatedAt: time.Now(),
	}

	promptOptimizeTasksMu.Lock()
	prunePromptOptimizeTasksLocked(task.CreatedAt)
	promptOptimizeTasks[task.ID] = task
	promptOptimizeTasksMu.Unlock()

	go h.runPromptOptimizeTask(task, prep)

	// 202：任务已受理，但结果还没出来。前端拿到 task_id 开始轮询。
	c.JSON(http.StatusAccepted, gin.H{
		"task_id": task.ID,
		"status":  promptOptimizeStatusRunning,
		"mode":    prep.Mode,
	})
}

// runPromptOptimizeTask 后台执行改写。注意：这里绝不能碰 gin.Context
// （请求早已结束，Context 会被回收），只用快照数据 + 独立的 context。
func (h *AIGatewayHandler) runPromptOptimizeTask(task *promptOptimizeTask, prep *preparedPromptOptimize) {
	ctx, cancel := context.WithTimeout(context.Background(), promptOptimizeTaskTimeout)
	defer cancel()

	promptOptimizeTasksMu.Lock()
	if cur, ok := promptOptimizeTasks[task.ID]; ok {
		cur.cancel = cancel // 供取消接口中断上游请求
	}
	promptOptimizeTasksMu.Unlock()

	// 上游 panic 不能把整个进程带走：单个任务失败就够了。
	defer func() {
		if r := recover(); r != nil {
			h.finishPromptOptimizeTask(task.ID, promptOptimizeStatusFailed, "", "", false, fmt.Sprintf("内部错误：%v", r))
		}
	}()

	out, usedModel, usedImage, err := h.optimizeImagePrompt(ctx, prep.Model, prep.Prompt, prep.Mode, prep.Req, prep.Image)
	if err != nil {
		// 用户主动取消和上游超时要分开说：前者不该显示成"失败"
		if h.promptOptimizeTaskCanceled(task.ID) || errors.Is(ctx.Err(), context.Canceled) {
			h.finishPromptOptimizeTask(task.ID, promptOptimizeStatusCanceled, "", usedModel, false, "已停止优化")
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			h.finishPromptOptimizeTask(task.ID, promptOptimizeStatusFailed, "", usedModel, false,
				fmt.Sprintf("模型超过 %d 秒没给结果，请换一个更快的模型重试", int(promptOptimizeTaskTimeout.Seconds())))
			return
		}
		h.finishPromptOptimizeTask(task.ID, promptOptimizeStatusFailed, "", usedModel, false, "优化失败："+strings.TrimSpace(err.Error()))
		return
	}

	h.finishPromptOptimizeTask(task.ID, promptOptimizeStatusDone, out, usedModel, usedImage, "")
}

func (h *AIGatewayHandler) promptOptimizeTaskCanceled(id string) bool {
	promptOptimizeTasksMu.Lock()
	defer promptOptimizeTasksMu.Unlock()
	if t, ok := promptOptimizeTasks[id]; ok {
		return t.cancelRequested
	}
	return false
}

func (h *AIGatewayHandler) finishPromptOptimizeTask(id, status, out, model string, usedImage bool, errMsg string) {
	promptOptimizeTasksMu.Lock()
	defer promptOptimizeTasksMu.Unlock()

	t, ok := promptOptimizeTasks[id]
	if !ok {
		return
	}
	// 终态不可被覆盖：取消之后上游可能还在飞，它回来时不能再把状态改回 done，
	// 否则用户已经关掉的结果会「复活」。
	if t.Status != promptOptimizeStatusRunning {
		return
	}
	t.Status = status
	t.Prompt = out
	t.Model = model
	t.UsedImage = usedImage
	t.Error = errMsg
	t.ElapsedMS = time.Since(t.CreatedAt).Milliseconds()
	t.finishedAt = time.Now()
	t.cancel = nil
}

// ChatPlusImagePromptOptimizeTaskGet GET /api/chat-plus/image/prompt/optimize/tasks/:id
// 前端轮询这个接口。响应体故意做小（不含参考图），轮询代价可以忽略。
func (h *AIGatewayHandler) ChatPlusImagePromptOptimizeTaskGet(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}

	id := strings.TrimSpace(c.Param("id"))

	promptOptimizeTasksMu.Lock()
	prunePromptOptimizeTasksLocked(time.Now())
	t, ok := promptOptimizeTasks[id]
	var snap promptOptimizeTask
	if ok {
		snap = *t
		if snap.Status == promptOptimizeStatusRunning {
			snap.ElapsedMS = time.Since(snap.CreatedAt).Milliseconds()
		}
	}
	promptOptimizeTasksMu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期，请重新优化"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":    snap.ID,
		"status":     snap.Status,
		"prompt":     snap.Prompt,
		"model":      snap.Model,
		"mode":       snap.Mode,
		"used_image": snap.UsedImage,
		"error":      snap.Error,
		"elapsed_ms": snap.ElapsedMS,
	})
}

// ChatPlusImagePromptOptimizeTaskCancel POST /api/chat-plus/image/prompt/optimize/tasks/:id/cancel
// 用户关掉弹窗/重开一次时调用：停掉上游请求，省得白等一个不会再被看的结果。
func (h *AIGatewayHandler) ChatPlusImagePromptOptimizeTaskCancel(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}

	id := strings.TrimSpace(c.Param("id"))

	cancel, ok := markPromptOptimizeCanceled(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期"})
		return
	}
	if cancel != nil {
		cancel()
	}
	// 立刻落终态：executeChatRequest 没有 ctx 参数，上游请求断不干净，
	// 不能等它回来才把状态改掉，否则这段时间轮询一直是 running，像是没停掉。
	h.finishPromptOptimizeTask(id, promptOptimizeStatusCanceled, "", "", false, "已停止优化")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// markPromptOptimizeCanceled 打上取消标记并返回中断函数。
// 只有 running 的任务会被标记：已终态的任务再打标记，只会让前端显示成「已停止」。
// 返回 (cancel, 任务是否存在)。
func markPromptOptimizeCanceled(id string) (context.CancelFunc, bool) {
	promptOptimizeTasksMu.Lock()
	defer promptOptimizeTasksMu.Unlock()

	t, ok := promptOptimizeTasks[id]
	if !ok {
		return nil, false
	}
	if t.Status != promptOptimizeStatusRunning {
		return nil, true
	}
	t.cancelRequested = true
	return t.cancel, true
}

// normalizePromptOptimizeMode 归一化模式：非法值一律按「有没有原话」推断，
// 前端少传/传错都不会让请求失败。
func normalizePromptOptimizeMode(raw, prompt, image string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case promptModeT2I:
		return promptModeT2I
	case promptModeEdit:
		return promptModeEdit
	case promptModeDescribe:
		return promptModeDescribe
	}
	if image == "" {
		return promptModeT2I
	}
	if prompt == "" {
		return promptModeDescribe
	}
	return promptModeEdit
}

// optimizeImagePrompt 真正干活：先试「带图」，失败自动降级「纯文本」。
func (h *AIGatewayHandler) optimizeImagePrompt(
	ctx context.Context, model, prompt, mode string,
	req imagePromptOptimizeRequest, image string,
) (out, usedModel string, usedImage bool, err error) {
	system := imagePromptOptimizeSystem()
	user := buildImagePromptOptimizeUser(prompt, mode, req)

	usedModel = model
	if usedModel == "" {
		usedModel = fallbackString(h.cfg.MiniMax.Model, "MiniMax-M2.5")
	}

	// 带图只对「局域网 Ollama + 视觉模型」有意义：MiniMax 那条 Anthropic 通道不带图，
	// 硬塞反而会 400 或让模型对着一张看不见的图瞎编。
	if image != "" && h.promptOptimizeCanSeeImage(usedModel) {
		if raw, vErr := h.completeChatOnce(ctx, usedModel, system, user, image); vErr == nil {
			if cleaned := cleanOptimizedPrompt(raw); cleaned != "" {
				return cleaned, usedModel, true, nil
			}
		}
		// 不记录为致命错误：有的视觉模型对 dataURL 挑剔，退回纯文本仍然能给好结果。
	}

	raw, err := h.completeChatOnce(ctx, usedModel, system, user, "")
	if err != nil {
		// 跟随聊天模型时（非 strict），本地 Ollama 可能刚好被删/掉线，上游直接 404。
		// 这种「跟随」场景不该让用户看一条上游原始报错，悄悄用服务端默认再试一次。
		if !req.Strict && strings.HasPrefix(usedModel, "ollama:") {
			if fallback := fallbackString(h.cfg.MiniMax.Model, "MiniMax-M2.5"); strings.TrimSpace(h.cfg.MiniMax.APIKey) != "" {
				if raw2, err2 := h.completeChatOnce(ctx, fallback, system, user, ""); err2 == nil {
					if cleaned := cleanOptimizedPrompt(raw2); cleaned != "" {
						return cleaned, fallback, false, nil
					}
				}
			}
		}
		return "", usedModel, false, err
	}
	cleaned := cleanOptimizedPrompt(raw)
	if cleaned == "" {
		return "", usedModel, false, fmt.Errorf("模型没有返回可用的提示词，请重试")
	}
	return cleaned, usedModel, false, nil
}

// promptOptimizeCanSeeImage 判断这个模型能不能吃图：必须是 ollama:xxx，
// 且 /api/show 或名称启发式判定为视觉模型。
func (h *AIGatewayHandler) promptOptimizeCanSeeImage(model string) bool {
	name, ok := strings.CutPrefix(model, "ollama:")
	if !ok || strings.TrimSpace(name) == "" {
		return false
	}
	prov := h.resolveOllamaProviderConfig()
	if prov == nil {
		return false
	}
	return h.ollamaModelVision(prov.APIURL, name)
}

// completeChatOnce 单轮非流式补全；image 非空时按 OpenAI 多模态格式带上参考图。
func (h *AIGatewayHandler) completeChatOnce(ctx context.Context, model, system, user, image string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	userContent := interface{}(user)
	if image != "" {
		userContent = []map[string]interface{}{
			{"type": "text", "text": user},
			{"type": "image_url", "image_url": map[string]interface{}{"url": image}},
		}
	}

	messages := make([]map[string]interface{}, 0, 2)
	if system != "" {
		messages = append(messages, map[string]interface{}{"role": "system", "content": system})
	}
	messages = append(messages, map[string]interface{}{"role": "user", "content": userContent})

	// 提示词改写要稳不要飘：低温 + 收住长度，避免每次给出风格迥异的长篇。
	temp := 0.4
	maxTokens := 1200
	result, _, err := h.executeChatRequest(ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Stream:      false,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	})
	if err != nil {
		return "", err
	}
	// executeChatRequest 目前没有 ctx 参数（下面几个 provider 各自设了读超时），
	// 所以取消/超时打断不了已经在飞的上游请求；但它回来时必须让结果作废——
	// 否则用户已经关掉弹窗/换了一次优化，旧结果还会被当成成功写进任务。
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	if result == nil {
		return "", fmt.Errorf("上游无响应")
	}
	content, _ := result["content"].(string)
	if content == "" {
		if choices, ok := result["choices"].([]interface{}); ok {
			content = extractMessageContentFromChoices(choices)
		}
	}
	return content, nil
}

// imagePromptOptimizeSystem 系统提示词。
// 重点是三条：只输出正文、不许改用户原意、不许写无意义的画质堆砌词。
func imagePromptOptimizeSystem() string {
	return strings.Join([]string{
		"你是扩散模型（ComfyUI 生态，含 Qwen-Image 与 FLUX 系）的提示词工程师。",
		"用户会把一句大白话、甚至几个词丢给你，你负责把它改写成能直接出好图的高质量提示词。",
		"",
		"必须遵守：",
		"1. 只输出提示词正文：不要解释、不要标题、不要引号、不要 Markdown、不要「提示词：」这类前缀。",
		"2. 忠实于用户原意：用户提到的元素一个都不能丢；不要替换主体，也不要添加用户没提的关键人物或物体。",
		"3. 允许补充的只有让画面成立的细节：光线、氛围、材质、构图、镜头、色彩关系、画风。补充要具体可画，不要空话。",
		"4. 不要写「8K、超高清、杰作、best quality、masterpiece」这类对现代扩散模型无意义的堆砌词；不要写负面提示词，也不要写任何参数（--ar、steps、cfg）。",
		"5. 输出一段（可分句），中文 60~160 字；若用户原话是英文，则输出英文 40~100 词。",
	}, "\n")
}

// buildImagePromptOptimizeUser 把模式差异写进用户消息。三种模式的写法要求完全不同：
// 改图必须写成「编辑指令 + 保持不变清单」，否则模型会把整张图重画。
func buildImagePromptOptimizeUser(prompt, mode string, req imagePromptOptimizeRequest) string {
	var b strings.Builder

	switch mode {
	case promptModeDescribe:
		b.WriteString("【任务】用户给了一张参考图，但没写要求。\n")
		b.WriteString("先仔细看懂这张图（主体、动作、构图、场景、光线、色调、画风、镜头），")
		b.WriteString("再写一条能复现同一画面与风格的出图提示词。不要写「这张图」「如图所示」这类指代。\n")
	case promptModeEdit:
		b.WriteString("【任务】图生图（改图）：在用户给的参考图上做修改。\n")
		b.WriteString("写成「编辑指令」，而不是重新描述整张图：先说清改哪里、改成什么样（颜色、材质、文字、数量等细节写具体），")
		b.WriteString("再明确写出必须保持不变的部分（主体身份与五官、姿态、构图与视角、服装、画风、光线）。\n")
	default:
		b.WriteString("【任务】文生图：从零生成一张图。\n")
		b.WriteString("按「主体与动作 → 场景与背景 → 光线与氛围 → 镜头与构图 → 风格与质感」的顺序组织成连贯自然语言。")
		b.WriteString("主体要具体（外貌、服饰、动作），不要只写抽象名词。\n")
	}

	// 目标出图模型只影响写法偏好：FLUX 系对英文更稳，Qwen 系中文理解好。
	if label := strings.TrimSpace(req.ImageModel); label != "" {
		fmt.Fprintf(&b, "【目标出图模型】%s\n", label)
	}
	if strings.EqualFold(strings.TrimSpace(req.ImageFamily), comfyFamilyFlux2) && prompt == "" {
		b.WriteString("【语言】该模型对英文提示词更稳，请用英文输出。\n")
	}

	if prompt != "" {
		fmt.Fprintf(&b, "【用户原话】%s\n", prompt)
	}

	b.WriteString("【输出】只输出改写后的提示词正文。")
	return b.String()
}

// promptOptimizePrefixes 模型常见的「话痨前缀」，按长度从长到短匹配。
var promptOptimizePrefixes = []string{
	"优化后的提示词：", "优化后的提示词:", "改写后的提示词：", "改写后的提示词:",
	"提示词：", "提示词:", "提示词是：", "优化后：", "优化后:", "改写后：", "改写后:",
	"输出：", "prompt：", "prompt:", "Prompt:", "PROMPT:", "**提示词**：", "**Prompt**：",
}

// cleanOptimizedPrompt 清洗模型输出：去代码围栏、去前缀、去包裹引号、压成一段、截断。
// 模型再听话也可能加一句解释，这里兜住，保证塞回输入框的是干净的提示词。
func cleanOptimizedPrompt(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	// ```text ... ``` 代码围栏
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
		s = strings.TrimSpace(s)
	}

	// 前缀可能叠着出现（「提示词：优化后：…」），最多剥三层
	for i := 0; i < 3; i++ {
		trimmed := false
		for _, p := range promptOptimizePrefixes {
			if strings.HasPrefix(s, p) {
				s = strings.TrimSpace(strings.TrimPrefix(s, p))
				trimmed = true
				continue
			}
		}
		if !trimmed {
			break
		}
	}

	// 包裹引号：模型很喜欢给整段加引号
	s = strings.TrimSpace(strings.Trim(s, "\"'`“”‘’「」『』"))

	// 多行合并成一段：扩散模型编码器不关心换行，多行只会让用户复制时多带空行
	s = strings.Join(strings.Fields(s), " ")

	if utf8.RuneCountInString(s) > promptOptimizeMaxOutput {
		s = string([]rune(s)[:promptOptimizeMaxOutput])
	}
	return strings.TrimSpace(s)
}
