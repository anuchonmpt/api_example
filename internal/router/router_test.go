package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(Config{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(Config{AllowedOrigins: []string{"https://app.example.com"}})
	request := httptest.NewRequest(http.MethodOptions, "/health", nil)
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("allow origin = %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}
