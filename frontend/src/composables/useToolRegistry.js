import { computed, ref } from 'vue'
import { toolRoutes } from '../router'

// 共享的隐藏路由状态（控制台配置），所有消费方共用同一份
const hiddenRoutes = ref([])
let loadPromise = null

// 统一判定：可在导航中出现的工具路由
function isNavigableTool(route) {
  return !!route.meta?.title &&
    !route.path.includes(':') &&
    !!route.meta?.category &&
    route.meta.category !== 'home'
}

// 全部工具（忽略隐藏），控制台用它来勾选显隐
const allTools = computed(() => toolRoutes.filter(isNavigableTool))

// 对外可见工具（应用隐藏配置），首页与侧边栏共用
// 控制台本身始终保留，避免隐藏后无法再进入
const visibleTools = computed(() =>
  allTools.value.filter(t => t.path === '/console' || !hiddenRoutes.value.includes(t.path))
)

function loadHiddenRoutes(force = false) {
  if (loadPromise && !force) return loadPromise
  loadPromise = fetch('/api/console/settings')
    .then(r => r.json())
    .then(d => { hiddenRoutes.value = d.hidden_routes || [] })
    .catch(() => {})
  return loadPromise
}

// 控制台保存后即时同步共享状态，无需刷新页面
function setHiddenRoutes(list) {
  hiddenRoutes.value = Array.isArray(list) ? [...list] : []
}

export function useToolRegistry() {
  return {
    hiddenRoutes: computed(() => hiddenRoutes.value),
    allTools,
    visibleTools,
    loadHiddenRoutes,
    setHiddenRoutes
  }
}
