import { chromium } from 'playwright'
const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()

const errors = []
const consoleMsgs = []
const apiCalls = []

page.on('console', m => {
  const text = m.text()
  consoleMsgs.push({ type: m.type(), text: text.substring(0, 300) })
  if (m.type() === 'error') errors.push('[CONSOLE] ' + text.substring(0, 500))
})
page.on('pageerror', e => errors.push('[PAGEERROR] ' + e.message.substring(0, 500)))
page.on('requestfailed', r => {
  errors.push(`[REQFAIL] ${r.method()} ${r.url().substring(0, 200)} - ${r.failure()?.errorText || 'unknown'}`)
})
page.on('response', r => {
  const u = r.url()
  if (u.includes('/api/avatar/')) {
    apiCalls.push({ status: r.status(), method: r.request().method(), url: u.substring(u.indexOf('/api/avatar/')) })
  }
})

await page.goto('http://localhost:5173/dev/avatar?t=' + Date.now(), { waitUntil: 'networkidle' })
await page.waitForTimeout(2000)

// 找录制按钮并点击
console.log('--- 找录制按钮 ---')
const recBtn = await page.$('button:has-text("录制")')
if (!recBtn) {
  console.log('找不到录制按钮,看看页面上的按钮:')
  const allBtns = await page.$$eval('button', els => els.map(e => e.textContent.trim()).filter(t => t))
  console.log('按钮列表:', allBtns)
} else {
  console.log('找到录制按钮,点击开始录制...')
  await recBtn.click()
  await page.waitForTimeout(2000)
  
  // 再点一次停止
  const stopBtn = await page.$('button:has-text("停止"), button:has-text("停止录制")')
  if (stopBtn) {
    console.log('点击停止录制...')
    await stopBtn.click()
    await page.waitForTimeout(1500)
  } else {
    console.log('录制按钮还在,可能没切换状态')
  }
}

// 看 timeline 上是否出现片段
const segments = await page.$$eval('.tl-segment, [data-clip-id], .timeline-clip, .clip-row', els => els.length).catch(() => 0)
console.log('timeline 上 segments 数:', segments)

// 找播放/分享按钮
const playBtn = await page.$('button:has-text("播放")')
const shareBtn = await page.$('button:has-text("分享")')
console.log('播放按钮:', !!playBtn, '分享按钮:', !!shareBtn)

if (playBtn) {
  console.log('点击播放...')
  await playBtn.click()
  await page.waitForTimeout(1000)
}
if (shareBtn) {
  console.log('点击分享...')
  await shareBtn.click()
  await page.waitForTimeout(1500)
}

// 列出所有 avatar API 调用
console.log('\n--- /api/avatar/* 调用 ---')
for (const c of apiCalls) {
  console.log(`${c.status} ${c.method} ${c.url}`)
}

console.log('\n--- 错误数:', errors.length, '---')
for (const e of errors) console.log(e)

console.log('\n--- console error 全文 ---')
for (const m of consoleMsgs.filter(x => x.type === 'error')) console.log(`[${m.type}]`, m.text)

await page.screenshot({ path: '/tmp/avatar-debug.png' })
await browser.close()
