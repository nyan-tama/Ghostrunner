// Package idle は質問待ち（アイドル）の型定義・プロジェクトマッチング・
// 要約キャッシュの読み書きを提供する。
//
// # 概要
//
// 統括ダッシュボードの各プロジェクトへ質問待ち状態を紐付けるための純粋ロジックと
// 型定義を提供する。質問待ちの検出（Reader 実装）は会話ログを直読みする transcript
// パッケージが担い、本パッケージは共通型（Marker / RawTail）とインターフェース
// （Reader / Writer）、cwd によるプロジェクトマッチング、要約キャッシュの永続化を持つ。
// 要約の書き戻しは独立キャッシュ（~/.claude/gr-idle-summaries）に対してのみ行い、
// 待機が変われば別 key となる compare-and-swap で古い要約の復活を防ぐ。
// TTL を超過した待機は失効扱いとして判定するのみで、実ファイルは削除しない。
//
// 旧来の Claude Code フック＋.idle マーカーファイル方式（NewReader / NewWriter）は
// 会話ログ直読み方式へ一本化したため撤去済みで、本パッケージにマーカーファイルの
// 読み書き実装は残っていない。
//
// # 主要な型
//
//   - Marker: 1セッションの質問待ち状態（cwd, session_id, epoch秒のtimestamp, 要約等）
//   - RawTail: 検出時点の会話末尾（要約前の生テキスト。lastAssistant / lastPrompt）
//   - Reader: 質問待ちの読み取りを抽象化するインターフェース（transcript が実装）
//   - Writer: 要約書き戻しを抽象化するインターフェース（summaryCacheWriter が実装）
//
// # 主要な関数
//
//   - NewSummaryCacheWriter: 要約を独立キャッシュ（~/.claude/gr-idle-summaries）へ書き戻す
//     Writer を生成する（会話ログ直読み方式の唯一の Writer 実装）。.idle マーカーを持たない
//     transcript 方式のため、key <sessionID>_<timestamp>.json の timestamp（=待機開始
//     entry-time）で compare-and-swap を担保し、待機が変われば別 key となり旧要約が復活しない。
//   - MergeSummaries: marker 群に対応するキャッシュを読み Summary/SummarizedAt を反映した
//     新スライスをイミュータブルに返す（reader の List 内で呼び Summary 込み Marker を返す契約）。
//   - PruneSummaryCache: 現存 marker 以外の孤児キャッシュを掃除する。
//   - CacheKey: 要約キャッシュのファイル名キー（<sessionID>_<timestamp>）を生成する。
//   - MatchProject: cwd がどの登録プロジェクトに属するかをパス前方一致で判定する。
//     複数一致時は最長一致（最も深いパス）を優先し、セグメント境界を担保する。
//   - IsExpired: 待機が TTL を超過して失効しているかを判定する。
//
// # 設計方針
//
//   - 書き込みは要約キャッシュへの付与のみ: 質問待ちの検出・解消は transcript 側の判定に委ね、
//     本パッケージは要約キャッシュを書くだけで待機状態そのものは生成・削除しない
//   - 要約書き戻しは compare-and-swap: key の timestamp で照合し解消済み待機へ旧要約を復活させない
//   - 壊れたJSON・読み取り失敗のキャッシュは warning ログを出してスキップし、全体を失敗させない
//   - TTL 失効はメモリ上の判定のみで、実ファイルには手を加えない
//   - プロジェクトマッチングは filepath.Clean をパス両辺に適用し、
//     セグメント境界を担保した前方一致で誤マッチ（/a/b と /a/bc）を防ぐ
package idle
