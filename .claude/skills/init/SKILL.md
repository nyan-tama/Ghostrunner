---
name: init
description: プロジェクトスターター。新規プロジェクトを対話的に生成する
disable-model-invocation: true
---


# /init - プロジェクトスターター

引数 `$ARGUMENTS` から新プロジェクトを対話的に生成する。

## 対話ガイドライン（重要）

このスキルは非エンジニア向けに設計されている。以下を必ず守ること:

### MVPに徹する
- **まずは最小限で動くものを作る**ことを常に促す
- ユーザーが多くの機能を求めた場合: 「まずは○○だけで始めましょう。動くものを見てから拡張できます」
- 「その機能は後から `/coding` で追加できます。最初はシンプルに始めましょう」
- 複雑な構成を提案しない。迷ったらシンプルな方を選ぶ

### 分かりやすい言葉を使う
- 技術用語を避け、平易な日本語で質問する
- 「データベース」→「データの保存」、「ストレージ」→「ファイルのアップロード」
- 専門用語が必要な場合は括弧書きで補足する

### 「分からない」を歓迎する
- ユーザーが「分からない」「後で決める」と答えた場合、その項目を保留にする
- 保留にした項目はプロジェクトに含めず、後から追加できることを伝える
- 「分からなくて大丈夫です。後から簡単に追加できますよ」

## 処理フロー

### Step 1: 引数の処理とバリデーション

`$ARGUMENTS` からプロジェクト名を取得する。

**$ARGUMENTS が空の場合（GUI対話モード）:**
プロジェクト名が未指定なので、まず挨拶と概要の質問から始める。Step 2 の Q0 に進む。

**$ARGUMENTS がある場合（CLI モード）:**
- プロジェクト名が英数字+ハイフン以外を含む場合はエラー: 「プロジェクト名は英数字とハイフンのみ使用できます」
- 生成先 `~/<プロジェクト名>/` が既に存在する場合はエラー: 「ディレクトリが既に存在します」
- バリデーション通過後、Step 2 の Q1 に進む。

### Step 2: 対話で情報収集

AskUserQuestion を使って以下を順に質問する:

**Q0: 何を作りたいか（$ARGUMENTS が空の場合のみ）**
「こんにちは！ 新しいプロジェクトを作りましょう。何を作りたいですか？ 一言で教えてください。」
「例: 『予約管理システム』『社内の在庫管理ツール』『ブログサービス』など」
（自由入力。回答からプロジェクト名も提案する）

Q0 の回答を受けて:
- プロジェクト名を提案: 回答内容から英数字+ハイフンの名前を生成（例: 「予約管理」→ `reservation-app`）
- プロジェクトの説明として CLAUDE.md に記載する内容を決定

**Q1: どんなものを作りたいか（$ARGUMENTS がある場合のみ）**
「どんなWebアプリ/サービスを作りたいですか？ 一言で教えてください。」
「例: 『予約管理システム』『社内の在庫管理ツール』『ブログサービス』など」
（自由入力。ここで入力された内容がプロジェクトの説明としてCLAUDE.mdに記載される）

**プロジェクトタイプの判定（Q0/Q1 の回答後）**

ユーザーの回答から**プロジェクトタイプ**を自動判定する:

- **macOS アプリ**: 「macOS」「デスクトップアプリ」「ネイティブアプリ」「Mac アプリ」「メニューバーアプリ」「画面録画」「スクリーンショット」等のキーワードを含む場合 → **Swift macOS フロー**に進む
- **Web アプリ**: 上記に該当しない場合 → **Web フロー**（既存フロー）に進む

判断できない場合はユーザーに確認する:
「Web アプリと macOS ネイティブアプリ、どちらで作りますか？」

---

#### Web フロー（既存）

**Q2: MVP提案と承認**

ユーザーの回答（Q0 または Q1）を元に、以下を判断してMVP提案を行う:

1. **必要なサービスを自動判断**: ユーザーの説明から DB/ストレージ/キャッシュの要否を判断する
   - データの保存が必要そう → PostgreSQL を含める
   - ファイルアップロードが必要そう → ストレージを含める
   - 判断できない場合は含めない（後から追加可能）

2. **MVP機能を1つだけ提案**: 最も核となる機能を1つだけ選ぶ
   - 「予約管理システム」→ 「予約の一覧表示と新規登録」
   - 「社内の在庫管理ツール」→ 「商品の一覧表示と在庫数の登録」
   - 「ブログサービス」→ 「記事の一覧表示と新規投稿」
   - 複数機能を提案しない。**1つの機能だけ**に絞る

3. **提案を表示して承認を求める**:
```
まずは最小限の MVP を作りましょう！

プロジェクト名: <名前>
作るもの: <ユーザーの説明>

最初に実装する機能:
  <MVP機能の説明（1つだけ）>
  - 画面: <画面の説明>
  - データ: <扱うデータの説明>

必要な構成:
  - データベース: あり/なし
  - ファイル保存: あり/なし

生成先: ~/<名前>/

これで作成を始めてよいですか？
追加の機能は、完成後に `/coding` でいつでも追加できます。
```

**重要**:
- この Q2 が唯一の確認ポイント。承認されたら追加の確認は一切せず、すぐに Step 3 に進む
- 2回確認しない。1回だけ聞く
- ユーザーが「もっと機能を追加したい」と言った場合: 「まずはこの1つの機能だけで動くものを作りましょう。動くものを見てから追加する方が確実です」と促す

### Step 3: テンプレートコピー

```bash
# 生成先ディレクトリ作成
mkdir -p ~/<プロジェクト名>

# base テンプレートをコピー
cp -r ./templates/base/. ~/<プロジェクト名>/
```

PostgreSQL 選択時:
```bash
# Go ソース（registry, handler, infrastructure, model）を追加
cp -r ./templates/with-db/backend/internal/. ~/<プロジェクト名>/backend/internal/

# DB 初期化 SQL + docker-compose.yml を追加
cp -r ./templates/with-db/db/. ~/<プロジェクト名>/db/
cp ./templates/with-db/docker-compose.yml ~/<プロジェクト名>/docker-compose.yml

# フロントエンド /samples ページを追加
mkdir -p ~/<プロジェクト名>/frontend/src/app/samples
cp ./templates/with-db/frontend/src/app/samples/page.tsx ~/<プロジェクト名>/frontend/src/app/samples/page.tsx
```

ストレージ選択時:
```bash
# Go ソース（registry, handler, infrastructure）を追加
cp -r ./templates/with-storage/backend/internal/. ~/<プロジェクト名>/backend/internal/

# フロントエンド /storage ページを追加
mkdir -p ~/<プロジェクト名>/frontend/src/app/storage
cp ./templates/with-storage/frontend/src/app/storage/page.tsx ~/<プロジェクト名>/frontend/src/app/storage/page.tsx
```

docker-compose.yml（ストレージ選択時）:
- PostgreSQL 未選択時: MinIO 用 docker-compose.yml をコピー
```bash
cp ./templates/with-storage/docker-compose.yml ~/<プロジェクト名>/docker-compose.yml
```
- PostgreSQL 選択時（docker-compose.yml がコピー済み）: Edit ツールで MinIO サービスを追加する。追加する内容は `./templates/with-storage/docker-compose.yml` を参照し、services: に minio と minio-init を追加、volumes: に miniodata: を追加する。

Redis 選択時:
```bash
# Go ソース（registry, handler, infrastructure）を追加
cp -r ./templates/with-redis/backend/internal/. ~/<プロジェクト名>/backend/internal/

# フロントエンド /cache ページを追加
mkdir -p ~/<プロジェクト名>/frontend/src/app/cache
cp ./templates/with-redis/frontend/src/app/cache/page.tsx ~/<プロジェクト名>/frontend/src/app/cache/page.tsx
```

docker-compose.yml（Redis 選択時）:
- 他のオプション未選択時: Redis 用 docker-compose.yml をコピー
```bash
cp ./templates/with-redis/docker-compose.yml ~/<プロジェクト名>/docker-compose.yml
```
- 他のオプション選択済み（docker-compose.yml がコピー済み）: Edit ツールで Redis サービスを追加する。追加する内容は `./templates/with-redis/docker-compose.yml` を参照し、services: に redis を追加、volumes: に redisdata: を追加する。

**注意**: 各オプションは `internal/registry/` にファイルを追加する方式なので、複数オプションを同時に選択しても衝突しない。.env.example と go.mod は base のものをベースに、Step 5・6 で各オプションの変数・依存を追記する。

### Step 4: ポート割り当て + プレースホルダー置換

#### 4.1 ランダムポートの生成

下3桁を共通のランダム値（100〜999）で生成し、先頭桁でサービスを識別する。

```bash
# ランダムな下3桁を生成（100〜999）
PORT_SUFFIX=$((RANDOM % 900 + 100))
PORT_BACKEND=8${PORT_SUFFIX}
PORT_FRONTEND=3${PORT_SUFFIX}
PORT_DB=5${PORT_SUFFIX}
PORT_MINIO=9${PORT_SUFFIX}
PORT_MINIO_CONSOLE=$((PORT_MINIO + 1))
PORT_REDIS=6${PORT_SUFFIX}
```

予約済みポートとの衝突チェック（8080, 8888, 3000, 3333, 5432, 9000, 9001, 6379 に該当したら再生成）。
使用中ポートのチェック（`lsof -ti:${PORT_BACKEND}` 等で確認、使用中なら再生成）。

#### 4.2 プレースホルダー一括置換

`{{PROJECT_NAME}}` と `{{PORT_xxx}}` を一括置換する。

**重要**: バイナリファイル破損を防ぐため、対象はテキストファイル拡張子のみに限定する。

```bash
cd ~/<プロジェクト名>
find . -type f \( \
  -name "*.go" -o -name "*.mod" -o -name "*.json" -o -name "*.tsx" -o -name "*.ts" \
  -o -name "*.css" -o -name "*.yml" -o -name "*.yaml" -o -name "*.md" \
  -o -name "*.mjs" -o -name "*.sql" -o -name "Makefile" \
  -o -name "Dockerfile" -o -name ".gitignore" -o -name ".env*" \
\) -exec sed -i '' \
  -e "s/{{PROJECT_NAME}}/<プロジェクト名>/g" \
  -e "s/{{PORT_BACKEND}}/${PORT_BACKEND}/g" \
  -e "s/{{PORT_FRONTEND}}/${PORT_FRONTEND}/g" \
  -e "s/{{PORT_DB}}/${PORT_DB}/g" \
  -e "s/{{PORT_MINIO_CONSOLE}}/${PORT_MINIO_CONSOLE}/g" \
  -e "s/{{PORT_MINIO}}/${PORT_MINIO}/g" \
  -e "s/{{PORT_REDIS}}/${PORT_REDIS}/g" \
  {} +
```

**注意**: `{{PORT_MINIO_CONSOLE}}` を `{{PORT_MINIO}}` より先に置換すること（部分マッチ防止）。

### Step 5: .env 作成

base の `.env.example` に選択したオプションの環境変数を追記し、`.env` にコピーする。
ポート番号は Step 4 で置換済みの変数を使用する。

PostgreSQL 選択時:
```bash
echo "DATABASE_URL=postgres://postgres:postgres@localhost:${PORT_DB}/<プロジェクト名>?sslmode=disable" >> ~/<プロジェクト名>/backend/.env.example
```

ストレージ選択時:
```bash
echo "STORAGE_ENDPOINT=http://localhost:${PORT_MINIO}" >> ~/<プロジェクト名>/backend/.env.example
echo 'R2_ACCOUNT_ID=' >> ~/<プロジェクト名>/backend/.env.example
echo 'R2_ACCESS_KEY_ID=minioadmin' >> ~/<プロジェクト名>/backend/.env.example
echo 'R2_ACCESS_KEY_SECRET=minioadmin' >> ~/<プロジェクト名>/backend/.env.example
echo 'R2_BUCKET_NAME=uploads' >> ~/<プロジェクト名>/backend/.env.example
```

Redis 選択時:
```bash
echo "REDIS_URL=redis://localhost:${PORT_REDIS}" >> ~/<プロジェクト名>/backend/.env.example
```

最後に `.env` にコピー:
```bash
cp ~/<プロジェクト名>/backend/.env.example ~/<プロジェクト名>/backend/.env
```

### Step 6: 依存関係の解決

```bash
cd ~/<プロジェクト名>/backend
```

PostgreSQL 選択時:
```bash
go get gorm.io/gorm@v1.25.12 gorm.io/driver/postgres@v1.5.11
```

ストレージ選択時:
```bash
go get github.com/aws/aws-sdk-go-v2@latest github.com/aws/aws-sdk-go-v2/config@latest github.com/aws/aws-sdk-go-v2/credentials@latest github.com/aws/aws-sdk-go-v2/service/s3@latest
```

Redis 選択時:
```bash
go get github.com/redis/go-redis/v9@latest
```

共通:
```bash
cd ~/<プロジェクト名>/backend && go mod tidy
cd ~/<プロジェクト名>/frontend && npm install
```

### Step 7: .claude/ 資産の生成

Ghostrunnerの `.claude/` 資産を一括コピーし、CLAUDE.md だけ新プロジェクト用に生成する。

#### 7.1 一括コピー

```bash
mkdir -p ~/<プロジェクト名>/.claude/agents

# agents/ を一括コピー
cp ./.claude/agents/*.md ~/<プロジェクト名>/.claude/agents/

# skills/ を一括コピー
cp -r ./.claude/skills/ ~/<プロジェクト名>/.claude/skills/

# settings.json をコピー
cp ./.claude/settings.json ~/<プロジェクト名>/.claude/settings.json

# settings.local.json を動的生成（環境依存のパスを解決）
HOME_DIR=$(eval echo ~)
cat > ~/<プロジェクト名>/.claude/settings.local.json << SETTINGS_EOF
{
  "permissions": {
    "allow": ["Bash(*)", "Edit", "Write", "WebFetch(*)", "Skill(*)", "Read(*)"],
    "additionalDirectories": ["/tmp", "${HOME_DIR}"]
  }
}
SETTINGS_EOF
```

未選択のオプションに対応するエージェントを削除:

PostgreSQL 未選択時:
```bash
rm -f ~/<プロジェクト名>/.claude/agents/pg-*.md
```

ストレージ未選択時:
```bash
rm -f ~/<プロジェクト名>/.claude/agents/storage-*.md
```

Redis 未選択時:
```bash
rm -f ~/<プロジェクト名>/.claude/agents/redis-*.md
```

#### 7.2 CLAUDE.md 生成

Ghostrunnerの `.claude/CLAUDE.md` (`./.claude/CLAUDE.md`) の構造を参考に、新プロジェクト用に生成する。

含めるセクション:
- **プロジェクト概要**: ユーザーが入力した概要を反映。技術スタック（Go + Gin, Next.js, Tailwind CSS）を記載。DB選択時は PostgreSQL + GORM も記載
- **Backend (Go)**: コード構成、コードスタイル、エラーハンドリング、テスト、ファイル構造、ビルド・実行コマンド
- **Frontend (Next.js)**: 技術スタック、コード構成、コードスタイル、テスト、ファイル構造、ビルド・実行コマンド
- **共通ルール**: セキュリティ、Gitワークフロー（日本語コミットメッセージ）、Makefileコマンド

### Step 7.3: README.md 更新

選択したサービスに応じて、README.md の「実装済みの機能」セクションに追記する。Edit ツールで追記すること。

PostgreSQL 選択時に追記:
- ページ表に `| http://localhost:${PORT_FRONTEND}/samples | DB サンプル（CRUD操作） |` を追加
- API表に以下を追加:
  - `| GET | /api/samples | サンプル一覧取得 |`
  - `| POST | /api/samples | サンプル作成 |`
- 技術スタックに `PostgreSQL + GORM` を追加

ストレージ選択時に追記:
- ページ表に `| http://localhost:${PORT_FRONTEND}/storage | ファイルアップロード |` を追加
- API表に以下を追加:
  - `| GET | /api/storage/files | ファイル一覧取得 |`
  - `| POST | /api/storage/upload | ファイルアップロード |`
- 技術スタックに `Cloudflare R2 / MinIO` を追加

Redis 選択時に追記:
- ページ表に `| http://localhost:${PORT_FRONTEND}/cache | キャッシュ操作 |` を追加
- API表に以下を追加:
  - `| GET | /api/cache/:key | キャッシュ取得 |`
  - `| POST | /api/cache | キャッシュ設定 |`
- 技術スタックに `Redis` を追加

### Step 7.5: devtools シンボリックリンク作成

devtools（進捗ビューア）へのシンボリックリンクを作成する。

シンボリックリンクを作成する:
```bash
ln -s ./devtools ~/<プロジェクト名>/.devtools
```

`.gitignore` に `.devtools` を追加する:

```bash
echo '.devtools' >> ~/<プロジェクト名>/.gitignore
```

### Step 7.6: 開発フォルダ構成の作成

プロジェクトの開発ドキュメント用フォルダ構成を作成する:

```bash
mkdir -p ~/<プロジェクト名>/開発/検討中/アーカイブ
mkdir -p ~/<プロジェクト名>/開発/実装/実装待ち
mkdir -p ~/<プロジェクト名>/開発/実装/完了/アーカイブ
mkdir -p ~/<プロジェクト名>/開発/資料/アーカイブ
```

各末端ディレクトリに `.gitkeep` を配置:

```bash
touch ~/<プロジェクト名>/開発/検討中/アーカイブ/.gitkeep
touch ~/<プロジェクト名>/開発/実装/実装待ち/.gitkeep
touch ~/<プロジェクト名>/開発/実装/完了/アーカイブ/.gitkeep
touch ~/<プロジェクト名>/開発/資料/アーカイブ/.gitkeep
```

### Step 8: Git 初期化

```bash
cd ~/<プロジェクト名>
git init
git add -A
git commit -m "feat: プロジェクト初期化 - Go + Next.js フルスタック構成"
```

### Step 9: ポート確保と起動（PostgreSQL またはストレージまたは Redis 選択時）

PostgreSQL、ストレージ、Redis のいずれかを選択した場合:

#### 9.1 ポート確保

使用するポートが使用中か確認し、使用中の場合はユーザーに確認して停止する。

- PostgreSQL 選択時: ${PORT_DB}, ${PORT_BACKEND}
- ストレージ選択時: ${PORT_MINIO}, ${PORT_MINIO_CONSOLE}, ${PORT_BACKEND}
- Redis 選択時: ${PORT_REDIS}, ${PORT_BACKEND}
- 複数選択時: 選択したサービスのポートを全て含める

```bash
lsof -ti:${PORT_DB}     # PostgreSQL 選択時
lsof -ti:${PORT_MINIO}  # ストレージ選択時
lsof -ti:${PORT_REDIS}  # Redis 選択時
lsof -ti:${PORT_BACKEND}
```

プロセスが存在する場合:
1. ユーザーに確認する: 「ポート XXXX が既に使用されています。停止してよいですか？」
2. 承認された場合: 該当ポートのプロセスを停止
3. 拒否された場合: docker-compose.yml のポートを変更し、必要に応じて `.env` も調整する

#### 9.2 Docker サービス起動

```bash
cd ~/<プロジェクト名>

# 選択したサービスのみ起動（例）
docker-compose up -d db              # PostgreSQL 選択時
docker-compose up -d minio minio-init # ストレージ選択時
docker-compose up -d redis            # Redis 選択時

# 複数選択時は全てを含める
docker-compose up -d db minio minio-init redis
```

サービスが ready になるまで待機:

PostgreSQL 選択時:
```bash
docker exec <プロジェクト名>-db pg_isready -U postgres
```

ストレージ選択時:
```bash
# MinIO の起動を待機（minio-init がバケット作成を完了するまで）
sleep 5
curl -s http://localhost:${PORT_MINIO}/minio/health/live
```

Redis 選択時:
```bash
docker exec <プロジェクト名>-redis redis-cli ping
```

#### 9.3 バックエンド起動

```bash
cd ~/<プロジェクト名>
nohup sh -c 'cd backend && set -a && . ./.env && set +a && go run ./cmd/server' > /tmp/<プロジェクト名>-backend.log 2>&1 &
```

起動ログを確認し、「Listening on」が出ることを確認する。PostgreSQL 選択時は「Database migration completed」、ストレージ選択時は「Storage initialized」、Redis 選択時は「Redis connected」も確認する。

#### 9.4 フロントエンド起動

```bash
cd ~/<プロジェクト名>
nohup sh -c 'cd frontend && npm run dev' > /tmp/<プロジェクト名>-frontend.log 2>&1 &
```

起動ログを確認し、「Ready」が出ることを確認する。

#### 9.5 動作確認

```bash
# ヘルスチェック
curl -s http://localhost:${PORT_BACKEND}/api/health
```

PostgreSQL 選択時:
```bash
# DB書き込みテスト
curl -s -X POST http://localhost:${PORT_BACKEND}/api/samples \
  -H "Content-Type: application/json" \
  -d '{"name":"Hello","description":"Initial sample"}'

# 読み取り確認
curl -s http://localhost:${PORT_BACKEND}/api/samples
```

ストレージ選択時:
```bash
# ファイルアップロードテスト
echo "test" > /tmp/test-upload.txt
curl -s -X POST http://localhost:${PORT_BACKEND}/api/storage/upload -F "file=@/tmp/test-upload.txt"

# ファイル一覧確認
curl -s http://localhost:${PORT_BACKEND}/api/storage/files
rm /tmp/test-upload.txt
```

Redis 選択時:
```bash
# キャッシュ書き込みテスト
curl -s -X POST http://localhost:${PORT_BACKEND}/api/cache \
  -H "Content-Type: application/json" \
  -d '{"key":"hello","value":"world","ttl_seconds":60}'

# 読み取り確認
curl -s http://localhost:${PORT_BACKEND}/api/cache/hello
```

### Step 10: 起動（PostgreSQL もストレージも Redis も未選択時）

PostgreSQL もストレージも Redis も選択しなかった場合:

#### 10.1 ポート確保

ポート ${PORT_BACKEND} が使用中か確認し、使用中の場合はユーザーに確認して停止する。

#### 10.2 バックエンド起動

```bash
cd ~/<プロジェクト名>
nohup sh -c 'cd backend && set -a && . ./.env && set +a && go run ./cmd/server' > /tmp/<プロジェクト名>-backend.log 2>&1 &
```

#### 10.3 フロントエンド起動

```bash
cd ~/<プロジェクト名>
nohup sh -c 'cd frontend && npm run dev' > /tmp/<プロジェクト名>-frontend.log 2>&1 &
```

#### 10.4 動作確認

```bash
curl -s http://localhost:${PORT_BACKEND}/api/health
curl -s http://localhost:${PORT_FRONTEND} > /dev/null && echo "Frontend: OK"
```

### Step 11: 環境構築完了の中間報告

以下を表示する:

```
プロジェクトの土台ができました。これからMVP機能を実装します...
```

### Step 12: MVP機能の実装（/plan → /coding）

Step 2 の Q2 で承認されたMVP機能を、既存の `/plan` と `/coding` スキルを順に実行して実装する。

#### 12.1 /plan の実行

プロジェクトディレクトリ `~/<プロジェクト名>/` で `/plan` を実行する。

`/plan` への入力として、Step 2 の対話で決まった以下の情報を渡す:
- 作りたいもの（ユーザーの説明）
- MVP機能（1つだけ）
- 必要なサービス構成（DB/ストレージ/キャッシュ）

`/plan` が実装計画書を `開発/実装/実装待ち/` に生成する。

#### 12.2 /coding の実行

計画書が生成されたら、同じプロジェクトディレクトリで `/coding` を実行する。

`/coding` が以下の工程を自動で実行する:
- バックエンド: impl → reviewer → tester → documenter → コミット
- フロントエンド: impl → reviewer → tester → documenter → コミット

#### 12.3 動作確認

`/coding` 完了後、サーバーを起動して動作確認する:

```bash
cd ~/<プロジェクト名>
make dev
```

### Step 13: GETTING_STARTED.md の生成

プロジェクトディレクトリに `GETTING_STARTED.md` を生成する。
固定部分はそのまま記載し、「次に追加してみましょう」セクションはプロジェクトの内容に合わせて AI が3つの具体的な提案を生成する。

```markdown
# はじめに

プロジェクト「<プロジェクト名>」が作成されました。

## サーバーの起動

```bash
make dev
```

- フロントエンド: http://localhost:<PORT_FRONTEND>
- バックエンド API: http://localhost:<PORT_BACKEND>/api/health

サーバーを停止するには:

```bash
make stop
```

## 実装済みの機能

- <MVP機能の説明>

## 次に追加してみましょう

Claude Code を開いて、以下のようにやりたいことを伝えてみてください:

```
/discuss <プロジェクト内容に合わせた具体的な提案1。何を、どのように、どんな表示にするかまで具体的に書く>
```

```
/discuss <プロジェクト内容に合わせた具体的な提案2。何を、どのように、どんな表示にするかまで具体的に書く>
```

```
/discuss <プロジェクト内容に合わせた具体的な提案3。何を、どのように、どんな表示にするかまで具体的に書く>
```

やりたいことを自由に伝えるだけで、計画から実装まで全て行います。

## よく使うコマンド

| コマンド | 説明 |
|---------|------|
| `make dev` | サーバー起動 |
| `make stop` | サーバー停止 |
| `/discuss` | アイデアを相談する |
| `/plan` | 実装計画を作成する |
| `/fullstack` | フルスタック実装 |
| `/update` | Ghostrunner を最新化する |
```

**提案の書き方ルール:**
- 各提案は1〜2文で、何をどうしたいか具体的に書く
- 技術用語を避け、ユーザー目線で書く
- プロジェクトの内容から自然に想像される次の機能を提案する
- 例: 予約管理システムの場合:
  - `/discuss ユーザーがGoogleアカウントでログインできるようにしたい。ログイン後は自分の予約だけが見えるようにする`
  - `/discuss 予約の一覧をカレンダー形式で表示したい。月表示と週表示を切り替えられて、予約がある日にはマークが付くようにする`
  - `/discuss 予約が確定したらユーザーにメールで通知を送りたい。予約日時と内容と場所を含めた確認メールを自動送信する`

### Step 14: 完了メッセージ

以下を表示する:

```
プロジェクト「<プロジェクト名>」の作成が完了しました！

生成先: ~/<プロジェクト名>/

実装した機能:
  <MVP機能の説明>

アクセス:
  フロントエンド: http://localhost:${PORT_FRONTEND}
  バックエンド API: http://localhost:${PORT_BACKEND}/api/health

GETTING_STARTED.md に次のステップが書いてあります。
```

### Step 15: 本番デプロイ準備（PostgreSQL またはストレージまたは Redis 選択時）

PostgreSQL、ストレージ、Redis のいずれかを選択した場合、本番環境のセットアップを提案する。

AskUserQuestion で確認:
「本番デプロイの準備（GCP + Neon / R2 / Upstash）を行いますか？」
- 選択肢: はい / スキップ（後で手動で設定する）

**「スキップ」の場合**: Step 14 を終了する。

#### 15.1 gcloud CLI 確認・インストール

```bash
which gcloud
```

未インストールの場合:
```bash
brew install --cask google-cloud-sdk
```

#### 15.2 GCP 認証

```bash
gcloud auth list 2>&1
```

アクティブアカウントがない場合:
ユーザーに案内: 「GCP にログインします。ブラウザが開きます。」
```bash
gcloud auth login
```

#### 15.3 GCP プロジェクト選択

```bash
gcloud projects list --format="table(projectId,name)" 2>&1
```

AskUserQuestion で確認:
「使用する GCP プロジェクトを選択してください」
- プロジェクト一覧から選択肢を動的に生成
- 追加の選択肢: 「新規作成」

**「新規作成」の場合**:
AskUserQuestion でプロジェクトIDを入力させる。
```bash
gcloud projects create <プロジェクトID>
gcloud billing projects link <プロジェクトID> --billing-account=<請求先アカウントID>
```
- 請求先アカウントは `gcloud billing accounts list` で取得

**選択後**:
```bash
gcloud config set project <プロジェクトID>
```

#### 15.4 GCP API 有効化

```bash
gcloud services enable \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com
```

#### 15.5 GCP サービスアカウント作成

GitHub Actions からデプロイするためのサービスアカウントを作成する:
```bash
GCP_PROJECT=$(gcloud config get-value project)

# サービスアカウント作成
gcloud iam service-accounts create <プロジェクト名>-deployer \
  --display-name="<プロジェクト名> GitHub Actions Deployer"

# 必要なロールを付与
SA_EMAIL="<プロジェクト名>-deployer@${GCP_PROJECT}.iam.gserviceaccount.com"
gcloud projects add-iam-policy-binding $GCP_PROJECT --member="serviceAccount:${SA_EMAIL}" --role="roles/run.admin"
gcloud projects add-iam-policy-binding $GCP_PROJECT --member="serviceAccount:${SA_EMAIL}" --role="roles/storage.admin"
gcloud projects add-iam-policy-binding $GCP_PROJECT --member="serviceAccount:${SA_EMAIL}" --role="roles/iam.serviceAccountUser"
gcloud projects add-iam-policy-binding $GCP_PROJECT --member="serviceAccount:${SA_EMAIL}" --role="roles/secretmanager.secretAccessor"

# キーを生成
gcloud iam service-accounts keys create /tmp/<プロジェクト名>-sa-key.json --iam-account=$SA_EMAIL
```

#### 15.6 GitHub リポジトリ作成・Secrets 登録

`gh` CLI の確認:
```bash
which gh
```

未インストールの場合:
```bash
brew install gh
gh auth login
```

リポジトリ作成・push:
```bash
cd ~/<プロジェクト名>
gh repo create <プロジェクト名> --private --source=. --push
```

GitHub Secrets（リポジトリレベル、全環境共通）を登録:
```bash
gh secret set GCP_SA_KEY < /tmp/<プロジェクト名>-sa-key.json
gh secret set GCP_PROJECT_ID --body="$GCP_PROJECT"

# ローカルのキーファイルを削除
rm /tmp/<プロジェクト名>-sa-key.json
```

#### 15.7 GitHub Environments 作成

production 環境を作成し、Variables を登録する:
```bash
# production 環境の Variables（初回デプロイ後に実際の URL に更新する）
gh variable set FRONTEND_URL --env production --body="https://<プロジェクト名>-frontend-xxxxxxxxxx-an.a.run.app"
gh variable set BACKEND_URL --env production --body="https://<プロジェクト名>-backend-xxxxxxxxxx-an.a.run.app"
```

注: `gh variable set --env <name>` を実行すると、GitHub Environment が自動作成される。

**以下の 12.8〜12.12 は PostgreSQL 選択時のみ実行する。**

#### 12.8 Neon CLI 確認・インストール

```bash
which neonctl
```

未インストールの場合:
```bash
brew install neonctl
```

#### 12.9 Neon ログイン確認

```bash
neonctl projects list 2>&1
```

未ログインの場合（エラーが出た場合）:
ユーザーに案内: 「Neon にログインします。ブラウザが開きます。」
```bash
neonctl auth
```

#### 15.10 Neon プロジェクト作成

AskUserQuestion で確認:
「Neon プロジェクトを新規作成しますか？」
- 選択肢: 新規作成 / 既存を使う

**「新規作成」の場合**:
```bash
neonctl projects create --name <プロジェクト名> --region-id aws-ap-northeast-1
```

**「既存を使う」の場合**:
`neonctl projects list` の結果を表示し、使用するプロジェクトを選択させる。

#### 15.11 スキーマ反映

```bash
PROD_CONNSTR=$(neonctl connection-string --project-id <プロジェクトID>)
psql "$PROD_CONNSTR" -f ~/<プロジェクト名>/db/init.sql
```

#### 15.12 Secret Manager に DATABASE_URL を登録

```bash
echo -n "$PROD_CONNSTR" | gcloud secrets create DATABASE_URL --data-file=-
```

注: サービスアカウントへの `roles/secretmanager.secretAccessor` は 12.5 で付与済み。

#### 15.13 Secret Manager に R2 クレデンシャルを登録（ストレージ選択時）

ストレージを選択した場合のみ実行する。

AskUserQuestion で確認:
「Cloudflare R2 のクレデンシャルを登録しますか？（R2 バケットと API トークンの事前作成が必要です）」
- 選択肢: はい / 後で設定する

**「はい」の場合**:
R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_ACCESS_KEY_SECRET, R2_BUCKET_NAME をそれぞれ質問し、Secret Manager に登録する:

```bash
echo -n "<値>" | gcloud secrets create R2_ACCOUNT_ID --data-file=-
echo -n "<値>" | gcloud secrets create R2_ACCESS_KEY_ID --data-file=-
echo -n "<値>" | gcloud secrets create R2_ACCESS_KEY_SECRET --data-file=-
echo -n "<値>" | gcloud secrets create R2_BUCKET_NAME --data-file=-
```

**「後で設定する」の場合**:
完了メッセージに Secret Manager 登録コマンドを含める。

**以下の 12.14〜12.15 は Redis 選択時のみ実行する。**

#### 15.14 Upstash CLI 確認・インストール

```bash
which upstash
```

未インストールの場合:
```bash
npm install -g @upstash/cli
```

ログイン確認:
```bash
upstash redis list 2>&1
```

未ログインの場合:
ユーザーに案内: 「Upstash にログインします。」
```bash
upstash auth login
```

#### 15.15 Upstash Redis 作成・Secret Manager 登録

AskUserQuestion で確認:
「Upstash Redis を新規作成しますか？」
- 選択肢: 新規作成 / 後で設定する

**「新規作成」の場合**:
```bash
upstash redis create --name <プロジェクト名> --region ap-northeast-1
```

接続文字列（`rediss://default:xxx@xxx.upstash.io:xxx`）を `upstash redis list` で取得し、Secret Manager に登録:

```bash
echo -n "$PROD_REDIS_URL" | gcloud secrets create REDIS_URL --data-file=-
```

**「後で設定する」の場合**:
完了メッセージに Secret Manager 登録コマンドを含める。

#### 15.16 deploy.yml に Secret Manager 参照を追加

生成したプロジェクトの `.github/workflows/deploy.yml` を Edit ツールで編集し、backend の `gcloud run deploy` コマンドに `--set-secrets` を追加する。

PostgreSQL 選択時に追加する行:
```
--set-secrets "DATABASE_URL=DATABASE_URL:latest"
```

ストレージ選択時に追加する行:
```
--set-secrets "R2_ACCOUNT_ID=R2_ACCOUNT_ID:latest,R2_ACCESS_KEY_ID=R2_ACCESS_KEY_ID:latest,R2_ACCESS_KEY_SECRET=R2_ACCESS_KEY_SECRET:latest,R2_BUCKET_NAME=R2_BUCKET_NAME:latest"
```

Redis 選択時に追加する行:
```
--set-secrets "REDIS_URL=REDIS_URL:latest"
```

複数選択時は1つの `--set-secrets` にカンマ区切りでまとめる。

追加位置: backend deploy ステップの `--set-env-vars` の行の直前に `\` で行を継続して挿入する。

#### 15.17 デプロイ準備完了メッセージ

```
本番デプロイ準備が完了しました！

GCP プロジェクト: <プロジェクトID>
GitHub: https://github.com/<ユーザー名>/<プロジェクト名>

デプロイフロー:
  main ブランチに push → production 環境に自動デプロイ
```

PostgreSQL 選択時は追加で表示:
```
Neon:
  neonctl projects list で確認

Secret Manager (DB):
  DATABASE_URL: 接続文字列
```

ストレージ選択時は追加で表示:
```
Secret Manager (R2):
  R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_ACCESS_KEY_SECRET, R2_BUCKET_NAME
```

Redis 選択時は追加で表示:
```
Upstash:
  upstash redis list で確認

Secret Manager (Redis):
  REDIS_URL
```

共通:
```
注意:
  初回デプロイ後、Cloud Run URL が確定したら
  GitHub Environments の FRONTEND_URL と BACKEND_URL を実際の URL に更新してください:
    gh variable set FRONTEND_URL --env production --body="https://実際のURL"
    gh variable set BACKEND_URL --env production --body="https://実際のURL"
```

---

#### macOS フロー（Swift）

プロジェクトタイプが macOS アプリと判定された場合、以下のフローを実行する。
Web フローの Step 3〜15 は実行しない。

**Q2-mac: MVP提案と承認**

ユーザーの回答（Q0 または Q1）を元に MVP 提案を行う:

1. **MVP機能を1つだけ提案**: 最も核となる機能を1つだけ選ぶ
   - Docker / ポート / DB の質問はスキップ
   - 必要な権限（画面録画、カメラ等）があれば言及する

2. **提案を表示して承認を求める**:
```
まずは最小限の MVP を作りましょう！

プロジェクト名: <名前>
作るもの: <ユーザーの説明>
種類: macOS ネイティブアプリ（Swift + SwiftUI）

最初に実装する機能:
  <MVP機能の説明（1つだけ）>

生成先: ~/<名前>/

これで作成を始めてよいですか？
追加の機能は、完成後に `/coding` でいつでも追加できます。
```

**重要**: 承認後、追加の確認は一切せず、すぐに Step M3 に進む。

### Step M3: テンプレートコピー

```bash
mkdir -p ~/<プロジェクト名>
cp -r ./templates/swift-macos/. ~/<プロジェクト名>/
```

### Step M4: プレースホルダー置換

```bash
cd ~/<プロジェクト名>
find . -type f \( \
  -name "*.swift" -o -name "*.md" -o -name "Makefile" \
  -o -name ".gitignore" -o -name "Package.swift" \
\) -exec sed -i '' \
  -e "s/{{PROJECT_NAME}}/<プロジェクト名>/g" \
  {} +
```

App エントリーポイントのファイル名をリネーム:
```bash
mv ~/<プロジェクト名>/Sources/App/'{{PROJECT_NAME}}App.swift' ~/<プロジェクト名>/Sources/App/'<プロジェクト名>App.swift'
```

### Step M5: .claude/ 資産の生成

```bash
mkdir -p ~/<プロジェクト名>/.claude/agents

# agents/ を一括コピー
cp ./.claude/agents/*.md ~/<プロジェクト名>/.claude/agents/

# skills/ を一括コピー
cp -r ./.claude/skills/ ~/<プロジェクト名>/.claude/skills/

# settings.json をコピー
cp ./.claude/settings.json ~/<プロジェクト名>/.claude/settings.json

# settings.local.json を動的生成
HOME_DIR=$(eval echo ~)
cat > ~/<プロジェクト名>/.claude/settings.local.json << SETTINGS_EOF
{
  "permissions": {
    "allow": ["Bash(*)", "Edit", "Write", "WebFetch(*)", "Skill(*)", "Read(*)"],
    "additionalDirectories": ["/tmp", "${HOME_DIR}"]
  }
}
SETTINGS_EOF
```

Web アプリ用エージェントを削除:
```bash
ls ~/<プロジェクト名>/.claude/agents/go-*.md ~/<プロジェクト名>/.claude/agents/nextjs-*.md ~/<プロジェクト名>/.claude/agents/pg-*.md ~/<プロジェクト名>/.claude/agents/storage-*.md ~/<プロジェクト名>/.claude/agents/redis-*.md ~/<プロジェクト名>/.claude/agents/staging-manager.md ~/<プロジェクト名>/.claude/agents/release-manager.md 2>/dev/null && rm -f ~/<プロジェクト名>/.claude/agents/go-*.md ~/<プロジェクト名>/.claude/agents/nextjs-*.md ~/<プロジェクト名>/.claude/agents/pg-*.md ~/<プロジェクト名>/.claude/agents/storage-*.md ~/<プロジェクト名>/.claude/agents/redis-*.md ~/<プロジェクト名>/.claude/agents/staging-manager.md ~/<プロジェクト名>/.claude/agents/release-manager.md
```

CLAUDE.md を Swift/SwiftUI 版で生成する:

含めるセクション:
- **プロジェクト概要**: ユーザーが入力した概要を反映。技術スタック（Swift 6, SwiftUI, macOS 14+, SPM）を記載
- **Swift macOS アプリ**: コード構成（Clean Architecture）、コードスタイル（Protocol Oriented, value type 優先）、エラーハンドリング（throws, Optional 安全性）、Concurrency（Swift 6 Strict Concurrency, @MainActor）、テスト、ファイル構造、ビルド・実行コマンド
- **共通ルール**: セキュリティ、Gitワークフロー（日本語コミットメッセージ）、Makefileコマンド

### Step M5.5: devtools シンボリックリンク作成

```bash
ln -s ./devtools ~/<プロジェクト名>/.devtools
```

`.gitignore` に `.devtools` を追加:
```bash
echo '.devtools' >> ~/<プロジェクト名>/.gitignore
```

### Step M6: Git 初期化

```bash
cd ~/<プロジェクト名>
git init
git add -A
git commit -m "feat: プロジェクト初期化 - Swift + SwiftUI macOS アプリ構成"
```

### Step M7: ビルド確認

```bash
cd ~/<プロジェクト名>
swift build
```

ビルドが失敗した場合はエラーを修正して再試行する。

### Step M8: 環境構築完了の中間報告

```
プロジェクトの土台ができました。これからMVP機能を実装します...
```

### Step M9: MVP機能の実装（/plan + /coding）

Step 2 の Q2-mac で承認されたMVP機能を実装する。

#### M9.1 /plan の実行

プロジェクトディレクトリ `~/<プロジェクト名>/` で `/plan` を実行する。
`/plan` への入力として、Step 2 の対話で決まった情報を渡す。

#### M9.2 /coding の実行

計画書が生成されたら、`/coding` を実行する。
macOS アプリの場合、Swift ワークフロー（swift-impl → swift-reviewer → swift-tester → swift-documenter → コミット）が実行される。

### Step M10: GETTING_STARTED.md の生成

```markdown
# はじめに

プロジェクト「<プロジェクト名>」が作成されました。

## ビルドと実行

```bash
make build   # ビルド
make run     # ビルドして実行
```

## 実装済みの機能

- <MVP機能の説明>

## 次に追加してみましょう

Claude Code を開いて、以下のようにやりたいことを伝えてみてください:

```
/discuss <プロジェクト内容に合わせた具体的な提案1>
```

```
/discuss <プロジェクト内容に合わせた具体的な提案2>
```

```
/discuss <プロジェクト内容に合わせた具体的な提案3>
```

やりたいことを自由に伝えるだけで、計画から実装まで全て行います。

## よく使うコマンド

| コマンド | 説明 |
|---------|------|
| `make build` | ビルド |
| `make run` | ビルドして実行 |
| `make test` | テスト実行 |
| `/discuss` | アイデアを相談する |
| `/plan` | 実装計画を作成する |
| `/coding` | 実装する |
| `/update` | Ghostrunner を最新化する |
```

### Step M11: 完了メッセージ

```
プロジェクト「<プロジェクト名>」の作成が完了しました！

生成先: ~/<プロジェクト名>/

実装した機能:
  <MVP機能の説明>

ビルド・実行:
  make build   # ビルド
  make run     # ビルドして実行

GETTING_STARTED.md に次のステップが書いてあります。
```
