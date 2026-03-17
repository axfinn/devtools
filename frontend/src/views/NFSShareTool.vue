<template>
  <div class="container mx-auto p-4 max-w-7xl">
    <!-- 未启用提示 -->
    <el-alert
      v-if="statusChecked && !nfsEnabled"
      title="NFS 分享功能未启用"
      type="warning"
      description="请在后端 config.yaml 中配置 nfs_share.enabled: true 及 mount_path 后重启服务。"
      show-icon
      :closable="false"
      class="mb-4"
    />

    <!-- 超管登录区 -->
    <el-card v-if="nfsEnabled && !adminLoggedIn" class="mb-4 max-w-md mx-auto">
      <template #header>
        <div class="flex items-center gap-2">
          <el-icon><Lock /></el-icon>
          <span class="font-bold text-lg">NFS 分享 · 超管登录</span>
        </div>
      </template>
      <el-form @submit.prevent="loginAdmin" class="space-y-4">
        <el-form-item label="超管密码">
          <el-input
            v-model="adminPassword"
            type="password"
            placeholder="请输入超管密码"
            show-password
            @keyup.enter="loginAdmin"
          />
        </el-form-item>
        <el-button type="primary" :loading="loginLoading" @click="loginAdmin" class="w-full">
          登录
        </el-button>
      </el-form>
    </el-card>

    <!-- 主面板（已登录） -->
    <template v-if="nfsEnabled && adminLoggedIn">
      <!-- 标题栏 -->
      <div class="flex items-center justify-between mb-4">
        <div class="flex items-center gap-2">
          <el-icon class="text-xl text-blue-500"><FolderOpened /></el-icon>
          <span class="text-xl font-bold">NFS 文件分享</span>
          <el-tag type="success" size="small">已登录</el-tag>
        </div>
        <el-button size="small" @click="logout">退出登录</el-button>
      </div>

      <el-tabs v-model="activeTab" class="nfs-tabs">
        <!-- Tab 1: 文件浏览 & 创建分享 -->
        <el-tab-pane label="文件浏览 & 创建分享" name="browse">
          <div class="flex gap-4 mt-2" style="min-height: 500px;">
            <!-- 左侧：目录浏览 -->
            <el-card class="flex-1" style="min-width: 0;">
              <template #header>
                <div class="flex items-center justify-between">
                  <span class="font-semibold">目录浏览</span>
                  <div class="flex items-center gap-2 text-sm text-gray-500">
                    <span>当前路径：</span>
                    <el-breadcrumb separator="/">
                      <el-breadcrumb-item
                        v-for="(seg, i) in breadcrumbs"
                        :key="i"
                        :class="{ 'cursor-pointer text-blue-500': i < breadcrumbs.length - 1 }"
                        @click="i < breadcrumbs.length - 1 && navigateTo(seg.path)"
                      >{{ seg.name }}</el-breadcrumb-item>
                    </el-breadcrumb>
                  </div>
                </div>
              </template>
              <el-skeleton :loading="browseLoading" animated>
                <template #default>
                  <el-table
                    :data="dirEntries"
                    size="small"
                    stripe
                    @row-click="handleEntryClick"
                    style="cursor: pointer;"
                  >
                    <el-table-column width="36">
                      <template #default="{ row }">
                        <el-icon :class="row.is_dir ? 'text-yellow-500' : 'text-blue-400'">
                          <Folder v-if="row.is_dir" />
                          <Document v-else />
                        </el-icon>
                      </template>
                    </el-table-column>
                    <el-table-column label="名称" prop="name" min-width="200">
                      <template #default="{ row }">
                        <span :class="row.is_dir ? 'font-medium text-yellow-700' : ''">
                          {{ row.name }}
                        </span>
                      </template>
                    </el-table-column>
                    <el-table-column label="大小" width="110">
                      <template #default="{ row }">
                        <span v-if="!row.is_dir" class="text-gray-500 text-xs">
                          {{ formatSize(row.size) }}
                        </span>
                        <span v-else class="text-gray-400 text-xs">—</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="修改时间" width="150">
                      <template #default="{ row }">
                        <span class="text-gray-400 text-xs">{{ formatDate(row.mod_time) }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column label="操作" width="90">
                      <template #default="{ row }">
                        <el-button
                          v-if="!row.is_dir"
                          size="small"
                          type="primary"
                          link
                          @click.stop="selectFile(row)"
                        >选为分享</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </template>
              </el-skeleton>
            </el-card>

            <!-- 右侧：创建分享 -->
            <el-card style="width: 340px; flex-shrink: 0;">
              <template #header>
                <span class="font-semibold">创建分享</span>
              </template>
              <el-form :model="createForm" label-width="90px" size="default">
                <el-form-item label="选中文件">
                  <div v-if="createForm.filePath" class="text-sm">
                    <div class="font-medium truncate" :title="createForm.filePath">
                      {{ createForm.filePath.split('/').pop() }}
                    </div>
                    <div class="text-gray-400 text-xs truncate" :title="createForm.filePath">
                      {{ createForm.filePath }}
                    </div>
                    <div class="text-gray-400 text-xs">{{ formatSize(selectedFileSize) }}</div>
                  </div>
                  <span v-else class="text-gray-400 text-sm">请在左侧点击"选为分享"</span>
                </el-form-item>
                <el-form-item label="显示名称">
                  <el-input v-model="createForm.name" placeholder="分享名称（必填）" />
                </el-form-item>
                <el-form-item label="访问次数">
                  <el-input-number
                    v-model="createForm.maxViews"
                    :min="1"
                    :max="99999"
                    controls-position="right"
                    class="w-full"
                  />
                </el-form-item>
                <el-form-item label="有效期">
                  <el-select v-model="createForm.expiresDays" class="w-full">
                    <el-option :value="0" label="永不过期" />
                    <el-option :value="1" label="1 天" />
                    <el-option :value="3" label="3 天" />
                    <el-option :value="7" label="7 天" />
                    <el-option :value="30" label="30 天" />
                    <el-option :value="90" label="90 天" />
                    <el-option :value="365" label="1 年" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-button
                    type="primary"
                    :loading="createLoading"
                    :disabled="!createForm.filePath || !createForm.name"
                    class="w-full"
                    @click="createShare"
                  >创建分享链接</el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </el-tab-pane>

        <!-- Tab 2: 挂载管理 -->
        <el-tab-pane label="挂载管理" name="mounts">
          <div class="mt-2">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm text-gray-500">挂载点来自 config.yaml → nfs_share.mounts</span>
              <el-button size="small" :loading="mountsLoading" @click="loadMounts">
                <el-icon><Refresh /></el-icon> 刷新状态
              </el-button>
            </div>
            <el-empty v-if="!mountsLoading && mountsList.length === 0" description="暂无挂载点，请在 config.yaml 中配置 nfs_share.mounts" />
            <div class="grid gap-3">
              <el-card
                v-for="m in mountsList"
                :key="m.name"
                :class="['border-l-4', m.mounted ? 'border-l-green-400' : 'border-l-red-400']"
                shadow="never"
              >
                <div class="flex items-start justify-between">
                  <div class="space-y-1">
                    <div class="flex items-center gap-2">
                      <span class="font-bold text-base">{{ m.name }}</span>
                      <el-tag :type="m.mounted ? 'success' : 'danger'" size="small">
                        {{ m.mounted ? '已挂载' : '未挂载' }}
                      </el-tag>
                      <el-tag size="small" type="info">{{ m.type.toUpperCase() }}</el-tag>
                    </div>
                    <div class="text-sm text-gray-600">
                      <span v-if="m.type === 'nfs'">{{ m.host }}:{{ m.export }}</span>
                      <span v-else-if="m.type === 'local'">{{ m.export }}</span>
                      <span v-else>//{{ m.host }}/{{ m.share }}
                        <span v-if="m.username" class="text-gray-400 ml-1">（用户：{{ m.username }}）</span>
                      </span>
                    </div>
                    <div class="text-xs text-gray-400">本地挂载点：{{ m.local_path }}</div>
                    <div v-if="m.mounted_at" class="text-xs text-gray-400">
                      挂载时间：{{ formatDate(m.mounted_at) }}
                    </div>
                    <div v-if="m.error" class="text-xs text-red-500 mt-1">
                      错误：{{ m.error }}
                    </div>
                  </div>
                  <div class="flex gap-2 ml-4 flex-shrink-0">
                    <el-button
                      size="small"
                      type="primary"
                      :loading="mountActionLoading === m.name + '_remount'"
                      @click="remount(m.name)"
                    >重新挂载</el-button>
                    <el-button
                      v-if="m.mounted"
                      size="small"
                      type="warning"
                      :loading="mountActionLoading === m.name + '_umount'"
                      @click="umount(m.name)"
                    >卸载</el-button>
                  </div>
                </div>
              </el-card>
            </div>

            <!-- config.yaml 示例 -->
            <el-collapse class="mt-4">
              <el-collapse-item title="config.yaml 配置示例" name="example">
                <pre class="bg-gray-50 rounded p-3 text-xs overflow-x-auto">{{configExample}}</pre>
              </el-collapse-item>
            </el-collapse>
          </div>
        </el-tab-pane>

        <!-- Tab 3: 分享列表 -->
        <el-tab-pane name="list">
          <template #label>
            <span>分享列表</span>
            <el-badge v-if="listTotal > 0" :value="listTotal" class="ml-1" type="info" />
          </template>
          <div class="mt-2">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm text-gray-500">共 {{ listTotal }} 条分享记录</span>
              <el-button size="small" :loading="listLoading" @click="loadShareList">
                <el-icon><Refresh /></el-icon> 刷新
              </el-button>
            </div>
            <el-table :data="shareList" size="small" stripe v-loading="listLoading">
              <el-table-column label="名称" min-width="160">
                <template #default="{ row }">
                  <div class="font-medium">{{ row.name }}</div>
                  <div class="text-xs text-gray-400 truncate" :title="row.file_path">{{ row.file_path }}</div>
                </template>
              </el-table-column>
              <el-table-column label="大小" width="90">
                <template #default="{ row }">
                  <span class="text-xs">{{ formatSize(row.file_size) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="访问次数" width="120" align="center">
                <template #default="{ row }">
                  <el-progress
                    :percentage="row.max_views > 0 ? Math.round((row.views / row.max_views) * 100) : 0"
                    :status="row.views >= row.max_views ? 'exception' : ''"
                    :stroke-width="6"
                  />
                  <div class="text-xs text-center mt-1">
                    {{ row.views }} / {{ row.max_views }}
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="80" align="center">
                <template #default="{ row }">
                  <el-tag :type="getShareStatus(row).type" size="small">
                    {{ getShareStatus(row).label }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="有效期" width="130">
                <template #default="{ row }">
                  <span v-if="row.expires_at" class="text-xs" :class="isExpired(row.expires_at) ? 'text-red-500' : 'text-gray-500'">
                    {{ formatDate(row.expires_at) }}
                  </span>
                  <span v-else class="text-xs text-green-500">永不过期</span>
                </template>
              </el-table-column>
              <el-table-column label="创建时间" width="130">
                <template #default="{ row }">
                  <span class="text-xs text-gray-400">{{ formatDate(row.created_at) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="200" align="center">
                <template #default="{ row }">
                  <el-button size="small" type="primary" link @click="copyShareLink(row)">复制链接</el-button>
                  <el-button size="small" type="info" link @click="viewLogs(row)">查看日志</el-button>
                  <el-button size="small" link @click="openEditDialog(row)">调整</el-button>
                  <el-button size="small" type="danger" link @click="deleteShare(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div class="flex justify-end mt-3">
              <el-pagination
                v-model:current-page="listPage"
                :page-size="listPageSize"
                :total="listTotal"
                layout="prev, pager, next"
                @current-change="loadShareList"
              />
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- 访问日志弹窗 -->
    <el-dialog
      v-model="logsDialogVisible"
      :title="`访问日志 · ${currentShare?.name || ''}`"
      width="800px"
      destroy-on-close
    >
      <div class="flex items-center justify-between mb-3">
        <span class="text-sm text-gray-500">共 {{ logsTotal }} 条访问记录</span>
        <el-button size="small" @click="loadLogs">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
      <el-table :data="logList" size="small" stripe v-loading="logsLoading" max-height="400">
        <el-table-column label="时间" width="155">
          <template #default="{ row }">
            <span class="text-xs">{{ formatDate(row.accessed_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="IP" width="130" prop="client_ip" />
        <el-table-column label="状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="getLogStatusTag(row.status)" size="small">
              {{ getLogStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="传输大小" width="100">
          <template #default="{ row }">
            <span class="text-xs">{{ row.bytes_sent > 0 ? formatSize(row.bytes_sent) : '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="User-Agent" min-width="200">
          <template #default="{ row }">
            <span class="text-xs text-gray-500 truncate block" :title="row.user_agent">{{ row.user_agent }}</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="flex justify-end mt-3">
        <el-pagination
          v-model:current-page="logsPage"
          :page-size="logsPageSize"
          :total="logsTotal"
          layout="prev, pager, next"
          @current-change="loadLogs"
        />
      </div>
    </el-dialog>

    <!-- 调整分享配置弹窗 -->
    <el-dialog v-model="editDialogVisible" title="调整分享配置" width="420px">
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="当前次数">
          <span class="text-sm text-gray-600">已访问 {{ editTarget?.views }} / {{ editTarget?.max_views }} 次</span>
        </el-form-item>
        <el-form-item label="追加次数">
          <el-input-number v-model="editForm.addViews" :min="0" :max="9999" controls-position="right" class="w-full" />
          <div class="text-xs text-gray-400 mt-1">在现有最大次数基础上增加</div>
        </el-form-item>
        <el-form-item label="延期天数">
          <el-input-number v-model="editForm.addDays" :min="0" :max="3650" controls-position="right" class="w-full" />
          <div class="text-xs text-gray-400 mt-1">在现有到期日基础上延期</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editLoading" @click="submitEdit">确认更新</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Lock, FolderOpened, Folder, Document, Refresh
} from '@element-plus/icons-vue'

// -------- 状态 --------
const statusChecked = ref(false)
const nfsEnabled = ref(false)
const adminLoggedIn = ref(false)
const adminPassword = ref('')
const loginLoading = ref(false)
const activeTab = ref('browse')

// 目录浏览
const currentPath = ref('.')
const dirEntries = ref([])
const browseLoading = ref(false)

// 创建分享
const selectedFileSize = ref(0)
const createForm = reactive({
  filePath: '',
  name: '',
  maxViews: 5,
  expiresDays: 7
})
const createLoading = ref(false)

// 分享列表
const shareList = ref([])
const listPage = ref(1)
const listPageSize = 20
const listTotal = ref(0)
const listLoading = ref(false)

// 访问日志弹窗
const logsDialogVisible = ref(false)
const currentShare = ref(null)
const logList = ref([])
const logsPage = ref(1)
const logsPageSize = 50
const logsTotal = ref(0)
const logsLoading = ref(false)

// 挂载管理
const mountsList = ref([])
const mountsLoading = ref(false)
const mountActionLoading = ref('')

// 编辑弹窗
const editDialogVisible = ref(false)
const editTarget = ref(null)
const editForm = reactive({ addViews: 0, addDays: 0 })
const editLoading = ref(false)

// -------- 计算属性 --------
const breadcrumbs = computed(() => {
  const segs = [{ name: 'NFS 根目录', path: '.' }]
  if (currentPath.value && currentPath.value !== '.') {
    const parts = currentPath.value.split('/')
    let built = ''
    for (const p of parts) {
      built = built ? `${built}/${p}` : p
      segs.push({ name: p, path: built })
    }
  }
  return segs
})

// -------- 生命周期 --------
onMounted(async () => {
  await checkStatus()
  if (nfsEnabled.value) {
    const saved = sessionStorage.getItem('nfs_admin_password')
    if (saved) {
      adminPassword.value = saved
      await loginAdmin(true)
    }
  }
})

// -------- 状态检查 --------
async function checkStatus() {
  try {
    const res = await fetch('/api/nfsshare/status')
    const data = await res.json()
    nfsEnabled.value = !!data.enabled
  } catch {
    nfsEnabled.value = false
  } finally {
    statusChecked.value = true
  }
}

// -------- 超管登录 --------
async function loginAdmin(silent = false) {
  if (!adminPassword.value) {
    if (!silent) ElMessage.warning('请输入超管密码')
    return
  }
  loginLoading.value = true
  try {
    // 用浏览目录来验证密码
    const res = await fetch(`/api/nfsshare/admin/browse?path=.&admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (res.ok) {
      adminLoggedIn.value = true
      sessionStorage.setItem('nfs_admin_password', adminPassword.value)
      await Promise.all([loadDir('.'), loadShareList(), loadMounts()])
    } else {
      const data = await res.json()
      if (!silent) ElMessage.error(data.error || '密码错误')
      adminPassword.value = ''
      sessionStorage.removeItem('nfs_admin_password')
    }
  } catch {
    if (!silent) ElMessage.error('网络错误，请重试')
  } finally {
    loginLoading.value = false
  }
}

function logout() {
  adminLoggedIn.value = false
  adminPassword.value = ''
  sessionStorage.removeItem('nfs_admin_password')
  dirEntries.value = []
  shareList.value = []
  currentPath.value = '.'
}

// -------- 目录浏览 --------
async function loadDir(path) {
  browseLoading.value = true
  try {
    const res = await fetch(`/api/nfsshare/admin/browse?path=${encodeURIComponent(path)}&admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '无法读取目录')
      return
    }
    const data = await res.json()
    currentPath.value = path
    dirEntries.value = (data.entries || []).sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
  } catch {
    ElMessage.error('网络错误')
  } finally {
    browseLoading.value = false
  }
}

function handleEntryClick(row) {
  if (row.is_dir) {
    loadDir(row.path)
  }
}

function navigateTo(path) {
  loadDir(path)
}

function selectFile(row) {
  createForm.filePath = row.path
  createForm.name = row.name
  selectedFileSize.value = row.size
  ElMessage.success(`已选择：${row.name}`)
}

// -------- 创建分享 --------
async function createShare() {
  if (!createForm.filePath || !createForm.name) {
    ElMessage.warning('请先选择文件并填写名称')
    return
  }
  createLoading.value = true
  try {
    const res = await fetch('/api/nfsshare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPassword.value,
        name: createForm.name,
        file_path: createForm.filePath,
        max_views: createForm.maxViews,
        expires_days: createForm.expiresDays
      })
    })
    const data = await res.json()
    if (!res.ok) {
      ElMessage.error(data.error || '创建失败')
      return
    }
    const link = `${location.origin}/api/nfsshare/${data.id}`
    await navigator.clipboard.writeText(link).catch(() => {})
    ElMessage.success(`创建成功！链接已复制：${link}`)
    createForm.filePath = ''
    createForm.name = ''
    createForm.maxViews = 5
    createForm.expiresDays = 7
    selectedFileSize.value = 0
    // 切换到列表 tab 并刷新
    activeTab.value = 'list'
    await loadShareList()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    createLoading.value = false
  }
}

// -------- 挂载管理 --------
async function loadMounts() {
  mountsLoading.value = true
  try {
    const res = await fetch(`/api/nfsshare/admin/mounts?admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (!res.ok) return
    const data = await res.json()
    mountsList.value = data.mounts || []
  } catch {
    // ignore
  } finally {
    mountsLoading.value = false
  }
}

async function remount(name) {
  mountActionLoading.value = name + '_remount'
  try {
    const res = await fetch(
      `/api/nfsshare/admin/mounts/${name}/remount?admin_password=${encodeURIComponent(adminPassword.value)}`,
      { method: 'POST' }
    )
    const data = await res.json()
    if (res.ok) {
      ElMessage.success(`${name} 挂载成功`)
    } else {
      ElMessage.error(data.error || '挂载失败')
    }
    await loadMounts()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    mountActionLoading.value = ''
  }
}

async function umount(name) {
  try {
    await ElMessageBox.confirm(`确定卸载 ${name}？`, '确认卸载', { type: 'warning' })
  } catch { return }
  mountActionLoading.value = name + '_umount'
  try {
    const res = await fetch(
      `/api/nfsshare/admin/mounts/${name}/umount?admin_password=${encodeURIComponent(adminPassword.value)}`,
      { method: 'POST' }
    )
    const data = await res.json()
    if (res.ok) {
      ElMessage.success(`${name} 已卸载`)
    } else {
      ElMessage.error(data.error || '卸载失败')
    }
    await loadMounts()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    mountActionLoading.value = ''
  }
}

const configExample = `nfs_share:
  enabled: true
  admin_password: "your_super_admin_pass"
  max_file_size_mb: 0   # 0 = 不限制

  mounts:
    # NFS 示例（无需密码，服务端 /etc/exports 控制权限）
    - name: "project-files"
      type: nfs
      host: "192.168.1.100"
      export: "/exports/data"
      options: "soft,timeo=30"   # 可选

    # SMB/CIFS 示例（Windows 共享 / Samba）
    - name: "team-share"
      type: smb
      host: "192.168.1.200"
      share: "TeamDocs"
      username: "alice"
      password: "secret"
      domain: "CORP"     # 可选，域账号时填写

    # 本地目录示例（直接映射，不执行 mount）
    - name: "local-backup"
      type: local
      export: "/mnt/backup"`

// -------- 分享列表 --------
async function loadShareList() {
  listLoading.value = true
  try {
    const res = await fetch(
      `/api/nfsshare/admin/list?admin_password=${encodeURIComponent(adminPassword.value)}&page=${listPage.value}&page_size=${listPageSize}`
    )
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '获取列表失败')
      return
    }
    const data = await res.json()
    shareList.value = data.shares || []
    listTotal.value = data.total || 0
  } catch {
    ElMessage.error('网络错误')
  } finally {
    listLoading.value = false
  }
}

function copyShareLink(row) {
  const link = `${location.origin}/api/nfsshare/${row.id}`
  navigator.clipboard.writeText(link)
    .then(() => ElMessage.success('链接已复制'))
    .catch(() => ElMessage.info(`链接：${link}`))
}

async function deleteShare(row) {
  try {
    await ElMessageBox.confirm(`确定删除分享「${row.name}」及其所有访问记录？`, '确认删除', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  try {
    const res = await fetch(
      `/api/nfsshare/admin/${row.id}?admin_password=${encodeURIComponent(adminPassword.value)}`,
      { method: 'DELETE' }
    )
    if (res.ok) {
      ElMessage.success('删除成功')
      await loadShareList()
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '删除失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

// -------- 访问日志 --------
function viewLogs(row) {
  currentShare.value = row
  logsPage.value = 1
  logsDialogVisible.value = true
  loadLogs()
}

async function loadLogs() {
  if (!currentShare.value) return
  logsLoading.value = true
  try {
    const res = await fetch(
      `/api/nfsshare/admin/${currentShare.value.id}/logs?admin_password=${encodeURIComponent(adminPassword.value)}&page=${logsPage.value}&page_size=${logsPageSize}`
    )
    if (!res.ok) {
      const data = await res.json()
      ElMessage.error(data.error || '获取日志失败')
      return
    }
    const data = await res.json()
    logList.value = data.logs || []
    logsTotal.value = data.total || 0
  } catch {
    ElMessage.error('网络错误')
  } finally {
    logsLoading.value = false
  }
}

// -------- 编辑分享 --------
function openEditDialog(row) {
  editTarget.value = row
  editForm.addViews = 0
  editForm.addDays = 0
  editDialogVisible.value = true
}

async function submitEdit() {
  if (!editTarget.value) return
  editLoading.value = true
  try {
    const res = await fetch(`/api/nfsshare/admin/${editTarget.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPassword.value,
        add_views: editForm.addViews,
        add_days: editForm.addDays
      })
    })
    const data = await res.json()
    if (!res.ok) {
      ElMessage.error(data.error || '更新失败')
      return
    }
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    await loadShareList()
  } catch {
    ElMessage.error('网络错误')
  } finally {
    editLoading.value = false
  }
}

// -------- 工具函数 --------
function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatDate(dateStr) {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function isExpired(dateStr) {
  return dateStr && new Date(dateStr) < new Date()
}

function getShareStatus(row) {
  if (row.expires_at && isExpired(row.expires_at)) {
    return { type: 'danger', label: '已过期' }
  }
  if (row.max_views > 0 && row.views >= row.max_views) {
    return { type: 'warning', label: '已耗尽' }
  }
  return { type: 'success', label: '有效' }
}

function getLogStatusTag(status) {
  const map = {
    success: 'success',
    denied_views: 'warning',
    denied_expired: 'warning',
    file_missing: 'danger',
    not_found: 'info',
    error: 'danger'
  }
  return map[status] || 'info'
}

function getLogStatusLabel(status) {
  const map = {
    success: '成功',
    denied_views: '次数耗尽',
    denied_expired: '已过期',
    file_missing: '文件缺失',
    not_found: '不存在',
    error: '错误'
  }
  return map[status] || status
}
</script>

<style scoped>
.nfs-tabs :deep(.el-tabs__content) {
  overflow: visible;
}
</style>
