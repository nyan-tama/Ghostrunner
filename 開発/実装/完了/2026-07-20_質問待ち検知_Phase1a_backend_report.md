# 質問待ち検知の統括ダッシュボード反映 - Phase 1a バックエンド実装レポート

対象計画: `開発/実装/実装待ち/2026-07-20_質問待ち検知の統括ダッシュボード反映_plan.md`

---

## 実装完了レポート

### 実装サマリー
- **実装日**: 2026-07-20
- **対象**: `devtools/backend/` の Phase 1a（バックエンド）のみ
- **スコープ**: フックが書く質問待ちマーカーを読み取り、`GET /api/dashboard/state` の各プロジェクトに `idle` オブジェクトを付与。質問待ちプロジェクトを Attention `required` へ昇格し、`idle` の有無を第1キーにソート
- **対象外（Phase 1b 以降）**: grasp CLI の `[質問待ち]` 表示、要約ジョブ（`claude -p --model haiku`）、`/api/dashboard/stream`（SSE）、Web カード。frontend も対象外
- **テスト結果**: `go build` / `go vet` / `gofmt` クリーン、`go test ./...` 全パス、新規/追記 24 ケース

### 概要（何を実装したか）

Claude Code のフックが `~/.claude/gr-idle-markers/<session_id>.idle` に書き込む「質問待ちマーカー」を、バックエンドが読み取り専用でロードし、統括ダッシュボードの各プロジェクトに `idle` オブジェクトとして付与する。付与されたプロジェクトは Attention を `required` に再評価し、ソートで最上位（`idle` 有無を独立第1キー）へ並べる。これにより「返事を待たせているプロジェクトの取りこぼし」を `/api/dashboard/state` のレスポンス上で可視化する。

要約生成・SSE 配信・grasp CLI/Web の表示は Phase 1b 以降に分離した（go-planner 推奨の Phase 分割）。Phase 1a は要約前の `preview`（`rawTail.lastAssistant` 先頭 80 字）を暫定表示し、`summary`/`summarizedAt` は空で返す。

### 変更ファイル一覧

| ファイル | 変更種別 | 変更内容 |
|---------|---------|---------|
| `internal/idle/marker.go` | 新規 | `Marker`/`RawTail` 型、`Reader` インターフェース、`fileReader.List`（`markerDir/*.idle` を Glob→ReadFile→Unmarshal、壊れ JSON・読み取り失敗は warning ログでスキップし全体を落とさない、markerDir 不在は空スライス）、`NewReader` |
| `internal/idle/match.go` | 新規 | `MatchProject`（`filepath.Clean` 両辺適用・完全一致/セグメント境界前方一致・最長一致優先。W7）、`IsExpired`（epoch 秒 timestamp と now の差が TTL 超過で true） |
| `internal/idle/doc.go` | 新規 | パッケージドキュメント（読み取り専用・壊れマーカー skip・TTL 失効は判定のみで削除しない・パス境界担保の設計方針） |
| `internal/idle/marker_test.go` | 新規 | `List` のテーブル駆動テスト（正常ロード・壊れ JSON skip・空/ディレクトリ不在。本番マーカー形状の回帰 fixture を含む） |
| `internal/idle/match_test.go` | 新規 | `MatchProject`（完全一致・サブディレクトリ・最長一致・パス境界誤マッチ防止・未正規化・不一致）と `IsExpired`（期限内/期限切れ/境界）のテーブル駆動テスト |
| `internal/dashboard/types.go` | 変更 | `IdleState` 型を追加（`timestamp`/`preview`/`sessionCount`/`summary`/`summarizedAt`）。`ProjectState` に `Idle *IdleState`（JSON `idle,omitempty`）を追加 |
| `internal/dashboard/service.go` | 変更 | `serviceImpl` に `idleReader idle.Reader` を注入（nil 許容）。`NewService`/`NewServiceWithClock` のシグネチャ変更。`GetState` 内で `attachIdleState` を呼び各プロジェクトへ集約付与（TTL 失効は除外）、付与後に `determineAttention` を再評価（C1）。`sort.SliceStable` の比較関数を「idle 存在 desc → attention 優先度 asc → 経過時間 desc → isSelf asc → name asc」に改修（C2）。内部ヘルパー `idleElapsed`/`truncateRunes`/`attentionPriority` を追加。定数 `idleTTL = 6h` |
| `internal/dashboard/scanner.go` | 変更 | `determineAttention` 先頭に「`state.Idle != nil` なら `AttentionRequired`」を追加（C1・既存の未回答/ops 分岐は非破壊） |
| `internal/dashboard/doc.go` | 変更 | 質問待ち集約・Attention 再評価・ソートの記述を反映 |
| `internal/dashboard/scanner_test.go` | 変更 | `determineAttention` の `Idle` 入りケース（idle→required）を追加、既存挙動の回帰確認 |
| `internal/dashboard/service_test.go` | 変更 | 付与後 Attention 再評価・idle 第1キーソート・idle 同士は経過降順・TTL 失効除外・`idleReader` nil 許容・マーカー 0 件時の既存順不変を追記 |
| `cmd/server/main.go` | 変更 | `os.UserHomeDir()` から `markerDir = ~/.claude/gr-idle-markers` を解決、`idle.NewReader` を生成して `dashboard.NewService` に注入 |
| `internal/handler/doc.go` | 変更 | ダッシュボードレスポンスに `idle` が含まれる旨を反映 |
| `docs/BACKEND_API.md` | 変更 | `GET /api/dashboard/state` レスポンスに `idle` フィールドとデータ契約を追記 |
| `.claude/hooks/mark-idle.sh` | 変更 | 契約整合の修正（下記「レビューで見つかった重要バグ」参照）。入れ子 `rawTail{lastAssistant,lastPrompt}` ＋ `summary`/`summarizedAt` を出力する形へ整形 |

### 実装の要点

#### レビュー反映（go-reviewer / plan-reviewer 確定分）
- **C1（Attention 昇格の順序）**: `determineAttention` は `ScanProject` 時点では `Idle` が nil。そこで `determineAttention` 先頭に `Idle != nil → required` を追加しつつ、`service.go` の `attachIdleState` が `Idle` を付与した**後に** `state.Attention = determineAttention(state)` を再評価する二段構えにした。
- **C2（ソートの第1キー）**: ソートは `service.go` の `GetState` 内 `sort.SliceStable`。`Idle` の有無を**独立した第1キー**にし、「質問待ち > 未回答由来 required」を分離。以降は attention 優先度 asc → 経過時間 desc → isSelf asc → name asc。経過時間はソート内部でのみ計算し State には露出しない（W1）。
- **W5（`NewService` nil 許容）**: `idleReader` を nil 許容とし、nil の場合は付与をスキップ。既存 `service_test.go` の呼び出し改修を最小化。
- **W7（パス境界）**: `MatchProject` は `filepath.Clean` を両辺に適用し `cwd == projPath || strings.HasPrefix(cwd, projPath+separator)` でセグメント境界を担保（`/a/b` が `/a/bc` に誤マッチしない）。

#### データ契約
- `idle` は `timestamp`（RFC3339）/`preview`/`sessionCount`/`summary`/`summarizedAt` を持つ。**`waiting` bool と `waitingMinutes` は持たない**（キー存在自体が質問待ちを意味し、経過分はフロントが `now - timestamp` で算出）。
- `idle,omitempty` により非待機時はキーごと欠落（後方互換・フロントは optional chaining）。
- 代表選定は 1 プロジェクト複数セッション時に**最古 timestamp（＝最長待機）を代表**とし、`sessionCount` に該当件数を保持。
- TTL は 6 時間。超過マーカーは**除外するのみで実ファイルは削除しない**（読み取り専用）。

### レビューで見つかった重要バグと解消

go-reviewer が Critical として検出: **`mark-idle.sh` はフラットな `lastAssistant`/`lastPrompt` を書くのに、Go 側は入れ子の `rawTail` を読む契約ズレ**があり、単体テストは通るが**本番では `preview` が常に空**になる。マーカーのデータ契約（フック書き込み ↔ バックエンド読み取り）の不整合。

解消: `.claude/hooks/mark-idle.sh` を、Go の `Marker`/`RawTail` 契約に合わせて `rawTail:{lastAssistant, lastPrompt}` の入れ子 ＋ `summary:""`/`summarizedAt:""` を出力する形に整形。実マーカーで `preview` が入ることを検証し、**本番マーカー形状の回帰 fixture を `marker_test.go` に追加**して再発を防止した。

### テスト結果
- `go build ./...` / `go vet ./...` / `gofmt` すべてクリーン
- `go test ./...` 全パス
- 新規/追記テストケース: 24（`idle` パッケージの `List`・`MatchProject`・`IsExpired`、`dashboard` の `determineAttention` 昇格・`attachIdleState` 再評価・第1キーソート・経過降順・TTL 失効除外・`idleReader` nil 許容・既存順不変）

### 実装時の課題

- **契約ズレバグ**（上記）が最大の論点。単体テスト green と本番挙動の乖離だったため、フック側スクリプトの契約整合修正と回帰 fixture の追加で担保した。それ以外のビルド・テストで苦戦した点は特になし。

### 残存する懸念点

- **フックの `settings.json` 恒久化が未完**。現状フック（`mark-idle.sh`/`unmark-idle.sh`）の登録は POC ブランチ `feat/idle-marker-poc` に設定済みで、恒久設定は Phase 1b 以降で本反映する。フックはセッション開始時ロードのため、恒久化後の新規/再起動セッションで有効になる。
- **会話ログ JSONL は公式非サポート**。Claude Code のバージョン差で `transcript_path` の形式が変わると `preview` が空になりうる（抽出失敗時は空フォールバックで盤本体は無事）。
- **`preview` は暫定**。Phase 1a は `rawTail.lastAssistant` 先頭 80 字のみで、日本語 1 行要約（`summary`）は Phase 1b で差し替え予定。
- **TTL 掃除は未実装**。6h 超マーカーは無視するだけで実ファイルは残る。実削除の掃除ジョブは将来対応。

### 動作確認フロー

```
1. cd devtools/backend && go test ./... で全テストがパスすること
2. make backend でdevtoolsバックエンドを起動
3. 質問待ちマーカーを配置:
   - 手動: ~/.claude/gr-idle-markers/<任意>.idle に
     {cwd, session_id, timestamp(epoch秒), rawTail:{lastAssistant,lastPrompt}, summary:"", summarizedAt:""}
     を置く（cwd は登録プロジェクトの絶対パス配下）
   - 実地: 別セッションを実際に質問待ち状態にし、フックにマーカーを書かせる
4. curl http://localhost:8888/api/dashboard/state を実行し、
   - 該当プロジェクトに idle オブジェクトが付いている（timestamp/preview/sessionCount、summary/summarizedAt は空）
   - 該当プロジェクトの attention が "required" になっている
   - projects 配列の上位（idle あり優先）に並んでいる
   ことを確認する
5. TTL 確認: timestamp を 6h 超に設定したマーカーは idle が付かず、ソート/Attention に影響しないこと
```

### デプロイ後の確認事項

- [ ] `cd devtools/backend && go test ./...` で全テストがパスすること
- [ ] `cd devtools/backend && go vet ./...` で警告がないこと
- [ ] `cd devtools/backend && go build ./cmd/server` でビルドが通ること
- [ ] `GET /api/dashboard/state` が既存プロジェクトで従来どおり応答すること（マーカー 0 件時の後方互換）
- [ ] 実マーカー投入時に `idle.preview` が空でなく入ること（契約ズレ回帰の確認）
- [ ] 質問待ちプロジェクトが `attention=required` かつ上位にソートされること
- [ ] フックの `settings.json` 恒久化（Phase 1b）を実施すること

### 残作業（Phase 1b 以降）

- grasp CLI（bash）の `[質問待ち]` 表示（`/api/dashboard/state` の `idle` を読み最上段・赤で表示）
- 要約ジョブ（`claude -p --model haiku` で日本語 1 行要約を生成し `summary`/`summarizedAt` を書き戻し。滞留マーカーのみ・1 回だけ）
- SSE 化（`/api/dashboard/stream`・`DashboardState` スナップショット全体を配信）
- Web カード（`WaitingBadge` 等・frontend）
- フックの `settings.json` 恒久化（現状 `feat/idle-marker-poc` ブランチに設定済み）
