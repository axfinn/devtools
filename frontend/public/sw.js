// 占位 service worker,只为占住 scope 并立即激活,实际不缓存/不拦截任何 fetch。
// 之前的空 fetch handler(`self.addEventListener('fetch', () => {})`)会让
// Chrome 在每次导航时报 "Fetch event handler is recognized as no-op",
// 干脆不挂 fetch 监听,让请求直接落到 network。
self.addEventListener('install', event => {
  event.waitUntil(self.skipWaiting())
})

self.addEventListener('activate', event => {
  event.waitUntil(self.clients.claim())
})
