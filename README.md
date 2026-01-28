# DevTools - 开发者工具网站

[![GitHub](https://img.shields.io/github/stars/axfinn/devtools?style=social)](https://github.com/axfinn/devtools)
[![GitHub forks](https://img.shields.io/github/forks/axfinn/devtools?style=social)](https://github.com/axfinn/devtools/fork)

一个功能丰富的开发者工具网站，包含多种常用的开发辅助工具。

**在线体验**: [https://t.jaxiu.cn](https://t.jaxiu.cn)

**GitHub**: [https://github.com/axfinn/devtools](https://github.com/axfinn/devtools)

## 功能特点

### 工具列表

| 工具 | 功能描述 |
|------|----------|
| **JSON 工具** | JSON 格式化、压缩、校验、转 Go Struct / TypeScript Interface、JSON Path 查询 |
| **Diff 对比** | 文本对比，支持字符级、单词级、行级差异高亮显示 |
| **Markdown** | Markdown 实时预览、语法高亮、导出 HTML/PDF，支持创建分享链接（限制查看次数） |
| **共享粘贴板** | 创建临时分享，支持过期时间、访问次数限制、密码保护 |
| **Base64** | 文本/图片 Base64 编解码 |
| **URL 编解码** | URL Encode/Decode、URL 解析、参数构建 |
| **时间戳转换** | Unix 时间戳与日期时间互转、时间计算 |
| **正则测试** | 正则表达式实时匹配测试、常用正则模板 |
| **文本转换** | 八进制/Unicode/十六进制转义编解码 |
| **IP/DNS** | 查看当前 IP、域名 DNS 解析（A/AAAA/CNAME/MX/NS/TXT） |
| **Mermaid 图表** | Mermaid 图表实时渲染、缩放平移、导出 SVG/PNG |
| **聊天室** | 实时聊天室，支持密码保护、图片/视频上传（WebSocket） |
| **短链生成** | URL 短链服务，支持自定义 ID、访问次数限制、过期时间 |
| **Mock API** | 创建 Mock API 端点，支持自定义响应状态码、响应体、请求日志 |
| **画图工具** | Excalidraw 在线画图，支持云端保存、密码保护、本地存储、导出 PNG/SVG/JSON |

### 安全与性能

- **智能限流**：基于 IP 的速率限制，防止滥用
  - 粘贴板：每 IP 每分钟最多创建 5 条
  - 短链：每 IP 每小时最多创建 10 条
  - Mock API：每 IP 每分钟最多创建 10 个
- **内容限制**：
  - 粘贴板/Markdown：单条最大 100KB
  - 聊天室图片：最大 5MB，视频最大 50MB
  - Excalidraw：最大 10MB（gzip 压缩）
- **访问控制**：
  - 密码保护：粘贴板、聊天室、Markdown 分享、Excalidraw 云端保存
  - 访问次数限制：粘贴板最多 1000 次，Markdown 分享 2-10 次
  - 点击次数限制：短链最多 1000 次
- **自动清理**：
  - 每小时清理过期数据（粘贴板、短链、Mock API、Markdown 分享、Excalidraw 画图）
  - 清理 7 天前的聊天室消息和上传文件
  - 清理 7 天未活跃的聊天室
- **管理功能**：
  - 支持管理员密码管理所有 Markdown 分享和 Excalidraw 画图
  - 创建者密钥管理自己的分享内容

## 技术栈

- **前端**：Vue 3 + Vite + Element Plus + TailwindCSS
- **后端**：Go Gin + SQLite
- **部署**：Docker + Docker Compose

## 快速开始

### Docker 部署（推荐）

```bash
# 克隆项目
cd devtools

# 复制配置文件（可选）
cp backend/config.example.yaml backend/config.yaml
# 编辑 config.yaml 设置管理员密码等配置

# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f devtools

# 访问
open http://localhost:8082
```

Docker 使用 8082 端口，可通过环境变量 `HOST_PORT` 修改。

### 本地开发

#### 前端

```bash
cd frontend
npm install
npm run dev
```

#### 后端

```bash
cd backend
go mod tidy
go run main.go
```

## 项目结构

```text
devtools/
├── frontend/                      # Vue 3 前端
│   ├── src/
│   │   ├── views/                # 工具页面组件
│   │   │   ├── JsonTool.vue      # JSON 工具
│   │   │   ├── DiffTool.vue      # Diff 对比
│   │   │   ├── MarkdownTool.vue  # Markdown 编辑器
│   │   │   ├── MermaidTool.vue   # Mermaid 图表
│   │   │   ├── ChatRoom.vue      # 聊天室
│   │   │   ├── ShortUrl.vue      # 短链生成
│   │   │   ├── MockApi.vue       # Mock API
│   │   │   ├── ExcalidrawTool.vue # 画图工具
│   │   │   └── ...               # 其他工具
│   │   ├── router/               # 路由配置
│   │   └── App.vue               # 主组件（响应式布局）
│   └── package.json
│
├── backend/                       # Go Gin 后端
│   ├── handlers/                 # API 处理器
│   │   ├── paste.go              # 粘贴板
│   │   ├── chat.go               # 聊天室（WebSocket）
│   │   ├── shorturl.go           # 短链服务
│   │   ├── mockapi.go            # Mock API
│   │   ├── mdshare.go            # Markdown 分享
│   │   ├── excalidraw.go         # Excalidraw 画图
│   │   └── dns.go                # IP/DNS 查询
│   ├── middleware/               # 中间件
│   │   └── ratelimit.go          # IP 限流
│   ├── models/                   # 数据模型（SQLite）
│   │   ├── paste.go              # 粘贴板模型
│   │   ├── chat.go               # 聊天室模型
│   │   ├── shorturl.go           # 短链模型
│   │   ├── mockapi.go            # Mock API 模型
│   │   ├── mdshare.go            # Markdown 分享模型
│   │   └── excalidraw.go         # Excalidraw 模型
│   ├── config/                   # 配置管理
│   │   └── config.go             # YAML 配置加载
│   ├── utils/                    # 工具函数
│   │   ├── crypto.go             # 密码哈希（SHA256）
│   │   └── cleanup.go            # 文件清理
│   ├── config.example.yaml       # 配置文件示例
│   ├── go.mod
│   └── main.go                   # 入口文件
│
├── Dockerfile                    # Docker 多阶段构建
├── docker-compose.yml            # Docker Compose 配置
├── CLAUDE.md                     # Claude Code 项目说明
└── README.md                     # 项目文档
```

## 环境变量

| 变量名 | 默认值 | 描述 |
|--------|--------|------|
| PORT | 8080（开发）/ 8082（Docker） | 后端服务端口 |
| HOST_PORT | 8082 | Docker 主机映射端口 |
| DB_PATH | ./data/paste.db | SQLite 数据库路径 |
| CONFIG_PATH | ./config.yaml | 配置文件路径 |
| GIN_MODE | release（生产环境） | Gin 运行模式 |
| TZ | Asia/Shanghai | 时区 |

## 配置文件

复制 `backend/config.example.yaml` 到 `backend/config.yaml` 并根据需要修改：

```yaml
# 短链服务配置
shorturl:
  password: ""  # 设置密码后可使用自定义短链 ID（如 /s/1、/s/abc），留空则只能随机 ID

# Markdown 分享配置
mdshare:
  admin_password: ""        # 管理员密码，用于管理所有分享
  default_max_views: 5      # 默认最大查看次数（2-10）
  default_expires_days: 30  # 默认过期天数

# Excalidraw 画图配置
excalidraw:
  admin_password: ""         # 管理员密码，用于永久保存和管理所有画图
  default_expires_days: 30   # 默认过期天数（1-365）
  max_content_size: 10485760 # 最大内容大小（10MB）
```

**配置说明：**
- **短链密码模式**：
  - 无密码：随机 ID，有速率限制（10/小时/IP）
  - 有密码：可自定义 ID，无速率限制
- **管理员功能**：
  - Markdown 分享：管理员可查看、删除所有分享
  - Excalidraw：管理员可设置永久保存、查看和删除所有画图
- **创建者密钥**：自动存储在浏览器 localStorage，用于管理自己创建的内容

## API 接口

### 粘贴板

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/paste | 创建分享（需 content，可选 title、language、password、expires_in、max_views） |
| GET | /api/paste/:id | 获取分享内容（可选 password 参数） |
| GET | /api/paste/:id/info | 获取分享信息（不含内容） |

### IP/DNS

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/ip | 获取客户端 IP 地址 |
| GET | /api/dns?domain=xxx | DNS 查询（支持 A/AAAA/CNAME/MX/NS/TXT） |

### 聊天室

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/chat/room | 创建聊天室（需 name，可选 password） |
| GET | /api/chat/rooms | 获取最近的聊天室列表 |
| GET | /api/chat/room/:id | 获取聊天室信息 |
| POST | /api/chat/room/:id/join | 加入聊天室（需 nickname，可选 password） |
| GET | /api/chat/room/:id/ws?nickname=xxx | WebSocket 连接（实时聊天） |
| POST | /api/chat/upload | 上传图片/视频（最大 5MB/50MB） |

### 短链服务

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/shorturl | 创建短链（需 original_url，可选 expires_in、max_clicks、custom_id、password） |
| GET | /api/shorturl/list | 获取短链列表 |
| GET | /api/shorturl/:id/stats | 获取短链点击统计 |
| GET | /s/:id | 短链重定向（非 API 路由） |

### Mock API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/mockapi | 创建 Mock API（需 method、response_status、response_body，可选 name、description、expires_in） |
| GET | /api/mockapi/:id | 获取 Mock API 详情 |
| GET | /api/mockapi/:id/logs | 获取 Mock API 请求日志 |
| PUT | /api/mockapi/:id | 更新 Mock API |
| DELETE | /api/mockapi/:id | 删除 Mock API |
| ANY | /mock/:id | 执行 Mock API（记录请求并返回配置的响应） |

### Markdown 分享

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/mdshare | 创建 Markdown 分享（需 content，可选 title、max_views 2-10、expires_in） |
| GET | /api/mdshare/:id?key=xxx | 获取 Markdown 内容（消耗 1 次查看） |
| GET | /api/mdshare/:id/creator?creator_key=xxx | 创建者获取内容（不消耗查看次数） |
| PUT | /api/mdshare/:id | 管理分享（actions: extend、reshare、edit） |
| DELETE | /api/mdshare/:id?creator_key=xxx | 删除分享 |
| GET | /api/mdshare/admin/list?admin_password=xxx | 管理员列出所有分享 |
| GET | /api/mdshare/admin/:id?admin_password=xxx | 管理员查看任意分享 |
| DELETE | /api/mdshare/admin/:id?admin_password=xxx | 管理员删除任意分享 |
| GET | /md/:id?key=xxx | 前端查看路由（自动生成短链 /s/xxx） |

### Excalidraw 画图

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/excalidraw | 创建云端画图（需 content、password，可选 title、expires_in、admin_password） |
| GET | /api/excalidraw/:id?password=xxx | 获取画图内容（需密码） |
| GET | /api/excalidraw/:id/creator?creator_key=xxx | 创建者获取内容 |
| PUT | /api/excalidraw/:id | 管理画图（actions: extend、edit、set_permanent） |
| DELETE | /api/excalidraw/:id?creator_key=xxx | 删除画图 |
| GET | /api/excalidraw/admin/list?admin_password=xxx | 管理员列出所有画图 |
| GET | /api/excalidraw/admin/:id?admin_password=xxx | 管理员查看任意画图 |
| DELETE | /api/excalidraw/admin/:id?admin_password=xxx | 管理员删除任意画图 |
| GET | /draw/:id | 前端查看路由 |

### 其他

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/health | 健康检查 |

## 支持项目

如果这个项目对你有帮助，欢迎请作者喝杯咖啡 ☕

<table>
  <tr>
    <td align="center"><b>支付宝</b></td>
    <td align="center"><b>微信</b></td>
  </tr>
  <tr>
    <td><img src="frontend/public/alipay.jpeg" width="200" alt="支付宝" /></td>
    <td><img src="frontend/public/wxpay.jpeg" width="200" alt="微信支付" /></td>
  </tr>
</table>

## License

MIT
