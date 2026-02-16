#!/bin/bash
set -euo pipefail

# ntfy Mac PC 通知受信アンインストールスクリプト
# LaunchAgent と設定ファイルを削除する

PLIST_LABEL="sh.ntfy.subscriber"
PLIST_PATH="$HOME/Library/LaunchAgents/$PLIST_LABEL.plist"
CONFIG_DIR="$HOME/Library/Application Support/ntfy"
CONFIG_FILE="$CONFIG_DIR/client.yml"

echo "=== ntfy Mac 通知アンインストール ==="
echo ""

# 1. LaunchAgent の停止
if [ -f "$PLIST_PATH" ]; then
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    echo "[OK] LaunchAgent を停止しました"
else
    echo "[SKIP] LaunchAgent は登録されていません"
fi

# 2. plist ファイルの削除
if [ -f "$PLIST_PATH" ]; then
    rm "$PLIST_PATH"
    echo "[OK] plist 削除: $PLIST_PATH"
else
    echo "[SKIP] plist は存在しません"
fi

# 3. 設定ファイルの削除
if [ -f "$CONFIG_FILE" ]; then
    rm "$CONFIG_FILE"
    echo "[OK] 設定ファイル削除: $CONFIG_FILE"
else
    echo "[SKIP] 設定ファイルは存在しません"
fi

# 設定ディレクトリが空なら削除
if [ -d "$CONFIG_DIR" ] && [ -z "$(ls -A "$CONFIG_DIR")" ]; then
    rmdir "$CONFIG_DIR"
    echo "[OK] 設定ディレクトリ削除: $CONFIG_DIR"
fi

# 4. 完了メッセージ
echo ""
echo "=== アンインストール完了 ==="
echo ""
echo "ntfy CLI 自体はアンインストールされていません。"
echo "ntfy CLI も削除する場合: brew uninstall ntfy"
