package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"devtools/config"
	"devtools/models"
	"devtools/state"
	"devtools/version"

	"github.com/gin-gonic/gin"
)

// ============================================================
// /api/health 依赖探测
//
// 两种形态，共用同一个 handler：
//   - 默认形态 GET /api/health      → 恒 200 {"status":"ok"}，字节级不变。
//     不探测、不鉴权、不读 header、不加任何响应头。
//     deploy.sh 与 docker-compose 的 healthcheck 依赖此形态（探活超时 3s）。
//   - 详细形态 GET /api/health?detail=1 → 需 X-Super-Admin-Password 鉴权，
//     鉴权语义与 handlers/monitoring.go 的 requireAdmin 一致（401 / 403）；
//     授权后恒 200，返回版本、uptime 与 5 个依赖的实时状态。
//
// 详细形态是公网可达的（t.jaxiu.cn → Nginx → devtools），因此：
//   1. 必须鉴权（未授权 401、未配置密码 403）；
//   2. message 只能取自下面的白名单闭集，绝不回原始 err.Error()
//      （`Get "http://ocr-service:8000/health": dial tcp 172.18.0.4:8000: ...`
//      会同时泄露 URL、容器服务名、内网 IP、端口）；
//   3. 带 Cache-Control: no-store, private + Vary: X-Super-Admin-Password，
//      防 Cloudflare / 中间层把已授权响应回给未授权访客。
// ============================================================

const (
	// healthProbeBudget 是整轮探测的硬上限（父 ctx）。
	// compose healthcheck timeout 是 3s（docker-compose.yml:50），留 1s 余量。
	healthProbeBudget = 2000 * time.Millisecond
	// healthHTTPTimeout 单依赖 HTTP 探测超时；5 个依赖并发，最坏 ≈ 800ms。
	healthHTTPTimeout = 800 * time.Millisecond
	// healthRedisTimeout Redis PING 超时。
	healthRedisTimeout = 300 * time.Millisecond
	// healthSQLiteTimeout SQLite Ping/Query 超时。
	healthSQLiteTimeout = 300 * time.Millisecond
	// healthCacheTTL 结果缓存窗口，抵御 deploy.sh 1s 间隔的轮询风暴。
	healthCacheTTL = 2000 * time.Millisecond
)

// 依赖名（逻辑 ID，绝不返回容器服务名）。
const (
	depSQLite = "sqlite"
	depRedis  = "redis"
	depOCR    = "ocr"
	depASR    = "asr"
	depTTS    = "tts"
)

var healthDepNames = [5]string{depSQLite, depRedis, depOCR, depASR, depTTS}

// 依赖状态枚举。
const (
	healthStatusUp       = "up"
	healthStatusDown     = "down"
	healthStatusWarming  = "warming"
	healthStatusDisabled = "disabled"
)

// 依赖关键性枚举。
const (
	healthCritCritical = "critical"
	healthCritSoft     = "soft"
)

// 顶层 status 枚举（与 HTTP 状态码无关，授权后恒 200）。
const (
	healthOverallOK       = "ok"
	healthOverallDegraded = "degraded"
	healthOverallError    = "error"
)

// 地址来源枚举。
const (
	healthSourceEnv     = "env"
	healthSourceDefault = "default"
)

// message 闭集白名单。message 字段的唯一赋值路径就是下面这些字面量
// 加 classifyProbeErr 的返回值，禁止任何 err.Error() 直达响应体。
const (
	healthMsgConnectRefused = "connection refused"
	healthMsgTimeout        = "timeout"
	healthMsgDNSFailure     = "DNS 解析失败（地址可能仅适用于容器内）"
	healthMsgWarmingUp      = "warming up"
	healthMsgRedisDisabled  = "未启用（使用内存存储）"
	healthMsgRedisNoAddr    = "已启用但未配置地址"
	healthMsgRedisDegraded  = "已降级到内存存储"
	healthMsgProbeFailed    = "探测失败"
)

// depTarget 是构造期解析一次、之后只读的探测目标。
// 读取时机必须与既有实现一致（构造期），绝不 per-request 重读 env，
// 否则「探测的地址」会与「实际调用用的地址」在运行期分叉，详情页会说谎。
type depTarget struct {
	base   string // 空 = 无网络地址（sqlite）
	source string // env | default
}

// HealthDependency 是 dependencies 数组的元素。
// 字段永远存在（不带 omitempty），前端不需要处理「字段偶发缺失」；
// 唯一例外是 backend —— 按字段表只对 redis 项返回。
type HealthDependency struct {
	Name        string `json:"name"`
	Criticality string `json:"criticality"`
	Status      string `json:"status"`
	LatencyMS   int64  `json:"latency_ms"`
	Source      string `json:"source"`
	Message     string `json:"message"`
	Backend     string `json:"backend,omitempty"`
}

// HealthPayload 是详细形态的响应体。
type HealthPayload struct {
	Status        string             `json:"status"`
	Version       string             `json:"version"`
	Commit        string             `json:"commit"`
	BuildTime     string             `json:"build_time"`
	UptimeSeconds int64              `json:"uptime_seconds"`
	StartedAt     string             `json:"started_at"`
	ServerTime    string             `json:"server_time"`
	ProbedAt      string             `json:"probed_at"`
	ProbeMS       int64              `json:"probe_ms"`
	Cached        bool               `json:"cached"`
	Dependencies  []HealthDependency `json:"dependencies"`
}

type HealthHandler struct {
	db        *models.DB
	cfg       *config.Config
	store     state.TransientStore
	startedAt time.Time
	client    *http.Client

	// 构造期解析一次（见 depTarget 注释）。
	ocrTarget    depTarget
	asrTarget    depTarget
	ttsTarget    depTarget
	sqliteSource string
	redisSource  string

	version   string
	commit    string
	buildTime string

	// 探测快照，2s TTL。mutex 只保护这两个字段。
	// 不做 single-flight（缓存冷时的并发请求各自探测一次）：并发探活只来自已授权前端
	// 的 60s 轮询，且默认形态根本不探测，最坏情况仍在预算内。
	snapshotMu sync.Mutex
	snapshot   *HealthPayload
	snapshotAt time.Time
}

// NewHealthHandler 构造 health handler。
// 地址全部在构造期解析一次，逐依赖对齐既有实现：
//   - ocr   → os.Getenv("OCR_SERVICE_URL")，空则 http://ocr-service:8000
//     （与 handlers/ocr.go:24-27 逐字一致：不 trim）
//   - asr   → os.Getenv("ASR_SERVICE_URL")，空则 http://asr-service:9000
//     （与 app_handlers.go 的 envOrDefault 一致）
//   - tts   → cfg.Chat.TTSServiceURL（env TTS_SERVICE_URL 已在 config.go 消费完，
//     此处绝不二次 os.Getenv，否则会产生双份默认值逻辑）。默认
//     http://127.0.0.1:8083 是「TTS 与 devtools 同容器」形态下的正确值
//     （entrypoint.sh 在容器内拉起 sidecar），不要「顺手修正」成服务名。
//   - redis → cfg.Redis（与 app_runtime.go 的 state.New(rt.cfg.Redis) 同一份），
//     容器形态靠 docker-compose 的 REDIS_ADDR=redis:6379 覆盖，
//     这里绝不出现 127.0.0.1:6379 字面量。
//   - sqlite→ 复用进程内 *models.DB 实例，不新建连接、不读 DB_PATH。
func NewHealthHandler(db *models.DB, cfg *config.Config, store state.TransientStore) *HealthHandler {
	h := &HealthHandler{
		db:        db,
		cfg:       cfg,
		store:     store,
		startedAt: time.Now(),
		// 照抄 handlers/ocr.go:29-34：禁用代理，确保能访问 Docker 内部网络服务。
		// 容器 env_file 可能带 HTTP_PROXY，不禁用会出现「探内网服务却走了外网代理」的假 down
		// （docker-compose 里专门设了 no_proxy，说明这条真实发生过）。
		client: &http.Client{
			Transport: &http.Transport{Proxy: nil},
			Timeout:   healthHTTPTimeout,
		},
		version:   version.Resolve(),
		commit:    version.Commit,
		buildTime: version.BuildTime,
	}

	// OCR 默认值只对「进程在 compose 网络内」的形态正确；本地 go run 解析不了
	// ocr-service 名字，会恒 down —— 这是预期内表现，靠 source=default + DNS 消息解释。
	h.ocrTarget = depTarget{base: envOr("OCR_SERVICE_URL", "http://ocr-service:8000"), source: sourceOf("OCR_SERVICE_URL")}
	h.asrTarget = depTarget{base: envOr("ASR_SERVICE_URL", "http://asr-service:9000"), source: sourceOf("ASR_SERVICE_URL")}
	h.ttsTarget = depTarget{base: cfg.Chat.TTSServiceURL, source: sourceOf("TTS_SERVICE_URL")}

	h.sqliteSource = sourceOf("DB_PATH")
	h.redisSource = sourceOf("REDIS_ADDR")

	if h.commit == "" {
		h.commit = "unknown"
	}
	if h.buildTime == "" {
		h.buildTime = "unknown"
	}
	if h.version == "" {
		h.version = "dev"
	}
	return h
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// sourceOf 只暴露「是否来自本次显式设置的环境变量」这一位，不暴露地址原文。
func sourceOf(key string) string {
	if os.Getenv(key) != "" {
		return healthSourceEnv
	}
	return healthSourceDefault
}

// Handle 处理 GET /api/health。
func (h *HealthHandler) Handle(c *gin.Context) {
	if !requestWantsDetail(c) {
		// 默认形态：与改造前的内联 handler 逐字节一致。
		// 不探测、不鉴权、不读 header、不加任何响应头 —— 探活路径与依赖健康彻底解耦。
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	setDetailCacheHeaders(c)

	if !h.authorize(c) {
		return
	}

	payload := h.snapshotOrProbe()

	// server_time / uptime_seconds 每次请求实时计算，不参与缓存。
	now := time.Now().UTC()
	uptime := int64(time.Since(h.startedAt).Seconds())
	if uptime < 0 {
		uptime = 0
	}
	payload.ServerTime = now.Format(time.RFC3339)
	payload.UptimeSeconds = uptime
	// 保证 server_time - started_at == uptime_seconds，两个字段不会互相矛盾。
	payload.StartedAt = now.Add(-time.Duration(uptime) * time.Second).Format(time.RFC3339)

	c.JSON(http.StatusOK, payload)
}

// requestWantsDetail 识别 ?detail=1 等写法（大小写不敏感、去空格）。
// 其余任何值按默认形态走，不返回 400。
func requestWantsDetail(c *gin.Context) bool {
	v := strings.ToLower(strings.TrimSpace(c.Query("detail")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// setDetailCacheHeaders 只作用于详细形态（含 401/403 出口），默认形态一个头都不加。
func setDetailCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, private")
	c.Header("Vary", "X-Super-Admin-Password")
	c.Header("X-Content-Type-Options", "nosniff")
}

// authorize 与 handlers/monitoring.go 的 requireAdmin 语义一致：
// 三级密码 fallback、header X-Super-Admin-Password、为空时回落 query
// super_admin_password（保留与既有端点一致的回落，作为已知残余风险）。
// 不给 401 加 WWW-Authenticate —— 那会触发浏览器原生登录框。
func (h *HealthHandler) authorize(c *gin.Context) bool {
	configured := strings.TrimSpace(h.cfg.Monitoring.AdminPassword)
	if configured == "" {
		configured = strings.TrimSpace(h.cfg.AIGateway.SuperAdminPassword)
	}
	if configured == "" {
		configured = strings.TrimSpace(h.cfg.Console.AdminPassword)
	}
	if configured == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "未配置 monitoring.admin_password、ai_gateway.super_admin_password 或 console.admin_password"})
		return false
	}
	password := c.GetHeader("X-Super-Admin-Password")
	if password == "" {
		password = c.Query("super_admin_password")
	}
	if password != configured {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "管理员密码错误"})
		return false
	}
	return true
}

// snapshotOrProbe 命中 2s 缓存则直接返回副本，否则真探测并刷新快照。
// 鉴权在缓存之外：未授权请求永远不会碰缓存。
func (h *HealthHandler) snapshotOrProbe() HealthPayload {
	h.snapshotMu.Lock()
	if h.snapshot != nil && time.Since(h.snapshotAt) < healthCacheTTL {
		p := *h.snapshot
		p.Cached = true
		p.ProbeMS = 0
		p.ProbedAt = h.snapshotAt.UTC().Format(time.RFC3339Nano)
		h.snapshotMu.Unlock()
		return p
	}
	h.snapshotMu.Unlock()

	probedAt := time.Now()
	payload := h.probeAll()
	payload.ProbedAt = probedAt.UTC().Format(time.RFC3339Nano)

	h.snapshotMu.Lock()
	h.snapshot = &payload
	h.snapshotAt = probedAt
	h.snapshotMu.Unlock()

	return payload
}

// probeAll 并发探测 5 个依赖。
//   - 并发而非串行：串行最坏 5×800ms = 4s，直接击穿 3s 探活超时（虽然默认形态不探测，
//     但详细形态被 CF 524 截断同样是事故）；
//   - 每个 goroutine 写自己的下标，无需加锁；
//   - 每个 goroutine 必须 recover：任一探测 panic 只能把该项标 down，绝不冒泡成 500。
func (h *HealthHandler) probeAll() HealthPayload {
	results := [5]HealthDependency{}

	ctx, cancel := context.WithTimeout(context.Background(), healthProbeBudget)
	defer cancel()

	probes := [5]func(context.Context) HealthDependency{
		h.probeSQLite,
		h.probeRedis,
		h.probeOCR,
		h.probeASR,
		h.probeTTS,
	}

	start := time.Now()
	var wg sync.WaitGroup
	for i := range probes {
		wg.Add(1)
		go func(i int, fn func(context.Context) HealthDependency) {
			defer wg.Done()
			defer func() {
				if recover() != nil {
					results[i] = HealthDependency{
						Name:        healthDepNames[i],
						Criticality: healthCriticality(healthDepNames[i]),
						Status:      healthStatusDown,
						Source:      h.sourceFor(healthDepNames[i]),
						Message:     healthMsgProbeFailed,
					}
				}
			}()
			results[i] = fn(ctx)
		}(i, probes[i])
	}
	wg.Wait()
	probeMS := time.Since(start).Milliseconds()

	deps := make([]HealthDependency, 0, len(results))
	overall := healthOverallOK
	for i := range results {
		dep := results[i]
		// 兜底：防御性地补齐 name/criticality/source，避免任何路径产出空字段。
		if dep.Name == "" {
			dep.Name = healthDepNames[i]
		}
		if dep.Criticality == "" {
			dep.Criticality = healthCriticality(dep.Name)
		}
		if dep.Source == "" {
			dep.Source = h.sourceFor(dep.Name)
		}
		if dep.Status != healthStatusUp && dep.Status != healthStatusDisabled {
			if dep.Criticality == healthCritCritical {
				overall = healthOverallError
			} else if overall != healthOverallError {
				overall = healthOverallDegraded
			}
		}
		deps = append(deps, dep)
	}

	return HealthPayload{
		Status:       overall,
		Version:      h.version,
		Commit:       h.commit,
		BuildTime:    h.buildTime,
		ProbeMS:      probeMS,
		Cached:       false,
		Dependencies: deps,
	}
}

func healthCriticality(name string) string {
	if name == depSQLite {
		return healthCritCritical
	}
	return healthCritSoft
}

func (h *HealthHandler) sourceFor(name string) string {
	switch name {
	case depSQLite:
		return h.sqliteSource
	case depRedis:
		return h.redisSource
	case depOCR:
		return h.ocrTarget.source
	case depASR:
		return h.asrTarget.source
	default:
		return h.ttsTarget.source
	}
}

// probeSQLite 进程内探测，只读（PingContext + SELECT 1）。
// 不做写探测：30s 一次的真写要么产生 WAL 写放大，要么抢写锁在高负载下误报 down。
func (h *HealthHandler) probeSQLite(ctx context.Context) (dep HealthDependency) {
	dep = HealthDependency{
		Name:        depSQLite,
		Criticality: healthCritCritical,
		Status:      healthStatusDown,
		Source:      h.sqliteSource,
	}
	start := time.Now()
	defer func() { dep.LatencyMS = time.Since(start).Milliseconds() }()

	if h.db == nil || h.db.Conn() == nil {
		dep.Message = healthMsgProbeFailed
		return dep
	}

	pctx, cancel := context.WithTimeout(ctx, healthSQLiteTimeout)
	defer cancel()

	if err := h.db.Conn().PingContext(pctx); err != nil {
		dep.Message = classifyProbeErr(err)
		return dep
	}
	var one int
	if err := h.db.Conn().QueryRowContext(pctx, "SELECT 1").Scan(&one); err != nil {
		dep.Message = classifyProbeErr(err)
		return dep
	}

	dep.Status = healthStatusUp
	dep.Message = ""
	return dep
}

// probeRedis 依据「配置开关 + 当前实际 store 后端」判定，见设计判定表。
// 任何分支都不调用 log.Fatal / os.Exit / panic，也不复用 state.New（那会重跑一次降级日志）。
// 降级状态的真相只有一个来源：store.Backend()。
func (h *HealthHandler) probeRedis(ctx context.Context) (dep HealthDependency) {
	dep = HealthDependency{
		Name:        depRedis,
		Criticality: healthCritSoft,
		Status:      healthStatusDown,
		Source:      h.redisSource,
		Backend:     string(state.BackendMemory),
	}

	enabled := h.cfg != nil && h.cfg.Redis.Enabled
	addr := ""
	if h.cfg != nil {
		addr = strings.TrimSpace(h.cfg.Redis.Addr)
	}

	switch {
	case !enabled:
		// 没开 Redis 的部署不该永远 degraded，所以 disabled 不拉低整体状态。
		dep.Status = healthStatusDisabled
		dep.Message = healthMsgRedisDisabled
		return dep
	case addr == "":
		dep.Status = healthStatusDown
		dep.Message = healthMsgRedisNoAddr
		return dep
	case h.store == nil || h.store.Backend() == state.BackendMemory:
		// 启动时 Ping 失败已降级。没有真实 client 可 ping，且 state.New 只在启动时
		// 判断一次 —— 降级后即使 Redis 恢复，进程仍用内存存储直到重启，
		// 因此 down + backend:memory 是准确描述当前进程状态，不是误报。
		dep.Status = healthStatusDown
		dep.Message = healthMsgRedisDegraded
		return dep
	}

	dep.Backend = string(h.store.Backend())

	start := time.Now()
	defer func() { dep.LatencyMS = time.Since(start).Milliseconds() }()

	pctx, cancel := context.WithTimeout(ctx, healthRedisTimeout)
	defer cancel()

	if err := h.store.Ping(pctx); err != nil {
		dep.Message = classifyProbeErr(err)
		return dep
	}

	dep.Status = healthStatusUp
	dep.Message = ""
	return dep
}

func (h *HealthHandler) probeOCR(ctx context.Context) HealthDependency {
	return h.probeHTTP(ctx, depOCR, h.ocrTarget)
}

func (h *HealthHandler) probeASR(ctx context.Context) HealthDependency {
	return h.probeHTTP(ctx, depASR, h.asrTarget)
}

func (h *HealthHandler) probeTTS(ctx context.Context) HealthDependency {
	return h.probeHTTP(ctx, depTTS, h.ttsTarget)
}

// probeHTTP 只看状态码 + 耗时，绝不解析响应体。
// 这样既避免与辅助服务的响应结构耦合，也避免把 ASR /health 原样回显的
// diarize_service_url 这类内网地址捞进我们自己的响应。
func (h *HealthHandler) probeHTTP(ctx context.Context, name string, target depTarget) (dep HealthDependency) {
	dep = HealthDependency{
		Name:        name,
		Criticality: healthCritSoft,
		Status:      healthStatusDown,
		Source:      target.source,
	}

	start := time.Now()
	defer func() { dep.LatencyMS = time.Since(start).Milliseconds() }()

	url := strings.TrimRight(target.base, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		dep.Message = classifyProbeErr(err)
		return dep
	}

	resp, err := h.client.Do(req)
	if err != nil {
		dep.Message = classifyProbeErr(err)
		return dep
	}
	defer resp.Body.Close()
	// 读干净并丢弃，让连接可复用；限制长度避免被异常大的响应体拖住。
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	switch {
	case resp.StatusCode == http.StatusOK:
		dep.Status = healthStatusUp
		dep.Message = ""
	case resp.StatusCode == http.StatusServiceUnavailable:
		// ocr-service / asr-service 在模型未就绪时返回 503 {"detail":"warming up"}。
		dep.Status = healthStatusWarming
		dep.Message = healthMsgWarmingUp
	default:
		dep.Status = healthStatusDown
		dep.Message = fmt.Sprintf("unexpected status %d", resp.StatusCode)
	}
	return dep
}

// classifyProbeErr 把任意底层 error 归一化到 message 闭集白名单。
// 这是本功能最大的泄露面：net/http 的 err.Error() 会同时带 URL、容器服务名、
// 内网 IP、端口。因此 message 字段的唯一赋值路径就是这个函数。
// 返回值只可能是白名单成员。
func classifyProbeErr(err error) string {
	if err == nil {
		return ""
	}
	// 超时（含 ctx deadline）优先，因为 url.Error 同时实现了 Timeout()。
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return healthMsgTimeout
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return healthMsgTimeout
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return healthMsgDNSFailure
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return healthMsgConnectRefused
	}
	// 兜底：不带任何原始错误串。
	return healthMsgProbeFailed
}
