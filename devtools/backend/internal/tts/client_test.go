package tts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient は baseURL を httptest サーバーに差し替えた Client を返します。
func newTestClient(baseURL, apiKey string) *Client {
	c := NewClient(Config{APIKey: apiKey, BaseURL: baseURL})
	return c
}

// --- 正常系 + リクエスト形式の検証 ---

func TestClient_Synthesize_Success(t *testing.T) {
	var (
		capturedMethod      string
		capturedPath        string
		capturedQuery       string
		capturedAPIKey      string
		capturedContentType string
		capturedAccept      string
		capturedBody        elevenLabsRequestBody
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		capturedAPIKey = r.Header.Get("xi-api-key")
		capturedContentType = r.Header.Get("Content-Type")
		capturedAccept = r.Header.Get("Accept")

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3-binary-content"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "test-key")
	audio, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text:         "hello",
		VoiceID:      "voice-xyz",
		ModelID:      "model-abc",
		OutputFormat: "mp3_44100_128",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(audio) != "mp3-binary-content" {
		t.Errorf("unexpected audio: %q", string(audio))
	}

	// リクエスト形式の検証
	if capturedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if !strings.HasSuffix(capturedPath, "/v1/text-to-speech/voice-xyz") {
		t.Errorf("unexpected path: %s", capturedPath)
	}
	if !strings.Contains(capturedQuery, "output_format=mp3_44100_128") {
		t.Errorf("expected output_format query param, got %s", capturedQuery)
	}
	if capturedAPIKey != "test-key" {
		t.Errorf("expected xi-api-key=test-key, got %q", capturedAPIKey)
	}
	if capturedContentType != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", capturedContentType)
	}
	if capturedAccept != "audio/mpeg" {
		t.Errorf("expected Accept=audio/mpeg, got %q", capturedAccept)
	}

	// リクエストボディの検証
	if capturedBody.Text != "hello" {
		t.Errorf("expected body.text=hello, got %q", capturedBody.Text)
	}
	if capturedBody.ModelID != "model-abc" {
		t.Errorf("expected body.model_id=model-abc, got %q", capturedBody.ModelID)
	}
	if capturedBody.VoiceSettings.Stability != 0.5 {
		t.Errorf("expected stability=0.5, got %v", capturedBody.VoiceSettings.Stability)
	}
	if capturedBody.VoiceSettings.SimilarityBoost != 0.75 {
		t.Errorf("expected similarity_boost=0.75, got %v", capturedBody.VoiceSettings.SimilarityBoost)
	}
}

// --- 非 200 系: 401/429/500 ---

func TestClient_Synthesize_UpstreamStatusError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus int
	}{
		{"401 unauthorized", http.StatusUnauthorized, "invalid key", 401},
		{"429 rate limit", http.StatusTooManyRequests, `{"detail":"rate limit"}`, 429},
		{"500 server error", http.StatusInternalServerError, "server error", 500},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c := newTestClient(srv.URL, "k")
			_, err := c.Synthesize(context.Background(), SynthesizeParams{
				Text: "x", VoiceID: "v", ModelID: "m", OutputFormat: "mp3_44100_128",
			})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			var ue *UpstreamStatusError
			if !errors.As(err, &ue) {
				t.Fatalf("expected *UpstreamStatusError, got %v", err)
			}
			if ue.Status != tc.wantStatus {
				t.Errorf("expected Status=%d, got %d", tc.wantStatus, ue.Status)
			}
			// Body は 200 文字以内に切り詰められている
			if len(ue.Body) > upstreamBodySnippetMax {
				t.Errorf("expected body <= %d chars, got %d", upstreamBodySnippetMax, len(ue.Body))
			}
		})
	}
}

// --- 非 audio Content-Type → ErrInvalidContentType ---

func TestClient_Synthesize_NonAudioContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"unexpected":"json"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "k")
	_, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text: "x", VoiceID: "v", ModelID: "m", OutputFormat: "mp3_44100_128",
	})
	if !errors.Is(err, ErrInvalidContentType) {
		t.Errorf("expected ErrInvalidContentType, got %v", err)
	}
}

// --- Content-Type 大文字混在 → 正常パース ---

func TestClient_Synthesize_CaseInsensitiveContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "Audio/MPEG")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "k")
	audio, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text: "x", VoiceID: "v", ModelID: "m", OutputFormat: "mp3_44100_128",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(audio) != "ok" {
		t.Errorf("unexpected body: %q", string(audio))
	}
}

// --- 大きなエラーボディの切り詰め(API キー文字列が含まれていてもログ汚染しない) ---

func TestClient_Synthesize_BodySnippetTruncation(t *testing.T) {
	hugeBody := strings.Repeat("x", 1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(hugeBody))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "secret-api-key")
	_, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text: "x", VoiceID: "v", ModelID: "m", OutputFormat: "mp3_44100_128",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var ue *UpstreamStatusError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *UpstreamStatusError, got %v", err)
	}
	if len(ue.Body) > upstreamBodySnippetMax {
		t.Errorf("body should be truncated to %d chars, got %d", upstreamBodySnippetMax, len(ue.Body))
	}
	if strings.Contains(ue.Body, "secret-api-key") {
		t.Errorf("api key must not leak into error body")
	}
}

// --- ネットワークエラー → ErrUpstreamTimeout 系 ---

func TestClient_Synthesize_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// サーバー側で接続を hijack して即切断
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("hijacker not supported")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack failed: %v", err)
		}
		_ = conn.Close()
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "k")
	_, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text: "x", VoiceID: "v", ModelID: "m", OutputFormat: "mp3_44100_128",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUpstreamTimeout) {
		t.Errorf("expected errors.Is(ErrUpstreamTimeout), got %v", err)
	}
}

// --- context cancel → ErrUpstreamTimeout(リーク無し) ---

func TestClient_Synthesize_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 応答を遅延させる
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "k")
	ctx, cancel := context.WithCancel(context.Background())
	// すぐ cancel
	cancel()

	_, err := c.Synthesize(ctx, SynthesizeParams{
		Text: "x", VoiceID: "v", ModelID: "m", OutputFormat: "mp3_44100_128",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUpstreamTimeout) {
		t.Errorf("expected errors.Is(ErrUpstreamTimeout) (wrapped context.Canceled), got %v", err)
	}
}
