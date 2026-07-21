#!/usr/bin/env bash
# 統括ライブ把握(CLI) - GET /api/dashboard/state を叩き、全プロジェクトの横断ボードを1回描画する。
# 各プロジェクトを3状態で表示: 質問待ち(idle・赤) / 動作中(running・青) / 静観(灰)。
# 質問待ちは [質問待ち N分]+要約/抜粋、動作中は [動作中]+今やってる内容の抜粋。
# ライブ表示(2秒ごと再描画)は `make grasp` を使う。
set -euo pipefail

API="${GRASP_API:-http://localhost:8888/api/dashboard/state}"
R=$'\033[31m'; Y=$'\033[33m'; C=$'\033[36m'; DIM=$'\033[2m'; B=$'\033[1m'; Z=$'\033[0m'

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

# バックエンドが (質問待ち>動作中>required>progress>watching) 順にソート済み。再ソートしない。
# フィールド: attention name kanban unanswered idleTimestamp("" なら非質問待ち) idleText runningText("" なら非動作中)
printf '%s' "$S" | jq -r '
  .projects[]
  | [ .attention,
      .name,
      "\(.kanban.waiting)/\(.kanban.running)/\(.kanban.done)",
      (.unanswered|length),
      (.idle.timestamp // ""),
      (if .idle then ((if (.idle.summary // "") != "" then .idle.summary else (.idle.preview // "") end) | gsub("[\n\r\t]+";" ") | .[0:48]) else "" end),
      (if .running then (.running.preview // "" | gsub("[\n\r\t]+";" ") | .[0:48]) else "" end)
    ] | @tsv
' | while IFS=$'\t' read -r att name kanban unans idlets preview runprev; do
  case "$att" in
    required) mk="${R}●${Z}"; lab="${R}要対応${Z}" ;;
    progress) mk="${Y}◐${Z}"; lab="${Y}進行  ${Z}" ;;
    watching) mk="${DIM}○ 静観${Z}"; lab="" ;;
    *)        mk="${DIM}·${Z}";  lab="" ;;
  esac

  badge=""
  if [ -n "$idlets" ]; then
    # 質問待ち: RFC3339(+09:00等)を date で epoch 化(オフセットを外しローカル時刻・同一マシン前提)。
    local_ts="${idlets%%+*}"; local_ts="${local_ts%%Z*}"
    epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "$local_ts" +%s 2>/dev/null || echo "")
    if [ -n "$epoch" ]; then
      min=$(( (NOW - epoch) / 60 )); [ "$min" -lt 0 ] && min=0
      badge="  ${R}${B}[質問待ち ${min}分]${Z} ${DIM}${preview}${Z}"
    else
      badge="  ${R}${B}[質問待ち]${Z} ${DIM}${preview}${Z}"
    fi
    mk="${R}●${Z}"; lab="${R}要対応${Z}"
  elif [ -n "$runprev" ]; then
    # 動作中: 青。Claudeが今作業中(自分待ちではない)。
    badge="  ${C}${B}[動作中]${Z} ${DIM}${runprev}${Z}"
    mk="${C}◐${Z}"; lab="${C}動作中${Z}"
  fi

  un=""; [ "$unans" != "0" ] && un=" ${R}未回答${unans}${Z}"
  printf '%s %s %-22s %s待%s%s%s%s\n' "$mk" "$lab" "$name" "$DIM" "$Z" "$kanban" "$un" "$badge"
done

printf '%s────────────────────────────────────────────────────────────────%s\n' "$DIM" "$Z"
printf '%s待N/実N/完N=実装待ち/実行中/完了   %s[質問待ち]%s=自分待ち  %s[動作中]%s=Claude作業中%s\n' "$DIM" "$R" "$DIM" "$C" "$DIM" "$Z"
