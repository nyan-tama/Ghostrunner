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
- 「その機能は後から `/fullstack` で追加できます。最初はシンプルに始めましょう」
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
- 生成先 `/Users/user/<プロジェクト名>/` が既に存在する場合はエラー: 「ディレクトリが既に存在します」
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

生成先: /Users/user/<名前>/

これで作成を始めてよいですか？
追加の機能は、完成後に `/fullstack` でいつでも追加できます。
```

**重要**: ユーザーが「もっと機能を追加したい」と言った場合:
- 「まずはこの1つの機能だけで動くものを作りましょう。動くものを見てから追加する方が確実です」と促す
- それでも追加を求める場合は、最大でもう1つだけ追加を許容する

### Step 3: テンプレートコピー

```bash
# 生成先ディレクトリ作成
mkdir -p /Users/user/<プロジェクト名>

# base テンプレートをコピー
cp -r /Users/user/Ghostrunner/templates/base/. /Users/user/<プロジェクト名>/
```

PostgreSQL 選択時:
```bash
# Go ソース（registry, handler, infrastructure, model）を追加
cp -r /Users/user/Ghostrunner/templates/with-db/backend/internal/. /Users/user/<プロジェクト名>/backend/internal/

# DB 初期化 SQL + docker-compose.yml を追加
cp -r /Users/user/Ghostrunner/templates/with-db/db/. /Users/user/<プロジェクト名>/db/
cp /Users/user/Ghostrunner/templates/with-db/docker-compose.yml /Users/user/<プロジェクト名>/docker-compose.yml

# フロントエンド /samples ページを追加
mkdir -p /Users/user/<プロジェクト名>/frontend/src/app/samples
cp /Users/user/Ghostrunner/templates/with-db/frontend/src/app/samples/page.tsx /Users/user/<プロジェクト名>/frontend/src/app/samples/page.tsx
```

ストレージ選択時:
```bash
# Go ソース（registry, handler, infrastructure）を追加
cp -r /Users/user/Ghostrunner/templates/with-storage/backend/internal/. /Users/user/<プロジェクト名>/backend/internal/

# フロントエンド /storage ページを追加
mkdir -p /Users/user/<プロジェクト名>/frontend/src/app/storage
cp /Users/user/Ghostrunner/templates/with-storage/frontend/src/app/storage/page.tsx /Users/user/<プロジェクト名>/frontend/src/app/storage/page.tsx
```

docker-compose.yml（ストレージ選択時）:
- PostgreSQL 未選択時: MinIO 用 docker-compose.yml をコピー
```bash
cp /Users/user/Ghostrunner/templates/with-storage/docker-compose.yml /Users/user/<プロジェクト名>/docker-compose.yml
```
- PostgreSQL 選択時（docker-compose.yml がコピー済み）: Edit ツールで MinIO サービスを追加する。追加する内容は `/Users/user/Ghostrunner/templates/with-storage/docker-compose.yml` を参照し、services: に minio と minio-init を追加、volumes: に miniodata: を追加する。

Redis 選択時:
```bash
# Go ソース（registry, handler, infrastructure）を追加
cp -r /Users/user/Ghostrunner/templates/with-redis/backend/internal/. /Users/user/<プロジェクト名>/backend/internal/

# フロントエンド /cache ページを追加
mkdir -p /Users/user/<プロジェクト名>/frontend/src/app/cache
cp /Users/user/Ghostrunner/templates/with-redis/frontend/src/app/cache/page.tsx /Users/user/<プロジェクト名>/frontend/src/app/cache/page.tsx
```

docker-compose.yml（Redis 選択時）:
- 他のオプション未選択時: Redis 用 docker-compose.yml をコピー
```bash
cp /Users/user/Ghostrunner/templates/with-redis/docker-compose.yml /Users/user/<プロジェクト名>/docker-compose.yml
```
- 他のオプション選択済み（docker-compose.yml がコピー済み）: Edit ツールで Redis サービスを追加する。追加する内容は `/Users/user/Ghostrunner/templates/with-redis/docker-compose.yml` を参照し、services: に redis を追加、volumes: に redisdata: を追加する。

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
cd /Users/user/<プロジェクト名>
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
echo "DATABASE_URL=postgres://postgres:postgres@localhost:${PORT_DB}/<プロジェクト名>?sslmode=disable" >> /Users/user/<プロジェクト名>/backend/.env.example
```

ストレージ選択時:
```bash
echo "STORAGE_ENDPOINT=http://localhost:${PORT_MINIO}" >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_ACCOUNT_ID=' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_ACCESS_KEY_ID=minioadmin' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_ACCESS_KEY_SECRET=minioadmin' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_BUCKET_NAME=uploads' >> /Users/user/<プロジェクト名>/backend/.env.example
```

Redis 選択時:
```bash
echo "REDIS_URL=redis://localhost:${PORT_REDIS}" >> /Users/user/<プロジェクト名>/backend/.env.example
```

最後に `.env` にコピー:
```bash
cp /Users/user/<プロジェクト名>/backend/.env.example /Users/user/<プロジェクト名>/backend/.env
```

### Step 6: 依存関係の解決

```bash
cd /Users/user/<プロジェクト名>/backend
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
cd /Users/user/<プロジェクト名>/backend && go mod tidy
cd /Users/user/<プロジェクト名>/frontend && npm install
```

### Step 7: .claude/ 資産の生成

Ghostrunnerの `.claude/` 資産を一括コピーし、CLAUDE.md だけ新プロジェクト用に生成する。

#### 7.1 一括コピー

```bash
mkdir -p /Users/user/<プロジェクト名>/.claude/agents

# agents/ を一括コピー
cp /Users/user/Ghostrunner/.claude/agents/*.md /Users/user/<プロジェクト名>/.claude/agents/

# skills/ を一括コピー
cp -r /Users/user/Ghostrunner/.claude/skills/ /Users/user/<プロジェクト名>/.claude/skills/

# settings.json をコピー
cp /Users/user/Ghostrunner/.claude/settings.json /Users/user/<プロジェクト名>/.claude/settings.json
```

未選択のオプションに対応するエージェントを削除:

PostgreSQL 未選択時:
```bash
rm -f /Users/user/<プロジェクト名>/.claude/agents/pg-*.md
```

ストレージ未選択時:
```bash
rm -f /Users/user/<プロジェクト名>/.claude/agents/storage-*.md
```

Redis 未選択時:
```bash
rm -f /Users/user/<プロジェクト名>/.claude/agents/redis-*.md
```

#### 7.2 CLAUDE.md 生成

Ghostrunnerの `.claude/CLAUDE.md` (`/Users/user/Ghostrunner/.claude/CLAUDE.md`) の構造を参考に、新プロジェクト用に生成する。

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
ln -s /Users/user/Ghostrunner/devtools /Users/user/<プロジェクト名>/.devtools
```

`.gitignore` に `.devtools` を追加する:

```bash
echo '.devtools' >> /Users/user/<プロジェクト名>/.gitignore
```

### Step 7.6: 開発フォルダ構成の作成

プロジェクトの開発ドキュメント用フォルダ構成を作成する:

```bash
mkdir -p /Users/user/<プロジェクト名>/開発/検討中/アーカイブ
mkdir -p /Users/user/<プロジェクト名>/開発/実装/実装待ち
mkdir -p /Users/user/<プロジェクト名>/開発/実装/完了/アーカイブ
mkdir -p /Users/user/<プロジェクト名>/開発/資料/アーカイブ
```

各末端ディレクトリに `.gitkeep` を配置:

```bash
touch /Users/user/<プロジェクト名>/開発/検討中/アーカイブ/.gitkeep
touch /Users/user/<プロジェクト名>/開発/実装/実装待ち/.gitkeep
touch /Users/user/<プロジェクト名>/開発/実装/完了/アーカイブ/.gitkeep
touch /Users/user/<プロジェクト名>/開発/資料/アーカイブ/.gitkeep
```

### Step 8: Git 初期化

```bash
cd /Users/user/<プロジェクト名>
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
cd /Users/user/<プロジェクト名>

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
cd /Users/user/<プロジェクト名>
nohup sh -c 'cd backend && set -a && . ./.env && set +a && go run ./cmd/server' > /tmp/<プロジェクト名>-backend.log 2>&1 &
```

起動ログを確認し、「Listening on」が出ることを確認する。PostgreSQL 選択時は「Database migration completed」、ストレージ選択時は「Storage initialized」、Redis 選択時は「Redis connected」も確認する。

#### 9.4 フロントエンド起動

```bash
cd /Users/user/<プロジェクト名>
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
cd /Users/user/<プロジェクト名>
nohup sh -c 'cd backend && set -a && . ./.env && set +a && go run ./cmd/server' > /tmp/<プロジェクト名>-backend.log 2>&1 &
```

#### 10.3 フロントエンド起動

```bash
cd /Users/user/<プロジェクト名>
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

### Step 12: MVP機能の実装

Step 2 の Q2 で承認されたMVP機能を実装する。

**重要なルール:**
- 作業ディレクトリは `/Users/user/<プロジェクト名>/` に移動して実装する
- 既存のコードパターン（registry パターン、handler 構造）に従う
- **1つの機能だけ**を実装する。スコープを広げない
- バックエンドとフロントエンドの両方を実装する

#### 12.1 バックエンド実装

1. DB を使う場合: model（GORM構造体）を作成
2. handler を作成（CRUD の必要な部分だけ）
3. registry にルーティング登録
4. `cd /Users/user/<プロジェクト名>/backend && go build ./...` でビルド確認

**実装パターン**: 既存の `handler/hello.go` と `registry/base.go` を参考にする。

#### 12.2 フロントエンド実装

1. MVP機能のページを作成（`src/app/<機能名>/page.tsx`）
2. トップページ（`src/app/page.tsx`）にリンクを追加
3. API呼び出しとデータ表示を実装
4. `cd /Users/user/<プロジェクト名>/frontend && npm run build` でビルド確認

**実装パターン**: シンプルな Server Component or Client Component。Tailwind CSS でスタイリング。

#### 12.3 サーバー再起動と動作確認

```bash
cd /Users/user/<プロジェクト名>
make dev
```

起動後、実装した機能が動作することを確認:
- フロントエンドの画面が表示されること
- APIが正しくレスポンスを返すこと
- DB を使う場合、データの登録と取得ができること

#### 12.4 README.md 更新

実装した機能を README.md の「実装済みの機能」セクションに追記する。
- ページ表に新しい画面を追加
- API表に新しいエンドポイントを追加

#### 12.5 Git コミット

```bash
cd /Users/user/<プロジェクト名>
git add -A
git commit -m "feat: <MVP機能名>の初期実装"
```

### Step 13: 完了メッセージ

以下を表示する:

```
プロジェクト「<プロジェクト名>」の作成が完了しました！

生成先: /Users/user/<プロジェクト名>/

実装した機能:
  <MVP機能の説明>

アクセス:
  フロントエンド: http://localhost:${PORT_FRONTEND}
  バックエンド API: http://localhost:${PORT_BACKEND}/api/health

次の機能追加は `/fullstack` コマンドで行えます。
```

### Step 14: 本番デプロイ準備（PostgreSQL またはストレージまたは Redis 選択時）

PostgreSQL、ストレージ、Redis のいずれかを選択した場合、本番環境のセットアップを提案する。

AskUserQuestion で確認:
「本番デプロイの準備（GCP + Neon / R2 / Upstash）を行いますか？」
- 選択肢: はい / スキップ（後で手動で設定する）

**「スキップ」の場合**: Step 14 を終了する。

#### 14.1 gcloud CLI 確認・インストール

```bash
which gcloud
```

未インストールの場合:
```bash
brew install --cask google-cloud-sdk
```

#### 14.2 GCP 認証

```bash
gcloud auth list 2>&1
```

アクティブアカウントがない場合:
ユーザーに案内: 「GCP にログインします。ブラウザが開きます。」
```bash
gcloud auth login
```

#### 14.3 GCP プロジェクト選択

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

#### 14.4 GCP API 有効化

```bash
gcloud services enable \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com
```

#### 14.5 GCP サービスアカウント作成

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

#### 14.6 GitHub リポジトリ作成・Secrets 登録

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
cd /Users/user/<プロジェクト名>
gh repo create <プロジェクト名> --private --source=. --push
```

GitHub Secrets（リポジトリレベル、全環境共通）を登録:
```bash
gh secret set GCP_SA_KEY < /tmp/<プロジェクト名>-sa-key.json
gh secret set GCP_PROJECT_ID --body="$GCP_PROJECT"

# ローカルのキーファイルを削除
rm /tmp/<プロジェクト名>-sa-key.json
```

#### 14.7 GitHub Environments 作成

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

#### 14.10 Neon プロジェクト作成

AskUserQuestion で確認:
「Neon プロジェクトを新規作成しますか？」
- 選択肢: 新規作成 / 既存を使う

**「新規作成」の場合**:
```bash
neonctl projects create --name <プロジェクト名> --region-id aws-ap-northeast-1
```

**「既存を使う」の場合**:
`neonctl projects list` の結果を表示し、使用するプロジェクトを選択させる。

#### 14.11 スキーマ反映

```bash
PROD_CONNSTR=$(neonctl connection-string --project-id <プロジェクトID>)
psql "$PROD_CONNSTR" -f /Users/user/<プロジェクト名>/db/init.sql
```

#### 14.12 Secret Manager に DATABASE_URL を登録

```bash
echo -n "$PROD_CONNSTR" | gcloud secrets create DATABASE_URL --data-file=-
```

注: サービスアカウントへの `roles/secretmanager.secretAccessor` は 12.5 で付与済み。

#### 14.13 Secret Manager に R2 クレデンシャルを登録（ストレージ選択時）

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

#### 14.14 Upstash CLI 確認・インストール

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

#### 14.15 Upstash Redis 作成・Secret Manager 登録

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

#### 14.16 deploy.yml に Secret Manager 参照を追加

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

#### 14.17 デプロイ準備完了メッセージ

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
