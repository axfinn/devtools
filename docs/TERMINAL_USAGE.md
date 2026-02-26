# SSH Terminal 使用指南

## 功能特性

### ✨ 核心功能
- **Web SSH 客户端**：通过浏览器访问远程服务器
- **会话持久化**：刷新页面后自动恢复连接
- **命令历史**：完整记录所有输入输出
- **多会话管理**：同时管理多个 SSH 连接
- **安全加密**：密码和私钥使用 AES-256-GCM 加密存储

### 🔐 安全特性
- 密码和私钥加密存储（AES-256-GCM）
- SSH 主机密钥验证（防止 MITM 攻击）
- 用户令牌隔离（每个用户只能看到自己的会话）
- 管理员密码保护
- 速率限制（防止滥用）

### 🚀 用户体验
- 打开网页即可看到历史 SSH 连接
- 点击即可恢复会话
- 支持密码和私钥两种认证方式
- 响应式设计，支持移动端

## 快速开始

### 1. 配置加密密钥

**重要**：首次使用前，设置加密密钥以保护 SSH 密码：

```bash
# 生成随机密钥
export TERMINAL_ENCRYPTION_KEY=$(openssl rand -base64 32)

# 或者使用自定义密钥（至少32字符）
export TERMINAL_ENCRYPTION_KEY="your-super-secret-key-here-32+"
```

**保存密钥**：将密钥添加到 `.bashrc` 或 `.zshrc` 以永久保存：
```bash
echo 'export TERMINAL_ENCRYPTION_KEY="your-key-here"' >> ~/.bashrc
```

⚠️ **警告**：如果不设置加密密钥，服务器会生成临时密钥，重启后已保存的 SSH 密码将无法解密！

### 2. 配置文件（可选）

编辑 `backend/config.yaml`：

```yaml
ssh:
  # 管理员密码（可查看和管理所有 SSH 会话）
  admin_password: ""

  # 是否验证 SSH 主机密钥（推荐启用）
  host_key_verification: true

  # 每个用户最大会话数
  max_sessions_per_user: 10

  # 会话空闲超时（分钟）
  session_idle_timeout: 5

  # 历史记录最大保存天数
  history_max_age_days: 30

  # 不活跃会话最大保存天数
  session_max_age_days: 7
```

### 3. 启动服务

```bash
# 开发模式
cd backend
export TERMINAL_ENCRYPTION_KEY="your-key-here"
go run main.go

cd frontend
npm run dev

# 生产模式（Docker）
docker-compose up -d
```

### 4. 访问终端

1. 打开浏览器访问 `http://localhost:5173`（开发模式）或 `http://localhost:8082`（生产模式）
2. 点击侧边栏的 "SSH 终端"
3. 输入用户令牌（首次可留空自动生成）
4. 创建新的 SSH 连接

## 使用教程

### 创建 SSH 连接

1. 点击 "新建连接" 按钮
2. 填写连接信息：
   - **连接名称**：自定义名称（可选）
   - **主机地址**：SSH 服务器 IP 或域名
   - **端口**：SSH 端口（默认 22）
   - **用户名**：SSH 用户名
   - **认证方式**：
     - 密码：输入 SSH 密码
     - 私钥：粘贴私钥内容
   - **保持连接**：启用后刷新页面自动重连
   - **过期时间**：会话保留时间（0=永不过期）
3. 点击 "创建并连接"

### 会话管理

#### 查看会话列表
- 未连接时自动显示所有历史会话
- 显示会话状态：在线/离线/已过期
- 显示最后活跃时间

#### 连接到会话
- 点击会话卡片即可连接
- 如果 SSH 已断开，会自动重新建立连接
- 历史命令会自动加载到终端

#### 重命名会话
1. 点击会话卡片的编辑按钮
2. 输入新名称
3. 确认

#### 删除会话
1. 点击会话卡片的删除按钮
2. 确认删除
3. 会话记录和命令历史将被永久删除

### 终端操作

#### 基本操作
- **输入命令**：直接在终端中输入
- **复制文本**：选中文本后自动复制
- **粘贴文本**：Ctrl+V（或 Cmd+V）
- **清屏**：输入 `clear`
- **退出**：点击 "断开连接" 按钮

#### 快捷键
- `Ctrl+C`：中断当前命令
- `Ctrl+D`：发送 EOF
- `Ctrl+L`：清屏
- `Tab`：自动补全
- `↑/↓`：命令历史

### 命令历史

系统会自动记录所有输入和输出：
- 每次连接时自动加载历史
- 支持搜索和过滤（规划中）
- 可导出为文本文件（规划中）
- 自动清理旧历史（默认30天）

### 用户令牌

#### 令牌作用
- 用于隔离不同用户的会话
- 保存在浏览器 localStorage
- 可在多个设备间共享

#### 管理令牌
1. 点击设置按钮
2. 复制用户令牌
3. 在其他设备登录时输入相同令牌
4. 即可访问相同的会话列表

#### 切换用户
1. 清除浏览器缓存
2. 或在设置中清除所有会话
3. 重新登录时输入新令牌

## API 使用示例

### 创建 SSH 会话

```bash
curl -X POST http://localhost:8082/api/terminal \
  -H "Content-Type: application/json" \
  -d '{
    "host": "192.168.1.100",
    "port": 22,
    "username": "root",
    "password": "your-password",
    "name": "我的服务器",
    "user_token": "user_abc123",
    "keep_alive": true,
    "expires_in": 0
  }'
```

响应：
```json
{
  "id": "abc12345",
  "name": "我的服务器",
  "host": "192.168.1.100",
  "port": 22,
  "username": "root",
  "status": "idle",
  "creator_key": "def67890",
  "created_at": "2026-02-26T10:00:00Z",
  "expires_at": null
}
```

### 获取会话列表

```bash
curl "http://localhost:8082/api/terminal/list?user_token=user_abc123"
```

### 获取命令历史

```bash
curl "http://localhost:8082/api/terminal/abc12345/history?user_token=user_abc123"
```

### WebSocket 连接

```javascript
const ws = new WebSocket('ws://localhost:8082/api/terminal/abc12345/ws?user_token=user_abc123')

ws.onopen = () => {
  console.log('连接成功')
}

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)

  if (msg.type === 'output') {
    // 显示输出
    console.log(msg.data)
  } else if (msg.type === 'history') {
    // 加载历史
    console.log('历史:', msg.data)
  }
}

// 发送输入
ws.send(JSON.stringify({
  type: 'input',
  data: 'ls -la\n'
}))

// 调整终端大小
ws.send(JSON.stringify({
  type: 'resize',
  cols: 100,
  rows: 30
}))
```

## 管理员功能

### 查看所有会话

```bash
curl "http://localhost:8082/api/terminal/admin/list?admin_password=your_admin_password"
```

### 删除任意会话

```bash
curl -X DELETE "http://localhost:8082/api/terminal/admin/abc12345?admin_password=your_admin_password"
```

## 安全最佳实践

### 1. 加密密钥管理
- ✅ 使用强随机密钥（至少32字节）
- ✅ 在生产环境通过环境变量传递
- ✅ 定期轮换密钥（需要重新保存所有会话）
- ❌ 不要将密钥硬编码在代码中
- ❌ 不要将密钥提交到版本控制

### 2. 主机密钥验证
- ✅ 生产环境启用 `host_key_verification: true`
- ✅ 首次连接后验证主机指纹
- ⚠️ 如果主机密钥变更，需要从数据库删除旧记录

### 3. 网络安全
- ✅ 使用 HTTPS/WSS 传输（生产环境）
- ✅ 配置防火墙限制访问
- ✅ 启用速率限制
- ✅ 设置会话超时

### 4. 密码管理
- ✅ 优先使用 SSH 私钥认证
- ✅ 使用强密码
- ✅ 定期更换密码
- ✅ 不要在不安全的环境下输入密码

### 5. 会话管理
- ✅ 定期清理不使用的会话
- ✅ 设置合理的过期时间
- ✅ 使用唯一的用户令牌
- ✅ 不要共享创建者密钥

## 故障排查

### 问题：无法连接到 SSH 服务器
**可能原因**：
- SSH 服务器地址或端口错误
- 防火墙阻止连接
- SSH 服务未启动
- 用户名或密码错误

**解决方法**：
1. 确认 SSH 服务器可访问：`ssh user@host -p port`
2. 检查防火墙规则
3. 验证凭据正确性

### 问题：连接后无输出
**可能原因**：
- WebSocket 连接失败
- SSH shell 未正确启动
- 终端大小未正确设置

**解决方法**：
1. 查看浏览器控制台错误
2. 查看服务器日志
3. 尝试重新连接

### 问题：刷新页面后无法恢复会话
**可能原因**：
- 用户令牌丢失
- SSH 连接已超时
- 会话已过期

**解决方法**：
1. 确认 localStorage 中有 `ssh_user_token`
2. 检查会话是否过期
3. 手动重新连接

### 问题：密码解密失败
**可能原因**：
- 加密密钥更改或丢失
- 数据库损坏

**解决方法**：
1. 检查环境变量 `TERMINAL_ENCRYPTION_KEY`
2. 恢复原始密钥
3. 如果无法恢复，删除会话重新创建

## 数据库维护

### 查看会话表

```sql
SELECT id, name, host, port, username, status, last_active_at, created_at
FROM ssh_sessions;
```

### 清理旧会话

```sql
-- 删除7天未活跃的会话
DELETE FROM ssh_sessions
WHERE last_active_at < datetime('now', '-7 days') AND status = 'idle';

-- 删除30天以上的历史记录
DELETE FROM ssh_history
WHERE timestamp < datetime('now', '-30 days');
```

### 重置主机密钥

```sql
-- 删除特定主机的密钥记录（下次连接时重新记录）
DELETE FROM ssh_host_keys
WHERE host = '192.168.1.100' AND port = 22;
```

## 性能优化

### 1. 会话限制
- 限制每个用户最大会话数
- 设置合理的空闲超时
- 定期清理过期会话

### 2. 历史记录
- 设置历史记录最大保存时间
- 定期清理旧记录
- 限制单次查询数量

### 3. WebSocket
- 启用心跳保活
- 设置重连策略
- 限制输出缓冲区大小

### 4. 数据库
- 定期清理过期数据
- 创建适当索引
- 定期备份

## 更新日志

### Version 2.0.0 (2026-02-26) - 完全重构
- ✨ 新增密码加密存储（AES-256-GCM）
- ✨ 新增会话持久化功能
- ✨ 新增命令历史记录
- ✨ 新增会话状态管理
- ✨ 新增主机密钥验证
- ✨ 完全重构前端界面
- ✨ 完全重构后端架构
- 🔒 修复多个安全漏洞
- 🐛 修复 WebSocket 双重读取问题
- 🐛 修复资源泄漏问题
- 📝 完善配置文件支持
- 📝 完善API文档

## 技术架构

### 前端架构
- **框架**：Vue 3 Composition API
- **UI 库**：Element Plus
- **终端库**：xterm.js + addons
- **状态管理**：Reactive API
- **通信**：Fetch API + WebSocket

### 后端架构
- **语言**：Go 1.21+
- **框架**：Gin
- **数据库**：SQLite3
- **SSH 库**：golang.org/x/crypto/ssh
- **WebSocket**：gorilla/websocket
- **加密**：AES-256-GCM

### 数据库设计
- **ssh_sessions**：会话元数据
- **ssh_history**：命令历史记录
- **ssh_host_keys**：SSH 主机密钥

### 安全设计
- 密码加密存储
- 主机密钥验证
- 用户令牌隔离
- 速率限制
- 自动清理

## 常见问题 (FAQ)

**Q: 为什么需要设置加密密钥？**
A: 加密密钥用于保护存储在数据库中的 SSH 密码和私钥。没有密钥，任何能访问数据库的人都能看到明文密码。

**Q: 可以在多个设备使用同一个用户令牌吗？**
A: 可以。只需在另一个设备登录时输入相同的用户令牌即可。

**Q: 会话会永久保存吗？**
A: 取决于创建时设置的过期时间。默认情况下，超过7天未活跃的会话会被自动清理。

**Q: 如何备份我的 SSH 会话？**
A: 备份用户令牌和创建者密钥即可。会话数据存储在服务器数据库中，定期备份数据库文件。

**Q: 支持 SSH 跳板机吗？**
A: 当前版本不支持。需要先 SSH 到跳板机，再从跳板机连接目标服务器。

**Q: 支持文件传输吗？**
A: 当前版本不支持。可以使用 `scp`、`rsync` 等命令行工具。

## 反馈与支持

- **Bug 报告**：https://github.com/your-repo/issues
- **功能建议**：https://github.com/your-repo/discussions
- **文档贡献**：欢迎提交 PR

## 许可证

MIT License - 详见 LICENSE 文件
