// Package tts provides an ElevenLabs Text-to-Speech proxy for the devtools backend.
//
// # Overview
//
// このパッケージは ElevenLabs API を backend で中継し、フロントエンドに
// audio/mpeg バイナリを返す HTTP プロキシを提供する。目的は以下の三点である:
//
//  1. ElevenLabs API キーを backend に隔離してフロントへ漏らさない
//  2. インメモリ LRU キャッシュ(SHA256 キー / TTL 24h / 50MB 上限)で
//     同一テキストの重複読み上げを抑制し、課金事故を防ぐ
//  3. 重複リクエストを singleflight で統合し、上流呼び出しの増幅を防ぐ
//
// handler + service + client + cache を 1 パッケージに集約しているのは、
// TTS が単機能・閉じたドメインで、client/cache が他パッケージから利用
// されないためである(grrun パッケージと同じ集約方針)。
//
// # Key Components
//
//   - [Service]: cache + singleflight + client を統合するビジネスロジック層。
//     [NewService] は環境変数 ELEVENLABS_API_KEY 未設定時 nil を返す
//     (既存 OpenAIService と同型のオプショナル機能パターン)。
//   - [Handler]: Gin ハンドラ。[NewHandler] に nil Service を渡しても
//     初期化は成功し、リクエスト時に 503 を返す(無条件登録パターン)。
//   - [Client]: ElevenLabs non-stream エンドポイント
//     POST /v1/text-to-speech/{voice_id} を呼び出す HTTP クライアント。
//   - [Cache]: LRU + TTL + バイト数上限のインメモリキャッシュ。
//     [NewLRUCache] で生成する。
//   - [UpstreamStatusError]: ElevenLabs 非 200 応答を表す型エラー。
//     Status / Body フィールドを持ち、errors.As で取り出して
//     [mapErrorToStatus] が HTTP ステータスへマッピングする。
//
// # Design Decisions
//
//   - non-stream エンドポイントを採用: 一括方式採用が確定済で
//     stream を使っても backend で読み切るならメリットなし。
//     非 stream の方が Content-Length が確定し扱いやすい。
//   - キャッシュは LRU + TTL + バイト数のハイブリッド: エントリ数固定では
//     1 リクエストあたりのサイズ変動(数十 KB〜数 MB)に追従できない。
//     バイト数厳密管理 + LRU + TTL の 3 軸を持つ。
//   - singleflight 中の上流呼び出しは個別の context cancel に従わず、
//     最初に開始したリクエストの ctx が支配する。結果はキャッシュへ
//     書き込まれるため、後続の同一キーリクエストは即 hit に倒れる。
//   - エラーボディ(UpstreamStatusError.Body)は先頭 200 文字のみ保持。
//     API キー混入リスクを避け、ログにそのまま出しても安全にする。
//   - 上流 401(キー無効)は backend ログのみで把握し、フロントには 502 に
//     丸める(攻撃者にキー状態を漏らさないため)。
//   - [TTSRequest] は MVP+ では text のみ受け取り、voice_id / model_id は
//     backend env 固定。フロント↔バックエンド契約の片側化を防ぎ、将来の
//     Voice 選択 UI 追加時(MVP++)に camelCase で voiceId/modelId を足す
//     拡張余地を残す。
//
// # Logging
//
// ログ接頭辞は [TTSHandler] / [TTSService] の 2 種に集約する
// (既存 OpenAI が [OpenAIHandler] / [OpenAIService] の 2 接頭辞構成と整合)。
// client / cache の内部ログも [TTSService] 接頭辞で書く。
// API キー文字列は絶対にログに含めない。
package tts
