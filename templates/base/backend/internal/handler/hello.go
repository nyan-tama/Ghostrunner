package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HelloHandler は Hello World 用のハンドラー。
type HelloHandler struct{}

// NewHelloHandler は HelloHandler を生成する。
func NewHelloHandler() *HelloHandler {
	return &HelloHandler{}
}

// Handle は Hello World メッセージを返す。
// GET /api/hello -> {"message": "Hello, World!"}
func (h *HelloHandler) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
}
