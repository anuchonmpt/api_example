package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/repository/db"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type authQueries interface {
	CreateUser(context.Context, db.CreateUserParams) (*db.CreateUserRow, error)
	GetUserByEmail(context.Context, string) (*db.GetUserByEmailRow, error)
	GetUserByID(context.Context, int64) (*db.GetUserByIDRow, error)
	CreateRefreshSession(context.Context, db.CreateRefreshSessionParams) error
	GetRefreshSessionByTokenID(context.Context, string) (*db.RefreshSession, error)
	RevokeRefreshSession(context.Context, string) (int64, error)
}

type AuthRepository struct {
	pool    *pgxpool.Pool
	queries authQueries
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool, queries: db.New(pool)}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email, passwordHash string, role domain.Role) (*domain.User, error) {
	row, err := r.queries.CreateUser(ctx, db.CreateUserParams{Email: email, PasswordHash: passwordHash, RoleName: string(role)})
	if err != nil {
		return nil, mapRepositoryError(err, apperrors.ErrAuthEmailConflict)
	}
	return userFromCreateRow(row), nil
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapRepositoryError(err, apperrors.ErrAuthUserNotFound)
	}
	return userFromGetByEmailRow(row), nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapRepositoryError(err, apperrors.ErrAuthUserNotFound)
	}
	return userFromGetByIDRow(row), nil
}

func (r *AuthRepository) CreateRefreshSession(ctx context.Context, session domain.RefreshSession) error {
	return r.queries.CreateRefreshSession(ctx, refreshSessionParams(session))
}

func (r *AuthRepository) GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (*domain.RefreshSession, error) {
	row, err := r.queries.GetRefreshSessionByTokenID(ctx, tokenID)
	if err != nil {
		return nil, mapRepositoryError(err, apperrors.ErrAuthInvalidRefresh)
	}
	return refreshSessionFromDB(row), nil
}

func (r *AuthRepository) RotateRefreshSession(ctx context.Context, oldTokenID string, replacement domain.RefreshSession) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.New(tx)
	rows, err := queries.RotateRefreshSession(ctx, db.RotateRefreshSessionParams{ReplacedByTokenID: &replacement.TokenID, TokenID: oldTokenID})
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	if rows != 1 {
		return apperrors.ErrAuthInvalidRefresh
	}
	if err := queries.CreateRefreshSession(ctx, refreshSessionParams(replacement)); err != nil {
		return fmt.Errorf("create replacement refresh session: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *AuthRepository) RevokeRefreshSession(ctx context.Context, tokenID string) error {
	_, err := r.queries.RevokeRefreshSession(ctx, tokenID)
	return err
}

func mapRepositoryError(err error, notFound *apperrors.AppError) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.Wrap(notFound, err)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.Wrap(apperrors.ErrAuthEmailConflict, err)
	}
	return err
}

func userFromCreateRow(row *db.CreateUserRow) *domain.User {
	return &domain.User{ID: row.ID, Email: row.Email, PasswordHash: row.PasswordHash, Role: domain.Role(row.RoleName), IsActive: row.IsActive, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt), DeletedAt: timestampPtr(row.DeletedAt)}
}

func userFromGetByEmailRow(row *db.GetUserByEmailRow) *domain.User {
	return &domain.User{ID: row.ID, Email: row.Email, PasswordHash: row.PasswordHash, Role: domain.Role(row.RoleName), IsActive: row.IsActive, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt), DeletedAt: timestampPtr(row.DeletedAt)}
}

func userFromGetByIDRow(row *db.GetUserByIDRow) *domain.User {
	return &domain.User{ID: row.ID, Email: row.Email, PasswordHash: row.PasswordHash, Role: domain.Role(row.RoleName), IsActive: row.IsActive, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt), DeletedAt: timestampPtr(row.DeletedAt)}
}

func refreshSessionParams(session domain.RefreshSession) db.CreateRefreshSessionParams {
	return db.CreateRefreshSessionParams{UserID: session.UserID, TokenID: session.TokenID, TokenHash: session.TokenHash, ExpiresAt: pgtype.Timestamptz{Time: session.ExpiresAt, Valid: true}}
}

func refreshSessionFromDB(row *db.RefreshSession) *domain.RefreshSession {
	return &domain.RefreshSession{ID: row.ID, UserID: row.UserID, TokenID: row.TokenID, TokenHash: row.TokenHash, ExpiresAt: timestamp(row.ExpiresAt), RevokedAt: timestampPtr(row.RevokedAt), ReplacedByTokenID: row.ReplacedByTokenID, CreatedAt: timestamp(row.CreatedAt)}
}

func timestamp(value pgtype.Timestamptz) time.Time { return value.Time }

func timestampPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
