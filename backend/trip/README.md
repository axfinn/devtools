# 旅游行程助手(trip)模块

> DevTools 第 42 个模块 ——「旅游行程助手」限界上下文。
> 聚合根 `Trip` → 实体 `DayPlan` → 实体 `Activity`;值对象 `Destination`、`ActivityKind`。
> 仓储 `Repository` + 服务 `Service` + Markdown 渲染器,全部在 `backend/trip/` 内。
> CLI 入口:`backend/cmd/trip/main.go`(构建产物 `trip`)。

---

## 1. 设计概览(DDD)

```
┌──────────────────────────── 限界上下文:trip ────────────────────────────┐
│                                                                         │
│   Trip(聚合根) ──owns──▶ DayPlan(实体) ──owns──▶ Activity(实体)         │
│       │                                       │                          │
│       │ owns references to                    │ references               │
│       ▼                                       ▼                          │
│   Destination(值对象) ──────────────  trip_destinations 表                │
│                                                                         │
│   Repository(仓储接口) ←─ SQLiteRepository(基础设施)                     │
│                                                                         │
│   Service(领域服务)                                                    │
│     ├── CreateTrip / AddDay / AddActivity                                │
│     ├── Show / List / DeleteTrip                                        │
│     └── RenderMarkdown(trip) → string                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

| 层 | 文件 | 说明 |
|----|------|------|
| 领域 | `backend/trip/domain.go` | `Trip` / `DayPlan` / `Activity` / `Destination` / `ActivityKind` |
| 仓储 | `backend/trip/repository.go` | `Repository` 接口 + `SQLiteRepository` 实现 + `OpenSQLite` / `InitSchema` |
| 服务 | `backend/trip/service.go` | 业务用例 + Markdown 渲染器 |
| Fixture | `backend/trip/seed.go` | `SeedFirstTrip` —— 国庆 2026 粤港澳潮汕 14 日深度游 |
| CLI | `backend/cmd/trip/main.go` | `trip create / list / show / delete / add-day / add-activity / days / activities / seed / init` |

---

## 2. 数据模型

| 表 | 主键 | 关键字段 |
|----|------|---------|
| `trip_trips` | `id`(12-hex) | `name` / `start_date` / `end_date` / `summary` / `cities` / `tags` / 时间戳 |
| `trip_day_plans` | `id` | `trip_id` / `day_index` / `date` / `city` / `title` / `summary` |
| `trip_activities` | `id` | `day_plan_id` / `seq` / `kind` / `time` / `title` / `location` / `notes` / `destination_id` |
| `trip_destinations` | `id` | `name` / `region` / `info`(可空,upsert by name+region) |

`ActivityKind` 枚举:`transport` 🚆 / `sight` 🗺️ / `food` 🍜 / `lodging` 🏨 / `shopping` 🛍️ / `note` 📝。

> 命名空间:表名统一 `trip_*` 前缀,与现有 41 个模块无碰撞。

---

## 3. CLI 用法

构建:

```bash
cd backend
go build -o /tmp/trip ./cmd/trip/
```

### 3.1 子命令一览

| 子命令 | 用途 |
|--------|------|
| `trip init` | 建表(幂等) |
| `trip seed [--force]` | 落地第一个行程(已存在则跳过;`--force` 重建) |
| `trip create --name N --start S --end E [--summary X] [--cities X] [--tags X]` | 新建行程 |
| `trip list [--json]` | 列出所有行程 |
| `trip show <trip-id>` | 渲染完整 Markdown 行程单到 stdout |
| `trip delete <trip-id>` | 删除行程(级联 days + activities) |
| `trip add-day <trip-id> --date S [--city X] [--title X] [--summary X]` | 加一天 |
| `trip add-activity <day-id> --kind K --title N [--time X] [--location X] [--notes X]` | 加一项活动 |
| `trip days <trip-id>` | 列出某 trip 的所有 DayPlan |
| `trip activities <day-id>` | 列出某天的所有 Activity |
| `trip help` | 详细帮助 |

> 全局:`--db PATH` 切换 SQLite 路径(默认 `./data/trip.db`)。

### 3.2 完整示例 —— 启动到看到 14 天攻略

```bash
# 1. 初始化数据库(幂等)
./trip --db ./data/trip.db init

# 2. 落地第一个行程
./trip --db ./data/trip.db seed
# → trip: seeded 国庆 2026 粤港澳潮汕 14 日深度游 (id=da38a9e82a8e, 14 days)

# 3. 列出
./trip --db ./data/trip.db list
# NAME                              START         END           ID            CITIES
# ---------------------------------------------------------------------------------
# 国庆 2026 粤港澳潮汕 14 日深度游         2026-09-24    2026-10-07    da38a9e82a8e  深圳,香港,...

# 4. 渲染 Markdown 攻略
./trip --db ./data/trip.db show da38a9e82a8e > 国庆攻略.md
```

### 3.3 命令交互

`--flag value` 与位置参数可以任意穿插,如:

```bash
trip add-day TRIP_ID --date 2026-10-08 --city 深圳 --title "D15 缓冲日"
trip add-day --date 2026-10-08 --city 深圳 --title "D15 缓冲日" TRIP_ID
trip add-day TRIP_ID --date=2026-10-08 --city=深圳
```

未知 flag 会立即报错。

---

## 4. Markdown 渲染器

`Service.RenderMarkdown(*Trip) string` 输出:

- `# 行程名` + 一句话 `>` summary
- 概览表(日期 / 城市 / 标签 / 总天数 / ID)
- 每个 DayPlan 一段 `## Day 标题`,包含日期 · 城市 + 当天 summary + 按 kind 排序的 bullet 列表
- 每个 Activity 一行 bullet:`<icon> **<title>** _(时刻:<time>)_`,下方缩进 `地点:` 和 `备注:`

排序保证阅读节奏:transport → sight → food → lodging → shopping → note。

样例片段:

```markdown
## D2 深圳 → 香港

**2026-09-25** · 香港

> 上午过关走东铁,下午尖沙咀 + 维港,晚上太平山顶看幻彩咏香江。

- 🚆 **福田 / 罗湖口岸过关** _(时刻:上午)_
  - 地点: 福田口岸 / 罗湖口岸
  - 建议福田口岸过 → 落马洲 → 东铁线红磡,避开罗湖早高峰
- 🗺️ **尖沙咀 + 维多利亚港星光大道** _(时刻:下午)_
  - 地点: 尖沙咀
  - 李小龙铜像、天星小轮钟楼、1881 Heritage
```

---

## 5. 第一个真实行程 —— 国庆 2026 粤港澳潮汕 14 日深度游

来源:用户提供。落地为 `seed.go` 中的 `seedDays` 切片(14 天 / 70+ 项 activity)。

| Day | 日期 | 城市 | 标题 |
|-----|------|------|------|
| D1 | 2026-09-24 | 深圳 | 深圳集合日 |
| D2 | 2026-09-25 | 香港 | 深圳 → 香港 |
| D3 | 2026-09-26 | 香港 | 香港迪士尼或海洋公园 |
| D4 | 2026-09-27 | 香港 | 香港经典一日 |
| D5 | 2026-09-28 | 澳门 | 香港 → 澳门 |
| D6 | 2026-09-29 | 澳门 | 澳门深度 |
| D7 | 2026-09-30 | 珠海 | 澳门 → 珠海 |
| D8 | 2026-10-01 | 广州 | 珠海 → 广州(国庆当天) |
| D9 | 2026-10-02 | 广州 | 广州经典一日 |
| D10 | 2026-10-03 | 汕头 | 广州 → 潮汕(汕头) |
| D11 | 2026-10-04 | 南澳岛 | 汕头 → 南澳岛 |
| D12 | 2026-10-05 | 潮州 | 南澳 → 潮州 |
| D13 | 2026-10-06 | 揭阳 | 潮州 → 揭阳 / 返程缓冲日 |
| D14 | 2026-10-07 | 散团 | 散团返程 |

种子幂等:按 `Name == "国庆 2026 粤港澳潮汕 14 日深度游"` 检测,已存在则直接返回不重建;`--force` 强制重建。

---

## 6. 与现有 41 个模块的关系

| 维度 | 现状 |
|------|------|
| HTTP 路由 | 本期未挂路由(CLI-only);Service + Repository 接口可被未来 `backend/handlers/trip.go` 直接复用 |
| 数据库 | 独立 SQLite 文件 `./data/trip.db`(可用 `--db` 切换) |
| 鉴权 | 无 —— 纯 CLI / 本地数据;若挂路由,建议套 `cfg.Console.AdminPassword` 或独立 password 字段 |
| 软依赖 | 无 —— 不依赖 Redis / 外部服务 |

后续若要接入前端,只需:
1. 新建 `frontend/src/views/life/TripTool.vue`(参照 Recipe / PlannerTool 风格)
2. 在 `frontend/src/router/index.js` 加路由条目
3. 在 `backend/handlers/trip.go` 实现 Gin handler,直接调 `trip.NewService(repo)`
4. 在 `backend/routes/trip.go` 注册 + `routes/index.go` 接入 `RegisterAllRoutes`

---

## 7. 验证记录

```bash
$ cd backend && go build ./...
(trip 包 + cmd/trip 全部编译通过)

$ go vet ./trip/... ./cmd/trip/...
(无告警)

$ /tmp/trip --db /tmp/trip-final.db init
trip: schema initialized at /tmp/trip-final.db

$ /tmp/trip --db /tmp/trip-final.db seed
trip: seeded 国庆 2026 粤港澳潮汕 14 日深度游 (id=da38a9e82a8e, 14 days)

$ /tmp/trip --db /tmp/trip-final.db list
NAME                              START         END           ID            CITIES
------------------------------------------------------------------------------------------------
国庆 2026 粤港澳潮汕 14 日深度游         2026-09-24    2026-10-07    da38a9e82a8e  深圳,香港,澳门,珠海,广州,汕头,南澳岛,潮州,揭阳

$ /tmp/trip --db /tmp/trip-final.db show da38a9e82a8e | wc -l
296
```

完整 show 输出 296 行 Markdown,已在 `backend/trip/` 同目录作为 `seed_output.md` 留档(可选)。
