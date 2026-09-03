package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/service"
	"github.com/gin-gonic/gin"
)

func TestAuthHandlerRejectsInvalidRegisterJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuth(&fakeAuthService{})
	router := gin.New()
	router.POST("/register", handler.Register)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

type fakeAuthService struct{}

func (*fakeAuthService) Register(context.Context, service.RegisterInput) (*service.AuthResult, error) {
	return nil, nil
}
func (*fakeAuthService) Login(context.Context, service.LoginInput) (*service.AuthResult, error) {
	return nil, nil
}
func (*fakeAuthService) Refresh(context.Context, string) (*service.AuthResult, error) {
	return nil, nil
}
func (*fakeAuthService) Logout(context.Context, string) error            { return nil }
func (*fakeAuthService) Me(context.Context, int64) (*domain.User, error) { return &domain.User{}, nil }
