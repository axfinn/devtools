package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"devtools/config"
)

// 本文件覆盖本地出图链路里所有「纯逻辑」：工作流构造、参数收敛、图片头解析、任务表。
// 网络相关的部分（ComfyUI 的 /prompt、/history、/view）靠 httptest 单独测，见
// TestComfyClientUploadAndQueue。

func testComfyConfig() config.ComfyUIConfig {
	return config.ComfyUIConfig{
		BaseURL:    "http://127.0.0.1:8188",
		Model:      "m.gguf",
		Clip:       "c.safetensors",
		Vae:        "v.safetensors",
		ClipType:   "qwen_image",
		Steps:      25,
		CFG:        1.0,
		Sampler:    "euler",
		Scheduler:  "simple",
		Resolution: 1024,
		TimeoutSec: 60,
	}
}

func workflowInputs(t *testing.T, wf map[string]interface{}, node string) map[string]interface{} {
	t.Helper()
	raw, ok := wf[node]
	if !ok {
		t.Fatalf("工作流缺少节点 %s", node)
	}
	n, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("节点 %s 结构异常", node)
	}
	inputs, ok := n["inputs"].(map[string]interface{})
	if !ok {
		t.Fatalf("节点 %s 缺少 inputs", node)
	}
	return inputs
}

func TestBuildComfyWorkflow_TextToImage(t *testing.T) {
	cfg := testComfyConfig()
	wf := buildComfyWorkflow(cfg, "a red apple", "", comfyGenOptions{
		Steps: 12, CFG: 2.5, Seed: 42, Resolution: 768,
	})

	// 所有必需节点都在
	for _, id := range []string{"1", "2", "3", "5", "8", "9", "10"} {
		if _, ok := wf[id]; !ok {
			t.Fatalf("缺少节点 %s", id)
		}
	}
	// 纯文生图不应出现 LoadImage
	if _, ok := wf["4"]; ok {
		t.Fatal("文生图模式不应包含 LoadImage 节点")
	}

	enc := workflowInputs(t, wf, "5")
	if enc["prompt"] != "a red apple" {
		t.Fatalf("prompt 未透传: %v", enc["prompt"])
	}
	if enc["resolution"] != 768 {
		t.Fatalf("resolution 未透传: %v", enc["resolution"])
	}
	// Autogrow 的扁平键名不该出现在纯文生图里
	if _, ok := enc["images.image_1"]; ok {
		t.Fatal("文生图模式不应携带参考图输入")
	}

	ks := workflowInputs(t, wf, "8")
	if ks["steps"] != 12 {
		t.Fatalf("steps 未透传: %v", ks["steps"])
	}
	if ks["cfg"] != 2.5 {
		t.Fatalf("cfg 未透传: %v", ks["cfg"])
	}
	// seed 是 int64，JSON 序列化后才不会被截断
	if seed, ok := ks["seed"].(int64); !ok || seed != 42 {
		t.Fatalf("seed 类型/取值异常: %T %v", ks["seed"], ks["seed"])
	}
	if ks["sampler_name"] != "euler" || ks["scheduler"] != "simple" {
		t.Fatalf("采样器/调度器未从配置读取: %v / %v", ks["sampler_name"], ks["scheduler"])
	}
}

func TestBuildComfyWorkflow_EditMode(t *testing.T) {
	cfg := testComfyConfig()
	wf := buildComfyWorkflow(cfg, "make it sunset", "", comfyGenOptions{
		Steps: 25, CFG: 1.0, Seed: 1, Resolution: 1024, ImageName: "aichat/ref.png",
	})

	load := workflowInputs(t, wf, "4")
	if load["image"] != "aichat/ref.png" {
		t.Fatalf("参考图名未透传: %v", load["image"])
	}

	enc := workflowInputs(t, wf, "5")
	link, ok := enc["images.image_1"].([]interface{})
	if !ok {
		// 回归保护：Autogrow 输入写成嵌套 dict 会被执行器静默丢弃，
		// 表现为"图没接上、模型凭空生成"，必须用扁平点号键名。
		t.Fatal(`编辑模式必须使用扁平键名 "images.image_1"，否则参考图会被静默忽略`)
	}
	if len(link) != 2 || link[0] != "4" || link[1] != 0 {
		t.Fatalf("参考图连线异常: %v", link)
	}
}

func TestComfyResolveParams(t *testing.T) {
	cfg := config.ComfyUIConfig{Steps: 25, CFG: 1.0, Resolution: 1024}

	t.Run("默认走服务端配置", func(t *testing.T) {
		opts, err := comfyResolveParams(cfg, comfyImageRequest{})
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		if opts.Steps != 25 || opts.CFG != 1.0 || opts.Resolution != 1024 {
			t.Fatalf("默认值异常: %+v", opts)
		}
		if opts.Seed <= 0 {
			t.Fatalf("未指定 seed 时应随机生成，得到 %d", opts.Seed)
		}
	})

	t.Run("size 按短边收敛并对齐 64", func(t *testing.T) {
		opts, err := comfyResolveParams(cfg, comfyImageRequest{Size: "1344x768"})
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		if opts.Resolution != 768 {
			t.Fatalf("短边应为 768，得到 %d", opts.Resolution)
		}
		opts, err = comfyResolveParams(cfg, comfyImageRequest{Width: 1000, Height: 800})
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		if opts.Resolution%64 != 0 {
			t.Fatalf("分辨率必须 64 对齐，得到 %d", opts.Resolution)
		}
		if opts.Resolution != 768 {
			t.Fatalf("1000x800 应按短边 800 收敛到 768，得到 %d", opts.Resolution)
		}
	})

	t.Run("非法 size 报错", func(t *testing.T) {
		if _, err := comfyResolveParams(cfg, comfyImageRequest{Size: "huge"}); err == nil {
			t.Fatal("非法 size 应报错")
		}
	})

	t.Run("越界分辨率报错", func(t *testing.T) {
		if _, err := comfyResolveParams(cfg, comfyImageRequest{Resolution: 4096}); err == nil {
			t.Fatal("超过 2048 应报错")
		}
	})

	t.Run("显式 seed 不被覆盖", func(t *testing.T) {
		seed := int64(777)
		opts, _ := comfyResolveParams(cfg, comfyImageRequest{Seed: &seed})
		if opts.Seed != 777 {
			t.Fatalf("seed 被覆盖: %d", opts.Seed)
		}
	})

	t.Run("steps 上限收敛", func(t *testing.T) {
		opts, _ := comfyResolveParams(cfg, comfyImageRequest{Steps: 9999})
		if opts.Steps != 150 {
			t.Fatalf("steps 应被夹到 150，得到 %d", opts.Steps)
		}
	})
}

func TestParseImageSize(t *testing.T) {
	cases := []struct {
		in   string
		w, h int
		ok   bool
	}{
		{"1024x1024", 1024, 1024, true},
		{"1024X768", 1024, 768, true},
		{"1024*768", 1024, 768, true},
		{" 512 x 512 ", 512, 512, true},
		{"1024", 0, 0, false},
		{"axb", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := parseImageSize(c.in)
		if ok != c.ok || (ok && (w != c.w || h != c.h)) {
			t.Fatalf("parseImageSize(%q) = %d,%d,%v; 期望 %d,%d,%v", c.in, w, h, ok, c.w, c.h, c.ok)
		}
	}
}

// pngHeader 拼一个只有头部的 PNG（IHDR 里带宽高），用于测维度解析。
func pngHeader(w, h uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte("\x89PNG\r\n\x1a\n"))
	_ = binary.Write(&buf, binary.BigEndian, uint32(13))
	buf.WriteString("IHDR")
	_ = binary.Write(&buf, binary.BigEndian, w)
	_ = binary.Write(&buf, binary.BigEndian, h)
	buf.Write([]byte{0, 0, 0, 0}) // IHDR 的 CRC，解析不需要
	return buf.Bytes()
}

// jpegHeader 拼一个 SOF0 段带宽高的最小 JPEG 头。
func jpegHeader(w, h uint16) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xD8})                         // SOI
	buf.Write([]byte{0xFF, 0xE0, 0x00, 0x04, 0x00, 0x00}) // APP0（长度 4，内容 2 字节）
	buf.Write([]byte{0xFF, 0xC0, 0x00, 0x11, 0x08})       // SOF0，长度 17，精度 8
	_ = binary.Write(&buf, binary.BigEndian, h)
	_ = binary.Write(&buf, binary.BigEndian, w)
	buf.Write(make([]byte, 9)) // 其余分量信息，解析不需要
	return buf.Bytes()
}

func TestImageDimensions(t *testing.T) {
	if w, h := imageDimensions(pngHeader(1000, 800)); w != 1000 || h != 800 {
		t.Fatalf("PNG 尺寸解析错误: %dx%d", w, h)
	}
	if w, h := imageDimensions(jpegHeader(640, 480)); w != 640 || h != 480 {
		t.Fatalf("JPEG 尺寸解析错误: %dx%d", w, h)
	}
	if w, h := imageDimensions([]byte("not an image")); w != 0 || h != 0 {
		t.Fatalf("非图片应返回 0,0，得到 %dx%d", w, h)
	}
}

func TestImageExt(t *testing.T) {
	if got := imageExt(pngHeader(8, 8)); got != ".png" {
		t.Fatalf("PNG 魔数识别失败: %s", got)
	}
	if got := imageExt([]byte{0xFF, 0xD8, 0xFF, 0xE0}); got != ".jpg" {
		t.Fatalf("JPEG 魔数识别失败: %s", got)
	}
	if got := imageExt([]byte("RIFF????WEBPVP8 ")); got != ".webp" {
		t.Fatalf("WebP 魔数识别失败: %s", got)
	}
}

func TestComfyTaskStore(t *testing.T) {
	store := newComfyTaskStore()
	task := &comfyTask{ID: "img_1", Status: "queued", Phase: "排队中", CreatedAt: time.Now()}
	store.put(task)

	got, ok := store.get("img_1")
	if !ok || got.Status != "queued" {
		t.Fatalf("写入后应能读到: %+v", got)
	}

	// 未开始的计时不计 elapsed
	if _, ok := store.mutate("img_1", func(t *comfyTask) {
		t.Status = "running"
		t.started = time.Now().Add(-2 * time.Second)
	}); !ok {
		t.Fatal("mutate 应命中任务")
	}
	got, _ = store.get("img_1")
	if got.ElapsedMS < 1000 {
		t.Fatalf("running 后应统计耗时，得到 %d", got.ElapsedMS)
	}

	if _, ok := store.mutate("nope", func(*comfyTask) {}); ok {
		t.Fatal("不存在的任务不应命中")
	}
	if _, ok := store.get("nope"); ok {
		t.Fatal("不存在的任务不应读到")
	}
}

func TestWaitComfyTask(t *testing.T) {
	task := &comfyTask{ID: "img_wait", Status: "running", Phase: "采样中", CreatedAt: time.Now()}
	comfyTasks.put(task)
	defer func() {
		comfyTasks.mu.Lock()
		delete(comfyTasks.items, "img_wait")
		comfyTasks.mu.Unlock()
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		comfyTasks.mutate("img_wait", func(t *comfyTask) {
			t.Status = "succeeded"
			t.Images = []comfyImage{{URL: "/api/chat/uploads/ai/20260101/x.png", Filename: "x.png"}}
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	final, err := waitComfyTask(ctx, "img_wait")
	if err != nil {
		t.Fatalf("不应超时: %v", err)
	}
	if final.Status != "succeeded" || len(final.Images) != 1 {
		t.Fatalf("终态异常: %+v", final)
	}
}

func TestWaitComfyTask_Timeout(t *testing.T) {
	task := &comfyTask{ID: "img_slow", Status: "running", CreatedAt: time.Now()}
	comfyTasks.put(task)
	defer func() {
		comfyTasks.mu.Lock()
		delete(comfyTasks.items, "img_slow")
		comfyTasks.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if _, err := waitComfyTask(ctx, "img_slow"); err == nil {
		t.Fatal("应返回超时错误")
	}
}

func TestDecodeImagePayload(t *testing.T) {
	// 1x1 透明 PNG 的 base64
	const b64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	data, err := decodeImagePayload("data:image/png;base64," + b64)
	if err != nil {
		t.Fatalf("data URL 解析失败: %v", err)
	}
	if len(data) == 0 || imageExt(data) != ".png" {
		t.Fatalf("解码结果不是 PNG: %d 字节", len(data))
	}

	for _, bad := range []string{"", "ftp://x/y.png", "data:image/png;base64,", "https://"} {
		if _, err := decodeImagePayload(bad); err == nil {
			t.Fatalf("非法输入 %q 应报错", bad)
		}
	}
}

func TestComfyHistoryParsing(t *testing.T) {
	// 真实形状：outputs 按节点 id 分组，status.messages 是 [kind, payload] 对
	raw := `{
		"prompt-1": {
			"outputs": {"10": {"images": [{"filename": "aichat_00001_.png", "subfolder": "", "type": "output"}]}},
			"status": {"status_str": "success", "completed": true, "messages": [["execution_start", {}]]}
		}
	}`
	var all map[string]*comfyHistoryEntry
	if err := json.Unmarshal([]byte(raw), &all); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	refs := collectComfyImages(all["prompt-1"])
	if len(refs) != 1 || refs[0].Filename != "aichat_00001_.png" {
		t.Fatalf("图片收集异常: %+v", refs)
	}

	// 错误任务应能抽出异常信息
	errRaw := `{"outputs":{},"status":{"status_str":"error","completed":false,
		"messages":[["execution_error",{"node_id":"8","node_type":"KSampler","exception_message":"CUDA out of memory"}]]}}`
	var entry comfyHistoryEntry
	if err := json.Unmarshal([]byte(errRaw), &entry); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	msg := comfyExecutionError(&entry)
	if !strings.Contains(msg, "CUDA out of memory") || !strings.Contains(msg, "KSampler") {
		t.Fatalf("异常信息抽取不完整: %s", msg)
	}

	// 被 interrupt 的任务没有 exception_message，应给出「被中断」而不是「未返回详细原因」
	interruptRaw := `{"outputs":{},"status":{"status_str":"error","completed":false,
		"messages":[["execution_interrupted",{"prompt_id":"p1","node_id":"8","node_type":"KSampler"}]]}}`
	var interrupted comfyHistoryEntry
	if err := json.Unmarshal([]byte(interruptRaw), &interrupted); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if got := comfyExecutionError(&interrupted); !strings.Contains(got, "中断") {
		t.Fatalf("被中断的任务应提示中断: %s", got)
	}
}

func TestComfyErrorDetail(t *testing.T) {
	raw := []byte(`{"error":{"type":"prompt_outputs_failed_validation","message":"Prompt outputs failed validation"},
		"node_errors":{"1":{"errors":[{"message":"Value not in list: unet_name"}]}}}`)
	detail := comfyErrorDetail(raw)
	if !strings.Contains(detail, "Prompt outputs failed validation") || !strings.Contains(detail, "unet_name") {
		t.Fatalf("错误详情抽取不完整: %s", detail)
	}
	// 非 JSON 时退化为原文截断，不能 panic
	if got := comfyErrorDetail([]byte("<html>502</html>")); got == "" {
		t.Fatal("非 JSON 错误体应回退为原文")
	}
}

// ── 内置预设的工作流骨架 ────────────────────────────────────────────────────
// 每个预设都必须能独立拼出一条形状正确的 ComfyUI 工作流：
// 节点号对应关系一旦错（比如 KSampler 还指着 unet 而不是 LoRA/AuraFlow），
// 表现是出图质量莫名其妙变差或采样直接报错，很难从日志看出原因。

func wfNode(t *testing.T, wf map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	n, ok := wf[id].(map[string]interface{})
	if !ok {
		t.Fatalf("工作流缺少节点 %s", id)
	}
	return n
}

func wfClass(t *testing.T, wf map[string]interface{}, id string) string {
	t.Helper()
	c, _ := wfNode(t, wf, id)["class_type"].(string)
	return c
}

func wfModelSource(t *testing.T, wf map[string]interface{}) string {
	t.Helper()
	inputs := wfNode(t, wf, "8")["inputs"].(map[string]interface{})
	ref, ok := inputs["model"].([]interface{})
	if !ok || len(ref) == 0 {
		t.Fatal("KSampler 缺少 model 输入")
	}
	src, _ := ref[0].(string)
	return src
}

func TestBuiltinPresetWorkflowShape(t *testing.T) {
	for _, p := range builtinComfyPresets {
		p = comfyFillPresetDefaults(p)
		cfg := comfyApplyPreset(config.ComfyUIConfig{}, p)
		opts := comfyGenOptions{
			Steps: p.Steps, CFG: p.CFG, Seed: 1, Resolution: 1024,
			ImageName: "ref.png", // 给上参考图，让 edit 家族能拼出完整图
		}
		t.Run(p.ID, func(t *testing.T) {
			wf := buildComfyWorkflow(cfg, "一只猫", "", opts)

			if got := wfClass(t, wf, "8"); got != "KSampler" {
				t.Fatalf("节点 8 应是 KSampler，实际 %s", got)
			}

			switch p.Family {
			case comfyFamilyZImage:
				// 关键：model 必须先过 ModelSamplingAuraFlow，且编码器走 CLIPLoader（非 GGUF）
				if got := wfClass(t, wf, "7"); got != "ModelSamplingAuraFlow" {
					t.Fatalf("zimage 节点 7 应是 ModelSamplingAuraFlow，实际 %q", got)
				}
				if src := wfModelSource(t, wf); src != "7" {
					t.Fatalf("zimage 的 KSampler.model 应指向 7，实际 %s", src)
				}
				if got := wfClass(t, wf, "2"); got != "CLIPLoader" {
					t.Fatalf("zimage 必须用 CLIPLoader（GGUF loader 会导致维度不匹配），实际 %q", got)
				}
				if got := wfClass(t, wf, "5"); got != "EmptySD3LatentImage" {
					t.Fatalf("zimage latent 应是 EmptySD3LatentImage，实际 %q", got)
				}
				if cfg.Vae != "ae.safetensors" {
					t.Fatalf("zimage 必须配 Flux 的 ae.safetensors，实际 %q", cfg.Vae)
				}
			case comfyFamilyQwenEdit:
				if got := wfClass(t, wf, "5"); got != "TextEncodeQwenImageEditPlus" {
					t.Fatalf("edit 家族节点 5 应是 TextEncodeQwenImageEditPlus，实际 %q", got)
				}
				if got := wfClass(t, wf, "7"); got != "VAEEncode" {
					t.Fatalf("edit 家族的 latent 应来自 VAEEncode，实际 %q", got)
				}
			}

			if p.Lora != "" {
				if got := wfClass(t, wf, "11"); got != "LoraLoaderModelOnly" {
					t.Fatalf("挂了 LoRA 的预设应有 LoraLoaderModelOnly 节点，实际 %q", got)
				}
				if src := wfModelSource(t, wf); src != "11" {
					t.Fatalf("挂 LoRA 时 KSampler.model 应指向 11，实际 %s", src)
				}
			} else {
				if _, ok := wf["11"]; ok {
					t.Fatal("没声明 LoRA 的预设不该出现 LoRA 节点")
				}
				if p.Family != comfyFamilyZImage {
					if src := wfModelSource(t, wf); src != "1" {
						t.Fatalf("无 LoRA 时 KSampler.model 应指向 1，实际 %s", src)
					}
				}
			}
		})
	}
}

// 蒸馏版模型必须用少步数，否则白等 3~6 倍时间。
func TestPresetStepsMatchModelKind(t *testing.T) {
	want := map[string]int{
		"qwen-image-2.1-turbo":   8,
		"qwen-image-2.1":         25,
		"flux-2-klein":           4,
		"z-image-turbo":          8,
		"qwen-image-edit-2511":   4,
	}
	seen := map[string]bool{}
	for _, p := range builtinComfyPresets {
		if w, ok := want[p.ID]; ok {
			if p.Steps != w {
				t.Errorf("%s 步数 = %d，应为 %d", p.ID, p.Steps, w)
			}
			seen[p.ID] = true
		}
	}
	if len(seen) != len(want) {
		t.Errorf("内置预设缺少: 期望 %d 个，命中 %d 个", len(want), len(seen))
	}
}

// 默认预设必须是「日常」那档，不能让人默认撞上 75 秒的 25 步。
func TestDefaultPresetIsTurbo(t *testing.T) {
	if got := comfyDefaultPresetID(config.ComfyUIConfig{}); got != "qwen-image-2.1-turbo" {
		t.Fatalf("默认预设 = %q，应为 qwen-image-2.1-turbo", got)
	}
}
