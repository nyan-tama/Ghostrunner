// Package tts provides a VOICEVOX Text-to-Speech proxy for the devtools backend.
//
// # Overview
//
// このパッケージは VOICEVOX Engine (ローカル HTTP サーバー) を backend で中継し、
// フロントエンドに audio/wav バイナリを返す HTTP プロキシを提供する。
// 目的は以下の二点である:
//
//  1. インメモリ LRU キャッシュ(SHA256 キー / TTL 24h / 50MB 上限)で
//     同一テキストの重複読み上げを抑制し、VOICEVOX への負荷を軽減する
//  2. 重複リクエストを singleflight で統合し、上流呼び出しの増幅を防ぐ
//
// VOICEVOX Engine はローカルで動作し、API キーは不要である。
// デフォルトで http://localhost:50021/ をリッスンする。
//
// # VOICEVOX 2-Stage API
//
// VOICEVOX は音声合成を 2 段階の HTTP 呼び出しで行う:
//
//  1. POST /audio_query?text=<urlencoded>&speaker=<id>
//     テキストから音声クエリ(アクセント・イントネーション等の中間表現)を生成する。
//     レスポンスは JSON (AudioQuery)。
//
//  2. POST /synthesis?speaker=<id>
//     audio_query の結果 JSON をボディに送り、WAV バイナリを取得する。
//
// handler + service + client + cache を 1 パッケージに集約しているのは、
// TTS が単機能・閉じたドメインで、client/cache が他パッケージから利用
// されないためである(grrun パッケージと同じ集約方針)。
//
// # Key Components
//
//   - [Service]: cache + singleflight + client を統合するビジネスロジック層。
//     [NewService] は常に非 nil を返す(API キー不要のため)。
//   - [Handler]: Gin ハンドラ。POST /api/tts を処理する。
//   - [Client]: VOICEVOX Engine の 2-stage API (audio_query + synthesis)
//     を呼び出す HTTP クライアント。
//   - [Cache]: LRU + TTL + バイト数上限のインメモリキャッシュ。
//     [NewLRUCache] で生成する。
//   - [UpstreamStatusError]: VOICEVOX 非 200 応答を表す型エラー。
//     Status / Body フィールドを持ち、errors.As で取り出して
//     [mapErrorToStatus] が HTTP ステータスへマッピングする。
//
// # Design Decisions
//
//   - キャッシュは LRU + TTL + バイト数のハイブリッド: エントリ数固定では
//     1 リクエストあたりのサイズ変動(数十 KB 〜 数 MB)に追従できない。
//     バイト数厳密管理 + LRU + TTL の 3 軸を持つ。
//   - singleflight 中の上流呼び出しは個別の context cancel に従わず、
//     最初に開始したリクエストの ctx が支配する。結果はキャッシュへ
//     書き込まれるため、後続の同一キーリクエストは即 hit に倒れる。
//   - エラーボディ(UpstreamStatusError.Body)は先頭 200 文字のみ保持。
//     ログにそのまま出しても安全にする。
//   - AudioQuery の JSON 構造体は定義しない。json.RawMessage で透過的に
//     受け渡す(VOICEVOX バージョン間の互換性を保つため)。
//
// # Logging
//
// ログ接頭辞は [TTSHandler] / [TTSService] の 2 種に集約する
// (既存 OpenAI が [OpenAIHandler] / [OpenAIService] の 2 接頭辞構成と整合)。
// client / cache の内部ログも [TTSService] 接頭辞で書く。
package tts
