import { describe, it, expect } from 'vitest'
import {
  createCounterSession,
  isSessionReached,
  DEFAULT_PREFS,
  MAX_SESSION_TAPS,
  MAX_SESSIONS,
  SESSION_IDLE_MS,
  TRAINING_STORAGE_KEY
} from './useCounterSession'

const DAY1 = new Date('2026-09-16T10:00:00').getTime()
const DAY1_23_00 = new Date('2026-09-16T23:00:00').getTime()
const DAY1_23_50 = new Date('2026-09-16T23:50:00').getTime()
const DAY2_00_00 = new Date('2026-09-17T00:00:00').getTime()
const DAY2_07_50 = new Date('2026-09-17T07:50:00').getTime()
const DAY1_23_59_59_999 = new Date('2026-09-16T23:59:59.999').getTime()

function makeMemoryStorage(seed = {}) {
  const map = new Map(Object.entries(seed).map(([key, value]) => [key, String(value)]))
  return {
    getItem: (key) => (map.has(key) ? map.get(key) : null),
    setItem: (key, value) => {
      map.set(key, String(value))
    },
    removeItem: (key) => {
      map.delete(key)
    },
    clear: () => map.clear(),
    key: (index) => [...map.keys()][index] ?? null,
    get length() {
      return map.size
    },
    raw: map
  }
}

function makeSession({ startAt = DAY1, seed = {}, overrides = {} } = {}) {
  const storage = makeMemoryStorage(seed)
  let clock = startAt
  const session = createCounterSession({ now: () => clock, storage, ...overrides })
  return {
    session,
    storage,
    at: () => clock,
    setClock: (value) => {
      clock = value
    },
    advance: (ms) => {
      clock += ms
      return clock
    },
    stored() {
      const raw = storage.getItem(TRAINING_STORAGE_KEY)
      return raw ? JSON.parse(raw) : null
    }
  }
}

describe('useCounterSession · 时长双口径', () => {
  it('#7 start → end：durationMs === endedAt - startedAt', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })
    const startedAt = session.active.value.startedAt

    advance(60000)
    const finished = session.endSession({ todayCount: 42 })

    expect(finished.endedAt - finished.startedAt).toBe(60000)
    expect(finished.durationMs).toBe(60000)
    expect(finished.activeMs).toBe(60000)
    expect(finished.count).toBe(42)
    expect(startedAt).toBe(DAY1)
  })

  it('#8 暂停 60s 后继续：activeMs 扣掉暂停，pausedMs 记 60s', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })

    advance(10000)
    session.pauseSession()
    advance(60000)
    session.resumeSession()
    advance(30000)
    const finished = session.endSession({ todayCount: 5 })

    expect(finished.durationMs).toBe(100000)
    expect(finished.pausedMs).toBe(60000)
    expect(finished.activeMs).toBe(40000)
    expect(finished.activeMs).toBe(finished.durationMs - finished.pausedMs)
  })

  it('#9 后台 5 分钟：backgroundMs 记 300000，activeMs 不含该区间', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })

    advance(10000)
    session.handleBackground()
    advance(300000)
    const result = session.handleForeground(undefined, { todayCount: 0 })
    expect(result.resumed).toBe(true)

    advance(20000)
    const finished = session.endSession({ todayCount: 7 })

    expect(finished.backgroundMs).toBe(300000)
    expect(finished.activeMs).toBe(30000)
    expect(finished.durationMs).toBe(330000)
  })

  it('#10 后台超过 SESSION_IDLE_MS：自动收尾，endReason=idle，endedAt=max(lastHitAtMs,hiddenSinceMs)', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })

    advance(1000)
    session.recordTap({ at: session.active.value.startedAt + 1000, delta: 1, kind: 'tap' })
    const lastHitAtMs = session.active.value.lastHitAtMs

    advance(1000)
    session.handleBackground()
    const hiddenSinceMs = session.active.value.hiddenSinceMs

    advance(SESSION_IDLE_MS + 60000)
    const result = session.handleForeground(undefined, { todayCount: 0 })

    expect(result.idleEnded).toBe(true)
    const finished = session.sessions.value[0]
    expect(finished.endReason).toBe('idle')
    expect(finished.endedAt).toBe(Math.max(lastHitAtMs, hiddenSinceMs))
    expect(session.active.value).toBe(null)
  })
})

describe('useCounterSession · 跨零点', () => {
  it('#11 23:59 开始、00:00:30 拆分 → 两段，日归属与计数都不丢', () => {
    const { session, setClock } = makeSession({ startAt: DAY1_23_00 })
    session.startSession({ todayCount: 0 })

    let todayCount = 0
    for (let index = 0; index < 3; index += 1) {
      todayCount += 1
      session.recordTap({ at: session.active.value.startedAt + index, delta: 1, kind: 'tap' })
    }

    // 跨零点：组件先取快照（此刻 todayCount 还是昨天的值），再调 handleRollover，最后才清零
    setClock(DAY2_00_00 + 30000)
    const autoContinue = true
    session.handleRollover(undefined, { todayCount, autoContinue })

    const first = session.sessions.value[0]
    expect(first.endedAt).toBe(DAY1_23_59_59_999)
    expect(first.endReason).toBe('midnight')
    expect(first.startDate).toBe('2026-09-16')
    expect(first.count).toBe(3)

    // 清零之后新段重新计数
    todayCount = 0
    const second = session.active.value
    expect(second.startedAt).toBe(DAY2_00_00)
    expect(second.startDate).toBe('2026-09-17')
    expect(second.splitFrom).toBe(first.id)
    expect(second.countAtStart).toBe(0)

    todayCount += 2
    session.recordTap({ at: DAY2_00_00 + 100, delta: 2, kind: 'tap' })
    const finishedSecond = session.endSession({ todayCount })

    expect(finishedSecond.count).toBe(2)
    expect(finishedSecond.count + first.count).toBe(5)
  })

  it('#12 跨零点顺序回归：快照必须在清零之前取，第一段 count 不为 0', () => {
    const { session, setClock } = makeSession({ startAt: DAY1_23_00 })
    session.startSession({ todayCount: 0 })
    session.recordTap({ at: DAY1_23_00 + 1, delta: 1, kind: 'tap' })
    session.recordTap({ at: DAY1_23_00 + 2, delta: 1, kind: 'tap' })

    setClock(DAY2_00_00 + 1000)
    session.handleRollover(undefined, { todayCount: 2, autoContinue: true })

    expect(session.sessions.value[0].count).toBe(2)
    expect(session.sessions.value[0].count).not.toBe(0)
  })

  it('#25a 跨零点新段 countAtStart 严格等于 0（不是 todayCount、不是上一段的值）', () => {
    const { session, setClock } = makeSession({ startAt: DAY1_23_00 })
    session.startSession({ todayCount: 137 })
    session.recordTap({ at: DAY1_23_00 + 1, delta: 1, kind: 'tap' })

    setClock(DAY2_00_00 + 1000)
    session.handleRollover(undefined, { todayCount: 500, autoContinue: true })

    expect(session.active.value.countAtStart).toBe(0)
    expect(session.countOf(0)).toBe(0)
    expect(session.active.value.countAtStart).not.toBe(500)
    expect(session.active.value.countAtStart).not.toBe(137)
  })
})

describe('useCounterSession · 回前台两阶段顺序与 pendingSplitFrom', () => {
  function setupHiddenAcrossMidnight() {
    const fixture = makeSession({ startAt: DAY1_23_00 })
    const { session } = fixture
    session.startSession({ todayCount: 0 })
    let todayCount = 3
    for (let index = 0; index < 3; index += 1) {
      session.recordTap({ at: DAY1_23_00 + index, delta: 1, kind: 'tap' })
    }
    fixture.setClock(DAY1_23_50)
    session.handleBackground()
    return { ...fixture, getTodayCount: () => todayCount, setTodayCount: (v) => { todayCount = v } }
  }

  it('#25d 回前台先结算 idle：(away > idle) 收尾且不产生次日新会话；(away <= idle) 续跑', () => {
    // (1) 睡了 8 小时 —— 按 idle 收尾，不补开
    const longAway = setupHiddenAcrossMidnight()
    longAway.setClock(DAY2_07_50)
    const resultLong = longAway.session.handleForeground(undefined, { todayCount: 0 })

    expect(resultLong.away).toBe(DAY2_07_50 - DAY1_23_50)
    expect(resultLong.idleEnded).toBe(true)
    expect(longAway.session.sessions.value.length).toBe(1)
    expect(longAway.session.sessions.value[0].endReason).toBe('idle')
    expect(longAway.session.sessions.value.some((item) => item.startDate === '2026-09-17')).toBe(false)
    expect(longAway.session.active.value).toBe(null)

    // (2) 只离开了 8 分钟 —— 续跑，不产生新会话
    const shortAway = setupHiddenAcrossMidnight()
    const resumeAt = DAY1_23_50 + 8 * 60 * 1000
    shortAway.setClock(resumeAt)
    const resultShort = shortAway.session.handleForeground(undefined, { todayCount: 3 })

    expect(resultShort.idleEnded).toBe(false)
    expect(resultShort.resumed).toBe(true)
    expect(shortAway.session.active.value).not.toBe(null)
    expect(shortAway.session.active.value.backgroundMs).toBe(8 * 60 * 1000)
    expect(shortAway.session.sessions.value.length).toBe(0)
  })

  it('#25e 后台跑到跨零点（autoContinue=false）→ 置 pendingSplitFrom；回前台按 away 决定是否补开', () => {
    // 未超 idle：补开新段
    const reopen = setupHiddenAcrossMidnight()
    reopen.setClock(DAY2_00_00 + 30000)
    reopen.session.handleRollover(undefined, { todayCount: reopen.getTodayCount(), autoContinue: false })
    const previousId = reopen.session.pendingSplitFrom.value
    expect(reopen.session.active.value).toBe(null)
    expect(previousId).not.toBe(null)

    reopen.setClock(DAY2_00_00 + 10 * 60 * 1000)
    const result = reopen.session.handleForeground(undefined, { todayCount: 0 })

    expect(result.continued).toBe(true)
    const reopened = reopen.session.active.value
    expect(reopened.startedAt).toBe(DAY2_00_00)
    expect(reopened.splitFrom).toBe(previousId)
    expect(reopened.countAtStart).toBe(0)
    expect(reopen.session.pendingSplitFrom.value).toBe(null)

    // 超 idle：不补开
    const abandoned = setupHiddenAcrossMidnight()
    abandoned.setClock(DAY2_00_00 + 30000)
    abandoned.session.handleRollover(undefined, { todayCount: abandoned.getTodayCount(), autoContinue: false })
    abandoned.setClock(DAY2_07_50)
    const resultAbandoned = abandoned.session.handleForeground(undefined, { todayCount: 0 })

    expect(resultAbandoned.continued).toBe(false)
    expect(abandoned.session.active.value).toBe(null)
    expect(abandoned.session.pendingSplitFrom.value).toBe(null)
    expect(abandoned.session.sessions.value.some((item) => item.startDate === '2026-09-17')).toBe(false)
  })
})

describe('useCounterSession · 会话目标独立于每日目标（B6）', () => {
  it('#16 默认目标恒为 0（自由训练），不是 dailyGoal - todayCount', () => {
    const { session } = makeSession()
    // 组件侧的「每日目标 300 / 今日 80」不会传入本模块 —— 本模块也不得读取它
    session.startSession({ todayCount: 80 })

    expect(session.active.value.goal.value).toBe(0)
    expect(session.active.value.goal.value).not.toBe(220)
    expect(session.prefs.value.lastGoalValue).toBe(0)
  })

  it('#17 预设目标写回 lastGoalValue，下一次 start 仍默认自由训练', () => {
    const { session } = makeSession()
    session.startSession({ goalValue: 200, todayCount: 0 })
    expect(session.active.value.goal.value).toBe(200)
    expect(session.prefs.value.lastGoalValue).toBe(200)

    session.endSession({ todayCount: 200 })
    session.startSession({ todayCount: 0 })
    expect(session.active.value.goal.value).toBe(0)
  })

  it('#18 value === 0 时 reached 恒为 false；count / duration 两种模式各自判定', () => {
    expect(isSessionReached({ mode: 'count', value: 0 }, 99999, 99999)).toBe(false)
    expect(isSessionReached({ mode: 'count', value: 100 }, 99, 0)).toBe(false)
    expect(isSessionReached({ mode: 'count', value: 100 }, 100, 0)).toBe(true)
    expect(isSessionReached({ mode: 'duration', value: 1 }, 0, 59999)).toBe(false)
    expect(isSessionReached({ mode: 'duration', value: 1 }, 0, 60000)).toBe(true)

    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })
    advance(1000)
    const finished = session.endSession({ todayCount: 500 })
    expect(finished.reached).toBe(false)
  })
})

describe('useCounterSession · 敲击日志（B5）', () => {
  it('#19 上限 500：溢出 FIFO 丢弃最旧，tapsDropped / tapsTotal 正确', () => {
    const { session } = makeSession()
    session.startSession({ todayCount: 0 })

    for (let index = 1; index <= 600; index += 1) {
      session.recordTap({ at: DAY1 + index, delta: 1, kind: 'tap' })
    }

    const active = session.active.value
    expect(active.taps.length).toBe(MAX_SESSION_TAPS)
    expect(active.tapsDropped).toBe(100)
    expect(active.tapsTotal).toBe(600)
    // 保留的是最新 500 条 → 首条对应第 101 次
    expect(active.taps[0].at).toBe(DAY1 + 101)
    expect(active.taps.at(-1).at).toBe(DAY1 + 600)
  })

  it('#20 active.taps 不落盘，但 tapsTotal / tapsDropped 落盘', () => {
    const { session, stored } = makeSession()
    session.startSession({ todayCount: 0 })
    session.recordTap({ at: DAY1 + 1, delta: 3, kind: 'tap' })
    session.flush()

    const payload = stored()
    expect(payload.active.taps).toBeUndefined()
    expect(payload.active.tapsTotal).toBe(1)
    expect(payload.active.tapsDropped).toBe(0)
  })

  it('#21 时间轴删一条 → 返回被删条目，delta 带符号', () => {
    const { session } = makeSession()
    session.startSession({ todayCount: 0 })
    session.recordTap({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    session.recordTap({ at: DAY1 + 2, delta: -3, kind: 'minus' })
    session.recordTap({ at: DAY1 + 3, delta: 1, kind: 'tap' })

    expect(session.removeTapAt(0)).toEqual({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    expect(session.removeTapAt(0)).toEqual({ at: DAY1 + 2, delta: -3, kind: 'minus' })
    expect(session.active.value.taps.length).toBe(1)
    expect(session.removeTapAt(9)).toBe(null)
    expect(session.removeTapAt(-1)).toBe(null)
  })

  it('#25b 末条 delta 恒等于最后一次计数动作（含减号 / 改步长）', () => {
    const { session } = makeSession()
    session.startSession({ todayCount: 0 })

    let lastActionDelta = 0
    const actions = [1, 1, -1, 5, -5, 20, 1]
    for (const delta of actions) {
      const applied = session.recordTap({ at: DAY1 + actions.indexOf(delta), delta, kind: 'tap' })
      if (applied) lastActionDelta = applied.delta
      expect(session.active.value.taps.at(-1).delta).toBe(lastActionDelta)
    }

    // 撤销 = 弹出末条，不变量对新末条继续成立
    const popped = session.popLastTap()
    expect(popped.delta).toBe(lastActionDelta)
    expect(session.active.value.taps.at(-1).delta).toBe(20)
  })

  it('#25c undoLast 同步会话：弹出末条后 count 自动跟随', () => {
    const { session } = makeSession()
    session.startSession({ todayCount: 0 })

    let todayCount = 0
    todayCount += 1
    session.recordTap({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    todayCount += 1
    session.recordTap({ at: DAY1 + 2, delta: 1, kind: 'tap' })
    expect(session.countOf(todayCount)).toBe(2)

    const popped = session.popLastTap()
    todayCount = Math.max(0, todayCount - popped.delta)
    expect(session.active.value.taps.length).toBe(1)
    expect(session.countOf(todayCount)).toBe(1)
  })

  it('countOf 是 todayCount 增量的投影（下限 0）', () => {
    const { session } = makeSession()
    session.startSession({ todayCount: 100 })
    expect(session.countOf(103)).toBe(3)
    expect(session.countOf(40)).toBe(0)
    expect(session.countOf(100)).toBe(0)
  })
})

describe('useCounterSession · 不变量与重置', () => {
  it('#13/#15 session.count === max(0, todayCount - countAtStart)', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 50 })

    let todayCount = 50
    for (let index = 0; index < 10; index += 1) {
      todayCount += 1
      session.recordTap({ at: DAY1 + index, delta: 1, kind: 'tap' })
    }
    // 撤销 3 次
    for (let index = 0; index < 3; index += 1) {
      const popped = session.popLastTap()
      todayCount = Math.max(0, todayCount - popped.delta)
    }
    expect(session.countOf(todayCount)).toBe(7)
    expect(session.countOf(todayCount)).toBe(Math.max(0, todayCount - 50))

    advance(1000)
    const finished = session.endSession({ todayCount })
    expect(finished.count).toBe(7)
  })

  it('#14 reset：会话收尾 endReason=reset，count 不为负，active 归 null', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 200 })
    advance(1000)

    const finished = session.endSession({ todayCount: 0, reason: 'reset' })

    expect(finished.endReason).toBe('reset')
    expect(finished.count).toBe(0)
    expect(session.active.value).toBe(null)
    expect(session.prefs.value.trainingMode).toBe(false)
  })

  it('endSession 把 taps 快照写进会话历史（结束后可回溯修正）', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })
    session.recordTap({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    session.recordTap({ at: DAY1 + 2, delta: 1, kind: 'tap' })
    advance(5000)

    const finished = session.endSession({ todayCount: 2 })
    expect(finished.taps.length).toBe(2)
    expect(session.tapsForSession(finished.id).taps.length).toBe(2)
  })

  it('#21b 已结束会话的时间轴删除：冻结 count 按 delta 反向修正且不为负，并落盘', () => {
    const { session, advance, stored } = makeSession()
    const frozen = (id) => session.sessions.value.find((item) => item.id === id)
    session.startSession({ todayCount: 0 })
    session.recordTap({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    session.recordTap({ at: DAY1 + 2, delta: -3, kind: 'minus' })
    session.recordTap({ at: DAY1 + 3, delta: 5, kind: 'tap' })
    advance(5000)

    const finished = session.endSession({ todayCount: 3 })
    expect(frozen(finished.id).count).toBe(3)

    // 删掉 +5 → count 3 - 5 = 0
    const removed = session.removeSessionTap(finished.id, 2)
    expect(removed).toEqual({ at: DAY1 + 3, delta: 5, kind: 'tap' })
    expect(frozen(finished.id).count).toBe(0)
    expect(session.tapsForSession(finished.id).taps.length).toBe(2)
    expect(stored().sessions[0].count).toBe(0)

    // 再删 -3 → count 0 - (-3) = 3（负 delta 反向加回来）
    session.removeSessionTap(finished.id, 1)
    expect(frozen(finished.id).count).toBe(3)

    // 越界 / 非法 / 不存在的会话一律返回 null 且不改动
    expect(session.removeSessionTap(finished.id, 9)).toBe(null)
    expect(session.removeSessionTap(finished.id, -1)).toBe(null)
    expect(session.removeSessionTap('nope', 0)).toBe(null)
    expect(frozen(finished.id).count).toBe(3)

    // 删空最后一条：count = 3 - 1 = 2，且空时间轴再删返回 null 不改动
    session.removeSessionTap(finished.id, 0)
    expect(session.tapsForSession(finished.id).taps.length).toBe(0)
    expect(frozen(finished.id).count).toBe(2)
    expect(session.removeSessionTap(finished.id, 0)).toBe(null)
    expect(frozen(finished.id).count).toBe(2)
  })

  it('removeSessionTap 只动已结束会话，不影响进行中的 active', () => {
    const { session, advance } = makeSession()
    session.startSession({ todayCount: 0 })
    session.recordTap({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    advance(1000)
    const finished = session.endSession({ todayCount: 1 })

    session.startSession({ todayCount: 10 })
    session.recordTap({ at: DAY1 + 9, delta: 2, kind: 'tap' })

    expect(session.removeSessionTap(finished.id, 0).delta).toBe(1)
    expect(session.active.value.taps.length).toBe(1)
    expect(session.countOf(12)).toBe(2)
  })
})

describe('useCounterSession · 存储健壮性', () => {
  it('#22 归一化容错：null / 数组 / 版本不符 / 字段缺失 / 类型错误一律不抛', () => {
    const cases = [
      null,
      '[]',
      '{"version":99}',
      JSON.stringify({ version: 1, prefs: null, active: [], sessions: 'oops' }),
      JSON.stringify({ version: 1, active: { startedAt: 'x' }, sessions: [null, 3, {}] }),
      JSON.stringify({ version: 1, active: { startedAt: DAY1, taps: 'oops', goal: 7 } }),
      JSON.stringify([1, 2, 3])
    ]

    for (const raw of cases) {
      const fixture = makeSession({ seed: raw ? { [TRAINING_STORAGE_KEY]: raw } : {} })
      expect(() => fixture.session.init()).not.toThrow()
      expect(Array.isArray(fixture.session.sessions.value)).toBe(true)
      expect(fixture.session.active.value === null || typeof fixture.session.active.value === 'object').toBe(true)
      expect(Object.keys(DEFAULT_PREFS).every((key) => key in fixture.session.prefs.value)).toBe(true)
    }
  })

  it('归一化会剔除非法 taps 条目并把 active 的 taps 补成数组', () => {
    const raw = JSON.stringify({
      version: 1,
      active: {
        startedAt: DAY1,
        countAtStart: 0,
        goal: { mode: 'count', value: 100 },
        taps: [{ at: DAY1 + 1, delta: 1, kind: 'tap' }, { at: 'x', delta: 1 }, null, { at: DAY1 + 2, delta: 2 }]
      },
      sessions: [
        { id: 'a', startedAt: DAY1, endedAt: DAY1 + 10, count: 3, activeMs: 5, taps: [{ at: DAY1, delta: 1 }] }
      ]
    })
    const fixture = makeSession({ seed: { [TRAINING_STORAGE_KEY]: raw } })
    fixture.session.init()

    expect(fixture.session.active.value.taps.length).toBe(2)
    expect(fixture.session.active.value.taps[1]).toEqual({ at: DAY1 + 2, delta: 2, kind: 'tap' })
    expect(fixture.session.sessions.value[0].count).toBe(3)
    expect(fixture.session.sessions.value[0].endReason).toBe('manual')
  })

  it('#23 容量：250 条会话 → 保留最新 200 条并按 startedAt 倒序', () => {
    const raw = JSON.stringify({
      version: 1,
      sessions: Array.from({ length: 250 }, (_, index) => ({
        id: `s-${index}`,
        startedAt: DAY1 + index * 1000,
        endedAt: DAY1 + index * 1000 + 500,
        count: index,
        activeMs: 400,
        goal: { mode: 'count', value: 0 },
        endReason: 'manual'
      }))
    })
    const fixture = makeSession({ seed: { [TRAINING_STORAGE_KEY]: raw } })
    fixture.session.init()

    const list = fixture.session.sessions.value
    expect(list.length).toBe(MAX_SESSIONS)
    expect(list[0].id).toBe('s-249')
    expect(list.at(-1).id).toBe('s-50')
    for (let index = 1; index < list.length; index += 1) {
      expect(list[index - 1].startedAt).toBeGreaterThanOrEqual(list[index].startedAt)
    }
  })

  it('#24 多标签页：单键整体写、后写覆盖，不做合并', () => {
    const storage = makeMemoryStorage()
    const a = createCounterSession({ now: () => DAY1, storage })
    const b = createCounterSession({ now: () => DAY2_00_00, storage })

    a.init()
    b.init()
    a.startSession({ todayCount: 0, at: DAY1 })
    b.startSession({ todayCount: 0, at: DAY2_00_00 })

    const stored = JSON.parse(storage.getItem(TRAINING_STORAGE_KEY))
    expect(stored.active.startedAt).toBe(DAY2_00_00)
    expect(stored.active.id).toBe(b.active.value.id)
    expect(stored.active.id).not.toBe(a.active.value.id)
  })

  it('#25 红线：本模块任何读写路径都不改动 counter_v4_state / counter_v4_records', () => {
    const state = JSON.stringify({ todayCount: 12, dailyGoal: 300 })
    const records = JSON.stringify([{ date: '2026-09-15', dayName: '周二', count: 88, peakSpeed: 120, dailyGoal: 300 }])
    const seed = { counter_v4_state: state, counter_v4_records: records }

    const storage = makeMemoryStorage(seed)
    const session = createCounterSession({ now: () => DAY1, storage })

    session.init()
    session.startSession({ todayCount: 12 })
    session.recordTap({ at: DAY1 + 1, delta: 1, kind: 'tap' })
    session.handleBackground()
    session.handleForeground(DAY1 + 1000, { todayCount: 13 })
    session.endSession({ todayCount: 13 })
    session.flush()
    session.clearSessions()
    session.resetAll()

    expect(storage.getItem('counter_v4_state')).toBe(state)
    expect(storage.getItem('counter_v4_records')).toBe(records)
  })

  it('#47 语义：resetAll 会抹掉整个训练键并回到默认 prefs', () => {
    const { session, storage } = makeSession()
    session.startSession({ goalValue: 200, todayCount: 0 })
    session.endSession({ todayCount: 0 })
    expect(storage.getItem(TRAINING_STORAGE_KEY)).not.toBe(null)

    session.resetAll()

    expect(storage.getItem(TRAINING_STORAGE_KEY)).toBe(null)
    expect(session.sessions.value).toEqual([])
    expect(session.active.value).toBe(null)
    expect(session.prefs.value.goalValue).toBe(0)
    expect(session.prefs.value.trainingMode).toBe(false)
  })

  it('clearSessions 只清会话历史，不动进行中的 active', () => {
    const { session } = makeSession()
    session.startSession({ todayCount: 0, at: DAY1 })
    session.endSession({ todayCount: 1, at: DAY1 + 1000 })
    session.startSession({ todayCount: 1, at: DAY1 + 2000 })

    const activeId = session.active.value.id
    session.clearSessions()

    expect(session.sessions.value).toEqual([])
    expect(session.active.value.id).toBe(activeId)
  })

  it('配额异常时丢弃最旧会话重试，仍失败则静默放弃', () => {
    let failTimes = 1
    const storage = makeMemoryStorage()
    const original = storage.setItem.bind(storage)
    storage.setItem = (key, value) => {
      if (failTimes > 0) {
        failTimes -= 1
        throw new Error('QuotaExceededError')
      }
      original(key, value)
    }

    const session = createCounterSession({ now: () => DAY1, storage })
    // 先塞满 60 条历史（按 startedAt 倒序 —— 与模块落盘口径一致）
    for (let index = 0; index < 60; index += 1) {
      session.sessions.value.push({
        id: `s-${index}`,
        startedAt: DAY1 - index,
        endedAt: DAY1 + index + 1,
        startDate: '2026-09-16',
        count: 1,
        durationMs: 1,
        activeMs: 1,
        pausedMs: 0,
        backgroundMs: 0,
        goal: { mode: 'count', value: 0 },
        reached: false,
        speedPeak: 0,
        endReason: 'manual',
        splitFrom: null,
        tapsTotal: 0,
        tapsDropped: 0,
        taps: []
      })
    }

    expect(session.flush()).toBe(true)
    expect(session.sessions.value.length).toBeLessThanOrEqual(10)
  })
})
