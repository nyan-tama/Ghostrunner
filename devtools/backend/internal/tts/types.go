package tts

import (
	"errors"
	"fmt"
	"time"
)

// パッケージ定数
const (
	// MaxTextLength はリクエスト text の最大文字数
	MaxTextLength = 2000

	// DefaultOutputFormat は ElevenLabs から取得する音声フォーマット
	DefaultOutputFormat = "mp3_44100_128"

	// CacheMaxBytes はキャッシュ全体のバイト数上限(50MB)
	CacheMaxBytes = 50 * 1024 * 1024

	// CacheTTL はキャッシュエントリの有効期限
	CacheTTL = 24 * time.Hour

	// HTTPTimeout は ElevenLabs API への HTTP リクエストタイムアウト
	HTTPTimeout = 30 * time.Second

	// DefaultVoiceID は VOICE_ID 未設定時のフォールバック(Romaco)
	DefaultVoiceID = "KgETZ36CCLD1Cob4xpkv"

	// DefaultModelID は MODEL 未設定時のフォールバック
	DefaultModelID = "eleven_flash_v2_5"

	// DefaultBaseURL は ElevenLabs API のベース URL
	DefaultBaseURL = "https://api.elevenlabs.io"
)

// TTSRequest は POST /api/tts のリクエストボディです。
// MVP+ では text のみ受け取り、voice_id/model_id は backend env 固定で扱います。
type TTSRequest struct {
	Text string `json:"text"`
}

// TTSErrorResponse はエラー時のレスポンス JSON 形式です。
// 成功時は audio/mpeg バイナリを直接返すため、この型は使いません。
type TTSErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Config は ElevenLabs クライアント/サービスの設定値です。
type Config struct {
	APIKey         string
	DefaultVoiceID string
	DefaultModelID string
	BaseURL        string
}

// SynthesizeParams はサービス/クライアント内部で受け渡す合成パラメータです。
// Handler 層が env デフォルトで空フィールドを埋めて Service に渡し、
// Service が cacheKey 計算と Client 呼出に利用します。
type SynthesizeParams struct {
	Text         string
	VoiceID      string
	ModelID      string
	OutputFormat string
}

// SynthesizeResult はサービスの合成結果です。
type SynthesizeResult struct {
	Audio       []byte
	ContentType string
	FromCache   bool
}

// UpstreamStatusError は ElevenLabs が非 200 を返したことを示すエラーです。
// errors.As で取り出して Status を見ることで、ハンドラ側で適切な HTTP
// ステータスへマッピングできます。Body には API レスポンスボディの先頭
// 200 文字のみを保持し、API キー混入リスクを抑えます。
type UpstreamStatusError struct {
	Status int
	Body   string
}

// Error は error インターフェースを満たします。
func (e *UpstreamStatusError) Error() string {
	return fmt.Sprintf("elevenlabs upstream returned status %d: %s", e.Status, e.Body)
}

// センチネルエラー: 用途別に errors.Is で判定する。
var (
	// ErrAPIKeyMissing は ELEVENLABS_API_KEY 未設定時を表します。
	ErrAPIKeyMissing = errors.New("elevenlabs api key is not set")

	// ErrTextEmpty は text が空であることを表します。
	ErrTextEmpty = errors.New("text is empty")

	// ErrTextTooLong は text が MaxTextLength を超えたことを表します。
	ErrTextTooLong = errors.New("text exceeds max length")

	// ErrUpstreamTimeout は ElevenLabs へのリクエストがタイムアウトしたことを表します。
	ErrUpstreamTimeout = errors.New("elevenlabs request timeout")

	// ErrInvalidContentType は ElevenLabs から audio/mpeg 以外が返ったことを表します。
	ErrInvalidContentType = errors.New("elevenlabs returned non audio content type")
)
