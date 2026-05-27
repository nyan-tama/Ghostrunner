package tts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// upstreamBodySnippetMax は UpstreamStatusError.Body に保持する最大文字数
const upstreamBodySnippetMax = 200

// Client は VOICEVOX Engine への HTTP クライアントです。
// 2-stage API (audio_query + synthesis) を呼び出します。
type Client struct {
	host       string
	speakerID  int
	httpClient *http.Client
}

// NewClient は新しい Client を生成します。
// timeout が 0 の場合は HTTPTimeout を使用します(テスト時に短縮可能)。
func NewClient(cfg Config, timeout time.Duration) *Client {
	host := cfg.Host
	if host == "" {
		host = DefaultBaseURL
	}
	if timeout == 0 {
		timeout = HTTPTimeout
	}
	speakerID := cfg.SpeakerID
	if speakerID == 0 {
		speakerID = DefaultSpeakerID
	}
	return &Client{
		host:      host,
		speakerID: speakerID,
		httpClient: &http.Client{
			// 個別リクエストのタイムアウトは context.WithTimeout で制御する。
			// http.Client.Timeout はフォールバックとして設定する。
			Timeout: timeout + 5*time.Second,
		},
	}
}

// Synthesize は VOICEVOX の 2-stage API を呼び出して WAV バイナリを取得します。
//
// 戻り値:
//   - 成功: ([]byte, nil)
//   - 非 200 応答: *UpstreamStatusError
//   - 非 audio/wav Content-Type: ErrInvalidContentType
//   - タイムアウト/接続拒否: ErrUpstreamTimeout
func (c *Client) Synthesize(ctx context.Context, params SynthesizeParams) ([]byte, error) {
	start := time.Now()

	speakerID := params.SpeakerID
	if speakerID == 0 {
		speakerID = c.speakerID
	}

	// 2-stage 合計で HTTPTimeout に収まるよう context にタイムアウトを設定
	ctx, cancel := context.WithTimeout(ctx, HTTPTimeout)
	defer cancel()

	// Stage 1: audio_query
	query, err := c.audioQuery(ctx, params.Text, speakerID)
	if err != nil {
		return nil, err
	}

	// Stage 2: synthesis
	audio, err := c.synthesis(ctx, speakerID, query)
	if err != nil {
		return nil, err
	}

	durationMs := time.Since(start).Milliseconds()
	log.Printf("[TTSService] VOICEVOX synthesis completed: speakerID=%d, bytes=%d, durationMs=%d",
		speakerID, len(audio), durationMs)

	return audio, nil
}

// audioQuery は VOICEVOX /audio_query エンドポイントを呼び出します。
// テキストからアクセント・イントネーション等の中間表現 (AudioQuery JSON) を取得します。
func (c *Client) audioQuery(ctx context.Context, text string, speakerID int) (AudioQuery, error) {
	endpoint := fmt.Sprintf("%s/audio_query?text=%s&speaker=%d",
		c.host, url.QueryEscape(text), speakerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio_query request: %w", err)
	}
	req.Header.Set("Content-Length", "0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapClientError(err, "audio_query")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodySnippet := readBodySnippet(resp.Body, upstreamBodySnippetMax)
		log.Printf("[TTSService] VOICEVOX audio_query failed: status=%d, bodySnippet=%q",
			resp.StatusCode, bodySnippet)
		return nil, &UpstreamStatusError{Status: resp.StatusCode, Body: bodySnippet}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio_query response: %w", err)
	}

	return AudioQuery(body), nil
}

// synthesis は VOICEVOX /synthesis エンドポイントを呼び出します。
// audio_query の結果 JSON をボディに送り、WAV バイナリを取得します。
func (c *Client) synthesis(ctx context.Context, speakerID int, query AudioQuery) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/synthesis?speaker=%d", c.host, speakerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesis request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/wav")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapClientError(err, "synthesis")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodySnippet := readBodySnippet(resp.Body, upstreamBodySnippetMax)
		log.Printf("[TTSService] VOICEVOX synthesis failed: status=%d, bodySnippet=%q",
			resp.StatusCode, bodySnippet)
		return nil, &UpstreamStatusError{Status: resp.StatusCode, Body: bodySnippet}
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "audio/wav") {
		log.Printf("[TTSService] VOICEVOX synthesis returned non-wav content type: contentType=%s",
			contentType)
		return nil, ErrInvalidContentType
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read synthesis response body: %w", err)
	}

	return audio, nil
}

// mapClientError は HTTP クライアントのエラーを適切なセンチネルエラーにマッピングします。
func mapClientError(err error, stage string) error {
	// context.DeadlineExceeded / context.Canceled
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		log.Printf("[TTSService] VOICEVOX %s canceled/deadline: error=%v", stage, err)
		return fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
	}

	// syscall.ECONNREFUSED (VOICEVOX Engine 未起動)
	if errors.Is(err, syscall.ECONNREFUSED) {
		log.Printf("[TTSService] VOICEVOX %s connection refused: error=%v", stage, err)
		return fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
	}

	// net.Error の Timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		log.Printf("[TTSService] VOICEVOX %s timeout: error=%v", stage, err)
		return fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
	}

	// その他のネットワークエラーも ErrUpstreamTimeout に丸める
	log.Printf("[TTSService] VOICEVOX %s request failed: error=%v", stage, err)
	return fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
}

// readBodySnippet はレスポンスボディの先頭 max バイトを読み取って文字列で返します。
// 残りは破棄します(リーク防止のため defer Close と併用)。
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

// speakerIDStr は int を文字列に変換するヘルパーです。
// strconv.Itoa のエイリアスとして cache.go から利用します。
var speakerIDStr = strconv.Itoa
