import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), react()],
  // 支持域名部署，base 可以通过环境变量配置
  base: process.env.VITE_BASE_URL || '/',
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  // 优化依赖预构建
  optimizeDeps: {
    include: ['react', 'react-dom', '@excalidraw/excalidraw'],
    exclude: []
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    // 代码分割配置
    rollupOptions: {
      output: {
        // 手动分包策略
        manualChunks: {
          // Vue 核心
          'vue-vendor': ['vue', 'vue-router'],
          // Element Plus UI 库
          'element-plus': ['element-plus'],
          // React 核心（Excalidraw 依赖）
          'react-vendor': ['react', 'react-dom'],
          // Excalidraw 画图库（单独分包）
          'excalidraw': ['@excalidraw/excalidraw'],
          // Mermaid 图表库（按需加载）
          'mermaid': ['mermaid'],
          // KaTeX 数学公式
          'katex': ['katex', 'markdown-it-texmath'],
          // Markdown 解析
          'markdown': ['markdown-it'],
          // 代码高亮
          'highlight': ['highlight.js'],
          // 二维码
          'qrcode': ['qrcode'],
          // diff 库
          'diff': ['diff']
        },
        // 文件命名
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]'
      }
    },
    // 压缩选项
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true, // 移除 console
        drop_debugger: true
      }
    },
    // 关闭 chunk 大小警告（因为 Mermaid 等库本身较大）
    chunkSizeWarningLimit: 1000
  }
})
