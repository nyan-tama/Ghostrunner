package idle

import "time"

// このファイルはセッション状態分類のしきい値を集約する単一の定義源（SSOT）です。
// transcript（会話ログ直読み）と dashboard（状態集約）の双方が本パッケージを import しており、
// しきい値をここへ集約することでパッケージ間の定数重複（旧来 idleTTL が両パッケージに二重定義
// されていた轍）を解消します。分類ロジックは transcript.classifyRepresentative が担い、
// dashboard は同じ TTL / MinAge を参照して整合させます。
const (
	// TTL はセッションを終了扱いとする鮮度上限です。
	// transcript では mtime がこれより古いセッションを走査から除外する粗い liveness ゲート、
	// dashboard では待機マーカーの失効判定（IsExpired の ttl 引数）に用います。
	TTL = 6 * time.Hour

	// MinAge は質問待ち（waiting）と見なす最小滞留時間です。
	// waiting-kind のセッションはこれ未満なら「応答直後でユーザーが読んでいる最中」とみなして
	// running（動作中）扱いにし、これ以上で waiting（質問待ち）へ一方向に遷移します（境界は 60s で一本化）。
	MinAge = 60 * time.Second

	// BusyThreshold は none-kind（応答済/解釈不能）のセッションを mtime 鮮度だけで running とみなす窓です。
	// none-kind でも mtime がこれ未満なら「今まさに書き込み中の稼働セッション」とみなし running にします。
	BusyThreshold = 45 * time.Second

	// RunningMaxAge は midTurn（未応答通常tool_use / thinking / user末尾）を running とみなす鮮度上限です。
	// これを超えた midTurn は Ctrl-C / クラッシュで固まった last=tool_use とみなし none へ倒します
	// （固まったセッションが最大 TTL(6h) にわたり青表示され続けるのを防ぐ）。
	RunningMaxAge = 10 * time.Minute
)
