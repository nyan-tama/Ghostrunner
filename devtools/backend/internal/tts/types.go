package tts

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// パッケージ定数
const (
	// MaxTextLength はリクエスト text の最大文字数(rune 単位)
	MaxTextLength = 2000

	// DefaultBaseURL は VOICEVOX Engine のデフォルトアドレス
	DefaultBaseURL = "http://localhost:50021"

	// DefaultSpeakerID は VOICEVOX_SPEAKER_ID 未設定時のデフォルト(春日部つむぎ)
	DefaultSpeakerID = 8

	// DefaultOutputFormat は VOICEVOX から取得する音声フォーマット
	DefaultOutputFormat = "wav"

	// CacheMaxBytes はキャッシュ全体のバイト数上限(50MB)
	CacheMaxBytes = 50 * 1024 * 1024

	// CacheTTL はキャッシュエントリの有効期限
	CacheTTL = 24 * time.Hour

	// HTTPTimeout は VOICEVOX Engine への HTTP リクエストタイムアウト(2-stage 合計)
	HTTPTimeout = 60 * time.Second
)

// AudioQuery は VOICEVOX audio_query エンドポイントが返す JSON を透過的に保持する型です。
// 構造体として定義しない理由: VOICEVOX バージョン間の互換性を保ち、
// synthesis にそのまま渡すだけなので中間解析が不要なため。
type AudioQuery = json.RawMessage

// TTSRequest は POST /api/tts のリクエストボディです。
type TTSRequest struct {
	Text string `json:"text"`
}

// TTSErrorResponse はエラー時のレスポンス JSON 形式です。
// 成功時は audio/wav バイナリを直接返すため、この型は使いません。
type TTSErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Config は VOICEVOX クライアントの設定値です。
type Config struct {
	Host      string
	SpeakerID int
}

// SynthesizeParams はサービス/クライアント内部で受け渡す合成パラメータです。
type SynthesizeParams struct {
	Text         string
	SpeakerID    int
	OutputFormat string
}

// SynthesizeResult はサービスの合成結果です。
type SynthesizeResult struct {
	Audio       []byte
	ContentType string
	FromCache   bool
}

// UpstreamStatusError は VOICEVOX が非 200 を返したことを示すエラーです。
// errors.As で取り出して Status を見ることで、ハンドラ側で適切な HTTP
// ステータスへマッピングできます。Body には API レスポンスボディの先頭
// 200 文字のみを保持します。
type UpstreamStatusError struct {
	Status int
	Body   string
}

// Error は error インターフェースを満たします。
func (e *UpstreamStatusError) Error() string {
	return fmt.Sprintf("voicevox upstream returned status %d: %s", e.Status, e.Body)
}

// センチネルエラー: 用途別に errors.Is で判定する。
var (
	// ErrTextEmpty は text が空であることを表します。
	ErrTextEmpty = errors.New("text is empty")

	// ErrTextTooLong は text が MaxTextLength を超えたことを表します。
	ErrTextTooLong = errors.New("text exceeds max length")

	// ErrUpstreamTimeout は VOICEVOX へのリクエストがタイムアウトまたは接続拒否されたことを表します。
	// context.DeadlineExceeded, net.Error.Timeout(), syscall.ECONNREFUSED をすべて包含する。
	ErrUpstreamTimeout = errors.New("voicevox request timeout or connection refused")

	// ErrInvalidContentType は VOICEVOX の synthesis から audio/wav 以外が返ったことを表します。
	ErrInvalidContentType = errors.New("voicevox returned non audio/wav content type")
)
