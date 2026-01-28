# 聊天室媒体发送功能开发计划

## 功能需求

### 图片功能 (已完成)
- 支持发送图片消息
- 支持从粘贴板直接粘贴图片发送
- 支持点击选择图片上传
- 支持图片预览

### 视频功能
- 支持发送视频消息
- 支持点击选择视频上传
- 支持从粘贴板粘贴视频文件
- 视频消息内嵌播放器

## 技术方案

### 后端修改

1. **图片上传API** - `POST /api/chat/upload`
   - 接收 multipart/form-data 格式的图片文件
   - 限制大小: 5MB
   - 支持格式: jpg, png, gif, webp
   - 保存到 `./data/uploads/` 目录
   - 返回图片URL

2. **图片静态服务** - `GET /api/chat/uploads/:filename`
   - 使用 Gin 的 Static 中间件提供图片访问

3. **消息类型扩展**
   - `msg_type: "image"` - 图片消息
   - `content` 字段存储图片URL

### 前端修改

1. **输入区域增强**
   - 添加图片上传按钮
   - 隐藏的 file input 用于选择图片

2. **粘贴板监听**
   - 监听 input 的 paste 事件
   - 检测粘贴内容是否包含图片
   - 自动上传并发送

3. **图片上传处理**
   - 显示上传中状态
   - 上传成功后发送 image 类型消息

4. **图片消息渲染**
   - 根据 msg_type 判断渲染方式
   - 图片显示为缩略图
   - 点击可预览大图

5. **图片预览**
   - 使用 Element Plus 的 el-image-viewer
   - 支持缩放、关闭

## 文件修改清单

| 文件 | 修改内容 |
|------|----------|
| `backend/handlers/chat.go` | 添加 UploadImage 处理函数 |
| `backend/main.go` | 添加上传路由和静态文件服务 |
| `frontend/src/views/ChatRoom.vue` | 添加图片上传、粘贴、渲染、预览功能 |

## 实现步骤

- [x] 1. 后端：添加图片上传API端点
- [x] 2. 后端：扩展消息类型支持image（已有 msg_type 字段）
- [x] 3. 前端：添加粘贴板图片监听
- [x] 4. 前端：实现图片上传和发送
- [x] 5. 前端：添加图片消息渲染
- [x] 6. 前端：添加图片预览功能
- [x] 7. 测试功能完整性（前后端编译通过）

## API 规范

### 上传图片
```
POST /api/chat/upload
Content-Type: multipart/form-data

Request:
  - image: File (required)

Response:
{
  "url": "/api/chat/uploads/xxx.jpg",
  "filename": "xxx.jpg"
}
```

### WebSocket 图片消息
```json
{
  "type": "message",
  "content": "/api/chat/uploads/xxx.jpg",
  "msg_type": "image"
}
```

---

## 视频功能扩展

### 后端修改

1. **视频上传API** - 复用 `POST /api/chat/upload`
   - 扩展支持视频文件类型
   - 视频限制大小: 50MB
   - 支持格式: mp4, webm, mov, avi
   - 保存到同一 `./data/uploads/` 目录

2. **消息类型扩展**
   - `msg_type: "video"` - 视频消息
   - `content` 字段存储视频URL

### 前端修改

1. **输入区域增强**
   - 添加视频上传按钮
   - 支持视频文件选择

2. **粘贴板监听增强**
   - 检测粘贴内容是否包含视频
   - 自动上传并发送

3. **视频消息渲染**
   - 使用 HTML5 video 标签
   - 显示视频控制条
   - 支持全屏播放

### 实现步骤

- [x] 1. 后端：扩展上传API支持视频文件
- [x] 2. 前端：添加视频上传按钮
- [x] 3. 前端：扩展粘贴板监听支持视频
- [x] 4. 前端：添加视频消息渲染和播放
- [x] 5. 测试功能完整性（前后端编译通过）

### API 规范

#### 上传视频
```
POST /api/chat/upload
Content-Type: multipart/form-data

Request:
  - image: File (required) // 字段名保持为 image，支持图片和视频

Response:
{
  "url": "/api/chat/uploads/xxx.mp4",
  "filename": "xxx.mp4",
  "type": "video"  // 新增：返回文件类型
}
```

#### WebSocket 视频消息
```json
{
  "type": "message",
  "content": "/api/chat/uploads/xxx.mp4",
  "msg_type": "video"
}
```
