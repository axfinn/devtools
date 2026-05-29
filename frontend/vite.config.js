import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src/neon'),
      '@neon': path.resolve(__dirname, './src/neon')
    }
  },
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
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }

          const groups = [
            { name: 'vue-vendor', patterns: ['node_modules/vue/', 'node_modules/vue-router/'] },
            { name: 'react-vendor', patterns: ['node_modules/react/', 'node_modules/react-dom/', 'node_modules/react-router-dom/', 'node_modules/react-i18next/', 'node_modules/i18next/', 'node_modules/zustand/'] },
            { name: 'element-plus', patterns: ['node_modules/element-plus/', 'node_modules/@element-plus/'] },
            { name: 'katex', patterns: ['node_modules/katex/', 'node_modules/markdown-it-texmath/'] },
            { name: 'markdown', patterns: ['node_modules/markdown-it/', 'node_modules/markdown-it-footnote/', 'node_modules/markdown-it-mark/', 'node_modules/markdown-it-sub/', 'node_modules/markdown-it-sup/', 'node_modules/markdown-it-task-lists/'] },
            { name: 'highlight', patterns: ['node_modules/highlight.js/'] },
            { name: 'qrcode', patterns: ['node_modules/qrcode/', 'node_modules/html5-qrcode/'] },
            { name: 'diff', patterns: ['node_modules/diff/'] },
            { name: 'terminal-vendor', patterns: ['node_modules/@xterm/', 'node_modules/xterm/'] },
            { name: 'media-vendor', patterns: ['node_modules/artplayer/', 'node_modules/hls.js/'] },
            { name: 'ffmpeg', patterns: ['node_modules/@ffmpeg/ffmpeg/', 'node_modules/@ffmpeg/util/'] },
            { name: 'pdf-vendor', patterns: ['node_modules/vue-pdf-embed/'] },
            { name: 'zip-vendor', patterns: ['node_modules/jszip/'] },
            { name: 'crypto-vendor', patterns: ['node_modules/crypto-js/'] },
            { name: 'echarts', patterns: ['node_modules/echarts/'] },
            { name: 'three', patterns: ['node_modules/three/'] },
          ]

          for (const group of groups) {
            if (group.patterns.some(pattern => id.includes(pattern))) {
              return group.name
            }
          }
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
