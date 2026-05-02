---
name: photowall
description: 档案照片墙 — 分类照片管理、时间线聚合、分享链接、批量下载ZIP。触发：照片墙、上传照片、相册、家庭相册
triggers:
  - "照片墙"
  - "相册"
  - "上传照片"
  - "家庭相册"
  - "photowall"
---

# 档案照片墙 (PhotoWall)

通过 DevTools 后端 API 管理分类照片档案，支持时间线、分享和打包下载。后端默认运行在 `https://t.jaxiu.cn`。

## 认证

- 创建时获得 `creator_key` 和 `access_key`
- `creator_key` 用于管理（创建者）
- `access_key` 用于分享给他人查看
- 也可以 `password` 创建，通过密码登录

## 1. 档案管理

### 1.1 创建档案

```bash
curl -s -X POST https://t.jaxiu.cn/api/photowall/profile \
  -H "Content-Type: application/json" \
  -d '{
    "title": "宝宝成长记录",
    "password": "your_password",
    "expires_in": 180
  }'
# 返回: {"id":"xxx","creator_key":"xxx","access_key":"xxx","short_code":"xxx","share_url":"/wall/xxx?key=xxx"}
```

expires_in 可选，默认90天，最大180天（普通用户），admin 可设永久。

### 1.2 登录

```bash
curl -s -X POST https://t.jaxiu.cn/api/photowall/profile/login \
  -H "Content-Type: application/json" \
  -d '{"id":"xxx","password":"your_password"}'
```

### 1.3 获取档案

```bash
curl -s "https://t.jaxiu.cn/api/photowall/profile/{id}?creator_key=xxx"
```

### 1.4 更新档案

```bash
curl -s -X PUT https://t.jaxiu.cn/api/photowall/profile/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "xxx",
    "action": "rename",
    "title": "新标题"
  }'
```

action 可选: `rename`、`extend`（延长过期）、`reshare`（生成新 access_key）

## 2. 照片管理

### 2.1 上传照片

```bash
curl -X POST https://t.jaxiu.cn/api/photowall/profile/{id}/items \
  -F "file=@baby_photo.jpg" \
  -F "creator_key=xxx" \
  -F "category=成长记录" \
  -F "description=宝宝第一次走路" \
  -F "taken_at=2026-04-30T10:00:00+08:00"
```

### 2.2 更新照片信息

```bash
curl -s -X PUT https://t.jaxiu.cn/api/photowall/profile/{id}/items/{itemId} \
  -H "Content-Type: application/json" \
  -d '{"creator_key":"xxx","category":"新分类","description":"新描述"}'
```

### 2.3 删除照片

```bash
curl -s -X DELETE "https://t.jaxiu.cn/api/photowall/profile/{id}/items/{itemId}?creator_key=xxx"
```

## 3. 分享访问

```bash
# 通过 access_key 查看（无需认证）
curl -s "https://t.jaxiu.cn/api/photowall/share/{id}?key={access_key}"
```

## 4. 下载

```bash
# 单张下载
curl -o photo.jpg "https://t.jaxiu.cn/api/photowall/profile/{id}/download?creator_key=xxx&item_ids=item1"

# 多选打包下载 (ZIP)
curl -o album.zip "https://t.jaxiu.cn/api/photowall/profile/{id}/download?creator_key=xxx&item_ids=item1,item2,item3"
```

## 5. 文件访问

```bash
curl -o photo.jpg https://t.jaxiu.cn/api/photowall/files/{filename}
```

## 快速操作

当用户说"上传这张照片到相册"或"分享照片"时，使用上传 API。
当用户说"下载相册"时，使用下载 API。
分享链接格式: `https://t.jaxiu.cn/wall/{id}?key={access_key}`

Base URL: `https://t.jaxiu.cn`
