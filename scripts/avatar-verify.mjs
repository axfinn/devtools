// 硬刷新 + 截图当前 UI 真实状态
import { chromium } from 'playwright'
const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 1,
})
const page = await ctx.newPage()
page.on('console', m => { if (m.type()==='error') console.log('[err]', m.text().substring(0,200)) })
await ctx.route('**/*', (route) => {
  const h = route.request().headers()
  h['cache-control'] = 'no-cache'
  route.continue({ headers: h })
})
await page.goto('http://localhost:5173/dev/avatar?t=' + Date.now(), { waitUntil: 'networkidle' })
await page.waitForTimeout(2500)
const layout = await page.evaluate(() => {
  const body = document.querySelector('.tool-body')
  const left = document.querySelector('.tool-left')
  const main = document.querySelector('.tool-main')
  const inspector = document.querySelector('.inspector')
  const rail = document.querySelector('.tool-rail')
  const railPanel = document.querySelector('.rail-panel')
  const stage = document.querySelector('.stage-wrap, .stage-main')
  const f = (el) => el ? el.getBoundingClientRect().width : null
  return {
    bodyWidth: f(body),
    bodyCS: body && getComputedStyle(body).gridTemplateColumns,
    left: f(left), leftStart: left?.offsetLeft, leftCS: left && getComputedStyle(left).gridColumn,
    main: f(main), mainStart: main?.offsetLeft,
    inspector: f(inspector), inspectorStart: inspector?.offsetLeft,
    rail: f(rail), railPanel: f(railPanel),
    stage: f(stage),
  }
})
console.log('LAYOUT:', JSON.stringify(layout, null, 2))
await page.screenshot({ path: '/tmp/avatar-now.png', fullPage: false })
await browser.close()