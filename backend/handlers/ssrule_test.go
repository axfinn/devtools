package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const ssRuleTestPassword = "test-admin-pwd"

// newSSRuleTestRouter 复用同包的 newMemDB(t)(已遵守 SetMaxOpenConns(1) 的
// :memory: 连接池约定),只挂本模块的端点。
func newSSRuleTestRouter(t *testing.T, adminPassword string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := newMemDB(t)
	h := NewSSRuleHandler(db, adminPassword)

	r := gin.New()
	r.POST("/api/ssrules/verify", h.Verify)
	r.GET("/api/ssrules", h.Get)
	r.POST("/api/ssrules", h.Save)
	r.POST("/api/ssrules/regenerate-token", h.RegenerateToken)
	r.GET("/ss/:token/:filename", h.Download)
	return r
}

func ssRuleDo(r *gin.Engine, method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func ssRuleAuthHeader(pwd string) map[string]string {
	return map[string]string{"X-Super-Admin-Password": pwd}
}

type ssRuleViewResp struct {
	Source  string   `json:"source"`
	Content string   `json:"content"`
	Token   string   `json:"token"`
	Paths   []string `json:"paths"`
	Skipped []string `json:"skipped"`
}

func ssRuleDecode(t *testing.T, w *httptest.ResponseRecorder) ssRuleViewResp {
	t.Helper()
	var out ssRuleViewResp
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("响应不是合法 JSON: %v\n%s", err, w.Body.String())
	}
	return out
}

func TestSSRule_VerifyPassword(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	t.Run("无密码 401", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodPost, "/api/ssrules/verify", nil, []byte("{}"))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("想要 401,得到 %d", w.Code)
		}
	})

	t.Run("错误密码 401", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodPost, "/api/ssrules/verify", ssRuleAuthHeader("wrong"), []byte("{}"))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("想要 401,得到 %d", w.Code)
		}
	})

	t.Run("正确密码 200", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodPost, "/api/ssrules/verify", ssRuleAuthHeader(ssRuleTestPassword), []byte("{}"))
		if w.Code != http.StatusOK {
			t.Fatalf("想要 200,得到 %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestSSRule_DisabledWhenPasswordNotConfigured(t *testing.T) {
	// 未配密码 = 功能关闭,而不是「无密码放行」。
	// 这条边界要是写反了,等于把规则编辑接口完全敞开。
	r := newSSRuleTestRouter(t, "")

	cases := []struct {
		method, path string
		body         []byte
	}{
		{http.MethodPost, "/api/ssrules/verify", []byte("{}")},
		{http.MethodGet, "/api/ssrules", nil},
		{http.MethodPost, "/api/ssrules", []byte(`{"source":"a.com"}`)},
		{http.MethodPost, "/api/ssrules/regenerate-token", nil},
	}

	for _, c := range cases {
		// 即便带上一个空密码头也不该放行
		w := ssRuleDo(r, c.method, c.path, ssRuleAuthHeader(""), c.body)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s 在未配置密码时应 401,得到 %d", c.method, c.path, w.Code)
		}
	}
}

func TestSSRule_TokenCannotBeUsedToEdit(t *testing.T) {
	// token 只授予「读规则」,不授予「编辑」。这条边界丢了,
	// 拿到公开链接的任何人都能改规则。
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	saved := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"a.com"}`)))

	t.Run("GET 管理接口不接受 token", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodGet, "/api/ssrules?token="+saved.Token, nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("token 不该被当作管理凭据,得到 %d", w.Code)
		}
	})

	t.Run("POST 管理接口不接受 token", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodPost, "/api/ssrules?admin_password="+saved.Token,
			nil, []byte(`{"source":"evil.com"}`))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("token 不该被当作管理凭据,得到 %d", w.Code)
		}
	})
}

func TestSSRule_SaveThenDownload(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	w := ssRuleDo(r, http.MethodPost, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword),
		[]byte(`{"source":"google.com\nyoutube.com"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("保存失败 %d: %s", w.Code, w.Body.String())
	}
	saved := ssRuleDecode(t, w)

	if saved.Token == "" {
		t.Fatal("保存后应返回 token")
	}
	if len(saved.Paths) != len(ssRuleFileNames) {
		t.Fatalf("应返回 %d 个链接,得到 %v", len(ssRuleFileNames), saved.Paths)
	}

	t.Run("按 token 可公开下载", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodGet, "/ss/"+saved.Token+"/rules.acl", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("想要 200,得到 %d", w.Code)
		}
		body := w.Body.String()
		if !bytes.Contains([]byte(body), []byte(`(^|\.)google\.com$`)) {
			t.Errorf("下载内容缺少规则:\n%s", body)
		}
		if !bytes.Contains([]byte(body), []byte(`(^|\.)youtube\.com$`)) {
			t.Errorf("下载内容缺少规则:\n%s", body)
		}
	})

	t.Run("同一个内容多个后缀都能取到", func(t *testing.T) {
		for _, name := range ssRuleFileNames {
			w := ssRuleDo(r, http.MethodGet, "/ss/"+saved.Token+"/"+name, nil, nil)
			if w.Code != http.StatusOK {
				t.Errorf("%s 想要 200,得到 %d", name, w.Code)
			}
		}
	})

	t.Run("未知 token 404", func(t *testing.T) {
		w := ssRuleDo(r, http.MethodGet, "/ss/deadbeefdeadbeef/rules.acl", nil, nil)
		if w.Code != http.StatusNotFound {
			t.Fatalf("想要 404,得到 %d", w.Code)
		}
	})

	t.Run("白名单外的文件名 404", func(t *testing.T) {
		// .pac 是有意排除的:PAC 是 JavaScript,把 ACL 正文当 .pac 发出去
		// 客户端会静默拿到语法错误的脚本。
		for _, name := range []string{"rules.pac", "index.html", "rules.yaml", "rules"} {
			w := ssRuleDo(r, http.MethodGet, "/ss/"+saved.Token+"/"+name, nil, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("%s 想要 404,得到 %d", name, w.Code)
			}
		}
	})
}

func TestSSRule_RegenerateTokenInvalidatesOldLink(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	saved := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"a.com"}`)))
	oldToken := saved.Token

	// 换发前旧链接可用
	if w := ssRuleDo(r, http.MethodGet, "/ss/"+oldToken+"/rules.acl", nil, nil); w.Code != http.StatusOK {
		t.Fatalf("换发前旧链接应可用,得到 %d", w.Code)
	}

	w := ssRuleDo(r, http.MethodPost, "/api/ssrules/regenerate-token", ssRuleAuthHeader(ssRuleTestPassword), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("换发失败 %d: %s", w.Code, w.Body.String())
	}
	regenerated := ssRuleDecode(t, w)

	if regenerated.Token == oldToken {
		t.Fatal("换发后 token 应当变化")
	}
	if w := ssRuleDo(r, http.MethodGet, "/ss/"+oldToken+"/rules.acl", nil, nil); w.Code != http.StatusNotFound {
		t.Errorf("换发后旧链接应 404,得到 %d", w.Code)
	}
	if w := ssRuleDo(r, http.MethodGet, "/ss/"+regenerated.Token+"/rules.acl", nil, nil); w.Code != http.StatusOK {
		t.Errorf("换发后新链接应可用,得到 %d", w.Code)
	}
}

func TestSSRule_GetReturnsSourceForEditing(t *testing.T) {
	// 页面需要拿回用户原始输入才能继续编辑,而不是反向解析生成的 ACL。
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	source := "google.com\nhttps://github.com/foo/bar"
	ssRuleDo(r, http.MethodPost, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword),
		[]byte(`{"source":`+mustJSON(t, source)+`}`))

	w := ssRuleDo(r, http.MethodGet, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("想要 200,得到 %d", w.Code)
	}
	got := ssRuleDecode(t, w)

	if got.Source != source {
		t.Errorf("原始输入未原样返回\n想要: %q\n得到: %q", source, got.Source)
	}
	if !bytes.Contains([]byte(got.Content), []byte(`(^|\.)github\.com$`)) {
		t.Errorf("URL 未抽成域名规则:\n%s", got.Content)
	}
}

func mustJSON(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal 失败: %v", err)
	}
	return string(b)
}

// ssRuleTraversalPayloads 是路径穿越回归载荷集。
//
// 来源:方案 §3.4.1(c) 的 10 行载荷表展开后的 14 个具体值。
// 这里注入的是**文件名段**本身(不经 gin 路由),目的是证明白名单函数对
// 任何含 / 、%2f、null byte、.. 的输入都免疫 —— 即便路由层将来被改坏。
var ssRuleTraversalPayloads = []string{
	"../../etc/passwd",         // 相对路径穿越
	"..%2f..%2fetc%2fpasswd",   // URL 编码的斜杠
	"%2e%2e%2f%2e%2e%2fpasswd", // 全编码的 ..
	"....//....//etc/passwd",   // 连续点 + 双斜杠
	"rules.acl\x00.txt",        // null byte
	"rules.acl\x00",            // null byte 结尾
	"rules.acl.txt",            // 双扩展名
	"rules.acl/",               // 尾斜杠
	`rules.acl\..\x`,           // 反斜杠
	"rules.pac",                // 白名单外的后缀(PAC 是 JS,故意排除)
	"index.html",               // 白名单外
	"rules.yaml",               // 白名单外
	"rules",                    // 白名单外
	"",                         // 空 filename
}

// TestSSRule_DownloadRejectsTraversalPayloads 白名单回归(handler 级)。
//
// 断言必须**逐字节**比 body:白名单拒绝 / token 不存在 / 模块关闭三者的响应
// 长得一模一样,外部观察者分辨不出「文件名不对」与「链接不存在」。
//
// 反事实:把 ssRuleIsAllowedFile 改成 strings.HasSuffix(name, ".acl"),
// "rules.acl.txt" 这条必须红。
func TestSSRule_DownloadRejectsTraversalPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMemDB(t)
	h := NewSSRuleHandler(db, ssRuleTestPassword)

	for _, payload := range ssRuleTraversalPayloads {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{
			{Key: "token", Value: "TOK"},
			{Key: "filename", Value: payload},
		}
		h.Download(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("载荷 %q 想要 404,得到 %d", payload, w.Code)
		}
		if got := w.Body.String(); got != SSRuleNotFoundBody {
			t.Errorf("载荷 %q 的响应体想要 %q,得到 %q", payload, SSRuleNotFoundBody, got)
		}
		if strings.Contains(w.Body.String(), "[proxy_list]") {
			t.Errorf("载荷 %q 泄露了规则正文:\n%s", payload, w.Body.String())
		}
	}
}

// TestSSRule_ModuleDisabledBlocksDistribution 「未配密码 = 模块关闭」
// 在分发端必须同样成立(P0-3)。
//
// 反事实:删掉 Download 开头的 adminPassword 判断 → 本用例必须红。
func TestSSRule_ModuleDisabledBlocksDistribution(t *testing.T) {
	db := newMemDB(t)

	// 先用「开着模块」的 handler 存一份规则,拿到真实 token。
	enabled := NewSSRuleHandler(db, ssRuleTestPassword)
	saved, err := enabled.db.SaveSSRuleFile("google.com", "(^|\\.)google\\.com$")
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}

	// 模块关闭(handler 拿到的密码为空)。
	disabled := NewSSRuleHandler(db, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "token", Value: saved.Token},
		{Key: "filename", Value: "rules.acl"},
	}
	disabled.Download(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("模块关闭时想要 404,得到 %d", w.Code)
	}
	if got := w.Body.String(); got != SSRuleNotFoundBody {
		t.Fatalf("模块关闭时的响应体想要 %q,得到 %q", SSRuleNotFoundBody, got)
	}

	// 对照:同一个 token 在模块开着时是能取到的 —— 证明上面的 404 是「模块关闭」
	// 造成的,不是「文件本来就不在」。
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = gin.Params{
		{Key: "token", Value: saved.Token},
		{Key: "filename", Value: "rules.acl"},
	}
	enabled.Download(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("模块开启时想要 200,得到 %d", w2.Code)
	}
}

// TestSSRule_AdminPasswordOnlyViaHeader 管理端鉴权只认 header。
//
// query 兜底会把密码塞进 URL —— Cloudflare 访问日志、浏览器历史、Referer 头
// 全都会留下明文,而这些都不是能随时擦掉的存储。
//
// 反事实:恢复 c.Query("admin_password") 兜底 → 本用例必须红。
func TestSSRule_AdminPasswordOnlyViaHeader(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	for _, q := range []string{
		"?admin_password=" + ssRuleTestPassword,
		"?super_admin_password=" + ssRuleTestPassword,
		"?password=" + ssRuleTestPassword,
	} {
		w := ssRuleDo(r, http.MethodGet, "/api/ssrules"+q, nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s 不该被当作管理凭据,得到 %d", q, w.Code)
		}
	}

	// header 仍然要能用。
	w := ssRuleDo(r, http.MethodGet, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword), nil)
	if w.Code != http.StatusOK {
		t.Errorf("header 鉴权想要 200,得到 %d", w.Code)
	}
}

// TestSSRule_RegenerateTokenRecomputesSkipped 换发 token 后 skipped 必须
// 是实际重算值,而不是恒为空数组。
//
// 反事实:把 RegenerateToken 改回 newSSRuleView(file, nil) → 本用例必须红。
func TestSSRule_RegenerateTokenRecomputesSkipped(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	// 两行非法输入:一行不是合法域名,一行解析后为空。
	w := ssRuleDo(r, http.MethodPost, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword),
		[]byte(`{"source":"google.com\nnot a domain\n???"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("保存失败 %d: %s", w.Code, w.Body.String())
	}

	regen := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules/regenerate-token",
		ssRuleAuthHeader(ssRuleTestPassword), nil))
	if len(regen.Skipped) == 0 {
		t.Fatal("换发后 skipped 不该为空")
	}

	got := ssRuleDecode(t, ssRuleDo(r, http.MethodGet, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword), nil))
	if len(got.Skipped) != len(regen.Skipped) {
		t.Fatalf("换发后 skipped 长度 %d 与 GET 的 %d 不一致", len(regen.Skipped), len(got.Skipped))
	}
	for i := range regen.Skipped {
		if regen.Skipped[i] != got.Skipped[i] {
			t.Errorf("第 %d 条 skipped 不一致:换发 %q,GET %q", i, regen.Skipped[i], got.Skipped[i])
		}
	}
}

// TestSSRule_DownloadSecurityHeaders 分发端的响应头。
//
// Content-Disposition 只拼白名单常量:filename 是外部可控的,拼进响应头
// 就是注入面。
//
// 反事实:删掉 nosniff 头 / 改成拼 c.Param("filename") → 本用例必须红。
func TestSSRule_DownloadSecurityHeaders(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	saved := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"google.com"}`)))

	w := ssRuleDo(r, http.MethodGet, "/ss/"+saved.Token+"/rules.acl", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("想要 200,得到 %d", w.Code)
	}

	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options 想要 nosniff,得到 %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="rules.acl"` {
		t.Errorf("Content-Disposition 想要白名单常量,得到 %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control 想要 no-store,得到 %q", got)
	}
	if got := w.Header().Get("Content-Type"); got != ssRuleFileContentType {
		t.Errorf("Content-Type 想要 %q,得到 %q", ssRuleFileContentType, got)
	}
}
