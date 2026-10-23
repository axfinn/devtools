package handlers

// =====================================================================
//  FINN-94 Stage 5 · ssrules 后端单测补测
//
//  范围:只补 Stage 3 交付的 23 条用例(backend/handlers/ssrule_test.go、
//  ssrule_rules_test.go、backend/routes/ssrule_test.go、
//  backend/app_http_ssrule_test.go)**没有覆盖**的断言。
//  方案 v3 §6.3 的 A1–A20 里,本文件补的是:
//
//    A1   管理端响应体必须带 paths(白名单 4 条)+ source 原样
//    A4   换发后旧 token 的 404 正文必须与「乱 token」的 404 正文逐字节一致
//    A5/A19 模块关闭时,分发端对**任意** token 都是同款 404
//    A9   updated_at 必须能被 RFC3339 解析、且时区偏移为 0(UTC)
//    A10  ss_rule_files.token 上有唯一索引(且重复 token 真的插不进去)
//    M3   source 超长时 400 的报文
//    P0-5 绝对路径载荷 + 「全程不碰文件系统」的行为证据
//    Q4   token 没有有效期(表里没有过期列、同一 token 可反复拉取)
//    Q6   只有一套规则(库里单行、管理端只有 4 个端点)
//
//  已覆盖的部分**不重复造** —— 重复的断言既是恒真闸门的温床,
//  也让「哪条要求由哪个测试覆盖」这张对照表失去意义。
// =====================================================================

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestSSRule_AdminViewCarriesPathsAndSource A1:管理端响应体必须带
// paths(白名单 4 条,顺序一致)与 source(原样回显)。
//
// 反事实:把 newSSRuleView 的 paths 改成只回一条 → 本用例必须红;
//
//	把 Get 里的 file.Source 换成 file.Content → source 子项必须红。
func TestSSRule_AdminViewCarriesPathsAndSource(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	source := "google.com\nyoutube.com"
	ssRuleDo(r, http.MethodPost, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword),
		[]byte(`{"source":`+mustJSON(t, source)+`}`))

	w := ssRuleDo(r, http.MethodGet, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("想要 200,得到 %d: %s", w.Code, w.Body.String())
	}
	got := ssRuleDecode(t, w)

	if got.Source != source {
		t.Errorf("source 应原样回显\n想要: %q\n得到: %q", source, got.Source)
	}

	want := []string{
		"/ss/" + got.Token + "/rules.acl",
		"/ss/" + got.Token + "/rules.txt",
		"/ss/" + got.Token + "/rules.list",
		"/ss/" + got.Token + "/rules.conf",
	}
	if len(got.Paths) != len(want) {
		t.Fatalf("paths 应当恰好 %d 条,得到 %d 条: %v", len(want), len(got.Paths), got.Paths)
	}
	for i := range want {
		if got.Paths[i] != want[i] {
			t.Errorf("paths 第 %d 条不符,想要 %q,得到 %q", i, want[i], got.Paths[i])
		}
	}
}

// TestSSRule_RegenerateOldTokenBodyIdenticalToUnknownToken A4 补:
// 换发后旧 token 的 404 正文,必须与「token 根本不存在」的 404 正文
// **逐字节一致** —— 外部观察者分辨不出「链接被换掉了」与「链接从来没存在过」。
//
// 反事实:把 Download 里 token 不存在的分支改成另一句文案 → 本用例必须红。
func TestSSRule_RegenerateOldTokenBodyIdenticalToUnknownToken(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	saved := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"a.com"}`)))
	oldToken := saved.Token

	if w := ssRuleDo(r, http.MethodPost, "/api/ssrules/regenerate-token",
		ssRuleAuthHeader(ssRuleTestPassword), nil); w.Code != http.StatusOK {
		t.Fatalf("换发失败 %d: %s", w.Code, w.Body.String())
	}

	oldW := ssRuleDo(r, http.MethodGet, "/ss/"+oldToken+"/rules.acl", nil, nil)
	randomW := ssRuleDo(r, http.MethodGet, "/ss/deadbeefdeadbeefdeadbeefdeadbeef/rules.acl", nil, nil)

	if oldW.Code != http.StatusNotFound {
		t.Fatalf("换发后旧链接想要 404,得到 %d", oldW.Code)
	}
	if randomW.Code != http.StatusNotFound {
		t.Fatalf("乱 token 想要 404,得到 %d", randomW.Code)
	}
	if oldW.Body.String() != randomW.Body.String() {
		t.Errorf("旧 token 与乱 token 的 404 正文必须逐字节一致\n旧 token: %q\n乱 token: %q",
			oldW.Body.String(), randomW.Body.String())
	}
	if oldW.Body.String() != SSRuleNotFoundBody {
		t.Errorf("404 正文想要 %q,得到 %q", SSRuleNotFoundBody, oldW.Body.String())
	}
}

// TestSSRule_ModuleDisabledAnyTokenSameBody A5/A19 补:
// 「未配密码 = 模块关闭」在分发端必须对**任意** token 成立,
// 且与「文件名不在白名单」的那句 404 逐字节一致 —— 三种「不存在」
// 在外部观察者眼里必须完全一样。
//
// 关键:必须用**真实 token**。token 不存在时,「白名单放行后查库落空」
// 与「白名单拒绝」回的是同一句 404,断言无从区分(恒真闸门)。
// 反事实:删掉 Download 开头的 adminPassword 判断 → 本用例必须红。
func TestSSRule_ModuleDisabledAnyTokenSameBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMemDB(t)

	// 先用「开着模块」的 handler 存一份规则,拿到真实 token。
	enabled := NewSSRuleHandler(db, ssRuleTestPassword)
	file, err := db.SaveSSRuleFile("google.com", "(^|\\.)google\\.com$")
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}

	// 模块关闭:同一个 db、同一个真实 token,但 handler 拿到的密码为空。
	disabled := NewSSRuleHandler(db, "")
	r := gin.New()
	r.GET("/ss/:token/:filename", disabled.Download)

	// 真实 token —— 模块关闭时必须 404。
	w := ssRuleDo(r, http.MethodGet, "/ss/"+file.Token+"/rules.acl", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("模块关闭时真实 token 想要 404,得到 %d", w.Code)
	}
	if w.Body.String() != SSRuleNotFoundBody {
		t.Errorf("模块关闭时真实 token 的正文想要 %q,得到 %q", SSRuleNotFoundBody, w.Body.String())
	}

	// 乱 token 也一样。
	for _, token := range []string{
		"00000000000000000000000000000000",
		"ffffffffffffffffffffffffffffffff",
		"not-a-token-at-all",
	} {
		w := ssRuleDo(r, http.MethodGet, "/ss/"+token+"/rules.acl", nil, nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("模块关闭时 token=%q 想要 404,得到 %d", token, w.Code)
		}
		if w.Body.String() != SSRuleNotFoundBody {
			t.Errorf("模块关闭时 token=%q 的正文想要 %q,得到 %q", token, SSRuleNotFoundBody, w.Body.String())
		}
	}

	// 白名单拒绝的那一句也必须一模一样。
	w = ssRuleDo(r, http.MethodGet, "/ss/"+file.Token+"/rules.pac", nil, nil)
	if w.Body.String() != SSRuleNotFoundBody {
		t.Errorf("白名单拒绝的正文想要 %q,得到 %q", SSRuleNotFoundBody, w.Body.String())
	}

	// 正向对照:同一个 token 在模块**开着**时是能取到的 ——
	// 没有这条,上面的 404 可能只是「文件本来就不在」。
	open := gin.New()
	open.GET("/ss/:token/:filename", enabled.Download)
	w = ssRuleDo(open, http.MethodGet, "/ss/"+file.Token+"/rules.acl", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("模块开启时想要 200,得到 %d", w.Code)
	}
}

// TestSSRule_UpdatedAtIsUTCRFC3339 A9:updated_at 必须能被浏览器解析,
// 且时区偏移为 0(UTC)—— 存本地偏移的话前端 toLocaleString() 会算错。
//
// 反事实:把 models 里的 time.Now().UTC() 改成 time.Now() → 本机在
// 非 UTC 时区下偏移子项必须红;把写入改成零值 → 非零子项必须红。
func TestSSRule_UpdatedAtIsUTCRFC3339(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	ssRuleDo(r, http.MethodPost, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword),
		[]byte(`{"source":"a.com"}`))

	w := ssRuleDo(r, http.MethodGet, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("想要 200,得到 %d: %s", w.Code, w.Body.String())
	}

	var raw struct {
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("响应不是合法 JSON: %v\n%s", err, w.Body.String())
	}
	if raw.UpdatedAt == "" {
		t.Fatal("updated_at 不该为空")
	}

	parsed, err := time.Parse(time.RFC3339, raw.UpdatedAt)
	if err != nil {
		t.Fatalf("updated_at 应当能被 RFC3339 解析: %v(原值 %q)", err, raw.UpdatedAt)
	}
	if parsed.IsZero() {
		t.Errorf("updated_at 不该是零值(页面会显示 1/1/1): %q", raw.UpdatedAt)
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		t.Errorf("updated_at 的时区偏移应当是 0(UTC),得到 %d 秒: %q", offset, raw.UpdatedAt)
	}
	if strings.HasSuffix(raw.UpdatedAt, "+08:00") {
		t.Errorf("updated_at 不该带本地偏移 +08:00: %q", raw.UpdatedAt)
	}
}

// TestSSRule_TokenColumnHasUniqueIndex A10:token 上必须有唯一索引,
// 且重复 token 真的插不进去(只查 DDL 文本不够 —— 行为也要咬合)。
//
// 反事实:删掉 CREATE UNIQUE INDEX → 插入子项必须红。
func TestSSRule_TokenColumnHasUniqueIndex(t *testing.T) {
	db := newMemDB(t)

	var ddl string
	if err := db.Conn().QueryRow(
		`SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_ss_rule_files_token'`,
	).Scan(&ddl); err != nil {
		t.Fatalf("查不到 idx_ss_rule_files_token 索引: %v", err)
	}
	if !strings.Contains(strings.ToUpper(ddl), "UNIQUE") {
		t.Errorf("idx_ss_rule_files_token 应当是唯一索引,实际 DDL: %s", ddl)
	}

	file, err := db.GetSSRuleFile()
	if err != nil {
		t.Fatalf("取规则文件失败: %v", err)
	}
	_, err = db.Conn().Exec(
		`INSERT INTO ss_rule_files (id, token, source, content, updated_at) VALUES ('dup', ?, '', '', ?)`,
		file.Token, time.Now().UTC())
	if err == nil {
		t.Fatal("重复 token 应当被唯一索引拒绝,却插入成功了")
	}
}

// TestSSRule_TokenHasNoExpiry Q4:token 一旦生成长期有效,直到手工换发。
//
// 断言两件事:表里**没有**过期列(有列就意味着存在过期逻辑);
// 同一个 token 可以反复拉取(不是一次性凭据)。
func TestSSRule_TokenHasNoExpiry(t *testing.T) {
	db := newMemDB(t)

	rows, err := db.Conn().Query(`PRAGMA table_info(ss_rule_files)`)
	if err != nil {
		t.Fatalf("读表结构失败: %v", err)
	}
	var cols []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dflt any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			rows.Close()
			t.Fatalf("扫描列信息失败: %v", err)
		}
		cols = append(cols, name)
	}
	rows.Close() // 必须先关: :memory: 单连接,游标不关就拿不到下一条连接

	for _, c := range cols {
		if strings.Contains(strings.ToLower(c), "expir") || strings.Contains(strings.ToLower(c), "ttl") {
			t.Errorf("ss_rule_files 不该有有效期列,却发现 %q(列清单: %v)", c, cols)
		}
	}

	r := newSSRuleTestRouter(t, ssRuleTestPassword)
	saved := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"a.com"}`)))
	for i := 1; i <= 3; i++ {
		w := ssRuleDo(r, http.MethodGet, "/ss/"+saved.Token+"/rules.acl", nil, nil)
		if w.Code != http.StatusOK {
			t.Errorf("第 %d 次拉取想要 200,得到 %d(token 不是一次性凭据)", i, w.Code)
		}
	}
}

// TestSSRule_SaveOversizeSourceReturns400 M3 契约:
// source 超过 512 KB 时是 400 + 「输入过长」提示(不是 413,也不是 500)。
//
// 反事实:把 buildACLRules 的长度判断删掉 → 本用例必须红。
func TestSSRule_SaveOversizeSourceReturns400(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	oversize := strings.Repeat("a.com\n", ssRuleMaxSourceSize/len("a.com\n")+1)
	w := ssRuleDo(r, http.MethodPost, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword),
		[]byte(`{"source":`+mustJSON(t, oversize)+`}`))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("超长 source 想要 400,得到 %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "输入过长") {
		t.Errorf("超长 source 的提示应当包含「输入过长」: %s", w.Body.String())
	}
}

// TestSSRule_SkippedIsAlwaysArrayNotNull §3.2.3:skipped 恒为数组,
// 空时是 [] 而不是 null —— 前端少写一个 ?? 就白屏。
//
// 反事实:把 newSSRuleView 的 nil 归一删掉 → 本用例必须红。
func TestSSRule_SkippedIsAlwaysArrayNotNull(t *testing.T) {
	r := newSSRuleTestRouter(t, ssRuleTestPassword)

	w := ssRuleDo(r, http.MethodGet, "/api/ssrules", ssRuleAuthHeader(ssRuleTestPassword), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("想要 200,得到 %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"skipped":[]`) {
		t.Errorf("skipped 为空时应当是 [] 而不是 null: %s", w.Body.String())
	}
}

// TestSSRule_DownloadRejectsTraversalPayloadsWithRealToken A3 的**可证伪**版本。
//
// 为什么必须再写一条:上游那条 handler 级用例(backend/handlers/ssrule_test.go
// 的 TestSSRule_DownloadRejectsTraversalPayloads)注入的 token 是 "TOK" ——
// 库里根本没有这一行。于是「白名单放行 → 查库 → token 不存在 → 404」
// 与「白名单拒绝 → 404」返回的是**同一句**「规则文件不存在」,
// 断言无从区分 ⇒ 恒真闸门。
//
// 实测(FINN-94 反事实自检):把白名单改成 strings.Contains(name, "rules")
// 之后,上游那条用例**依然全绿**。方案 §6.3 给它写的反事实
// (「改成 strings.HasSuffix(name, ".acl") → rules.acl.txt 那条必须红」)
// 实测同样不成立 —— "rules.acl.txt" 并不以 ".acl" 结尾。
//
// 本用例用**真实 token** + **正向对照**把这条断言补成真闸门:
//
//	白名单内的文件名 + 真实 token → 200 且带规则正文
//	白名单外的任意载荷 + 真实 token → 404 且正文逐字节等于「规则文件不存在」
//
// ⇒ 白名单一旦被放松,本用例立刻红。
func TestSSRule_DownloadRejectsTraversalPayloadsWithRealToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMemDB(t)
	h := NewSSRuleHandler(db, ssRuleTestPassword)

	file, err := db.SaveSSRuleFile("google.com", "(^|\\.)google\\.com$")
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}

	// 正向对照:白名单内的文件名 + 真实 token → 200 + 规则正文。
	// 没有这条,「全部 404」本身就成了恒真闸门。
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "token", Value: file.Token},
		{Key: "filename", Value: "rules.acl"},
	}
	h.Download(c)
	if w.Code != http.StatusOK {
		t.Fatalf("正向对照:真实 token + 白名单文件名想要 200,得到 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "(^|\\.)google\\.com$") {
		t.Fatalf("正向对照:响应里没有规则正文:\n%s", w.Body.String())
	}

	for _, payload := range ssRuleTraversalPayloads {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{
			{Key: "token", Value: file.Token},
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

// TestSSRule_DownloadRejectsAbsolutePathPayloads P0-5 补:绝对路径载荷。
//
// 上游那 14 个载荷里有相对穿越、URL 编码、null byte、双扩展名,
// 但**没有**绝对路径(Windows 盘符 / 根目录形式)。
//
// 同时给「全程不碰文件系统」一个行为级证据:把一个真实存在的文件
// 塞进 filename 参数,响应里既不能出现 200,也不能出现文件内容 ——
// 一旦有人改成 os.ReadFile(filepath.Join(base, name)),本用例立刻红。
//
// 用**真实 token**:token 不存在时白名单放行与拒绝回同一句 404,
// 断言无从区分(见上面那条用例的注释)。
func TestSSRule_DownloadRejectsAbsolutePathPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMemDB(t)
	h := NewSSRuleHandler(db, ssRuleTestPassword)

	file, err := db.SaveSSRuleFile("google.com", "(^|\\.)google\\.com$")
	if err != nil {
		t.Fatalf("保存规则失败: %v", err)
	}

	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.acl")
	const secretBody = "TOP-SECRET-RULE-CONTENT"
	if err := os.WriteFile(secretPath, []byte(secretBody), 0o600); err != nil {
		t.Fatalf("写临时文件失败: %v", err)
	}

	payloads := []string{
		"/etc/passwd",
		"/etc/shadow",
		`C:\windows\win.ini`,
		`\\server\share\secret.acl`,
		secretPath,
	}
	for _, payload := range payloads {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{
			{Key: "token", Value: file.Token},
			{Key: "filename", Value: payload},
		}
		h.Download(c)

		if w.Code != http.StatusNotFound {
			t.Errorf("绝对路径载荷 %q 想要 404,得到 %d", payload, w.Code)
		}
		if got := w.Body.String(); got != SSRuleNotFoundBody {
			t.Errorf("绝对路径载荷 %q 的正文想要 %q,得到 %q", payload, SSRuleNotFoundBody, got)
		}
		if strings.Contains(w.Body.String(), secretBody) {
			t.Errorf("绝对路径载荷 %q 读出了文件系统内容:\n%s", payload, w.Body.String())
		}
	}
}

// TestSSRule_OnlyOneRuleRowAndOneToken Q6:全站只有一份规则文件。
//
// 反复保存不新增行、也不换 token(只有 regenerate-token 才换);
// 管理端没有「多套规则」的概念。
func TestSSRule_OnlyOneRuleRowAndOneToken(t *testing.T) {
	db := newMemDB(t)

	// 显式搭路由:这条用例要数**这个 db** 里的行数,
	// 不能用 newSSRuleTestRouter(它自建了一个独立的 db)。
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSSRuleHandler(db, ssRuleTestPassword)
	r.POST("/api/ssrules", h.Save)

	first := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"a.com"}`)))
	second := ssRuleDecode(t, ssRuleDo(r, http.MethodPost, "/api/ssrules",
		ssRuleAuthHeader(ssRuleTestPassword), []byte(`{"source":"b.com"}`)))

	if first.Token != second.Token {
		t.Errorf("普通保存不该换 token:第一次 %q,第二次 %q", first.Token, second.Token)
	}

	var count int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM ss_rule_files`).Scan(&count); err != nil {
		t.Fatalf("数行数失败: %v", err)
	}
	if count != 1 {
		t.Errorf("ss_rule_files 应当只有 1 行(不做多套规则),得到 %d 行", count)
	}
}
