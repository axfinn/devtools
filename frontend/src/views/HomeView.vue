<template>
  <div class="home-container">
    <!-- 搜索区域 -->
    <div class="search-section">
      <div class="search-box">
        <el-icon class="search-icon"><Search /></el-icon>
        <input
          v-model="searchText"
          type="text"
          placeholder="搜索工具..."
          class="search-input"
          @keyup.enter="goToFirst"
        />
        <span v-if="searchText" class="search-clear" @click="searchText = ''">
          <el-icon><Close /></el-icon>
        </span>
      </div>
    </div>

    <!-- 快捷入口 -->
    <div v-if="!searchText" class="shortcuts-section">
      <div class="section-title">常用工具</div>
      <div class="shortcuts-grid">
        <div
          v-for="tool in shortcutTools"
          :key="tool.path"
          class="shortcut-card"
          @click="goTo(tool.path)"
        >
          <div class="shortcut-icon">
            <el-icon :size="28"><component :is="tool.meta.icon" /></el-icon>
          </div>
          <div class="shortcut-name">{{ tool.meta.title }}</div>
          <div class="shortcut-desc">{{ tool.meta.description }}</div>
        </div>
      </div>
    </div>

    <!-- 搜索结果 -->
    <div v-if="searchText" class="search-results">
      <div v-if="searchResults.length === 0" class="no-results">
        <el-icon :size="48"><Search /></el-icon>
        <p>未找到相关工具</p>
      </div>
      <div v-else class="results-grid">
        <div
          v-for="tool in searchResults"
          :key="tool.path"
          class="result-card"
          @click="goTo(tool.path)"
        >
          <el-icon :size="24"><component :is="tool.meta.icon" /></el-icon>
          <div class="result-info">
            <div class="result-title">{{ tool.meta.title }}</div>
            <div class="result-desc">{{ tool.meta.description }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 工具分类 -->
    <div v-if="!searchText" class="categories-section">
      <div
        v-for="(tools, category) in categorizedTools"
        :key="category"
        class="category-block"
      >
        <div class="section-title">
          <el-icon><component :is="categories[category]?.icon || 'Tools'" /></el-icon>
          {{ categories[category]?.name || '其他' }}
        </div>
        <div class="tools-grid">
          <div
            v-for="tool in tools"
            :key="tool.path"
            class="tool-card"
            @click="goTo(tool.path)"
          >
            <el-icon :size="22"><component :is="tool.meta.icon" /></el-icon>
            <span class="tool-name">{{ tool.meta.title }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { toolRoutes, categories } from '../router'

const router = useRouter()
const searchText = ref('')

// 过滤掉分享类路由
const validTools = computed(() =>
  toolRoutes.filter(t => !t.path.includes('/:'))
)

// 常用工具
const shortcutTools = computed(() =>
  validTools.value.filter(t => t.meta?.shortcut).slice(0, 6)
)

// 搜索结果
const searchResults = computed(() => {
  if (!searchText.value) return []
  const keyword = searchText.value.toLowerCase()
  return validTools.value.filter(t =>
    t.meta?.title?.toLowerCase().includes(keyword) ||
    t.meta?.description?.toLowerCase().includes(keyword)
  )
})

// 分类工具
const categorizedTools = computed(() => {
  const result = {}
  validTools.value.forEach(tool => {
    const category = tool.meta?.category || 'other'
    if (!result[category]) result[category] = []
    result[category].push(tool)
  })
  return result
})

const goTo = (path) => {
  router.push(path)
}

const goToFirst = () => {
  if (searchResults.value.length > 0) {
    goTo(searchResults.value[0].path)
  }
}

// URL 参数跳转
onMounted(() => {
  const q = new URLSearchParams(location.search).get('q')
  if (q) {
    searchText.value = q
    const results = searchResults.value
    if (results.length > 0) {
      goTo(results[0].path)
    }
  }
})
</script>

<style scoped>
.home-container {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
}

.search-section {
  margin-bottom: 32px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  background: var(--bg-overlay);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 12px 16px;
  transition: all 0.2s;
}

.search-box:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1);
}

.search-icon {
  color: var(--text-tertiary);
  margin-right: 12px;
  font-size: 20px;
}

.search-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 16px;
  color: var(--text-primary);
  outline: none;
}

.search-input::placeholder {
  color: var(--text-tertiary);
}

.search-clear {
  cursor: pointer;
  color: var(--text-tertiary);
  padding: 4px;
}

.search-clear:hover {
  color: var(--text-primary);
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 快捷入口 */
.shortcuts-section {
  margin-bottom: 32px;
}

.shortcuts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.shortcut-card {
  background: var(--bg-overlay);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 20px 16px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
}

.shortcut-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.shortcut-icon {
  color: var(--color-primary);
  margin-bottom: 12px;
}

.shortcut-name {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.shortcut-desc {
  font-size: 12px;
  color: var(--text-tertiary);
}

/* 搜索结果 */
.search-results {
  margin-bottom: 32px;
}

.no-results {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-tertiary);
}

.no-results p {
  margin-top: 16px;
}

.results-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.result-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.result-card:hover {
  border-color: var(--color-primary);
  background: var(--bg-base);
}

.result-info {
  flex: 1;
}

.result-title {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.result-desc {
  font-size: 13px;
  color: var(--text-tertiary);
}

/* 分类区域 */
.categories-section {
  margin-bottom: 32px;
}

.category-block {
  margin-bottom: 28px;
}

.tools-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tool-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: var(--bg-overlay);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--text-primary);
}

.tool-card:hover {
  border-color: var(--color-primary);
  background: var(--bg-base);
}

.tool-name {
  font-size: 14px;
}
</style>
