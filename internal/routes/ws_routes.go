package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/handler"
)

func RegisterWebSocketRoutes(rg *gin.RouterGroup, h handler.WebSocketHandler) {
	rg.GET("/tournaments/:id", h.Connect)
}
