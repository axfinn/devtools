<template>
  <div class="tool-container" :class="{ 'is-fullscreen': isFullscreen }">
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
        <el-button class="fullscreen-btn" @click="toggleFullscreen">
          <el-icon><FullScreen v-if="!isFullscreen" /><Close v-else /></el-icon>
          {{ isFullscreen ? '退出' : '放大' }}
        </el-button>
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { diffChars, diffWords, diffLines } from 'diff'
import { FullScreen, Close } from '@element-plus/icons-vue'

const isFullscreen = ref(false)

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

const onKeydown = (e) => {
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})

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
.diff-result {
  background-color: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--radius-md);
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
  color: var(--color-success);
  margin-right: 10px;
}

.diff-stats .removed {
  color: var(--color-danger);
}

.diff-content {
  padding: 15px;
  font-family: var(--font-family-mono);
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-y: auto;
  flex: 1;
  color: var(--text-primary);
}

.diff-added {
  background-color: var(--success-light);
  color: var(--success-dark);
}

.diff-removed {
  background-color: var(--danger-light);
  color: var(--danger-dark);
  text-decoration: line-through;
}
</style>
