// Package handler はHTTPハンドラーを提供します
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ghostrunner/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// sseKeepaliveInterval はSSEキープアライブコメントの送信間隔
// クライアントやプロキシのアイドルタイムアウトを防止するため15秒に設定
const sseKeepaliveInterval = 15 * time.Second

// setSSEHeaders はSSEレスポンスに必要なHTTPヘッダーを設定します
func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

// writeSSEEvents はeventChからイベントを読み取り、SSE形式でクライアントに送信します。
// Ginのc.Stream()の代わりにselectベースのループを使用し、
// コンテキストキャンセル（クライアント切断）を即座に検出します。
// また、30秒ごとにキープアライブコメントを送信して接続を維持します。
func writeSSEEvents(c *gin.Context, eventCh <-chan service.StreamEvent, handlerName string) {
	w := c.Writer
	flusher, ok := w.(interface{ Flush() })
	if !ok {
		log.Printf("[%s] ResponseWriter does not support Flush", handlerName)
		return
	}

	keepalive := time.NewTicker(sseKeepaliveInterval)
	defer keepalive.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Client disconnected (context canceled)", handlerName)
			return

		case event, ok := <-eventCh:
			if !ok {
				// チャネルが閉じられた（ストリーム正常完了）
				log.Printf("[%s] Event channel closed, stream completed", handlerName)
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("[%s] Marshal error: %v", handlerName, err)
				continue
			}

			log.Printf("[%s] SSE sending: type=%s, tool=%s", handlerName, event.Type, event.ToolName)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				log.Printf("[%s] SSE write error (client disconnected): %v", handlerName, err)
				return
			}
			flusher.Flush()

		case <-keepalive.C:
			// SSEキープアライブコメント（プロキシのタイムアウト防止）
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				log.Printf("[%s] Keepalive write error (client disconnected): %v", handlerName, err)
				return
			}
			flusher.Flush()
		}
	}
}
