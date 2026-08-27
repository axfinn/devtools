# 存储归一方案

## 现状

DevTools 目前使用混合存储策略：

| 数据 | 存储位置 | 说明 |
|------|---------|------|
| 主要业务数据 | SQLite (`data/paste.db`) | paste/shorturl/chat/mdshare/excalidraw/photowall/terminal 等 |
| Proxy 配置 | `./data/proxy_config.json` | 节点列表、订阅 URL、路由模式 |
| Proxy 订阅缓存 | `./data/subscription_cache.json` | 订阅格式缓存 |
| Proxy 订阅备份 | `./data/subscription_latest.yaml` | 最近一次完整订阅 YAML |
| 上传文件 | `./data/uploads/` | 聊天室文件、TTS 音频等 |

## 目标

将 proxy 相关 JSON 文件统一到 SQLite，消除文件系统依赖，简化备份/迁移。

## 影响范围

### 1. `data/proxy_config.json` → `proxy_nodes` 表

当前 `proxyPersist` 结构：
```json
{
  "source_url": "",
  "source_urls": [],
  "yaml_content": "",
  "routing_mode": "smart",
  "default_node_name": "",
  "default_node_regex": "",
  "ai_node_name": "",
  "ai_node_regex": "",
  "nodes": [/* ProxyNode array */]
}
```

对应 SQLite schema：
```sql
CREATE TABLE proxy_nodes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    server     TEXT NOT NULL,
    port       INTEGER NOT NULL,
    extra      TEXT,  -- JSON
    latency    INTEGER DEFAULT -1,
    status     TEXT,  -- "unsupported" 等
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE proxy_config (
    id                    INTEGER PRIMARY KEY DEFAULT 1,
    source_url           TEXT,
    source_urls          TEXT,  -- JSON array
    yaml_content         TEXT,
    routing_mode         TEXT DEFAULT 'smart',
    default_node_name    TEXT,
    default_node_regex   TEXT,
    ai_node_name         TEXT,
    ai_node_regex        TEXT,
    updated_at           DATETIME DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT single_row CHECK (id = 1)
);
```

### 2. `data/subscription_cache.json` → `proxy_subscription_cache` 表

```sql
CREATE TABLE proxy_subscription_cache (
    source_url TEXT PRIMARY KEY,
    content    TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3. 订阅 YAML 备份 → `proxy_subscription_backup` 表

```sql
CREATE TABLE proxy_subscription_backup (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    content   TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 迁移步骤

### Phase 1：双写阶段（向后兼容）
- 添加 `models/proxy.go`，实现 SQLite 读写
- `proxy.go` 中的 `loadPersistedProxy()` 同时读取 SQLite 和 JSON，以 SQLite 优先
- `savePersistedProxy()` 同时写入 SQLite 和 JSON（保持 JSON 兼容）
- 验证数据一致性

### Phase 2：SQLite only
- 移除 JSON 文件读写
- JSON 文件在下次启动时由迁移逻辑读取并删除
- 添加 `migration_vX_to_vY()` 函数处理版本升级

### Phase 3：清理
- 移除 `handlers/proxy.go` 中的 JSON 相关代码
- 删除 `proxyPersist` 结构（替换为 DB model）
- 更新 `data/proxy_config.json` 等文件的 .gitignore 条目

## 风险与回滚

- **回滚**：保留 JSON 文件直到 Phase 3 确认稳定
- **风险**：订阅状态迁移期间可能丢失最近刷新状态 → 通过备份机制缓解
- **影响**：proxy.go 改动较大（约 5400 行），建议在独立分支操作

## 实施顺序

1. 创建 `backend/models/proxy_config.go`（表定义 + 基础 CRUD）
2. 在 `handlers/proxy.go` 中引入双写
3. 添加迁移函数 `migrateProxyJSONToSQLite()`
4. 验证后切换到 SQLite-only
5. 清理 JSON 代码和文件

## 相关文件

- `backend/handlers/proxy.go` — 主要改动文件（~5400 行）
- `backend/models/` — 新增 `proxy_config.go`
- `backend/routes/proxy.go` — 路由（不变）
- `backend/app_runtime.go` — 启动时调用迁移
