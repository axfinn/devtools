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
await page.waitForTimeout(3000)

// 直接通过 Vue 内部暴露的 __vueParentComponent / __v_isRef 调 onClipPlay / onClipShare
console.log('=== 直接 evaluate 调父组件方法 ===')
const r = await page.evaluate(async () => {
  const out = []
  // 找 Timeline 组件实例,emit rec-toggle
  const root = document.querySelector('#app')?.__vue_app__
  if (!root) return ['no vue app']

  // 找 Timeline 组件
  function find(node, name) {
    if (!node) return null
    if (node.type?.__name === name || node.type?.name === name) return node
    if (node.subTree) {
      const r = findByTree(node.subTree, name)
      if (r) return r
    }
    return null
  }
  function findByTree(tree, name) {
    if (!tree) return null
    if (tree.type?.__name === name || tree.type?.name === name) return tree.component
    if (Array.isArray(tree.children)) {
      for (const c of tree.children) {
        const r = findByTree(c, name)
        if (r) return r
      }
    }
    return null
  }

  // 直接用 el-button 拿 Vue 组件引用
  const recBtn = [...document.querySelectorAll('button')].find(b => /录制|停止录制/.test(b.textContent))
  if (!recBtn) return ['no rec button']
  out.push(`找到录制按钮 text="${recBtn.textContent.trim()}"`)

  // 点击切换录制状态
  recBtn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
  await new Promise(r => setTimeout(r, 600))
  out.push('点了一次')

  // 等 1 秒后再点停止
  await new Promise(r => setTimeout(r, 600))
  const recBtn2 = [...document.querySelectorAll('button')].find(b => /录制|停止录制/.test(b.textContent))
  if (recBtn2) {
    recBtn2.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    out.push('点了停止')
  }
  await new Promise(r => setTimeout(r, 2500))

  // 看 clip-block 数
  const blocks = document.querySelectorAll('.clip-block')
  out.push(`clip-block 数: ${blocks.length}`)

  if (!blocks.length) return out

  // 展开第一个
  blocks[0].dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
  await new Promise(r => setTimeout(r, 600))

  // 找 ClipCard 按钮
  const cardBtns = document.querySelectorAll('.card-actions .el-button')
  out.push(`clip-card 按钮数: ${cardBtns.length}`)

  if (cardBtns.length >= 1) {
    cardBtns[0].dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await new Promise(r => setTimeout(r, 1200))
    out.push('点了播放')
  }
  if (cardBtns.length >= 2) {
    cardBtns[1].dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await new Promise(r => setTimeout(r, 2500))
    out.push('点了分享')
  }

  return out
})

for (const l of r) console.log(' •', l)

const messages = await page.$$eval('.el-message', els => els.map(e => e.textContent.trim()))
console.log('\n=== toast ===')
for (const m of messages) console.log(' •', m)

console.log('\n=== API ===')
for (const c of apiCalls) console.log(`  ${c.s} ${c.m} ${c.u}`)
console.log('\n=== 错误:', errors.length, '===')
for (const e of errors) console.log(e)

await browser.close()
