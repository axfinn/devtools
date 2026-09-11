// scripts/shoot-avatar.mjs —— 截图 Avatar 工具当前 UI
import { chromium } from 'playwright'
import { writeFile } from 'node:fs/promises'
const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1 })
const page = await ctx.newPage()
page.on('console', m => console.log(`[${m.type()}]`, m.text().substring(0, 200)))
page.on('pageerror', e => console.log('[pageerror]', e.message))
await page.goto('http://localhost:5173/dev/avatar', { waitUntil: 'networkidle' })
await page.waitForTimeout(2000)
console.log('=== 截图 1440x900 ===')
await page.screenshot({ path: '/tmp/avatar-1440.png', fullPage: false })
console.log('=== 全屏 ===')
await page.screenshot({ path: '/tmp/avatar-full.png', fullPage: true })

// 试不同分辨率
await page.setViewportSize({ width: 1920, height: 1080 })
await page.waitForTimeout(800)
await page.screenshot({ path: '/tmp/avatar-1920.png', fullPage: false })

await page.setViewportSize({ width: 1280, height: 800 })
await page.waitForTimeout(800)
await page.screenshot({ path: '/tmp/avatar-1280.png', fullPage: false })

await browser.close()
console.log('done')