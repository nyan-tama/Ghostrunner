#!/bin/bash
# 質問待ち検知POC: ユーザーが送信した(UserPromptSubmit)瞬間にマーカーを消す。
set -u
INPUT=$(cat)
MARKER_DIR="$HOME/.claude/gr-idle-markers"
SID=$(printf '%s' "$INPUT" | jq -r '.session_id // ""')
NOW=$(date +%s)

printf '%s\tevent=UserPromptSubmit\tsid=%s\tACTION=unmark\n' "$NOW" "$SID" >> "$MARKER_DIR/_debug.log"

[ -n "$SID" ] && rm -f "$MARKER_DIR/${SID}.idle"
exit 0
