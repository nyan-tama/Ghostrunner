# 実装完了レポート: 動線整備（patrol → dashboard 一本化）

対応計画書: `2026-05-26_動線整備_patrol→dashboard移行_plan.md`

## 実装サマリー

- **実装日**: 2026-05-26
- **対象ブランチ**: `feat/dashboard-gui`
- **スコープ**: フロントエンド単独（Next.js のみ、バックエンド・DB 変更なし）
- **変更ファイル数**: 2 files
- **目的**: トップ `/` の「巡回」動線を新統括GUI `/dashboard` に差し替え、旧 `/patrol` ブックマークは 308 リダイレクトで救済

---

## 変更ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `devtools/frontend/src/app/page.tsx` | 507-513 行の「巡回」リンクボタンを差し替え。`href="/patrol"` → `href="/dashboard"`、ラベル `巡回` → `統括`、`title` 属性 `巡回ダッシュボード` → `統括ダッシュボード`、Tailwind 配色 orange 系 → blue 系（`bg-blue-100 text-blue-700 hover:bg-blue-200`）。ボタン位置（プロジェクト作成・ファイル一覧の左隣）は変更なし |
| `devtools/frontend/next.config.ts` | 新規 `async redirects()` を追加。`source: "/patrol"` → `destination: "/dashboard"` を `permanent: true`（308 Permanent Redirect）で定義。既存 `rewrites()`（API プロキシ用）には触れていない。パス対象は `/patrol` 完全一致のみ（サブパスは存在しないためワイルドカードなし） |

---

## 計画からの変更点

特になし。計画書「3.1 変更ファイル一覧」「3.2 リンクボタンの最終形」「3.3 リダイレクト仕様」の通りに 2 ファイルのみ修正した。設計判断（FE-1〜FE-7）も全て計画書記載通りに採用。

---

## 実装しなかった事項（スコープ外、計画通り）

計画書「5. 残課題」の方針通り、以下は本 MVP では実施しない：

- 古い `/patrol` ページ（`devtools/frontend/src/app/patrol/`）の削除
- `devtools/frontend/src/components/patrol/` 削除
- `usePatrol*.ts` / `patrolApi.ts` / `types/patrol.ts` 削除
- `devtools/backend/internal/patrol/` および `/api/patrol/*` ハンドラ削除
- リダイレクト自体の撤去（ブックマーク救済不要時点で）

これらはリダイレクト経由で当面温存し、別タスクで段階廃止する。

---

## 自動テストを書かなかった理由（計画通り）

計画書「6.1 設計方針」「6.3 自動テストを書かない理由」の判断を踏襲：

- リンクの `href` / ラベル / Tailwind クラス文字列の置換に対する Snapshot/RTL での assert は保守コスト > 利益
- `next.config.ts` の `redirects()` は Next.js フレームワーク機能の保証範囲。E2E 導入してまで検証するスコープではない
- 検証は `npm run build` 通過と手動目視（dev サーバー＋スマホ Safari 実機確認）に集中

---

## 実装時の課題

### ビルド・テストで苦戦した点

特になし。実質 5 行以下の変更で、ビルドは一発通過。

### 技術的に難しかった点

特になし。

---

## 検証結果

### ビルド検証

| コマンド | 結果 | 備考 |
|---|---|---|
| `npm run build` | 成功 | `/dashboard` `/patrol` 共にルートテーブルに登録される |
| `npx tsc --noEmit` | 既存ファイルに型エラー 1 件 | `src/__tests__/hooks/useTTS.test.ts`。**本タスク変更とは無関係**（前コミット bad72ba 由来） |
| `npm run lint` | 既存ファイルに error 5 件 / warning 12 件 | `useTTS.ts` 等。**本タスク変更とは無関係**（前コミット bad72ba 由来） |

本タスクで新規追加・変更したファイルに起因するエラー・警告はない。

---

## 残存する懸念点

### 既存の lint / tsc エラー（本タスク外）

前コミット bad72ba（統括GUIダッシュボードのフロントエンド実装）で混入した以下が残存：

- `src/__tests__/hooks/useTTS.test.ts` の型エラー 1 件
- `useTTS.ts` 等の lint error 5 件 / warning 12 件

これらは本タスクのスコープ外。別タスクで対処する。`npm run build` は通るため動作には影響しない。

### 古い patrol コードの存在

計画通り、リダイレクト経由で温存している。`/api/patrol/*` ハンドラ・`PatrolService`・`patrolApi` などは現状動作するため、将来削除する際にはリダイレクトと依存関係を整理する必要がある。検討書「後続」セクションに記載済み。

### 308 キャッシュ

`permanent: true`（308）はブラウザ側でキャッシュされる。将来 `/patrol` を別用途に再利用する場合、既存ユーザーのブラウザでキャッシュが効いて意図しない遷移が起きるリスクあり。再利用予定はないため現状問題なし。

---

## 動作確認チェックリスト

計画書「7. 実装後の動作確認チェックリスト」を踏襲：

1. `make frontend` で dev サーバーが起動する
2. `http://localhost:3333/` を開き、ヘッダに blue 系「統括」ボタン（プロジェクト作成・ファイル一覧の左隣）が表示されることを目視確認
3. 「統括」ボタンをクリックして `/dashboard` に遷移し、統括GUI MVP のダッシュボードが描画されることを確認
4. ブラウザの URL バーに直接 `http://localhost:3333/patrol` を入力し、`/dashboard` に 308 リダイレクトされることを DevTools の Network タブで確認（Status: 308、Location: /dashboard）
5. スマホ Safari（Tailscale 経由）でトップを開き、blue 系「統括」ボタンが表示されタップで `/dashboard` に遷移することを実機確認
6. スマホ Safari で旧 `/patrol` URL を入力し、自動で `/dashboard` に遷移すること（旧ブックマーク救済）を実機確認
7. `/dashboard` 内のチャット入力で「状況は？」を送信し、既存 MVP の SSE 経由でレスポンスが返ること（回帰確認）
8. ヘッダ「ファイル一覧」「プロジェクト作成」ボタンが既存通り遷移すること（回帰確認）

### デプロイ後の確認事項

- [ ] 本番ビルドで `/patrol` → `/dashboard` の 308 リダイレクトが効くこと
- [ ] スマホ Safari のホーム画面追加済みショートカット（旧 `/patrol` 指している場合）が `/dashboard` に正しく着地すること
- [ ] サーバーログに `/patrol` への直接アクセスが減衰していくこと（リダイレクト経由で新 URL を学習）
