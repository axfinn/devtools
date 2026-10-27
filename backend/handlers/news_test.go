package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// 用例里反复出现的固定值集中成常量,避免同一字面量散落多处后改一处漏一处。
const (
	newsTestDate        = "2026-10-25"      // 用例基准日期
	newsTestDatePrev    = "2026-10-24"      // 基准日期的前一天
	newsTestDatePrev2   = "2026-10-23"      // 基准日期的前两天
	newsTestAdminPwd    = "s3cret"          // 用例管理员密码
	newsUnknownSource   = "未知来源"            // 空来源条目落库时的占位来源
	newsDefaultAIModel  = "WorkBuddy-Agent" // ai_model 缺省时的默认值
	newsTestDateLabel   = "2026年10月25日 周日"  // newsTestDate 对应的中文展示标签
	newsTestSchedule    = "08:30"           // schedule_hour:schedule_minute 的展示值
	newsScheduleMinute  = 30                // 用例配置的分钟
	newsDefaultMaxItems = 30                // max_items 缺省值
	newsDefaultPageSize = 30                // 归档列表默认 limit
	newsMaxPageSize     = 100               // 归档列表 limit 上限
)

// ---------- 测试脚手架 ----------

// newsTestDB 构造只初始化了 news 表的内存库。
// 注意(R5):sqlite 的 ":memory:" 每个连接都是一份独立内存库,必须 SetMaxOpenConns(1),
// 否则后台 goroutine / 另一条连接会拿到另一份库,表现为「刚插入就查不到」。
func newsTestDB(t *testing.T) *models.DB {
	t.Helper()
	db, err := models.NewDB(":memory:")
	if err != nil {
		t.Fatalf("创建内存库失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.InitNews(); err != nil {
		t.Fatalf("初始化 news 表失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// newsTestHandler 构造 NewsHandler。mutate 用于按用例改配置(nil 表示用默认配置)。
func newsTestHandler(t *testing.T, db *models.DB, mutate func(*config.Config)) *NewsHandler {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.News.Enabled = true
	cfg.News.ScheduleHour = 8
	cfg.News.ScheduleMinute = newsScheduleMinute
	if mutate != nil {
		mutate(cfg)
	}
	return NewNewsHandler(db, cfg, nil)
}

// newNewsMockContext 构造 news handler 测试用的 gin.Context。
// gin.SetMode 放在 helper 内部,避免新增包级 init()。
func newNewsMockContext(t *testing.T, method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	u, err := url.Parse(target)
	if err != nil {
		t.Fatalf("非法 target %q: %v", target, err)
	}
	req := &http.Request{
		Method:     method,
		URL:        u,
		Header:     make(http.Header),
		RemoteAddr: "127.0.0.1:1234",
		Body:       http.NoBody,
	}
	if body != "" {
		req.Body = io.NopCloser(strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	return c, rec
}

// newsPublishResponse Publish 接口的响应结构。
type newsPublishResponse struct {
	OK          bool      `json:"ok"`
	Date        string    `json:"date"`
	ItemCount   int       `json:"item_count"`
	SourceCount int       `json:"source_count"`
	PublishedAt time.Time `json:"published_at"`
	Error       string    `json:"error"`
}

// ---------- A. 纯函数 ----------

// TestNewsValidNewsDate 日期格式校验:只接受 YYYY-MM-DD 且必须是真实存在的日期。
func TestNewsValidNewsDate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"常规日期", newsTestDate, true},
		{"闰年2月29日", "2024-02-29", true},
		{"平年2月29日", "2023-02-29", false},
		{"月份越界13", "2024-13-01", false},
		{"月份越界00", "2024-00-01", false},
		{"日期越界2月30", "2024-02-30", false},
		{"紧凑格式无分隔", "20240101", false},
		{"缺零的月日", "2026-1-05", false},
		{"空串", "", false},
		{"带前后空格", " 2026-10-25", false},
		{"含时间部分", "2026-10-25T00:00:00Z", false},
		{"超长串", "2026-10-25-extra-long-suffix", false},
	}
	for _, c := range cases {
		if got := validNewsDate(c.in); got != c.want {
			t.Errorf("[%s] validNewsDate(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

// TestNewsIsMostlyEnglish 英文判定:用于决定条目是否送 AI 翻译。
func TestNewsIsMostlyEnglish(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"空串", "", false},
		{"纯英文", "Hello world", true},
		{"长英文标题", "This is a long English headline about technology", true},
		{"纯中文", "你好，世界", false},
		{"中文长句", "今天天气不错，适合出门散步", false},
		{"英文占优的中英混排", "Hello 世界", true},
		{"中文占优的中英混排", "a 世界世界世界", false},
		{"含日文假名", "こんにちは", false},
		{"数字与符号", "123 !!!", false},
		{"英文缩写+中文", "AI 时代", true},
	}
	for _, c := range cases {
		if got := isMostlyEnglish(c.in); got != c.want {
			t.Errorf("[%s] isMostlyEnglish(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

// TestNewsParseTranslation 翻译结果解析:
// 实现只认「标题：/摘要：」行格式(半角冒号同样支持),解析不到时回退原文。
func TestNewsParseTranslation(t *testing.T) {
	cases := []struct {
		name        string
		out         string
		wantTitle   string
		wantSummary string
	}{
		{"全角冒号正常输出", "标题：中文标题\n摘要：中文摘要", "中文标题", "中文摘要"},
		{"半角冒号正常输出", "标题: 中文标题\n摘要: 中文摘要", "中文标题", "中文摘要"},
		{"多行噪声中提取", "思考中...\n标题：中文标题\n摘要：中文摘要\n完毕", "中文标题", "中文摘要"},
		{"只给标题时摘要回退", "标题：中文标题", "中文标题", "原文摘要"},
		{"只给摘要时标题回退", "摘要：中文摘要", "原文标题", "中文摘要"},
		{"无法解析时全部回退", "这是完全无关的输出", "原文标题", "原文摘要"},
		{"空输出时全部回退", "", "原文标题", "原文摘要"},
		{"JSON 输出不被识别(当前实现只认行格式)", `{"title":"JSON标题","summary":"JSON摘要"}`, "原文标题", "原文摘要"},
	}
	for _, c := range cases {
		title, summary, err := parseTranslation(c.out, "原文标题", "原文摘要")
		if err != nil {
			t.Errorf("[%s] parseTranslation 返回错误: %v", c.name, err)
			continue
		}
		if title != c.wantTitle {
			t.Errorf("[%s] title = %q, want %q", c.name, title, c.wantTitle)
		}
		if summary != c.wantSummary {
			t.Errorf("[%s] summary = %q, want %q", c.name, summary, c.wantSummary)
		}
	}
}

// TestNewsLabelForDate 日期标签:YYYY-MM-DD → 「2026年10月25日 周日」;非法日期原样返回。
func TestNewsLabelForDate(t *testing.T) {
	h := newsTestHandler(t, newsTestDB(t), nil)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"常规日期", newsTestDate, newsTestDateLabel},
		{"闰年2月29日", "2024-02-29", "2024年2月29日 周四"},
		{"非法日期原样返回", "2024-13-01", "2024-13-01"},
		{"空串原样返回", "", ""},
	}
	for _, c := range cases {
		if got := h.labelForDate(c.in); got != c.want {
			t.Errorf("[%s] labelForDate(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestNewsStripHTML 去标签 + 折叠空白。
// 注意:本函数不转义引号、也不做 HTML 实体解码(这是 FINN-184 的背景),测试按当前实现钉住语义。
func TestNewsStripHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"去掉成对标签", "<p>Hello <b>World</b></p>", "Hello World"},
		{"折叠连续空格", "  多   空格  ", "多 空格"},
		{"换行折成空格", "<div>\n换行\n</div>", "换行"},
		{"空串", "", ""},
		{"保留双引号(不转义)", `<p>他说"你好"</p>`, `他说"你好"`},
		{"不解析 HTML 实体", "<p>a &amp; b</p>", "a &amp; b"},
		{"纯文本原样", "普通文本", "普通文本"},
	}
	for _, c := range cases {
		if got := stripHTML(c.in); got != c.want {
			t.Errorf("[%s] stripHTML(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestNewsCleanTextAndFeedSourceName 抓取侧的文本清洗与来源名提取。
func TestNewsCleanTextAndFeedSourceName(t *testing.T) {
	if got := cleanText("  <b>hi</b>  "); got != "hi" {
		t.Errorf("cleanText = %q, want %q", got, "hi")
	}
	cases := []struct {
		in   string
		want string
	}{
		{"https://www.ithome.com/rss/", "ithome.com"},
		{"http://sspai.com/feed", "sspai.com"},
		{"https://news.ycombinator.com/rss", "news.ycombinator.com"},
	}
	for _, c := range cases {
		if got := feedSourceName(c.in); got != c.want {
			t.Errorf("feedSourceName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNewsNormalizePubDate RSS/Atom 时间字符串归一化为 RFC3339,解析不了返回空串。
func TestNewsNormalizePubDate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"RFC1123Z", "Mon, 02 Jan 2006 15:04:05 -0700", "2006-01-02T22:04:05Z"},
		{"ISO8601", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05Z"},
		{"带偏移的 ISO8601", "2006-01-02T15:04:05+08:00", "2006-01-02T07:04:05Z"},
		{"空串", "", ""},
		{"无法解析", "not a date", ""},
	}
	for _, c := range cases {
		if got := normalizePubDate(c.in); got != c.want {
			t.Errorf("[%s] normalizePubDate(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestNewsAtoiDefault 分页参数解析:空串/非法值走默认值。
func TestNewsAtoiDefault(t *testing.T) {
	cases := []struct {
		in   string
		def  int
		want int
	}{
		{"", 30, 30},
		{"12", 30, 12},
		{"abc", 30, 30},
		{"0", 30, 0},
	}
	for _, c := range cases {
		if got := atoiDefault(c.in, c.def); got != c.want {
			t.Errorf("atoiDefault(%q, %d) = %d, want %d", c.in, c.def, got, c.want)
		}
	}
}

// TestNewsTimezoneAndTodayDate 时区解析与「今天」取值。
func TestNewsTimezoneAndTodayDate(t *testing.T) {
	t.Run("时区为空时用本地时区", func(t *testing.T) {
		h := newsTestHandler(t, newsTestDB(t), func(c *config.Config) { c.News.Timezone = "" })
		if h.tz() != time.Local {
			t.Errorf("tz() = %v, want time.Local", h.tz())
		}
	})
	t.Run("时区非法时回落本地时区", func(t *testing.T) {
		h := newsTestHandler(t, newsTestDB(t), func(c *config.Config) { c.News.Timezone = "Not/AZone" })
		if h.tz() != time.Local {
			t.Errorf("tz() = %v, want time.Local(非法时区应回落)", h.tz())
		}
	})
	t.Run("时区为 Asia/Shanghai 时按该时区取今天", func(t *testing.T) {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			t.Skipf("本机缺少 Asia/Shanghai 时区数据: %v", err)
		}
		h := newsTestHandler(t, newsTestDB(t), func(c *config.Config) { c.News.Timezone = "Asia/Shanghai" })
		if h.tz().String() != loc.String() {
			t.Errorf("tz() = %v, want %v", h.tz(), loc)
		}
		want := time.Now().In(loc).Format("2006-01-02")
		if got := h.todayDate(); got != want {
			t.Errorf("todayDate() = %q, want %q", got, want)
		}
	})
	t.Run("todayDate 恒为 YYYY-MM-DD", func(t *testing.T) {
		h := newsTestHandler(t, newsTestDB(t), nil)
		got := h.todayDate()
		if len(got) != 10 || got[4] != '-' || got[7] != '-' {
			t.Errorf("todayDate() = %q, want YYYY-MM-DD", got)
		}
		if !validNewsDate(got) {
			t.Errorf("todayDate() = %q 不是合法日期", got)
		}
	})
}

// TestNewsNewHandlerDefaults NewNewsHandler 把非法/缺省配置归一化。
func TestNewsNewHandlerDefaults(t *testing.T) {
	h := newsTestHandler(t, newsTestDB(t), func(c *config.Config) {
		c.News.ScheduleHour = 99
		c.News.ScheduleMinute = 99
		c.News.MaxItems = -1
		c.News.Feeds = nil
	})
	if h.cfg.News.ScheduleHour != 8 {
		t.Errorf("ScheduleHour = %d, want 8", h.cfg.News.ScheduleHour)
	}
	if h.cfg.News.ScheduleMinute != 0 {
		t.Errorf("ScheduleMinute = %d, want 0", h.cfg.News.ScheduleMinute)
	}
	if h.cfg.News.MaxItems != newsDefaultMaxItems {
		t.Errorf("MaxItems = %d, want %d", h.cfg.News.MaxItems, newsDefaultMaxItems)
	}
	if len(h.cfg.News.Feeds) != len(defaultNewsFeeds) {
		t.Errorf("Feeds = %d 条, want %d 条内置默认源", len(h.cfg.News.Feeds), len(defaultNewsFeeds))
	}
}

// ---------- B. HTTP handler ----------

// TestNewsPublishAuth Publish 的鉴权分支。
// 当前实现语义:admin_password 为空 = 放行(不鉴权);非空时 header 或 query 命中即放行。
func TestNewsPublishAuth(t *testing.T) {
	cases := []struct {
		name       string
		adminPwd   string
		headerPwd  string
		queryPwd   string
		wantStatus int
	}{
		{"未配置密码时放行", "", "", "", http.StatusOK},
		{"配置密码后缺少凭据", newsTestAdminPwd, "", "", http.StatusForbidden},
		{"配置密码后 header 密码错误", newsTestAdminPwd, "wrong", "", http.StatusForbidden},
		{"配置密码后 query 密码错误", newsTestAdminPwd, "", "wrong", http.StatusForbidden},
		{"配置密码后 header 正确", newsTestAdminPwd, newsTestAdminPwd, "", http.StatusOK},
		{"配置密码后 query 正确", newsTestAdminPwd, "", newsTestAdminPwd, http.StatusOK},
	}
	for _, c := range cases {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, func(cfg *config.Config) { cfg.News.AdminPassword = c.adminPwd })
		target := "/api/news/publish"
		if c.queryPwd != "" {
			target += "?admin_password=" + url.QueryEscape(c.queryPwd)
		}
		ctx, rec := newNewsMockContext(t, http.MethodPost, target, `{"date":"`+newsTestDate+`","items":[]}`)
		if c.headerPwd != "" {
			ctx.Request.Header.Set("X-Admin-Password", c.headerPwd)
		}
		h.Publish(ctx)

		if rec.Code != c.wantStatus {
			t.Errorf("[%s] status = %d, want %d, body=%s", c.name, rec.Code, c.wantStatus, rec.Body.String())
			continue
		}
		var resp newsPublishResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("[%s] 响应不是合法 JSON: %v, body=%s", c.name, err, rec.Body.String())
			continue
		}
		if c.wantStatus == http.StatusForbidden {
			if resp.Error == "" {
				t.Errorf("[%s] 403 响应缺少 error 说明, body=%s", c.name, rec.Body.String())
			}
		} else if !resp.OK {
			t.Errorf("[%s] 200 响应 ok 应为 true, body=%s", c.name, rec.Body.String())
		}
	}
}

// TestNewsPublishValidation Publish 的入参校验与落库语义。
func TestNewsPublishValidation(t *testing.T) {
	t.Run("日期非法返回400", func(t *testing.T) {
		for _, date := range []string{"2024-13-01", "20240101", "not-a-date"} {
			db := newsTestDB(t)
			h := newsTestHandler(t, db, nil)
			body := `{"date":"` + date + `","items":[]}`
			ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", body)
			h.Publish(ctx)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("date=%q status = %d, want 400, body=%s", date, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("请求体非法返回400", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", `{"date":`)
		h.Publish(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("日期缺省时取今天", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", `{"items":[]}`)
		h.Publish(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp newsPublishResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.Date != h.todayDate() {
			t.Errorf("date = %q, want %q(今天)", resp.Date, h.todayDate())
		}
	})

	t.Run("空条目时状态为 empty", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", `{"date":"`+newsTestDate+`","items":[]}`)
		h.Publish(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp newsPublishResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.ItemCount != 0 || resp.SourceCount != 0 {
			t.Errorf("item_count=%d source_count=%d, want 0/0", resp.ItemCount, resp.SourceCount)
		}
		digest, err := db.GetNewsDigest(newsTestDate)
		if err != nil {
			t.Fatalf("读取落库归档失败: %v", err)
		}
		if digest.Status != "empty" {
			t.Errorf("status = %q, want %q", digest.Status, "empty")
		}
	})

	t.Run("来源去重且空来源被替换为未知来源", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		body := `{"date":"` + newsTestDate + `","items":[` +
			`{"title":"A","link":"https://a","source":"IT之家"},` +
			`{"title":"B","link":"https://b","source":"IT之家"},` +
			`{"title":"C","link":"https://c","source":""}]}`
		ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", body)
		h.Publish(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp newsPublishResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.ItemCount != 3 {
			t.Errorf("item_count = %d, want 3", resp.ItemCount)
		}
		if resp.SourceCount != 1 {
			t.Errorf("source_count = %d, want 1(空来源不计入)", resp.SourceCount)
		}
		digest, err := db.GetNewsDigest(newsTestDate)
		if err != nil {
			t.Fatalf("读取落库归档失败: %v", err)
		}
		items, err := models.UnmarshalNewsItems(digest.Items)
		if err != nil {
			t.Fatalf("解析落库条目失败: %v", err)
		}
		if len(items) != 3 {
			t.Fatalf("落库条目数 = %d, want 3", len(items))
		}
		if items[2].Source != newsUnknownSource {
			t.Errorf("空来源条目 source = %q, want %q", items[2].Source, newsUnknownSource)
		}
	})

	t.Run("ai_model 缺省时填充默认值", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", `{"date":"`+newsTestDate+`","items":[]}`)
		h.Publish(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		digest, err := db.GetNewsDigest(newsTestDate)
		if err != nil {
			t.Fatalf("读取落库归档失败: %v", err)
		}
		if digest.AIModel != newsDefaultAIModel {
			t.Errorf("ai_model = %q, want %q", digest.AIModel, newsDefaultAIModel)
		}
	})

	t.Run("ai_model 指定时原样保存", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		body := `{"date":"` + newsTestDate + `","items":[],"ai_model":"My-Agent"}`
		ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", body)
		h.Publish(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		digest, err := db.GetNewsDigest(newsTestDate)
		if err != nil {
			t.Fatalf("读取落库归档失败: %v", err)
		}
		if digest.AIModel != "My-Agent" {
			t.Errorf("ai_model = %q, want %q", digest.AIModel, "My-Agent")
		}
	})
}

// TestNewsPublishThenReadBack 发布后可被 GetLatest / GetByDate 读回。
func TestNewsPublishThenReadBack(t *testing.T) {
	db := newsTestDB(t)
	h := newsTestHandler(t, db, nil)
	body := `{"date":"` + newsTestDate + `","summary":"## 今日总览","items":[` +
		`{"title":"标题一","link":"https://a","source":"IT之家","summary":"摘要一","category":"科技","published":"2026-10-25T01:00:00Z"}]}`
	ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", body)
	h.Publish(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	t.Run("GetLatest 返回最新一期", func(t *testing.T) {
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/latest", "")
		h.GetLatest(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Digest *models.NewsDigest `json:"digest"`
			Config map[string]any     `json:"config"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.Digest == nil {
			t.Fatalf("digest 为空, body=%s", rec.Body.String())
		}
		if resp.Digest.Date != newsTestDate {
			t.Errorf("digest.date = %q, want %q", resp.Digest.Date, newsTestDate)
		}
		if resp.Digest.ItemCount != 1 || resp.Digest.SourceCount != 1 {
			t.Errorf("item_count=%d source_count=%d, want 1/1", resp.Digest.ItemCount, resp.Digest.SourceCount)
		}
		if resp.Digest.Summary != "## 今日总览" {
			t.Errorf("summary = %q, want %q", resp.Digest.Summary, "## 今日总览")
		}
		if resp.Digest.DateLabel != newsTestDateLabel {
			t.Errorf("date_label = %q, want %q", resp.Digest.DateLabel, newsTestDateLabel)
		}
		if resp.Config == nil {
			t.Error("响应缺少 config 字段")
		}
	})

	t.Run("GetByDate 按日期读回", func(t *testing.T) {
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/"+newsTestDate, "")
		ctx.Params = gin.Params{{Key: "date", Value: newsTestDate}}
		h.GetByDate(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Digest *models.NewsDigest `json:"digest"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.Digest == nil || resp.Digest.Date != newsTestDate {
			t.Fatalf("digest = %+v, want date=2026-10-25", resp.Digest)
		}
		items, err := models.UnmarshalNewsItems(resp.Digest.Items)
		if err != nil {
			t.Fatalf("解析条目失败: %v", err)
		}
		if len(items) != 1 || items[0].Title != "标题一" || items[0].Category != "科技" {
			t.Errorf("落库条目 = %+v, want 1 条标题「标题一」分类「科技」", items)
		}
	})
}

// TestNewsGetLatestEmptyDB 空库时 GetLatest 返回 digest=null 且仍带 config。
func TestNewsGetLatestEmptyDB(t *testing.T) {
	db := newsTestDB(t)
	h := newsTestHandler(t, db, nil)
	ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/latest", "")
	h.GetLatest(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Digest *models.NewsDigest `json:"digest"`
		Config map[string]any     `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp.Digest != nil {
		t.Errorf("空库时 digest 应为 null, got %+v", resp.Digest)
	}
	if resp.Config == nil {
		t.Error("空库时仍应返回 config")
	}
}

// TestNewsGetByDateErrors GetByDate 的错误分支:非法日期 400、日期不存在 404。
func TestNewsGetByDateErrors(t *testing.T) {
	t.Run("非法日期返回400", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/2024-13-01", "")
		ctx.Params = gin.Params{{Key: "date", Value: "2024-13-01"}}
		h.GetByDate(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("日期不存在返回404", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/"+newsTestDate, "")
		ctx.Params = gin.Params{{Key: "date", Value: newsTestDate}}
		h.GetByDate(ctx)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
		}
	})
}

// TestNewsListDates 归档索引接口:分页、总数与排序。
func TestNewsListDates(t *testing.T) {
	t.Run("空库返回空列表与 total=0", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/list", "")
		h.ListDates(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp struct {
			List   []models.NewsDigest `json:"list"`
			Total  int                 `json:"total"`
			Limit  int                 `json:"limit"`
			Offset int                 `json:"offset"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if len(resp.List) != 0 {
			t.Errorf("空库 list 长度 = %d, want 0", len(resp.List))
		}
		if resp.Total != 0 {
			t.Errorf("total = %d, want 0", resp.Total)
		}
		if resp.Limit != newsDefaultPageSize || resp.Offset != 0 {
			t.Errorf("limit=%d offset=%d, want %d/0(默认值)", resp.Limit, resp.Offset, newsDefaultPageSize)
		}
	})

	t.Run("有数据时按日期倒序并返回总数", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		for _, date := range []string{newsTestDatePrev2, newsTestDate, newsTestDatePrev} {
			ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/publish", `{"date":"`+date+`","items":[]}`)
			h.Publish(ctx)
			if rec.Code != http.StatusOK {
				t.Fatalf("publish %s status = %d, body=%s", date, rec.Code, rec.Body.String())
			}
		}
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/list", "")
		h.ListDates(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp struct {
			List  []models.NewsDigest `json:"list"`
			Total int                 `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.Total != 3 {
			t.Errorf("total = %d, want 3", resp.Total)
		}
		if len(resp.List) != 3 {
			t.Fatalf("list 长度 = %d, want 3", len(resp.List))
		}
		wantOrder := []string{newsTestDate, newsTestDatePrev, newsTestDatePrev2}
		for i, want := range wantOrder {
			if resp.List[i].Date != want {
				t.Errorf("list[%d].date = %q, want %q", i, resp.List[i].Date, want)
			}
		}
	})

	t.Run("limit 上限被截断为100", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/list?limit=1000", "")
		h.ListDates(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.Limit != newsMaxPageSize {
			t.Errorf("limit = %d, want %d(上限截断)", resp.Limit, newsMaxPageSize)
		}
	})

	t.Run("非法 limit 回退默认值30", func(t *testing.T) {
		db := newsTestDB(t)
		h := newsTestHandler(t, db, nil)
		ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/list?limit=abc", "")
		h.ListDates(ctx)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应不是合法 JSON: %v", err)
		}
		if resp.Limit != newsDefaultPageSize {
			t.Errorf("limit = %d, want %d(回退默认值)", resp.Limit, newsDefaultPageSize)
		}
	})
}

// TestNewsGetConfig 公开配置接口:字段齐全且不泄漏管理员密码。
func TestNewsGetConfig(t *testing.T) {
	db := newsTestDB(t)
	h := newsTestHandler(t, db, func(c *config.Config) {
		c.News.AdminPassword = newsTestAdminPwd
		c.News.Timezone = "Asia/Shanghai"
	})
	ctx, rec := newNewsMockContext(t, http.MethodGet, "/api/news/config", "")
	h.GetConfig(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	for _, key := range []string{"enabled", "schedule", "timezone", "feed_count", "has_ai", "max_items", "last_run_at", "today"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("config 缺少字段 %q", key)
		}
	}
	if resp["schedule"] != newsTestSchedule {
		t.Errorf("schedule = %v, want %q", resp["schedule"], newsTestSchedule)
	}
	if resp["timezone"] != "Asia/Shanghai" {
		t.Errorf("timezone = %v, want %q", resp["timezone"], "Asia/Shanghai")
	}
	if resp["has_ai"] != false {
		t.Errorf("has_ai = %v, want false(未配置模型)", resp["has_ai"])
	}
	if resp["max_items"].(float64) != newsDefaultMaxItems {
		t.Errorf("max_items = %v, want %d", resp["max_items"], newsDefaultMaxItems)
	}
	if resp["today"] != h.todayDate() {
		t.Errorf("today = %v, want %q", resp["today"], h.todayDate())
	}
	if strings.Contains(rec.Body.String(), newsTestAdminPwd) {
		t.Error("公开配置不应包含管理员密码")
	}
}

// TestNewsGenerateAuthRejected 手动触发接口的鉴权分支。
// 只验证鉴权(403);鉴权通过后会真的去抓 RSS,不在单测里跑。
func TestNewsGenerateAuthRejected(t *testing.T) {
	db := newsTestDB(t)
	h := newsTestHandler(t, db, func(c *config.Config) { c.News.AdminPassword = newsTestAdminPwd })
	ctx, rec := newNewsMockContext(t, http.MethodPost, "/api/news/generate", "")
	h.Generate(ctx)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp.Error == "" {
		t.Error("403 响应缺少 error 说明")
	}
}
