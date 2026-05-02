---
name: paste
description: 粘贴板 — 创建/查看代码片段分享，支持文件上传、分片上传、代码分析、安全扫描。触发：分享代码、粘贴代码、上传文件、分享文本、pastebin
triggers:
  - "粘贴板"
  - "分享代码"
  - "粘贴代码"
  - "上传文件"
  - "pastebin"
  - "paste"
---

# 粘贴板 (PasteBin)

通过 DevTools 后端 API 创建和分享代码片段、文件。后端默认运行在 `https://t.jaxiu.cn`。

## 1. 创建粘贴

```bash
curl -s -X POST https://t.jaxiu.cn/api/paste \
  -H "Content-Type: application/json" \
  -d '{
    "content": "console.log(\"Hello World\")",
    "title": "测试代码",
    "language": "javascript",
    "password": "optional_password",
    "expires_in": 7,
    "max_views": 100
  }'
# 返回: {"id":"abc12345","url":"/paste/abc12345",...}
```

参数说明:
- `content`: 必填，文本内容，最大 100KB
- `title`: 可选，标题
- `language`: 可选，代码语言标记
- `password`: 可选，访问密码 (SHA256 hash)
- `expires_in`: 可选，过期天数 (默认7天，最大7天)
- `max_views`: 可选，最大查看次数 (默认1000)

## 2. 获取粘贴

```bash
# 无需密码
curl -s https://t.jaxiu.cn/api/paste/{id}

# 需要密码
curl -s "https://t.jaxiu.cn/api/paste/{id}?password=xxx"

# 查看信息（不消耗查看次数）
curl -s https://t.jaxiu.cn/api/paste/{id}/info
```

## 3. 文件上传

```bash
# 普通上传（最大55MB）
curl -X POST https://t.jaxiu.cn/api/paste/upload \
  -F "file=@document.pdf" \
  -F "title=项目文档"

# 返回: {"id":"xxx","url":"/api/paste/files/filename","filename":"..."}
```

### 3.1 分片上传（大文件）

```bash
# 初始化
curl -s -X POST https://t.jaxiu.cn/api/paste/chunk/init \
  -H "Content-Type: application/json" \
  -d '{"filename":"large_video.mp4","total_size":1073741824,"chunk_size":5242880}'
# 返回: {"file_id":"xxx"}

# 上传分片 (分片大小默认5MB)
curl -X POST "https://t.jaxiu.cn/api/paste/chunk/{file_id}" \
  -F "chunk=@chunk_data" \
  -F "index=0"

# 合并分片
curl -s -X POST "https://t.jaxiu.cn/api/paste/chunk/{file_id}/merge"

# 查看进度
curl -s "https://t.jaxiu.cn/api/paste/chunk/{file_id}/status"
```

## 4. 文件访问

```bash
# 访问上传的文件
curl -o downloaded.pdf https://t.jaxiu.cn/api/paste/files/{filename}
```

## 5. 代码分析

```bash
# 分析代码文本
curl -s -X POST https://t.jaxiu.cn/api/paste/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "content": "func main() {\n\tfmt.Println(\"hello\")\n}",
    "language": "go"
  }'

# 分析上传的文件
curl -s "https://t.jaxiu.cn/api/paste/analyze/{file_id}"
```

## 6. 安全扫描

```bash
# 扫描内容
curl -s -X POST https://t.jaxiu.cn/api/paste/scan \
  -H "Content-Type: application/json" \
  -d '{"content":"..."}'

# 验证文件
curl -s "https://t.jaxiu.cn/api/paste/validate/{file_id}"
```

## 7. 搜索 & 统计

```bash
# 搜索粘贴
curl -s "https://t.jaxiu.cn/api/paste/search?q=关键词&language=go"

# 统计信息
curl -s "https://t.jaxiu.cn/api/paste/stats"
```

## 快速操作

当用户说"把这个分享到粘贴板"或"分享这段代码"时，使用创建粘贴 API。
当用户说"上传这个文件分享"时，使用文件上传 API。

Base URL: `https://t.jaxiu.cn`
