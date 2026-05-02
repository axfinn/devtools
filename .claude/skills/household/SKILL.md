---
name: household
description: 家庭物品整理 — 物品库存管理、位置标注、到期提醒、AI 智能添加/分析、小票 OCR 识别。触发：整理物品、添加物品、库存管理、物品快过期、收纳整理
triggers:
  - "整理物品"
  - "库存管理"
  - "添加物品"
  - "物品整理"
  - "收纳"
  - "household"
---

# 家庭物品整理 (Household)

通过 DevTools 后端 API 管理家庭物品库存、位置和到期提醒。后端默认运行在 `https://t.jaxiu.cn`。

## 认证

初始化后会获得 token（存储在前端 localStorage），所有操作需要 token。

## 1. 初始化 & 物品管理

### 1.1 初始化

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/init \
  -H "Content-Type: application/json" \
  -d '{}'
# 返回: {"token":"xxx"}
```

### 1.2 添加物品

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/items \
  -H "Content-Type: application/json" \
  -d '{
    "token": "xxx",
    "name": "洗衣液",
    "category": "日用品",
    "quantity": 2,
    "unit": "瓶",
    "location": "阳台柜子",
    "expire_date": "2027-01-01",
    "min_stock": 1,
    "notes": "蓝月亮薰衣草味"
  }'
```

常用 category: 食品饮料、日用品、药品、化妆品、衣物、电子、文具、其他

### 1.3 查看物品列表

```bash
# 全部物品
curl -s "https://t.jaxiu.cn/api/household/items?token=xxx"

# 按分类筛选
curl -s "https://t.jaxiu.cn/api/household/items?token=xxx&category=日用品"

# 按位置筛选
curl -s "https://t.jaxiu.cn/api/household/items?token=xxx&location=阳台柜子"

# 低库存提醒
curl -s "https://t.jaxiu.cn/api/household/items?token=xxx&low_stock=true"

# 即将过期
curl -s "https://t.jaxiu.cn/api/household/items?token=xxx&expiring=true"
```

### 1.4 使用/补充物品

```bash
# 使用物品（减少库存）
curl -s -X POST https://t.jaxiu.cn/api/household/items/{item_id}/use \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","quantity":1}'

# 补充物品（增加库存）
curl -s -X POST https://t.jaxiu.cn/api/household/items/{item_id}/restock \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","quantity":2}'

# 开封物品
curl -s -X POST https://t.jaxiu.cn/api/household/items/{item_id}/open \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx"}'
```

### 1.5 更新/删除物品

```bash
# 更新
curl -s -X PUT https://t.jaxiu.cn/api/household/items/{item_id} \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","name":"新名称","quantity":5,"location":"新位置"}'

# 删除
curl -s -X DELETE "https://t.jaxiu.cn/api/household/items/{item_id}?token=xxx"
```

## 2. 模板管理

### 2.1 创建物品模板

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/templates \
  -H "Content-Type: application/json" \
  -d '{
    "token": "xxx",
    "name": "牛奶",
    "category": "食品饮料",
    "unit": "盒",
    "default_location": "冰箱",
    "min_stock": 3,
    "expire_days": 7
  }'
```

### 2.2 查看模板

```bash
curl -s "https://t.jaxiu.cn/api/household/templates?token=xxx"
```

## 3. 通知 & 待办

### 3.1 查看通知

```bash
curl -s "https://t.jaxiu.cn/api/household/notifications?token=xxx"
```

### 3.2 标记已读

```bash
# 单条
curl -s -X POST https://t.jaxiu.cn/api/household/notifications/{notif_id}/read \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx"}'

# 全部已读
curl -s -X POST https://t.jaxiu.cn/api/household/notifications/read-all \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx"}'
```

### 3.3 待购清单

```bash
# 查看
curl -s "https://t.jaxiu.cn/api/household/todos?token=xxx"

# 创建
curl -s -X POST https://t.jaxiu.cn/api/household/todos \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","title":"购买牙膏","quantity":2,"priority":"medium"}'

# 完成/更新
curl -s -X PUT https://t.jaxiu.cn/api/household/todos/{todo_id} \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","done":true}'
```

## 4. AI 智能功能

### 4.1 AI 添加物品（自然语言）

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/ai/add \
  -H "Content-Type: application/json" \
  -d '{
    "token": "xxx",
    "text": "冰箱里有3盒安慕希酸奶，保质期到6月30号，还有2斤车厘子"
  }'
```

### 4.2 AI 分析库存

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/ai/analyze \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx"}'
# 返回: 库存健康度、过期风险、补货建议
```

### 4.3 AI 推荐补充

```bash
curl -s "https://t.jaxiu.cn/api/household/ai/restock?token=xxx"
# 基于使用频率和低库存自动推荐需要购买的物品
```

### 4.4 AI 解析物品清单

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/ai/parse \
  -H "Content-Type: application/json" \
  -d '{
    "token": "xxx",
    "text": "超市购物清单：牛奶2盒、面包1袋、鸡蛋30个、洗衣液1瓶"
  }'
```

### 4.5 AI 合并待办

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/ai/todos/merge \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx"}'
# 智能合并相似的待购项
```

## 5. 小票 OCR

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/ocr/receipt \
  -H "Content-Type: application/json" \
  -d '{
    "token": "xxx",
    "image": "data:image/png;base64,iVBORw0KGgo..."
  }'
# 识别小票上的商品、金额、日期，自动添加到库存
```

## 6. 条码查询

```bash
curl -s -X POST https://t.jaxiu.cn/api/household/barcode/lookup \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","barcode":"6901234567890"}'
```

## 7. 统计

```bash
curl -s "https://t.jaxiu.cn/api/household/stats?token=xxx"
# 返回: 总物品数、分类分布、低库存数、即将过期数、本月消耗/补充统计
```

## 8. 对话功能

```bash
# 发送消息
curl -s -X POST https://t.jaxiu.cn/api/household/chat \
  -H "Content-Type: application/json" \
  -d '{"token":"xxx","message":"冰箱里有什么吃的？"}'

# 查看历史
curl -s "https://t.jaxiu.cn/api/household/chat/history?token=xxx"

# 清除历史
curl -s -X DELETE "https://t.jaxiu.cn/api/household/chat/history?token=xxx"
```

## 快速操作

当用户说"家里还有多少牛奶"或"整理一下冰箱"时，使用物品列表查询。
当用户说"我买了xxx"时，使用 AI 添加物品 API。
当用户说"帮我整理超市小票"时，使用小票 OCR。
当用户说"什么东西快过期了"时，使用 expiring=true 筛选。

Base URL: `https://t.jaxiu.cn`
