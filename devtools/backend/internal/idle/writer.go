package idle

import "time"

// Writer は質問待ちの要約書き戻しを提供します。
// 現行の実装は要約を独立キャッシュへ書き戻す会話ログ方式（summaryCacheWriter）です。
type Writer interface {
	// WriteSummary は sessionID の待機に要約を付与します。
	// expectedTimestamp は要約対象を List した時点（T0）の待機開始 timestamp です。
	// expectedTimestamp を key に埋めることで compare-and-swap を担保し、
	// 要約実行中（数十秒）にユーザー回答→待機解消→同session新timestampで再生成が
	// 起きても、基準が T0 のため新しい待機へ旧要約を上書きしません（C1/C3）。
	WriteSummary(sessionID string, expectedTimestamp int64, summary string, at time.Time) error
}
