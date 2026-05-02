---
name: proxy
description: 科学上网线路管理 — 查看节点、切换线路、测速。当用户请求切换代理线路、查看代理状态、管理科学上网节点时使用此技能。
---

# 科学上网线路管理

通过 DevTools 后端 API（`https://t.jaxiu.cn`）管理代理节点。

## 关键常量

- **Base URL**: `https://t.jaxiu.cn`
- **Admin Password**: `123654789`

## 可用操作

### 1. 查看状态和节点列表

```bash
curl -s "https://t.jaxiu.cn/api/proxy/status?admin_password=123654789" | python3 -m json.tool
```

返回字段：
- `running`: 代理是否运行中
- `node`: 当前活动节点名
- `http_port`: 代理端口
- `proxy_url`: 代理地址 (如 `http://127.0.0.1:18081`)
- `nodes[].name`: 所有可用节点名称
- `nodes[].latency`: 节点延迟（ms，-1=不可达）
- `nodes[].type`: 节点类型（ss, ssr, vmess, trojan）
- `default_node_name` / `ai_node_name`: 当前手动选择的默认/AI 线路
- `routing_mode`: 路由模式（ai_priority / smart / global）

### 2. 切换到指定节点

```bash
curl -s -X POST "https://t.jaxiu.cn/api/proxy/start" \
  -H "Content-Type: application/json" \
  -d '{"admin_password":"123654789","node_name":"节点名称"}'
```

切换成功后返回新的端口和节点信息。

### 3. 停止代理

```bash
curl -s -X POST "https://t.jaxiu.cn/api/proxy/stop" \
  -H "Content-Type: application/json" \
  -d '{"admin_password":"123654789"}'
```

### 4. 测速所有节点

```bash
curl -s -X POST "https://t.jaxiu.cn/api/proxy/speedtest" \
  -H "Content-Type: application/json" \
  -d '{"admin_password":"123654789"}' | python3 -m json.tool
```

按延迟排序显示所有节点的 TCP 连接延迟。

### 5. 自动选择最优节点

```bash
curl -s -X POST "https://t.jaxiu.cn/api/proxy/auto-start" \
  -H "Content-Type: application/json" \
  -d '{"admin_password":"123654789"}'
```

自动测速并选择延迟最低的节点启动代理。

## 使用方式 — 设置终端代理

代理启动后，通过环境变量让终端/工具走代理。代理认证格式：`http://proxy:<admin_password>@<host>:<port>`

### 本地直连（本机 devtools 在 Docker）

```bash
# 先通过 /proxy status 确认端口（默认 18081）
export HTTP_PROXY="http://proxy:123654789@127.0.0.1:18081"
export HTTPS_PROXY="http://proxy:123654789@127.0.0.1:18081"
export ALL_PROXY="socks5://127.0.0.1:18081"   # 如果支持 SOCKS
export NO_PROXY="localhost,127.0.0.1,192.168.*,*.local"
```

### 通过 NPS 隧道（远程访问，域名穿透内网）

```bash
export HTTP_PROXY="http://proxy:123654789@nps.jaxiu.cn:18080"
export HTTPS_PROXY="http://proxy:123654789@nps.jaxiu.cn:18080"
export NO_PROXY="localhost,127.0.0.1,192.168.*,*.local"
```

### 常用工具代理设置

```bash
# Git
git config --global http.proxy "http://proxy:123654789@127.0.0.1:18081"
git config --global https.proxy "http://proxy:123654789@127.0.0.1:18081"
git config --global --unset http.proxy   # 取消

# npm / pnpm
npm config set proxy "http://proxy:123654789@127.0.0.1:18081"
npm config delete proxy   # 取消

# Docker (pull 镜像时走代理)
export DOCKER_BUILDKIT=1
docker build --build-arg HTTP_PROXY="$HTTP_PROXY" --build-arg HTTPS_PROXY="$HTTPS_PROXY" .
# 或用 ~/.docker/config.json 配置 proxies 字段

# curl 单次使用
curl -x "http://proxy:123654789@127.0.0.1:18081" https://www.google.com

# 取消代理
unset HTTP_PROXY HTTPS_PROXY ALL_PROXY NO_PROXY
```

### 快捷函数（可选添加到 ~/.zshrc）

```bash
# 开启代理
proxy_on() {
  export HTTP_PROXY="http://proxy:123654789@127.0.0.1:18081"
  export HTTPS_PROXY="http://proxy:123654789@127.0.0.1:18081"
  export ALL_PROXY="http://proxy:123654789@127.0.0.1:18081"
  export NO_PROXY="localhost,127.0.0.1,192.168.*,*.local"
  echo "代理已开启: $HTTP_PROXY"
}

# 关闭代理
proxy_off() {
  unset HTTP_PROXY HTTPS_PROXY ALL_PROXY NO_PROXY
  echo "代理已关闭"
}

# 查看代理状态
proxy_status() {
  curl -s "https://t.jaxiu.cn/api/proxy/status?admin_password=123654789" | python3 -c "
import json,sys
d=json.load(sys.stdin)
print(f'运行: {\"是\" if d.get(\"running\") else \"否\"}  |  节点: {d.get(\"node\",\"无\")}  |  端口: {d.get(\"http_port\",\"无\")}  |  NPS隧道: {\"是\" if d.get(\"npc_running\") else \"否\"}')
if d.get('nodes'):
    print('--- 可用节点 ---')
    for n in sorted(d['nodes'],key=lambda x:x.get('latency',999)):
        lat=f\"{n['latency']}ms\" if n['latency']>0 else '不可达'
        print(f'  {n[\"name\"]:30s} {n[\"type\"]:6s} {lat}')
"
}
```

## 用户交互模式

当用户输入这些命令时，执行对应操作：

| 用户输入 | 操作 |
|---------|------|
| `/proxy` 或 `查看线路` | 执行 status 查询，展示当前线路和可用节点列表 |
| `/proxy 切换 <节点名>` 或 `切换到 <节点名>` | 先查 status 找匹配节点，再调用 start 切换 |
| `/proxy stop` 或 `停止代理` | 调用 stop 停止代理 |
| `/proxy speed` 或 `测速` | 执行 speedtest 并展示排名 |
| `/proxy auto` 或 `自动线路` | 执行 auto-start 自动选最优 |
| `/proxy stop` | 停止代理 |

## 输出格式

根据用户请求，解析 JSON 并友好展示：

- **状态**: 代理是否运行、当前节点、端口
- **节点列表**: 表格展示节点名、类型、延迟（绿色<100ms，黄色100-300ms，红色>300ms/不可达）
- **切换结果**: 是否成功、新节点名、代理地址
