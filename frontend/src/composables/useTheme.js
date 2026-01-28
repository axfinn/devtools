import { ref, watch, onMounted } from 'vue'

// 主题模式: 'light' | 'dark' | 'auto'
const themeMode = ref('auto')
// 当前实际应用的主题: 'light' | 'dark'
const currentTheme = ref('dark')

export function useTheme() {
  // 从 localStorage 加载主题设置
  const loadThemeFromStorage = () => {
    const saved = localStorage.getItem('theme-mode')
    if (saved && ['light', 'dark', 'auto'].includes(saved)) {
      themeMode.value = saved
    }
  }

  // 保存主题设置到 localStorage
  const saveThemeToStorage = (mode) => {
    localStorage.setItem('theme-mode', mode)
  }

  // 检测系统主题
  const getSystemTheme = () => {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }
    return 'light'
  }

  // 应用主题到 DOM
  const applyTheme = (theme) => {
    const html = document.documentElement
    const body = document.body
    if (theme === 'dark') {
      html.classList.add('dark')
      body.classList.add('dark')
    } else {
      html.classList.remove('dark')
      body.classList.remove('dark')
    }
    currentTheme.value = theme
  }

  // 更新主题
  const updateTheme = () => {
    let theme
    if (themeMode.value === 'auto') {
      theme = getSystemTheme()
    } else {
      theme = themeMode.value
    }
    applyTheme(theme)
  }

  // 设置主题模式
  const setThemeMode = (mode) => {
    themeMode.value = mode
    saveThemeToStorage(mode)
    updateTheme()
  }

  // 切换到下一个主题模式: auto -> light -> dark -> auto
  const toggleTheme = () => {
    const modes = ['auto', 'light', 'dark']
    const currentIndex = modes.indexOf(themeMode.value)
    const nextIndex = (currentIndex + 1) % modes.length
    setThemeMode(modes[nextIndex])
  }

  // 监听系统主题变化（仅在 auto 模式下生效）
  const setupSystemThemeListener = () => {
    if (!window.matchMedia) return

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const listener = (e) => {
      if (themeMode.value === 'auto') {
        applyTheme(e.matches ? 'dark' : 'light')
      }
    }

    // 现代浏览器使用 addEventListener
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', listener)
    } else {
      // 旧浏览器使用 addListener
      mediaQuery.addListener(listener)
    }

    return () => {
      if (mediaQuery.removeEventListener) {
        mediaQuery.removeEventListener('change', listener)
      } else {
        mediaQuery.removeListener(listener)
      }
    }
  }

  // 初始化
  const init = () => {
    loadThemeFromStorage()
    updateTheme()
    const cleanup = setupSystemThemeListener()
    return cleanup
  }

  return {
    themeMode,
    currentTheme,
    setThemeMode,
    toggleTheme,
    init
  }
}
