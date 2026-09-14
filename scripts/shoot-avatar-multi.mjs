import { chromium } from 'playwright'
const browser = await chromium.launch({ headless: true })
for (const [w, h, label] of [[1440, 900, '1440'], [1280, 800, '1280'], [1920, 1080, '1920']]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h } })
  const page = await ctx.newPage()
  await page.goto('http://localhost:5173/dev/avatar?t=' + Date.now(), { waitUntil: 'networkidle' })
  await page.waitForTimeout(1500)
  await page.screenshot({ path: `/tmp/avatar-${label}.png`, fullPage: false })
  await ctx.close()
}
await browser.close()
