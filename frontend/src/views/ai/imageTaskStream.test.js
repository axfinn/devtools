import { describe, it, expect } from 'vitest'
import { drainTaskFrames, applyTaskEvent } from './imageTaskStream'

describe('drainTaskFrames', () => {
  it('解析完整帧并保留残行', () => {
    const buf = 'data: {"status":"running","progress":12}\n\ndata: {"status":"run'
    const { tasks, rest } = drainTaskFrames(buf)
    expect(tasks).toHaveLength(1)
    expect(tasks[0]).toEqual({ status: 'running', progress: 12 })
    // 半截帧不能丢，必须留给下一轮
    expect(rest).toBe('data: {"status":"run')
  })

  it('跳过心跳注释与空行', () => {
    const { tasks } = drainTaskFrames(': ping\n\ndata: {"status":"queued"}\n\n')
    expect(tasks).toHaveLength(1)
    expect(tasks[0].status).toBe('queued')
  })

  it('非法 JSON 不影响同一批次的其它帧', () => {
    const { tasks } = drainTaskFrames('data: {oops}\ndata: {"status":"running"}\n')
    expect(tasks).toHaveLength(1)
    expect(tasks[0].status).toBe('running')
  })

  it('忽略 [DONE] 与空载荷', () => {
    const { tasks } = drainTaskFrames('data: [DONE]\ndata:\n')
    expect(tasks).toHaveLength(0)
  })

  it('非字符串输入不抛错', () => {
    expect(drainTaskFrames(null).tasks).toEqual([])
  })
})

describe('applyTaskEvent', () => {
  const baseMsg = () => ({ prompt: 'a cat', progress: 0, phase: '排队中', generating: true, genImages: [], genError: '' })

  it('运行中只更新进度与阶段', () => {
    const msg = baseMsg()
    expect(applyTaskEvent(msg, { status: 'running', progress: 48, phase: '采样中 12/25' })).toBe('running')
    expect(msg.progress).toBe(48)
    expect(msg.phase).toBe('采样中 12/25')
    expect(msg.generating).toBe(true)
  })

  it('成功时写入图片并收尾', () => {
    const msg = baseMsg()
    const images = [{ url: '/api/chat/uploads/ai/20261028/a.png', width: 1024, height: 1024 }]
    expect(applyTaskEvent(msg, { status: 'succeeded', images })).toBe('succeeded')
    expect(msg.genImages).toEqual(images)
    expect(msg.progress).toBe(100)
    expect(msg.generating).toBe(false)
    expect(msg.genError).toBe('')
  })

  it('成功但没有 images 字段时退化为空数组', () => {
    const msg = baseMsg()
    applyTaskEvent(msg, { status: 'succeeded' })
    expect(msg.genImages).toEqual([])
  })

  it('失败时带上后端原因', () => {
    const msg = baseMsg()
    expect(applyTaskEvent(msg, { status: 'failed', error: '出图超时' })).toBe('failed')
    expect(msg.genError).toBe('出图超时')
    expect(msg.generating).toBe(false)
  })

  it('失败但无原因时给兜底文案', () => {
    const msg = baseMsg()
    applyTaskEvent(msg, { status: 'failed' })
    expect(msg.genError).toBe('出图失败')
  })

  it('异常入参不炸', () => {
    expect(applyTaskEvent(null, { status: 'running' })).toBe('running')
    const msg = baseMsg()
    expect(applyTaskEvent(msg, null)).toBe('running')
    expect(msg.progress).toBe(0)
  })
})
