<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>共享粘贴板</h2>
      <div class="header-right">
        <div class="info-text">
          支持文本、图片、视频分享 - 自动压缩优化
        </div>
        <el-button size="small" @click="showMyShares = true">
          <el-icon><FolderOpened /></el-icon>
          我的分享
        </el-button>
        <el-button size="small" @click="showAdminPanel = true">
          <el-icon><Lock /></el-icon>
          管理
        </el-button>
      </div>
    </div>

    <!-- 简洁模式 -->
    <div class="quick-section" v-if="!showAdvanced">
      <div
        class="quick-editor"
        :class="{ 'is-dragging': isDragging }"
        @dragenter="onDragEnter"
        @dragleave="onDragLeave"
        @dragover="onDragOver"
        @drop="onDrop"
      >
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="粘贴或输入内容,支持拖拽图片/视频..."
          spellcheck="false"
        ></textarea>
        <div v-if="isDragging" class="drop-overlay">
          <el-icon :size="48"><Upload /></el-icon>
          <span>拖放文件到此处</span>
        </div>
      </div>

      <!-- 文件上传区域 -->
      <div class="file-section">
        <div class="file-header">
          <span class="file-title">
            <el-icon><Folder /></el-icon>
            文件 ({{ files.length }}/{{ MAX_FILES }})
          </span>
          <span class="size-info" v-if="files.length > 0">
            总大小: {{ (totalSize / 1024 / 1024).toFixed(2) }} MB
          </span>
        </div>

        <div class="file-grid" v-if="files.length > 0">
          <div class="file-item" v-for="(file, index) in files" :key="index">
            <div class="file-preview">
              <img v-if="file.type === 'image'" :src="file.preview" alt="预览" />
              <video v-else-if="file.type === 'video'" :src="file.preview" controls></video>
              <audio v-else-if="file.type === 'audio'" :src="file.preview" controls></audio>
              <div v-else class="file-icon">
                <el-icon :size="48">
                  <Document v-if="file.type === 'document'" />
                  <Folder v-else-if="file.type === 'archive'" />
                  <Files v-else />
                </el-icon>
                <span class="file-ext">{{ getFileExt(file.name) }}</span>
              </div>
            </div>
            <div class="file-info">
              <span class="file-name">{{ file.name }}</span>
              <span class="file-size">{{ (file.size / 1024 / 1024).toFixed(2) }} MB</span>
              <el-tag v-if="file.compressed" type="success" size="small">已压缩</el-tag>
              <el-tag v-if="file.compressing" type="warning" size="small">压缩中...</el-tag>
              <el-tag v-if="file.uploading" type="info" size="small">上传中 {{ file.uploadProgress }}%</el-tag>
            </div>
            <div class="file-actions">
              <el-button v-if="!file.compressed && canCompress(file)" size="small" @click="compressFile(index)" :loading="file.compressing">
                压缩
              </el-button>
              <el-button type="danger" size="small" @click="removeFile(index)" :disabled="file.uploading">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-add" v-if="canAddMore" @click="selectFiles">
            <el-icon :size="24"><Plus /></el-icon>
            <span>添加文件</span>
          </div>
        </div>

        <div class="file-upload" v-else @click="selectFiles">
          <el-icon :size="32"><Upload /></el-icon>
          <span>点击上传文件或直接拖拽</span>
          <span class="upload-hint">支持图片、视频、音频、PDF、Office文档、压缩包等 (最大 200MB)</span>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip,.rar,.7z,.mov"
          multiple
          style="display: none"
          @change="onFileChange"
        />
      </div>

      <div class="quick-actions">
        <el-button type="primary" size="large" @click="quickCreate" :loading="creating" :disabled="!content.trim() && files.length === 0">
          <el-icon><Share /></el-icon>
          一键分享
        </el-button>
        <el-button size="small" text @click="showAdvanced = true">
          高级选项
        </el-button>
      </div>
    </div>

    <!-- 高级模式 -->
    <div class="create-section" v-else>
      <div
        class="editor-panel"
        :class="{ 'is-dragging': isDragging }"
        @dragenter="onDragEnter"
        @dragleave="onDragLeave"
        @dragover="onDragOver"
        @drop="onDrop"
      >
        <div class="panel-header">
          <el-input
            v-model="title"
            placeholder="标题（可选）"
            style="width: 200px"
            size="small"
          />
          <el-select v-model="language" placeholder="语言" style="width: 120px" size="small">
            <el-option label="纯文本" value="text" />
            <el-option label="JSON" value="json" />
            <el-option label="JavaScript" value="javascript" />
            <el-option label="Python" value="python" />
            <el-option label="Go" value="go" />
            <el-option label="Markdown" value="markdown" />
          </el-select>
          <el-button size="small" text @click="showAdvanced = false">简洁模式</el-button>
        </div>
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="在此输入要分享的内容..."
          spellcheck="false"
        ></textarea>
        <div v-if="isDragging" class="drop-overlay">
          <el-icon :size="48"><Upload /></el-icon>
          <span>拖放文件到此处</span>
        </div>
      </div>

      <!-- 高级模式文件区域 (同上) -->
      <div class="file-section">
        <div class="file-header">
          <span class="file-title">
            <el-icon><Folder /></el-icon>
            文件 ({{ files.length }}/{{ MAX_FILES }})
          </span>
          <span class="size-info" v-if="files.length > 0">
            总大小: {{ (totalSize / 1024 / 1024).toFixed(2) }} MB
          </span>
        </div>

        <div class="file-grid" v-if="files.length > 0">
          <div class="file-item" v-for="(file, index) in files" :key="index">
            <div class="file-preview">
              <img v-if="file.type === 'image'" :src="file.preview" alt="预览" />
              <video v-else-if="file.type === 'video'" :src="file.preview" controls></video>
              <audio v-else-if="file.type === 'audio'" :src="file.preview" controls></audio>
              <div v-else class="file-icon">
                <el-icon :size="48">
                  <Document v-if="file.type === 'document'" />
                  <Folder v-else-if="file.type === 'archive'" />
                  <Files v-else />
                </el-icon>
                <span class="file-ext">{{ getFileExt(file.name) }}</span>
              </div>
            </div>
            <div class="file-info">
              <span class="file-name">{{ file.name }}</span>
              <span class="file-size">{{ (file.size / 1024 / 1024).toFixed(2) }} MB</span>
              <el-tag v-if="file.compressed" type="success" size="small">已压缩</el-tag>
              <el-tag v-if="file.compressing" type="warning" size="small">压缩中...</el-tag>
              <el-tag v-if="file.uploading" type="info" size="small">上传中 {{ file.uploadProgress }}%</el-tag>
            </div>
            <div class="file-actions">
              <el-button v-if="!file.compressed && canCompress(file)" size="small" @click="compressFile(index)" :loading="file.compressing">
                压缩
              </el-button>
              <el-button type="danger" size="small" @click="removeFile(index)" :disabled="file.uploading">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
          <div class="file-add" v-if="canAddMore" @click="selectFiles">
            <el-icon :size="24"><Plus /></el-icon>
            <span>添加文件</span>
          </div>
        </div>

        <div class="file-upload" v-else @click="selectFiles">
          <el-icon :size="32"><Upload /></el-icon>
          <span>点击上传文件或直接拖拽</span>
          <span class="upload-hint">支持图片、视频、音频、PDF、Office文档、压缩包等 (最大 200MB)</span>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip,.rar,.7z,.mov"
          multiple
          style="display: none"
          @change="onFileChange"
        />
      </div>

      <div class="options-row">
        <div class="option-item">
          <span class="option-label">过期时间</span>
          <el-select v-model="expiresIn" style="width: 120px">
            <el-option label="1 小时" :value="1" />
            <el-option label="6 小时" :value="6" />
            <el-option label="24 小时" :value="24" />
            <el-option label="3 天" :value="72" />
            <el-option label="7 天" :value="168" />
          </el-select>
        </div>
        <div class="option-item">
          <span class="option-label">最大访问次数</span>
          <el-input-number v-model="maxViews" :min="1" :max="hasVideo ? 10 : 1000" />
          <span v-if="hasVideo" class="hint-text">(视频默认限制10次)</span>
        </div>
        <div class="option-item">
          <span class="option-label">访问密码</span>
          <el-input
            v-model="password"
            type="password"
            placeholder="可选"
            style="width: 150px"
            show-password
          />
        </div>
        <div class="option-item" v-if="hasVideo">
          <span class="option-label">管理员密码</span>
          <el-input
            v-model="adminPassword"
            type="password"
            placeholder="可设置更多次数"
            style="width: 150px"
            show-password
          />
        </div>
        <el-button type="primary" size="large" @click="createPaste" :loading="creating" :disabled="!content.trim() && files.length === 0">
          创建分享
        </el-button>
      </div>
    </div>

    <!-- 分享结果 -->
    <div v-if="showResult" class="result-section">
      <div class="result-card">
        <div class="result-header">
          <el-icon class="success-icon"><CircleCheck /></el-icon>
          <span>分享创建成功！链接已复制</span>
        </div>

        <div class="share-url-box">
          <div class="url-display">{{ shareUrl }}</div>
          <div class="url-actions">
            <el-button type="primary" @click="copyUrl">
              <el-icon><CopyDocument /></el-icon>
              复制链接
            </el-button>
            <el-button @click="openShare">
              <el-icon><Link /></el-icon>
              打开
            </el-button>
          </div>
        </div>

        <div class="qr-section">
          <div class="qr-title">扫码访问</div>
          <canvas ref="qrCanvas" class="qr-code"></canvas>
        </div>

        <div class="result-info">
          <span>ID: {{ createdId }}</span>
          <span>过期: {{ createdExpires }}</span>
          <span>最大访问: {{ createdMaxViews }} 次</span>
          <span v-if="password">密码: {{ password }}</span>
        </div>

        <el-button class="new-share-btn" @click="resetForm" type="success" plain>
          <el-icon><Plus /></el-icon>
          创建新分享
        </el-button>
      </div>
    </div>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon :closable="false" />
    </div>

    <div class="tips-section" v-if="!showResult">
      <h4>使用提示</h4>
      <ul>
        <li>支持文本、图片、视频、音频、PDF、Office文档、压缩包等多种格式</li>
        <li>大文件自动分片上传,支持断点续传</li>
        <li>视频默认最多10次访问(防止滥用)</li>
        <li>管理员可设置更多访问次数或永久访问</li>
        <li>所有文件最大支持200MB</li>
      </ul>
    </div>

    <!-- 支持项目 Footer -->
    <div class="project-footer">
      <div class="footer-content">
        <div class="footer-section">
          <h4>💖 支持项目</h4>
          <p>如果这个工具对你有帮助，欢迎通过以下方式支持项目发展</p>
        </div>
        <div class="footer-donate">
          <div class="donate-item">
            <span class="donate-label">⭐ GitHub Star</span>
            <a href="https://github.com" target="_blank" class="donate-link">给项目加星</a>
          </div>
          <div class="donate-item">
            <span class="donate-label">💰 赞赏支持</span>
            <el-button size="small" @click="showDonateDialog = true">查看二维码</el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 赞赏二维码弹窗 -->
    <el-dialog v-model="showDonateDialog" title="赞赏支持" width="500px" center>
      <div class="donate-dialog">
        <div class="donate-qr-group">
          <div class="donate-qr-item">
            <img src="/alipay.jpeg" alt="支付宝" />
            <span>支付宝</span>
          </div>
          <div class="donate-qr-item">
            <img src="/wxpay.jpeg" alt="微信支付" />
            <span>微信支付</span>
          </div>
        </div>
        <p class="donate-thanks">感谢您的支持，这将帮助我们持续改进项目！</p>
      </div>
    </el-dialog>

    <!-- 我的分享 -->
    <el-dialog v-model="showMyShares" title="我的分享" width="90%" :close-on-click-modal="false">
      <div v-if="mySharesList.length === 0" style="text-align: center; padding: 40px; color: var(--text-secondary);">
        <el-icon :size="64"><FolderOpened /></el-icon>
        <p style="margin-top: 20px;">暂无分享记录</p>
      </div>
      <el-table v-else :data="mySharesList" style="width: 100%" max-height="500">
        <el-table-column prop="id" label="ID" width="100">
          <template #default="{ row }">
            <el-button link @click="openMyShare(row.id)">{{ row.id }}</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" width="150">
          <template #default="{ row }">
            {{ row.title || '(无标题)' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTimestamp(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="expires_at" label="过期时间" width="180">
          <template #default="{ row }">
            {{ formatTimestamp(row.expires_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="views" label="访问次数" width="120">
          <template #default="{ row }">
            {{ row.views }} / {{ row.max_views }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="copyMyShareUrl(row.id)">
              <el-icon><CopyDocument /></el-icon>
              复制链接
            </el-button>
            <el-button link type="danger" size="small" @click="removeMyShare(row.id)">
              <el-icon><Delete /></el-icon>
              移除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 管理员面板 -->
    <el-dialog v-model="showAdminPanel" title="管理员面板" width="90%" :close-on-click-modal="false">
      <div v-if="!adminAuthenticated">
        <el-form @submit.prevent="adminLogin">
          <el-form-item label="管理员密码">
            <el-input v-model="adminPasswordInput" type="password" show-password placeholder="请输入管理员密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="adminLogin" :loading="adminLoading">登录</el-button>
          </el-form-item>
        </el-form>
      </div>
      <div v-else>
        <el-button type="primary" @click="loadAdminPastes" :loading="adminLoading" style="margin-bottom: 15px">
          <el-icon><Refresh /></el-icon>
          刷新列表
        </el-button>
        <el-table :data="adminPastes" style="width: 100%" max-height="500">
          <el-table-column prop="id" label="ID" width="100" />
          <el-table-column prop="title" label="标题" width="150">
            <template #default="{ row }">
              {{ row.title || '(无标题)' }}
            </template>
          </el-table-column>
          <el-table-column label="内容" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.has_content" type="success" size="small">有文本</el-tag>
              <el-tag v-if="row.file_count > 0" type="warning" size="small">{{ row.file_count }} 文件</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="views" label="访问" width="80">
            <template #default="{ row }">
              {{ row.views }}/{{ row.max_views }}
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              {{ new Date(row.created_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column prop="expires_at" label="过期时间" width="180">
            <template #default="{ row }">
              {{ new Date(row.expires_at).toLocaleString('zh-CN') }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="viewAdminPaste(row.id)">查看</el-button>
              <el-button size="small" type="primary" @click="editAdminPaste(row)">编辑</el-button>
              <el-button size="small" type="danger" @click="deleteAdminPaste(row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 管理员编辑弹窗 -->
    <el-dialog v-model="showEditDialog" title="编辑粘贴板" width="500px">
      <el-form label-width="120px">
        <el-form-item label="ID">
          <el-input v-model="editingPaste.id" disabled />
        </el-form-item>
        <el-form-item label="延长时间">
          <el-select v-model="editExpiresIn" placeholder="选择延长时间">
            <el-option label="1 小时" :value="1" />
            <el-option label="6 小时" :value="6" />
            <el-option label="24 小时" :value="24" />
            <el-option label="3 天" :value="72" />
            <el-option label="7 天" :value="168" />
            <el-option label="30 天" :value="720" />
          </el-select>
        </el-form-item>
        <el-form-item label="最大访问次数">
          <el-input-number v-model="editMaxViews" :min="1" :max="999999" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="saveAdminEdit" :loading="adminLoading">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, nextTick, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Share, CircleCheck, CopyDocument, Link, Plus, Folder, FolderOpened, Delete, Upload, Lock, Refresh, Document, Files } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import { API_BASE } from '../api'
import { FFmpeg } from '@ffmpeg/ffmpeg'
import { fetchFile, toBlobURL } from '@ffmpeg/util'

const content = ref('')
const title = ref('')
const language = ref('text')
const expiresIn = ref(24)
const maxViews = ref(0)
const password = ref('')
const adminPassword = ref('')
const creating = ref(false)
const showResult = ref(false)
const showAdvanced = ref(false)
const createdId = ref('')
const createdExpires = ref('')
const createdMaxViews = ref(0)
const shareUrl = ref('')
const errorMsg = ref('')
const qrCanvas = ref(null)
const files = ref([]) // [{ file: File, preview: string, type: 'image'|'video'|'audio'|'document'|'archive'|'file', name: string, size: number, compressed: boolean, compressing: boolean, uploadedId: string, uploading: boolean, uploadProgress: number }]
const fileInput = ref(null)
const isDragging = ref(false)
const showDonateDialog = ref(false)

const MAX_FILES = 10
const MAX_FILE_SIZE = 200 * 1024 * 1024 // 200MB
const CHUNK_SIZE = 2 * 1024 * 1024 // 2MB per chunk for chunked upload

// FFmpeg 实例 (懒加载)
let ffmpegInstance = null
let ffmpegLoaded = false

// 我的分享功能
const showMyShares = ref(false)
const MY_SHARES_KEY = 'paste_my_shares'

// 从localStorage读取我的分享列表
const mySharesList = computed(() => {
  try {
    const stored = localStorage.getItem(MY_SHARES_KEY)
    if (!stored) return []
    const shares = JSON.parse(stored)
    // 过滤过期的分享
    const now = Date.now()
    return shares.filter(share => new Date(share.expires_at).getTime() > now)
  } catch (e) {
    return []
  }
})

// 保存分享到localStorage
const saveMyShare = (shareData) => {
  try {
    const stored = localStorage.getItem(MY_SHARES_KEY)
    const shares = stored ? JSON.parse(stored) : []
    shares.unshift(shareData) // 添加到开头
    // 只保留最近100条
    if (shares.length > 100) {
      shares.splice(100)
    }
    localStorage.setItem(MY_SHARES_KEY, JSON.stringify(shares))
  } catch (e) {
    console.error('保存分享失败:', e)
  }
}

// 打开我的分享
const openMyShare = (id) => {
  window.open(`/paste/${id}`, '_blank')
}

// 复制我的分享链接
const copyMyShareUrl = async (id) => {
  const url = `${window.location.origin}/paste/${id}`
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('链接已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

// 从列表移除分享（不删除服务器数据）
const removeMyShare = (id) => {
  ElMessageBox.confirm('确定从列表中移除此分享？(不会删除服务器数据)', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    try {
      const stored = localStorage.getItem(MY_SHARES_KEY)
      if (stored) {
        const shares = JSON.parse(stored)
        const filtered = shares.filter(s => s.id !== id)
        localStorage.setItem(MY_SHARES_KEY, JSON.stringify(filtered))
        ElMessage.success('已移除')
      }
    } catch (e) {
      ElMessage.error('移除失败')
    }
  }).catch(() => {})
}

// 格式化时间戳
const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString('zh-CN')
}

// 管理员功能
const showAdminPanel = ref(false)
const adminAuthenticated = ref(false)
const adminPasswordInput = ref('')
const adminPasswordStored = ref('')
const adminLoading = ref(false)
const adminPastes = ref([])
const showEditDialog = ref(false)
const editingPaste = ref({})
const editExpiresIn = ref(24)
const editMaxViews = ref(100)

const totalSize = computed(() => {
  return files.value.reduce((sum, file) => sum + file.size, 0)
})

const canAddMore = computed(() => files.value.length < MAX_FILES)

const hasVideo = computed(() => files.value.some(f => f.type === 'video'))

// 快捷创建
const quickCreate = async () => {
  await createPaste()
}

// 拖拽处理
const onDragEnter = (e) => {
  e.preventDefault()
  isDragging.value = true
}

const onDragLeave = (e) => {
  e.preventDefault()
  isDragging.value = false
}

const onDragOver = (e) => {
  e.preventDefault()
}

const onDrop = async (e) => {
  e.preventDefault()
  isDragging.value = false
  const droppedFiles = e.dataTransfer?.files
  if (droppedFiles) {
    for (const file of droppedFiles) {
      await addFile(file)
    }
  }
}

// 选择文件
const selectFiles = () => {
  fileInput.value?.click()
}

// 文件选择变化
const onFileChange = async (e) => {
  const selectedFiles = e.target.files
  for (const file of selectedFiles) {
    await addFile(file)
  }
  e.target.value = ''
}

// 添加文件
const addFile = async (file) => {
  if (files.value.length >= MAX_FILES) {
    ElMessage.warning(`最多只能上传 ${MAX_FILES} 个文件`)
    return
  }

  if (file.size > MAX_FILE_SIZE) {
    ElMessage.warning(`文件 ${file.name} 超过 200MB 限制`)
    return
  }

  // 检测文件类型
  let fileType = 'file'
  if (file.type.startsWith('image/')) {
    fileType = 'image'
  } else if (file.type.startsWith('video/')) {
    fileType = 'video'
  } else if (file.type.startsWith('audio/')) {
    fileType = 'audio'
  } else if (file.type === 'application/pdf' ||
             file.type.includes('document') ||
             file.type.includes('word') ||
             file.type.includes('excel') ||
             file.type.includes('powerpoint') ||
             file.type.includes('openxmlformats')) {
    fileType = 'document'
  } else if (file.type.includes('zip') ||
             file.type.includes('rar') ||
             file.type.includes('7z') ||
             file.type.includes('compressed')) {
    fileType = 'archive'
  }

  // 创建预览 (仅限图片、视频、音频)
  let preview = null
  if (fileType === 'image' || fileType === 'video' || fileType === 'audio') {
    preview = URL.createObjectURL(file)
  }

  files.value.push({
    file,
    preview,
    type: fileType,
    name: file.name,
    size: file.size,
    compressed: false,
    compressing: false,
    uploadedId: null,
    uploading: false,
    uploadProgress: 0
  })
}

// 获取文件扩展名
const getFileExt = (filename) => {
  const ext = filename.split('.').pop()
  return ext ? `.${ext.toUpperCase()}` : ''
}

// 删除文件
const removeFile = (index) => {
  const file = files.value[index]
  if (file.preview) {
    URL.revokeObjectURL(file.preview)
  }
  files.value.splice(index, 1)
}

// 是否可以压缩
const canCompress = (file) => {
  // 图片大于 1MB 或视频大于 10MB 可以压缩
  if (file.type === 'image' && file.size > 1024 * 1024) {
    return true
  }
  if (file.type === 'video' && file.size > 10 * 1024 * 1024) {
    return true
  }
  return false
}

// 压缩文件
const compressFile = async (index) => {
  const fileObj = files.value[index]
  if (fileObj.compressing) return

  fileObj.compressing = true

  try {
    if (fileObj.type === 'image') {
      await compressImage(index)
    } else if (fileObj.type === 'video') {
      await compressVideo(index)
    }
  } catch (err) {
    ElMessage.error(`压缩失败: ${err.message}`)
    fileObj.compressing = false
  }
}

// 压缩图片 (Canvas)
const compressImage = async (index) => {
  const fileObj = files.value[index]

  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      const canvas = document.createElement('canvas')
      let width = img.width
      let height = img.height

      // 限制最大尺寸为 1920x1080
      const maxWidth = 1920
      const maxHeight = 1080

      if (width > maxWidth || height > maxHeight) {
        const ratio = Math.min(maxWidth / width, maxHeight / height)
        width *= ratio
        height *= ratio
      }

      canvas.width = width
      canvas.height = height

      const ctx = canvas.getContext('2d')
      ctx.drawImage(img, 0, 0, width, height)

      canvas.toBlob(
        (blob) => {
          if (!blob) {
            reject(new Error('压缩失败'))
            return
          }

          const compressedFile = new File([blob], fileObj.name, { type: 'image/jpeg' })

          // 更新文件对象
          URL.revokeObjectURL(fileObj.preview)
          fileObj.file = compressedFile
          fileObj.preview = URL.createObjectURL(compressedFile)
          fileObj.size = compressedFile.size
          fileObj.compressed = true
          fileObj.compressing = false

          ElMessage.success(`图片已压缩: ${(fileObj.size / 1024 / 1024).toFixed(2)} MB`)
          resolve()
        },
        'image/jpeg',
        0.8 // 质量 80%
      )
    }

    img.onerror = () => {
      reject(new Error('图片加载失败'))
    }

    img.src = fileObj.preview
  })
}

// 初始化 FFmpeg
const initFFmpeg = async () => {
  if (ffmpegLoaded) return ffmpegInstance

  try {
    const ffmpeg = new FFmpeg()

    // 加载 FFmpeg 核心
    const baseURL = 'https://unpkg.com/@ffmpeg/core@0.12.6/dist/umd'
    await ffmpeg.load({
      coreURL: await toBlobURL(`${baseURL}/ffmpeg-core.js`, 'text/javascript'),
      wasmURL: await toBlobURL(`${baseURL}/ffmpeg-core.wasm`, 'application/wasm'),
    })

    ffmpegInstance = ffmpeg
    ffmpegLoaded = true
    return ffmpeg
  } catch (err) {
    console.error('FFmpeg 加载失败:', err)
    throw new Error('FFmpeg 初始化失败')
  }
}

// 压缩视频 (FFmpeg.wasm)
const compressVideo = async (index) => {
  const fileObj = files.value[index]

  try {
    ElMessage.info('正在初始化视频压缩工具...')
    const ffmpeg = await initFFmpeg()

    ElMessage.info('正在压缩视频，请稍候...')

    // 读取文件
    await ffmpeg.writeFile('input.mp4', await fetchFile(fileObj.file))

    // 压缩视频: 降低分辨率和比特率
    await ffmpeg.exec([
      '-i', 'input.mp4',
      '-vf', 'scale=-2:720', // 720p
      '-b:v', '1M', // 1 Mbps
      '-c:v', 'libx264',
      '-preset', 'fast',
      '-c:a', 'aac',
      '-b:a', '128k',
      'output.mp4'
    ])

    // 读取输出
    const data = await ffmpeg.readFile('output.mp4')
    const compressedBlob = new Blob([data.buffer], { type: 'video/mp4' })
    const compressedFile = new File([compressedBlob], fileObj.name.replace(/\.[^.]+$/, '.mp4'), { type: 'video/mp4' })

    // 清理 FFmpeg 文件
    await ffmpeg.deleteFile('input.mp4')
    await ffmpeg.deleteFile('output.mp4')

    // 更新文件对象
    URL.revokeObjectURL(fileObj.preview)
    fileObj.file = compressedFile
    fileObj.preview = URL.createObjectURL(compressedFile)
    fileObj.size = compressedFile.size
    fileObj.compressed = true
    fileObj.compressing = false

    ElMessage.success(`视频已压缩: ${(fileObj.size / 1024 / 1024).toFixed(2)} MB`)
  } catch (err) {
    console.error('视频压缩失败:', err)
    fileObj.compressing = false
    throw err
  }
}

// 上传文件到服务器 (支持分片上传)
const uploadFile = async (fileObj) => {
  const file = fileObj.file

  // 小于10MB的文件直接上传
  if (file.size < 10 * 1024 * 1024) {
    return await uploadFileDirectly(fileObj)
  }

  // 大文件使用分片上传
  return await uploadFileInChunks(fileObj)
}

// 直接上传小文件
const uploadFileDirectly = async (fileObj) => {
  const formData = new FormData()
  formData.append('file', fileObj.file)

  try {
    fileObj.uploading = true
    fileObj.uploadProgress = 0

    const response = await fetch(`${API_BASE}/api/paste/upload`, {
      method: 'POST',
      body: formData
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '上传失败')
    }

    fileObj.uploadProgress = 100
    fileObj.uploading = false

    return data.id
  } catch (err) {
    fileObj.uploading = false
    throw new Error(`文件 ${fileObj.name} 上传失败: ${err.message}`)
  }
}

// 分片上传大文件
const uploadFileInChunks = async (fileObj) => {
  const file = fileObj.file
  const totalSize = file.size
  const chunkSize = CHUNK_SIZE
  const totalChunks = Math.ceil(totalSize / chunkSize)

  try {
    fileObj.uploading = true
    fileObj.uploadProgress = 0

    // 1. 初始化分片上传
    const initResponse = await fetch(`${API_BASE}/api/paste/chunk/init`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        file_name: file.name,
        file_size: totalSize,
        chunk_size: chunkSize,
        total_chunks: totalChunks
      })
    })

    const initData = await initResponse.json()
    if (!initResponse.ok) {
      throw new Error(initData.error || '初始化上传失败')
    }

    const fileID = initData.file_id

    // 2. 上传每个分片
    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize
      const end = Math.min(start + chunkSize, totalSize)
      const chunk = file.slice(start, end)

      const chunkFormData = new FormData()
      chunkFormData.append('chunk', chunk)
      chunkFormData.append('chunk_index', i.toString())

      const chunkResponse = await fetch(`${API_BASE}/api/paste/chunk/${fileID}`, {
        method: 'POST',
        body: chunkFormData
      })

      const chunkData = await chunkResponse.json()
      if (!chunkResponse.ok) {
        throw new Error(chunkData.error || `分片 ${i + 1} 上传失败`)
      }

      // 更新进度
      fileObj.uploadProgress = Math.round(((i + 1) / totalChunks) * 90)
    }

    // 3. 合并分片
    const mergeResponse = await fetch(`${API_BASE}/api/paste/chunk/${fileID}/merge`, {
      method: 'POST'
    })

    const mergeData = await mergeResponse.json()
    if (!mergeResponse.ok) {
      throw new Error(mergeData.error || '合并分片失败')
    }

    fileObj.uploadProgress = 100
    fileObj.uploading = false

    return mergeData.id
  } catch (err) {
    fileObj.uploading = false
    throw new Error(`文件 ${fileObj.name} 上传失败: ${err.message}`)
  }
}

// 创建分享
const createPaste = async () => {
  if (!content.value.trim() && files.value.length === 0) {
    errorMsg.value = '请输入内容或上传文件'
    return
  }

  creating.value = true
  errorMsg.value = ''

  try {
    // 1. 上传所有文件
    const fileIDs = []
    for (const fileObj of files.value) {
      if (!fileObj.uploadedId) {
        ElMessage.info(`正在上传 ${fileObj.name}...`)
        const id = await uploadFile(fileObj)
        fileObj.uploadedId = id
        fileIDs.push(id)
      } else {
        fileIDs.push(fileObj.uploadedId)
      }
    }

    // 2. 创建粘贴板
    const response = await fetch(`${API_BASE}/api/paste`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: content.value,
        title: title.value,
        language: language.value,
        expires_in: expiresIn.value,
        max_views: maxViews.value,
        password: password.value,
        file_ids: fileIDs,
        admin_password: adminPassword.value
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }

    createdId.value = data.id
    createdExpires.value = new Date(data.expires_at).toLocaleString('zh-CN')
    createdMaxViews.value = data.max_views
    shareUrl.value = `${window.location.origin}/paste/${data.id}`
    showResult.value = true

    // 保存到我的分享列表
    saveMyShare({
      id: data.id,
      title: title.value || '',
      created_at: data.created_at || new Date().toISOString(),
      expires_at: data.expires_at,
      max_views: data.max_views,
      views: 0
    })

    // 自动复制链接
    try {
      await navigator.clipboard.writeText(shareUrl.value)
      ElMessage.success('链接已自动复制到剪贴板')
    } catch {
      ElMessage.success('分享创建成功')
    }

    // 生成二维码
    await nextTick()
    if (qrCanvas.value) {
      QRCode.toCanvas(qrCanvas.value, shareUrl.value, {
        width: 150,
        margin: 2,
        color: {
          dark: '#333',
          light: '#fff'
        }
      })
    }
  } catch (e) {
    errorMsg.value = e.message
  } finally {
    creating.value = false
  }
}

const copyUrl = async () => {
  try {
    await navigator.clipboard.writeText(shareUrl.value)
    ElMessage.success('链接已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const openShare = () => {
  window.open(shareUrl.value, '_blank')
}

const resetForm = () => {
  content.value = ''
  title.value = ''
  password.value = ''
  adminPassword.value = ''
  showResult.value = false
  createdId.value = ''

  // 清理文件
  for (const fileObj of files.value) {
    if (fileObj.preview) {
      URL.revokeObjectURL(fileObj.preview)
    }
  }
  files.value = []
}

// 管理员登录
const adminLogin = async () => {
  if (!adminPasswordInput.value) {
    ElMessage.warning('请输入管理员密码')
    return
  }

  adminLoading.value = true

  try {
    const response = await fetch(`${API_BASE}/api/paste/admin/list?admin_password=${encodeURIComponent(adminPasswordInput.value)}`)
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '登录失败')
    }

    adminAuthenticated.value = true
    adminPasswordStored.value = adminPasswordInput.value
    adminPastes.value = data.pastes || []
    ElMessage.success('登录成功')

    // 存储到 sessionStorage
    sessionStorage.setItem('paste_admin_password', adminPasswordInput.value)
  } catch (err) {
    ElMessage.error(err.message)
  } finally {
    adminLoading.value = false
  }
}

// 加载管理员粘贴板列表
const loadAdminPastes = async () => {
  adminLoading.value = true

  try {
    const response = await fetch(`${API_BASE}/api/paste/admin/list?admin_password=${encodeURIComponent(adminPasswordStored.value)}`)
    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '加载失败')
    }

    adminPastes.value = data.pastes || []
    ElMessage.success('刷新成功')
  } catch (err) {
    ElMessage.error(err.message)
  } finally {
    adminLoading.value = false
  }
}

// 查看粘贴板详情
const viewAdminPaste = (id) => {
  window.open(`/paste/${id}`, '_blank')
}

// 编辑粘贴板
const editAdminPaste = (paste) => {
  editingPaste.value = paste
  editExpiresIn.value = 24
  editMaxViews.value = paste.max_views
  showEditDialog.value = true
}

// 保存编辑
const saveAdminEdit = async () => {
  adminLoading.value = true

  try {
    const response = await fetch(`${API_BASE}/api/paste/admin/${editingPaste.value.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        admin_password: adminPasswordStored.value,
        expires_in: editExpiresIn.value,
        max_views: editMaxViews.value
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '更新失败')
    }

    ElMessage.success('更新成功')
    showEditDialog.value = false
    await loadAdminPastes()
  } catch (err) {
    ElMessage.error(err.message)
  } finally {
    adminLoading.value = false
  }
}

// 删除粘贴板
const deleteAdminPaste = async (id) => {
  try {
    await ElMessageBox.confirm('确定要删除这个粘贴板吗？', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })

    adminLoading.value = true

    const response = await fetch(`${API_BASE}/api/paste/admin/${id}?admin_password=${encodeURIComponent(adminPasswordStored.value)}`, {
      method: 'DELETE'
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '删除失败')
    }

    ElMessage.success('删除成功')
    await loadAdminPastes()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error(err.message || '删除失败')
    }
  } finally {
    adminLoading.value = false
  }
}

// 从 sessionStorage 恢复管理员密码
const restoreAdminPassword = () => {
  const stored = sessionStorage.getItem('paste_admin_password')
  if (stored) {
    adminPasswordInput.value = stored
    adminPasswordStored.value = stored
  }
}

restoreAdminPassword()
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.tool-header h2 {
  margin: 0;
  color: var(--text-primary);
}

.info-text {
  color: #67c23a;
  font-size: 14px;
}

/* 简洁模式 */
.quick-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.quick-editor {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  min-height: 200px;
  display: flex;
  position: relative;
}

.quick-editor .code-editor {
  flex: 1;
  padding: 20px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  border: none;
  resize: none;
  font-family: var(--font-family-mono);
  font-size: 15px;
  line-height: 1.6;
  outline: none;
}

.quick-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}

.quick-actions .el-button--large {
  padding: 15px 40px;
  font-size: 16px;
}

/* 高级模式 */
.create-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  min-height: 200px;
  position: relative;
}

.panel-header {
  padding: 10px 15px;
  background-color: var(--bg-secondary);
  display: flex;
  gap: 10px;
  align-items: center;
  border-bottom: 1px solid var(--border-base);
}

.code-editor {
  flex: 1;
  width: 100%;
  padding: 15px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  border: none;
  resize: none;
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.5;
  outline: none;
}

.options-row {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  padding: 15px;
  border-radius: var(--radius-md);
}

.option-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.option-label {
  color: var(--text-secondary);
  font-size: 14px;
}

.hint-text {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* 文件上传区域 */
.file-section {
  background-color: var(--bg-primary);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  padding: 15px;
}

.file-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.file-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
  font-size: 14px;
}

.size-info {
  color: #67c23a;
  font-size: 13px;
}

.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.file-item {
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
  overflow: hidden;
  background-color: var(--bg-secondary);
  display: flex;
  flex-direction: column;
}

.file-preview {
  width: 100%;
  height: 150px;
  background-color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-preview img,
.file-preview video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.file-preview audio {
  width: 100%;
  padding: 10px;
  background-color: #252525;
}

.file-icon {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #67c23a;
}

.file-ext {
  font-size: 12px;
  color: var(--text-tertiary);
  font-weight: bold;
}

.file-info {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.file-name {
  font-size: 13px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  font-size: 12px;
  color: var(--text-tertiary);
}

.file-actions {
  padding: 10px;
  display: flex;
  gap: 5px;
  border-top: 1px solid var(--border-base);
}

.file-add {
  border: 2px dashed #d0d0d0;
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s;
  min-height: 150px;
}

.file-add:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.file-upload {
  border: 2px dashed #d0d0d0;
  border-radius: var(--radius-md);
  padding: 30px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s;
}

.file-upload:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.upload-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* 拖拽状态 */
.is-dragging {
  border: 2px dashed #409eff !important;
}

.drop-overlay {
  position: absolute;
  inset: 0;
  background: rgba(30, 30, 30, 0.95);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 15px;
  color: var(--color-primary);
  font-size: 18px;
  z-index: 10;
  border-radius: var(--radius-md);
}

/* 结果展示 */
.result-section {
  display: flex;
  justify-content: center;
  margin-top: 30px;
  margin-bottom: 40px;
}

.result-card {
  background: linear-gradient(135deg, #1e3a2f 0%, #1e1e1e 100%);
  border: 2px solid #67c23a;
  border-radius: 16px;
  padding: 30px;
  max-width: 500px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  box-shadow: 0 8px 32px rgba(103, 194, 58, 0.2);
}

.result-header {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #67c23a;
  font-size: 18px;
  font-weight: 500;
}

.success-icon {
  font-size: 28px;
}

.share-url-box {
  width: 100%;
  background: #252525;
  border-radius: var(--radius-md);
  padding: 15px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.url-display {
  font-family: var(--font-family-mono);
  font-size: 14px;
  color: #67c23a;
  word-break: break-all;
  padding: 10px;
  background: #1a1a1a;
  border-radius: var(--radius-sm);
  text-align: center;
}

.url-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.qr-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.qr-title {
  color: var(--text-secondary);
  font-size: 14px;
}

.qr-code {
  border-radius: var(--radius-md);
  background: #fff;
  padding: 10px;
}

.result-info {
  display: flex;
  gap: 20px;
  color: #808080;
  font-size: 13px;
  flex-wrap: wrap;
  justify-content: center;
}

.new-share-btn {
  margin-top: 10px;
}

.tips-section {
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-base);
  padding: 20px;
  border-radius: var(--radius-md);
}

.tips-section h4 {
  margin: 0 0 10px 0;
  color: var(--text-primary);
}

.tips-section ul {
  margin: 0;
  padding-left: 20px;
  color: var(--text-secondary);
  line-height: 1.8;
}

.error-msg {
  margin-top: 10px;
}

/* 支持项目 Footer */
.project-footer {
  margin-top: 40px;
  padding: 30px 20px;
  background: linear-gradient(135deg, #1a1a1a 0%, #252525 100%);
  border: 1px solid var(--border-base);
  border-radius: var(--radius-md);
}

.footer-content {
  max-width: 800px;
  margin: 0 auto;
}

.footer-section {
  text-align: center;
  margin-bottom: 20px;
}

.footer-section h4 {
  margin: 0 0 10px 0;
  color: #67c23a;
  font-size: 20px;
}

.footer-section p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.footer-donate {
  display: flex;
  justify-content: center;
  gap: 40px;
  flex-wrap: wrap;
}

.donate-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.donate-label {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 500;
}

.donate-link {
  color: #409eff;
  text-decoration: none;
  padding: 6px 16px;
  border: 1px solid #409eff;
  border-radius: 4px;
  transition: all 0.3s;
}

.donate-link:hover {
  background: #409eff;
  color: white;
}

/* 赞赏弹窗 */
.donate-dialog {
  text-align: center;
}

.donate-qr-group {
  display: flex;
  justify-content: center;
  gap: 30px;
  margin: 20px 0;
}

.donate-qr-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.donate-qr-item img {
  width: 180px;
  height: 180px;
  border-radius: 8px;
  border: 2px solid var(--border-base);
}

.donate-qr-item span {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 500;
}

.donate-thanks {
  margin-top: 20px;
  color: #67c23a;
  font-size: 14px;
}

@media (max-width: 768px) {
  .donate-qr-group {
    flex-direction: column;
    gap: 20px;
  }

  .footer-donate {
    flex-direction: column;
    gap: 20px;
  }
}
</style>
