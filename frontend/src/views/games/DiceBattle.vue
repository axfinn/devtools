<template>
  <div class="dice-battle">
    <div class="game-header">
      <h2>🎲 色子比大小</h2>
      <button @click="initGame" class="btn-reset">重新开始</button>
    </div>

    <div class="info">
      <p>🎮 共10轮，投掷三颗色子，点数和大者获胜</p>
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
        <div class="dices" v-if="gameState.player_dice[0]">
          <span
            v-for="(d, i) in gameState.player_dice"
            :key="i"
            class="dice"
          >{{ getDiceEmoji(d) }}</span>
        </div>
        <div class="dices placeholder" v-else>
          <span class="dice">🎲</span>
          <span class="dice">🎲</span>
          <span class="dice">🎲</span>
        </div>
        <div class="sum" v-if="gameState.player_sum">
          合计: {{ gameState.player_sum }}
        </div>
      </div>

      <div class="vs">VS</div>

      <div class="ai-side">
        <div class="avatar">🤖</div>
        <div class="dices" v-if="gameState.ai_dice[0]">
          <span
            v-for="(d, i) in gameState.ai_dice"
            :key="i"
            class="dice"
          >{{ getDiceEmoji(d) }}</span>
        </div>
        <div class="dices placeholder" v-else>
          <span class="dice">🎲</span>
          <span class="dice">🎲</span>
          <span class="dice">🎲</span>
        </div>
        <div class="sum" v-if="gameState.ai_sum">
          合计: {{ gameState.ai_sum }}
        </div>
      </div>
    </div>

    <div class="result-text" v-if="gameState.winner && !isGameOver">
      <span v-if="gameState.winner === 'player'" class="win">✅ 这一局你赢了!</span>
      <span v-else-if="gameState.winner === 'ai'" class="lose">❌ 这一局AI赢了!</span>
      <span v-else class="draw">🤝 平局!</span>
    </div>

    <button
      v-if="!isGameOver && !gameState.winner"
      class="roll-btn"
      @click="rollDice"
      :disabled="rolling"
    >
      {{ rolling ? '🤔 AI在思考...' : '🎲 投掷色子' }}
    </button>

    <div class="round-info" v-if="!isGameOver">
      第 {{ gameState.round }} / 10 轮
    </div>

    <div class="final-result" v-if="isGameOver">
      <p class="final-text">
        <span v-if="gameState.winner === 'player_final'" class="win">🎉 你赢了比赛!</span>
        <span v-else-if="gameState.winner === 'ai_final'" class="lose">😅 AI赢了比赛!</span>
        <span v-else class="draw">🤝 比赛平局!</span>
      </p>
      <p>最终比分: 你 {{ gameState.player_score }} : {{ gameState.ai_score }} AI</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'

const API_BASE = '/api/game'
const rolling = ref(false)

const gameState = reactive({
  player_dice: [0, 0, 0],
  ai_dice: [0, 0, 0],
  player_sum: 0,
  ai_sum: 0,
  player_score: 0,
  ai_score: 0,
  winner: '',
  round: 1,
  mode: 'ai'
})

const isGameOver = computed(() => gameState.round > 10)

const getDiceEmoji = (n) => {
  const emojis = ['🎲', '⚀', '⚁', '⚂', '⚃', '⚄', '⚅']
  return emojis[n] || '🎲'
}

const initGame = async () => {
  try {
    const res = await fetch(`${API_BASE}/dice/init`, {
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

const rollDice = async () => {
  rolling.value = true
  try {
    const res = await fetch(`${API_BASE}/dice/roll`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ state: gameState })
    })
    const data = await res.json()
    Object.assign(gameState, data)
  } catch (e) {
    console.error('投掷失败', e)
  } finally {
    rolling.value = false
  }
}

onMounted(() => {
  initGame()
})
</script>

<style scoped>
.dice-battle { text-align: center; }

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

.dices {
  display: flex;
  gap: 8px;
}

.dices.placeholder .dice { opacity: 0.3; }

.dice { font-size: 2.5rem; }

.sum {
  font-size: 1.2rem;
  color: #606266;
  font-weight: bold;
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

.roll-btn {
  padding: 16px 40px;
  font-size: 1.2rem;
  background: linear-gradient(135deg, #409eff 0%, #66b1ff 100%);
  color: white;
  border: none;
  border-radius: 30px;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 4px 15px rgba(64, 158, 255, 0.4);
}

.roll-btn:hover:not(:disabled) {
  transform: translateY(-3px);
  box-shadow: 0 6px 20px rgba(64, 158, 255, 0.5);
}

.roll-btn:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
  box-shadow: none;
}

.round-info {
  margin-top: 16px;
  color: #909399;
}

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
