// 统一 API 配置
export const API_BASE = import.meta.env.VITE_API_BASE || ''

// WebSocket 基础地址
export const WS_BASE = (() => {
  if (import.meta.env.VITE_WS_BASE) {
    return import.meta.env.VITE_WS_BASE
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}`
})()
