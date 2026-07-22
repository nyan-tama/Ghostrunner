#!/usr/bin/env bash
# スマホ直結操作アプリ (案A: Blink + Tailscale + tmux) のセッション管理ヘルパ。
# 「各プロジェクト = 1 tmux セッション」を patrol_projects.json から実現する。
# セッション名 = プロジェクト名。cwd = プロジェクトのパス。
# Mac の VS Code ターミナルもスマホ(Blink)も同じセッションに attach すれば PTY が完全一致する。
#
# 使い方:
#   gr-tmux ls            登録プロジェクトとセッション状態(●起動中/○未起動)を一覧
#   gr-tmux up [name...]  未起動のセッションを detached で作成(引数省略で全プロジェクト)
#   gr-tmux attach <name> セッションに attach(無ければ作成してから)。スマホ側はこれを叩く
#   gr-tmux kill <name>   セッションを終了
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PROJECTS_JSON="${GR_PROJECTS_JSON:-$ROOT/devtools/backend/patrol_projects.json}"
G=$'\033[32m'; DIM=$'\033[2m'; B=$'\033[1m'; Z=$'\033[0m'

if [ ! -f "$PROJECTS_JSON" ]; then
  echo "patrol_projects.json が見つかりません: $PROJECTS_JSON" >&2
  exit 1
fi

# name<TAB>path を1行ずつ返す
projects() {
  jq -r '.projects[] | "\(.name)\t\(.path)"' "$PROJECTS_JSON"
}

path_of() {
  jq -r --arg n "$1" '.projects[] | select(.name==$n) | .path' "$PROJECTS_JSON"
}

has_session() {
  tmux has-session -t "=$1" 2>/dev/null
}

# セッションを作成(なければ)。存在すれば何もしない。
ensure_session() {
  local name="$1" path
  path="$(path_of "$name")"
  if [ -z "$path" ]; then
    echo "未登録のプロジェクト: $name" >&2
    return 1
  fi
  if ! has_session "$name"; then
    tmux new-session -d -s "$name" -c "$path"
  fi
}

cmd_ls() {
  printf '%s登録プロジェクト(セッション状態)%s  %s%s%s\n' "$B" "$Z" "$DIM" "$PROJECTS_JSON" "$Z"
  while IFS=$'\t' read -r name path; do
    if has_session "$name"; then
      printf '  %s● %-24s%s %s%s%s\n' "$G" "$name" "$Z" "$DIM" "$path" "$Z"
    else
      printf '  %s○ %-24s%s %s%s%s\n' "$DIM" "$name" "$Z" "$DIM" "$path" "$Z"
    fi
  done < <(projects)
}

cmd_up() {
  local targets=("$@")
  if [ "${#targets[@]}" -eq 0 ]; then
    while IFS=$'\t' read -r name _; do targets+=("$name"); done < <(projects)
  fi
  for name in "${targets[@]}"; do
    if has_session "$name"; then
      printf '  %s● %s (既起動)%s\n' "$DIM" "$name" "$Z"
    else
      ensure_session "$name" && printf '  %s● %s (作成)%s\n' "$G" "$name" "$Z"
    fi
  done
}

cmd_attach() {
  local name="${1:-}"
  [ -z "$name" ] && { echo "使い方: gr-tmux attach <name>" >&2; exit 1; }
  ensure_session "$name"
  # tmux 内から呼ばれた場合は switch、外からは attach
  if [ -n "${TMUX:-}" ]; then
    tmux switch-client -t "=$name"
  else
    tmux attach-session -t "=$name"
  fi
}

cmd_kill() {
  local name="${1:-}"
  [ -z "$name" ] && { echo "使い方: gr-tmux kill <name>" >&2; exit 1; }
  tmux kill-session -t "=$name" && echo "終了: $name"
}

case "${1:-ls}" in
  ls|list) cmd_ls ;;
  up)      shift; cmd_up "$@" ;;
  attach|a) shift; cmd_attach "$@" ;;
  kill)    shift; cmd_kill "$@" ;;
  *) echo "使い方: gr-tmux {ls|up [name...]|attach <name>|kill <name>}" >&2; exit 1 ;;
esac
