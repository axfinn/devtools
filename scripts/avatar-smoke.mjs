// scripts/avatar-smoke.mjs
// ----------------------------------------------------------------
// 本地可用性 smoke —— 对应 DEVT-12 验收契约的 8 项断言:
//
//   1. 路由 /dev/avatar             HTTP 200
//   2. 控制台                       0 error (已过滤环境伪错误)
//   3. 画布                          WebGL2 上下文 + drawingBuffer + 截图非空
//   4. 三项交互
//        a) 姿态预设 → 按钮激活
//        b) 选骨骼 → 改 transform 数值
//        c) 时间轴 scrub → 时间显示变化
//   5. 主题翻转                     html.dark 类切换 + 浅/深 各截一张
//   6. 摄像头无设备降级              状态切到 no_device / unsupported / error
//      + "再试" 按钮出现
//   7. 导出                         真的下载文件 + glTF magic 'glTF' (glb)
//   8. VRM 载入 hookup              上传 .vrm 后,VirtualAvatarTool.onImported
//                                    真的调到了 vrmLoader.loadVRMIntoScene
//
// 调用约定 (与本仓 loops 流程一致):
//   假设 backend :8080 与 vite :5173 已由本轮 orchestrator 起好;
//   在 frontend/ 下执行以让 Node 解析到根的 node_modules:
//     cd frontend && node ../scripts/avatar-smoke.mjs
//   或者任何 cwd,但需要 ./frontend/node_modules 能被解析到 playwright
//   (本仓已经 symlink scripts/node_modules → ../frontend/node_modules)
//
// 产出:
//   ~/.avatar-loop/smoke-report.json
//   ~/.avatar-loop/shots/shot-light.png
//   ~/.avatar-loop/shots/shot-dark.png
//
// 退出码: 0 全绿 / 1 有红 / 2 前置失败 (vite 不可达) / 3 顶层异常
// ----------------------------------------------------------------

import { chromium } from 'playwright'
import { mkdir, writeFile, stat, readFile } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import path from 'node:path'
import os from 'node:os'

const VITE = 'http://localhost:5173'
const AVATAR = `${VITE}/dev/avatar`
const OUT_DIR = path.join(os.homedir(), '.avatar-loop')
const SHOT_DIR = path.join(OUT_DIR, 'shots')
const REPORT = path.join(OUT_DIR, 'smoke-report.json')

// 已知的环境/后端伪错误 —— 不计入 console.error
const ENV_ERROR_FILTERS = [
    /\/api\/avatar\/models/,        // 资产面板可能 404, 显示 initError 告警, 非前端回归
    /\/api\/assets/,                // 同上
    /favicon\.ico/,
    /\/__vite_ping/,
    /hot-update\.json/,
]
function isEnvError(text) { return ENV_ERROR_FILTERS.some(r => r.test(text || '')) }

function log(...a) { console.log('[SMOKE]', ...a) }

async function assert(name, fn) {
    const t0 = Date.now()
    try {
        const detail = await fn()
        return { name, pass: true, ms: Date.now() - t0, detail: detail ?? null }
    } catch (e) {
        return { name, pass: false, ms: Date.now() - t0, error: String(e?.message || e) }
    }
}

const results = []
function rec(r) { results.push(r); log((r.pass ? '✓ ' : '✗ ') + r.name + (r.error ? ' — ' + r.error : '')) }

async function main() {
    await mkdir(SHOT_DIR, { recursive: true })

    // ---- preflight: vite 不可达就尽早失败 ----
    const browser = await chromium.launch({ headless: true })
    const ctx = await browser.newContext({
        viewport: { width: 1440, height: 900 },
        acceptDownloads: true,
        locale: 'zh-CN',
    })
    const page = await ctx.newPage()
    const consoleErrors = []
    const consoleInfos = []
    page.on('console', m => {
        if (m.type() === 'error') {
            const text = m.text()
            if (!isEnvError(text)) consoleErrors.push(text)
        }
        // 收 info 级 console,VRM 断言要用
        if (m.type() === 'info') consoleInfos.push(m.text())
    })
    page.on('pageerror', e => consoleErrors.push('pageerror: ' + e.message))

    try {
        await page.goto(VITE, { waitUntil: 'domcontentloaded', timeout: 10000 })
    } catch (e) {
        await browser.close()
        const report = { ok: false, reason: 'vite 不可达 ' + VITE, results: [], consoleErrors }
        await writeFile(REPORT, JSON.stringify(report, null, 2))
        console.error('[SMOKE] vite 不可达:', e.message)
        process.exit(2)
    }

    // ===== 1. 路由 /dev/avatar HTTP 200 + 挂载证明 =====
    let resp
    rec(await assert('1. 路由 /dev/avatar 返回 200', async () => {
        resp = await page.goto(AVATAR, { waitUntil: 'networkidle', timeout: 20000 })
        if (!resp || resp.status() !== 200) throw new Error('status=' + (resp?.status?.() ?? 'null'))
        await page.waitForSelector('.topbar-title', { timeout: 10000 })
        const title = (await page.textContent('.topbar-title'))?.trim()
        if (!title || !title.includes('Virtual Avatar')) throw new Error('topbar 标题异常: ' + title)
        return { status: resp.status(), url: page.url(), title }
    }))

    // ===== 3. 画布: WebGL2 + drawingBuffer + 截图 =====
    let canvasPng
    rec(await assert('3. 画布 WebGL 上下文 + 非空截图', async () => {
        const fallback = await page.$('.avatar-canvas-fallback')
        if (fallback) throw new Error('WebGL fallback 已显示: ' + (await fallback.textContent()))
        const canvas = await page.waitForSelector('.avatar-canvas', { timeout: 10000 })
        // 等几帧让 rAF loop 画上
        await page.waitForTimeout(600)
        const ctxInfo = await page.evaluate(() => {
            const c = document.querySelector('.avatar-canvas')
            const gl = c.getContext('webgl2')
            return {
                hasWebgl2: !!gl,
                w: gl ? gl.drawingBufferWidth : 0,
                h: gl ? gl.drawingBufferHeight : 0,
                isContextLost: gl ? gl.isContextLost() : true,
            }
        })
        if (!ctxInfo.hasWebgl2) throw new Error('getContext("webgl2") 返回 null')
        if (ctxInfo.w <= 0 || ctxInfo.h <= 0) throw new Error('drawingBuffer 为 0: ' + JSON.stringify(ctxInfo))
        canvasPng = await canvas.screenshot()
        if (canvasPng.length < 4096) throw new Error('canvas 截图过小 bytes=' + canvasPng.length)
        return { drawingBuffer: { w: ctxInfo.w, h: ctxInfo.h }, screenshotBytes: canvasPng.length }
    }))

    // ===== 5. 主题翻转 (浅/深 各截一张) =====
    rec(await assert('5. 主题翻转 (html.dark) + 双截图', async () => {
        const html = page.locator('html')
        const before = await html.evaluate(el => el.classList.contains('dark'))
        const toggle = page.locator('.tool-topbar .topbar-right .el-button.is-circle')
        await toggle.waitFor({ timeout: 5000 })
        // cycle: auto -> light -> dark -> auto
        await toggle.click(); await page.waitForTimeout(200)   // auto -> light
        const after1 = await html.evaluate(el => el.classList.contains('dark'))
        await toggle.click(); await page.waitForTimeout(250)   // light -> dark
        const after2 = await html.evaluate(el => el.classList.contains('dark'))
        if (after2 !== true || after1 !== false) {
            throw new Error(`主题翻转异常 before=${before} light=${after1} dark=${after2} (期望 false/false/true)`)
        }
        await page.screenshot({ path: path.join(SHOT_DIR, 'shot-dark.png'), fullPage: false })
        await toggle.click(); await page.waitForTimeout(200)   // dark -> auto(回到 light, headless prefers-color-scheme=light)
        const after3 = await html.evaluate(el => el.classList.contains('dark'))
        await page.screenshot({ path: path.join(SHOT_DIR, 'shot-light.png'), fullPage: false })
        return { before, light: after1, dark: after2, autoResolved: after3 }
    }))

    // ===== 4a. 姿态预设按钮激活 =====
    rec(await assert('4a. 姿态预设按钮激活', async () => {
        const waveBtn = page.locator('.stage-topbar button:has-text("Wave")')
        await waveBtn.waitFor({ timeout: 5000 })
        await waveBtn.click()
        await page.waitForTimeout(200)
        const cls = (await waveBtn.getAttribute('class')) || ''
        if (!cls.includes('el-button--primary')) throw new Error('未激活 class=' + cls)
        const tp = page.locator('.stage-topbar button:has-text("T-pose")')
        const tpCls = (await tp.getAttribute('class')) || ''
        if (tpCls.includes('el-button--primary')) throw new Error('其他按钮也被激活')
        return { active: 'Wave', cls }
    }))

    // ===== 4b. 骨骼 transform 数值可编辑 =====
    rec(await assert('4b. 骨骼 transform 数值可编辑', async () => {
        const bone = page.locator('.bone-tree .bone-row:has-text("forearm_r")').first()
        await bone.click()
        await page.waitForTimeout(200)
        const nameTxt = (await page.textContent('.bt-name'))?.trim()
        if (nameTxt !== 'forearm_r') throw new Error('bt-name=' + nameTxt)
        const firstInput = page.locator('.bone-transform .el-input-number input').first()
        const before = await firstInput.inputValue()
        await firstInput.fill('1.5')
        await page.keyboard.press('Tab')   // blur -> @change 触发
        await page.waitForTimeout(250)
        const after = await firstInput.inputValue()
        if (after === before) throw new Error(`数值未变 before=${before} after=${after}`)
        return { bone: nameTxt, before, after }
    }))

    // ===== 4c. 时间轴 scrub =====
    rec(await assert('4c. 时间轴 scrub 改变时间', async () => {
        const t0 = (await page.textContent('.cursor-time'))?.trim()
        const track = await page.waitForSelector('.timeline-track', { timeout: 5000 })
        const box = await track.boundingBox()
        // 点右半
        await page.mouse.click(box.x + box.width * 0.75, box.y + box.height / 2)
        await page.waitForTimeout(250)
        const t1 = (await page.textContent('.cursor-time'))?.trim()
        if (!t0 || !t1 || t0 === t1) throw new Error(`时间未变: '${t0}' == '${t1}'`)
        return { before: t0, after: t1 }
    }))

    // ===== 6. 摄像头无设备降级 =====
    rec(await assert('6. 摄像头无设备降级态', async () => {
        const openBtn = page.locator('.tool-topbar button:has-text("摄像头调试")')
        await openBtn.click()
        await page.waitForSelector('.camera-peek', { timeout: 5000 })
        const startBtn = page.locator('.camera-actions .el-button--primary:has-text("启动摄像头")')
        await startBtn.waitFor({ timeout: 5000 })
        await startBtn.click()
        await page.waitForFunction(() => {
            const el = document.querySelector('.camera-peek')
            return el && /state-(no_device|unsupported|error)/.test(el.className)
        }, { timeout: 10000 })
        const cls = (await page.getAttribute('.camera-peek', 'class')) || ''
        const state = (cls.match(/state-([a-z_]+)/) || [])[1] || 'unknown'
        // NO_DEVICE/ERROR 应有"再试"按钮; UNSUPPORTED 可能没有
        const retryCount = await page.locator('.camera-alert button:has-text("再试")').count()
        if (state === 'no_device' || state === 'error') {
            if (retryCount < 1) throw new Error(`状态 ${state} 缺"再试"按钮`)
        }
        return { state, retryCount }
    }))

    // ===== 7. 导出 glTF (glb) =====
    let exportDetail
    rec(await assert('7. 导出 glb 文件可解析 (magic glTF)', async () => {
        const openBtn = page.locator('.tool-topbar button:has-text("导出")')
        await openBtn.click()
        await page.waitForSelector('.el-dialog', { timeout: 5000 })
        const [download] = await Promise.all([
            page.waitForEvent('download', { timeout: 15000 }),
            page.click('.el-dialog__footer .el-button--primary'),
        ])
        const filename = download.suggestedFilename()
        const dlPath = await download.path()
        const st = await stat(dlPath)
        if (st.size < 8) throw new Error('下载文件过小 bytes=' + st.size)
        const buf = await readFile(dlPath)
        const magic = buf.slice(0, 4).toString('ascii')
        if (magic !== 'glTF') throw new Error('glb magic 错误: ' + JSON.stringify(magic))
        exportDetail = { filename, bytes: st.size, magic }
        return exportDetail
    }))

    // ===== 8. VRM 载入 hookup — 上传 → onImported → loadVRMIntoScene =====
    rec(await assert('8. 上传 .vrm 后 loadVRMIntoScene 被调用', async () => {
        // 打开导入抽屉
        await page.click('.tool-topbar button:has-text("导入模型")')
        await page.waitForSelector('.import-drop', { timeout: 5000 })
        // 注入虚拟 .vrm 字节(后端能存,VRMLoaderPlugin 会判它不是合法 VRM 抛错;
        // 测的是"代码路径真的被调了",不是 VRM 内容合法性)
        // magic 'glTF' + version=2, 后端按 glb 落盘
        const vrmBytes = Buffer.from(new Uint8Array([0x67, 0x6c, 0x54, 0x46, 0x02, 0x00, 0x00, 0x00]))
        const input = await page.$('.import-drop input[type=file]')
        await input.setInputFiles({ name: 'fake.vrm', mimeType: 'application/octet-stream', buffer: vrmBytes })
        await page.waitForTimeout(300)
        // 点"上传"
        await page.click('.file-card .el-button--primary:has-text("上传")')
        // 等 console.info "[vrmLoader] loadVRMIntoScene" 出现 — 证明 onImported → loadVRMIntoScene 真被调
        let hookupSeen = false
        for (let i = 0; i < 60; i++) {
            if (consoleInfos.some(t => /\[vrmLoader\]/.test(t))) {
                hookupSeen = true
                break
            }
            await page.waitForTimeout(150)
        }
        if (!hookupSeen) {
            throw new Error('VRM 载入流程没触发:onImported → loadVRMIntoScene 没被调用')
        }
        return { info: consoleInfos.find(t => /\[vrmLoader\]/.test(t)) }
    }))

    // ===== 2. console.error = 0 =====
    rec(await assert('2. 控制台 0 error (已过滤环境伪错误)', async () => {
        if (consoleErrors.length > 0) throw new Error('errors: ' + JSON.stringify(consoleErrors))
        return { filteredArtifacts: true }
    }))

    await browser.close()

    const passed = results.filter(r => r.pass).length
    const report = {
        ok: passed === results.length,
        passed, total: results.length,
        generatedAt: new Date().toISOString(),
        results,
        consoleErrors,
        export: exportDetail,
    }
    await writeFile(REPORT, JSON.stringify(report, null, 2))
    log(`报告写入 ${REPORT}`)
    log(`结果 ${passed}/${results.length}`)
    process.exit(passed === results.length ? 0 : 1)
}

main().catch(async e => {
    console.error('[SMOKE] 顶层异常:', e)
    try {
        await writeFile(REPORT, JSON.stringify({
            ok: false, fatal: String(e?.stack || e), results, consoleErrors: [],
        }, null, 2))
    } catch {}
    process.exit(3)
})