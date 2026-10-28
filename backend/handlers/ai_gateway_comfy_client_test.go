package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devtools/config"

	"github.com/gin-gonic/gin"
)

// tinyPNGBase64 1x1 透明 PNG，用于构造「参考图」请求。
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// waitCtx 给任务等待加超时保险，避免测试挂死。
func waitCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// fakeComfy 搭一个最小可用的 ComfyUI 假服务，覆盖出图链路真正会碰的 5 个端点：
// /system_stats、/upload/image、/prompt、/history/{id}、/view。
// 这样可以不依赖真机就验证「提交 → 轮询 → 下载 → 落盘」这条主链路。
type fakeComfy struct {
	t           *testing.T
	server      *httptest.Server
	promptBody  map[string]interface{} // 收到的 /prompt 请求体，测试里断言工作流
	uploaded    bool                   // 是否收到参考图上传
	queueStatus int                    // /prompt 返回码，用于构造提交失败
	queueError  string                 // /prompt 失败时的响应体
	failExec    bool                   // /history 返回执行失败
}

func newFakeComfy(t *testing.T) *fakeComfy {
	f := &fakeComfy{t: t}
	mux := http.NewServeMux()

	mux.HandleFunc("/system_stats", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"system":{"os":"macos"}}`)
	})

	mux.HandleFunc("/upload/image", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			f.t.Errorf("参考图不是合法 multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("subfolder") != "aichat" {
			f.t.Errorf("subfolder 未透传: %q", r.FormValue("subfolder"))
		}
		if _, _, err := r.FormFile("image"); err != nil {
			f.t.Errorf("缺少 image 字段: %v", err)
		}
		f.uploaded = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"ref.png","subfolder":"aichat","type":"input"}`)
	})

	mux.HandleFunc("/prompt", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Prompt   map[string]interface{} `json:"prompt"`
			ClientID string                 `json:"client_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.promptBody = body.Prompt
		if f.queueStatus >= 400 {
			w.WriteHeader(f.queueStatus)
			_, _ = io.WriteString(w, f.queueError)
			return
		}
		_, _ = io.WriteString(w, `{"prompt_id":"p1","number":1}`)
	})

	mux.HandleFunc("/history/p1", func(w http.ResponseWriter, _ *http.Request) {
		if f.failExec {
			_, _ = io.WriteString(w, `{"p1":{"outputs":{},"status":{"status_str":"error","completed":false,
				"messages":[["execution_error",{"node_id":"8","node_type":"KSampler","exception_message":"out of memory"}]]}}}`)
			return
		}
		_, _ = io.WriteString(w, `{"p1":{"outputs":{"10":{"images":[
			{"filename":"aichat_00001_.png","subfolder":"","type":"output"}]}},
			"status":{"status_str":"success","completed":true,"messages":[]}}}`)
	})

	mux.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filename") != "aichat_00001_.png" {
			f.t.Errorf("filename 参数异常: %q", r.URL.Query().Get("filename"))
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngHeader(1024, 1024))
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeComfy) cfg() config.ComfyUIConfig {
	return config.ComfyUIConfig{
		BaseURL:    f.server.URL,
		Model:      "m.gguf",
		Clip:       "c.safetensors",
		Vae:        "v.safetensors",
		Steps:      4,
		CFG:        1.0,
		Resolution: 1024,
		TimeoutSec: 30,
		UploadDir:  "aichat",
	}.Normalized()
}

func TestFakeComfyPing(t *testing.T) {
	f := newFakeComfy(t)
	if err := newComfyClient(f.cfg()).ping(t.Context()); err != nil {
		t.Fatalf("探活失败: %v", err)
	}
}

// TestComfyTask_TextToImage 跑完整链路：提交 → 轮询 → 下载 → 落盘。
func TestComfyTask_TextToImage(t *testing.T) {
	f := newFakeComfy(t)
	t.Chdir(t.TempDir()) // saveComfyImage 写相对路径 ./data/uploads，落到临时目录避免污染仓库

	h := &AIGatewayHandler{}
	cfg := f.cfg()
	opts, err := comfyResolveParams(cfg, comfyImageRequest{Prompt: "a red apple"})
	if err != nil {
		t.Fatalf("参数收敛失败: %v", err)
	}
	task := h.startComfyTask(cfg, comfyImageRequest{Prompt: "a red apple"}, "a red apple", "", opts, "t2i")

	final, err := waitComfyTask(waitCtx(t), task.ID)
	if err != nil {
		t.Fatalf("等待任务失败: %v", err)
	}
	if final.Status != "succeeded" {
		t.Fatalf("任务未成功: %+v", final)
	}
	if len(final.Images) != 1 {
		t.Fatalf("应产出 1 张图，得到 %d", len(final.Images))
	}
	img := final.Images[0]
	if !strings.HasPrefix(img.URL, comfyImageURLBase+"/") {
		t.Fatalf("图片 URL 前缀异常: %s", img.URL)
	}
	if img.Width != 1024 || img.Height != 1024 {
		t.Fatalf("图片尺寸解析异常: %dx%d", img.Width, img.Height)
	}
	// 文件必须真的落盘
	if _, err := os.Stat(filepath.Join(uploadDir, img.RelPath)); err != nil {
		t.Fatalf("图片未落盘: %v", err)
	}
	// 工作流里不该有参考图节点
	if _, ok := f.promptBody["4"]; ok {
		t.Fatal("文生图工作流不应包含 LoadImage")
	}
}

// TestComfyTask_EditMode 验证编辑模式：参考图上传 + LoadImage + 扁平键名连线。
func TestComfyTask_EditMode(t *testing.T) {
	f := newFakeComfy(t)
	t.Chdir(t.TempDir())

	h := &AIGatewayHandler{}
	cfg := f.cfg()
	req := comfyImageRequest{Prompt: "make it sunset", Image: "data:image/png;base64," + tinyPNGBase64}
	opts, err := comfyResolveParams(cfg, req)
	if err != nil {
		t.Fatalf("参数收敛失败: %v", err)
	}
	task := h.startComfyTask(cfg, req, "make it sunset", "", opts, "edit")
	final, err := waitComfyTask(waitCtx(t), task.ID)
	if err != nil {
		t.Fatalf("等待任务失败: %v", err)
	}
	if final.Status != "succeeded" {
		t.Fatalf("任务未成功: %+v", final)
	}
	if !f.uploaded {
		t.Fatal("编辑模式必须先上传参考图")
	}
	if _, ok := f.promptBody["4"]; !ok {
		t.Fatal("编辑模式工作流缺少 LoadImage 节点")
	}
	enc, _ := f.promptBody["5"].(map[string]interface{})
	inputs, _ := enc["inputs"].(map[string]interface{})
	if _, ok := inputs["images.image_1"]; !ok {
		t.Fatal("编辑模式必须使用扁平键名 images.image_1")
	}
}

// TestComfyTask_QueueRejected 提交被 ComfyUI 拒绝时，任务要落到 failed 并带上原因。
func TestComfyTask_QueueRejected(t *testing.T) {
	f := newFakeComfy(t)
	f.queueStatus = http.StatusBadRequest
	f.queueError = `{"error":{"type":"prompt_outputs_failed_validation","message":"Prompt outputs failed validation"},
		"node_errors":{"1":{"errors":[{"message":"Value not in list: unet_name"}]}}}`
	t.Chdir(t.TempDir())

	h := &AIGatewayHandler{}
	cfg := f.cfg()
	opts, _ := comfyResolveParams(cfg, comfyImageRequest{Prompt: "x"})
	task := h.startComfyTask(cfg, comfyImageRequest{Prompt: "x"}, "x", "", opts, "t2i")
	final, err := waitComfyTask(waitCtx(t), task.ID)
	if err != nil {
		t.Fatalf("等待任务失败: %v", err)
	}
	if final.Status != "failed" {
		t.Fatalf("应失败，实际 %s", final.Status)
	}
	if !strings.Contains(final.Error, "unet_name") {
		t.Fatalf("错误信息未透出节点错误: %s", final.Error)
	}
}

// TestComfyTask_ExecutionError 执行期异常（如显存不足）同样要落到 failed。
func TestComfyTask_ExecutionError(t *testing.T) {
	f := newFakeComfy(t)
	f.failExec = true
	t.Chdir(t.TempDir())

	h := &AIGatewayHandler{}
	cfg := f.cfg()
	opts, _ := comfyResolveParams(cfg, comfyImageRequest{Prompt: "x"})
	task := h.startComfyTask(cfg, comfyImageRequest{Prompt: "x"}, "x", "", opts, "t2i")
	final, err := waitComfyTask(waitCtx(t), task.ID)
	if err != nil {
		t.Fatalf("等待任务失败: %v", err)
	}
	if final.Status != "failed" || !strings.Contains(final.Error, "out of memory") {
		t.Fatalf("执行错误未正确落库: %+v", final)
	}
}

// TestOpenAIImages_ModelMismatch 模型名对不上要在解析完参数后立刻拒，不能落到出图环节。
func TestOpenAIImages_ModelMismatch(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AIGateway.SuperAdminPassword = "pw"
	cfg.ChatPlus.ComfyUI.BaseURL = "http://127.0.0.1:1" // 不该被访问
	h := &AIGatewayHandler{cfg: cfg}

	router := gin.New()
	router.POST("/v1/images/generations", h.OpenAIImagesGenerations)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations",
		strings.NewReader(`{"model":"dall-e-3","prompt":"a cat"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Super-Admin-Password", "pw")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), comfyImageModel) {
		t.Fatalf("错误信息应指出可用模型: %s", w.Body.String())
	}
}

// TestChatPlusImageStatus_Disabled 未配置 base_url 时必须 fail-closed（available=false + 200）。
func TestChatPlusImageStatus_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ChatPlus.Password = "pw"
	cfg.ChatPlus.ComfyUI.BaseURL = ""
	h := &AIGatewayHandler{cfg: cfg}

	router := gin.New()
	router.GET("/status", h.ChatPlusImageStatus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("X-Chat-Password", "pw")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var body struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
		Model     string `json:"model"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %s", w.Body.String())
	}
	if body.Available {
		t.Fatal("未配置 base_url 时不应报告可用")
	}
	if body.Error == "" {
		t.Fatal("应给出禁用原因")
	}
	if body.Model == "" {
		t.Fatal("默认模型名应回填，方便前端展示")
	}
}

// TestChatPlusImageStatus_PasswordGate 密码错误必须 401，且不泄漏探活信息。
func TestChatPlusImageStatus_PasswordGate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ChatPlus.Password = "pw"
	h := &AIGatewayHandler{cfg: cfg}

	router := gin.New()
	router.GET("/status", h.ChatPlusImageStatus)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("X-Chat-Password", "wrong")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}
