# DevTools 开发者文档

本文档面向 DevTools 项目的开发者，提供详细的开发指南、架构说明和最佳实践。

## 目录

- [开发环境设置](#开发环境设置)
- [项目架构](#项目架构)
- [开发规范](#开发规范)
- [测试指南](#测试指南)
- [调试技巧](#调试技巧)
- [常见问题](#常见问题)

---

## 开发环境设置

### 必备工具

- **Go**: 1.21+ (后端开发)
- **Node.js**: 18+ (前端开发)
- **Docker**: 20+ (容器化部署)
- **Git**: 2.30+ (版本控制)

### 初始化开发环境

```bash
# 1. 克隆项目
git clone https://github.com/your-org/devtools.git
cd devtools

# 2. 后端设置
cd backend
cp config.example.yaml config.yaml  # 复制并编辑配置
go mod download                      # 下载依赖
go run main.go                       # 启动后端

# 3. 前端设置（新终端）
cd frontend
npm install                          # 安装依赖
npm run dev                          # 启动开发服务器

# 4. 访问应用
# 前端: http://localhost:5173
# 后端: http://localhost:8080
```

### IDE 推荐配置

#### VS Code

推荐扩展:
- Go (Go Team at Google)
- Vue Language Features (Volar)
- ESLint
- Prettier
- GitLens

#### GoLand / WebStorm

自动识别项目结构，开箱即用。

---

## 项目架构

### 整体架构

```
DevTools
├── 前端层 (Vue 3)          用户界面
│   └── HTTP Requests
│
├── API 网关 (Gin)          路由、中间件、限流
│   └── Handlers
│
├── 业务逻辑层 (Services)   核心业务逻辑（规划中）
│   └── Models
│
└── 数据层 (SQLite)         数据持久化
```

### 后端架构

#### 分层设计

```
backend/
├── main.go                # 入口：初始化服务器、路由、中间件
├── config/                # 配置管理
│   └── config.go          # 配置加载、默认值、环境变量
├── middleware/            # 中间件
│   ├── ratelimit.go       # IP 限流
│   └── error_handler.go   # 全局错误处理
├── handlers/              # HTTP 处理器（Controller 层）
│   ├── paste.go           # 粘贴板 API
│   ├── chat.go            # 聊天室 API + WebSocket
│   ├── shorturl.go        # 短链 API
│   ├── mockapi.go         # Mock API
│   ├── mdshare.go         # Markdown 分享 API
│   ├── excalidraw.go      # Excalidraw 画图 API
│   └── dns.go             # DNS 查询 API
├── models/                # 数据模型（Model 层）
│   ├── paste.go           # 粘贴板数据操作
│   ├── chat.go            # 聊天室数据操作
│   ├── shorturl.go        # 短链数据操作
│   ├── mockapi.go         # Mock API 数据操作
│   ├── mdshare.go         # Markdown 分享数据操作
│   └── excalidraw.go      # Excalidraw 数据操作
└── utils/                 # 工具函数
    ├── crypto.go          # 密码哈希（bcrypt）
    ├── cleanup.go         # 文件清理
    └── errors.go          # 错误定义和处理
```

#### 关键设计模式

1. **Repository Pattern** - models 包充当数据访问层
2. **Middleware Pattern** - 限流、CORS、错误处理
3. **Singleton Pattern** - 全局配置、数据库连接

#### 数据库设计

使用 SQLite，包含以下表：

| 表名 | 用途 | 关键字段 |
|------|------|----------|
| `pastes` | 粘贴板内容 | id, content, password, expires_at, views |
| `chat_rooms` | 聊天室 | id, name, password, created_at, last_active |
| `chat_messages` | 聊天消息 | id, room_id, nickname, content, created_at |
| `short_urls` | 短链 | id, original_url, clicks, expires_at |
| `mock_apis` | Mock API | id, method, response_status, response_body |
| `markdown_shares` | Markdown 分享 | id, content, creator_key, access_key, views |
| `excalidraw_shares` | Excalidraw 画图 | id, content (gzip), creator_key, access_key |

### 前端架构

```
frontend/
├── src/
│   ├── main.js            # 入口文件
│   ├── App.vue            # 根组件（响应式布局）
│   ├── router/            # 路由配置
│   │   └── index.js       # Vue Router 配置
│   ├── views/             # 页面组件
│   │   ├── JsonTool.vue   # JSON 工具
│   │   ├── DiffTool.vue   # Diff 对比
│   │   ├── MarkdownTool.vue # Markdown 编辑器
│   │   ├── ChatRoom.vue   # 聊天室
│   │   ├── ShortUrl.vue   # 短链生成
│   │   ├── MockApi.vue    # Mock API 管理
│   │   └── ExcalidrawTool.vue # 画图工具
│   ├── components/        # 可复用组件
│   │   └── ExcalidrawWrapper.vue # Excalidraw 封装
│   ├── composables/       # 组合式函数
│   ├── api.js             # API 调用封装
│   ├── styles/            # 样式文件
│   └── assets/            # 静态资源
└── public/                # 公共静态文件
```

#### 技术栈

- **Vue 3**: Composition API
- **Vue Router**: 客户端路由
- **Element Plus**: UI 组件库
- **TailwindCSS**: 实用优先的 CSS 框架
- **Monaco Editor**: 代码编辑器
- **Mermaid**: 图表渲染
- **Excalidraw**: 手绘风格画图

---

## 开发规范

### Go 代码规范

遵循 [Effective Go](https://golang.org/doc/effective_go)。

#### 命名规范

```go
// ✅ 好的命名
type PasteHandler struct {
    db *models.DB
}

func (h *PasteHandler) Create(c *gin.Context) { }

// ❌ 不好的命名
type paste_handler struct { }
func (h *paste_handler) create_paste(c *gin.Context) { }
```

#### 错误处理

```go
// ✅ 使用统一的错误类型
if err != nil {
    utils.RespondError(c, utils.ErrDatabaseError)
    return
}

// ❌ 不统一的错误响应
if err != nil {
    c.JSON(500, gin.H{"error": "failed"})
    return
}
```

#### 文档注释

```go
// CreatePaste 创建一个新的粘贴板
//
// 参数:
//   - c: Gin 上下文，包含 HTTP 请求和响应
//
// 响应:
//   - 200: 创建成功，返回粘贴板 ID
//   - 400: 请求参数错误
//   - 429: 请求过于频繁
//   - 500: 服务器内部错误
func (h *PasteHandler) Create(c *gin.Context) {
    // ...
}
```

### Vue 代码规范

遵循 [Vue Style Guide](https://vuejs.org/style-guide/)。

#### 组件命名

```vue
<!-- ✅ PascalCase for component files -->
<!-- JsonTool.vue -->
<template>
  <div class="json-tool">
    <!-- ... -->
  </div>
</template>

<script setup>
// Composition API
import { ref, computed } from 'vue'
</script>
```

#### Props 和 Emits

```vue
<script setup>
// ✅ 明确定义 props 和 emits
const props = defineProps({
  modelValue: {
    type: String,
    required: true
  },
  language: {
    type: String,
    default: 'text'
  }
})

const emit = defineEmits(['update:modelValue', 'save'])
</script>
```

### Git 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type 类型**:
- `feat`: 新功能
- `fix`: Bug 修复
- `refactor`: 重构（不改变功能）
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `test`: 测试相关
- `chore`: 构建工具、依赖更新

**示例**:
```
feat(paste): 支持图片粘贴功能

- 添加图片上传接口
- 前端支持拖拽上传
- 限制图片大小和数量

Closes #123
```

---

## 测试指南

### 后端测试

#### 单元测试

```go
// models/paste_test.go
package models

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreatePaste(t *testing.T) {
    // 使用内存数据库测试
    db, err := NewDB(":memory:")
    assert.NoError(t, err)
    defer db.Close()

    paste := &Paste{
        Content:  "test content",
        Title:    "Test",
        Language: "text",
    }

    err = db.CreatePaste(paste)
    assert.NoError(t, err)
    assert.NotEmpty(t, paste.ID)
}
```

#### 运行测试

```bash
cd backend

# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 前端测试

#### 组件测试（使用 Vitest）

```javascript
// views/JsonTool.spec.js
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import JsonTool from './JsonTool.vue'

describe('JsonTool', () => {
  it('格式化 JSON', async () => {
    const wrapper = mount(JsonTool)
    const textarea = wrapper.find('textarea')

    await textarea.setValue('{"name":"test"}')
    await wrapper.find('[data-test="format-btn"]').trigger('click')

    expect(wrapper.text()).toContain('"name": "test"')
  })
})
```

#### 运行测试

```bash
cd frontend

# 运行测试
npm run test

# 运行测试并监听文件变化
npm run test:watch

# 生成覆盖率报告
npm run test:coverage
```

---

## 调试技巧

### 后端调试

#### 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试会话
cd backend
dlv debug main.go
```

#### 日志调试

```go
import "log"

// 调试时添加日志
log.Printf("Debug: paste ID = %s, views = %d", paste.ID, paste.Views)
```

### 前端调试

#### Vue Devtools

安装 [Vue Devtools](https://devtools.vuejs.org/)，可以查看组件树、状态、事件等。

#### 浏览器断点

在浏览器开发者工具中设置断点，或在代码中添加 `debugger`：

```javascript
// 在这里暂停执行
debugger

const result = somethingComplex()
console.log(result)
```

---

## 常见问题

### 1. 数据库锁定错误

**问题**: `database is locked`

**原因**: SQLite 不支持高并发写入

**解决**:
```go
// 在 NewDB 中设置连接池参数
db.SetMaxOpenConns(1)  // SQLite 推荐单连接
db.SetMaxIdleConns(1)
```

### 2. CORS 错误

**问题**: 前端请求被 CORS 策略阻止

**解决**: 检查 `config.yaml` 中的 `cors_origins` 配置

```yaml
security:
  cors_origins:
    - "http://localhost:5173"  # 开发环境
```

### 3. 前端构建失败

**问题**: `Module not found` 或 `Cannot resolve`

**解决**:
```bash
# 清理并重新安装依赖
rm -rf node_modules package-lock.json
npm install
```

### 4. Go 模块依赖问题

**问题**: `cannot find module`

**解决**:
```bash
cd backend
go mod tidy      # 清理并更新依赖
go mod download  # 重新下载依赖
```

---

## 性能优化

### 后端性能

1. **数据库索引**: 为常查询字段添加索引
   ```sql
   CREATE INDEX IF NOT EXISTS idx_expires_at ON pastes(expires_at);
   CREATE INDEX IF NOT EXISTS idx_creator_ip ON pastes(creator_ip);
   ```

2. **限流**: 使用 `middleware.NewRateLimiter()` 防止滥用

3. **连接池**: 合理设置数据库连接池大小

### 前端性能

1. **代码分割**: 路由懒加载
   ```javascript
   {
     path: '/json',
     component: () => import('./views/JsonTool.vue')
   }
   ```

2. **资源优化**: 压缩图片，使用 WebP 格式

3. **缓存**: 使用 `keep-alive` 缓存组件

---

## 部署

### Docker 部署

```bash
# 构建镜像
docker-compose build

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f devtools

# 停止服务
docker-compose down
```

### 手动部署

```bash
# 1. 构建前端
cd frontend
npm run build

# 2. 复制前端构建产物到后端
cp -r dist ../backend/

# 3. 构建后端
cd ../backend
go build -o devtools main.go

# 4. 运行
./devtools
```

---

## 贡献流程

1. Fork 项目
2. 创建功能分支: `git checkout -b feat/amazing-feature`
3. 提交更改: `git commit -m 'feat: add amazing feature'`
4. 推送分支: `git push origin feat/amazing-feature`
5. 创建 Pull Request

详见 [CONTRIBUTING.md](./CONTRIBUTING.md)

---

## 资源链接

- [项目 README](./README.md)
- [优化方案](./docs/OPTIMIZATION_PLAN.md)
- [Claude Code 指南](./CLAUDE.md)
- [Gin 文档](https://gin-gonic.com/)
- [Vue 3 文档](https://vuejs.org/)
- [Element Plus 文档](https://element-plus.org/)

---

## 联系方式

- GitHub Issues: [https://github.com/your-org/devtools/issues](https://github.com/your-org/devtools/issues)
- 讨论区: [https://github.com/your-org/devtools/discussions](https://github.com/your-org/devtools/discussions)
