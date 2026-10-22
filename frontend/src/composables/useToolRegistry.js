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
    .then(r => {
      // 不判 res.ok 的话,后端 500 返回非 JSON body 时 r.json() 抛错,
      // 被下面的 catch 吞掉 → hiddenRoutes 保持默认全量可见,
      // 用户在控制台里勾的隐藏项刷新后悄悄失效,且没有任何提示。
      if (!r.ok) throw new Error(`console settings ${r.status}`)
      return r.json()
    })
    .then(d => { hiddenRoutes.value = d.hidden_routes || [] })
    .catch(() => {
      // 失败后清掉缓存的 promise:否则这个已 settle 的 promise 会被一直复用,
      // 之后即使后端恢复也永远不会重试(除非显式 force=true)。
      loadPromise = null
    })
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
