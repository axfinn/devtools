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

const app = createApp(App)

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
