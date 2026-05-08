import { computed, ref } from 'vue'

const RECENT_KEY = 'devtools_recent_tools'
const FAVORITE_KEY = 'devtools_favorite_tools'
const MAX_RECENT = 8

const recentPaths = ref(readList(RECENT_KEY))
const favoritePaths = ref(readList(FAVORITE_KEY))

function readList(key) {
  try {
    const value = JSON.parse(localStorage.getItem(key) || '[]')
    return Array.isArray(value) ? value.filter(Boolean) : []
  } catch {
    return []
  }
}

function writeList(key, value) {
  localStorage.setItem(key, JSON.stringify(value))
}

function rememberTool(path) {
  if (!path) return
  recentPaths.value = [path, ...recentPaths.value.filter(item => item !== path)].slice(0, MAX_RECENT)
  writeList(RECENT_KEY, recentPaths.value)
}

function toggleFavorite(path) {
  if (!path) return
  if (favoritePaths.value.includes(path)) {
    favoritePaths.value = favoritePaths.value.filter(item => item !== path)
  } else {
    favoritePaths.value = [path, ...favoritePaths.value]
  }
  writeList(FAVORITE_KEY, favoritePaths.value)
}

function isFavorite(path) {
  return favoritePaths.value.includes(path)
}

function sortRoutesByPreference(routes = []) {
  const routeMap = new Map(routes.map(route => [route.path, route]))
  const preferred = [...favoritePaths.value, ...recentPaths.value]
  const seen = new Set()
  const result = []

  preferred.forEach(path => {
    const route = routeMap.get(path)
    if (route && !seen.has(path)) {
      result.push(route)
      seen.add(path)
    }
  })

  routes.forEach(route => {
    if (!seen.has(route.path)) result.push(route)
  })

  return result
}

export function useToolPreferences() {
  return {
    recentPaths: computed(() => recentPaths.value),
    favoritePaths: computed(() => favoritePaths.value),
    rememberTool,
    toggleFavorite,
    isFavorite,
    sortRoutesByPreference
  }
}
