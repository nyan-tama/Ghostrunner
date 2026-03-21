# /update

プロジェクトの `.claude/` フォルダを Ghostrunner の最新版で更新します。

## 処理手順

### Step 0: Ghostrunner を最新化

Ghostrunner のパスを特定し、最新版を取得する。

1. 環境変数 `GHOSTRUNNER_HOME` が設定されていればそのパスを使用
2. なければ `~/Ghostrunner` を使用
3. パスが存在しなければエラー終了: 「Ghostrunner が見つかりません。GHOSTRUNNER_HOME 環境変数を設定してください。」

```bash
cd <GHOSTRUNNER_PATH> && git checkout main && git pull origin main
```

### Step 1: 既存ファイルを削除

古いファイルが残らないよう、既存の agents/ と skills/ を削除してからコピーする。

```bash
rm -rf ./.claude/agents/
rm -rf ./.claude/skills/
mkdir -p ./.claude/agents
mkdir -p ./.claude/skills
```

### Step 2: Ghostrunner から最新ファイルをコピー

```bash
cp <GHOSTRUNNER_PATH>/.claude/agents/*.md ./.claude/agents/
cp -r <GHOSTRUNNER_PATH>/.claude/skills/* ./.claude/skills/
cp <GHOSTRUNNER_PATH>/.claude/settings.json ./.claude/settings.json
```

### Step 3: 不要エージェントの削除

プロジェクトの `docker-compose.yml` を確認し、使用していないサービスのエージェントを削除する。

- `docker-compose.yml` が存在しない場合: 全エージェントを残す
- PostgreSQL サービスがない場合: `rm -f ./.claude/agents/pg-*.md`
- MinIO/S3 サービスがない場合: `rm -f ./.claude/agents/storage-*.md`
- Redis サービスがない場合: `rm -f ./.claude/agents/redis-*.md`

### Step 4: 更新しないファイル

以下のファイルは更新しない（プロジェクト固有の内容を含むため）:

- `.claude/CLAUDE.md` - プロジェクト固有の設定・ルール
- `.claude/settings.local.json` - 環境固有の設定

### Step 5: 完了メッセージ

更新結果を表示する:

```
.claude/ を最新版に更新しました。

  agents: <N>個
  skills: <N>個
  settings.json: 更新済み

CLAUDE.md は更新されていません（プロジェクト固有のため）。
必要に応じて手動で更新してください。
```
