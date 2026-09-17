import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import react from '@vitejs/plugin-react'
import path from 'path'
// FINN-29 A2 · Element Plus 按需引入
import Components from 'unplugin-vue-components/vite'
import AutoImport from 'unplugin-auto-import/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// simple-peer 链路里 readable-stream 等 CJS 模块用 process.nextTick / process.stdout
// 这些在浏览器不存在,需要替换成 shim。dev 时 vite 的 optimizeDeps 通过 esbuild 预构建
// 处理一部分,但 production build 走 Rollup 不做这个替换,直接 ReferenceError。
//
// 只 shim 三个 process.X(其他 process.env 之类由 vite 的 define 处理),
// 并跳过 'buffer' / 'events' / 'util' 模块自身(否则会跟自己产生循环 import)。
const processShim = () => ({
  name: 'simple-peer-process-shim',
  enforce: 'pre',
  transform(code, id) {
    // 只处理 simple-peer 的依赖链(原始 node_modules 路径)
    // 注意要排除 buffer / events / util 自身,否则会产生循环 import / 双重声明。
    // dev 模式下 vite 把 simple-peer 的所有依赖打进 .vite/deps/simple-peer.js,
    // 这种大文件路径里没有模块名,但包含 `__commonJS(...)` 的 esbuild CJS wrapper 特征 —
    // 用这个判断要不要 shim。
    const isSimplePeerDep = /node_modules[\\/](?:simple-peer|readable-stream|randombytes|safe-buffer|debug|queue-microtask|err-code|get-browser-rtc|readable-web-to-node-stream|stream-browserify)/.test(id)
    const isVitePreBundle = /node_modules[\\/].vite[\\/]deps[\\/]/.test(id)
                          && (/__commonJS\(/.test(code) || /require_(safe_buffer|randombytes|simple_peer|readable_stream|debug)\b/.test(code))
    if (!isSimplePeerDep && !isVitePreBundle) return
    if (!/\bprocess\.(nextTick|stderr|stdout)\b/.test(code)) return
    return {
      code: 'var __processShim_nextTick = function (fn) { var args = []; for (var i = 1; i < arguments.length; i++) args.push(arguments[i]); setTimeout(function () { fn.apply(null, args); }, 0); };\n'
        + 'var __processShim_stderr = { write: function () {} };\n'
        + 'var __processShim_stdout = { write: function () {} };\n'
        + code.replace(/\bprocess\.nextTick\b/g, '__processShim_nextTick')
          .replace(/\bprocess\.stderr\b/g, '__processShim_stderr')
          .replace(/\bprocess\.stdout\b/g, '__processShim_stdout'),
      map: null,
    }
  },
})

// FINN-29 A9 · 把 index-*.css 改成异步加载的钩子
// Vite 默认会发出 <link rel="stylesheet" href="/assets/index-*.css">，
// 同步 CSS 阻塞首屏渲染。这里把所有指向 /assets/*.css 的 <link> 改写为
// 媒体查询为 print 的非阻塞形式，加载完切到 all，避免 FOUC（关键 CSS 已
// 在 index.html 内联，async CSS 只是剩下的工具/视图样式）。
const asyncIndexCssPlugin = () => ({
  name: 'finn-29-async-index-css',
  enforce: 'post',
  transformIndexHtml(html) {
    // Vite 实际产出的 link 形如:
    //   <link rel="stylesheet" crossorigin href="/assets/index-XXXX.css">
    // 我们把所有指向 /assets/*.css 的 link 改成 media=print 的非阻塞形式,
    // onload 后切回 all —— 不阻塞首屏渲染,关键 CSS 已内联进 index.html。
    return html.replace(
      /<link([^>]*?)href="(\/assets\/[^"]+\.css)"([^>]*)>/g,
      (_, before, href, after) => {
        // 已经手工处理过的不重复;只处理首次出现的。
        if (/media=/.test(before) || /media=/.test(after)) {
          return `<link${before}href="${href}"${after}>`
        }
        return `<link${before}href="${href}"${after} media="print" onload="this.media='all'">`
      }
    )
  },
})

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue({
      template: {
        compilerOptions: {
          // pet-widget 走 web component，不当 Vue 组件解析
          isCustomElement: (tag) => tag === 'pet-widget',
        },
      },
    }),
    react(),
    processShim(),
    // FINN-29 A2 · EP 按需 — 仅当组件真的被模板使用时才 import 它的 CSS/JS
    AutoImport({
      resolvers: [ElementPlusResolver({ importStyle: 'css' })],
    }),
    Components({
      resolvers: [ElementPlusResolver({ importStyle: 'css' })],
    }),
    asyncIndexCssPlugin(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src/neon'),
      '@neon': path.resolve(__dirname, './src/neon'),
      // simple-peer 等 browserify 时代模块假设 Node 全局(process / global / Buffer)在浏览器存在。
      // 我们的方案:useScreenRTC.js 顶部用 `import { Buffer } from 'buffer'` + `globalThis.Buffer = Buffer` 把
      // Buffer 挂到全局。process / global 由 esbuild 的 define 默认替换,或者 simple-peer 的浏览器分支
      // 不调用它们(dev / prod 都 OK)。
      buffer: path.resolve(__dirname, './src/polyfills/buffer.js'),
      util: path.resolve(__dirname, './src/polyfills/util.js'),
      events: path.resolve(__dirname, 'node_modules/events/events.js'),
    }
  },
  // 支持域名部署，base 可以通过环境变量配置
  base: process.env.VITE_BASE_URL || '/',
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        // vite 7 默认不代理 WebSocket upgrade — 必须显式 ws: true,
        // 否则屏幕共享等需要 WS 的功能会卡住(浏览器永远等不到握手完成)
        ws: true,
        // 把用户原始访问的 host 透传给后端。后端用它生成 viewer_url,
        // 否则 LAN IP / 隧道访问时,viewer_url 会被改写成 localhost:8082,
        // 用户点链接就会跳到不可达的地址。
        bypass(req) {
          // http-proxy 不直接支持改 header,得通过 configure 钩子
          return undefined
        },
        configure(proxy) {
          proxy.on('proxyReq', (proxyReq, req) => {
            const fwdHost = req.headers.host
            if (fwdHost) proxyReq.setHeader('X-Forwarded-Host', fwdHost)
          })
        },
      }
    }
  },
  // 优化依赖预构建(dev 用)
  optimizeDeps: {
    include: [
      'react', 'react-dom', '@excalidraw/excalidraw',
      'simple-peer',
      'buffer',
      'safe-buffer',
      'randombytes',
      'debug',
    ],
  },
  define: {
    // simple-peer 的 browserify 分支会检查 process.browser。我们强制为 false,
    // 让它走真实 browser 分支(esbuild 默认不替换 process.X,这里显式处理最常见的几个)
    'process.browser': 'false',
    'process.env': '{}',
    'global': 'globalThis',
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    // simple-peer 是 CJS (`module.exports = Peer`),需要 vite 的 @rollup/plugin-commonjs
    // 把它包成 ESM 默认导出。dev 时通过 optimizeDeps 走 esbuild 预构建,
    // production build 直接走 Rollup 必须显式启用 commonjsOptions.include。
    commonjsOptions: {
      include: [/node_modules/],
      transformMixedEsModules: true,
    },
    // 代码分割配置
    rollupOptions: {
      output: {
        // 手动分包策略
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }

          // FINN-29 A2 · EP 按需后不再单独把 element-plus 打成一个 chunk
          // ——各 view 的懒加载 chunk 自己 import 自己用到的 EP 组件即可。
          const groups = [
            { name: 'vue-vendor', patterns: ['node_modules/vue/', 'node_modules/vue-router/'] },
            { name: 'react-vendor', patterns: ['node_modules/react/', 'node_modules/react-dom/', 'node_modules/react-router-dom/', 'node_modules/react-i18next/', 'node_modules/i18next/', 'node_modules/zustand/'] },
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
        drop_debugger: true
        // 注意: 不设 drop_console — simple-peer 等库可能在异常路径 console.error,
        // 完全去掉会让 prod 调试变得完全黑盒(我们这次 debug 时就是因此看不到任何日志)。
        // drop_console: true 会让我们看不到 user.error 时的浏览器栈
      },
      // 保留模板里用到的函数名,避免 <script setup> 模板引用断裂导致白屏
      keep_fnames: true,
      mangle: {
        // 不混淆 setup 暴露给模板的函数名
        reserved: [
          'exportAllDataJSON',
          'exportTasksCSV',
          'openGlobalQuickAdd',
          'closeGlobalQuickAdd',
          'submitGlobalQuickAdd',
          'onGlobalQuickAddInput',
          'openVoice',
          'startVoiceCapture',
          'stopVoiceCapture',
          'cancelVoice'
        ],
        // 不混淆 <script setup> 顶层 let/const 标识符,避免 mangler 把
        // 同一个名字的实例方法和本地变量搞混(出现过 host 端
        // relayUpload.start(...) 被改成 relayDownload.start(...) 的 bug)。
        // 仅保留变量名,体积代价 ~2-5KB,可忽略。
        properties: false
      }
    },
    // 关闭 chunk 大小警告（因为 Mermaid 等库本身较大）
    chunkSizeWarningLimit: 1000
  }
})