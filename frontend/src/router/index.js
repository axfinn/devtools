import { createRouter, createWebHistory } from 'vue-router'

// 工具分类配置
const categories = {
  dev: { name: '开发调试', icon: 'Monitor', description: '编码、接口、终端与调试', accent: '#2563eb', order: 10 },
  convert: { name: '文本转换', icon: 'Switch', description: '编码、解析、文本处理', accent: '#0891b2', order: 20 },
  draw: { name: '创作图表', icon: 'Edit', description: '文档、图表、白板与导图', accent: '#7c3aed', order: 30 },
  collab: { name: '协作分享', icon: 'Connection', description: '粘贴、短链、文件与聊天', accent: '#059669', order: 40 },
  ai: { name: 'AI 工作台', icon: 'MagicStick', description: '模型、媒体生成与网关管理', accent: '#db2777', order: 50 },
  life: { name: '生活记录', icon: 'Calendar', description: '记账、事项、健康与家庭', accent: '#ea580c', order: 60 },
  other: { name: '系统管理', icon: 'Tools', description: '代理、控制台与辅助工具', accent: '#475569', order: 70 }
}

// 工具路由配置
const toolRoutes = [
  {
    path: '/json',
    name: 'JSON',
    component: () => import('../views/dev/JsonTool.vue'),
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
    component: () => import('../views/dev/DiffTool.vue'),
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
    component: () => import('../views/dev/MockApi.vue'),
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
    component: () => import('../views/dev/RegexTool.vue'),
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
    component: () => import('../views/dev/TerminalTool.vue'),
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
    component: () => import('../views/dev/AutoDevTool.vue'),
    meta: {
      title: 'AutoDev AI',
      icon: 'MagicStick',
      category: 'dev',
      description: 'AI 自动开发任务助手'
    }
  },
  {
    path: '/hermes',
    name: 'Hermes',
    component: () => import('../views/dev/HermesTool.vue'),
    meta: {
      title: 'Hermes Agent',
      icon: 'Connection',
      category: 'dev',
      description: 'Hermes Dashboard、Gateway 与 API 调试'
    }
  },
  {
    path: '/proxy',
    name: 'Proxy',
    component: () => import('../views/other/ProxyTool.vue'),
    meta: {
      title: '科学上网',
      icon: 'Connection',
      category: 'other',
      description: 'Clash 节点管理与代理'
    }
  },
  {
    path: '/nps',
    name: 'NPS',
    component: () => import('../views/other/NPSTool.vue'),
    meta: {
      title: 'NPS 端口映射',
      icon: 'Share',
      category: 'other',
      description: 'NPS 端口映射快速管理'
    }
  },
  {
    path: '/askit-invites',
    name: 'AskitInvites',
    component: () => import('../views/other/AskitInviteTool.vue'),
    meta: {
      title: 'AskIt 邀请码',
      icon: 'Ticket',
      category: 'other',
      description: 'AskIt 云同步邀请码生成'
    }
  },
  {
    path: '/askit-mydata',
    name: 'AskitMyData',
    component: () => import('../views/other/AskitMyDataTool.vue'),
    meta: {
      title: '我的同步数据',
      icon: 'FolderOpened',
      category: 'other',
      description: '登录查看自己同步到云端的对话、笔记、书签等'
    }
  },
  {
    path: '/base64',
    name: 'Base64',
    component: () => import('../views/convert/Base64Tool.vue'),
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
    component: () => import('../views/convert/UrlTool.vue'),
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
    component: () => import('../views/convert/TimestampTool.vue'),
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
    component: () => import('../views/convert/TextTool.vue'),
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
    component: () => import('../views/convert/ReplaceTool.vue'),
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
    component: () => import('../views/convert/DnsTool.vue'),
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
    component: () => import('../views/draw/MarkdownTool.vue'),
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
    component: () => import('../views/draw/MermaidTool.vue'),
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
    component: () => import('../views/draw/MindMapTool.vue'),
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
    component: () => import('../views/draw/ExcalidrawTool.vue'),
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
    component: () => import('../views/collab/PasteBin.vue'),
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
    component: () => import('../views/collab/NFSShareTool.vue'),
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
    component: () => import('../views/collab/ShortUrl.vue'),
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
    component: () => import('../views/collab/ChatRoom.vue'),
    meta: {
      title: '聊天室',
      icon: 'ChatLineSquare',
      category: 'collab',
      description: '实时聊天房间'
    }
  },
  {
    path: '/screen',
    name: 'ScreenShare',
    component: () => import('../views/collab/ScreenShareTool.vue'),
    meta: {
      title: '屏幕共享',
      icon: 'VideoCamera',
      category: 'collab',
      shortcut: true,
      description: 'WebRTC 屏幕共享 + 远程协助'
    }
  },
  {
    path: '/screen/view/:id',
    name: 'ScreenView',
    component: () => import('../views/collab/ScreenView.vue'),
    meta: {
      title: '观看屏幕',
      hidden: true
    }
  },
  {
    path: '/yun',
    name: 'Pregnancy',
    component: () => import('../views/life/PregnancyTool.vue'),
    meta: {
      title: '孕期管理',
      icon: 'Calendar',
      category: 'life',
      shortcut: true,
      description: '孕期记录工具'
    }
  },
  {
    path: '/recipe',
    name: 'Recipe',
    component: () => import('../views/life/RecipeTool.vue'),
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
    component: () => import('../views/life/ExpenseTool.vue'),
    meta: {
      title: '生活记账',
      icon: 'Money',
      category: 'life',
      description: '日常开支记录统计'
    }
  },
  {
    path: '/planner',
    name: 'Planner',
    component: () => import('../views/life/PlannerTool.vue'),
    meta: {
      title: '事项管理',
      icon: 'Calendar',
      category: 'life',
      shortcut: true,
      description: '工作事项与生活事项分区管理，支持语音、提醒和日历导入'
    }
  },
  {
    path: '/household',
    name: 'Household',
    component: () => import('../views/life/HouseholdTool.vue'),
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
    component: () => import('../views/life/HouseholdSpace3D.vue'),
    meta: {
      title: '整理空间3D',
      icon: 'Location',
      category: 'life',
      description: '位置标注与空间规划'
    }
  },
  {
    path: '/photowall',
    name: 'PhotoWall',
    component: () => import('../views/life/PhotoWallTool.vue'),
    meta: {
      title: '档案照片墙',
      icon: 'PictureFilled',
      category: 'life',
      description: '分类、时间线、分享和打包下载的照片墙档案'
    }
  },
  {
    path: '/glucose',
    name: 'Glucose',
    component: () => import('../views/life/GlucoseTool.vue'),
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
    component: () => import('../views/convert/QrCodeTool.vue'),
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
    component: () => import('../views/other/BackgroundTool.vue'),
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
    component: () => import('../views/ai/BailianImageTool.vue'),
    meta: {
      title: '百炼图片',
      icon: 'Picture',
      category: 'ai',
      description: '阿里云百炼图片/视频模型调试台'
    }
  },
  {
    path: '/image-understanding',
    name: 'ImageUnderstanding',
    component: () => import('../views/ai/ImageUnderstandingTool.vue'),
    meta: {
      title: '图像理解',
      icon: 'Picture',
      category: 'ai',
      description: 'MiniMax MCP 图像理解'
    }
  },
  {
    path: '/ai-chat',
    name: 'AIChat',
    component: () => import('../views/ai/AIChatTool.vue'),
    meta: {
      title: 'AI Chat',
      icon: 'ChatDotRound',
      category: 'ai',
      shortcut: true,
      shortcutPriority: 95,
      description: '多模型智能对话 + AskIt 浏览器扩展'
    }
  },
  {
    path: '/ai-gateway',
    name: 'AIGateway',
    component: () => import('../views/ai/AIGatewayTool.vue'),
    meta: {
      title: 'AI Gateway',
      icon: 'Key',
      category: 'ai',
      description: '统一 AI API Key 管理与对外开放'
    }
  },
  {
    path: '/english-tutor',
    name: 'EnglishTutor',
    component: () => import('../views/ai/EnglishTutorTool.vue'),
    meta: {
      title: 'AI 学英语',
      icon: 'Reading',
      category: 'ai',
      shortcut: true,
      description: '翻译、拼读、音标、例句和纠错练习'
    }
  },
  {
    path: '/minimax-studio',
    name: 'MiniMaxStudio',
    component: () => import('../views/ai/MiniMaxStudioTool.vue'),
    meta: {
      title: 'MiniMax Studio',
      icon: 'MagicStick',
      category: 'ai',
      description: 'MiniMax 文本、语音、视频、音乐与结果归档工作台'
    }
  },
  {
    path: '/edge-tts',
    name: 'EdgeTTS',
    component: () => import('../views/dev/EdgeTTSTool.vue'),
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
    component: () => import('../views/life/GameHall.vue'),
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
    component: () => import('../views/other/ImageViewerTool.vue'),
    meta: {
      title: '批量看图',
      icon: 'Picture',
      category: 'other',
      description: '批量输入图片 URL，逐一对照展示，支持跨域代理'
    }
  },
  {
    path: '/console',
    name: 'Console',
    component: () => import('../views/other/ConsoleTool.vue'),
    meta: {
      title: '控制台',
      icon: 'Setting',
      category: 'other',
      description: '管理导航模块显隐'
    }
  },
	  {
	    path: '/voice-inbox',
	    name: 'VoiceInbox',
	    component: () => import('../views/life/VoiceInboxTool.vue'),
	    meta: {
	      title: '语音收件箱',
	      icon: 'Headset',
	      category: 'life',
	      shortcut: true,
	      shortcutPriority: 100,
	      description: '语音备忘录，自动转写为文字'
	    }
	  },
	  {
	    path: '/counter',
	    name: 'Counter',
	    component: () => import('../views/life/CounterTool.vue'),
	    meta: {
	      title: '敲击计数器',
	      icon: 'Timer',
	      category: 'life',
	      shortcut: true,
	      description: '多形象、多主题、带音效反馈的沉浸式计数器'
	    }
	  },
	  {
	    path: '/gallery',
	    name: 'MediaGallery',
	    component: () => import('../views/share/MediaGallery.vue'),
	    meta: {
	      title: '媒体画廊',
	      icon: 'PictureFilled',
	      category: 'ai',
	      description: 'AI 生成的音乐、视频、图片作品画廊'
	    }
	  },
  // 分享类路由（不显示在侧边栏）
  {
    path: '/md/:id',
    name: 'MarkdownShare',
    component: () => import('../views/share/MarkdownShareView.vue'),
    meta: { title: 'Markdown 分享', icon: 'EditPen', hideSidebar: true }
  },
  {
    path: '/paste/:id',
    name: 'PasteView',
    component: () => import('../views/share/PasteView.vue'),
    meta: { title: '查看分享', icon: 'DocumentCopy', hideSidebar: true }
  },
  {
    path: '/chat/:id',
    name: 'ChatShareView',
    component: () => import('../views/share/ChatShareView.vue'),
    meta: { title: '对话分享', icon: 'ChatLineSquare', hideSidebar: true }
  },
  {
    path: '/draw/:id',
    name: 'ExcalidrawShare',
    component: () => import('../views/share/ExcalidrawShareView.vue'),
    meta: { title: '查看画图', icon: 'Edit', hideSidebar: true }
  },
  {
    path: '/nfs/:id',
    name: 'NFSShareView',
    component: () => import('../views/share/NFSShareView.vue'),
    meta: { title: '文件分享', icon: 'FolderOpened', hideSidebar: true }
  },
  {
    path: '/wall/:id',
    name: 'PhotoWallShareView',
    component: () => import('../views/share/PhotoWallShareView.vue'),
    meta: { title: '照片墙分享', icon: 'PictureFilled', hideSidebar: true }
  },
  {
    path: '/minimax/share/:id',
    name: 'MiniMaxResultShare',
    component: () => import('../views/share/MiniMaxResultShareView.vue'),
    meta: { title: 'MiniMax 分享', icon: 'MagicStick', hideSidebar: true }
  }
]

// 首页路由
const homeRoute = {
  path: '/',
  name: 'Home',
  component: () => import('../views/home/HomeView.vue'),
  meta: { title: '首页', icon: 'Home', category: 'home' }
}

// 分享跳转路由
const shareRoutes = [
  {
    path: '/s/:id',
    name: 'ShortUrlRedirect',
    component: () => import('../views/home/HomeView.vue'),
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
