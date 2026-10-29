package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

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
	promptOptimizeTimeout       = 150 * time.Second
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

// ChatPlusImagePromptOptimize POST /api/chat-plus/image/prompt/optimize
func (h *AIGatewayHandler) ChatPlusImagePromptOptimize(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}

	var req imagePromptOptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if utf8.RuneCountInString(prompt) > promptOptimizeMaxInput {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("原话过长（上限 %d 字），先精简一下再优化", promptOptimizeMaxInput),
		})
		return
	}

	// 参考图只接受 dataURL，且体积要可控；不合格就当没附图（降级而不是报错）
	image := strings.TrimSpace(req.Image)
	if !strings.HasPrefix(image, "data:image/") || len(image) > promptOptimizeMaxImageBytes {
		image = ""
	}

	mode := normalizePromptOptimizeMode(req.Mode, prompt, image)
	if mode == promptModeT2I && prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "先写一句你想画什么，或者附一张参考图"})
		return
	}

	model := strings.TrimSpace(req.Model)
	// 前端传的模型有两种来源：
	//   1) 用户在优化弹窗里明确选的（strict=true）→ 不可用就报错，不能偷偷换，
	//      否则「我选了 A 却拿到 B 的结果」，排查起来像见鬼；
	//   2) 跟随聊天页当前模型（strict=false）→ 聊天模型可能已被换成不可用引擎
	//      （Ollama 掉线等），这种情况回落到服务端默认，别让「优化提示词」跟着一起挂。
	if model != "" && h.resolveChatProvider(model) == "" {
		if req.Strict {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("所选优化模型当前不可用：%s（请换一个，或选「跟随聊天模型」）", model),
			})
			return
		}
		model = ""
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), promptOptimizeTimeout)
	defer cancel()

	out, usedModel, usedImage, err := h.optimizeImagePrompt(ctx, model, prompt, mode, req, image)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "优化失败：" + strings.TrimSpace(err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prompt":     out,
		"model":      usedModel,
		"mode":       mode,
		"used_image": usedImage,
	})
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
