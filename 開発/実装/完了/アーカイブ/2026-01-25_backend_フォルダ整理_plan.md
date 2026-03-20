# backend フォルダ整理 実装計画

## 1. 概要

バックエンド関連ファイル（`cmd/`, `internal/`, `go.mod`, `go.sum`, `docs/`）をルート直下から `backend/` フォルダに移動し、プロジェクト構造を整理する。

## 2. 現状と目標

### 現状の構造

```
Ghostrunner/
├── cmd/server/main.go           # バックエンドエントリーポイント
├── internal/
│   ├── handler/                 # HTTPハンドラー
│   └── service/                 # ビジネスロジック
├── docs/BACKEND_API.md          # APIドキュメント
├── go.mod                       # module ghostrunner
├── go.sum
├── server                       # ビルド済みバイナリ
├── web/index.html               # 旧フロントエンド（Next.js移行済み）
├── frontend/                    # Next.jsフロントエンド
└── .claude/                     # Claude Code設定
```

### 目標の構造

```
Ghostrunner/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── handler/
│   │   └── service/
│   ├── docs/BACKEND_API.md
│   ├── go.mod                   # module ghostrunner/backend
│   └── go.sum
├── frontend/
└── .claude/
```

## 3. 変更対象ファイル一覧

### 移動するファイル

| 移動元 | 移動先 |
|-------|-------|
| `cmd/` | `backend/cmd/` |
| `internal/` | `backend/internal/` |
| `docs/BACKEND_API.md` | `backend/docs/BACKEND_API.md` |
| `go.mod` | `backend/go.mod` |
| `go.sum` | `backend/go.sum` |

### 修正するファイル

| ファイル | 修正内容 |
|---------|---------|
| `backend/go.mod` | モジュール名 `ghostrunner` → `ghostrunner/backend` |
| `backend/cmd/server/main.go` | import文修正、静的ファイル配信削除 |
| `backend/internal/handler/command.go` | import文修正 |
| `backend/internal/handler/plan.go` | import文修正 |
| `.gitignore` | `backend/server` 追加 |

### 削除するファイル

| ファイル | 理由 |
|---------|-----|
| `server` | ビルド済みバイナリ（再ビルドで生成可能） |
| `web/` | Next.jsフロントエンドに完全移行済み |

### Claude Code 設定ファイル（パス参照の修正不要）

既存のエージェント/コマンドファイルは既に `backend/` パスを前提として記述されているため、修正不要。

## 4. 実装ステップ

### Step 1: ディレクトリ作成とファイル移動

```bash
# backend/ディレクトリ作成
mkdir -p backend/docs

# ファイル移動
mv cmd/ backend/
mv internal/ backend/
mv docs/BACKEND_API.md backend/docs/
mv go.mod backend/
mv go.sum backend/
```

### Step 2: go.mod モジュール名変更

**対象:** `backend/go.mod`

```diff
- module ghostrunner
+ module ghostrunner/backend
```

### Step 3: import文の修正

**対象ファイル:**

1. `backend/cmd/server/main.go`
   ```diff
   - "ghostrunner/internal/handler"
   - "ghostrunner/internal/service"
   + "ghostrunner/backend/internal/handler"
   + "ghostrunner/backend/internal/service"
   ```

2. `backend/internal/handler/command.go`
   ```diff
   - "ghostrunner/internal/service"
   + "ghostrunner/backend/internal/service"
   ```

3. `backend/internal/handler/plan.go`
   ```diff
   - "ghostrunner/internal/service"
   + "ghostrunner/backend/internal/service"
   ```

### Step 4: main.go の静的ファイル配信削除

**対象:** `backend/cmd/server/main.go`

以下の2行を削除（Next.jsフロントエンドに移行済み）:
```go
r.StaticFile("/", "./web/index.html")
r.Static("/web", "./web")
```

### Step 5: .gitignore 更新

```diff
.env
+ backend/server
```

### Step 6: 旧ファイル削除

```bash
rm -rf web/
rm -f server
rmdir docs/  # 空の場合のみ
```

### Step 7: ビルド確認

```bash
cd backend
go build -o server ./cmd/server
go test ./...
go vet ./...
```

## 5. 確認事項

実装前に以下を確認:

1. **`web/index.html` の削除** - Next.jsフロントエンドに完全移行済みと判断してよいか
2. **`server` バイナリの削除** - 再ビルドで生成可能なため削除してよいか
3. **モジュール名 `ghostrunner/backend`** - この命名でよいか

## 6. 補足

### エージェント/コマンドファイルについて

`.claude/agents/` および `.claude/commands/` 内のファイルは既に `backend/` パスを前提として記述されているため、今回の移動作業で整合性が取れる状態になる。フォルダ分けは行わない。

### 次回実装（スコープ外）

- Dockerfile作成（Cloud Run用）
- Makefile作成（ビルド自動化）
- `backend/docs/` 配下のドキュメント拡充（BACKEND_CONTRIB.md, BACKEND_RUNBOOK.md）

---

## 実装完了レポート

### 実装サマリー
- **実装日**: 2026-01-25
- **変更ファイル数**: 10+ files（移動・修正・削除含む）
- **ビルド確認**: 成功
- **レビュー結果**: Critical/Warning なし

### 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `cmd/server/` -> `backend/cmd/server/` | ディレクトリ移動 |
| `internal/handler/` -> `backend/internal/handler/` | ディレクトリ移動 |
| `internal/service/` -> `backend/internal/service/` | ディレクトリ移動 |
| `docs/BACKEND_API.md` -> `backend/docs/BACKEND_API.md` | ファイル移動 |
| `go.mod`, `go.sum` -> `backend/` | ファイル移動 |
| `backend/go.mod` | モジュール名を `ghostrunner` から `ghostrunner/backend` に変更 |
| `backend/cmd/server/main.go` | import文修正、静的ファイル配信削除 |
| `backend/internal/handler/command.go` | import文修正 |
| `backend/internal/handler/plan.go` | import文修正 |
| `.gitignore` | `backend/server` を追加 |
| `backend/internal/handler/doc.go` | インポートパスの使用例を追加 |
| `backend/internal/service/doc.go` | インポートパスの使用例を追加 |
| `backend/docs/BACKEND_API.md` | モジュール構成セクションを追加 |
| `web/` | 削除（Next.js移行済み） |
| `server` | 削除（ビルドバイナリ） |
| 旧 `cmd/`, `internal/`, `docs/` | 削除 |

### 計画からの変更点

実装計画に記載がなかった判断・選択：

- `backend/internal/handler/doc.go` に新しいインポートパスの使用例を追加（開発者への案内）
- `backend/internal/service/doc.go` に新しいインポートパスの使用例を追加（開発者への案内）
- `backend/docs/BACKEND_API.md` にモジュール構成セクションを追加（ドキュメント拡充）

### 実装時の課題

#### ビルド・テストで苦戦した点

特になし

#### 技術的に難しかった点

特になし

### 残存する懸念点

今後注意が必要な点：

- CI/CD設定がある場合、`backend/` ディレクトリを前提としたパスに更新が必要
- Cloud Build等の設定ファイルがある場合、同様にパス修正が必要
- 他の開発者がローカルで作業している場合、`git pull` 後に `go mod tidy` が必要な可能性あり

### 動作確認フロー

```
1. cd backend
2. go build -o server ./cmd/server
   -> ビルド成功を確認
3. go vet ./...
   -> 静的解析エラーなしを確認
4. go fmt ./...
   -> フォーマット変更なし（既に整形済み）
```

### デプロイ後の確認事項

- [ ] Cloud Run等のCI/CD設定でパスが正しく設定されているか確認
- [ ] 本番環境でのビルド・デプロイが正常に完了するか確認
- [ ] APIエンドポイントが正常に動作するか確認
