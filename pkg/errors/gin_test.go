package errors

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteGinUsesSafeErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("request_id", "request-123")

	err := Wrap(ErrDocumentNotFound, errors.New("database host is secret.internal"))
	WriteGin(ctx, err)

	if recorder.Code != 404 {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{`"code":"E02440001"`, `"message":"document not found"`, `"request_id":"request-123"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
	if strings.Contains(body, "secret.internal") {
		t.Fatalf("body exposed internal cause: %s", body)
	}
}

func TestWriteGinMapsUnknownErrorToInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	WriteGin(ctx, errors.New("sensitive failure"))

	if recorder.Code != 500 || !strings.Contains(recorder.Body.String(), `"code":"E00500001"`) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "sensitive failure") {
		t.Fatal("unknown error cause leaked to client")
	}
}
