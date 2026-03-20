// Package handler はHTTPハンドラーを提供します
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler はヘルスチェック関連のHTTPハンドラを提供します
type HealthHandler struct{}

// NewHealthHandler は新しいHealthHandlerを生成します
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Handle はサーバーのヘルスチェックを実行する。
//
// サーバーが正常に動作しているかを確認するためのシンプルなエンドポイント。
// 外部サービスへの依存はなく、サーバープロセスが起動していれば常に成功を返す。
//
// レスポンス:
//   - 200: 成功 {"status": "ok"}
func (h *HealthHandler) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
