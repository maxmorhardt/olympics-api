package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/olympics-api/internal/service"
)

type WebSocketHandler interface {
	Connect(c *gin.Context)
}

type websocketHandler struct {
	wsService service.WebSocketService
	upgrader  websocket.Upgrader
}

func NewWebSocketHandler(wsService service.WebSocketService, allowedOrigins []string) WebSocketHandler {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}

	return &websocketHandler{
		wsService: wsService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// allow non-browser clients (no Origin); enforce the allow-list otherwise
				origin := r.Header.Get("Origin")
				return origin == "" || originSet[origin]
			},
		},
	}
}

func (h *websocketHandler) Connect(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid tournament id"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.wsService.Register(id, conn)
}
