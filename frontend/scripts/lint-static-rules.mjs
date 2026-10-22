#!/usr/bin/env node
/**
 * DevTools 前端静态规则检查（零依赖，直接 node 跑）。
 *
 * 为什么不用 ESLint：项目此前 CI 里写的是 `npm run lint || echo "ESLint not configured yet"`
 * + continue-on-error，而 package.json 里根本没有 lint 脚本 → 这道闸门从来没真正跑过，
 * 永远显示绿色。这里补一个不依赖任何包的真闸门，覆盖项目自己在 AGENTS.md / memory 里
 * 定死的那几条会白屏/崩溃的硬规则：
 *
 *   R3  模板里不要对 ref/computed 绑定写 .value（Vue 自动 unwrap，写了读到 undefined）
 *   R4  模板里不要写裸下标 obj.x[0]（字段缺失时炸挂载），要写 obj.x?.[0]
 *   L1  first-party 代码里不要留 console.log（第三方 vendored 目录 src/neon 除外）
 *
 * 退出码 0 = 通过，1 = 有违规（CI 会红）。
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const SRC = path.resolve(here, '../src')

// 第三方 vendored 产物，不归我们管
const VENDORED_DIRS = new Set(['neon'])
// 测试文件里的 console.log 是断言对象，不算残留
const SKIP_FILE_RE = /\.test\.(js|mjs|ts)$/

function walk(dir, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (VENDORED_DIRS.has(entry.name)) continue
      walk(full, out)
    } else if (/\.(vue|js|mjs)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

/**
 * 取顶层 <template> ... </template> 之间的内容。
 * 必须用行锚定(^)匹配:组件里有大量嵌套的 <template #slot> ... </template>,
 * 用 lastIndexOf('</template>') 会抓到脚本区里出现的那个,把整段脚本误当成模板。
 */
function templateOf(source) {
  const start = source.search(/^<template>/m)
  const end = source.search(/^<\/template>/m)
  if (start < 0 || end <= start) return ''
  return source.slice(start, end)
}

/** script setup 里用 ref()/computed()/shallowRef() 声明的绑定名 */
function refBindings(script) {
  const names = new Set()
  const re = /\bconst\s+([A-Za-z_$][\w$]*)\s*=\s*(?:ref|computed|shallowRef|reactive)\s*\(/g
  let m
  while ((m = re.exec(script)) !== null) names.add(m[1])
  return names
}

/** script setup 里用 ref()/computed() 声明的对象属性的顶层名（obj.state → obj） */
function objectBindings(script) {
  const names = new Set()
  const re = /\bconst\s+([A-Za-z_$][\w$]*)\s*=\s*(?:ref|reactive)\s*\(\s*\{/g
  let m
  while ((m = re.exec(script)) !== null) names.add(m[1])
  return names
}

const violations = []

function report(file, line, rule, message) {
  violations.push({ file: path.relative(path.resolve(here, '..'), file), line, rule, message })
}

function checkTemplateRules(file, source) {
  const template = templateOf(source)
  if (!template) return
  const scriptStart = source.search(/^<script setup>/m)
  const scriptEnd = source.search(/^<\/script>/m)
  const script = scriptStart >= 0 && scriptEnd > scriptStart
    ? source.slice(scriptStart, scriptEnd)
    : ''
  const refs = refBindings(script)
  const objects = objectBindings(script)
  const templateOffset = source.search(/^<template>/m)
  // 模板的起止行号(纯行号单位,不要和字符偏移混用)
  const templateStartLine = templateOffset >= 0
    ? source.slice(0, templateOffset).split('\n').length - 1
    : -1
  const templateEndLine = templateStartLine + template.split('\n').length

  source.split('\n').forEach((line, idx) => {
    if (idx < templateStartLine || idx >= templateEndLine) return
    const lineNo = idx + 1

    // R3: {{ xxx.value }} 且 xxx 是本文件声明的 ref/computed 绑定
    const valueRe = /\{\{[^}]*?\b([A-Za-z_$][\w$]*)\.value\b[^}]*?\}\}/g
    let m
    while ((m = valueRe.exec(line)) !== null) {
      if (refs.has(m[1])) {
        report(file, lineNo, 'R3', `模板里对 ref "${m[1]}" 写了 .value（Vue 会自动 unwrap），应改成 {{ ${m[1]} }}`)
      }
    }

    // R4: obj.x[0] 裸下标（没有 ?.）
    const indexRe = /\b([A-Za-z_$][\w$]*)\.([A-Za-z_$][\w$]*)\[\s*0\s*\]/g
    while ((m = indexRe.exec(line)) !== null) {
      const owner = m[1]
      // 只对本文件里确实声明过的对象/绑定报警，避免误伤后端返回的普通字段
      if (!objects.has(owner) && !refs.has(owner)) continue
      if (line.includes('?.[0]')) continue
      report(file, lineNo, 'R4', `模板里裸下标 ${owner}.${m[2]}[0]，字段缺失时会炸挂载，应改成 ${owner}.${m[2]}?.[0]`)
    }
  })
}

function checkConsoleLog(file, source) {
  if (SKIP_FILE_RE.test(file)) return
  source.split('\n').forEach((line, idx) => {
    if (/\bconsole\.log\s*\(/.test(line)) {
      report(file, idx + 1, 'L1', 'first-party 代码里残留 console.log（改用 console.debug / console.error）')
    }
  })
}

const files = walk(SRC)
for (const file of files) {
  const source = fs.readFileSync(file, 'utf8')
  checkConsoleLog(file, source)
  if (file.endsWith('.vue')) checkTemplateRules(file, source)
}

if (violations.length === 0) {
  console.log(`[lint] 通过：扫描 ${files.length} 个文件，无违规`)
  process.exit(0)
}

console.error(`[lint] 发现 ${violations.length} 处违规：`)
for (const v of violations) {
  console.error(`  ${v.file}:${v.line}  [${v.rule}] ${v.message}`)
}
process.exit(1)
