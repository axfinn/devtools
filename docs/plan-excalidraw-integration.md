# DevTools Excalidraw 集成实现计划

## 1. 需求概述

在 DevTools 项目中集成 Excalidraw 画图功能，支持：
- 导出（PNG、SVG、JSON）
- 本地保存（localStorage）
- 云端保存（密码保护，可设过期时间）
- 导入（本地文件、云端图）
- 素材库接入
- 延期即将过期的图
- 查看和管理已保存的图

## 2. 技术方案

### 2.1 Vue-React 集成

Excalidraw 是 React 组件，需要在 Vue 中集成。采用 **Vue 包装组件** 方案：

```bash
# 前端依赖
npm install react react-dom @excalidraw/excalidraw
```

创建 `ExcalidrawWrapper.vue` 组件：
1. 在 mounted 时创建 React root
2. 渲染 Excalidraw React 组件
3. 暴露 API 方法给 Vue（导出、获取/加载场景）
4. 在 unmounted 时清理 React root

### 2.2 数据格式

Excalidraw 使用 JSON 格式：
```json
{
  "type": "excalidraw",
  "version": 2,
  "elements": [...],
  "appState": {...},
  "files": {...}
}
```

考虑到内容可能较大（含图片），后端存储时使用 gzip 压缩。

## 3. 数据库设计

### 3.1 新表：`excalidraw_shares`

```sql
CREATE TABLE IF NOT EXISTS excalidraw_shares (
    id TEXT PRIMARY KEY,                      -- 8字符 hex ID
    content BLOB NOT NULL,                    -- Excalidraw JSON (gzip压缩)
    title TEXT DEFAULT '',                    -- 标题
    creator_key TEXT NOT NULL,                -- 创建者密钥 (SHA256)
    access_key TEXT NOT NULL,                 -- 访问密码 (SHA256)
    expires_at DATETIME,                      -- 过期时间 (NULL=永久)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    creator_ip TEXT,                          -- 创建者IP
    short_code TEXT DEFAULT '',               -- 短链码
    is_permanent BOOLEAN DEFAULT 0            -- 永久保存标记
);

CREATE INDEX IF NOT EXISTS idx_excalidraw_expires_at ON excalidraw_shares(expires_at);
CREATE INDEX IF NOT EXISTS idx_excalidraw_creator_ip ON excalidraw_shares(creator_ip);
CREATE INDEX IF NOT EXISTS idx_excalidraw_short_code ON excalidraw_shares(short_code);
```

### 3.2 与 MDShare 的区别

| 特性 | MDShare | Excalidraw |
|------|---------|------------|
| 查看限制 | max_views (2-10次) | 无限制 |
| 密码 | 可选 | 必需 |
| 永久保存 | 无 | 管理员可设置 |
| 内容存储 | TEXT | BLOB (gzip) |
| 最大大小 | 2MB | 10MB |

## 4. API 设计

### 4.1 端点列表

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/excalidraw` | POST | 创建云端保存 |
| `/api/excalidraw/:id` | GET | 获取内容（需密码） |
| `/api/excalidraw/:id/creator` | GET | 创建者获取（无需密码） |
| `/api/excalidraw/:id` | PUT | 更新（延期、编辑） |
| `/api/excalidraw/:id` | DELETE | 删除 |
| `/api/excalidraw/admin/list` | GET | 管理员列表 |
| `/api/excalidraw/admin/:id` | GET | 管理员查看 |
| `/api/excalidraw/admin/:id` | DELETE | 管理员删除 |
| `/draw/:id` | GET | 前端查看路由 |

### 4.2 请求/响应示例

**创建请求：**
```json
{
  "content": "...",           // Excalidraw JSON (必需)
  "title": "我的画图",         // 可选
  "password": "xxx",          // 访问密码 (必需)
  "expires_in": 30,           // 过期天数 (1-365, 默认30)
  "admin_password": "..."     // 管理员密码 (永久保存时需要)
}
```

**创建响应：**
```json
{
  "id": "abc12345",
  "creator_key": "xxx",
  "short_code": "zzz",
  "share_url": "/s/zzz",
  "expires_at": "2024-02-28T00:00:00Z",
  "is_permanent": false
}
```

**更新请求（延期）：**
```json
{
  "action": "extend",
  "expires_in": 30,
  "creator_key": "xxx"
}
```

**更新请求（编辑）：**
```json
{
  "action": "edit",
  "content": "...",
  "title": "新标题",
  "creator_key": "xxx"
}
```

### 4.3 限流策略

- 创建：10次/小时/IP
- 内容大小：10MB（可配置）

## 5. 配置设计

### 5.1 config.go 新增

```go
type ExcalidrawConfig struct {
    AdminPassword      string `yaml:"admin_password"`
    DefaultExpiresDays int    `yaml:"default_expires_days"`
    MaxContentSize     int    `yaml:"max_content_size"`
}
```

### 5.2 config.yaml 示例

```yaml
excalidraw:
  admin_password: ""           # 管理员密码，可永久保存
  default_expires_days: 30     # 默认过期天数
  max_content_size: 10485760   # 最大10MB
```

## 6. 前端组件设计

### 6.1 文件结构

```
frontend/src/
├── components/
│   └── ExcalidrawWrapper.vue    # React 包装组件
├── views/
│   ├── ExcalidrawTool.vue       # 主编辑器页面
│   └── ExcalidrawShareView.vue  # 查看分享页面
```

### 6.2 ExcalidrawWrapper.vue

核心功能：
- `exportToSvg()` - 导出 SVG
- `exportToPng()` - 导出 PNG
- `exportToJson()` - 导出 JSON
- `getSceneData()` - 获取当前场景
- `loadSceneData(data)` - 加载场景
- `resetScene()` - 重置画布

### 6.3 ExcalidrawTool.vue

界面布局：
```
┌─────────────────────────────────────────────┐
│ 画图工具                    [导出▼] [保存▼] │
├─────────────────────────────────────────────┤
│                                             │
│           Excalidraw 编辑器                  │
│                                             │
└─────────────────────────────────────────────┘
```

功能模块：
1. **导出菜单**
   - 导出 PNG
   - 导出 SVG
   - 导出 JSON

2. **保存菜单**
   - 保存到本地（localStorage）
   - 保存到云端（打开对话框）

3. **云端保存对话框**
   - 标题输入
   - 访问密码（必填）
   - 过期时间选择（1天-1年，或永久）
   - 永久保存需输入管理员密码

4. **我的画图对话框**
   - 本地画图列表（可加载、删除）
   - 云端画图列表（可加载、延期、删除）
   - 搜索过滤

5. **管理员面板**（管理员登录后显示）
   - 所有云端画图列表
   - 查看、删除任意画图
   - 设置永久保存

6. **导入功能**
   - 从本地文件导入 (.excalidraw, .json)
   - 从云端链接导入

### 6.4 ExcalidrawShareView.vue

查看分享页面：
```
┌─────────────────────────────────────────────┐
│ 画图分享 - 标题               [下载▼] [复制链接] │
├─────────────────────────────────────────────┤
│  ┌─────────────────────────────────────┐    │
│  │      请输入访问密码                    │    │
│  │  [密码输入框]  [查看]                  │    │
│  └─────────────────────────────────────┘    │
│                                             │
│           (验证后显示 Excalidraw 只读视图)     │
│                                             │
├─────────────────────────────────────────────┤
│ ⏰ 此分享将于 2024-02-28 过期                 │
└─────────────────────────────────────────────┘
```

### 6.5 localStorage 结构

```javascript
// 本地画图
localStorage['excalidraw_local_drawings'] = JSON.stringify({
  "local_1": {
    title: "未命名",
    content: "...", // Excalidraw JSON
    updated_at: "2024-01-28T00:00:00Z"
  }
})

// 云端画图的创建者密钥
localStorage['excalidraw_creator_keys'] = JSON.stringify({
  "abc12345": {
    key: "xxx",
    title: "我的画图",
    expires_at: "2024-02-28T00:00:00Z",
    is_permanent: false
  }
})
```

## 7. 路由配置

```javascript
// frontend/src/router/index.js
{
  path: '/excalidraw',
  name: 'Excalidraw',
  component: () => import('../views/ExcalidrawTool.vue'),
  meta: { title: '画图工具', icon: 'Edit' }
},
{
  path: '/draw/:id',
  name: 'ExcalidrawShare',
  component: () => import('../views/ExcalidrawShareView.vue'),
  meta: { title: '查看画图', hideSidebar: true }
}
```

## 8. 实现步骤

### 阶段一：后端基础 (预计文件)
- [ ] `backend/config/config.go` - 添加 ExcalidrawConfig
- [ ] `backend/models/excalidraw.go` - 数据库模型和操作
- [ ] `backend/handlers/excalidraw.go` - API 处理器
- [ ] `backend/main.go` - 注册路由和清理任务
- [ ] `backend/config.example.yaml` - 添加配置示例

### 阶段二：前端核心
- [ ] 安装 React 依赖
- [ ] `frontend/src/components/ExcalidrawWrapper.vue` - React 包装
- [ ] `frontend/src/views/ExcalidrawTool.vue` - 主编辑器
- [ ] `frontend/src/router/index.js` - 添加路由
- [ ] 实现本地保存/加载
- [ ] 实现导出功能

### 阶段三：云端功能
- [ ] 实现云端保存（带密码）
- [ ] `frontend/src/views/ExcalidrawShareView.vue` - 查看页面
- [ ] 实现"我的画图"管理
- [ ] 实现延期功能

### 阶段四：管理员功能
- [ ] 实现管理员登录
- [ ] 实现管理员面板（列表、查看、删除）
- [ ] 实现永久保存功能

### 阶段五：优化完善
- [ ] 素材库集成（可选）
- [ ] 移动端适配
- [ ] 错误处理和加载状态
- [ ] 性能优化

## 9. 关键参考文件

| 用途 | 文件路径 |
|------|---------|
| API 处理器模式 | `backend/handlers/mdshare.go` |
| 数据库模型模式 | `backend/models/mdshare.go` |
| 前端编辑器模式 | `frontend/src/views/MarkdownTool.vue` |
| 分享查看模式 | `frontend/src/views/MarkdownShareView.vue` |
| 配置结构 | `backend/config/config.go` |
| 路由配置 | `frontend/src/router/index.js` |
| 主程序入口 | `backend/main.go` |

## 10. 验证方案

### 10.1 后端测试

```bash
# 创建云端保存
curl -X POST http://localhost:8082/api/excalidraw \
  -H "Content-Type: application/json" \
  -d '{"content":"{\"type\":\"excalidraw\",\"elements\":[]}","title":"测试","password":"123456","expires_in":30}'

# 获取内容（需密码）
curl "http://localhost:8082/api/excalidraw/{id}?password=123456"

# 创建者获取
curl "http://localhost:8082/api/excalidraw/{id}/creator?creator_key={key}"

# 延期
curl -X PUT http://localhost:8082/api/excalidraw/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"extend","expires_in":30,"creator_key":"xxx"}'

# 删除
curl -X DELETE "http://localhost:8082/api/excalidraw/{id}?creator_key={key}"

# 管理员列表
curl "http://localhost:8082/api/excalidraw/admin/list?admin_password=xxx"
```

### 10.2 前端测试

1. 访问 `/excalidraw`，确认编辑器正常加载
2. 绘制图形，测试导出 PNG/SVG/JSON
3. 保存到本地，刷新页面后加载
4. 保存到云端，获取分享链接
5. 在新窗口打开分享链接，输入密码查看
6. 测试延期、编辑、删除功能
7. 测试管理员登录和管理功能

## 11. 安全考虑

1. **密码保护**：访问密码必填，SHA256 哈希存储
2. **限流**：10次/小时/IP
3. **内容大小**：最大 10MB
4. **内容验证**：验证 Excalidraw JSON 结构
5. **XSS 防护**：标题等文本字段转义
6. **管理员隔离**：单独的管理员密码

## 12. 参考资源

- [Excalidraw 集成文档](https://docs.excalidraw.com/docs/@excalidraw/excalidraw/integration)
- [Excalidraw 导出工具](https://docs.excalidraw.com/docs/@excalidraw/excalidraw/api/utils/export)
- [Excalidraw JSON Schema](https://docs.excalidraw.com/docs/codebase/json-schema)
