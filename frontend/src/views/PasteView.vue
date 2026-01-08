<template>
  <div class="tool-container">
    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
      <span>加载中...</span>
    </div>

    <div v-else-if="needPassword" class="password-section">
      <el-card class="password-card">
        <template #header>
          <div class="card-header">
            <el-icon><Lock /></el-icon>
            <span>需要密码访问</span>
          </div>
        </template>
        <el-form @submit.prevent="submitPassword">
          <el-form-item label="访问密码">
            <el-input
              v-model="password"
              type="password"
              placeholder="请输入访问密码"
              show-password
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submitPassword" :loading="verifying">
              验证
            </el-button>
          </el-form-item>
        </el-form>
        <div v-if="passwordError" class="error-text">{{ passwordError }}</div>
      </el-card>
    </div>

    <div v-else-if="error" class="error-section">
      <el-result icon="error" :title="error">
        <template #extra>
          <el-button type="primary" @click="$router.push('/paste')">
            创建新分享
          </el-button>
        </template>
      </el-result>
    </div>

    <div v-else-if="paste" class="paste-content">
      <div class="paste-header">
        <div class="paste-title">
          <h2>{{ paste.title || '无标题' }}</h2>
          <el-tag size="small">{{ paste.language }}</el-tag>
        </div>
        <div class="paste-meta">
          <span>创建于 {{ formatDate(paste.created_at) }}</span>
          <span>·</span>
          <span>访问 {{ paste.views }}/{{ paste.max_views }}</span>
          <span>·</span>
          <span>过期于 {{ formatDate(paste.expires_at) }}</span>
        </div>
      </div>

      <div class="editor-panel">
        <div class="panel-header">
          <span>内容</span>
          <div class="actions">
            <el-button size="small" @click="copyContent">
              <el-icon><CopyDocument /></el-icon>
              复制
            </el-button>
            <el-button size="small" @click="downloadContent">
              <el-icon><Download /></el-icon>
              下载
            </el-button>
          </div>
        </div>
        <pre class="code-content" v-html="highlightedContent"></pre>
      </div>

      <div class="back-section">
        <el-button @click="$router.push('/paste')">
          <el-icon><Back /></el-icon>
          创建新分享
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading, Lock, Back } from '@element-plus/icons-vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

const API_BASE = import.meta.env.VITE_API_BASE || ''

const route = useRoute()
const loading = ref(true)
const paste = ref(null)
const error = ref('')
const needPassword = ref(false)
const password = ref('')
const passwordError = ref('')
const verifying = ref(false)

const fetchPaste = async (pwd = '') => {
  const id = route.params.id
  if (!id) {
    error.value = '无效的分享 ID'
    loading.value = false
    return
  }

  try {
    const url = pwd
      ? `${API_BASE}/api/paste/${id}?password=${encodeURIComponent(pwd)}`
      : `${API_BASE}/api/paste/${id}`

    const response = await fetch(url)
    const data = await response.json()

    if (response.status === 401) {
      if (data.has_password) {
        needPassword.value = true
        passwordError.value = pwd ? '密码错误' : ''
      } else {
        error.value = data.error || '访问被拒绝'
      }
    } else if (!response.ok) {
      error.value = data.error || '加载失败'
    } else {
      paste.value = data
      needPassword.value = false
    }
  } catch (e) {
    error.value = '网络错误，请稍后重试'
  } finally {
    loading.value = false
    verifying.value = false
  }
}

const submitPassword = async () => {
  if (!password.value) {
    passwordError.value = '请输入密码'
    return
  }
  verifying.value = true
  passwordError.value = ''
  await fetchPaste(password.value)
}

const highlightedContent = computed(() => {
  if (!paste.value) return ''
  const lang = paste.value.language
  const content = paste.value.content

  if (hljs.getLanguage(lang)) {
    try {
      return hljs.highlight(content, { language: lang }).value
    } catch (e) {
      return escapeHtml(content)
    }
  }
  return escapeHtml(content)
})

const escapeHtml = (text) => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const formatDate = (dateStr) => {
  return new Date(dateStr).toLocaleString('zh-CN')
}

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(paste.value.content)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const downloadContent = () => {
  const blob = new Blob([paste.value.content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${paste.value.title || paste.value.id}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  fetchPaste()
})
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-height: 400px;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 15px;
  min-height: 300px;
  color: #a0a0a0;
}

.password-section {
  display: flex;
  justify-content: center;
  padding-top: 50px;
}

.password-card {
  width: 400px;
  background-color: #1e1e1e;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #e0e0e0;
}

.error-text {
  color: #f56c6c;
  margin-top: 10px;
}

.error-section {
  display: flex;
  justify-content: center;
  padding-top: 50px;
}

.paste-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.paste-header {
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.paste-title {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 10px;
}

.paste-title h2 {
  margin: 0;
  color: #e0e0e0;
}

.paste-meta {
  display: flex;
  gap: 10px;
  color: #a0a0a0;
  font-size: 14px;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
}

.panel-header {
  padding: 10px 15px;
  background-color: #2d2d2d;
  color: #a0a0a0;
  font-size: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.actions {
  display: flex;
  gap: 10px;
}

.code-content {
  margin: 0;
  padding: 20px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-x: auto;
  min-height: 200px;
  max-height: 600px;
  overflow-y: auto;
}

.back-section {
  display: flex;
  justify-content: center;
}
</style>
