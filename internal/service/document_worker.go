package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/example/api-example/internal/domain"
)

type DocumentWorker struct {
	repository interface {
		GetByID(context.Context, int64) (*domain.Document, error)
		ClaimProcessing(context.Context, int64) (bool, error)
		MarkReady(context.Context, int64, string, int64) error
		MarkFailed(context.Context, int64, string) error
	}
	storage domain.ObjectStorage
}

func NewDocumentWorker(repository domain.DocumentRepository, storage domain.ObjectStorage) *DocumentWorker {
	return &DocumentWorker{repository: repository, storage: storage}
}

func (w *DocumentWorker) Process(ctx context.Context, job domain.DocumentJob) error {
	document, err := w.repository.GetByID(ctx, job.DocumentID)
	if err != nil {
		return err
	}
	if document.Status == domain.DocumentReady {
		return nil
	}
	claimed, err := w.repository.ClaimProcessing(ctx, document.ID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	object, err := w.storage.Open(ctx, document.StorageKey)
	if err != nil {
		return w.fail(ctx, document.ID, fmt.Errorf("open object: %w", err))
	}
	defer func() { _ = object.Body.Close() }()
	hash := sha256.New()
	size, err := io.Copy(hash, object.Body)
	if err != nil {
		return w.fail(ctx, document.ID, fmt.Errorf("read object: %w", err))
	}
	if size != document.SizeBytes {
		return w.fail(ctx, document.ID, fmt.Errorf("size mismatch: metadata=%d actual=%d", document.SizeBytes, size))
	}
	return w.repository.MarkReady(ctx, document.ID, hex.EncodeToString(hash.Sum(nil)), size)
}

func (w *DocumentWorker) Run(ctx context.Context, queue domain.DocumentQueue) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		claimed, err := queue.Consume(ctx)
		if err != nil {
			return err
		}
		if err := w.Process(ctx, claimed.Job()); err != nil {
			if retryErr := claimed.Retry(ctx, err); retryErr != nil {
				return fmt.Errorf("process job: %v; retry: %w", err, retryErr)
			}
			continue
		}
		if err := claimed.Ack(ctx); err != nil {
			return err
		}
	}
}

func (w *DocumentWorker) fail(ctx context.Context, id int64, cause error) error {
	message := cause.Error()
	if len(message) > 512 {
		message = message[:512]
	}
	if err := w.repository.MarkFailed(ctx, id, message); err != nil {
		return fmt.Errorf("%v; mark failed: %w", cause, err)
	}
	return cause
}
