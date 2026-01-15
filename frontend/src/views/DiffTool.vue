<template>
  <div class="tool-container">
    <div class="tool-header">
      <h2>Diff 对比工具</h2>
      <div class="actions">
        <el-radio-group v-model="diffMode" size="default">
          <el-radio-button value="chars">字符级</el-radio-button>
          <el-radio-button value="words">单词级</el-radio-button>
          <el-radio-button value="lines">行级</el-radio-button>
        </el-radio-group>
        <el-button type="primary" @click="runDiff" style="margin-left: 15px">
          对比
        </el-button>
        <el-button @click="swapTexts">
          <el-icon><Switch /></el-icon>
          交换
        </el-button>
        <el-button @click="clearAll">清空</el-button>
      </div>
    </div>

    <div class="editor-container">
      <div class="editor-panel">
        <div class="panel-header">原始文本</div>
        <textarea
          ref="leftEditor"
          v-model="leftText"
          class="code-editor"
          placeholder="输入原始文本..."
          spellcheck="false"
          @scroll="onScroll('left')"
        ></textarea>
      </div>
      <div class="editor-panel">
        <div class="panel-header">对比文本</div>
        <textarea
          ref="rightEditor"
          v-model="rightText"
          class="code-editor"
          placeholder="输入要对比的文本..."
          spellcheck="false"
          @scroll="onScroll('right')"
        ></textarea>
      </div>
    </div>

    <div v-if="diffResult.length > 0" class="diff-result">
      <div class="panel-header">
        对比结果
        <span class="diff-stats">
          <span class="added">+{{ addedCount }}</span>
          <span class="removed">-{{ removedCount }}</span>
        </span>
      </div>
      <div class="diff-content">
        <span
          v-for="(part, index) in diffResult"
          :key="index"
          :class="{
            'diff-added': part.added,
            'diff-removed': part.removed
          }"
        >{{ part.value }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { diffChars, diffWords, diffLines } from 'diff'

const leftText = ref('')
const rightText = ref('')
const diffResult = ref([])
const diffMode = ref('lines')

const leftEditor = ref(null)
const rightEditor = ref(null)
let isScrolling = false

const onScroll = (source) => {
  if (isScrolling) return
  isScrolling = true

  const sourceEl = source === 'left' ? leftEditor.value : rightEditor.value
  const targetEl = source === 'left' ? rightEditor.value : leftEditor.value

  if (sourceEl && targetEl) {
    targetEl.scrollTop = sourceEl.scrollTop
    targetEl.scrollLeft = sourceEl.scrollLeft
  }

  requestAnimationFrame(() => {
    isScrolling = false
  })
}

const addedCount = computed(() => {
  return diffResult.value.filter(p => p.added).reduce((sum, p) => sum + p.value.length, 0)
})

const removedCount = computed(() => {
  return diffResult.value.filter(p => p.removed).reduce((sum, p) => sum + p.value.length, 0)
})

const runDiff = () => {
  let diffFn
  switch (diffMode.value) {
    case 'chars':
      diffFn = diffChars
      break
    case 'words':
      diffFn = diffWords
      break
    case 'lines':
    default:
      diffFn = diffLines
  }
  diffResult.value = diffFn(leftText.value, rightText.value)
}

const swapTexts = () => {
  const temp = leftText.value
  leftText.value = rightText.value
  rightText.value = temp
  if (diffResult.value.length > 0) {
    runDiff()
  }
}

const clearAll = () => {
  leftText.value = ''
  rightText.value = ''
  diffResult.value = []
}
</script>

<style scoped>
.tool-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.tool-header h2 {
  margin: 0;
  color: #e0e0e0;
}

.editor-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 15px;
  min-height: 300px;
}

.editor-panel {
  display: flex;
  flex-direction: column;
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
}

.panel-header {
  padding: 10px 15px;
  background-color: #2d2d2d;
  color: #a0a0a0;
  font-size: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.code-editor {
  flex: 1;
  width: 100%;
  padding: 15px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.5;
  outline: none;
}

.diff-result {
  background-color: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  flex: 1;
  min-height: 200px;
  display: flex;
  flex-direction: column;
}

.diff-stats {
  font-size: 12px;
}

.diff-stats .added {
  color: #4caf50;
  margin-right: 10px;
}

.diff-stats .removed {
  color: #f44336;
}

.diff-content {
  padding: 15px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-y: auto;
  flex: 1;
  color: #d4d4d4;
}

.diff-added {
  background-color: rgba(76, 175, 80, 0.3);
  color: #81c784;
}

.diff-removed {
  background-color: rgba(244, 67, 54, 0.3);
  color: #e57373;
  text-decoration: line-through;
}
</style>
