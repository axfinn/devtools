---
name: recipe
description: 每日菜谱 — 查看/创建/更新菜谱，支持详细步骤、AI 推荐。触发：今天吃什么、菜谱、做饭、推荐菜
triggers:
  - "菜谱"
  - "今天吃什么"
  - "做饭"
  - "推荐菜"
  - "recipe"
---

# 每日菜谱 (Recipe)

通过 DevTools 后端 API 管理菜谱。后端默认运行在 `https://t.jaxiu.cn`。

## 1. 获取今日菜谱

```bash
# 今日推荐（无需登录）
curl -s https://t.jaxiu.cn/api/recipe/default

# 带详细步骤的菜谱
curl -s https://t.jaxiu.cn/api/recipe/detailed
```

## 2. 创建菜谱档案

```bash
curl -s -X POST https://t.jaxiu.cn/api/recipe \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的菜谱本",
    "password": "your_password"
  }'
# 返回: {"id":"xxx","creator_key":"xxx"}
```

## 3. 登录

```bash
curl -s -X POST https://t.jaxiu.cn/api/recipe/login \
  -H "Content-Type: application/json" \
  -d '{"id":"xxx","password":"your_password"}'
```

## 4. 获取/更新/删除档案

```bash
# 获取
curl -s "https://t.jaxiu.cn/api/recipe/{id}?creator_key=xxx"
# 或创建者
curl -s "https://t.jaxiu.cn/api/recipe/{id}/creator?creator_key=xxx"

# 更新
curl -s -X PUT https://t.jaxiu.cn/api/recipe/{id} \
  -H "Content-Type: application/json" \
  -d '{"creator_key":"xxx","title":"新名称","content":"更新内容"}'

# 删除
curl -s -X DELETE "https://t.jaxiu.cn/api/recipe/{id}?creator_key=xxx"
```

## 快速操作

当用户说"今天吃什么"时，调用获取今日菜谱 API。
当用户说"记录这个菜谱"时，使用更新档案 API。

Base URL: `https://t.jaxiu.cn`
