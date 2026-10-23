# 贡献指南

感谢你考虑为 DevTools 项目做出贡献！本指南将帮助你了解如何参与到项目中来。

## 行为准则

我们致力于为每个人提供一个友好、安全和受欢迎的环境。请遵循以下原则：

- 使用友好和包容的语言
- 尊重不同的观点和经历
- 优雅地接受建设性的批评
- 关注对社区最有利的事情
- 对其他社区成员表现出同理心

## 如何贡献

### 报告 Bug

如果你发现了 bug，请通过 GitHub Issues 报告：

1. 访问 [Issues 页面](https://github.com/your-org/devtools/issues)
2. 点击 "New Issue"
3. 选择 "Bug Report" 模板
4. 填写以下信息：
   - **Bug 描述**: 清晰简洁地描述问题
   - **重现步骤**: 详细的重现步骤
   - **预期行为**: 你期望发生什么
   - **实际行为**: 实际发生了什么
   - **环境信息**: 操作系统、浏览器版本、Go/Node 版本等
   - **截图**: 如果适用，添加截图

**Bug 报告示例**:
```markdown
**Bug 描述**
在 JSON 工具中粘贴超过 100KB 的内容时，界面会卡顿。

**重现步骤**
1. 访问 /json 页面
2. 粘贴一个 200KB 的 JSON 文件
3. 点击"格式化"按钮

**预期行为**
应该正常格式化并显示

**实际行为**
浏览器卡顿 5-10 秒

**环境**
- OS: macOS 13.0
- Browser: Chrome 120
- Frontend: Vue 3.5.24
```

### 提出功能请求

如果你有新功能的想法：

1. 访问 [Issues 页面](https://github.com/your-org/devtools/issues)
2. 点击 "New Issue"
3. 选择 "Feature Request" 模板
4. 描述你的想法：
   - **功能描述**: 你想要什么功能
   - **使用场景**: 为什么需要这个功能
   - **解决方案**: 你建议如何实现
   - **替代方案**: 其他可能的解决方案

### 提交代码

#### 准备工作

1. **Fork 项目**
   ```bash
   # 在 GitHub 上点击 Fork 按钮
   ```

2. **克隆你的 Fork**
   ```bash
   git clone https://github.com/your-username/devtools.git
   cd devtools
   ```

3. **添加上游仓库**
   ```bash
   git remote add upstream https://github.com/original-org/devtools.git
   ```

4. **创建功能分支**
   ```bash
   git checkout -b feat/my-feature
   # 或
   git checkout -b fix/some-bug
   ```

#### 开发流程

1. **编写代码**
   - 遵循 [代码规范](./README_DEV.md#开发规范)
   - 添加必要的注释
   - 确保代码可读性

2. **添加测试**（强烈推荐）
   ```go
   // 后端测试示例
   func TestMyFeature(t *testing.T) {
       // ...
   }
   ```

   ```javascript
   // 前端测试示例
   describe('MyFeature', () => {
     it('should work correctly', () => {
       // ...
     })
   })
   ```

3. **运行测试**
   ```bash
   # 后端
   cd backend && go test ./...

   # 前端
   cd frontend && pnpm test
   ```

4. **检查代码质量**
   ```bash
   # 后端: 运行 gofmt
   cd backend && gofmt -w .

   # 前端: 运行 ESLint
   cd frontend && pnpm lint
   ```

#### 提交代码

1. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add my feature"
   ```

   **提交信息格式** (Conventional Commits):
   ```
   <type>(<scope>): <subject>

   <body>

   <footer>
   ```

   **Type 类型**:
   - `feat`: 新功能
   - `fix`: Bug 修复
   - `docs`: 文档更新
   - `style`: 代码格式（不影响功能）
   - `refactor`: 重构
   - `test`: 测试相关
   - `chore`: 构建工具、依赖更新

   **示例**:
   ```
   feat(json): 添加 JSON Schema 验证功能

   - 支持自定义 Schema
   - 提供常用 Schema 模板
   - 显示验证错误详情

   Closes #123
   ```

2. **推送到你的 Fork**
   ```bash
   git push origin feat/my-feature
   ```

3. **创建 Pull Request**
   - 访问你的 Fork 页面
   - 点击 "New Pull Request"
   - 填写 PR 描述：
     - 简要说明你的更改
     - 关联相关 Issue（如 `Closes #123`）
     - 提供测试说明
     - 添加截图（如果适用）

#### Pull Request 检查清单

在提交 PR 前，请确认：

- [ ] 代码遵循项目的代码规范
- [ ] 添加了必要的注释和文档
- [ ] 所有测试通过
- [ ] 没有引入新的警告或错误
- [ ] 提交信息清晰且遵循规范
- [ ] PR 描述完整，关联了相关 Issue
- [ ] 代码已经在本地测试过
- [ ] 更新了相关文档（如果需要）

### 代码审查

所有提交都需要经过代码审查：

1. **响应审查意见**
   - 维护者会审查你的代码
   - 可能会提出修改建议
   - 请及时响应并更新代码

2. **更新 PR**
   ```bash
   # 在你的分支上继续开发
   git add .
   git commit -m "fix: address review comments"
   git push origin feat/my-feature
   ```

3. **合并**
   - 审查通过后，维护者会合并你的 PR
   - 你的贡献将出现在下一个版本中

## 开发指南

### 项目结构

详见 [开发者文档](./README_DEV.md#项目架构)

### 添加新工具

如果你想添加一个新的开发工具（如 "UUID 生成器"）：

#### 1. 纯前端工具

```bash
# 1. 创建 Vue 组件
touch frontend/src/views/UuidTool.vue

# 2. 注册路由
# 编辑 frontend/src/router/index.js
{
  path: '/uuid',
  name: 'Uuid',
  component: () => import('../views/UuidTool.vue'),
  meta: { title: 'UUID 生成', icon: 'Key' }
}

# 3. 实现功能
# 在 UuidTool.vue 中实现 UUID 生成逻辑
```

#### 2. 需要后端支持的工具

```bash
# 1. 创建数据模型
# backend/models/your_model.go

# 2. 创建 API 处理器
# backend/handlers/your_handler.go

# 3. 注册路由
# 在 backend/main.go 中注册 API 路由

# 4. 创建前端组件
# frontend/src/views/YourTool.vue
```

详见 [CLAUDE.md 开发流程](./CLAUDE.md#开发流程)

### 数据库迁移

如果需要修改数据库结构：

```go
// 在相应的 models/*.go 文件中的 Init 函数中添加
db.conn.Exec("ALTER TABLE xxx ADD COLUMN yyy TEXT DEFAULT ''")
```

**注意**: SQLite 的 ALTER TABLE 有限制，复杂迁移可能需要重建表。

## 文档贡献

文档同样重要！你可以：

- 修正拼写错误
- 改进说明
- 添加示例
- 翻译文档

## 问题和讨论

- **Bug 和功能请求**: 使用 [GitHub Issues](https://github.com/your-org/devtools/issues)
- **一般讨论**: 使用 [GitHub Discussions](https://github.com/your-org/devtools/discussions)
- **安全问题**: 请发送邮件到 security@example.com（不要公开）

## 版本发布流程

（维护者专用）

1. 更新版本号
2. 更新 CHANGELOG.md
3. 创建 Git tag
4. 构建 Docker 镜像
5. 发布 GitHub Release

## 许可证

通过贡献代码，你同意你的贡献将在 [MIT License](./LICENSE) 下发布。

## 获取帮助

如果你在贡献过程中遇到问题：

- 查看 [开发者文档](./README_DEV.md)
- 查看 [常见问题](./README_DEV.md#常见问题)
- 在 [Discussions](https://github.com/your-org/devtools/discussions) 中提问
- 联系维护者

---

再次感谢你的贡献！🎉
