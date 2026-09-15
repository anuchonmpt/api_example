package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
)

func TestDocumentServiceCreateStoresPersistsAndEnqueues(t *testing.T) {
	repository, storage, queue := &fakeDocumentRepository{}, &fakeObjectStorage{}, &fakeDocumentQueue{}
	service := newTestDocumentService(repository, storage, queue)
	document, err := service.Create(context.Background(), domain.Actor{UserID: 7, Role: domain.RoleUser}, UploadInput{OriginalName: "../report.pdf", MediaType: "application/pdf", SizeBytes: 5, Body: strings.NewReader("hello")})
	if err != nil {
		t.Fatal(err)
	}
	if storage.key != "api-example/documents/7/fixed-key" || storage.body != "hello" {
		t.Fatalf("stored key=%q body=%q", storage.key, storage.body)
	}
	if repository.created.OriginalName != "report.pdf" || document.ID == 0 {
		t.Fatalf("created = %+v", repository.created)
	}
	if repository.created.StorageKey != storage.key || document.URL != "https://cdn.example.com/assets/api-example/documents/7/fixed-key" {
		t.Fatalf("storage key=%q URL=%q", repository.created.StorageKey, document.URL)
	}
	if queue.job.DocumentID != document.ID || queue.job.Attempt != 1 {
		t.Fatalf("job = %+v", queue.job)
	}
}

func TestDocumentServiceCreateCompensatesRepositoryFailure(t *testing.T) {
	repository := &fakeDocumentRepository{createErr: errors.New("database unavailable")}
	storage := &fakeObjectStorage{}
	service := newTestDocumentService(repository, storage, &fakeDocumentQueue{})
	_, err := service.Create(context.Background(), domain.Actor{UserID: 1, Role: domain.RoleUser}, UploadInput{OriginalName: "file.txt", MediaType: "text/plain", SizeBytes: 1, Body: strings.NewReader("x")})
	if err == nil || storage.deletedKey != "api-example/documents/1/fixed-key" {
		t.Fatalf("error=%v deleted=%q", err, storage.deletedKey)
	}
}

func TestDocumentServiceCreateLeavesPendingDocumentWhenQueueFails(t *testing.T) {
	repository := &fakeDocumentRepository{}
	service := newTestDocumentService(repository, &fakeObjectStorage{}, &fakeDocumentQueue{err: errors.New("redis unavailable")})
	document, err := service.Create(context.Background(), domain.Actor{UserID: 1, Role: domain.RoleUser}, UploadInput{OriginalName: "file.txt", MediaType: "text/plain", SizeBytes: 1, Body: strings.NewReader("x")})
	if document == nil || document.Status != domain.DocumentPending || !errors.Is(err, apperrors.ErrQueueUnavailable) {
		t.Fatalf("document=%+v error=%v", document, err)
	}
}

func TestDocumentServiceRejectsOversizedStream(t *testing.T) {
	service := NewDocumentService(&fakeDocumentRepository{}, &fakeObjectStorage{}, &fakeDocumentQueue{}, 4, []string{"text/plain"}, "https://cdn.example.com", "api-example")
	service.keyGenerator = func() (string, error) { return "fixed-key", nil }
	_, err := service.Create(context.Background(), domain.Actor{UserID: 1, Role: domain.RoleUser}, UploadInput{OriginalName: "file.txt", MediaType: "text/plain", SizeBytes: 4, Body: strings.NewReader("12345")})
	if !errors.Is(err, apperrors.ErrDocumentInvalidRequest) {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestDocumentServiceAuthorizationAndDownload(t *testing.T) {
	document := &domain.Document{ID: 9, OwnerID: 2, Status: domain.DocumentReady, StorageKey: "documents/2/key"}
	repository := &fakeDocumentRepository{document: document}
	storage := &fakeObjectStorage{openBody: "content"}
	service := newTestDocumentService(repository, storage, &fakeDocumentQueue{})
	if _, err := service.Get(context.Background(), domain.Actor{UserID: 3, Role: domain.RoleUser}, 9); !errors.Is(err, apperrors.ErrDocumentForbidden) {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := service.Get(context.Background(), domain.Actor{UserID: 3, Role: domain.RoleAdmin}, 9); err != nil {
		t.Fatalf("admin Get() error = %v", err)
	}
	object, gotDocument, err := service.Download(context.Background(), domain.Actor{UserID: 2, Role: domain.RoleUser}, 9)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = object.Body.Close() }()
	data, _ := io.ReadAll(object.Body)
	if string(data) != "content" || gotDocument.ID != 9 || gotDocument.URL != "https://cdn.example.com/assets/documents/2/key" {
		t.Fatalf("download = %q %+v", data, gotDocument)
	}
}

func TestDocumentServiceAddsCDNURLToCreateAndList(t *testing.T) {
	repository := &fakeDocumentRepository{}
	service := newTestDocumentService(repository, &fakeObjectStorage{}, &fakeDocumentQueue{})
	document, err := service.Create(context.Background(), domain.Actor{UserID: 7, Role: domain.RoleUser}, UploadInput{OriginalName: "report.pdf", MediaType: "application/pdf", SizeBytes: 5, Body: strings.NewReader("hello")})
	if err != nil {
		t.Fatal(err)
	}
	if document.URL != "https://cdn.example.com/assets/api-example/documents/7/fixed-key" {
		t.Fatalf("Create() URL = %q", document.URL)
	}
	page, err := service.List(context.Background(), domain.Actor{UserID: 7, Role: domain.RoleUser}, domain.Pagination{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].URL != document.URL {
		t.Fatalf("List() items = %+v", page.Items)
	}
}

func newTestDocumentService(repository domain.DocumentRepository, storage domain.ObjectStorage, queue domain.DocumentQueue) *DocumentService {
	service := NewDocumentService(repository, storage, queue, 1024, []string{"application/pdf", "text/plain"}, "https://cdn.example.com/assets/", "api-example")
	service.keyGenerator = func() (string, error) { return "fixed-key", nil }
	return service
}

type fakeDocumentRepository struct {
	document      *domain.Document
	created       domain.NewDocument
	createErr     error
	deletedID     int64
	readyChecksum string
	readySize     int64
	failedMessage string
}

func (r *fakeDocumentRepository) Create(_ context.Context, input domain.NewDocument) (*domain.Document, error) {
	r.created = input
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.document = &domain.Document{ID: 10, OwnerID: input.OwnerID, OriginalName: input.OriginalName, MediaType: input.MediaType, SizeBytes: input.SizeBytes, StorageKey: input.StorageKey, Status: domain.DocumentPending}
	return r.document, nil
}
func (r *fakeDocumentRepository) GetByID(context.Context, int64) (*domain.Document, error) {
	if r.document == nil {
		return nil, apperrors.ErrDocumentNotFound
	}
	return r.document, nil
}
func (r *fakeDocumentRepository) List(context.Context, *int64, domain.Pagination) ([]domain.Document, int64, error) {
	if r.document == nil {
		return []domain.Document{}, 0, nil
	}
	return []domain.Document{*r.document}, 1, nil
}
func (r *fakeDocumentRepository) SoftDelete(_ context.Context, id int64) error {
	r.deletedID = id
	return nil
}
func (r *fakeDocumentRepository) ClaimProcessing(context.Context, int64) (bool, error) {
	return true, nil
}
func (r *fakeDocumentRepository) MarkReady(_ context.Context, _ int64, checksum string, size int64) error {
	r.readyChecksum, r.readySize = checksum, size
	return nil
}
func (r *fakeDocumentRepository) MarkFailed(_ context.Context, _ int64, message string) error {
	r.failedMessage = message
	return nil
}

type fakeObjectStorage struct {
	key, body, deletedKey, openBody string
	putErr, deleteErr               error
}

func (s *fakeObjectStorage) Put(_ context.Context, key string, body io.Reader, _ domain.ObjectMetadata) error {
	s.key = key
	data, err := io.ReadAll(body)
	s.body = string(data)
	if err != nil {
		return err
	}
	return s.putErr
}
func (s *fakeObjectStorage) Open(context.Context, string) (*domain.StoredObject, error) {
	return &domain.StoredObject{Body: io.NopCloser(bytes.NewBufferString(s.openBody)), SizeBytes: int64(len(s.openBody))}, nil
}
func (s *fakeObjectStorage) Delete(_ context.Context, key string) error {
	s.deletedKey = key
	return s.deleteErr
}

type fakeDocumentQueue struct {
	job domain.DocumentJob
	err error
}

func (q *fakeDocumentQueue) Enqueue(_ context.Context, job domain.DocumentJob) error {
	q.job = job
	return q.err
}
func (q *fakeDocumentQueue) Consume(context.Context) (domain.ClaimedDocumentJob, error) {
	return nil, errors.New("not implemented")
}
