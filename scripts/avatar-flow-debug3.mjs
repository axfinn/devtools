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

// 用 evaluate 直接调函数(走 vue 事件链)
console.log('=== 直接 evaluate 触发 emit ===')

// 先展开
await page.click('.clip-block', { force: true })
await page.waitForTimeout(500)

const results = await page.evaluate(async () => {
  const logs = []
  // 拿 ClipCard 按钮位置 + 通过原生 click 触发
  const btns = document.querySelectorAll('.card-actions .el-button')
  logs.push(`找到 ${btns.length} 个按钮`)
  if (btns.length < 2) return logs

  // 看按钮 icon 是 VideoPlay / Share
  btns.forEach((b, i) => {
    const cls = b.querySelector('svg')?.getAttribute('class') || ''
    logs.push(`btn[${i}] icon-class=${cls.substring(0, 80)}`)
  })

  // 触发播放按钮原生 click
  btns[0].dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
  await new Promise(r => setTimeout(r, 600))
  logs.push('点击了 btn[0]')

  btns[1].dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
  await new Promise(r => setTimeout(r, 1800))
  logs.push('点击了 btn[1]')

  return logs
})

for (const l of results) console.log('  •', l)

const messages = await page.$$eval('.el-message', els => els.map(e => e.textContent.trim()))
console.log('\n=== toast ===')
for (const m of messages) console.log(' •', m)

console.log('\n=== API ===')
for (const c of apiCalls) console.log(`  ${c.s} ${c.m} ${c.u}`)
console.log('\n=== 错误:', errors.length, '===')
for (const e of errors) console.log(e)

await browser.close()
