<template>
  <div class="game-hall">
    <div class="header">
      <h1>🎮 趣味小游戏</h1>
      <p class="subtitle">AI 陪玩 · 语音朗读 · 轻松一刻</p>
    </div>

    <div class="games-grid">
      <div
        v-for="game in games"
        :key="game.id"
        class="game-card"
        :class="{ active: selectedGame === game.id }"
        @click="selectGame(game)"
      >
        <div class="game-icon">{{ game.icon }}</div>
        <h3>{{ game.name }}</h3>
        <p>{{ game.description }}</p>
        <div class="game-tags">
          <span class="tag ai" v-if="game.hasAI">🤖 AI</span>
          <span class="tag tts" v-if="game.hasTTS">🔊 语音</span>
        </div>
      </div>
    </div>

    <div class="game-container" v-if="selectedGame">
      <component :is="currentGameComponent" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, shallowRef } from 'vue'
import SheepSolitaire from '../games/SheepSolitaire.vue'
import RiddleGame from '../games/RiddleGame.vue'
import FortuneGame from '../games/FortuneGame.vue'
import QuizGame from '../games/QuizGame.vue'
import IdiomGame from '../games/IdiomGame.vue'

const games = [
  {
    id: 'sheep',
    name: '羊羊消消乐',
    icon: '🐑',
    description: '层叠麻将牌，点击配对消除',
    hasAI: false,
    hasTTS: false
  },
  {
    id: 'riddle',
    name: '谜语猜猜',
    icon: '🧩',
    description: 'AI 出题，语音朗读，猜猜是什么',
    hasAI: true,
    hasTTS: true
  },
  {
    id: 'fortune',
    name: '今日运势',
    icon: '🎊',
    description: '抽签测运势，AI 生成 + 语音解读',
    hasAI: true,
    hasTTS: true
  },
  {
    id: 'quiz',
    name: '趣味问答',
    icon: '❓',
    description: '趣味知识问答，AI 出题评判',
    hasAI: true,
    hasTTS: false
  },
  {
    id: 'idiom',
    name: '成语接龙',
    icon: '📜',
    description: '成语接龙，AI 陪你玩',
    hasAI: true,
    hasTTS: false
  }
]

const selectedGame = ref(null)
const currentGameComponent = shallowRef(null)

const gameComponents = {
  sheep: SheepSolitaire,
  riddle: RiddleGame,
  fortune: FortuneGame,
  quiz: QuizGame,
  idiom: IdiomGame
}

const selectGame = (game) => {
  selectedGame.value = game.id
  currentGameComponent.value = gameComponents[game.id]
}
</script>

<style scoped>
.game-hall {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.header {
  text-align: center;
  margin-bottom: 40px;
}

.header h1 {
  font-size: 2.5rem;
  color: #409eff;
  margin-bottom: 10px;
}

.subtitle {
  color: #909399;
  font-size: 1.1rem;
}

.games-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 40px;
}

.game-card {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  border: 2px solid transparent;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.game-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 4px 20px rgba(64, 158, 255, 0.3);
}

.game-card.active {
  border-color: #409eff;
  background: linear-gradient(135deg, #ecf5ff 0%, #fff 100%);
}

.game-icon {
  font-size: 3rem;
  margin-bottom: 12px;
}

.game-card h3 {
  color: #303133;
  margin-bottom: 8px;
  font-size: 1.2rem;
}

.game-card p {
  color: #909399;
  font-size: 0.9rem;
  margin-bottom: 12px;
}

.game-tags {
  display: flex;
  justify-content: center;
  gap: 8px;
}

.tag {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 0.8rem;
}

.tag.ai {
  background: #f0f9ff;
  color: #409eff;
}

.tag.tts {
  background: #fef0f0;
  color: #f56c6c;
}

.game-container {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 30px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}
</style>
