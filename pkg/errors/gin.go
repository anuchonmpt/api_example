package errors

import "github.com/gin-gonic/gin"

const RequestIDContextKey = "request_id"

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func WriteGin(c *gin.Context, err error) {
	appErr := As(err)
	requestID, _ := c.Get(RequestIDContextKey)
	c.JSON(appErr.Status, errorEnvelope{Error: errorBody{Code: appErr.Code, Message: appErr.Message, RequestID: stringValue(requestID)}})
}

func AbortGin(c *gin.Context, err error) {
	WriteGin(c, err)
	c.Abort()
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
