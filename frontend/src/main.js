import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import {
  Box,
  Calendar,
  ChatDotRound,
  ChatLineSquare,
  Clock,
  Close,
  Collection,
  Connection,
  Delete,
  Document,
  DocumentCopy,
  Edit,
  EditPen,
  FirstAidKit,
  FolderOpened,
  Food,
  Headset,
  HomeFilled,
  Key,
  Link,
  Location,
  MagicStick,
  Money,
  Monitor,
  Picture,
  PictureFilled,
  Position,
  QuestionFilled,
  Reading,
  Refresh,
  Search,
  Setting,
  Share,
  Switch,
  Timer,
  Tools,
  VideoPause,
  VideoPlay
} from '@element-plus/icons-vue'
import './styles/index.css'
import App from './App.vue'
import router from './router'

// 全局错误捕获:把异常显示在页面上,避免白屏看不到原因
function showFatalError(err, source) {
  console.error('[Fatal]', source, err)
  let box = document.getElementById('__fatal_error__')
  if (!box) {
    box = document.createElement('div')
    box.id = '__fatal_error__'
    box.style.cssText = 'position:fixed;top:0;left:0;right:0;z-index:99999;background:#fef2f2;color:#991b1b;padding:16px;font-family:monospace;font-size:13px;border-bottom:2px solid #dc2626;max-height:50vh;overflow:auto;white-space:pre-wrap;'
    document.body.appendChild(box)
  }
  const msg = (err && (err.stack || err.message)) || String(err)
  const divider = '\n\n--- ' + (source || 'error') + ' @ ' + new Date().toISOString() + ' ---\n'
  box.textContent += divider + msg
}
// 已知良性的 rejection,不显示到 fatal 浮层(但仍 console.warn 留底,方便排查)
//   - 视频 play() 被新的 load 请求打断:Chrome 在切换 src / 销毁重建视频元素时正常行为
//   - AbortError:fetch 主动 abort(切路由、组件卸载、用户取消上传等场景)
const BENIGN_REJECTION_PATTERNS = [
  /play\(\) request was interrupted/i,
  /\bAbortError\b/i,
]
function isBenignRejection(reason) {
  const msg = String((reason && (reason.stack || reason.message)) || reason || '')
  return BENIGN_REJECTION_PATTERNS.some(p => p.test(msg))
}

window.addEventListener('error', (e) => showFatalError(e.error || e.message, 'window.error'))
window.addEventListener('unhandledrejection', (e) => {
  if (isBenignRejection(e.reason)) {
    console.warn('[Benign rejection, suppressed]', e.reason)
    e.preventDefault()
    return
  }
  showFatalError(e.reason, 'unhandledrejection')
})

const app = createApp(App)
app.config.errorHandler = (err, instance, info) => {
  showFatalError(err, 'vue errorHandler [' + info + ']')
}

const globalIcons = {
  Box,
  Calendar,
  ChatDotRound,
  ChatLineSquare,
  Clock,
  Close,
  Collection,
  Connection,
  Delete,
  Document,
  DocumentCopy,
  Edit,
  EditPen,
  FirstAidKit,
  FolderOpened,
  Food,
  Headset,
  Home: HomeFilled,
  Key,
  Link,
  Location,
  MagicStick,
  Money,
  Monitor,
  Picture,
  PictureFilled,
  Position,
  QuestionFilled,
  Reading,
  Refresh,
  Search,
  Setting,
  Share,
  Switch,
  Timer,
  Tools,
  VideoPause,
  VideoPlay
}

for (const [key, component] of Object.entries(globalIcons)) {
  app.component(key, component)
}

app.use(ElementPlus)
app.use(router)
app.mount('#app')

if ('serviceWorker' in navigator && import.meta.env.PROD) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {})
  })
}
