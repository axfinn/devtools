# trip 行程 ↔ 记账(expense)自动建账 —— main 现状上的移植方案

- **文档**：`docs/plans/trip-expense-autobridge.md`
- **基线**：`origin/main` @ `37a1da259e7cffca4192191b09fd6f23824adf0d`（= `43c389f` + chat-plus 合并；`43c389f` 是 `37a1da2` 的祖先）
- **参考来源分支**：`agent/devtools-engineer/devt-20-d5efcf63a034` @ `6bc2f889ea93713ad22209df6bb34de7202f6993`（**本地独有，origin 上不存在；只读参考，未 checkout / 未 merge / 未 cherry-pick**）
- **对应单**：[FINN-205]（Stage 1 技术方案设计），父单 FINN-204
- **阶段**：Stage 1 产出方案 → Stage 2 `devtools-design-reviewer` 评审 → Stage 3 `devtools-backend` 实现 → Stage 4 审查 → Stage 5 QA → Stage 6 PM 合并

> 本文所有 `文件:行` 均在上述基线上**实际 grep / sed 核对过**，原始输出见 §附录 A。未经核对的引用一律不写。

---

## 目标与非目标

### 目标

1. 在 **main 现状**上重新实现「行程活动标记完成 → 自动在记账模块建账」的跨模块能力（`ExpenseHook` 机制 + SQL 桥接）。
2. 给出**幂等判定依据**（PM 边界 4）：同一 activity 重复提交「标记完成」不得重复建账，且判定依据可在数据库层被证伪。
3. 给出**幂等 schema 变更**写法（PM 边界 3）：只允许「启动期幂等 `ALTER TABLE ADD COLUMN` + 吞重复列错」，**不改** `backend/models/expense.go` 既有建表语句。
4. 给出 main 上的**装配点**（文件:行级落点）。
5. 给出**失败语义**：建账失败不得让「标记完成」失败（PM 边界 5）。
6. 给出**回退方案**（开关位 / 不装配 hook）。
7. 列出**未定义边界**，交 PM 在 Stage 2 前逐条拍板（不自行留悬空问题）。

### 非目标

- ❌ 不改记账模块的既有对外 API 契约（端点路径 / 入参 / 返回结构）——PM 边界 2。
- ❌ 不改 `backend/models/expense.go` 的既有建表语句与列定义——PM 边界 3。
- ❌ 不做反向依赖：记账模块**不得** import trip 模块——PM 边界 1。
- ❌ 不写任何 Go 代码（本单是 Stage 1，只出方案）。
- ❌ 不做前端改动（PM 边界 6：Stage 3 后端单不得碰 `frontend/`；前端兄弟单由 PM 另行追加）。
- ❌ 不引入 activity 删除端点（main 上根本没有该能力，见 §现状 2.4，超出本单范围）。
- ❌ 不部署、不跑 `./deploy.sh`、不碰 `192.168.31.201`。

---

## 现状

### 1. main 上**没有**这套机制（已核实）

```
$ grep -n 'ExpenseHook\|hooks' backend/trip/service.go
exit=1            # 0 命中
```

```
$ git ls-tree -r --name-only refs/remotes/origin/main -- backend/trip/
backend/trip/README.md
backend/trip/domain.go
backend/trip/reminder.go
backend/trip/repository.go
backend/trip/seed.go
backend/trip/seed_output_sample.md
backend/trip/service.go
                   # ← 无 expense.go
```

```
$ grep -c 'trip_id\|activity_id' backend/models/expense.go
0                 # expense_transactions 上无任何 trip/activity 关联列
```

⇒ 与本单 issue §4 的预期一致：**main 确实没有这套机制**，方案要补的正是它。

### 2. 关键发现：**触发面（勾 done）在 main 上也不存在** ⚠️

这是本次勘察最重要的结论，直接决定 Stage 3 的工作量边界。

| 判据 | main 现状（已核实） | 旧分支 `6bc2f88` |
|---|---|---|
| `trip.Activity` 字段 | `backend/trip/domain.go:70-86` —— **无 `Status`、无 `ActualCostCents`、无 `LoggedAt`** | `domain.go:85` 有 `ActualCostCents`，`:43` 有 `StatusDone` |
| `trip_activities` 列 | `backend/trip/repository.go:113-128` —— **无 `status` / `actual_cost_cents`** | `repository.go:136` `ALTER TABLE trip_activities ADD COLUMN actual_cost_cents` |
| 标记完成的端点 | `backend/routes/trip.go:46` 只有 `PATCH /activities/:activityId`（改 title/time/location/duration/note）；**无任何 status 入口** | `routes/trip.go:37` `PUT /activities/:activityId` 带 `status` / `actual_cost_cents` |
| 前端 UI | `frontend/src/views/life/TripTool.vue` —— grep `done` 仅命中 HTTP 错误判断（`if (!r.ok)`），**无勾选框** | 有勾 done UI |

```
$ grep -rn 'StatusDone\|ActualCostCents\|actual_cost_cents' backend/ | grep -v planner
                                        # trip 域 0 命中（planner 的 done 是另一套）
$ grep -n 'done\|actual_cost\|status' frontend/src/views/life/TripTool.vue | head
101:  :status="budgetSpent > (selectedTrip.budget_total || 0) ? 'exception' : ''"
595:    if (!r.ok) throw new Error(`list: HTTP ${r.status}`)
...    # 全是 HTTP status，无 activity done 语义
```

⇒ **只移植 `ExpenseHook` 机制永远不会触发**（没有「标记完成」这个动作）。
Stage 3 必须**同时补最小触发面**，否则交付的是一具永不执行的机制。
本方案的 §设计 4 给出了最小触发面。

### 3. main 上 trip 自有账本与记账模块的现状

**trip 自有账本**（`backend/trip/`）：

- `backend/trip/domain.go:121-133` `type Expense struct` —— 有 `TripID` / `Date` / `Category(ExpenseKind)` / `AmountCents int64` / `Currency` / `Note` / `PaymentMethod` / **`ActivityID string`（第 130 行，main 上已有！）**
- `backend/trip/repository.go:134-147` `CREATE TABLE IF NOT EXISTS trip_expenses` —— 列：`id, trip_id, date, category, amount_cents, currency, note, payment_method, activity_id, created_at, updated_at`；`FK trip_id → trip_trips(id) ON DELETE CASCADE`
- `backend/trip/service.go:157` `AddExpense` / `:175` `UpdateExpense` / `:191` `DeleteExpense` / `:198` `ListExpenses`
- `backend/trip/service.go:232` `TripSummary` —— 预算/汇总**只读 `trip_expenses`**（第 240 行 `s.repo.ListExpensesByTrip`）
- 路由：`backend/routes/trip.go:33-36`（`GET/POST /:id/expenses`、`PUT/DELETE /expenses/:expenseId`）

**记账模块**（`backend/models/expense.go` + `backend/handlers/expense.go`）：

- `backend/models/expense.go:53-66` `type ExpenseTransaction struct` —— `ID / ProfileID / AccountID / CategoryID / Amount float64 / Type / Date / Remark / Tags / VoiceText / CreatedAt / UpdatedAt`
  ⚠️ **无币种字段**（隐式单一币种，金额单位是**元**，REAL）
- `backend/models/expense.go:134-150` `CREATE TABLE IF NOT EXISTS expense_transactions` —— `profile_id / account_id / category_id` 均 `NOT NULL` 且带 `FOREIGN KEY … ON DELETE CASCADE`
- `backend/models/expense.go:164` `ALTER TABLE expense_transactions ADD COLUMN voice_text TEXT DEFAULT ''`（既有幂等 ALTER 的写法范例；该 `Exec` **不检查 error**，靠 `IF NOT EXISTS` 的建表 + 忽略重复列错）
- `backend/models/expense.go:407` `CreateExpenseTransaction` —— 不动账户余额
- `backend/handlers/expense.go:777` `CreateTransaction` —— **第 845-850 行才扣账户余额**（`balance = account.Balance - amount`）
- 记账模块是**按档案（profile）隔离**的：`backend/handlers/expense.go:811-824` 校验 account / category 归属 profile，且 `category.Type != req.Type` 直接 400

⇒ **两套账本的语义差异（决定了集成形态）**：

| 维度 | trip 自有 `trip_expenses` | 记账 `expense_transactions` |
|---|---|---|
| 金额 | `amount_cents INTEGER`（**分**，精确） | `amount REAL`（**元**，浮点） |
| 币种 | 有 `currency`（ISO 4217） | **无币种字段** |
| 分类 | 自定义 `ExpenseKind` 枚举（`backend/trip/domain.go:44-51`） | `expense_categories` 表（带 profile 归属 + `type`） |
| 账户 | 无 | `expense_accounts` 表（带 `balance`） |
| 隔离维度 | trip（无鉴权） | **profile**（密码鉴权） |
| 已有 activity 关联 | ✅ `activity_id` 列（`repository.go:143`） | ❌ 无 |

### 4. main 的装配与启动顺序（已核实，ALTER 时机安全）

```
backend/app_runtime.go:47    db, err := models.NewDB(dbPath)
backend/app_runtime.go:52    if err := db.InitAll(); err != nil {     // ← 建出 expense_transactions
backend/main.go:12            handlerSet, err := buildRouteHandlers(rt)
backend/app_handlers.go:37    tripHandler := handlers.NewTripHandler(db, notif.Config{…})
backend/handlers/trip.go:28   repo := trip.NewSQLiteRepository(db.Conn())
backend/handlers/trip.go:29   svc := trip.NewService(repo)
backend/handlers/trip.go:30   if err := svc.EnsureSchema(context.Background()); err != nil { … }
backend/routes/index.go:93    RegisterTripRoutes(api, h, createRateLimiter)
backend/models/paste.go:37   func (db *DB) Conn() *sql.DB   // trip 复用主库同一连接
```

⇒ `NewTripHandler` 执行时 `expense_transactions` **必然已存在** ⇒ 启动期做幂等 ALTER 是安全的。

### 5. CLI 路径是**另一个 SQLite 文件** ⇒ CLI 不装配桥接

```
backend/cmd/trip/main.go:50   const defaultDBPath = "./data/trip.db"
backend/cmd/trip/main.go:89   conn, err := sql.Open("sqlite3", dbPath+"?_parse_time=true")
backend/cmd/trip/main.go:95   repo := trip.NewSQLiteRepository(conn)
backend/cmd/trip/main.go:96   svc := trip.NewService(repo)
```

⇒ CLI 连的是 `./data/trip.db`，**看不到**主库的 `expense_transactions`。
⇒ 桥接**只在 HTTP 路径装配**；CLI 保持 `hooks == nil`（与旧分支「CLI 默认跳过」一致）。**CLI 不得装配桥接**，否则会对着一个没有 `expense_transactions` 的库执行 ALTER。

### 6. 旧分支的可移植资产与不可移植资产

**可移植（设计意图，需在 main 上重写）**：旧分支 `backend/trip/expense.go`（213 行）的
`ExpenseHook` 接口思路 / `EnsureExpenseColumns` 幂等 ALTER 思路 / `AutoCreateOnDone` 的
幂等查重 + 建账 + 扣余额流程 / `resolveAccountCategory` 的兜底解析思路。

**不可移植（旧底，必须丢弃）**：

- 旧分支 `backend/trip/service.go` 是 main 的**旧底**：它把 main 的 `UpdateBudget` 换成了 `UpdateTripBudget`、删掉了 `SetReminder`、删掉了 main 新增的 `GetDay` / `DeleteDay` 等 —— 整文件取件会**整体回退 main 已推进的代码**（本单 issue §2 明令禁止）。
- 旧分支 `backend/trip/expense.go` 依赖 main 上**不存在的**字段：`Trip.DefaultAccountID` / `Trip.DefaultCategoryID`（main 的 `Trip` 结构 `backend/trip/domain.go:99-117` 无这两个字段）、`Activity.Status` / `Activity.ActualCostCents`。
- 旧分支 `ExpenseHook` 接口含 `ListByTrip` / `DeleteByActivity` / `SumByTrip` 三个方法，在 main 的「双写 + 读路径零改动」设计下**全部用不到**（照抄即触发 S1 死代码）。

```
$ git diff --stat origin/main 6bc2f88 -- backend/
 backend/trip/service.go     | 569 ++++-------
 backend/trip/repository.go  | 358 +++----
 backend/trip/domain.go      | 179 ++--
 backend/trip/expense.go     | 213 ++++      ← 新增
 backend/handlers/trip.go    | 382 ++++---
 backend/routes/trip.go      |  24 +-
 backend/cmd/trip/main.go    | 337 ++---
```

---

## 设计

### 1. 集成形态（PM 边界问题 ①）

#### 推荐：**双写同事务，记账模块为真相源，trip 侧保留镜像**

| 项 | 决策 |
|---|---|
| trip 自有表 `trip_expenses` | **保留**，不删、不改既有语义；新增 `link_tx_id` 列标记镜像行 |
| 记账表 `expense_transactions` | **真相源**（有账户余额 / 分类体系 / 统计 / AI 总结） |
| 写入方式 | **双写**：`expense_transactions` + `trip_expenses` 镜像行，**同一 `sql.Tx` 内** |
| trip 侧读路径 | **零改动**（`TripSummary` / `ListExpenses` / `RenderSummaryMarkdown` 仍读 `trip_expenses`） |
| 镜像行定位 | `trip_expenses.link_tx_id != ''` ⇒ 该行是自动建账的镜像 |

**理由**

1. **读路径零改动 = 最小风险**。main 的预算、汇总、Markdown、明细四条链路全部经由 `backend/trip/service.go:240` 的 `ListExpensesByTrip` 读 `trip_expenses`。若只写 `expense_transactions`，用户勾完 done 会发现**行程预算纹丝不动** —— 功能在 trip 内部等于没生效。双写后这四条链路零改动即自动包含新账。
2. **金额精度**。`expense_transactions.amount` 是 `REAL`（元），`trip_expenses.amount_cents` 是 `INTEGER`（分）。trip 侧镜像存**精确的分**，不受浮点往返影响；`expense_transactions` 侧的浮点是既有记账模块的设计，不在本次改动范围。
3. **原子性**。两个 `INSERT` 放在同一个 `sql.Tx` 内提交 ⇒ 不会出现「记账有、行程没有」或反之。
4. **DDD 边界不破**。镜像行仍由 trip 模块自己的 `Repository` 写（新增一个显式带 `*sql.Tx` 参数的方法，见 §4.3），不靠 bridge 裸写 SQL。

**备选 A（纯记账侧 + trip 读并集）**：不双写，改 `ListExpenses` / `TripSummary` 去并集 `trip_expenses` ∪ `expense_transactions(trip_id=?)` 并按 `activity_id` 去重。
✗ 改动面更大（动 main 的汇总核心逻辑）；✗ `REAL(元)` ↔ `int64(分)` 往返有浮点误差；✗ `GET /trips/:id/expenses` 的响应体要混入两种来源的记录，前端需改；✗ 同一笔账可能在两个模块各显示一次。
**备选 B（只写 trip 自有表）**：违背本单目标（父单 PM 边界 2 明确「只允许**新增** trip 侧写入记账表的能力」）。

### 2. 幂等判定依据（PM 边界 4 —— 本方案最硬的部分）

三层防线，**L1 是唯一硬保证**，L2/L3 是优化与可观测性。

#### L1 · 数据库层硬保证：partial UNIQUE index

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_expense_trip_activity
  ON expense_transactions(trip_id, activity_id)
  WHERE trip_id IS NOT NULL AND activity_id IS NOT NULL;
```

- 同一 `(trip_id, activity_id)` 的**第二条** `INSERT` 会被 SQLite 直接拒绝（`UNIQUE constraint failed`）⇒ 并发 / 重试 / 重复提交都**不可能**产生第二行。
- **老库安全（关键论证）**：`trip_id` / `activity_id` 是本次**新增**的列，存量行两列均为 `NULL`；partial index 的 `WHERE` 子句**排除 NULL 行** ⇒ 存量行不参与唯一性 ⇒ 索引**不可能因存量重复而创建失败**。
- 幂等创建：`CREATE UNIQUE INDEX IF NOT EXISTS` 本身可重复执行。

#### L2 · 状态位（快路径 + 可观测）

```sql
ALTER TABLE trip_activities ADD COLUMN expense_tx_id TEXT NOT NULL DEFAULT '';
```

- 建账成功后把 `expense_transactions.id` 写回 `trip_activities.expense_tx_id`。
- 再次「标记完成」时 `expense_tx_id != ''` ⇒ **连 INSERT 都不发**，直接返回既有 tx id（幂等命中）。
- 兼作**「已建账」状态位**：QA / 前端都能直接查这一列判定，不需要猜。

#### L3 · 状态跃迁（仅优化，不承担保证责任）

- service 层只在 **非 `done` → `done`** 的跃迁时触发 hook。
- ⚠️ L3 **不是**保证：取消 done 后再勾 done，L3 会再次触发，但 L1 会挡住第二条记录。
- 因此本方案的定义是：**「一个 activity 终身最多一笔自动账」**，而不是「每次 done 跃迁建一笔」。

#### 判定依据汇总表

| 层 | 载体 | 判定动作 | 保证强度 |
|---|---|---|---|
| L1 | `expense_transactions` partial UNIQUE index | DB 拒绝第二条 INSERT | **硬保证**（并发安全） |
| L2 | `trip_activities.expense_tx_id` | `!= ''` → 直接返回既有 tx id | 快路径 + 状态可观测 |
| L3 | `trip_activities.status` 跃迁检测 | 非 done→done 才触发 | 仅优化 |

### 3. Schema 变更（PM 边界 3 —— 只允许幂等 ALTER）

**全部集中在新文件 `backend/trip/expense.go` 的 `EnsureExpenseColumns(ctx) error` 里；
`backend/models/expense.go` 一行都不改。**

| # | 语句 | 目标表 | 说明 |
|---|---|---|---|
| 1 | `ALTER TABLE expense_transactions ADD COLUMN trip_id TEXT DEFAULT NULL` | 记账模块 | 幂等；吞重复列错 |
| 2 | `ALTER TABLE expense_transactions ADD COLUMN activity_id TEXT DEFAULT NULL` | 记账模块 | 幂等；吞重复列错 |
| 3 | `ALTER TABLE trip_activities ADD COLUMN status TEXT NOT NULL DEFAULT 'planned'` | trip 自有 | 幂等；吞重复列错 |
| 4 | `ALTER TABLE trip_activities ADD COLUMN actual_cost_cents INTEGER DEFAULT NULL` | trip 自有 | 幂等；吞重复列错 |
| 5 | `ALTER TABLE trip_activities ADD COLUMN expense_tx_id TEXT NOT NULL DEFAULT ''` | trip 自有 | 幂等；吞重复列错 |
| 6 | `ALTER TABLE trip_expenses ADD COLUMN link_tx_id TEXT NOT NULL DEFAULT ''` | trip 自有 | 幂等；吞重复列错 |
| 7 | `ALTER TABLE trip_trips ADD COLUMN default_account_id TEXT NOT NULL DEFAULT ''` | trip 自有 | 幂等；吞重复列错（可选，见未决 Q5） |
| 8 | `ALTER TABLE trip_trips ADD COLUMN default_category_id TEXT NOT NULL DEFAULT ''` | trip 自有 | 幂等；吞重复列错（可选，见未决 Q5） |
| 9 | `CREATE UNIQUE INDEX IF NOT EXISTS idx_expense_trip_activity ON expense_transactions(trip_id, activity_id) WHERE trip_id IS NOT NULL AND activity_id IS NOT NULL` | 记账模块 | 幂等；**L1 硬保证** |
| 10 | `CREATE INDEX IF NOT EXISTS idx_expense_tx_trip ON expense_transactions(trip_id)` | 记账模块 | 幂等；删除/汇总用 |
| 11 | `CREATE INDEX IF NOT EXISTS idx_trip_expenses_link_tx ON trip_expenses(link_tx_id)` | trip 自有 | 幂等 |

**吞重复列错的写法**：直接复用同包已有的 `backend/trip/repository.go:170` 的
`isDuplicateColumnErr(err)`（判 `duplicate column name` / `already exists`），
`backend/trip/expense.go` 与 `repository.go` 同属 `package trip` ⇒ **无需新写一份**。

**为什么 trip 自有表也走 ALTER 而不是改建表语句**：`backend/trip/repository.go:90-168`
的 `EnsureSchema` 已经是「`CREATE TABLE IF NOT EXISTS` + `ALTER TABLE ADD COLUMN` 吞重复列错」的成熟模式
（见 `:157-166` 给 `trip_trips` 加 `budget_total` / `budget_currency`）⇒ 沿用同一写法，风格一致。

**「不改 `backend/models/expense.go` 建表语句」的落实**：上面 #1 #2 #9 #10 四条都作用在
`expense_transactions` 上，但**执行代码在 `backend/trip/expense.go`**，不在 `models` 包。
`backend/models/expense.go` 的 `CREATE TABLE IF NOT EXISTS expense_transactions`（`:134-150`）
与 `ExpenseTransaction` 结构体（`:53-66`）**保持原样**，老库升级路径不受影响。
⇒ PM 边界 3 满足。

### 4. 后端改动清单

#### 4.1 新增文件 `backend/trip/expense.go`（`package trip`）

```
type ExpenseHook interface {
    // AutoCreateOnDone 在活动被标记 done 且有实际花费时调用;
    // 幂等:同一 activity 已有账时直接返回既有 tx id(见 §2 三层防线)。
    AutoCreateOnDone(ctx context.Context, t *Trip, day *DayPlan, a *Activity) (txID string, err error)
    // DeleteByTrip 删某 trip 关联的所有自动账(expense_transactions 上无 FK,需手动清理)。
    DeleteByTrip(ctx context.Context, tripID string) error
}

type sqlExpenseBridge struct {
    db   *sql.DB
    repo Repository        // 写 trip_expenses 镜像行用(保持 DDD 边界)
}

func NewSQLExpenseBridge(db *sql.DB, repo Repository) *sqlExpenseBridge
func (b *sqlExpenseBridge) EnsureExpenseColumns(ctx context.Context) error
```

> **接口只含 2 个方法**。旧分支的 `ListByTrip` / `DeleteByActivity` / `SumByTrip` 在本设计下
> **没有任何调用点** ⇒ 不引入（否则触发 S1 死代码，见 §D3 自查）。
> `EnsureExpenseColumns` 挂在**具体类型**上而非接口上 —— 它是启动期 schema 动作，不是业务副作用。

`AutoCreateOnDone` 内部流程（伪码，逐行对应 §2 / §3）：

```
1. if a.ActualCostCents == nil || *a.ActualCostCents <= 0 → return "", nil   // 无花费不建账
2. 币种闸门：trip.BudgetCurrency 非空且 != "CNY" → return "", errSkipCurrency  // 见 Q6
3. 幂等快路径：SELECT id FROM trip_activities WHERE id = a.ID AND expense_tx_id != ''
                → 命中则 return expense_tx_id, nil
4. 解析账户/分类：resolveAccountCategory(ctx, t, reqOverride) → accountID, categoryID, profileID
                   解析不出 → return "", err（外层吞）
5. tx, err := b.db.BeginTx(ctx, nil); defer tx.Rollback()
6.   INSERT INTO expense_transactions
       (id, profile_id, account_id, category_id, amount, type, date, remark, tags, voice_text,
        trip_id, activity_id, created_at, updated_at)
       VALUES (?,?,?,?,?, 'expense', ?, ?, 'trip', '', ?, ?, now, now)
       amount = float64(*a.ActualCostCents) / 100.0      // 分 → 元
       remark = "<Trip.Name> · <kindCN(a.Kind)> <a.Title>"
       date   = day.Date（空则 today）
     → UNIQUE 冲突(L1) → tx.Rollback(); SELECT 既有 id; return 既有 id, nil   ← 幂等命中
7.   repo.AddExpenseTx(ctx, tx, mirrorExpense)          // 写 trip_expenses 镜像行(精确分)
8.   UPDATE trip_activities SET expense_tx_id = <newTxID> WHERE id = a.ID
9.   UPDATE expense_accounts SET balance = balance - <amount>, updated_at = now WHERE id = accountID
10.  tx.Commit()
11.  return newTxID, nil
```

> 第 6 步的 `UNIQUE 冲突` 判定：SQLite 错误串含 `UNIQUE constraint failed`。
> 实现上应先 `tx.Rollback()` 再重查（不能在 aborted tx 里查），并把「已存在」当作**成功**返回。

#### 4.2 `backend/trip/service.go`（**只加 hook 字段与 2 个触发点**）

| 位置 | 改动 |
|---|---|
| `:14-16` `type Service struct` | 增加 `hooks ExpenseHook` 字段（`nil` 时全部跳过） |
| `:19-21` `NewService` | 签名**不变**（保持 `cmd/trip/main.go:96` 等现有调用点不动） |
| 新增 `SetHooks(h ExpenseHook)` | 装配期注入一次 |
| `:24-26` `EnsureSchema` | 保持现状（trip 自有表的 schema 由 repo 管）；**桥接的 `EnsureExpenseColumns` 由 handler 在 `EnsureSchema` 之后单独调**（见 §4.5），避免 service 反向依赖具体类型 |
| `:130-142` `UpdateActivity` | 末尾追加触发点（见下） |
| `:112-125` `AddActivity` | **不触发**（新建活动不可能是 done；保持语义简单） |
| `:78-83` `DeleteTrip` | 追加 `s.hooks.DeleteByTrip(ctx, id)`（吞错） |

**触发点伪码**（`UpdateActivity` 内，`s.repo.UpdateActivity` 成功之后）：

```go
old, err := s.repo.GetActivity(ctx, a.ID)   // 已有:main 的 UpdateActivity 已先 GetActivity 做过存在性校验
…
if err := s.repo.UpdateActivity(ctx, a); err != nil {
    return err
}
// —— 自动建账触发点 ——
if s.hooks != nil && a.Status == StatusDone &&
   a.ActualCostCents != nil && *a.ActualCostCents > 0 {
    t, dp := s.findTripByActivity(ctx, a)    // 新增:activity → (trip, day) 反查
    if t != nil && dp != nil {
        if txID, err := s.hooks.AutoCreateOnDone(ctx, t, dp, a); err != nil {
            // PM 边界 5:建账失败不阻断「标记完成」
            log.Printf("[trip] 自动建账失败(activity=%s,已忽略): %v", a.ID, err)
        } else {
            a.ExpenseTxID = txID
            _ = s.repo.SetActivityExpenseTxID(ctx, a.ID, txID)   // L2 状态位落库
        }
    }
}
return nil
```

⚠️ **`findTripByActivity` 是新增能力**：main 的 `Repository`（`backend/trip/repository.go:23`）
**没有**「按 activity 反查 trip / day」的方法（旧分支有）。
需新增 `Repository.GetActivityContext(ctx, activityID) (*Trip, *DayPlan, error)`
（一条 JOIN：`trip_activities a JOIN trip_days d ON a.day_id = d.id JOIN trip_trips t ON d.trip_id = t.id`）。
**接口加方法**是安全的：main 上 `Repository` 只有 `SQLiteRepository` 一个实现，且 trip 模块**当前 0 个测试文件**（`ls backend/trip/*_test.go` → no matches）⇒ 无 mock 会被打破。

#### 4.3 `backend/trip/repository.go`（新增 4 个方法 + 2 条 ALTER）

| 新增 | 作用 |
|---|---|
| `GetActivityContext(ctx, activityID) (*Trip, *DayPlan, error)` | activity → (trip, day) 反查（触发点需要） |
| `UpdateActivityProgress(ctx, activityID, status string, actualCostCents *int64) error` | 只更新 `status` / `actual_cost_cents` 两列（**不复用** `UpdateActivity`，因为 `:449` 的 `UpdateActivity` 是「只改 7 个主字段」的局部 UPDATE，语义不同） |
| `SetActivityExpenseTxID(ctx, activityID, txID string) error` | 落 L2 状态位 |
| `AddExpenseTx(ctx, tx *sql.Tx, e *Expense) error` | **在外部事务里**写 `trip_expenses` 镜像行（双写原子性） |

`Repository` 接口（`:23`）相应加 4 个方法；`SQLiteRepository` 是唯一实现，无破坏面。
`AddExpenseTx` 与 `AddExpense`（`:471`）的差别只在「用哪个 executor」⇒ 实现上抽一个
`insertExpense(ctx, execer, e)` 私有函数，`AddExpense` 传 `r.db`、`AddExpenseTx` 传 `tx`，**不重复 SQL**（S5 去重）。

#### 4.4 `backend/trip/domain.go`（字段扩展）

| 结构 | 新增字段 |
|---|---|
| `Activity`（`:70-86`） | `Status ActivityStatus` / `ActualCostCents *int64` / `ExpenseTxID string` |
| 新增 `ActivityStatus` 类型 | `StatusPlanned` / `StatusDone` / `StatusSkipped`（**不引入**旧分支的 `StatusLogged` / `ActivityReminder` 等，主用不上） |
| `Expense`（`:121-133`） | `LinkTxID string`（供 `GET /:id/expenses` 暴露镜像标记；`json:"link_tx_id,omitempty"` ⇒ 向后兼容） |
| `Trip`（`:99-117`） | `DefaultAccountID` / `DefaultCategoryID`（**可选**，见未决 Q5） |

`Activity.Validate()`（`:194-210`）追加：`status` 必须是合法枚举；`actual_cost_cents` 若非 nil 必须 `>= 0`。

#### 4.5 `backend/handlers/trip.go`（装配 + 新端点）

**装配点**（`backend/handlers/trip.go:27-38` `NewTripHandler` 内）：

```go
func NewTripHandler(db *models.DB, notifCfg notif.Config, tripCfg config.TripConfig) *TripHandler {
    conn := db.Conn()                                   // :28 现状
    repo := trip.NewSQLiteRepository(conn)              // :28 现状
    svc := trip.NewService(repo)                        // :29 现状
    if tripCfg.ExpenseBridgeEnabled {                   // ← 开关位(见 §风险与回滚)
        svc.SetHooks(trip.NewSQLExpenseBridge(conn, repo))
    }
    if err := svc.EnsureSchema(context.Background()); err != nil {   // :30 现状
        _ = err
    }
    if tripCfg.ExpenseBridgeEnabled {
        // 幂等 ALTER:必须在 models.InitAll() 之后(= NewTripHandler 天然满足,见 §现状 4)
        if err := trip.NewSQLExpenseBridge(conn, repo).EnsureExpenseColumns(context.Background()); err != nil {
            log.Printf("[trip] 记账表列扩展失败(自动建账将不可用): %v", err)
        }
    }
    proc := trip.NewReminderProcessor(repo, notifCfg)
    return &TripHandler{service: svc, processor: proc}
}
```

⚠️ **签名变更**：`NewTripHandler` 多了 `config.TripConfig` 参数 ⇒
调用点 `backend/app_handlers.go:37` 需同步改一行（`cfg.Trip`）。
这是本方案**唯一**需要改的既有调用点（CLI 走 `backend/cmd/trip/main.go:95-96`，**不改**）。

**新端点 handler**：

| 位置 | 内容 |
|---|---|
| `backend/handlers/trip.go` 新增 `type tripActivityProgressRequest struct` | 见 §5 API 契约 |
| `backend/handlers/trip.go` 新增 `func (h *TripHandler) UpdateActivityProgress(c *gin.Context)` | 解析 → `service.UpdateActivityProgress` → 触发建账 → 返回 `expense_tx_id` / `expense_linked` |

#### 4.6 `backend/routes/trip.go`（挂新端点）

```go
// :46 现有 PATCH /activities/:activityId 保持不变(向后兼容)
// 新增:
t.PATCH("/activities/:activityId/progress", createRateLimiter.Middleware(), h.TripHandler.UpdateActivityProgress)
```

#### 4.7 `backend/config/config.go` + `backend/config.example.yaml`（开关位）

- `backend/config/config.go` 新增 `TripConfig`：
  ```go
  type TripConfig struct {
      ExpenseBridgeEnabled bool `yaml:"expense_bridge_enabled"` // 行程→记账 自动建账开关;默认 true
  }
  ```
  并在 `Config` 结构体（`:14-56` 区）加 `Trip TripConfig \`yaml:"trip"\``。
- `backend/config.example.yaml` 新增 `trip:` 段（放在 `expense:` 段附近，见 `:165`）。

> 默认值语义：`bool` 零值是 `false`。要让「默认开启」成立，需在 `config.Load` 里做
> **「字段缺省 vs 显式 false」的区分**（用 `*bool` 或在 Load 里判断原始 map 是否含该键）。
> ⇒ 见未决 **Q8**（开关默认值），Stage 3 实现时按 PM 拍板结果落地，**不要默认猜**。

#### 4.8 `backend/models/expense.go` —— **不改一行** ✅

PM 边界 2 / 3 的落实点。记账模块的建表语句、结构体、既有 CRUD 全部保持原样；
trip 侧只**新增**写入能力（`backend/trip/expense.go` 里的裸 SQL `INSERT`）。

#### 4.9 `backend/cmd/trip/main.go` —— **不改一行** ✅

CLI 连 `./data/trip.db`（`:50`），看不到 `expense_transactions` ⇒ **不装配桥接**。
`trip.NewService(repo)`（`:96`）签名不变 ⇒ CLI 无需任何改动。

### 5. API 契约

#### 5.1 新增：`PATCH /api/trips/activities/:activityId/progress`

**请求**

```json
{
  "status": "done",
  "actual_cost_cents": 8850,
  "actual_cost_yuan": "88.50",
  "profile_id": "",
  "account_id": "",
  "category_id": ""
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `status` | string | 是 | `planned` / `done` / `skipped`；非法 → 400 |
| `actual_cost_cents` | int64 | 二选一 | **分**；`< 0` → 400；`0` 视为「无花费，不建账」 |
| `actual_cost_yuan` | string | 二选一 | `"88.50"`；`actual_cost_cents` 存在时**优先**用 cents |
| `profile_id` / `account_id` / `category_id` | string | 否 | 记账侧绑定覆盖；缺省走 §6 的解析顺序 |

**响应 `200`**

```json
{
  "activity": {
    "id": "…", "day_id": "…", "kind": "food", "title": "早茶",
    "status": "done", "actual_cost_cents": 8850,
    "expense_tx_id": "9f3c…"
  },
  "expense_tx_id": "9f3c…",
  "expense_linked": true
}
```

| 字段 | 说明 |
|---|---|
| `expense_tx_id` | 建账成功 = 记账侧 tx id；**建账失败/未建账 = 空串** |
| `expense_linked` | `expense_tx_id != ""` ⇒ `true` |

> ⚠️ **建账失败仍返回 `200`**（PM 边界 5）。前端靠 `expense_linked` 决定是否提示「未建账」。

**错误码**

| 场景 | 状态码 | body |
|---|---|---|
| body 非法 / `status` 非法 / `actual_cost_cents < 0` | `400` | `{"error":"…","code":400}` |
| activity 不存在 | `404` | `{"error":"activity not found","code":404}` |
| **建账失败**（无记账档案 / 无账户分类 / DB 错） | **`200`** | 正常 activity + `expense_linked:false` |
| 标记完成本身失败（DB 错） | `500` | `{"error":"…","code":500}` |

#### 5.2 既有端点 —— **契约不变**

| 端点 | 位置 | 变化 |
|---|---|---|
| `PATCH /api/trips/activities/:activityId` | `backend/routes/trip.go:46` | **不变**（不塞 status 进去，避免改既有契约） |
| `GET /api/trips/:id/expenses` | `backend/routes/trip.go:33` | 响应中 `expenses[]` 元素**新增** `link_tx_id` 字段（`omitempty` ⇒ 老客户端不受影响，向后兼容） |
| `GET /api/trips/:id/summary` | `backend/routes/trip.go:25` | **不变**（镜像行已在 `trip_expenses` 里 ⇒ 汇总自动含新账，但**响应结构不变**） |
| 记账模块全部端点 | `backend/routes/expense.go` | **不变**（PM 边界 2） |

### 6. 账户 / 分类 / 档案解析（记账侧三列 NOT NULL 的必然要求）

`expense_transactions.profile_id / account_id / category_id` 均 `NOT NULL` 且带 FK，
而 **trip 模块完全没有档案概念**（无鉴权）。因此必须有解析规则：

| 优先级 | 来源 | 说明 |
|---|---|---|
| 1 | **请求级覆盖** `profile_id` / `account_id` / `category_id` | 前端在记账模块登录后知道当前档案，可显式指定（最准确） |
| 2 | **trip 级默认绑定** `trip_trips.default_account_id` / `default_category_id` | 前端保存一次，之后免传；需新增 `PUT /api/trips/:id/expense-binding`（可选，见 Q5） |
| 3 | **单档案兜底** | **仅当 `expense_profiles` 恰好只有 1 个档案**时：取它的第一个账户（`ORDER BY sort, created_at`）+ 第一个 `type='expense'` 的分类 |
| 4 | 都没有 | **建账失败** → 吞错 + 记日志 → `expense_linked:false` |

> ⚠️ **兜底只在「恰好 1 个档案」时启用**。多档案时选「第一个」是任意选择，可能把账建进**别人的档案**
> —— 违反数据主权。故多档案 ⇒ 直接失败，不猜。

校验（对齐 `backend/handlers/expense.go:811-824` 的既有规则）：
`account_id` 必须属于 `profile_id`；`category_id` 必须属于 `profile_id` 且 `category.type == 'expense'`；不满足 ⇒ 建账失败（吞错）。

### 7. 币种策略（`expense_transactions` 无币种列的必然约束）

- `expense_transactions` **没有 currency 字段**（`backend/models/expense.go:53-66`），记账模块隐含单一币种。
- `trip.Expense.Currency` 支持 ISO 4217 多币种，`TripSummary`（`backend/trip/service.go:257-268`）对非预算币种**不折算**，只单列 `OtherCurrencies`。
- ⇒ **推荐闸门**：`trip.BudgetCurrency` 为空或 `== "CNY"` 才自动建账；其它币种 ⇒ **跳过建账 + 记日志**（否则把 HKD/JPY 当 CNY 记，金额失真且不可察）。
  见未决 **Q6**。

### 8. 失败语义（PM 边界 5）

| 失败点 | 行为 |
|---|---|
| 建账（`AutoCreateOnDone` 返回 err） | **吞错**：`log.Printf` 后继续；「标记完成」**返回 200**，`expense_linked:false` |
| `status` 跃迁检测失败 | 不触发 hook，不报错 |
| `findTripByActivity` 查不到 trip/day | 不触发 hook，不报错 |
| 桥接未装配（开关关闭 / CLI） | `s.hooks == nil` ⇒ **完全跳过**，零开销、零副作用 |

对照旧分支语义：`_, _ = s.hooks.AutoCreateOnDone(ctx, t, dp, a)` —— 一致（吞错、不阻断）。
本方案只是**额外**把失败状态通过 `expense_linked` 暴露给前端（可选，见 Q9）。

### 9. 联动清理

| 动作 | 记账侧 | trip 侧 | 触发点 |
|---|---|---|---|
| 删 **trip** | `DELETE FROM expense_transactions WHERE trip_id = ?`（**无 FK，必须手动**） | `trip_expenses` 靠 `FK trip_id → trip_trips ON DELETE CASCADE`（`backend/trip/repository.go:146`）自动清 | `backend/trip/service.go:78-83` `DeleteTrip` 追加 `s.hooks.DeleteByTrip(ctx, id)`（吞错） |
| 删 **activity** | — | — | ⛔ **不做**：main 上根本没有 activity 删除端点（`backend/routes/trip.go` 无 `DELETE /activities/...`；`backend/trip/service.go` 无 `DeleteActivity`）。旧分支有该能力（`routes/trip.go:38`），但**引入它超出本单范围**，故本方案不做，也不预留死代码 |
| 改金额 | 见未决 **Q3 / Q7** | — | — |

> `DeleteByTrip` 的执行顺序：必须**在** `s.repo.DeleteTrip(ctx, id)` **之前**调用
> —— 因为 trip 删除会 CASCADE 掉 `trip_days` / `trip_activities`，之后拿不到任何 activity 信息。
> （`DeleteByTrip` 只按 `trip_id` 删 `expense_transactions`，不依赖 activity，故先后皆可；
> 但为可读性放在前面更安全。）

---

## 兼容性与迁移

### 老库升级路径

| 对象 | 老库现状 | 升级后 | 是否需回填 |
|---|---|---|---|
| `expense_transactions` | 无 `trip_id` / `activity_id` | 新增两列，存量行均为 `NULL` | **不需要** |
| `expense_transactions` partial UNIQUE index | 不存在 | 新建；`WHERE` 排除 `NULL` 行 ⇒ **存量行不参与唯一性** ⇒ 不可能建索引失败 | 不需要 |
| `trip_activities` | 无 `status` / `actual_cost_cents` / `expense_tx_id` | 新增三列；存量行 `status='planned'` / `actual_cost_cents=NULL` / `expense_tx_id=''` | **不需要** |
| `trip_expenses` | 无 `link_tx_id` | 新增列；存量行 `''` ⇒ **不被识别为镜像行** | 不需要 |
| `trip_trips` | 无 `default_account_id` / `default_category_id` | 新增两列；存量行 `''` ⇒ 走单档案兜底 | 不需要 |

⇒ **纯加列 + 加索引，零回填，零数据迁移。** 所有既有 `CREATE TABLE` 语句与列定义保持不变。

### 语义兼容

- `trip_activities.status` 存量行 = `'planned'` ⇒ 与「旧数据没有状态概念」等价。
- `trip.Expense` 新增 `LinkTxID`（`json:"link_tx_id,omitempty"`）⇒ 老客户端解析多出的字段无影响（JS 忽略未知字段）。
- `Activity` 新增 `Status`（`json:"status,omitempty"`）/ `ActualCostCents` / `ExpenseTxID` ⇒ 同上向后兼容。
  ⚠️ 但 **`Status` 用 `omitempty` 时 `planned` 也会被省略** ⇒ 建议 `Status` 用 `json:"status"`（不带 omitempty），让前端能稳定读到 `"planned"`。这是本方案的一条具体约定。

### 既有 API 响应

- 全部**既有端点的响应结构不变**，仅 `GET /api/trips/:id/expenses` 的**元素**多一个 `link_tx_id` 字段。
- 记账模块端点**零变化**（PM 边界 2）。

---

## 风险与回滚

### 回退方案（三层，从轻到重）

| 层级 | 手段 | 生效方式 | 数据影响 |
|---|---|---|---|
| **L1 开关（首选）** | `config.yaml` → `trip.expense_bridge_enabled: false` ⇒ `NewTripHandler` **不调** `svc.SetHooks(...)` ⇒ `s.hooks == nil` ⇒ 全链路跳过 | 改配置 + **重启** | **零**：已建的账保留，新勾的 done 不再建账 |
| **L2 不装配 hook** | 删掉 `backend/handlers/trip.go` 里的 `svc.SetHooks(...)` 一行 | 重新构建部署 | 零（同 L1） |
| **L3 代码级回退** | 删 `backend/trip/expense.go` + §4.2 的 2 个触发点 + §4.3 的 4 个 repo 方法 + 新端点 | 重新构建部署 | **零**：新增的列全部是默认值 / `NULL`，残留无害；`expense_transactions.trip_id` 上的行仍是有效记账记录，只是没有反向关联 |

> ⚠️ **回滚不自动回补账户余额**。若已建过账且要彻底清理，需手工：
> `DELETE FROM expense_transactions WHERE trip_id IS NOT NULL` + 手工调 `expense_accounts.balance`。
> ⇒ 建议上线后**先在测试库验证**，生产首次开启时**盯着 `expense_accounts.balance`**。

### 风险表

| # | 风险 | 触发条件 | 影响 | 缓解 |
|---|---|---|---|---|
| R-1 | 记账模块**没有档案** | 用户没建过记账档案 | 建账失败 | 吞错 + 日志；`expense_linked:false`；前端引导去记账模块建档案 |
| R-2 | **多档案时建错档案** | `expense_profiles` > 1 行 | 账进别人的档案 | **兜底仅在恰好 1 个档案时启用**；多档案 ⇒ 直接失败不猜 |
| R-3 | **跨币种金额失真** | `trip.BudgetCurrency` = HKD/JPY | 金额被当成 CNY | 币种闸门（§7）：非 CNY ⇒ 跳过建账 + 日志 |
| R-4 | **并发重复建账** | 两请求同时标 done | 重复账 | L1 partial UNIQUE index（DB 级硬保证）+ L2 状态位 |
| R-5 | **浮点误差** | `expense_transactions.amount` REAL | 记账侧金额尾差 | trip 侧镜像存**精确分**；记账侧 REAL 是既有设计，不在本次范围 |
| R-6 | **trip 删除留孤儿账** | 删 trip | `expense_transactions` 残留 | `DeleteTrip` 里手动 `DELETE … WHERE trip_id=?` |
| R-7 | **`ALTER` 在表不存在时执行** | 记账模块 init 失败 | 桥接不可用 | `EnsureExpenseColumns` 的错误**只记日志不 fatal**；桥接不可用时 trip 全功能正常 |
| R-8 | **CLI 误装配** | 有人给 CLI 加 SetHooks | 对 `./data/trip.db` 执行 ALTER ⇒ 报错 | 装配只在 `backend/handlers/trip.go`；CLI 不动（§4.9） |
| R-9 | **接口变更破坏老前端** | `Activity` 加字段 | 老前端拿到多字段 | 全部 `omitempty` / 结构不变 ⇒ 向后兼容 |
| R-10 | **两套账本漂移** | 用户在记账模块改了自动建的账 | trip 侧镜像金额过期 | 见未决 **Q7**（推荐：trip 侧镜像行只读 + 以记账模块为准） |

---

## 测试要点（交 `devtools-qa`，可执行 / 可判定）

> ⚠️ **R5 硬约束**：SQLite `:memory:` 测试**必须** `db.SetMaxOpenConns(1)`，否则后台 goroutine
> 会触发「刚插入查不到」的假 404。
> ⚠️ **R11**：需要跨平台产物时走 Docker，不从 macOS 直接出二进制。
> ⚠️ main 上 trip 模块**当前 0 个测试文件**（`ls backend/trip/*_test.go` → no matches）⇒ 本单需**新建**测试文件。

### A. 幂等（PM 边界 4 —— 最高优先级）

| 编号 | 步骤 | 判定 |
|---|---|---|
| **A1** | 对同一 activity 连续 2 次 `PATCH /activities/:id/progress {status:"done", actual_cost_cents:1000}` | `SELECT COUNT(*) FROM expense_transactions WHERE trip_id=? AND activity_id=?` **== 1** |
| **A2** | 承 A1 | `SELECT expense_tx_id FROM trip_activities WHERE id=?` **非空**，且 **==** 上面那行的 `id` |
| **A3** | 承 A1 | 账户 `balance` 只被扣一次：`balance_after == balance_before - 10.0`（**不是** -20.0） |
| **A4** | 承 A1 | `trip_expenses` 中该 `activity_id` 的镜像行**只有 1 条**，`amount_cents == 1000`，`link_tx_id ==` 记账侧 tx id |
| **A5** | 承 A1 | `GET /api/trips/:id/summary` 的 `total_spent_cents` **包含**这 1000 分（镜像行生效 ⇒ 汇总自动含） |
| **A6** | 取消 done（`status:"planned"`）后再勾 done | `COUNT(*) == 1`（**仍然只有 1 笔**；L3 只是优化，L1 才是保证） |
| **A7** | 手工 `INSERT INTO expense_transactions (…, trip_id, activity_id, …)` 第二行同键 | SQLite **拒绝**：`UNIQUE constraint failed`（L1 生效的可证伪断言） |
| **A8** | 并发：对同一 activity 同时发 2 个请求（`go test -race` 或双 goroutine） | 最终 `COUNT(*) == 1`，**无** data race |

### B. 失败不阻断（PM 边界 5）

| 编号 | 步骤 | 判定 |
|---|---|---|
| **B1** | 把 `expense_profiles` 清空（无档案），再标 done | HTTP **200**；`activity.status == "done"` 已落库；`expense_linked == false` |
| **B2** | 同上 | 进程**不 panic**；日志里有 `[trip] 自动建账失败` |
| **B3** | 把 `expense_accounts` 清空（无账户） | 同 B1/B2 |
| **B4** | 开关关闭（`expense_bridge_enabled: false`）后标 done | HTTP 200；`expense_transed` 表**无新行**；`expense_tx_id == ""` |

### C. schema 幂等（PM 边界 3）

| 编号 | 步骤 | 判定 |
|---|---|---|
| **C1** | 连续 2 次执行 `EnsureExpenseColumns` | 第 2 次**无报错**（吞重复列错） |
| **C2** | 在**只有旧列**的 `expense_transactions` 上跑一次 | 新列出现；存量行 `trip_id/activity_id IS NULL` |
| **C3** | 在**只有旧列**的 `trip_activities` 上跑一次 | `status` 默认 `'planned'`、`actual_cost_cents` NULL、`expense_tx_id` `''` |
| **C4** | 老 `trip_expenses` 数据升级后 | `link_tx_id == ''` ⇒ **不被识别为镜像行** |
| **C5** | 升级后建 partial UNIQUE index | **成功**（存量行 NULL ⇒ 不参与唯一性） |
| **C6** | `git diff backend/models/expense.go` | **零改动**（PM 边界 3 的机器可验断言） |

### D. 归属与解析

| 编号 | 步骤 | 判定 |
|---|---|---|
| **D1** | `expense_profiles` 有 **1** 行 + 正常账户分类 | 建账成功；`profile_id` == 那一行 |
| **D2** | `expense_profiles` 有 **2** 行 | 建账**失败**（不猜档案）；`expense_linked == false` |
| **D3** | 请求显式带 `profile_id/account_id/category_id` | 用请求值；覆盖 trip 级默认 |
| **D4** | `category.type == 'income'` 却用来建 expense 账 | 建账失败（校验对齐 `backend/handlers/expense.go:821-824`） |
| **D5** | `account_id` 不属于 `profile_id` | 建账失败 |

### E. 币种

| 编号 | 步骤 | 判定 |
|---|---|---|
| **E1** | `trip.BudgetCurrency == "CNY"` | 建账成功 |
| **E2** | `trip.BudgetCurrency == "HKD"` | **不建账**；`expense_linked == false`；日志说明币种不符 |
| **E3** | `trip.BudgetCurrency == ""`（老数据） | 建账成功（视为 CNY） |

### F. 联动清理与回归

| 编号 | 步骤 | 判定 |
|---|---|---|
| **F1** | 建账后 `DELETE /api/trips/:id` | `expense_transactions WHERE trip_id=?` **0 行**；`trip_expenses` 也 0 行（CASCADE） |
| **F2** | 建账后 `GET /api/trips/:id/expenses` | 返回含镜像行，`link_tx_id` 非空 |
| **F3** | `GET /api/trips/:id/summary` | 响应**结构不变**（字段集合与改动前一致），仅数值含新账 |
| **F4** | `GET /api/trips/:id/summary/markdown` | 明细表里出现这笔 |
| **F5** | `PATCH /api/trips/activities/:activityId`（既有端点，**不带** status） | 行为与改动前**完全一致**（`status` / `actual_cost_cents` **不被清零**）← ⚠️ 重点回归项 |
| **F6** | `PATCH /api/trips/activities/:activityId/progress` 只带 `status`，不带金额 | `actual_cost_cents` 保持原值（不被清空） |
| **F7** | `backend/cmd/trip` CLI 全子命令 | 全部正常（**未装配桥接**，`hooks == nil`） |

### G. 反事实自检（Stage 5 必做）

1. 把 `AutoCreateOnDone` 整体注释掉 ⇒ 触发点仍返回 200（证明「建账失败不阻断」不依赖实现细节）。
2. 把 partial UNIQUE index 去掉 ⇒ A1 **应当失败**（证明 A1 真正依赖 L1，而非碰巧通过）。
3. 把 L2 状态位去掉 ⇒ A1 仍应通过（证明 L2 只是优化，硬保证在 L1）。
4. `git diff origin/main..HEAD -- backend/models/expense.go` ⇒ 空（PM 边界 3）。
5. `git diff origin/main..HEAD -- backend/cmd/trip/main.go` ⇒ 空（CLI 不装配）。

---

## 未决问题（交 PM 在 Stage 2 前逐条拍板）

> 每条给出「可选解释 + 我的倾向」。按 D6：**不自行取舍后静默执行**；下面的「倾向」是默认落地值，
> PM 若不回复则按倾向执行并在 Stage 3 交付说明里注明。

| # | 问题 | 可选解释 | 我的倾向 |
|---|---|---|---|
| **Q1** | **集成形态** | ① 双写同事务（记账=真相源，trip=镜像）② 纯记账侧 + trip 读并集 ③ 只写 trip 自有表 | **① 双写同事务** —— trip 的预算/汇总/Markdown 全部读 `trip_expenses`（`backend/trip/service.go:240`），双写让**读路径零改动**即生效；且两表同 tx 提交 ⇒ 原子 |
| **Q2** | **触发面是否纳入 Stage 3** | ① 一并补最小触发面（`status` + `actual_cost_cents` + 新端点）② Stage 3 只做桥接机制，触发面另开单 | **①** —— main 上**根本没有「勾 done」**（§现状 2），只做②等于交付永不触发的死机制，PM 边界 4 的幂等也无从验收 |
| **Q3** | **重复勾 done 且金额不同** | ① 不覆盖原账，记 warn（幂等 = 不新建也不改）② 覆盖原账金额 | **① 不覆盖** —— 静默改写已入账的金额会让人对不上账；原账保留用户显式核对 |
| **Q4** | **「取消 done」是否连带删账** | ① 不删（只把 `status` 改回 planned，账留着）② 删账 + 回补余额 | **① 不删** —— 账是已经发生的事实，取消勾选不应抹掉已发生的开销；余额回补容易算错 |
| **Q5** | **账户 / 分类绑定** | ① 三级：请求级覆盖 → trip 级默认 → 单档案兜底 ② 仅请求级 ③ 仅 trip 级 | **① 三级** —— ②每次都要前端带，③无法应对多档案；「单档案兜底」既零配置可用又避免建错档案 |
| **Q6** | **币种策略** | ① 仅 `CNY`/空 自动建账，其它币种跳过 + 日志 ② 全部建账，把币种写进 `remark` | **①** —— `expense_transactions` **无币种列**，②会把 HKD/JPY 当 CNY 记进账，金额失真且事后不可察 |
| **Q7** | **镜像行的编辑同步** | ① 镜像行**只读**，以记账模块为准（推荐：改动面小、不会双写冲突）② trip 侧编辑镜像行时**同步写回**记账模块 | **① 只读** —— ②要实现双向同步 + 冲突处理，Stage 3 工作量与 bug 面显著变大；建议后续单再做 |
| **Q8** | **开关默认值** | ① 默认**开**（`yaml` 缺省 = true，需在 `config.Load` 区分「缺省」与「显式 false」）② 默认**关**（灰度后再开） | **② 默认关** —— 新能力零配置上线风险更高；首次开启时人工确认档案/账户/分类配置。⚠️ 实现上要用 `*bool` 或原始 map 判断，**不能**直接用 `bool` 零值 |
| **Q9** | **是否新增「建账状态查询」端点** | ① 不加（`GET /trips/:id/expenses` 已能通过 `link_tx_id` 判断）② 加 `GET /activities/:activityId/expense-link` | **① 不加** —— 少一个端点；`expense_linked` 已在写响应里给出即时反馈 |

### 已在本方案内自行定案（不再请示）

- **ExpenseHook 接口只含 `AutoCreateOnDone` + `DeleteByTrip`** —— 旧分支的 `ListByTrip` / `DeleteByActivity` / `SumByTrip` 在本设计下无调用点，不引入（避免 S1 死代码）。
- **不引入 activity 删除端点** —— main 上没有该能力，超出本单范围。
- **不实现「取消 done → 删账」**（见 Q4 的默认取值）。
- **桥接只在 HTTP 路径装配**，CLI 不装配（§现状 5）。
- **`backend/models/expense.go` / `backend/cmd/trip/main.go` 零改动**。
- **不碰 `backend/models/expense.go` 的既有 `CREATE TABLE`**（PM 边界 3）。

---

## 附录 A · 勘察原始输出

> 全部在 `origin/main` @ `37a1da2`（= 本 worktree HEAD）上执行。

```
$ grep -n 'ExpenseHook\|hooks' backend/trip/service.go
exit=1                                    # 0 命中

$ grep -n 'type Service struct\|func NewService\|func (s \*Service) EnsureSchema\|func (s \*Service) AddExpense\|func (s \*Service) ListExpenses\|func (s \*Service) TripSummary\|func (s \*Service) UpdateActivity\|func (s \*Service) AddActivity\|func (s \*Service) DeleteTrip\|func (s \*Service) DeleteExpense\|func (s \*Service) UpdateExpense\|func (s \*Service) SetReminder' backend/trip/service.go
14:type Service struct {
19:func NewService(repo Repository) *Service {
24:func (s *Service) EnsureSchema(ctx context.Context) error {
78:func (s *Service) DeleteTrip(ctx context.Context, id string) error {
112:func (s *Service) AddActivity(ctx context.Context, a *Activity) error {
130:func (s *Service) UpdateActivity(ctx context.Context, a *Activity) error {
145:func (s *Service) SetReminder(ctx context.Context, activityID string, remindBeforeMinutes int, remindEmails []string) error {
157:func (s *Service) AddExpense(ctx context.Context, e *Expense) error {
175:func (s *Service) UpdateExpense(ctx context.Context, e *Expense) error {
191:func (s *Service) DeleteExpense(ctx context.Context, id string) error {
198:func (s *Service) ListExpenses(ctx context.Context, tripID string) ([]*Expense, error) {
232:func (s *Service) TripSummary(ctx context.Context, tripID string) (*Summary, error) {

$ grep -n 'type Activity struct\|type Expense struct\|ActivityID\|ActualCostCents\|StatusDone\|type ActivityStatus' backend/trip/domain.go
70:type Activity struct {
121:type Expense struct {
130:	ActivityID    string      `json:"activity_id,omitempty"`    # 可选:关联到具体 activity
                                          # ← ActualCostCents / StatusDone / ActivityStatus 全 0 命中

$ grep -n 'CREATE TABLE IF NOT EXISTS trip_activities' -A 16 backend/trip/repository.go
113:	`CREATE TABLE IF NOT EXISTS trip_activities (
114-		id TEXT PRIMARY KEY,
115-		day_id TEXT NOT NULL,
116-		kind TEXT NOT NULL,
117-		title TEXT NOT NULL,
118-		location TEXT NOT NULL DEFAULT '',
119-		destination TEXT NOT NULL DEFAULT '{}',
120-		start_time TEXT NOT NULL DEFAULT '',
121-		duration_min INTEGER NOT NULL DEFAULT 0,
122-		note TEXT NOT NULL DEFAULT '',
123-		"order" INTEGER NOT NULL DEFAULT 0,
124-		remind_before_minutes INTEGER NOT NULL DEFAULT 0,
125-		remind_emails TEXT NOT NULL DEFAULT '[]',
126-		remind_sent_at DATETIME NOT NULL DEFAULT '',
127-		FOREIGN KEY (day_id) REFERENCES trip_days(id) ON DELETE CASCADE
128-	)`,
                                          # ← 无 status / actual_cost_cents 列

$ grep -n 'CREATE TABLE IF NOT EXISTS trip_expenses' -A 15 backend/trip/repository.go
134:	`CREATE TABLE IF NOT EXISTS trip_expenses (
135-		id TEXT PRIMARY KEY,
136-		trip_id TEXT NOT NULL,
137-		date TEXT NOT NULL,
138-		category TEXT NOT NULL,
139-		amount_cents INTEGER NOT NULL,
140-		currency TEXT NOT NULL DEFAULT '',
141-		note TEXT NOT NULL DEFAULT '',
142-		payment_method TEXT NOT NULL DEFAULT '',
143-		activity_id TEXT NOT NULL DEFAULT '',
144-		created_at DATETIME NOT NULL,
145-		updated_at DATETIME NOT NULL,
146-		FOREIGN KEY (trip_id) REFERENCES trip_trips(id) ON DELETE CASCADE
147-	)`,

$ grep -n 'func (r \*SQLiteRepository) AddActivity\|func (r \*SQLiteRepository) UpdateActivity\|func (r \*SQLiteRepository) GetActivity\|func (r \*SQLiteRepository) DeleteActivity\|func (r \*SQLiteRepository) ListExpensesByTrip\|func (r \*SQLiteRepository) AddExpense\|func isDuplicateColumnErr\|func (r \*SQLiteRepository) EnsureSchema' backend/trip/repository.go
90:func (r *SQLiteRepository) EnsureSchema(ctx context.Context) error {
170:func isDuplicateColumnErr(err error) bool {
377:func (r *SQLiteRepository) AddActivity(ctx context.Context, a *Activity) error {
414:func (r *SQLiteRepository) UpdateActivityReminder(ctx context.Context, activityID string, remindBeforeMinutes int, remindEmails []string) error {
431:func (r *SQLiteRepository) DeleteActivity(ctx context.Context, id string) error {
437:func (r *SQLiteRepository) GetActivity(ctx context.Context, id string) (*Activity, error) {
449:func (r *SQLiteRepository) UpdateActivity(ctx context.Context, a *Activity) error {
471:func (r *SQLiteRepository) AddExpense(ctx context.Context, e *Expense) error {
529:func (r *SQLiteRepository) ListExpensesByTrip(ctx context.Context, tripID string) ([]*Expense, error) {

$ grep -n 'type Repository interface' -A 30 backend/trip/repository.go
23:type Repository interface {
24-	EnsureSchema(ctx context.Context) error
   …（Trip / DayPlan / Activity / Expense 四组方法）

$ grep -n 'func NewTripHandler\|repo := trip.NewSQLiteRepository\|svc := trip.NewService\|svc.EnsureSchema\|func (h \*TripHandler) UpdateActivity\|func (h \*TripHandler) AddExpense\|func (h \*TripHandler) ListExpenses\|type tripUpdateActivityRequest\|type tripExpenseRequest\|type tripAddActivityRequest' backend/handlers/trip.go
27:func NewTripHandler(db *models.DB, notifCfg notif.Config) *TripHandler {
28:	repo := trip.NewSQLiteRepository(db.Conn())
29:	svc := trip.NewService(repo)
30:	if err := svc.EnsureSchema(context.Background()); err != nil {
172:type tripAddActivityRequest struct {
339:type tripExpenseRequest struct {
350:func (h *TripHandler) AddExpense(c *gin.Context) {
386:func (h *TripHandler) ListExpenses(c *gin.Context) {
473:type tripUpdateActivityRequest struct {
487:func (h *TripHandler) UpdateActivity(c *gin.Context) {

$ grep -n 'RegisterTripRoutes\|activities/:activityId\|:id/expenses\|:id/budget' backend/routes/trip.go
9:// RegisterTripRoutes 挂载 /api/trips/* 全部端点。
12:func RegisterTripRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
30:		t.PUT("/:id/budget", createRateLimiter.Middleware(), h.TripHandler.UpdateBudget)
33:		t.GET("/:id/expenses", h.TripHandler.ListExpenses)
34:		t.POST("/:id/expenses", createRateLimiter.Middleware(), h.TripHandler.AddExpense)
44:		t.PUT("/activities/:activityId/reminder", createRateLimiter.Middleware(), h.TripHandler.SetReminder)
46:		t.PATCH("/activities/:activityId", createRateLimiter.Middleware(), h.TripHandler.UpdateActivity)
                                          # ← 无 DELETE /activities/...、无 status 相关端点

$ grep -n 'RegisterTripRoutes' backend/routes/index.go
93:	RegisterTripRoutes(api, h, createRateLimiter)

$ grep -n 'NewTripHandler' backend/app_handlers.go
37:	tripHandler := handlers.NewTripHandler(db, notif.Config{

$ grep -n 'db.InitAll()\|models.NewDB' backend/app_runtime.go
47:	db, err := models.NewDB(dbPath)
52:	if err := db.InitAll(); err != nil {

$ grep -n 'func (db \*DB) Conn' backend/models/paste.go
37:func (db *DB) Conn() *sql.DB {

$ grep -n 'CREATE TABLE IF NOT EXISTS expense_transactions\|ALTER TABLE expense_transactions\|func (db \*DB) CreateExpenseTransaction\|type ExpenseTransaction struct' backend/models/expense.go
53:type ExpenseTransaction struct {
134:		CREATE TABLE IF NOT EXISTS expense_transactions (
164:	db.conn.Exec("ALTER TABLE expense_transactions ADD COLUMN voice_text TEXT DEFAULT ''")
407:func (db *DB) CreateExpenseTransaction(tx *ExpenseTransaction) error {

$ grep -c 'trip_id\|activity_id' backend/models/expense.go
0

$ grep -n 'func (h \*ExpenseHandler) CreateTransaction\|UpdateExpenseAccountBalance\|func (h \*ExpenseHandler) GetTransactions' backend/handlers/expense.go
742:func (h *ExpenseHandler) GetTransactions(c *gin.Context) {
777:func (h *ExpenseHandler) CreateTransaction(c *gin.Context) {
850:	h.db.UpdateExpenseAccountBalance(req.AccountID, newBalance)
954:	h.db.UpdateExpenseAccountBalance(oldTx.AccountID, oldBalance)
971:	h.db.UpdateExpenseAccountBalance(tx.AccountID, newBalance)
1008:	h.db.UpdateExpenseAccountBalance(tx.AccountID, newBalance)

$ grep -n 'type ExpenseConfig struct' -A 4 backend/config/config.go
340:type ExpenseConfig struct {
341-	DefaultExpiresDays int `yaml:"default_expires_days"` // 默认过期天数，默认365
342-	MaxDataSize        int `yaml:"max_data_size"`        // 最大数据大小，默认1MB
343-}

$ grep -n 'defaultDBPath' backend/cmd/trip/main.go
50:const defaultDBPath = "./data/trip.db"
75:		dbPath = defaultDBPath

$ grep -n "path: '/trip'" -A 8 frontend/src/router/index.js
354:    path: '/trip',
355-    name: 'Trip',
356-    component: () => import('../views/life/TripTool.vue'),
357-    meta: {
358-      title: '旅游行程',
359-      icon: 'Guide',
360-      category: 'life',
361-      description: '行程规划 + 落地 fixture + 邮件提醒'
362-    }

$ ls backend/trip/*_test.go backend/handlers/trip_test.go
zsh: no matches found: backend/trip/*_test.go      # trip 模块 0 个测试文件

$ git ls-tree -r --name-only refs/remotes/origin/main -- backend/trip/
backend/trip/README.md
backend/trip/domain.go
backend/trip/reminder.go
backend/trip/repository.go
backend/trip/seed.go
backend/trip/seed_output_sample.md
backend/trip/service.go
```

## 附录 B · 旧分支（只读参考）关键行

```
$ git show refs/heads/agent/devtools-engineer/devt-20-d5efcf63a034:backend/trip/service.go | grep -n 'ExpenseHook\|hooks\|AutoCreateOnDone'
11:// ExpenseHook 把「勾 done → 自动建账」这类跨域副作用抽出来,
16:type ExpenseHook interface {
17-	// AutoCreateOnDone 在活动被标记 done + 有 actual_cost_cents 时调用,
20:	AutoCreateOnDone(ctx context.Context, t *Trip, day *DayPlan, a *Activity) (string, error)
50:	hooks ExpenseHook // 跨域副作用(记账);nil 时跳过(单元测试 + CLI 默认)
59:func (s *Service) SetHooks(h ExpenseHook) {
118:	if s.hooks != nil {
120-		_ = s.hooks.DeleteByTrip(ctx, id)
184:	if a.Status == StatusDone && a.ActualCostCents != nil && *a.ActualCostCents > 0 && s.hooks != nil {
187-			_, _ = s.hooks.AutoCreateOnDone(ctx, t, dp, a)
231:	if s.hooks != nil {
232:		_ = s.hooks.DeleteByActivity(ctx, id)

$ git show refs/heads/agent/devtools-engineer/devt-20-d5efcf63a034:backend/trip/expense.go | grep -n 'EnsureExpenseColumns\|ALTER TABLE\|CREATE INDEX\|SELECT id FROM expense_transactions\|INSERT INTO expense_transactions\|UPDATE expense_accounts'
33:func (b *sqlExpenseBridge) EnsureExpenseColumns(ctx context.Context) error {
35:		`ALTER TABLE expense_transactions ADD COLUMN trip_id TEXT DEFAULT NULL`,
36:		`ALTER TABLE expense_transactions ADD COLUMN activity_id TEXT DEFAULT NULL`,
39:		_, _ = b.db.ExecContext(ctx, s) // duplicate column → 吞错
41:	_, _ = b.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_expense_trip_id ON expense_transactions(trip_id)`)
42:	_, _ = b.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_expense_activity_id ON expense_transactions(activity_id)`)
98:	err := b.db.QueryRowContext(ctx,
99:		`SELECT id FROM expense_transactions WHERE trip_id = ? AND activity_id = ? LIMIT 1`,
127:	if _, err := b.db.ExecContext(ctx,
128:		`INSERT INTO expense_transactions
129-		 (id, profile_id, account_id, category_id, amount, type, date, remark, tags, voice_text, trip_id, activity_id, created_at, updated_at)
130:		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
138:	if _, err := b.db.ExecContext(ctx,
139:		`UPDATE expense_accounts SET balance = balance - ?, updated_at = ? WHERE id = ?`,

$ git show refs/heads/agent/devtools-engineer/devt-20-d5efcf63a034:backend/handlers/trip.go | grep -n 'SetHooks\|NewSQLExpenseBridge'
28:	conn := db.Conn()
32:	svc.SetHooks(trip.NewSQLExpenseBridge(conn))
```

> ⚠️ 旧分支 `expense.go` 的幂等靠 **「先 SELECT 再 INSERT」**（`:98-104`）—— 这在并发下**会重复建账**。
> 本方案的 L1（partial UNIQUE index）正是为了修掉这个缺陷。

---

## 附录 C · 前端影响面（**Stage 3 不做**，供 PM 开前端兄弟单）

| 项 | 内容 |
|---|---|
| 视图 | `frontend/src/views/life/TripTool.vue` —— 活动行加「✓ 完成」勾选 + 实际金额输入 → `PATCH /api/trips/activities/:id/progress` |
| 路由 | **无需新增条目** —— `/trip` 已存在（`frontend/src/router/index.js:354-362`，`meta.title:'旅游行程'` / `meta.icon:'Guide'`） |
| 状态展示 | 用写响应里的 `expense_linked` / `expense_tx_id` 显示「已入账 / 未入账」 |
| R3 | Vue 3 模板里**永远不要写 `.value`**（自动 unwrap，写了白屏） |
| R4 | 模板裸下标 → `obj.x?.[0] \|\| ''`（字段缺失会炸挂载） |
| 依赖 | 无新依赖 |

> Stage 3 后端单**不得**碰 `frontend/`（PM 边界 6）。
