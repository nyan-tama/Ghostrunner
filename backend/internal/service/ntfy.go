// Package service はビジネスロジックを提供します
package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// NtfyService はntfy.sh通知操作のインターフェースを定義します
type NtfyService interface {
	// Notify は通常の通知を送信します（完了通知など）
	Notify(title, message string)
	// NotifyError はエラー通知を送信します
	NotifyError(title, message string)
}

// ntfyServiceImpl はNtfyServiceの実装です
type ntfyServiceImpl struct {
	topicURL            string
	httpClient          *http.Client
	terminalNotifierPath string
}

// NewNtfyService は新しいNtfyServiceを生成します
// 環境変数 NTFY_TOPIC が未設定の場合は nil を返します（オプショナル機能）
func NewNtfyService() NtfyService {
	topic := os.Getenv("NTFY_TOPIC")
	if topic == "" {
		log.Printf("[NtfyService] NTFY_TOPIC is not set, ntfy notification will not be available")
		return nil
	}

	topicURL := fmt.Sprintf("https://ntfy.sh/%s", topic)

	// ローカル実行時は terminal-notifier でMac通知も出す
	tnPath, _ := exec.LookPath("terminal-notifier")
	if tnPath != "" {
		log.Printf("[NtfyService] terminal-notifier found: %s (Mac desktop notifications enabled)", tnPath)
	}

	log.Printf("[NtfyService] Initialized with topic: %s", topicURL)
	return &ntfyServiceImpl{
		topicURL: topicURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		terminalNotifierPath: tnPath,
	}
}

// Notify は通常の通知を送信します（fire-and-forget）
func (s *ntfyServiceImpl) Notify(title, message string) {
	go s.send(title, message, "default", "white_check_mark")
}

// NotifyError はエラー通知を送信します（fire-and-forget）
func (s *ntfyServiceImpl) NotifyError(title, message string) {
	go s.send(title, message, "high", "x")
}

// send はntfy.shへHTTP POSTで通知を送信し、ローカル実行時はMac通知も表示します
func (s *ntfyServiceImpl) send(title, message, priority, tags string) {
	log.Printf("[NtfyService] Sending notification: title=%s, priority=%s", title, priority)

	// ntfy.sh へ送信（iPhone通知）
	s.sendNtfy(title, message, priority, tags)

	// ローカル実行時はMac通知も表示
	if s.terminalNotifierPath != "" {
		s.sendDesktop(title, message, priority)
	}
}

// sendNtfy はntfy.shへHTTP POSTで通知を送信します
func (s *ntfyServiceImpl) sendNtfy(title, message, priority, tags string) {
	req, err := http.NewRequest(http.MethodPost, s.topicURL, strings.NewReader(message))
	if err != nil {
		log.Printf("[NtfyService] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", tags)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[NtfyService] Failed to send ntfy notification: %v", err)
		return
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[NtfyService] Unexpected response status: %d", resp.StatusCode)
		return
	}

	log.Printf("[NtfyService] ntfy notification sent: title=%s", title)
}

// sendDesktop はterminal-notifierでMacデスクトップ通知を表示します
func (s *ntfyServiceImpl) sendDesktop(title, message, priority string) {
	sound := "default"
	if priority == "high" {
		sound = "Basso"
	}

	cmd := exec.Command(s.terminalNotifierPath,
		"-title", title,
		"-message", message,
		"-sound", sound,
	)
	if err := cmd.Run(); err != nil {
		log.Printf("[NtfyService] Failed to send desktop notification: %v", err)
	}
}
