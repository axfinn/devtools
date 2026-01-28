<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Excalidraw 画图</h2>
      <div class="actions">
        <el-dropdown @command="handleExport" trigger="click">
          <el-button type="primary">
            <el-icon><Download /></el-icon>
            导出
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="png">导出 PNG</el-dropdown-item>
              <el-dropdown-item command="svg">导出 SVG</el-dropdown-item>
              <el-dropdown-item command="json">导出 JSON</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <el-dropdown @command="handleSave" trigger="click">
          <el-button type="success">
            <el-icon><FolderAdd /></el-icon>
            保存
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="local">保存到本地</el-dropdown-item>
              <el-dropdown-item command="cloud">保存到云端</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <el-button @click="showImportDialog = true">
          <el-icon><Upload /></el-icon>
          导入
        </el-button>

        <el-button v-if="hasLocalOrCloudDrawings" type="warning" @click="showMyDrawings = true">
          <el-icon><Folder /></el-icon>
          我的画图
        </el-button>

        <el-button v-if="isAdmin" type="danger" @click="openAdminPanel">
          <el-icon><Setting /></el-icon>
          管理
        </el-button>
        <el-button v-else @click="showAdminLogin = true">
          <el-icon><Setting /></el-icon>
          管理
        </el-button>

        <el-button @click="resetCanvas">
          <el-icon><Refresh /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <div class="editor-wrapper">
      <ExcalidrawWrapper
        ref="excalidrawRef"
        :initial-data="initialData"
        :theme="currentTheme"
        @change="onSceneChange"
        @ready="onExcalidrawReady"
      />
    </div>

    <!-- Local Save Dialog -->
    <el-dialog v-model="showLocalSaveDialog" title="保存到本地" width="400px">
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="localSaveForm.title" placeholder="画图名称" maxlength="50" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showLocalSaveDialog = false">取消</el-button>
        <el-button type="primary" @click="saveToLocal">保存</el-button>
      </template>
    </el-dialog>

    <!-- Cloud Save Dialog -->
    <el-dialog v-model="showCloudSaveDialog" title="保存到云端" width="450px" :close-on-click-modal="false">
      <el-form label-width="100px" class="cloud-save-form">
        <el-form-item label="标题">
          <el-input v-model="cloudSaveForm.title" placeholder="可选" maxlength="50" show-word-limit />
        </el-form-item>
        <el-form-item label="访问密码" required>
          <el-input v-model="cloudSaveForm.password" type="password" placeholder="至少4个字符" show-password />
        </el-form-item>
        <el-form-item label="有效期">
          <el-select v-model="cloudSaveForm.expiresIn" style="width: 100%">
            <el-option v-for="opt in expiresOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="cloudSaveForm.expiresIn === -1" label="管理员密码" required>
          <el-input v-model="cloudSaveForm.adminPassword" type="password" placeholder="永久保存需要管理员密码" show-password />
        </el-form-item>
      </el-form>
      <el-alert type="info" :closable="false">
        <template #title>访问密码用于保护画图内容，分享时需提供给他人</template>
      </el-alert>
      <template #footer>
        <el-button @click="showCloudSaveDialog = false">取消</el-button>
        <el-button type="primary" @click="saveToCloud" :loading="cloudSaveLoading">保存到云端</el-button>
      </template>
    </el-dialog>

    <!-- Cloud Save Result Dialog -->
    <el-dialog v-model="showCloudSaveResult" title="保存成功" width="500px" :close-on-click-modal="false">
      <el-result icon="success" title="画图已保存到云端">
        <template #sub-title>
          <p v-if="cloudSaveResult.isPermanent">永久保存</p>
          <p v-else>{{ cloudSaveResult.expiresDays }} 天后过期</p>
        </template>
      </el-result>
      <el-form label-width="80px" class="result-form">
        <el-form-item label="短链接" v-if="cloudSaveResult.shortUrl">
          <el-input :model-value="cloudSaveResult.shortUrl" readonly>
            <template #append>
              <el-button @click="copyToClipboard(cloudSaveResult.shortUrl, '短链接')">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="完整链接">
          <el-input :model-value="cloudSaveResult.fullUrl" readonly>
            <template #append>
              <el-button @click="copyToClipboard(cloudSaveResult.fullUrl, '完整链接')">复制</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <el-alert type="warning" :closable="false" style="margin-top: 15px;">
        <template #title>管理密钥已自动保存到浏览器</template>
        <div style="font-size: 12px;">用此浏览器可管理画图（延期、编辑、删除）</div>
      </el-alert>
      <template #footer>
        <el-button @click="showCloudSaveResult = false">关闭</el-button>
        <el-button type="primary" @click="copyAndCloseResult">复制链接并关闭</el-button>
      </template>
    </el-dialog>

    <!-- Import Dialog -->
    <el-dialog v-model="showImportDialog" title="导入画图" width="450px">
      <el-tabs v-model="importTab">
        <el-tab-pane label="本地文件" name="file">
          <el-upload
            class="upload-area"
            drag
            :auto-upload="false"
            :show-file-list="false"
            accept=".excalidraw,.json"
            @change="handleFileImport"
          >
            <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
            <div class="el-upload__text">
              拖拽文件到此处，或 <em>点击上传</em>
            </div>
            <template #tip>
              <div class="el-upload__tip">支持 .excalidraw 或 .json 文件</div>
            </template>
          </el-upload>
        </el-tab-pane>
        <el-tab-pane label="云端链接" name="cloud">
          <el-form label-width="80px">
            <el-form-item label="画图 ID">
              <el-input v-model="importCloudForm.id" placeholder="输入画图 ID 或短链码" />
            </el-form-item>
            <el-form-item label="访问密码">
              <el-input v-model="importCloudForm.password" type="password" placeholder="输入访问密码" show-password />
            </el-form-item>
          </el-form>
          <div style="text-align: right; margin-top: 16px;">
            <el-button type="primary" @click="importFromCloud" :loading="importLoading">导入</el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <!-- My Drawings Dialog -->
    <el-dialog v-model="showMyDrawings" title="我的画图" width="800px">
      <el-tabs v-model="myDrawingsTab">
        <el-tab-pane label="本地画图" name="local">
          <el-table :data="localDrawings" empty-text="暂无本地画图" max-height="350">
            <el-table-column prop="title" label="名称" min-width="150">
              <template #default="{ row }">{{ row.title || '未命名' }}</template>
            </el-table-column>
            <el-table-column label="更新时间" width="180">
              <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="loadLocalDrawing(row)">加载</el-button>
                <el-button size="small" type="danger" @click="deleteLocalDrawing(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="云端画图" name="cloud">
          <el-table :data="cloudDrawings" v-loading="loadingCloudDrawings" empty-text="暂无云端画图" max-height="350">
            <el-table-column prop="title" label="名称" min-width="150">
              <template #default="{ row }">{{ row.title || '未命名' }}</template>
            </el-table-column>
            <el-table-column label="过期时间" width="180">
              <template #default="{ row }">
                <el-tag v-if="row.is_permanent" type="success" size="small">永久</el-tag>
                <span v-else :class="{ 'text-danger': isExpiringSoon(row.expires_at) }">
                  {{ formatDate(row.expires_at) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="loadCloudDrawing(row)">加载</el-button>
                <el-button v-if="!row.is_permanent" size="small" type="warning" @click="extendDrawing(row)">延期</el-button>
                <el-button size="small" type="danger" @click="deleteCloudDrawing(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <!-- Extend Dialog -->
    <el-dialog v-model="showExtendDialog" title="延期画图" width="400px">
      <el-form label-width="100px">
        <el-form-item label="延期天数">
          <el-select v-model="extendForm.days" style="width: 100%">
            <el-option :value="7" label="7 天" />
            <el-option :value="30" label="30 天" />
            <el-option :value="90" label="90 天" />
            <el-option :value="180" label="180 天" />
            <el-option :value="365" label="365 天" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showExtendDialog = false">取消</el-button>
        <el-button type="primary" @click="doExtend" :loading="extendLoading">确认延期</el-button>
      </template>
    </el-dialog>

    <!-- Admin Login Dialog -->
    <el-dialog v-model="showAdminLogin" title="管理员登录" width="350px" :close-on-click-modal="false">
      <el-form @submit.prevent="verifyAdminPassword">
        <el-form-item>
          <el-input
            v-model="adminPassword"
            type="password"
            placeholder="请输入管理员密码"
            show-password
            @keyup.enter="verifyAdminPassword"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdminLogin = false">取消</el-button>
        <el-button type="primary" @click="verifyAdminPassword" :loading="adminLoading">确认</el-button>
      </template>
    </el-dialog>

    <!-- Admin Panel Dialog -->
    <el-dialog v-model="showAdminPanel" title="画图管理" width="900px" :close-on-click-modal="false">
      <div class="admin-toolbar">
        <el-input
          v-model="adminSearchKeyword"
          placeholder="搜索标题或 ID"
          clearable
          style="width: 250px"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button @click="loadAllDrawings" :loading="loadingAllDrawings">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <span class="drawing-count">共 {{ filteredAllDrawings.length }} 条</span>
      </div>

      <el-table
        :data="paginatedDrawings"
        v-loading="loadingAllDrawings"
        empty-text="暂无画图"
        max-height="400"
        stripe
      >
        <el-table-column prop="id" label="ID" width="100">
          <template #default="{ row }">
            <el-tooltip :content="row.id" placement="top">
              <span class="id-cell">{{ row.id }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="150">
          <template #default="{ row }">{{ row.title || '(无标题)' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_permanent" type="success" size="small">永久</el-tag>
            <el-tag v-else type="info" size="small">有效</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="过期时间" width="160">
          <template #default="{ row }">
            <span v-if="row.is_permanent">-</span>
            <span v-else :class="{ 'text-danger': isExpiringSoon(row.expires_at) }">
              {{ formatDate(row.expires_at) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="loadAdminDrawing(row)">加载</el-button>
            <el-button v-if="!row.is_permanent" size="small" type="success" @click="setPermanent(row)">永久</el-button>
            <el-button size="small" type="danger" @click="deleteAdminDrawing(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="admin-pagination">
        <el-pagination
          v-model:current-page="adminCurrentPage"
          :page-size="adminPageSize"
          :total="filteredAllDrawings.length"
          layout="prev, pager, next"
          :hide-on-single-page="true"
        />
      </div>

      <template #footer>
        <el-button @click="logoutAdmin" type="warning">退出登录</el-button>
        <el-button @click="showAdminPanel = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ExcalidrawWrapper from '../components/ExcalidrawWrapper.vue'
import { useTheme } from '../composables/useTheme'

const excalidrawRef = ref(null)
const { currentTheme } = useTheme()
const initialData = ref(null)

const onExcalidrawReady = (api) => {
  // Excalidraw is ready
}

const onSceneChange = ({ elements, appState, files }) => {
  // Auto-save could be implemented here
}

// Export functions
const handleExport = async (command) => {
  if (!excalidrawRef.value) return

  try {
    switch (command) {
      case 'png': {
        const blob = await excalidrawRef.value.exportToPng()
        if (blob) {
          downloadBlob(blob, 'excalidraw.png')
          ElMessage.success('PNG 已下载')
        }
        break
      }
      case 'svg': {
        const svgString = await excalidrawRef.value.exportToSvgString()
        if (svgString) {
          const blob = new Blob([svgString], { type: 'image/svg+xml' })
          downloadBlob(blob, 'excalidraw.svg')
          ElMessage.success('SVG 已下载')
        }
        break
      }
      case 'json': {
        const json = excalidrawRef.value.getSceneJSON()
        if (json) {
          const blob = new Blob([json], { type: 'application/json' })
          downloadBlob(blob, 'excalidraw.json')
          ElMessage.success('JSON 已下载')
        }
        break
      }
    }
  } catch (err) {
    ElMessage.error('导出失败: ' + err.message)
  }
}

const downloadBlob = (blob, filename) => {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

// Save functions
const handleSave = (command) => {
  if (command === 'local') {
    showLocalSaveDialog.value = true
  } else if (command === 'cloud') {
    showCloudSaveDialog.value = true
  }
}

// Local save
const showLocalSaveDialog = ref(false)
const localSaveForm = ref({ title: '' })

const saveToLocal = () => {
  if (!excalidrawRef.value) return

  const json = excalidrawRef.value.getSceneJSON()
  if (!json) {
    ElMessage.error('获取画图数据失败')
    return
  }

  const id = 'local_' + Date.now()
  const drawings = getLocalDrawings()
  drawings[id] = {
    title: localSaveForm.value.title || '未命名',
    content: json,
    updated_at: new Date().toISOString()
  }
  localStorage.setItem('excalidraw_local_drawings', JSON.stringify(drawings))

  showLocalSaveDialog.value = false
  localSaveForm.value.title = ''
  ElMessage.success('已保存到本地')
}

const getLocalDrawings = () => {
  try {
    return JSON.parse(localStorage.getItem('excalidraw_local_drawings') || '{}')
  } catch {
    return {}
  }
}

// Cloud save
const showCloudSaveDialog = ref(false)
const showCloudSaveResult = ref(false)
const cloudSaveLoading = ref(false)
const cloudSaveForm = ref({
  title: '',
  password: '',
  expiresIn: 30,
  adminPassword: ''
})
const cloudSaveResult = ref({
  shortUrl: '',
  fullUrl: '',
  expiresDays: 30,
  isPermanent: false
})

const expiresOptions = [
  { value: 7, label: '7 天' },
  { value: 30, label: '30 天' },
  { value: 90, label: '90 天' },
  { value: 180, label: '180 天' },
  { value: 365, label: '1 年' },
  { value: -1, label: '永久保存 (需管理员密码)' }
]

const saveToCloud = async () => {
  if (!excalidrawRef.value) return

  if (!cloudSaveForm.value.password || cloudSaveForm.value.password.length < 4) {
    ElMessage.warning('密码至少需要4个字符')
    return
  }

  if (cloudSaveForm.value.expiresIn === -1 && !cloudSaveForm.value.adminPassword) {
    ElMessage.warning('永久保存需要管理员密码')
    return
  }

  const json = excalidrawRef.value.getSceneJSON()
  if (!json) {
    ElMessage.error('获取画图数据失败')
    return
  }

  cloudSaveLoading.value = true
  try {
    const res = await fetch('/api/excalidraw', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: json,
        title: cloudSaveForm.value.title,
        password: cloudSaveForm.value.password,
        expires_in: cloudSaveForm.value.expiresIn,
        admin_password: cloudSaveForm.value.adminPassword
      })
    })
    const data = await res.json()

    if (!res.ok) {
      ElMessage.error(data.error || '保存失败')
      return
    }

    // Save creator key and password
    saveCreatorKey(data.id, data.creator_key, {
      title: cloudSaveForm.value.title,
      expires_at: data.expires_at,
      is_permanent: data.is_permanent,
      password: cloudSaveForm.value.password // Save password for easier management
    })

    // Show result
    const baseUrl = window.location.origin
    cloudSaveResult.value = {
      shortUrl: data.share_url ? baseUrl + data.share_url : '',
      fullUrl: `${baseUrl}/draw/${data.id}`,
      expiresDays: cloudSaveForm.value.expiresIn,
      isPermanent: data.is_permanent
    }

    showCloudSaveDialog.value = false
    showCloudSaveResult.value = true

    // Reset form
    cloudSaveForm.value = { title: '', password: '', expiresIn: 30, adminPassword: '' }
  } catch {
    ElMessage.error('网络错误')
  } finally {
    cloudSaveLoading.value = false
  }
}

const copyAndCloseResult = () => {
  const url = cloudSaveResult.value.shortUrl || cloudSaveResult.value.fullUrl
  copyToClipboard(url, '链接')
  showCloudSaveResult.value = false
}

// Creator keys management
const getCreatorKeys = () => {
  try {
    return JSON.parse(localStorage.getItem('excalidraw_creator_keys') || '{}')
  } catch {
    return {}
  }
}

const saveCreatorKey = (id, key, meta = {}) => {
  const keys = getCreatorKeys()
  keys[id] = { key, ...meta, created_at: new Date().toISOString() }
  localStorage.setItem('excalidraw_creator_keys', JSON.stringify(keys))
}

const removeCreatorKey = (id) => {
  const keys = getCreatorKeys()
  delete keys[id]
  localStorage.setItem('excalidraw_creator_keys', JSON.stringify(keys))
}

// Import
const showImportDialog = ref(false)
const importTab = ref('file')
const importLoading = ref(false)
const importCloudForm = ref({ id: '', password: '' })

const handleFileImport = (file) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const data = JSON.parse(e.target.result)
      if (excalidrawRef.value) {
        excalidrawRef.value.loadScene(data)
        showImportDialog.value = false
        ElMessage.success('导入成功')

        // Auto scroll to content after loading
        setTimeout(() => {
          if (excalidrawRef.value) {
            excalidrawRef.value.scrollToContent({ delay: 300 })
          }
        }, 100)
      }
    } catch (err) {
      ElMessage.error('文件格式错误')
    }
  }
  reader.readAsText(file.raw)
}

const importFromCloud = async () => {
  if (!importCloudForm.value.id || !importCloudForm.value.password) {
    ElMessage.warning('请输入画图 ID 和密码')
    return
  }

  importLoading.value = true
  try {
    const id = importCloudForm.value.id
    const res = await fetch(`/api/excalidraw/${id}?password=${encodeURIComponent(importCloudForm.value.password)}`)
    const data = await res.json()

    if (!res.ok) {
      ElMessage.error(data.error || '导入失败')
      return
    }

    if (excalidrawRef.value && data.content) {
      const sceneData = JSON.parse(data.content)
      excalidrawRef.value.loadScene(sceneData)
      showImportDialog.value = false
      importCloudForm.value = { id: '', password: '' }
      ElMessage.success('导入成功')

      // Auto scroll to content after loading
      setTimeout(() => {
        if (excalidrawRef.value) {
          excalidrawRef.value.scrollToContent({ delay: 300 })
        }
      }, 100)
    }
  } catch {
    ElMessage.error('网络错误')
  } finally {
    importLoading.value = false
  }
}

// My Drawings
const showMyDrawings = ref(false)
const myDrawingsTab = ref('local')
const loadingCloudDrawings = ref(false)
const localDrawings = ref([])
const cloudDrawings = ref([])

const hasLocalOrCloudDrawings = computed(() => {
  const localKeys = Object.keys(getLocalDrawings())
  const cloudKeys = Object.keys(getCreatorKeys())
  return localKeys.length > 0 || cloudKeys.length > 0
})

watch(showMyDrawings, async (val) => {
  if (val) {
    loadLocalDrawingsList()
    await loadCloudDrawingsList()
  }
})

const loadLocalDrawingsList = () => {
  const drawings = getLocalDrawings()
  localDrawings.value = Object.entries(drawings).map(([id, data]) => ({
    id,
    ...data
  })).sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at))
}

const loadCloudDrawingsList = async () => {
  loadingCloudDrawings.value = true
  const keys = getCreatorKeys()
  const drawings = []

  for (const [id, data] of Object.entries(keys)) {
    try {
      const res = await fetch(`/api/excalidraw/${id}/creator?creator_key=${data.key}`)
      if (res.ok) {
        const drawing = await res.json()
        drawings.push({ ...drawing, creator_key: data.key })
      } else if (res.status === 404 || res.status === 410) {
        removeCreatorKey(id)
      }
    } catch {
      // Network error, skip
    }
  }

  cloudDrawings.value = drawings.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
  loadingCloudDrawings.value = false
}

const loadLocalDrawing = (drawing) => {
  try {
    const data = JSON.parse(drawing.content)
    if (excalidrawRef.value) {
      excalidrawRef.value.loadScene(data)
      showMyDrawings.value = false
      ElMessage.success('已加载')

      // Auto scroll to content after loading
      setTimeout(() => {
        if (excalidrawRef.value) {
          excalidrawRef.value.scrollToContent({ delay: 300 })
        }
      }, 100)
    }
  } catch {
    ElMessage.error('加载失败')
  }
}

const deleteLocalDrawing = async (drawing) => {
  try {
    await ElMessageBox.confirm('确定删除此本地画图？', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  const drawings = getLocalDrawings()
  delete drawings[drawing.id]
  localStorage.setItem('excalidraw_local_drawings', JSON.stringify(drawings))
  loadLocalDrawingsList()
  ElMessage.success('已删除')
}

const loadCloudDrawing = async (drawing) => {
  try {
    const res = await fetch(`/api/excalidraw/${drawing.id}/creator?creator_key=${drawing.creator_key}`)
    if (res.ok) {
      const data = await res.json()
      if (excalidrawRef.value && data.content) {
        const sceneData = JSON.parse(data.content)
        excalidrawRef.value.loadScene(sceneData)
        showMyDrawings.value = false
        ElMessage.success('已加载')

        // Auto scroll to content after loading
        setTimeout(() => {
          if (excalidrawRef.value) {
            excalidrawRef.value.scrollToContent({ delay: 300 })
          }
        }, 100)
      }
    } else {
      ElMessage.error('加载失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

const deleteCloudDrawing = async (drawing) => {
  try {
    await ElMessageBox.confirm('确定删除此云端画图？删除后无法恢复。', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  try {
    const res = await fetch(`/api/excalidraw/${drawing.id}?creator_key=${drawing.creator_key}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      removeCreatorKey(drawing.id)
      cloudDrawings.value = cloudDrawings.value.filter(d => d.id !== drawing.id)
      ElMessage.success('删除成功')
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '删除失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

// Extend
const showExtendDialog = ref(false)
const extendLoading = ref(false)
const extendForm = ref({ days: 30 })
const currentExtendDrawing = ref(null)

const extendDrawing = (drawing) => {
  currentExtendDrawing.value = drawing
  extendForm.value.days = 30
  showExtendDialog.value = true
}

const doExtend = async () => {
  if (!currentExtendDrawing.value) return

  extendLoading.value = true
  try {
    const res = await fetch(`/api/excalidraw/${currentExtendDrawing.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'extend',
        expires_in: extendForm.value.days,
        creator_key: currentExtendDrawing.value.creator_key
      })
    })
    const data = await res.json()

    if (res.ok) {
      showExtendDialog.value = false
      await loadCloudDrawingsList()
      ElMessage.success('延期成功')
    } else {
      ElMessage.error(data.error || '延期失败')
    }
  } catch {
    ElMessage.error('网络错误')
  } finally {
    extendLoading.value = false
  }
}

// Admin
const isAdmin = ref(false)
const adminPassword = ref('')
const showAdminLogin = ref(false)
const showAdminPanel = ref(false)
const adminLoading = ref(false)
const loadingAllDrawings = ref(false)
const allDrawings = ref([])
const adminSearchKeyword = ref('')
const adminCurrentPage = ref(1)
const adminPageSize = 10

onMounted(() => {
  const savedPwd = sessionStorage.getItem('excalidraw_admin_pwd')
  if (savedPwd) {
    isAdmin.value = true
  }
})

const filteredAllDrawings = computed(() => {
  if (!adminSearchKeyword.value) return allDrawings.value
  const keyword = adminSearchKeyword.value.toLowerCase()
  return allDrawings.value.filter(d =>
    d.id.toLowerCase().includes(keyword) ||
    (d.title && d.title.toLowerCase().includes(keyword))
  )
})

const paginatedDrawings = computed(() => {
  const start = (adminCurrentPage.value - 1) * adminPageSize
  return filteredAllDrawings.value.slice(start, start + adminPageSize)
})

const verifyAdminPassword = async () => {
  if (!adminPassword.value.trim()) {
    ElMessage.warning('请输入密码')
    return
  }

  adminLoading.value = true
  try {
    const res = await fetch(`/api/excalidraw/admin/list?admin_password=${encodeURIComponent(adminPassword.value)}`)
    if (res.ok) {
      sessionStorage.setItem('excalidraw_admin_pwd', adminPassword.value)
      isAdmin.value = true
      showAdminLogin.value = false
      const data = await res.json()
      allDrawings.value = data.list || []
      showAdminPanel.value = true
      ElMessage.success('登录成功')
    } else {
      ElMessage.error('密码错误')
    }
  } catch {
    ElMessage.error('网络错误')
  } finally {
    adminLoading.value = false
  }
}

const openAdminPanel = async () => {
  showAdminPanel.value = true
  await loadAllDrawings()
}

const loadAllDrawings = async () => {
  const pwd = sessionStorage.getItem('excalidraw_admin_pwd')
  if (!pwd) {
    isAdmin.value = false
    showAdminPanel.value = false
    showAdminLogin.value = true
    return
  }

  loadingAllDrawings.value = true
  try {
    const res = await fetch(`/api/excalidraw/admin/list?admin_password=${encodeURIComponent(pwd)}`)
    if (res.ok) {
      const data = await res.json()
      allDrawings.value = data.list || []
    } else if (res.status === 401 || res.status === 403) {
      sessionStorage.removeItem('excalidraw_admin_pwd')
      isAdmin.value = false
      showAdminPanel.value = false
      ElMessage.error('密码已失效，请重新登录')
      showAdminLogin.value = true
    }
  } catch {
    ElMessage.error('加载失败')
  } finally {
    loadingAllDrawings.value = false
  }
}

const logoutAdmin = () => {
  sessionStorage.removeItem('excalidraw_admin_pwd')
  isAdmin.value = false
  showAdminPanel.value = false
  adminPassword.value = ''
  allDrawings.value = []
  ElMessage.success('已退出管理')
}

const loadAdminDrawing = async (drawing) => {
  const pwd = sessionStorage.getItem('excalidraw_admin_pwd')
  try {
    const res = await fetch(`/api/excalidraw/admin/${drawing.id}?admin_password=${encodeURIComponent(pwd)}`)
    if (res.ok) {
      const data = await res.json()
      if (excalidrawRef.value && data.content) {
        const sceneData = JSON.parse(data.content)
        excalidrawRef.value.loadScene(sceneData)
        showAdminPanel.value = false
        ElMessage.success('已加载')

        // Auto scroll to content after loading
        setTimeout(() => {
          if (excalidrawRef.value) {
            excalidrawRef.value.scrollToContent({ delay: 300 })
          }
        }, 100)
      }
    } else {
      ElMessage.error('加载失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

const setPermanent = async (drawing) => {
  try {
    await ElMessageBox.confirm('确定设为永久保存？', '确认', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      type: 'info'
    })
  } catch {
    return
  }

  const pwd = sessionStorage.getItem('excalidraw_admin_pwd')
  try {
    const res = await fetch(`/api/excalidraw/${drawing.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        action: 'set_permanent',
        admin_password: pwd
      })
    })
    if (res.ok) {
      await loadAllDrawings()
      ElMessage.success('已设为永久保存')
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '操作失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

const deleteAdminDrawing = async (drawing) => {
  try {
    await ElMessageBox.confirm(`确定要删除 "${drawing.title || drawing.id}" 吗？`, '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  const pwd = sessionStorage.getItem('excalidraw_admin_pwd')
  try {
    const res = await fetch(`/api/excalidraw/admin/${drawing.id}?admin_password=${encodeURIComponent(pwd)}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      allDrawings.value = allDrawings.value.filter(d => d.id !== drawing.id)
      ElMessage.success('删除成功')
    } else {
      const data = await res.json()
      ElMessage.error(data.error || '删除失败')
    }
  } catch {
    ElMessage.error('网络错误')
  }
}

// Reset canvas
const resetCanvas = async () => {
  try {
    await ElMessageBox.confirm('确定要重置画布吗？未保存的内容将丢失。', '确认重置', {
      confirmButtonText: '重置',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  if (excalidrawRef.value) {
    excalidrawRef.value.resetScene()
    ElMessage.success('画布已重置')
  }
}

// Utils
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

const isExpiringSoon = (dateStr) => {
  if (!dateStr) return false
  const expires = new Date(dateStr)
  const now = new Date()
  const daysLeft = (expires - now) / (1000 * 60 * 60 * 24)
  return daysLeft <= 3
}

const copyToClipboard = async (text, name) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${name}已复制`)
  } catch {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding-bottom: 15px;
  flex-shrink: 0;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.editor-wrapper {
  flex: 1;
  min-height: 0;
  border-radius: 8px;
  overflow: hidden;
  background: #121212;
}

.result-form {
  margin-top: 20px;
}

.upload-area {
  width: 100%;
}

.admin-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.drawing-count {
  color: #909399;
  font-size: 13px;
  margin-left: auto;
}

.id-cell {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: #409eff;
  cursor: pointer;
}

.text-danger {
  color: #f56c6c;
}

.admin-pagination {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .tool-container {
    height: auto;
  }

  .editor-wrapper {
    height: 500px;
  }
}
</style>
