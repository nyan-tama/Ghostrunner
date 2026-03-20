# {{PROJECT_NAME}} - Development Commands
PROJECT_ROOT := $(shell pwd)

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "{{PROJECT_NAME}} - 利用可能なコマンド:"
	@echo ""
	@echo "起動:"
	@echo "  make dev              - サーバーを起動（自動で再起動）"
	@echo "  make backend          - バックエンドのみ起動"
	@echo "  make frontend         - フロントエンドのみ起動"
	@echo ""
	@echo "停止:"
	@echo "  make stop             - サーバーを停止"
	@echo ""
	@echo "ビルド・ヘルスチェック:"
	@echo "  make build            - 両方をビルド"
	@echo "  make health           - ヘルスチェック"
	@echo ""

# 起動（自プロジェクトのみ停止してから起動）
.PHONY: backend frontend dev

backend:
	cd $(PROJECT_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server

frontend:
	cd $(PROJECT_ROOT)/frontend && npm run dev

dev: stop
	@echo "{{PROJECT_NAME}} を起動中..."
	@make -j2 backend frontend

# 停止（自プロジェクトのポートのみ）
.PHONY: stop

stop:
	-lsof -ti:{{PORT_BACKEND}} | xargs kill -9 2>/dev/null || true
	-lsof -ti:{{PORT_FRONTEND}} | xargs kill -9 2>/dev/null || true

# ビルド
.PHONY: build

build:
	cd $(PROJECT_ROOT)/backend && go build -o server ./cmd/server
	cd $(PROJECT_ROOT)/frontend && npm run build

# ヘルスチェック
.PHONY: health

health:
	@curl -s http://localhost:{{PORT_BACKEND}}/api/health || echo "Backend: NG"
	@curl -s http://localhost:{{PORT_FRONTEND}} > /dev/null && echo "Frontend: OK" || echo "Frontend: NG"
