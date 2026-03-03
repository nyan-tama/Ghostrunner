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

**Q2: データサービスの選択**
「使用するデータサービスを選択してください（複数選択可）」（multiSelect: true）
- 選択肢: PostgreSQL（Neon） / オブジェクトストレージ（Cloudflare R2） / Redis（Upstash）

**Q3: 最終確認**
収集した情報を表示し、生成を開始してよいか確認する:
```
プロジェクト名: <名前>
生成先: /Users/user/<名前>/
概要: <入力された概要>
データサービス: PostgreSQL, ストレージ, Redis（選択したもの）
```

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

### Step 5: .env 作成

base の `.env.example` に選択したオプションの環境変数を追記し、`.env` にコピーする。

PostgreSQL 選択時:
```bash
echo 'DATABASE_URL=postgres://postgres:postgres@localhost:5432/<プロジェクト名>?sslmode=disable' >> /Users/user/<プロジェクト名>/backend/.env.example
```

ストレージ選択時:
```bash
echo 'STORAGE_ENDPOINT=http://localhost:9000' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_ACCOUNT_ID=' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_ACCESS_KEY_ID=minioadmin' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_ACCESS_KEY_SECRET=minioadmin' >> /Users/user/<プロジェクト名>/backend/.env.example
echo 'R2_BUCKET_NAME=uploads' >> /Users/user/<プロジェクト名>/backend/.env.example
```

Redis 選択時:
```bash
echo 'REDIS_URL=redis://localhost:6379' >> /Users/user/<プロジェクト名>/backend/.env.example
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
mkdir -p /Users/user/<プロジェクト名>/.claude/agents /Users/user/<プロジェクト名>/.claude/commands

# agents/ と commands/ を一括コピー
cp /Users/user/Ghostrunner/.claude/agents/*.md /Users/user/<プロジェクト名>/.claude/agents/
cp /Users/user/Ghostrunner/.claude/commands/*.md /Users/user/<プロジェクト名>/.claude/commands/

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

- PostgreSQL 選択時: 5432, 8080
- ストレージ選択時: 9000, 9001, 8080
- Redis 選択時: 6379, 8080
- 複数選択時: 選択したサービスのポートを全て含める

```bash
lsof -ti:5432  # PostgreSQL 選択時
lsof -ti:9000  # ストレージ選択時
lsof -ti:6379  # Redis 選択時
lsof -ti:8080
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
curl -s http://localhost:9000/minio/health/live
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
curl -s http://localhost:8080/api/health
```

PostgreSQL 選択時:
```bash
# DB書き込みテスト
curl -s -X POST http://localhost:8080/api/samples \
  -H "Content-Type: application/json" \
  -d '{"name":"Hello","description":"Initial sample"}'

# 読み取り確認
curl -s http://localhost:8080/api/samples
```

ストレージ選択時:
```bash
# ファイルアップロードテスト
echo "test" > /tmp/test-upload.txt
curl -s -X POST http://localhost:8080/api/storage/upload -F "file=@/tmp/test-upload.txt"

# ファイル一覧確認
curl -s http://localhost:8080/api/storage/files
rm /tmp/test-upload.txt
```

Redis 選択時:
```bash
# キャッシュ書き込みテスト
curl -s -X POST http://localhost:8080/api/cache \
  -H "Content-Type: application/json" \
  -d '{"key":"hello","value":"world","ttl_seconds":60}'

# 読み取り確認
curl -s http://localhost:8080/api/cache/hello
```

### Step 10: 起動（PostgreSQL もストレージも Redis も未選択時）

PostgreSQL もストレージも Redis も選択しなかった場合:

#### 10.1 ポート確保

ポート 8080 が使用中か確認し、使用中の場合はユーザーに確認して停止する。

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
curl -s http://localhost:8080/api/health
curl -s http://localhost:3000 > /dev/null && echo "Frontend: OK"
```

### Step 11: 完了メッセージ

以下を表示する:

```
プロジェクト「<プロジェクト名>」の生成・起動が完了しました！

生成先: /Users/user/<プロジェクト名>/

アクセス:
  フロントエンド: http://localhost:3000
  バックエンド API: http://localhost:8080/api/health

サーバー停止:
  cd /Users/user/<プロジェクト名>
  make stop
```

DB選択時は追加で表示:
```
DB接続:
  docker exec <プロジェクト名>-db psql -U postgres -d <プロジェクト名>
```

Redis 選択時は追加で表示:
```
Redis:
  docker exec <プロジェクト名>-redis redis-cli
```

### Step 12: 本番デプロイ準備（PostgreSQL またはストレージまたは Redis 選択時）

PostgreSQL、ストレージ、Redis のいずれかを選択した場合、本番環境のセットアップを提案する。

AskUserQuestion で確認:
「本番デプロイの準備（GCP + Neon / R2 / Upstash）を行いますか？」
- 選択肢: はい / スキップ（後で手動で設定する）

**「スキップ」の場合**: Step 12 を終了する。

#### 12.1 gcloud CLI 確認・インストール

```bash
which gcloud
```

未インストールの場合:
```bash
brew install --cask google-cloud-sdk
```

#### 12.2 GCP 認証

```bash
gcloud auth list 2>&1
```

アクティブアカウントがない場合:
ユーザーに案内: 「GCP にログインします。ブラウザが開きます。」
```bash
gcloud auth login
```

#### 12.3 GCP プロジェクト選択

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

#### 12.4 GCP API 有効化

```bash
gcloud services enable \
  run.googleapis.com \
  secretmanager.googleapis.com \
  containerregistry.googleapis.com
```

#### 12.5 GCP サービスアカウント作成

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

#### 12.6 GitHub リポジトリ作成・Secrets 登録

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

#### 12.7 GitHub Environments 作成

staging と production の2環境を作成し、環境ごとの Variables を登録する:
```bash
# staging 環境の Variables（初回デプロイ後に実際の URL に更新する）
gh variable set FRONTEND_URL --env staging --body="https://<プロジェクト名>-frontend-staging-xxxxxxxxxx-an.a.run.app"
gh variable set BACKEND_URL --env staging --body="https://<プロジェクト名>-backend-staging-xxxxxxxxxx-an.a.run.app"

# production 環境の Variables
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

#### 12.10 Neon プロジェクト作成（staging + production）

staging 用と production 用の2つの Neon プロジェクトを作成する。

AskUserQuestion で確認:
「Neon プロジェクトを新規作成しますか？（staging + production の2つ作成します）」
- 選択肢: 新規作成 / 既存を使う

**「新規作成」の場合**:
```bash
# staging 用
neonctl projects create --name <プロジェクト名>-staging --region-id aws-ap-northeast-1

# production 用
neonctl projects create --name <プロジェクト名> --region-id aws-ap-northeast-1
```

**「既存を使う」の場合**:
`neonctl projects list` の結果を表示し、staging 用と production 用をそれぞれ選択させる。

#### 12.11 スキーマ反映

両方のプロジェクトに init.sql を適用する:
```bash
# staging の接続文字列を取得
STAGING_CONNSTR=$(neonctl connection-string --project-id <staging のプロジェクトID>)

# production の接続文字列を取得
PROD_CONNSTR=$(neonctl connection-string --project-id <production のプロジェクトID>)

# staging にスキーマ反映
psql "$STAGING_CONNSTR" -f /Users/user/<プロジェクト名>/db/init.sql

# production にスキーマ反映
psql "$PROD_CONNSTR" -f /Users/user/<プロジェクト名>/db/init.sql
```

#### 12.12 Secret Manager に DATABASE_URL を登録

staging と production それぞれの接続文字列を Secret Manager に登録する:

```bash
# staging 用
echo -n "$STAGING_CONNSTR" | gcloud secrets create DATABASE_URL_STAGING --data-file=-

# production 用
echo -n "$PROD_CONNSTR" | gcloud secrets create DATABASE_URL --data-file=-
```

注: サービスアカウントへの `roles/secretmanager.secretAccessor` は 12.5 で付与済み。

#### 12.13 Secret Manager に R2 クレデンシャルを登録（ストレージ選択時）

ストレージを選択した場合のみ実行する。

AskUserQuestion で確認:
「Cloudflare R2 のクレデンシャルを登録しますか？（R2 バケットと API トークンの事前作成が必要です）」
- 選択肢: はい / 後で設定する

**「はい」の場合**:
R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_ACCESS_KEY_SECRET, R2_BUCKET_NAME（production 用）, R2_BUCKET_NAME_STAGING（staging 用）をそれぞれ質問し、Secret Manager に登録する:

```bash
echo -n "<値>" | gcloud secrets create R2_ACCOUNT_ID --data-file=-
echo -n "<値>" | gcloud secrets create R2_ACCESS_KEY_ID --data-file=-
echo -n "<値>" | gcloud secrets create R2_ACCESS_KEY_SECRET --data-file=-
echo -n "<値>" | gcloud secrets create R2_BUCKET_NAME --data-file=-
echo -n "<値>" | gcloud secrets create R2_BUCKET_NAME_STAGING --data-file=-
```

**「後で設定する」の場合**:
完了メッセージに Secret Manager 登録コマンドを含める。

**以下の 12.14〜12.15 は Redis 選択時のみ実行する。**

#### 12.14 Upstash CLI 確認・インストール

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

#### 12.15 Upstash Redis 作成・Secret Manager 登録

staging 用と production 用の2つの Redis DB を作成する。

AskUserQuestion で確認:
「Upstash Redis を新規作成しますか？（staging + production の2つ作成します）」
- 選択肢: 新規作成 / 後で設定する

**「新規作成」の場合**:
```bash
# staging 用
upstash redis create --name <プロジェクト名>-staging --region ap-northeast-1

# production 用
upstash redis create --name <プロジェクト名> --region ap-northeast-1
```

各 DB の接続文字列（`rediss://default:xxx@xxx.upstash.io:xxx`）を `upstash redis list` で取得し、Secret Manager に登録:

```bash
echo -n "$STAGING_REDIS_URL" | gcloud secrets create REDIS_URL_STAGING --data-file=-
echo -n "$PROD_REDIS_URL" | gcloud secrets create REDIS_URL --data-file=-
```

**「後で設定する」の場合**:
完了メッセージに Secret Manager 登録コマンドを含める。

#### 12.16 deploy.yml に Secret Manager 参照を追加

生成したプロジェクトの `.github/workflows/deploy.yml` を Edit ツールで編集し、backend の `gcloud run deploy` コマンドに `--set-secrets` を追加する。

PostgreSQL 選択時に追加する行:
```
--set-secrets "DATABASE_URL=${{ github.ref_name == 'main' && 'DATABASE_URL' || 'DATABASE_URL_STAGING' }}:latest"
```

ストレージ選択時に追加する行:
```
--set-secrets "R2_ACCOUNT_ID=R2_ACCOUNT_ID:latest,R2_ACCESS_KEY_ID=R2_ACCESS_KEY_ID:latest,R2_ACCESS_KEY_SECRET=R2_ACCESS_KEY_SECRET:latest,R2_BUCKET_NAME=${{ github.ref_name == 'main' && 'R2_BUCKET_NAME' || 'R2_BUCKET_NAME_STAGING' }}:latest"
```

Redis 選択時に追加する行:
```
--set-secrets "REDIS_URL=${{ github.ref_name == 'main' && 'REDIS_URL' || 'REDIS_URL_STAGING' }}:latest"
```

複数選択時は1つの `--set-secrets` にカンマ区切りでまとめる。

追加位置: backend deploy ステップの `--set-env-vars` の行の直前に `\` で行を継続して挿入する。

#### 12.17 staging ブランチ作成・push

```bash
cd /Users/user/<プロジェクト名>
git checkout -b staging
git push -u origin staging
git checkout main
```

#### 12.18 デプロイ準備完了メッセージ

```
本番デプロイ準備が完了しました！

GCP プロジェクト: <プロジェクトID>
GitHub: https://github.com/<ユーザー名>/<プロジェクト名>

デプロイフロー:
  feat ブランチ → staging にマージ → staging 環境に自動デプロイ
  staging → main にマージ → production 環境に自動デプロイ

環境:
  staging:    push to staging ブランチで自動デプロイ
  production: push to main ブランチで自動デプロイ
```

PostgreSQL 選択時は追加で表示:
```
Neon:
  staging:    neonctl projects list で確認
  production: neonctl projects list で確認

Secret Manager (DB):
  DATABASE_URL_STAGING: staging 用接続文字列
  DATABASE_URL:         production 用接続文字列
```

ストレージ選択時は追加で表示:
```
Secret Manager (R2):
  R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_ACCESS_KEY_SECRET
  R2_BUCKET_NAME (production), R2_BUCKET_NAME_STAGING (staging)
```

Redis 選択時は追加で表示:
```
Upstash:
  staging:    upstash redis list で確認
  production: upstash redis list で確認

Secret Manager (Redis):
  REDIS_URL (production), REDIS_URL_STAGING (staging)
```

共通:
```
注意:
  初回デプロイ後、Cloud Run URL が確定したら
  GitHub Environments の FRONTEND_URL と BACKEND_URL を実際の URL に更新してください:
    gh variable set FRONTEND_URL --env staging --body="https://実際のURL"
    gh variable set BACKEND_URL --env staging --body="https://実際のURL"
    gh variable set FRONTEND_URL --env production --body="https://実際のURL"
    gh variable set BACKEND_URL --env production --body="https://実際のURL"
```
