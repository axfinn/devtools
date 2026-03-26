<template>
  <div class="fortune-game">
    <div class="game-header">
      <h2>🎊 今日运势</h2>
      <button @click="drawFortune" class="btn-primary" :disabled="loading">
        {{ loading ? '🤔 抽签中...' : (drawn ? '再抽一次' : '开始抽签') }}
      </button>
    </div>

    <div class="fortune-content" v-if="fortune">
      <div class="fortune-card" :class="fortune.level">
        <div class="fortune-icon">{{ fortune.icon }}</div>
        <h3>{{ fortune.title }}</h3>
        <p class="fortune-level">{{ fortune.levelText }}</p>
      </div>

      <div class="fortune-details">
        <div class="detail-item">
          <span class="label">💡 今日提示</span>
          <span class="value">{{ fortune.tip }}</span>
        </div>
        <div class="detail-item">
          <span class="label">🍀 幸运数字</span>
          <span class="value">{{ fortune.luckyNumber }}</span>
        </div>
        <div class="detail-item">
          <span class="label">🎨 幸运颜色</span>
          <span class="value">{{ fortune.luckyColor }}</span>
        </div>
        <div class="detail-item">
          <span class="label">⏰ 幸运时段</span>
          <span class="value">{{ fortune.luckyTime }}</span>
        </div>
      </div>

      <div class="speaker-btn" @click="speakFortune" :class="{ speaking: isSpeaking }">
        {{ isSpeaking ? '🔊 朗读中...' : '🔊 点击朗读运势' }}
      </div>

      <div class="ai-comment" v-if="fortune.comment">
        <h4>📝 AI 点评</h4>
        <p>{{ fortune.comment }}</p>
      </div>
    </div>

    <div class="draw-area" v-else>
      <div class="lottery-box" @click="drawFortune" :class="{ shaking: loading }">
        <span class="box-icon">🎁</span>
        <p>{{ loading ? '抽取中...' : '点击抽签' }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'

const API_BASE = '/api/game/ai-chat'
const TTS_API = '/api/edge-tts/tts'

const loading = ref(false)
const isSpeaking = ref(false)
const drawn = ref(false)
const fortune = ref(null)

const fortuneLevels = [
  { level: 'excellent', icon: '🌟', title: '大吉', levelText: '今日运势极佳！' },
  { level: 'good', icon: '😊', title: '吉', levelText: '今日运势不错！' },
  { level: 'average', icon: '😐', title: '中平', levelText: '今日运势一般' },
  { level: 'bad', icon: '😔', title: '小凶', levelText: '今日运势欠佳' }
]

const fetchTTSUrl = async (text) => {
  const voice = 'zh-CN-XiaoxiaoNeural'
  const res = await fetch(TTS_API, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, voice })
  })
  const data = await res.json()
  return data.url
}

const speakFortune = async () => {
  if (!fortune.value || isSpeaking.value) return

  isSpeaking.value = true
  try {
    const text = `${fortune.value.title}！今日运势：${fortune.value.levelText}。${fortune.value.tip}。幸运数字是${fortune.value.luckyNumber}。`
    const audioUrl = await fetchTTSUrl(text)
    const audio = new Audio(audioUrl)
    audio.onended = () => { isSpeaking.value = false }
    audio.onerror = () => { isSpeaking.value = false }
    audio.play()
  } catch (e) {
    console.error('TTS failed', e)
    isSpeaking.value = false
  }
}

const drawFortune = async () => {
  loading.value = true

  try {
    // 随机选择一个运势级别
    const levelInfo = fortuneLevels[Math.floor(Math.random() * fortuneLevels.length)]

    // 请求 AI 生成详细运势
    const res = await fetch(API_BASE, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        game: 'fortune',
        message: `以"${levelInfo.title}"为主题，写今日运势，包含：1. 一句运势描述（20字内）2. 一个幸运数字（1-99）3. 一个幸运颜色 4. 一个幸运时段（如上午9-11点）5. 一句今日提示（30字内）6. 一句AI点评（50字内，鼓励的话）`,
        history: []
      })
    })
    const data = await res.json()
    const response = data.response

    // 解析 AI 回复
    const lines = response.split('\n').filter(l => l.trim())
    const fortuneData = {
      ...levelInfo,
      luckyNumber: '',
      luckyColor: '',
      luckyTime: '',
      tip: '',
      comment: ''
    }

    for (const line of lines) {
      if (line.includes('数字')) {
        const num = line.match(/\d+/)
        fortuneData.luckyNumber = num ? num[0] : String(Math.floor(Math.random() * 99) + 1)
      } else if (line.includes('颜色')) {
        const colors = ['红色', '蓝色', '绿色', '黄色', '紫色', '粉色', '白色', '黑色', '金色', '银色']
        fortuneData.luckyColor = colors[Math.floor(Math.random() * colors.length)]
      } else if (line.includes('时段') || line.includes('时间')) {
        fortuneData.luckyTime = line.replace(/[^0-9-点时分]/g, '').trim() || '上午9-11点'
      } else if (line.includes('提示') || line.includes('建议')) {
        fortuneData.tip = line.replace(/[^a-zA-Z\u4e00-\u9fa5，。！]/g, '').trim()
      } else if (line.includes('点评') || line.includes('鼓励')) {
        fortuneData.comment = line.replace(/[^a-zA-Z\u4e00-\u9fa5，。！]/g, '').trim()
      }
    }

    // 如果解析不完整，生成默认值
    if (!fortuneData.tip) {
      fortuneData.tip = fortuneData.level === 'excellent' ? '今天是心想事成的一天！' :
                       fortuneData.level === 'good' ? '保持积极，好运连连' :
                       fortuneData.level === 'average' ? '平常心对待' : '低调行事，注意安全'
    }
    if (!fortuneData.luckyNumber) {
      fortuneData.luckyNumber = String(Math.floor(Math.random() * 99) + 1)
    }
    if (!fortuneData.luckyColor) {
      fortuneData.luckyColor = ['红色', '蓝色', '绿色'][Math.floor(Math.random() * 3)]
    }
    if (!fortuneData.luckyTime) {
      fortuneData.luckyTime = ['上午9-11点', '下午2-4点', '晚上7-9点'][Math.floor(Math.random() * 3)]
    }
    if (!fortuneData.comment) {
      fortuneData.comment = '相信自己，你是最棒的！'
    }

    fortune.value = fortuneData
    drawn.value = true

    // 自动朗读
    setTimeout(() => speakFortune(), 500)
  } catch (e) {
    console.error('Failed to draw fortune', e)
    // 使用默认运势
    fortune.value = {
      ...fortuneLevels[0],
      luckyNumber: '8',
      luckyColor: '金色',
      luckyTime: '上午9-11点',
      tip: '今天是充满惊喜的一天！',
      comment: '保持开放的心态，好事即将发生！'
    }
    drawn.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.fortune-game {
  text-align: center;
  max-width: 600px;
  margin: 0 auto;
}

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.game-header h2 {
  margin: 0;
  color: #303133;
}

.btn-primary {
  padding: 10px 24px;
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
  border: none;
  border-radius: 20px;
  cursor: pointer;
  font-size: 1rem;
}

.btn-primary:hover:not(:disabled) {
  transform: scale(1.05);
}

.btn-primary:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}

.fortune-content {
  background: #f5f7fa;
  border-radius: 16px;
  padding: 30px;
}

.fortune-card {
  background: white;
  border-radius: 16px;
  padding: 30px;
  margin-bottom: 20px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.1);
}

.fortune-card.excellent {
  background: linear-gradient(135deg, #ffd700 0%, #ffed4e 100%);
}

.fortune-card.good {
  background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
}

.fortune-card.average {
  background: linear-gradient(135deg, #e0c3fc 0%, #8ec5fc 100%);
}

.fortune-card.bad {
  background: linear-gradient(135deg, #d4d4d4 0%, #a0a0a0 100%);
}

.fortune-icon {
  font-size: 4rem;
  margin-bottom: 10px;
}

.fortune-card h3 {
  font-size: 2rem;
  margin: 0 0 8px 0;
  color: #303133;
}

.fortune-level {
  color: #606266;
  font-size: 1.1rem;
}

.fortune-details {
  background: white;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
}

.detail-item:last-child {
  border-bottom: none;
}

.detail-item .label {
  color: #909399;
}

.detail-item .value {
  color: #303133;
  font-weight: bold;
}

.speaker-btn {
  display: inline-block;
  padding: 10px 24px;
  background: #fff;
  border: 2px solid #67c23a;
  border-radius: 20px;
  color: #67c23a;
  cursor: pointer;
  margin-bottom: 20px;
  transition: all 0.2s;
}

.speaker-btn:hover, .speaker-btn.speaking {
  background: #67c23a;
  color: white;
}

.ai-comment {
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  text-align: left;
}

.ai-comment h4 {
  margin: 0 0 8px 0;
  color: #409eff;
}

.ai-comment p {
  margin: 0;
  color: #606266;
  line-height: 1.6;
}

.draw-area {
  padding: 60px 0;
}

.lottery-box {
  display: inline-block;
  padding: 40px 60px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 20px;
  cursor: pointer;
  transition: transform 0.3s;
}

.lottery-box:hover {
  transform: scale(1.05);
}

.lottery-box.shaking {
  animation: shake 0.5s infinite;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

.box-icon {
  font-size: 4rem;
  display: block;
  margin-bottom: 10px;
}

.lottery-box p {
  margin: 0;
  color: white;
  font-size: 1.2rem;
}
</style>
