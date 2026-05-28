# even-terminal ポート固有化（G2 ホストズレの根本対策）検討

## 背景・課題

`make g2-all` は `patrol_projects.json` の **index 順に port 3456 から連番**で even-terminal を起動している。そのため `patrol_projects.json` の順序変更・追加で各プロジェクトのポートが動的に変わる。

G2 アプリは URL の `name` パラメータより**登録時の入力名を優先**して保持するため、後からポート割当が変わるとズレが固定化される。

実例（2026-05-28 検知）: G2 アプリで「x-liker」と命名したホストが、再起動後は port 3457 (face-search) を指していた。

### 核心: ポート台帳がない

現状の `/init` はポートを生成して各ファイル（Makefile, .env 等）に**埋め込むだけ**で、「このプロジェクトの even-terminal ポートは何番か」を**後から引ける場所がない**。だから `make g2-all` は index 連番にせざるを得なかった。

even-terminal ポートを `/init` で割り当てても、`make g2-all` がそれを**読める台帳**がないと無意味。台帳 = `.claude/ports.json`。

## 目的

**新規プロジェクトは生成時点で even-terminal ポートが固定され、G2 のズレが根本から起きない**ようにする。既存プロジェクトも台帳を1つ置けば恩恵を受けられる。

## 確定方針（対話で合意済み）

| 論点 | 確定 | 備考 |
|---|---|---|
| even-terminal の先頭桁 | **`4xxx`** | 既存予約: Backend=8, Frontend=3, DB=5, MinIO=9(+1), Redis=6 |
| ポート保存場所 | **`.claude/ports.json`**（台帳） | プロジェクト側に新設 |
| ポート番号ルール | **下3桁共通**（既存仕様準拠） | `4${PORT_SUFFIX}`。x-liker なら 4909 |
| `.claude/ports.json` の位置づけ | **二重管理**（SSOT にしない） | Makefile 等は従来のプレースホルダー方式維持。台帳は併設 |
| スコープ | **MVP+** | 新規根本対策 + x-liker 恒久固定 |

## スコープ（MVP+）

### 範囲内

1. **`/init` 改修**
   - ポート生成に `PORT_EVEN_TERMINAL=4${PORT_SUFFIX}` を追加
   - 予約除外リストに 4xxx の著名ポート（`4040`, `4444`）を追加
   - `.claude/ports.json` を生成（台帳）。新規プロジェクトは生成時点で even-terminal ポートが確定
2. **`make g2-all` 改修**
   - 各プロジェクトの `.claude/ports.json` を読み、`even_terminal` ポートがあればそれで起動
   - **なければ従来の index 連番（3456+i）にフォールバック**（＝既存プロジェクトは無改修で動く・後方互換）
   - QR 表示にポート番号とプロジェクトパスを併記（登録時の照合を容易に）
3. **x-liker の恒久固定**
   - x-liker に `.claude/ports.json`（`{"even_terminal": 4909}` 等）を**手で1つ置く**
   - これだけで make g2-all 改修の恩恵を受け、port 4909 に固定される

### 範囲外（やらない）

- 既存プロジェクトの一斉移行（x-liker 以外の face-search / about_create_game / auto-daysupport-cloudrun / Ghostrunner は触らない＝従来の index 連番のまま）
- `devtools/backend/internal/projects/loader.go` の schema 拡張（make g2-all は Makefile 内で直接 `.claude/ports.json` を読むため、Go 側は介さない）
- `patrol_projects.json` の schema 拡張（`even_terminal_port` フィールドは追加しない。台帳は各プロジェクトの `.claude/ports.json` に一本化）
- G2 アプリ側の登録挙動改善（Ghostrunner 外）

## 設計判断

### 判断1: フォールバックは index 連番（スキップ警告にしない）

前回計画では「index 連番フォールバックはズレ再発するから避ける」としたが、MVP+ では方針を変える。

- 新規プロジェクト → `.claude/ports.json` を持つ → 固定ポート（ズレない）
- 既存プロジェクト → 台帳なし → 従来の index 連番（既存を壊さない・後方互換）

**理由**: スキップ警告にすると既存4プロジェクトが起動しなくなる。MVP+ では「新規はズレない、既存は従来通り（必要なら個別に台帳を置く）」で十分。

### 判断2: 台帳と patrol_projects.json の役割分離

- `patrol_projects.json`: **どのプロジェクトを統括するか**（path + name のリスト。Ghostrunner 独自・gitignore）
- `.claude/ports.json`: **そのプロジェクトのポート群**（各プロジェクト配下の台帳）

`make g2-all` = patrol_projects.json でプロジェクト列挙 → 各 `.claude/ports.json` でポート取得。クリーンな分離。

### 判断3: `.claude/ports.json` のキー構成

- 必須: `backend`, `frontend`, `even_terminal`
- 選択時のみ: `db`（PostgreSQL）, `minio` / `minio_console`（ストレージ）, `redis`（Redis）

### 判断4: ポートの混在は許容

x-liker が台帳で 4909 を使い、他の既存が index 連番（3456〜3459）を使う混在状態になるが、ポート番号が衝突しないため問題なし。index はあくまで「台帳がない場合のフォールバック値」（配列位置ベース）。

## 既存プロジェクトの扱い（参考）

| プロジェクト | サフィックス | 既存ポート | 今回の対応 |
|---|---|---|---|
| x-liker | 909（仕様準拠） | 8909/3909 | **`.claude/ports.json` を手で配置（4909）** ← MVP+ で実施 |
| face-search | 未割当 | 8080/3000 | 触らない（index 連番のまま） |
| about_create_game | 未割当 | - | 触らない |
| auto-daysupport-cloudrun | 081/001 | 8081/3001 | 触らない |
| Ghostrunner | devtools 専用 | 8888/3333 | 触らない |

他の既存プロジェクトも恒久固定したくなったら、同様に `.claude/ports.json` を1つ置くだけ（運用でカバー）。

## 次のステップ

`/plan` で実装計画を作成する。計画化のポイント:

- `/init` SKILL.md の変更箇所（Step 4 ポート生成、Step 7 `.claude/ports.json` 生成）
- `Makefile` の `g2-all` / `g2-qr` / `stop-g2-all` 改修（Python ヘルパーで台帳優先・index 連番フォールバック）
- x-liker への `.claude/ports.json` 配置手順
- 検証手順（`/init` で新規生成 → `.claude/ports.json` 確認、`make g2-all` で固定ポート起動確認、G2 再登録確認）

## 確認事項

なし（スコープ・設計判断ともに対話で確定済み）。
