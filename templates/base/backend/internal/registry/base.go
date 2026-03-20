package registry

import (
	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/handler"
)

func init() {
	OnRoute(func(api *gin.RouterGroup) {
		healthHandler := handler.NewHealthHandler()
		helloHandler := handler.NewHelloHandler()

		api.GET("/health", healthHandler.Handle)
		api.GET("/hello", helloHandler.Handle)
	})
}
