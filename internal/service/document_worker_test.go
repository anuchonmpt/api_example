package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/example/api-example/internal/domain"
)

func TestDocumentWorkerProcessesPendingDocument(t *testing.T) {
	t.Parallel()
	repository := &fakeDocumentRepository{document: &domain.Document{ID: 3, Status: domain.DocumentPending, SizeBytes: 5, StorageKey: "key"}}
	worker := NewDocumentWorker(repository, &fakeObjectStorage{openBody: "hello"})
	if err := worker.Process(context.Background(), domain.DocumentJob{DocumentID: 3, Attempt: 1}); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("hello"))
	if repository.readyChecksum != hex.EncodeToString(digest[:]) || repository.readySize != 5 {
		t.Fatalf("ready checksum=%q size=%d", repository.readyChecksum, repository.readySize)
	}
}

func TestDocumentWorkerMarksSizeMismatchFailed(t *testing.T) {
	t.Parallel()
	repository := &fakeDocumentRepository{document: &domain.Document{ID: 3, Status: domain.DocumentPending, SizeBytes: 99, StorageKey: "key"}}
	worker := NewDocumentWorker(repository, &fakeObjectStorage{openBody: "hello"})
	if err := worker.Process(context.Background(), domain.DocumentJob{DocumentID: 3, Attempt: 1}); err == nil {
		t.Fatal("size mismatch succeeded")
	}
	if !strings.Contains(repository.failedMessage, "size mismatch") {
		t.Fatalf("failure = %q", repository.failedMessage)
	}
}

func TestDocumentWorkerSkipsReadyDocument(t *testing.T) {
	t.Parallel()
	repository := &fakeDocumentRepository{document: &domain.Document{ID: 3, Status: domain.DocumentReady}}
	worker := NewDocumentWorker(repository, &fakeObjectStorage{})
	if err := worker.Process(context.Background(), domain.DocumentJob{DocumentID: 3, Attempt: 1}); err != nil {
		t.Fatal(err)
	}
	if repository.readyChecksum != "" || repository.failedMessage != "" {
		t.Fatal("ready document was reprocessed")
	}
}
