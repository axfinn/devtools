import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// FINN-66 Stage5 静态校验
// R3：Vue 模板里不要在 ref / computed 上写 .value（模板会自动 unwrap）
// R4：Vue 模板里不要写裸下标 obj.x[0]（field 缺失时炸挂载），要写 obj.x?.[0]
// 其余为方案 v3 §「实现自查 / 验收 grep」清单的固化（防回退用）
const here = path.dirname(fileURLToPath(import.meta.url))

const PANEL = path.resolve(here, '../../components/AIGatewaySpeechPanel.vue')
const IMAGE_TOOL = path.resolve(here, './ImageUnderstandingTool.vue')
const STUDIO = path.resolve(here, './MiniMaxStudioTool.vue')

const TARGETS = [
  { name: 'AIGatewaySpeechPanel.vue', file: PANEL },
  { name: 'ImageUnderstandingTool.vue', file: IMAGE_TOOL },
  { name: 'MiniMaxStudioTool.vue', file: STUDIO },
]

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

describe('FINN-66 Stage5 静态校验：R3 / R4', () => {
  for (const target of TARGETS) {
    describe(target.name, () => {
      const source = readSource(target.file)
      const template = templateOf(source)
      const script = scriptSetupOf(source)

      it('R3：模板里没有对 ref/computed 绑定写 .value', () => {
        const bindings = refBindings(script)
        expect(bindings.size, '应当解析出 ref/computed 绑定').toBeGreaterThan(0)
        const violations = []
        for (const name of bindings) {
          const re = new RegExp(`\\b${name}\\s*\\?\\.value\\b`)
          if (re.test(template)) {
            violations.push(name)
          }
        }
        expect(violations, `R3 违规（模板里对 ref 写了 .value）：${violations.join(', ')}`).toEqual([])
      })

      it('R4：模板里没有裸下标访问', () => {
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
    })
  }
})

describe('FINN-66 Stage5 静态校验：方案 v3 验收 grep 固化（防回退）', () => {
  it('不得把 blob URL 写进 draft（MiniMaxStudioTool.vue / ImageUnderstandingTool.vue 无 "blob:"）', () => {
    expect(linesContaining(readSource(STUDIO), 'blob:')).toEqual([])
    expect(linesContaining(readSource(IMAGE_TOOL), 'blob:')).toEqual([])
  })

  it('三个改动文件里没有 console.log / TODO / FIXME', () => {
    for (const target of TARGETS) {
      const hits = linesContaining(readSource(target.file), 'console.log')
        .concat(linesContaining(readSource(target.file), 'TODO'))
        .concat(linesContaining(readSource(target.file), 'FIXME'))
      expect(hits, `${target.name} 有调试残留`).toEqual([])
    }
  })

  it('U1 防回退：payload 不得直接塞上游响应（无 "raw: syncResponse"）', () => {
    expect(linesContaining(readSource(PANEL), 'raw: syncResponse')).toEqual([])
    expect(linesContaining(readSource(STUDIO), 'raw: syncResponse')).toEqual([])
  })

  it('U1 落实：stripInlineAudio 存在，且 payload 用 stripInlineAudio(syncResponse)', () => {
    const panel = readSource(PANEL)
    expect(linesContaining(panel, 'function stripInlineAudio(')).not.toEqual([])
    expect(linesContaining(panel, 'stripInlineAudio(syncResponse')).not.toEqual([])
  })

  it('M1 防回退：同步调用点用 guessAudioMime(syncForm.format)，异步调用点才用 audio/mpeg', () => {
    const panel = readSource(PANEL)
    const callSites = linesContaining(panel, 'blobUrlToDataUrl(')
      .filter((line) => line.includes('AudioUrl.value'))
    expect(callSites.length, '应当有同步 + 异步两个调用点').toBe(2)

    const syncCall = callSites.find((line) => line.includes('syncAudioUrl.value'))
    const asyncCall = callSites.find((line) => line.includes('asyncAudioUrl.value'))
    expect(syncCall, '缺少同步调用点').toBeTruthy()
    expect(asyncCall, '缺少异步调用点').toBeTruthy()

    expect(syncCall).toContain('guessAudioMime(syncForm.format)')
    expect(syncCall).not.toContain("'audio/mpeg'")
    expect(asyncCall).toContain("'audio/mpeg'")
  })

  it('U4 落实：10MB 闸门按原始文件字节（imageFile.value?.size）', () => {
    const imageTool = readSource(IMAGE_TOOL)
    const hits = linesContaining(imageTool, 'imageFile.value?.size')
      .concat(linesContaining(imageTool, 'imageFile.value.size'))
    expect(hits, '闸门必须按原始字节计').not.toEqual([])
  })

  it('U10 落实：图像理解页 raw 用 stripInlineImage 剔除图片 base64', () => {
    const imageTool = readSource(IMAGE_TOOL)
    expect(linesContaining(imageTool, 'const stripInlineImage')).not.toEqual([])
    expect(linesContaining(imageTool, 'stripInlineImage(rawResult.value')).not.toEqual([])
  })
})
