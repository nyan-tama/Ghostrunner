// Package transcript は Claude Code の会話ログ JSONL を直読みして質問待ち（アイドル）を
// 検出する idle.Reader 実装を提供する。
//
// # 概要
//
// フック（Stop/Notification 等）は環境（VS Code 拡張 / CLI）でイベント挙動が異なり、
// AskUserQuestion の回答待ちを取りこぼす。本パッケージはフックに依存せず、backend が
// ~/.claude/projects/<project-id>/<session-id>.jsonl を直接読み、「ノイズ行除外後の最終実質
// エントリが未応答の assistant」であるセッションを質問待ちとして idle.Marker 化する。
// これにより環境非依存で AskUserQuestion を含む全待機を捕捉する。
//
// # 主要な型・関数
//
//   - transcriptReader（idle.Reader 実装）: 登録プロジェクトの会話ログを走査し idle.Marker を返す
//   - NewReader: homeDir / projectsProvider / now を注入して Reader を生成する
//   - parseTail: 末尾 tailReadBytes だけを読み最終実質エントリから待機状態を判定する
//   - deriveProjectID / discoverSessions: 走査ディレクトリ絞り込み用の project-id と候補列挙
//   - parseCache: mtime 不変時の再パース抑制と entry-time 欠落版の署名→初回検出時刻の保持
//
// # 設計方針
//
//   - 環境非依存: フックのイベント種別に依存せず会話ログから直接待機を検出する
//   - best-effort: 会話ログは公式非サポート形式（バージョンで変わる）のため、抽出失敗時は
//     ParseOK=false として保守的に非待機へ倒す（誤検知回避を優先し取りこぼす方に倒す）。
//     新しい bookkeeping type（*-mode / *-state 等）が現れたら parse.go の noiseEntryTypes への
//     追加が必要（未追加は「未知＝実質エントリ」扱いで待機取りこぼしに直結する）
//   - C1: Marker.Timestamp は最後の assistant エントリ自身の entry-time。mtime は age/同一性に
//     使わず、parseCache の再パース抑制と終了セッションの粗い liveness ゲートにのみ用いる
//   - C2: セッション帰属は実 cwd + idle.MatchProject。lossy な project-id glob は走査絞り込み専用
//   - C3: 要約マージ（MergeSummaries）は List 内で行い Summary 込みの完成 Marker を返す契約。
//     Phase A は要約キャッシュを持たず常に空 Summary（フック位置のみ用意し実体は Phase B）
//   - 下流無改造流用: 1セッション1マーカーで返し、代表選定・SessionCount は dashboard 側に委ねる
package transcript
