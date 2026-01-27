# Markdown 分享管理员面板实现计划

## 概述

为 Markdown 分享功能添加前端管理员面板，支持查看和管理所有分享。

## 当前状态

- 后端 API 已实现完成
- 缺少前端管理界面

## 已有 API

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/mdshare/admin/list?admin_password=xxx` | 获取所有分享列表 |
| GET | `/api/mdshare/admin/:id?admin_password=xxx` | 查看分享内容（不消耗次数） |
| DELETE | `/api/mdshare/admin/:id?admin_password=xxx` | 删除分享 |

## 实现计划

### 1. 前端界面设计

**入口位置**: `MarkdownTool.vue` 页面，header 区域添加"管理"按钮

**界面组件**:
- 密码输入对话框（首次使用时）
- 分享列表表格
- 内容预览对话框
- 批量操作功能

### 2. 功能需求

#### 2.1 密码验证
- 输入管理员密码
- 密码存储到 sessionStorage（当前会话有效）
- 密码错误提示

#### 2.2 分享列表
- 显示所有分享：ID、标题、剩余次数、创建时间、过期时间
- 支持搜索/筛选
- 分页显示

#### 2.3 管理操作
- 查看内容（预览模式，渲染 Markdown）
- 删除分享
- 批量删除
- 复制分享链接

### 3. 文件改动

```
frontend/src/views/MarkdownTool.vue
├── 添加"管理"按钮
├── 添加密码输入对话框
├── 添加管理面板对话框
└── 添加相关逻辑函数
```

### 4. 代码示例

```vue
<!-- 管理员按钮 -->
<el-button v-if="isAdmin" type="danger" @click="showAdminPanel = true">
  <el-icon><Setting /></el-icon>
  管理
</el-button>

<!-- 密码对话框 -->
<el-dialog v-model="showAdminLogin" title="管理员登录" width="350px">
  <el-input v-model="adminPassword" type="password" placeholder="请输入管理员密码" />
  <template #footer>
    <el-button @click="showAdminLogin = false">取消</el-button>
    <el-button type="primary" @click="verifyAdminPassword">确认</el-button>
  </template>
</el-dialog>

<!-- 管理面板 -->
<el-dialog v-model="showAdminPanel" title="分享管理" width="900px">
  <el-table :data="allShares" v-loading="loadingAllShares">
    <el-table-column prop="id" label="ID" width="100" />
    <el-table-column prop="title" label="标题" />
    <el-table-column label="次数" width="100">
      <template #default="{ row }">
        {{ row.remaining_views }}/{{ row.max_views }}
      </template>
    </el-table-column>
    <el-table-column prop="created_at" label="创建时间" width="180" />
    <el-table-column label="操作" width="150">
      <template #default="{ row }">
        <el-button size="small" @click="previewShare(row)">查看</el-button>
        <el-button size="small" type="danger" @click="deleteShareAdmin(row)">删除</el-button>
      </template>
    </el-table-column>
  </el-table>
</el-dialog>
```

### 5. 逻辑函数

```javascript
// 状态
const isAdmin = ref(false)
const adminPassword = ref('')
const showAdminLogin = ref(false)
const showAdminPanel = ref(false)
const allShares = ref([])

// 验证密码
const verifyAdminPassword = async () => {
  const res = await fetch(`/api/mdshare/admin/list?admin_password=${adminPassword.value}`)
  if (res.ok) {
    sessionStorage.setItem('mdshare_admin_pwd', adminPassword.value)
    isAdmin.value = true
    showAdminLogin.value = false
    await loadAllShares()
    showAdminPanel.value = true
  } else {
    ElMessage.error('密码错误')
  }
}

// 加载所有分享
const loadAllShares = async () => {
  const pwd = sessionStorage.getItem('mdshare_admin_pwd')
  const res = await fetch(`/api/mdshare/admin/list?admin_password=${pwd}`)
  const data = await res.json()
  allShares.value = data.list || []
}

// 删除分享
const deleteShareAdmin = async (share) => {
  const pwd = sessionStorage.getItem('mdshare_admin_pwd')
  await fetch(`/api/mdshare/admin/${share.id}?admin_password=${pwd}`, { method: 'DELETE' })
  await loadAllShares()
}
```

## 优先级

**中等** - 非核心功能，但对运维有帮助

## 预估工作量

- 前端界面: ~100 行代码
- 测试验证: 需要

## 注意事项

1. 密码仅存 sessionStorage，关闭浏览器失效
2. 管理员查看不消耗查看次数
3. 需要在 config.yaml 中配置 admin_password 才能使用
