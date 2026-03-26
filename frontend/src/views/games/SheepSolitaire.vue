<template>
  <div class="sheep-solitaire">
    <div class="game-header">
      <h2>🐑 羊羊消消乐</h2>
      <div class="controls">
        <span class="score">得分: {{ score }}</span>
        <button @click="initGame" class="btn-reset">重新开始</button>
      </div>
    </div>

    <div class="status">
      <span v-if="gameState.status === 'playing'">
        剩余牌数: {{ remainingTiles }} | 已选: {{ gameState.selected.length }} 张
      </span>
      <span v-else-if="gameState.status === 'won'" class="win">🎉 你赢了!</span>
      <span v-else-if="gameState.status === 'lost'" class="lose">😅 无法消除，你输了!</span>
    </div>

    <div class="board">
      <!-- 三层结构 -->
      <div class="layers-container">
        <!-- 第三层（最上层，最后渲染） -->
        <div class="layer layer-2">
          <div
            v-for="(tile, idx) in gameState.layer3"
            :key="'l3-'+idx"
            class="tile"
            :class="{ empty: tile === 0, clickable: canClick(2, idx) }"
            :style="getTileStyle(idx)"
            @click="selectTile(2, idx)"
          >
            <span v-if="tile !== 0" class="tile-icon">{{ getIcon(tile) }}</span>
          </div>
        </div>

        <!-- 第二层 -->
        <div class="layer layer-1">
          <div
            v-for="(tile, idx) in gameState.layer2"
            :key="'l2-'+idx"
            class="tile"
            :class="{ empty: tile === 0, blocked: isBlocked(2, idx) }"
            :style="getTileStyle(idx)"
          >
            <span v-if="tile !== 0" class="tile-icon">{{ getIcon(tile) }}</span>
          </div>
        </div>

        <!-- 第一层（最下层，最先渲染） -->
        <div class="layer layer-0">
          <div
            v-for="(tile, idx) in gameState.layer1"
            :key="'l1-'+idx"
            class="tile"
            :class="{ empty: tile === 0, blocked: isBlocked(1, idx) }"
            :style="getTileStyle(idx)"
          >
            <span v-if="tile !== 0" class="tile-icon">{{ getIcon(tile) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="selected-area">
      <h4>已选中的牌（共 {{ gameState.selected.length }} 张，需 {{ Math.ceil(gameState.selected.length / 2) }} 对）</h4>
      <div class="selected-tiles">
        <div
          v-for="(tile, idx) in gameState.selected"
          :key="idx"
          class="selected-tile"
          :class="{ matched: isPair(idx) }"
          @click="unselectTile(idx)"
        >
          {{ getIcon(tile) }}
        </div>
      </div>
      <p class="tip">点击已选中的牌可以放回去</p>
    </div>

    <div class="help">
      <p>🎯 规则：点击两张相同的牌即可消除 · 同一花色三张也可消除</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'

// 水果图标
const ICONS = ['🍎', '🍊', '🍋', '🍇', '🍓', '🍒', '🥝', '🍑', '🥭', '🍍', '🥥', '🥑']

const score = ref(0)

const gameState = reactive({
  layer1: [], // 第一层 3x4 = 12
  layer2: [], // 第二层 3x4 = 12
  layer3: [], // 第三层 3x4 = 12
  selected: [], // 已选中的牌索引
  selectedInfo: [], // {layer, idx, tile}
  status: 'playing'
})

const remainingTiles = computed(() => {
  return gameState.layer1.filter(t => t !== 0).length +
         gameState.layer2.filter(t => t !== 0).length +
         gameState.layer3.filter(t => t !== 0).length
})

const getIcon = (tile) => {
  if (tile === 0) return ''
  return ICONS[(tile - 1) % ICONS.length]
}

const getTileStyle = (idx) => {
  const row = Math.floor(idx / 4)
  const col = idx % 4
  const tileSize = 52
  const gap = 4
  return {
    left: `${col * (tileSize + gap)}px`,
    top: `${row * (tileSize + gap)}px`
  }
}

// 检查某位置的牌是否被挡住
const isBlocked = (targetLayer, targetIdx) => {
  // 从上往下检查，被任何层挡住都不可点击
  for (let layer = 2; layer > targetLayer; layer--) {
    const layerData = getLayer(layer)
    for (let idx = 0; idx < 12; idx++) {
      if (layerData[idx] === 0) continue
      // 检查 idx 是否覆盖 targetIdx
      if (blocksLayer(layer, idx, targetLayer, targetIdx)) {
        return true
      }
    }
  }
  return false
}

const blocksLayer = (blockerLayer, blockerIdx, targetLayer, targetIdx) => {
  // 简单判断：blockerLayer 的角位置会挡住 targetLayer
  const blockerRow = Math.floor(blockerIdx / 4)
  const blockerCol = blockerIdx % 4
  const targetRow = Math.floor(targetIdx / 4)
  const targetCol = targetIdx % 4

  // 层叠关系：第一层的角被第二层盖
  // 第一层位置: (0,0), (0,3), (2,0), (2,3) 被第二层相应位置盖
  // 第二层位置: (0,1), (0,2), (1,1), (1,2) 被第三层相应位置盖

  if (blockerLayer === 2 && targetLayer === 1) {
    // 第三层挡住第二层的位置
    if (blockerRow === targetRow && blockerCol === targetCol) return true
  }

  if (blockerLayer === 1 && targetLayer === 0) {
    // 第二层挡住第一层的位置
    if (blockerRow === targetRow && blockerCol === targetCol) return true
  }

  return false
}

const canClick = (layer, idx) => {
  if (gameState.status !== 'playing') return false
  const layerData = getLayer(layer)
  if (layerData[idx] === 0) return false
  return !isBlocked(layer, idx)
}

const getLayer = (layer) => {
  if (layer === 0) return gameState.layer1
  if (layer === 1) return gameState.layer2
  return gameState.layer3
}

const selectTile = (layer, idx) => {
  if (!canClick(layer, idx)) return

  const layerData = getLayer(layer)
  const tile = layerData[idx]

  gameState.selected.push(tile)
  gameState.selectedInfo.push({ layer, idx, tile })

  // 检查是否可以消除
  checkAndEliminate()
}

const isPair = (idx) => {
  if (idx % 2 === 1) return false
  const curr = gameState.selected[idx]
  const prev = gameState.selected[idx - 1]
  return curr !== undefined && prev !== undefined && curr === prev
}

const unselectTile = (idx) => {
  if (gameState.status !== 'playing') return
  if (gameState.selected.length === 0) return

  const info = gameState.selectedInfo[idx]
  const layerData = getLayer(info.layer)
  layerData[info.idx] = info.tile

  gameState.selected.splice(idx, 1)
  gameState.selectedInfo.splice(idx, 1)
}

const checkAndEliminate = () => {
  if (gameState.selected.length < 3) return

  const len = gameState.selected.length
  const last = gameState.selected[len - 1]
  const secondLast = gameState.selected[len - 2]
  const thirdLast = gameState.selected[len - 3]

  // 检查最后三张是否相同（三张消除）
  if (last === secondLast && secondLast === thirdLast) {
    // 消除三张
    gameState.selected.pop()
    gameState.selected.pop()
    gameState.selected.pop()
    gameState.selectedInfo.pop()
    gameState.selectedInfo.pop()
    gameState.selectedInfo.pop()
    score.value += 3
    return
  }

  // 检查最后两张是否相同（两张消除）
  if (len >= 2 && last === secondLast) {
    gameState.selected.pop()
    gameState.selected.pop()
    gameState.selectedInfo.pop()
    gameState.selectedInfo.pop()
    score.value += 2
    return
  }

  // 检查是否选了太多牌（超过6张没消除就输了）
  if (gameState.selected.length > 6) {
    // 放回所有牌
    for (const info of [...gameState.selectedInfo]) {
      const layerData = getLayer(info.layer)
      layerData[info.idx] = info.tile
    }
    gameState.selected = []
    gameState.selectedInfo = []
    gameState.status = 'lost'
  }
}

const initGame = () => {
  // 6种图标，每种6张 = 36张，正好12+12+12
  const tiles = []
  for (let icon = 1; icon <= 6; icon++) {
    tiles.push(icon, icon, icon, icon, icon, icon)
  }

  // 打乱
  for (let i = tiles.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [tiles[i], tiles[j]] = [tiles[j], tiles[i]]
  }

  gameState.layer1 = tiles.slice(0, 12)
  gameState.layer2 = tiles.slice(12, 24)
  gameState.layer3 = tiles.slice(24, 36)
  gameState.selected = []
  gameState.selectedInfo = []
  gameState.status = 'playing'
  score.value = 0
}

onMounted(() => {
  initGame()
})
</script>

<style scoped>
.sheep-solitaire {
  text-align: center;
  max-width: 500px;
  margin: 0 auto;
}

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.game-header h2 {
  margin: 0;
  color: #303133;
}

.controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.score {
  font-size: 1.1rem;
  color: #409eff;
  font-weight: bold;
}

.btn-reset {
  padding: 8px 16px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.status {
  font-size: 1.1rem;
  margin-bottom: 16px;
  color: #606266;
}

.status .win { color: #67c23a; font-size: 1.3rem; }
.status .lose { color: #f56c6c; font-size: 1.3rem; }

.board {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  padding: 20px;
  margin-bottom: 20px;
}

.layers-container {
  position: relative;
  width: 240px;
  height: 180px;
  margin: 0 auto;
}

.layer {
  position: absolute;
  width: 216px;
  height: 168px;
  display: flex;
  flex-wrap: wrap;
  padding: 4px;
  border-radius: 8px;
}

.layer-0 {
  background: rgba(255,255,255,0.1);
  z-index: 1;
}

.layer-1 {
  background: rgba(255,255,255,0.2);
  z-index: 2;
  transform: translate(12px, 10px) scale(0.9);
}

.layer-2 {
  background: rgba(255,255,255,0.3);
  z-index: 3;
  transform: translate(24px, 20px) scale(0.8);
}

.tile {
  position: absolute;
  width: 52px;
  height: 52px;
  background: white;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.8rem;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  transition: all 0.15s;
  cursor: default;
}

.tile.clickable {
  cursor: pointer;
}

.tile.clickable:hover {
  transform: scale(1.1);
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
}

.tile.empty {
  background: transparent;
  box-shadow: none;
}

.tile.blocked {
  opacity: 0.7;
}

.tile-icon {
  user-select: none;
}

.selected-area {
  background: #f5f7fa;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 16px;
}

.selected-area h4 {
  margin: 0 0 12px 0;
  color: #303133;
  font-size: 0.95rem;
}

.selected-tiles {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 50px;
}

.selected-tile {
  width: 42px;
  height: 42px;
  background: white;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  transition: transform 0.2s;
}

.selected-tile:hover {
  transform: scale(1.15);
}

.selected-tile.matched {
  background: #67c23a;
}

.tip {
  margin: 8px 0 0 0;
  font-size: 0.8rem;
  color: #909399;
}

.help {
  background: #fff9e6;
  border-radius: 8px;
  padding: 10px 16px;
  color: #e6a23c;
  font-size: 0.9rem;
}

.help p {
  margin: 0;
}
</style>
