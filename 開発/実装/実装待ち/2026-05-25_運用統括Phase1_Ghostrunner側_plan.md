# 運用統括 Phase1（把握）Ghostrunner 側 実装計画

作成日: 2026-05-25
元検討: `開発/検討中/2026-05-25_運用統括（ランタイム状態把握と一括運用）.md`（設計確定済み）
対象: 運用 Phase1（把握）の **Ghostrunner 側のみ**。x-liker 側（userscript POST・backend エンドポイント・`運用/` フォルダ）は x-liker リポジトリで別途 /plan する。

## 1. 概要

統括ハブの chief-director（把握）に「運用軸」を1本足し、各プロジェクトの運用ステータス
（x-liker のフォロー進捗など）を開発カンバンと並べて横断報告できるようにする。さらに CLAUDE.md に
「統括（運用把握）」節を追記し、運用も「状況は？」で報告される運用ルールを明文化する。

これは **agent 定義（.md）と CLAUDE.md の markdown 変更のみ**。Go/Next.js コードは含まない
（よってコード用 planner/reviewer/tester は使わず、末尾に手動検証プランを置く）。

## 2. スコープと契約

### スコープ（Ghostrunner 側・2点）

1. `.claude/agents/chief-director.md` に運用軸を追加
2. `.claude/CLAUDE.md` に「統括（運用把握）」節を追記

### 契約（x-liker と一致・確定済み。検討書のスキーマ準拠。変更しない）

- ステータスファイル: `<project>/運用/状態/<kind>-<account>.json`（chief-director は `運用/状態/*.json` を Glob して読む）
- JSON スキーマ:
  ```json
  {
    "account": "akiba",
    "kind": "auto-follow",
    "status": "running",
    "progress": { "index": 123, "total": 800 },
    "today": { "count": 12, "target": 40 },
    "stats": { "followed": 300, "already": 50, "skipped": 0, "error": 2 },
    "consecutiveErrors": 0,
    "updatedAt": "2026-05-25T09:30:00+09:00"
  }
  ```
  - `status`: `running / idle / paused / blocked / done`
  - `updatedAt`: ISO8601＋TZ。staleness 判定の基準
- opt-in マーカー: `<project>/運用/` フォルダ（`運用/manifest.json`）の有無。無ければ運用なし＝0件（前方互換、`実行中/` と同じ思想）
- staleness 既定閾値: 3時間（`status=running` なのに更新が3時間以上止まっていれば「実行停止疑い」）

注: 運用宣言はフォルダベース（検討書の決定4）なので、**`patrol_projects.json` は変更しない**。chief-director は既存のプロジェクト一覧を辿り、各プロジェクトの `運用/` を見るだけ。

## 3. 成果物1: chief-director に運用軸を追加

[chief-director.md](.claude/agents/chief-director.md) の各手順に運用の読み取り・導出・報告を足す。**読み取り専用は厳守**（運用ファイルも読むだけ。POST やステータス書き換えはしない）。

### 3-1. 手順2（状況収集）への追加

各プロジェクトの `path` について、開発フォルダの読み取りに加えて:
- 運用ステータス: `Glob <path>/運用/状態/*.json`。存在する各ファイルを `Read` し、契約スキーマの
  `account / kind / status / progress / today / stats / consecutiveErrors / updatedAt` を取得。
- 運用の有無判定: `運用/manifest.json` または `運用/状態/*.json` が無ければ運用なし（0件・エラーにしない）。
  `運用/` はあるが `運用/状態/*.json` が無い／空 → 「運用あり・データなし（未起動 or 未送信）」として扱う。

### 3-2. 手順3（状態の導出）への追加

各運用ステータス（kind×account 単位）について:
- `status` をそのまま採用（running / idle / paused / blocked / done）。
- staleness: 現在時刻と `updatedAt` の差を求め、`status=running` かつ差が既定閾値（3時間）超 → 「**実行停止疑い（stale）**」。
- 連続エラー: `consecutiveErrors` が既定閾値（3）以上 → 「**連続エラー**」。

### 3-3. 手順4（注意度判定）への追加

運用シグナルを開発カンバンと統合して 要対応/進行/静観 を決める:
- **要対応（運用）**: `status=blocked` ／ stale（実行停止疑い）／ 連続エラー閾値超 のいずれか。
- **進行（運用）**: `status=running`（stale でない）。
- それ以外（idle/paused/done）は運用の注意度を上げない。
- 開発側の要対応（確認事項待ち）と運用側の要対応はどちらも 要対応 に集約。

### 3-4. 出力フォーマットへの追加

運用を持つプロジェクトには、開発サマリに続けて運用行を **kind×account 単位**で並記する（運用なしのプロジェクトは従来どおり）。レイアウト案（実装時に微調整可）:

```
[要対応] x-liker
  開発: 実装待ち1 / 確認事項0
  運用: [auto-follow/akiba] running フォロー300 本日12/40  ★stale 4時間無更新=実行停止疑い
        [auto-follow/sub]   blocked フォロー150 本日0/40   ★制限検知で停止
[進行]   face-search   : 実装待ち3 / 検討中1            （運用なし）
[自身]   Ghostrunner   : 検討中6 / 実装待ち1             （運用なし）
```

末尾「今日の注目」に運用の要対応も含める（例: 「x-liker akiba が4時間無更新。Chrome が閉じた可能性。確認を推奨」）。

### 3-5. 注意事項（agent 文面）への追加

- 運用ステータスも**読み取りのみ**。`運用/状態/` の書き換え・POST・再点火はしない。
- 運用を持たないプロジェクト（`運用/` 無し）は 0 件扱い（前方互換）。
- stale/blocked/連続エラーは**検知・報告まで**。解消（再点火・調査）はユーザーの明示指示で別途（運用 Phase2 / 後フェーズ）。
- 運用ステータスファイルが壊れている（JSON パース不可）場合は、その旨を1行報告して続行（落とさない）。

## 4. 成果物2: CLAUDE.md「統括（運用把握）」節

[CLAUDE.md](.claude/CLAUDE.md) の「## 統括（把握）」の直後（「## 統括（一括操作）」の前）に「## 統括（運用把握）」を追記し、把握同士を隣接させる。記述内容:

- **報告に運用も含む**: 「状況は？」等の横断把握で、chief-director が各プロジェクトの `運用/状態/` も読み、開発カンバンと並べて運用状態（進捗・stale・blocked・連続エラー）を報告する。
- **把握＝自動・読み取り**: 運用把握は読み取りのみ。運用の状態変更（一括運用＝bulk-ops 発火、再点火）は**明示動詞でのみ**（運用 Phase2・未実装）。既存の「把握＝自動／操作＝明示指示」原則を運用にも適用。
- **検知まで・解消は人間トリガー**: stale/blocked/連続エラーは検知・報告するが、解消（Chrome 再起動・再点火・調査）はユーザー指示で別途（後フェーズ）。開発側の異常終了の扱いと同型。
- **運用を持つ判定**: プロジェクトに `運用/` フォルダがあれば運用あり（`開発/` と対称。検討書の決定4を参照）。

## 5. 変更ファイル一覧

| ファイル | 区分 | 変更内容 |
|---|---|---|
| `.claude/agents/chief-director.md` | 修正 | 手順2/3/4・出力フォーマット・注意事項に運用軸を追加（3章） |
| `.claude/CLAUDE.md` | 修正 | 「## 統括（運用把握）」節を追記（4章） |

新規ファイルなし。コードなし。`patrol_projects.json` 変更なし（フォルダベース opt-in のため）。

## 6. 実装ステップ

1. chief-director.md を 3-1〜3-5 に沿って改訂（読み取り専用を維持）。
2. CLAUDE.md に「統括（運用把握）」節を追記（4章）。
3. 手動検証（9章）: サンプル運用ステータスファイルを使い chief-director を実行して確認。
4. コミット。

## 7. 設計判断・既定値（実装時に確定・ユーザー調整可）

| 項目 | 既定 | 補足 |
|---|---|---|
| staleness 計算方法 | `updatedAt`（契約フィールド）を基準に現在時刻との差を算出。閾値3時間 | macOS `date` での ISO8601＋TZ パースが脆い場合は、`運用/状態/<file>` の mtime（`stat -f %m`）を頑健なフォールバックにする（backend は POST ごとに書き出すため mtime ≒ updatedAt）。表示は updatedAt を使う |
| 連続エラー閾値 | `consecutiveErrors >= 3` で要対応 | 運用で調整可 |
| 出力レイアウト | 3-4 のサブ行案（kind×account 単位） | 可読性優先。実装時に微調整 |
| CLAUDE.md 節の位置 | 「統括（把握）」直後 | 把握同士を隣接 |

## 8. x-liker との独立性

契約（4スキーマ・ファイルパス）で両リポジトリは疎結合。**Ghostrunner 側は x-liker の実装を待たずに実装・検証できる**（手作りのサンプル `運用/状態/*.json` を任意のプロジェクト配下に置けば chief-director の動作を確認できる）。x-liker 側が稼働すれば、同じ契約のファイルがそのまま読まれる。

## 9. スコープ外（この計画でやらない）

- x-liker 側（auto-follow userscript の進捗 POST、backend 受信エンドポイント、`運用/` フォルダ・manifest・.gitignore）← x-liker リポジトリで別途 /plan
- 一括運用（bulk-ops スキル）＝運用 Phase2
- いいね側（search-linker）の運用把握＝将来
- 点火後の結果通知（ntfy）・stale 自動解消（人間トリガー）・devtools 運用ダッシュボード＝後フェーズ
- `patrol_projects.json` への運用フィールド追加（フォルダベース opt-in を採用したため不要）

## 10. 自己レビュー結果（コード reviewer 非該当のため自己点検）

- **読み取り専用の堅持**: 運用軸の追加はすべて Read/Glob/Bash(date,stat,git) の参照のみ。書き込み系の指示を一切入れない（chief-director の既存ツール `Read, Grep, Glob, Bash` の範囲内。ツール追加不要）。OK。
- **前方互換**: 運用なしプロジェクト＝0件扱いを明記。既存の開発カンバン報告は壊さない（運用行は運用ありの時だけ追加）。OK。
- **契約整合**: スキーマ・ファイルパス・staleness 基準が検討書および x-liker 指示と一致。OK。
- **原則整合**: 把握＝自動・読み取り／操作＝明示指示、検知まで・解消は人間トリガー、フォルダ＝真実、を踏襲。OK。
- **用語統一**: kind/account/status/updatedAt/stale/blocked/consecutiveErrors を全節で統一。OK。
- **網羅性**: 変更ファイルは chief-director.md と CLAUDE.md の2点のみで漏れなし（patrol_projects.json は不変更を明記）。OK。
- **エッジケース**: 運用/ありデータなし（未起動）、JSON 破損、updatedAt パース不可（mtime フォールバック）を手順に明記。OK。

## 11. 手動検証プラン（コード test 非該当）

自動ユニットテストに馴染まない（agent 定義・markdown）ため、実装後に以下を手で確認する。x-liker の実装は不要（サンプルファイルで検証可）。

| # | 観点 | 手順 | 合格条件 |
|---|---|---|---|
| 1 | 運用なしは従来どおり | 運用/ を持たないプロジェクトのみで chief-director を実行 | 従来の開発カンバン報告のみ。運用行は出ない・エラーなし |
| 2 | 運用ありの報告 | 任意プロジェクトに `運用/manifest.json` と `運用/状態/auto-follow-test.json`（updatedAt=現在時刻・status=running）を置いて実行 | その運用が「進行」で進捗（フォロー数・本日n/m）と共に報告される |
| 3 | stale 検知 | 上記の `updatedAt` を4時間前に書き換えて実行 | 「実行停止疑い（stale）」として要対応に上がる |
| 4 | blocked 検知 | `status=blocked` のサンプルで実行 | 「制限検知で停止」等の要対応として報告 |
| 5 | 連続エラー検知 | `consecutiveErrors=3` のサンプルで実行 | 連続エラーとして要対応に上がる |
| 6 | データなし | `運用/manifest.json` だけ置き `運用/状態/` を空にして実行 | 「運用あり・データなし（未起動/未送信）」と報告・落ちない |
| 7 | JSON 破損耐性 | 壊れた JSON を `運用/状態/` に置いて実行 | 1行の警告で続行・全体は落ちない |
| 8 | 読み取り専用の確認 | 上記実行後にファイル変更が無いか確認（git status、ファイル mtime） | chief-director がいかなるファイルも変更していない |
| 9 | 検証後の後片付け | サンプルの `運用/` を削除 | 検証用ファイルが残らない |
