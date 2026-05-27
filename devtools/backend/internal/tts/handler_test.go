package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// --- モック Service ---

type mockService struct {
	synthesizeFunc func(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error)
	calls          int
	lastParams     SynthesizeParams
}

func (m *mockService) Synthesize(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error) {
	m.calls++
	m.lastParams = params
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, params)
	}
	return &SynthesizeResult{
		Audio:       []byte("mock-audio"),
		ContentType: "audio/mpeg",
		FromCache:   false,
	}, nil
}

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/api/tts", h.HandleSynthesize)
	return r
}

func postJSON(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/api/tts", strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeErrorResponse(t *testing.T, w *httptest.ResponseRecorder) TTSErrorResponse {
	t.Helper()
	var resp TTSErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal failed: %v (body=%s)", err, w.Body.String())
	}
	return resp
}

// --- svc == nil で 503 ---

func TestHandler_NilService_Returns503(t *testing.T) {
	r := setupRouter(nil)
	w := postJSON(t, r, `{"text":"hello"}`)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	resp := decodeErrorResponse(t, w)
	if resp.Success {
		t.Errorf("expected success=false")
	}
	if resp.Error != "ElevenLabs サービスが利用できません" {
		t.Errorf("unexpected error message: %q", resp.Error)
	}
}

// --- 不正 JSON で 400 ---

func TestHandler_InvalidJSON_Returns400(t *testing.T) {
	r := setupRouter(&mockService{})
	w := postJSON(t, r, `{`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// --- text 空で 400 ---

func TestHandler_EmptyText_Returns400(t *testing.T) {
	r := setupRouter(&mockService{})
	w := postJSON(t, r, `{"text":""}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	resp := decodeErrorResponse(t, w)
	if !strings.Contains(resp.Error, "text") {
		t.Errorf("expected error message to contain 'text', got %q", resp.Error)
	}
}

// --- text 上限超で 400 ---

func TestHandler_TextTooLong_Returns400(t *testing.T) {
	longText := strings.Repeat("a", MaxTextLength+1)
	r := setupRouter(&mockService{})

	body, err := json.Marshal(map[string]string{"text": longText})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/tts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for textLen=%d, got %d", MaxTextLength+1, w.Code)
	}
}

// --- text 上限ちょうど → 200 ---

func TestHandler_TextAtLimit_Returns200(t *testing.T) {
	boundaryText := strings.Repeat("a", MaxTextLength)
	r := setupRouter(&mockService{})

	body, err := json.Marshal(map[string]string{"text": boundaryText})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/tts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for boundary text, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// --- 正常系: audio/mpeg + X-TTS-Cache: miss ---

func TestHandler_Success_CacheMiss(t *testing.T) {
	svc := &mockService{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error) {
			return &SynthesizeResult{
				Audio:       []byte("mp3-bytes"),
				ContentType: "audio/mpeg",
				FromCache:   false,
			}, nil
		},
	}
	r := setupRouter(svc)
	w := postJSON(t, r, `{"text":"hello"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "audio/mpeg") {
		t.Errorf("expected Content-Type audio/mpeg, got %q", ct)
	}
	if cache := w.Header().Get("X-TTS-Cache"); cache != "miss" {
		t.Errorf("expected X-TTS-Cache=miss, got %q", cache)
	}
	if got := w.Body.String(); got != "mp3-bytes" {
		t.Errorf("unexpected body: %q", got)
	}
}

// --- cache hit ヘッダ ---

func TestHandler_Success_CacheHit(t *testing.T) {
	svc := &mockService{
		synthesizeFunc: func(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error) {
			return &SynthesizeResult{
				Audio:       []byte("cached"),
				ContentType: "audio/mpeg",
				FromCache:   true,
			}, nil
		},
	}
	r := setupRouter(svc)
	w := postJSON(t, r, `{"text":"hello"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if cache := w.Header().Get("X-TTS-Cache"); cache != "hit" {
		t.Errorf("expected X-TTS-Cache=hit, got %q", cache)
	}
}

// --- mapErrorToStatus の各分岐 ---

func TestMapErrorToStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsgSub string
	}{
		{
			name:       "ErrAPIKeyMissing → 503",
			err:        ErrAPIKeyMissing,
			wantStatus: http.StatusServiceUnavailable,
			wantMsgSub: "利用できません",
		},
		{
			name:       "ErrTextEmpty → 400",
			err:        ErrTextEmpty,
			wantStatus: http.StatusBadRequest,
			wantMsgSub: "text",
		},
		{
			name:       "ErrTextTooLong → 400",
			err:        ErrTextTooLong,
			wantStatus: http.StatusBadRequest,
			wantMsgSub: "text",
		},
		{
			name:       "ErrUpstreamTimeout → 504",
			err:        ErrUpstreamTimeout,
			wantStatus: http.StatusGatewayTimeout,
			wantMsgSub: "タイムアウト",
		},
		{
			name:       "UpstreamStatusError{429} → 429",
			err:        &UpstreamStatusError{Status: 429, Body: "rate limit"},
			wantStatus: http.StatusTooManyRequests,
			wantMsgSub: "レート",
		},
		{
			name:       "UpstreamStatusError{401} → 502 (clamped)",
			err:        &UpstreamStatusError{Status: 401, Body: "unauthorized"},
			wantStatus: http.StatusBadGateway,
			wantMsgSub: "音声を取得できませんでした",
		},
		{
			name:       "UpstreamStatusError{500} → 502",
			err:        &UpstreamStatusError{Status: 500, Body: "server error"},
			wantStatus: http.StatusBadGateway,
			wantMsgSub: "音声を取得できませんでした",
		},
		{
			name:       "ErrInvalidContentType → 502",
			err:        ErrInvalidContentType,
			wantStatus: http.StatusBadGateway,
			wantMsgSub: "音声を取得できませんでした",
		},
		{
			name:       "unknown error → 502 (fallback)",
			err:        errors.New("unknown"),
			wantStatus: http.StatusBadGateway,
			wantMsgSub: "音声を取得できませんでした",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, msg := mapErrorToStatus(tc.err)
			if status != tc.wantStatus {
				t.Errorf("status: want=%d got=%d", tc.wantStatus, status)
			}
			if !strings.Contains(msg, tc.wantMsgSub) {
				t.Errorf("message: want to contain %q, got %q", tc.wantMsgSub, msg)
			}
		})
	}
}

// --- mapErrorToStatus が errors.As でラップを貫通 ---

func TestMapErrorToStatus_WrappedUpstreamError(t *testing.T) {
	wrapped := fmt.Errorf("service: %w", &UpstreamStatusError{Status: 429, Body: "rate limit"})
	status, msg := mapErrorToStatus(wrapped)
	if status != http.StatusTooManyRequests {
		t.Errorf("expected 429 through wrap, got %d", status)
	}
	if !strings.Contains(msg, "レート") {
		t.Errorf("unexpected message: %q", msg)
	}
}

// --- ハンドラ経由でのエラーステータスマッピング ---

func TestHandler_ErrorMappingViaHTTP(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"ErrAPIKeyMissing → 503", ErrAPIKeyMissing, http.StatusServiceUnavailable},
		{"ErrTextEmpty → 400 (via service err)", ErrTextEmpty, http.StatusBadRequest},
		{"ErrUpstreamTimeout → 504", fmt.Errorf("wrap: %w", ErrUpstreamTimeout), http.StatusGatewayTimeout},
		{"UpstreamStatusError{429} → 429", &UpstreamStatusError{Status: 429}, http.StatusTooManyRequests},
		{"UpstreamStatusError{401} → 502", &UpstreamStatusError{Status: 401}, http.StatusBadGateway},
		{"UpstreamStatusError{500} → 502", &UpstreamStatusError{Status: 500}, http.StatusBadGateway},
		{"ErrInvalidContentType → 502", ErrInvalidContentType, http.StatusBadGateway},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.err
			svc := &mockService{
				synthesizeFunc: func(ctx context.Context, params SynthesizeParams) (*SynthesizeResult, error) {
					return nil, err
				},
			}
			r := setupRouter(svc)
			w := postJSON(t, r, `{"text":"hello"}`)
			if w.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d (body=%s)", tc.wantStatus, w.Code, w.Body.String())
			}

			// JSON 形式の検証 + API キー文字列を含まない
			resp := decodeErrorResponse(t, w)
			if resp.Success {
				t.Errorf("expected success=false")
			}
			if resp.Error == "" {
				t.Errorf("expected non-empty error message")
			}
			if strings.Contains(resp.Error, "api-key") || strings.Contains(resp.Error, "xi-api-key") {
				t.Errorf("error message must not contain api key strings, got %q", resp.Error)
			}
		})
	}
}
