// Package transcript は Claude Code の会話ログ JSONL を直読みして質問待ち（waiting）・
// 動作中（running）を検出する idle.Reader 実装を提供する。
//
// # 概要
//
// フック（Stop/Notification 等）は環境（VS Code 拡張 / CLI）でイベント挙動が異なり、
// AskUserQuestion の回答待ちを取りこぼす。本パッケージはフックに依存せず、backend が
// ~/.claude/projects/<project-id>/<session-id>.jsonl を直接読み、最終実質エントリの種別と
// mtime 鮮度を合成してセッションを質問待ち（waiting）/動作中（running）/静観（none）に分類する。
// プロジェクトごとに最新 mtime のセッション1件を代表として collapse し、waiting/running のみ
// idle.Marker 化する（none はマーカー化しない）。実質エントリは user/assistant の allowlist で判定し、
// 末尾に追記される帳簿型（ai-title/last-prompt/*-mode/*-state 等）は自動で無視する。
// これにより環境非依存で AskUserQuestion を含む全待機と作業中セッションを捕捉する。
//
// # 主要な型・関数
//
//   - transcriptReader（idle.Reader 実装）: 登録プロジェクトの会話ログを走査し代表 idle.Marker を返す
//   - NewReader: homeDir / projectsProvider / now を注入して Reader を生成する
//   - parseTail: 末尾 tailReadBytes だけを読み最終実質エントリの種別（tailKind）を判定する
//   - classifyRepresentative: 種別（内容）と mtime 鮮度を合成して最終 status を確定する純粋関数
//   - deriveProjectID / discoverSessions: 走査ディレクトリ絞り込み用の project-id と候補列挙
//   - parseCache: mtime 不変時の再パース抑制と entry-time 欠落版の署名→初回検出時刻の保持
//
// # 設計方針
//
//   - 環境非依存: フックのイベント種別に依存せず会話ログから直接待機を検出する
//   - best-effort: 会話ログは公式非サポート形式（バージョンで変わる）のため、抽出失敗時は
//     ParseOK=false として保守的に非待機へ倒す（誤検知回避を優先し取りこぼす方に倒す）。
//     「最終実質エントリ」は parse.go の substantiveEntryTypes（user/assistant の allowlist）で判定し、
//     新しい bookkeeping type（ai-title/last-prompt/*-mode/*-state 等）が現れても自動で無視される。
//     denylist を保守し続ける必要はなく、列挙漏れによる待機取りこぼし（false-negative）が構造的に起きない
//   - 種別と鮮度の責務分担: parse は最終実質エントリの内容だけを tailKind（waiting/midTurn/none）に
//     分類し、reader が classifyRepresentative で mtime 鮮度を合成して最終 status を確定する。
//     しきい値（idle.MinAge=60s / idle.BusyThreshold=45s / idle.RunningMaxAge / idle.TTL）は
//     idle パッケージに集約（SSOT）し、kind 別境界で 45〜60秒のデッドゾーンを作らない（C-2）
//   - 代表 collapse: プロジェクトごとに最新 mtime のセッション1件を代表とし、waiting/running のみ
//     Marker 化する（none は skip）。Marker.SessionCount は代表と同一 status のセッション数
//   - Marker.Timestamp の意味は Status で分岐: waiting は最後の assistant エントリの entry-time
//     （要約 key の安定同一性・C1。mtime は使わない）、running は代表セッションの mtime
//   - C2: セッション帰属は実 cwd + idle.MatchProject。lossy な project-id glob は走査絞り込み専用
//   - C3: 要約マージ（MergeSummaries）は List 内で行い Summary 込みの完成 Marker を返す契約。
//     要約は waiting のみが対象で、孤児キャッシュ掃除の aliveKeys も waiting marker のみで構築する（W-2）
package transcript
