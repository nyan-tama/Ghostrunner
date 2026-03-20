package registry

import (
	"github.com/gin-gonic/gin"

	"{{PROJECT_NAME}}/backend/internal/handler"
)

func init() {
	OnRoute(func(api *gin.RouterGroup) {
		healthHandler := handler.NewHealthHandler()

		api.GET("/health", healthHandler.Handle)
	})
}
