package util

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/model"
)

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(model.LoggerKey).(*slog.Logger); ok {
		return log
	}

	return slog.Default()
}

func LoggerFromGinContext(c *gin.Context) *slog.Logger {
	if log, ok := c.Get(model.LoggerKey); ok {
		return log.(*slog.Logger)
	}

	return slog.Default()
}

func ClaimsFromContext(ctx context.Context) *model.Claims {
	if claims, ok := ctx.Value(model.ClaimsKey).(*model.Claims); ok {
		return claims
	}

	return nil
}

func SetGinContextValue(c *gin.Context, key model.CTXKey, value any) {
	c.Set(key, value)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), key, value))
}
