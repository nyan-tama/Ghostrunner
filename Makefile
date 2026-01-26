# プロジェクトルート
PROJECT_ROOT := $(shell pwd)

# デフォルトターゲット（ヘルプ表示）
.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "利用可能なコマンド:"
	@echo ""
	@echo "起動（フォアグラウンド、ログ直接表示）:"
	@echo "  make backend          - バックエンドを起動"
	@echo "  make frontend         - フロントエンドを起動"
	@echo "  make dev              - 両方を並列起動"
	@echo ""
	@echo "起動（バックグラウンド + ログ自動表示）:"
	@echo "  make start-backend    - バックエンドを起動してログ表示"
	@echo "  make start-frontend   - フロントエンドを起動してログ表示"
	@echo ""
	@echo "停止:"
	@echo "  make stop-backend     - バックエンドを停止"
	@echo "  make stop-frontend    - フロントエンドを停止"
	@echo "  make stop             - 両方を停止"
	@echo ""
	@echo "再起動（バックグラウンド）:"
	@echo "  make restart-backend  - バックエンドを再起動"
	@echo "  make restart-frontend - フロントエンドを再起動"
	@echo "  make restart          - 両方を再起動"
	@echo ""
	@echo "再起動（ログ自動表示）:"
	@echo "  make restart-backend-logs  - バックエンドを再起動してログ表示"
	@echo "  make restart-frontend-logs - フロントエンドを再起動してログ表示"
	@echo ""
	@echo "ログ確認:"
	@echo "  make logs-backend     - バックエンドのログを表示"
	@echo "  make logs-frontend    - フロントエンドのログを表示"
	@echo ""
	@echo "ビルド・ヘルスチェック:"
	@echo "  make build            - 両方をビルド"
	@echo "  make health           - ヘルスチェック"
	@echo ""

# 起動（フォアグラウンド、ログ直接表示）
.PHONY: backend frontend dev

backend:
	cd $(PROJECT_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server

frontend:
	cd $(PROJECT_ROOT)/frontend && npm run dev

dev:
	@echo "両サーバーを起動..."
	@make -j2 backend frontend

# 起動（バックグラウンド + ログ自動表示）
.PHONY: start-backend start-frontend start

start-backend:
	@echo "Starting backend in background..."
	@nohup sh -c 'cd $(PROJECT_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server' > /tmp/backend.log 2>&1 &
	@sleep 2
	@echo "Backend started. Showing logs (Ctrl+C to exit):"
	@tail -f /tmp/backend.log | LC_ALL=C sed \
		-e 's/.*ERROR.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*WARN.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Listening.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Starting.*/\x1b[1;36m&\x1b[0m/' \
		-e 's/GET\|POST\|PUT\|DELETE\|PATCH/\x1b[1;34m&\x1b[0m/' \
		-e 's/200\|201\|204/\x1b[1;32m&\x1b[0m/' \
		-e 's/400\|401\|403\|404\|500\|502\|503/\x1b[1;31m&\x1b[0m/'

start-frontend:
	@echo "Starting frontend in background..."
	@nohup sh -c 'cd $(PROJECT_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &
	@sleep 2
	@echo "Frontend started. Showing logs (Ctrl+C to exit):"
	@tail -f /tmp/frontend.log | LC_ALL=C sed \
		-e 's/.*error.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Error.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*warn.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Warn.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Ready.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Compiled.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Starting.*/\x1b[1;36m&\x1b[0m/' \
		-e 's/GET\|POST\|PUT\|DELETE\|PATCH/\x1b[1;34m&\x1b[0m/' \
		-e 's/200\|201\|204/\x1b[1;32m&\x1b[0m/' \
		-e 's/404\|500\|502/\x1b[1;31m&\x1b[0m/'

start:
	@echo "両サーバーをバックグラウンドで起動します"
	@echo "ログは別々のターミナルで確認してください:"
	@echo "  make start-backend"
	@echo "  make start-frontend"

# 停止
.PHONY: stop-backend stop-frontend stop

stop-backend:
	-pkill -f "go run.*cmd/server" || true
	-pkill -f "backend/server" || true
	-lsof -ti:8080 | xargs kill -9 2>/dev/null || true

stop-frontend:
	-pkill -f "next dev" || true
	-pkill -f "npm.*dev" || true
	-pkill -f "npm.*start" || true
	-lsof -ti:3000 | xargs kill -9 2>/dev/null || true

stop: stop-backend stop-frontend

# 再起動（kill + start、バックグラウンド）
.PHONY: restart-backend restart-frontend restart

restart-backend: stop-backend
	@sleep 1
	@nohup sh -c 'cd $(PROJECT_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server' > /tmp/backend.log 2>&1 &

restart-frontend: stop-frontend
	@sleep 1
	@nohup sh -c 'cd $(PROJECT_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &

restart:
	@make -j2 restart-backend restart-frontend

# 再起動（kill + start + ログ表示）
.PHONY: restart-backend-logs restart-frontend-logs

restart-backend-logs: stop-backend
	@echo "Restarting backend..."
	@sleep 1
	@nohup sh -c 'cd $(PROJECT_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server' > /tmp/backend.log 2>&1 &
	@sleep 2
	@echo "Backend restarted. Showing logs (Ctrl+C to exit):"
	@tail -f /tmp/backend.log | LC_ALL=C sed \
		-e 's/.*ERROR.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*WARN.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Listening.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Starting.*/\x1b[1;36m&\x1b[0m/' \
		-e 's/GET\|POST\|PUT\|DELETE\|PATCH/\x1b[1;34m&\x1b[0m/' \
		-e 's/200\|201\|204/\x1b[1;32m&\x1b[0m/' \
		-e 's/400\|401\|403\|404\|500\|502\|503/\x1b[1;31m&\x1b[0m/'

restart-frontend-logs: stop-frontend
	@echo "Restarting frontend..."
	@sleep 1
	@nohup sh -c 'cd $(PROJECT_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &
	@sleep 2
	@echo "Frontend restarted. Showing logs (Ctrl+C to exit):"
	@tail -f /tmp/frontend.log | LC_ALL=C sed \
		-e 's/.*error.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Error.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*warn.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Warn.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Ready.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Compiled.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Starting.*/\x1b[1;36m&\x1b[0m/' \
		-e 's/GET\|POST\|PUT\|DELETE\|PATCH/\x1b[1;34m&\x1b[0m/' \
		-e 's/200\|201\|204/\x1b[1;32m&\x1b[0m/' \
		-e 's/404\|500\|502/\x1b[1;31m&\x1b[0m/'

# ビルド
.PHONY: build

build:
	cd $(PROJECT_ROOT)/backend && go build -o server ./cmd/server
	cd $(PROJECT_ROOT)/frontend && npm run build

# ヘルスチェック
.PHONY: health

health:
	@curl -s http://localhost:8080/api/health || echo "Backend: NG"
	@curl -s http://localhost:3000 > /dev/null && echo "Frontend: OK" || echo "Frontend: NG"

# ログ確認
.PHONY: logs-backend logs-frontend logs

logs-backend:
	@echo "=== Backend Logs (Ctrl+C to exit) ==="
	@tail -f /tmp/backend.log | LC_ALL=C sed \
		-e 's/.*ERROR.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*WARN.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Listening.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Starting.*/\x1b[1;36m&\x1b[0m/' \
		-e 's/GET\|POST\|PUT\|DELETE\|PATCH/\x1b[1;34m&\x1b[0m/' \
		-e 's/200\|201\|204/\x1b[1;32m&\x1b[0m/' \
		-e 's/400\|401\|403\|404\|500\|502\|503/\x1b[1;31m&\x1b[0m/'

logs-frontend:
	@echo "=== Frontend Logs (Ctrl+C to exit) ==="
	@tail -f /tmp/frontend.log | LC_ALL=C sed \
		-e 's/.*error.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Error.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*warn.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Warn.*/\x1b[1;33m&\x1b[0m/' \
		-e 's/.*Failed.*/\x1b[1;31m&\x1b[0m/' \
		-e 's/.*Ready.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Compiled.*/\x1b[1;32m&\x1b[0m/' \
		-e 's/.*Starting.*/\x1b[1;36m&\x1b[0m/' \
		-e 's/GET\|POST\|PUT\|DELETE\|PATCH/\x1b[1;34m&\x1b[0m/' \
		-e 's/200\|201\|204/\x1b[1;32m&\x1b[0m/' \
		-e 's/404\|500\|502/\x1b[1;31m&\x1b[0m/'

logs:
	@echo "両方のログを表示します（別々のターミナルで実行してください）"
	@echo "  make logs-backend"
	@echo "  make logs-frontend"
