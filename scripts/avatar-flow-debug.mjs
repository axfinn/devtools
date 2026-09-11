import { chromium } from 'playwright'
const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()

const apiCalls = []
const errors = []
page.on('console', m => {
  if (m.type() === 'error') errors.push('[CONSOLE] ' + m.text().substring(0, 400))
})
page.on('pageerror', e => errors.push('[PAGEERROR] ' + e.message.substring(0, 400)))
page.on('response', r => {
  const u = r.url()
  if (u.includes('/api/avatar/')) {
    apiCalls.push({ status: r.status(), method: r.request().method(), url: u.substring(u.indexOf('/api/avatar/')) })
  }
})

await page.goto('http://localhost:5173/dev/avatar?t=' + Date.now(), { waitUntil: 'networkidle' })
await page.waitForTimeout(2000)

// 1. 点录制
console.log('=== 1. 点录制 → 停 ===')
await page.click('button:has-text("录制")')
await page.waitForTimeout(800)
await page.click('button:has-text("录制"), button:has-text("停止录制")')
await page.waitForTimeout(2500)

// 2. 看 clip 状态 + 后端 sync
const clipState = await page.$$eval('.clip-block', els => els.map(e => ({ text: e.textContent.trim(), id: e.dataset.clipId || null })))
console.log('clip 数:', clipState.length, JSON.stringify(clipState))

// 3. 拿所有 .card-actions 按钮并强制点击(跳过遮挡)
console.log('\n=== 3. 拿 clip-card 按钮 + 强制点击 ===')
const btns = await page.$$('.card-actions .el-button')
console.log('按钮数:', btns.length)
if (btns.length >= 1) {
  await btns[0].click({ force: true })
  await page.waitForTimeout(1000)
  console.log('✓ 已点播放(force)')
}
if (btns.length >= 2) {
  await btns[1].click({ force: true })
  await page.waitForTimeout(2500)
  console.log('✓ 已点分享(force)')
}

// 4. 看 toast
const messages = await page.$$eval('.el-message', els => els.map(e => e.textContent.trim()))
console.log('\n=== 4. toast 消息 ===')
for (const m of messages) console.log(' •', m)

// 5. 最终 clip 状态
const finalClips = await page.$$eval('.clip-block', els => els.map(e => e.textContent.trim()))
console.log('\n=== 5. 最终 clip 数:', finalClips.length, '===')

console.log('\n=== /api/avatar/* 调用 ===')
for (const c of apiCalls) console.log(`  ${c.status} ${c.method} ${c.url}`)
console.log('\n=== 错误数:', errors.length, '===')
for (const e of errors) console.log(e)

await browser.close()
