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
| **Markdown** | Markdown 实时预览、语法高亮、导出 HTML/PDF |
| **共享粘贴板** | 创建临时分享，支持过期时间、访问次数限制、密码保护 |
| **Base64** | 文本/图片 Base64 编解码 |
| **URL 编解码** | URL Encode/Decode、URL 解析、参数构建 |
| **时间戳转换** | Unix 时间戳与日期时间互转、时间计算 |
| **正则测试** | 正则表达式实时匹配测试、常用正则模板 |
| **文本转换** | 八进制/Unicode/十六进制转义编解码 |
| **IP/DNS** | 查看当前 IP、域名 DNS 解析（A/AAAA/CNAME/MX/NS/TXT） |

### 粘贴板安全措施

- **IP 限流**：每 IP 每分钟最多创建 5 条
- **内容限制**：单条内容最大 100KB
- **访问限制**：单条最多被访问 1000 次
- **过期清理**：自动清理过期数据
- **密码保护**：可选的访问密码

## 技术栈

- **前端**：Vue 3 + Vite + Element Plus + TailwindCSS
- **后端**：Go Gin + SQLite
- **部署**：Docker + Docker Compose

## 快速开始

### Docker 部署（推荐）

```bash
# 克隆项目
cd devtools

# 构建并启动
docker-compose up -d

# 访问
open http://localhost:8080
```

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

```
devtools/
├── frontend/                 # Vue 3 前端
│   ├── src/
│   │   ├── views/           # 页面组件
│   │   ├── router/          # 路由配置
│   │   └── App.vue          # 主组件
│   └── package.json
│
├── backend/                  # Go Gin 后端
│   ├── handlers/            # API 处理器
│   ├── middleware/          # 中间件（限流等）
│   ├── models/              # 数据模型
│   └── main.go
│
├── Dockerfile               # Docker 构建文件
├── docker-compose.yml       # Docker Compose 配置
└── README.md
```

## 环境变量

| 变量名 | 默认值 | 描述 |
|--------|--------|------|
| PORT | 8080 | 服务端口 |
| DB_PATH | ./data/paste.db | SQLite 数据库路径 |
| TZ | Asia/Shanghai | 时区 |

## API 接口

### 粘贴板

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/paste | 创建分享 |
| GET | /api/paste/:id | 获取分享内容 |
| GET | /api/paste/:id/info | 获取分享信息（不含内容） |
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
