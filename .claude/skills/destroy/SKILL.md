---
name: destroy
description: プロジェクトのリソースを検出し選択的に削除する
disable-model-invocation: true
---


# /destroy - プロジェクト削除

引数 `$ARGUMENTS` から対象プロジェクトを特定し、選択したリソースを削除する。

## 処理フロー

### Step 1: 引数の処理とバリデーション

`$ARGUMENTS` からプロジェクト名を取得する。

**バリデーション:**
- プロジェクト名が空の場合はエラー: 「プロジェクト名を指定してください。例: `/destroy my-project`」
- `~/<プロジェクト名>/` が存在しない場合はエラー: 「プロジェクトが見つかりません」

### Step 2: リソース検出

削除対象のリソースを自動検出する。

```bash
cd ~/<プロジェクト名>
```

以下を並列で検出する:
- サーバープロセス: `lsof -ti:8080`, `lsof -ti:3000`
- Docker: `docker-compose ps 2>/dev/null`
- GitHub: `gh repo view <プロジェクト名> 2>&1`
- GCP: `gcloud config get-value project 2>&1`, `gcloud secrets list --filter="name:DATABASE_URL" 2>&1`, `gcloud secrets list --filter="name:R2_" 2>&1`, `gcloud secrets list --filter="name:REDIS_URL" 2>&1`
- Neon: `neonctl projects list 2>/dev/null` で `<プロジェクト名>` と `<プロジェクト名>-staging` を検出
- Upstash: `upstash redis list 2>/dev/null` で `<プロジェクト名>` と `<プロジェクト名>-staging` を検出

### Step 3: 削除対象の選択

検出結果を一覧表示し、AskUserQuestion（multiSelect: true）で削除対象を選択させる。

「削除するリソースを選択してください（複数選択可）」

選択肢（検出されたもののみ表示）:
- **Docker** - コンテナ・ボリュームを停止・削除
- **Neon (staging)** - staging 用 Neon プロジェクトを削除
- **Neon (production)** - production 用 Neon プロジェクトを削除（本番データが失われます）
- **Upstash (staging)** - staging 用 Upstash Redis を削除
- **Upstash (production)** - production 用 Upstash Redis を削除（本番データが失われます）
- **GitHub** - GitHub リポジトリ・GCP デプロイ基盤を削除
- **ソースコード** - ~/<プロジェクト名>/ を削除

**注意**: サーバープロセスの停止は常に実行する（選択不要）。GCP サービスアカウント・Secret Manager は GitHub を選択した場合に一緒に削除する（デプロイ基盤とセットのため）。

### Step 4: 実行

選択に基づいて以下の順序で実行する。リモートリソースの削除はディレクトリ内の情報が必要なため、ソースコード削除より先に行う。

#### 4.1 サーバー停止（常に実行）

```bash
cd ~/<プロジェクト名>
make stop 2>/dev/null || true
lsof -ti:8080 | xargs kill -9 2>/dev/null || true
lsof -ti:3000 | xargs kill -9 2>/dev/null || true
```

#### 4.2 Docker（選択時）

```bash
cd ~/<プロジェクト名>
docker-compose down -v 2>/dev/null || true
```

#### 4.3 Neon staging（選択時）

```bash
# neonctl projects list から <プロジェクト名>-staging のプロジェクトIDを取得して削除
STAGING_ID=$(neonctl projects list --output json 2>/dev/null | jq -r '.[] | select(.name == "<プロジェクト名>-staging") | .id')
if [ -n "$STAGING_ID" ]; then
  neonctl projects delete "$STAGING_ID"
fi
```

#### 4.4 Neon production（選択時）

```bash
PROD_ID=$(neonctl projects list --output json 2>/dev/null | jq -r '.[] | select(.name == "<プロジェクト名>") | .id')
if [ -n "$PROD_ID" ]; then
  neonctl projects delete "$PROD_ID"
fi
```

#### 4.5 Upstash staging（選択時）

```bash
# upstash redis list から <プロジェクト名>-staging の DB を特定して削除
upstash redis delete --name <プロジェクト名>-staging 2>/dev/null || true
```

#### 4.6 Upstash production（選択時）

```bash
upstash redis delete --name <プロジェクト名> 2>/dev/null || true
```

#### 4.7 GitHub + GCP デプロイ基盤（選択時）

GitHub リポジトリと、それに紐づく GCP デプロイ基盤をまとめて削除する:

```bash
# GCP サービスアカウント削除
GCP_PROJECT=$(gcloud config get-value project 2>/dev/null)
SA_EMAIL="<プロジェクト名>-deployer@${GCP_PROJECT}.iam.gserviceaccount.com"
gcloud iam service-accounts delete "$SA_EMAIL" --quiet 2>/dev/null || true

# Secret Manager 削除（DB）
gcloud secrets delete DATABASE_URL_STAGING --quiet 2>/dev/null || true
gcloud secrets delete DATABASE_URL --quiet 2>/dev/null || true

# Secret Manager 削除（R2）
gcloud secrets delete R2_ACCOUNT_ID --quiet 2>/dev/null || true
gcloud secrets delete R2_ACCESS_KEY_ID --quiet 2>/dev/null || true
gcloud secrets delete R2_ACCESS_KEY_SECRET --quiet 2>/dev/null || true
gcloud secrets delete R2_BUCKET_NAME --quiet 2>/dev/null || true
gcloud secrets delete R2_BUCKET_NAME_STAGING --quiet 2>/dev/null || true

# Secret Manager 削除（Redis）
gcloud secrets delete REDIS_URL --quiet 2>/dev/null || true
gcloud secrets delete REDIS_URL_STAGING --quiet 2>/dev/null || true

# GitHub リポジトリ削除
gh repo delete <プロジェクト名> --yes 2>/dev/null || true
```

#### 4.8 ソースコード（選択時）

最終確認を行う:
AskUserQuestion: 「~/<プロジェクト名>/ を完全に削除します。よろしいですか？」
- 選択肢: はい、削除する / キャンセル

```bash
rm -rf ~/<プロジェクト名>
```

### Step 5: 完了メッセージ

選択した内容に応じて、実際に削除したリソースのみを表示する:

```
プロジェクト「<プロジェクト名>」の削除が完了しました。

削除済み:
  - サーバープロセス停止
  - Docker コンテナ・ボリューム削除             ← 選択時のみ
  - Neon staging プロジェクト削除               ← 選択時のみ
  - Neon production プロジェクト削除            ← 選択時のみ
  - Upstash Redis staging 削除                 ← 選択時のみ
  - Upstash Redis production 削除              ← 選択時のみ
  - GitHub リポジトリ削除                       ← 選択時のみ
  - GCP サービスアカウント・Secret Manager 削除  ← GitHub選択時のみ
  - ~/<プロジェクト名>/ 削除           ← 選択時のみ
```
