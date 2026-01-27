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

## エンドポイント一覧

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/api/health` | GET | ヘルスチェック |
| `/api/command` | POST | コマンドの同期実行 |
| `/api/command/stream` | POST | コマンドのストリーミング実行 (SSE) |
| `/api/command/continue` | POST | セッション継続 |
| `/api/command/continue/stream` | POST | セッション継続のストリーミング実行 (SSE) |
| `/api/files` | GET | 開発フォルダ内のmdファイル一覧取得 |
| `/api/plan` | POST | /planコマンドの同期実行（後方互換性） |
| `/api/plan/stream` | POST | /planコマンドのストリーミング実行（後方互換性） |
| `/api/plan/continue` | POST | セッション継続（後方互換性） |
| `/api/plan/continue/stream` | POST | セッション継続のストリーミング実行（後方互換性） |
| `/api/gemini/token` | POST | Gemini Live API 用エフェメラルトークン発行 |
| `/api/openai/realtime/session` | POST | OpenAI Realtime API 用エフェメラルキー発行 |

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
