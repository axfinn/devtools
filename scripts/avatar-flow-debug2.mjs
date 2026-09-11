import { chromium } from 'playwright'
const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()

const apiCalls = []
const errors = []
page.on('console', m => { if (m.type() === 'error') errors.push('[CONSOLE] ' + m.text().substring(0, 400)) })
page.on('pageerror', e => errors.push('[PAGEERROR] ' + e.message.substring(0, 400)))
page.on('response', r => {
  const u = r.url()
  if (u.includes('/api/avatar/')) apiCalls.push({ s: r.status(), m: r.request().method(), u: u.substring(u.indexOf('/api/avatar/')) })
})

await page.goto('http://localhost:5173/dev/avatar?t=' + Date.now(), { waitUntil: 'networkidle' })
await page.waitForTimeout(2000)

// 录 → 停
await page.click('button:has-text("录制")')
await page.waitForTimeout(800)
await page.click('button:has-text("录制"), button:has-text("停止录制")')
await page.waitForTimeout(2500)

// 1. 点 clip 展开
console.log('=== 1. 点 clip 展开 ===')
await page.click('.clip-block', { force: true })
await page.waitForTimeout(500)

// 2. 拿 clip-card 按钮
const btns = await page.$$('.card-actions .el-button')
console.log('clip-card 按钮数:', btns.length)

// 3. 点播放
if (btns[0]) {
  console.log('=== 2. 点播放 ===')
  await btns[0].click({ force: true })
  await page.waitForTimeout(1000)
}

// 4. 点分享
if (btns[1]) {
  console.log('=== 3. 点分享 ===')
  await btns[1].click({ force: true })
  await page.waitForTimeout(2000)
}

// toast
const messages = await page.$$eval('.el-message', els => els.map(e => e.textContent.trim()))
console.log('\n=== toast ===')
for (const m of messages) console.log(' •', m)

console.log('\n=== API 调用 ===')
for (const c of apiCalls) console.log(`  ${c.s} ${c.m} ${c.u}`)
console.log('\n=== 错误数:', errors.length, '===')
for (const e of errors) console.log(e)

await browser.close()
