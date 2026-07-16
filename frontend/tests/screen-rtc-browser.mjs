// 浏览器端 E2E 测试:复现真实用户流程,验证 viewer 能拿到画面
//
// 流程:
//   1. 用 askit 的 request-code + login-code 流程注册一个测试用户
//   2. host 页面登录 → 创建会话
//   3. viewer 页面加入
//   4. host 触发 getDisplayMedia (用 chrome fake-ui)
//   5. 验证 viewer.video 拿到 track
//
// 用法:
//   node tests/screen-rtc-browser.mjs

import { chromium } from 'playwright'

const BASE = 'http://localhost:5173'
const TIMEOUT = 30000

function log(...a) { console.log('[TEST]', ...a) }

async function loginViaAskit(page, email) {
  await page.goto(`${BASE}/screen`, { waitUntil: 'networkidle' })
  await page.waitForSelector('input[placeholder="you@example.com"]', { timeout: 10000 })

  // 填邮箱
  await page.fill('input[placeholder="you@example.com"]', email)
  await page.click('button:has-text("发送验证码")')

  // 等验证码生成:从后端 sqlite 直接读太麻烦,改为通过 inviteCode 注册
  // 先看后端是否需要 inviteCode
  const needInvite = await page.locator('input[placeholder*="邀请码"]').isVisible().catch(() => false)
  if (needInvite) {
    // 注入一个 invite code — 后端开发模式可能没开启注册
    log('need invite code, skipping')
    return false
  }

  // 等验证码显示
  await page.waitForTimeout(500)

  // 直接到后端 sqlite 拿验证码 (开发模式邮件未配置,验证码应在响应里或日志里)
  // 简化:从页面 toast 看
  return true
}

async function run() {
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'],
  })

  try {
    // ── HOST ──
    const hostCtx = await browser.newContext({ permissions: ['camera', 'microphone'] })
    const hostPage = await hostCtx.newPage()
    hostPage.on('console', (msg) => log('HOST', msg.type(), msg.text().slice(0, 200)))
    hostPage.on('pageerror', (e) => log('HOST PAGEERR', e.message))

    log('host go to share page')
    await hostPage.goto(`${BASE}/screen`)
    await hostPage.waitForTimeout(1000)

    // 检查是否需要登录
    const loginVisible = await hostPage.locator('input[placeholder="you@example.com"]').isVisible().catch(() => false)
    if (loginVisible) {
      log('login required, attempting...')
      // registration_mode 是 invite,需要先生成邀请码
      const adminPw = '123654789'
      const invRes = await fetch(`${BASE}/api/askit/v1/admin/invites`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Admin-Password': adminPw },
        body: JSON.stringify({ count: 1, expires_days: 7 }),
      })
      const invData = await invRes.json()
      log('invite result:', invData)
      if (!invData.codes?.[0]) { log('FAILED: no invite code'); process.exit(2) }
      const inviteCode = invData.codes[0]

      const ts = Date.now()
      const email = `test-${ts}@example.com`
      const r1 = await fetch(`${BASE}/api/askit/v1/auth/request-code`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, inviteCode }),
      })
      const d1 = await r1.json()
      log('request-code:', d1)
      // 验证码不在响应里(邮件已发),从 sqlite 读 — docker backend 的 db 路径在容器里
      // 简化:从 docker cp 出来的 /tmp/paste-docker.db 读
      const { execSync } = await import('child_process')
      // docker 容器名可能变,从 compose 文件读
      const container = execSync("docker ps --filter ancestor=devtools-devtools --format '{{.ID}}' | head -1").toString().trim()
      if (!container) { log('FAILED: no devtools container'); process.exit(2) }
      execSync(`docker cp ${container}:/app/data/paste.db /tmp/paste-test.db`).toString()
      const codeRes = execSync(`sqlite3 /tmp/paste-test.db "SELECT code FROM askit_email_codes WHERE email='${email}' ORDER BY expires_at DESC LIMIT 1"`).toString().trim()
      log('got code from db:', codeRes)
      if (!codeRes) { log('FAILED: no code in db for', email); process.exit(2) }
      const r2 = await fetch(`${BASE}/api/askit/v1/auth/login-code`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, code: codeRes, inviteCode }),
      })
      const d2 = await r2.json()
      log('login-code: accessToken len=', d2.accessToken?.length || 0)
      if (!d2.accessToken) { log('login failed', d2); process.exit(2) }

      // 把 token 注入 localStorage
      await hostPage.evaluate((t) => localStorage.setItem('askit_screen_access', t), d2.accessToken)
      if (d2.refreshToken) {
        await hostPage.evaluate((t) => localStorage.setItem('askit_screen_refresh', t), d2.refreshToken)
      }
      await hostPage.reload()
      await hostPage.waitForTimeout(1000)
    }

    // 创建会话
    log('creating session')
    await hostPage.click('button:has-text("创建并准备共享")')
    await hostPage.waitForTimeout(2000)

    // 抓 viewerUrl
    let viewerUrl = await hostPage.locator('input[readonly]').first().inputValue().catch(() => '')
    log('viewerUrl (raw):', viewerUrl)
    // viewerUrl 是后端返回的完整 URL(host:port),改成 vite 的 host:port 让 viewer 也走 vite dev server,
    // 否则 viewer 加载的是 docker 里的旧 dist,跟 host 端的 vite 模块不一致
    if (viewerUrl) {
      try {
        const u = new URL(viewerUrl)
        u.host = new URL(BASE).host
        u.protocol = new URL(BASE).protocol
        viewerUrl = u.toString()
        log('viewerUrl (rewritten to vite):', viewerUrl)
      } catch (e) {
        log('URL rewrite failed:', e.message)
      }
    }
    if (!viewerUrl || !viewerUrl.includes('/screen/view/')) {
      log('FAILED: no viewerUrl')
      process.exit(2)
    }

    // ── VIEWER ──
    const viewerCtx = await browser.newContext()
    const viewerPage = await viewerCtx.newPage()
    viewerPage.on('console', (msg) => log('VIEWER', msg.type(), msg.text().slice(0, 200)))
    viewerPage.on('pageerror', (e) => log('VIEWER PAGEERR', e.message))

    log('viewer go to', viewerUrl)
    await viewerPage.goto(viewerUrl)
    await viewerPage.waitForTimeout(1000)

    // 加入
    const joinBtn = viewerPage.locator('button:has-text("加入观看")')
    if (await joinBtn.isVisible().catch(() => false)) {
      await joinBtn.click()
      log('viewer clicked join')
    }
    await viewerPage.waitForTimeout(1500)

    // ── HOST 开始共享 ──
    log('host clicks 开始共享')
    const startBtn = hostPage.locator('button:has-text("开始共享")')
    if (await startBtn.isVisible().catch(() => false)) {
      await startBtn.click()
      log('host clicked start share')
    } else {
      log('host share button NOT visible — capture already started?')
    }

    // 等视频
    log('waiting for viewer to receive stream...')
    const viewerGot = await viewerPage.evaluate(async () => {
      return new Promise((resolve) => {
        const v = document.querySelector('video')
        if (!v) return resolve('no-video-el')
        const start = Date.now()
        const t = setInterval(() => {
          if (v.srcObject && v.srcObject.getVideoTracks().length > 0) {
            const tracks = v.srcObject.getVideoTracks()
            const settings = tracks[0].getSettings()
            clearInterval(t)
            resolve({ ok: true, width: settings.width, height: settings.height, ms: Date.now() - start })
          } else if (Date.now() - start > 25000) {
            clearInterval(t)
            resolve('timeout')
          }
        }, 200)
      })
    })
    log('viewer result:', JSON.stringify(viewerGot))

    if (typeof viewerGot === 'object' && viewerGot.ok) {
      log('✅ TEST PASSED — viewer received video:', viewerGot.width + 'x' + viewerGot.height, 'after', viewerGot.ms + 'ms')
      process.exit(0)
    } else {
      log('❌ TEST FAILED — viewer did not get video:', viewerGot)
      process.exit(1)
    }
  } finally {
    await browser.close()
  }
}

run().catch(e => { console.error('FATAL', e); process.exit(2) })
