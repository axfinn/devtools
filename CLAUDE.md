# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevTools is a web-based developer tools suite providing JSON formatting, text diff, Markdown preview, shared pastebin, Base64 encoding/decoding, URL encoding/decoding, timestamp conversion, and regex testing.

**Tech Stack:**
- Frontend: Vue 3 + Vite + Element Plus + TailwindCSS
- Backend: Go (Gin framework) + SQLite
- Deployment: Docker multi-stage build with Nginx proxy

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
# Development (docker-compose.yml)
docker-compose up -d                    # Start services
docker-compose down                     # Stop services
docker-compose logs -f devtools         # View logs

# Production (docker-compose.prod.yml)
docker-compose -f docker-compose.prod.yml up -d
```

The production setup uses Nginx reverse proxy on port 80. Development mode exposes the backend directly on port 8082.

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
  - Starts background goroutine for hourly cleanup of expired pastes
  - Serves static frontend files from `./dist` directory
- **handlers/paste.go**: PasteBin API handlers
  - Rate limiting: 5 creates per IP per minute, 60 requests total per minute
  - Content limits: 100KB max per paste, 1000 max views, 7 days max expiration
  - Password protection with SHA256 hashing
  - Auto-delete on expiration or max views reached
- **models/paste.go**: SQLite operations
  - Schema includes indexes on `expires_at` and `creator_ip`
  - Generates 8-character random hex IDs
  - Tracks views, expiration, password, and creator IP
- **middleware/ratelimit.go**: IP-based rate limiting middleware

### API Endpoints
- `POST /api/paste`: Create shared paste (requires `content`, optional `title`, `language`, `password`, `expires_in`, `max_views`)
- `GET /api/paste/:id?password=xxx`: Get paste content (increments view count)
- `GET /api/paste/:id/info`: Get paste metadata without content
- `GET /api/health`: Health check endpoint

### Docker Build Process
The Dockerfile uses multi-stage builds:
1. **frontend-builder**: Node 20 Alpine - builds Vue app
2. **backend-builder**: Go 1.21 Alpine - compiles static binary with CGO for SQLite
3. **Production**: Alpine with ca-certificates and tzdata, copies both artifacts

### Environment Variables
- `PORT`: Backend server port (default: 8080 in dev, 8082 in Docker)
- `DB_PATH`: SQLite database file path (default: ./data/paste.db)
- `TZ`: Timezone (default: Asia/Shanghai)
- `GIN_MODE`: Gin mode, set to "release" in production
- `HOST_PORT`: Host port for Docker exposure (default: 8082)

## Key Design Patterns

### Frontend
- **Keep-alive**: Router views are cached for better performance
- **Responsive Design**: Mobile-first approach with breakpoints at 768px and 480px
- **Dynamic Icons**: Element Plus icons loaded via component :is directive
- **Code Splitting**: Lazy-loaded routes reduce initial bundle size

### Backend
- **Security**: Rate limiting per IP, content size limits, password hashing, input validation
- **Auto-cleanup**: Background goroutine removes expired pastes every hour
- **Graceful Degradation**: Attempts cleanup when storage limit reached before rejecting
- **Static-first**: SPA routing handled by serving index.html for unmatched routes

## Database Schema

```sql
CREATE TABLE pastes (
  id TEXT PRIMARY KEY,           -- 8-char hex ID
  content TEXT NOT NULL,         -- Paste content
  title TEXT DEFAULT '',
  language TEXT DEFAULT 'text',  -- Syntax highlighting hint
  password TEXT DEFAULT '',      -- SHA256 hash
  expires_at DATETIME,
  max_views INTEGER DEFAULT 100,
  views INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  creator_ip TEXT
);
CREATE INDEX idx_expires_at ON pastes(expires_at);
CREATE INDEX idx_creator_ip ON pastes(creator_ip);
```

## Common Tasks

### Adding a New Tool
1. Create new Vue component in `frontend/src/views/YourTool.vue`
2. Add route to `frontend/src/router/index.js` with `meta.title` and `meta.icon`
3. Icon automatically appears in navigation menu

### Modifying Rate Limits
- Global rate limit: `middleware.NewRateLimiter()` call in main.go
- Paste creation limit: `maxPerIP` and `ipWindow` in handlers/paste.go
- Content size limit: `middleware.ContentSizeLimiter()` in main.go

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
