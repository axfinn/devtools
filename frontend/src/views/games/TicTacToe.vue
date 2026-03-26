<template>
  <div class="tictactoe">
    <div class="game-header">
      <h2>⭕ 井字棋</h2>
      <div class="controls">
        <button @click="initGame" class="btn-reset">重新开始</button>
      </div>
    </div>

    <div class="status">
      <span v-if="gameState.winner === 0">
        {{ gameState.current === 1 ? '🔵 你的回合' : '🔴 AI回合' }}
      </span>
      <span v-else-if="gameState.winner === 1" class="win">🎉 你赢了!</span>
      <span v-else-if="gameState.winner === 2" class="lose">😅 AI赢了!</span>
      <span v-else class="draw">🤝 平局!</span>
    </div>

    <div class="board">
      <div
        v-for="(cell, index) in gameState.board"
        :key="index"
        class="cell"
        :class="{ clickable: cell === 0 && gameState.winner === 0 && gameState.current === 1 }"
        @click="makeMove(index)"
      >
        <span v-if="cell === 1" class="player-x">✕</span>
        <span v-else-if="cell === 2" class="player-o">○</span>
      </div>
    </div>

    <div class="score">
      <span>你: {{ playerScore }}</span>
      <span>AI: {{ aiScore }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'

const API_BASE = '/api/game'

const playerScore = ref(0)
const aiScore = ref(0)

const gameState = reactive({
  board: [0, 0, 0, 0, 0, 0, 0, 0, 0],
  current: 1,
  winner: 0,
  mode: 'ai',
  ai_type: 'smart'
})

const initGame = async () => {
  try {
    const res = await fetch(`${API_BASE}/tictactoe/init`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode: 'ai', ai_type: 'smart' })
    })
    const data = await res.json()
    Object.assign(gameState, data)
  } catch (e) {
    console.error('初始化失败', e)
  }
}

const makeMove = async (position) => {
  if (gameState.board[position] !== 0 || gameState.winner !== 0 || gameState.current !== 1) {
    return
  }

  try {
    const res = await fetch(`${API_BASE}/tictactoe/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ position, state: gameState })
    })
    const data = await res.json()
    Object.assign(gameState, data)

    if (gameState.winner === 1) playerScore.value++
    if (gameState.winner === 2) aiScore.value++
  } catch (e) {
    console.error('落子失败', e)
  }
}

onMounted(() => {
  initGame()
})
</script>

<style scoped>
.tictactoe {
  text-align: center;
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

.btn-reset {
  padding: 8px 20px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.btn-reset:hover {
  background: #66b1ff;
}

.status {
  font-size: 1.3rem;
  margin-bottom: 20px;
  color: #606266;
}

.status .win { color: #67c23a; }
.status .lose { color: #f56c6c; }
.status .draw { color: #909399; }

.board {
  display: grid;
  grid-template-columns: repeat(3, 100px);
  gap: 8px;
  justify-content: center;
  margin-bottom: 20px;
}

.cell {
  width: 100px;
  height: 100px;
  background: #f5f7fa;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 3rem;
  transition: all 0.2s;
}

.cell.clickable {
  cursor: pointer;
}

.cell.clickable:hover {
  background: #ecf5ff;
  transform: scale(1.05);
}

.player-x { color: #409eff; }
.player-o { color: #f56c6c; }

.score {
  display: flex;
  justify-content: center;
  gap: 40px;
  font-size: 1.2rem;
  color: #606266;
}
</style>
