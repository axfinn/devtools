# DevTools 项目优化方案

## 概述

本文档提供了一个系统化的优化计划，旨在提升 DevTools 项目的代码质量、可维护性、安全性和性能。

**优化时间线**: 建议分阶段实施
**优先级**: 🔴 高优先级 | 🟡 中优先级 | 🟢 低优先级

---

## 一、后端优化

### 1.1 代码架构重构 🔴

#### 问题
- 业务逻辑直接写在 handler 中，违反单一职责原则
- 缺少服务层抽象，代码复用性差
- 数据验证逻辑分散

#### 解决方案
```
backend/
├── handlers/      # HTTP 处理层（只处理 HTTP 请求/响应）
├── services/      # 业务逻辑层（新增）
├── models/        # 数据模型和数据库操作
├── validators/    # 统一的输入验证（新增）
├── middleware/    # 中间件
├── config/        # 配置管理
└── utils/         # 工具函数
```

**实施步骤**:
1. 创建 `services` 包，将业务逻辑从 handlers 迁移到 services
2. 创建 `validators` 包，统一处理输入验证
3. Handler 只负责：解析请求 → 调用 service → 返回响应

**示例**:
```go
// services/paste_service.go
type PasteService struct {
    db *models.DB
}

func (s *PasteService) CreatePaste(req *CreatePasteRequest, ip string) (*Paste, error) {
    // 业务逻辑在这里
}

// handlers/paste.go
func (h *PasteHandler) Create(c *gin.Context) {
    var req CreatePasteRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    paste, err := h.service.CreatePaste(&req, c.ClientIP())
    if err != nil {
        handleError(c, err) // 统一错误处理
        return
    }

    c.JSON(200, paste)
}
```

---

### 1.2 安全性增强 🔴

#### 1.2.1 密码哈希算法升级

**问题**: 当前使用 SHA256，不安全（无盐值、可快速破解）

**解决方案**: 使用 bcrypt 或 argon2

```go
// utils/crypto.go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

#### 1.2.2 CORS 配置优化

**问题**: `AllowOrigins: "*"` 过于宽松

**解决方案**:
```go
// 在配置文件中设置允许的域名
r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://t.jaxiu.cn", "http://localhost:5173"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

#### 1.2.3 输入验证增强

**解决方案**: 使用 `go-playground/validator` 进行统一验证

```go
type CreatePasteRequest struct {
    Content   string `json:"content" validate:"max=102400"` // 100KB
    Title     string `json:"title" validate:"max=200"`
    Language  string `json:"language" validate:"max=50"`
    Password  string `json:"password" validate:"max=100"`
    ExpiresIn int    `json:"expires_in" validate:"min=0,max=168"` // 最多7天
    MaxViews  int    `json:"max_views" validate:"min=1,max=1000"`
}
```

---

### 1.3 日志系统改进 🟡

**问题**: 使用标准 `log`，缺少结构化日志、日志级别、日志轮转

**解决方案**: 使用 `zap` 或 `logrus`

```go
// utils/logger.go
import "go.uber.org/zap"

var Logger *zap.Logger

func InitLogger() {
    var err error
    if gin.Mode() == gin.ReleaseMode {
        Logger, err = zap.NewProduction()
    } else {
        Logger, err = zap.NewDevelopment()
    }
    if err != nil {
        panic(err)
    }
}

// 使用示例
Logger.Info("粘贴板已创建",
    zap.String("id", paste.ID),
    zap.String("ip", ip),
    zap.Int("size", len(paste.Content)))
```

---

### 1.4 错误处理标准化 🟡

**问题**: 错误响应格式不一致，有的返回 `error`，有的返回 `code`

**解决方案**: 定义统一的错误响应和错误处理中间件

```go
// utils/errors.go
type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}

var (
    ErrInvalidInput    = &AppError{400, "无效的输入", ""}
    ErrUnauthorized    = &AppError{401, "未授权", ""}
    ErrNotFound        = &AppError{404, "资源不存在", ""}
    ErrTooManyRequests = &AppError{429, "请求过于频繁", ""}
    ErrInternal        = &AppError{500, "内部服务器错误", ""}
)

// middleware/error_handler.go
func ErrorHandler() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            if appErr, ok := err.(*AppError); ok {
                c.JSON(appErr.Code, appErr)
            } else {
                c.JSON(500, ErrInternal)
            }
        }
    }
}
```

---

### 1.5 配置管理优化 🟡

**问题**: 很多配置硬编码在代码中

**解决方案**: 统一到配置文件中

```yaml
# config.yaml
server:
  port: 8080
  mode: release

database:
  path: ./data/paste.db
  max_connections: 10

limits:
  paste:
    max_content_size: 102400        # 100KB
    max_images: 15
    max_total_size: 31457280        # 30MB
    max_views: 1000
    max_expires_hours: 168          # 7 days
  rate_limit:
    paste_per_minute: 10
    paste_per_hour: 100
    shorturl_per_hour: 10

security:
  cors_origins:
    - https://t.jaxiu.cn
    - http://localhost:5173
  bcrypt_cost: 10

logging:
  level: info
  format: json
  file: ./logs/devtools.log
```

---

### 1.6 数据库优化 🟢

**改进点**:
1. 添加连接池配置
2. 添加更多索引以提升查询性能
3. 使用事务保证数据一致性

```go
func NewDB(dbPath string, maxConns int) (*DB, error) {
    conn, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    conn.SetMaxOpenConns(maxConns)
    conn.SetMaxIdleConns(maxConns / 2)
    conn.SetConnMaxLifetime(time.Hour)

    return &DB{conn: conn}, nil
}
```

---

### 1.7 API 文档生成 🟡

**问题**: 缺少 Swagger/OpenAPI 文档

**解决方案**: 使用 `swaggo/swag` 生成 API 文档

```go
// main.go
import "github.com/swaggo/gin-swagger"
import "github.com/swaggo/files"

// @title DevTools API
// @version 1.0
// @description 开发者工具 API 文档
// @host localhost:8080
// @BasePath /api
func main() {
    r := gin.Default()

    // Swagger 文档
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // ...
}

// handlers/paste.go
// @Summary 创建粘贴板
// @Tags paste
// @Accept json
// @Produce json
// @Param request body CreatePasteRequest true "粘贴板内容"
// @Success 200 {object} PasteResponse
// @Failure 400 {object} AppError
// @Router /paste [post]
func (h *PasteHandler) Create(c *gin.Context) {
    // ...
}
```

---

### 1.8 添加单元测试 🔴

**问题**: 没有测试覆盖

**解决方案**: 为每个包添加测试文件

```go
// models/paste_test.go
func TestCreatePaste(t *testing.T) {
    db, err := NewDB(":memory:")
    require.NoError(t, err)
    defer db.Close()

    paste := &Paste{
        Content:  "test content",
        Title:    "Test",
        Language: "text",
    }

    err = db.CreatePaste(paste)
    assert.NoError(t, err)
    assert.NotEmpty(t, paste.ID)

    // 验证可以读取
    retrieved, err := db.GetPaste(paste.ID)
    assert.NoError(t, err)
    assert.Equal(t, paste.Content, retrieved.Content)
}
```

**测试覆盖目标**: 至少 60% 代码覆盖率

---

## 二、前端优化

### 2.1 引入 TypeScript 🟡

**收益**: 类型安全、更好的 IDE 支持、减少运行时错误

**实施步骤**:
1. 安装 TypeScript: `npm install -D typescript @vue/typescript`
2. 创建 `tsconfig.json`
3. 将 `.vue` 文件中的 `<script setup>` 改为 `<script setup lang="ts">`
4. 为 API 响应定义类型接口

```typescript
// types/api.ts
export interface Paste {
  id: string
  content: string
  title: string
  language: string
  expires_at: string
  max_views: number
  views: number
  created_at: string
  has_password: boolean
}

export interface CreatePasteRequest {
  content: string
  title?: string
  language?: string
  password?: string
  expires_in?: number
  max_views?: number
}
```

---

### 2.2 统一 API 调用 🟡

**问题**: API 调用分散在各个组件中

**解决方案**: 统一到 `api/` 目录

```typescript
// api/paste.ts
import axios from './axios'

export const pasteAPI = {
  create(data: CreatePasteRequest): Promise<Paste> {
    return axios.post('/api/paste', data)
  },

  get(id: string, password?: string): Promise<Paste> {
    return axios.get(`/api/paste/${id}`, { params: { password } })
  },

  getInfo(id: string): Promise<PasteInfo> {
    return axios.get(`/api/paste/${id}/info`)
  }
}
```

---

### 2.3 状态管理优化 🟢

**建议**: 引入 Pinia 进行状态管理（可选）

```typescript
// stores/user.ts
import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    creatorKeys: [] as string[],
    theme: 'light'
  }),

  actions: {
    addCreatorKey(key: string) {
      this.creatorKeys.push(key)
      localStorage.setItem('creator_keys', JSON.stringify(this.creatorKeys))
    }
  }
})
```

---

### 2.4 错误处理统一 🟡

**解决方案**: 创建 Axios 拦截器统一处理错误

```typescript
// api/axios.ts
import axios from 'axios'
import { ElMessage } from 'element-plus'

const instance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 30000
})

instance.interceptors.response.use(
  response => response.data,
  error => {
    const message = error.response?.data?.error || error.message || '请求失败'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default instance
```

---

### 2.5 组件优化 🟢

**改进点**:
1. 提取可复用的组件（如 CodeEditor、MarkdownPreview）
2. 使用 `defineProps` 和 `defineEmits` 定义组件接口
3. 添加 PropTypes 验证

```vue
<!-- components/CodeEditor.vue -->
<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  modelValue: string
  language?: string
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  language: 'text',
  readonly: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>
```

---

### 2.6 添加组件测试 🟢

**使用**: Vitest + Vue Test Utils

```typescript
// views/JsonTool.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import JsonTool from './JsonTool.vue'

describe('JsonTool', () => {
  it('格式化 JSON', async () => {
    const wrapper = mount(JsonTool)
    const input = wrapper.find('textarea')

    await input.setValue('{"name":"test"}')
    await wrapper.find('button[aria-label="格式化"]').trigger('click')

    expect(wrapper.text()).toContain('"name": "test"')
  })
})
```

---

## 三、文档优化

### 3.1 API 文档 🟡

- 使用 Swagger UI 自动生成交互式 API 文档
- 部署到 `/swagger/index.html`

### 3.2 代码文档 🟡

**Go 代码**:
- 为所有导出的函数、类型添加注释
- 使用 `godoc` 生成文档

```go
// PasteService 提供粘贴板相关的业务逻辑
type PasteService struct {
    db *models.DB
}

// CreatePaste 创建一个新的粘贴板
//
// 参数:
//   - req: 创建请求，包含内容、标题等信息
//   - ip: 创建者的 IP 地址，用于限流
//
// 返回:
//   - paste: 创建成功的粘贴板对象
//   - error: 错误信息（如果有）
func (s *PasteService) CreatePaste(req *CreatePasteRequest, ip string) (*Paste, error) {
    // ...
}
```

### 3.3 部署文档 🟢

创建 `docs/DEPLOYMENT.md`:
- 详细的部署步骤
- 常见问题排查
- 性能调优建议
- 监控和日志查看

### 3.4 贡献指南 🟢

创建 `CONTRIBUTING.md`:
- 如何提交 Issue
- 如何提交 PR
- 代码规范
- 提交信息规范

---

## 四、DevOps 优化

### 4.1 CI/CD 增强 🟡

**当前状态**: 已有基础的 GitHub Actions 配置

**建议增强**:
1. 添加自动化测试步骤
2. 添加代码覆盖率报告
3. 添加代码质量检查（golangci-lint）
4. 添加安全扫描（Trivy）

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: |
          cd backend
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./backend/coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          working-directory: backend
```

---

### 4.2 监控和指标 🟢

**建议**: 添加 Prometheus 指标

```go
// middleware/metrics.go
import "github.com/prometheus/client_golang/prometheus"

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "endpoint"},
    )
)

func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start).Seconds()
        status := fmt.Sprintf("%d", c.Writer.Status())

        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
    }
}
```

---

### 4.3 健康检查增强 🟢

**当前**: 简单的 `/api/health` 端点

**增强**: 添加依赖检查

```go
type HealthResponse struct {
    Status   string            `json:"status"`
    Services map[string]string `json:"services"`
    Version  string            `json:"version"`
    Uptime   string            `json:"uptime"`
}

func (h *HealthHandler) Check(c *gin.Context) {
    services := make(map[string]string)

    // 检查数据库
    if err := h.db.Ping(); err != nil {
        services["database"] = "unhealthy: " + err.Error()
    } else {
        services["database"] = "healthy"
    }

    status := "healthy"
    for _, s := range services {
        if strings.HasPrefix(s, "unhealthy") {
            status = "unhealthy"
            break
        }
    }

    c.JSON(200, HealthResponse{
        Status:   status,
        Services: services,
        Version:  version.Version,
        Uptime:   time.Since(startTime).String(),
    })
}
```

---

## 五、实施优先级

### 第一阶段（1-2周）🔴 高优先级
1. 安全性增强（密码哈希、CORS）
2. 错误处理标准化
3. 添加后端单元测试（核心功能）
4. 代码架构重构（创建 services 层）

### 第二阶段（2-3周）🟡 中优先级
1. 日志系统改进
2. 配置管理优化
3. API 文档生成
4. 前端 API 调用统一
5. 前端错误处理统一

### 第三阶段（长期）🟢 低优先级
1. 引入 TypeScript
2. 添加前端测试
3. 数据库优化
4. 监控和指标
5. 完善文档

---

## 六、代码规范

### 6.1 Go 代码规范

遵循官方 [Effective Go](https://golang.org/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

**关键点**:
- 导出的函数、类型必须有文档注释
- 错误优先返回
- 使用 `gofmt` 格式化代码
- 避免裸返回
- 接口命名以 `er` 结尾（如 `Handler`, `Service`）

### 6.2 Vue 代码规范

遵循 [Vue.js 风格指南](https://vuejs.org/style-guide/)

**关键点**:
- 组件名使用 PascalCase
- Prop 名使用 camelCase
- 事件名使用 kebab-case
- 使用 `<script setup>` 语法
- 组件文件一个文件一个组件

### 6.3 Git 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type**:
- `feat`: 新功能
- `fix`: 修复 bug
- `refactor`: 重构
- `docs`: 文档更新
- `test`: 测试相关
- `chore`: 构建/工具配置

**示例**:
```
feat(paste): 添加图片上传功能

- 支持最多15张图片
- 总大小限制30MB
- 使用base64编码存储

Closes #123
```

---

## 七、性能优化建议

### 7.1 后端性能

1. **数据库查询优化**
   - 添加必要的索引
   - 使用预编译语句
   - 避免 N+1 查询

2. **缓存策略**
   - 使用 Redis 缓存热点数据
   - 设置合理的过期时间

3. **并发控制**
   - 使用 goroutine pool 限制并发
   - 合理设置数据库连接池

### 7.2 前端性能

1. **代码分割**
   - 使用动态导入 `() => import('./component.vue')`
   - 按路由分割代码

2. **资源优化**
   - 图片懒加载
   - 使用 WebP 格式
   - 压缩静态资源

3. **渲染优化**
   - 使用 `v-show` 替代频繁切换的 `v-if`
   - 合理使用 `keep-alive`
   - 虚拟滚动处理长列表

---

## 八、总结

本优化方案涵盖了代码质量、架构设计、安全性、性能、文档和 DevOps 等多个方面。建议按照优先级分阶段实施，每个阶段完成后进行验收和回顾。

**预期收益**:
- 代码质量提升 40%
- 测试覆盖率达到 60%+
- 安全性显著提升
- 可维护性提升 50%
- 开发效率提升 30%

**需要的资源**:
- 开发时间: 约 4-6 周（按阶段划分）
- 技术栈学习: bcrypt, zap, swaggo, Vitest 等

如有任何问题或需要更详细的实施指导，请参考本文档或联系项目维护者。
