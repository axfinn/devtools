<template>
  <div class="zip-viewer-container">
    <!-- 工具栏 -->
    <div class="zip-toolbar">
      <div class="toolbar-left">
        <el-icon><Folder /></el-icon>
        <span class="zip-title">{{ filename }}</span>
        <el-tag size="small">{{ fileList.length }} 个文件</el-tag>
      </div>
      <div class="toolbar-right">
        <el-button size="small" @click="downloadZip">
          <el-icon><Download /></el-icon>
          下载
        </el-button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="zip-loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
      <span>正在解压...</span>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="zip-error">
      <el-result icon="error" :title="error">
        <template #extra>
          <el-button type="primary" @click="retryLoad">重试</el-button>
          <el-button @click="downloadZip">下载文件</el-button>
        </template>
      </el-result>
    </div>

    <!-- 文件列表 -->
    <div v-else class="zip-content">
      <!-- 路径导航 -->
      <div class="breadcrumb" v-if="currentPath">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item :to="{ path: '' }" @click="goToPath('')">
            根目录
          </el-breadcrumb-item>
          <el-breadcrumb-item
            v-for="(part, index) in pathParts"
            :key="index"
            :to="{ path: '' }"
            @click="goToPath(pathParts.slice(0, index + 1).join('/'))"
          >
            {{ part }}
          </el-breadcrumb-item>
        </el-breadcrumb>
      </div>

      <!-- 目录/文件表格 -->
      <el-table
        :data="displayFiles"
        style="width: 100%"
        @row-click="handleRowClick"
        :row-class-name="tableRowClassName"
        max-height="500"
      >
        <el-table-column label="类型" width="60">
          <template #default="{ row }">
            <el-icon v-if="row.isDirectory">
              <Folder />
            </el-icon>
            <el-icon v-else>
              <Document />
            </el-icon>
          </template>
        </el-table-column>

        <el-table-column prop="name" label="文件名" min-width="200">
          <template #default="{ row }">
            <span :class="{ 'folder-name': row.isDirectory }">{{ row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="size" label="大小" width="120">
          <template #default="{ row }">
            <span v-if="!row.isDirectory">{{ formatSize(row.size) }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button
              v-if="!row.isDirectory && canPreview(row)"
              size="small"
              type="primary"
              @click.stop="previewFile(row)"
            >
              预览
            </el-button>
            <el-button
              v-if="!row.isDirectory"
              size="small"
              @click.stop="downloadFile(row)"
            >
              下载
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 文件预览弹窗 -->
      <el-dialog
        v-model="showPreview"
        :title="previewFileInfo?.name || '文件预览'"
        width="70%"
        :close-on-click-modal="true"
        destroy-on-close
      >
        <!-- 文本文件预览 -->
        <div v-if="previewType === 'text'" class="text-preview">
          <pre>{{ previewContent }}</pre>
        </div>

        <!-- 图片预览 -->
        <div v-else-if="previewType === 'image'" class="image-preview-container">
          <img :src="previewImageUrl" alt="预览图片" />
        </div>

        <!-- 不支持预览 -->
        <div v-else class="unsupported-preview">
          <el-result icon="info" title="暂不支持预览">
            <template #sub-title>
              <p>此文件类型暂不支持在线预览</p>
            </template>
            <template #extra>
              <el-button type="primary" @click="downloadPreviewFile">下载文件</el-button>
            </template>
          </el-result>
        </div>

        <template #footer>
          <el-button @click="showPreview = false">关闭</el-button>
          <el-button type="primary" @click="downloadPreviewFile">下载</el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading, Folder, Document, Download } from '@element-plus/icons-vue'
import JSZip from 'jszip'

const props = defineProps({
  url: {
    type: String,
    required: true
  },
  filename: {
    type: String,
    default: 'archive.zip'
  }
})

const emit = defineEmits(['download'])

// 状态
const loading = ref(true)
const error = ref('')
const fileList = ref([])
const currentPath = ref('')
const currentFolderFiles = ref([])

// 预览状态
const showPreview = ref(false)
const previewFileInfo = ref(null)
const previewContent = ref('')
const previewImageUrl = ref('')
const previewType = ref('')

// 可预览的文本类型
const textExtensions = ['.txt', '.md', '.json', '.js', '.ts', '.jsx', '.tsx', '.vue',
  '.html', '.css', '.scss', '.less', '.xml', '.yaml', '.yml', '.toml',
  '.py', '.go', '.java', '.c', '.cpp', '.h', '.hpp', '.cs', '.rb', '.php',
  '.sh', '.bash', '.sql', '.log', '.ini', '.conf', '.cfg']

// 可预览的图片类型
const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.ico']

// 路径部分
const pathParts = computed(() => {
  if (!currentPath.value) return []
  return currentPath.value.split('/').filter(p => p)
})

// 当前目录显示的文件
const displayFiles = computed(() => {
  return currentFolderFiles.value
})

// 加载压缩包
const loadZip = async () => {
  loading.value = true
  error.value = ''

  try {
    // 获取文件
    const response = await fetch(props.url)
    if (!response.ok) {
      throw new Error('无法加载压缩包文件')
    }

    const arrayBuffer = await response.arrayBuffer()

    // 解压
    const zip = await JSZip.loadAsync(arrayBuffer)

    // 构建文件列表
    const files = []
    zip.forEach((relativePath, zipEntry) => {
      const pathParts = relativePath.split('/')
      const name = pathParts[pathParts.length - 1]
      const isDirectory = zipEntry.dir

      // 计算父路径
      let parentPath = ''
      if (pathParts.length > 1) {
        parentPath = pathParts.slice(0, -1).join('/')
      }

      files.push({
        name,
        path: relativePath,
        isDirectory,
        size: zipEntry._data?.uncompressedSize || 0,
        parentPath,
        zipEntry
      })
    })

    fileList.value = files

    // 加载根目录文件
    loadCurrentFolder()

  } catch (err) {
    console.error('Zip load error:', err)
    error.value = '解压失败，请尝试下载后查看'
  } finally {
    loading.value = false
  }
}

// 加载当前目录的文件
const loadCurrentFolder = () => {
  currentFolderFiles.value = fileList.value.filter(file => {
    if (!currentPath.value) {
      // 根目录：显示顶级文件和文件夹
      return !file.parentPath
    }
    return file.parentPath === currentPath.value
  })

  // 按文件夹在前、文件在后排序
  currentFolderFiles.value.sort((a, b) => {
    if (a.isDirectory && !b.isDirectory) return -1
    if (!a.isDirectory && b.isDirectory) return 1
    return a.name.localeCompare(b.name)
  })
}

// 跳转到路径
const goToPath = (path) => {
  currentPath.value = path
  loadCurrentFolder()
}

// 处理行点击
const handleRowClick = (row) => {
  if (row.isDirectory) {
    goToPath(row.path)
  }
}

// 表格行样式
const tableRowClassName = ({ row }) => {
  return row.isDirectory ? 'folder-row' : 'file-row'
}

// 格式化文件大小
const formatSize = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 判断文件是否可以预览
const canPreview = (file) => {
  const ext = file.name.toLowerCase().slice(file.name.lastIndexOf('.'))
  return textExtensions.includes(ext) || imageExtensions.includes(ext)
}

// 获取预览类型
const getPreviewType = (filename) => {
  const ext = filename.toLowerCase().slice(filename.lastIndexOf('.'))
  if (textExtensions.includes(ext)) return 'text'
  if (imageExtensions.includes(ext)) return 'image'
  return 'unsupported'
}

// 预览文件
const previewFile = async (file) => {
  previewFileInfo.value = file
  const type = getPreviewType(file.name)
  previewType.value = type

  if (type === 'text') {
    try {
      const content = await file.zipEntry.async('string')
      // 限制预览内容大小 (最大 100KB)
      if (content.length > 100 * 1024) {
        previewContent.value = content.substring(0, 100 * 1024) + '\n\n... (文件过大，仅显示前 100KB)'
        ElMessage.warning('文件过大，仅显示前 100KB')
      } else {
        previewContent.value = content
      }
    } catch (err) {
      previewContent.value = '无法加载文件内容'
    }
  } else if (type === 'image') {
    try {
      const blob = await file.zipEntry.async('blob')
      previewImageUrl.value = URL.createObjectURL(blob)
    } catch (err) {
      previewImageUrl.value = ''
    }
  }

  showPreview.value = true
}

// 下载单个文件
const downloadFile = async (file) => {
  try {
    const blob = await file.zipEntry.async('blob')
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = file.name
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('下载成功')
  } catch (err) {
    ElMessage.error('下载失败')
  }
}

// 下载预览的文件
const downloadPreviewFile = () => {
  if (previewFileInfo.value) {
    downloadFile(previewFileInfo.value)
  }
}

// 下载整个压缩包
const downloadZip = () => {
  emit('download', props.url)
}

// 重试加载
const retryLoad = () => {
  loadZip()
}

// 监听 URL 变化
watch(() => props.url, () => {
  loadZip()
}, { immediate: true })

onMounted(() => {
  loadZip()
})
</script>

<style scoped>
.zip-viewer-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: var(--bg-primary, #fff);
}

.zip-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 15px;
  background-color: var(--bg-secondary, #f5f5f5);
  border-bottom: 1px solid var(--border-base, #e0e0e0);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.zip-title {
  font-weight: 500;
  color: var(--text-primary, #333);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.zip-loading,
.zip-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: var(--text-secondary, #666);
}

.zip-loading span {
  margin-top: 10px;
}

.zip-content {
  flex: 1;
  overflow: auto;
}

.breadcrumb {
  padding: 12px 15px;
  background-color: var(--bg-secondary, #f5f5f5);
  border-bottom: 1px solid var(--border-base, #e0e0e0);
}

:deep(.folder-row) {
  cursor: pointer;
}

:deep(.folder-row:hover) {
  background-color: #f5f7fa;
}

:deep(.file-row) {
  cursor: pointer;
}

:deep(.file-row:hover) {
  background-color: #ecf5ff;
}

.folder-name {
  color: #409eff;
  font-weight: 500;
}

/* 文本预览 */
.text-preview {
  max-height: 60vh;
  overflow: auto;
  background-color: #1e1e1e;
  border-radius: 4px;
  padding: 15px;
}

.text-preview pre {
  margin: 0;
  font-family: var(--font-family-mono, 'Consolas', monospace);
  font-size: 14px;
  line-height: 1.6;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 图片预览 */
.image-preview-container {
  display: flex;
  justify-content: center;
  align-items: center;
  max-height: 60vh;
  overflow: auto;
}

.image-preview-container img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

/* 不支持预览 */
.unsupported-preview {
  padding: 20px;
}
</style>
