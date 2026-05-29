#!/usr/bin/env python3
"""patrol_projects.json を読み、各プロジェクトの even-terminal 起動ポートを解決する。

各プロジェクトにつき `<port> <name> <path>` を標準出力に 1 行ずつ出力する。
ポート決定の優先順位:
  1. <path>/.claude/ports.json の "even_terminal" キー（プロジェクト固有の台帳）
  2. 台帳が無い・壊れている・キーが無い場合は index + 3456（index 連番フォールバック・後方互換）

make g2-all / g2-qr / stop-g2-all から共通で利用する。
1 プロジェクトの台帳破損が他プロジェクトの解決に波及しないよう、読み取り失敗は握りつぶす。
"""
import json
import os
import sys

# 台帳が無いプロジェクトのフォールバック起点ポート（従来の make g2-all 互換）
BASE_PORT = 3456


def resolve_port(project_path, index):
    """プロジェクトの even-terminal ポートを決定する。

    台帳 (.claude/ports.json) の even_terminal を最優先。
    台帳が無い・壊れている・キーが無効な場合は index + BASE_PORT にフォールバックする。
    """
    ports_file = os.path.join(project_path, ".claude", "ports.json")
    try:
        with open(ports_file, encoding="utf-8") as f:
            data = json.load(f)
        port = data.get("even_terminal")
        if isinstance(port, int) and port > 0:
            return port
    except (OSError, ValueError):
        # 台帳が読めない/壊れている → フォールバック
        pass
    return index + BASE_PORT


def main():
    if len(sys.argv) < 2:
        print("usage: g2_resolve_ports.py <patrol_projects.json>", file=sys.stderr)
        return 1

    patrol_path = sys.argv[1]
    try:
        with open(patrol_path, encoding="utf-8") as f:
            config = json.load(f)
    except (OSError, ValueError) as exc:
        print(f"failed to read {patrol_path}: {exc}", file=sys.stderr)
        return 1

    for index, project in enumerate(config.get("projects", [])):
        path = project.get("path", "")
        name = project.get("name", "")
        if not path or not name:
            continue
        print(f"{resolve_port(path, index)} {name} {path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
