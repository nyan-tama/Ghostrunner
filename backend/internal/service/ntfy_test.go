package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestNewNtfyService(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantNil  bool
	}{
		{
			name:     "NTFY_TOPIC未設定時はnilを返す",
			envValue: "",
			wantNil:  true,
		},
		{
			name:     "NTFY_TOPIC設定時は非nilを返す",
			envValue: "test-topic",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を設定し、テスト終了後に元に戻す
			original := os.Getenv("NTFY_TOPIC")
			os.Setenv("NTFY_TOPIC", tt.envValue)
			t.Cleanup(func() {
				os.Setenv("NTFY_TOPIC", original)
			})

			svc := NewNtfyService()

			if tt.wantNil && svc != nil {
				t.Errorf("NewNtfyService() = %v, want nil", svc)
			}
			if !tt.wantNil && svc == nil {
				t.Error("NewNtfyService() = nil, want non-nil")
			}
		})
	}
}

// capturedRequest はモックサーバーで受信したリクエストの情報を保持します
type capturedRequest struct {
	Method   string
	Title    string
	Priority string
	Tags     string
	Body     string
}

func TestNtfyService_Send(t *testing.T) {
	tests := []struct {
		name         string
		callMethod   string // "Notify" or "NotifyError"
		title        string
		message      string
		wantPriority string
		wantTags     string
	}{
		{
			name:         "Notify_リクエストヘッダーにdefault優先度とwhite_check_markタグが設定される",
			callMethod:   "Notify",
			title:        "Test Title",
			message:      "Test message body",
			wantPriority: "default",
			wantTags:     "white_check_mark",
		},
		{
			name:         "NotifyError_リクエストヘッダーにhigh優先度とxタグが設定される",
			callMethod:   "NotifyError",
			title:        "Error Title",
			message:      "Error occurred",
			wantPriority: "high",
			wantTags:     "x",
		},
		{
			name:         "Notify_メッセージ本文がPOSTボディに含まれる",
			callMethod:   "Notify",
			title:        "Body Check",
			message:      "This is the message content to verify",
			wantPriority: "default",
			wantTags:     "white_check_mark",
		},
		{
			name:         "NotifyError_メッセージ本文がPOSTボディに含まれる",
			callMethod:   "NotifyError",
			title:        "Error Body Check",
			message:      "Error details here",
			wantPriority: "high",
			wantTags:     "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var captured *capturedRequest

			// モックサーバーを作成してリクエストをキャプチャ
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read request body: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				mu.Lock()
				captured = &capturedRequest{
					Method:   r.Method,
					Title:    r.Header.Get("Title"),
					Priority: r.Header.Get("Priority"),
					Tags:     r.Header.Get("Tags"),
					Body:     string(body),
				}
				mu.Unlock()

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// モックサーバーのURLを使ってntfyServiceImplを直接作成
			svc := &ntfyServiceImpl{
				topicURL:   server.URL,
				httpClient: server.Client(),
			}

			// メソッド呼び出し（ゴルーチンで実行されるため待機が必要）
			switch tt.callMethod {
			case "Notify":
				svc.Notify(tt.title, tt.message)
			case "NotifyError":
				svc.NotifyError(tt.title, tt.message)
			}

			// ゴルーチンの完了を待つ（最大2秒）
			deadline := time.After(2 * time.Second)
			for {
				mu.Lock()
				done := captured != nil
				mu.Unlock()
				if done {
					break
				}
				select {
				case <-deadline:
					t.Fatal("timed out waiting for notification request")
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}

			mu.Lock()
			defer mu.Unlock()

			// HTTPメソッドの検証
			if captured.Method != http.MethodPost {
				t.Errorf("HTTP method: got %q, want %q", captured.Method, http.MethodPost)
			}

			// Titleヘッダーの検証
			if captured.Title != tt.title {
				t.Errorf("Title header: got %q, want %q", captured.Title, tt.title)
			}

			// Priorityヘッダーの検証
			if captured.Priority != tt.wantPriority {
				t.Errorf("Priority header: got %q, want %q", captured.Priority, tt.wantPriority)
			}

			// Tagsヘッダーの検証
			if captured.Tags != tt.wantTags {
				t.Errorf("Tags header: got %q, want %q", captured.Tags, tt.wantTags)
			}

			// メッセージ本文の検証
			if captured.Body != tt.message {
				t.Errorf("Body: got %q, want %q", captured.Body, tt.message)
			}
		})
	}
}
