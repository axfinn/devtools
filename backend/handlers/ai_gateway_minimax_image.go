package handlers

// 本文件：把 MiniMax 在线出图（image-01）接进 /ai-chat 的「画图」模式。
//
// 为什么要有云端通道：ComfyUI 是局域网自建，免费但慢（4~25 步，20~75 秒，图生图首帧
// 还要加载权重）；MiniMax image-01 是云端按量计费、10 秒级返回，且自带
// subject_reference（参考主体/图生图），在手机弱网、临时想看效果时明显更好用。
//
// 关键设计：**复用 ComfyUI 那套任务模型**（comfyTask / comfyTaskStore / SSE 推进度 /
// 终态落 chat_plus_messages）。前端完全不用感知是本地还是云端，只是下拉框里多一个
// 模型 id —— 换供应商不换交互，也不会出现两套进度语义。
//
// 上游接口：POST {base_url}/v1/image_generation
//
//	{ "model": "image-01", "prompt": "...", "aspect_ratio": "1:1",
//	  "response_format": "url", "n": 1, "prompt_optimizer": true,
//	  "subject_reference": [{ "type": "character", "image_file": "<data URL 或公网 URL>" }] }
//	→ { "data": { "image_urls": [...] }, "base_resp": { "status_code": 0, ... } }
//
// 两个实测坑（2026-10-30 验证）：
//  1. 参考图必须传 **data URL**（`data:image/jpeg;base64,...`）。传裸 base64 会被当成
//     普通 URL 去解析，直接报 `disallowed image url: localhost or private address not allowed`。
//  2. 上游返回的是带签名的 OSS 临时链接（Expires 约 24h），不能直接存进消息里当长期
//     URL —— 所以这里统一「下载 → 落盘 → 回本服务的 /api/chat/uploads/... 地址」，
//     与本地图链路完全一致，过期问题一次性消失。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"devtools/config"
)

const (
	miniMaxImagePath        = "/v1/image_generation"
	miniMaxImageMaxPrompt   = 1500                   // image-01 官方提示词上限 1500 字符
	miniMaxImageMaxRefBytes = 8 << 20                // 参考图原始字节上限（base64 后约 10.7MB，再大容易顶掉上游 body 限制）
	miniMaxImagePollGap     = 300 * time.Millisecond // 只是给 UI 一点「已提交」的观感，上游是同步返回的
)

// miniMaxImageClient 走环境代理：MiniMax 与其返回的 OSS 链接都在公网，
// 容器里若配了 HTTP_PROXY 就该用上（与 ComfyUI 的「直连局域网、禁代理」正好相反）。
var miniMaxImageClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          8,
		IdleConnTimeout:       60 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
	},
}

// miniMaxSubjectRef 上游的参考主体结构。type 目前只有 character（人物/主体一致性）。
type miniMaxSubjectRef struct {
	Type      string `json:"type"`
	ImageFile string `json:"image_file"`
}

// miniMaxImageResponse 上游响应。data 可能为 null（如参考图地址不合法、内容审核拒绝）。
type miniMaxImageResponse struct {
	ID   string `json:"id"`
	Data *struct {
		ImageURLs []string `json:"image_urls"`
	} `json:"data"`
	Metadata struct {
		FailedCount  string `json:"failed_count"`
		SuccessCount string `json:"success_count"`
	} `json:"metadata"`
	BaseResp struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

// miniMaxAspectFromRequest 决定上游的 aspect_ratio。
// 优先级与 comfyResolveParams 保持一致：size（OpenAI 风格）> width/height > 正方形分辨率。
// 云端不认「1024x1536 这种任意尺寸」，只认 8 个固定比例，所以这里折算成最接近的一个。
func miniMaxAspectFromRequest(req comfyImageRequest) string {
	if w, h, ok := parseImageSize(strings.TrimSpace(req.Size)); ok {
		return miniMaxAspectRatio(w, h)
	}
	if req.Width > 0 && req.Height > 0 {
		return miniMaxAspectRatio(req.Width, req.Height)
	}
	if req.AspectRatio != "" && miniMaxAspectRatioValid(req.AspectRatio) {
		return req.AspectRatio
	}
	return "1:1"
}

// miniMaxAspectRatioValid 校验前端直接传的比例是否是上游枚举之一。
func miniMaxAspectRatioValid(v string) bool {
	switch strings.TrimSpace(v) {
	case "1:1", "16:9", "4:3", "3:2", "2:3", "3:4", "9:16", "21:9":
		return true
	}
	return false
}

// miniMaxAspectRatio 把宽高折算成上游枚举。
// 上游只认这 8 个固定比例，所以取「最接近」的那个而不是报错。
func miniMaxAspectRatio(width, height int) string {
	if width <= 0 || height <= 0 {
		return "1:1"
	}
	type ratio struct {
		name string
		v    float64
	}
	candidates := []ratio{
		{"1:1", 1.0 / 1.0},
		{"4:3", 4.0 / 3.0},
		{"3:2", 3.0 / 2.0},
		{"16:9", 16.0 / 9.0},
		{"21:9", 21.0 / 9.0},
		{"3:4", 3.0 / 4.0},
		{"2:3", 2.0 / 3.0},
		{"9:16", 9.0 / 16.0},
	}
	target := float64(width) / float64(height)
	best, bestDiff := candidates[0].name, -1.0
	for _, c := range candidates {
		diff := c.v - target
		if diff < 0 {
			diff = -diff
		}
		if bestDiff < 0 || diff < bestDiff {
			best, bestDiff = c.name, diff
		}
	}
	return best
}

// miniMaxImageFile 把前端传来的参考图统一成上游认的形态。
//
// 上游接受「公网可访问 URL」或「data URL」两种；我们自己的图片是
// /api/chat/uploads/... 这种站内相对路径，公网未必能访问，所以：
//  1. data URL → 原样传（最快，且已验证可用）
//  2. 站内相对路径 / 内网绝对地址 → 先自己取字节，再转 data URL
//  3. 公网 http(s) → 原样传，省一次下载；除非取字节更省心（这里选原样传）
func miniMaxImageFile(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "data:image/") {
		// 已经是 data URL：只做体积校验，避免把 20MB 的图塞进上游
		if idx := strings.Index(raw, ","); idx > 0 {
			if decoded := len(raw) - idx - 1; decoded*3/4 > miniMaxImageMaxRefBytes {
				return "", fmt.Errorf("参考图过大（超过 %d MB），请压缩后再试", miniMaxImageMaxRefBytes>>20)
			}
		}
		return raw, nil
	}

	// 站内相对路径或内网地址：自己取字节再转 data URL（上游抓不到我们的内网）
	needFetch := strings.HasPrefix(raw, "/") || isPrivateHTTPURL(raw)
	if !needFetch {
		// 公网 URL：上游能直接抓，省一次下载
		return raw, nil
	}

	data, err := miniMaxReadRefBytes(raw)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("参考图内容为空")
	}
	if len(data) > miniMaxImageMaxRefBytes {
		return "", fmt.Errorf("参考图过大（%d MB，上限 %d MB），请压缩后再试",
			len(data)>>20, miniMaxImageMaxRefBytes>>20)
	}
	return "data:" + miniMaxImageMime(data) + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// miniMaxImageMime 按魔数给出 data URL 的 MIME。
// ⚠️ 不能直接拼 imageExt() —— 它返回的是带点的扩展名（".jpg"），
// 拼出来会变成 image/.jpg，上游会回 2013 "the image format is not supported"。
func miniMaxImageMime(data []byte) string {
	switch imageExt(data) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	}
	return "image/jpeg"
}

// miniMaxReadRefBytes 取参考图原始字节。
//
// 刷新页面后历史消息里的图片是站内地址（/api/chat/uploads/ai/...），这类路径
// 既要避免出网自连（容器回环不一定通），也要避免上游拒绝内网 URL，
// 所以直接按 uploadDir 映射到本地文件读，最稳。
func miniMaxReadRefBytes(raw string) ([]byte, error) {
	if path, ok := savedImageRelPath(raw); ok {
		data, err := readSavedImage(comfyImage{RelPath: path})
		if err != nil {
			return nil, fmt.Errorf("读取本地参考图失败: %w", err)
		}
		return data, nil
	}
	data, err := decodeImagePayload(raw)
	if err != nil {
		return nil, fmt.Errorf("读取参考图失败: %w", err)
	}
	return data, nil
}

// savedImageRelPath 把站内图片 URL 还原成 uploadDir 下的相对路径。
// 认两种写法：/api/chat/uploads/xxx 与带域名的 https://host/api/chat/uploads/xxx。
func savedImageRelPath(raw string) (string, bool) {
	const marker = "/api/chat/uploads/"
	idx := strings.Index(raw, marker)
	if idx < 0 {
		return "", false
	}
	rel := raw[idx+len(marker):]
	if i := strings.IndexAny(rel, "?#"); i >= 0 {
		rel = rel[:i]
	}
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || strings.Contains(rel, "..") {
		return "", false
	}
	return filepath.FromSlash(rel), true
}

// isPrivateHTTPURL 粗判「内网 / 回环」地址。上游对这些地址会直接拒绝，必须自己转 data URL。
func isPrivateHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := u.Hostname()
	switch host {
	case "", "localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal":
		return true
	}
	// 10.0.0.0/8、192.168.0.0/16、172.16.0.0/12 这类内网段：只做前缀判断足够了
	return strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "172.16.") ||
		strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") ||
		strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.2") ||
		strings.HasPrefix(host, "172.30.") ||
		strings.HasPrefix(host, "172.31.")
}

// runMiniMaxImageTask 云端出图主链路：提交 → 下载 → 落盘 → 落任务终态。
// 与 runComfyTask 的差别只是「怎么拿到图」，任务状态机与前端协议完全一致。
func (h *AIGatewayHandler) runMiniMaxImageTask(cfg config.ComfyUIConfig, taskID string, req comfyImageRequest, prompt, negative string, opts comfyGenOptions, mode string) {
	mm := h.cfg.ChatPlus.MiniMaxImage.Normalized(h.cfg.MiniMax.APIKey)
	if !mm.Enabled() {
		h.failComfyTask(taskID, "未配置 chat_plus.minimax_image.api_key，在线出图不可用")
		return
	}

	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Status = "running"
		t.Phase = "提交到 MiniMax"
		t.started = time.Now()
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mm.TimeoutSec)*time.Second)
	defer cancel()

	// 1) 参考图（图生图 / 参考主体）。上游要求 data URL，本地存储的图我们自己转。
	var refFile string
	if mode == "edit" {
		raw := strings.TrimSpace(req.Image)
		if raw == "" && len(req.Images) > 0 {
			raw = req.Images[0]
		}
		if raw == "" {
			h.failComfyTask(taskID, "图生图模式需要一张参考图")
			return
		}
		comfyTasks.mutate(taskID, func(t *comfyTask) { t.Phase = "上传参考图" })
		f, err := miniMaxImageFile(raw)
		if err != nil {
			h.failComfyTask(taskID, err.Error())
			return
		}
		refFile = f
	}

	// 2) 组装上游请求体
	want := req.N
	if want <= 0 {
		want = 1
	}
	if want > mm.MaxImages {
		want = mm.MaxImages
	}
	body := map[string]interface{}{
		"model":            mm.Model,
		"prompt":           prompt,
		"aspect_ratio":     miniMaxAspectFromRequest(req),
		"response_format":  "url", // 拿 URL 后自己下载落盘，避免临时链接过期
		"n":                want,
		"prompt_optimizer": mm.OptimizeEnabled(),
	}
	if opts.Seed > 0 {
		body["seed"] = opts.Seed
	}
	if refFile != "" {
		body["subject_reference"] = []miniMaxSubjectRef{{Type: "character", ImageFile: refFile}}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		h.failComfyTask(taskID, "请求体序列化失败: "+err.Error())
		return
	}

	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Phase = "MiniMax 生成中"
		t.Progress = 40
	})

	// 3) 调用上游
	upURL := mm.BaseURL + miniMaxImagePath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upURL, bytes.NewReader(payload))
	if err != nil {
		h.failComfyTask(taskID, "构造请求失败: "+err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+mm.APIKey)

	resp, err := miniMaxImageClient.Do(httpReq)
	if err != nil {
		h.failComfyTask(taskID, miniMaxImageErrorText(err))
		return
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		h.failComfyTask(taskID, "读取上游响应失败: "+err.Error())
		return
	}
	if resp.StatusCode >= 400 {
		h.failComfyTask(taskID, fmt.Sprintf("MiniMax 返回 HTTP %d: %s", resp.StatusCode, truncateString(string(raw), 300)))
		return
	}

	var parsed miniMaxImageResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		h.failComfyTask(taskID, "上游响应解析失败: "+truncateString(string(raw), 200))
		return
	}
	if parsed.BaseResp.StatusCode != 0 {
		h.failComfyTask(taskID, fmt.Sprintf("MiniMax 拒绝了本次请求（%d）：%s",
			parsed.BaseResp.StatusCode, parsed.BaseResp.StatusMsg))
		return
	}
	if parsed.Data == nil || len(parsed.Data.ImageURLs) == 0 {
		h.failComfyTask(taskID, "MiniMax 没有返回图片（可能触发内容审核，换个描述再试）")
		return
	}

	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Phase = "下载图片"
		t.Progress = 80
	})

	// 4) 下载 + 落盘。上游链接 24h 过期，必须转存到本服务。
	images := make([]comfyImage, 0, len(parsed.Data.ImageURLs))
	var firstErr error
	for i, u := range parsed.Data.ImageURLs {
		data, err := miniMaxDownload(ctx, u)
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
		comfyTasks.mutate(taskID, func(t *comfyTask) {
			t.Progress = 80 + (i+1)*15/len(parsed.Data.ImageURLs)
		})
	}
	if len(images) == 0 {
		h.failComfyTask(taskID, fmt.Sprintf("成品图下载失败: %v", firstErr))
		return
	}

	comfyTasks.mutate(taskID, func(t *comfyTask) {
		t.Status = "succeeded"
		t.Phase = "完成"
		t.Progress = 100
		t.Images = images
		t.ImageURL = images[0].URL
	})
	log.Printf("[minimax-image] 任务 %s 成功，%d 张（mode=%s）", taskID, len(images), mode)
}

// miniMaxDownload 取上游成品图字节。
func miniMaxDownload(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := miniMaxImageClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("下载返回 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("下载内容为空")
	}
	return data, nil
}

// miniMaxImageErrorText 把网络错误翻成人话。最常见的两种：容器不通外网、
// 以及 key 没有 image-01 权限（后者会走 HTTP 4xx，由调用方处理）。
func miniMaxImageErrorText(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "context deadline exceeded") {
		return "MiniMax 请求超时（网络不通或上游排队），可调大 chat_plus.minimax_image.timeout_sec"
	}
	if strings.Contains(msg, "no such host") || strings.Contains(msg, "dial tcp") {
		return "连不上 MiniMax（检查容器出网 / DNS）：" + truncateString(msg, 200)
	}
	return "调用 MiniMax 失败: " + truncateString(msg, 300)
}
