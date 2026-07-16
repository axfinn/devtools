// 走生产路径:viewerUrl 不重写,直接用 http://localhost:8082
// 模拟用户在 docker 部署环境下访问(不经过 vite dev server)
import { chromium } from 'playwright'
import { execSync } from 'child_process'

const BASE = 'http://localhost:8082'
const TIMEOUT = 30000

function log(...a) { console.log('[TEST]', ...a) }

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

    const loginVisible = await hostPage.locator('input[placeholder="you@example.com"]').isVisible().catch(() => false)
    if (loginVisible) {
      log('login required, attempting...')
      const adminPw = '123654789'
      const invRes = await fetch(`${BASE}/api/askit/v1/admin/invites`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Admin-Password': adminPw },
        body: JSON.stringify({ count: 1, expires_days: 7 }),
      })
      const invData = await invRes.json()
      const inviteCode = invData.codes[0]
      log('invite:', inviteCode)

      const ts = Date.now()
      const email = `prod-${ts}@example.com`
      await fetch(`${BASE}/api/askit/v1/auth/request-code`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, inviteCode }),
      })
      const container = execSync("docker ps --filter ancestor=devtools-devtools --format '{{.ID}}' | head -1").toString().trim()
      execSync(`docker cp ${container}:/app/data/paste.db /tmp/prod-test.db`)
      const codeRes = execSync(`sqlite3 /tmp/prod-test.db "SELECT code FROM askit_email_codes WHERE email='${email}' ORDER BY expires_at DESC LIMIT 1"`).toString().trim()
      log('code:', codeRes)
      const r2 = await fetch(`${BASE}/api/askit/v1/auth/login-code`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, code: codeRes, inviteCode }),
      })
      const d2 = await r2.json()
      log('accessToken len:', d2.accessToken?.length)
      await hostPage.evaluate((t) => localStorage.setItem('askit_screen_access', t), d2.accessToken)
      if (d2.refreshToken) {
        await hostPage.evaluate((t) => localStorage.setItem('askit_screen_refresh', t), d2.refreshToken)
      }
      await hostPage.reload()
      await hostPage.waitForTimeout(1000)
    }

    log('creating session')
    await hostPage.click('button:has-text("创建并准备共享")')
    await hostPage.waitForTimeout(2000)

    const viewerUrl = await hostPage.locator('input[readonly]').first().inputValue().catch(() => '')
    log('viewerUrl (production, NOT rewritten):', viewerUrl)
    if (!viewerUrl || !viewerUrl.includes('/screen/view/')) {
      log('FAILED: no viewerUrl'); process.exit(2)
    }

    // ── VIEWER ──
    const viewerCtx = await browser.newContext()
    const viewerPage = await viewerCtx.newPage()
    viewerPage.on('console', (msg) => log('VIEWER', msg.type(), msg.text().slice(0, 200)))
    viewerPage.on('pageerror', (e) => log('VIEWER PAGEERR', e.message))

    log('viewer go to', viewerUrl)
    await viewerPage.goto(viewerUrl)
    await viewerPage.waitForTimeout(1000)

    const joinBtn = viewerPage.locator('button:has-text("加入观看")')
    if (await joinBtn.isVisible().catch(() => false)) {
      await joinBtn.click()
      log('viewer clicked join')
    }
    await viewerPage.waitForTimeout(1500)

    log('host clicks 开始共享')
    const startBtn = hostPage.locator('button:has-text("开始共享")')
    if (await startBtn.isVisible().catch(() => false)) {
      await startBtn.click()
    }

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
      log('✅ TEST PASSED (production path) — viewer received video:', viewerGot.width + 'x' + viewerGot.height, 'after', viewerGot.ms + 'ms')
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
