# 既存プロジェクトの .claude/ 更新 実装計画

## 概要

既存プロジェクトの `.claude/` フォルダを Ghostrunner 本体の最新版で上書き更新する `/update` スキルを作成する。

## 背景

- Ghostrunner で作成済みプロジェクトは作成時点の `.claude/` がそのまま残る
- Ghostrunner 側でエージェント・スキル・設定が更新されても既存プロジェクトには反映されない
- `/update` を手動実行することで最新化する

## 実装内容

### 変更ファイル

| ファイル | 操作 | 内容 |
|---------|------|------|
| `.claude/skills/update/SKILL.md` | 新規作成 | /update スキル定義 |

### SKILL.md の処理フロー

```mermaid
flowchart TD
    A[/update 実行] --> B[Ghostrunner パス特定]
    B --> C{Ghostrunner が存在するか?}
    C -->|なし| D[エラー: Ghostrunner が見つかりません]
    C -->|あり| P[git pull で最新化]
    P --> E[既存の agents/ skills/ を削除]
    E --> F[agents/ を上書きコピー]
    F --> G[skills/ を上書きコピー]
    G --> H[settings.json を上書きコピー]
    H --> I[不要エージェント削除判定]
    I --> J[CLAUDE.md は更新しない]
    J --> K[完了メッセージ表示]
```

### 詳細仕様

**Step 0: Ghostrunner を最新化**
- Ghostrunner のパスを特定（環境変数 `GHOSTRUNNER_HOME` or `~/Ghostrunner`）
- 存在しなければエラー終了
- `cd <Ghostrunner> && git pull` で最新版を取得
- これにより、ユーザーは `/update` を実行するだけで Ghostrunner の最新化 + プロジェクト更新が一気に行われる

**Step 1: コピー前準備**

**Step 2: バックアップ**
- `.claude/agents/` と `.claude/skills/` の既存ファイルを削除してからコピー（古いファイルが残らないように）
- `settings.json` は上書き

**Step 3: コピー**
- `cp Ghostrunner/.claude/agents/*.md ./.claude/agents/`
- `cp -r Ghostrunner/.claude/skills/ ./.claude/skills/`
- `cp Ghostrunner/.claude/settings.json ./.claude/settings.json`

**Step 4: 不要エージェント削除**
- プロジェクトに `docker-compose.yml` がある場合、その中身を確認
  - PostgreSQL サービスがなければ `pg-*.md` を削除
  - MinIO/S3 サービスがなければ `storage-*.md` を削除
  - Redis サービスがなければ `redis-*.md` を削除
- `docker-compose.yml` がなければ全て残す（手動で不要なものを削除してもらう）

**Step 5: CLAUDE.md は更新しない**
- CLAUDE.md はプロジェクト固有の内容（プロジェクト名、概要、ビルドコマンド等）を含むため上書きしない
- ユーザーが希望すれば手動で更新

**Step 6: 完了メッセージ**
- 更新されたファイル数を表示
- `agents/` のファイル数、`skills/` のディレクトリ数を表示

### 除外事項

- `CLAUDE.md` の自動更新（プロジェクト固有のため）
- `settings.local.json` の上書き（環境固有のため）
- ホームUIからの一括更新（手動で十分）
- バージョン管理・差分チェック（全上書きで十分）

## テストプラン

SKILL.md のみの変更のため、自動テストは不要。手動確認:

1. 既存プロジェクトで `/update` を実行
2. `.claude/agents/` が最新のエージェント一覧に更新されていること
3. `.claude/skills/` が最新のスキル一覧に更新されていること
4. `.claude/settings.json` が更新されていること
5. `CLAUDE.md` が変更されていないこと
6. DB なしプロジェクトで `pg-*.md` が削除されていること
