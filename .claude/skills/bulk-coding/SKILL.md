---
name: bulk-coding
description: 全プロジェクトの実装待ちタスクを一括でgr-run起動する
---


# /bulk-coding

**ultrathink**

全登録プロジェクトの実装待ちタスクをスキャンし、対象プロジェクトに対して gr-run を背景起動する。

**発火条件**: 「一括codingして」「実装待ちを一括で開始」等の**明示動詞**でのみ呼ばれる。
「状況は？」「今日どう？」といった把握系の問いかけでは呼ばない（chief-director を使う）。

## 手順

### 1. 状態スキャン

`devtools/backend/patrol_projects.json` の `projects[]` を読み、各プロジェクトの状態を Glob で確認する。

```
Read: devtools/backend/patrol_projects.json
```

各プロジェクトの `path` に対して:
- `Glob <path>/開発/実装/実行中/*.md` — 実行中タスクの有無
- `Glob <path>/開発/実装/実装待ち/*.md` — 実装待ちタスクの一覧

### 2. 対象選定

以下の条件を**両方**満たすプロジェクトのみ対象とする:
- `実行中/` が空（= 他の gr-run や手動 /coding が走っていない）
- `実装待ち/` に `.md` ファイルが1件以上ある

1プロジェクト = 同時1タスク。対象プロジェクトごとに `実装待ち/` の**最古1件**（ファイル名の昇順で先頭）を選ぶ。

### 3. gr-run ビルド確認

gr-run バイナリの絶対パスを算出する:

```bash
# Ghostrunner プロジェクトルートからの相対パス
GR_RUN="$(cd "$(dirname "$0")/../.." && pwd)/devtools/backend/gr-run"
```

実際にはスキル実行時に以下で算出:
- Ghostrunner のプロジェクトルートは `devtools/backend/patrol_projects.json` がある位置の2階層上
- gr-run バイナリ: `<プロジェクトルート>/devtools/backend/gr-run`

バイナリが存在しない場合は `make gr-run` でビルドを試みる。それでも失敗したらエラー報告して終了。

### 4. ディスパッチ

対象プロジェクトごとに gr-run を背景起動する。macOS は setsid が非標準のため使わない。

```bash
nohup <gr-run絶対パス> --project <プロジェクト絶対パス> --task <タスクファイル名> </dev/null >/dev/null 2>&1 &
disown
```

### 5. 報告

起動結果を一覧表示する:

```
[起動] face-search: 2026-05-24_顔認識API改善_plan.md
[起動] akiba-media: 2026-05-24_記事一覧API_plan.md
[スキップ] sns-poster: 実行中タスクあり（2026-05-20_投稿機能_plan.md）
[スキップ] Ghostrunner: 実装待ちなし
```

末尾に案内を追加:
```
完了・確認事項は ntfy 通知で届きます。
状況の確認は「状況は？」で chief-director に聞いてください。
```

## 注意事項

- gr-run の起動のみを行い、実行結果は待たない（背景プロセスとして切り離す）
- 通知は各 gr-run プロセスが ntfy 経由で送信する
- 対象の選定は Glob によるフォルダ確認のみ。chief-director の出力テキストをパースしない
- Ghostrunner 自身のタスクも patrol_projects.json に含まれていれば対象になる


## タスク

$ARGUMENTS
