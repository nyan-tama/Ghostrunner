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
go build -o server ./cmd/server
```

## テスト

```bash
cd backend
go test ./...
```

## ディレクトリ構成

```
backend/
|-- cmd/
|   |-- server/       # メインエントリーポイント
|-- internal/
|   |-- handler/      # HTTPハンドラー（リクエスト受信、レスポンス返却）
|   |-- service/      # ビジネスロジック（Claude CLI実行、外部API連携、通知）
|-- docs/             # ドキュメント
```

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

## コーディング規約

- `gofmt`/`goimports` でフォーマット
- 本番コードに `fmt.Println` を使用しない（`log` パッケージを使用）
- エラーは `fmt.Errorf("failed to X: %w", err)` でコンテキストを追加して返す
- パニックは避ける
- 公開関数・型にはGoDocコメントを付与
- テストはテーブル駆動テストを使用
