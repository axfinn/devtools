<template>
  <div class="photo-wall-page">
    <div v-if="!profileId" class="auth-grid">
      <el-card shadow="hover" class="auth-card">
        <template #header>
          <div class="card-title">创建照片墙档案</div>
        </template>
        <el-form label-position="top" @submit.prevent>
          <el-form-item label="档案标题">
            <el-input v-model="createForm.title" maxlength="50" placeholder="例如：宝宝成长 / 家庭相册" />
          </el-form-item>
          <el-form-item label="管理密码">
            <el-input v-model="createForm.password" type="password" show-password placeholder="至少 4 位" />
          </el-form-item>
          <el-form-item label="有效期">
            <el-select v-model="createForm.expiresIn" style="width: 100%">
              <el-option :value="30" label="30 天" />
              <el-option :value="90" label="90 天" />
              <el-option :value="180" label="180 天（普通用户上限）" />
              <el-option :value="-1" label="永久（超管）" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="createForm.expiresIn === -1" label="超管密码">
            <el-input v-model="createForm.adminPassword" type="password" show-password placeholder="永久档案需要超管密码" />
          </el-form-item>
          <el-button type="primary" class="full-btn" :loading="creating" @click="createProfile">创建档案</el-button>
        </el-form>
      </el-card>

      <el-card shadow="hover" class="auth-card">
        <template #header>
          <div class="card-title">登录已有档案</div>
        </template>
        <el-form label-position="top" @submit.prevent>
          <el-form-item label="管理密码">
            <el-input v-model="loginForm.password" type="password" show-password placeholder="输入创建时设置的密码" @keyup.enter="loginProfile" />
          </el-form-item>
          <el-button type="success" class="full-btn" :loading="loggingIn" @click="loginProfile">登录照片墙</el-button>
        </el-form>
        <div class="auth-tip">同一密码对应同一个档案，适合做长期分类照片墙。</div>
      </el-card>
    </div>

    <template v-else>
      <el-card shadow="never" class="summary-card">
        <div class="summary-head">
          <div>
            <div class="summary-kicker">档案照片墙</div>
            <h2>{{ profile.title || '未命名档案' }}</h2>
          </div>
          <div class="summary-actions">
            <el-button type="primary" @click="openShareDialog">分享</el-button>
            <el-button @click="showExtendDialog = true" :disabled="profile.is_permanent">延期</el-button>
            <el-button @click="downloadSelected" :disabled="selectedIds.length === 0">下载所选</el-button>
            <el-button @click="downloadAll" :disabled="profile.items.length === 0">打包全部</el-button>
            <el-button type="danger" plain @click="deleteProfile">删除档案</el-button>
            <el-button @click="logout">退出</el-button>
          </div>
        </div>

        <div class="summary-meta">
          <el-tag type="info">照片 {{ profile.items.length }}</el-tag>
          <el-tag v-if="profile.is_permanent" type="success">永久档案</el-tag>
          <el-tag v-else type="warning">到期 {{ formatDate(profile.expires_at) }}</el-tag>
          <el-tag v-if="profile.categories.length > 0">{{ profile.categories.length }} 个分类</el-tag>
        </div>

        <div class="summary-edit">
          <el-input v-model="renameTitle" maxlength="50" placeholder="修改档案标题" />
          <el-button @click="renameProfile" :loading="renaming">保存标题</el-button>
          <el-button @click="showAdminDialog = true">超管管理</el-button>
        </div>
      </el-card>

      <el-card shadow="hover" class="upload-card">
        <template #header>
          <div class="card-title">新增照片</div>
        </template>
        <div class="upload-grid">
          <el-input v-model="uploadMeta.category" placeholder="分类，例如：旅行 / 生日 / 证件" />
          <el-date-picker
            v-model="uploadMeta.takenAt"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            placeholder="拍摄时间，可选"
            style="width: 100%"
          />
          <el-input v-model="uploadMeta.description" placeholder="上传时附加说明，可选" />
        </div>
        <div class="upload-actions">
          <el-button type="primary" :loading="uploading" @click="pickFiles">选择照片</el-button>
          <el-button type="success" :loading="uploading" @click="takePhoto">直拍上传</el-button>
          <span class="upload-hint">支持一次多选，直拍适合手机端。</span>
        </div>
        <input ref="pickerRef" hidden type="file" accept="image/*" multiple @change="handlePick" />
        <input ref="cameraRef" hidden type="file" accept="image/*" capture="environment" @change="handlePick" />
      </el-card>

      <div class="toolbar">
        <div class="filter-row">
          <el-select v-model="activeCategory" style="width: 180px">
            <el-option label="全部分类" value="all" />
            <el-option v-for="category in profile.categories" :key="category" :label="category" :value="category" />
          </el-select>
          <el-select v-model="activeMonth" style="width: 180px">
            <el-option label="全部时间" value="all" />
            <el-option v-for="item in profile.timeline" :key="item.month" :label="`${item.month} (${item.count})`" :value="item.month" />
          </el-select>
          <el-input v-model="keyword" placeholder="搜索标题或说明" clearable />
        </div>
        <div class="selection-row">
          <el-checkbox :model-value="allVisibleSelected" @change="toggleSelectVisible">全选当前筛选结果</el-checkbox>
          <span>已选 {{ selectedIds.length }} 张</span>
        </div>
      </div>

      <div v-if="filteredItems.length === 0" class="empty-state">
        <el-empty description="当前没有匹配的照片" />
      </div>

      <div v-else class="photo-grid">
        <el-card v-for="item in filteredItems" :key="item.id" shadow="hover" class="photo-card">
          <template #header>
            <div class="photo-card-head">
              <el-checkbox :model-value="selectedIds.includes(item.id)" @change="toggleSelection(item.id)" />
              <div class="photo-card-actions">
                <el-button text @click="openEditDialog(item)">编辑</el-button>
                <el-button text type="danger" @click="deleteItem(item)">删除</el-button>
              </div>
            </div>
          </template>
          <img :src="item.image_url" :alt="item.title || profile.title" class="photo-image" @click="previewImage(item.image_url)" />
          <div class="photo-info">
            <div class="photo-title">{{ item.title || '未命名照片' }}</div>
            <div class="photo-meta">
              <el-tag v-if="item.category" size="small">{{ item.category }}</el-tag>
              <span>{{ formatDate(item.taken_at || item.created_at) }}</span>
            </div>
            <div v-if="item.description" class="photo-desc">{{ item.description }}</div>
          </div>
        </el-card>
      </div>
    </template>

    <el-dialog v-model="showEditDialog" title="编辑照片" width="480px">
      <el-form label-position="top">
        <el-form-item label="标题">
          <el-input v-model="editForm.title" maxlength="60" />
        </el-form-item>
        <el-form-item label="分类">
          <el-input v-model="editForm.category" maxlength="30" />
        </el-form-item>
        <el-form-item label="拍摄时间">
          <el-date-picker v-model="editForm.takenAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width: 100%" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="editForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingEdit" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showShareDialog" title="分享照片墙" width="560px">
      <el-alert type="info" :closable="false" style="margin-bottom: 16px">
        <template #title>如果当前浏览器没有保存历史分享密钥，系统会重新生成分享链接，旧链接将失效。</template>
      </el-alert>
      <el-form label-position="top">
        <el-form-item label="分享链接">
          <el-input :model-value="fullShareUrl" readonly>
            <template #append>
              <el-button @click="copyText(fullShareUrl)">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="profile.short_code" label="短链">
          <el-input :model-value="shortShareUrl" readonly>
            <template #append>
              <el-button @click="copyText(shortShareUrl)">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="regenerateShare" :loading="sharing">重新生成</el-button>
        <el-button type="primary" @click="showShareDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showExtendDialog" title="延期档案" width="360px">
      <el-form label-position="top">
        <el-form-item label="延期天数">
          <el-select v-model="extendDays" style="width: 100%">
            <el-option :value="30" label="30 天" />
            <el-option :value="90" label="90 天" />
            <el-option :value="180" label="180 天" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showExtendDialog = false">取消</el-button>
        <el-button type="primary" :loading="extending" @click="extendProfile">确认延期</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showAdminDialog" title="超管管理" width="860px">
      <div class="admin-login">
        <el-input v-model="adminPassword" type="password" show-password placeholder="输入超管密码" />
        <el-button type="primary" :loading="adminLoading" @click="loadAdminList">加载</el-button>
      </div>
      <el-table :data="adminProfiles" v-loading="adminLoading" max-height="420">
        <el-table-column prop="title" label="档案" min-width="160" />
        <el-table-column prop="item_count" label="照片数" width="90" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.is_permanent" type="success">永久</el-tag>
            <span v-else>{{ formatDate(row.expires_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="170">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewAdminProfile(row)">查看</el-button>
            <el-button size="small" type="success" @click="setPermanent(row)" :disabled="row.is_permanent">永久</el-button>
            <el-button size="small" type="warning" @click="extendAdmin(row)">延期</el-button>
            <el-button size="small" type="danger" @click="deleteAdmin(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="showAdminPreview" title="档案预览" width="860px">
      <div v-if="adminPreview" class="admin-preview">
        <div class="admin-preview-meta">
          <el-tag>{{ adminPreview.title }}</el-tag>
          <el-tag type="info">照片 {{ adminPreview.items.length }}</el-tag>
        </div>
        <div class="preview-grid">
          <div v-for="item in adminPreview.items" :key="item.id" class="preview-item">
            <img :src="item.image_url" :alt="item.title || adminPreview.title" />
            <div>{{ item.title || '未命名照片' }}</div>
          </div>
        </div>
      </div>
    </el-dialog>

    <el-image-viewer v-if="previewUrl" :url-list="[previewUrl]" @close="previewUrl = ''" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElImageViewer, ElMessage, ElMessageBox } from 'element-plus'
import { API_BASE } from '../api'

const STORAGE_KEY = 'photowall_session_v1'
const ADMIN_KEY = 'photowall_admin_password'

const createForm = ref({
  title: '我的照片墙',
  password: '',
  expiresIn: 90,
  adminPassword: ''
})
const loginForm = ref({ password: '' })
const profileId = ref('')
const password = ref('')
const creatorKey = ref('')
const accessKey = ref('')
const creating = ref(false)
const loggingIn = ref(false)
const uploading = ref(false)
const renaming = ref(false)
const savingEdit = ref(false)
const extending = ref(false)
const sharing = ref(false)
const profile = ref({
  title: '',
  items: [],
  categories: [],
  timeline: [],
  expires_at: null,
  short_code: '',
  is_permanent: false
})

const renameTitle = ref('')
const uploadMeta = ref({
  category: '',
  takenAt: '',
  description: ''
})
const activeCategory = ref('all')
const activeMonth = ref('all')
const keyword = ref('')
const selectedIds = ref([])
const pickerRef = ref(null)
const cameraRef = ref(null)
const previewUrl = ref('')

const showEditDialog = ref(false)
const showShareDialog = ref(false)
const showExtendDialog = ref(false)
const showAdminDialog = ref(false)
const showAdminPreview = ref(false)

const editTargetId = ref('')
const editForm = ref({
  title: '',
  category: '',
  takenAt: '',
  description: ''
})

const extendDays = ref(90)
const adminPassword = ref(sessionStorage.getItem(ADMIN_KEY) || '')
const adminLoading = ref(false)
const adminProfiles = ref([])
const adminPreview = ref(null)

const filteredItems = computed(() => {
  const keywordValue = keyword.value.trim().toLowerCase()
  return (profile.value.items || []).filter(item => {
    if (activeCategory.value !== 'all' && item.category !== activeCategory.value) return false
    if (activeMonth.value !== 'all' && formatMonth(item.taken_at || item.created_at) !== activeMonth.value) return false
    if (!keywordValue) return true
    return [item.title, item.description, item.category].some(text => (text || '').toLowerCase().includes(keywordValue))
  })
})

const allVisibleSelected = computed(() => filteredItems.value.length > 0 && filteredItems.value.every(item => selectedIds.value.includes(item.id)))
const fullShareUrl = computed(() => accessKey.value ? `${window.location.origin}/wall/${profileId.value}?key=${accessKey.value}` : '')
const shortShareUrl = computed(() => profile.value.short_code ? `${window.location.origin}/s/${profile.value.short_code}` : '')

onMounted(() => {
  restoreSession()
})

function restoreSession() {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    profileId.value = saved.profileId || ''
    password.value = saved.password || ''
    creatorKey.value = saved.creatorKey || ''
    accessKey.value = saved.accessKey || ''
    if (profileId.value && password.value) {
      loadProfile()
    }
  } catch {
    localStorage.removeItem(STORAGE_KEY)
  }
}

function persistSession() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({
    profileId: profileId.value,
    password: password.value,
    creatorKey: creatorKey.value,
    accessKey: accessKey.value
  }))
}

function clearSession() {
  profileId.value = ''
  password.value = ''
  creatorKey.value = ''
  accessKey.value = ''
  profile.value = { title: '', items: [], categories: [], timeline: [], expires_at: null, short_code: '', is_permanent: false }
  selectedIds.value = []
  localStorage.removeItem(STORAGE_KEY)
}

async function createProfile() {
  if (createForm.value.password.length < 4) {
    ElMessage.warning('密码至少 4 位')
    return
  }
  creating.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/profile`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(createForm.value)
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '创建失败')

    profileId.value = data.id
    password.value = createForm.value.password
    creatorKey.value = data.creator_key
    accessKey.value = data.access_key
    persistSession()
    await loadProfile()
    showShareDialog.value = true
    ElMessage.success('照片墙档案已创建')
  } catch (error) {
    ElMessage.error(error.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function loginProfile() {
  if (!loginForm.value.password) {
    ElMessage.warning('请输入密码')
    return
  }
  loggingIn.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/profile/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(loginForm.value)
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '登录失败')
    profileId.value = data.id
    password.value = loginForm.value.password
    persistSession()
    await loadProfile()
    ElMessage.success('登录成功')
  } catch (error) {
    ElMessage.error(error.message || '登录失败')
  } finally {
    loggingIn.value = false
  }
}

async function loadProfile() {
  if (!profileId.value || !password.value) return
  try {
    const params = new URLSearchParams({ password: password.value })
    if (creatorKey.value) {
      params.set('creator_key', creatorKey.value)
    }
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}?${params.toString()}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    profile.value = data
    renameTitle.value = data.title || ''
    selectedIds.value = selectedIds.value.filter(id => data.items.some(item => item.id === id))
    persistSession()
  } catch (error) {
    clearSession()
    ElMessage.error(error.message || '加载失败')
  }
}

function pickFiles() {
  pickerRef.value?.click()
}

function takePhoto() {
  cameraRef.value?.click()
}

async function handlePick(event) {
  const files = Array.from(event.target.files || [])
  if (files.length === 0) return
  uploading.value = true
  try {
    for (const file of files) {
      const form = new FormData()
      form.append('file', file)
      form.append('category', uploadMeta.value.category || '')
      form.append('description', uploadMeta.value.description || '')
      form.append('taken_at', uploadMeta.value.takenAt || '')
      form.append('title', file.name.replace(/\.[^.]+$/, ''))
      if (creatorKey.value) {
        form.append('creator_key', creatorKey.value)
      }
      form.append('password', password.value)

      const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}/items`, {
        method: 'POST',
        body: form
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || `${file.name} 上传失败`)
    }
    await loadProfile()
    ElMessage.success(`已上传 ${files.length} 张照片`)
  } catch (error) {
    ElMessage.error(error.message || '上传失败')
  } finally {
    uploading.value = false
    event.target.value = ''
  }
}

function toggleSelection(id) {
  if (selectedIds.value.includes(id)) {
    selectedIds.value = selectedIds.value.filter(item => item !== id)
  } else {
    selectedIds.value = [...selectedIds.value, id]
  }
}

function toggleSelectVisible(checked) {
  if (checked) {
    const visibleIds = filteredItems.value.map(item => item.id)
    selectedIds.value = Array.from(new Set([...selectedIds.value, ...visibleIds]))
  } else {
    const visibleIds = new Set(filteredItems.value.map(item => item.id))
    selectedIds.value = selectedIds.value.filter(id => !visibleIds.has(id))
  }
}

async function renameProfile() {
  if (!renameTitle.value.trim()) {
    ElMessage.warning('标题不能为空')
    return
  }
  renaming.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'rename',
        title: renameTitle.value.trim(),
        creator_key: creatorKey.value,
        password: password.value
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '更新失败')
    profile.value.title = renameTitle.value.trim()
    ElMessage.success('标题已更新')
  } catch (error) {
    ElMessage.error(error.message || '更新失败')
  } finally {
    renaming.value = false
  }
}

async function openShareDialog() {
  if (!accessKey.value) {
    await regenerateShare()
  }
  showShareDialog.value = true
}

async function regenerateShare() {
  sharing.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'reshare',
        creator_key: creatorKey.value,
        password: password.value
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成分享失败')
    const key = new URL(data.share_url, window.location.origin).searchParams.get('key') || ''
    accessKey.value = key
    profile.value.short_code = data.short_code || ''
    persistSession()
    ElMessage.success('分享链接已更新')
  } catch (error) {
    ElMessage.error(error.message || '生成分享失败')
  } finally {
    sharing.value = false
  }
}

async function extendProfile() {
  extending.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'extend',
        expires_in: extendDays.value,
        creator_key: creatorKey.value,
        password: password.value
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '延期失败')
    profile.value.expires_at = data.expires_at
    showExtendDialog.value = false
    ElMessage.success('档案已延期')
  } catch (error) {
    ElMessage.error(error.message || '延期失败')
  } finally {
    extending.value = false
  }
}

async function deleteProfile() {
  try {
    await ElMessageBox.confirm('确定删除整个照片墙档案？此操作不可恢复。', '删除确认', { type: 'warning' })
    const query = new URLSearchParams()
    if (creatorKey.value) query.set('creator_key', creatorKey.value)
    query.set('password', password.value)
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}?${query.toString()}`, { method: 'DELETE' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '删除失败')
    clearSession()
    ElMessage.success('档案已删除')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

function downloadSelected() {
  if (selectedIds.value.length === 0) return
  const query = new URLSearchParams({ password: password.value, item_ids: selectedIds.value.join(',') })
  if (creatorKey.value) query.set('creator_key', creatorKey.value)
  window.open(`${API_BASE}/api/photowall/profile/${profileId.value}/download?${query.toString()}`, '_blank')
}

function downloadAll() {
  const query = new URLSearchParams({ password: password.value })
  if (creatorKey.value) query.set('creator_key', creatorKey.value)
  window.open(`${API_BASE}/api/photowall/profile/${profileId.value}/download?${query.toString()}`, '_blank')
}

function openEditDialog(item) {
  editTargetId.value = item.id
  editForm.value = {
    title: item.title || '',
    category: item.category || '',
    takenAt: item.taken_at || '',
    description: item.description || ''
  }
  showEditDialog.value = true
}

async function saveEdit() {
  savingEdit.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}/items/${editTargetId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: editForm.value.title,
        category: editForm.value.category,
        taken_at: editForm.value.takenAt,
        description: editForm.value.description,
        creator_key: creatorKey.value,
        password: password.value
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '保存失败')
    await loadProfile()
    showEditDialog.value = false
    ElMessage.success('照片已更新')
  } catch (error) {
    ElMessage.error(error.message || '保存失败')
  } finally {
    savingEdit.value = false
  }
}

async function deleteItem(item) {
  try {
    await ElMessageBox.confirm(`确定删除照片“${item.title || '未命名照片'}”？`, '删除确认', { type: 'warning' })
    const query = new URLSearchParams({ password: password.value })
    if (creatorKey.value) query.set('creator_key', creatorKey.value)
    const res = await fetch(`${API_BASE}/api/photowall/profile/${profileId.value}/items/${item.id}?${query.toString()}`, {
      method: 'DELETE'
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '删除失败')
    await loadProfile()
    ElMessage.success('照片已删除')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

async function loadAdminList() {
  if (!adminPassword.value) {
    ElMessage.warning('请输入超管密码')
    return
  }
  adminLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/admin/list?admin_password=${encodeURIComponent(adminPassword.value)}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '加载失败')
    adminProfiles.value = data.profiles || []
    sessionStorage.setItem(ADMIN_KEY, adminPassword.value)
  } catch (error) {
    ElMessage.error(error.message || '加载失败')
  } finally {
    adminLoading.value = false
  }
}

async function viewAdminProfile(row) {
  adminLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/photowall/admin/${row.id}?admin_password=${encodeURIComponent(adminPassword.value)}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '查看失败')
    adminPreview.value = data
    showAdminPreview.value = true
  } catch (error) {
    ElMessage.error(error.message || '查看失败')
  } finally {
    adminLoading.value = false
  }
}

async function setPermanent(row) {
  try {
    const res = await fetch(`${API_BASE}/api/photowall/admin/${row.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'set_permanent', admin_password: adminPassword.value })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '设置失败')
    await loadAdminList()
    if (row.id === profileId.value) {
      await loadProfile()
    }
    ElMessage.success('已设为永久档案')
  } catch (error) {
    ElMessage.error(error.message || '设置失败')
  }
}

async function extendAdmin(row) {
  try {
    const { value } = await ElMessageBox.prompt('输入延期天数', '超管延期', {
      inputValue: '180',
      inputPattern: /^[1-9]\d*$/,
      inputErrorMessage: '请输入正整数'
    })
    const res = await fetch(`${API_BASE}/api/photowall/admin/${row.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'extend', expires_in: Number(value), admin_password: adminPassword.value })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '延期失败')
    await loadAdminList()
    if (row.id === profileId.value) {
      await loadProfile()
    }
    ElMessage.success('档案已延期')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '延期失败')
    }
  }
}

async function deleteAdmin(row) {
  try {
    await ElMessageBox.confirm(`确定删除档案“${row.title}”？`, '超管删除', { type: 'warning' })
    const res = await fetch(`${API_BASE}/api/photowall/admin/${row.id}?admin_password=${encodeURIComponent(adminPassword.value)}`, {
      method: 'DELETE'
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '删除失败')
    await loadAdminList()
    if (row.id === profileId.value) {
      clearSession()
    }
    ElMessage.success('档案已删除')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

function previewImage(url) {
  previewUrl.value = url
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

function logout() {
  clearSession()
  loginForm.value.password = ''
}

function formatDate(value) {
  if (!value) return '未设置'
  return new Date(value).toLocaleString('zh-CN')
}

function formatMonth(value) {
  return new Date(value).toISOString().slice(0, 7)
}
</script>

<style scoped>
.photo-wall-page {
  max-width: 1280px;
  margin: 0 auto;
  padding: 20px;
}

.auth-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.auth-card,
.summary-card,
.upload-card {
  border-radius: 20px;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
}

.auth-tip,
.upload-hint {
  margin-top: 12px;
  font-size: 13px;
  color: var(--text-secondary);
}

.full-btn {
  width: 100%;
}

.summary-head {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  align-items: flex-start;
}

.summary-kicker {
  font-size: 12px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.summary-head h2 {
  margin: 6px 0 0;
  font-size: 28px;
}

.summary-actions,
.summary-meta,
.summary-edit,
.upload-actions,
.filter-row,
.selection-row,
.admin-login,
.admin-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.summary-meta {
  margin-top: 16px;
}

.summary-edit {
  margin-top: 18px;
}

.summary-edit .el-input {
  max-width: 320px;
}

.upload-card {
  margin-top: 18px;
}

.upload-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.upload-actions {
  margin-top: 16px;
}

.toolbar {
  margin: 20px 0 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-row {
  display: grid;
  grid-template-columns: 180px 180px minmax(0, 1fr);
}

.selection-row {
  justify-content: space-between;
}

.empty-state {
  margin-top: 40px;
  padding: 40px 0;
  background: var(--bg-overlay);
  border-radius: 18px;
}

.photo-grid,
.preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

.photo-card {
  border-radius: 18px;
  overflow: hidden;
}

.photo-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.photo-card-actions {
  display: flex;
  gap: 8px;
}

.photo-image {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  border-radius: 12px;
  cursor: zoom-in;
}

.photo-info {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.photo-title {
  font-size: 16px;
  font-weight: 700;
}

.photo-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.photo-desc {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.preview-item img {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  border-radius: 12px;
  margin-bottom: 8px;
}

@media (max-width: 900px) {
  .auth-grid,
  .upload-grid,
  .filter-row {
    grid-template-columns: 1fr;
  }

  .summary-head {
    flex-direction: column;
  }
}
</style>
