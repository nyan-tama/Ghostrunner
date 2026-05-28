#!/usr/bin/env bash
# Ghostrunner backend のリーク疑い時のみ restart するガード付きスクリプト
#
# 動作:
#   1. backend (go run ./cmd/server) が動いているか確認
#   2. RSS が閾値(デフォルト 1 GB)を超えているか確認
#   3. /api/patrol/states で polling 中のプロジェクトが無いか確認(中断回避)
#   4. 上記すべて満たしたときだけ make restart-backend を実行
#
# launchd から毎日 03:00 に呼ばれる想定。
# 通常運用(RSS 数十 MB)では何もしない。巡回中も何もしない。

set -uo pipefail

GHOSTRUNNER_ROOT="/Users/user/Ghostrunner"
LOG_FILE="/tmp/ghostrunner-backend-restart.log"
RSS_THRESHOLD_KB=$((1 * 1024 * 1024))   # 1 GB
BACKEND_HOST="http://localhost:8888"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE"
}

log "=== maybe-restart-backend check start ==="

# 1) backend プロセスを探す
# `go run` は実は 3 階層のプロセスを作る:
#   - 親 bash (envvars 設定)
#   - go ビルドツール
#   - /var/folders/.../go-build/.../server  ← これが実サーバー、:8888 を LISTEN
# 17 GB リークするのは実サーバー側。:8888 を LISTEN している PID を直接見るのが最確実。
BACKEND_PID=$(lsof -nP -iTCP:8888 -sTCP:LISTEN 2>/dev/null | awk 'NR>1 {print $2; exit}')

if [ -z "$BACKEND_PID" ]; then
    log "backend not running, skip"
    exit 0
fi

# 2) RSS 取得 (KB)
RSS_KB=$(ps -p "$BACKEND_PID" -o rss= 2>/dev/null | tr -d ' ' || true)
if [ -z "$RSS_KB" ]; then
    log "could not get RSS for PID $BACKEND_PID, skip"
    exit 0
fi

RSS_MB=$((RSS_KB / 1024))
log "PID=$BACKEND_PID RSS=${RSS_MB} MB (threshold $((RSS_THRESHOLD_KB / 1024)) MB)"

if [ "$RSS_KB" -lt "$RSS_THRESHOLD_KB" ]; then
    log "RSS under threshold, skip (healthy)"
    exit 0
fi

# 3) 巡回 polling 中か確認
PATROL_STATES=$(curl -s --max-time 3 "${BACKEND_HOST}/api/patrol/states" || echo "")
if echo "$PATROL_STATES" | grep -q '"polling":true'; then
    log "patrol polling is active, skip (avoid interruption)"
    exit 0
fi

# 4) 条件すべて満たした: restart 実行
log "RSS over threshold AND patrol idle → restart-backend"
cd "$GHOSTRUNNER_ROOT"
/usr/bin/make restart-backend >> "$LOG_FILE" 2>&1
log "restart-backend done, exit code=$?"
log "=== end ==="
