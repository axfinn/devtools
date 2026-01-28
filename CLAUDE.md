# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevTools is a web-based developer tools suite providing JSON formatting, text diff, Markdown preview/sharing, shared pastebin, Base64 encoding/decoding, URL encoding/decoding, timestamp conversion, regex testing, text escaping, Mermaid diagrams, IP/DNS lookup, chat rooms, URL shortening, Mock API testing, and Excalidraw drawing.

**Tech Stack:**
- Frontend: Vue 3 + Vite + Element Plus + TailwindCSS
- Backend: Go (Gin framework) + SQLite
- Deployment: Docker multi-stage build

## Development Commands

### Frontend Development
```bash
cd frontend
npm install              # Install dependencies
npm run dev              # Start dev server (default: http://localhost:5173)
npm run build            # Build for production (output: frontend/dist)
npm run preview          # Preview production build
```

### Backend Development
```bash
cd backend
go mod tidy              # Install/update dependencies
go run main.go           # Start backend server (default: :8080 or :8082)
```

### Docker Deployment
```bash
docker-compose up -d                    # Start services
docker-compose down                     # Stop services
docker-compose logs -f devtools         # View logs
```

Docker exposes the backend on port 8082.

## Architecture

### Frontend Structure
- **Views**: Self-contained tool components in `frontend/src/views/`
  - Each tool (JsonTool, DiffTool, MarkdownTool, etc.) is a standalone Vue component
  - PasteBin.vue: Create shared pastes
  - PasteView.vue: View shared pastes by ID
- **Router**: Vue Router configuration in `frontend/src/router/index.js`
  - Routes are dynamically loaded for code splitting
  - Menu items are auto-generated from routes with `meta.title` and `meta.icon`
- **Layout**: App.vue provides responsive layout with:
  - Desktop: Collapsible sidebar navigation
  - Mobile: Drawer-based navigation with top bar

### Backend Structure
- **main.go**: Entry point
  - Initializes SQLite database with automatic schema creation
  - Configures Gin with CORS, rate limiting, and content size limits
  - Starts background goroutine for hourly cleanup (expired pastes, inactive chat rooms, old messages, expired short URLs)
  - Serves static frontend files from `./dist` directory
- **handlers/paste.go**: PasteBin API handlers
  - Rate limiting: 5 creates per IP per minute, 60 requests total per minute
  - Content limits: 100KB max per paste, 1000 max views, 7 days max expiration
  - Password protection with SHA256 hashing
  - Auto-delete on expiration or max views reached
- **handlers/chat.go**: WebSocket-based chat room handlers
  - Real-time messaging via gorilla/websocket
  - In-memory room/client management with sync.RWMutex
  - Image/video upload (5MB/50MB limits)
  - Password-protected rooms with SHA256 hashing
- **handlers/shorturl.go**: URL shortener handlers
  - Rate limiting: 10 per IP per hour
  - Default 30-day expiration, 1000 max clicks
  - Auto-delete on expiration or max clicks
- **handlers/dns.go**: IP/DNS lookup handlers
- **handlers/mockapi.go**: Mock API endpoint handlers
  - Rate limiting: 10 creates per IP per minute
  - Supports all HTTP methods (GET, POST, PUT, DELETE, etc.)
  - Custom response status codes and bodies
  - Request logging with optional expiration
- **handlers/mdshare.go**: Markdown share handlers
  - Shareable Markdown with 2-10 view limit, shows remaining views
  - Auto-generates short URL (`/s/xxx`) for easy sharing
  - Creator key for management (stored in localStorage)
  - Admin password for global management (list, view, delete all)
  - Reshare generates new access key and short URL
  - Supports image paste/drag (Base64 embedded, max 1MB per image)
- **handlers/excalidraw.go**: Excalidraw drawing handlers
  - Cloud save with password protection (required)
  - Default 30-day expiration, configurable 1-365 days
  - Admin password can set permanent save (no expiration)
  - Creator key for management (stored in localStorage)
  - Content compressed with gzip (max 10MB)
  - Auto-generates short URL for sharing
- **config/config.go**: Configuration file loader for YAML config
- **models/paste.go**: SQLite operations for pastes
  - Generates 8-character random hex IDs
- **models/chat.go**: SQLite operations for chat rooms and messages
- **models/shorturl.go**: SQLite operations for short URLs
- **models/mockapi.go**: SQLite operations for Mock APIs and request logs
- **models/mdshare.go**: SQLite operations for Markdown shares
- **models/excalidraw.go**: SQLite operations for Excalidraw drawings (with gzip compression)
- **middleware/ratelimit.go**: IP-based rate limiting middleware
- **utils/crypto.go**: Password hashing utilities (SHA256)
- **utils/cleanup.go**: File cleanup utilities for expired uploads

### API Endpoints

**PasteBin:**
- `POST /api/paste`: Create shared paste (requires `content`, optional `title`, `language`, `password`, `expires_in`, `max_views`)
- `GET /api/paste/:id?password=xxx`: Get paste content (increments view count)
- `GET /api/paste/:id/info`: Get paste metadata without content

**IP/DNS:**
- `GET /api/ip`: Get client IP address
- `GET /api/dns?domain=xxx`: DNS lookup for a domain

**Chat Rooms:**
- `POST /api/chat/room`: Create a chat room (requires `name`, optional `password`)
- `GET /api/chat/rooms`: List recent rooms
- `GET /api/chat/room/:id`: Get room info
- `POST /api/chat/room/:id/join`: Join room (requires `nickname`, optional `password`)
- `GET /api/chat/room/:id/ws?nickname=xxx`: WebSocket connection for real-time chat
- `POST /api/chat/upload`: Upload image/video (max 5MB image, 50MB video)

**Short URL:**
- `POST /api/shorturl`: Create short URL (requires `original_url`, optional `expires_in`, `max_clicks`, `custom_id`, `password`)
- `GET /api/shorturl/list`: List short URLs
- `GET /api/shorturl/:id/stats`: Get click stats
- `GET /s/:id`: Redirect to original URL (non-API route)

**Mock API:**
- `POST /api/mockapi`: Create Mock API endpoint (requires `method`, `response_status`, `response_body`, optional `name`, `description`, `expires_in`)
- `GET /api/mockapi/:id`: Get Mock API details
- `GET /api/mockapi/:id/logs`: Get request logs for Mock API
- `PUT /api/mockapi/:id`: Update Mock API endpoint
- `DELETE /api/mockapi/:id`: Delete Mock API endpoint
- `ANY /mock/:id`: Execute Mock API endpoint (logs request and returns configured response)

**Markdown Share:**
- `POST /api/mdshare`: Create Markdown share (requires `content`, optional `title`, `max_views` 2-10, `expires_in` days)
  - Returns `id`, `creator_key`, `access_key`, `short_code`, `share_url`
- `GET /api/mdshare/:id?key=xxx`: Get content (increments view count, returns `remaining_views`)
- `GET /api/mdshare/:id/creator?creator_key=xxx`: Get content for creator (no view increment)
- `PUT /api/mdshare/:id`: Manage share (actions: `extend`, `reshare`, `edit`)
- `DELETE /api/mdshare/:id?creator_key=xxx`: Delete share
- `GET /api/mdshare/admin/list?admin_password=xxx`: Admin list all shares
- `GET /api/mdshare/admin/:id?admin_password=xxx`: Admin view any share
- `DELETE /api/mdshare/admin/:id?admin_password=xxx`: Admin delete any share
- `GET /md/:id?key=xxx`: Frontend view route (auto-generates short URL `/s/xxx`)

**Excalidraw:**
- `POST /api/excalidraw`: Create drawing (requires `content`, `password`, optional `title`, `expires_in` days, `admin_password` for permanent)
  - Returns `id`, `creator_key`, `short_code`, `share_url`, `expires_at`, `is_permanent`
- `GET /api/excalidraw/:id?password=xxx`: Get drawing content (requires password)
- `GET /api/excalidraw/:id/creator?creator_key=xxx`: Get drawing for creator
- `PUT /api/excalidraw/:id`: Manage drawing (actions: `extend`, `edit`, `set_permanent`)
- `DELETE /api/excalidraw/:id?creator_key=xxx`: Delete drawing
- `GET /api/excalidraw/admin/list?admin_password=xxx`: Admin list all drawings
- `GET /api/excalidraw/admin/:id?admin_password=xxx`: Admin view any drawing
- `DELETE /api/excalidraw/admin/:id?admin_password=xxx`: Admin delete any drawing
- `GET /draw/:id`: Frontend view route

**Health:**
- `GET /api/health`: Health check endpoint

### Docker Build Process
The Dockerfile uses multi-stage builds:
1. **frontend-builder**: Node 20 Alpine - builds Vue app
2. **backend-builder**: Go 1.21 Alpine - compiles static binary with CGO for SQLite
3. **Production**: Alpine with ca-certificates and tzdata, copies both artifacts

### Environment Variables
- `PORT`: Backend server port (default: 8080 in dev, 8082 in Docker)
- `DB_PATH`: SQLite database file path (default: ./data/paste.db)
- `CONFIG_PATH`: Config file path (default: ./config.yaml)
- `TZ`: Timezone (default: Asia/Shanghai)
- `GIN_MODE`: Gin mode, set to "release" in production
- `HOST_PORT`: Host port for Docker exposure (default: 8082)

### Configuration File
Copy `backend/config.example.yaml` to `backend/config.yaml`:
```yaml
shorturl:
  password: "your_password"  # 设置后可使用自定义短链ID

mdshare:
  admin_password: ""         # 管理员密码
  default_max_views: 5       # 默认最大查看次数 (2-10)
  default_expires_days: 30   # 默认过期天数

excalidraw:
  admin_password: ""         # 管理员密码（可永久保存）
  default_expires_days: 30   # 默认过期天数
  max_content_size: 10485760 # 最大内容大小（10MB）
```

**Short URL modes:**
- 无密码：随机ID，有限流（10/小时/IP）
- 有密码：可自定义ID如 `/s/1` `/s/abc`，无限流

**Markdown Share:**
- 创建者密钥自动存储在浏览器 localStorage
- 访问密钥用于分享给他人查看
- 管理员密码可管理所有分享（查看、删除）
- 管理员面板入口：Markdown 编辑器页面右上角"管理"按钮
- 管理员密码存储在 sessionStorage（关闭浏览器失效）

**Excalidraw:**
- 云端保存需要设置访问密码（必填）
- 默认30天过期，可设置1-365天
- 管理员密码可设置永久保存（无过期）
- 创建者密钥自动存储在浏览器 localStorage
- 支持本地保存到浏览器 localStorage
- 支持导出 PNG、SVG、JSON 格式
- 内容使用 gzip 压缩存储（最大10MB）

## Key Design Patterns

### Frontend
- **Keep-alive**: Router views are cached for better performance
- **Responsive Design**: Mobile-first approach with breakpoints at 768px and 480px
- **Dynamic Icons**: Element Plus icons loaded via component :is directive
- **Code Splitting**: Lazy-loaded routes reduce initial bundle size

### Backend
- **Security**: Rate limiting per IP, content size limits, password hashing (SHA256), input validation
- **Auto-cleanup**: Background goroutine removes expired data every hour (pastes, chat rooms inactive >7 days, messages >7 days old, expired short URLs, expired Mock APIs, expired Markdown shares, expired Excalidraw drawings, uploaded files >7 days old)
- **Graceful Degradation**: Attempts cleanup when storage limit reached before rejecting
- **Static-first**: SPA routing handled by serving index.html for unmatched routes
- **Config System**: YAML-based configuration loaded from `./config.yaml` or `CONFIG_PATH` env var

## Database Schema

The SQLite database contains seven main tables: `pastes`, `chat_rooms`, `chat_messages`, `short_urls`, `mock_apis` (with `mock_api_logs`), `markdown_shares`, and `excalidraw_shares`. Most tables use 8-character hex IDs and include cleanup-related indexes. Schema is auto-created on startup in `models/*.go` files.

## Common Tasks

### Adding a New Tool
1. Create new Vue component in `frontend/src/views/YourTool.vue`
2. Add route to `frontend/src/router/index.js` with `meta.title` and `meta.icon`
3. Icon automatically appears in navigation menu

### Modifying Rate Limits
- Global create rate limit: `middleware.NewRateLimiter()` in main.go (10 per IP per minute)
- Paste creation limit: `maxPerIP` and `ipWindow` in handlers/paste.go (5 per minute)
- Short URL limit: `CountShortURLsByIP()` check in handlers/shorturl.go (10 per IP per hour)
- Mock API limit: Same as global create rate limit (10 per IP per minute)
- Content size limit: `middleware.ContentSizeLimiter()` in main.go (55MB for video upload support)

### Testing PasteBin
Use curl or API clients:
```bash
# Create paste
curl -X POST http://localhost:8082/api/paste \
  -H "Content-Type: application/json" \
  -d '{"content":"test","title":"Test","expires_in":1}'

# Get paste (returns id in response)
curl http://localhost:8082/api/paste/{id}
```

### Testing Markdown Share
```bash
# Create share
curl -X POST http://localhost:8082/api/mdshare \
  -H "Content-Type: application/json" \
  -d '{"content":"# Hello\nThis is **Markdown**","title":"Test","max_views":5,"expires_in":30}'

# Response: {"id":"abc12345","creator_key":"xxx","access_key":"yyy","short_code":"zzz","share_url":"/s/zzz"}

# View share (consumes 1 view)
curl "http://localhost:8082/api/mdshare/{id}?key={access_key}"

# View as creator (no view consumption)
curl "http://localhost:8082/api/mdshare/{id}/creator?creator_key={creator_key}"

# Reshare (generate new access key)
curl -X PUT http://localhost:8082/api/mdshare/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"reshare","max_views":5,"creator_key":"xxx"}'

# Delete share
curl -X DELETE "http://localhost:8082/api/mdshare/{id}?creator_key={creator_key}"

# Admin: List all shares (requires admin_password in config.yaml)
curl "http://localhost:8082/api/mdshare/admin/list?admin_password=your_password"

# Admin: View any share
curl "http://localhost:8082/api/mdshare/admin/{id}?admin_password=your_password"

# Admin: Delete any share
curl -X DELETE "http://localhost:8082/api/mdshare/admin/{id}?admin_password=your_password"
```

### Testing Excalidraw
```bash
# Create cloud save (password required)
curl -X POST http://localhost:8082/api/excalidraw \
  -H "Content-Type: application/json" \
  -d '{"content":"{\"type\":\"excalidraw\",\"elements\":[]}","title":"Test","password":"123456","expires_in":30}'

# Response: {"id":"abc12345","creator_key":"xxx","short_code":"zzz","share_url":"/s/zzz","expires_at":"...","is_permanent":false}

# Get drawing (requires password)
curl "http://localhost:8082/api/excalidraw/{id}?password=123456"

# Get as creator (no password needed)
curl "http://localhost:8082/api/excalidraw/{id}/creator?creator_key={creator_key}"

# Extend expiration
curl -X PUT http://localhost:8082/api/excalidraw/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"extend","expires_in":30,"creator_key":"xxx"}'

# Set permanent (admin only)
curl -X PUT http://localhost:8082/api/excalidraw/{id} \
  -H "Content-Type: application/json" \
  -d '{"action":"set_permanent","admin_password":"your_password"}'

# Delete drawing
curl -X DELETE "http://localhost:8082/api/excalidraw/{id}?creator_key={creator_key}"

# Admin: List all drawings
curl "http://localhost:8082/api/excalidraw/admin/list?admin_password=your_password"
```

### WebSocket Chat Architecture
The chat system uses in-memory state management with gorilla/websocket:
- **Room/Client Management**: `sync.RWMutex` protects shared state for concurrent access
- **Message Broadcasting**: Messages are broadcast to all clients in a room via goroutines
- **File Uploads**: Images/videos are stored in `./data/uploads` with 7-day auto-cleanup
- **Connection Handling**: Each WebSocket connection runs in its own goroutine with read/write loops
- **Cleanup**: Rooms inactive for 7+ days and messages older than 7 days are auto-deleted

## Module Path

The Go backend uses module path `devtools`. Imports should use this prefix:
```go
import (
    "devtools/handlers"
    "devtools/middleware"
    "devtools/models"
)
```
