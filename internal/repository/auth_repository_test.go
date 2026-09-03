package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/repository/db"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestUserFromGetByEmailRow(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	row := &db.GetUserByEmailRow{ID: 7, Email: "user@example.com", PasswordHash: "hash", RoleName: "user", IsActive: true, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	got := userFromGetByEmailRow(row)
	if got.ID != 7 || got.Role != domain.RoleUser || got.CreatedAt != now || got.DeletedAt != nil {
		t.Fatalf("mapped user = %+v", got)
	}
}

func TestMapRepositoryError(t *testing.T) {
	t.Parallel()
	err := mapRepositoryError(pgx.ErrNoRows, apperrors.ErrAuthUserNotFound)
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeAuthUserNotFound {
		t.Fatalf("mapped error = %v", err)
	}
}
