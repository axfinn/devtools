<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Base64 编解码</h2>
    </div>

    <el-tabs v-model="activeTab" class="tool-tabs">
      <el-tab-pane label="文本编解码" name="text">
        <div class="tab-content">
          <div class="editor-panel">
            <div class="panel-header">原始文本</div>
            <textarea
              v-model="textInput"
              class="code-editor"
              placeholder="输入文本..."
              spellcheck="false"
            ></textarea>
          </div>
          <div class="button-group">
            <el-button type="primary" @click="encodeText">
              编码 →
            </el-button>
            <el-button type="success" @click="decodeText">
              ← 解码
            </el-button>
          </div>
          <div class="editor-panel">
            <div class="panel-header">
              Base64
              <el-button size="small" @click="copyBase64">复制</el-button>
            </div>
            <textarea
              v-model="base64Output"
              class="code-editor"
              placeholder="Base64 结果..."
              spellcheck="false"
            ></textarea>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="图片编解码" name="image">
        <div class="image-tab-content">
          <div class="upload-section">
            <el-upload
              class="image-uploader"
              drag
              :auto-upload="false"
              :show-file-list="false"
              accept="image/*"
              @change="handleImageUpload"
            >
              <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
              <div class="el-upload__text">
                拖拽图片到此处，或 <em>点击上传</em>
              </div>
            </el-upload>
          </div>

          <div v-if="imagePreview" class="image-preview">
            <img :src="imagePreview" alt="Preview" />
          </div>

          <div class="editor-panel" v-if="imageBase64">
            <div class="panel-header">
              Base64 (Data URL)
              <el-button size="small" @click="copyImageBase64">复制</el-button>
            </div>
            <textarea
              v-model="imageBase64"
              class="code-editor small"
              readonly
            ></textarea>
          </div>

          <div class="decode-section">
            <div class="panel-header">从 Base64 解码图片</div>
            <el-input
              v-model="base64ToImage"
              type="textarea"
              :rows="3"
              placeholder="粘贴 Base64 图片数据 (data:image/... 或纯 Base64)"
            />
            <el-button type="primary" @click="decodeImage" style="margin-top: 10px">
              解码显示
            </el-button>
            <div v-if="decodedImage" class="image-preview">
              <img :src="decodedImage" alt="Decoded" />
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon :closable="false" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

const activeTab = ref('text')
const textInput = ref('')
const base64Output = ref('')
const imageBase64 = ref('')
const imagePreview = ref('')
const base64ToImage = ref('')
const decodedImage = ref('')
const errorMsg = ref('')

const encodeText = () => {
  try {
    base64Output.value = btoa(unescape(encodeURIComponent(textInput.value)))
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = '编码错误: ' + e.message
  }
}

const decodeText = () => {
  try {
    textInput.value = decodeURIComponent(escape(atob(base64Output.value)))
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = '解码错误: ' + e.message
  }
}

const copyBase64 = async () => {
  try {
    await navigator.clipboard.writeText(base64Output.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const handleImageUpload = (file) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    imageBase64.value = e.target.result
    imagePreview.value = e.target.result
  }
  reader.readAsDataURL(file.raw)
}

const copyImageBase64 = async () => {
  try {
    await navigator.clipboard.writeText(imageBase64.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const decodeImage = () => {
  try {
    let src = base64ToImage.value.trim()
    if (!src.startsWith('data:image')) {
      src = 'data:image/png;base64,' + src
    }
    decodedImage.value = src
    errorMsg.value = ''
  } catch (e) {
    errorMsg.value = '图片解码错误: ' + e.message
  }
}
</script>

<style scoped>
.tool-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.tool-tabs {
  flex: 1;
}

.tool-tabs :deep(.el-tabs__content) {
  height: calc(100% - 40px);
}

.tool-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.tab-content {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 15px;
  height: 100%;
  min-height: 400px;
}

.image-tab-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
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

.code-editor.small {
  max-height: 150px;
}

.button-group {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
}

.upload-section {
  max-width: 400px;
}

.image-uploader :deep(.el-upload-dragger) {
  background-color: #1e1e1e;
  border-color: #444;
}

.image-uploader :deep(.el-upload-dragger:hover) {
  border-color: #409eff;
}

.image-preview {
  max-width: 100%;
  max-height: 300px;
  overflow: auto;
  background-color: #1e1e1e;
  padding: 10px;
  border-radius: 8px;
}

.image-preview img {
  max-width: 100%;
  max-height: 280px;
  object-fit: contain;
}

.decode-section {
  background-color: #1e1e1e;
  padding: 15px;
  border-radius: 8px;
}

.error-msg {
  margin-top: 10px;
}
</style>
