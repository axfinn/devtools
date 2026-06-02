<template>
  <div class="home-workbench">
    <section class="workbench-hero">
      <div class="hero-copy">
        <span class="hero-kicker">DevTools Workbench</span>
        <h1>今天要处理什么？</h1>
        <p>把高频工具、生活记录和 AI 工作流放在更短的路径里，移动端优先进入常用动作。</p>
      </div>

      <div class="search-card">
        <div class="search-box">
          <el-icon class="search-icon"><Search /></el-icon>
          <input
            v-model="searchText"
            type="text"
            placeholder="搜索 JSON、记账、短链、AI..."
            class="search-input"
            @keyup.enter="goToFirst"
          />
          <button v-if="searchText" type="button" class="search-clear" @click="searchText = ''">
            <el-icon><Close /></el-icon>
          </button>
        </div>
      </div>
    </section>

    <section v-if="searchText" class="search-results">
      <div class="section-head">
        <h2>搜索结果</h2>
        <span>{{ searchResults.length }} 个工具</span>
      </div>
      <div v-if="searchResults.length === 0" class="empty-state">
        <el-icon :size="42"><Search /></el-icon>
        <p>未找到相关工具</p>
      </div>
      <div v-else class="result-list">
        <div
          v-for="tool in searchResults"
          :key="tool.path"
          role="button"
          tabindex="0"
          class="result-row"
          @click="goTo(tool.path)"
          @keydown.enter="goTo(tool.path)"
        >
          <span class="result-icon" :style="{ color: categoryMeta(tool).accent }">
            <el-icon><component :is="tool.meta.icon" /></el-icon>
          </span>
          <span class="result-main">
            <strong>{{ tool.meta.title }}</strong>
            <small>{{ tool.meta.description }}</small>
          </span>
          <button
            type="button"
            class="favorite-btn"
            :class="{ active: isFavorite(tool.path) }"
            :aria-label="isFavorite(tool.path) ? '取消收藏' : '收藏工具'"
            @click.stop="toggleFavorite(tool.path)"
          >
            <el-icon><Star /></el-icon>
          </button>
          <el-icon class="row-arrow"><ArrowRight /></el-icon>
        </div>
      </div>
    </section>

    <template v-else>
      <section class="quick-section">
        <div class="section-head">
          <h2>{{ hasPersonalTools ? '我的常用' : '常用入口' }}</h2>
          <span>{{ hasPersonalTools ? '最近与收藏优先' : '一触即达' }}</span>
        </div>
        <div class="quick-grid">
          <article
            v-for="tool in preferredTools"
            :key="tool.path"
            role="button"
            tabindex="0"
            class="quick-card"
            :style="{ '--tool-accent': categoryMeta(tool).accent }"
            @click="goTo(tool.path)"
            @keydown.enter="goTo(tool.path)"
          >
            <span class="quick-icon">
              <el-icon><component :is="tool.meta.icon" /></el-icon>
            </span>
            <button
              type="button"
              class="quick-fav"
              :class="{ active: isFavorite(tool.path) }"
              :aria-label="isFavorite(tool.path) ? '取消收藏' : '收藏工具'"
              @click.stop="toggleFavorite(tool.path)"
            >
              <el-icon><Star /></el-icon>
            </button>
            <span class="quick-name">{{ tool.meta.title }}</span>
            <small>{{ tool.meta.description }}</small>
          </article>
        </div>
      </section>

      <section class="scenario-section">
        <div class="section-head">
          <h2>按场景进入</h2>
          <span>{{ validTools.length }} 个工具</span>
        </div>
        <div class="scenario-grid">
          <article
            v-for="group in orderedCategories"
            :key="group.key"
            class="scenario-card"
            :style="{ '--group-accent': group.meta.accent }"
          >
            <button type="button" class="scenario-head" @click="activeCategory = activeCategory === group.key ? '' : group.key">
              <span class="scenario-icon">
                <el-icon><component :is="group.meta.icon" /></el-icon>
              </span>
              <span class="scenario-copy">
                <strong>{{ group.meta.name }}</strong>
                <small>{{ group.meta.description }}</small>
              </span>
              <span class="scenario-count">{{ group.tools.length }}</span>
            </button>
            <div class="scenario-tools" :class="{ expanded: activeCategory === group.key }">
              <div
                v-for="tool in group.tools"
                :key="tool.path"
                role="button"
                tabindex="0"
                class="tool-pill"
                @click="goTo(tool.path)"
                @keydown.enter="goTo(tool.path)"
              >
                <el-icon><component :is="tool.meta.icon" /></el-icon>
                <span>{{ tool.meta.title }}</span>
                <button
                  type="button"
                  class="pill-fav"
                  :class="{ active: isFavorite(tool.path) }"
                  :aria-label="isFavorite(tool.path) ? '取消收藏' : '收藏工具'"
                  @click.stop="toggleFavorite(tool.path)"
                >
                  <el-icon><Star /></el-icon>
                </button>
              </div>
            </div>
          </article>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, Close, ArrowRight, Star } from '@element-plus/icons-vue'
import { categories } from '../../router'
import { useToolPreferences } from '../../composables/useToolPreferences'
import { useToolRegistry } from '../../composables/useToolRegistry'

const router = useRouter()
const searchText = ref('')
const activeCategory = ref('dev')
const { recentPaths, favoritePaths, rememberTool, toggleFavorite, isFavorite, sortRoutesByPreference } = useToolPreferences()
const { visibleTools, loadHiddenRoutes } = useToolRegistry()

const validTools = computed(() => visibleTools.value)

const shortcutTools = computed(() =>
  validTools.value
    .filter(t => t.meta?.shortcut)
    .sort((a, b) => (b.meta?.shortcutPriority || 0) - (a.meta?.shortcutPriority || 0))
    .slice(0, 8)
)

const hasPersonalTools = computed(() => favoritePaths.value.length > 0 || recentPaths.value.length > 0)

const preferredTools = computed(() => {
  if (!hasPersonalTools.value) return shortcutTools.value
  return sortRoutesByPreference(validTools.value).slice(0, 8)
})

const searchResults = computed(() => {
  if (!searchText.value) return []
  const keyword = searchText.value.toLowerCase().trim()
  return validTools.value.filter(t =>
    t.meta?.title?.toLowerCase().includes(keyword) ||
    t.meta?.description?.toLowerCase().includes(keyword) ||
    t.name?.toLowerCase().includes(keyword)
  )
})

const orderedCategories = computed(() => {
  const groups = {}
  validTools.value.forEach(tool => {
    const category = tool.meta?.category || 'other'
    if (!groups[category]) groups[category] = []
    groups[category].push(tool)
  })
  return Object.entries(groups)
    .map(([key, tools]) => ({
      key,
      tools,
      meta: categories[key] || categories.other
    }))
    .sort((a, b) => (a.meta?.order || 999) - (b.meta?.order || 999))
})

function categoryMeta(tool) {
  return categories[tool.meta?.category] || categories.other
}

function goTo(path) {
  rememberTool(path)
  router.push(path)
}

function goToFirst() {
  if (searchResults.value.length > 0) {
    goTo(searchResults.value[0].path)
  }
}

onMounted(() => {
  loadHiddenRoutes()
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
.home-workbench {
  width: min(1120px, 100%);
  margin: 0 auto;
  padding: 6px 0 28px;
}

.workbench-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 440px);
  gap: 22px;
  align-items: end;
  margin-bottom: 24px;
  padding: 28px;
  border: 1px solid var(--border-base);
  border-radius: 18px;
  background:
    linear-gradient(135deg, rgba(64, 158, 255, 0.12), transparent 34%),
    linear-gradient(160deg, var(--bg-overlay), var(--bg-secondary));
}

.hero-kicker {
  display: inline-flex;
  color: var(--color-primary);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.hero-copy h1 {
  margin: 8px 0 10px;
  color: var(--text-primary);
  font-size: 32px;
  line-height: 1.15;
  letter-spacing: 0;
}

.hero-copy p {
  max-width: 620px;
  margin: 0;
  color: var(--text-secondary);
  font-size: 15px;
  line-height: 1.7;
}

.search-card {
  padding: 12px;
  border: 1px solid var(--border-base);
  border-radius: 16px;
  background: var(--bg-primary);
  box-shadow: 0 12px 34px rgba(15, 23, 42, 0.08);
}

.search-box {
  min-height: 52px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  border: 1px solid var(--border-base);
  border-radius: 12px;
  background: var(--bg-base);
  transition: all 0.2s;
}

.search-box:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.12);
}

.search-icon {
  color: var(--text-tertiary);
  font-size: 20px;
  flex-shrink: 0;
}

.search-input {
  width: 100%;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 16px;
}

.search-input::placeholder {
  color: var(--text-tertiary);
}

.search-clear {
  width: 34px;
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: none;
  border-radius: 10px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  cursor: pointer;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 0 0 12px;
}

.section-head h2 {
  margin: 0;
  color: var(--text-primary);
  font-size: 18px;
  letter-spacing: 0;
}

.section-head span {
  color: var(--text-tertiary);
  font-size: 13px;
}

.quick-section,
.scenario-section,
.search-results {
  margin-top: 22px;
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.quick-card {
  position: relative;
  min-width: 0;
  min-height: 132px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--border-base);
  border-radius: 14px;
  background: var(--bg-overlay);
  color: var(--text-primary);
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: all 0.18s ease;
}

.quick-card:hover {
  transform: translateY(-2px);
  border-color: var(--tool-accent);
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08);
}

.quick-icon {
  width: 38px;
  height: 38px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: color-mix(in srgb, var(--tool-accent) 13%, transparent);
  color: var(--tool-accent);
  font-size: 18px;
}

.quick-fav,
.favorite-btn,
.pill-fav {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.18s ease;
}

.quick-fav {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 30px;
  height: 30px;
  border-radius: 10px;
  background: var(--bg-secondary);
}

.quick-fav.active,
.favorite-btn.active,
.pill-fav.active {
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.12);
}

.quick-name {
  font-size: 15px;
  font-weight: 800;
}

.quick-card small {
  color: var(--text-tertiary);
  font-size: 12px;
  line-height: 1.45;
}

.scenario-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.scenario-card {
  min-width: 0;
  border: 1px solid var(--border-base);
  border-radius: 16px;
  background: var(--bg-overlay);
  overflow: hidden;
}

.scenario-head {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-family: inherit;
  text-align: left;
  cursor: pointer;
}

.scenario-icon {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: color-mix(in srgb, var(--group-accent) 12%, transparent);
  color: var(--group-accent);
}

.scenario-copy {
  min-width: 0;
}

.scenario-copy strong {
  display: block;
  color: var(--text-primary);
  font-size: 16px;
  line-height: 1.2;
}

.scenario-copy small {
  display: block;
  margin-top: 4px;
  color: var(--text-tertiary);
  font-size: 12px;
  line-height: 1.45;
}

.scenario-count {
  min-width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 800;
}

.scenario-tools {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  max-height: 0;
  padding: 0 14px;
  overflow: hidden;
  transition: max-height 0.24s ease, padding-bottom 0.24s ease;
}

.scenario-tools.expanded {
  max-height: 520px;
  padding-bottom: 14px;
}

.tool-pill {
  min-width: 0;
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid var(--border-base);
  border-radius: 10px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 13px;
  cursor: pointer;
}

.pill-fav {
  width: 26px;
  height: 26px;
  margin-left: auto;
  flex-shrink: 0;
  border-radius: 8px;
  background: transparent;
}

.tool-pill span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.result-row {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--border-base);
  border-radius: 12px;
  background: var(--bg-overlay);
  color: var(--text-primary);
  font-family: inherit;
  text-align: left;
  cursor: pointer;
}

.favorite-btn {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  background: var(--bg-secondary);
}

.result-icon {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: var(--bg-secondary);
}

.result-main {
  min-width: 0;
}

.result-main strong,
.result-main small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-main strong {
  font-size: 15px;
}

.result-main small {
  margin-top: 3px;
  color: var(--text-tertiary);
  font-size: 12px;
}

.row-arrow {
  color: var(--text-tertiary);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 54px 20px;
  border: 1px dashed var(--border-base);
  border-radius: 16px;
  color: var(--text-tertiary);
}

.empty-state p {
  margin: 0;
}

@media (max-width: 900px) {
  .workbench-hero {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .quick-grid,
  .scenario-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .home-workbench {
    padding: 0 0 18px;
  }

  .workbench-hero {
    gap: 16px;
    margin: -2px -2px 18px;
    padding: 20px 16px;
    border-radius: 16px;
  }

  .hero-copy h1 {
    font-size: 26px;
  }

  .hero-copy p {
    font-size: 14px;
  }

  .search-card {
    padding: 8px;
    border-radius: 14px;
  }

  .quick-grid {
    display: flex;
    gap: 10px;
    overflow-x: auto;
    padding: 1px 2px 8px;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }

  .quick-grid::-webkit-scrollbar {
    display: none;
  }

  .quick-card {
    width: 148px;
    min-height: 126px;
    flex: 0 0 auto;
    padding: 14px;
  }

  .scenario-grid {
    grid-template-columns: 1fr;
  }

  .scenario-tools {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 420px) {
  .workbench-hero {
    padding: 18px 14px;
  }

  .hero-copy h1 {
    font-size: 24px;
  }

  .section-head h2 {
    font-size: 17px;
  }

  .scenario-head {
    padding: 14px;
  }

  .scenario-tools {
    grid-template-columns: 1fr;
  }
}
</style>
