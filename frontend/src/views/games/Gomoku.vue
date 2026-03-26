<template>
  <div class="gomoku">
    <div class="game-header">
      <h2>🔘 五子棋</h2>
      <button @click="initGame" class="btn-reset">重新开始</button>
    </div>

    <div class="status">
      <span v-if="gameState.winner === 0">
        {{ gameState.current === 1 ? '🔵 你的回合 (黑棋)' : '🔴 AI回合 (白棋)' }}
      </span>
      <span v-else-if="gameState.winner === 1" class="win">🎉 你赢了!</span>
      <span v-else-if="gameState.winner === 2" class="lose">😅 AI赢了!</span>
    </div>

    <div class="board">
      <div v-for="(row, r) in gameState.board" :key="r" class="row">
        <div
          v-for="(cell, c) in row"
          :key="c"
          class="cell"
          :class="{ clickable: cell === 0 && gameState.winner === 0 && gameState.current === 1 }"
          @click="makeMove(r, c)"
        >
          <span v-if="cell === 1" class="black"></span>
          <span v-else-if="cell === 2" class="white"></span>
        </div>
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
  board: Array(15).fill(null).map(() => Array(15).fill(0)),
  current: 1,
  winner: 0,
  mode: 'ai'
})

const initGame = async () => {
  try {
    const res = await fetch(`${API_BASE}/gomoku/init`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mode: 'ai' })
    })
    const data = await res.json()
    gameState.board = data.board
    gameState.current = data.current
    gameState.winner = data.winner
    gameState.mode = data.mode
  } catch (e) {
    console.error('初始化失败', e)
  }
}

const makeMove = async (row, col) => {
  if (gameState.board[row][col] !== 0 || gameState.winner !== 0 || gameState.current !== 1) {
    return
  }

  try {
    const res = await fetch(`${API_BASE}/gomoku/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ row, col, state: gameState })
    })
    const data = await res.json()
    gameState.board = data.board
    gameState.current = data.current
    gameState.winner = data.winner

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
.gomoku { text-align: center; }

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

.status {
  font-size: 1.3rem;
  margin-bottom: 20px;
  color: #606266;
}

.status .win { color: #67c23a; }
.status .lose { color: #f56c6c; }

.board {
  display: inline-block;
  background: #deb887;
  padding: 5px;
  border-radius: 4px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.2);
}

.row {
  display: flex;
}

.cell {
  width: 32px;
  height: 32px;
  background: #f0d9b5;
  border: 1px solid #b58863;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.cell.clickable { cursor: pointer; }
.cell.clickable:hover { background: #ffe4b5; }

.black, .white {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: block;
}

.black {
  background: #000;
  box-shadow: 1px 1px 3px rgba(0,0,0,0.5);
}

.white {
  background: #fff;
  box-shadow: 1px 1px 3px rgba(0,0,0,0.3);
}

.score {
  display: flex;
  justify-content: center;
  gap: 40px;
  font-size: 1.2rem;
  margin-top: 20px;
  color: #606266;
}
</style>
