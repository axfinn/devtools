---
name: planner
description: 事项管理 — 创建/查看/更新/删除工作与生活任务，支持 AI 解析、会议纪要、日历下载、时间线回顾。触发：记录事情、添加任务、事项管理、会议记录、今日计划
triggers:
  - "事项管理"
  - "添加任务"
  - "今日计划"
  - "会议记录"
  - "待办事项"
  - "记录事情"
  - "planner"
---

# 事项管理 (Planner)

通过 DevTools 后端 API 管理任务、会议纪要和日程。后端默认运行在 `https://t.jaxiu.cn`。

## 认证

- 创建 profile 后会获得 `creator_key`，后续操作需要传递
- 或者使用 `password` 创建，之后通过 `password` 登录获取 `creator_key`

## 1. 档案管理

### 1.1 创建档案

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的事项管理",
    "password": "your_password"
  }'
# 返回: {"id":"xxx","creator_key":"xxx","password":"..."}
```

### 1.2 登录获取档案

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/login \
  -H "Content-Type: application/json" \
  -d '{
    "id": "profile_id",
    "password": "your_password"
  }'
# 返回: {"id":"xxx","creator_key":"xxx",...}
```

### 1.3 获取档案信息

```bash
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}?creator_key=xxx"
```

### 1.4 更新档案（重命名/延长过期）

```bash
curl -s -X PUT https://t.jaxiu.cn/api/planner/profile/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "action": "rename",
    "title": "新名称"
  }'
```

action 可选: `rename`、`extend`（延长过期）

## 2. 任务管理

### 2.1 创建任务

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "title": "完成项目方案",
    "description": "编写Q3项目方案文档",
    "priority": "high",
    "category": "work",
    "due_date": "2026-05-03T18:00:00+08:00",
    "tags": ["项目", "文档"]
  }'
```

priority 可选: `low`、`medium`、`high`、`urgent`
category 可选: `work`、`life`、`study`、`other`

### 2.2 批量创建任务

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/tasks/batch \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "tasks": [
      {"title": "任务1", "priority": "high", "category": "work"},
      {"title": "任务2", "priority": "medium", "category": "life"}
    ]
  }'
```

### 2.3 更新任务状态

```bash
curl -s -X PUT https://t.jaxiu.cn/api/planner/profile/{id}/tasks/{taskId} \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "status": "done",
    "title": "更新后的标题",
    "description": "更新后的描述",
    "priority": "medium"
  }'
```

status 可选: `todo`、`in_progress`、`done`、`cancelled`

### 2.4 删除任务

```bash
curl -s -X DELETE "https://t.jaxiu.cn/api/planner/profile/{id}/tasks/{taskId}?creator_key=xxx"
```

### 2.5 任务评论

```bash
# 添加评论
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/tasks/{taskId}/comments \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "content": "这个任务需要注意时间节点"
  }'

# 查看评论
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}/tasks/{taskId}/comments?creator_key=xxx"
```

### 2.6 任务活动日志

```bash
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}/tasks/{taskId}/activities?creator_key=xxx"
```

### 2.7 下载日历文件 (ICS)

```bash
curl -o task.ics "https://t.jaxiu.cn/api/planner/profile/{id}/tasks/{taskId}/calendar?creator_key=xxx"
```

## 3. 时间线 & 回顾

### 3.1 时间线视图

```bash
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}/timeline?creator_key=xxx&year=2026&month=5"
```

### 3.2 回顾总结

```bash
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}/review?creator_key=xxx&period=week"
```

period 可选: `day`、`week`、`month`

## 4. AI 功能

### 4.1 AI 解析任务（自然语言→结构化任务）

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/ai/parse \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "text": "明天下午3点前完成项目方案，高优先级；周末买牛奶和鸡蛋"
  }'
```

### 4.2 AI 建议

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/ai/advise \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "prompt": "帮我安排下周的工作计划"
  }'
```

## 5. 会议纪要

### 5.1 创建会议纪要

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/meetings \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "title": "Q3 项目启动会",
    "date": "2026-05-01T10:00:00+08:00",
    "attendees": ["张三", "李四", "王五"],
    "content": "1. 讨论项目目标\n2. 分配任务...",
    "action_items": ["张三负责后端开发", "李四负责前端"],
    "duration_minutes": 60
  }'
```

### 5.2 查看会议纪要列表

```bash
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}/meetings?creator_key=xxx"
```

### 5.3 查看单个会议

```bash
curl -s "https://t.jaxiu.cn/api/planner/profile/{id}/meetings/{meetingId}?creator_key=xxx"
```

### 5.4 AI 总结会议

```bash
curl -s -X POST https://t.jaxiu.cn/api/planner/profile/{id}/meetings/{meetingId}/summarize \
  -H "Content-Type: application/json" \
  -d '{"creator_key": "xxx"}'
```

### 5.5 上传会议录音

```bash
curl -X POST https://t.jaxiu.cn/api/planner/profile/{id}/recordings \
  -H "creator_key: xxx" \
  -F "file=@meeting_audio.mp3"
```

## 快速操作

当用户说"记录一个任务"或"帮我添加待办"时，使用创建任务 API；说"查看计划"时，使用时间线 API；说"AI分析我的任务"时，使用 AI parse API。

Base URL: `https://t.jaxiu.cn`
