# プロジェクト CLAUDE.md

データ復旧サービスの業務管理システム - Go バックエンド + Next.js フロントエンドのフルスタックプロジェクト。

## プロジェクト概要

Google Cloud Run上で動作するデータ復旧サービスの業務管理システム。

**バックエンド** (`backend/`)
- Go 1.24 + Gin Framework
- Google Sheets API（顧客データ管理）
- Gmail API（メール送信・下書き作成）
- Generative AI API（テキスト生成）
- OAuth2 / サービスアカウント認証

**フロントエンド** (`frontend/`)
- Next.js 15 (App Router) + React 19 + TypeScript
- Tailwind CSS
- NextAuth.js v5（認証）
- OpenAI Realtime API / Gemini Live API（音声AI）

**インフラ**
- Google Cloud Run（コンテナホスティング）
- Cloud Build（CI/CD）
- Google Spreadsheet（データストア）

---

# Backend (Go)

`backend/` ディレクトリのコード用ルール。

## 重要なルール

### コード構成

- Clean Architectureに従う
- 機能/ドメイン別にパッケージを整理
- 1ファイル200-400行、最大600行
- インターフェースで依存性を抽象化

### コードスタイル

- コード、コメント、ドキュメントに絵文字禁止
- `gofmt`/`goimports`でフォーマット
- 本番コードに`fmt.Println`禁止（`log`パッケージを使用）
- エラーは必ずハンドリング（`_`で無視しない）
- 意味のある変数名を使用

### エラーハンドリング

- エラーは呼び出し元に返す
- エラーにコンテキストを追加: `fmt.Errorf("failed to X: %w", err)`
- パニックは避ける（回復不能な状況のみ）
- APIレスポンスには適切なHTTPステータスコード

### テスト

- テーブル駆動テストを使用
- モックにはインターフェースを活用
- `_test.go`ファイルにテストを配置
- 最低80%カバレッジ目標

## ファイル構造

```
backend/
|-- cmd/
|   |-- server/       # メインエントリーポイント
|   |-- test_*/       # テスト用コマンド
|-- internal/
|   |-- config/       # 設定管理
|   |-- domain/       # ドメインモデル
|   |-- handler/      # HTTPハンドラー
|   |-- infrastructure/ # 外部サービス連携
|   |-- middleware/   # HTTPミドルウェア
|   |-- service/      # ビジネスロジック
|   |-- util/         # ユーティリティ
```

## ビルド・実行

```bash
cd backend
go build -o server ./cmd/server
go test ./...
go run ./cmd/server
```

## バックエンド用コマンド

- `/backend-build-fix` - ビルドエラー修正
- `/backend-test` - テスト実行
- `/backend-lint` - 静的解析
- `/backend-plan` - 実装計画作成
- `/backend-code-review` - コードレビュー

---

# Frontend (Next.js)

`frontend/` ディレクトリのコード用ルール。

## 技術スタック

- Next.js 15 (App Router) + React 19
- TypeScript (strict mode)
- Tailwind CSS
- NextAuth.js v5（認証）
- OpenAI Realtime API / Gemini Live API（音声AI）

## 重要なルール

### コード構成

- 大きなファイルを少数より、小さなファイルを多数
- 高凝集、低結合
- 通常200-400行、最大800行/ファイル
- 型別ではなく機能/ドメイン別に整理

### コードスタイル

- コード、コメント、ドキュメントに絵文字禁止
- 常にイミュータブル - オブジェクトや配列を変更しない
- 本番コードにconsole.log禁止
- try/catchで適切なエラーハンドリング
- Zodなどで入力バリデーション

### テスト

- TDD: テストを先に書く
- 最低80%カバレッジ
- ユーティリティにユニットテスト
- APIにインテグレーションテスト
- 重要なフローにE2Eテスト

## ファイル構造

```
frontend/
|-- src/
|   |-- app/          # Next.js app router
|   |-- components/   # 再利用可能なUIコンポーネント
|   |-- hooks/        # カスタムReactフック
|   |-- lib/          # ユーティリティライブラリ
|   |-- types/        # TypeScript型定義
|   |-- actions/      # Server Actions
```

## ビルド・実行

```bash
cd frontend
npm install
npm run dev
npm run build
```

## フロントエンド用コマンド

- `/frontend-build-fix` - ビルドエラー修正
- `/frontend-plan` - 実装計画作成
- `/frontend-code-review` - コードレビュー
- `/frontend-tdd` - テスト駆動開発
- `/frontend-e2e` - E2Eテスト実行

---

# 共通ルール

## セキュリティ

- シークレットのハードコード禁止
- 機密データは環境変数で管理
- 全ユーザー入力をバリデーション
- パラメータ化クエリのみ使用

## Gitワークフロー

- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`
- mainに直接コミットしない
- PRにはレビューが必要
- マージ前に全テストが通ること

## 開発サーバーの起動・停止

プロジェクトルートの `Makefile` を使用してサーバーを管理する。

### エージェント向けルール

- サーバーの起動・停止・再起動には**必ず `make` コマンドを使用**
- 直接 `go run ./cmd/server` や `npm run dev` を実行しない
- ログの確認には `make logs-backend` または `make logs-frontend` を使用

### 主要コマンド

**起動（フォアグラウンド）**
```bash
make backend          # バックエンドを起動
make frontend         # フロントエンドを起動
make dev              # 両方を並列起動
```

**再起動（ログ付き）**
```bash
make restart-backend-logs   # バックエンドを再起動してログ表示
make restart-frontend-logs  # フロントエンドを再起動してログ表示
```

**停止**
```bash
make stop-backend     # バックエンドを停止
make stop-frontend    # フロントエンドを停止
make stop             # 両方を停止
```

**ログ確認**
```bash
make logs-backend     # バックエンドのログを表示
make logs-frontend    # フロントエンドのログを表示
```

**その他**
```bash
make health           # ヘルスチェック
make build            # 両方をビルド
```
