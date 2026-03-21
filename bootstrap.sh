#!/bin/bash
# 既存プロジェクトに /update スキルをインストールする
# 使い方: cd ~/my-project && ~/Ghostrunner/bootstrap.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ ! -d ".claude" ]; then
  echo ".claude/ フォルダが見つかりません。Ghostrunner で作成されたプロジェクトのディレクトリで実行してください。"
  exit 1
fi

mkdir -p .claude/skills/update
cp "$SCRIPT_DIR/.claude/skills/update/SKILL.md" .claude/skills/update/SKILL.md

echo "/update スキルをインストールしました。claude /update を実行してください。"
