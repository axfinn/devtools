import { chromium } from 'playwright'
const browser = await chromium.launch({ headless: true, args: ['--use-gl=swiftshader', '--enable-webgl', '--ignore-gpu-blocklist'] })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const errors = []
page.on('pageerror', e => errors.push('[PE] ' + e.message.substring(0, 200)))
page.on('console', m => { if (m.type()==='error') errors.push('[CE] ' + m.text().substring(0, 200)) })

await page.goto('http://localhost:5173/dev/avatar?t=' + Date.now(), { waitUntil: 'networkidle' })
await page.waitForTimeout(2500)

// 用 evaluate 触发点击(避免 canvas 拦截)
const results = await page.evaluate(async () => {
  const out = []
  const cards = document.querySelectorAll('.preset-card')
  out.push(`找到 ${cards.length} 个 preset 卡片`)
  for (let i = 0; i < cards.length; i++) {
    cards[i].dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await new Promise(r => setTimeout(r, 500))
    out.push(`点击 [${i}] "${cards[i].textContent.trim()}"`)
  }
  await new Promise(r => setTimeout(r, 600))
  // 看当前 active
  const active = document.querySelector('.preset-card.active')?.textContent?.trim()
  out.push(`当前 active: "${active}"`)
  return out
})
for (const r of results) console.log(' •', r)
console.log('错误:', errors.length)
for (const e of errors) console.log(' •', e.substring(0, 150))
await browser.close()
