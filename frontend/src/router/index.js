import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/json'
  },
  {
    path: '/json',
    name: 'JSON',
    component: () => import('../views/JsonTool.vue'),
    meta: { title: 'JSON 工具', icon: 'Document' }
  },
  {
    path: '/diff',
    name: 'Diff',
    component: () => import('../views/DiffTool.vue'),
    meta: { title: 'Diff 对比', icon: 'Switch' }
  },
  {
    path: '/markdown',
    name: 'Markdown',
    component: () => import('../views/MarkdownTool.vue'),
    meta: { title: 'Markdown', icon: 'EditPen' }
  },
  {
    path: '/paste',
    name: 'PasteBin',
    component: () => import('../views/PasteBin.vue'),
    meta: { title: '粘贴板', icon: 'DocumentCopy' }
  },
  {
    path: '/paste/:id',
    name: 'PasteView',
    component: () => import('../views/PasteView.vue'),
    meta: { title: '查看分享', icon: 'DocumentCopy' }
  },
  {
    path: '/base64',
    name: 'Base64',
    component: () => import('../views/Base64Tool.vue'),
    meta: { title: 'Base64', icon: 'Key' }
  },
  {
    path: '/url',
    name: 'URL',
    component: () => import('../views/UrlTool.vue'),
    meta: { title: 'URL 编解码', icon: 'Link' }
  },
  {
    path: '/timestamp',
    name: 'Timestamp',
    component: () => import('../views/TimestampTool.vue'),
    meta: { title: '时间戳', icon: 'Clock' }
  },
  {
    path: '/regex',
    name: 'Regex',
    component: () => import('../views/RegexTool.vue'),
    meta: { title: '正则测试', icon: 'Search' }
  },
  {
    path: '/text',
    name: 'Text',
    component: () => import('../views/TextTool.vue'),
    meta: { title: '文本转换', icon: 'ChatDotRound' }
  },
  {
    path: '/dns',
    name: 'DNS',
    component: () => import('../views/DnsTool.vue'),
    meta: { title: 'IP/DNS', icon: 'Position' }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

export default router
