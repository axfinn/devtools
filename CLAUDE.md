# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevTools is a web-based developer tools suite providing JSON formatting, text diff, Markdown preview, shared pastebin, Base64 encoding/decoding, URL encoding/decoding, timestamp conversion, regex testing, text escaping, Mermaid diagrams, IP/DNS lookup, chat rooms, and URL shortening.

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
- **models/paste.go**: SQLite operations for pastes
  - Generates 8-character random hex IDs
- **models/chat.go**: SQLite operations for chat rooms and messages
- **models/shorturl.go**: SQLite operations for short URLs
- **middleware/ratelimit.go**: IP-based rate limiting middleware

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
- `GET /api/shorturl/:id/stats`: Get click stats
- `GET /s/:id`: Redirect to original URL (non-API route)

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
```

**Short URL modes:**
- 无密码：随机ID，有限流（10/小时/IP）
- 有密码：可自定义ID如 `/s/1` `/s/abc`，无限流

## Key Design Patterns

### Frontend
- **Keep-alive**: Router views are cached for better performance
- **Responsive Design**: Mobile-first approach with breakpoints at 768px and 480px
- **Dynamic Icons**: Element Plus icons loaded via component :is directive
- **Code Splitting**: Lazy-loaded routes reduce initial bundle size

### Backend
- **Security**: Rate limiting per IP, content size limits, password hashing, input validation
- **Auto-cleanup**: Background goroutine removes expired data every hour (pastes, chat rooms inactive >7 days, messages >7 days old, expired short URLs)
- **Graceful Degradation**: Attempts cleanup when storage limit reached before rejecting
- **Static-first**: SPA routing handled by serving index.html for unmatched routes

## Database Schema

The SQLite database contains four main tables: `pastes`, `chat_rooms`, `chat_messages`, and `short_urls`. Each table uses 8-character hex IDs and includes cleanup-related indexes. Schema is auto-created on startup in `models/*.go` files.

## Common Tasks

### Adding a New Tool
1. Create new Vue component in `frontend/src/views/YourTool.vue`
2. Add route to `frontend/src/router/index.js` with `meta.title` and `meta.icon`
3. Icon automatically appears in navigation menu

### Modifying Rate Limits
- Global create rate limit: `middleware.NewRateLimiter()` in main.go (10 per IP per minute)
- Paste creation limit: `maxPerIP` and `ipWindow` in handlers/paste.go
- Short URL limit: `CountShortURLsByIP()` check in handlers/shorturl.go (10 per IP per hour)
- Content size limit: `middleware.ContentSizeLimiter()` in main.go (55MB for video support)

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

## Module Path

The Go backend uses module path `devtools`. Imports should use this prefix:
```go
import (
    "devtools/handlers"
    "devtools/middleware"
    "devtools/models"
)
```
