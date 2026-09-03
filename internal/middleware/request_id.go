package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if requestID == "" || len(requestID) > 128 {
			requestID = newRequestID()
		}
		c.Set(apperrors.RequestIDContextKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "request-unavailable"
	}
	return hex.EncodeToString(buffer)
}
