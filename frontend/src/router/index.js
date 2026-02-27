import { createRouter, createWebHistory } from 'vue-router'

// 工具分类配置
const categories = {
  dev: { name: '开发工具', icon: 'Monitor' },
  convert: { name: '转换工具', icon: 'Switch' },
  draw: { name: '绘图/图表', icon: 'Edit' },
  collab: { name: '协作分享', icon: 'Connection' },
  life: { name: '生活工具', icon: 'Calendar' },
  other: { name: '其他工具', icon: 'Tools' }
}

// 工具路由配置
const toolRoutes = [
  {
    path: '/json',
    name: 'JSON',
    component: () => import('../views/JsonTool.vue'),
    meta: {
      title: 'JSON 格式化',
      icon: 'Document',
      category: 'dev',
      shortcut: true,
      description: 'JSON 格式化、校验、压缩'
    }
  },
  {
    path: '/diff',
    name: 'Diff',
    component: () => import('../views/DiffTool.vue'),
    meta: {
      title: 'Diff 对比',
      icon: 'Switch',
      category: 'dev',
      shortcut: true,
      description: '文本对比、差异高亮'
    }
  },
  {
    path: '/mockapi',
    name: 'MockApi',
    component: () => import('../views/MockApi.vue'),
    meta: {
      title: 'Mock API',
      icon: 'Connection',
      category: 'dev',
      description: '模拟 API 接口'
    }
  },
  {
    path: '/regex',
    name: 'Regex',
    component: () => import('../views/RegexTool.vue'),
    meta: {
      title: '正则测试',
      icon: 'Search',
      category: 'dev',
      description: '正则表达式测试'
    }
  },
  {
    path: '/terminal',
    name: 'Terminal',
    component: () => import('../views/TerminalTool.vue'),
    meta: {
      title: 'SSH 终端',
      icon: 'Monitor',
      category: 'dev',
      description: '远程 SSH 连接'
    }
  },
  {
    path: '/base64',
    name: 'Base64',
    component: () => import('../views/Base64Tool.vue'),
    meta: {
      title: 'Base64',
      icon: 'Key',
      category: 'convert',
      shortcut: true,
      description: 'Base64 编码解码'
    }
  },
  {
    path: '/url',
    name: 'URL',
    component: () => import('../views/UrlTool.vue'),
    meta: {
      title: 'URL 编解码',
      icon: 'Link',
      category: 'convert',
      shortcut: true,
      description: 'URL 编码解码'
    }
  },
  {
    path: '/timestamp',
    name: 'Timestamp',
    component: () => import('../views/TimestampTool.vue'),
    meta: {
      title: '时间戳',
      icon: 'Clock',
      category: 'convert',
      description: '时间戳转换'
    }
  },
  {
    path: '/text',
    name: 'Text',
    component: () => import('../views/TextTool.vue'),
    meta: {
      title: '文本转换',
      icon: 'ChatDotRound',
      category: 'convert',
      description: '文本大小写、排序等'
    }
  },
  {
    path: '/replace',
    name: 'Replace',
    component: () => import('../views/ReplaceTool.vue'),
    meta: {
      title: '批量替换',
      icon: 'Switch',
      category: 'convert',
      description: '批量文本替换'
    }
  },
  {
    path: '/dns',
    name: 'DNS',
    component: () => import('../views/DnsTool.vue'),
    meta: {
      title: 'IP/DNS',
      icon: 'Position',
      category: 'convert',
      description: 'IP 查询、DNS 解析'
    }
  },
  {
    path: '/markdown',
    name: 'Markdown',
    component: () => import('../views/MarkdownTool.vue'),
    meta: {
      title: 'Markdown',
      icon: 'EditPen',
      category: 'draw',
      shortcut: true,
      description: 'Markdown 编辑预览'
    }
  },
  {
    path: '/mermaid',
    name: 'Mermaid',
    component: () => import('../views/MermaidTool.vue'),
    meta: {
      title: 'Mermaid 图表',
      icon: 'Share',
      category: 'draw',
      description: '流程图、时序图'
    }
  },
  {
    path: '/excalidraw',
    name: 'Excalidraw',
    component: () => import('../views/ExcalidrawTool.vue'),
    meta: {
      title: 'Excalidraw 画图',
      icon: 'Edit',
      category: 'draw',
      description: '手绘风格图表'
    }
  },
  {
    path: '/vibe',
    name: 'VibeMotion',
    component: () => import('../views/VibeMotionTool.vue'),
    meta: {
      title: 'AI 动效',
      icon: 'VideoCamera',
      category: 'draw',
      description: 'AI 动效生成'
    }
  },
  {
    path: '/paste',
    name: 'PasteBin',
    component: () => import('../views/PasteBin.vue'),
    meta: {
      title: '粘贴板',
      icon: 'DocumentCopy',
      category: 'collab',
      shortcut: true,
      description: '代码片段分享'
    }
  },
  {
    path: '/shorturl',
    name: 'ShortUrl',
    component: () => import('../views/ShortUrl.vue'),
    meta: {
      title: '短链生成',
      icon: 'Link',
      category: 'collab',
      description: 'URL 短链接生成'
    }
  },
  {
    path: '/chat',
    name: 'Chat',
    component: () => import('../views/ChatRoom.vue'),
    meta: {
      title: '聊天室',
      icon: 'ChatLineSquare',
      category: 'collab',
      description: '实时聊天房间'
    }
  },
  {
    path: '/yun',
    name: 'Pregnancy',
    component: () => import('../views/PregnancyTool.vue'),
    meta: {
      title: '孕期管理',
      icon: 'Calendar',
      category: 'life',
      description: '孕期记录工具'
    }
  },
  {
    path: '/recipe',
    name: 'Recipe',
    component: () => import('../views/RecipeTool.vue'),
    meta: {
      title: '每日菜谱',
      icon: 'Food',
      category: 'life',
      description: '菜谱查询记录'
    }
  },
  // 分享类路由（不显示在侧边栏）
  {
    path: '/md/:id',
    name: 'MarkdownShare',
    component: () => import('../views/MarkdownShareView.vue'),
    meta: { title: 'Markdown 分享', icon: 'EditPen', hideSidebar: true }
  },
  {
    path: '/paste/:id',
    name: 'PasteView',
    component: () => import('../views/PasteView.vue'),
    meta: { title: '查看分享', icon: 'DocumentCopy', hideSidebar: true }
  },
  {
    path: '/draw/:id',
    name: 'ExcalidrawShare',
    component: () => import('../views/ExcalidrawShareView.vue'),
    meta: { title: '查看画图', icon: 'Edit', hideSidebar: true }
  }
]

// 首页路由
const homeRoute = {
  path: '/',
  name: 'Home',
  component: () => import('../views/HomeView.vue'),
  meta: { title: '首页', icon: 'Home', category: 'home' }
}

// 分享跳转路由
const shareRoutes = [
  {
    path: '/s/:id',
    name: 'ShortUrlRedirect',
    component: () => import('../views/HomeView.vue'),
    meta: { title: '跳转中...', hideSidebar: true }
  }
]

const routes = [
  homeRoute,
  ...toolRoutes,
  ...shareRoutes
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

// 导出供其他地方使用
export { categories, toolRoutes }
export default router
