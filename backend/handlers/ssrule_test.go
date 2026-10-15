package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
