<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>共享粘贴板</h2>
      <div class="info-text">
        多设备快速同步 - 创建后自动复制链接
      </div>
    </div>

    <!-- 快捷模式：简洁界面 -->
    <div class="quick-section" v-if="!showAdvanced">
      <div class="quick-editor">
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="粘贴或输入内容，点击分享即可获得链接..."
          spellcheck="false"
          @paste="onPaste"
        ></textarea>
      </div>
      <div class="quick-actions">
        <el-button type="primary" size="large" @click="quickCreate" :loading="creating" :disabled="!content.trim()">
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
      <div class="editor-panel">
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
            <el-option label="TypeScript" value="typescript" />
            <el-option label="Python" value="python" />
            <el-option label="Go" value="go" />
            <el-option label="Java" value="java" />
            <el-option label="C/C++" value="cpp" />
            <el-option label="HTML" value="html" />
            <el-option label="CSS" value="css" />
            <el-option label="SQL" value="sql" />
            <el-option label="Markdown" value="markdown" />
            <el-option label="Shell" value="bash" />
            <el-option label="YAML" value="yaml" />
            <el-option label="XML" value="xml" />
          </el-select>
          <el-button size="small" text @click="showAdvanced = false">简洁模式</el-button>
        </div>
        <textarea
          v-model="content"
          class="code-editor"
          placeholder="在此输入要分享的内容..."
          spellcheck="false"
        ></textarea>
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
          <el-input-number v-model="maxViews" :min="1" :max="1000" />
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
        <el-button type="primary" size="large" @click="createPaste" :loading="creating">
          创建分享
        </el-button>
      </div>
    </div>

    <!-- 分享结果：突出显示 -->
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
        <li>直接粘贴内容，点击分享即可</li>
        <li>链接自动复制到剪贴板</li>
        <li>手机可扫描二维码访问</li>
        <li>默认 24 小时后过期</li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Share, CircleCheck, CopyDocument, Link, Plus } from '@element-plus/icons-vue'
import QRCode from 'qrcode'

const API_BASE = import.meta.env.VITE_API_BASE || ''

const content = ref('')
const title = ref('')
const language = ref('text')
const expiresIn = ref(24)
const maxViews = ref(100)
const password = ref('')
const creating = ref(false)
const showResult = ref(false)
const showAdvanced = ref(false)
const createdId = ref('')
const createdExpires = ref('')
const shareUrl = ref('')
const errorMsg = ref('')
const qrCanvas = ref(null)

// 快捷创建
const quickCreate = async () => {
  await createPaste()
}

// 粘贴时自动检测内容类型
const onPaste = (e) => {
  const text = e.clipboardData?.getData('text') || ''
  // 简单检测 JSON
  if (text.trim().startsWith('{') || text.trim().startsWith('[')) {
    try {
      JSON.parse(text)
      language.value = 'json'
    } catch {}
  }
}

const createPaste = async () => {
  if (!content.value.trim()) {
    errorMsg.value = '请输入要分享的内容'
    return
  }

  creating.value = true
  errorMsg.value = ''

  try {
    const response = await fetch(`${API_BASE}/api/paste`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: content.value,
        title: title.value,
        language: language.value,
        expires_in: expiresIn.value,
        max_views: maxViews.value,
        password: password.value
      })
    })

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || '创建失败')
    }

    createdId.value = data.id
    createdExpires.value = new Date(data.expires_at).toLocaleString('zh-CN')
    shareUrl.value = `${window.location.origin}/paste/${data.id}`
    showResult.value = true

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
  showResult.value = false
  createdId.value = ''
}
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
  color: #e0e0e0;
}

.info-text {
  color: #67c23a;
  font-size: 14px;
}

/* 快捷模式 */
.quick-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.quick-editor {
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  min-height: 300px;
  display: flex;
}

.quick-editor .code-editor {
  flex: 1;
  padding: 20px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', monospace;
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
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  min-height: 300px;
}

.panel-header {
  padding: 10px 15px;
  background-color: #2d2d2d;
  display: flex;
  gap: 10px;
  align-items: center;
}

.code-editor {
  flex: 1;
  width: 100%;
  padding: 15px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.5;
  outline: none;
}

.options-row {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
  background-color: #1e1e1e;
  padding: 15px;
  border-radius: 8px;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.option-label {
  color: #a0a0a0;
  font-size: 14px;
}

/* 结果展示 */
.result-section {
  display: flex;
  justify-content: center;
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
  border-radius: 8px;
  padding: 15px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.url-display {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  color: #67c23a;
  word-break: break-all;
  padding: 10px;
  background: #1a1a1a;
  border-radius: 4px;
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
  color: #a0a0a0;
  font-size: 14px;
}

.qr-code {
  border-radius: 8px;
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
  background-color: #1e1e1e;
  padding: 20px;
  border-radius: 8px;
}

.tips-section h4 {
  margin: 0 0 10px 0;
  color: #e0e0e0;
}

.tips-section ul {
  margin: 0;
  padding-left: 20px;
  color: #a0a0a0;
  line-height: 1.8;
}

.error-msg {
  margin-top: 10px;
}
</style>
