// 本地出图任务的 SSE 解析（纯函数，便于单测）。
//
// 后端 /api/chat-plus/image/tasks/:id/stream 的事件体统一是任务 JSON，
// 因此这里只需要「按行取出 JSON」+「把 JSON 合进消息对象」两件事。
// 与对话流的解析差异：对话流是 OpenAI chunk（choices[].delta），
// 出图流是完整任务快照（status/progress/images），不能共用一套。

/**
 * 从缓冲区里切出完整的 SSE 帧。
 * @param {string} buffer 累积的原始文本
 * @returns {{tasks: object[], rest: string}} tasks 为已解析的任务快照，rest 是未完结的残行
 */
export function drainTaskFrames(buffer) {
  const tasks = []
  const lines = String(buffer || '').split('\n')
  const rest = lines.pop() // 最后一行可能不完整，留到下次
  for (const raw of lines) {
    const line = raw.trim()
    if (!line.startsWith('data:')) continue // 心跳注释行 / 空行
    const payload = line.slice(5).trim()
    if (!payload || payload === '[DONE]') continue
    try {
      tasks.push(JSON.parse(payload))
    } catch {
      // 截断的帧直接跳过，下一轮会重新对齐
    }
  }
  return { tasks, rest }
}

/**
 * 把一条任务快照合进消息对象。
 * @param {object} msg 消息（会被原地修改）
 * @param {object} task 后端任务快照
 * @returns {'running'|'succeeded'|'failed'}
 */
export function applyTaskEvent(msg, task) {
  if (!msg || !task || typeof task !== 'object') return 'running'

  if (typeof task.progress === 'number') msg.progress = task.progress
  if (task.phase) msg.phase = task.phase
  // 出图常常几分钟没有进度变化（上传参考图/排队/首次加载模型），
  // 有 elapsed_ms 就能让用户看到「确实在跑」，不至于以为卡死。
  if (typeof task.elapsed_ms === 'number') msg.elapsedMs = task.elapsed_ms

  if (task.status === 'succeeded') {
    msg.genImages = Array.isArray(task.images) ? task.images : []
    msg.progress = 100
    msg.phase = '完成'
    msg.generating = false
    return 'succeeded'
  }
  if (task.status === 'failed') {
    msg.genError = task.error || '出图失败'
    msg.generating = false
    return 'failed'
  }
  return 'running'
}
