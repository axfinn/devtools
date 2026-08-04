import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// 最小配置:让 vue 单文件组件可以被测试组件时正常 import;
// 纯函数测试本身用不到 environment,但保留 jsdom 以便后续 component 测试。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src/neon'),
      '@neon': path.resolve(__dirname, './src/neon'),
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.js', 'src/**/*.spec.js'],
    // 纯函数测试跑得很快,不要因为 http 请求 / timer 之类的事卡住
    testTimeout: 5000,
  },
})
