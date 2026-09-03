package handler

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/middleware"
	"github.com/example/api-example/internal/service"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/example/api-example/pkg/validator"
	"github.com/gin-gonic/gin"
)

type documentService interface {
	Create(context.Context, domain.Actor, service.UploadInput) (*domain.Document, error)
	Get(context.Context, domain.Actor, int64) (*domain.Document, error)
	List(context.Context, domain.Actor, domain.Pagination) (*domain.Page[domain.Document], error)
	Download(context.Context, domain.Actor, int64) (*domain.StoredObject, *domain.Document, error)
	Delete(context.Context, domain.Actor, int64) error
}

type Document struct{ service documentService }

func NewDocument(service documentService) *Document { return &Document{service: service} }

func (h *Document) Create(c *gin.Context) {
	actor, ok := middleware.GetActor(c)
	if !ok {
		apperrors.WriteGin(c, apperrors.ErrDocumentUnauthorized)
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		apperrors.WriteGin(c, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, "multipart file field is required"))
		return
	}
	file, err := header.Open()
	if err != nil {
		apperrors.WriteGin(c, apperrors.ErrDocumentInvalidRequest)
		return
	}
	defer func() { _ = file.Close() }()
	mediaType := header.Header.Get("Content-Type")
	document, err := h.service.Create(c.Request.Context(), actor, service.UploadInput{OriginalName: header.Filename, MediaType: mediaType, SizeBytes: header.Size, Body: file})
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusCreated, document)
}

func (h *Document) Get(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}
	document, err := h.service.Get(c.Request.Context(), actor, id)
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusOK, document)
}

func (h *Document) List(c *gin.Context) {
	actor, ok := middleware.GetActor(c)
	if !ok {
		apperrors.WriteGin(c, apperrors.ErrDocumentUnauthorized)
		return
	}
	page, pageSize, err := validator.Pagination(c.Query("page"), c.Query("page_size"), 20, 100)
	if err != nil {
		apperrors.WriteGin(c, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, err.Error()))
		return
	}
	result, err := h.service.List(c.Request.Context(), actor, domain.Pagination{Page: page, PageSize: pageSize})
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Document) Download(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}
	object, document, err := h.service.Download(c.Request.Context(), actor, id)
	if err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	defer func() { _ = object.Body.Close() }()
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": document.OriginalName})
	c.Header("Content-Disposition", disposition)
	c.DataFromReader(http.StatusOK, object.SizeBytes, document.MediaType, object.Body, nil)
}

func (h *Document) Delete(c *gin.Context) {
	actor, id, ok := actorAndID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), actor, id); err != nil {
		apperrors.WriteGin(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func actorAndID(c *gin.Context) (domain.Actor, int64, bool) {
	actor, ok := middleware.GetActor(c)
	if !ok {
		apperrors.WriteGin(c, apperrors.ErrDocumentUnauthorized)
		return domain.Actor{}, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		apperrors.WriteGin(c, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, fmt.Sprintf("invalid document id %q", c.Param("id"))))
		return domain.Actor{}, 0, false
	}
	return actor, id, true
}
