# Terminal 模块重构架构设计

## 1. 核心设计原则

### 1.1 安全第一
- 所有敏感数据（密码、私钥）使用 AES-256-GCM 加密存储
- 主密钥通过环境变量或配置文件管理
- SSH 连接使用主机密钥验证（可配置）
- 用户令牌使用 JWT 或加密的随机令牌

### 1.2 会话持久化
- 会话元数据存储在数据库
- 命令历史完整记录（输入+输出+时间戳）
- 支持会话恢复（刷新页面后继续使用）
- 会话状态：`active`（连接中）、`idle`（已断开）、`expired`（已过期）

### 1.3 简洁的数据流
```
用户输入 → WebSocket → SSH Session → 输出缓冲 → WebSocket → 前端显示
                ↓
            命令历史存储
```

## 2. 数据库设计

### 2.1 ssh_sessions 表
```sql
CREATE TABLE ssh_sessions (
    id TEXT PRIMARY KEY,                    -- 8字符随机ID
    name TEXT,                              -- 会话名称
    host TEXT NOT NULL,                     -- SSH主机
    port INTEGER DEFAULT 22,                -- SSH端口
    username TEXT NOT NULL,                 -- SSH用户名
    password_encrypted TEXT,                -- AES加密的密码
    private_key_encrypted TEXT,             -- AES加密的私钥
    host_key_fingerprint TEXT,              -- 主机密钥指纹（用于验证）

    user_token TEXT NOT NULL,               -- 用户令牌（隔离会话）
    creator_ip TEXT,                        -- 创建者IP

    width INTEGER DEFAULT 80,               -- 终端宽度
    height INTEGER DEFAULT 24,              -- 终端高度

    status TEXT DEFAULT 'idle',             -- 状态: active/idle/expired
    keep_alive BOOLEAN DEFAULT 1,           -- 是否保持会话
    last_active_at DATETIME,                -- 最后活跃时间

    expires_at DATETIME,                    -- 过期时间
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ssh_user_token ON ssh_sessions(user_token);
CREATE INDEX idx_ssh_status ON ssh_sessions(status);
CREATE INDEX idx_ssh_expires ON ssh_sessions(expires_at);
```

### 2.2 ssh_history 表
```sql
CREATE TABLE ssh_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,               -- 关联会话ID
    type TEXT NOT NULL,                     -- 'input' or 'output'
    content TEXT NOT NULL,                  -- 命令或输出内容
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (session_id) REFERENCES ssh_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_history_session ON ssh_history(session_id, timestamp);
```

### 2.3 ssh_host_keys 表（可选，用于主机密钥管理）
```sql
CREATE TABLE ssh_host_keys (
    host TEXT PRIMARY KEY,
    port INTEGER,
    key_type TEXT,
    fingerprint TEXT NOT NULL,
    public_key TEXT,
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 3. 后端架构

### 3.1 加密模块 (utils/encryption.go)
```go
type EncryptionService struct {
    masterKey []byte  // 32字节主密钥
}

func NewEncryptionService(masterKey string) *EncryptionService
func (e *EncryptionService) Encrypt(plaintext string) (string, error)
func (e *EncryptionService) Decrypt(ciphertext string) (string, error)
```

### 3.2 SSH 会话管理器 (handlers/terminal.go)
```go
// 内存中的活跃会话管理
type SSHSession struct {
    ID            string
    Client        *ssh.Client
    Session       *ssh.Session

    // 通道管理
    InputChan     chan []byte
    OutputChan    chan []byte
    ControlChan   chan string  // 控制命令：resize, close

    // 状态
    Status        string
    LastActive    time.Time

    // 同步
    mu            sync.RWMutex
    cancelCtx     context.Context
    cancel        context.CancelFunc
}

type SSHManager struct {
    sessions      map[string]*SSHSession
    encryption    *EncryptionService
    db            *sql.DB
    mu            sync.RWMutex
}

func (m *SSHManager) CreateSession(config SSHConfig) (*SSHSession, error)
func (m *SSHManager) GetSession(id string) (*SSHSession, error)
func (m *SSHManager) ResumeSession(id string, userToken string) (*SSHSession, error)
func (m *SSHManager) CloseSession(id string)
func (m *SSHManager) StartHeartbeat(session *SSHSession)
func (m *SSHManager) SaveHistory(sessionID, msgType, content string) error
```

### 3.3 WebSocket 处理流程
```go
func HandleWebSocket(c *gin.Context) {
    sessionID := c.Param("id")
    userToken := c.Query("user_token")

    // 1. 验证权限
    // 2. 获取或恢复会话
    session, err := sshManager.ResumeSession(sessionID, userToken)

    // 3. 升级到 WebSocket
    conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)

    // 4. 启动三个协程
    go handleInput(conn, session)      // WebSocket → SSH
    go handleOutput(conn, session)     // SSH → WebSocket
    go handleControl(conn, session)    // 控制命令

    // 5. 等待断开
    <-session.cancelCtx.Done()
}
```

### 3.4 API 端点设计
```
POST   /api/terminal                     # 创建新会话
POST   /api/terminal/login               # 用户登录（获取token）
GET    /api/terminal/list                # 获取用户所有会话
GET    /api/terminal/:id                 # 获取会话详情
GET    /api/terminal/:id/history         # 获取会话历史
PUT    /api/terminal/:id                 # 更新会话（重命名等）
DELETE /api/terminal/:id                 # 删除会话
GET    /api/terminal/:id/ws              # WebSocket连接

POST   /api/terminal/:id/resume          # 恢复会话（重连SSH）
POST   /api/terminal/:id/disconnect      # 断开SSH但保留会话

GET    /api/terminal/admin/list          # 管理员列表
DELETE /api/terminal/admin/:id           # 管理员删除
```

## 4. 前端架构

### 4.1 组件结构
```
TerminalTool.vue
├── SessionList (会话列表侧边栏)
│   ├── 显示所有历史会话
│   ├── 状态指示器（在线/离线）
│   └── 快速连接按钮
├── TerminalView (终端显示区)
│   ├── xterm.js 实例
│   ├── 命令历史查看
│   └── 连接状态提示
└── ControlPanel (控制面板)
    ├── 重命名会话
    ├── 断开/恢复连接
    └── 删除会话
```

### 4.2 状态管理
```javascript
const state = reactive({
    userToken: localStorage.getItem('terminal_user_token'),
    sessions: [],           // 所有会话列表
    activeSession: null,    // 当前激活会话
    wsConnection: null,     // WebSocket连接
    terminal: null,         // xterm实例
    connectionStatus: 'disconnected'  // connected/connecting/disconnected
})
```

### 4.3 WebSocket 消息协议
```javascript
// 客户端 → 服务器
{
    type: "input",
    data: "command\n"
}
{
    type: "resize",
    cols: 80,
    rows: 24
}
{
    type: "ping"
}

// 服务器 → 客户端
{
    type: "output",
    data: "output text"
}
{
    type: "status",
    status: "connected",
    message: "SSH连接成功"
}
{
    type: "pong"
}
{
    type: "history",
    items: [...]
}
```

## 5. 核心功能实现

### 5.1 会话创建流程
```
1. 用户输入 SSH 连接信息
2. 前端加密密码（可选，或传输时HTTPS保护）
3. POST /api/terminal 创建会话
4. 后端：
   - 加密密码存储到数据库
   - 生成会话ID
   - 尝试建立 SSH 连接
   - 返回会话ID和状态
5. 前端自动连接 WebSocket
```

### 5.2 会话恢复流程
```
1. 用户打开页面
2. 从 localStorage 读取 user_token
3. GET /api/terminal/list 获取所有会话
4. 显示会话列表（含状态）
5. 用户点击会话
6. 如果会话已断开：
   - POST /api/terminal/:id/resume 重新连接
7. 连接 WebSocket
8. 显示历史命令（GET /api/terminal/:id/history）
```

### 5.3 命令历史功能
```
1. 所有输入输出自动保存到 ssh_history
2. 用户可以查看完整历史
3. 支持搜索和过滤
4. 类似 Claude Code 的对话记录展示
5. 可导出为文本文件
```

### 5.4 自动清理机制
```
每小时运行：
1. 删除超过7天未活跃的会话
2. 删除已过期的会话
3. 清理孤立的历史记录
4. 关闭僵尸 SSH 连接
```

## 6. 安全措施

### 6.1 密码加密
- 主密钥从环境变量 `TERMINAL_ENCRYPTION_KEY` 读取
- 如果未设置，生成随机密钥并警告（重启后无法解密）
- 使用 AES-256-GCM（提供认证加密）
- 每次加密使用随机 nonce

### 6.2 SSH 主机密钥验证
```go
// 首次连接记录主机密钥
// 后续连接验证是否匹配
hostKeyCallback := ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
    // 查询数据库中的已知主机密钥
    // 匹配则通过，不匹配则拒绝
})
```

### 6.3 权限控制
- 用户只能访问自己的会话（通过 user_token）
- 管理员密码访问所有会话
- IP 速率限制：每小时最多创建 20 个会话

## 7. 性能优化

### 7.1 连接复用
- 保持 SSH 连接活跃（30秒心跳）
- WebSocket 断开不立即关闭 SSH
- 5分钟无活动后关闭 SSH 但保留会话记录

### 7.2 历史记录分页
- 只加载最近 100 条历史
- 支持上拉加载更多
- 长期历史定期归档

### 7.3 输出缓冲
- 使用环形缓冲区（10MB）
- 避免内存无限增长
- 超过限制丢弃旧数据

## 8. 配置示例

### backend/config.yaml
```yaml
terminal:
  admin_password: "admin123"
  encryption_key: ""  # 留空则使用环境变量

  # SSH 配置
  ssh_timeout: 30s
  max_sessions_per_user: 10
  session_idle_timeout: 5m
  session_max_lifetime: 24h

  # 历史记录
  history_max_age: 30d
  history_max_items: 10000

  # 安全
  host_key_verification: true  # 是否验证主机密钥
  allow_password_auth: true
  allow_key_auth: true
```

## 9. 实施步骤

1. ✅ 架构设计（当前）
2. ⏳ 实现加密模块
3. ⏳ 重写数据库模型
4. ⏳ 重写 SSH 管理器
5. ⏳ 重构 WebSocket
6. ⏳ 实现缺失 API
7. ⏳ 重构前端
8. ⏳ 配置和测试
9. ⏳ 文档更新

## 10. 与现有功能对比

| 功能 | 旧实现 | 新实现 |
|------|--------|--------|
| 密码存储 | 明文 | AES-256-GCM 加密 |
| 会话持久化 | 无 | 完整支持 |
| 命令历史 | 无 | 完整记录 |
| 会话恢复 | 无 | 刷新页面可恢复 |
| 主机密钥验证 | InsecureIgnore | 可配置验证 |
| 输出管理 | 双重读取 | 单一通道 |
| 资源清理 | 泄漏 | 自动清理 |
| API 完整性 | 缺失3个方法 | 全部实现 |

## 11. 用户体验改进

### 打开网页即可看到历史连接
- 自动加载用户的所有会话
- 显示每个会话的状态（在线/离线/已过期）
- 点击即可恢复连接

### 类似 Claude Code 的对话体验
- 完整的命令历史记录
- 支持搜索和导出
- 时间戳和分组显示
- 可以查看任意时间点的会话快照

### 无缝重连
- WebSocket 断开自动重连
- 刷新页面自动恢复会话
- 网络波动不影响使用
