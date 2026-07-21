// Package idle は質問待ち（アイドル）マーカーの読み取り・プロジェクトマッチング・
// 要約書き戻しを提供する。
//
// # 概要
//
// Claude Code のフックが ~/.claude/gr-idle-markers/<session_id>.idle に書き込む
// 質問待ちマーカーファイルを読み取り専用でロードし、cwd を手がかりに統括ダッシュボードの
// 各プロジェクトへ紐付けるための純粋ロジックを提供する。
// マーカーの新規作成・削除はフック側の責務であり、本パッケージはこれらを行わない。
// 唯一の書き込みは既存マーカーへの要約（summary / summarizedAt）の付与のみで、
// フックが削除/更新したマーカーを復活させないよう compare-and-swap ガードを持つ。
// TTL を超過したマーカーは失効扱いとして判定するのみで、実ファイルは削除しない。
//
// # 主要な型
//
//   - Marker: 1セッションの質問待ちマーカー（cwd, session_id, epoch秒のtimestamp, 要約等）
//   - RawTail: マーカー書き込み時点の会話末尾（要約前の生テキスト。lastAssistant / lastPrompt）
//   - Reader: 質問待ちマーカーの読み取りを抽象化するインターフェース
//   - Writer: 既存マーカーへの要約書き戻しを抽象化するインターフェース
//
// # 主要な関数
//
//   - NewReader: markerDir 配下の *.idle を読む Reader を生成する。
//     壊れたファイルや読み取り失敗はスキップし、markerDir 不在時は空スライスを返す。
//   - NewWriter: 既存マーカーへ要約を書き戻す Writer を生成する（フック方式）。
//     WriteSummary は List 時点（T0）の timestamp を基準に compare-and-swap で
//     temp+rename する。不在/不一致（同session新timestamp含む）なら書き戻しを破棄する。
//   - NewSummaryCacheWriter: 要約を独立キャッシュ（~/.claude/gr-idle-summaries）へ書き戻す
//     Writer を生成する（会話ログ直読み方式）。.idle マーカーを持たない transcript 方式の
//     ため、key <sessionID>_<timestamp>.json の timestamp（=待機開始 entry-time）で
//     compare-and-swap を担保し、待機が変われば別 key となり旧要約が復活しない。
//     Writer 実装はマーカー / キャッシュの2種が並存する（配線側で一方を注入する）。
//   - MergeSummaries: marker 群に対応するキャッシュを読み Summary/SummarizedAt を反映した
//     新スライスをイミュータブルに返す（reader の List 内で呼び Summary 込み Marker を返す契約）。
//   - PruneSummaryCache: 現存 marker 以外の孤児キャッシュを掃除する。
//   - CacheKey: 要約キャッシュのファイル名キー（<sessionID>_<timestamp>）を生成する。
//   - MatchProject: cwd がどの登録プロジェクトに属するかをパス前方一致で判定する。
//     複数一致時は最長一致（最も深いパス）を優先し、セグメント境界を担保する。
//   - IsExpired: マーカーが TTL を超過して失効しているかを判定する。
//
// # 設計方針
//
//   - 新規作成・削除はしない: マーカーの生成/削除はフック側の責務。書き込みは要約付与のみ
//   - 要約書き戻しは compare-and-swap: rename 前に基準 timestamp と照合し解消済みを復活させない
//   - 壊れたJSON・読み取り失敗のマーカーは warning ログを出してスキップし、全体を失敗させない
//   - TTL 失効はメモリ上の判定のみで、実ファイルには手を加えない
//   - プロジェクトマッチングは filepath.Clean をパス両辺に適用し、
//     セグメント境界を担保した前方一致で誤マッチ（/a/b と /a/bc）を防ぐ
package idle
