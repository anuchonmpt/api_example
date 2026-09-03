package repository

import (
	"context"
	"fmt"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/repository/db"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type documentQueries interface {
	CreateDocument(context.Context, db.CreateDocumentParams) (*db.Document, error)
	GetDocumentByID(context.Context, int64) (*db.Document, error)
	ListDocumentsByOwner(context.Context, db.ListDocumentsByOwnerParams) ([]*db.Document, error)
	CountDocumentsByOwner(context.Context, int64) (int64, error)
	ListAllDocuments(context.Context, db.ListAllDocumentsParams) ([]*db.Document, error)
	CountAllDocuments(context.Context) (int64, error)
	SoftDeleteDocument(context.Context, int64) (int64, error)
	ClaimDocumentProcessing(context.Context, int64) (int64, error)
	MarkDocumentReady(context.Context, db.MarkDocumentReadyParams) (int64, error)
	MarkDocumentFailed(context.Context, db.MarkDocumentFailedParams) (int64, error)
}

type DocumentRepository struct{ queries documentQueries }

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{queries: db.New(pool)}
}

func (r *DocumentRepository) Create(ctx context.Context, input domain.NewDocument) (*domain.Document, error) {
	row, err := r.queries.CreateDocument(ctx, db.CreateDocumentParams{OwnerID: input.OwnerID, OriginalName: input.OriginalName, MediaType: input.MediaType, SizeBytes: input.SizeBytes, StorageKey: input.StorageKey})
	if err != nil {
		return nil, mapRepositoryError(err, apperrors.ErrDocumentConflict)
	}
	return documentFromDB(row), nil
}

func (r *DocumentRepository) GetByID(ctx context.Context, id int64) (*domain.Document, error) {
	row, err := r.queries.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err, apperrors.ErrDocumentNotFound)
	}
	return documentFromDB(row), nil
}

func (r *DocumentRepository) List(ctx context.Context, ownerID *int64, page domain.Pagination) ([]domain.Document, int64, error) {
	var rows []*db.Document
	var total int64
	var err error
	if ownerID == nil {
		rows, err = r.queries.ListAllDocuments(ctx, db.ListAllDocumentsParams{PageLimit: page.Limit(), PageOffset: page.Offset()})
		if err == nil {
			total, err = r.queries.CountAllDocuments(ctx)
		}
	} else {
		rows, err = r.queries.ListDocumentsByOwner(ctx, db.ListDocumentsByOwnerParams{OwnerID: *ownerID, PageLimit: page.Limit(), PageOffset: page.Offset()})
		if err == nil {
			total, err = r.queries.CountDocumentsByOwner(ctx, *ownerID)
		}
	}
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		result = append(result, *documentFromDB(row))
	}
	return result, total, nil
}

func (r *DocumentRepository) SoftDelete(ctx context.Context, id int64) error {
	rows, err := r.queries.SoftDeleteDocument(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return apperrors.ErrDocumentNotFound
	}
	return nil
}

func (r *DocumentRepository) ClaimProcessing(ctx context.Context, id int64) (bool, error) {
	rows, err := r.queries.ClaimDocumentProcessing(ctx, id)
	return rows == 1, err
}

func (r *DocumentRepository) MarkReady(ctx context.Context, id int64, checksum string, size int64) error {
	rows, err := r.queries.MarkDocumentReady(ctx, db.MarkDocumentReadyParams{ID: id, ChecksumSha256: &checksum, SizeBytes: size})
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("document %d is not processing", id)
	}
	return nil
}

func (r *DocumentRepository) MarkFailed(ctx context.Context, id int64, message string) error {
	rows, err := r.queries.MarkDocumentFailed(ctx, db.MarkDocumentFailedParams{ID: id, ProcessingError: &message})
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("document %d is not processing", id)
	}
	return nil
}

func documentFromDB(row *db.Document) *domain.Document {
	return &domain.Document{ID: row.ID, OwnerID: row.OwnerID, OriginalName: row.OriginalName, MediaType: row.MediaType, SizeBytes: row.SizeBytes, StorageKey: row.StorageKey, ChecksumSHA256: row.ChecksumSha256, Status: domain.DocumentStatus(row.Status), ProcessingError: row.ProcessingError, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt), DeletedAt: timestampPtr(row.DeletedAt)}
}
