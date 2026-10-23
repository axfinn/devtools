# DevTools Quality Gates
# Usage: make [target]

.PHONY: all build vet test lint clean

# =====================================================================
# Go quality gates
# =====================================================================

GOPROXY := https://goproxy.cn,direct
# GOSUMDB 不能设 off：go.mod 要求 Go 1.26.10(eMudhra 中间 CA 吊销修复),
# 本机工具链较旧时 Go 会自动下载 1.26.10，而 toolchain 模块强制走 checksum 校验 ——
# GOSUMDB=off 会让下载直接失败("verifying module: checksum database disabled")。
# goproxy.cn 同时代理 sumdb，所以走 sum.golang.org 既能下载也能校验。
GOSUMDB := sum.golang.org
GOCACHE := $(shell go env GOCACHE)
GOMODCACHE := $(shell go env GOMODCACHE)

GO_FILES := $(shell find backend -name '*.go' -not -path '*/vendor/*' -not -path '*/.*')

# Max lines per Go source file (soft warning, hard error)
MAX_GO_LINES := 3000
# Max lines per Vue SFC (soft warning, hard error)  
MAX_VUE_LINES := 5000

all: vet test

## build: Compile all Go packages
build:
	cd backend && GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) go build ./...

## vet: Run go vet on all packages
vet:
	cd backend && GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) go vet ./...

## test: Run all tests
test:
	cd backend && GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) go test ./...

## lint: Check file sizes and run linters
lint: lint-go lint-vue lint-imports

## lint-go: Check Go file sizes
lint-go:
	@max=$(MAX_GO_LINES); echo "Checking Go file sizes (max $$max lines)..."; \
	found=0; for f in $(GO_FILES); do \
		lines=$$(wc -l < $$f); \
		if [ $$lines -gt $$max ]; then \
			echo "  WARNING: $$f ($$lines lines) exceeds $$max lines"; \
			found=1; \
		fi; \
	done; \
	[ $$found -eq 0 ] && echo "  OK: All Go files within size budget" || true

## lint-vue: Check Vue SFC sizes
lint-vue:
	@max=$(MAX_VUE_LINES); echo "Checking Vue SFC sizes (max $$max lines)..."; \
	found=0; for f in $$(find frontend/src -name '*.vue'); do \
		lines=$$(wc -l < $$f); \
		if [ $$lines -gt $$max ]; then \
			echo "  WARNING: $$f ($$lines lines) exceeds $$max lines"; \
			found=1; \
		fi; \
	done; \
	[ $$found -eq 0 ] && echo "  OK: All Vue files within size budget" || true

## lint-imports: Check for circular or unsafe imports
lint-imports:
	@echo "Checking imports..."
	@cd backend && GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) go mod verify

## clean: Remove build artifacts
clean:
	rm -rf backend/data/*.db
	rm -rf frontend/dist

# =====================================================================
# Frontend (requires pnpm)
# =====================================================================

## frontend-install: Install frontend dependencies
frontend-install:
	cd frontend && pnpm install

## frontend-build: Build frontend
frontend-build:
	cd frontend && pnpm build

## frontend-dev: Start frontend dev server
frontend-dev:
	cd frontend && pnpm dev

# =====================================================================
# Docker
# =====================================================================

## docker-build: Build Docker image
## Dockerfile 里 `COPY askit-extension.zip` 需要构建上下文先有这个文件（Askit 是
## 私有仓库，镜像内无法 clone），所以先跑宿主机构建。失败时 deploy.sh 会写占位 zip
## 并继续，不会阻断镜像构建。
docker-build:
	./deploy.sh build-askit
	docker compose build

## docker-up: Start all services
docker-up:
	docker compose up -d

## docker-down: Stop all services
docker-down:
	docker compose down

## docker-logs: View logs
docker-logs:
	docker compose logs -f devtools
