package repository

import (
	"testing"
	"time"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/repository/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestDocumentFromDB(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	checksum := "abc"
	row := &db.Document{ID: 9, OwnerID: 3, OriginalName: "report.pdf", MediaType: "application/pdf", SizeBytes: 12, StorageKey: "documents/3/key", ChecksumSha256: &checksum, Status: "ready", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	got := documentFromDB(row)
	if got.ID != 9 || got.Status != domain.DocumentReady || got.ChecksumSHA256 == nil || *got.ChecksumSHA256 != checksum {
		t.Fatalf("mapped document = %+v", got)
	}
}
