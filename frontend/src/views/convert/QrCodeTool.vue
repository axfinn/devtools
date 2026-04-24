<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>二维码生成器</h2>
    </div>

    <el-tabs v-model="activeTab" class="tool-tabs">
      <el-tab-pane label="生成二维码" name="generate">
        <div class="generate-content">
          <div class="input-section">
            <el-input
              v-model="qrContent"
              type="textarea"
              :rows="4"
              placeholder="输入文本、网址或其他内容..."
              @input="generateQRCode"
            />
            <div class="options-row">
              <el-select v-model="qrSize" placeholder="尺寸" @change="generateQRCode">
                <el-option :value="128" label="128x128" />
                <el-option :value="200" label="200x200" />
                <el-option :value="256" label="256x256" />
                <el-option :value="300" label="300x300" />
                <el-option :value="400" label="400x400" />
              </el-select>
              <el-select v-model="qrLevel" placeholder="容错级别" @change="generateQRCode">
                <el-option value="L" label="低 (7%)" />
                <el-option value="M" label="中 (15%)" />
                <el-option value="Q" label="较高 (25%)" />
                <el-option value="H" label="高 (30%)" />
              </el-select>
              <el-color-picker v-model="qrColor" @change="generateQRCode" />
            </div>
          </div>

          <div class="preview-section">
            <div v-if="qrDataUrl" class="qr-preview">
              <img :src="qrDataUrl" alt="二维码" />
            </div>
            <div v-else class="qr-placeholder">
              <el-icon :size="80" color="#ccc"><Picture /></el-icon>
              <p>输入内容生成二维码</p>
            </div>
            <div class="action-buttons">
              <el-button type="primary" :disabled="!qrDataUrl" @click="downloadQRCode">
                下载 PNG
              </el-button>
              <el-button :disabled="!qrDataUrl" @click="copyQRCode">
                复制图片
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="解析二维码" name="parse">
        <div class="parse-content">
          <div class="paste-hint">
            <el-icon><Picture /></el-icon>
            <span>支持 Ctrl+V / Cmd+V 粘贴图片，或点击/拖拽上传</span>
          </div>
          <div class="upload-section">
            <el-upload
              ref="uploadRef"
              class="qr-upload"
              :auto-upload="false"
              :show-file-list="false"
              :on-change="handleFileChange"
              accept="image/*"
            >
              <div class="upload-area">
                <el-icon :size="60" color="#409eff"><Upload /></el-icon>
                <p>点击或拖拽上传二维码图片</p>
              </div>
            </el-upload>
          </div>

          <div v-if="previewImage" class="parse-preview">
            <img :src="previewImage" alt="待解析图片" />
          </div>

          <div v-if="parsedText !== null" class="parse-result">
            <div class="result-header">
              <span>解析结果：{{ parsedText.includes('---') ? '(多个)' : '' }}</span>
              <el-button size="small" @click="copyParsedText">复制</el-button>
            </div>
            <el-input
              v-model="parsedText"
              type="textarea"
              :rows="parsedText.split('\n').length + parsedText.split('---').length * 2"
              readonly
            />
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Picture, Upload } from '@element-plus/icons-vue'
import QRCode from 'qrcode'

const activeTab = ref('generate')
const qrContent = ref('')
const qrDataUrl = ref('')
const qrSize = ref(300)
const qrLevel = ref('H')
const qrColor = ref('#000000')

const previewImage = ref('')
const parsedText = ref(null)
const uploadRef = ref(null)
const isListening = ref(false)

// 粘贴板解析
const handlePaste = async (e) => {
  const items = e.clipboardData?.items
  if (!items) return

  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const blob = item.getAsFile()
      if (blob) {
        const reader = new FileReader()
        reader.onload = (event) => {
          previewImage.value = event.target.result
          parseQRCode(event.target.result)
        }
        reader.readAsDataURL(blob)
      }
      break
    }
  }
}

// 监听/取消粘贴板
const setupPasteListener = (enable) => {
  if (enable && !isListening.value) {
    window.addEventListener('paste', handlePaste)
    isListening.value = true
  } else if (!enable && isListening.value) {
    window.removeEventListener('paste', handlePaste)
    isListening.value = false
  }
}

// 监听标签页切换
watch(activeTab, (newTab) => {
  setupPasteListener(newTab === 'parse')
})

onMounted(() => {
  if (activeTab.value === 'parse') {
    setupPasteListener(true)
  }
})

onUnmounted(() => {
  setupPasteListener(false)
})

const generateQRCode = async () => {
  if (!qrContent.value) {
    qrDataUrl.value = ''
    return
  }

  try {
    qrDataUrl.value = await QRCode.toDataURL(qrContent.value, {
      width: qrSize.value,
      margin: 2,
      errorCorrectionLevel: qrLevel.value,
      color: {
        dark: qrColor.value,
        light: '#ffffff'
      }
    })
  } catch (err) {
    ElMessage.error('生成失败: ' + err.message)
  }
}

const downloadQRCode = () => {
  if (!qrDataUrl.value) return

  const link = document.createElement('a')
  link.download = 'qrcode.png'
  link.href = qrDataUrl.value
  link.click()
  ElMessage.success('下载成功')
}

const copyQRCode = async () => {
  if (!qrDataUrl.value) return

  try {
    const response = await fetch(qrDataUrl.value)
    const blob = await response.blob()
    await navigator.clipboard.write([
      new ClipboardItem({ 'image/png': blob })
    ])
    ElMessage.success('已复制到剪贴板')
  } catch (err) {
    ElMessage.error('复制失败，请手动保存图片')
  }
}

const handleFileChange = (file) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    previewImage.value = e.target.result
    parseQRCode(e.target.result)
  }
  reader.readAsDataURL(file.raw)
}

const parseQRCode = async (imageData) => {
  const results = []
  const triedMethods = []

  try {
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    const img = new Image()
    img.src = imageData
    await new Promise((resolve) => { img.onload = resolve })

    canvas.width = img.width
    canvas.height = img.height
    ctx.drawImage(img, 0, 0)

    // 获取图像数据
    const imageDataObj = ctx.getImageData(0, 0, canvas.width, canvas.height)

    // 预处理函数：尝试多种图像处理方式
    const preprocessMethods = [
      { name: '原始', fn: () => ctx.getImageData(0, 0, canvas.width, canvas.height) },
      { name: '灰度', fn: () => toGrayscale(imageDataObj) },
      { name: '二值化127', fn: () => toBinary(imageDataObj, 127) },
      { name: '二值化100', fn: () => toBinary(imageDataObj, 100) },
      { name: '二值化80', fn: () => toBinary(imageDataObj, 80) },
      { name: '二值化60', fn: () => toBinary(imageDataObj, 60) },
      { name: '反色', fn: () => invertColors(imageDataObj) },
      { name: '高对比度', fn: () => highContrast(imageDataObj) },
      { name: '亮度增强', fn: () => brightness(imageDataObj, 1.3) },
      { name: '平滑', fn: () => smooth(imageDataObj) },
    ]

    // 尝试每种预处理方法
    for (const method of preprocessMethods) {
      try {
        const processedData = method.fn()
        ctx.putImageData(processedData, 0, 0)

        // 使用 jsQR 检测多个二维码
        const codes = detectQRCodes(ctx, canvas.width, canvas.height)
        for (const code of codes) {
          if (code.data && !results.includes(code.data)) {
            results.push(code.data)
          }
        }
      } catch (e) {
        // 继续尝试下一个方法
      }
      // 重绘原图
      ctx.drawImage(img, 0, 0)
    }

    // 总是调用 OCR 服务来检测二维码（它使用 pyzbar 更准确）
    try {
      const base64 = canvas.toDataURL('image/jpeg', 0.8).split(',')[1]
      const response = await fetch('/api/ocr', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ image: base64 })
      })
      const ocrResult = await response.json()

      // 优先处理二维码检测结果
      if (ocrResult.qr_codes && ocrResult.qr_codes.length > 0) {
        for (const qr of ocrResult.qr_codes) {
          if (qr.data && !results.includes(qr.data)) {
            results.push(qr.data)
          }
        }
      }

      // 如果没有二维码，再尝试 OCR 文字结果
      if (results.length === 0 && ocrResult.text) {
        const texts = Array.isArray(ocrResult.text) ? ocrResult.text : [ocrResult.text]
        for (const text of texts) {
          if (text && text.trim().length > 0) {
            results.push(text.trim())
          }
        }
      }
    } catch (e) {
      // OCR 不可用
    }

    if (results.length > 0) {
      parsedText.value = results.join('\n\n--- 分隔 ---\n\n')
      ElMessage.success(`解析成功，发现 ${results.length} 个二维码`)
    } else {
      parsedText.value = ''
      ElMessage.error('未能识别出二维码，请确保图片清晰')
    }
  } catch (err) {
    parsedText.value = null
    ElMessage.error('解析失败: ' + err.message)
  }
}

// 图像处理辅助函数
function toGrayscale(imageData) {
  const data = imageData.data
  for (let i = 0; i < data.length; i += 4) {
    const gray = data[i] * 0.299 + data[i + 1] * 0.587 + data[i + 2] * 0.114
    data[i] = data[i + 1] = data[i + 2] = gray
  }
  return imageData
}

function toBinary(imageData, threshold) {
  const data = imageData.data
  for (let i = 0; i < data.length; i += 4) {
    const gray = data[i] * 0.299 + data[i + 1] * 0.587 + data[i + 2] * 0.114
    const binary = gray > threshold ? 255 : 0
    data[i] = data[i + 1] = data[i + 2] = binary
  }
  return imageData
}

function invertColors(imageData) {
  const data = imageData.data
  for (let i = 0; i < data.length; i += 4) {
    data[i] = 255 - data[i]
    data[i + 1] = 255 - data[i + 1]
    data[i + 2] = 255 - data[i + 2]
  }
  return imageData
}

function highContrast(imageData) {
  const data = imageData.data
  for (let i = 0; i < data.length; i += 4) {
    for (let j = 0; j < 3; j++) {
      data[i + j] = data[i + j] > 128 ? Math.min(255, data[i + j] * 1.4) : Math.max(0, data[i + j] * 0.6)
    }
  }
  return imageData
}

function brightness(imageData, factor) {
  const data = imageData.data
  for (let i = 0; i < data.length; i += 4) {
    for (let j = 0; j < 3; j++) {
      data[i + j] = Math.min(255, Math.max(0, data[i + j] * factor))
    }
  }
  return imageData
}

function smooth(imageData) {
  const data = imageData.data
  const width = imageData.width
  const height = imageData.height
  const result = new Uint8ClampedArray(data)

  for (let y = 1; y < height - 1; y++) {
    for (let x = 1; x < width - 1; x++) {
      for (let c = 0; c < 3; c++) {
        let sum = 0
        for (let dy = -1; dy <= 1; dy++) {
          for (let dx = -1; dx <= 1; dx++) {
            sum += data[((y + dy) * width + (x + dx)) * 4 + c]
          }
        }
        result[(y * width + x) * 4 + c] = sum / 9
      }
    }
  }

  return new ImageData(result, width, height)
}

// 检测多个二维码
function detectQRCodes(ctx, width, height) {
  const results = []
  try {
    // 方法1: 尝试直接检测
    try {
      const code = QRCode.from(ctx.canvas).decode()
      if (code) results.push({ data: code })
    } catch (e) {}

    // 方法2: 尝试使用 jsQR (如果可用)
    if (typeof jsQR !== 'undefined') {
      try {
        const imageData = ctx.getImageData(0, 0, width, height)
        const code = jsQR(imageData.data, width, height)
        if (code && code.data) results.push({ data: code.data })
      } catch (e) {}
    }

    // 方法3: 尝试在图像区域中寻找二维码（切分图像）
    const gridSizes = [2, 3, 4] // 将图像切分成多个区域
    for (const gridSize of gridSizes) {
      const cellWidth = Math.floor(width / gridSize)
      const cellHeight = Math.floor(height / gridSize)

      for (let row = 0; row < gridSize; row++) {
        for (let col = 0; col < gridSize; col++) {
          try {
            const sx = col * cellWidth
            const sy = row * cellHeight
            const sw = Math.min(cellWidth, width - sx)
            const sh = Math.min(cellHeight, height - sy)

            const tempCanvas = document.createElement('canvas')
            tempCanvas.width = sw
            tempCanvas.height = sh
            const tempCtx = tempCanvas.getContext('2d')
            tempCtx.drawImage(ctx.canvas, sx, sy, sw, sh, 0, 0, sw, sh)

            const code = QRCode.from(tempCanvas).decode()
            if (code && !results.find(r => r.data === code)) {
              results.push({ data: code })
            }
          } catch (e) {
            // 继续尝试下一个区域
          }
        }
      }
    }
  } catch (e) {}

  return results
}

const copyParsedText = async () => {
  if (!parsedText.value) return

  try {
    await navigator.clipboard.writeText(parsedText.value)
    ElMessage.success('已复制')
  } catch (err) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.tool-tabs {
  flex: 1;
}

.generate-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 30px;
  padding: 20px;
}

.input-section {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.options-row {
  display: flex;
  gap: 15px;
  align-items: center;
}

.preview-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.qr-preview {
  padding: 20px;
  background: white;
  border-radius: var(--radius-md);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.qr-preview img {
  display: block;
  max-width: 100%;
}

.qr-placeholder {
  width: 300px;
  height: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--card-bg);
  border: 2px dashed var(--card-border);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
}

.qr-placeholder p {
  margin-top: 10px;
}

.action-buttons {
  display: flex;
  gap: 10px;
}

.parse-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px;
  max-width: 600px;
  margin: 0 auto;
}

.upload-section {
  width: 100%;
}

.paste-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: 14px;
}

.qr-upload {
  width: 100%;
}

.upload-area {
  width: 100%;
  height: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 2px dashed var(--card-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color 0.3s;
}

.upload-area:hover {
  border-color: #409eff;
}

.upload-area p {
  margin-top: 10px;
  color: var(--text-secondary);
}

.parse-preview {
  display: flex;
  justify-content: center;
  padding: 20px;
  background: white;
  border-radius: var(--radius-md);
}

.parse-preview img {
  max-width: 300px;
  max-height: 300px;
}

.parse-result {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}
</style>
