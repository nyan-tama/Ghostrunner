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

生成先プロジェクトに `.claude/` ディレクトリを作成し、以下のファイルをAIが生成する。
Ghostrunnerの対応ファイルを参照し、構造・パターンを踏襲しつつ、新プロジェクトの文脈に合わせて書き換える。

#### 6.1 CLAUDE.md

Ghostrunnerの `.claude/CLAUDE.md` (`/Users/user/Ghostrunner/.claude/CLAUDE.md`) の構造を参考に生成する。

含めるセクション:
- **プロジェクト概要**: ユーザーが入力した概要を反映。技術スタック（Go + Gin, Next.js, Tailwind CSS）を記載。DB選択時は PostgreSQL + GORM も記載
- **Backend (Go)**: コード構成、コードスタイル、エラーハンドリング、テスト、ファイル構造、ビルド・実行コマンド
- **Frontend (Next.js)**: 技術スタック、コード構成、コードスタイル、テスト、ファイル構造、ビルド・実行コマンド
- **共通ルール**: セキュリティ、Gitワークフロー（日本語コミットメッセージ）、Makefileコマンド

#### 6.2 agents/

各エージェントはGhostrunnerの対応ファイル (`/Users/user/Ghostrunner/.claude/agents/`) を参照し、ドメイン固有の記述を新プロジェクトに合わせて書き換える。

**常に含める:**
- `discuss.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/discuss.md`
- `research.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/research.md`
- `reporter.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/reporter.md`
- `fix-judge.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/fix-judge.md`
- `test-planner.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/test-planner.md`

**Go + Next.js 構成（常に含める）:**
- `go-impl.md`, `go-reviewer.md`, `go-tester.md`, `go-planner.md`, `go-documenter.md`, `go-plan-reviewer.md`
- `nextjs-impl.md`, `nextjs-reviewer.md`, `nextjs-tester.md`, `nextjs-planner.md`, `nextjs-documenter.md`, `nextjs-plan-reviewer.md`
- 各ファイル参照元: `/Users/user/Ghostrunner/.claude/agents/<対応ファイル>`

**Cloud Run 構成（常に含める）:**
- `staging-manager.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/staging-manager.md`
- `release-manager.md` - 参照: `/Users/user/Ghostrunner/.claude/agents/release-manager.md`

**DB選択時のみ追加:**
- `pg-impl.md`, `pg-reviewer.md`, `pg-planner.md`, `pg-tester.md`
- 各ファイル参照元: `/Users/user/Ghostrunner/.claude/agents/<対応ファイル>`

#### 6.3 commands/

各コマンドはGhostrunnerの対応ファイル (`/Users/user/Ghostrunner/.claude/commands/`) を参照し、パスやプロジェクト名を書き換える。

**常に含める:**
- `discuss.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/discuss.md`
- `research.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/research.md`
- `plan.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/plan.md`
- `fix.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/fix.md`

**Go + Next.js 構成（常に含める）:**
- `go.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/go.md`
- `nextjs.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/nextjs.md`
- `fullstack.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/fullstack.md`

**Cloud Run 構成（常に含める）:**
- `stage.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/stage.md`
- `release.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/release.md`
- `hotfix.md` - 参照: `/Users/user/Ghostrunner/.claude/commands/hotfix.md`

#### 6.4 settings.json

PostToolUseフックを含む settings.json を生成する:

- `.go` ファイル編集後: `gofmt -w` で自動フォーマット
- `.go` ファイル編集後: `go vet` でチェック
- `.go` ファイル編集後: `fmt.Print` の警告
- `.ts/.tsx` ファイル編集後: `tsc --noEmit` で型チェック
- `.ts/.tsx/.js/.jsx` ファイル編集後: `console.log` の警告

参照: `/Users/user/Ghostrunner/.claude/settings.json`

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
