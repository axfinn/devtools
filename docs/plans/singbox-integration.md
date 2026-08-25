# sing-box 集成设计 — devtools 科学上网模块

> **状态**: 设计中,等待批准后开工
> **作者**: claude · 2026-08-19
> **前置修复**: `proxy.go` 已经把 `dialUpstream` 改为显式失败(2026-08-19 P0),本设计文档是该修复之后的下一步

---

## 1. 背景

devtools 后端自研的代理拨号(dialUpstream)只实现了 3 种协议:**http / socks5 / trojan**。其他解析得到的 vmess / vless / ss / ssr / hysteria2 / anytls / tuic / hy2 节点**目前无法用作代理**,只能参与测速。

```go
// proxy.go:3813-3845
switch node.Type {
case "http":    // ✓
case "socks5":  // ✓
case "trojan":  // ✓
default:
    // 显式失败(2026-08-19 修复,原行为是静默直连目标——bug)
    return nil, fmt.Errorf("代理节点类型 %q 暂未实现...", node.Type)
}
```

协议实现工作量极大,VMess 的 AES-128-CFB + 16 字节 IV + HMAC、VLESS 的 XOR 长度前缀 + XTLS-Vision flow、SS 的 AEAD 加密、hy2 的 QUIC + UDP — 每种都是几百行 Go 代码,且加密常量、协议握手顺序、padding 规则容易写错。

**目标**:不重写协议,把 **sing-box 作为外部子进程**集成进 devtools,所有节点解析后转 sing-box outbounds,所有出站代理流量走 sing-box,devtools 自研 dialUpstream 退役。

---

## 2. 选型对比

| 方案 | 协议覆盖 | 维护成本 | 风险 | 评分 |
|---|---|---|---|---|
| **自研全部协议** | 难扩展,每加一种 200+ 行 | 高 | 加密细节容易错 | ✗ |
| **集成 sing-box** | 全协议,自带 routing/fake-ip | 中(配置生成 + IPC) | 多一个外部进程 | ✓✓✓ |
| 集成 Xray-core | 全协议 | 中 | 体积大(40MB+),启动慢 | ✓✓ |
| 集成 mihomo(原 Clash) | 全协议 + GUI 协议 | 中 | 代码复杂(150k 行 Go) | ✓✓ |

**结论**: 选 **sing-box**。原因:
- 单文件二进制,无运行时依赖(Rust 静态编译)
- 体积小(~15MB Linux amd64)
- 启动快(<100ms)
- 协议覆盖全(包括最新的 AnyTLS / TUIC v5)
- 配置是 JSON,Go 端生成简单
- 内置 mixed / socks / http inbound,直接对接 devtools 的 18081 端口

---

## 3. 架构总览

```
┌────────────────────────────────────────────────────────────────────┐
│                       devtools (Go 后端)                           │
│                                                                    │
│   ┌──────────────┐    解析     ┌──────────────┐                    │
│   │ 订阅/ShYdx   │ ─────────► │ ProxyNode[]  │                    │
│   │ /Clash YAML  │            │ (内存结构)   │                    │
│   └──────────────┘            └──────┬───────┘                    │
│                                      │                            │
│   ┌──────────────────┐    生成       │                            │
│   │ singbox.ConfigGen│ ◄─────────────┘                            │
│   │                  │                                             │
│   │  inbounds:       │    写入     ┌──────────────┐               │
│   │   - mixed:18081  │ ─────────► │ /tmp/sb.json │               │
│   │  outbounds:      │            └──────┬───────┘               │
│   │   - 节点1..N     │                   │                       │
│   │  route:          │                   ▼                       │
│   │   - ai/smart/... │           ┌────────────────────┐           │
│   └──────────────────┘           │   sing-box proc    │           │
│                                  │  (subprocess)      │           │
│   ┌──────────────────┐  启动     │                    │           │
│   │ startSingBox()   │ ────────► │  - listen 18081    │           │
│   │   - exec.Command │           │  - dial upstream  │           │
│   │   - log capture  │           │  - relay traffic  │           │
│   └──────────────────┘           └────────────────────┘           │
│                                                                    │
│   退役:                                                              │
│   - dialUpstream() → 删除                                          │
│   - dialSocks5()  → 删除                                            │
│   - dialTrojan()  → 删除                                            │
│   - serveHTTPProxy() 简化为「转给 sing-box 18081 端口」              │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
        ▲                                       │
        │ HTTP CONNECT                          │ 出站流量
        │ (客户端 127.0.0.1:18081)              │ (走 sing-box 加密)
        │                                       ▼
   浏览器 / curl /                  ┌──────────────────┐
   Claude Code /                    │  节点供应商服务器   │
   proxy-client                     │ (Trojan / VLESS / │
                                    │  Hysteria2 ...)   │
                                    └──────────────────┘
```

---

## 4. 关键设计决策

### 4.1 sing-box 二进制分发

| 渠道 | 路径 | 适用 |
|---|---|---|
| Docker 构建时下载 | `docker-compose.yml` build 阶段 `RUN curl -L https://github.com/SagerNet/sing-box/releases/latest/download/sing-box-linux-amd64.tar.gz \| tar xz -C /usr/local/bin` | 生产 |
| macOS dev | `brew install sing-box` 或下载 macOS build 放 `./proxy-client-bins/sing-box-darwin-arm64` | 本地开发 |
| 自动 fallback | 启动时找不到二进制 → 报错提示用户安装,而不是 panic | 通用 |

```go
// handlers/singbox.go
func singboxBin() string {
    candidates := []string{
        "/usr/local/bin/sing-box",
        "/app/data/sing-box",
        "./bin/sing-box",
        os.ExpandEnv("$HOME/.local/bin/sing-box"),
    }
    for _, p := range candidates {
        if _, err := os.Stat(p); err == nil {
            return p
        }
    }
    return ""
}
```

### 4.2 IPC 选型:**stdio JSON-RPC**(不复用 stdio sing-box API,而是配置文件驱动)

sing-box 提供三种控制接口:

| 接口 | 复杂度 | 实时切换节点 | 选 |
|---|---|---|---|
| CLI flag 启动 + 配置文件 | ★ 极简 | ✗ 需重启 | **✓ 选这个** |
| gRPC API(实验性) | ★★★ | ✓ | ✗ 太复杂,devtools 用不到 |
| Clash API(外部 HTTP) | ★★ | ✓ 半实时 | ✗ 需要单独启用 inbound |

**决策**: 用 **配置文件驱动**。每次切换节点,sing-box 重启(单进程启动 <100ms,无感知)。
理由:
- 节点切换不是高频操作(用户手动点),不需要热切换
- 配置文件是 JSON,生成简单,完全可控
- gRPC API 是实验性的,版本兼容风险大
- sing-box 重启非常快,无需考虑热加载

### 4.3 Inbound 设计

devtools 现有 18081 端口是 Go 自实现的 HTTP CONNECT 代理。集成 sing-box 后:

```jsonc
// sing-box 配置 inbounds
{
  "inbounds": [
    {
      "type": "mixed",  // 同时支持 SOCKS5 + HTTP CONNECT
      "listen": "127.0.0.1",
      "listen_port": 18081,
      "users": [{"username": "proxy", "password": "<admin_password>"}],
      "sniff": true,
      "sniff_override_destination": false
    }
  ]
}
```

**重要**: sing-box 的 mixed inbound 自带 HTTP CONNECT 鉴权,devtools 的 `fakeWebPage` 防探测机制就**失效了** — 需要在 devtools 这层补,或单独再起一个中间层。

**简化方案**:**保留 devtools 18081 端口的 Go HTTP CONNECT 监听,内部把所有 CONNECT 请求转给 sing-box 的 18082 SOCKS5 inbound**。

```
客户端 → devtools 18081 (HTTP CONNECT, 防探测 + admin 鉴权)
       → sing-box 18082 (SOCKS5, 不鉴权,只在 127.0.0.1 listen)
       → 出站节点
```

代码量最小: `serveHTTPProxy` 里 `net.Dial("tcp", "127.0.0.1:18082")` 拿到 sing-box 的 SOCKS5 conn,然后跟原来一样把客户端的 CONNECT 字节转过去。

### 4.4 Outbound 配置生成

每个 ProxyNode 转一个 sing-box outbound:

| ProxyNode.Type | sing-box outbound type | 关键字段 |
|---|---|---|
| `trojan` | `trojan` | server/port/password |
| `vless` | `vless` | server/port/uuid/flow/transport |
| `vmess` | `vmess` | server/port/uuid/security/transport |
| `ss` | `shadowsocks` | server/port/method/password |
| `ssr` | ❌ sing-box 不支持 SSR | 标 Status=unsupported,跳过 |
| `hysteria` / `hy2` | `hysteria2` | server/port/password/obfs |
| `anytls` | ❌ sing-box 还不支持(0.5+ 可能加) | 标 Status=unsupported |
| `tuic` | `tuic` | server/port/uuid/password/congestion_control |
| `http` | `http` | server/port/username/password |
| `socks5` | `socks5` | server/port/username/password |
| `unknown:*` | (之前 stub 节点) | 解析 Extra.scheme → 走对应分支 |

```go
// handlers/singbox.go
func nodeToOutbound(n ProxyNode) (map[string]interface{}, error) {
    switch n.Type {
    case "trojan":
        out := map[string]interface{}{
            "type": "trojan",
            "tag":  n.Name,
            "server": n.Server, "server_port": n.Port,
            "password": n.Extra["password"],
        }
        // SNI / skip-cert-verify / network (tcp/ws) 全透传
        return out, nil
    case "vless":
        out := map[string]interface{}{
            "type": "vless", "tag": n.Name,
            "server": n.Server, "server_port": n.Port,
            "uuid": n.Extra["uuid"],
        }
        if flow, ok := n.Extra["flow"].(string); ok && flow != "" {
            out["flow"] = flow
        }
        // transport (ws/grpc/h2) + tls + reality 全透传
        return out, nil
    // ... 其他类型类似
    default:
        return nil, fmt.Errorf("unsupported type: %s", n.Type)
    }
}
```

### 4.5 路由模式复用

devtools 现有 `ai_priority / smart / global` 三种路由模式,继续在前端配置。生成 sing-box route 规则:

```jsonc
{
  "route": {
    "rules": [
      // ai_priority: AI 域名走 AI 专线节点
      {"domain_suffix": ["openai.com", "claude.ai", "anthropic.com", ...],
       "outbound": "<ai_node_name>"},
      // smart: GFW 域名列表走代理,其他直连
      {"rule_set": ["geosite-gfw"], "outbound": "<default_node_name>"},
      // global: 全部走代理
    ],
    "final": "<default_node_name>",  // 默认出口
    "auto_detect_interface": true
  }
}
```

GFW 列表现有逻辑(`gfwlist.go` 24h 拉 + 缓存)保留,生成 sing-box rule-set 文件 `geosite-gfw.srs` 即可(sing-box 内置 `rule-set` 直接吃 geosite 源)。

### 4.6 进程生命周期

```go
// handlers/singbox.go
type SingBoxProc struct {
    cmd       *exec.Cmd
    configPath string
    mu        sync.Mutex
    lastErr   error
}

func (s *SingBoxProc) Start(nodes []ProxyNode, routingMode string, adminPwd string) error {
    cfg := generateConfig(nodes, routingMode, adminPwd)
    if err := os.WriteFile(s.configPath, cfg, 0600); err != nil {
        return err
    }
    s.cmd = exec.Command(singboxBin(), "run", "-c", s.configPath)
    s.cmd.Stdout = log.Writer()
    s.cmd.Stderr = log.Writer()
    return s.cmd.Start()
}

func (s *SingBoxProc) Stop() {
    if s.cmd != nil && s.cmd.Process != nil {
        _ = s.cmd.Process.Signal(syscall.SIGTERM)
        done := make(chan struct{})
        go func() { _ = s.cmd.Wait(); close(done) }()
        select {
        case <-done:
        case <-time.After(3 * time.Second):
            _ = s.cmd.Process.Kill()
        }
    }
}
```

**进程重启触发点**:
- 节点列表变化(LoadConfig 后)
- 默认/AI 节点变化
- 路由模式变化
- 用户手动"切换线路"

每次重启 <100ms。sing-box 进程崩溃 → devtools 检测到 exit,自动重启并 email 告警(复用 `sendAlert`)。

---

## 5. 迁移路径(增量,不停服)

### 阶段 0:基础设施(本设计批准后立即可做,1 天)
- 新建 `handlers/singbox.go`:binary 查找、ConfigGen、进程生命周期
- 加 `singbox.enabled: false` 配置项,默认关闭
- Dockerfile 加 sing-box 二进制下载步骤

### 阶段 1:旁路验证,默认关闭(2 天)
- 加 `/api/proxy/singbox/status` 端点
- 加 UI 开关"启用 sing-box 后端"
- 用户开启后,启动 sing-box 作为**额外**代理端口 18082,跟现有 18081 并存
- 用户可手动切到 18082 验证节点可用性
- 不影响现有 18081(dialUpstream 路径)

### 阶段 2:默认启用,18081 走 sing-box(2 天)
- `singbox.enabled: true` 默认开
- 18081 listener 保留,但内部 `handleProxyConn` 把 CONNECT 转给 127.0.0.1:18082(SOCKS5)
- 删 `dialTrojan` / `dialSocks5` / `dialUpstream` 协议实现,**保留函数签名加 deprecation 注释**
- 全量测试

### 阶段 3:退役 dialUpstream(1 天)
- 删 `dialUpstream` / `dialSocks5` / `dialTrojan` 函数体,改为 `panic("use sing-box")`
- 删除 `proxyNodeDialableType` 黑名单(所有解析节点都可用)
- 把 `parseUnknownProxyURL` 改为转 real outbound

### 阶段 4:扩展协议(可选,1 天)
- sing-box 支持但 devtools 还没识别的协议(WARP / ShadowTLS / WireGuard 等),按需加 `parseXXXURL`

---

## 6. 测试策略

### 6.1 单元测试(覆盖 ConfigGen)

| 测试 | 覆盖 |
|---|---|
| `TestNodeToOutbound_Trojan` | password / sni / skip-cert-verify / network=tcp / ws 全部字段正确 |
| `TestNodeToOutbound_VlessFlow` | `flow=xtls-rprx-vision` 透传 |
| `TestNodeToOutbound_VmessWS` | `network=ws` + `path` + `host` 嵌套 transport 正确 |
| `TestNodeToOutbound_Hysteria2` | password + obfs.type=obfs.salamander 嵌套 |
| `TestNodeToOutbound_AnyTLSUnsupported` | anytls 节点返回 error,不进 config |
| `TestGenerateConfig_RoutingAIPriority` | AI 域名走 ai_node,其他走 default_node |
| `TestGenerateConfig_RoutingSmart` | geosite-gfw 规则集挂上 |

### 6.2 集成测试(需要网络)

- `TestSingBoxEndToEnd_TrojanNode`:启 sing-box → curl -x 通过 → 成功
- `TestSingBoxEndToEnd_NodeSwitch`:启节点 A → 切到节点 B → curl → 流量从 B 出
- `TestSingBoxRestart_CleanShutdown`:发 SIGTERM → 进程在 3s 内退出

### 6.3 兼容性测试

- 现有 `TestParseShadowrocketSubscription` / `TestShadowrocketRoundTrip` 必须继续通过
- 现有 `TestDialUpstream_UnsupportedTypeReturnsError` 必须继续返回错误(直到阶段 3 删掉)

---

## 7. 风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| sing-box 配置文件 JSON schema 升级导致解析失败 | 中 | 高 | ConfigGen 加 schema 版本号,启动 sing-box 前 dry-run 验证 `sing-box check -c config.json` |
| sing-box 二进制下载失败 / 版本错乱 | 低 | 中 | Dockerfile 用 SHA256 校验;本地开发用 brew |
| sing-box 进程崩溃 / 内存泄漏 | 中 | 中 | devtools 检测进程退出,自动重启 + 邮件告警;统计崩溃频率,>3 次/小时告警 |
| sing-box 不支持 SSR | 高 | 低 | 标 Status=unsupported,前端展示"未启用",给出明确提示 |
| sing-box 不支持 anytls | 高(直到 0.5+)| 低 | 同上,等 sing-box release |
| 节点切换 sing-box 重启造成 100ms 延迟 | 低 | 低 | 用户感知不到;后续可加 gRPC 热切换(代价是复杂度↑) |
| sing-box 配置文件含敏感信息(password)| 低 | 中 | 文件权限 0600,`/tmp/sb.json` 启动后清理,或在 `data/sb.json` 并 gitignore |

---

## 8. 验收标准

阶段 2 完成后:
1. 用户提供的 shydx 订阅,16 个 trojan 节点全部能用作代理
2. 任何含 hysteria2 / anytls / tuic / vless-reality 的订阅,这些节点都能用作代理
3. 前端"节点列表"不再有"未启用"标签
4. 启动延迟 <200ms(原 Go 实现 ~10ms,差异可接受)
5. 单元测试覆盖率 >85%,集成测试通过
6. 现有 API(`/api/proxy/start`、`/api/proxy/status`、`/api/proxy/speedtest`)无破坏性变化

---

## 9. 工作量估算

| 阶段 | 工时 |
|---|---|
| 阶段 0: 基础设施 | 1 天 |
| 阶段 1: 旁路验证 | 2 天 |
| 阶段 2: 默认启用 | 2 天 |
| 阶段 3: 退役 dialUpstream | 1 天 |
| 阶段 4: 扩展协议 | 1 天(可选) |
| **合计** | **6-7 天** |

---

## 10. 待用户确认事项

1. ✅/❌ 选 sing-box(默认推荐)
2. ✅/❌ 阶段 1 用"旁路 + 手动切换"做灰度
3. ✅/❌ 阶段 3 删 dialUpstream(还是保留作为 fallback)
4. ✅/❌ sing-box 二进制跟随 devtools docker 镜像发(用户无需自装)
5. ✅/❌ macOS dev 也用真 sing-box 二进制(推荐 Homebrew 安装)

确认后我会:
1. 进入 plan mode 输出详细实现计划
2. 按阶段 0 → 1 → 2 → 3 顺序实施
3. 每个阶段结尾跑完整测试套件 + 给你 review diff
