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
    path: '/autodev',
    name: 'AutoDev',
    component: () => import('../views/AutoDevTool.vue'),
    meta: {
      title: 'AutoDev AI',
      icon: 'MagicStick',
      category: 'dev',
      description: 'AI 自动开发任务助手'
    }
  },
  {
    path: '/proxy',
    name: 'Proxy',
    component: () => import('../views/ProxyTool.vue'),
    meta: {
      title: '科学上网',
      icon: 'Connection',
      category: 'other',
      description: 'Clash 节点管理与代理'
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
    path: '/mindmap',
    name: 'MindMap',
    component: () => import('../views/MindMapTool.vue'),
    meta: {
      title: '思维导图',
      icon: 'Share',
      category: 'draw',
      shortcut: true,
      description: '思维导图，支持 AI 生成、云端保存、分享'
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
    path: '/nfsshare',
    name: 'NFSShare',
    component: () => import('../views/NFSShareTool.vue'),
    meta: {
      title: 'NFS 文件分享',
      icon: 'FolderOpened',
      category: 'collab',
      description: 'NFS 挂载文件受控分享，超管专用'
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
  {
    path: '/expense',
    name: 'Expense',
    component: () => import('../views/ExpenseTool.vue'),
    meta: {
      title: '生活记账',
      icon: 'Money',
      category: 'life',
      description: '日常开支记录统计'
    }
  },
  {
    path: '/household',
    name: 'Household',
    component: () => import('../views/HouseholdTool.vue'),
    meta: {
      title: '整理模块',
      icon: 'Box',
      category: 'life',
      description: '家庭物品管理与库存整理'
    }
  },
  {
    path: '/household/space',
    name: 'HouseholdSpace',
    component: () => import('../views/HouseholdSpace3D.vue'),
    meta: {
      title: '整理空间3D',
      icon: 'Location',
      category: 'life',
      description: '位置标注与空间规划'
    }
  },
  {
    path: '/glucose',
    name: 'Glucose',
    component: () => import('../views/GlucoseTool.vue'),
    meta: {
      title: '血糖记录',
      icon: 'FirstAidKit',
      category: 'life',
      description: '血糖检测记录与分析'
    }
  },
  {
    path: '/qrcode',
    name: 'QrCode',
    component: () => import('../views/QrCodeTool.vue'),
    meta: {
      title: '二维码',
      icon: 'Picture',
      category: 'convert',
      description: '二维码生成与解析'
    }
  },
  {
    path: '/background',
    name: 'Background',
    component: () => import('../views/BackgroundTool.vue'),
    meta: {
      title: '背景图库',
      icon: 'PictureFilled',
      category: 'other',
      description: '博客动态背景图片'
    }
  },
  {
    path: '/bailian-image',
    name: 'BailianImage',
    component: () => import('../views/BailianImageTool.vue'),
    meta: {
      title: '百炼图片',
      icon: 'Picture',
      category: 'other',
      description: '阿里云百炼图片/视频模型调试台'
    }
  },
  {
    path: '/image-understanding',
    name: 'ImageUnderstanding',
    component: () => import('../views/ImageUnderstandingTool.vue'),
    meta: {
      title: '图像理解',
      icon: 'Picture',
      category: 'other',
      description: 'MiniMax MCP 图像理解'
    }
  },
  {
    path: '/ai-gateway',
    name: 'AIGateway',
    component: () => import('../views/AIGatewayTool.vue'),
    meta: {
      title: 'AI Gateway',
      icon: 'Key',
      category: 'other',
      description: '统一 AI API Key 管理与对外开放'
    }
  },
  {
    path: '/edge-tts',
    name: 'EdgeTTS',
    component: () => import('../views/EdgeTTSTool.vue'),
    meta: {
      title: 'Edge TTS',
      icon: 'Headset',
      category: 'dev',
      description: 'Microsoft Edge 语音合成，支持多种中文音色'
    }
  },
  {
    path: '/games',
    name: 'GameHall',
    component: () => import('../views/GameHall.vue'),
    meta: {
      title: '小游戏',
      icon: 'Connection',
      category: 'life',
      description: '双人小游戏，支持 AI 陪玩'
    }
  },
  {
    path: '/image-viewer',
    name: 'ImageViewer',
    component: () => import('../views/ImageViewerTool.vue'),
    meta: {
      title: '批量看图',
      icon: 'Picture',
      category: 'other',
      description: '批量输入图片 URL，逐一对照展示，支持跨域代理'
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
  },
  {
    path: '/nfs/:id',
    name: 'NFSShareView',
    component: () => import('../views/NFSShareView.vue'),
    meta: { title: '文件分享', icon: 'FolderOpened', hideSidebar: true }
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
