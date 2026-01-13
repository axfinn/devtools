<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>文本转换工具</h2>
    </div>

    <el-tabs v-model="activeTab" class="tool-tabs">
      <!-- 八进制转义 -->
      <el-tab-pane label="八进制转义" name="octal">
        <div class="tab-content">
          <div class="editor-panel">
            <div class="panel-header">
              输入文本
              <div class="feature-hints">
                <el-tag size="small">支持 \344\270\215 格式</el-tag>
              </div>
            </div>
            <textarea
              v-model="octalInput"
              class="code-editor"
              placeholder="粘贴包含八进制转义序列的文本，如：\344\270\215\346\224\257\346\214\201\344\272\214\347\273\264\347\240\201"
              spellcheck="false"
              rows="8"
            ></textarea>
          </div>
          <div class="button-group">
            <el-button type="primary" @click="decodeOctal">
              解码 ↓
            </el-button>
            <el-button @click="encodeOctal">
              编码 ↑
            </el-button>
            <el-button @click="clearOctal">清空</el-button>
          </div>
          <div class="editor-panel">
            <div class="panel-header">
              转换结果
              <el-button size="small" @click="copyOctalResult">复制</el-button>
            </div>
            <textarea
              v-model="octalOutput"
              class="code-editor"
              placeholder="转换后的文本..."
              spellcheck="false"
              rows="8"
            ></textarea>
          </div>

          <div class="tips-section">
            <el-alert type="info" :closable="false">
              <template #title>
                <div class="tips-title">使用说明</div>
              </template>
              <ul class="tips-list">
                <li><strong>解码：</strong>将 <code>\344\270\215\346\224\257\346\214\201</code> 转换为 "不支持"</li>
                <li><strong>编码：</strong>将中文转换为八进制转义序列</li>
                <li><strong>支持：</strong>UTF-8 编码的所有字符</li>
                <li><strong>常见场景：</strong>日志输出、protobuf 调试、Go/Python 字符串转义</li>
              </ul>
            </el-alert>
          </div>
        </div>
      </el-tab-pane>

      <!-- Unicode 转义 -->
      <el-tab-pane label="Unicode 转义" name="unicode">
        <div class="tab-content">
          <div class="editor-panel">
            <div class="panel-header">
              输入文本
              <div class="feature-hints">
                <el-tag size="small">支持 \u4e0d 或 \U00004e0d 格式</el-tag>
              </div>
            </div>
            <textarea
              v-model="unicodeInput"
              class="code-editor"
              placeholder="粘贴包含 Unicode 转义的文本，如：\u4e0d\u652f\u6301"
              spellcheck="false"
              rows="8"
            ></textarea>
          </div>
          <div class="button-group">
            <el-button type="primary" @click="decodeUnicode">
              解码 ↓
            </el-button>
            <el-button @click="encodeUnicode">
              编码 ↑
            </el-button>
            <el-button @click="clearUnicode">清空</el-button>
          </div>
          <div class="editor-panel">
            <div class="panel-header">
              转换结果
              <el-button size="small" @click="copyUnicodeResult">复制</el-button>
            </div>
            <textarea
              v-model="unicodeOutput"
              class="code-editor"
              placeholder="转换后的文本..."
              spellcheck="false"
              rows="8"
            ></textarea>
          </div>

          <div class="tips-section">
            <el-alert type="info" :closable="false">
              <template #title>
                <div class="tips-title">使用说明</div>
              </template>
              <ul class="tips-list">
                <li><strong>解码：</strong>将 <code>\u4e0d\u652f\u6301</code> 转换为 "不支持"</li>
                <li><strong>编码：</strong>将文本转换为 Unicode 转义序列</li>
                <li><strong>支持格式：</strong>\uXXXX (4位) 和 \UXXXXXXXX (8位)</li>
                <li><strong>常见场景：</strong>JSON 转义、JavaScript 字符串、Java 字符串</li>
              </ul>
            </el-alert>
          </div>
        </div>
      </el-tab-pane>

      <!-- 十六进制转义 -->
      <el-tab-pane label="十六进制转义" name="hex">
        <div class="tab-content">
          <div class="editor-panel">
            <div class="panel-header">
              输入文本
              <div class="feature-hints">
                <el-tag size="small">支持 \xe4\xb8\x8d 格式</el-tag>
              </div>
            </div>
            <textarea
              v-model="hexInput"
              class="code-editor"
              placeholder="粘贴包含十六进制转义的文本，如：\xe4\xb8\x8d\xe6\x94\xaf\xe6\x8c\x81"
              spellcheck="false"
              rows="8"
            ></textarea>
          </div>
          <div class="button-group">
            <el-button type="primary" @click="decodeHex">
              解码 ↓
            </el-button>
            <el-button @click="encodeHex">
              编码 ↑
            </el-button>
            <el-button @click="clearHex">清空</el-button>
          </div>
          <div class="editor-panel">
            <div class="panel-header">
              转换结果
              <el-button size="small" @click="copyHexResult">复制</el-button>
            </div>
            <textarea
              v-model="hexOutput"
              class="code-editor"
              placeholder="转换后的文本..."
              spellcheck="false"
              rows="8"
            ></textarea>
          </div>

          <div class="tips-section">
            <el-alert type="info" :closable="false">
              <template #title>
                <div class="tips-title">使用说明</div>
              </template>
              <ul class="tips-list">
                <li><strong>解码：</strong>将 <code>\xe4\xb8\x8d</code> 转换为 "不"</li>
                <li><strong>编码：</strong>将文本转换为十六进制转义序列</li>
                <li><strong>格式：</strong>每个字节用 \xXX 表示</li>
                <li><strong>常见场景：</strong>C/C++ 字符串、Python bytes、Shell 脚本</li>
              </ul>
            </el-alert>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <div v-if="errorMsg" class="error-msg">
      <el-alert :title="errorMsg" type="error" show-icon @close="errorMsg = ''" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

const activeTab = ref('octal')

// 八进制转义相关
const octalInput = ref('')
const octalOutput = ref('')

// Unicode 转义相关
const unicodeInput = ref('')
const unicodeOutput = ref('')

// 十六进制转义相关
const hexInput = ref('')
const hexOutput = ref('')

const errorMsg = ref('')

// ===== 八进制转义功能 =====
const decodeOctal = () => {
  try {
    errorMsg.value = ''
    const input = octalInput.value

    // 匹配 \xxx 格式的八进制转义序列
    const result = input.replace(/\\(\d{3})/g, (match, octal) => {
      const byte = parseInt(octal, 8)
      return String.fromCharCode(byte)
    })

    // 尝试将字节序列转换为 UTF-8 字符串
    octalOutput.value = decodeUTF8Bytes(result)
    ElMessage.success('解码成功')
  } catch (e) {
    errorMsg.value = '解码错误: ' + e.message
    ElMessage.error('解码失败')
  }
}

const encodeOctal = () => {
  try {
    errorMsg.value = ''
    const input = octalOutput.value

    // 将字符串编码为 UTF-8 字节，然后转换为八进制转义
    const bytes = encodeUTF8Bytes(input)
    const result = bytes.map(byte => {
      const octal = byte.toString(8).padStart(3, '0')
      return '\\' + octal
    }).join('')

    octalInput.value = result
    ElMessage.success('编码成功')
  } catch (e) {
    errorMsg.value = '编码错误: ' + e.message
    ElMessage.error('编码失败')
  }
}

const clearOctal = () => {
  octalInput.value = ''
  octalOutput.value = ''
  errorMsg.value = ''
}

const copyOctalResult = async () => {
  try {
    await navigator.clipboard.writeText(octalOutput.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

// ===== Unicode 转义功能 =====
const decodeUnicode = () => {
  try {
    errorMsg.value = ''
    let input = unicodeInput.value

    // 处理 \uXXXX 格式
    input = input.replace(/\\u([0-9a-fA-F]{4})/g, (match, hex) => {
      return String.fromCharCode(parseInt(hex, 16))
    })

    // 处理 \UXXXXXXXX 格式
    input = input.replace(/\\U([0-9a-fA-F]{8})/g, (match, hex) => {
      const codePoint = parseInt(hex, 16)
      return String.fromCodePoint(codePoint)
    })

    unicodeOutput.value = input
    ElMessage.success('解码成功')
  } catch (e) {
    errorMsg.value = '解码错误: ' + e.message
    ElMessage.error('解码失败')
  }
}

const encodeUnicode = () => {
  try {
    errorMsg.value = ''
    const input = unicodeOutput.value

    let result = ''
    for (let i = 0; i < input.length; i++) {
      const char = input[i]
      const code = char.charCodeAt(0)

      // 只转义非 ASCII 字符
      if (code > 127) {
        result += '\\u' + code.toString(16).padStart(4, '0')
      } else {
        result += char
      }
    }

    unicodeInput.value = result
    ElMessage.success('编码成功')
  } catch (e) {
    errorMsg.value = '编码错误: ' + e.message
    ElMessage.error('编码失败')
  }
}

const clearUnicode = () => {
  unicodeInput.value = ''
  unicodeOutput.value = ''
  errorMsg.value = ''
}

const copyUnicodeResult = async () => {
  try {
    await navigator.clipboard.writeText(unicodeOutput.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

// ===== 十六进制转义功能 =====
const decodeHex = () => {
  try {
    errorMsg.value = ''
    const input = hexInput.value

    // 匹配 \xXX 格式的十六进制转义序列
    const result = input.replace(/\\x([0-9a-fA-F]{2})/g, (match, hex) => {
      const byte = parseInt(hex, 16)
      return String.fromCharCode(byte)
    })

    // 尝试将字节序列转换为 UTF-8 字符串
    hexOutput.value = decodeUTF8Bytes(result)
    ElMessage.success('解码成功')
  } catch (e) {
    errorMsg.value = '解码错误: ' + e.message
    ElMessage.error('解码失败')
  }
}

const encodeHex = () => {
  try {
    errorMsg.value = ''
    const input = hexOutput.value

    // 将字符串编码为 UTF-8 字节，然后转换为十六进制转义
    const bytes = encodeUTF8Bytes(input)
    const result = bytes.map(byte => {
      const hex = byte.toString(16).padStart(2, '0')
      return '\\x' + hex
    }).join('')

    hexInput.value = result
    ElMessage.success('编码成功')
  } catch (e) {
    errorMsg.value = '编码错误: ' + e.message
    ElMessage.error('编码失败')
  }
}

const clearHex = () => {
  hexInput.value = ''
  hexOutput.value = ''
  errorMsg.value = ''
}

const copyHexResult = async () => {
  try {
    await navigator.clipboard.writeText(hexOutput.value)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

// ===== 辅助函数 =====
// 将 UTF-8 字节序列解码为字符串
function decodeUTF8Bytes(str) {
  const bytes = []
  for (let i = 0; i < str.length; i++) {
    bytes.push(str.charCodeAt(i) & 0xFF)
  }

  // 使用 TextDecoder 解码 UTF-8
  const uint8Array = new Uint8Array(bytes)
  const decoder = new TextDecoder('utf-8')
  return decoder.decode(uint8Array)
}

// 将字符串编码为 UTF-8 字节数组
function encodeUTF8Bytes(str) {
  const encoder = new TextEncoder()
  const uint8Array = encoder.encode(str)
  return Array.from(uint8Array)
}
</script>

<style scoped>
.tool-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.tool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.tool-header h2 {
  margin: 0;
  color: #409eff;
  font-size: 24px;
}

.tool-tabs {
  background: #1e1e1e;
  border-radius: 8px;
  padding: 20px;
}

.tab-content {
  display: flex;
  flex-direction: column;
  gap: 15px;
  margin-top: 20px;
}

.editor-panel {
  background: #252525;
  border-radius: 8px;
  padding: 15px;
  border: 1px solid #333;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: #909399;
  font-size: 14px;
  font-weight: 500;
}

.feature-hints {
  display: flex;
  gap: 8px;
  align-items: center;
}

.code-editor {
  width: 100%;
  background: #1a1a1a;
  color: #e0e0e0;
  border: 1px solid #333;
  border-radius: 4px;
  padding: 12px;
  font-family: 'Courier New', Consolas, Monaco, monospace;
  font-size: 14px;
  line-height: 1.6;
  resize: vertical;
  min-height: 120px;
}

.code-editor:focus {
  outline: none;
  border-color: #409eff;
}

.button-group {
  display: flex;
  gap: 10px;
  justify-content: center;
  align-items: center;
}

.tips-section {
  margin-top: 20px;
  background: #1e1e1e;
  border-radius: 8px;
  padding: 15px;
}

.tips-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 8px;
}

.tips-list {
  margin: 10px 0 0 0;
  padding-left: 20px;
  line-height: 1.8;
}

.tips-list li {
  margin: 6px 0;
  color: #909399;
}

.tips-list code {
  background: #252525;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', Consolas, Monaco, monospace;
  color: #409eff;
}

.error-msg {
  margin-top: 15px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .tool-header h2 {
    font-size: 20px;
  }

  .tool-tabs {
    padding: 15px;
  }

  .code-editor {
    font-size: 13px;
    min-height: 150px;
  }

  .button-group {
    flex-wrap: wrap;
  }

  .button-group .el-button {
    flex: 1;
    min-width: 80px;
  }

  .feature-hints {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
