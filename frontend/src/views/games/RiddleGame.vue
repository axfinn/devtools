<template>
  <div class="riddle-game">
    <div class="game-header">
      <h2>🧩 谜语猜猜</h2>
      <button @click="newRiddle" class="btn-primary" :disabled="loading">
        {{ loading ? '🤔 AI出题中...' : '换一题' }}
      </button>
    </div>

    <div class="score-board">
      <span>答对: {{ correct }}</span>
      <span>答错: {{ wrong }}</span>
    </div>

    <div class="riddle-content" v-if="currentRiddle">
      <div class="speaker-btn" @click="speakRiddle" :class="{ speaking: isSpeaking }">
        {{ isSpeaking ? '🔊 朗读中...' : '🔊 点击朗读' }}
      </div>

      <p class="riddle-text">{{ currentRiddle.question }}</p>

      <div class="answer-section" v-if="showAnswer">
        <p class="answer-hint">答案是：<strong>{{ currentRiddle.answer }}</strong></p>
        <p class="answer-tip">{{ currentRiddle.hint }}</p>
      </div>

      <div class="input-section" v-else>
        <input
          type="text"
          v-model="userAnswer"
          placeholder="输入你的答案"
          @keyup.enter="submitAnswer"
          :disabled="loading"
        />
        <button @click="submitAnswer" :disabled="loading || !userAnswer.trim()">
          提交答案
        </button>
      </div>

      <div class="result-feedback" v-if="feedback">
        <span :class="feedback.type">{{ feedback.message }}</span>
      </div>
    </div>

    <div class="loading" v-else-if="loading">
      <p>🤔 AI正在想一个有趣的谜语...</p>
    </div>

    <div class="init-hint" v-else>
      <p>点击"换一题"开始猜谜语吧！</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'

const API_BASE = '/api/game/ai-chat'
const TTS_API = '/api/edge-tts/tts'

const loading = ref(false)
const isSpeaking = ref(false)
const currentRiddle = ref(null)
const userAnswer = ref('')
const showAnswer = ref(false)
const feedback = ref(null)
const correct = ref(0)
const wrong = ref(0)

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

const speakRiddle = async () => {
  if (!currentRiddle.value || isSpeaking.value) return

  isSpeaking.value = true
  try {
    const audioUrl = await fetchTTSUrl(currentRiddle.value.question)
    const audio = new Audio(audioUrl)
    audio.onended = () => { isSpeaking.value = false }
    audio.onerror = () => { isSpeaking.value = false }
    audio.play()
  } catch (e) {
    console.error('TTS failed', e)
    isSpeaking.value = false
  }
}

const newRiddle = async () => {
  loading.value = true
  feedback.value = null
  showAnswer.value = false
  userAnswer.value = ''

  try {
    const res = await fetch(API_BASE, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        game: 'riddle',
        message: '出一个简单的中文谜语，谜底是一个常见物品或动物',
        history: []
      })
    })
    const data = await res.json()
    const response = data.response

    // 解析 AI 回复，提取谜面和谜底
    // 格式可能是：谜面\n答案：xxx 或者 直接一段话
    const lines = response.split('\n')
    let question = ''
    let answer = ''
    let hint = ''

    for (const line of lines) {
      if (line.includes('答案') || line.includes('谜底')) {
        const parts = line.split(/答案|谜底/)
        if (parts.length > 1) {
          answer = parts[1].replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '').trim()
        }
      } else if (line.trim() && !question) {
        question = line.trim()
      }
    }

    // 如果没找到明确格式，使用整个回复作为谜面
    if (!question) {
      question = response
    }

    // 默认答案（如果没找到）
    if (!answer) {
      answer = '（见下方提示）'
    }

    currentRiddle.value = {
      question,
      answer,
      hint: hint || '发挥你的想象力吧！'
    }

    // 自动朗读
    setTimeout(() => speakRiddle(), 500)
  } catch (e) {
    console.error('Failed to get riddle', e)
    // 使用默认谜语
    currentRiddle.value = {
      question: '身穿鲜艳绿衣裳，头顶黄色花冠帽。夏天唱歌不停歇，秋天来了就逃跑。（打一动物）',
      answer: '蝉',
      hint: '会鸣叫的昆虫'
    }
  } finally {
    loading.value = false
  }
}

const submitAnswer = () => {
  if (!userAnswer.value.trim() || loading.value) return

  const userAns = userAnswer.value.trim()
  const correctAns = currentRiddle.value.answer

  // 简单匹配（忽略量词等）
  const isCorrect = correctAns.includes(userAns) || userAns.includes(correctAns)

  if (isCorrect) {
    feedback.value = { type: 'correct', message: '✅ 回答正确！太厉害了！' }
    correct.value++
    showAnswer.value = true
  } else {
    feedback.value = { type: 'wrong', message: `❌ 不对哦，再想想~` }
    wrong.value++
    // 允许继续猜
    userAnswer.value = ''
  }
}

onMounted(() => {
  newRiddle()
})
</script>

<style scoped>
.riddle-game {
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
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
}

.btn-primary:hover:not(:disabled) {
  background: #66b1ff;
}

.btn-primary:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}

.score-board {
  display: flex;
  justify-content: center;
  gap: 40px;
  margin-bottom: 20px;
  font-size: 1.1rem;
  color: #606266;
}

.riddle-content {
  background: #f5f7fa;
  border-radius: 16px;
  padding: 30px;
}

.speaker-btn {
  display: inline-block;
  padding: 8px 20px;
  background: #fff;
  border: 2px solid #409eff;
  border-radius: 20px;
  color: #409eff;
  cursor: pointer;
  margin-bottom: 20px;
  transition: all 0.2s;
}

.speaker-btn:hover, .speaker-btn.speaking {
  background: #409eff;
  color: white;
}

.riddle-text {
  font-size: 1.4rem;
  color: #303133;
  line-height: 1.8;
  margin-bottom: 20px;
}

.answer-section {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
}

.answer-hint {
  font-size: 1.3rem;
  color: #67c23a;
  margin-bottom: 8px;
}

.answer-hint strong {
  font-size: 1.5rem;
}

.answer-tip {
  color: #909399;
  font-size: 0.9rem;
}

.input-section {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.input-section input {
  width: 200px;
  padding: 12px 16px;
  border: 2px solid #dcdfe6;
  border-radius: 8px;
  font-size: 1rem;
}

.input-section input:focus {
  border-color: #409eff;
  outline: none;
}

.input-section button {
  padding: 12px 24px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1rem;
}

.input-section button:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}

.result-feedback {
  margin-top: 16px;
  font-size: 1.2rem;
}

.result-feedback .correct {
  color: #67c23a;
}

.result-feedback .wrong {
  color: #f56c6c;
}

.loading, .init-hint {
  padding: 60px;
  color: #909399;
  font-size: 1.2rem;
}
</style>
