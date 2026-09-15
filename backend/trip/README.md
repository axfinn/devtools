# 旅游行程助手 (trip)

独立 DDD 限界上下文,与 planner / recipe / household 等模块平级。

## 概览

| 维度 | 设计 |
|------|------|
| 领域对象 | `Trip` 聚合根(预算)→ `DayPlan` 实体 → `Activity` 实体 + `Expense` 实体 + `Destination` 值对象 |
| 数据访问 | `Repository` 接口 + `SQLiteRepository` 实现,复用 `models.DB.Conn()` 同连接 |
| 用例层 | `trip.Service`:CRUD + Markdown 渲染 + 总结(`TripSummary` + `RenderSummaryMarkdown`) |
| 邮件通道 | 复用 `notif.Config`(SMTP 客户端 + multipart)+ `cfg.AskitSync.SMTPHost/Port/User/Pass` |
| HTTP 入口 | `backend/handlers/trip.go` + `backend/routes/trip.go`,挂在 `/api/trips/*` |
| CLI 入口 | `backend/cmd/trip/`,默认独立 `./data/trip.db`(可 `--db` 切主库) |
| 前端入口 | `frontend/src/views/life/TripTool.vue`,路由 `/trip`(预算顶栏 / 按天行程 / 费用 / 总结 / 行程设置) |
| 提醒调度 | cleanup goroutine 每小时扫一次 `trip_activities.remind_sent_at = 0` 行 |

## 对外 URI(完整)

### 行程 / Day / Activity

| Method | URI | 行为 |
|--------|-----|------|
| `GET` | `/api/trips` | 列出全部行程(轻量,无 days) |
| `POST` | `/api/trips` | 新建空行程 |
| `GET` | `/api/trips/:id` | 完整聚合(Trip + Days + Activities,JSON) |
| `GET` | `/api/trips/:id/markdown` | 渲染 Markdown 行程单(`text/markdown`) |
| `GET` | `/api/trips/:id/summary` | 汇总 JSON(预算 / 实际 / 按类别 / 按天) |
| `GET` | `/api/trips/:id/summary/markdown` | 汇总 Markdown(可复制贴 Notion) |
| `DELETE` | `/api/trips/:id` | 级联删除 trip + days + activities + expenses |
| `POST` | `/api/trips/:id/days` | 给行程加一天 |
| `PATCH` | `/api/trips/days/:dayId` | 出行中临时调整某天(date / city / region / country) |
| `POST` | `/api/trips/days/:dayId/activities` | 给一天加一项活动 |
| `PUT` | `/api/trips/activities/:activityId/reminder` | 配置提醒;`reset=true` 重发 |
| `PATCH` | `/api/trips/activities/:activityId` | 出行中临时调整活动(title / kind / start_time / location / duration_min / note) |

### 预算 / 费用

| Method | URI | 行为 |
|--------|-----|------|
| `PUT` | `/api/trips/:id/budget` | 设置整程预算(支持 `budget_total_cents` / `budget_yuan` / `budget_total` 三种入参) |
| `GET` | `/api/trips/:id/expenses` | 列出该 trip 的全部费用 |
| `POST` | `/api/trips/:id/expenses` | 新增一笔费用(支持 `amount` 元 / `amount_cents` 分) |
| `PUT` | `/api/trips/expenses/:expenseId` | 编辑一笔费用 |
| `DELETE` | `/api/trips/expenses/:expenseId` | 删除一笔费用 |

### Seed / 提醒

| Method | URI | 行为 |
|--------|-----|------|
| `POST` | `/api/trips/seed?force=true\|false` | 落「国庆 14 日」fixture,幂等 |
| `POST` | `/api/trips/process-now` | 立刻扫一次提醒(前端"测试提醒"按钮用) |

## CLI 速查

```bash
go build -o /tmp/trip ./backend/cmd/trip

# 行程 CRUD
/tmp/trip --db /tmp/trip.db init
/tmp/trip --db /tmp/trip.db seed
/tmp/trip --db /tmp/trip.db list
/tmp/trip --db /tmp/trip.db show <trip-id>          # 完整 Markdown 行程单
/tmp/trip --db /tmp/trip.db create --name X --start S --end E --cities ...
/tmp/trip --db /tmp/trip.db add-day <trip-id> --date 2026-09-24 --city 深圳
/tmp/trip --db /tmp/trip.db add-activity <day-id> --kind food --title "..."
/tmp/trip --db /tmp/trip.db update-activity <act-id> --title "改后标题" --time 14:30
/tmp/trip --db /tmp/trip.db remind <act-id> --before 60 --email foo@bar.com --reset

# 预算 + 费用
/tmp/trip --db /tmp/trip.db budget <trip-id> --total 20000              # 设预算 2w 元
/tmp/trip --db /tmp/trip.db expense-add <trip-id> --date 2026-09-24 --category food --amount 88.50 --payment alipay --note "八合里"
/tmp/trip --db /tmp/trip.db expense-list <trip-id>
/tmp/trip --db /tmp/trip.db expense-delete <expense-id>

# 总结(回程后)
/tmp/trip --db /tmp/trip.db summary <trip-id>                          # 文本
/tmp/trip --db /tmp/trip.db summary <trip-id> --md                     # Markdown 总结
```

## 数据模型

### Trip(聚合根)

```
ID, Name, Description, StartDate, EndDate,
Tags[], CoverCities[],
NotifyEmail,
BudgetTotal (int64 cents), BudgetCurrency (ISO 4217, 默认 CNY),
CreatedAt, UpdatedAt
Days[]         -> DayPlan[]
```

### DayPlan

```
ID, TripID, Date (YYYY-MM-DD), Order
Destination {City, Region, Country}
Activities[]   -> Activity[]
```

### Activity

```
ID, DayID, Kind (transit/sight/food/leisure/shopping/lodging),
Title, Location, Destination, StartTime (HH:MM or 上午/下午/晚上),
DurationMin, Note, Order,
RemindBeforeMinutes (0 = 关闭), RemindEmails[], RemindSentAt
```

### Expense

```
ID, TripID, Date (YYYY-MM-DD), Category (transport/lodging/food/sight/shopping/misc),
AmountCents (int64), Currency (ISO 4217, 默认沿用 trip),
Note, PaymentMethod (cash/card/alipay/wechat/other),
ActivityID (可选,关联到具体活动),
CreatedAt, UpdatedAt
```

## 预算 / 费用 / 总结行为约定

- **金额精度**:存「分」避免浮点漂移;`amount_yuan × 100` 入库。
- **货币默认**:不填时,expense 沿用 trip.BudgetCurrency;trip.BudgetCurrency 不填则默认 `CNY`。
- **多币种**:总结时只在同币种之间累加;不同币种记入 `summary.other_currencies` 提示,
  不折算(避免汇率漂移)。单币种 demo 直接可读。
- **删除级联**:`trip_trips` ON DELETE CASCADE 自动清 days + activities + expenses。
- **总结触发**:用户点「总结 → 复制 Markdown」调 `RenderSummaryMarkdown`,
  内含预算概要表 + 按类别 + 按天 + 明细四块。

## SMTP 配置

邮件提醒走现有 SMTP 通道,**无需新增配置项**:

```yaml
askit_sync:
  smtp_host: smtp.qq.com
  smtp_port: 465
  smtp_user: foo@qq.com
  smtp_pass: ********
```

未配齐四项时,所有行程提醒静默跳过(启动期 `trip reminder: SMTP 未配置` 提示)。

## 提醒触发时机

每条活动可独立配置 `remind_before_minutes`(0 = 关闭)。

```
触发时刻 = activity_date + activity_start_time - remind_before_minutes
```

例:`09-25 上午` 的活动 + `remind_before_minutes = 120`,触发时刻为 `09-25 08:00`。
cleanup goroutine 每小时扫一次 `[now-30m, now+30m)` 窗口,命中即发邮件并把
`remind_sent_at` 写 DATETIME,实现"每条活动最多发一次"。

## 测试

```bash
go vet ./backend/...   # 无告警
go build ./backend/... # 整个 backend 编译通过
```

CLI 端到端:

```bash
rm -f /tmp/trip.db
go run ./backend/cmd/trip --db /tmp/trip.db seed
go run ./backend/cmd/trip --db /tmp/trip.db list
go run ./backend/cmd/trip --db /tmp/trip.db show $(go run ./backend/cmd/trip --db /tmp/trip.db list | tail -1 | awk '{print $NF}')

# 出行中
go run ./backend/cmd/trip --db /tmp/trip.db budget <trip-id> --total 20000
go run ./backend/cmd/trip --db /tmp/trip.db expense-add <trip-id> --date 2026-09-24 --category food --amount 88.50 --note "八合里"
go run ./backend/cmd/trip --db /tmp/trip.db summary <trip-id> --md
```

## 设计权衡

- **数据库隔离**:HTTP 端点共用 `models.DB.Conn()`(即 `./data/paste.db` 主库),
  CLI 默认独立 `./data/trip.db`。两条路径走同一份领域代码(`trip.Service`),
  但 SQL 连接池不同——避免 CLI 误改 HTTP 数据。
- **schema 升级**:预算字段通过 `ALTER TABLE … ADD COLUMN` 加进已存在的
  `trip_trips`;duplicate column name 错误吞掉,保证幂等。新表 `trip_expenses`
  通过 `CREATE TABLE IF NOT EXISTS` 处理。
- **不引入 PUT trip**:本期没实现 `PUT /api/trips/:id`,前端"保存设置"走
  「删除重建」流程(对 fixture demo 安全,真实场景应补 PUT)。
- **不重构 planner/proxy 邮件代码**:SMTP 抽取成 `notif.Send` 仅为新模块使用,
  planner/proxy 现有路径保持不动,避免改动面扩散。
- **多币种不折算**:目前只累加同币种费用;`other_currencies` 提示用户哪几笔需要手工换算。
  这避免了汇率漂移误算,也避免引入 FX rate 服务依赖。
- **PATCH 不重置 reminder**:活动 inline edit 只改 title/kind/start_time/location/
  duration_min/note,不动 `remind_before_minutes` 和 `remind_emails`(由独立
  PUT reminder 维护),也不动 `order`(避免误改排序)。