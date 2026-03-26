<template>
  <div class="guess-number">
    <div class="game-header">
      <h2>🔢 猜数字</h2>
      <button @click="initGame" class="btn-reset">重新开始</button>
    </div>

    <div class="info">
      <p>🎯 目标数字在 1-100 之间</p>
      <p>每轮各猜一次，猜对得分，5轮结束比分高者获胜</p>
    </div>

    <div class="status">
      <span v-if="gameState.winner === 0">
        第 {{ gameState.round }} / 5 轮 | 你的回合
      </span>
      <span v-else-if="gameState.winner === 1" class="win">🎉 你赢了!</span>
      <span v-else-if="gameState.winner === 2" class="lose">😅 AI赢了!</span>
      <span v-else class="draw">🤝 平局!</span>
    </div>

    <div class="score-board">
      <div class="score-item">
        <span class="label">你的分数</span>
        <span class="value">{{ gameState.player_score }}</span>
      </div>
      <div class="score-item">
        <span class="label">AI分数</span>
        <span class="value">{{ gameState.ai_score }}</span>
      </div>
    </div>

    <div class="input-area" v-if="gameState.winner === 0">
      <input
        type="number"
        v-model.number="guessValue"
        :min="1"
        :max="100"
        placeholder="1-100"
        @keyup.enter="makeGuess"
      />
      <button @click="makeGuess" :disabled="gameState.winner !== 0">猜测</button>
    </div>

    <div class="history">
      <div class="history-section">
        <h4>你的猜测</h4>
        <div class="guess-list">
          <span
            v-for="(guess, i) in gameState.player_guesses"
            :key="'p'+i"
            class="guess-item"
            :class="getGuessClass(guess)"
          >
            {{ guess }}
          </span>
        </div>
      </div>

      <div class="history-section">
        <h4>AI猜测</h4>
        <div class="guess-list">
          <span
            v-for="(item, i) in gameState.ai_guesses"
            :key="'a'+i"
            class="guess-item ai"
          >
            {{ item.guess }} ({{ item.hint === 'big' ? '太大了' : item.hint === 'small' ? '太小了' : '猜对了!' }})
          </span>
        </div>
      </div>
    </div>

    <div class="final-result" v-if="gameState.winner !== 0">
      <p>最终结果: 你 {{ gameState.player_score }} : {{ gameState.ai_score }} AI</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'

const API_BASE = '/api/game'
const guessValue = ref(null)

const gameState = reactive({
  target: 0,
  player_guesses: [],
  ai_guesses: [],
  player_score: 0,
  ai_score: 0,
  round: 1,
  winner: 0,
  mode: 'ai'
})

const getGuessClass = (guess) => {
  if (guess === gameState.target) return 'correct'
  if (guess > gameState.target) return 'big'
  return 'small'
}

const initGame = async () => {
  try {
    const res = await fetch(`${API_BASE}/guessnumber/init`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode: 'ai' })
    })
    const data = await res.json()
    Object.assign(gameState, data)
    guessValue.value = null
  } catch (e) {
    console.error('初始化失败', e)
  }
}

const makeGuess = async () => {
  if (!guessValue.value || guessValue.value < 1 || guessValue.value > 100) {
    alert('请输入1-100之间的数字')
    return
  }

  try {
    const res = await fetch(`${API_BASE}/guessnumber/guess`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ guess: guessValue.value, state: gameState })
    })
    const data = await res.json()
    Object.assign(gameState, data)
    guessValue.value = null
  } catch (e) {
    console.error('猜测失败', e)
  }
}

onMounted(() => {
  initGame()
})
</script>

<style scoped>
.guess-number { text-align: center; }

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.game-header h2 { margin: 0; color: #303133; }

.btn-reset {
  padding: 8px 20px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.info {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.info p { margin: 4px 0; color: #606266; }

.status {
  font-size: 1.3rem;
  margin-bottom: 20px;
}

.status .win { color: #67c23a; }
.status .lose { color: #f56c6c; }
.status .draw { color: #909399; }

.score-board {
  display: flex;
  justify-content: center;
  gap: 40px;
  margin-bottom: 20px;
}

.score-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.score-item .label { color: #909399; }
.score-item .value { font-size: 2rem; color: #409eff; font-weight: bold; }

.input-area {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-bottom: 24px;
}

.input-area input {
  width: 120px;
  padding: 10px;
  font-size: 1.2rem;
  border: 2px solid #dcdfe6;
  border-radius: 6px;
  text-align: center;
}

.input-area button {
  padding: 10px 30px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.input-area button:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}

.history {
  display: flex;
  justify-content: center;
  gap: 40px;
  text-align: left;
}

.history-section h4 { margin-bottom: 8px; color: #303133; }

.guess-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-height: 100px;
}

.guess-item {
  padding: 4px 12px;
  background: #ecf5ff;
  border-radius: 4px;
  font-size: 0.9rem;
}

.guess-item.correct { background: #67c23a; color: white; }
.guess-item.big { background: #f56c6c; color: white; }
.guess-item.small { background: #409eff; color: white; }
.guess-item.ai { background: #f0f9ff; }

.final-result {
  margin-top: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
  font-size: 1.2rem;
}
</style>
