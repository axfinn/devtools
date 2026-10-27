/**
 * newsMarkdown —— 每日新闻页（`views/life/NewsTool.vue`）的轻量 Markdown 渲染，零依赖。
 *
 * 为什么独立成模块（FINN-191）：修复 href 属性注入（存储型 XSS）时，
 * 「转义 + 链接构造」必须能被单测**直接断言渲染结果**。逻辑内联在 .vue 里时，
 * 只能靠读源码做静态断言，证明不了「渲染出来的 HTML 里没有注入」。
 *
 * 安全约定（改这里之前先读完）：
 *   1. 进入 HTML 的文本一律先过 escapeHtml —— 它转义 `& < > " '` 五个字符。
 *      引号是必需的：`href="..."` 是双引号属性，一个裸 `"` 就能闭合属性、注入事件属性
 *      （修复前 escapeHtml 只转义 `& < >`，见本仓库 origin/main @ 16205d5 的 NewsTool.vue）。
 *   2. 链接 URL 走「还原原文 → 协议白名单 → 危险字符校验 → 属性安全转义」四步，
 *      白名单只有 http/https/mailto，其余一律降级成 `#`。
 *   3. 链接文本（label）与 URL 分开处理：label 随整段文本走 escapeHtml，URL 走 safeHref。
 */

/** HTML 实体表（`&` 必须最先替换，否则会把已生成的实体再转义一次） */
const HTML_ESCAPES = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;'
}

/** 允许出现在链接里的协议白名单（javascript:/data:/vbscript: 等一律降级） */
const ALLOWED_URL_SCHEME = /^(https?:|mailto:)/i

/**
 * URL 里出现空白 / 引号 / 尖括号 / 反引号 ⇒ 极可能是在尝试逃逸 href 属性，一律降级为 `#`。
 * 合法 URL 不该含裸空白（应写 `%20`），所以这条规则不会误伤正常链接。
 */
const UNSAFE_URL_CHARS = /[\s"'`<>]/

/**
 * 转义 HTML 敏感字符。
 * 与修复前的实现相比多了 `"` 和 `'`：旧版只转义 `& < >`，导致 `href="..."` 里出现裸引号时
 * 属性被提前闭合、可以注入 `onmouseover` 之类的事件属性。
 *
 * @param {unknown} s
 * @returns {string}
 */
export function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (ch) => HTML_ESCAPES[ch])
}

/**
 * escapeHtml 的逆运算：把 URL 从「已转义文本」还原回原文。
 * 链接是在转义后的文本上匹配的，捕获到的 URL 是转义过的形态；
 * 要对**原始 URL** 做协议白名单 / 危险字符校验，就必须先还原。
 *
 * `&amp;` 必须最后还原，否则会把刚解出来的实体二次解码。
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
 * 把链接 URL 变成可以安全写进 `href="..."` 的值。
 * 返回 `'#'` = 降级（协议不在白名单，或 URL 里有想逃逸属性的字符）。
 * 正常 URL 的输出与修复前的实现逐字节一致（escape ∘ unescape 是恒等）。
 *
 * @param {unknown} rawUrl
 * @returns {string}
 */
export function safeHref(rawUrl) {
  const url = unescapeHtml(rawUrl)
  if (!ALLOWED_URL_SCHEME.test(url)) return '#'
  if (UNSAFE_URL_CHARS.test(url)) return '#'
  return escapeHtml(url)
}

/**
 * 行内元素：链接 / 加粗 / 斜体 / 行内代码。
 * 输入是**原文**，函数内部先整体转义再解析，因此调用方不需要预先转义。
 *
 * @param {unknown} text
 * @returns {string}
 */
export function inlineMd(text) {
  let t = escapeHtml(text)
  // 链接 [文本](url) —— URL 单独做协议白名单 + 危险字符校验，不合规降级为 '#'
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
 *
 * @param {unknown} src
 * @returns {string}
 */
export function renderMarkdown(src) {
  if (!src) return ''
  const lines = String(src).split(/\r?\n/)
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
