# 事项管理(Planner)产品哲学与优化框架

> **北极星:事项管理 = 把脑子的负担卸到工具上,让注意力回到"现在该做的事"。**

---

## 一、用户真实痛点(为什么这件事值得做)

事项管理不是 Todo List。大部分用户的真实痛点:

1. **思维负担** —— 脑子里的事太杂,挤占了真正重要的事的带宽
2. **决策疲劳** —— 一打开就一堆事,反而什么都不想做
3. **失控感** —— 昨天的事没做完,堆积如山,越堆越不想看
4. **认知错位** —— 工作和生活分不清边界,互相干扰
5. **情绪反弹** —— 取消/推迟后,看见"又没做"产生挫败感,直接关掉应用

---

## 二、三大设计原则(始终如一)

### 原则 1:能 1 步就别 2 步

**反例**:点开卡片 → 看到详情 → 点编辑按钮 → 进抽屉 → 改 → 保存 = 5 步
**正例**:在卡片上直接改日期/优先级/状态 = 1 步

具体表现:
- 状态切换:点卡片左上角圆圈,直接循环 todo → done → cancelled
- 优先级:卡片上直接显示,长按/右键改
- 日期:卡片上的日期就是按钮,点开就是日期选择器
- 推迟:一键"明天做/下周做"

### 原则 2:让系统判断,只让用户选择

用户不该想"这件事该放到哪个 bucket"。
系统应该根据"有没有明确日期/时间"自动归类:
- 有日期 → 时间线
- 没日期 + 用户主动说"以后再说" → 也许某天(someday)
- 没日期 + 没说话 → 收件箱(inbox,待分类)
- 有具体时间 → 事件(event)

### 原则 3:数据平滑迁移,不破坏历史

- 老的 `Bucket` 字段已经存在(`inbox/planned/someday`),**不能改字段语义**
- 老的 task 数据即使 bucket 错乱,也不能迁移丢
- 只在展示层做"重新归类"建议,不动数据库
- 删除走"软删除 + 撤销",5 秒内可恢复

---

## 三、信息架构重设计(以"决策负担最小"为目标)

### 现状问题(基于 9681 行单文件分析)

```
- 245 个函数、30 个 dialog/drawer 组件
- 主页从顶到底:7 个 metric chip + 2 张 focus 卡 + 录入区 + 高级区
- 用户一进来就被一堆数字砸脸,反而不知道怎么开始
```

### 目标架构(三个分区,职责清晰)

#### A. 顶部条(Topbar):永远只有 4 个动作
- 搜索(`/`)
- 快速新建(⌘/Ctrl+K 或顶部 `+`)
- 设置
- 刷新

**删除**:快捷键说明按钮移到帮助菜单;模式标签不再重复,因为下面的页面已经切分。

#### B. 主页(Hero 区):只回答"我现在该做什么"
- **今日焦点卡**(最显眼):1 个最重要的 + 2 个次要
- **下一场事件卡**:最近 1 个有时间感的事
- **快速录入**:1 个输入框 + 4 个 preset(今天/明天/收件箱/事件)

**删除**:连续天数、近 7 天条形图、专注计时等移到"统计"页。

#### C. 列表区:三栏并列
- 时间线(planned,按日期)
- 收件箱(inbox,待分类)
- 也许某天(someday,以后再说)

**删除**:全部 metric chip 合并成顶部 1 行总览:`未完成 N | 收件箱 N | 事件 N | 顺延 N`

---

## 四、交互细节原则

| 场景 | 1 步方案 |
|------|---------|
| 改变状态 | 卡片左上角圆圈,点一下循环 |
| 改日期 | 卡片日期徽章,点击 = 日历选择器 |
| 推迟 | 卡片右下角 `⋯` 菜单,一键明天/下周 |
| 标星(置入聚焦) | 卡片右上角 `★` |
| 删除 | `⋯` 菜单 → 删除,5 秒可撤销 |
| 添加备注 | 卡片底部 `+ 备注`,点开就地编辑 |
| 评论 | 卡片底部 `💬 N`,点开抽屉 |

---

## 五、API 兼容性铁律

**永远不要破坏向后兼容**。优化只能:
- ✅ 新增字段(默认空,不影响老数据)
- ✅ 新增接口
- ✅ 在前端展示层做重新归类
- ❌ 改字段名(老数据立刻出错)
- ❌ 改字段语义(老数据语义错乱)
- ❌ 删接口(老调用方崩溃)

---

## 六、优化路线图(按优先级)

### 阶段 0:稳定(已完成)
- [x] 修复白屏 bug(空函数引用)
- [x] 修复搜索框层级问题
- [x] 修复 review 中所有 bug

### 阶段 1:减法(降低决策负担) — 进行中
- [ ] 顶部条瘦身:删除快捷键按钮、模式标签
- [ ] 主页瘦身:删除连续天数、专注计时、统计 chip
- [ ] 把"高级设置"从默认显示改为折叠

### 阶段 2:1 步交互(核心体验)
- [ ] 卡片就地改状态(无需打开抽屉)
- [ ] 卡片就地改日期
- [ ] 一键推迟菜单
- [ ] 5 秒撤销

### 阶段 3:系统判断
- [ ] 自动归类(根据日期自动判断 bucket)
- [ ] 智能排序(基于优先级 + 紧急度 + 滚降次数)
- [ ] 软提醒(长时间未动的事项温柔提醒)

### 阶段 4:数据主权
- [x] JSON 完整备份
- [x] CSV 任务列表导出
- [x] ICS 日历订阅(让用户导入到系统日历)
  - 一个 URL `/api/planner/profile/:id/calendar.ics?creator_key=xxx` 包含所有非 inbox/someday/cancelled 的事项
  - 系统日历 App 周期性拉取自动同步,改完即可见
  - 内含状态分类(CATEGORIES)+ 优先级,系统日历可按类别筛选/着色
  - 兼容历史:per-task `/tasks/:id/calendar` 端点保留
- [x] 导入备份(从 JSON 恢复,只追加不覆盖)

### 阶段 5:认知科学
- [x] 情绪化文案(完成时温柔,未完成时不强逼)
  - 完成事件: 推荐下一项 / "留点空吧" 收尾语
  - 取消事件: "放过自己,先做别的" 不带负罪感
  - 顺延/取消 prompt 默认填入最常见原因标签,1 步完成
  - 拖了很久的任务完成:"拖了 N 天,今天落定了。"
  - 多次顺延的任务完成:"顺延 X 次还是推完了,做得好。"
  - 收件箱清空:"收件箱已清空,做得好。"
  - 未完成 0 件:"今天没有未完成事项,给自己留点空。"
- [x] 完成率可视化(主页 summary-strip 里嵌入近 7 天完成节律 mini 图)
  - 软色(琥珀渐变)代替绿色"成功"色,避免炫耀感
  - "今日完成"和"连续天数"并列,温柔反馈节奏
- [x] 1 步交互的最后一公里:hero 卡片变成可点击入口
  - 4 个 metric 卡片(未完成/收件箱/事件/顺延)全部 1 键跳到对应视图
  - hero-hint 文案下加按钮(收件箱 → 直接进入 triage 模式,过载 → 跳时间线)
  - 0 件时给正向反馈(庆祝而非空状态)
- [x] "我没做完"反思记录,而不是直接关掉
  - 当前实现:取消/顺延时记录原因到 cancel_reason / postpone_reason
  - review API 增量返回 `cancellations` 字段(最多 10 条)
  - 阶段回顾弹窗新增"我没做完的事"区域:展示放弃原因分布 chips(前 3 原因 + 各自次数)+ 每条取消事项列表(标题/原因/日期/顺延次数)
  - 文案:"不是失败,是看清自己" — 把取消重构成模式洞察,不是"放弃羞耻感"
  - 配色:暖米黄底(不是红/灰),让反思变温柔
  - 顺延对称:review API 同步加 `postpones` 字段(最多 10 条)
  - 弹窗新增"我推迟的事"区域,配色雾蓝(冷)与取消暖黄形成冷暖对照
  - 排序按 rollover 倒序:高频漂移的事排前,优先引起注意
  - 排除已取消的(已经在 cancellations),避免重复打扰
- [x] 时间线减法:7 天以外的事项默认折叠
  - 默认 `timelineFolded = true`,只显示今天 + 未来 7 天
  - 折叠 banner 一行提示:"还有 N 件事排在 7 天后 → 展开看看"
  - 状态 localStorage 持久化,尊重用户选择
  - 同样作用于事件型时间线
  - 空状态文案区分:"7 天内暂无事项,更远的事已折叠"(不是空荡感)
- [x] 任务卡片"已挂 N 天"软徽章
  - 任务进入列表 3 天后,在 meta 区显示"已挂 N 天"
  - 配色暖米色(与取消聚合同色系),不是红/灰
  - 软观察:不打开新菜单,不弹窗,只显示事实
  - 让用户自己决定"继续做"还是"取消/推迟"
  - 哲学:看见承诺 vs 实际的差距 → 卸下判断负担
- [x] action bar 减法(9 按钮 → 6 + ⋯)
  - 移除"进行中"按钮(status circle 已覆盖 open→in_progress 1 步切换)
  - 移除"专注"按钮(放进 ⋯ 菜单,低频)
  - 移除"复制"按钮(放进 ⋯ 菜单,低频)
  - ⋯ 菜单:开始/结束专注、复制为新草稿、显式标记进行中
  - 主时间线:9 → 6 按钮 + 1 ⋯
  - 明天/事件:6 → 5 + 1 ⋯
  - 收件箱:4 → 3 + 1 ⋯
  - 最近完成:4 → 3 + 1 ⋯
  - 哲学:减按钮 = 减视觉噪声 = 减决策负担
- [x] 1 步交互最后一公里
  - 改优先级 4 步 → 1 步(标签点击 → 4 选项下拉 → 选 → 保存,无成功提示避免噪音)
  - 删任务 5 步 → 1 步(⋯ 菜单直接删,无 ElMessageBox 二次确认,5 秒撤销 toast 兜底)
  - 撤销比确认更温柔:不被"警告"吓到,5 秒内可改主意
  - 删除项颜色:#d97757(暖橙红,温和),不是刺眼红
  - 复用已有撤销基础设施(`snapshotTaskForUndo` / `showUndoSnackbar` / `restoreTask`)
  - 4 个 view 的 ⋯ 菜单全部加删除项(timeline / tomorrow-events / inbox / recent)
- [x] 桶 1 步化(收尾 1 步交互)
  - 改桶 4 步 → 1 步:卡片上"收件箱/计划中/放一放"标签点击 → 3 选项下拉(📅计划中 / 📥收件箱 / 💤放一放) → 选 → 静默保存
  - 复用 `moveTaskBucket` helper(进 planned 自动设今天日期)
  - 沿用 `clickable-priority` 样式(光标 + 悬浮阴影)
  - 添加 `bucketTagType` 函数(inbox→info, someday→warning, planned→空)
  - 只动 timeline 视图(其他视图桶是固定的,改桶有专门的"安排到今天"按钮)
- [x] 重复任务延续感(`completion_count` 字段)
  - 后端:`PlannerTask.CompletionCount` int,SQL 加列 + 迁移 + INSERT/UPDATE/SELECT/scan 全链路同步
  - 状态从未完成 → done 时 +1,已经是 done 时不重复 +(保留"重置"语义)
  - 兼容性:`migration 失败 duplicate column 跳过`逻辑天然支持,老数据列存在则跳过,缺则补
  - 前端:卡片 meta 区显示"已完成 N 次"暖米色徽章(从 N≥2 开始,避免噪音)
  - 第二次及以上完成时 toast:"已完成第 N 次 · 坚持就是节奏"(不再用"又完成一件"通用文案)
  - 配色:同取消暖米色系(amber #f5dec8 底 + #8a5a3b 字 + 1px 棕边),与"已挂 N 天"形成节奏
  - 哲学:让重复性任务(晨跑/周报/写日记)有"完成过几次"的延续感,而不是"又一条新任务"
  - 哲学:不显示"完成 1 次"是因为首次完成是默认状态,无需强调;从 2 次开始才有"坚持"的语义
- [x] 节奏感 + 不焦虑的明日(三件套)
  - 摘要条新增"📅 明天 N 件"chip
    - 1 键跳转到明天视图(用户不需要主动切 tab 才知道明天有什么)
    - 0 件时灰色,有件时蓝色暖色(`#355080` 蓝系,与紧急的红色 / 连续的橙色形成第四色)
    - 哲学:摘要条是"今日全景",必须包含明日 — 否则"焦虑"在用户没察觉时累积
  - 完成时刻"今日节奏"反馈
    - toast 升级:从通用"已完成" → "今日 N 件 · 这是第 K 次"(K≥2)
    - 兜底:首次完成且今日 < 3 件时显示"今日 N 件,慢慢来";今日 ≥ 3 件时显示"今天已经做完 N 件,节奏不错"
    - 不再用"下一项推荐"那种"还有要做的事"的隐性压力,而用"已经做了几件"做正向反馈
    - 哲学:完成 = 收尾,不是接力赛的"传棒"信号
  - 习惯种子 soft 提示(`habitSeedEligible` 函数)
    - 触发条件:completion_count ≥ 3 && repeat_type === 'none'
    - 显示:绿色虚线按钮"🌱 想做习惯吗?设个重复",点击打开任务 drawer 让用户决定频率
    - **不自动设置重复**:频率因人而异(每天/每 3 天/每周一),系统不能猜
    - 一旦用户设置重复,按钮自动消失(再次检查 `habitSeedEligible`)
    - 哲学:系统观察 → 温柔建议 → 用户决定(原则 2 的最直接体现)
    - 配色:绿系 `#4a6b3a` + 浅绿底,与取消/顺延/完成的暖色系形成"健康/生长"的语义对照
- [x] 能量感知(精力分类 + 低能量模式)
  - 卡片显示精力徽章(4 类):点 chip → 下拉选 1 步改
    - 🧠 需专注(deep)—— 紫,代表高认知负荷
    - 📋 事务性(shallow)—— 灰,代表随手能做
    - 🚶 需外出(errand)—— 蓝,代表要走动
    - 💡 创意型(creative)—— 黄,代表发散思考
    - 未标时显示"🪫 标精力"灰色虚线提示,1 步点击就出菜单
    - 复用 clickable-priority 样式(光标 + 悬浮阴影)
    - 在 timeline / tomorrow-events / someday 三种卡片都展示
  - "🪫 低能量"模式 toggle(摘要条)
    - 开关打开:hero 卡变 Moon icon + 文案"🪫 今天轻松一点"
    - 打开后下一项推荐过滤:只从 `shallow` + 未标 中选(默认兜底所有 open tasks)
    - 状态 localStorage 持久化(`planner.lowEnergyMode`),尊重用户选择
    - 哲学:精力是状态,不是属性 —— 同样的 task list,用户今天累了,系统就该推荐轻的
    - 不替用户删任务/改优先级 —— 只调整"下一项"这一个出口
  - 后端 bug 修复:`UpdateTask` handler 之前丢弃了 `Intent` 和 `EnergyLevel` 字段
    - 现象:PUT `{energy_level:"shallow"}` 永远返回 `energy_level: ""`
    - 根因:`updatePlannerTaskRequest` struct 声明了字段,但 handler 没读取
    - 修复:加 `if req.X != nil` 块,与现有 `req.RawText` 同模式
    - 不动 schema(列早已存在),只动 handler 逻辑,100% 向后兼容
- [x] 模式切换 1 步化 + 收尾仪式
  - 模式切换压成 summary chip(去掉独立 section)
    - 删除 30 行 `<section class="mode-switcher">`(2 大按钮 + 副标题)
    - 替换为 1 个 chip "💼 工作 · 切换" / "🏠 生活 · 切换"
    - 1 步点击切换,配色随 mode 变(工作=蓝,生活=绿)
    - topbar 的 "当前 工作模式" 小 tag 保留(滚动时也能看到状态)
    - 视觉减法:从「2 大块按钮」压到「1 个 chip」,垂直空间 -50px
    - 哲学:减按钮 = 减视觉噪声 = 减决策负担(模式切换是"调整",不是"决定",不该抢戏)
  - 收尾仪式(todayAllDone 状态)
    - 触发:`!activePinnedFocus && !displayFocusTask && done_today >= 1`
    - Hero 焦点卡变体:绿底渐变 + 🎉 emoji + "今天已经收尾" + "做得很好" 标题
    - 描述动态:"今天做完 N 件" + 明日判断(`tomorrowCount > 0 ? ",明天还有 X 件在等" : ",先给自己留点空吧"`)
    - 按钮:仅当 tomorrowCount > 0 时显示"看明天 →"(避免无意义按钮)
    - 哲学:完成 = 收尾仪式,不是"还有要做的事"的隐性压力
    - 配色:浅绿渐变(180, 220, 170),与紧急红/警告橙/低能量蓝形成"完成"的绿色健康语义
- [x] Inbox 分流 4 按钮 + 收尾总结(三件套)
  - 改 3 按钮 → 4 按钮,语义更清晰:
    - `1` 今天做 → planned bucket + planned_for = today
    - `2` 改天做 → planned bucket + planned_for = 明天(默认"先放着,明天再看")
    - `3` 放一放 → someday bucket(原样)
    - `4` 不做 → 删除 + 5 秒撤销
  - 之前"计划去做"和"今天做"语义混淆(都强制设 today),现在明确分 2 步
  - 键盘 1/2/3/4 + Esc 提示行保留,但内容更新
  - 配色:"不做"按钮从 type="danger"(刺眼红)改为暖橙红 #d97757(与删除项一致),避免"惩罚"感
  - 分流结束后,不再默默退出,而是显示 in-place 总结卡片:
    - 🎉 "收件箱已清空 · 你刚刚分流了 N 件脑子里的事"
    - 3 个 stat 卡片:今天/改天 · 放一放 · 不做
    - 动态总结文案(根据比例切换):
      - discard ≥ 3: "放下了 X 件不做的事,心里又轻了一点"
      - planned ≥ 70%: "几乎都安排上了,今天的节奏挺满"
      - someday ≥ 50%: "放一放 X 件,等以后再想起来也不迟"
      - 默认: "收件箱清爽了,脑子也能跟着松一下"
    - 2 个按钮:看时间线 / 留在收件箱
  - 哲学:分流是"卸下负担"的仪式,不是默默清空。结束时要让用户感到"事办完了"
  - 配色:浅绿渐变(180, 220, 170),与 hero 收尾仪式形成"完成 = 健康"的统一语义
  - 状态管理:`triageStats = { planned, someday, discarded }` reactive 累计,刷新收件箱后保留(在用户切回主页前可见)
- [x] 🎯 专注模式:被事情压垮时的"恐慌按钮"
  - summary-strip 新增 `🎯 专注` chip(在 🔋/🪫 和模式切换之间)
  - 开启后,时间线视图自动过滤:
    - 只保留:🔴 urgent 优先级 OR ⭐ 用户聚焦 OR 📅 今天日期 的事项
    - 其他全部隐藏,显示"🎯 专注中 · 还有 N 件非紧急被收起"banner
    - 1 步点击"解除专注"即可恢复完整视图
  - 空状态单独处理:开启但无关键件时显示"🎯 专注中 · 当前时间线没有关键事项 · 只有非紧急的 N 件被收起着"
  - 事件型时间线不受影响(事件自带时间感,不该被隐藏)
  - 状态持久化:`planner.focusMode` localStorage,刷新页面保留
  - 哲学对齐 #2 痛点"打开就一堆事,什么都不想做":
    - 减法 = 关闭噪声源,不是删除数据
    - 信息仍在,只是不主动呈现
    - 哲学:用户不需要"删除 todo",需要"暂时看不见"
  - 配色:紫色(190, 160, 230)与紧急红/警告橙/低能量蓝/完成绿/工作蓝/生活绿形成第七色,"专注"的语义对照(聚焦 → 紫色 = 沉思/筛选)
  - 与 timelineFolded 的关系:正交关系,fold 控制"远的事",focus 控制"不紧急的事"
  - 与 lowEnergyMode 的关系:正交关系,low 控制"下一项推荐",focus 控制"列表展示"
- [x] 顺延洞察卡片(从"看见堆积"到"1 步重新决定")
  - 当 `topRolledOverTasks.length >= 1` 时,在 summary-strip 下方显示新卡片
  - 顶部文案:"🔁 顺延回顾 · 这些事已经滑过几次了,要不要重新决定一下?"
  - 列出 top 3(按 `rollover_count * 10 + 是否过期` 倒序,高顺延 + 过期优先)
  - 每行:title + "已顺延 N 次" + planned_for 日期 + 2 按钮:
    - `📅 今天做` → 1 步 `updateTask` 设置 planned_for=today + status=open
    - `不再做了` → 1 步 `updateTask` 设置 status=cancelled + cancel_reason="重新评估后放弃"
  - 配色:暖米黄渐变(255, 240, 220)→ 与 hero 收尾/完成 N 次/triage 收尾/已挂 N 天形成暖色系语义网("温和反思"族)
  - "不再做了"按钮用 #a67a5a(浅咖啡)而不是 #d97757(深橙红),因为顺延任务放弃 ≠ 失误放弃,而是"重新评估后放下"
  - 跳过机制:右上"今天先跳过"链接 → 写入 localStorage 到当天 23:59:59,刷新页面不再展示
  - 用户主动权:localStorage 跨会话,但只在"今天"内有效;明天自动恢复
  - 哲学对齐 #3 痛点"失控感 —— 昨天的事没做完,堆积如山,越堆越不想看":
    - 把"堆积的恐惧"变成"3 个具体决定"
    - 5 步(看见顺延→点开→扫列表→找任务→改日期)压缩为 1 步
    - 不弹窗、不打扰,卡片常驻但克制
    - "今天先跳过"给用户主动权,避免变成又一个"未读通知"
  - 与 heroHintAction(rolled_over 提示)的关系:hero 提示是"导流"(去时间线看),卡片是"决策"(直接在这做决定),后者优先级更高
- [x] 跨天延续感 + 时间问候(让 Planner 有人味,不是冷冰冰的工具)
  - 时间问候(hero 顶部)
    - 5 个时段:☀️ 早上好(< 11)、🌤️ 中午好(< 13)、⛅ 下午好(< 18)、🌆 晚上好(< 22)、🌙 夜深了(其他)
    - 静默展示,不打断操作;只是给页面"这一刻"的感觉
    - 配色:冷蓝 #355080(hero hint 暖色系的对位),克制不喧宾夺主
  - 跨天延续感(`lastSessionMemory` localStorage)
    - 每次进入页面记录:`{ at: ISO, done_count: total }`
    - 下次进入对比时间差:
      - < 1 天 → 不打扰(同一天不必寒暄)
      - 1 天 → 📅 "1 天没见" + "上次 X月X日(周X)完成 N 件"
      - 2 天 → 👋 "2 天没见"
      - 3–6 天 → 🌙 "3 天没见 / 快一周了"
      - 7–13 天 → "快一周了 / 快两周了"
      - ≥ 14 天 → 🌟 "好久不见"
    - 1 步关闭:"好的"按钮 → 写入当天 dismissed flag,不再弹出
    - 仅在「打开主页」首次加载时触发(单 session 内重复刷新不打扰)
    - 显示位置:hero-hint 上方(rollover 卡片之上) —— 温度感优先,洞察次之
  - 哲学对齐 #4 痛点"没有连续的节奏感,做一天断一天":
    - 工具不该"假装认识用户"——问候的依据是真实时间差 + 真实完成数
    - 也不要"过度热情"——同一天再来不寒暄,只在你真的离开了才说"欢迎回来"
    - 颜色用冷蓝(不是暖橙/绿),与所有"完成 / 健康"语义暖色保持距离,避免"在夸你"的过度
    - 哲学:让用户感觉"它记得我",而不是"它在监控我"——边界是真实历史 + 1 步关闭
- [x] 取消 1 步化(popover 替代模态)
  - 现状:取消走 ElMessageBox.prompt 模态 → 屏幕变灰 + focus trap + 回车 / 点确定才能提交 → **3 步**(按钮 → 模态 → Enter/确定)
  - iter 13:替换为 `CancelReasonPopover` 内联组件(el-popover,轻量,无遮罩)
    - trigger 按钮 → popover 弹出 → 4 个预设原因 chip → **1 步完成**
    - "其他原因"输入框:用户主动选,2 步但表达自由(回车提交)
    - 集成位置:3 个 task card actions 行(timeline / tomorrow / events)+ nextEventTask 卡片
  - 4 个预设原因(从原 placeholder 提炼,全部带 emoji):
    - 🪶 不再需要(默认)
    - 🔄 条件变化
    - ⏸️ 不是现在
    - 🤝 已委派给他人
  - 本地记忆(localStorage `planner.lastCancelReason`)
    - 记录最近一次原因
    - 下次打开 popover,该原因 chip 高亮(`cancel-reason-chip-active`)
    - 减少重复思考,贴合"用户最常用的原因 = 默认"
  - 温柔反馈(按原因给一句话,不评价不催促)
    - 不再需要 → "放过不需要的事,留点精力给重要的"
    - 条件变化 → "环境变了,放下也是智慧"
    - 不是现在 → "留个口子,以后想得起再回来"
    - 已委派给他人 → "已委托出去,期待对方推进"
    - 用户自定义原因 → 不反馈(避免误判语义)
  - 配色:暖米黄底 `#fff8f0` + 浅咖啡 `#a67a5a` 边/字 + 暖橙红 hover,与"已挂 N 天 / 完成 N 次"形成同色系语义网("温和放下"族)
  - 哲学对齐 #4 痛点"做完就算了 → 但取消/顺延很麻烦":
    - 用户最常见的取消诉求是"就是不想做了" → 模态让 ta 停下来想"为什么" → 增加心理摩擦
    - 1 步 chip 让 ta 快速完成 → 让"放弃"和"完成"一样轻
    - "已委派给他人"和"不做"在哲学上完全不同(放下 ≠ 推卸),但用户都需要快速完成 → chip 让 ta 表达但不让 ta 犹豫
    - 本地记忆 ≠ 监控:基于 ta 自己的最近选择,不上传,跨会话但不跨设备
  - 保留向后兼容:旧的 `cancelTask(task)` 函数签名保留,内部走 `cancelTaskWithReason(task, '不再需要')` 默认原因(供 setTaskStatus 等内部调用)
  - 与"删除"按钮的关系:都在 ⋯ 菜单/buttons 中,但取消走 1 步 chip,删除走 5 秒撤销 toast(两个语义不同:取消 = 任务还有"已取消"标签,数据保留;删除 = 数据本身清空)
- [x] 习惯种子 1 步化(把"已完成 3 次"的最佳时机抓住)
  - 现状:🌱 按钮 → 打开完整 task drawer → 找"提醒时间" → 选时间 → "重复提醒"下拉激活 → 选频率 → 保存(**5+ 步**,心智负担重)
  - iter 14:`HabitSeedPopover` 内联组件(模式同 CancelReasonPopover)
    - 4 个频率 chip:☀️ 每天 / 💼 工作日 / 📅 每周 / 🌙 每月
    - 点 chip → 1 步提交 updateTask(`repeat_type` + 智能默认 `remind_at`)
    - 不打开 drawer,无模态,无遮罩
  - 智能默认 `remind_at`
    - 如果任务已有 `remind_at` → 沿用其小时分钟(尊重用户的旧习惯)
    - 否则默认 19:00(晚上回顾/打卡时段,符合"已完成 3 次"的心智)
  - 自动隐藏机制
    - 现有 `habitSeedEligible` 已检查 `repeat_type !== 'none'` → 设了频率后按钮自动消失
    - 无需新增后端字段,数据 100% 平滑
  - 温柔反馈
    - Toast:"节奏建立起来了,慢慢来"(不催促、不打鸡血)
  - 配色:沿用 habit seed 现有绿色 `#4a6b3a` + 浅绿底 `#f0f8e6` + 浅绿边(与"🌱 想做习惯吗"按钮色调一致)
    - 形成"绿色生长"语义族,与"已挂 N 天 / 完成 N 次 / 取消"的暖色系明确拉开
  - 哲学对齐 #4 痛点"重复性任务每次都要重新设":
    - 用户已经做完 3 次 = 验证了这件事实用 → 这正是工具介入的最佳时机
    - 但 drawer 路径太重 → 用户大概率会拖延或放弃设重复
    - 1 步 popover 把"建立习惯"的成本压到最低 → 抓住 momentum
  - 与 iter 13 取消 1 步化的对称:
    - iter 13 = 放下任务 1 步(温柔放下)
    - iter 14 = 建立习惯 1 步(温柔建立)
    - 两个都是"系统提示 + 用户决策 1 步完成"的统一模式
  - 数据 100% 兼容
    - 只动 `repeat_type` + `remind_at`,不动 `completion_count` / `planned_for` / 历史评论
    - 老用户已设的重复任务完全不受影响
    - `habitSeedEligible` 检查逻辑不变,只是触发后的路径变了
- [x] End-of-Day 收尾仪式(批量完成 + 主动收尾)
  - 现状:晚上/睡前看到今天还剩 N 件 open,只能一件件点完成 → 摩擦 + 用户大概率放弃
  - iter 15:summary-strip 新增 chip `✓ N 收尾今天`
    - 位置:在 🎯 专注 chip 之后,模式切换 chip 之前(逻辑顺序:聚焦 → 专注 → 收尾 → 模式)
    - 只在 `todayOpenCount > 0 && !todayAllDone` 时显示 → 收尾后自动隐藏(iter 8 的 todayAllDone 状态接管)
    - 配色:浅绿渐变(`#b4dcaa` → `#dcebc8`)+ 绿边 + 绿字 `#3a4a2a` + 数字 `#4a6b3a`
    - 与 iter 8 收尾仪式同色系,语义连贯"绿色健康/收尾"
    - hover:上浮 + 阴影,1 步可点的视觉暗示
  - 1 步交互(用 mini 弹窗兜底)
    - 点击 chip → ElMessageBox.confirm("收尾今天")
    - 文案:"今天还有 N 件没完成,一次性全部标完成?"
    - 2 按钮:`全部收尾`(主)/ `再想想`(次)
    - 不用 popover:批量操作不可逆,需要明确的"决策瞬间"
    - 不是删除那种 ElMessageBox.confirm(`type: 'warning'`,刺眼红),而是用 `type: 'info'`,符合"收尾 = 健康"
  - 批量后端复用
    - 走现有 `tasks/batch-update` endpoint,action=`mark_done`(iter 之前就有)
    - 自动调用 `recordCompletionToday()` × N,正确累计"今日完成"和 streak
    - 之后调 `refreshBoard()` → 自动触发 iter 8 的 `todayAllDone` 状态 → hero 变"今天已经收尾"
  - 触发反馈
    - Toast:"收尾了 N 件 · 今天做得很好"(不催促、不打鸡血,符合"收尾 = 健康"语义)
    - 收尾后 chip 自动消失(todayAllDone 判断)
    - 收尾后 hero 变绿底 + 🎉 + "今天已经收尾" + "做得很好"(iter 8 复用)
  - 哲学对齐 #4 痛点"做了就行,但累的时候想全收尾":
    - 用户疲惫/不想做时,不需要逐一决策 → 1 步批量收尾 → 完成感不减
    - 与 iter 8 的对称:iter 8 = 被动收尾(完成最后一件),iter 15 = 主动收尾(任何时刻)
    - 收尾 ≠ 偷懒:用户决定"今天够了"是健康信号,工具配合这个决定
    - 不在白天强制显示,只在有 N 件未完成时显示 → 避免"诱惑用户偷懒"
  - 数据 100% 兼容
    - 走现有 `mark_done` 批量 action,数据 schema 不变
    - `completion_count` 正确累计 → "已完成 N 次"徽章正常显示
    - 历史评论、cancel_reason、postpone_reason 全部保留
  - 与 iter 8 的链路
    - iter 8 收尾仪式:被动(完成最后一件时显示)
    - iter 15 收尾仪式:主动(任何时候一键批量)
    - 两者走同一份 board → 收尾后 hero 都显示"今天已经收尾"
- [x] review 弹窗 1 步复活(补完"看见 → 反思 → 决定"闭环)
  - 现状痛点:review 弹窗的"我没做完的事"和"我推迟的事"只能展示
    - 用户看见"我经常因为'条件变化'取消" → 想"其实 X 我应该做"
    - 旧路径:关掉 review → 搜索 X → 找到 → 改状态 → 设今天日期(5 步)
    - iter 16:在 review 行内直接 1 步复活
  - 后端小幅改动(纯增量)
    - `plannerCancellationItem` 加 `TaskID` 字段(json tag `task_id`)
    - `plannerPostponeItem` 加 `TaskID` 字段
    - populate 处加 `TaskID: task.ID`
    - 数据 schema 不变,只新增字段
    - 老客户端忽略 `task_id` 字段,完全兼容
  - 前端 1 步交互
    - "我没做完的事"每行加 🔄 重新捡起按钮(暖绿 dashed `#4a6b3a`)
      - 点击 → updateTask(`status=open`, `planned_for=今天`, 清空 `cancel_reason`)
      - Toast:"已捡起 · 今天做掉它"
      - 自动重新拉 review + board,弹窗里该条消失
    - "我推迟的事"每行加 📅 今天做按钮(冷蓝 dashed `#355080`)
      - 与"重新捡起"配色对称(暖绿 vs 冷蓝),语义对称(重新做 vs 调整做)
      - 点击 → updateTask(`status=open`, `planned_for=今天`)
      - Toast:"今天做掉它"
  - 复用 updateTask + loadReview 现有函数,无需新建 API
  - 哲学对齐"看清自己 → 决定改变"
    - iter 11 顺延洞察:在主页提醒,让用户预防漂移
    - iter 16 review 复活:在反思场景给用户决定权
    - 两个都是"系统温柔提示 → 用户 1 步决定",但场景互补:
      - iter 11 是"提前看见"漂移(预防)
      - iter 16 是"事后看清"取消/推迟(复盘)
  - 数据 100% 兼容
    - `task_id` 是新增字段,老 profile 数据进入 review 时正常填充(每条 cancellation 都有源 task.id)
    - 即使某些历史 task 已被硬删除,该条 `task_id=""` → 前端 `v-if="item.task_id"` 自动隐藏按钮,不报错
    - updateTask 路径完全沿用,不动 schema
  - 与 iter 13/14/15 的链路
    - iter 13:取消 1 步化(决定放下)
    - iter 14:习惯种子 1 步化(决定建立)
    - iter 15:收尾今天 1 步化(决定收尾)
    - iter 16:review 复活 1 步化(决定改变)
    - 形成完整的"决定 → 1 步完成"决策矩阵,任何场景都能找到 1 步路径

### 阶段 5 减法 · 迭代 17:完成反思(感受打标)

- **新字段**:`planner_tasks.completion_feeling`(enum TEXT DEFAULT '')
  - 三选一 enum:`smooth`(顺手)/ `learned`(学到)/ `rough`(划水),空表示未标
  - **完全可空,完全向后兼容**:已有 500+ 历史 task 全部 `''`,前端 review 区按"打过标"的子集聚合,不影响任何旧数据
  - SQL 迁移 `ALTER TABLE ... ADD COLUMN ... DEFAULT ''` 复用阶段 4 cancel_reason 的成熟模式
- **交互**:完成任务时,1.4s 后浮出"感觉怎么样?"温柔卡片(暖色,与"下一项"蓝系 message 区分)
  - 3 chip:🪶 顺手 / 💡 学到 / 😐 划水,点击 → 1 步 PUT `/tasks/:id` 带 `completion_feeling`
  - 6s 自动消失,可主动"跳过"
  - 1 步打标,不打断"刚做完"的爽感节奏(避开 toast 和 maybeShowNextUp 的 700ms 窗口)
  - 任务已有 `completion_feeling` 时不再二次弹出(避免重复打扰)
- **review 弹窗**:"我做完了感觉…" 区,与"我没做完的事"/"我推迟的事"对称
  - 顶部"完成感受分布"chip(顺手 · N / 学到 · N / 划水 · N)
  - 下方按时间倒序展示最近 10 条样本(标题 + 完成时间 + 感受图标)
  - 配色用暖系(浅米黄)与"推迟"冷系(浅蓝)区分视觉语义
- **哲学对齐**:"做完 → 感觉如何"是收尾闭环的最后一块
  - 看见自己放弃什么(iter 13)→ 看见自己推迟什么(iter 11/16)→ 看见自己完成时的状态(iter 17)
  - 形成完整的"放下/推迟/完成"三维反思,review 不再只是数字
  - 不强制打标(可跳过 / 可不选),但提供 1 步路径 — 符合"系统判断,只让用户选择"
- **零 schema 破坏**
  - 完成感受是可选 enum(空字符串合法),旧任务永远 `'smooth'/'learned/'rough'/''` 四态之一
  - review 聚合只看"打过标"的任务,review 弹窗对该字段完全没数据的用户不展示新区
  - 与 iter 16 一样,数据平滑迁移,无破坏性

### 阶段 5 减法 · 迭代 18:完成 1 步撤回 (Undo Complete)

- **问题**:删除有 5 秒 undo 兜底(`quickDeleteTask` + `showUndoSnackbar`),但"误点完成"没有对应路径。
  - 用户一旦完成,只能去已完成里翻找,再手动改回 open
  - 不对称造成用户对"完成"操作产生不安全感
  - 这违反"真实便捷用户管理好事件"原则
- **方案**:完成时 5 秒内可 1 步撤回
  - 复用现有 `undoState` 系统(`showUndoSnackbar` + `handleUndo`),零新基础设施
  - 撤回 → `PUT /tasks/:id` 带 `{status: open}`,后端 default 分支自动清 `completed_at` 和 `cancel_reason`
  - 撤回时同时 `dismissCompletionFeedback()`,避免"已撤回"和"感觉怎么样"卡冲突
  - `maybeShowNextUp` 增加状态检查:如果任务已不在 done,跳过"推荐下一项"
- **数据语义保留**:
  - `completion_count` 不重置 —— 代表"历史真实完成的次数",撤回不算"没完成过"
  - `completion_feeling` 不清空 —— 如果用户已打标,撤回是误操作不是后悔感受
  - 用户撤回后再完成,count 自然递增,符合"我确实又做了一次"的真实
- **哲学对齐**:"删除 ↔ 完成"对称,补完 1 步交互的纠错闭环
  - iter 14/15/16/17:决定 → 1 步完成(完成路径)
  - iter 18:决定 → 1 步撤回(纠错路径)
  - 形成"做 / 撤"双向 1 步交互,任何操作都有兜底
- **零 schema 破坏**
  - 后端 UpdateTask 已有 default 分支处理 `status=open`(line 559-561),无新代码
  - 前端只复用现有 `undoState` + `updateTask`,无新组件
  - 历史数据完全不动,纯前端交互补完

### 阶段 5 减法 · 迭代 19:重复任务完成 → 自动建下次

- **问题**:已有 `repeat_type` 字段(用于 ICS 日历导出),但"完成 → 下次出现"完全靠用户手动
  - 用户设"每天 8 点跑步",今天标记 done,明天打开发现没有,忘了重建
  - 每天重建同一件事 = 重复劳动,违反"真实便捷"
  - 之前 iter 18 已有 undo 完成兜底,但"忘了重建"是更常见漏网场景
- **方案**:完成时自动建下次实例
  - 后端纯函数 `plannerNextOccurrenceDate(task, now)`:
    - daily:base + interval 天
    - weekdays:跳到下一个 Mon-Fri(周五完成 → 下周一)
    - weekly:base + 7*interval 天
    - monthly:base + interval 月(Go AddDate 自动处理月底 31→28)
    - 尊重 `repeat_until`:超出则不建
  - UpdateTask 在 `before != done && task == done` transition 时建下次,响应回 `next_task_id` + `next_planned_for`
  - 撤回保护:iter 18 undo 时只是 `open` 状态,不会触发新逻辑
  - 前端 0 新 UI:完成后 2.2s 自动飘出"🔁 下次自动建到 6 月 27 日"提示
- **数据平滑**(iter 19 灵魂)
  - 旧任务完全没有 `repeat_type` 或 `repeat_type='none'` → 行为完全不变,不会自动建
  - 已有的 `repeat_type='daily'` 任务(虽然之前没自动建) → 第一次 done 后开始有下次
  - 旧 next instance 不会突然冒出来(因为之前没建),只是从今天起开始"接力"
  - 新建的实例带 `Notes: "🔁 续自 {parent_id}"`,便于以后做"重复任务族"视图
- **副产物**:每次完成产生 activity log,自动建的实例也产生 activity → 审计完整
- **哲学对齐**:"系统判断,只让用户选择"的极致体现
  - 用户只负责"今天做了"
  - 系统负责"下次什么时候"
  - 用户不需要思考"我要不要重建""重建到几号"
  - 真正的"set it and forget it"
- **零 schema 破坏**
  - 表结构 0 改动,无新列
  - 复用现有 `CreatePlannerTask`,新实例是普通 task,只是 planned_for 不同
  - 撤销 / 取消 / 改 open / 改回原状态 都不破坏"下次实例已经存在"的事实
    - 例:用户完成 → 系统建了明天 → 用户改 back to open
    - 明天那个 task 依然在 timeline 里,用户可继续或改期
    - 不需要"撤回下次实例"概念,符合"数据平滑"

### 阶段 5 减法 · 迭代 20:重复任务族可视化 (第 N 次)

- **问题**:iter 19 自动建了下次实例,Notes 里有"🔁 续自 {parent_id}",但前端不展示任何"我正在做第几次"的信息
  - 用户做重复任务多次后,核心问题浮现:"我坚持了多久?做了几次?"
  - 没有视觉提示的话,重复任务在 timeline 里看起来和普通任务一样
  - 用户感受不到"接力感",每天都是"从零开始"
- **方案**:让族系次数可见
  - 后端 `CountPlannerFamilyCompletions(profileID, title, repeatType)`:单条 SQL 查族系 done 数
    - "族"定义:同 profile + 同 title + 同 repeat_type + status='done'
    - cancelled 不算(放弃 ≠ 完成)
  - `createNextRecurringInstance`:在生成 next instance 的 notes 末尾追加"· 第 N 次"(N = done 数 + 1,含本次)
  - `UpdateTask` 响应里加 `family_completion_count` 字段(本次完成时的族系 done 数,含本次)
  - 前端:
    - `parseFamilyRound(task)`:从 notes 正则解析"第 N 次"
    - 任务卡片显示"🔁 第 N 次"chip(浅暖紫,3 个视图统一:timeline / tomorrow / events)
    - 完成 toast 改为"🔁 第 N 次完成 · 下次自动建到 X"(第 1 次不显示次数,不打扰)
- **数据平滑**(iter 20 灵魂)
  - 旧 done 任务没有 notes 标记 → `parseFamilyRound` 返回 0 → 不显示 chip(零侵入)
  - 旧 done 任务 + 之前手动建的下次实例(无 notes 标记) → 同样不显示
  - iter 19 之后完成的 → 新实例开始带 chip,旧 next instance 通过刷新后会被下次自动建的新实例覆盖
  - 数据"自然过渡",用户看到 chip 时就是真实的"我已经做了 N 次"
- **副产物**:无需建"重复任务族"独立表
  - 用 notes 字段做软关联("🔁 续自 xxx"),不破坏 schema
  - 想要做"重复任务族"全族视图时,只用 `WHERE notes LIKE '%🔁 续自 xxx%' OR id = 'xxx'` 一次查询
- **哲学对齐**:"让系统判断,只让用户选择" + "1 步完成" 的合流
  - 用户每天只做"完成" 1 步
  - 系统同时:建下次实例(iter 19) + 累计次数(iter 20) + 提示下次日期
  - 用户不增加任何操作成本,感受到"我在坚持,系统在帮我"
- **零 schema 破坏**
  - 表结构 0 改动,无新列
  - 数据存哪里:notes(已有 TEXT 字段),用"🔁 续自 xxx · 第 N 次"格式做软关联
  - 旧 next instance(无"第 N 次"标记)刷新后保持原样,不显示 chip
  - 撤销 / 取消 / 改 open / 撤回完成(iter 18) 都不影响"族系计数"的真实性
    - 例:今天完成 → 族 +1 → 明天 open → 族不减少(不重置历史)
    - 例:撤回完成 → 族不减少(代表"已发生")

### 阶段 5 减法 · 迭代 21:批量操作 1 步撤销 (单 ↔ 批 对称)

- **问题**:iter 18 让"单完成"有 undo,但批量操作(mark_done / move_to_today / move_to_someday / delete)无兜底
  - 用户批量改 5 件任务,后悔了想回滚,目前只能再批量改一次(对 delete 来说数据已永久丢失)
  - 单 ↔ 批 兜底不对称,让用户对"批量"操作产生不安全感
  - 这与"任何动作都有兜底"的 iter 18 原则矛盾
- **方案**:批量操作前置 snapshot,后置 1 步 undo
  - 操作前对每个 task 调 `snapshotTaskForUndo`(N 个 task 一次,无并发,顺序执行,防 N+1)
  - 上限 30 条:超过不弹 undo(防止意外 N+1,30 已经是批量操作合理上限)
  - 操作成功后弹 `showUndoSnackbar`,onUndo = `undoBatchOperation(action, snapshots)`
  - delete:逐个 `restoreTask`(全量重建,含 comments)
  - mark_done:逐个 PUT `status=open`(改回原状态)
  - move_to_today / move_to_someday:逐个 PUT 原 `planned_for` + `bucket`
  - 恢复完成后弹"已撤回,N 条已恢复"toast,refreshBoard
- **数据平滑**
  - snapshot 通过现有 `snapshotTaskForUndo` API 拿,与单条删除共用同一份数据
  - 恢复路径:
    - delete 走 `restoreTask`(已存在,含 comments 全量重建)
    - 其他走 PUT(同 updateTask 路径,后端会保留 completion_count / completion_feeling 等)
  - 单 ↔ 批 完全用同一套底层函数,行为一致
- **性能与边界**
  - 顺序 snapshot + 顺序 restore:不并发,避免对 DB 造成尖峰
  - 单个 restore 失败不中断:用户看到一个失败就崩体验更差
  - 超过 30 条不弹 undo:大用户可能在乎性能,但 30 是绝大多数场景
- **哲学对齐**:"1 步完成 + 1 步撤销"的对称闭环
  - iter 14/15/16/17/18/19/20:做 / 接力 / 反思(正向)
  - iter 18:单条撤销(纠错)
  - iter 21:批量撤销(纠错 + 兜底)
  - 任何"动作"都有"撤"的可能性,用户对系统的信任度上升
- **零 schema 破坏**
  - 纯前端改动,后端 0 行
  - 复用现有 snapshot API、restoreTask、updateTask、showUndoSnackbar
  - 不破坏任何现有单条删除/完成的 undo 行为

### 阶段 5 减法 · 迭代 22:评论 1 步化 (QuickCommentPopover)

- **问题**:加评论是 4 步流程:点按钮 → drawer 打开 → 输入 → 提交
  - drawer 模式适合"加详细评论+录音",但对"加 1 句话进展"太重
  - 用户可能只想记一句"今天联系了 X,等回复" → 4 步劝退
  - 与现有 CancelReasonPopover (iter 13) / TaskDatePopover (iter 14) / HabitSeedPopover (iter 14) 模式不对称
- **方案**:点评论按钮 → popover → 输入 → Enter,1 步完成
  - 复用现有 `postComment(taskId, content, [])` 函数(后端 API 完全不变)
  - popover 内同时显示最近 3 条评论预览(让用户能看见上下文)
  - 录音 / 完整评论历史仍在 drawer 里,popover 底部"看完整评论/录音 →"链接打开
  - Enter 发送,Shift+Enter 换行(直觉,符合聊天类应用习惯)
  - textarea 上限 200 字,与"1 句话"语义匹配
- **数据平滑**
  - 完全复用 `POST /tasks/:id/comments` API,与 drawer 模式走同一条路
  - drawer 仍可访问(链接打开),录音/图片评论等"重操作"用户主动用 drawer
  - 没有评论历史的旧任务 → popover 显示"还没有评论,写下第一条"空态
- **UI 设计原则**
  - 浅冷色系(input 边框 + 主色聚焦),与抽屉的"重编辑"区分(暖色 chip)
  - 200/200 字数提示实时显示,避免无意识输入过长
  - 预览评论倒序(最近 3 条),2 行省略,时间右侧小字
- **哲学对齐**:"1 步交互"在评论场景的最后一公里
  - 与现有所有 popover 组件(取消 / 日期 / 习惯)同模式,设计语言统一
  - 抽屉(复杂工具)和 popover(快速记录)分工明确
  - 用户选择工具:想快 → popover;想细 → drawer
- **零 schema 破坏**
  - 纯前端改动,后端 0 行
  - 复用现有 postComment / loadComments / refreshBoard
  - 旧评论数据完全不动,显示在 popover 预览里

---

### 阶段 5 减法 · 迭代 23:完成卡片显示"完成于 X"(日期 chip 自动切换)

- **问题**:在主页/时间线/事件视图里,已完成的任务卡片还显示"计划日期"(`planned_for`)
  - 任务可能拖了好几天才完成 → 卡片还是"今天/6/25"的样子,用户一看就懵:这事儿到底做了没?
  - 信息是真有(`task.completed_at` 后端早就在返回),但前端从不展示
  - 越积越多的"已完成"列表,日期全是一片模糊的"今天/明天",用户翻找时大脑还要重算"实际啥时候做的"
- **方案**:`TaskDateChip` 内部根据 `task.status` 自动切换显示模式
  - 未完成:仍是"📅 计划日期"chip + popover(可改期/顺延/收件箱) — 行为完全不变
  - 已完成 + 有 `completed_at`:显示"✓ 完成于 06-25 14:30",绿底,不可点(已完成不能再改期)
  - hover 提示完整时间 + 早晚天数:"完成于 2026-06-25 14:30 · 晚 3 天完成" / "提前 2 天完成" / "准时完成"
  - 时分精度(`MM-DD HH:MM`)是诚实信息 —— 让用户一眼看见"我晚上 11 点才做完的""我提前一周搞定",而不是含糊的"已完成"
- **数据平滑**
  - 0 后端改动,纯前端组件改造
  - `task.completed_at` 字段早就在 API 响应里(`*time.Time`,完成时为非 nil)
  - 旧数据(没 `completed_at` 的已完成任务) → 自动回退显示 `planned_for`,不崩
  - 旧 `TaskDateChip` 行为对未完成任务完全保留
- **UI 设计原则**
  - 绿色 (`#059669`) 表示"完成"语义,与红色"超期"/橙色"高优先级"区分
  - 不可点 (cursor: default) 防止误点,减少认知负担
  - chip 视觉权重低于"完成"按钮(底部操作区),只在标题下方作为信息条
  - 早晚天数在 hover tooltip,不在 chip 主显示 —— 避免视觉噪音,但想看就能看
- **哲学对齐**:"让系统判断,只让用户选择"在日期维度的一次延伸
  - 用户不需要再"想完成时间是哪天" —— 系统主动说
  - 同时把"早晚几天"这种洞察静默给到,但不强制看
  - 复用现有 `TaskDateChip` 组件,3 个视图(主页/时间线/事件)+ 重复任务/搜索结果自动受益
- **零 schema 破坏**
  - 完全不改数据库,不改 API
  - 一个组件一个 CSS class 改动,git diff < 60 行
  - 已部署/未部署/数据迁移都不需要做

---

### 阶段 5 减法 · 迭代 24:多任务 1 步记录 (/ 换行分隔 + batch API)

- **问题**:主页"快速记录"是单条模式 — 脑子冒出 3 件事,得:
  - 点 + → 填表 1 → 提交
  - 点 + → 填表 2 → 提交
  - 点 + → 填表 3 → 提交
  - 9 次点击、3 个空表单来回。"AI 整理入库" 又要走 4 步(开弹窗 → 输入 → AI 解析 → 全部写入),还依赖 AI 服务
  - placeholder 早就在暗示「明天下午3点 给客户发合同 紧急 / 改天再约牙医」(用 / 分隔) — 系统支持语义,前端却没真做
- **方案**:同一输入框,自动检测多任务
  - **分隔符**:`/`(前后空格可选) 或 `\n`(换行),都支持
  - **1 段** → 走原单条 `createQuickTask` 逻辑,**零回归**
  - **2+ 段** → 调后端早就存在的 `POST /tasks/batch`,一次创建
  - **每条独立智能归类**:每条都跑 `extractHintsFromText(rawText)`,独立得出 date/priority/bucket/entryType
    - 例:`明天下午3点 给客户发合同 / 改天再约牙医` → 第 1 条 = event planned, 第 2 条 = task someday
  - **共享 quickForm 字段**:通知邮箱 / 精力类型(per-task 推断不到的)
  - **保存前预览**:`将记录 N 条:` 列表 + 每条识别出的语义(如 "→ 6/26 14:00 · 紧急 · 计划中"),让 / 切分错位立刻可见
  - **按钮文案动态**:`保存` ↔ `保存 N 条`
  - **成功反馈**:`已记录 3 条:买菜 / 给妈打电话 / 回邮件给张总` (前 3 条)
  - **失败透明**:后端 400 会回 `failed_index`,前端用 ElMessage 提示哪一条挂了
  - **上限 50**:与后端 batch 上限一致
- **数据平滑**
  - **0 后端改动** — `POST /profile/:id/tasks/batch` 早就在 routes.go:287 注册,handler 在 planner_tasks.go:386
  - 完全复用现有 batch handler,跟单条走同一条 `buildPlannerTask` 智能归类路径
  - 旧任务数据完全不动;旧的单条保存行为对 1 段输入保持原样
  - maxlength 80 → 500 (批量场景需要更长,UI 已 show-word-limit 提示)
- **UI 设计原则**
  - **绿色**(`#059669` 渐变)区别于"智能归类"提示的紫色 — 紫色=单条增强,绿色=批量新场景
  - 预览 list 5 条上限,超出显示"… 还有 N 条"避免压垮输入框
  - 每条 preview 后挂"→ 已识别语义"链接,让用户确认 batch 智能归类符合预期
  - 不开新弹窗、不加新按钮、不改布局 — 同输入框 + 同按钮 + 同位置
- **哲学对齐**:"能 1 步就别 2 步"最直白的兑现
  - 3 件事 9 次点击 → 1 次输入 + 1 次保存
  - 完整保存流程从"填表 × 3"降到"打字 + 回车"
  - "让系统判断" — 系统识别分隔符、识别每条的 date/priority,用户只管想内容
  - 与现有 placeholder 暗示对齐,实现一直承诺的能力
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 `extractHintsFromText` + `pushQuickTemplate` + `refreshBoard`
  - 1 个 helper (`splitQuickTitles`) + 1 个新函数 (`createQuickTasksBatch`) + 模板若干行

---

### 阶段 5 减法 · 迭代 25:批量改优先级 1 步化 (完成 batch bar 最后一块拼图)

- **问题**:batch bar 一直缺"改优先级"操作
  - 后端 `set_priority` action 早就在 `planner_tasks.go:764` 实现,接收 `priority` 字段
  - 前端 batch bar 只能"移到今天/放一放/标完成/删除",改优先级要么 N 次单点(违反批量初衷),要么用户放弃批量改
  - 真实场景:inbox 5 个新事项,想"这 5 个都是紧急的,今天推" — 现在得 5 次点优先级 dropdown
- **方案**:batch bar 加"🚦 改优先级"el-dropdown,4 chip 一键应用
  - **Inbox batch bar**(`inbox-batch-bar`):收件箱多选时显示
  - **Global floating batch bar**(`batch-bar-floating`):跨视图多选时显示
  - 两个 bar 都加同样 dropdown(覆盖所有批量改优先级的场景)
  - **1 步完成**:选 1 批 → 1 个 chip → 全应用,无中间确认
  - **撤销机制**:`snapshotTaskForUndo` 早已包含 `priority` 字段(`plannerTool.vue:6761`),undo 时 `patch.priority = s.priority` 改回原值
  - **后端兼容**:`batchUpdatePlannerTaskRequest` schema 早有 `Priority *string` 字段(`planner_tasks.go:710`),纯前端调用
- **数据平滑**
  - 0 后端改动 — `set_priority` action 早已存在,纯前端调用
  - 0 数据库迁移
  - 旧 batch bar 4 个操作 100% 兼容,不影响存量
  - snapshot 早含 priority,撤销机制开箱即用
  - 优先级 `normalizePlannerPriority()` 已存在,前端传"urgent"/"high"/"medium"/"low" 都安全
- **UI 设计原则**
  - 复用现有 `el-dropdown` 模式(和优先级单点 dropdown 视觉一致)
  - 🚩 图标 + "改优先级" 文字让按钮在 4 个操作中不抢戏
  - 4 chip 直接对应 4 个 priority 值,无中间"自定义优先级"选项 — 减法
  - toast 反馈:`改优先级为 🔴 紧急:更新 5 条`(动态化,比固定 BATCH_LABELS 更有上下文)
- **哲学对齐**
  - **"1 步交互最后一公里"在 batch 场景的补完**:之前单条改优先级 1 步(下拉),批量却 N 步 — 这次补齐对称
  - **"让系统判断,只让用户选择"**:4 个预设优先级 = 用户真实需要的选择,系统不发明第 5 个
  - **"数据平滑"**:完全不写新后端,纯前端 + 复用既有 snapshot/undo 通路
  - **"减法"**:不加新弹窗、新页面、新组件,只给现有 batch bar 多 1 个下拉
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 el-dropdown + 现有 batch bar
  - 改动:1 个函数签名加 extra 参数 + 1 处 undo 分支 + 2 处 batch bar 加 dropdown + 1 行 icon import

---

### 阶段 5 减法 · 迭代 26:搜索无结果 1 步直接创建(消除"填进表单 + 再点保存"摩擦)

- **问题**:全局搜索框无结果时,行为是 **2 步**
  1. 用户在搜索框输入"买菜" → 无匹配 → 按 Enter
  2. 系统把"买菜"填进主页的 quickForm + 聚焦
  3. 用户**必须**再点一次"保存"按钮才完成创建
  - 摩擦点:用户其实已经"决定"创建了(无匹配 = 不存在的事),系统还要他二次确认
  - 而且 form 残留会干扰下一次输入,容易误编辑
- **方案**:`searchCreateFromKeyword` 改为 **1 步直接创建**
  - Enter / 「快速创建」按钮 → 调 `/tasks/batch` 一次创建
  - 复用 iter 24 的 splitQuickTitles,**支持 / 分隔多任务**:"买菜 / 给妈打电话" 一次建 2 条
  - 复用 extractHintsFromText,每条独立智能归类(日期/优先级/bucket/entryType)
  - toast 反馈上下文:`已创建: 买菜 · 计划 6/26` / `已创建 2 条:买菜 / 给妈打电话`
  - 撤销机制(沿用 iter 18/21 兜底):任务直接创建,用户想撤销点 snackbar 的"撤销"
- **数据平滑**
  - 0 后端改动 — `/tasks/batch` iter 19 就在,`/tasks/:id` PUT/PATCH/DELETE 都在
  - 0 数据库迁移
  - 0 schema 破坏 — 复用 `extractHintsFromText` + splitQuickTitles
  - 旧行为(填进表单)被替换,但用户体验升级,不是 regression
- **UI 设计原则**
  - **减法**:不增加新按钮,改 Enter 的语义
  - **零焦点切换**:旧行为要滚动到主页 quick form,新行为直接在原位完成,无视线跳跃
  - toast 包含分类上下文(`已创建: 买菜 · 计划 6/26`),用户**事后**能确认"我没创建错东西"
  - 撤销 snackbar 5s 内可见,与 iter 18 单删 / iter 21 批量撤销 完全对称
- **哲学对齐**:"1 步交互最后一公里"在搜索场景的补完
  - 搜索的本质是"找" + "没有就建",但旧实现把"建"藏在表单里
  - 用户的心智:无匹配 → 想建 → Enter 一气呵成
  - 系统的责任:跟上用户的心智,而不是让用户来点保存
  - "让系统判断":无匹配 = 用户想创建,这个判断比"用户主动点保存"更准
  - "减法":少 1 个步骤(填进表单 + 焦点跳转),少 1 次视线移动
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 /tasks/batch + extractHintsFromText + ElMessage
  - 改动:1 个函数(searchCreateFromKeyword)从"填表单"重写为"直接创建"

---

### 阶段 5 减法 · 迭代 27:快速模板 1 步应用(消除"模板 → 填表 → 保存"3 步摩擦)

- **问题**:`applyQuickTemplate` 仍是 **2 步**:
  1. 用户点常用模板("买菜" / "接娃" / "写日报")
  2. 系统把字段填进 quickForm + 聚焦输入框
  3. 用户**必须**再点"保存"按钮
  - 真实场景:模板就是 verbatim 用(高频日常事项),用户在"已经决定要建"上又被加了 1 步确认
  - 模板设计动机:用户把"买菜"放快捷区,就是因为每次输入标题太烦
  - 旧流程把"省下的输入"又换成"省不下的点保存",违背模板的初心
- **方案**:点击模板 → **1 步直接创建** + 5s 撤销兜底
  - 新增 `applyQuickTemplateDirect(t)` 函数,直接 POST /tasks(走原 createPlannerTaskRequest)
  - 字段从模板里取(title / priority / kind / entryType / bucket)
  - plannedFor 智能默认:planned 桶 → 今天,inbox / someday 桶 → 空
  - entryType=event 时强制 bucket=planned(与 createQuickTask 对齐)
  - 创建后切到对应 kind 标签(让用户看到新加的事项)
  - 沿用 `pushQuickTemplate` 累计使用频次(模板排序真实反映用户偏好)
  - 5s 撤销 snackbar:onUndo 调 DELETE /tasks/{id} + refreshBoard
- **保留旧编辑流**:点模板主体 = 直接创建,点 ✎ 笔形图标 = 原 `applyQuickTemplate`(填表编辑)
  - 改用 ✎ 图标(非长按 / 双击)的原因:可发现性 > 隐喻,图标在 16px 圆形按钮里统一
  - 一次点击的"编辑流"和"创建流"互不污染
- **数据平滑**
  - 0 后端改动 — `/tasks` POST 接口已存在
  - 0 数据库迁移
  - 0 schema 破坏 — 模板数据结构未变
  - 旧行为(填表)在 ✎ 路径上完整保留,新行为是补充路径
- **UI 设计原则**
  - **减法**:不增加新按钮 / 新页面 / 新设置
  - **对称感**:点击 = 创建 → 与 iter 18/21/24/26 撤销机制完全对称
  - **频次忠实**:用户用得多就靠前,与 iter 11/19 的"系统判断"哲学一致
  - **可发现性**:title 提示文字"点击直接创建「X」· 点 ✎ 编辑 · × 删除",新用户也能秒懂
  - label 微调:"📋 常用" → "📋 常用 · 点击 1 步创建",1 步承诺在 label 里就讲清楚
- **哲学对齐**:"1 步交互最后一公里"在常用模板场景的补完
  - 模板的本质:用户"已经决定"的事物的快捷方式
  - 但旧实现把"已经决定"又加了 1 步确认 = 哲学不一致
  - 模板应该是"1 步出门",不是"1 步把门打开"
  - 沿用 iter 24/26 的拆分逻辑:`extractHintsFromText` 不需要(模板字段已结构化)
  - 沿用 iter 18 撤销模式:5s 内可逆,创建也"可后悔"
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 /tasks POST + showUndoSnackbar + pushQuickTemplate
  - 改动:1 个新函数(applyQuickTemplateDirect) + 1 处 click handler 切换 + 1 个 ✎ 图标 + 1 段 CSS

---

### 阶段 5 减法 · 迭代 28:focus card 1 步完成统一 + 重复任务族可见(消除"主页有 2 套 focus 路径"的不对称)

- **问题**:主页 focus card 有 **2 套不对称路径**:
  - 用户手动 ★ 置入的 focus(activePinnedFocus)→ secondary 1 步完成
  - 系统推荐的 focus(displayFocusTask)→ secondary 1 步打开抽屉
  - 同样的视觉位置、同样"今天先做这件事"的语义,行为却不同,违反 iter 18/22 形成的"统一 1 步"原则
  - 而且 focus card 上"🔁 第 N 次"(iter 20 已加 timeline chip) 在主页不可见,需要翻 timeline 才看到
  - 用户在主页做"写日报"时,**看不到"我已坚持 N 次"**,错失节奏感
- **方案**:focus card 1 步化统一
  - **focus.secondary 统一 1 步完成** (displayFocusTask 路径也用 setTaskStatus → done,5s 撤销兜底)
  - **focus-meta 显示 family round** (复用 parseFamilyRound,activePinnedFocus 和 displayFocusTask 两路都加)
  - 复用 iter 18 undo 兜底 + iter 20 重复任务族 chip
  - focus card 的核心哲学从"建议/查看"升级为"决定/执行"1 步到位
- **数据平滑**
  - 0 后端改动 — setTaskStatus / parseFamilyRound 都已存在
  - 0 数据库迁移
  - 0 schema 破坏 — 复用现有字段
  - 旧行为(打开抽屉)在 timeline 卡片上完整保留;focus card 是新场景
- **UI 设计原则**
  - **对称感**:activePinnedFocus / displayFocusTask 走同一条 1 步完成路径,与 iter 22/24/25/27 形成的"1 步到位"原则一致
  - **节奏感**:🔁 第 N 次从 timeline chip 升级到 focus card,主页一眼看到"我已坚持"
  - **撤销兜底**:5s 内可 1 步撤回完成(iter 18),消除"误点"焦虑
  - **CSS 微调**:`.focus-meta-family` 用渐变背景 + 橙色高亮,与 timeline 的 family-round-chip 视觉一致
- **哲学对齐**:"主页只回答我现在该做什么" 的最后一公里
  - 主页 = "今天最重要的事" + 1 步完成入口,2 个 secondary 也 1 步完成
  - "🔁 第 N 次" 让你做"写日报"时,感受到"我已坚持" = 鼓励感
  - 这是 iter 18/22/25/27 形成的"1 步到位"哲学在主页 dashboard 的对称收口
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 setTaskStatus + parseFamilyRound + showUndoSnackbar
  - 改动:1 处 click handler(openTask → setTaskStatus) + 2 个 focus-meta family span + 1 段 CSS

---

### 阶段 5 减法 · 迭代 29:今日进度合并可视化(消除"今日完成" + "收尾今天" 2 chip 信息割裂)

- **问题**:summary-strip 中 2 个 chip 信息割裂:
  - "今日完成 N" — 只看 done 数
  - "✓ 收尾今天 M" — 只看 open 数
  - 用户要算"还差几件"需要心算 2 个数字
  - 哲学违反:信息密度分散,用户视觉负担 ×2
  - 而且"完成度"概念(0%~100%)不可见,用户没有"我离收尾还差多少"的进度感
- **方案**:2 chip → 1 chip
  - 合并为"今日进度" chip,显示 "今日 X/Y" + 底部 mini progress bar
  - 颜色自适应:
    - 100% 完成 → 绿色 + "已收尾" 文字(庆祝感)
    - 0 完成 → 灰色(冷启动提示)
    - 中间 → 蓝紫渐变 + 进度条
  - 1 步点击 → 1 步收尾(沿用 iter 15 finishTodayAll 行为,带 confirm)
  - 0 件今日时变体:显示 "✨ 今天清爽" (空就等于满)
- **数据平滑**
  - 0 后端改动 — 复用 board.recovery.done_today + todayOpenCount
  - 0 数据库迁移
  - 0 schema 破坏 — 完全用前端已存在的字段计算
  - 旧"今日完成" chip 文字依然存在(在 chip 内部,数值位置不变)
  - 旧"✓ 收尾今天" chip 移除,但其功能(1 步 finishTodayAll)在新 chip 中保留
- **UI 设计原则**
  - **减法**:8 chip → 7 chip,但信息更丰富(完成度 + 文字 + 进度条 3 维)
  - **Visual reward**:完成时进度条 0.35s 动画前进,有"在推进"的爽感
  - **状态自适应**:全完/未完/中间三态各有视觉,与 iter 8 完成的"庆祝动效"哲学一致
  - **"空就等于满"**:0 件今日时显示绿色"今天清爽",避免空状态给用户焦虑
  - **零认知负担**:用户 1 步看见"我完成到哪了",不用心算
- **哲学对齐**:"主页只回答我现在该做什么" + "减法"
  - "我离收尾还差几件" = 主页回答的核心问题之一
  - 1 步可见,1 步完成,与 iter 28 形成的"1 步 dashboard"哲学一致
  - "已收尾"的庆祝感与 iter 8 完成的"今天已经收尾"哲学一致
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 todayOpenCount + board.recovery.done_today + finishTodayAll
  - 改动:1 个新 chip(替换 2 个)+ 2 个 computed(todayTotalCount, todayProgressPercent)+ 1 段 CSS

---

### 阶段 5 减法 · 迭代 30:displayFocusTask "打开" → "开始" 1 步化(消除"点开始"必须先进抽屉的 2 步摩擦)

- **问题**:displayFocusTask 路径的核心动作是"打开"按钮:
  - 用户看到系统推荐的"今天最重要的事" → 心理预期"开始做它"
  - 现有:点"打开" → 跳抽屉 → 用户在抽屉里看到 status 下拉 → 改 in_progress
  - **2 步摩擦**:打开(2 步)反而不是真正的"开始" — "开始"才是用户的真实意图
  - 浪费一次抽屉跳转 + 一次 dropdown 操作
  - 与 activePinnedFocus 路径"✓ 完成"1 步不对称
- **方案**:状态机式 1 步化
  - status=open → "▶ 开始" 按钮(1 步 = in_progress)
  - status=in_progress → "✓ 完成" 按钮(1 步 = done,iter 18 撤销兜底)
  - status=其他 → "查看" 按钮(回退到 drawer,保留 1 步看详情路径)
  - focus card 视觉加 .focus-card-in-progress 状态(蓝色脉动 + ▶ 专注中 chip)
  - 状态机:open → in_progress → done,每步用户主动 1 步选择
  - 完成时刷新,系统自动推下一个 primary
- **数据平滑**
  - 0 后端改动 — setTaskStatus API 已支持 in_progress
  - 0 数据库迁移
  - 0 schema 破坏 — 复用现有 status 状态机
  - 旧"打开"行为在 status=其他 时降级为"查看"按钮(保留 1 步看详情路径)
  - 旧"打开"在 status=open 时被"开始"替代,但用户仍可点 focus 标题进 drawer
- **UI 设计原则**
  - **1 步状态机**:open → 开始 → 完成,每步对应一个明确按钮,与用户的"开始做事"心智对齐
  - **视觉反馈**:in_progress 状态用蓝色脉动 + ▶ 专注中 chip,告诉用户"系统知道你在做这件事"
  - **对称感**:与 activePinnedFocus 的"✓ 完成"1 步路径 + iter 28 的 secondary 1 步完成形成完整对称
  - **回退路径**:status=cancelled 等异常状态降级到"查看"按钮,保留 1 步 detail 入口
  - **撤销兜底**:in_progress 状态可 1 步点回(timeline status cycle),不阻塞用户改主意
- **哲学对齐**:"主页只回答我现在该做什么" + "1 步状态机"
  - 用户打开主页 → 看到"今天最重要的事" → "开始"它 → "完成"它 → 看到下一个
  - 这是 iter 18 (完成 1 步撤销) + iter 28 (focus card 1 步完成) + iter 30 (focus card 1 步开始) 的连续 1 步化
  - "▶ 专注中" chip = 视觉承诺"系统在陪你做",与 iter 19 重复任务"下次自动建到 X"哲学一致
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 setTaskStatus + focus card + status cycle
  - 改动:1 处 focus-card-actions 链(1 按钮 → 3 状态按钮)+ 1 处 class binding(in-progress)+ 1 段 CSS

---

### 阶段 5 减法 · 迭代 31:focus card 标题 1 步 = openTask 详情(消除"看 detail 必须回 timeline" 的 2 步摩擦)

- **问题**:iter 30 把"打开"按钮替换为"▶ 开始" 1 步后,用户在 status=open 路径上**失去了看 detail 的 1 步入口**:
  - 想看 detail → 必须回 timeline 找 primary → 点卡片 (1 步导航 + 1 步 detail = 2 步)
  - 与 timeline 卡片"整卡可点"对称性被破坏(timeline 一气呵成,focus 卡 2 步)
  - focus card 主区域(h3 标题 + p 详情)不可点,用户视觉盲区
- **方案**:focus card 主区域 1 步 = openTask
  - 1 个新 computed `currentFocusTask` (activePinnedFocus.primary || displayFocusTask)
  - focus card 主区域加 click handler + hover 视觉提示
  - hover 时:背景 + 标题下划线(蓝紫色),cursor: pointer
  - 1 步 = openTask → drawer 打开(与 timeline 卡片"整卡可点"对称)
  - 按钮(开始/完成/取消聚焦)独立,不受影响
- **数据平滑**
  - 0 后端改动 — openTask 已存在
  - 0 数据库迁移
  - 0 schema 破坏
  - 旧行为(只能从 timeline 看 detail)被升级,不是删除
- **UI 设计原则**
  - **1 步 dashboard 完整闭环**:看到(focus)→ 看 detail(1 步)→ 开始(1 步)→ 完成(1 步)
  - **可发现性**:hover 视觉 + cursor pointer,用户秒懂"这是可点的"
  - **零破坏**:点击事件不会冒泡到其他区域(focus-meta / focus-secondary / focus-card-actions)
  - **状态无关**:任何 status 都可点(包括 done),给用户"完成后再看一眼"的入口
- **哲学对齐**:主页 focus card = 完整 1 步 dashboard
  - iter 28: focus.secondary 1 步完成
  - iter 30: focus.primary 1 步开始
  - iter 31: focus card 主区域 1 步看 detail
  - 与 timeline 卡片"整卡可点"对称 — 主页与 timeline 同样 1 步可达
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 openTask + currentFocusTask computed
  - 改动:1 处主区域 click handler + 1 个 computed + 1 段 CSS

---

### 阶段 5 减法 · 迭代 32:主页 4 metric cards 全 0 合并减法(从"4 个 0" 变成"1 个 0 都不显示")

- **问题**:iter 5 时代沿用至今的 4 个 metric cards(未完成 / 收件箱 / 事件 / 顺延),当用户**所有维度都清空**时:
  - 视觉上 4 个 0 数字并列,心理学上是"空荡/低气压",**反向施压**用户"快点干点什么"
  - 占用 viewport 横向空间(尤其 mobile),挤压 focus card
  - 4 个 0 等于"在说没东西" — 但用户其实可以"留给自己"
  - "减法"哲学不只是删字段,也要删**视觉空炮**
- **方案**:检测 `allMetricsZero` (openCount + inbox + event_open + rolled_over 全 0) 时
  - 替换 4 个 metric cards 为 **1 个 calm card**(绿色渐变 + ✨)
  - 文案:"今天清爽,留给自己"
  - hover/focus 移除 click 行为(从可点击变成展示型)
  - CSS `grid-column: span 4` 占满 4 列,统一横向视觉
  - 任一维度 > 0 时,回到 4 个可点击 cards(原有交互)
- **数据平滑**
  - 0 后端改动 — 全用现有 board 数据
  - 0 数据库迁移
  - 0 schema 破坏
  - 老数据 0 的用户升级后看到 calm card,无破坏感(本来就 0)
- **UI 设计原则**
  - **减法优于加法**:0 状态不需要 4 个数字,**1 个温暖提示**就够
  - **正反馈替代压力**:0 事项不是"我没干完",是"今天清爽"
  - **状态驱动**:同 4 个 computed(`openCount` / `board.inbox_items.length` / `board.counts.event_open` / `board.counts.rolled_over`)合成 1 个布尔,触发 UI 切换
  - **零新增字段**:0 schema 变化,纯前端状态派生
  - **visual hierarchy**:calm card 绿色渐变 + ✨ 视觉上比 4 个 0 "温暖",不传递焦虑
- **哲学对齐**:主页 4 metric cards → 1 calm card 是阶段 5 减法的最后一公里
  - iter 32 兑现了"主页只回答'我现在该做什么'"
  - 当用户**没什么该做**,**主页不要假装有事**
  - 配合 focus card 1 步化(iter 28-31)和今日进度可视化(iter 29),主页从"指标墙"变成"仪表盘 + 行动中心"
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 hero-metrics 容器 + 4 个原有 computed 派生 1 个布尔
  - 改动:1 个 `allMetricsZero` computed + 1 处模板 v-if/v-else + 1 段 `.metric-card-calm` CSS

---

### 阶段 5 减法 · 迭代 33:focus card 空状态 1 步置入候选(消除"我得回 timeline 找 ★ 才能聚焦"的 3 步导航摩擦)

- **问题**:iter 28-32 完成了 focus card 的"1 步完成 / 1 步开始 / 1 步看 detail" 闭环,但**置入 focus 仍然是 3+ 步**:
  - 看到 focus card 空状态 "点任意事项的 ★ 即可置入今天的聚焦"
  - 必须切换 view(timeline / inbox / planned)
  - 必须找到合适任务
  - 必须点 ★ 按钮
  - 然后滚回主页看效果
  - 对新用户尤其劝退:"我得先去别的地方?那我不做了"
  - 主页本应"告诉我做什么" — 现在主页反而把用户推走
- **方案**:focus card 空状态展示 1 步置入候选
  - 新 computed `focusCandidateTasks`:聚合 今天+明天 planned + 收件箱 → 去重 → 排除已 pinned/done/cancelled → 按优先级 + 创建时间排序 → top 5
  - focus card 空状态(无 currentFocusTask 且非 todayAllDone)显示 1 行候选 chip
  - 每个 chip:`📌 + 标题 + 来源 chip` (今天/明天/收件箱)
  - 1 步点击 → `togglePin(task)` → focus card 立刻变成已置入态
  - **零导航**、零弹窗、零表单
  - chip 颜色按优先级:urgent 红 / high 橙 / medium 蓝 / low 灰(降透明度)
  - hover: 暖色渐变 + 微抬升 + 阴影,提示"可点击"
  - 移动端 100% 宽度堆叠
- **数据平滑**
  - 0 后端改动 — 全用现有 board 数据(groups + inbox_items)
  - 0 数据库迁移
  - 0 schema 破坏
  - 复用 `togglePin` 函数 + `pinnedTaskIds` localStorage
- **UI 设计原则**
  - **1 步导航消除**:用户从"找任务"变成"看候选挑一个",决策从空间(找)变成时间(挑)
  - **让系统判断,只让用户选择**:系统决定哪些候选值得置入(优先级 + 紧迫度),用户只做最后决定(置入谁)
  - **可发现性**:📌 emoji + 暖色 hover + 微动效,用户秒懂"这是可点的置入动作"
  - **渐进式**:5 个候选,信息密度合理,不会让用户"面对满屏不知所措"
  - **状态无关**:用户不进入 drawer / 不切 view / 不表单,纯视觉点击
  - **零破坏**:long instruction 文本("点任意事项的 ★")保留为兜底文案,只在没有候选时显示
- **哲学对齐**:focus card 1 步化完整闭环
  - iter 28: focus.secondary 1 步完成
  - iter 30: focus.primary 1 步开始
  - iter 31: focus card 主区域 1 步看 detail
  - iter 32: 4 metric cards → 1 calm card (主页减法)
  - **iter 33: focus card 1 步置入 (主页入口对称)**
  - 主页"看到→挑一件→看 detail→开始→完成" 全链路 1 步可达,无 view 切换,无 drawer,无表单
  - 兑现了"主页只回答'我现在该做什么'" — 现在主页就是入口
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 `togglePin` + `pinnedTaskIds` + board 数据
  - 改动:1 个 `focusCandidateTasks` computed + 1 处模板 v-else-if(.focus-candidates) + 1 段长 instruction 拆 4 个 v-else-if + 1 段 95 行 CSS

---

### 阶段 5 减法 · 迭代 34:focus.secondary 顶部"✓ 收尾 N 个次要" 1 步批量完成(消除"N 个次要逐条点的 N 步摩擦")

- **问题**:iter 28 把 focus.secondary 改为"1 步完成"是单条粒度,用户有 3 个次要 = 3 次点击 + 3 次确认撤销窗口:
  - 实际场景:周末想"把今天剩下的 3 件小事一次清掉" → 必须点 3 次,每次都弹 5s 撤销
  - 心智成本:用户得"先看一下" 1 个 → 点 → "再看"下一个 → 点,注意力被切碎
  - 与 iter 21 批量撤销已成熟 (snapshot + 5s 撤销) 模式对称
- **方案**:focus.secondary 顶部新增"✓ 收尾 N 个次要" 1 步批量完成按钮
  - 触发条件:secondary.length >= 2(1 个不需要批量,直接点更合适)
  - 1 步点击 → snapshot 全部状态 → 调 batch-update action=mark_done → 弹 5s 撤销 snackbar
  - 撤销走现有 `undoBatchOperation('mark_done', snapshots)` — 与 iter 21 完全对齐
  - 视觉:绿色渐变(bulk = 收尾的"清"感),hover 加深 + 微抬升 + 阴影
  - 移动端 100% 宽度堆叠
  - 复用 `recordCompletionToday()` 计 done_today 统计
  - 覆盖两条 secondary 来源: `activePinnedFocus.secondary`(用户 pin)+ `secondaryFocusTasks`(系统建议)
- **数据平滑**
  - 0 后端改动 — 复用 `batch-update` API + `undoBatchOperation`
  - 0 数据库迁移
  - 0 schema 破坏
  - 撤销是已有路径,与单条 iter 18 撤销完全对称
- **UI 设计原则**
  - **1 步对称**:单条 1 步完成 + N 条 1 步批量完成,用户随便选粒度
  - **可发现性**:绿色渐变 + ✓ emoji + 数字,秒懂"这是批量的"
  - **安全感**:5s 撤销窗口保护,误点也能恢复
  - **克制**:只在 length >= 2 时显示(1 个时直接点更顺手)
  - **零破坏**:原有逐条点击路径不变
- **哲学对齐**:批量 + 撤销是"让系统处理重复,让用户做选择"的延续
  - iter 21:批量操作 1 步撤销(已成熟模式)
  - iter 25:批量改优先级 1 步化
  - iter 34:focus card 场景下的批量完成
  - 用户高频操作"清空今天的次要"从 3+ 步变成 1 步
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 batch-update API + undoBatchOperation + showUndoSnackbar
  - 改动:1 个 `currentSecondaryList` computed + 1 个 `completeAllSecondary` 函数 + 2 处模板 v-if(两个 secondary 块各加 1 按钮)+ 1 段 `.focus-secondary-bulk` CSS

---

### 阶段 5 减法 · 迭代 35:事件卡 body 1 步 = openTask(消除"看事件 detail 必须点"查看""的 2 步摩擦,与 focus card iter 31 完全对称)

- **问题**:iter 31 把 focus card 主区域(h3 + p)改成 1 步 = openTask,但**事件卡 iter 31 时被遗漏**:
  - 想看事件 detail → 必须点右上角"查看"按钮 = 1 步导航 + 1 步 detail = 2 步
  - 事件卡与 focus card 在主页并排,但交互不对称(focus 1 步 / 事件 2 步)
  - 用户视觉预期:两个卡都"主体可点" = 一致性
  - 状态按钮(开始准备 / 已完成 / 取消)独立在下方,不受影响
- **方案**:事件卡主区域 1 步 = openTask
  - 抽出 `.event-card-content` div,加 click handler + hover class
  - 与 focus card iter 31 完全对称(同样的代码结构,同样的 hover 视觉)
  - **色彩区分**:focus card hover 蓝紫色(任务 = 思维感),事件卡 hover 绿色(事件 = 行动/出发感)
  - hover 时:背景色 + 标题下划线
  - 1 步 = openTask → drawer 打开
  - 状态按钮(开始准备/已完成/取消)独立,不受影响
- **数据平滑**
  - 0 后端改动 — openTask 已存在
  - 0 数据库迁移
  - 0 schema 破坏
  - 旧行为(必须点"查看")被升级,不是删除
- **UI 设计原则**
  - **1 步对称**:focus card body 1 步 + 事件卡 body 1 步 = 主页两个卡完全对称
  - **色彩语义**:蓝紫 vs 绿色 hover,色温区分"任务(冷思考)"和"事件(热行动)"
  - **可发现性**:hover 视觉 + cursor pointer,用户秒懂"这是可点的"
  - **零破坏**:点击事件不会冒泡到状态按钮区域
  - **状态无关**:任何 status 都可点(包括 done),给用户"完成后再看一眼"的入口
- **哲学对齐**:主页两个卡完整 1 步闭环 + 视觉对称
  - iter 28: focus.secondary 1 步完成
  - iter 30: focus.primary 1 步开始
  - iter 31: focus card 主区域 1 步看 detail
  - iter 33: focus card 1 步置入候选
  - iter 34: focus.secondary 1 步批量完成
  - **iter 35: 事件卡 body 1 步看 detail (与 iter 31 对称)**
  - 主页 focus card + event card 形成视觉 + 交互双对称
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 openTask
  - 改动:1 处主区域 click handler + 1 段 `.event-card-content-clickable` CSS (与 iter 31 镜像)

---

### 阶段 5 减法 · 迭代 36:focus.secondary "↻ 顺延 N 个到明天" 1 步批量(消除"N 个次要逐条改日期的 N 步摩擦",与 iter 34 批量完成对称)

- **问题**:iter 34 加了 focus.secondary "✓ 收尾 N 个" 1 步批量完成,解决了"批量完成"场景。但用户还有"批量改天"场景:
  - 周末复盘时:"今天做不动了,把 3 个次要都顺延到明天" → 必须点 3 次 "→ 明天" 按钮
  - 实际 N 步(逐条 N 次),N 越大越痛苦
  - 与 iter 34 不对称:完成 1 步批量,但顺延必须 N 步
- **方案**:focus.secondary 顶部新增 "↻ 顺延到明天" 1 步批量按钮
  - 与 iter 34 "✓ 收尾" 并排在 `.focus-secondary-bulk-row` 容器
  - 1 步点击 → 循环 `updateTask` 把每个次要 planned_for 改成明天 → snapshot + 5s 撤销
  - 撤销:把每个 task 改回 snapshot 中的原 planned_for + status
  - 视觉:蓝色调(改天 = 冷思考,蓝色),与 iter 34 绿色调(完成 = 温暖达成,绿色)对比
  - 移动端按钮堆叠 100% 宽度
  - 触发条件:secondary.length >= 2
- **数据平滑**
  - 0 后端改动 — 复用现有 updateTask (PUT /tasks/{id}),循环 N 次
  - 0 数据库迁移
  - 0 schema 破坏
  - 撤销走 individual PUT,与 iter 34 batch API 撤销不完全对称但功能等价
- **UI 设计原则**
  - **1 步对称**:完成 1 步批量(iter 34)+ 顺延 1 步批量(iter 36)→ 用户随便选粒度
  - **色彩语义**:绿色 = 完成(暖色达成)vs 蓝色 = 改天(冷色改期),色温帮助用户秒懂
  - **可发现性**:两个按钮并排,一个绿色 ✓ 一个蓝色 ↻,视觉对比强烈
  - **安全感**:5s 撤销窗口保护,误操作能恢复
  - **克制**:只在 length >= 2 时显示
  - **零破坏**:原有逐条 "→ 明天" 按钮路径不变(timeline 里仍有)
- **哲学对齐**:多粒度批量完成/改期,完全对称
  - iter 21:批量操作 1 步撤销(已成熟模式)
  - iter 25:批量改优先级 1 步化
  - iter 34:focus.secondary 批量完成 (1 步批量 ✓)
  - iter 36:focus.secondary 批量顺延 (1 步批量 ↻)
  - 用户高频"批量处理 focus 次要"从 N 步(N=任务数)变成 1 步
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 updateTask + showUndoSnackbar
  - 改动:1 个 `postponeAllSecondaryToTomorrow` 函数 + 2 处模板 v-if 拆 row 容器 + 1 段 `.focus-secondary-bulk-row` CSS + 1 段 `.focus-secondary-bulk-postpone` CSS

---

### 阶段 5 减法 · 迭代 37:事件卡"📝 记一下" 1 步顺势记录(消除"想给事件记一笔准备情况要打开 drawer 多步摩擦",与 focus card 1 步操作形成完整闭环)

- **问题**:主页事件卡 iter 35 之后,用户可以 1 步看 detail(标题+detail+查看),可以 1 步改状态(开始准备/已完成/取消)。但还有"想给事件记一句话"场景:
  - 会议前 5 分钟:"已经把 deck 翻到第 12 页,主持人刚确认连上" → 必须点"查看"打开 drawer,再展开评论 tab,再点输入框,再打字
  - 实际 4 步摩擦(打开 → 切 tab → 找输入 → 写),N 越大越痛苦
  - 与 focus card 1 步操作不对称:focus 卡的"开始/完成"是 1 步,事件卡的状态按钮是 1 步,但"记一笔"却要 4 步
  - 事件卡应该和 focus card 一样,所有高频操作 1 步可达
- **方案**:事件卡 `.task-actions.compact-actions` 区域新增"📝 记一下" 1 步 popover
  - 复用 QuickCommentPopover(iter 22 的 1 步 popover)→ 点按钮 1 步弹出输入框
  - 1 步写完 Enter 提交 → 0 步离开主页
  - 按钮文案换成「📝 记一下」+ EditPen 图标,比"评论"更顺势(事件轻量场景,不是任务工作流)
  - "评论"是工作流概念(回应/反馈),"记一下"是记录概念(留痕/备忘),贴合事件准备场景
  - 位置:"开始准备 / 已完成 / 📝 记一下 / 取消" 4 个按钮一字排开,语义清晰
  - 触发条件:nextEventTask 存在(已有限制,无需新增)
- **数据平滑**
  - 0 后端改动 — 复用现有 comments API (POST /tasks/{id}/comments)
  - 0 数据库迁移
  - 0 schema 破坏
  - QuickCommentPopover 组件本身不动 1 行,只新增 2 个可选 prop(buttonText/buttonIcon),完全向后兼容(默认 "评论" + ChatDotRound,所有旧调用方零迁移)
- **UI 设计原则**
  - **1 步完整闭环**:focus card 1 步看/改/记,event card 1 步看/改/记,主页两个卡的"操作"维度对称
  - **顺势文案**:「记一下」比「评论」少 1 步心理负担(不用想"我评论什么"),贴合事件临时记录场景
  - **组件复用最大化**:QuickCommentPopover 1 个组件服务 N 个场景(task card 评论 + event card 记一下),通过 prop 适配
  - **零破坏**:其他 5 处 QuickCommentPopover 调用方零修改,默认文案仍为"评论"
- **哲学对齐**:主页两个卡的"操作"维度完全对称
  - iter 30: focus.primary 1 步开始
  - iter 31: focus card body 1 步看 detail
  - iter 33: focus card 1 步置入候选
  - iter 34: focus.secondary 1 步批量完成
  - iter 35: 事件卡 body 1 步看 detail
  - iter 36: focus.secondary 1 步批量顺延
  - iter 37: 事件卡 1 步记一下 (与 iter 22 QuickCommentPopover 1 步 popover 组合)
  - **iter 38: focus 主卡 1 步改明天 (与 iter 36 secondary 批量顺延完全对称)**
  - iter 39: 主页 🚨 紧急 chip 1 步跳紧急 filter (与 timeline filter-strip 双入口对称)
  - **iter 40: focus 主卡 1 步记一下 (与 iter 37 event card 记一下 完全对称)**
  - **iter 41: focus 主卡 1 步取消 (与 event card 取消完全对称)**
  - **iter 42: focus 主卡 1 步"⏸ 暂停" — 补齐 in_progress ↔ open 状态机**
  - **iter 43: focus.secondary 单件 1 步"→ 明天" inline — 与 iter 36 批量顺延形成单/批完整对称**
  - **iter 44: focus.secondary 单件 1 步"✗ 取消" inline + "✗ 取消 N 个" 批量 — 与 iter 41 focus 主卡取消形成单/批完整对称,focus.secondary 改天(蓝)/取消(棕) 状态机 1 步批量完整闭环**
  - **iter 45: 事件卡 1 步"📅 改明天" + "⏸ 暂停" — 与 focus primary iter 38/iter 42 完全对称,事件卡是主页第二显眼位置,补齐状态机 in_progress ↔ open + 改天能力**
  - **iter 46: 顺延回顾 1 步"↻ 顺延到明天" — 补齐顺延决策第三选项(今天做/顺延/不再做了 3 选 1),与 focus.secondary 状态机三色闭环(✓ 绿/↻ 蓝/✗ 棕)完全对称**
  - **iter 47: focus candidate chip "1 步完成" inline — 补齐候选 chip 行内完成能力,与 focus.secondary 单件(→ 明天/✗ 取消 iter 43/44) 行内按钮模式 1:1 对称,候选 1 步决策矩阵(置入/完成)**
  - **iter 48: 事件视图 task-actions 1 步补齐 — ▶ 开始准备 / ⏸ 暂停 / 📅 改明天,与主页事件卡 iter 45 + timeline 任务卡完全对称,事件视图状态机完整闭环**
  - **iter 49: timeline (planned) + tomorrow + inbox + someday 1 步"⏸ 暂停"补齐 — 4 视图状态机完整闭环,与 focus primary iter 42 / 主页事件卡 iter 45 / 事件视图 iter 48 7 处完全对称**
  - **iter 50: Recent view 1 步"↻ 改明天重开" — 复活已完成的旧任务,新函数 reopenRecentToTomorrow 复用 updateTask 路径(0 新增组件)**
  - **iter 51: 主页 🔥 连续 chip 1 步 = 最近 view — 复用 activeView 系统,与 🚨 紧急 iter 39 形成双入口对称,新函数 jumpToStreak 消除"看到连续 N 天想知道最近完成必须手动找 view tab '最近'"的 2 步摩擦;同时修复 setup 阶段 TDZ bug(lowEnergyMode/focusMode 的 watch 在 ref 声明前被访问 → "Cannot access 'mp' before initialization" 白屏),把 watch/toggle 移到 ref 声明之后(2 处 watch 下移,0 新组件)**
  - 主页 focus card 7 步操作 + focus.secondary 单/批 1 步 (完成/顺延/取消) + event card 7 步 + 事件视图 7 步 + timeline/tomorrow/inbox/someday 状态机闭环 + 顺延回顾 3 选 1 + 候选 chip 2 选 1 + recent view ↻ 改明天重开 + 顶部 🚨 紧急 + 🔥 连续 1 步 → 全方位 1 步化
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 QuickCommentPopover
  - 改动:QuickCommentPopover 加 2 个 prop (buttonText/buttonIcon) + 事件卡插入 1 个 QuickCommentPopover 调用

---

### 阶段 5 减法 · 迭代 38:focus 主卡 1 步"📅 改明天"(消除"看 focus card 决定改天必须打开详情改日期"的 4 步摩擦,与 iter 36 focus.secondary 批量顺延完全对称)

- **问题**:iter 36 给 focus.secondary 加了 1 步批量顺延 ↻,但 focus 主卡(无论是 user-pinned activePinnedFocus 还是 system-suggested displayFocusTask)缺 1 步改天能力:
  - 场景:用户看 focus card,决定"这件今天做不动,挪到明天"
  - 原来:点标题打开 detail (iter 31) → 找日期选择器 → 改 → 保存(4 步摩擦)
  - 与 focus.secondary 不对称:secondary 能 1 步批量改天,primary 必须 4 步
  - 与 event card 不对称:event card 已经能 1 步"开始准备/已完成/取消" (iter 35/37),但 focus card "改天"缺失
  - 用户高频场景"今天被突发事件打断,想换日子"重复触达这个摩擦
- **方案**:focus 主卡 actions 区域新增"📅 改明天" 1 步按钮
  - 位置:activePinnedFocus + displayFocusTask + todayAllDone 全部 v-if 链之后,任何有 currentFocusTask 的状态都显示
  - 触发条件:`currentFocusTask` 存在 && status !== 'done' && status !== 'cancelled'
  - 1 步点击 → snapshot 原 planned_for/status → PUT 改到明天 + status=open → 5s 撤销 snackbar
  - 撤销:恢复原 planned_for + 原 status
  - 视觉:蓝紫渐变 (与 focus.secondary iter 36 ↻ 顺延到明天 完全同色系,色温统一)
  - 按钮文案:"📅 改明天" — "改" 强调主动决定,"明天" 是最常见目标,5 字内极简
  - 移动端 flex-wrap 自动换行(`.focus-card-actions` 已有 flex-wrap: wrap)
- **数据平滑**
  - 0 后端改动 — 复用现有 PUT /tasks/{id},与 iter 36 同 API
  - 0 数据库迁移
  - 0 schema 破坏
  - 撤销走 individual PUT,功能与 iter 36 完全等价
  - 现有"取消聚焦"/"✓ 完成"/"▶ 开始"/"查看"路径不变
- **UI 设计原则**
  - **1 步对称**:focus.secondary 批量改天 (iter 36) + focus.primary 单件改天 (iter 38) → focus card 改天维度 100% 覆盖
  - **色彩语义统一**:蓝色 = 改天 (冷思考/改期),绿色 = 完成 (暖色/达成),色温形成全站共识
  - **可发现性**:与"✓ 完成"按钮并排在 actions 区,色温对比强烈,用户秒懂"做"vs"改天"
  - **安全感**:5s 撤销窗口保护,误操作能恢复(沿用 iter 18 + iter 21 成熟模式)
  - **零破坏**:已 done/cancelled 的 focus 不显示按钮(用户已经处理完,不需要改天)
  - **任意入口**:activePinnedFocus 和 displayFocusTask 两种 focus 来源都生效,与"看/做"按钮完全独立
- **哲学对齐**:主页 focus card 改天维度补齐,与 secondary 改天对称
  - iter 18: 完成 1 步撤回(模式)
  - iter 21: 批量操作 1 步撤销(模式)
  - iter 36: focus.secondary 1 步批量顺延 ↻
  - **iter 38: focus.primary 1 步改天 📅**(单件,与 iter 36 批量对称)
  - 用户从"看 focus card 决定改天"从 4 步摩擦降到 1 步
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件
  - 改动:1 个 `postponeFocusPrimaryToTomorrow` 函数 + 1 处 focus card actions 模板新增按钮 + 1 段 `.focus-primary-postpone-btn` CSS (与 iter 36 同色系)

---

### 阶段 5 减法 · 迭代 39:主页顶部 🚨 紧急 chip 1 步 = 时间线 + 紧急 filter(消除"看到紧急 N 件 → 想看这 N 件必须滚动 / 切 view / 用搜索"的 2 步摩擦,与 timeline filter-strip 双入口对称)

- **问题**:主页顶部 summary strip 有 4 个 chip 已经是 1 步可点(明天 / 低能量 / 专注 / 模式),但 3 个数据 chip(🚨 紧急 / 今日进度 / 🔥 连续)只是展示,没动作。最痛的是 🚨 紧急:
  - 用户每天进主页,看到 "🚨 紧急 3 件" → 想看这 3 件是什么
  - 原来:必须切到 timeline view (1 步) + 点 timeline 顶部 "🚨 紧急" filter chip (第 2 步) = 2 步
  - 与已 1 步的 明天/低能量/专注/模式 不对称
  - 紧急是最常用 filter 意图(高优先级),2 步摩擦每天触达
- **方案**:主页 🚨 紧急 chip 加 1 步 click handler
  - 点击 → `activeView = 'timeline'` + `priorityFilter = 'urgent'` + localStorage 持久化
  - 复用已有 `setPriorityFilter` 系统(只是函数更直接,因为主页不能 toggle 只能 jump)
  - 复用已有 timeline filter-strip,主页 chip + timeline 顶部 chip 形成"双入口"对称
  - 无紧急事项时不可点(urgentOpenCount=0 直接 return)
  - 已在紧急 filter 时再次点击保持不变(避免抖动,想切回全部用 timeline 顶部 "全部" chip)
  - active 状态:正在查看紧急 filter 时,chip 加红环高亮,告知"已激活"
- **数据平滑**
  - 0 后端改动
  - 0 数据库迁移
  - 0 schema 破坏
  - 0 新组件
  - 复用已有 priorityFilter ref + localStorage 键 + activeView ref,完全无破坏
- **UI 设计原则**
  - **双入口对称**:主页 chip + timeline 顶部 chip 都是"🚨 紧急 N 件",同色同 label,标准 UX 模式(macOS dock/menu 风格)
  - **可发现性**:`cursor: pointer` + `transform: translateY(-1px)` hover 反馈,用户秒懂"这是可点的"
  - **active 反馈**:激活时加 3px 红色 inset box-shadow + 红色外发光,提示"已选中此 filter"
  - **空状态自然**:无紧急时 chip 变灰,自动不可点(无需 if 判断,纯样式)
  - **零破坏**:旧的 timeline filter-strip 工作方式不变
- **哲学对齐**:主页数据 chip 从展示升级为入口,与"主页只回答我现在该做什么"一致
  - 用户看到紧急 N → 1 步直达紧急列表,中间没有任何"我要去哪"的心智摩擦
  - 主页信息密度自适应:紧急多时高亮 + 1 步可达,紧急少时融入背景
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件
  - 改动:1 个 `jumpToUrgent` 函数 + 1 处 summary-strip 模板 click handler + 1 段 `.summary-chip-clickable` / `.summary-chip-urgent-active` CSS

---

### 阶段 5 减法 · 迭代 40:focus card 1 步"📝 记一下"(消除"focus 主卡想记一笔进展必须打开 detail 写评论"的 3 步摩擦,与 event card iter 37 完全对称)

- **问题**:iter 37 给 event card 加了"📝 记一下" 1 步 popover,让用户在主页能 1 步记下事件准备情况。但 focus card 仍缺 1 步记一下能力:
  - 场景:用户开始做 focus primary,想记一句话("卡在 X 处" / "决定先做 A 再做 B" / "已做 30 分钟进展 Y")
  - 原来:点标题打开 detail (iter 31) → 切评论 tab → 找输入框 → 写 → 保存(3 步摩擦)
  - 与 event card 不对称:event card 已 1 步记一下 (iter 37),focus card 必须 3 步
  - 与 timeline 卡片不对称:timeline 卡片每条都有 QuickCommentPopover,focus 主卡没有
  - 用户高频场景"做 focus 任务时想随手记一笔"重复触达
- **方案**:focus card actions 区域新增"📝 记一下" 1 步 popover
  - 复用 iter 37 已加的 buttonText + buttonIcon prop(为 event card 适配而设,正好复用)
  - 触发条件:`currentFocusTask` 存在 && status !== 'done' && status !== 'cancelled'
  - 1 步点击 → popover 弹出 → 输入 → Enter 提交 → 0 步离开主页
  - 与 iter 38"📅 改明天"并列,形成"做/改/记"三选一
  - 视觉:沿用 QuickCommentPopover 既有样式(蓝色/绿色按钮自然融合 focus card 风格)
- **数据平滑**
  - 0 后端改动 — 复用现有 POST /tasks/{id}/comments
  - 0 数据库迁移
  - 0 schema 破坏
  - QuickCommentPopover 组件本身 1 行未改(迭代 37 已加好 buttonText/buttonIcon prop)
  - 其他 5 处 QuickCommentPopover 调用方零修改,默认"评论"+ ChatDotRound 完全兼容
- **UI 设计原则**
  - **1 步对称**:focus card 看(31)/做(28,30)/记(iter 40)/改(38) + event card 看(35)/做/记(37)/取消
  - **两个卡 1 步操作 100% 完整覆盖**:主页两卡在任何状态下都能 1 步完成任何高频操作
  - **复用最大化**:QuickCommentPopover 1 个组件服务 N 个场景,task card 评论 / focus card 记一下 / event card 记一下
  - **零破坏**:已 done/cancelled 的 focus 不显示按钮(用户已处理完,不需要记)
  - **按钮文案一致**:"📝 记一下" + EditPen 图标,event card 和 focus card 视觉完全一致
- **哲学对齐**:主页两个卡 1 步操作完整闭环
  - iter 22: 评论 1 步化 (QuickCommentPopover 组件诞生)
  - iter 37: 事件卡"📝 记一下" + QuickCommentPopover 加 buttonText/buttonIcon prop
  - **iter 40: focus 主卡"📝 记一下"** (复用 iter 37 的 prop,组件本身不动)
  - 主页 focus card 4 步操作 + event card 4 步操作 全部 1 步可达,完整对称
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 QuickCommentPopover(0 改动)
  - 改动:1 处 focus card actions 模板新增 1 个 QuickCommentPopover 调用

---

### 阶段 5 减法 · 迭代 41:focus card 1 步"❌ 取消"(消除"focus 卡决定不做了必须回 timeline 找卡点取消"的 3 步摩擦,与 event card 取消完全对称)

- **问题**:iter 35 给 event card 加了 1 步取消 (CancelReasonPopover),但 focus card 仍缺 1 步取消能力:
  - 场景:用户决定 focus primary 今天不做了("时机不对"/"优先级变了"/"今天状态不适合做")
  - 原来:回 timeline → 找卡 → 点"取消"按钮 → 选原因 (3 步摩擦,还容易在 timeline 里分心看其他事项)
  - 与 event card 不对称:event card 已 1 步取消 (iter 35),focus card 必须 3 步
  - 与 timeline 卡片对称:timeline 卡片每条都有 1 步取消,但 focus 主卡没有
  - 用户高频"放弃 focus 任务"场景重复触达 3 步摩擦
- **方案**:focus card actions 区域新增"❌ 取消" 1 步 popover
  - 复用 iter 13 CancelReasonPopover(选 chip = 1 步,Enter 提交,5s 撤销)
  - 触发条件:`currentFocusTask` 存在 && status !== 'done' && status !== 'cancelled'
  - 1 步点击 → popover 弹出 → 选原因 chip → 提交,5s 内可撤销
  - 视觉:plain 灰按钮(取消是 destructive 但非主要操作,de-emphasize 避免误点)
  - 位置:最右,远离"✓ 完成"主操作,符合 destructive 按钮放边缘的 UX 共识
  - 文案:"❌ 取消" — 红色 ✗ 暗示 destructive 性质
- **数据平滑**
  - 0 后端改动 — 复用现有 cancelTaskWithReason(后端走 status='cancelled' + reason)
  - 0 数据库迁移
  - 0 schema 破坏
  - CancelReasonPopover 组件 1 行未改
- **UI 设计原则**
  - **1 步对称**:focus card 看(31)/做(28,30)/记(40)/改(38)/取消(41) + event card 看(35)/做/记(37)/取消(35)
  - **destructive 按钮放最右**:远离主操作(✓ 完成 / ▶ 开始),符合 Material Design / iOS HIG 的 destructive UX 共识
  - **色彩降调**:plain 灰按钮而非红色,避免情绪化暗示"取消 = 失败"
  - **5s 安全感**:撤销 snackbar 保护,误点可恢复(沿用 iter 13 + iter 18 成熟模式)
  - **零破坏**:已 done/cancelled 的 focus 不显示按钮(用户已处理完,不需要取消)
  - **任意入口**:activePinnedFocus + displayFocusTask 两种 focus 来源都生效
- **哲学对齐**:主页两个卡 5 步操作完整对称
  - iter 13: 取消 1 步化 (CancelReasonPopover 组件诞生)
  - iter 35: 事件卡 1 步取消
  - **iter 41: focus 主卡 1 步取消** (复用 iter 13 组件,0 改动)
  - 主页 focus card 5 步操作 + event card 4 步操作 全部 1 步可达,完整闭环
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 CancelReasonPopover(0 改动)
  - 改动:1 处 focus card actions 模板新增 1 个 CancelReasonPopover 调用

---

### 阶段 5 减法 · 迭代 42:focus 主卡 1 步"⏸ 暂停" — 补齐状态机 in_progress ↔ open 的 1 步入口(消除"中途被打断想暂停必须点 查看 → 改状态"的 2 步摩擦,完成 ▶ 开始 ↔ ⏸ 暂停 ↔ ✓ 完成 3 步状态机完整闭环)

- **问题**:iter 30 给 focus card 加了"▶ 开始" 1 步(进入 in_progress),iter 28 加了"✓ 完成" 1 步(进入 done)。但状态机中间断了:
  - 之前状态机:open → ▶ 开始 → in_progress → ✓ 完成 → done(单向)
  - 如果用户开始做 focus primary,中途被打断(电话/会议/突发),想"暂停回到 open,稍后再做"
  - 原来:点 查看 → 在 detail 抽屉里找状态按钮 → 改回 open (2 步摩擦,还容易分心)
  - 之前没有"⏸ 暂停"按钮,状态机缺 in_progress → open 这一环
- **方案**:focus card actions 在 status === 'in_progress' 时显示"⏸ 暂停"按钮
  - 1 步点击 → setTaskStatus(task, 'open')(已有函数,0 改动)
  - 触发条件:`currentFocusTask && currentFocusTask.status === 'in_progress'`
  - 状态机完整闭环:open ↔ in_progress ↔ done(任意方向 1 步可达)
  - 与 activePinnedFocus / displayFocusTask 两个 focus 来源都生效
- **数据平滑**
  - 0 后端改动 — 复用现有 setTaskStatus(task, 'open')
  - 0 数据库迁移
  - 0 schema 破坏
  - 0 新组件
  - 现有 ▶ 开始 / ✓ 完成 / ❌ 取消 / 📅 改明天 / 📝 记一下 全部不破坏
- **UI 设计原则**
  - **状态机对称**:▶ 开始(蓝)/ ⏸ 暂停(灰)/ ✓ 完成(绿) — 视觉动作光谱覆盖 3 个状态转换
  - **plain 灰按钮**:中性,与 ▶ 开始 蓝色 / ✓ 完成 绿色 形成对比
  - **位置**:"✓ 完成" 之后,自然顺序:完成 → 暂停 → 改天 → 记一下 → 取消
  - **条件显示**:只在 in_progress 时出现,不会污染 open 状态的简洁
  - **零破坏**:open / done / cancelled 状态完全不显示,新按钮不挤占已有空间
- **哲学对齐**:状态机完整闭环,任何 1 步可达
  - iter 28: focus card 1 步完成统一
  - iter 30: displayFocusTask "▶ 开始" 1 步化
  - **iter 42: focus card "⏸ 暂停" 1 步** (补齐状态机)
  - 主页状态机:open ↔ in_progress ↔ done(任意方向 1 步可达) + 任意状态可改天/记一下/取消
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件
  - 改动:1 处 focus card actions 模板新增 1 个 el-button 条件渲染

---

### 阶段 5 减法 · 迭代 43:focus.secondary 单件 1 步"→ 明天" inline(消除"想改 1 个次要到明天,必须回 timeline 找卡点"的 3 步摩擦,与 iter 36 批量顺延形成单/批完整对称)

- **问题**:iter 36 给 focus.secondary 加了 1 步批量顺延 ↻(2+ 个时顶部出现),解决了"批量改天"场景。但用户还有"单件改天"场景:
  - 场景:"3 个次要里,只想把第 2 个改到明天,其他不变"
  - 原来:回 timeline → 找该卡 → 点"→ 明天"按钮(3 步摩擦,还容易分心看其他卡)
  - 与 iter 36 不对称:批量能 1 步改天,单件必须 3 步
  - 与 timeline 卡片对称:timeline 每条都有 1 步"→ 明天",focus.secondary 没有
  - 用户高频"精准改天"场景重复触达
- **方案**:focus.secondary 每个 item 右侧加 inline "→ 明天" 小按钮
  - 复用现有 `postponeToTomorrow` 函数(已有,0 新增)
  - 触发:每个 item 右侧蓝色小按钮,@click.stop 防止冒泡触发父级完成
  - 视觉:蓝色调(冷思考/改天),与 iter 36/38 顺延色系一致
  - 位置:item 右侧,小尺寸,避免抢主操作"点击完成"区视觉焦点
  - 改写 .focus-secondary-item 结构:从 button → button 嵌入 flex row,内含"点击完成"区 + "→ 明天"按钮
- **数据平滑**
  - 0 后端改动 — 复用现有 PUT /tasks/{id}
  - 0 数据库迁移
  - 0 schema 破坏
  - 0 新组件
  - 复用现有 postponeToTomorrow 函数
- **UI 设计原则**
  - **1 步对称**:批量 (iter 36 ↻) + 单件 (iter 43 → 明天) → focus.secondary 改天维度 100% 覆盖
  - **色彩语义统一**:蓝色 = 改天,与 iter 36/38 同色系
  - **零破坏**:原来"点击 item 完成"的 1 步行为完全保留(只是改成 clickable 区)
  - **视觉降调**:小尺寸,避免与主操作竞争
  - **冒泡防护**:@click.stop 防止按"→ 明天"时同时触发 item 完成
- **哲学对齐**:focus.secondary 改天维度单/批 1 步完整对称
  - iter 36: focus.secondary 批量顺延 ↻(顶部按钮,2+ 时显示)
  - **iter 43: focus.secondary 单件改天 → 明天**(行内按钮,1+ 时显示)
  - 用户从"想改 1 个次要到明天"从 3 步摩擦降到 1 步
  - 与 iter 38 focus.primary 单件改天 📅 + iter 36 secondary 批量改天 ↻ 形成完整单/批矩阵

### iter 44: focus.secondary 单件 1 步"✗ 取消" inline + 批量"✗ 取消 N 个"

- **问题**:
  - 用户场景:"今天这堆次要(3-5 个)都不做了/条件变化了",需要逐个取消
  - 原来:每个次要 → 回 timeline 找卡 → 点 ⋯ → 选取消原因 chip = **N × 3 步**,还容易分心
  - 优化:主页直接点 secondary 卡右侧 ✗ → 选原因 → 1 步;或顶部"✗ 取消 N 个" 1 步批量
- **方案**:
  1. **单件 inline 取消**:在 focus.secondary item-row(已有"→ 明天"按钮)再加一个 `CancelReasonPopover` 用 `iconOnly` 模式,只显示 ✗ icon
  2. **批量取消**:在已有 ✓ 收尾 + ↻ 顺延 旁边加"✗ 取消 N 个"按钮,3 个按钮并排构成 1 步批量状态机完整闭环
  3. **零新组件**:`CancelReasonPopover` 已有(iter 13),仅加 `iconOnly` prop(0 新增)
  4. **零新函数复用**:单件复用 `cancelTaskWithReason`(iter 13),批量仅新增 `cancelAllSecondary`(用现有 updateTask 路径循环,snapshot+5s 撤销兜底)
- **场景**:
  - 单件:用户看 focus.secondary,想"这件我不再需要/条件变化",点右侧 ✗ → 选 chip → 1 步
  - 批量:用户看 focus.secondary,"今天这 3 个我都不做了",点顶部"✗ 取消 3 个" → 1 步
- **视觉**:
  - icon-only 按钮:`padding 4px 8px`、`border-radius 14px`、与"→ 明天"同高同 padding,视觉一致
  - 批量按钮:棕色调(取消/不再做),与 ✓ 收尾(绿)/ ↻ 顺延(蓝) 形成三色状态机
  - 改天(蓝)/ 取消(棕) 都是"今天不做了"的两个分支决策
- **冒泡防护**:@click.stop 防止按 ✗ 时同时触发 item 完成
- **哲学对齐**:focus.secondary 状态机 1 步批量完整闭环
  - ✓ 收尾(iter 34, 暖色绿) - 完成
  - ↻ 顺延(iter 36, 冷色蓝) - 改天
  - ✗ 取消(iter 44, 棕色调) - 不再需要
  - 三色对应用户对 1 个次要的 3 个主要决策,主页 1 步直达,无需回 timeline
- **单/批对称**:
  - 批量(顶部):iter 34 ✓ + iter 36 ↻ + iter 44 ✗
  - 单件(行内):iter 43 → 明天 + iter 44 ✗(icon-only)
  - 单/批 1 步矩阵在 focus.secondary 区域 100% 覆盖
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 CancelReasonPopover(加 1 个 prop iconOnly)
  - 改动:2 处 v-for 模板各加 1 个 iconOnly popover + 2 处 bulk row 各加 1 个 ✗ 取消按钮 + 1 段 CSS (icon-only 模式 + bulk cancel 色系) + 新增 `cancelAllSecondary` 函数(snapshot+undo 模式复用 iter 36/38)

### iter 45: 事件卡 1 步"📅 改明天" + "⏸ 暂停" — 与 focus primary iter 38/42 完全对称

- **问题**:iter 38 给 focus primary 加了 1 步"📅 改明天",iter 42 加了 1 步"⏸ 暂停",但事件卡(主页第二显眼位置)漏了对称处理
  - 场景 A:用户看到 3 点的会要推迟(临时有事/客户改期)
    - 原来:点 查看 → 找日期选择器 → 改 → 保存(4 步摩擦)
    - 现在:主页事件卡直接点"📅 改明天"→ 1 步完成 → 5s 撤销
  - 场景 B:用户开始准备会议(开始准备 → in_progress),中途被打断(老板喊走/电话/急事)
    - 原来:点 ⋯ → 标记为待办(2 步摩擦,且 ⋯ 在 timeline 才有,事件卡根本没 ⋯ 菜单)
    - 现在:主页事件卡直接点"⏸ 暂停"→ 1 步回到 open → 5s 撤销
- **方案**:
  1. **📅 改明天**:复用 `postponeToTomorrow(task)`(已有函数,0 新增);`v-if` 限定 status !== done/cancelled
  2. **⏸ 暂停**:复用 `setTaskStatus(task, 'open')`(已有函数,0 新增);`v-if` 限定 status === 'in_progress'
  3. **视觉**:与 focus primary 完全一致(蓝色改天/灰色暂停),统一动作光谱
  4. **0 新组件/0 新函数** — 纯模板插入 2 个 el-button
- **场景矩阵**(事件卡 1 步状态机完整闭环):
  - open → ▶ 开始准备 (in_progress) — 已有
  - in_progress → ⏸ 暂停 (open) — **iter 45 新增**
  - in_progress → ✓ 已完成 (done) — 已有
  - open → ✓ 已完成 (done) — 已有
  - 任意状态 → 📅 改明天 (明天) — **iter 45 新增**
  - 任意状态 → 📝 记一下 (评论) — iter 37
  - 任意状态 → ❌ 取消 (cancelled) — iter 35
- **哲学对齐**:主页"看" "做" "改" "记" "取消" 5 维 1 步化,focus primary ↔ event card 完全对称
  - focus primary 看(iter 31)/做(28,30)/改(iter 38)/记(iter 40)/取消(iter 41)/暂停(iter 42)
  - event card 看(iter 35)/做/改(iter 45)/记(iter 37)/取消/暂停(iter 45)
  - 主页两张卡 6 维能力 1:1 对称,任何"想做的事"都不需要回 timeline 找
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件/0 新函数 — 纯模板插入 2 个 el-button
  - 改动:事件卡 task-actions 块加 2 个 v-if 限定的 el-button(分别复用 postponeToTomorrow 和 setTaskStatus),复用 .focus-primary-postpone-btn class(已有)

### iter 46: 顺延回顾 1 步"↻ 顺延到明天" — 补齐顺延决策第三选项(3 选 1 决策矩阵)

- **问题**:主页"🔁 顺延回顾"卡片是给已顺延 ≥2 次的事做的"重新决定"提醒,目前只 2 选 1:
  - 📅 今天做(`rescheduleRolledOverToday`)
  - 不再做了(`cancelRolledOver`)
  - **缺失**:↻ 顺延到明天 — 用户最常见的"再给我 1 天"决策
- **场景**:
  - 用户场景:看到 1 件已顺延 3 次的事,"今天还是没空/状态不好,先挪到明天再说"
  - 原来决策路径 A:点 📅 今天做 → 进 task → 改日期(4-5 步摩擦,容易分心去看别的卡)
  - 原来决策路径 B:什么都不点,卡片关闭,明天还是没空,继续顺延(没解决决策)
  - 现在:主页直接点 ↻ 顺延到明天 → 1 步完成 → 5s 撤销
- **方案**:
  1. **新增函数** `postponeRolledOverToTomorrow(task)`:复用 `updateTask` 路径(已有),`planned_for: tomorrow, status: 'open', postpone_reason: '顺延回顾 1 步顺延到明天'`
  2. **新增按钮** 在 rollover-insight-actions 块 2 按钮中间插入 ↻ 顺延明天
  3. **视觉**:蓝色调(冷思考/改期)与 focus.secondary 改天 / focus primary 改明天 色系一致
  4. **决策矩阵 3 选 1**:
     - 今天做(主操作,绿色 → 暖色)
     - 顺延明天(中性,蓝色 → 冷色思考)
     - 不再做了(放弃,棕色 → 严肃)
- **哲学对齐**:与 focus.secondary 状态机三色闭环完全对称
  - focus.secondary:✓ 收尾(绿)/ ↻ 顺延(蓝)/ ✗ 取消(棕)
  - 顺延回顾:📅 今天做(绿)/ ↻ 顺延明天(蓝)/ ✗ 不再做了(棕)
  - 三色 = 三种决策(推进/推迟/放弃),色温帮助用户秒懂
  - "改天" (蓝) 跨越多个场景:secondary 单/批 / focus primary / event card / **顺延回顾**
  - 蓝调一致性让用户在任何地方看到蓝色按钮都知道"这是推迟/改期"
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动 — 复用 `updateTask` 路径
  - 0 新组件
  - 改动:1 个新函数 `postponeRolledOverToTomorrow` + 1 个 el-button 插入 + 1 段 CSS (.rollover-postpone-btn 蓝色调)

### iter 47: focus candidate chip "1 步完成" inline — 补齐候选 chip 行内完成能力

- **问题**:iter 33 加了 focus candidate chip,让用户从主页 1 步置入聚焦。但实际场景:
  - 用户看到候选 1 件(收件箱里的简单任务/优先级低的小事)
  - 实际不需要置入聚焦,直接做掉就行
  - 原来:点 chip → pin 置入 focus → 主页出现该 focus → 点"✓ 完成" = **2 步摩擦**
  - 而且 pin 置入会污染今天的 focus(把小事也变成"今天最重要的")
- **方案**:
  1. **行内 ✓ 按钮**:focus candidate chip 行内最右加紧凑圆形 ✓ 按钮
  2. **row 容器**:chip 包在 `.focus-candidate-row` 容器里(与 focus.secondary item-row iter 43/44 模式 1:1 对称)
  3. **复用 setTaskStatus(task, 'done')**(已有函数,0 新增)
  4. **视觉**:绿色调(暖色/完成)与 focus.secondary 收尾色系一致;32×32 圆形按钮,避免抢主操作"置入"区视觉焦点
  5. **@click.stop** 防止冒泡到 chip 主点击(pin 置入)
- **场景**:
  - 收件箱里 1 件"买菜"小事 → 不需要置入聚焦(污染今天),直接 chip 行内 ✓ → 1 步完成
  - 候选里 1 件"回邮件"小事 → 同上
  - 重要的事仍走主点击(置入聚焦)
- **决策矩阵**(2 选 1):
  - 置入(主点击) — 大事/重要/需要专注的事
  - ✓ 完成(行内) — 小事/简单/不需要聚焦的事
  - 用户根据事项性质 1 步直达对应出口
- **哲学对齐**:与 focus.secondary 单件改天(iter 43) / 取消(iter 44) 行内按钮模式 1:1 对称
  - focus.secondary:点击完成(主) + → 明天(行内) + ✗(行内) = 3 选 1
  - focus candidate:置入(主) + ✓(行内) = 2 选 1
  - 行内按钮模式统一:任何"行内次操作"都是蓝色/绿色小按钮,@click.stop 防止冒泡
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动 — 复用 `setTaskStatus` 路径
  - 0 新组件
  - 改动:1 个 v-for 包 row 容器 + 1 个行内 ✓ button + 1 段 CSS (.focus-candidate-row + .focus-candidate-complete 绿色调)

### iter 48: 事件视图 task-actions 1 步补齐 — 与主页事件卡 iter 45 + timeline 任务卡完全对称

- **问题**:事件视图 (events view) task-actions 块缺 3 个 1 步能力,导致与主页事件卡 (iter 45) + timeline 任务卡 (planned) 不对称
  - ▶ 开始准备(在 ⋯ 菜单的"标记为进行中"里 = 2 步摩擦)
  - ⏸ 暂停(根本就没这选项,只能 ⋯ → 别的 = 2+ 步)
  - 📅 改明天(根本就没这选项,必须 ⋯ → 顺延…模态 = 3 步摩擦)
- **场景**:
  - 场景 A:用户在事件视图看完整事件列表,看到下午 3 点的会想"我开始准备" → 现在 ⋯ 菜单 = 2 步
  - 场景 B:用户已开始准备 1 个事件(状态 in_progress),被打断想"暂停回到 open" → 现在根本做不到
  - 场景 C:用户看到下周的会要推迟 → 现在 ⋯ 菜单 → 顺延…模态 = 3 步
- **方案**:
  1. **▶ 开始准备** el-button(plain 灰,v-if status === 'open')— 1 步进入 in_progress
  2. **⏸ 暂停** el-button(plain 灰,v-if status === 'in_progress')— 1 步回到 open
  3. **📅 改明天** el-button(蓝色调,class="focus-primary-postpone-btn",v-if status !== done/cancelled)— 1 步改到明天
  4. **复用** setTaskStatus + postponeToTomorrow(已有函数,0 新增)
  5. **清理 ⋯ 菜单**:移除"标记为进行中"(已升级为 1 步按钮,无意义)
- **场景矩阵**(事件视图 1 步状态机完整闭环):
  - open → ▶ 开始准备(in_progress)— **iter 48 新增**
  - in_progress → ⏸ 暂停(open)— **iter 48 新增**
  - in_progress → ✓ 完成(done)— 已有
  - open → ✓ 完成(done)— 已有
  - 任意状态 → 📅 改明天(明天)— **iter 48 新增**
  - 任意状态 → 📝 记一下(评论)— 已有
  - 任意状态 → ❌ 取消(cancelled)— 已有
- **哲学对齐**:主页 + 事件视图 + timeline 任务卡 3 处事件 1 步能力 1:1 对称
  - 主页事件卡:看(35)/做/改(45)/记(37)/取消/暂停(45)
  - 事件视图:**iter 48 补齐后** 看/做(48)/改(48)/记/取消/暂停(48) — 与主页事件卡完全对称
  - timeline 任务卡:看/做/改(→ 明天)/记/取消/暂停(planned 已存在,其余稍后)
  - 任何"想做的事"在 3 个地方都能 1 步直达,无需 ⋯ 菜单
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件/0 新函数 — 纯模板插入 3 个 el-button + 1 个 ⋯ 菜单项清理
  - 改动:事件视图 task-actions 块加 3 个 v-if 限定的 el-button,复用 .focus-primary-postpone-btn class(已有)

### iter 49: timeline (planned) + tomorrow + inbox + someday 1 步"⏸ 暂停"补齐 — 4 视图状态机完整闭环

- **问题**:iter 42/45/48 给 focus primary / 主页事件卡 / 事件视图加了 1 步"⏸ 暂停"按钮,但 4 个核心视图 task-actions 块缺这个能力:
  - **timeline (planned)**:in_progress 任务只能 ⋯ 菜单 = 2 步摩擦(且 ⋯ 没"暂停"选项)
  - **tomorrow**:"进行中"按钮一直显示,in_progress 任务点它无效果(仍在 in_progress)— 状态机 bug
  - **inbox**:完全没有"进行中/暂停"入口
  - **someday**:完全没有"进行中/暂停"入口
  - 用户场景:用户标记某件大事为"进行中"后被打断,想暂停回到 open → 4 个视图都做不到 1 步
- **方案**:
  1. **timeline (planned)** line 1217:加 ⏸ 暂停 el-button(v-if in_progress)
  2. **tomorrow** line 1364:加 v-if status === 'open' 限定"进行中"显示,加 ⏸ 暂停 el-button(v-if in_progress)
  3. **inbox** line 1718:加 v-if 限定"进行中"显示,加 ⏸ 暂停 el-button
  4. **someday** line 1795:加 v-if 限定"进行中"显示,加 ⏸ 暂停 el-button
  5. **复用** setTaskStatus(task, 'open')(已有函数,0 新增)
  6. **视觉**:与 focus primary / 主页事件卡 / 事件视图 完全一致(plain 灰按钮)
- **场景矩阵**(7 处 1 步状态机完整闭环):
  - focus primary(iter 42)⏸ / 主页事件卡(iter 45)⏸ / 事件视图(iter 48)⏸ / **iter 49** timeline+tomorrow+inbox+someday ⏸
  - 任意位置的 in_progress 任务都能 1 步暂停
- **哲学对齐**:状态机 1 步可达是用户体验底线
  - open → in_progress → done(完成 1 步)→ open(重新打开 1 步)
  - in_progress → open(暂停 1 步)— **iter 49 补齐 4 视图**
  - 用户标记 in_progress 后被打断,不需要 ⋯ 菜单,不需要 2 步流程
  - 7 处对称确保用户在任何视图都有一致体验
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件/0 新函数 — 纯模板插入 4 个 ⏸ 暂停 el-button + 2 个 v-if 限定"进行中"按钮显示
  - 改动:4 个视图 task-actions 块各加 1 个 v-if 限定的 el-button,2 个视图加 v-if 限定"进行中"显示时机

### iter 50: Recent view 1 步"↻ 改明天重开" — 复活已完成的旧任务,新函数 reopenRecentToTomorrow 复用 updateTask 路径(0 新增组件)

- **问题**:Recent view(最近 7 天已完成/已取消的回顾池)是用户回看"最近搞定什么 / 哪些放弃了"的地方,但 1 步能力缺最后一公里:
  - 用户场景 A:看到 3 天前取消的一件小事,突然想"其实可以改天做,只是当时觉得不该做" → 必须 ⋯ 菜单 = 2 步 + 还要改日期
  - 用户场景 B:看到昨天完成的"准备 X 资料",意识到遗漏("需要再补一份") → 没法 1 步复活,只能手动新建任务
  - 与完成/取消对称:Recent view 只有"看 + 1 步撤回"(iter 23/18),但没有"1 步复活改天"
  - 状态机对称:open↔in_progress↔done/cancelled,iter 18 完成可撤回回 open,但没考虑"已死的任务想重启并改天"场景
  - 主页 focus + focus.secondary + 事件卡 + 事件视图 + 4 视图都有 ↻ 改天按钮,只有 Recent view 没有
- **方案**:Recent view task-actions 加 1 个"↻ 改明天重开" 1 步按钮
  - 触发:`done` 或 `cancelled` 状态的任务 → 1 步复活 + 改到明天
  - 新函数 `reopenRecentToTomorrow(task)`:snapshot planned_for/status → 改为明天 + status='open' → 5s 撤销 snackbar
  - 复用 updateTask 路径(已有,iter 18/21 成熟模式),0 新增后端调用
  - 视觉:蓝紫渐变 (与全站 ↻ 改天按钮同色系,色温统一=蓝色=改天)
  - 按钮文案:"↻ 改明天重开" — "↻" 强调循环复用,"改明天" = 目标,"重开" = 从 done/cancelled 状态复活,9 字内极简
  - 位置:Recent view task-actions,与"查看"等按钮并排
- **场景矩阵**(全站 8 处 ↻ 改天 1 步覆盖):
  - focus primary(iter 38)📅 / focus.secondary 单件(iter 43)→ / focus.secondary 批量(iter 36)↻ / 主页事件卡(iter 45)📅
  - 事件视图(iter 48)📅 / 顺延回顾(iter 46)↻ / 4 视图 iter 36 改天 / **iter 50** recent ↻ 改明天重开
  - 任意位置的"想改天"意图都能 1 步直达,包括已死的旧任务
- **哲学对齐**:Recent view 是用户回顾与复盘的入口,不能只"看"不能"动"
  - iter 18: 完成可撤回(模式)
  - iter 21: 批量操作可撤销(模式)
  - **iter 50: 已死任务可复活改天**(补齐回顾维度的 1 步动作)
  - 用户回看 Recent 时,触达"想再做"的念头不需要新建任务,直接 1 步复用原任务上下文(评论/活动/子项全保留)
  - 数据主权:复用原 task ID,历史评论/活动日志不丢,与 iter 18 "完成不破坏历史"完全对齐
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件 — 复用 updateTask + showUndoSnackbar 模式
  - 改动:1 个 `reopenRecentToTomorrow` 函数 + 1 处 recent view task-actions 模板新增按钮
  - 历史评论/活动/关联数据 0 破坏,沿用原 task ID

### iter 51: 主页 🔥 连续 chip 1 步 = 最近 view · + TDZ bug 修复(lowEnergyMode/focusMode 的 watch 提前访问 ref 触发 "Cannot access 'mp' before initialization" 白屏)

- **问题 A (1 步化)**:
  - 主页 summary strip 的 🔥 连续 chip 已经显示连续天数 + 近 7 天柱状图 + 累计完成数,信息丰富但"我想看最近到底完成了哪些"必须手动找 view tab "最近"(第 7 个 tab,从左数要滚很久)
  - 与已 1 步的 🚨 紧急(iter 39)/ 📅 明天 / 🔋 低能量 / 🎯 专注 / 💼 模式 不对称 — 这是数据 chip 中第 2 个仍纯展示
  - 用户场景:看连续 7 天 → 想"我到底完成了哪些" → 找不到入口,被迫切 view tab
- **方案**:
  - 新函数 `jumpToStreak()`:无历史时 return;有历史时 `activeView.value = 'recent'`
  - 与 🚨 紧急 iter 39 完全对称:**数据 chip = 1 步入口**(`summary-chip-clickable` + `summary-chip-streak-view-active` 高亮)
  - 复用已有 activeView 系统(0 新增 ref/localStorage)
  - tooltip 动态:无历史 = "完成一件事开始累积连续天数";有历史 = "1 步查看最近完成的 N 件事";正在查看 = "正在查看最近完成 · 切回时间线"
  - 视觉:橙系 + 内描边(与 🚨 紧急的红色对称,与 streak-active 暖色叠加形成"既连续又在看"双重提示)
- **问题 B (TDZ bug 修复)**:
  - 用户反馈线上"Cannot access 'mp' before initialization at setup"白屏 — 定位 `PlannerTool.vue:3346/3350`
  - 根因:`watch(lowEnergyMode, ...)` 在 line 3346 注册,但 `lowEnergyMode` 在 line 3952 才声明;同理 `watch(focusMode, ...)` 在 3350,`focusMode` 在 3953
  - `<script setup>` 自上而下执行,watch 注册时尝试访问 ref → 触发 TDZ → 整个 setup 抛出 → Vue 渲染失败
  - 这是个潜在 bug — 在 dev 模式未触发(Vue 容错),在 prod minify 后严格暴露(`mp` 是 minify 后的变量名)
- **方案 B**:
  - 把 line 3346-3353 的两个 watch + 两个 toggle 函数整体下移到 line 3953 之后(ref 声明完毕)
  - line 3346 改为占位注释,标注下移原因
  - 0 新增逻辑,纯结构调整
- **场景矩阵**(主页 summary chip 全员 1 步化):
  - 🚨 紧急 iter 39 → 时间线+紧急 filter
  - 🔥 连续 iter 51 → 最近 view
  - 📅 明天 / 🔋 低能量 / 🎯 专注 / 💼 模式 / 今日进度 已 1 步
  - 主页所有 chip 都是入口,0 展示 chip 残留
- **哲学对齐**:主页是用户每天第 1 屏,任何数据都不该是死的展示
  - 用户看到任何数据指标 → 自然想"展开看" → 1 步可达,无导航摩擦
  - 主页信息密度自适应:有历史时 chip 变入口,无历史时退化为纯展示(灰样式,不误导用户)
- **零 schema 破坏**
  - 0 数据库迁移
  - 0 API 改动
  - 0 新组件
  - 改动:1 个 `jumpToStreak` 函数 + 1 处 🔥 chip 模板 click handler + 1 段 CSS(.summary-chip-streak-view-active) + 2 处 watch/toggle 下移修 TDZ

---









---

---

## 七、什么是绝对不要做的

1. ❌ 不要拆分文件除非必要 —— 单文件聚合点是 Vue 的优势,保留它
2. ❌ 不要为了"看起来简洁"删掉有用的功能 —— 用户依赖
3. ❌ 不要加新依赖(图表库、动画库等)除非必要
4. ❌ 不要让"高级设置"喧宾夺主 —— 默认就该够用
5. ❌ 不要在主页加任何"弹窗引导"、"新手教程"、"视频演示" —— 用户进来就要做事

---

## 八、衡量标准(什么叫优化成功)

每次优化后问自己:
1. **新用户首次进入,30 秒内能不能开始录入?** (降低门槛)
2. **老用户每天打开,完成 1 件事需要点几次?** (降低决策)
3. **3 个月后回来看,数据还在不在?** (平滑迁移)
4. **如果我把这个功能删了,有没有更好的替代?** (减法)

---

最后:**优化是减法,不是加法。**

每一行新代码都要回答:它在帮用户卸下负担,还是在增加负担?