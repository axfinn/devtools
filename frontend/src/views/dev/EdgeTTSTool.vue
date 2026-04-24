<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Edge TTS 语音合成</h2>
      <p class="tool-desc">基于 Microsoft Edge Neural TTS 的文本转语音工具，支持多种中文音色</p>
    </div>

    <div class="main-content">
      <!-- 左侧：输入控制 -->
      <div class="control-panel">
        <el-card class="panel-card">
          <template #header>
            <div class="panel-header">
              <span>语音合成</span>
              <el-button text @click="loadVoices" :loading="loadingVoices">
                刷新音色
              </el-button>
            </div>
          </template>

          <el-form label-position="top">
            <el-form-item label="音色选择">
              <el-select v-model="selectedVoice" placeholder="请选择音色" class="w-full" filterable>
                <el-option-group label="女声">
                  <el-option
                    v-for="voice in femaleVoices"
                    :key="voice.id"
                    :value="voice.id"
                    :label="`${voice.name} (${voice.style || '默认'})`"
                  >
                    <div class="voice-option">
                      <span class="voice-name">{{ voice.name }}</span>
                      <span class="voice-style">{{ voice.style || '默认' }}</span>
                    </div>
                  </el-option>
                </el-option-group>
                <el-option-group label="男声">
                  <el-option
                    v-for="voice in maleVoices"
                    :key="voice.id"
                    :value="voice.id"
                    :label="`${voice.name} (${voice.style || '默认'})`"
                  >
                    <div class="voice-option">
                      <span class="voice-name">{{ voice.name }}</span>
                      <span class="voice-style">{{ voice.style || '默认' }}</span>
                    </div>
                  </el-option>
                </el-option-group>
              </el-select>
            </el-form-item>

            <el-form-item label="输入文本">
              <el-input
                v-model="text"
                type="textarea"
                :rows="6"
                placeholder="请输入要转换的文本..."
                maxlength="500"
                show-word-limit
              />
            </el-form-item>

            <div class="action-buttons">
              <el-button type="primary" @click="synthesize" :loading="synthesizing" :disabled="!text.trim()">
                <el-icon><VideoPlay /></el-icon>
                合成试听
              </el-button>
              <el-button @click="clearAll">清空</el-button>
            </div>
          </el-form>
        </el-card>

        <!-- 健康状态 -->
        <el-card class="panel-card health-card">
          <template #header>
            <div class="panel-header">
              <span>服务状态</span>
            </div>
          </template>
          <div class="health-status">
            <el-tag :type="healthStatus.ok ? 'success' : 'danger'">
              {{ healthStatus.ok ? '服务正常' : '服务不可用' }}
            </el-tag>
            <span class="health-url">{{ healthStatus.url || ttsServiceURL }}</span>
          </div>
        </el-card>
      </div>

      <!-- 右侧：音频播放和下载 -->
      <div class="result-panel">
        <el-card v-if="audioUrl" class="panel-card">
          <template #header>
            <div class="panel-header">
              <span>试听结果</span>
              <div class="audio-info">
                <el-tag size="small" type="info">{{ currentVoiceName }}</el-tag>
              </div>
            </div>
          </template>

          <div class="audio-player">
            <audio ref="audioPlayer" :src="audioUrl" controls autoplay class="audio-element"></audio>
          </div>

          <div class="download-section">
            <div class="download-title">下载音频</div>
            <div class="download-buttons">
              <el-button type="primary" @click="downloadMp3">
                <el-icon><Download /></el-icon>
                下载 MP3
              </el-button>
              <el-button type="success" @click="downloadWav" :loading="converting">
                <el-icon><Download /></el-icon>
                下载 WAV
              </el-button>
            </div>
            <div class="download-tip">
              <el-alert type="info" :closable="false" show-icon>
                <template #title>
                  WAV 格式通过 ffmpeg 转换，文件体积较大但兼容性好
                </template>
              </el-alert>
            </div>
          </div>
        </el-card>

        <el-card v-else class="panel-card empty-card">
          <div class="empty-state">
            <el-icon class="empty-icon"><Headset /></el-icon>
            <p>输入文本并选择音色后，点击"合成试听"生成语音</p>
          </div>
        </el-card>

        <!-- 音色说明 -->
        <el-card class="panel-card">
          <template #header>
            <div class="panel-header">
              <span>音色说明</span>
            </div>
          </template>
          <div class="voice-list">
            <div v-for="voice in voices" :key="voice.id" class="voice-item">
              <span class="voice-badge" :class="voice.gender">{{ voice.name }}</span>
              <span class="voice-desc">{{ voice.style || '默认音色' }}</span>
            </div>
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { VideoPlay, Download, Headset } from '@element-plus/icons-vue'

const API_BASE = '/api/edge-tts'

// 状态
const text = ref('')
const selectedVoice = ref('zh-CN-XiaoxiaoNeural')
const voices = ref([])
const audioUrl = ref('')
const currentVoiceName = ref('')
const loadingVoices = ref(false)
const synthesizing = ref(false)
const converting = ref(false)
const audioPlayer = ref(null)
const healthStatus = ref({ ok: false, url: '' })

// 固定的 TTS 服务地址（与 chat 服务共用）
const ttsServiceURL = 'http://127.0.0.1:8083'

// 计算属性：女声和男声
const femaleVoices = computed(() => voices.value.filter(v => v.gender === 'female'))
const maleVoices = computed(() => voices.value.filter(v => v.gender === 'male'))

// 加载音色列表
const loadVoices = async () => {
  loadingVoices.value = true
  try {
    const resp = await fetch(`${API_BASE}/voices`)
    if (!resp.ok) throw new Error('获取音色列表失败')
    const data = await resp.json()
    voices.value = data.voices || []
    if (voices.value.length > 0 && !selectedVoice.value) {
      selectedVoice.value = voices.value[0].id
    }
  } catch (err) {
    console.error('加载音色失败:', err)
    ElMessage.error('加载音色列表失败，可能 TTS 服务未启动')
  } finally {
    loadingVoices.value = false
  }
}

// 检查服务健康状态
const checkHealth = async () => {
  try {
    const resp = await fetch(`${API_BASE}/health`)
    if (resp.ok) {
      const data = await resp.json()
      healthStatus.value = { ok: data.edge_tts !== false, url: data.url }
    } else {
      healthStatus.value = { ok: false, url: ttsServiceURL }
    }
  } catch (err) {
    healthStatus.value = { ok: false, url: ttsServiceURL }
  }
}

// 合成语音
const synthesize = async () => {
  if (!text.value.trim()) {
    ElMessage.warning('请输入要转换的文本')
    return
  }

  if (!selectedVoice.value) {
    ElMessage.warning('请选择音色')
    return
  }

  synthesizing.value = true
  try {
    const resp = await fetch(`${API_BASE}/tts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: text.value,
        voice: selectedVoice.value
      })
    })

    if (!resp.ok) {
      const err = await resp.json()
      throw new Error(err.error || '合成失败')
    }

    const data = await resp.json()
    // 音频文件 URL，需要拼接成完整路径
    audioUrl.value = `/api/chat/uploads/${data.filename}`
    currentVoiceName.value = voices.value.find(v => v.id === selectedVoice.value)?.name || selectedVoice.value
    ElMessage.success('合成成功')
  } catch (err) {
    console.error('合成失败:', err)
    ElMessage.error(err.message || '合成失败，请检查 TTS 服务是否启动')
  } finally {
    synthesizing.value = false
  }
}

// 下载 MP3
const downloadMp3 = () => {
  if (!audioUrl.value) {
    ElMessage.warning('请先合成语音')
    return
  }

  const link = document.createElement('a')
  link.href = audioUrl.value
  link.download = `voice_${Date.now()}.mp3`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  ElMessage.success('MP3 下载已开始')
}

// 下载 WAV
const downloadWav = async () => {
  if (!audioUrl.value) {
    ElMessage.warning('请先合成语音')
    return
  }

  converting.value = true
  try {
    const resp = await fetch(`${API_BASE}/convert`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        source_url: audioUrl.value,
        format: 'wav'
      })
    })

    if (!resp.ok) {
      const err = await resp.json()
      throw new Error(err.error || '转换失败')
    }

    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `voice_${Date.now()}.wav`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    ElMessage.success('WAV 下载已开始')
  } catch (err) {
    console.error('WAV 转换失败:', err)
    ElMessage.error(err.message || 'WAV 转换失败')
  } finally {
    converting.value = false
  }
}

// 清空
const clearAll = () => {
  text.value = ''
  audioUrl.value = ''
  currentVoiceName.value = ''
  if (audioPlayer.value) {
    audioPlayer.value.pause()
  }
}

// 初始化
onMounted(() => {
  loadVoices()
  checkHealth()
})
</script>

<style scoped>
.tool-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
}

.tool-header h2 {
  font-size: 24px;
  color: var(--color-primary);
  margin-bottom: 8px;
}

.tool-desc {
  color: var(--text-secondary);
  font-size: 14px;
  margin-bottom: 20px;
}

.main-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.panel-card {
  margin-bottom: 16px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.w-full {
  width: 100%;
}

.voice-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.voice-name {
  font-weight: 500;
}

.voice-style {
  color: var(--text-secondary);
  font-size: 12px;
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}

.health-card {
  background: var(--bg-secondary);
}

.health-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.health-url {
  color: var(--text-secondary);
  font-size: 12px;
  font-family: var(--font-family-mono);
}

.audio-player {
  margin: 20px 0;
}

.audio-element {
  width: 100%;
  height: 40px;
}

.download-section {
  border-top: 1px solid var(--border-base);
  padding-top: 20px;
}

.download-title {
  font-weight: 500;
  margin-bottom: 12px;
  color: var(--text-primary);
}

.download-buttons {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.download-tip {
  margin-top: 12px;
}

.empty-card {
  min-height: 200px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 150px;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.voice-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.voice-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.voice-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.voice-badge.female {
  background: #fce7f3;
  color: #be185d;
}

.voice-badge.male {
  background: #dbeafe;
  color: #1d4ed8;
}

.voice-desc {
  color: var(--text-secondary);
  font-size: 12px;
}

/* 响应式 */
@media (max-width: 900px) {
  .main-content {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .tool-header h2 {
    font-size: 20px;
  }

  .action-buttons {
    flex-direction: column;
  }

  .download-buttons {
    flex-direction: column;
  }
}
</style>
