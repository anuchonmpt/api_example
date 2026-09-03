package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/api-example/internal/auth"
	"github.com/example/api-example/internal/domain"
	"github.com/gin-gonic/gin"
)

func TestAuthMiddlewareSetsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("01234567890123456789012345678901", "issuer", "audience", time.Hour, 24*time.Hour)
	pair, err := manager.GenerateTokenPair(&domain.User{ID: 42, Email: "user@example.com", Role: domain.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(Auth(manager))
	router.GET("/", func(c *gin.Context) {
		actor, ok := GetActor(c)
		if !ok || actor.UserID != 42 || actor.Role != domain.RoleUser {
			t.Fatalf("actor = %+v ok=%v", actor, ok)
		}
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthMiddlewareRejectsMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("01234567890123456789012345678901", "issuer", "audience", time.Hour, 24*time.Hour)
	router := gin.New()
	router.Use(RequestID(), Auth(manager))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}
