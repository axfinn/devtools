<template>
  <div class="share-container">
    <div class="share-header">
      <h2>{{ shareData?.title || 'Excalidraw 画图' }}</h2>
      <div class="header-actions" v-if="isVerified">
        <el-dropdown @command="handleExport" trigger="click">
          <el-button type="primary" size="small">
            <el-icon><Download /></el-icon>
            下载
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="png">下载 PNG</el-dropdown-item>
              <el-dropdown-item command="svg">下载 SVG</el-dropdown-item>
              <el-dropdown-item command="json">下载 JSON</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button size="small" @click="copyLink">
          <el-icon><Link /></el-icon>
          复制链接
        </el-button>
      </div>
    </div>

    <!-- Password verification -->
    <div v-if="!isVerified && !loading && !errorMsg" class="password-card">
      <el-card shadow="hover">
        <template #header>
          <div class="card-header">
            <el-icon><Lock /></el-icon>
            <span>此画图需要密码访问</span>
          </div>
        </template>
        <el-form @submit.prevent="verifyPassword">
          <el-form-item>
            <el-input
              v-model="password"
              type="password"
              placeholder="请输入访问密码"
              show-password
              @keyup.enter="verifyPassword"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="verifyPassword" :loading="verifying" style="width: 100%">
              查看画图
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-container">
      <el-icon class="loading-icon"><Loading /></el-icon>
      <span>加载中...</span>
    </div>

    <!-- Error -->
    <div v-if="errorMsg" class="error-container">
      <el-result icon="error" :title="errorMsg">
        <template #extra>
          <el-button @click="goBack">返回首页</el-button>
        </template>
      </el-result>
    </div>

    <!-- Excalidraw viewer -->
    <div v-if="isVerified && !loading && !errorMsg" class="viewer-wrapper">
      <ExcalidrawWrapper
        ref="excalidrawRef"
        :initial-data="sceneData"
        :view-mode-enabled="true"
        :theme="'light'"
      />
    </div>

    <!-- Expiration notice -->
    <div v-if="isVerified && shareData && !shareData.is_permanent" class="expiration-notice">
      <el-alert
        :type="isExpiringSoon ? 'warning' : 'info'"
        :closable="false"
      >
        <template #title>
          <span v-if="isExpiringSoon">此画图即将过期：{{ formatDate(shareData.expires_at) }}</span>
          <span v-else>此画图有效期至：{{ formatDate(shareData.expires_at) }}</span>
        </template>
      </el-alert>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import ExcalidrawWrapper from '../components/ExcalidrawWrapper.vue'
import { useTheme } from '../composables/useTheme'

const route = useRoute()
const router = useRouter()

const excalidrawRef = ref(null)
const { currentTheme } = useTheme()
const loading = ref(false)
const verifying = ref(false)
const errorMsg = ref('')
const password = ref('')
const isVerified = ref(false)
const shareData = ref(null)
const sceneData = ref(null)

const isExpiringSoon = computed(() => {
  if (!shareData.value?.expires_at) return false
  const expires = new Date(shareData.value.expires_at)
  const now = new Date()
  const daysLeft = (expires - now) / (1000 * 60 * 60 * 24)
  return daysLeft <= 3
})

onMounted(async () => {
  const id = route.params.id
  if (!id) {
    errorMsg.value = '无效的画图链接'
    return
  }

  // Try to auto-load with saved password if user is the creator
  const savedPassword = getSavedPassword(id)
  if (savedPassword) {
    password.value = savedPassword
    await verifyPassword(true) // silent mode for auto-login
  }
})

const getSavedPassword = (id) => {
  try {
    const creatorKeys = JSON.parse(localStorage.getItem('excalidraw_creator_keys') || '{}')
    if (creatorKeys[id] && creatorKeys[id].password) {
      return creatorKeys[id].password
    }
  } catch {
    // ignore
  }
  return null
}

const verifyPassword = async (silent = false) => {
  if (!password.value) {
    if (!silent) {
      ElMessage.warning('请输入密码')
    }
    return
  }

  const id = route.params.id
  verifying.value = true
  errorMsg.value = ''

  try {
    const res = await fetch(`/api/excalidraw/${id}?password=${encodeURIComponent(password.value)}`)
    const data = await res.json()

    if (!res.ok) {
      if (res.status === 401) {
        if (!silent) {
          ElMessage.error('密码错误')
        }
      } else if (res.status === 410) {
        errorMsg.value = '此画图已过期'
      } else if (res.status === 404) {
        errorMsg.value = '画图不存在'
      } else {
        if (!silent) {
          ElMessage.error(data.error || '获取失败')
        }
      }
      return
    }

    shareData.value = data
    if (data.content) {
      sceneData.value = JSON.parse(data.content)
    }
    isVerified.value = true

    // Auto scroll to content after scene is loaded
    setTimeout(() => {
      if (excalidrawRef.value) {
        excalidrawRef.value.scrollToContent({ delay: 500 })
      }
    }, 100)
  } catch (err) {
    if (!silent) {
      ElMessage.error('网络错误')
    }
  } finally {
    verifying.value = false
  }
}

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
    ElMessage.error('导出失败')
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

const copyLink = async () => {
  try {
    await navigator.clipboard.writeText(window.location.href)
    ElMessage.success('链接已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const goBack = () => {
  router.push('/excalidraw')
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}
</script>

<style scoped>
.share-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  padding: 20px;
  box-sizing: border-box;
}

.share-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.share-header h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 20px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.password-card {
  display: flex;
  justify-content: center;
  align-items: center;
  flex: 1;
}

.password-card .el-card {
  width: 360px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
}

.loading-container {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  flex: 1;
  gap: 12px;
  color: var(--text-tertiary);
}

.loading-icon {
  font-size: 32px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.error-container {
  display: flex;
  justify-content: center;
  align-items: center;
  flex: 1;
}

.viewer-wrapper {
  flex: 1;
  min-height: 0;
  border-radius: var(--radius-md);
  overflow: hidden;
  position: relative;
}

.expiration-notice {
  margin-top: 16px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .share-container {
    padding: 12px;
  }

  .share-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .viewer-wrapper {
    min-height: 400px;
  }
}
</style>
