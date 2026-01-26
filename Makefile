# プロジェクトルート
PROJECT_ROOT := $(shell pwd)

# 起動
.PHONY: backend frontend dev

backend:
	cd $(PROJECT_ROOT)/backend && go run ./cmd/server

frontend:
	cd $(PROJECT_ROOT)/frontend && npm run dev

dev:
	@echo "両サーバーを起動..."
	@make -j2 backend frontend

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

# 再起動（kill + start）
.PHONY: restart-backend restart-frontend restart

restart-backend: stop-backend
	@sleep 1
	nohup sh -c 'cd $(PROJECT_ROOT)/backend && go run ./cmd/server' > /tmp/backend.log 2>&1 &

restart-frontend: stop-frontend
	@sleep 1
	nohup sh -c 'cd $(PROJECT_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &

restart:
	@make -j2 restart-backend restart-frontend

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
