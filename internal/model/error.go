package model

import (
	"time"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestId"`
}

func NewAPIError(code int, message string, c *gin.Context) APIError {
	return APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		RequestID: c.GetString(RequestIDKey),
	}
}
