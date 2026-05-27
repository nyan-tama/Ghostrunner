package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// upstreamBodySnippetMax は UpstreamStatusError.Body に保持する最大文字数
const upstreamBodySnippetMax = 200

// Client は ElevenLabs API への HTTP クライアントです。
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient は新しい Client を生成します。
func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: HTTPTimeout},
	}
}

// elevenLabsRequestBody は ElevenLabs API へ送る JSON ボディです。
type elevenLabsRequestBody struct {
	Text          string             `json:"text"`
	ModelID       string             `json:"model_id"`
	VoiceSettings elevenVoiceSetting `json:"voice_settings"`
}

// elevenVoiceSetting は voice_settings サブオブジェクトです。
// MVP+ では backend 固定値(stability=0.5, similarity_boost=0.75)を使います。
type elevenVoiceSetting struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

// Synthesize は ElevenLabs API を呼び出して MP3 バイナリを取得します。
//
// 戻り値:
//   - 成功: ([]byte, nil)
//   - 非 200 応答: *UpstreamStatusError
//   - 非 audio/mpeg Content-Type: ErrInvalidContentType
//   - タイムアウト/ネットワークエラー: ErrUpstreamTimeout ラップ
func (c *Client) Synthesize(ctx context.Context, params SynthesizeParams) ([]byte, error) {
	start := time.Now()

	endpoint := fmt.Sprintf("%s/v1/text-to-speech/%s", c.baseURL, url.PathEscape(params.VoiceID))
	query := url.Values{}
	query.Set("output_format", params.OutputFormat)
	endpoint = endpoint + "?" + query.Encode()

	reqBody := elevenLabsRequestBody{
		Text:    params.Text,
		ModelID: params.ModelID,
		VoiceSettings: elevenVoiceSetting{
			Stability:       0.5,
			SimilarityBoost: 0.75,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// context cancel はそのまま伝播(リソースリーク防止のため上層で扱う)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[TTSService] ElevenLabs request canceled/deadline: error=%v", err)
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		// net/http の Timeout 由来も ErrUpstreamTimeout に丸める
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			log.Printf("[TTSService] ElevenLabs request timeout: error=%v", err)
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		log.Printf("[TTSService] ElevenLabs request failed: error=%v", err)
		return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
	}
	defer resp.Body.Close()

	durationMs := time.Since(start).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		// エラーボディを先頭 200 文字のみ読む(API キー漏洩防止 + ログ過大化防止)
		bodySnippet := readBodySnippet(resp.Body, upstreamBodySnippetMax)
		log.Printf("[TTSService] ElevenLabs response: status=%d, durationMs=%d, bodySnippet=%q",
			resp.StatusCode, durationMs, bodySnippet)
		return nil, &UpstreamStatusError{Status: resp.StatusCode, Body: bodySnippet}
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "audio/") {
		log.Printf("[TTSService] ElevenLabs returned non-audio content type: contentType=%s, durationMs=%d",
			contentType, durationMs)
		return nil, ErrInvalidContentType
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TTSService] failed to read elevenlabs response body: error=%v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("[TTSService] ElevenLabs response: status=%d, bytes=%d, durationMs=%d",
		resp.StatusCode, len(audio), durationMs)

	return audio, nil
}

// readBodySnippet はレスポンスボディの先頭 max 文字を読み取って文字列で返します。
// 残りは捨てます(リーク防止のため defer Close と併用)。
func readBodySnippet(r io.Reader, max int) string {
	buf := make([]byte, max)
	n, _ := io.ReadFull(r, buf)
	// 余剰は破棄して接続再利用を妨げない
	_, _ = io.Copy(io.Discard, r)
	if n <= 0 {
		return ""
	}
	return string(buf[:n])
}
