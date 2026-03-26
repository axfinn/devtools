<template>
  <div class="rps">
    <div class="game-header">
      <h2>✋ 石头剪刀布</h2>
      <button @click="initGame" class="btn-reset">重新开始</button>
    </div>

    <div class="info">
      <p>🎮 共10局，得分高者获胜</p>
    </div>

    <div class="score-board">
      <div class="score-item">
        <span class="label">你的分数</span>
        <span class="value you">{{ gameState.player_score }}</span>
      </div>
      <div class="score-item">
        <span class="label">AI分数</span>
        <span class="value ai">{{ gameState.ai_score }}</span>
      </div>
    </div>

    <div class="battle-area">
      <div class="player-side">
        <div class="avatar">👤</div>
        <div class="choice" v-if="gameState.player_choice">
          {{ getChoiceEmoji(gameState.player_choice) }}
        </div>
        <div class="choice-placeholder" v-else>?</div>
      </div>

      <div class="vs">VS</div>

      <div class="ai-side">
        <div class="avatar">🤖</div>
        <div class="choice" v-if="gameState.ai_choice">
          {{ getChoiceEmoji(gameState.ai_choice) }}
        </div>
        <div class="choice-placeholder" v-else>?</div>
      </div>
    </div>

    <div class="result-text" v-if="gameState.player_choice">
      <span v-if="gameState.winner === 'player'" class="win">✅ 你赢了这一局!</span>
      <span v-else-if="gameState.winner === 'ai'" class="lose">❌ AI赢了这一局!</span>
      <span v-else class="draw">🤝 平局!</span>
    </div>

    <div class="choices" v-if="!isGameOver">
      <button
        v-for="choice in ['rock', 'paper', 'scissors']"
        :key="choice"
        class="choice-btn"
        @click="play(choice)"
      >
        <span class="emoji">{{ getChoiceEmoji(choice) }}</span>
        <span class="name">{{ getChoiceName(choice) }}</span>
      </button>
    </div>

    <div class="final-result" v-else>
      <p class="final-text">
        <span v-if="gameState.winner === 'player'" class="win">🎉 你赢了比赛!</span>
        <span v-else-if="gameState.winner === 'ai'" class="lose">😅 AI赢了比赛!</span>
        <span v-else class="draw">🤝 比赛平局!</span>
      </p>
      <p>最终比分: 你 {{ gameState.player_score }} : {{ gameState.ai_score }} AI</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'

const API_BASE = '/api/game'

const gameState = reactive({
  player_choice: '',
  ai_choice: '',
  player_score: 0,
  ai_score: 0,
  winner: '',
  round: 1,
  mode: 'ai'
})

const isGameOver = computed(() => gameState.round > 10)

const getChoiceEmoji = (choice) => {
  const map = { rock: '🪨', paper: '📄', scissors: '✂️' }
  return map[choice] || '?'
}

const getChoiceName = (choice) => {
  const map = { rock: '石头', paper: '布', scissors: '剪刀' }
  return map[choice] || ''
}

const initGame = async () => {
  try {
    const res = await fetch(`${API_BASE}/rps/init`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode: 'ai' })
    })
    const data = await res.json()
    Object.assign(gameState, data)
  } catch (e) {
    console.error('初始化失败', e)
  }
}

const play = async (choice) => {
  try {
    const res = await fetch(`${API_BASE}/rps/play`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ choice, state: gameState })
    })
    const data = await res.json()
    Object.assign(gameState, data)
  } catch (e) {
    console.error('出拳失败', e)
  }
}

onMounted(() => {
  initGame()
})
</script>

<style scoped>
.rps { text-align: center; }

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

.score-board {
  display: flex;
  justify-content: center;
  gap: 60px;
  margin-bottom: 30px;
}

.score-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.score-item .label { color: #909399; }
.score-item .value { font-size: 2.5rem; font-weight: bold; }
.score-item .value.you { color: #409eff; }
.score-item .value.ai { color: #f56c6c; }

.battle-area {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 40px;
  margin-bottom: 20px;
}

.player-side, .ai-side {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.avatar { font-size: 3rem; }

.choice {
  font-size: 4rem;
  min-width: 80px;
  min-height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.choice-placeholder {
  font-size: 3rem;
  color: #c0c4cc;
  min-width: 80px;
  min-height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.vs {
  font-size: 2rem;
  color: #909399;
  font-weight: bold;
}

.result-text {
  font-size: 1.3rem;
  margin-bottom: 20px;
  height: 30px;
}

.result-text .win { color: #67c23a; }
.result-text .lose { color: #f56c6c; }
.result-text .draw { color: #909399; }

.choices {
  display: flex;
  justify-content: center;
  gap: 20px;
}

.choice-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 24px;
  background: #fff;
  border: 2px solid #dcdfe6;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.choice-btn:hover {
  border-color: #409eff;
  background: #ecf5ff;
  transform: translateY(-3px);
}

.choice-btn .emoji { font-size: 2.5rem; }
.choice-btn .name { color: #606266; font-size: 0.9rem; }

.final-result {
  padding: 20px;
  background: #f5f7fa;
  border-radius: 12px;
}

.final-text {
  font-size: 1.5rem;
  margin-bottom: 10px;
}

.final-text .win { color: #67c23a; }
.final-text .lose { color: #f56c6c; }
.final-text .draw { color: #909399; }
</style>
