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

## VOICEVOX エンジン設定

### インストール

1. VOICEVOX 公式サイト（https://voicevox.hiroshiba.jp/）から macOS 用 .dmg をダウンロード
2. .dmg を開き、VOICEVOX.app を `/Applications` にドラッグ＆ドロップ
3. VOICEVOX.app を起動（初回はモデルデータのダウンロードが走る。完了まで待つ）

### ヘルスチェック

```bash
# バージョン確認（正常時: バージョン文字列が返る）
curl http://localhost:50021/version

# Swagger UI で全エンドポイントを確認
open http://localhost:50021/docs
```

### スピーカー一覧の取得

```bash
# スピーカー名とIDの一覧を取得
curl -s http://localhost:50021/speakers | jq '.[].styles[] | {name: .name, id: .id}'
```

出力例:

```json
{ "name": "ノーマル", "id": 0 }
{ "name": "あまあま", "id": 1 }
```

`id` の値を `VOICEVOX_SPEAKER_ID` に設定する。

### 環境変数の設定

```bash
# backend/.env に追加
VOICEVOX_HOST=http://localhost:50021
VOICEVOX_SPEAKER_ID=0
```

| 環境変数 | デフォルト | 説明 |
|----------|-----------|------|
| `VOICEVOX_HOST` | `http://localhost:50021` | VOICEVOX エンジンのベースURL |
| `VOICEVOX_SPEAKER_ID` | `0` | 使用するスピーカーのID（スピーカー一覧から選択） |

設定後、サーバーを再起動する。

```bash
make restart-backend-logs
```

### クレジット表記

TTS音声を使用する画面には `VOICEVOX:<スピーカー名>` のクレジット表記を掲載する。個人利用の範囲では厳密に必須ではないが、配布・公開する場合は必須。

### 配布時の注意事項

- スピーカーごとに利用規約が異なる。配布前に使用するスピーカーの個別ライセンスを確認すること
- 立ち絵画像は使用しないため、立ち絵に関するライセンスは対象外
- 音声データの再配布条件はスピーカーごとに異なる。再配布が必要な場合は個別に確認すること

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

## プロジェクト登録

統括機能（把握・一括実装）の対象プロジェクトは `devtools/backend/patrol_projects.json` で管理する。

### ファイルの場所と性質

- パス: `devtools/backend/patrol_projects.json`
- `.gitignore` 対象（ローカル専用、リモートには共有されない）
- 絶対パスを含むため、各マシンごとに設定する

### 新しいプロジェクトを追加する

`projects` 配列にエントリを追加する:

```json
{
  "projects": [
    {
      "path": "/Users/user/既存プロジェクト",
      "name": "既存プロジェクト"
    },
    {
      "path": "/Users/user/新しいプロジェクト",
      "name": "新しいプロジェクト"
    }
  ]
}
```

| フィールド | 説明 |
|-----------|------|
| `path` | プロジェクトの絶対パス。`開発/実装/実装待ち/` 等のカンバンフォルダがあること |
| `name` | 表示名（通常はディレクトリ名と同じ） |

### 追加の前提条件

登録するプロジェクトには以下のフォルダ構造が必要:

```
プロジェクト/
|-- 開発/
|   |-- 実装/
|   |   |-- 実装待ち/    # 計画書（*_plan.md）を置く
|   |   |-- 実行中/      # gr-run が着手時に移動
|   |   |-- 完了/        # /coding が完了時に移動
```

この構造は Ghostrunner の `/init` でプロジェクトを作成すれば自動的に生成される。

### プロジェクトを削除する

`projects` 配列から該当エントリを削除する。ロックファイル（`~/.ghostrunner/locks/`）は自動では消えないが、放置しても問題ない。

### 登録済みプロジェクトの確認

```bash
cat devtools/backend/patrol_projects.json
```

または Ghostrunner ターミナルで「状況は？」と聞けば chief-director が全登録プロジェクトの状態を報告する。

---

## ダッシュボード（Dashboard API）

### 概要

統括GUIダッシュボードは、patrol_projects.json に登録された全プロジェクトのカンバン状態、未回答確認事項、運用状態を横断的に集約する。巡回機能（Patrol）と設定ファイルを共有する。

### 状態の取得

```bash
# 全プロジェクトの集約状態を取得
curl http://localhost:8888/api/dashboard/state
```

レスポンスにはプロジェクトごとのカンバン件数、未回答確認事項、運用エントリが含まれる。プロジェクトは注目度（required > progress > watching）順でソートされる。

### 確認事項への回答

```bash
# 未回答確認事項に回答を書き戻す
curl -X POST http://localhost:8888/api/dashboard/answer \
  -H "Content-Type: application/json" \
  -d '{
    "projectPath": "/Users/user/my-project",
    "planPath": "開発/実装/実行中/feature_plan.md",
    "lineStart": 42,
    "answer": "A案で進めてください"
  }'
```

回答を書き戻すと、計画書の対象行が `**ステータス**: 未回答` から `**ステータス**: 回答済` に更新され、直下に `**回答**: A案で進めてください` が挿入される。

### トラブルシューティング

#### 状態取得で空の配列が返る

**確認事項**:
1. `patrol_projects.json` にプロジェクトが登録されているか
   ```bash
   cat devtools/backend/patrol_projects.json
   ```
2. 登録されたプロジェクトのパスが実在するディレクトリか

#### 回答書き戻しが409を返す

**原因**: 対象行が既に回答済みか、行のずれにより前後2行以内に未回答行が見つからない

**対処**: `GET /api/dashboard/state` で最新の `lineStart` を確認してから再送信する

#### 回答書き戻しが400を返す

**原因**: バリデーションエラー（未登録プロジェクト、不正パス、空回答等）

**確認事項**:
1. `projectPath` がpatrol_projects.jsonに登録されているか
2. `planPath` が `開発/実装/実装待ち/` または `開発/実装/実行中/` 配下の.mdファイルか
3. `answer` が空でないか

---

## gr-run（一括実装CLI）

### 概要

gr-run はタスクファイルを1つ受け取り、Claude CLI で `/coding` を実行するワンショットCLI。複数インスタンスを並列起動することで一括実装を実現する。プロジェクト単位の排他ロック（flock）により同一プロジェクトへの多重実行を防止する。

### 手動実行

```bash
# 基本的な実行
gr-run --project /Users/user/my-project --task "001-feature.md"

# ロックディレクトリを明示的に指定
gr-run --project /Users/user/my-project --task "001-feature.md" --locks-dir /tmp/gr-locks
```

終了コード: 異常終了（OutcomeAbnormal）の場合は `1`、それ以外は `0` を返す。

### ロックファイルの管理

ロックファイルは `~/.ghostrunner/locks/` に格納される。ファイル名は `<プロジェクト名>-<SHA256先頭12文字>.lock` の形式。

```bash
# ロックファイルの一覧を確認
ls -la ~/.ghostrunner/locks/

# 特定のプロジェクトのロックファイルを確認
ls -la ~/.ghostrunner/locks/my-project-*.lock

# ロックを保持しているプロセスを確認
fuser ~/.ghostrunner/locks/my-project-*.lock 2>/dev/null
# または
lsof ~/.ghostrunner/locks/my-project-*.lock
```

flock は保持プロセスの終了時に自動解放されるため、通常はロックファイルの手動削除は不要。プロセスが正常にもクラッシュでも終了すればロックは解放される。

### ロックファイルのクリーンアップ

ロックファイル自体（空ファイル）はプロセス終了後も残る。ディスク容量への影響は無視できるが、定期的にクリーンアップしたい場合は以下を実行する。

```bash
# ロックを保持しているプロセスがないことを確認してから削除
rm ~/.ghostrunner/locks/*.lock
```

**注意**: 実行中の gr-run プロセスがある状態でロックファイルを削除すると、別のプロセスが同一プロジェクトを多重実行する恐れがある。必ず全プロセスが終了していることを確認すること。

### 結果分類（Outcome）

| Outcome | 意味 | 終了コード |
|---------|------|-----------|
| `completed` | タスク正常完了（完了ディレクトリへ移動済み） | 0 |
| `waiting_answer` | 確認事項が未回答 | 0 |
| `abnormal` | 異常終了（Claude起動失敗、タスク移動失敗等） | 1 |
| `needs_check` | 完了ディレクトリ未移動（人手確認が必要） | 0 |
| `lock_busy` | 他プロセスが実行中 | 0 |

---

## トラブルシューティング（gr-run）

### 同一プロジェクトで lock_busy になる

**症状**: gr-run が `lock_busy` を返し、タスクが実行されない

**確認事項**:
1. 該当プロジェクトの gr-run プロセスが実行中でないか確認
   ```bash
   ps aux | grep "gr-run.*my-project"
   ```
2. ロックファイルを保持しているプロセスを確認
   ```bash
   lsof ~/.ghostrunner/locks/my-project-*.lock
   ```

**対処**:
- 実行中のプロセスがあれば完了を待つ
- プロセスが存在しないのにロックが残る場合（通常は発生しない）、ロックファイルを削除して再実行

### タスクの移動に失敗する

**症状**: `タスクの移動に失敗` というエラーが出る

**確認事項**:
1. タスクファイルが `開発/実装/実装待ち/` に存在するか
   ```bash
   ls "開発/実装/実装待ち/"
   ```
2. `開発/実装/実行中/` に同名ファイルが存在しないか
3. ファイルシステムの権限

### needs_check になる

**症状**: Claude は正常終了（exitCode=0）したが、タスクファイルが完了ディレクトリに移動されていない

**原因**: /coding スキルがタスク完了後のファイル移動を行わなかった可能性がある（フォーマット不一致等）

**対処**:
1. タスクの実装内容を確認し、問題なければ手動で完了ディレクトリに移動
   ```bash
   mv "開発/実装/実行中/001-feature.md" "開発/実装/完了/"
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
