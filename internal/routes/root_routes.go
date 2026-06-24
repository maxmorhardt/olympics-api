package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/handler"
)

func RegisterRootRoutes(rg *gin.RouterGroup, healthHandler handler.HealthHandler) {
	rg.GET("/health/live", healthHandler.Liveness)
	rg.GET("/health/ready", healthHandler.Readiness)
}
