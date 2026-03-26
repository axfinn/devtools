<template>
  <div class="idiom-game">
    <div class="game-header">
      <h2>📜 成语接龙</h2>
      <button @click="resetGame" class="btn-primary">
        重新开始
      </button>
    </div>

    <div class="game-info">
      <p>你说一个成语，AI 接龙下一个（尾字谐音也可）</p>
    </div>

    <div class="chat-area">
      <div
        v-for="(msg, idx) in chatHistory"
        :key="idx"
        class="message"
        :class="msg.type"
      >
        <span class="avatar">{{ msg.type === 'user' ? '👤' : '🤖' }}</span>
        <div class="content">{{ msg.text }}</div>
      </div>

      <div class="message ai thinking" v-if="aiThinking">
        <span class="avatar">🤖</span>
        <div class="content">思考中...</div>
      </div>
    </div>

    <div class="input-area">
      <input
        type="text"
        v-model="userInput"
        placeholder="输入成语（如：一马当先）"
        @keyup.enter="sendMessage"
        :disabled="aiThinking"
      />
      <button @click="sendMessage" :disabled="aiThinking || !userInput.trim()">
        发送
      </button>
    </div>

    <div class="idiom-tip">
      <p>💡 小提示：可以谐音接龙，如"一马当先"→"先天不足"→"足智多谋"...</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'

const API_BASE = '/api/game/ai-chat'

const userInput = ref('')
const aiThinking = ref(false)
const chatHistory = reactive([
  { type: 'ai', text: '你好！我是成语接龙游戏搭档~ 你先说一个成语吧！' }
])

const sendMessage = async () => {
  const text = userInput.value.trim()
  if (!text || aiThinking.value) return

  // 添加用户消息
  chatHistory.push({ type: 'user', text })
  userInput.value = ''
  aiThinking.value = true

  try {
    const res = await fetch(API_BASE, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        game: 'idiom',
        message: text,
        history: chatHistory.map(m => `${m.type === 'user' ? '用户' : 'AI'}：${m.text}`)
      })
    })
    const data = await res.json()
    chatHistory.push({ type: 'ai', text: data.response })
  } catch (e) {
    console.error('Failed to get response', e)
    // 生成一个默认回复
    const defaultResponses = [
      '让我想想...「千言万语」→「语重心长」',
      '接上了！「一帆风顺」→「顺藤摸瓜」',
      '这个有点难...「瓜田李下」→「下不为例」'
    ]
    chatHistory.push({
      type: 'ai',
      text: defaultResponses[Math.floor(Math.random() * defaultResponses.length)]
    })
  } finally {
    aiThinking.value = false
  }
}

const resetGame = () => {
  chatHistory.splice(0, chatHistory.length)
  chatHistory.push({ type: 'ai', text: '游戏重新开始！你先说一个成语吧~' })
  userInput.value = ''
}
</script>

<style scoped>
.idiom-game {
  max-width: 700px;
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

.btn-primary {
  padding: 8px 20px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.game-info {
  text-align: center;
  color: #909399;
  margin-bottom: 20px;
  font-size: 0.95rem;
}

.chat-area {
  background: #f5f7fa;
  border-radius: 16px;
  padding: 20px;
  min-height: 300px;
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 16px;
}

.message {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.message.user {
  flex-direction: row-reverse;
}

.message.user .content {
  background: #409eff;
  color: white;
}

.message.ai .content {
  background: white;
  color: #303133;
}

.avatar {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.content {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 12px;
  line-height: 1.5;
}

.thinking .content {
  background: #fff9e6;
  color: #909399;
  font-style: italic;
}

.input-area {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.input-area input {
  flex: 1;
  padding: 12px 16px;
  border: 2px solid #dcdfe6;
  border-radius: 8px;
  font-size: 1rem;
}

.input-area input:focus {
  border-color: #409eff;
  outline: none;
}

.input-area button {
  padding: 12px 24px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1rem;
}

.input-area button:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}

.idiom-tip {
  background: #f0f9ff;
  border-radius: 8px;
  padding: 12px 16px;
  color: #409eff;
  font-size: 0.9rem;
}

.idiom-tip p {
  margin: 0;
}
</style>
