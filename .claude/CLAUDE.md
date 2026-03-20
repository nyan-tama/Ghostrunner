# プロジェクト CLAUDE.md

Ghostrunner - Claude Code用フルスタック開発フレームワーク（Claude Code Plugin）。

## プロジェクト概要

個人開発者向けのClaude Code Pluginとして、エージェント・スキル・テンプレート・devtoolsを統合し、`/plugin install ghostrunner` でどのプロジェクトからでも利用可能にする。

**Plugin構成:**
- `agents/` - 23エージェント（Go/Next.js/PostgreSQL/ストレージ/Redis/運用）
- `skills/` - 13スキル（/init, /fullstack, /plan, /stage, /release 等）
- `hooks/` - フック設定（コード品質チェック、フォーマッター）
- `templates/` - プロジェクト雛形（base, with-db, with-storage, with-redis）
- `devtools/` - 進捗ビューア（Go + Next.js アプリ）

## ファイル構造

```
Ghostrunner/
|-- .claude-plugin/
|   |-- plugin.json        # プラグインマニフェスト
|-- agents/                # エージェント定義（.md）
|-- skills/                # スキル定義（*/SKILL.md）
|-- hooks/
|   |-- hooks.json         # フック設定
|-- templates/
|   |-- base/              # Go + Next.js 基本構成
|   |-- with-db/           # PostgreSQL + GORM
|   |-- with-storage/      # Cloudflare R2 / MinIO
|   |-- with-redis/        # Redis
|-- devtools/
|   |-- backend/           # 進捗ビューア API（Go + Gin）
|   |-- frontend/          # 進捗ビューア UI（Next.js）
|-- 開発/                  # 開発ドキュメント
|-- .claude/
|   |-- CLAUDE.md          # このファイル
|   |-- settings.json      # Ghostrunner自体の開発用設定
```

---

# devtools (Go バックエンド)

`devtools/backend/` ディレクトリのコード用ルール。

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

## ビルド・実行

```bash
cd devtools/backend
go build -o server ./cmd/server
go test ./...
```

---

# devtools (Next.js フロントエンド)

`devtools/frontend/` ディレクトリのコード用ルール。

## 技術スタック

- Next.js 15 (App Router) + React 19
- TypeScript (strict mode)
- Tailwind CSS

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

### テスト

- TDD: テストを先に書く
- 最低80%カバレッジ

## ビルド・実行

```bash
cd devtools/frontend
npm install
npm run dev
npm run build
```

---

# 共通ルール

## セキュリティ

- シークレットのハードコード禁止
- 機密データは環境変数で管理
- 全ユーザー入力をバリデーション

## Gitワークフロー

### コミットメッセージ

**重要: コミットメッセージは必ず日本語で詳細に記述する**

- Conventional commitsの接頭辞を使用: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`
- 接頭辞の後は**日本語**で詳細に記述
- 後から`git log`で検索しやすいように具体的なキーワードを含める

**良い例:**
```
feat: 質問の逐次表示機能を追加、ユーザーが回答するまで次の質問を非表示にする処理を実装
fix: MakefileのsedコマンドにLC_ALL=Cを追加、日本語ログでのエラーを修正
```

### ブランチとマージ

- mainに直接コミットしない（開発/ 配下は例外）
- PRにはレビューが必要
- マージ前に全テストが通ること

## 開発サーバーの起動・停止

プロジェクトルートの `Makefile` を使用してサーバーを管理する。

### エージェント向けルール

- サーバーの起動・停止・再起動には**必ず `make` コマンドを使用**
- 直接 `go run ./cmd/server` や `npm run dev` を実行しない

### 主要コマンド

```bash
make backend          # devtools バックエンドを起動
make frontend         # devtools フロントエンドを起動
make dev              # 両方を並列起動
make stop             # 両方を停止
make health           # ヘルスチェック
```

## Plugin開発・テスト

```bash
# ローカルでPluginとしてテスト
claude --plugin-dir /Users/user/Ghostrunner

# 変更後のリロード（セッション内）
/reload-plugins
```
