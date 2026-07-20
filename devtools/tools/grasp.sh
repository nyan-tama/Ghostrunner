#!/usr/bin/env bash
# 統括ライブ把握(CLI) - GET /api/dashboard/state を叩き、全プロジェクトの横断ボードを1回描画する。
# 質問待ち(idle)のプロジェクトは [質問待ち N分] + 何を待っているかの先頭抜粋を赤で表示する。
# ライブ表示(2秒ごと再描画)は `make grasp` を使う。
set -euo pipefail

API="${GRASP_API:-http://localhost:8888/api/dashboard/state}"
R=$'\033[31m'; Y=$'\033[33m'; DIM=$'\033[2m'; B=$'\033[1m'; Z=$'\033[0m'

S=$(curl -s --max-time 3 "$API" 2>/dev/null || true)
if [ -z "$S" ]; then
  printf '%s統括ライブ把握%s  バックエンド未応答 (%s)\n' "$B" "$Z" "$API"
  printf '  %smake backend で起動してください%s\n' "$DIM" "$Z"
  exit 0
fi

NOW=$(date +%s)
TS=$(printf '%s' "$S" | jq -r '.generatedAt // "-"')
printf '%s統括ライブ把握%s  %s\n' "$B" "$Z" "$TS"
printf '%s────────────────────────────────────────────────────────────────%s\n' "$DIM" "$Z"

# バックエンドが (質問待ち>required>progress>watching) 順にソート済み。再ソートしない。
# フィールド: attention  name  kanban  unanswered  idleTimestamp("" なら非質問待ち)  preview
printf '%s' "$S" | jq -r '
  .projects[]
  | [ .attention,
      .name,
      "\(.kanban.waiting)/\(.kanban.running)/\(.kanban.done)",
      (.unanswered|length),
      (.idle.timestamp // ""),
      (if .idle then (.idle.preview // "" | gsub("[\n\r\t]+";" ") | .[0:44]) else "" end)
    ] | @tsv
' | while IFS=$'\t' read -r att name kanban unans idlets preview; do
  case "$att" in
    required) mk="${R}●${Z}"; lab="${R}要対応${Z}" ;;
    progress) mk="${Y}◐${Z}"; lab="${Y}進行  ${Z}" ;;
    watching) mk="${DIM}○ 静観${Z}"; lab="" ;;
    *)        mk="${DIM}·${Z}";  lab="" ;;
  esac

  idle=""
  if [ -n "$idlets" ]; then
    # RFC3339(+09:00等)を date で epoch 化。オフセットを外しローカル時刻として解釈(同一マシン前提)。
    local_ts="${idlets%%+*}"; local_ts="${local_ts%%Z*}"
    epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "$local_ts" +%s 2>/dev/null || echo "")
    if [ -n "$epoch" ]; then
      min=$(( (NOW - epoch) / 60 )); [ "$min" -lt 0 ] && min=0
      idle="  ${R}${B}[質問待ち ${min}分]${Z} ${DIM}${preview}${Z}"
    else
      idle="  ${R}${B}[質問待ち]${Z} ${DIM}${preview}${Z}"
    fi
  fi

  un=""; [ "$unans" != "0" ] && un=" ${R}未回答${unans}${Z}"
  printf '%s %s %-22s %s待%s%s%s%s\n' "$mk" "$lab" "$name" "$DIM" "$Z" "$kanban" "$un" "$idle"
done

printf '%s────────────────────────────────────────────────────────────────%s\n' "$DIM" "$Z"
printf '%s待N/実N/完N=実装待ち/実行中/完了   [質問待ち N分]=セッションが返答待ち(見失い防止)%s\n' "$DIM" "$Z"
