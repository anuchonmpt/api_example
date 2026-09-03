package domain

import (
	"context"
	"io"
)

type AuthRepository interface {
	CreateUser(context.Context, string, string, Role) (*User, error)
	GetUserByEmail(context.Context, string) (*User, error)
	GetUserByID(context.Context, int64) (*User, error)
	CreateRefreshSession(context.Context, RefreshSession) error
	GetRefreshSessionByTokenID(context.Context, string) (*RefreshSession, error)
	RotateRefreshSession(context.Context, string, RefreshSession) error
	RevokeRefreshSession(context.Context, string) error
}

type DocumentRepository interface {
	Create(context.Context, NewDocument) (*Document, error)
	GetByID(context.Context, int64) (*Document, error)
	List(context.Context, *int64, Pagination) ([]Document, int64, error)
	SoftDelete(context.Context, int64) error
	ClaimProcessing(context.Context, int64) (bool, error)
	MarkReady(context.Context, int64, string, int64) error
	MarkFailed(context.Context, int64, string) error
}

type ObjectMetadata struct{ MediaType string }

type StoredObject struct {
	Body      io.ReadCloser
	SizeBytes int64
	MediaType string
}

type ObjectStorage interface {
	Put(context.Context, string, io.Reader, ObjectMetadata) error
	Open(context.Context, string) (*StoredObject, error)
	Delete(context.Context, string) error
}

type DocumentJob struct {
	DocumentID int64 `json:"document_id"`
	Attempt    int   `json:"attempt"`
}

type ClaimedDocumentJob interface {
	Job() DocumentJob
	Ack(context.Context) error
	Retry(context.Context, error) error
}

type DocumentQueue interface {
	Enqueue(context.Context, DocumentJob) error
	Consume(context.Context) (ClaimedDocumentJob, error)
}
