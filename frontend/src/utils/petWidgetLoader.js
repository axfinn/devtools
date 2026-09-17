// FINN-29 A6 · pet-widget.js 推迟到首屏之后
//
// 旧实现：frontend/index.html 用 <script src="/widgets/pet-widget.js"></script>
// 同步加载，浏览器在解析 HTML 时会阻塞等 1.87 MB 全部下载执行完毕才继续，
// 首屏关键路径被它独占 ~17s（4G+4×CPU 实测）。
//
// 新实现：
//   - index.html 不再同步引用 pet-widget.js；
//   - 三处使用点（GlobalPet.vue / MiniMaxStudioTool.vue / PetTool.vue）在
//     onMounted 里调用 ensurePetWidget() 异步注入；
//   - 首次访客在 mount 后立即拉取（不阻塞首屏 DCL）；
//   - 回头访客（localStorage['pet-widget.global'] === '0'）→ 0 字节；
//     用户点 🐾 才拉。

const SRC = '/widgets/pet-widget.js'
const TAG = 'pet-widget'

let promise = null

export function ensurePetWidget() {
  if (promise) return promise
  if (typeof window === 'undefined') return Promise.resolve()
  if (typeof customElements !== 'undefined' && customElements.get(TAG)) {
    return Promise.resolve()
  }
  promise = new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = SRC
    s.async = true
    // fetchpriority='low' 是 Chromium 专属；其他浏览器忽略。
    // 这里只用于首屏后再加载的 widget，不与首屏关键资源抢带宽。
    s.setAttribute('fetchpriority', 'low')
    s.onload = () => {
      if (typeof customElements !== 'undefined' && customElements.whenDefined) {
        customElements.whenDefined(TAG).then(resolve, resolve)
      } else {
        resolve()
      }
    }
    s.onerror = () => {
      promise = null
      try { s.remove() } catch { /* ignore */ }
      reject(new Error('pet-widget.js 加载失败'))
    }
    document.head.appendChild(s)
  })
  return promise
}

export const PET_WIDGET_TAG = TAG