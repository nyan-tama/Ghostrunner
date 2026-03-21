# Backend API 仕様書

Ghostrunner API サーバーのエンドポイント仕様。

## 概要

Claude CLI のスラッシュコマンドをHTTP API経由で実行するためのサーバー。
Server-Sent Events (SSE) によるストリーミング出力とセッション継続をサポートする。

## モジュール構成

- **モジュール名**: `ghostrunner/backend`
- **エントリーポイント**: `backend/cmd/server/main.go`
- **ビルド**: `cd backend && go build -o server ./cmd/server`
- **実行**: `cd backend && go run ./cmd/server`

---

## 環境変数

| 環境変数 | 必須 | 説明 |
|----------|------|------|
| `GEMINI_API_KEY` | No | Gemini API のAPIキー。未設定時はGemini関連エンドポイントが503を返す |
| `OPENAI_API_KEY` | No | OpenAI API のAPIキー（sk-xxx形式）。未設定時はOpenAI関連エンドポイントが503を返す |
| `NTFY_TOPIC` | No | ntfy.shのトピック名。設定するとコマンド完了・エラー時にプッシュ通知を送信する。未設定時は通知機能が無効になる |

---

## エンドポイント一覧

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/api/health` | GET | ヘルスチェック |
| `/api/command` | POST | コマンドの同期実行 |
| `/api/command/stream` | POST | コマンドのストリーミング実行 (SSE) |
| `/api/command/continue` | POST | セッション継続 |
| `/api/command/continue/stream` | POST | セッション継続のストリーミング実行 (SSE) |
| `/api/files` | GET | 開発フォルダ内のmdファイル一覧取得 |
| `/api/projects` | GET | プロジェクト候補のディレクトリ一覧取得 |
| `/api/projects/destroy` | POST | プロジェクトディレクトリの削除 |
| `/api/plan` | POST | /planコマンドの同期実行（後方互換性） |
| `/api/plan/stream` | POST | /planコマンドのストリーミング実行（後方互換性） |
| `/api/plan/continue` | POST | セッション継続（後方互換性） |
| `/api/plan/continue/stream` | POST | セッション継続のストリーミング実行（後方互換性） |
| `/api/projects/validate` | GET | プロジェクト名のバリデーション |
| `/api/projects/create/stream` | POST | SSEによるプロジェクト生成（10ステップ進捗配信） |
| `/api/projects/open` | POST | 生成されたプロジェクトをVS Codeで開く |
| `/api/gemini/token` | POST | Gemini Live API 用エフェメラルトークン発行 |
| `/api/openai/realtime/session` | POST | OpenAI Realtime API 用エフェメラルキー発行 |
| `/api/patrol/projects` | POST | 巡回対象プロジェクトを登録 |
| `/api/patrol/projects/remove` | POST | 巡回対象プロジェクトを解除 |
| `/api/patrol/projects` | GET | 登録済みプロジェクト一覧を取得 |
| `/api/patrol/scan` | GET | 全登録プロジェクトの状態をスキャン |
| `/api/patrol/start` | POST | 巡回を開始（未処理タスクのあるプロジェクトを最大5並列で実行） |
| `/api/patrol/stop` | POST | 実行中の巡回を停止 |
| `/api/patrol/resume` | POST | 承認待ちプロジェクトにユーザー回答を送信して再開 |
| `/api/patrol/states` | GET | 全プロジェクトの実行状態を取得 |
| `/api/patrol/stream` | GET | 巡回イベントのSSEストリーミング |
| `/api/patrol/polling/start` | POST | 5分間隔の定期ポーリングを開始 |
| `/api/patrol/polling/stop` | POST | 定期ポーリングを停止 |

---

## 許可コマンド

| コマンド | 説明 |
|----------|------|
| `plan` | 実装計画の作成 |
| `fullstack` | バックエンド + フロントエンドの実装 |
| `go` | Go バックエンドのみの実装 |
| `nextjs` | Next.js フロントエンドのみの実装 |
| `discuss` | アイデアや構想の対話形式での深掘り |
| `research` | 外部情報の調査・収集 |

---

## Health API

### GET /api/health

サーバーのヘルスチェックを実行する。

サーバープロセスが正常に動作しているかを確認するためのシンプルなエンドポイント。
外部サービス（データベース、外部API等）への依存はなく、サーバーが起動していれば常に成功を返す。
ロードバランサーやコンテナオーケストレーション（Cloud Run等）のヘルスチェックに使用する。

#### リクエスト

```
GET /api/health
```

パラメータなし。認証不要。

#### レスポンス（成功）

```json
{
    "status": "ok"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `status` | string | サーバーの状態（常に "ok"） |

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | サーバー正常稼働中 |

---

## Command API

### POST /api/command

コマンドを同期実行する。テキストと画像を組み合わせた指示に対応。

#### リクエスト

```json
{
    "project": "/path/to/project",
    "command": "fullstack",
    "args": "implement feature X",
    "images": [
        {
            "name": "screenshot.png",
            "data": "Base64エンコードされた画像データ",
            "mimeType": "image/png"
        }
    ]
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `project` | string | Yes | 対象プロジェクトの絶対パス |
| `command` | string | Yes | 実行するコマンド（plan, fullstack, go, nextjs, discuss, research） |
| `args` | string | Yes | コマンドの引数 |
| `images` | array | No | 画像データの配列（最大5枚） |

#### ImageData オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `name` | string | ファイル名 |
| `data` | string | Base64エンコードされた画像データ |
| `mimeType` | string | MIMEタイプ（image/jpeg, image/png, image/gif, image/webp） |

#### 画像の制約

- 最大枚数: 5枚
- 最大サイズ: 1枚あたり5MB
- 対応形式: JPEG, PNG, GIF, WebP

#### レスポンス（成功）

```json
{
    "success": true,
    "session_id": "abc123-def456",
    "output": "実行結果のテキスト",
    "questions": [],
    "completed": true,
    "cost_usd": 0.01
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `success` | boolean | 成功フラグ |
| `session_id` | string | セッションID（継続用） |
| `output` | string | 実行結果のテキスト |
| `questions` | array | 質問がある場合の配列 |
| `completed` | boolean | 実行が完了したかどうか |
| `cost_usd` | number | 実行コスト（USD） |

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "エラーメッセージ"
}
```

---

### POST /api/command/stream

コマンドをストリーミング実行する（Server-Sent Events）。

#### リクエスト

`POST /api/command` と同じ。

#### レスポンス

`Content-Type: text/event-stream` 形式で StreamEvent を送信する。

```
data: {"type":"init","session_id":"abc123","message":"Claude CLI started"}

data: {"type":"tool_use","tool_name":"Read","message":"Reading: .../path/to/file.go"}

data: {"type":"complete","session_id":"abc123","result":{...}}
```

#### StreamEvent タイプ

| タイプ | 説明 |
|--------|------|
| `init` | セッション開始 |
| `thinking` | 思考中 |
| `tool_use` | ツール使用（Read, Write, Edit, Bash等） |
| `text` | テキスト出力 |
| `question` | 質問（ユーザー入力待ち） |
| `complete` | 完了 |
| `error` | エラー |

---

### POST /api/command/continue

セッションを継続してユーザーの回答を送信する。

#### リクエスト

```json
{
    "project": "/path/to/project",
    "session_id": "abc123-def456",
    "answer": "yes"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `project` | string | Yes | 対象プロジェクトの絶対パス |
| `session_id` | string | Yes | 継続するセッションのID |
| `answer` | string | Yes | ユーザーの回答 |

#### レスポンス

`POST /api/command` と同じ形式。

---

### POST /api/command/continue/stream

セッション継続をストリーミング実行する（Server-Sent Events）。

#### リクエスト

`POST /api/command/continue` と同じ。

#### レスポンス

`POST /api/command/stream` と同じ形式。

---

## Files API

### GET /api/files

開発フォルダ内のmdファイル一覧を取得する。

プロジェクト内の `開発/` ディレクトリ配下にある以下のフォルダをスキャンし、
各フォルダ内の `.md` ファイル一覧を返却する。

スキャン対象フォルダ:
- `実装/実装待ち`
- `実装/完了`
- `検討中`
- `資料`
- `アーカイブ`

#### リクエスト

```
GET /api/files?project=/path/to/project
```

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `project` | query string | Yes | プロジェクトの絶対パス |

#### レスポンス（成功）

```json
{
    "success": true,
    "files": {
        "実装/実装待ち": [
            {
                "name": "2026-01-25_feature_plan.md",
                "path": "開発/実装/実装待ち/2026-01-25_feature_plan.md"
            }
        ],
        "実装/完了": [],
        "検討中": [],
        "資料": [],
        "アーカイブ": []
    }
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `success` | boolean | 成功フラグ |
| `files` | object | フォルダ別のファイル一覧（キー: フォルダ名、値: ファイル情報の配列） |

#### FileInfo オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `name` | string | ファイル名 |
| `path` | string | プロジェクトルートからの相対パス（`開発/` から始まる） |

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "エラーメッセージ"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |
| 400 | リクエスト不正（projectパラメータ未指定、パスが絶対パスでない等） |
| 404 | 開発ディレクトリが存在しない |
| 500 | サーバー内部エラー（フォルダ読み取り失敗等） |

---

## Projects API

### GET /api/projects

プロジェクト候補のディレクトリ一覧を取得する。

`/Users/user/` 直下のディレクトリをスキャンし、プロジェクト候補として返却する。
フロントエンドのProjectPath選択ドロップダウンの候補データを提供するためのエンドポイント。

フィルタリング条件:
- 隠しディレクトリ（`.` で始まるもの）を除外
- ファイルを除外
- シンボリックリンクを除外

結果はディレクトリ名のアルファベット順でソートされる。

#### リクエスト

```
GET /api/projects
```

パラメータなし。認証不要。

#### レスポンス（成功）

```json
{
    "success": true,
    "projects": [
        {
            "name": "ProjectA",
            "path": "/Users/user/ProjectA"
        },
        {
            "name": "ProjectB",
            "path": "/Users/user/ProjectB"
        }
    ]
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `success` | boolean | 成功フラグ |
| `projects` | array | プロジェクトディレクトリ情報の配列 |

#### ProjectInfo オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `name` | string | ディレクトリ名 |
| `path` | string | ディレクトリの絶対パス |

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "ディレクトリ一覧の取得に失敗しました"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |
| 500 | ディレクトリ読み取りエラー |

### POST /api/projects/destroy

プロジェクトディレクトリを削除する。

ホームディレクトリ直下のディレクトリのみ削除を許可する（パストラバーサル防止）。
対象ディレクトリに `docker-compose.yml` が存在する場合、削除前に `docker compose down -v` を実行してコンテナとボリュームを停止・削除する。
`docker compose down -v` の実行は30秒のタイムアウトが設定されており、失敗してもディレクトリ削除は続行する。

#### リクエスト

```
POST /api/projects/destroy
Content-Type: application/json
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `path` | string | Yes | 削除対象ディレクトリの絶対パス |

```json
{
    "path": "/Users/user/my-project"
}
```

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "ホームディレクトリ直下のプロジェクトのみ削除できます"
}
```

#### パス制限

- `filepath.Clean` でパスを正規化した上で、`filepath.Dir` がホームディレクトリと一致することを検証する
- ホームディレクトリ自体や、サブディレクトリ（例: `~/project/subdir`）の指定は拒否される

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 削除成功 |
| 400 | パス未指定、パストラバーサル検出、対象がディレクトリでない |
| 404 | 対象ディレクトリが存在しない |
| 500 | ディレクトリ削除失敗 |

---

## Create API（プロジェクト生成）

GUIからGhostrunnerテンプレートを使ったプロジェクト生成を行うAPI群。
プロジェクト名のバリデーション、テンプレートベースの生成（SSE進捗配信）、VS Code起動の3エンドポイントで構成される。

### GET /api/projects/validate

プロジェクト名のバリデーションを実行する。

命名規則の検証と、生成先ディレクトリの重複チェックを行う。フロントエンドの入力フォームでリアルタイムバリデーションに使用する。

#### リクエスト

```
GET /api/projects/validate?name=my-project
```

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `name` | query string | Yes | バリデーション対象のプロジェクト名 |

#### バリデーションルール

- 空でないこと
- 小文字英数字とハイフンのみ（正規表現: `^[a-z0-9]+(-[a-z0-9]+)*$`）
- 生成先ディレクトリに同名のディレクトリが存在しないこと

#### レスポンス（成功）

```json
{
    "valid": true,
    "path": "/Users/user/my-project"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `valid` | boolean | バリデーション成功かどうか |
| `path` | string | 生成先のディレクトリパス（validがtrueの場合のみ） |

#### レスポンス（バリデーションエラー）

```json
{
    "valid": false,
    "error": "プロジェクト名は小文字英数字とハイフンのみ使用できます（例: my-project）"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `valid` | boolean | 常にfalse |
| `error` | string | エラーメッセージ |
| `path` | string | ディレクトリパス（重複エラーの場合のみ） |

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | バリデーション結果を返却（valid/invalidに関わらず200） |

---

### POST /api/projects/create/stream

プロジェクトをテンプレートから生成する（Server-Sent Events）。

10ステップの処理を順次実行し、各ステップの進捗をSSEイベントとして配信する。クライアントが切断した場合は処理を中断する。

#### リクエスト

```json
{
    "name": "my-project",
    "description": "プロジェクトの概要説明",
    "services": ["database", "cache"]
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `name` | string | Yes | プロジェクト名（バリデーション済みであること） |
| `description` | string | No | プロジェクト概要（CLAUDE.mdに記載される） |
| `services` | array | No | 追加サービスの配列 |

#### 選択可能なサービス

| サービス名 | 説明 | テンプレート |
|-----------|------|------------|
| `database` | PostgreSQL 16 + GORM | `templates/with-db` |
| `storage` | MinIO（S3互換オブジェクトストレージ） | `templates/with-storage` |
| `cache` | Redis 7 | `templates/with-redis` |

#### レスポンス

`Content-Type: text/event-stream` 形式で CreateEvent を送信する。

```
data: {"type":"progress","step":"template_copy","message":"テンプレートをコピー中...","progress":10}

data: {"type":"progress","step":"placeholder_replace","message":"プロジェクト名を設定中...","progress":20}

data: {"type":"progress","step":"env_create","message":"環境設定ファイルを作成中...","progress":30}

data: {"type":"progress","step":"dependency_install","message":"依存パッケージをインストール中...","progress":40}

data: {"type":"progress","step":"claude_assets","message":"開発支援ツールを設定中...","progress":50}

data: {"type":"progress","step":"claude_md","message":"プロジェクト設定を生成中...","progress":60}

data: {"type":"progress","step":"devtools_link","message":"devtools を接続中...","progress":70}

data: {"type":"progress","step":"git_init","message":"バージョン管理を初期化中...","progress":80}

data: {"type":"progress","step":"server_start","message":"サーバーを起動中...","progress":90}

data: {"type":"progress","step":"health_check","message":"動作確認中...","progress":95}

data: {"type":"complete","step":"done","message":"プロジェクトの作成が完了しました","progress":100,"path":"/Users/user/my-project"}
```

#### CreateEvent オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `type` | string | イベントタイプ: "progress", "complete", "error" |
| `step` | string | ステップID |
| `message` | string | 表示用メッセージ |
| `progress` | number | 進捗率（0-100） |
| `path` | string | 生成されたプロジェクトパス（completeのみ） |
| `error` | string | エラーメッセージ（errorのみ） |

#### 生成ステップ一覧

| ステップID | 進捗率 | 内容 |
|-----------|--------|------|
| `template_copy` | 10% | baseテンプレート + 選択サービステンプレートのコピー、docker-compose.ymlマージ |
| `placeholder_replace` | 20% | `{{PROJECT_NAME}}` プレースホルダーの一括置換 |
| `env_create` | 30% | `.env.example` を基にした `.env` ファイル生成 |
| `dependency_install` | 40% | `go mod tidy` + `npm install` |
| `claude_assets` | 50% | `.claude/` ディレクトリのコピー、不要エージェントの削除 |
| `claude_md` | 60% | プロジェクト用 `CLAUDE.md` の生成 |
| `devtools_link` | 70% | devtools フロントエンドへのシンボリックリンク作成 |
| `git_init` | 80% | `git init` + `git add -A` + 初回コミット |
| `server_start` | 90% | `make start-backend` でバックエンドをバックグラウンド起動 |
| `health_check` | 95% | `http://localhost:8080/api/health` へのポーリング（最大10回、2秒間隔） |

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | SSEストリーム開始（個別のエラーはイベント内で通知） |
| 400 | リクエスト不正、プロジェクト名バリデーションエラー、不明なサービス名 |

---

### POST /api/projects/open

生成されたプロジェクトをVS Codeで開く。

`code` コマンドを使用してVS Codeを起動する。パストラバーサル防止のため、ホームディレクトリ配下のパスのみ許可する。

#### リクエスト

```json
{
    "path": "/Users/user/my-project"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `path` | string | Yes | プロジェクトの絶対パス |

#### パスの制約

- 空でないこと
- ホームディレクトリ配下であること（パストラバーサル防止）
- 指定パスが存在すること

#### レスポンス（成功）

```json
{
    "success": true,
    "message": "VS Codeでプロジェクトを開きました"
}
```

#### レスポンス（エラー）

```json
{
    "error": "エラーメッセージ"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | VS Code起動成功 |
| 400 | リクエスト不正、pathが空、許可されていないパス |
| 404 | 指定パスが存在しない |
| 500 | ホームディレクトリ取得失敗、VS Code起動失敗 |

---

## Plan API（後方互換性）

`/api/plan` エンドポイントは `/api/command` の登場以前から存在する。
内部的に `command: "plan"` として `/api/command` と同じサービスを使用する。

### POST /api/plan

```json
{
    "project": "/path/to/project",
    "args": "implement feature X"
}
```

`/api/command` で `command: "plan"` を指定した場合と同等。

### POST /api/plan/stream

`/api/command/stream` で `command: "plan"` を指定した場合と同等。

### POST /api/plan/continue

`/api/command/continue` と同じ。

### POST /api/plan/continue/stream

`/api/command/continue/stream` と同じ。

---

## Gemini API

### POST /api/gemini/token

Gemini Live API 用のエフェメラルトークンを発行する。

フロントエンドがGemini Live APIへWebSocket接続する際に使用する一時的なトークンを生成する。
トークンには有効期限が設定され、期限切れ後は再発行が必要となる。

#### 必要な環境変数

| 環境変数 | 説明 |
|----------|------|
| `GEMINI_API_KEY` | Gemini API のAPIキー |

#### リクエスト

```json
{
    "expireSeconds": 3600
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `expireSeconds` | number | No | トークンの有効期間（秒）。デフォルト: 3600、範囲: 60-86400 |

#### レスポンス（成功）

```json
{
    "success": true,
    "token": "ephemeral-token-string",
    "expireTime": "2026-01-26T12:00:00Z"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `success` | boolean | 成功フラグ |
| `token` | string | エフェメラルトークン |
| `expireTime` | string | トークンの有効期限（RFC3339形式） |

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "エラーメッセージ"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |
| 400 | リクエスト不正（expireSecondsが範囲外等） |
| 500 | トークン発行に失敗（Gemini API エラー） |
| 503 | GEMINI_API_KEY 未設定 |

---

## OpenAI Realtime API

### POST /api/openai/realtime/session

OpenAI Realtime API 用のエフェメラルキーを発行する。

フロントエンドがOpenAI Realtime APIへWebSocket接続する際に使用する一時的なトークン（ek_xxx形式）を生成する。

#### 必要な環境変数

| 環境変数 | 説明 |
|----------|------|
| `OPENAI_API_KEY` | OpenAI API のAPIキー（sk-xxx形式） |

#### リクエスト

```json
{
    "model": "gpt-4o-realtime-preview-2024-12-17",
    "voice": "verse"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `model` | string | No | 使用するモデル。デフォルト: `gpt-4o-realtime-preview-2024-12-17` |
| `voice` | string | No | 音声タイプ。デフォルト: `verse` |

#### レスポンス（成功）

```json
{
    "success": true,
    "token": "ek_xxx...",
    "expireTime": "2026-01-27T12:00:00Z"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `success` | boolean | 成功フラグ |
| `token` | string | エフェメラルキー（ek_xxx形式） |
| `expireTime` | string | トークンの有効期限（RFC3339形式） |

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "エラーメッセージ"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |
| 400 | リクエスト不正 |
| 500 | セッション作成に失敗（OpenAI API エラー） |
| 503 | OPENAI_API_KEY 未設定 |

---

## HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |
| 400 | リクエスト不正、バリデーションエラー、許可されていないコマンド |
| 500 | Claude CLI実行エラー |

---

## バリデーション

### プロジェクトパス

- 空でないこと
- 絶対パスであること（`/` で始まる）
- 存在するディレクトリであること

### コマンド（/api/command のみ）

- 空でないこと
- 許可コマンドリスト（plan, fullstack, go, nextjs, discuss, research）に含まれること

### 引数

- 空でないこと

### 画像（/api/command、/api/command/stream のみ）

- 枚数が5枚以下であること
- MIMEタイプがimage/jpeg, image/png, image/gif, image/webpのいずれかであること
- Base64デコードが可能であること
- デコード後のサイズが5MB以下であること

---

## Question オブジェクト

質問がある場合、`questions` 配列に以下の形式で格納される。

```json
{
    "question": "質問文",
    "header": "ヘッダー",
    "options": [
        {
            "label": "選択肢1",
            "description": "説明"
        }
    ],
    "multiSelect": false
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `question` | string | 質問文 |
| `header` | string | ヘッダーテキスト |
| `options` | array | 選択肢の配列 |
| `multiSelect` | boolean | 複数選択可能かどうか |

---

## ntfy 通知

コマンド実行の完了・エラー時に [ntfy.sh](https://ntfy.sh) を使ってプッシュ通知を送信する機能。
環境変数 `NTFY_TOPIC` が設定されている場合のみ有効になる。

### 通知の仕組み

- ntfy.sh の公開サーバー (`https://ntfy.sh/{NTFY_TOPIC}`) にHTTP POSTで通知を送信
- 通知はfire-and-forget方式で非同期送信されるため、API レスポンスの遅延には影響しない
- 通知送信の失敗はログに記録されるが、コマンド実行結果のエラーにはならない

### 通知タイミング

| タイミング | 通知タイプ | 優先度 |
|-----------|----------|--------|
| コマンド正常完了 | Notify | default |
| コマンド実行エラー | NotifyError | high |
| タイムアウト | NotifyError | high |
| 巡回: 承認待ち発生 | Notify | default |

### 受信方法

ntfy.sh のモバイルアプリやブラウザで同じトピック名を購読することで通知を受信できる。

---

## Patrol API（複数プロジェクト自動巡回）

複数のGhostrunnerプロジェクトを自動巡回し、未処理タスク（`開発/実装/実装待ち/` 配下のファイル）を検出して `claude -p /coding` を最大5並列で実行する機能。

### 概要

- 登録されたプロジェクトをスキャンし、`開発/実装/実装待ち/` ディレクトリに未処理タスクファイルがあれば自動実行する
- 最大5プロジェクトを並列実行（セマフォによる制御）
- 手動実行と5分間隔の定期ポーリングに対応
- 承認待ち（設計判断等）が発生した場合、ntfy通知を送信しダッシュボードから回答可能
- プロジェクト一覧はJSONファイル（`devtools/backend/patrol_projects.json`）に永続化

### プロジェクト状態

| 状態 | 説明 |
|------|------|
| `idle` | 待機中 |
| `running` | Claude CLI 実行中 |
| `waiting_approval` | ユーザーの承認待ち（設計判断等の質問） |
| `queued` | 実行待ちキュー |
| `completed` | 実行完了 |
| `error` | エラー発生 |

### POST /api/patrol/projects

巡回対象プロジェクトを登録する。

#### リクエスト

```json
{
    "path": "/Users/user/my-project"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `path` | string | Yes | プロジェクトの絶対パス |

#### バリデーション

- `path` が空でないこと
- 絶対パスであること
- 指定パスがディレクトリとして存在すること
- 同一パスが未登録であること

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "project already registered: /Users/user/my-project"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 登録成功 |
| 400 | バリデーションエラー（パス不正、重複登録等） |

---

### POST /api/patrol/projects/remove

巡回対象プロジェクトを解除する。

#### リクエスト

```json
{
    "path": "/Users/user/my-project"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `path` | string | Yes | 解除するプロジェクトの絶対パス |

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "project not registered: /Users/user/my-project"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 解除成功 |
| 400 | 未登録のプロジェクト |

---

### GET /api/patrol/projects

登録済みプロジェクト一覧を取得する。パスのアルファベット順でソートされる。

#### リクエスト

```
GET /api/patrol/projects
```

パラメータなし。

#### レスポンス（成功）

```json
{
    "success": true,
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

#### PatrolProject オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `path` | string | プロジェクトの絶対パス |
| `name` | string | プロジェクト名（ディレクトリ名） |

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |

---

### GET /api/patrol/scan

全登録プロジェクトをスキャンし、git logと未処理タスクを取得する。巡回は開始しない。

#### リクエスト

```
GET /api/patrol/scan
```

パラメータなし。

#### レスポンス（成功）

```json
{
    "success": true,
    "results": [
        {
            "project": {
                "path": "/Users/user/my-project",
                "name": "my-project"
            },
            "gitLog": "abc1234 feat: 機能Aを追加\ndef5678 fix: バグ修正",
            "pendingTasks": ["2026-03-20_feature_b.md"]
        }
    ]
}
```

#### ScanResult オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `project` | PatrolProject | プロジェクト情報 |
| `gitLog` | string | `git log --oneline -5` の出力 |
| `pendingTasks` | array | `開発/実装/実装待ち/` 配下のファイル名一覧（隠しファイル・ディレクトリ除外） |

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |

---

### POST /api/patrol/start

巡回を開始する。全登録プロジェクトをスキャンし、未処理タスクのあるプロジェクトを最大5並列で実行する。

既に実行中（running）または承認待ち（waiting_approval）のプロジェクトはスキップする。巡回が既に実行中の場合は409を返す。

#### リクエスト

```
POST /api/patrol/start
```

ボディなし。

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "patrol is already running"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 巡回開始 |
| 409 | 巡回が既に実行中 |

---

### POST /api/patrol/stop

実行中の巡回を停止する。キューで待機中のプロジェクトはキャンセルされる。実行中のClaude CLIプロセスには影響しない。

#### リクエスト

```
POST /api/patrol/stop
```

ボディなし。

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 停止完了 |

---

### POST /api/patrol/resume

承認待ち（waiting_approval）状態のプロジェクトにユーザーの回答を送信して実行を再開する。

#### リクエスト

```json
{
    "projectPath": "/Users/user/my-project",
    "answer": "yes"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `projectPath` | string | Yes | 対象プロジェクトのパス |
| `answer` | string | Yes | ユーザーの回答テキスト |

#### バリデーション

- `projectPath` が空でないこと
- `answer` が空でないこと
- 対象プロジェクトが承認待ち（waiting_approval）状態であること
- セッションIDが存在すること

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### レスポンス（エラー）

```json
{
    "success": false,
    "error": "project is not waiting for approval: /Users/user/my-project (status=running)"
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 再開成功 |
| 400 | バリデーションエラー（パス未指定、回答未指定、承認待ちでない等） |

---

### GET /api/patrol/states

全プロジェクトの実行状態を取得する。

#### リクエスト

```
GET /api/patrol/states
```

パラメータなし。

#### レスポンス（成功）

```json
{
    "success": true,
    "states": {
        "/Users/user/project-a": {
            "project": {
                "path": "/Users/user/project-a",
                "name": "project-a"
            },
            "status": "running",
            "sessionId": "session-xxx",
            "startedAt": "2026-03-20T10:00:00Z",
            "updatedAt": "2026-03-20T10:05:00Z"
        },
        "/Users/user/project-b": {
            "project": {
                "path": "/Users/user/project-b",
                "name": "project-b"
            },
            "status": "waiting_approval",
            "sessionId": "session-yyy",
            "question": {
                "question": "この設計で進めてよいですか？",
                "header": "",
                "options": [],
                "multiSelect": false
            },
            "startedAt": "2026-03-20T10:00:00Z",
            "updatedAt": "2026-03-20T10:03:00Z"
        }
    }
}
```

#### ProjectState オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `project` | PatrolProject | プロジェクト情報 |
| `status` | string | 現在の状態（idle, running, waiting_approval, queued, completed, error） |
| `sessionId` | string | Claude CLIのセッションID（実行中・承認待ち時） |
| `question` | Question | 承認待ちの質問内容（waiting_approval時のみ） |
| `gitLog` | string | 直近のgit log |
| `pendingTasks` | array | 未処理タスクのファイル名一覧 |
| `error` | string | エラーメッセージ（error時のみ） |
| `startedAt` | string | 実行開始時刻（RFC3339形式） |
| `updatedAt` | string | 最終更新時刻（RFC3339形式） |

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | 正常完了 |

---

### GET /api/patrol/stream

巡回イベントをServer-Sent Events形式でストリーミング配信する。

30秒間隔でkeepaliveコメントを送信する。クライアントが切断した場合はサブスクリプションを解除する。

#### リクエスト

```
GET /api/patrol/stream
```

パラメータなし。

#### レスポンス

`Content-Type: text/event-stream` 形式で PatrolEvent を送信する。

```
data: {"type":"scan_completed","message":"Scan completed: 3 projects"}

data: {"type":"project_started","projectPath":"/Users/user/project-a","state":{...}}

data: {"type":"project_question","projectPath":"/Users/user/project-b","state":{...}}

data: {"type":"project_completed","projectPath":"/Users/user/project-a","state":{...}}

data: {"type":"project_error","projectPath":"/Users/user/project-c","state":{...}}

: keepalive
```

#### PatrolEvent オブジェクト

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `type` | string | イベントタイプ |
| `projectPath` | string | 対象プロジェクトのパス |
| `state` | ProjectState | プロジェクトの状態 |
| `message` | string | メッセージ（scan_completedで使用） |

#### PatrolEvent タイプ

| タイプ | 説明 |
|--------|------|
| `project_started` | プロジェクトのClaude CLI実行を開始 |
| `project_question` | 承認待ちの質問が発生 |
| `project_completed` | プロジェクトの実行が完了 |
| `project_error` | プロジェクトの実行でエラーが発生 |
| `scan_completed` | 全プロジェクトのスキャンが完了 |

---

### POST /api/patrol/polling/start

5分間隔の定期ポーリングを開始する。ポーリングごとに `StartPatrol` を呼び出し、未処理タスクのあるプロジェクトを自動実行する。

既にポーリング中の場合は既存のポーリングを停止して再開始する。

#### リクエスト

```
POST /api/patrol/polling/start
```

ボディなし。

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | ポーリング開始 |

---

### POST /api/patrol/polling/stop

定期ポーリングを停止する。実行中の巡回には影響しない。

#### リクエスト

```
POST /api/patrol/polling/stop
```

ボディなし。

#### レスポンス（成功）

```json
{
    "success": true
}
```

#### HTTPステータスコード

| コード | 説明 |
|--------|------|
| 200 | ポーリング停止 |
