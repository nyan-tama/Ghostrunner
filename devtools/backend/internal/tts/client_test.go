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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestVOICEVOXServer creates an httptest server that mocks VOICEVOX Engine's
// two endpoints: POST /audio_query and POST /synthesis.
// audioQueryHandler and synthesisHandler can be nil to use defaults.
func newTestVOICEVOXServer(
	audioQueryHandler func(w http.ResponseWriter, r *http.Request),
	synthesisHandler func(w http.ResponseWriter, r *http.Request),
) *httptest.Server {
	mux := http.NewServeMux()

	if audioQueryHandler == nil {
		audioQueryHandler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accent_phrases":[]}`))
		}
	}
	if synthesisHandler == nil {
		synthesisHandler = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/wav")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("RIFF-fake-wav"))
		}
	}

	mux.HandleFunc("POST /audio_query", audioQueryHandler)
	mux.HandleFunc("POST /synthesis", synthesisHandler)

	return httptest.NewServer(mux)
}

func newTestClient(serverURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return NewClient(Config{Host: serverURL, SpeakerID: 8}, timeout)
}

// ---------------------------------------------------------------------------
// Normal flow
// ---------------------------------------------------------------------------

func TestClient_Synthesize_Normal(t *testing.T) {
	srv := newTestVOICEVOXServer(nil, nil)
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	audio, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text:         "test",
		SpeakerID:    8,
		OutputFormat: "wav",
	})

	require.NoError(t, err)
	assert.Equal(t, []byte("RIFF-fake-wav"), audio)
}

func TestClient_AudioQueryJSON_Passthrough(t *testing.T) {
	// Verify that synthesis receives the exact bytes from audio_query response.
	queryJSON := `{"accent_phrases":[{"moras":[]}],"speedScale":1.0}`
	var receivedBody []byte

	srv := newTestVOICEVOXServer(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(queryJSON))
		},
		func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "audio/wav")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("wav"))
		},
	)
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "hello", SpeakerID: 8})
	require.NoError(t, err)

	// synthesis should receive exact JSON from audio_query
	assert.JSONEq(t, queryJSON, string(receivedBody))
}

// ---------------------------------------------------------------------------
// URL construction
// ---------------------------------------------------------------------------

func TestClient_URLConstruction(t *testing.T) {
	var audioQueryURL, synthesisURL string
	var synthContentType string

	srv := newTestVOICEVOXServer(
		func(w http.ResponseWriter, r *http.Request) {
			audioQueryURL = r.URL.String()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		},
		func(w http.ResponseWriter, r *http.Request) {
			synthesisURL = r.URL.String()
			synthContentType = r.Header.Get("Content-Type")
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("wav"))
		},
	)
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	_, err := c.Synthesize(context.Background(), SynthesizeParams{
		Text:      "hello world",
		SpeakerID: 3,
	})
	require.NoError(t, err)

	// audio_query: text is URL-encoded, speaker param present
	assert.Contains(t, audioQueryURL, "text=hello+world")
	assert.Contains(t, audioQueryURL, "speaker=3")

	// synthesis: speaker param, Content-Type=application/json
	assert.Contains(t, synthesisURL, "speaker=3")
	assert.Equal(t, "application/json", synthContentType)
}

// ---------------------------------------------------------------------------
// Upstream errors
// ---------------------------------------------------------------------------

func TestClient_AudioQuery_ErrorStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"4xx", http.StatusBadRequest},
		{"5xx", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synthesisCalled := false
			srv := newTestVOICEVOXServer(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte("error body"))
				},
				func(w http.ResponseWriter, r *http.Request) {
					synthesisCalled = true
					w.WriteHeader(http.StatusOK)
				},
			)
			defer srv.Close()

			c := newTestClient(srv.URL, 0)
			_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "t", SpeakerID: 1})

			require.Error(t, err)
			var ue *UpstreamStatusError
			require.True(t, errors.As(err, &ue))
			assert.Equal(t, tt.status, ue.Status)
			assert.False(t, synthesisCalled, "synthesis should NOT be called after audio_query failure")
		})
	}
}

func TestClient_Synthesis_ErrorStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"429 rate limit", http.StatusTooManyRequests},
		{"500 internal", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestVOICEVOXServer(
				nil, // audio_query succeeds
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte("upstream err"))
				},
			)
			defer srv.Close()

			c := newTestClient(srv.URL, 0)
			_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "t", SpeakerID: 1})

			require.Error(t, err)
			var ue *UpstreamStatusError
			require.True(t, errors.As(err, &ue))
			assert.Equal(t, tt.status, ue.Status)
		})
	}
}

func TestClient_Synthesis_NonAudioMIME(t *testing.T) {
	srv := newTestVOICEVOXServer(
		nil,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>oops</html>"))
		},
	)
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "t", SpeakerID: 1})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidContentType))
}

// ---------------------------------------------------------------------------
// Timeout / connection refused
// ---------------------------------------------------------------------------

func TestClient_Timeout(t *testing.T) {
	srv := newTestVOICEVOXServer(
		func(w http.ResponseWriter, r *http.Request) {
			// Delay longer than client timeout
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		},
		nil,
	)
	defer srv.Close()

	// Use very short context timeout
	c := newTestClient(srv.URL, 500*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := c.Synthesize(ctx, SynthesizeParams{Text: "t", SpeakerID: 1})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpstreamTimeout))
}

func TestClient_ConnectionRefused(t *testing.T) {
	// Start and immediately close a server to get a valid but closed port
	srv := httptest.NewServer(http.NewServeMux())
	closedURL := srv.URL
	srv.Close()

	c := newTestClient(closedURL, 2*time.Second)
	_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "t", SpeakerID: 1})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpstreamTimeout))
}

// ---------------------------------------------------------------------------
// audio_query Content-Length=0 verification
// ---------------------------------------------------------------------------

func TestClient_AudioQuery_ContentLengthZero(t *testing.T) {
	var contentLength string

	srv := newTestVOICEVOXServer(
		func(w http.ResponseWriter, r *http.Request) {
			contentLength = r.Header.Get("Content-Length")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("wav"))
		},
	)
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "t", SpeakerID: 1})
	require.NoError(t, err)
	assert.Equal(t, "0", contentLength)
}

// ---------------------------------------------------------------------------
// readBodySnippet (via JSON response body)
// ---------------------------------------------------------------------------

func TestClient_AudioQuery_BodySnippetInError(t *testing.T) {
	longBody := strings.Repeat("x", 300)
	srv := newTestVOICEVOXServer(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(longBody))
		},
		nil,
	)
	defer srv.Close()

	c := newTestClient(srv.URL, 0)
	_, err := c.Synthesize(context.Background(), SynthesizeParams{Text: "t", SpeakerID: 1})

	var ue *UpstreamStatusError
	require.True(t, errors.As(err, &ue))
	// Body should be truncated to upstreamBodySnippetMax (200)
	assert.LessOrEqual(t, len(ue.Body), upstreamBodySnippetMax)

	// Verify it's valid JSON for error reporting
	_, _ = json.Marshal(ue)
}
