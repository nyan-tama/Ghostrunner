# /init - プロジェクトスターター

引数 `$ARGUMENTS` から新プロジェクトを対話的に生成する。

## 処理フロー

### Step 1: 引数の処理とバリデーション

`$ARGUMENTS` からプロジェクト名を取得する。

**バリデーション:**
- プロジェクト名が空の場合はエラー: 「プロジェクト名を指定してください。例: `/init my-project`」
- プロジェクト名が英数字+ハイフン以外を含む場合はエラー: 「プロジェクト名は英数字とハイフンのみ使用できます」
- 生成先 `/Users/user/<プロジェクト名>/` が既に存在する場合はエラー: 「ディレクトリが既に存在します」

### Step 2: 対話で情報収集

AskUserQuestion を使って以下を順に質問する:

**Q1: プロジェクトの概要**
「プロジェクトの概要を教えてください（CLAUDE.mdに記載されます）」

**Q2: DB使用の有無**
「データベース（PostgreSQL）を使いますか？」
- 選択肢: Yes / No

**Q3: 最終確認**
収集した情報を表示し、生成を開始してよいか確認する:
```
プロジェクト名: <名前>
生成先: /Users/user/<名前>/
概要: <入力された概要>
DB: あり/なし
```

### Step 3: テンプレートコピー

```bash
# 生成先ディレクトリ作成
mkdir -p /Users/user/<プロジェクト名>

# base テンプレートをコピー
cp -r /Users/user/Ghostrunner/templates/base/. /Users/user/<プロジェクト名>/
```

DB選択が「Yes」の場合:
```bash
# with-db テンプレートで上書きコピー（base版を置換 + 追加ファイル）
cp -r /Users/user/Ghostrunner/templates/with-db/. /Users/user/<プロジェクト名>/
```

### Step 4: プレースホルダー置換

`{{PROJECT_NAME}}` を実際のプロジェクト名に一括置換する。

**重要**: バイナリファイル破損を防ぐため、対象はテキストファイル拡張子のみに限定する。

```bash
cd /Users/user/<プロジェクト名>
find . -type f \( \
  -name "*.go" -o -name "*.mod" -o -name "*.json" -o -name "*.tsx" -o -name "*.ts" \
  -o -name "*.css" -o -name "*.yml" -o -name "*.yaml" -o -name "*.md" \
  -o -name "*.mjs" -o -name "*.sql" -o -name "Makefile" \
  -o -name "Dockerfile" -o -name ".gitignore" \
\) -exec sed -i '' "s/{{PROJECT_NAME}}/<プロジェクト名>/g" {} +
```

### Step 5: 依存関係の解決

```bash
# Go modules
cd /Users/user/<プロジェクト名>/backend && go mod tidy

# npm
cd /Users/user/<プロジェクト名>/frontend && npm install
```

### Step 6: .claude/ 資産の生成

Ghostrunnerの `.claude/` 資産を一括コピーし、CLAUDE.md だけ新プロジェクト用に生成する。

#### 6.1 一括コピー

```bash
mkdir -p /Users/user/<プロジェクト名>/.claude/agents /Users/user/<プロジェクト名>/.claude/commands

# agents/ と commands/ を一括コピー
cp /Users/user/Ghostrunner/.claude/agents/*.md /Users/user/<プロジェクト名>/.claude/agents/
cp /Users/user/Ghostrunner/.claude/commands/*.md /Users/user/<プロジェクト名>/.claude/commands/

# settings.json をコピー
cp /Users/user/Ghostrunner/.claude/settings.json /Users/user/<プロジェクト名>/.claude/settings.json
```

DB未選択時は pg 系エージェントを削除:
```bash
rm -f /Users/user/<プロジェクト名>/.claude/agents/pg-*.md
```

#### 6.2 CLAUDE.md 生成

Ghostrunnerの `.claude/CLAUDE.md` (`/Users/user/Ghostrunner/.claude/CLAUDE.md`) の構造を参考に、新プロジェクト用に生成する。

含めるセクション:
- **プロジェクト概要**: ユーザーが入力した概要を反映。技術スタック（Go + Gin, Next.js, Tailwind CSS）を記載。DB選択時は PostgreSQL + GORM も記載
- **Backend (Go)**: コード構成、コードスタイル、エラーハンドリング、テスト、ファイル構造、ビルド・実行コマンド
- **Frontend (Next.js)**: 技術スタック、コード構成、コードスタイル、テスト、ファイル構造、ビルド・実行コマンド
- **共通ルール**: セキュリティ、Gitワークフロー（日本語コミットメッセージ）、Makefileコマンド

### Step 7: Git 初期化

```bash
cd /Users/user/<プロジェクト名>
git init
git add -A
git commit -m "feat: プロジェクト初期化 - Go + Next.js フルスタック構成"
```

### Step 8: 完了メッセージ

以下を表示する:

```
プロジェクト「<プロジェクト名>」の生成が完了しました！

生成先: /Users/user/<プロジェクト名>/

ローカル起動手順:
  cd /Users/user/<プロジェクト名>
  make dev

アクセス:
  フロントエンド: http://localhost:3000
  バックエンド API: http://localhost:8080/api/health

DB使用の場合は先にDockerを起動してください:
  docker-compose up -d db
  make dev
```
