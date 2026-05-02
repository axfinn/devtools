---
name: expense
description: 生活记账 — 创建账本、记录收支、分类管理、统计分析、AI 语音记账。触发：记账、花了多少钱、今日开销、月度统计、添加支出、语音记账
triggers:
  - "记账"
  - "记账本"
  - "花销"
  - "支出"
  - "收入"
  - "财务统计"
  - "语音记账"
  - "expense"
---

# 生活记账 (Expense)

通过 DevTools 后端 API 管理个人财务。后端默认运行在 `https://t.jaxiu.cn`。

## 认证

创建账本后获得 `creator_key`，所有操作需要它。也支持 `password` 创建，通过登录获取 key。

## 1. 账本管理

### 1.1 创建账本

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的账本",
    "password": "your_password"
  }'
# 返回: {"id":"xxx","creator_key":"xxx"}
```

### 1.2 登录

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense/login \
  -H "Content-Type: application/json" \
  -d '{"id":"profile_id","password":"your_password"}'
```

### 1.3 延长过期

```bash
curl -s -X PUT https://t.jaxiu.cn/api/expense/{id}/extend \
  -H "Content-Type: application/json" \
  -d '{"creator_key":"xxx","expires_in":90}'
```

## 2. 账户管理

### 2.1 创建账户

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense/{id}/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "name": "工商银行储蓄卡",
    "balance": 50000.00,
    "currency": "CNY"
  }'
```

### 2.2 查看账户列表

```bash
curl -s "https://t.jaxiu.cn/api/expense/{id}/accounts?creator_key=xxx"
```

### 2.3 更新账户

```bash
curl -s -X PUT https://t.jaxiu.cn/api/expense/{id}/accounts/{accountId} \
  -H "Content-Type: application/json" \
  -d '{"creator_key":"xxx","name":"新名称","balance":45000.00}'
```

## 3. 分类管理

### 3.1 创建分类

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense/{id}/categories \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "name": "餐饮",
    "type": "expense",
    "icon": "food",
    "color": "#ff6b6b",
    "budget": 3000.00
  }'
```

type 可选: `income`（收入）、`expense`（支出）

常用分类（支出）: 餐饮、交通、购物、住房、娱乐、医疗、教育、通讯、人情、其他
常用分类（收入）: 工资、奖金、理财、兼职、报销、其他

### 3.2 查看分类

```bash
curl -s "https://t.jaxiu.cn/api/expense/{id}/categories?creator_key=xxx"
```

## 4. 交易记录

### 4.1 创建交易

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense/{id}/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "type": "expense",
    "amount": 35.50,
    "category_id": "category_id",
    "account_id": "account_id",
    "description": "午餐 - 牛肉面",
    "date": "2026-04-30T12:30:00+08:00",
    "tags": ["午餐", "牛肉面"]
  }'
```

type: `income` 或 `expense`

### 4.2 查看交易列表

```bash
# 全部
curl -s "https://t.jaxiu.cn/api/expense/{id}/transactions?creator_key=xxx"

# 按月筛选
curl -s "https://t.jaxiu.cn/api/expense/{id}/transactions?creator_key=xxx&year=2026&month=4"

# 按分类筛选
curl -s "https://t.jaxiu.cn/api/expense/{id}/transactions?creator_key=xxx&category_id=xxx"
```

### 4.3 更新/删除交易

```bash
# 更新
curl -s -X PUT https://t.jaxiu.cn/api/expense/{id}/transactions/{txId} \
  -H "Content-Type: application/json" \
  -d '{"creator_key":"xxx","amount":40.00,"description":"更正金额"}'

# 删除
curl -s -X DELETE "https://t.jaxiu.cn/api/expense/{id}/transactions/{txId}?creator_key=xxx"
```

## 5. 统计分析

```bash
curl -s "https://t.jaxiu.cn/api/expense/{id}/stats?creator_key=xxx&year=2026&month=4"
```

返回: 总收入、总支出、结余、分类汇总、日趋势等。

## 6. AI 功能

### 6.1 AI 分析财务

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense/{id}/analyze \
  -H "Content-Type: application/json" \
  -d '{"creator_key":"xxx"}'
```

### 6.2 AI 语音记账

```bash
curl -s -X POST https://t.jaxiu.cn/api/expense/{id}/voice-parse \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "text": "今天中午吃牛肉面花了35块5"
  }'
# AI 自动解析金额、分类、日期等
```

## 快速操作

当用户说"记账 35元 午餐"或"今天花了xxx"时，调用创建交易 API。
当用户说"本月开销"或"财务状况"时，调用统计 API。
当用户说"语音记账"时，使用 AI voice-parse API。

Base URL: `https://t.jaxiu.cn`
