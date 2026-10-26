package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"devtools/config"
	"devtools/models"

	"github.com/gin-gonic/gin"
)

// defaultNewsFeeds 未配置 feeds 时使用的内置订阅源(无需鉴权、公开 RSS/Atom)。
// 若部署环境网络受限,可自行在 config.yaml 的 news.feeds 覆盖。
var defaultNewsFeeds = []string{
	"https://www.ithome.com/rss/",
	"https://www.cnbeta.com/backend.php?type=feed",
	"https://sspai.com/feed",
	"https://www.ruanyifeng.com/blog/atom.xml",
	"https://news.ycombinator.com/rss",
	"https://feeds.bbci.co.uk/zhongwen/simp/rss.xml",
}

// htmlTagRe 用于把 RSS 摘要里的 HTML 标签剥掉。
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// NewsHandler 每日新闻模块:抓取 RSS/Atom → 去重 → 可选 AI 精编 → 存库归档。
type NewsHandler struct {
	db        *models.DB
	cfg       *config.Config
	ai        *AIGatewayHandler
	mu        sync.Mutex
	running   bool
	lastRunAt time.Time
}

func NewNewsHandler(db *models.DB, cfg *config.Config, ai *AIGatewayHandler) *NewsHandler {
	nc := cfg.News
	if nc.ScheduleHour < 0 || nc.ScheduleHour > 23 {
		nc.ScheduleHour = 8
	}
	if nc.ScheduleMinute < 0 || nc.ScheduleMinute > 59 {
		nc.ScheduleMinute = 0
	}
	if nc.MaxItems <= 0 {
		nc.MaxItems = 30
	}
	if len(nc.Feeds) == 0 {
		nc.Feeds = defaultNewsFeeds
	}
	// 把规范化后的默认值写回全局配置,供 GetConfig / 调度器读取。
	cfg.News = nc
	return &NewsHandler{db: db, cfg: cfg, ai: ai}
}

// ---------- 调度器 ----------

// StartScheduler 启动每日定时生成(在独立 goroutine 中运行,启动后若当日尚无归档会延迟 30s 先生成一次)。
// 若 internal_scheduler=false,则完全关闭后端内置定时任务,改由外部(如 WorkBuddy 自动化)调用 /api/news/generate 触发。
func (h *NewsHandler) StartScheduler() {
	if !h.cfg.News.Enabled {
		log.Printf("[news] 模块未启用(enabled=false),跳过定时任务")
		return
	}
	if !h.cfg.News.InternalSchedulerEnabled() {
		log.Printf("[news] 后端内置定时任务已关闭(internal_scheduler=false),由外部(如 WorkBuddy 自动化)调用 /api/news/generate 触发")
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[news] scheduler panic: %v", r)
			}
		}()
		// 启动后若今天还没有归档,延迟生成一次(避免清晨部署后当天空白)。
		if _, err := h.db.GetNewsDigest(h.todayDate()); err != nil {
			time.Sleep(30 * time.Second)
			_ = h.GenerateInternal(context.Background())
		}
		h.scheduleLoop()
	}()
}

func (h *NewsHandler) scheduleLoop() {
	for {
		next := h.nextRunTime()
		log.Printf("[news] 下次新闻生成时间: %s", next.Format(time.RFC3339))
		time.Sleep(time.Until(next))
		_ = h.GenerateInternal(context.Background())
	}
}

func (h *NewsHandler) nextRunTime() time.Time {
	loc := h.tz()
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), h.cfg.News.ScheduleHour, h.cfg.News.ScheduleMinute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func (h *NewsHandler) tz() *time.Location {
	if h.cfg.News.Timezone != "" {
		if loc, err := time.LoadLocation(h.cfg.News.Timezone); err == nil {
			return loc
		}
	}
	return time.Local
}

func (h *NewsHandler) todayDate() string {
	return time.Now().In(h.tz()).Format("2006-01-02")
}

func (h *NewsHandler) todayLabel() string {
	t := time.Now().In(h.tz())
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return fmt.Sprintf("%d年%d月%d日 %s", t.Year(), t.Month(), t.Day(), weekdays[t.Weekday()])
}

// ---------- 生成核心 ----------

// GenerateInternal 抓取并生成当日新闻归档(调度器与手动接口共用)。已加锁防并发重入。
func (h *NewsHandler) GenerateInternal(ctx context.Context) error {
	// 安全超时:防止某个 RSS 源或 AI 精编请求长时间挂起,拖垮每日定时任务。
	ctx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return nil
	}
	h.running = true
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.running = false
		h.lastRunAt = time.Now()
		h.mu.Unlock()
	}()

	feeds := h.cfg.News.Feeds
	results := make([][]models.NewsItem, len(feeds))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for i, f := range feeds {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items, err := h.fetchFeed(ctx, url)
			if err != nil {
				log.Printf("[news] 抓取 %s 失败: %v", url, err)
				return
			}
			results[i] = items
		}(i, f)
	}
	wg.Wait()

	seen := make(map[string]bool)
	var all []models.NewsItem
	srcSet := make(map[string]bool)
	for _, items := range results {
		for _, it := range items {
			if it.Link == "" && it.Title == "" {
				continue
			}
			key := it.Link
			if key == "" {
				key = it.Title
			}
			hkey := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
			if seen[hkey] {
				continue
			}
			seen[hkey] = true
			if it.Source != "" {
				srcSet[it.Source] = true
			}
			all = append(all, it)
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Published > all[j].Published
	})
	if len(all) > h.cfg.News.MaxItems {
		all = all[:h.cfg.News.MaxItems]
	}

	// 英文条目翻译:配置了 AI 模型且开启翻译时,把英文新闻逐条翻译为简体中文(非英文原样保留)。
	if h.cfg.News.Model != "" && h.ai != nil && h.cfg.News.TranslateEnabled() {
		all = h.translateItems(ctx, all)
	}

	digest := &models.NewsDigest{
		Date:        h.todayDate(),
		DateLabel:   h.todayLabel(),
		Items:       models.MarshalNewsItems(all),
		ItemCount:   len(all),
		SourceCount: len(srcSet),
		Status:      "ok",
		GeneratedAt: time.Now(),
	}
	if len(all) == 0 {
		digest.Status = "empty"
	}

	if h.cfg.News.Model != "" && h.ai != nil {
		if summary, err := h.generateSummary(ctx, all); err != nil {
			log.Printf("[news] AI 精编失败(已降级为原文): %v", err)
			if digest.Status == "ok" {
				digest.Status = "partial"
			}
			digest.ErrorMsg = err.Error()
		} else {
			digest.Summary = summary
			digest.AIModel = h.cfg.News.Model
		}
	}

	if err := h.db.UpsertNewsDigest(digest); err != nil {
		return err
	}

	// 历史保留:retention_days>0 时才清理更早的归档(默认永久保留)。
	if h.cfg.News.RetentionDays > 0 {
		before := time.Now().AddDate(0, 0, -h.cfg.News.RetentionDays)
		if n, err := h.db.DeleteNewsDigestsBefore(before); err == nil && n > 0 {
			log.Printf("[news] 已清理 %d 条过期新闻归档", n)
		}
	}
	log.Printf("[news] 已生成 %s 新闻: %d 条 / %d 源 (status=%s)", digest.Date, digest.ItemCount, digest.SourceCount, digest.Status)
	return nil
}

func (h *NewsHandler) generateSummary(ctx context.Context, items []models.NewsItem) (string, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("以下是今日抓取到的新闻条目(共 %d 条,已去重并翻译为简体中文,按时间排序):\n\n", len(items)))
	for i, it := range items {
		src := it.Source
		if src == "" {
			src = "未知来源"
		}
		summary := it.Summary
		if len([]rune(summary)) > 200 {
			summary = string([]rune(summary)[:200]) + "…"
		}
		fmt.Fprintf(&b, "%d. 【%s】%s", i+1, src, it.Title)
		if summary != "" {
			fmt.Fprintf(&b, "%s", "："+summary)
		}
		b.WriteString("\n")
	}

	system := "你是一名资深中文新闻编辑。根据下方抓取的今日新闻条目(已去重、翻译为简体中文、按时间排序)," +
		"撰写一份全面、专业、可读性强的「今日要闻」日报(使用 Markdown)。要求:\n" +
		"1) 开头先写一段『今日总览』:共多少条新闻、来自哪些来源、覆盖哪些主要领域;\n" +
		"2) 按来源或主题分组,确保覆盖全部条目,不要遗漏任何一条;\n" +
		"3) 每条新闻用一句话概括其核心事实或影响,句后括号标注来源;\n" +
		"4) 相关联的多条可合并评述并指出趋势;\n" +
		"5) 不编造未出现的信息,语言精炼、客观、流畅;\n" +
		"6) 力求『面面俱到』,但避免冗余啰嗦。\n" +
		"只输出日报正文,不要额外解释或标题。"
	return h.ai.Complete(ctx, h.cfg.News.Model, system, b.String())
}

// ---------- 英文翻译 ----------

// isMostlyEnglish 粗略判断文本是否以英文为主(用于决定是否翻译)。
func isMostlyEnglish(s string) bool {
	if s == "" {
		return false
	}
	var cjk, latin int
	for _, r := range s {
		switch {
		case (r >= 0x3040 && r <= 0x30FF) || // 日文假名
			(r >= 0x3400 && r <= 0x4DBF) || // 扩展 A
			(r >= 0x4E00 && r <= 0x9FFF) || // 汉字
			(r >= 0xF900 && r <= 0xFAFF): // 兼容汉字
			cjk++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			latin++
		}
	}
	if cjk == 0 {
		return latin > 0
	}
	return float64(latin) > float64(cjk)*0.5
}

// translateItems 并发把英文条目翻译为简体中文,非英文条目原样返回。
func (h *NewsHandler) translateItems(ctx context.Context, items []models.NewsItem) []models.NewsItem {
	need := make([]int, 0, len(items))
	for i := range items {
		if isMostlyEnglish(items[i].Title) {
			need = append(need, i)
		}
	}
	if len(need) == 0 {
		return items
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, i := range need {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			zhTitle, zhSum, err := h.translateOne(ctx, items[i].Title, items[i].Summary)
			if err != nil {
				log.Printf("[news] 翻译失败(保留原文): %v", err)
				return
			}
			items[i].Title = zhTitle
			if zhSum != "" {
				items[i].Summary = zhSum
			}
		}(i)
	}
	wg.Wait()
	return items
}

// translateOne 调用 AI 把单条英文新闻的标题与摘要翻译为简体中文。
func (h *NewsHandler) translateOne(ctx context.Context, title, summary string) (string, string, error) {
	system := "你是一名专业的中英新闻翻译。请将用户提供的英文新闻条目准确翻译为简体中文,保持专业、客观、简洁。只输出要求格式,不要多余解释或标点之外的字符。"
	user := fmt.Sprintf("请将下面的新闻条目翻译为简体中文。严格按以下两行格式输出:\n标题：<中文标题>\n摘要：<中文摘要>\n\n原文标题：%s\n原文摘要：%s",
		title, summary)
	out, err := h.ai.Complete(ctx, h.cfg.News.Model, system, user)
	if err != nil {
		return "", "", err
	}
	return parseTranslation(out, title, summary)
}

// parseTranslation 从模型输出中解析「标题：」与「摘要：」两行,解析失败回退原文。
func parseTranslation(out, fallbackTitle, fallbackSummary string) (string, string, error) {
	title := ""
	summary := ""
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(ln, "标题："):
			title = strings.TrimSpace(strings.TrimPrefix(ln, "标题："))
		case strings.HasPrefix(ln, "标题:"):
			title = strings.TrimSpace(strings.TrimPrefix(ln, "标题:"))
		case strings.HasPrefix(ln, "摘要："):
			summary = strings.TrimSpace(strings.TrimPrefix(ln, "摘要："))
		case strings.HasPrefix(ln, "摘要:"):
			summary = strings.TrimSpace(strings.TrimPrefix(ln, "摘要:"))
		}
	}
	if title == "" {
		title = fallbackTitle
	}
	if summary == "" {
		summary = fallbackSummary
	}
	return title, summary, nil
}

// ---------- HTTP 接口 ----------

// Generate 手动触发生成(也供外部定时任务/同步接口调用)。受 admin_password 保护。
func (h *NewsHandler) Generate(c *gin.Context) {
	if pwd := h.cfg.News.AdminPassword; pwd != "" {
		if c.GetHeader("X-Admin-Password") != pwd && c.Query("admin_password") != pwd {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员密码"})
			return
		}
	}
	if err := h.GenerateInternal(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "date": h.todayDate(), "generated_at": h.lastRunAt})
}

// NewsPublishRequest 智能体(WorkBuddy 本地定时任务)采编好之后发布的日报结构。
// 后端只负责接收、存储与展示;抓取/翻译/摘要撰写全部由智能体在本地完成。
type NewsPublishRequest struct {
	Date    string            `json:"date"`     // YYYY-MM-DD,缺省=今天(本地时区)
	Summary string            `json:"summary"`  // 面面俱到日报(Markdown),由智能体撰写
	Items   []models.NewsItem `json:"items"`    // 智能体整理好的条目(标题/链接/来源/摘要/分类/时间)
	AIModel string            `json:"ai_model"` // 采编者标识,缺省=WorkBuddy-Agent
}

// Publish 接收智能体(WorkBuddy 本地定时任务)采编好的日报并发布/归档。受 admin_password 保护。
// 这是「本地收集整理好 然后发布」的发布入口:采编在 WorkBuddy 侧完成,后端只做存储与展示。
func (h *NewsHandler) Publish(c *gin.Context) {
	if pwd := h.cfg.News.AdminPassword; pwd != "" {
		if c.GetHeader("X-Admin-Password") != pwd && c.Query("admin_password") != pwd {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员密码"})
			return
		}
	}
	var req NewsPublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体非法: " + err.Error()})
		return
	}
	date := req.Date
	if date == "" {
		date = h.todayDate()
	}
	if !validNewsDate(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日期,应为 YYYY-MM-DD"})
		return
	}
	items := req.Items
	if items == nil {
		items = []models.NewsItem{}
	}
	srcSet := make(map[string]bool)
	for i := range items {
		if items[i].Source == "" {
			items[i].Source = "未知来源"
		} else {
			srcSet[items[i].Source] = true
		}
	}
	status := "ok"
	if len(items) == 0 {
		status = "empty"
	}
	aiModel := req.AIModel
	if aiModel == "" {
		aiModel = "WorkBuddy-Agent"
	}
	digest := &models.NewsDigest{
		Date:        date,
		DateLabel:   h.labelForDate(date),
		Summary:     req.Summary,
		Items:       models.MarshalNewsItems(items),
		ItemCount:   len(items),
		SourceCount: len(srcSet),
		AIModel:     aiModel,
		Status:      status,
		GeneratedAt: time.Now(),
	}
	if err := h.db.UpsertNewsDigest(digest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("[news] 已发布 %s 新闻(智能体采编): %d 条 / %d 源 (status=%s)", digest.Date, digest.ItemCount, digest.SourceCount, digest.Status)
	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"date":         date,
		"item_count":   len(items),
		"source_count": len(srcSet),
		"published_at": digest.GeneratedAt,
	})
}

// labelForDate 由 YYYY-MM-DD 生成「2026年10月25日 周日」样式的展示标签。
func (h *NewsHandler) labelForDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return fmt.Sprintf("%d年%d月%d日 %s", t.Year(), t.Month(), t.Day(), weekdays[t.Weekday()])
}

// GetLatest 返回最新一期(可能是今天或最近一次生成)。
func (h *NewsHandler) GetLatest(c *gin.Context) {
	d, err := h.db.GetLatestNewsDigest()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"digest": nil, "config": h.publicConfig()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"digest": d, "config": h.publicConfig()})
}

// GetByDate 返回指定日期归档。
func (h *NewsHandler) GetByDate(c *gin.Context) {
	date := c.Param("date")
	if !validNewsDate(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日期,应为 YYYY-MM-DD"})
		return
	}
	d, err := h.db.GetNewsDigest(date)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该日期的新闻归档"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"digest": d})
}

// ListDates 返回归档日期索引(分页),供历史浏览。
func (h *NewsHandler) ListDates(c *gin.Context) {
	limit := atoiDefault(c.Query("limit"), 30)
	offset := atoiDefault(c.Query("offset"), 0)
	if limit > 100 {
		limit = 100
	}
	list, err := h.db.ListNewsDates(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	total, _ := h.db.CountNewsDigests()
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total, "limit": limit, "offset": offset})
}

// GetConfig 返回公开配置(不含密码),供前端展示调度信息。
func (h *NewsHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, h.publicConfig())
}

func (h *NewsHandler) publicConfig() gin.H {
	hasAI := h.cfg.News.Model != "" && h.ai != nil
	enabled := h.cfg.News.Enabled
	return gin.H{
		"enabled":     enabled,
		"schedule":    fmt.Sprintf("%02d:%02d", h.cfg.News.ScheduleHour, h.cfg.News.ScheduleMinute),
		"timezone":    h.cfg.News.Timezone,
		"feed_count":  len(h.cfg.News.Feeds),
		"has_ai":      hasAI,
		"max_items":   h.cfg.News.MaxItems,
		"last_run_at": h.lastRunAt,
		"today":       h.todayDate(),
	}
}

// ---------- RSS/Atom 抓取 ----------

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	Desc    string `xml:"description"`
	PubDate string `xml:"pubDate"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string   `xml:"title"`
	Link      atomLink `xml:"link"`
	Summary   string   `xml:"summary"`
	Content   string   `xml:"content"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

func (h *NewsHandler) httpClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func (h *NewsHandler) fetchFeed(ctx context.Context, url string) ([]models.NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DevToolsNewsBot/1.0)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

	resp, err := h.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}

	source := feedSourceName(url)

	// 先尝试 RSS 2.0
	var rf rssFeed
	if err := xml.Unmarshal(body, &rf); err == nil && len(rf.Channel.Items) > 0 {
		if source == "" {
			source = cleanText(rf.Channel.Title)
		}
		items := make([]models.NewsItem, 0, len(rf.Channel.Items))
		for _, it := range rf.Channel.Items {
			items = append(items, models.NewsItem{
				Title:     cleanText(it.Title),
				Link:      strings.TrimSpace(it.Link),
				Source:    source,
				Summary:   stripHTML(it.Desc),
				Published: normalizePubDate(it.PubDate),
			})
		}
		return items, nil
	}

	// 再试 Atom
	var af atomFeed
	if err := xml.Unmarshal(body, &af); err == nil && len(af.Entries) > 0 {
		if source == "" {
			source = cleanText(af.Title)
		}
		items := make([]models.NewsItem, 0, len(af.Entries))
		for _, e := range af.Entries {
			summary := e.Summary
			if summary == "" {
				summary = e.Content
			}
			pub := e.Published
			if pub == "" {
				pub = e.Updated
			}
			items = append(items, models.NewsItem{
				Title:     cleanText(e.Title),
				Link:      strings.TrimSpace(e.Link.Href),
				Source:    source,
				Summary:   stripHTML(summary),
				Published: normalizePubDate(pub),
			})
		}
		return items, nil
	}

	return nil, fmt.Errorf("无法解析为 RSS/Atom: %s", url)
}

func feedSourceName(url string) string {
	for _, prefix := range []string{"https://", "http://"} {
		url = strings.TrimPrefix(url, prefix)
	}
	url = strings.TrimPrefix(url, "www.")
	if i := strings.Index(url, "/"); i >= 0 {
		url = url[:i]
	}
	return url
}

func cleanText(s string) string {
	s = stripHTML(s)
	s = strings.TrimSpace(s)
	return s
}

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// normalizePubDate 把常见 RSS/Atom 时间字符串转成 RFC3339(解析失败返回空,由前端按抓取到顺序排列)。
func normalizePubDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	layouts := []string{
		time.RFC1123Z, time.RFC1123,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return ""
}

func validNewsDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}
