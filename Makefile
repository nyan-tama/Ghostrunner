# プロジェクトルート
PROJECT_ROOT := $(shell pwd)
DEVTOOLS_ROOT := $(PROJECT_ROOT)/devtools

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
	@echo "  make start-backend       - バックエンドを起動してログ表示"
	@echo "  make start-frontend      - フロントエンドを起動してログ表示"
	@echo "  make start-even-terminal - even-terminal を起動(BRIDGE_TOKEN 固定)"
	@echo "  make g2                  - Even G2 用 even-terminal を起動(--name G2)"
	@echo "  make g2-all              - patrol 全プロジェクトを並列起動 + 全 QR 表示"
	@echo "  make g2-qr               - 起動中の全 QR を再表示(G2 への登録時)"
	@echo "  make stop-g2-all         - g2-all 全インスタンス停止"
	@echo "  make g2-status           - 起動中の even-terminal 一覧"
	@echo "  make start-voicevox      - VOICEVOX を起動(:50021 まで待機)"
	@echo ""
	@echo "停止:"
	@echo "  make stop-backend        - バックエンドを停止"
	@echo "  make stop-frontend       - フロントエンドを停止"
	@echo "  make stop-even-terminal  - even-terminal を停止"
	@echo "  make stop-voicevox       - VOICEVOX を停止(17 GB 解放)"
	@echo "  make stop                - 全て停止(寝る前用)"
	@echo ""
	@echo "メモリリーク保険(launchd auto-restart):"
	@echo "  make install-restart-cron   - 毎日 03:00 にガード付き auto-restart を登録"
	@echo "  make uninstall-restart-cron - launchd から削除"
	@echo "  make status-restart-cron    - 登録状態を確認"
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
	@echo "  make gr-run           - gr-run CLIをビルド"
	@echo "  make health           - ヘルスチェック"
	@echo ""

# 起動（フォアグラウンド、ログ直接表示）
.PHONY: backend frontend dev

backend:
	cd $(DEVTOOLS_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server

frontend:
	cd $(DEVTOOLS_ROOT)/frontend && npm run dev -- -H 0.0.0.0

dev: stop
	@echo "両サーバーを起動..."
	@make -j2 backend frontend

# 起動（バックグラウンド + ログ自動表示）
.PHONY: start-backend start-backend-debug start-frontend start

# ENABLE_PPROF=1 つきで起動。メモリリーク調査時に使う。
# 取得例: go tool pprof http://localhost:8888/debug/pprof/heap
start-backend-debug: stop-backend
	@echo "Starting backend with ENABLE_PPROF=1..."
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; ENABLE_PPROF=1 go run ./cmd/server' > /tmp/backend.log 2>&1 &
	@sleep 3
	@echo "pprof: http://localhost:8888/debug/pprof/"

start-backend:
	@echo "Starting backend in background..."
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server' > /tmp/backend.log 2>&1 &
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
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &
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

# even-terminal 起動(BRIDGE_TOKEN を ~/.zshrc から固定で渡す)
# 明示しないと even-terminal が起動時にランダム token を毎回生成し、
# frontend .env.local の NEXT_PUBLIC_EVEN_TERMINAL_TOKEN と不一致になり 401。
.PHONY: start-even-terminal stop-even-terminal restart-even-terminal
start-even-terminal: stop-even-terminal
	@echo "Starting even-terminal in background (BRIDGE_TOKEN fixed from ~/.zshrc)..."
	@TOKEN=$$(grep '^export BRIDGE_TOKEN=' $$HOME/.zshrc 2>/dev/null | head -1 | cut -d= -f2); \
	if [ -z "$$TOKEN" ]; then \
		echo "ERROR: BRIDGE_TOKEN が ~/.zshrc に見つかりません。'export BRIDGE_TOKEN=<32hex>' を追加してください。"; \
		exit 1; \
	fi; \
	echo "Using BRIDGE_TOKEN=$$TOKEN"; \
	BRIDGE_TOKEN=$$TOKEN nohup /opt/homebrew/bin/even-terminal --tailscale --provider claude > /tmp/even-terminal.log 2>&1 &
	@sleep 3
	@echo "even-terminal started on :3456"

stop-even-terminal:
	-pkill -f "even-terminal" 2>/dev/null || true
	-lsof -ti:3456 | xargs kill -9 2>/dev/null || true

restart-even-terminal: stop-even-terminal start-even-terminal

# Even G2(スマートグラス)用 even-terminal 起動
# 既存セッションを kill してから G2 向けに新規起動。
# --name G2 で識別、ログも /tmp/even-terminal-g2.log に分離。
# VOICEVOX(17 GB メモリ)は任意起動。音声対話が要る時は別途 make start-voicevox。
.PHONY: g2 stop-g2
g2: stop-even-terminal
	@echo "Starting even-terminal for Even G2..."
	@TOKEN=$$(grep '^export BRIDGE_TOKEN=' $$HOME/.zshrc 2>/dev/null | head -1 | cut -d= -f2); \
	if [ -z "$$TOKEN" ]; then \
		echo "ERROR: BRIDGE_TOKEN が ~/.zshrc に見つかりません。"; \
		exit 1; \
	fi; \
	echo "Using BRIDGE_TOKEN=$$TOKEN"; \
	BRIDGE_TOKEN=$$TOKEN nohup /opt/homebrew/bin/even-terminal \
		--tailscale --provider claude --name G2 \
		--cwd $(PROJECT_ROOT) \
		> /tmp/even-terminal-g2.log 2>&1 &
	@sleep 4
	@echo ""
	@echo "===== Even G2 接続 QR (G2 グラスでスキャン) ====="
	@# QR コード行は箱罫線文字 (█▀▄) で構成される。connection URL も含めて先頭の起動メッセージを表示
	@awk '/^http:\/\/.+\?token=/{print; flag=1; next} flag && (/^\[/ || /Logging to/){exit} flag{print}' /tmp/even-terminal-g2.log
	@echo "================================================"
	@echo "even-terminal (G2 mode) started on :3456"
	@echo "詳細ログ: /tmp/even-terminal-g2.log"

stop-g2: stop-even-terminal

# patrol_projects.json の全プロジェクトを並列起動(各 port は 3456 から index 順)
# G2 アプリは複数接続先を保存・切替可能(実機検証済)。
# 1 回登録すれば G2 側でプロジェクト切替できるので、Mac 側の操作は不要。
.PHONY: g2-all stop-g2-all g2-status
g2-all: stop-g2-all
	@echo "Starting even-terminal for all patrol projects in parallel..."
	@TOKEN=$$(grep '^export BRIDGE_TOKEN=' $$HOME/.zshrc 2>/dev/null | head -1 | cut -d= -f2); \
	if [ -z "$$TOKEN" ]; then echo "ERROR: BRIDGE_TOKEN not found in ~/.zshrc"; exit 1; fi; \
	python3 -c "import json; d=json.load(open('$(PROJECT_ROOT)/devtools/backend/patrol_projects.json')); [print(f'{i+3456} {p[\"name\"]} {p[\"path\"]}') for i, p in enumerate(d['projects'])]" > /tmp/g2-all-projects.txt; \
	while read PORT NAME PROJECT_PATH; do \
		echo "  -> $$NAME on :$$PORT ($$PROJECT_PATH)"; \
		BRIDGE_TOKEN=$$TOKEN nohup /opt/homebrew/bin/even-terminal \
			--tailscale --provider claude --name "$$NAME" \
			--cwd "$$PROJECT_PATH" \
			-p $$PORT \
			> /tmp/even-terminal-$$NAME.log 2>&1 & \
	done < /tmp/g2-all-projects.txt
	@sleep 6
	@echo ""
	@while read PORT NAME PROJECT_PATH; do \
		echo "===== $$NAME 接続 QR (:$$PORT) ====="; \
		awk '/^http:\/\/.+\?token=/{print; flag=1; next} flag && (/^\[/ || /Logging to/){exit} flag{print}' /tmp/even-terminal-$$NAME.log; \
		echo "==============================================="; \
		echo ""; \
	done < /tmp/g2-all-projects.txt
	@rm -f /tmp/g2-all-projects.txt
	@echo "全 even-terminal インスタンス起動完了。G2 アプリで全 QR を登録してください。"
	@echo "状態確認: make g2-status / 停止: make stop-g2-all"

stop-g2-all:
	-pkill -f "/opt/homebrew/bin/even-terminal" 2>/dev/null || true
	-for p in 3456 3457 3458 3459 3460 3461 3462; do \
		lsof -ti:$$p 2>/dev/null | xargs kill -9 2>/dev/null || true; \
	done

g2-status:
	@echo "=== running even-terminal instances ==="
	@lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null | awk '/node.*:(345[0-9]|346[0-9])/ {for(i=1;i<=NF;i++) if($$i ~ /:345[0-9]|:346[0-9]/) print "  " $$i, "PID=" $$2}'
	@echo ""
	@echo "=== ログファイル ==="
	@ls -la /tmp/even-terminal-*.log 2>/dev/null | awk '{print "  " $$NF " (" $$5 " bytes)"}'

# 起動済み even-terminal の QR を再表示(再起動なし)。G2 への登録忘れリカバリ用。
.PHONY: g2-qr
g2-qr:
	@python3 -c "import json; d=json.load(open('$(PROJECT_ROOT)/devtools/backend/patrol_projects.json')); [print(p['name']) for p in d['projects']]" | \
	while read NAME; do \
		LOG=/tmp/even-terminal-$$NAME.log; \
		if [ -f "$$LOG" ]; then \
			echo "===== $$NAME 接続 QR ====="; \
			awk '/^http:\/\/.+\?token=/{print; flag=1; next} flag && (/^\[/ || /Logging to/){exit} flag{print}' "$$LOG"; \
			echo "==============================================="; \
			echo ""; \
		else \
			echo "===== $$NAME: ログなし(起動していない?)====="; \
			echo ""; \
		fi; \
	done

# VOICEVOX on-demand 起動/停止
# VOICEVOX Engine は起動中 1-17 GB のメモリを保持する。使わない時は停止しておく。
.PHONY: start-voicevox stop-voicevox restart-voicevox
start-voicevox:
	@echo "Starting VOICEVOX.app..."
	@open -a VOICEVOX
	@echo "Waiting for VOICEVOX Engine on :50021..."
	@for i in $$(seq 1 30); do \
		if curl -s -o /dev/null --max-time 1 http://127.0.0.1:50021/version; then \
			echo "VOICEVOX Engine ready ($$i seconds)"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "ERROR: VOICEVOX Engine did not become ready within 30s"; \
	exit 1

stop-voicevox:
	-osascript -e 'tell application "VOICEVOX" to quit' 2>/dev/null || true
	-pkill -f "VOICEVOX.app/Contents" 2>/dev/null || true
	-lsof -ti:50021 | xargs kill -9 2>/dev/null || true

restart-voicevox: stop-voicevox start-voicevox

# 停止
.PHONY: stop-backend stop-frontend stop

stop-backend:
	-pkill -f "go run.*cmd/server" || true
	-pkill -f "backend/server" || true
	-lsof -ti:8888 | xargs kill -9 2>/dev/null || true

stop-frontend:
	-pkill -f "next dev" || true
	-pkill -f "npm.*dev" || true
	-pkill -f "npm.*start" || true
	-lsof -ti:3333 | xargs kill -9 2>/dev/null || true

stop: stop-backend stop-frontend stop-even-terminal stop-voicevox

# メモリリーク保険(launchd auto-restart) - 詳細は scripts/launchd/maybe-restart-backend.sh
# 毎日 03:00 にチェック → RSS > 1 GB かつ 巡回アイドル の時だけ restart 実行
.PHONY: install-restart-cron uninstall-restart-cron status-restart-cron
install-restart-cron:
	@chmod +x $(PROJECT_ROOT)/scripts/launchd/maybe-restart-backend.sh
	@cp $(PROJECT_ROOT)/scripts/launchd/com.ghostrunner.backend-restart.plist $$HOME/Library/LaunchAgents/
	@launchctl unload $$HOME/Library/LaunchAgents/com.ghostrunner.backend-restart.plist 2>/dev/null || true
	@launchctl load $$HOME/Library/LaunchAgents/com.ghostrunner.backend-restart.plist
	@echo "Installed: 毎日 03:00 にガード付き auto-restart チェック実行"
	@echo "ログ: /tmp/ghostrunner-backend-restart.log"

uninstall-restart-cron:
	-launchctl unload $$HOME/Library/LaunchAgents/com.ghostrunner.backend-restart.plist 2>/dev/null || true
	-rm -f $$HOME/Library/LaunchAgents/com.ghostrunner.backend-restart.plist
	@echo "Uninstalled launchd auto-restart"

status-restart-cron:
	@echo "=== launchctl list ==="
	@launchctl list | grep ghostrunner || echo "(未登録)"
	@echo ""
	@echo "=== 最新ログ (tail) ==="
	@if [ -f /tmp/ghostrunner-backend-restart.log ]; then tail -20 /tmp/ghostrunner-backend-restart.log; else echo "(まだ実行されていません)"; fi

# 再起動（kill + start、バックグラウンド）
.PHONY: restart-backend restart-frontend restart

restart-backend: stop-backend
	@sleep 1
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server' > /tmp/backend.log 2>&1 &

restart-frontend: stop-frontend
	@sleep 1
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &

restart:
	@make -j2 restart-backend restart-frontend

# 再起動（kill + start + ログ表示）
.PHONY: restart-backend-logs restart-frontend-logs

restart-backend-logs: stop-backend
	@echo "Restarting backend..."
	@sleep 1
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/backend && [ -f .env ] && set -a && . ./.env && set +a; go run ./cmd/server' > /tmp/backend.log 2>&1 &
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
	@nohup sh -c 'cd $(DEVTOOLS_ROOT)/frontend && npm run dev' > /tmp/frontend.log 2>&1 &
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
	cd $(DEVTOOLS_ROOT)/backend && go build -o server ./cmd/server
	cd $(DEVTOOLS_ROOT)/backend && go build -o gr-run ./cmd/gr-run
	cd $(DEVTOOLS_ROOT)/frontend && npm run build

gr-run:
	cd $(DEVTOOLS_ROOT)/backend && go build -o gr-run ./cmd/gr-run

# ヘルスチェック
.PHONY: health

health:
	@curl -s http://localhost:8888/api/health || echo "Backend: NG"
	@curl -s http://localhost:3333 > /dev/null && echo "Frontend: OK" || echo "Frontend: NG"

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


