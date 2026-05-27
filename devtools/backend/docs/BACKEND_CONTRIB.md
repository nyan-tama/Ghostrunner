# Backend 開発者ガイド

Ghostrunner バックエンドの開発環境セットアップと開発手順。

## 前提条件

- Go 1.24
- Claude CLI（`claude` コマンドがPATHに存在すること）
- Node.js（フロントエンド開発時）

## 環境構築

### 1. 依存関係のインストール

```bash
cd backend
go mod download
```

### 2. 環境変数の設定

`backend/.env.example` を `backend/.env` にコピーし、必要な値を設定する。

```bash
cd backend
cp .env.example .env
```

#### 環境変数一覧

| 環境変数 | 必須 | 説明 |
|----------|------|------|
| `GEMINI_API_KEY` | No | Gemini Live API 用のAPIキー |
| `OPENAI_API_KEY` | No | OpenAI Realtime API 用のAPIキー（sk-xxx形式） |
| `NTFY_TOPIC` | No | ntfy.sh のトピック名（プッシュ通知用） |
| `ELEVENLABS_API_KEY` | No | ElevenLabs TTS 用のAPIキー（未設定時 `/api/tts` は503） |
| `ELEVENLABS_DEFAULT_VOICE_ID` | No | TTSデフォルトvoice_id（未指定時 `KgETZ36CCLD1Cob4xpkv`） |
| `ELEVENLABS_DEFAULT_MODEL` | No | TTSデフォルトmodel_id（未指定時 `eleven_flash_v2_5`） |

各環境変数が未設定の場合、対応する機能は無効になるが、サーバーの起動自体には影響しない。

### 3. ntfy.sh 通知のセットアップ（オプション）

ntfy.sh通知を有効にするとコマンド完了・エラー時にスマートフォンやブラウザへプッシュ通知が届く。

#### トピックの決定

トピック名は推測されにくいユニークな文字列にする。ntfy.sh のトピックは公開されているため、他のユーザーと重複しない名前を選ぶ。

```bash
# 例: ランダムな文字列を生成して使用
echo "ghostrunner-$(openssl rand -hex 6)"
```

#### 環境変数の設定

```bash
# backend/.env に追加
NTFY_TOPIC=ghostrunner-your-unique-id
```

#### 通知の受信

1. スマートフォンに ntfy アプリをインストール（[iOS](https://apps.apple.com/app/ntfy/id1625396347) / [Android](https://play.google.com/store/apps/details?id=io.heckel.ntfy)）
2. アプリ内で `NTFY_TOPIC` に設定した同じトピック名を購読
3. ブラウザからの受信: `https://ntfy.sh/your-topic-name` にアクセス

#### 動作確認

サーバー起動時のログに以下が表示されれば通知機能が有効。

```
[NtfyService] Initialized with topic: https://ntfy.sh/your-topic-name
```

未設定の場合は以下が表示される（正常動作）。

```
[NtfyService] NTFY_TOPIC is not set, ntfy notification will not be available
```

## サーバーの起動

プロジェクトルートの Makefile を使用する。

```bash
# バックエンドのみ起動（フォアグラウンド）
make backend

# バックエンドをバックグラウンドで起動してログ表示
make start-backend

# バックエンド + フロントエンドを同時起動
make dev
```

## ビルド

```bash
cd backend

# APIサーバー
go build -o server ./cmd/server

# gr-run CLI（一括実装用ワンショット実行ツール）
go build -o gr-run ./cmd/gr-run
# または、プロジェクトルートから
make gr-run
```

### gr-run の使い方

```bash
gr-run --project <プロジェクトの絶対パス> --task <タスクファイル名> [--locks-dir <ロックディレクトリ>]
```

| フラグ | 必須 | 説明 |
|--------|------|------|
| `--project` | Yes | 対象プロジェクトの絶対パス |
| `--task` | Yes | `開発/実装/実装待ち/` 内のタスクファイル名 |
| `--locks-dir` | No | flock ファイルの格納先（デフォルト: `~/.ghostrunner/locks/`） |

gr-run は1タスクを処理して終了するワンショットCLI。複数プロセスを並列起動することで一括実装を実現する。プロジェクト単位の排他ロック（flock）により、同一プロジェクトへの多重実行を防止する。

## テスト

```bash
cd backend
go test ./...
```

## ディレクトリ構成

```
backend/
|-- cmd/
|   |-- server/       # APIサーバーのエントリーポイント
|   |-- gr-run/       # 一括実装用ワンショットCLI
|-- internal/
|   |-- handler/      # HTTPハンドラー（リクエスト受信、レスポンス返却）
|   |-- service/      # ビジネスロジック（Claude CLI実行、外部API連携、通知、プロジェクト生成）
|   |-- grrun/        # gr-run CLIのコアロジック（ロック、クレーム、結果分類）
|   |-- projects/     # patrol_projects.json読み込み（PatrolServiceとdashboardの共通依存）
|   |-- dashboard/    # ダッシュボード状態集約・回答書き戻し（カンバン/未回答/運用）
|   |-- tts/          # ElevenLabs TTSプロキシ（handler + service + client + cache を集約）
|-- docs/             # ドキュメント
```

### パッケージ一覧

| パッケージ | 責務 |
|-----------|------|
| `internal/handler` | HTTPハンドラー（リクエスト受信、レスポンス返却） |
| `internal/service` | ビジネスロジック（Claude CLI実行、外部API連携、通知、プロジェクト生成） |
| `internal/grrun` | gr-run CLIのコアロジック（ロック、クレーム、結果分類） |
| `internal/projects` | patrol_projects.jsonの読み込み（PatrolServiceとdashboardの共通依存） |
| `internal/dashboard` | ダッシュボード状態集約・回答書き戻し（カンバン/未回答/運用） |
| `internal/tts` | ElevenLabs Text-to-Speech プロキシ。handler / service / client / cache を1パッケージに集約。単機能・閉じたドメインで client/cache が他から利用されないため（grrunと同じ集約方針） |

### 依存ライブラリ

主要な外部依存:

| ライブラリ | 用途 |
|-----------|------|
| `github.com/gin-gonic/gin` | HTTPルーティング |
| `github.com/gin-contrib/cors` | CORSミドルウェア |
| `google.golang.org/genai` | Gemini Live API クライアント |
| `github.com/hashicorp/golang-lru/v2` | TTSキャッシュ（LRUエビクション） |
| `golang.org/x/sync` | TTS singleflight（重複リクエスト統合） |
| `gopkg.in/yaml.v3` | テンプレートメタデータの読み込み |
| `github.com/stretchr/testify` | テスト用アサーション |

## アーキテクチャ

### 依存関係の方向

```
main.go
  |-- handler (HTTPリクエスト/レスポンス)
  |     |-- service (ビジネスロジック)
  |           |-- NtfyService (通知、オプション)
  |           |-- Claude CLI (外部プロセス)
  |           |-- OpenAI API (外部API)
  |           |-- Gemini API (外部API)
  |-- handler/CreateHandler
  |     |-- service/CreateProjectService (プロジェクト生成)
  |           |-- TemplateService (テンプレートコピー・加工)
  |           |-- git, go, npm (外部コマンド)
  |           |-- code (VS Code起動)
  |-- handler/PatrolHandler
  |     |-- service/PatrolService (複数プロジェクト自動巡回)
  |           |-- ClaudeService (CLI実行)
  |           |-- NtfyService (承認待ち通知)
  |           |-- JSONファイル (設定永続化)
  |-- handler/DashboardHandler
  |     |-- dashboard/Service (ダッシュボード状態集約・回答)
  |           |-- projects/LoadProjects (設定読み込み)
  |           |-- dashboard/ScanProject (ファイルシステム読み取り)
  |           |-- dashboard/AnswerQuestion (アトミック書き戻し)
  |-- tts/Handler
  |     |-- tts/Service (singleflight + cache を統合するビジネスロジック)
  |           |-- tts/Client (ElevenLabs APIクライアント)
  |           |-- tts/Cache (LRU + TTL + バイト数上限のインメモリキャッシュ)
```

### NtfyService の注入パターン

NtfyService はオプショナルな依存関係として設計されている。`NewNtfyService()` は環境変数 `NTFY_TOPIC` が未設定の場合に `nil` を返し、ClaudeService は `nil` チェックを行って通知をスキップする。

```go
// main.go での初期化
ntfyService := service.NewNtfyService()   // nil の場合がある
claudeService := service.NewClaudeService(ntfyService)

// ClaudeService 内での使用
func (s *claudeServiceImpl) notifyComplete(output string) {
    if s.ntfyService == nil {
        return  // 通知機能が無効の場合はスキップ
    }
    s.ntfyService.Notify("Claude Code - Complete", message)
}
```

### CreateProjectService の注入パターン

CreateProjectService はプロジェクト生成のインターフェースを定義する。CreateService がその実装を提供し、内部でTemplateServiceに委譲する。

```go
// main.go での初期化
ghostrunnerRoot := "/path/to/Ghostrunner"
projectBaseDir := "/Users/user"

templateService := service.NewTemplateService(ghostrunnerRoot)
createService := service.NewCreateService(templateService, projectBaseDir)
createHandler := handler.NewCreateHandler(createService)
```

CreateProjectService インターフェース:

```go
type CreateProjectService interface {
    ValidateProjectName(name string) *ValidateResult
    CreateProject(ctx context.Context, req *CreateRequest, eventCh chan<- CreateEvent)
    OpenInVSCode(path string) error
    ProjectBaseDir() string
}
```

CreateProject メソッドはチャンネル経由で進捗イベントを送信する。ハンドラーはこのチャンネルからイベントを読み取り、SSE形式でクライアントに配信する。チャンネルは CreateProject 内で close される。

### PatrolService の注入パターン

PatrolService は複数プロジェクトの自動巡回を担当する。ClaudeService と NtfyService に依存し、設定ファイルパスを受け取って初期化する。

```go
// main.go での初期化
patrolConfigPath := filepath.Join(ghostrunnerRoot, "devtools", "backend", "patrol_projects.json")
patrolService := service.NewPatrolService(claudeService, ntfyService, patrolConfigPath)
patrolHandler := handler.NewPatrolHandler(patrolService)
```

PatrolService インターフェース:

```go
type PatrolService interface {
    RegisterProject(path string) error
    UnregisterProject(path string) error
    ListProjects() []PatrolProject
    ScanProjects() []ScanResult
    StartPatrol() error
    StopPatrol()
    ResumeProject(projectPath, answer string) error
    GetStates() map[string]*ProjectState
    StartPolling()
    StopPolling()
    Subscribe() (<-chan PatrolEvent, func())
}
```

主要な設計判断:
- 並列実行数はセマフォ（バッファ付きチャンネル）で制御（最大5並列）
- SSEイベントはSubscribe/broadcastパターンで配信（バッファ100件のチャンネル）
- 設定ファイルへの保存はwrite-to-temp + renameパターンで安全に書き込み
- 実行中または承認待ちのプロジェクトは巡回時にスキップ

### DashboardService の注入パターン

DashboardService はダッシュボード状態集約のインターフェースを定義する。patrol_projects.json のパスとGhostrunnerルートパスを受け取って初期化する。projectsパッケージを共通依存としてPatrolServiceと設定ファイルの読み込みを共有する。

```go
// main.go での初期化
dashboardService := dashboard.NewService(patrolConfigPath, ghostrunnerRoot)
dashboardHandler := handler.NewDashboardHandler(dashboardService)
```

dashboard.Service インターフェース:

```go
type Service interface {
    GetState(ctx context.Context) (State, error)
    Answer(ctx context.Context, req AnswerRequest) error
}
```

主要な設計判断:
- ファイルシステムを唯一の真実源とし、キャッシュやDBを持たない
- 未回答検出の正規表現パターンはgrrunパッケージのSSOTを共有
- 回答書き戻しはwrite-to-temp + renameパターンでアトミックに書き込み
- テスト用にclock注入（NewServiceWithClock）をサポート
- ScanProject失敗時もwarning付きで結果に含め、他プロジェクトの集約を継続

### TTSService の注入パターン

TTSService はElevenLabs Text-to-Speechのプロキシ機能を提供する。`NewService()` は環境変数 `ELEVENLABS_API_KEY` が未設定の場合に `nil` を返し、Handler は `nil` チェックを行って503を返す（NtfyService と同型のオプショナル機能パターン）。

```go
// main.go での初期化
ttsService := tts.NewService()        // nil の場合がある
ttsHandler := tts.NewHandler(ttsService)
api.POST("/tts", ttsHandler.HandleTTS)
```

tts.Service / tts.Handler は同パッケージ内に集約されている。これは TTS が単機能・閉じたドメインで、client/cache が他パッケージから利用されないため（`internal/grrun` と同じ集約方針）。

主要な設計判断:
- non-streamエンドポイントを採用（バックエンドで読み切る前提なら stream のメリットなし、Content-Length が確定して扱いやすい）
- キャッシュは LRU + TTL + バイト数のハイブリッド（エントリ数固定では1リクエストあたりのサイズ変動に追従できない）
- singleflight中の上流呼び出しは個別の context cancel に従わず、最初に開始したリクエストの ctx が支配する
- エラーボディは先頭200文字のみ保持し、APIキー混入リスクを避ける
- 上流401（キー無効）は backend ログのみで把握し、フロントには502に丸める（攻撃者にキー状態を漏らさない）
- リクエストは `text` のみ受け付け、voice_id / model_id はbackend env固定。フロント↔バックエンド契約の片側化を防ぎ、将来のVoice選択UI追加時に camelCase で `voiceId` / `modelId` を足す拡張余地を残す

### TemplateService の責務

TemplateService はテンプレートファイルのコピーと加工に特化したサービス。CreateService から呼び出され、以下の操作を担当する。

- テンプレートディレクトリの再帰コピー（バイナリファイル判定付き）
- `{{PROJECT_NAME}}` プレースホルダーの一括置換
- 複数サービスの `docker-compose.yml` マージ（services と volumes キーを結合）
- `.claude/` ディレクトリのコピーとカスタマイズ
- プロジェクト用 `CLAUDE.md` の動的生成

サービス名からテンプレートディレクトリへのマッピング:

| サービス名 | テンプレートディレクトリ |
|-----------|----------------------|
| `database` | `templates/with-db` |
| `storage` | `templates/with-storage` |
| `cache` | `templates/with-redis` |

## コーディング規約

- `gofmt`/`goimports` でフォーマット
- 本番コードに `fmt.Println` を使用しない（`log` パッケージを使用）
- エラーは `fmt.Errorf("failed to X: %w", err)` でコンテキストを追加して返す
- パニックは避ける
- 公開関数・型にはGoDocコメントを付与
- テストはテーブル駆動テストを使用
