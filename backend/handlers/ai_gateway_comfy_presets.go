package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"devtools/config"
)

// ── 出图模型预设 ────────────────────────────────────────────────────────────
//
// 一个「预设」= 一整条可用工作流，而不只是 unet 文件名。三件东西必须配套，
// 单独换其中一个都会报错或出垃圾图：
//   - unet：扩散模型本体
//   - clip + clip_type + clip_loader：文本编码器（不同家族维度不同，Qwen-Image 用
//     qwen3vl 4096 维，Qwen-Image-Edit 必须配 Qwen2.5-VL 3584 维，Flux2 配 qwen3-4b）
//   - vae：必须与模型家族一致（flux2_ae / qwen_image_2.1_vae / qwen_image_vae）
//
// 家族决定工作流形状（见 buildComfyWorkflow），也决定能力：
//   qwen21   → 文生图 + 参考图编辑（TextEncodeQwenImage21 单节点出正负+latent）
//   flux2    → 仅文生图（CLIPTextEncode + EmptyFlux2LatentImage）
//   qwen_edit→ 仅参考图编辑（EditPlus 编码 + VAEEncode 源图）
const (
	comfyFamilyQwen21   = "qwen21"
	comfyFamilyFlux2    = "flux2"
	comfyFamilyQwenEdit = "qwen_edit"
)

// builtinComfyPresets 内置预设。实测耗时来自本机 ComfyUI 的真实 history 记录。
var builtinComfyPresets = []config.ComfyImageModelPreset{
	{
		ID: "qwen-image-2.1", Label: "Qwen-Image 2.1", Family: comfyFamilyQwen21,
		Unet:       "qwen-image-2.1-UC-Q4_K_M.gguf",
		Clip:       "qwen3vl_8b_int8_convrot.safetensors",
		ClipLoader: "CLIPLoader", ClipType: "qwen_image",
		Vae:        "qwen_image_2.1_vae_bf16.safetensors",
		Steps:      25, CFG: 1.0, Sampler: "euler", Scheduler: "simple", Resolution: 1024,
		SpeedHint: "文生图约 3.5 分钟，改图约 5 分钟",
	},
	{
		ID: "flux-2-klein", Label: "FLUX.2 Klein 4B", Family: comfyFamilyFlux2,
		Unet:       "flux-2-klein-4b-Q4_K_M.gguf",
		Clip:       "qwen3-4b-te-q5_k_m.gguf",
		ClipLoader: "CLIPLoaderGGUF", ClipType: "flux2",
		Vae:        "flux2_ae.safetensors",
		Steps:      4, CFG: 1.0, Sampler: "euler", Scheduler: "simple", Resolution: 1024,
		SpeedHint: "约 20~30 秒，只要文生图",
	},
}

// comfyPresets 合并内置预设与配置覆盖：同 id 的配置项整条替换内置项，
// 新 id 追加在后面。返回顺序稳定（内置在前），前端下拉框顺序固定。
func comfyPresets(cfg config.ComfyUIConfig) []config.ComfyImageModelPreset {
	out := make([]config.ComfyImageModelPreset, 0, len(builtinComfyPresets)+len(cfg.Models))
	out = append(out, builtinComfyPresets...)
	index := make(map[string]int, len(out))
	for i, p := range out {
		index[p.ID] = i
	}
	for _, override := range cfg.Models {
		id := strings.TrimSpace(override.ID)
		if id == "" {
			continue
		}
		override.ID = id
		if override.Label == "" {
			override.Label = id
		}
		if i, ok := index[id]; ok {
			out[i] = comfyFillPresetDefaults(override)
			continue
		}
		index[id] = len(out)
		out = append(out, comfyFillPresetDefaults(override))
	}
	return out
}

// comfyFillPresetDefaults 给配置里漏写的字段兜底，避免 0 步 / 空采样器直接进 ComfyUI。
func comfyFillPresetDefaults(p config.ComfyImageModelPreset) config.ComfyImageModelPreset {
	p.Family = strings.TrimSpace(p.Family)
	if p.Family == "" {
		p.Family = comfyFamilyQwen21
	}
	if p.Steps <= 0 {
		p.Steps = 25
	}
	if p.CFG <= 0 {
		p.CFG = 1.0
	}
	p.Sampler = strings.TrimSpace(p.Sampler)
	if p.Sampler == "" {
		p.Sampler = "euler"
	}
	p.Scheduler = strings.TrimSpace(p.Scheduler)
	if p.Scheduler == "" {
		p.Scheduler = "simple"
	}
	if p.Resolution <= 0 {
		p.Resolution = 1024
	}
	p.ClipLoader = strings.TrimSpace(p.ClipLoader)
	if p.ClipLoader == "" {
		p.ClipLoader = "CLIPLoader"
	}
	if p.ClipType == "" {
		p.ClipType = "qwen_image"
	}
	return p
}

// comfySupportsT2I / comfySupportsEdit 由家族推导能力，配置不用手写这两个开关。
func comfySupportsT2I(p config.ComfyImageModelPreset) bool {
	return p.Family != comfyFamilyQwenEdit
}

func comfySupportsEdit(p config.ComfyImageModelPreset) bool {
	return p.Family != comfyFamilyFlux2
}

// comfyDefaultPresetID 取配置里的默认预设；配了不存在的 id 就退回第一个。
func comfyDefaultPresetID(cfg config.ComfyUIConfig) string {
	def := strings.TrimSpace(cfg.DefaultModel)
	if def != "" {
		for _, p := range comfyPresets(cfg) {
			if p.ID == def {
				return def
			}
		}
	}
	if len(builtinComfyPresets) > 0 {
		return builtinComfyPresets[0].ID
	}
	return def
}

// comfyPresetByID 按 id 找预设；id 为空时返回默认预设（兼容老调用方）。
func comfyPresetByID(cfg config.ComfyUIConfig, id string) (config.ComfyImageModelPreset, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = comfyDefaultPresetID(cfg)
	}
	for _, p := range comfyPresets(cfg) {
		if p.ID == id {
			return p, true
		}
	}
	return config.ComfyImageModelPreset{}, false
}

// comfyPresetIDs 返回所有可选预设 id，用于错误提示。
func comfyPresetIDs(cfg config.ComfyUIConfig) []string {
	presets := comfyPresets(cfg)
	ids := make([]string, 0, len(presets))
	for _, p := range presets {
		ids = append(ids, p.ID)
	}
	return ids
}

// applyTo 把预设叠加到运行期配置上，得到「这条请求实际要用的」ComfyUI 配置。
// 采样器、步数、分辨率等也一并覆盖——否则用 flux2 的 4 步模型却跑 25 步会白白多花 6 倍时间。
func comfyApplyPreset(cfg config.ComfyUIConfig, p config.ComfyImageModelPreset) config.ComfyUIConfig {
	out := cfg
	out.Model = p.Unet
	out.Clip = p.Clip
	out.Vae = p.Vae
	out.ClipType = p.ClipType
	out.ClipLoader = p.ClipLoader
	out.Family = p.Family
	out.Steps = p.Steps
	out.CFG = p.CFG
	out.Sampler = p.Sampler
	out.Scheduler = p.Scheduler
	out.Resolution = p.Resolution
	return out
}

// ── 模型可用性探测 ──────────────────────────────────────────────────────────

// comfyNodeChoices 读某个节点某个下拉字段的可选值（ComfyUI 会把这些值填成
// 「磁盘上实际存在」的文件名），用来判断预设里的文件在不在。
func (c *comfyClient) nodeChoices(ctx context.Context, node, field string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/object_info/"+node, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ComfyUI 返回 HTTP %d", resp.StatusCode)
	}
	var body map[string]struct {
		Input struct {
			Required map[string]json.RawMessage `json:"required"`
		} `json:"input"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	raw, ok := body[node].Input.Required[field]
	if !ok {
		return nil, nil
	}
	// 形如 [[ "a", "b" ], { "tooltip": ... }]，第一项是候选值数组
	var spec []json.RawMessage
	if err := json.Unmarshal(raw, &spec); err != nil || len(spec) == 0 {
		return nil, nil
	}
	var names []string
	if err := json.Unmarshal(spec[0], &names); err != nil {
		return nil, nil
	}
	return names, nil
}

// comfyProbeFiles 批量取 unet / clip / vae 的可用文件名。
// 任一步失败返回 nil —— 探测失败不该拦住出图，前端把「未知」当可用处理。
func (h *AIGatewayHandler) comfyProbeFiles(ctx context.Context, cfg config.ComfyUIConfig) map[string]map[string]bool {
	client := newComfyClient(cfg)
	out := map[string]map[string]bool{}
	unets, err := client.nodeChoices(ctx, "UnetLoaderGGUF", "unet_name")
	if err != nil {
		return nil
	}
	out["unet"] = comfyToSet(unets)
	// 两个 loader 都要看：qwen 系列走 CLIPLoader（safetensors），flux2 走 CLIPLoaderGGUF
	clips := map[string]bool{}
	for _, loader := range []string{"CLIPLoader", "CLIPLoaderGGUF"} {
		names, err := client.nodeChoices(ctx, loader, "clip_name")
		if err != nil {
			return nil
		}
		for _, n := range names {
			clips[n] = true
		}
	}
	out["clip"] = clips
	vaes, err := client.nodeChoices(ctx, "VAELoader", "vae_name")
	if err != nil {
		return nil
	}
	out["vae"] = comfyToSet(vaes)
	return out
}

func comfyToSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, s := range items {
		set[s] = true
	}
	return set
}

// comfyPresetMissing 返回预设里缺失的文件名（空 = 齐全）。
func comfyPresetMissing(p config.ComfyImageModelPreset, files map[string]map[string]bool) []string {
	if files == nil {
		return nil
	}
	var missing []string
	if !files["unet"][p.Unet] {
		missing = append(missing, p.Unet)
	}
	if !files["clip"][p.Clip] {
		missing = append(missing, p.Clip)
	}
	if !files["vae"][p.Vae] {
		missing = append(missing, p.Vae)
	}
	return missing
}

// ChatPlusImageModels GET /api/chat-plus/image/models —— 可切换的出图模型清单。
func (h *AIGatewayHandler) ChatPlusImageModels(c *gin.Context) {
	if !h.checkChatPlusPassword(c) {
		return
	}
	cfg := h.cfg.ChatPlus.ComfyUI.Normalized()
	presets := comfyPresets(cfg)

	list := make([]gin.H, 0, len(presets))
	if cfg.Enabled() {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
		defer cancel()
		files := h.comfyProbeFiles(ctx, cfg)
		for _, p := range presets {
			missing := comfyPresetMissing(p, files)
			list = append(list, gin.H{
				"id":            p.ID,
				"label":         p.Label,
				"family":        p.Family,
				"steps":         p.Steps,
				"cfg":           p.CFG,
				"resolution":    p.Resolution,
				"supports_t2i":  comfySupportsT2I(p),
				"supports_edit": comfySupportsEdit(p),
				"speed_hint":    p.SpeedHint,
				"available":     len(missing) == 0,
				"missing":       missing,
				"probed":        files != nil,
			})
		}
	} else {
		for _, p := range presets {
			list = append(list, gin.H{
				"id": p.ID, "label": p.Label, "family": p.Family,
				"steps": p.Steps, "cfg": p.CFG, "resolution": p.Resolution,
				"supports_t2i": comfySupportsT2I(p), "supports_edit": comfySupportsEdit(p),
				"speed_hint": p.SpeedHint, "available": false, "missing": []string{}, "probed": false,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"models":        list,
		"default_model": comfyDefaultPresetID(cfg),
		"base_url":      cfg.BaseURL,
	})
}
