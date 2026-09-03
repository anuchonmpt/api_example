package queue

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/api-example/internal/domain"
)

func TestRedisQueueEnqueueAndConsume(t *testing.T) {
	t.Parallel()
	backend := &fakeListBackend{}
	queue := newRedisQueue(backend, "main", "processing", "dead", 3)
	if err := queue.Enqueue(context.Background(), domain.DocumentJob{DocumentID: 42, Attempt: 1}); err != nil {
		t.Fatal(err)
	}
	if backend.pushKey != "main" || !strings.Contains(backend.payload, `"document_id":42`) {
		t.Fatalf("push = %q %q", backend.pushKey, backend.payload)
	}
	backend.movePayload = backend.payload
	claimed, err := queue.Consume(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Job().DocumentID != 42 {
		t.Fatalf("job = %+v", claimed.Job())
	}
	if err := claimed.Ack(context.Background()); err != nil {
		t.Fatal(err)
	}
	if backend.removeKey != "processing" {
		t.Fatalf("remove key = %q", backend.removeKey)
	}
}

func TestRedisQueueRetryAndDeadLetter(t *testing.T) {
	t.Parallel()
	backend := &fakeListBackend{movePayload: `{"document_id":9,"attempt":2}`}
	queue := newRedisQueue(backend, "main", "processing", "dead", 2)
	claimed, err := queue.Consume(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := claimed.Retry(context.Background(), errors.New("failed")); err != nil {
		t.Fatal(err)
	}
	if backend.pushKey != "dead" {
		t.Fatalf("push key = %q, want dead", backend.pushKey)
	}
}

func TestRedisQueueRejectsInvalidPayload(t *testing.T) {
	t.Parallel()
	queue := newRedisQueue(&fakeListBackend{movePayload: `{"document_id":0}`}, "main", "processing", "dead", 3)
	if _, err := queue.Consume(context.Background()); err == nil {
		t.Fatal("invalid payload accepted")
	}
}

type fakeListBackend struct{ pushKey, payload, movePayload, removeKey string }

func (f *fakeListBackend) Push(_ context.Context, key, payload string) error {
	f.pushKey, f.payload = key, payload
	return nil
}
func (f *fakeListBackend) Move(context.Context, string, string, time.Duration) (string, error) {
	return f.movePayload, nil
}
func (f *fakeListBackend) Remove(_ context.Context, key, payload string) error {
	f.removeKey = key
	return nil
}
