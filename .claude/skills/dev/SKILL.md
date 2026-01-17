---
name: dev
description: DevTools 项目开发技能，遵循 DDD 领域驱动设计。当用户请求开发新功能、添加工具、修改现有功能或询问项目架构时使用此技能。
---

# DevTools 开发技能

当用户请求开发新功能、添加工具或修改现有功能时，使用此技能。

## 领域模型 (Domain Model)

### 核心领域 (Core Domain)

本项目是一个**开发者工具集合平台**，核心价值是提供便捷、安全的在线开发工具。

### 限界上下文 (Bounded Contexts)

```
┌─────────────────────────────────────────────────────────────┐
│                      DevTools Platform                       │
├──────────────┬──────────────┬──────────────┬────────────────┤
│   工具上下文  │  分享上下文   │  聊天上下文   │  短链上下文     │
│  (Tool BC)   │ (Share BC)   │  (Chat BC)   │ (ShortURL BC)  │
├──────────────┼──────────────┼──────────────┼────────────────┤
│ - JsonTool   │ - PasteBin   │ - ChatRoom   │ - ShortURL     │
│ - DiffTool   │ - PasteView  │ - Message    │ - Redirect     │
│ - MarkdownTool│             │ - WebSocket  │ - Stats        │
│ - Base64Tool │              │              │                │
│ - UrlTool    │              │              │                │
│ - TimestampTool│            │              │                │
│ - RegexTool  │              │              │                │
│ - TextTool   │              │              │                │
│ - MermaidTool│              │              │                │
│ - DnsTool    │              │              │                │
└──────────────┴──────────────┴──────────────┴────────────────┘
```

### 聚合根 (Aggregate Roots)

| 上下文 | 聚合根 | 实体 | 值对象 |
|--------|--------|------|--------|
| Share | Paste | - | ID, Content, Password, ExpiresAt |
| Chat | ChatRoom | Message | ID, Nickname, Password |
| ShortURL | ShortURL | - | ID, OriginalURL, Clicks |

## 开发流程 (Development Flow)

### 添加新的纯前端工具

1. **创建 Vue 组件**
   ```
   frontend/src/views/{ToolName}Tool.vue
   ```

2. **注册路由**
   ```javascript
   // frontend/src/router/index.js
   {
     path: '/{tool-name}',
     name: '{ToolName}',
     component: () => import('../views/{ToolName}Tool.vue'),
     meta: { title: '工具名称', icon: 'IconName' }
   }
   ```

3. **组件模板**
   ```vue
   <template>
     <div class="container mx-auto p-4 max-w-4xl">
       <el-card>
         <template #header>
           <span class="text-xl font-bold">工具名称</span>
         </template>
         <!-- 工具内容 -->
       </el-card>
     </div>
   </template>

   <script setup>
   import { ref } from 'vue'
   import { ElMessage } from 'element-plus'

   // 状态和逻辑
   </script>
   ```

### 添加需要后端的工具

遵循 **分层架构**：

```
┌─────────────────────────────────────────┐
│           Presentation Layer            │
│   frontend/src/views/{Tool}Tool.vue     │
├─────────────────────────────────────────┤
│           Application Layer             │
│      backend/handlers/{tool}.go         │
├─────────────────────────────────────────┤
│             Domain Layer                │
│       backend/models/{tool}.go          │
├─────────────────────────────────────────┤
│         Infrastructure Layer            │
│    SQLite (models) + Middleware         │
└─────────────────────────────────────────┘
```

#### 步骤 1: 定义领域模型

```go
// backend/models/{tool}.go
package models

type {Tool} struct {
    ID        string    `json:"id"`
    // ... 领域属性
    CreatedAt time.Time `json:"created_at"`
    CreatorIP string    `json:"creator_ip"`
}

// Init{Tool} 初始化数据库表
func (db *DB) Init{Tool}() error {
    _, err := db.conn.Exec(`
        CREATE TABLE IF NOT EXISTS {tools} (
            id TEXT PRIMARY KEY,
            -- ... 字段定义
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            creator_ip TEXT
        );
    `)
    return err
}

// Create{Tool} 创建
func (db *DB) Create{Tool}(...) (*{Tool}, error) { }

// Get{Tool} 获取
func (db *DB) Get{Tool}(id string) (*{Tool}, error) { }

// Delete{Tool} 删除
func (db *DB) Delete{Tool}(id string) error { }

// CleanExpired{Tool}s 清理过期数据
func (db *DB) CleanExpired{Tool}s() error { }
```

#### 步骤 2: 创建应用层处理器

```go
// backend/handlers/{tool}.go
package handlers

type {Tool}Handler struct {
    db *models.DB
}

func New{Tool}Handler(db *models.DB) *{Tool}Handler {
    return &{Tool}Handler{db: db}
}

type Create{Tool}Request struct {
    // 请求字段 + binding 验证
}

type Create{Tool}Response struct {
    // 响应字段
}

// Create handles POST /api/{tool}
func (h *{Tool}Handler) Create(c *gin.Context) {
    // 1. 绑定请求
    // 2. 验证输入
    // 3. 检查限流
    // 4. 调用领域服务
    // 5. 返回响应
}
```

#### 步骤 3: 注册路由

```go
// backend/main.go

// 初始化数据库表
if err := db.Init{Tool}(); err != nil {
    log.Fatalf("{tool}数据库初始化失败: %v", err)
}

// 定期清理 (在 goroutine 中)
err = db.CleanExpired{Tool}s()

// 创建处理器
{tool}Handler := handlers.New{Tool}Handler(db)

// 注册路由
{tool} := api.Group("/{tool}")
{
    {tool}.POST("", createRateLimiter.Middleware(), {tool}Handler.Create)
    {tool}.GET("/:id", {tool}Handler.Get)
}
```

#### 步骤 4: 创建前端组件

参考"添加新的纯前端工具"部分。

## 代码规范 (Code Standards)

### 后端规范

1. **错误处理**: 使用统一的错误响应格式
   ```go
   c.JSON(http.StatusBadRequest, gin.H{"error": "错误信息"})
   ```

2. **限流**: 创建操作必须使用限流中间件
   ```go
   {tool}.POST("", createRateLimiter.Middleware(), handler.Create)
   ```

3. **安全**:
   - 密码使用 SHA256 哈希
   - 输入验证在 handler 层完成
   - 敏感配置通过 config.yaml（不提交 git）

4. **ID 生成**: 使用 8 字符随机十六进制
   ```go
   bytes := make([]byte, 4)
   rand.Read(bytes)
   id := hex.EncodeToString(bytes)
   ```

### 前端规范

1. **组件结构**: 单文件组件，template-script-style 顺序

2. **状态管理**: 使用 `ref()` 和 `reactive()`

3. **UI 框架**: Element Plus + TailwindCSS
   ```vue
   <el-card>
     <div class="container mx-auto p-4">
   ```

4. **图标**: 使用 @element-plus/icons-vue

5. **消息提示**: 使用 ElMessage
   ```javascript
   ElMessage.success('操作成功')
   ElMessage.error('操作失败')
   ```

## API 设计原则

### RESTful 约定

| 操作 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 创建 | POST | /api/{resource} | 返回 201 Created |
| 获取 | GET | /api/{resource}/:id | 返回 200 OK |
| 列表 | GET | /api/{resource}s | 分页/过滤 |
| 更新 | PUT | /api/{resource}/:id | 返回 200 OK |
| 删除 | DELETE | /api/{resource}/:id | 返回 204 No Content |

### 响应格式

```json
// 成功
{
  "id": "abc123",
  "data": { ... }
}

// 错误
{
  "error": "错误描述信息"
}
```

## 配置管理

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 8080 | 服务端口 |
| DB_PATH | ./data/paste.db | 数据库路径 |
| CONFIG_PATH | ./config.yaml | 配置文件路径 |
| GIN_MODE | release | Gin 模式 |

### 配置文件 (config.yaml)

```yaml
shorturl:
  password: ""  # 短链管理密码
```

## 测试命令

```bash
# 后端
cd backend
go mod tidy
go build .
go run main.go

# 前端
cd frontend
npm install
npm run dev      # 开发
npm run build    # 构建

# Docker
docker-compose up -d
docker-compose logs -f devtools
```

## 检查清单

添加新功能前确认：

- [ ] 确定所属限界上下文
- [ ] 定义聚合根和实体
- [ ] 设计 API 接口
- [ ] 实现领域模型 (models)
- [ ] 实现应用层 (handlers)
- [ ] 注册路由和中间件
- [ ] 实现前端组件
- [ ] 添加限流保护
- [ ] 添加定期清理逻辑
- [ ] 更新 CLAUDE.md 文档
