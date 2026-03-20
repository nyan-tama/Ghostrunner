# staging 廃止・簡略化 実装計画

## 概要

`/init` のフローから staging 概念を削除し、デプロイフローを `feat -> main -> production` に簡略化する。

## 変更対象ファイル

| ファイル | 変更内容 |
|---------|---------|
| `.claude/skills/init/SKILL.md` | staging 関連ステップの削除・簡略化 |
| `templates/base/.github/workflows/deploy.yml` | main のみトリガー、staging 分岐ロジック削除 |

## 変更しないファイル

以下は将来の再検討用にそのまま残す:
- `.claude/skills/stage/SKILL.md`
- `.claude/skills/release/SKILL.md`
- `.claude/agents/staging-manager.md`
- `.claude/agents/release-manager.md`

---

## 変更詳細

### 1. deploy.yml（テンプレート）

**ファイル:** `templates/base/.github/workflows/deploy.yml`

#### 1.1 トリガー変更
- `branches: [main, staging]` -> `branches: [main]`

#### 1.2 staging 分岐ロジック削除（deploy-backend ジョブ）
- `environment:` の三項演算子を削除 -> `environment: production` に固定
- `SUFFIX` env 変数を削除
- イメージ名・サービス名から `${{ env.SUFFIX }}` を削除

#### 1.3 staging 分岐ロジック削除（deploy-frontend ジョブ）
- deploy-backend と同様の変更

### 2. SKILL.md（/init スキル）

**ファイル:** `.claude/skills/init/SKILL.md`

#### 2.1 Step 12 の条件変更
- 現状: 「PostgreSQL またはストレージまたは Redis 選択時」のみ提示
- 変更: 条件はそのまま（インフラ依存がない場合はデプロイ準備不要の判断は妥当）

#### 2.2 Step 12.7 GitHub Environments（簡略化）
- staging 環境の Variables 登録を削除
- production 環境の Variables のみ登録

変更前:
```bash
gh variable set FRONTEND_URL --env staging --body="https://..."
gh variable set BACKEND_URL --env staging --body="https://..."
gh variable set FRONTEND_URL --env production --body="https://..."
gh variable set BACKEND_URL --env production --body="https://..."
```

変更後:
```bash
gh variable set FRONTEND_URL --env production --body="https://..."
gh variable set BACKEND_URL --env production --body="https://..."
```

#### 2.3 Step 12.10 Neon プロジェクト作成（簡略化）
- staging + production の2つ -> production の1つのみ作成
- 質問文を変更: 「Neon プロジェクトを新規作成しますか？」

変更前:
```bash
neonctl projects create --name <プロジェクト名>-staging --region-id aws-ap-northeast-1
neonctl projects create --name <プロジェクト名> --region-id aws-ap-northeast-1
```

変更後:
```bash
neonctl projects create --name <プロジェクト名> --region-id aws-ap-northeast-1
```

#### 2.4 Step 12.11 スキーマ反映（簡略化）
- staging への適用を削除、production のみ

変更前:
```bash
STAGING_CONNSTR=$(neonctl connection-string --project-id <staging のプロジェクトID>)
PROD_CONNSTR=$(neonctl connection-string --project-id <production のプロジェクトID>)
psql "$STAGING_CONNSTR" -f ...
psql "$PROD_CONNSTR" -f ...
```

変更後:
```bash
PROD_CONNSTR=$(neonctl connection-string --project-id <production のプロジェクトID>)
psql "$PROD_CONNSTR" -f ...
```

#### 2.5 Step 12.12 Secret Manager DATABASE_URL（簡略化）
- DATABASE_URL_STAGING を削除、DATABASE_URL のみ

変更前:
```bash
echo -n "$STAGING_CONNSTR" | gcloud secrets create DATABASE_URL_STAGING --data-file=-
echo -n "$PROD_CONNSTR" | gcloud secrets create DATABASE_URL --data-file=-
```

変更後:
```bash
echo -n "$PROD_CONNSTR" | gcloud secrets create DATABASE_URL --data-file=-
```

#### 2.6 Step 12.13 Secret Manager R2（簡略化）
- R2_BUCKET_NAME_STAGING を削除

変更前: R2_BUCKET_NAME + R2_BUCKET_NAME_STAGING の2つ
変更後: R2_BUCKET_NAME のみ

#### 2.7 Step 12.15 Upstash Redis（簡略化）
- staging 用 Redis を削除、production の1つのみ
- REDIS_URL_STAGING を削除

変更前:
```bash
upstash redis create --name <プロジェクト名>-staging --region ap-northeast-1
upstash redis create --name <プロジェクト名> --region ap-northeast-1
echo -n "$STAGING_REDIS_URL" | gcloud secrets create REDIS_URL_STAGING --data-file=-
echo -n "$PROD_REDIS_URL" | gcloud secrets create REDIS_URL --data-file=-
```

変更後:
```bash
upstash redis create --name <プロジェクト名> --region ap-northeast-1
echo -n "$PROD_REDIS_URL" | gcloud secrets create REDIS_URL --data-file=-
```

#### 2.8 Step 12.16 deploy.yml の --set-secrets（簡略化）
- staging/main の条件分岐を削除、シークレット名を直接指定

変更前:
```
--set-secrets "DATABASE_URL=${{ github.ref_name == 'main' && 'DATABASE_URL' || 'DATABASE_URL_STAGING' }}:latest"
```

変更後:
```
--set-secrets "DATABASE_URL=DATABASE_URL:latest"
```

R2、Redis も同様に簡略化。

#### 2.9 Step 12.17 削除
- staging ブランチ作成・push のステップを丸ごと削除

#### 2.10 Step 12.18 完了メッセージ（簡略化）
- デプロイフローを `feat -> main -> production` に変更
- staging 環境への言及を削除
- GitHub Environments の URL 更新案内を production のみに変更

---

## 実装ステップ

1. `templates/base/.github/workflows/deploy.yml` を編集
2. `.claude/skills/init/SKILL.md` の Step 12 系を編集
3. セルフチェック（staging への言及が残っていないか確認）
