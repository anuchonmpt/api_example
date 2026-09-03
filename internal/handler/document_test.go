package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/middleware"
	"github.com/example/api-example/internal/service"
	"github.com/gin-gonic/gin"
)

func TestDocumentHandlerCreateRequiresFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewDocument(&fakeDocumentService{})
	router := gin.New()
	router.POST("/documents", withActor(handler.Create))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/documents", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func withActor(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ActorContextKey, domain.Actor{UserID: 1, Role: domain.RoleUser})
		next(c)
	}
}

type fakeDocumentService struct{}

func (*fakeDocumentService) Create(context.Context, domain.Actor, service.UploadInput) (*domain.Document, error) {
	return nil, nil
}
func (*fakeDocumentService) Get(context.Context, domain.Actor, int64) (*domain.Document, error) {
	return nil, nil
}
func (*fakeDocumentService) List(context.Context, domain.Actor, domain.Pagination) (*domain.Page[domain.Document], error) {
	return &domain.Page[domain.Document]{Items: []domain.Document{}}, nil
}
func (*fakeDocumentService) Download(context.Context, domain.Actor, int64) (*domain.StoredObject, *domain.Document, error) {
	return nil, nil, nil
}
func (*fakeDocumentService) Delete(context.Context, domain.Actor, int64) error { return nil }
