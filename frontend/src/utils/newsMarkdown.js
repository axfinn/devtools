/**
 * newsMarkdown —— 每日新闻页（views/life/NewsTool.vue）的轻量 Markdown 渲染，零依赖。
 *
 * 为什么独立成模块（FINN-187）：修复 href 属性注入（存储型 XSS）时，
 * 「转义 + 链接构造」必须能被单测**直接断言渲染结果**。逻辑内联在 .vue 里时，
 * 只能靠读源码做静态断言，证明不了「渲染出来的 HTML 里没有注入」。
 *
 * 安全约定（改这里之前先读完）：
 *   1. 进入 HTML 的文本一律先过 escapeHtml —— 它转义 & < > " ' 五个字符。
 *      引号是必需的：href="..." 是双引号属性，一个裸 " 就能闭合属性、注入事件属性。
 *   2. 链接 URL 走「还原原文 → 协议白名单 → 危险字符校验 → 属性安全转义」四步，
 *      白名单只有 http/https/mailto，其余一律降级成 '#'。
 *   3. 链接文本（label）与 URL 分开处理：label 走 escapeHtml，URL 走 safeHref。
 */

/** HTML 实体表（& 必须最先替换，否则会把已生成的实体再转义一次） */
const HTML_ESCAPES = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;'
}

/** 允许出现在链接里的协议白名单（其余协议 javascript:/data:/vbscript: 一律降级） */
const ALLOWED_URL_SCHEME = /^(https?:|mailto:)/i

/**
 * URL 里出现空白 / 引号 / 尖括号 / 反引号 ⇒ 极可能是在尝试逃逸 href 属性，一律降级。
 * 合法 URL 不该含裸空白（应写 %20），所以这条规则不会误伤正常链接。
 */
const UNSAFE_URL_CHARS = /[\s"'`<>]/

/**
 * 转义 HTML 敏感字符。
 * 与旧版实现相比多了 " 和 '：旧版只转义 & < >，导致 href="..." 里出现裸引号时
 * 属性被提前闭合、可以注入 onmouseover 之类的事件属性。
 */
export function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (ch) => HTML_ESCAPES[ch])
}

/**
 * escapeHtml 的逆运算：把 URL 从「已转义文本」还原回原文。
 * 链接是在转义后的文本上匹配的，捕获到的 URL 是转义过的形态；
 * 要对**原始 URL** 做协议白名单 / 危险字符校验，就必须先还原。
 */
function unescapeHtml(s) {
  return String(s)
    .replace(/&#39;/g, "'")
    .replace(/&quot;/g, '"')
    .replace(/&gt;/g, '>')
    .replace(/&lt;/g, '<')
    .replace(/&amp;/g, '&')
}

/**
 * 把链接 URL 变成可以安全写进 href="..." 的值。
 * 返回 '#' = 降级（协议不在白名单，或 URL 里有想逃逸属性的字符）。
 * 正常 URL 的输出与旧实现逐字节一致（escape ∘ unescape 是恒等）。
 */
export function safeHref(rawUrl) {
  const url = unescapeHtml(rawUrl)
  if (!ALLOWED_URL_SCHEME.test(url)) return '#'
  if (UNSAFE_URL_CHARS.test(url)) return '#'
  return escapeHtml(url)
}

/**
 * 行内元素：链接 / 加粗 / 斜体 / 行内代码（输入已是转义后文本）。
 */
export function inlineMd(text) {
  let t = escapeHtml(text)
  // 链接 [文本](url) —— URL 单独做白名单 + 危险字符校验，不合规降级为 '#'
  t = t.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, txt, url) => {
    return `<a href="${safeHref(url)}" target="_blank" rel="noopener noreferrer">${txt}</a>`
  })
  t = t.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  t = t.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
  t = t.replace(/`([^`]+)`/g, '<code>$1</code>')
  return t
}

/**
 * 块级渲染：标题 / 无序&有序列表 / 段落。
 */
export function renderMarkdown(src) {
  if (!src) return ''
  const lines = src.split(/\r?\n/)
  let html = ''
  let inList = false
  const closeList = () => { if (inList) { html += '</ul>'; inList = false } }
  for (const line of lines) {
    if (/^\s*$/.test(line)) { closeList(); continue }
    const h = line.match(/^(#{1,6})\s+(.*)$/)
    if (h) {
      closeList()
      const lv = h[1].length
      html += `<h${lv} class="md-h md-h${lv}">${inlineMd(h[2])}</h${lv}>`
      continue
    }
    if (/^\s*[-*]\s+/.test(line)) {
      if (!inList) { html += '<ul class="md-ul">'; inList = true }
      html += `<li>${inlineMd(line.replace(/^\s*[-*]\s+/, ''))}</li>`
      continue
    }
    if (/^\s*\d+\.\s+/.test(line)) {
      if (!inList) { html += '<ul class="md-ul">'; inList = true }
      html += `<li>${inlineMd(line.replace(/^\s*\d+\.\s+/, ''))}</li>`
      continue
    }
    closeList()
    html += `<p>${inlineMd(line)}</p>`
  }
  closeList()
  return html
}
