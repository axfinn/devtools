import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref } from 'vue'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// FINN-92 Stage 3 · ssrules 前端验收断言（方案 v3 §6.4 的 F1–F8）
//
// 静态规则（F1/F2/F3/F4/F6/F8）走源码文本扫描 —— 照抄先例
// frontend/src/views/ai/resultShareStaticRules.test.js，不用 @vue/test-utils mount，
// 避免环境差异造成假红。
// 行为断言（F5/F7）直接调用 SFC 编译产物的 setup()，不渲染模板。

const here = path.dirname(fileURLToPath(import.meta.url))
const VIEW = path.resolve(here, './SsRuleTool.vue')
const ROUTER = path.resolve(here, '../../router/index.js')

const mockElMessage = {
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
}
const mockElMessageBox = {
  confirm: vi.fn(async () => true),
}
const mockLogout = vi.fn()
const mockAuth = {
  isAuthenticated: ref(true),
  passwordInput: ref(''),
  loggingIn: ref(false),
  login: vi.fn(),
  logout: mockLogout,
  tryStored: vi.fn(async () => true),
  authHeader: (extra = {}) => ({ 'X-Super-Admin-Password': 'test-password', ...extra }),
}
const mockUseAdminAuth = vi.fn(() => mockAuth)

vi.mock('element-plus', () => ({
  ElMessage: mockElMessage,
  ElMessageBox: mockElMessageBox,
}))

vi.mock('../../composables/useAdminAuth', () => ({
  useAdminAuth: (options) => mockUseAdminAuth(options),
}))

const mockFetch = vi.fn()

// mock 必须在 SFC 被 import 之前就位，所以这里用动态 import 而不是静态 import。
const Sfc = (await import('./SsRuleTool.vue')).default

function readSource(file) {
  return fs.readFileSync(file, 'utf8')
}

function templateOf(source) {
  const start = source.indexOf('<template>')
  const end = source.lastIndexOf('</template>')
  expect(start, '应当有 <template> 块').toBeGreaterThanOrEqual(0)
  expect(end).toBeGreaterThan(start)
  return source.slice(start, end)
}

function scriptSetupOf(source) {
  const start = source.indexOf('<script setup>')
  const end = source.lastIndexOf('</script>')
  expect(start, '应当有 <script setup> 块').toBeGreaterThanOrEqual(0)
  return source.slice(start, end)
}

// script setup 里用 ref()/computed() 声明的绑定名
function refBindings(script) {
  const names = new Set()
  const re = /\bconst\s+([A-Za-z_$][\w$]*)\s*=\s*(?:ref|computed|shallowRef)\s*\(/g
  let m
  while ((m = re.exec(script)) !== null) {
    names.add(m[1])
  }
  return names
}

function linesContaining(source, needle) {
  return source.split('\n').filter((line) => line.includes(needle))
}

// 取出 router 里 path === '<pathLiteral>' 那一整个路由对象（含 meta）。
function routeBlockOf(source, pathLiteral) {
  const idx = source.indexOf(`path: '${pathLiteral}'`)
  if (idx < 0) return null
  const open = source.lastIndexOf('{', idx)
  if (open < 0) return null
  let depth = 0
  for (let i = open; i < source.length; i++) {
    if (source[i] === '{') depth++
    else if (source[i] === '}') {
      depth--
      if (depth === 0) return source.slice(open, i + 1)
    }
  }
  return null
}

function metaFieldOf(routeBlock, key) {
  const m = routeBlock.match(new RegExp(`${key}:\\s*['"\`]([^'"\`]*)['"\`]`))
  return m ? m[1] : null
}

// 直接调用 setup()，拿到 <script setup> 暴露出来的绑定（不渲染模板）。
function setupComponent() {
  const bindings = Sfc.setup({}, { expose: () => {} })
  expect(bindings, 'setup() 应当返回 script setup 的绑定').toBeTruthy()
  return bindings
}

function jsonResponse(status, body) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  }
}

describe('FINN-92 ssrules 前端静态校验（F1–F4 / F6 / F8）', () => {
  const source = readSource(VIEW)
  const template = templateOf(source)
  const script = scriptSetupOf(source)

  it('F1（R3）：模板里没有对 ref/computed 绑定写 .value', () => {
    const bindings = refBindings(script)
    expect(bindings.size, '应当解析出 ref/computed 绑定').toBeGreaterThan(0)
    const violations = []
    for (const name of bindings) {
      // 同时覆盖 content.value 与 content?.value 两种写法。
      // （先例 resultShareStaticRules.test.js 用的 \?\.value 只匹配后者，
      //   对 content.value 是恒绿闸门 —— 本单不沿用那个写法。）
      const re = new RegExp(`\\b${name}\\s*\\??\\.value\\b`)
      if (re.test(template)) violations.push(name)
    }
    expect(violations, `R3 违规（模板里对 ref 写了 .value）：${violations.join(', ')}`).toEqual([])
  })

  it('F2（R4）：模板里没有裸下标访问', () => {
    const violations = []
    const re = /([A-Za-z_$][\w$]*)\s*\[/g
    let m
    while ((m = re.exec(template)) !== null) {
      const before = template.slice(Math.max(0, m.index - 3), m.index)
      if (before.endsWith('?.')) continue // 可选链允许
      violations.push(m[0].trim())
    }
    expect(violations, `R4 违规（模板裸下标）：${violations.join(', ')}`).toEqual([])
  })

  it('F3：改动文件里没有调试残留', () => {
    // 本测试文件自身含有这些字面量（作为扫描模式），只扫被改的视图文件。
    const hits = ['console.log', 'TODO', 'FIXME', 'XXX', 'HACK']
      .flatMap((needle) => linesContaining(source, needle))
    expect(hits, 'SsRuleTool.vue 有调试残留').toEqual([])
  })

  it('F4：router 里有 /ssrules，且 meta.title / meta.icon / meta.category 都非空', () => {
    const block = routeBlockOf(readSource(ROUTER), '/ssrules')
    expect(block, 'router 里应当有 path: \'/ssrules\'').toBeTruthy()
    for (const key of ['title', 'icon', 'category']) {
      const value = metaFieldOf(block, key)
      expect(value, `meta.${key} 应当存在`).toBeTruthy()
      expect(value.trim().length, `meta.${key} 不应为空`).toBeGreaterThan(0)
    }
  })

  it('F6：本组件不含 blob:（本模块不涉及分享资产）', () => {
    expect(linesContaining(source, 'blob:'), 'SsRuleTool.vue 不该出现 blob:').toEqual([])
  })

  it('F8：前端改动文件里不含 compress.go 引用', () => {
    expect(linesContaining(source, 'compress.go'), '前端不该引用后端 compress.go').toEqual([])
  })
})

describe('FINN-92 ssrules 前端行为校验（F5 / F7）', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    mockElMessage.error.mockReset()
    mockElMessage.success.mockReset()
    mockLogout.mockReset()
    vi.stubGlobal('fetch', mockFetch)
  })

  it('F5：401 时组件调用 logout()（回到密码门）', async () => {
    mockFetch.mockResolvedValue(jsonResponse(401, { error: '密码错误' }))
    const bindings = setupComponent()
    await bindings.load()
    expect(mockLogout, '401 应当触发 logout()').toHaveBeenCalled()
  })

  it('F5：正常响应时 urls 每条 = window.location.origin + path', async () => {
    const paths = ['/ss/token-abc/rules.acl', '/ss/token-abc/rules.txt']
    mockFetch.mockResolvedValue(
      jsonResponse(200, {
        source: 'google.com',
        content: 'DOMAIN-SUFFIX,google.com',
        paths,
        skipped: [],
        updated_at: '2026-10-22T08:00:00Z',
      }),
    )
    const bindings = setupComponent()
    await bindings.load()
    expect(bindings.urls.value).toEqual(paths.map((p) => `${window.location.origin}${p}`))
  })

  it('F7：413 提示「过大/超限」，不能提示「格式错误」', async () => {
    mockFetch.mockResolvedValue(jsonResponse(413, { error: '内容大小超过限制', code: 413 }))
    const bindings = setupComponent()
    await bindings.save()
    expect(mockElMessage.error, '413 应当有错误提示').toHaveBeenCalled()
    const message = mockElMessage.error.mock.calls.at(-1)[0]
    expect(message, '413 的提示不能是「格式错误」').not.toContain('格式错误')
    expect(message, '413 的提示应当说明内容过大').toMatch(/过大|超限/)
  })

  it('F7：400 提示「格式错误」', async () => {
    // 后端 body 刻意不含「格式错误」四个字：提示文案必须由前端自己映射，
    // 而不是把后端原文透传（那样删掉 400 分支这条断言也会恒绿）。
    mockFetch.mockResolvedValue(jsonResponse(400, { error: '请求体不合法' }))
    const bindings = setupComponent()
    await bindings.save()
    expect(mockElMessage.error, '400 应当有错误提示').toHaveBeenCalled()
    const message = mockElMessage.error.mock.calls.at(-1)[0]
    expect(message, '400 的提示应当是「格式错误」').toContain('格式错误')
  })

  it('F7：429 提示「请求过于频繁」，不能提示「格式错误」', async () => {
    mockFetch.mockResolvedValue(jsonResponse(429, { error: 'too many requests', code: 429 }))
    const bindings = setupComponent()
    await bindings.save()
    expect(mockElMessage.error, '429 应当有错误提示').toHaveBeenCalled()
    const message = mockElMessage.error.mock.calls.at(-1)[0]
    expect(message, '429 的提示应当是限流提示').toContain('请求过于频繁')
    expect(message, '429 的提示不能是「格式错误」').not.toContain('格式错误')
  })
})

describe('FINN-92 ssrules PM 裁决固化', () => {
  const source = readSource(VIEW)

  it('P1-5：管理密码走 headerName（不是 credentialField）', () => {
    expect(linesContaining(source, "headerName: 'X-Super-Admin-Password'")).not.toEqual([])
    expect(linesContaining(source, "credentialField: 'X-Super-Admin-Password'")).toEqual([])
  })

  it('裁决 5：前端不得往 query 拼管理密码', () => {
    // 只认 header 通道：URL 里不能出现密码参数，也不能用 searchParams 拼 query。
    expect(linesContaining(source, 'admin_password='), 'URL 里不得出现密码参数').toEqual([])
    expect(linesContaining(source, 'searchParams'), '不得用 searchParams 拼 query').toEqual([])
  })

  it('裁决 6：updated_at 用 toLocaleString 渲染，不手动加时区偏移', () => {
    expect(linesContaining(source, 'toLocaleString()')).not.toEqual([])
    expect(linesContaining(source, '+08:00')).toEqual([])
    expect(linesContaining(source, '+8h')).toEqual([])
  })
})
