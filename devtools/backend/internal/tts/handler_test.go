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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockService implements Service for handler tests.
type mockService struct {
	result *SynthesizeResult
	err    error
}

func (m *mockService) Synthesize(_ context.Context, _ SynthesizeParams) (*SynthesizeResult, error) {
	return m.result, m.err
}

func setupRouter(svc Service) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/api/tts", h.HandleSynthesize)
	return r
}

func doTTSRequest(t *testing.T, router *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/tts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// mapErrorToStatus
// ---------------------------------------------------------------------------

func TestMapErrorToStatus(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "ErrTextEmpty",
			err:            ErrTextEmpty,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "テキストが不正です",
		},
		{
			name:           "ErrTextTooLong",
			err:            ErrTextTooLong,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "テキストが不正です",
		},
		{
			name:           "ErrUpstreamTimeout",
			err:            ErrUpstreamTimeout,
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "VOICEVOX への接続に失敗しました",
		},
		{
			name:           "UpstreamStatusError 429",
			err:            &UpstreamStatusError{Status: 429, Body: "rate limited"},
			expectedStatus: http.StatusTooManyRequests,
			expectedMsg:    "VOICEVOX レート制限",
		},
		{
			name:           "UpstreamStatusError 500",
			err:            &UpstreamStatusError{Status: 500, Body: "internal"},
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "VOICEVOX から音声を取得できませんでした",
		},
		{
			name:           "UpstreamStatusError 400 not transparent",
			err:            &UpstreamStatusError{Status: 400, Body: "bad request"},
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "VOICEVOX から音声を取得できませんでした",
		},
		{
			name:           "ErrInvalidContentType",
			err:            ErrInvalidContentType,
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "VOICEVOX から音声を取得できませんでした",
		},
		{
			name:           "unknown error",
			err:            errors.New("something unexpected"),
			expectedStatus: http.StatusBadGateway,
			expectedMsg:    "VOICEVOX から音声を取得できませんでした",
		},
		{
			name:           "wrapped UpstreamStatusError via fmt.Errorf",
			err:            fmt.Errorf("wrapped: %w", &UpstreamStatusError{Status: 429, Body: "wrapped"}),
			expectedStatus: http.StatusTooManyRequests,
			expectedMsg:    "VOICEVOX レート制限",
		},
		{
			name:           "wrapped ErrUpstreamTimeout",
			err:            fmt.Errorf("timeout: %w", ErrUpstreamTimeout),
			expectedStatus: http.StatusGatewayTimeout,
			expectedMsg:    "VOICEVOX への接続に失敗しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := mapErrorToStatus(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedMsg, msg)
		})
	}
}

func TestMapErrorToStatus_No503(t *testing.T) {
	// Verify that no error path ever returns 503
	errs := []error{
		ErrTextEmpty,
		ErrTextTooLong,
		ErrUpstreamTimeout,
		ErrInvalidContentType,
		&UpstreamStatusError{Status: 500},
		&UpstreamStatusError{Status: 429},
		&UpstreamStatusError{Status: 400},
		&UpstreamStatusError{Status: 503},
		errors.New("unknown"),
	}
	for _, err := range errs {
		status, _ := mapErrorToStatus(err)
		assert.NotEqual(t, 503, status, "503 must never be returned, got it for error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Text validation in handler
// ---------------------------------------------------------------------------

func TestHandler_TextValidation(t *testing.T) {
	successSvc := &mockService{
		result: &SynthesizeResult{
			Audio:       []byte("wav"),
			ContentType: "audio/wav",
			FromCache:   false,
		},
	}

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "empty text",
			body:           `{"text":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid text",
			body:           `{"text":"hello"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing text field",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "text exceeds MaxTextLength (2001 chars)",
			body:           `{"text":"` + strings.Repeat("a", 2001) + `"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "text at boundary (2000 chars)",
			body:           `{"text":"` + strings.Repeat("a", 2000) + `"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupRouter(successSvc)
			w := doTTSRequest(t, router, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_TextValidation_MultibyteBoundary(t *testing.T) {
	successSvc := &mockService{
		result: &SynthesizeResult{
			Audio:       []byte("wav"),
			ContentType: "audio/wav",
			FromCache:   false,
		},
	}

	tests := []struct {
		name           string
		runeCount      int
		expectedStatus int
	}{
		{
			name:           "2000 multibyte runes OK",
			runeCount:      2000,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "2001 multibyte runes rejected",
			runeCount:      2001,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := strings.Repeat("あ", tt.runeCount)
			body, _ := json.Marshal(TTSRequest{Text: text})
			router := setupRouter(successSvc)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/tts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Normal response headers and body
// ---------------------------------------------------------------------------

func TestHandler_Response_Success(t *testing.T) {
	wavData := []byte("RIFF-fake-wav-content")
	svc := &mockService{
		result: &SynthesizeResult{
			Audio:       wavData,
			ContentType: "audio/wav",
			FromCache:   false,
		},
	}
	router := setupRouter(svc)
	w := doTTSRequest(t, router, `{"text":"hello"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "audio/wav")
	assert.Equal(t, "miss", w.Header().Get("X-TTS-Cache"))
	assert.Equal(t, wavData, w.Body.Bytes())
}

func TestHandler_Response_CacheHit(t *testing.T) {
	svc := &mockService{
		result: &SynthesizeResult{
			Audio:       []byte("cached"),
			ContentType: "audio/wav",
			FromCache:   true,
		},
	}
	router := setupRouter(svc)
	w := doTTSRequest(t, router, `{"text":"hello"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hit", w.Header().Get("X-TTS-Cache"))
}

func TestHandler_Response_CacheMiss(t *testing.T) {
	svc := &mockService{
		result: &SynthesizeResult{
			Audio:       []byte("fresh"),
			ContentType: "audio/wav",
			FromCache:   false,
		},
	}
	router := setupRouter(svc)
	w := doTTSRequest(t, router, `{"text":"hello"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "miss", w.Header().Get("X-TTS-Cache"))
}

func TestHandler_Response_BodyBytesMatch(t *testing.T) {
	expected := []byte{0x52, 0x49, 0x46, 0x46, 0x01, 0x02, 0x03}
	svc := &mockService{
		result: &SynthesizeResult{
			Audio:       expected,
			ContentType: "audio/wav",
			FromCache:   false,
		},
	}
	router := setupRouter(svc)
	w := doTTSRequest(t, router, `{"text":"hello"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expected, w.Body.Bytes())
}

func TestHandler_InvalidJSON(t *testing.T) {
	svc := &mockService{}
	router := setupRouter(svc)
	w := doTTSRequest(t, router, `{invalid json`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp TTSErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

// ---------------------------------------------------------------------------
// Error mapping integration (through handler)
// ---------------------------------------------------------------------------

func TestHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		svcErr         error
		expectedStatus int
		expectedCT     string
	}{
		{
			name:           "ErrUpstreamTimeout -> 504",
			svcErr:         ErrUpstreamTimeout,
			expectedStatus: http.StatusGatewayTimeout,
			expectedCT:     "application/json",
		},
		{
			name:           "UpstreamStatusError 429 -> 429",
			svcErr:         &UpstreamStatusError{Status: 429, Body: "limit"},
			expectedStatus: http.StatusTooManyRequests,
			expectedCT:     "application/json",
		},
		{
			name:           "UpstreamStatusError 500 -> 502",
			svcErr:         &UpstreamStatusError{Status: 500, Body: "err"},
			expectedStatus: http.StatusBadGateway,
			expectedCT:     "application/json",
		},
		{
			name:           "ErrInvalidContentType -> 502",
			svcErr:         ErrInvalidContentType,
			expectedStatus: http.StatusBadGateway,
			expectedCT:     "application/json",
		},
		{
			name:           "success -> 200 audio/wav",
			svcErr:         nil,
			expectedStatus: http.StatusOK,
			expectedCT:     "audio/wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{err: tt.svcErr}
			if tt.svcErr == nil {
				svc.result = &SynthesizeResult{
					Audio:       []byte("wav"),
					ContentType: "audio/wav",
					FromCache:   false,
				}
			}
			router := setupRouter(svc)
			w := doTTSRequest(t, router, `{"text":"hello"}`)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), tt.expectedCT)
		})
	}
}
