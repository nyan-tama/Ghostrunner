# Backend 運用手順書

Ghostrunner バックエンドの運用に関する手順とトラブルシューティング。

## サーバー管理

### 起動

```bash
# フォアグラウンドで起動
make backend

# バックグラウンドで起動してログ表示
make start-backend

# バックエンド + フロントエンドを同時起動
make dev
```

### 停止

```bash
make stop-backend
```

### 再起動

```bash
# バックグラウンドで再起動
make restart-backend

# 再起動してログ表示
make restart-backend-logs
```

### ログ確認

```bash
make logs-backend
```

ログファイルは `/tmp/backend.log` に出力される。

### ヘルスチェック

```bash
make health
# または
curl http://localhost:8080/api/health
```

レスポンス: `{"status":"ok"}`

---

## ntfy.sh 通知

### 概要

ntfy.sh はオープンソースのプッシュ通知サービス。環境変数 `NTFY_TOPIC` を設定することで、コマンド完了時やエラー発生時にスマートフォンやブラウザへプッシュ通知を送信する。

### トピックの設定

#### 1. トピック名の決定

ntfy.sh のトピックは公開チャンネルであるため、推測されにくいユニークな名前を使用する。

```bash
# ランダムなトピック名を生成
echo "ghostrunner-$(openssl rand -hex 6)"
```

#### 2. 環境変数の設定

```bash
# backend/.env に追加
NTFY_TOPIC=ghostrunner-your-unique-id
```

#### 3. サーバーの再起動

```bash
make restart-backend-logs
```

#### 4. 有効化の確認

起動ログに以下が表示されることを確認する。

```
[NtfyService] Initialized with topic: https://ntfy.sh/ghostrunner-your-unique-id
```

### 通知の受信方法

#### スマートフォン

1. ntfy アプリをインストール
   - iOS: https://apps.apple.com/app/ntfy/id1625396347
   - Android: https://play.google.com/store/apps/details?id=io.heckel.ntfy
2. アプリ内で `NTFY_TOPIC` に設定したトピック名を購読

#### ブラウザ

`https://ntfy.sh/your-topic-name` にアクセスして購読する。

### 通知のタイミングと内容

| タイミング | タイトル | 優先度 | 説明 |
|-----------|---------|--------|------|
| コマンド正常完了 | Claude Code - Complete | default | 出力テキストの先頭100文字を通知本文に含む |
| コマンド実行エラー | Claude Code - Error | high | エラーメッセージを通知本文に含む |
| タイムアウト | Claude Code - Error | high | "Execution timeout" が通知される |
| パイプ生成失敗 | Claude Code - Error | high | エラー詳細が通知される |
| CLI起動失敗 | Claude Code - Error | high | エラー詳細が通知される |

### 通知の無効化

環境変数 `NTFY_TOPIC` を削除または空にしてサーバーを再起動する。

```bash
# backend/.env から NTFY_TOPIC の行を削除またはコメントアウト
# NTFY_TOPIC=ghostrunner-your-unique-id

make restart-backend
```

### 通知のテスト

ntfy.sh に直接 POST してトピックの疎通を確認できる。

```bash
curl -d "Test notification from Ghostrunner" \
  -H "Title: Test" \
  -H "Priority: default" \
  https://ntfy.sh/your-topic-name
```

---

## トラブルシューティング

### サーバーが起動しない

**症状**: `make backend` でエラーが発生する

**確認事項**:
1. Go 1.24 がインストールされているか: `go version`
2. ポート 8080 が他のプロセスで使用されていないか: `lsof -i :8080`
3. `.env` ファイルの構文が正しいか

**対処**:
```bash
# ポートを使用しているプロセスを停止
make stop-backend

# 再起動
make restart-backend-logs
```

### ntfy 通知が届かない

**症状**: コマンドを実行しても通知が届かない

**確認事項**:

1. 環境変数が設定されているか確認
   ```bash
   # サーバーのログを確認
   make logs-backend
   # 以下のログが出力されていれば有効
   # [NtfyService] Initialized with topic: https://ntfy.sh/your-topic
   ```

2. ntfy アプリで正しいトピックを購読しているか確認
   - アプリのトピック名と `NTFY_TOPIC` の値が一致していることを確認

3. 通知送信のログを確認
   ```bash
   # ログで以下を検索
   # [NtfyService] Sending notification: title=..., priority=...
   # [NtfyService] Notification sent successfully: title=...
   ```

4. ネットワーク接続を確認
   ```bash
   curl -s https://ntfy.sh/health
   ```

**対処**:
- トピック名に typo がないか確認
- ntfy アプリでトピックを再購読
- サーバーを再起動して環境変数を再読み込み

### ntfy 通知が送信失敗する

**症状**: ログに `[NtfyService] Failed to send notification` が表示される

**確認事項**:
1. ntfy.sh サーバーが稼働しているか: `curl -s https://ntfy.sh/health`
2. ネットワーク接続に問題がないか
3. HTTPタイムアウト（10秒）に達していないか

**対処**:
- ntfy.sh のステータスを確認: https://ntfy.sh
- 通知送信はfire-and-forget方式のため、送信失敗がコマンド実行には影響しない
- 一時的なネットワーク問題の場合は自然に回復する

### プロジェクト生成が途中で失敗する

**症状**: `/api/projects/create/stream` のSSEで `error` イベントが送信される

**確認事項**:

1. 失敗したステップIDをSSEイベントの `step` フィールドで特定する
2. サーバーログで `[CreateService]` または `[TemplateService]` のエラーを確認する
   ```bash
   make logs-backend
   # [CreateService] Step failed: step=dependency_install, error=...
   ```

**ステップ別の対処**:

| ステップ | よくある原因 | 対処 |
|---------|------------|------|
| `template_copy` | テンプレートディレクトリが見つからない | Ghostrunnerリポジトリの `templates/` ディレクトリの存在を確認 |
| `placeholder_replace` | ファイルの読み書き権限不足 | 生成先ディレクトリのパーミッションを確認 |
| `env_create` | `.env.example` がbaseテンプレートに存在しない | `templates/base/backend/.env.example` の存在を確認 |
| `dependency_install` | `go` や `npm` がPATHにない | `go version` と `npm --version` で確認 |
| `claude_assets` | `.claude/` ディレクトリが見つからない | Ghostrunnerリポジトリの `.claude/` ディレクトリの存在を確認 |
| `claude_md` | 書き込み権限不足 | 生成先の `.claude/` ディレクトリのパーミッションを確認 |
| `devtools_link` | シンボリックリンク作成権限がない、または同名ファイルが存在 | 生成先の `.devtools` の存在を確認 |
| `git_init` | `git` がPATHにない | `git --version` で確認 |
| `server_start` | ポート8080が使用中 | `lsof -i :8080` で確認し、プロセスを停止 |
| `health_check` | バックエンドの起動に時間がかかっている | 10回（約20秒）のリトライ後にタイムアウト。ログで起動エラーを確認 |

**生成途中のディレクトリの削除**:

エラーで中断した場合、生成途中のディレクトリが残る。手動で削除する。

```bash
# 生成先はホームディレクトリ直下
rm -rf ~/my-project
```

### プロジェクト名のバリデーションエラー

**症状**: `/api/projects/validate` で `valid: false` が返る

**原因と対処**:

| エラーメッセージ | 原因 | 対処 |
|----------------|------|------|
| プロジェクト名を入力してください | 名前が空 | 名前を入力する |
| プロジェクト名は小文字英数字とハイフンのみ使用できます | 大文字・アンダースコア・特殊文字を含んでいる | 小文字英数字とハイフンのみ使用する（例: `my-project`） |
| 同名のディレクトリが既に存在します | 生成先に同名ディレクトリがある | 別の名前を使用するか、既存ディレクトリを削除する |

---

### VS Codeでプロジェクトが開かない

**症状**: `/api/projects/open` が500エラーを返す

**確認事項**:
1. `code` コマンドがPATHに存在するか: `which code`
2. VS Code がインストールされているか

**対処**:
- VS Code のコマンドパレット（Cmd+Shift+P）から "Shell Command: Install 'code' command in PATH" を実行

---

---

## 巡回機能（Patrol）

### 概要

複数のGhostrunnerプロジェクトを自動巡回し、`開発/実装/実装待ち/` に未処理タスクがあれば `claude -p /coding` を最大5並列で実行する機能。プロジェクト一覧はJSONファイル（`devtools/backend/patrol_projects.json`）に永続化される。

### プロジェクトの登録・解除

```bash
# プロジェクトを巡回対象に登録
curl -X POST http://localhost:8888/api/patrol/projects \
  -H "Content-Type: application/json" \
  -d '{"path": "/Users/user/my-project"}'

# 登録済みプロジェクト一覧を確認
curl http://localhost:8888/api/patrol/projects

# プロジェクトを巡回対象から解除
curl -X POST http://localhost:8888/api/patrol/projects/remove \
  -H "Content-Type: application/json" \
  -d '{"path": "/Users/user/my-project"}'
```

### 手動で巡回を実行

```bash
# 全プロジェクトをスキャン（巡回は開始しない）
curl http://localhost:8888/api/patrol/scan

# 巡回を開始（未処理タスクのあるプロジェクトを自動実行）
curl -X POST http://localhost:8888/api/patrol/start

# 巡回を停止
curl -X POST http://localhost:8888/api/patrol/stop

# 全プロジェクトの実行状態を確認
curl http://localhost:8888/api/patrol/states
```

### 定期ポーリング

```bash
# 5分間隔の定期ポーリングを開始
curl -X POST http://localhost:8888/api/patrol/polling/start

# 定期ポーリングを停止
curl -X POST http://localhost:8888/api/patrol/polling/stop
```

### 承認待ちプロジェクトへの回答

承認待ち（waiting_approval）状態のプロジェクトが発生すると、ntfy通知が送信される。ダッシュボードまたはAPIから回答を送信して実行を再開する。

```bash
# 承認待ちプロジェクトに回答を送信
curl -X POST http://localhost:8888/api/patrol/resume \
  -H "Content-Type: application/json" \
  -d '{"projectPath": "/Users/user/my-project", "answer": "yes"}'
```

### SSEストリーミングの監視

```bash
# 巡回イベントをリアルタイムで監視
curl -N http://localhost:8888/api/patrol/stream
```

イベントタイプ: `project_started`, `project_question`, `project_completed`, `project_error`, `scan_completed`

### 設定ファイル

巡回対象プロジェクトは `devtools/backend/patrol_projects.json` に保存される。

```json
{
  "projects": [
    {
      "path": "/Users/user/project-a",
      "name": "project-a"
    },
    {
      "path": "/Users/user/project-b",
      "name": "project-b"
    }
  ]
}
```

手動で編集する場合はサーバーの再起動が必要。

---

## トラブルシューティング（巡回機能）

### 巡回が開始できない

**症状**: `/api/patrol/start` が409を返す

**原因**: 巡回が既に実行中

**対処**:
```bash
# 実行中の巡回を停止してから再開始
curl -X POST http://localhost:8888/api/patrol/stop
curl -X POST http://localhost:8888/api/patrol/start
```

### 承認待ちプロジェクトの再開に失敗する

**症状**: `/api/patrol/resume` が400を返す

**確認事項**:
1. プロジェクトの状態が `waiting_approval` であることを確認
   ```bash
   curl http://localhost:8888/api/patrol/states
   ```
2. `projectPath` が正確であることを確認（登録時のパスと一致する必要がある）
3. `answer` が空でないことを確認

### 巡回でプロジェクトがスキップされる

**症状**: 未処理タスクがあるのに実行されない

**確認事項**:
1. スキャン結果でタスクが検出されているか確認
   ```bash
   curl http://localhost:8888/api/patrol/scan
   ```
2. プロジェクトの状態が `running` または `waiting_approval` でないか確認（これらの状態はスキップ対象）
3. `開発/実装/実装待ち/` ディレクトリが存在するか確認
4. タスクファイルが隠しファイル（`.` で始まるファイル）でないか確認

### 巡回の通知が届かない

**確認事項**:
1. `NTFY_TOPIC` 環境変数が設定されているか
2. サーバーログで `[PatrolService]` のログを確認
   ```bash
   make logs-backend
   ```

---

### コマンドがタイムアウトする

**症状**: 60分後にタイムアウトエラーが返る

**確認事項**:
1. Claude CLI が正常に動作しているか: `claude --version`
2. 対象プロジェクトのパスが正しいか

**対処**:
- タイムアウト値はClaudeServiceの `timeout` フィールドで設定されている（60分）
- 長時間実行が必要な場合はタスクを分割して実行する

### ポートが既に使用されている

**症状**: `bind: address already in use` エラー

**対処**:
```bash
# バックエンドのプロセスを強制停止
make stop-backend

# 確認
lsof -i :8080

# それでも解決しない場合
lsof -ti:8080 | xargs kill -9
```
