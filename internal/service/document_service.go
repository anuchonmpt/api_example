package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
)

type UploadInput struct {
	OriginalName string
	MediaType    string
	SizeBytes    int64
	Body         io.Reader
}

type DocumentService struct {
	repository       domain.DocumentRepository
	storage          domain.ObjectStorage
	queue            domain.DocumentQueue
	maxUploadBytes   int64
	allowedMediaType map[string]struct{}
	keyGenerator     func() (string, error)
}

func NewDocumentService(repository domain.DocumentRepository, storage domain.ObjectStorage, queue domain.DocumentQueue, maxUploadBytes int64, allowedMediaTypes []string) *DocumentService {
	allowed := make(map[string]struct{}, len(allowedMediaTypes))
	for _, mediaType := range allowedMediaTypes {
		allowed[strings.ToLower(strings.TrimSpace(mediaType))] = struct{}{}
	}
	return &DocumentService{repository: repository, storage: storage, queue: queue, maxUploadBytes: maxUploadBytes, allowedMediaType: allowed, keyGenerator: randomObjectID}
}

func (s *DocumentService) Create(ctx context.Context, actor domain.Actor, input UploadInput) (*domain.Document, error) {
	if actor.UserID <= 0 {
		return nil, apperrors.ErrDocumentUnauthorized
	}
	name := filepath.Base(strings.ReplaceAll(strings.TrimSpace(input.OriginalName), "\\", "/"))
	if name == "" || name == "." {
		return nil, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, "file name is required")
	}
	mediaType := strings.ToLower(strings.TrimSpace(input.MediaType))
	if _, allowed := s.allowedMediaType[mediaType]; !allowed {
		return nil, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, "media type is not allowed")
	}
	if input.Body == nil || input.SizeBytes <= 0 || input.SizeBytes > s.maxUploadBytes {
		return nil, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, "file size is invalid")
	}
	id, err := s.keyGenerator()
	if err != nil {
		return nil, fmt.Errorf("generate storage key: %w", err)
	}
	key := fmt.Sprintf("documents/%d/%s", actor.UserID, id)
	counter := &countingReader{reader: &maxBytesReader{reader: input.Body, remaining: s.maxUploadBytes}}
	if err := s.storage.Put(ctx, key, counter, domain.ObjectMetadata{MediaType: mediaType}); err != nil {
		if errors.Is(err, errUploadTooLarge) {
			return nil, apperrors.WithMessage(apperrors.ErrDocumentInvalidRequest, "file exceeds maximum upload size")
		}
		return nil, apperrors.Wrap(apperrors.ErrStorageUnavailable, err)
	}
	document, err := s.repository.Create(ctx, domain.NewDocument{OwnerID: actor.UserID, OriginalName: name, MediaType: mediaType, SizeBytes: counter.count, StorageKey: key})
	if err != nil {
		if cleanupErr := s.storage.Delete(ctx, key); cleanupErr != nil {
			return nil, fmt.Errorf("persist document: %w; compensate storage: %v", err, cleanupErr)
		}
		return nil, err
	}
	if err := s.queue.Enqueue(ctx, domain.DocumentJob{DocumentID: document.ID, Attempt: 1}); err != nil {
		return document, apperrors.Wrap(apperrors.ErrQueueUnavailable, err)
	}
	return document, nil
}

func (s *DocumentService) Get(ctx context.Context, actor domain.Actor, id int64) (*domain.Document, error) {
	document, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := authorizeDocument(actor, document); err != nil {
		return nil, err
	}
	return document, nil
}

func (s *DocumentService) List(ctx context.Context, actor domain.Actor, pagination domain.Pagination) (*domain.Page[domain.Document], error) {
	if actor.UserID <= 0 {
		return nil, apperrors.ErrDocumentUnauthorized
	}
	pagination = domain.NormalizePagination(pagination.Page, pagination.PageSize)
	var ownerID *int64
	if !actor.IsAdmin() {
		owner := actor.UserID
		ownerID = &owner
	}
	items, total, err := s.repository.List(ctx, ownerID, pagination)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.Document{}
	}
	return &domain.Page[domain.Document]{Items: items, Page: pagination.Page, PageSize: pagination.PageSize, TotalItems: total}, nil
}

func (s *DocumentService) Download(ctx context.Context, actor domain.Actor, id int64) (*domain.StoredObject, *domain.Document, error) {
	document, err := s.Get(ctx, actor, id)
	if err != nil {
		return nil, nil, err
	}
	if document.Status != domain.DocumentReady {
		return nil, nil, apperrors.ErrDocumentNotReady
	}
	object, err := s.storage.Open(ctx, document.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return object, document, nil
}

func (s *DocumentService) Delete(ctx context.Context, actor domain.Actor, id int64) error {
	document, err := s.Get(ctx, actor, id)
	if err != nil {
		return err
	}
	return s.repository.SoftDelete(ctx, document.ID)
}

func authorizeDocument(actor domain.Actor, document *domain.Document) error {
	if actor.UserID <= 0 {
		return apperrors.ErrDocumentUnauthorized
	}
	if !actor.IsAdmin() && document.OwnerID != actor.UserID {
		return apperrors.ErrDocumentForbidden
	}
	return nil
}

func randomObjectID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

var errUploadTooLarge = errors.New("upload exceeds maximum size")

type maxBytesReader struct {
	reader          io.Reader
	remaining       int64
	checkedOverflow bool
}

func (r *maxBytesReader) Read(buffer []byte) (int, error) {
	if r.remaining == 0 {
		if r.checkedOverflow {
			return 0, io.EOF
		}
		r.checkedOverflow = true
		var extra [1]byte
		n, err := r.reader.Read(extra[:])
		if n > 0 {
			return 0, errUploadTooLarge
		}
		return 0, err
	}
	if int64(len(buffer)) > r.remaining {
		buffer = buffer[:r.remaining]
	}
	n, err := r.reader.Read(buffer)
	r.remaining -= int64(n)
	return n, err
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	r.count += int64(n)
	return n, err
}
