# プロジェクト CLAUDE.md

Ghostrunner - Claude Code用フルスタック開発フレームワーク。

## プロジェクト概要

個人開発者向けのClaude Code開発フレームワーク。Ghostrunnerを開いて `/init` を実行すると、エージェント・コマンド・テンプレートが組み込まれた新規プロジェクトを対話的に生成する。

**構成:**
- `.claude/agents/` - 29エージェント（Go/Next.js/Swift/PostgreSQL/ストレージ/Redis/運用）
- `.claude/skills/` - 13スキル（/init, /coding, /plan, /stage, /release 等）
- `.claude/settings.json` - フック設定（コード品質チェック、フォーマッター）
- `templates/` - プロジェクト雛形（base, with-db, with-storage, with-redis, swift-macos）
- `devtools/` - 進捗ビューア（Go + Next.js アプリ）

## 統括（把握）

Ghostrunner は個々の開発に加え、**全プロジェクトを横断管理する「統括ハブ」**を兼ねる。

- ユーザーから「全プロジェクトの状況は？」「今日どう？」など**横断的な状況確認**を求められたら、
  `chief-director` エージェントを呼んで状況を集約・報告する（読み取り専用）。
- 実装の実行・確認事項の回答・異常終了の解消など**状態を変える操作は、ユーザーの明示的な指示が
  あったときだけ**行う。「把握＝読み取り（自動）」と「操作＝指示があってから」を分ける。

## 統括（運用把握）

開発（カンバン）に加えて、各プロジェクトの**運用（ランタイム状態）**も横断把握する。

- 「全プロジェクトの状況は？」等の横断把握では、`chief-director` が各プロジェクトの `運用/状態/*.json` も
  読み、開発カンバンと並べて運用状態（進捗・stale・blocked・連続エラー）を報告する（読み取り専用）。
- **運用を持つ判定はフォルダベース**: プロジェクトに `運用/` フォルダ（`運用/manifest.json`）があれば運用あり。
  無ければ運用なし（`開発/` と対称。`実行中/` が任意なのと同じ前方互換）。
- **把握＝自動・読み取り**: 運用把握は読み取りのみ。運用の状態変更（一括運用＝`bulk-ops` の発火・再点火）は
  **明示動詞でのみ**行う（運用 Phase2・未実装）。「把握＝読み取り（自動）／操作＝指示があってから」を運用にも適用。
- **検知まで・解消は人間トリガー**: stale（実行停止疑い）・blocked（制限検知）・連続エラーは検知・報告するが、
  解消（Chrome 再起動・再点火・調査）はユーザーの明示指示で別途行う（後フェーズ）。開発側の異常終了の扱いと同型。

## 統括（一括操作）

- **発火条件**: 「一括codingして」「実装待ちを一括で開始」等の**明示動詞**でのみ `bulk-coding` スキルを呼ぶ。
  把握系（「状況は？」）では呼ばない。把握＝自動・読み取り と 操作＝明示指示 の線引き。
- **一括実装の流れ**: bulk-coding スキルが対象プロジェクトを選定し、gr-run を背景起動。
  完了・確認事項は ntfy 通知で届く。状況確認は chief-director（「状況は？」）。
- **確認事項の取り次ぎ**: chief-director が未回答確認事項を検知した場合、ユーザーに噛み砕いて伝え
  （A案/B案を提示）、回答を計画書に書き戻す（`**ステータス**: 回答済` に変更）。
  必要に応じて再度 bulk-coding で再ディスパッチ。
- **状態変更は明示指示のみ**（既存原則を一括操作にも適用）。

### プロジェクト登録

統括の対象プロジェクトは `devtools/backend/patrol_projects.json` で管理する（gitignore対象・ローカル専用）。
新しいプロジェクトを統括に追加するには、このJSONの `projects` 配列にエントリを追加する。
詳細な手順は `devtools/backend/docs/BACKEND_RUNBOOK.md` の「プロジェクト登録」を参照。

## ファイル構造

```
Ghostrunner/
|-- .claude/
|   |-- agents/            # エージェント定義（.md）
|   |-- skills/            # スキル定義（SKILL.md）
|   |-- settings.json      # フック設定
|   |-- CLAUDE.md          # このファイル
|-- templates/
|   |-- base/              # Go + Next.js 基本構成
|   |-- with-db/           # PostgreSQL + GORM
|   |-- with-storage/      # Cloudflare R2 / MinIO
|   |-- with-redis/        # Redis
|   |-- swift-macos/       # Swift + SwiftUI macOS アプリ
|-- devtools/
|   |-- backend/           # 進捗ビューア API（Go + Gin）
|   |-- frontend/          # 進捗ビューア UI（Next.js）
|-- 開発/                  # 開発ドキュメント
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
