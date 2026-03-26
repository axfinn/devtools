<template>
  <div class="quiz-game">
    <div class="game-header">
      <h2>❓ 趣味问答</h2>
      <button @click="newQuestion" class="btn-primary" :disabled="loading">
        {{ loading ? '🤔 出题中...' : '下一题' }}
      </button>
    </div>

    <div class="score-board">
      <span>答对: {{ correct }}</span>
      <span>答错: {{ wrong }}</span>
      <span>连续: {{ streak }} 🔥</span>
    </div>

    <div class="quiz-content" v-if="currentQuestion">
      <p class="category">{{ currentQuestion.category }}</p>
      <p class="question-text">{{ currentQuestion.question }}</p>

      <div class="options" v-if="currentQuestion.options">
        <button
          v-for="(option, idx) in currentQuestion.options"
          :key="idx"
          class="option-btn"
          :class="getOptionClass(idx)"
          @click="selectOption(idx)"
          :disabled="selectedIdx !== null"
        >
          {{ option }}
        </button>
      </div>

      <div class="input-answer" v-else>
        <input
          type="text"
          v-model="userAnswer"
          placeholder="输入你的答案"
          @keyup.enter="submitAnswer"
          :disabled="selectedIdx !== null"
        />
        <button @click="submitAnswer" :disabled="selectedIdx !== null || !userAnswer.trim()">
          提交
        </button>
      </div>

      <div class="result-section" v-if="selectedIdx !== null">
        <p class="result-text" :class="isCorrect ? 'correct' : 'wrong'">
          {{ isCorrect ? '✅ 回答正确！' : `❌ 错了~ 正确答案是：${currentQuestion.answer}` }}
        </p>
        <p class="explanation" v-if="currentQuestion.explanation">
          💡 {{ currentQuestion.explanation }}
        </p>
      </div>
    </div>

    <div class="loading" v-else-if="loading">
      <p>🤔 AI正在出一道有趣的题目...</p>
    </div>

    <div class="init-hint" v-else>
      <p>点击"下一题"开始答题吧！</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'

const API_BASE = '/api/game/ai-chat'

const loading = ref(false)
const currentQuestion = ref(null)
const userAnswer = ref('')
const selectedIdx = ref(null)
const isCorrect = ref(false)
const correct = ref(0)
const wrong = ref(0)
const streak = ref(0)

const categories = ['常识', '动物', '植物', '地理', '历史', '科学', '生活', '脑筋急转弯']

const newQuestion = async () => {
  loading.value = true
  selectedIdx.value = null
  isCorrect.value = false
  userAnswer.value = ''

  try {
    const category = categories[Math.floor(Math.random() * categories.length)]

    const res = await fetch(API_BASE, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        game: 'quiz',
        message: `出一道${category}类的趣味知识题，格式如下：
问题：（问题内容）
A. （选项1）
B. （选项2）
C. （选项3）
D. （选项4）
答案：（正确答案，如B）
解释：（简单解释，20字内）`,
        history: []
      })
    })
    const data = await res.json()
    const response = data.response

    // 解析 AI 回复
    const lines = response.split('\n').filter(l => l.trim())
    const questionData = {
      category,
      question: '',
      options: null,
      answer: '',
      explanation: ''
    }

    let currentOption = ''
    for (const line of lines) {
      const trimmed = line.trim()
      if (trimmed.startsWith('问题') || trimmed.startsWith('题目')) {
        questionData.question = trimmed.replace(/^[^：]+：/, '').trim()
      } else if (/^[A-D][\.、:]/.test(trimmed)) {
        if (!questionData.options) questionData.options = []
        questionData.options.push(trimmed.replace(/^[A-D][\.、:]\s*/, ''))
      } else if (trimmed.startsWith('答案')) {
        const ans = trimmed.replace('答案', '').replace(/[^A-Da-d]/g, '').trim()
        if (ans.toUpperCase() >= 'A' && ans.toUpperCase() <= 'D' && questionData.options) {
          selectedIdx.value = ans.toUpperCase().charCodeAt(0) - 65
        }
        questionData.answer = trimmed.replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '').replace('答案', '').trim()
      } else if (trimmed.startsWith('解释') || trimmed.startsWith('说明')) {
        questionData.explanation = trimmed.replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '').replace('解释', '').trim()
      }
    }

    // 如果解析失败，使用默认格式
    if (!questionData.question) {
      questionData.question = response
    }

    currentQuestion.value = questionData
  } catch (e) {
    console.error('Failed to get question', e)
    currentQuestion.value = {
      category: '生活',
      question: '以下哪种水果VC含量最高？',
      options: ['苹果', '橙子', '香蕉', '葡萄'],
      answer: '橙子',
      explanation: '橙子富含维生素C，每100克含约53毫克'
    }
  } finally {
    loading.value = false
  }
}

const getOptionClass = (idx) => {
  if (selectedIdx.value === null) return ''
  if (idx === selectedIdx.value) {
    return isCorrect.value ? 'correct' : 'wrong'
  }
  // 显示正确答案
  const correctLetter = currentQuestion.value.answer.charAt(0).toUpperCase()
  if (correctLetter.charCodeAt(0) - 65 === idx) {
    return 'show-correct'
  }
  return 'dimmed'
}

const selectOption = (idx) => {
  if (selectedIdx.value !== null) return

  selectedIdx.value = idx

  // 获取用户选择的答案
  const userAns = currentQuestion.value.options[idx]
  const correctAns = currentQuestion.value.answer

  // 检查是否正确（忽略A/B/C/D和标点）
  const userAnsClean = userAns.replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '')
  const correctAnsClean = correctAns.replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '')

  isCorrect.value = correctAnsClean.includes(userAnsClean) || userAnsClean.includes(correctAnsClean)

  if (isCorrect.value) {
    correct.value++
    streak.value++
  } else {
    wrong.value++
    streak.value = 0
  }
}

const submitAnswer = () => {
  if (!userAnswer.value.trim() || selectedIdx.value !== null) return

  const userAns = userAnswer.value.trim()
  const correctAns = currentQuestion.value.answer

  const userAnsClean = userAns.replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '')
  const correctAnsClean = correctAns.replace(/[^a-zA-Z\u4e00-\u9fa5]/g, '')

  isCorrect.value = correctAnsClean.includes(userAnsClean) || userAnsClean.includes(correctAnsClean)
  selectedIdx.value = -1 // 表示输入框模式

  if (isCorrect.value) {
    correct.value++
    streak.value++
  } else {
    wrong.value++
    streak.value = 0
  }
}
</script>

<style scoped>
.quiz-game {
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
  gap: 30px;
  margin-bottom: 20px;
  font-size: 1.1rem;
  color: #606266;
}

.score-board span:last-child {
  color: #f56c6c;
}

.quiz-content {
  background: #f5f7fa;
  border-radius: 16px;
  padding: 30px;
}

.category {
  display: inline-block;
  padding: 4px 16px;
  background: #409eff;
  color: white;
  border-radius: 20px;
  font-size: 0.9rem;
  margin-bottom: 16px;
}

.question-text {
  font-size: 1.3rem;
  color: #303133;
  line-height: 1.6;
  margin-bottom: 24px;
}

.options {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.option-btn {
  padding: 16px 24px;
  background: white;
  border: 2px solid #dcdfe6;
  border-radius: 12px;
  font-size: 1.1rem;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.option-btn:hover:not(:disabled) {
  border-color: #409eff;
  background: #ecf5ff;
}

.option-btn:disabled {
  cursor: default;
}

.option-btn.correct {
  border-color: #67c23a;
  background: #f0f9eb;
}

.option-btn.wrong {
  border-color: #f56c6c;
  background: #fef0f0;
}

.option-btn.show-correct {
  border-color: #67c23a;
  background: #d4edda;
}

.option-btn.dimmed {
  opacity: 0.5;
}

.input-answer {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.input-answer input {
  width: 200px;
  padding: 12px 16px;
  border: 2px solid #dcdfe6;
  border-radius: 8px;
  font-size: 1rem;
}

.input-answer input:focus {
  border-color: #409eff;
  outline: none;
}

.input-answer button {
  padding: 12px 24px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1rem;
}

.input-answer button:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}

.result-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e0e0e0;
}

.result-text {
  font-size: 1.2rem;
  margin-bottom: 8px;
}

.result-text.correct {
  color: #67c23a;
}

.result-text.wrong {
  color: #f56c6c;
}

.explanation {
  color: #909399;
  font-size: 0.95rem;
  margin: 0;
}

.loading, .init-hint {
  padding: 60px;
  color: #909399;
  font-size: 1.2rem;
}
</style>
