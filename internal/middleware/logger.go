package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

func LoggerMiddleware(c *gin.Context) {
	// extract or generate request id
	requestID := c.GetHeader("X-Request-Id")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	cfRay := c.GetHeader("Cf-Ray")
	cfCountry := c.GetHeader("Cf-Ipcountry")

	// create logger with request metadata
	log := slog.Default().With(
		"request_id", requestID,
		"client_ip", c.ClientIP(),
		"cf_ray", cfRay,
		"cf_country", cfCountry,
	)

	// store request id and logger in context
	util.SetGinContextValue(c, model.RequestIDKey, requestID)
	util.SetGinContextValue(c, model.LoggerKey, log)

	c.Next()
}
