package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/api-example/internal/auth"
	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
)

func TestAuthServiceRegisterNormalizesEmail(t *testing.T) {
	repository := &fakeAuthRepository{}
	service := NewAuthService(repository, auth.NewJWTManager(testJWTSecret, "issuer", "audience", time.Hour, 24*time.Hour))

	result, err := service.Register(context.Background(), RegisterInput{Email: " USER@Example.COM ", Password: "password-123"})
	if err != nil {
		t.Fatal(err)
	}
	if repository.createdEmail != "user@example.com" || result.User.Email != "user@example.com" {
		t.Fatalf("result = %+v, email = %q", result, repository.createdEmail)
	}
	if repository.createdPassword == "password-123" {
		t.Fatal("plaintext password passed to repository")
	}
	if repository.createdSession.TokenHash == result.Tokens.RefreshToken {
		t.Fatal("plaintext refresh token persisted")
	}
}

func TestAuthServiceLoginRejectsWrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeAuthRepository{user: &domain.User{ID: 1, Email: "user@example.com", PasswordHash: hash, Role: domain.RoleUser, IsActive: true}}
	service := NewAuthService(repository, auth.NewJWTManager(testJWTSecret, "issuer", "audience", time.Hour, 24*time.Hour))

	_, err = service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "wrong-password"})
	if !errors.Is(err, apperrors.ErrAuthInvalidCredentials) {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestAuthServiceLoginPreservesRepositoryFailure(t *testing.T) {
	want := errors.New("database unavailable")
	repository := &fakeAuthRepository{getUserErr: want}
	service := NewAuthService(repository, auth.NewJWTManager(testJWTSecret, "issuer", "audience", time.Hour, 24*time.Hour))
	_, err := service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "password-123"})
	if !errors.Is(err, want) {
		t.Fatalf("Login() error = %v, want repository failure", err)
	}
}

func TestAuthServiceRefreshRotatesSession(t *testing.T) {
	repository := &fakeAuthRepository{user: &domain.User{ID: 5, Email: "user@example.com", Role: domain.RoleUser, IsActive: true}}
	manager := auth.NewJWTManager(testJWTSecret, "issuer", "audience", time.Hour, 24*time.Hour)
	initial, err := manager.GenerateTokenPair(repository.user)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.ValidateRefreshToken(initial.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	repository.session = &domain.RefreshSession{UserID: 5, TokenID: claims.ID, TokenHash: HashRefreshToken(initial.RefreshToken), ExpiresAt: time.Now().Add(time.Hour)}
	service := NewAuthService(repository, manager)

	result, err := service.Refresh(context.Background(), initial.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if repository.rotatedOldID != claims.ID || repository.rotatedSession.TokenID == claims.ID || result.Tokens.RefreshToken == initial.RefreshToken {
		t.Fatalf("rotation did not replace token: old=%q replacement=%+v", repository.rotatedOldID, repository.rotatedSession)
	}
}

const testJWTSecret = "01234567890123456789012345678901"

type fakeAuthRepository struct {
	user            *domain.User
	session         *domain.RefreshSession
	createdEmail    string
	createdPassword string
	createdSession  domain.RefreshSession
	rotatedOldID    string
	rotatedSession  domain.RefreshSession
	getUserErr      error
}

func (r *fakeAuthRepository) CreateUser(_ context.Context, email, password string, role domain.Role) (*domain.User, error) {
	r.createdEmail, r.createdPassword = email, password
	r.user = &domain.User{ID: 1, Email: email, PasswordHash: password, Role: role, IsActive: true}
	return r.user, nil
}
func (r *fakeAuthRepository) GetUserByEmail(context.Context, string) (*domain.User, error) {
	if r.getUserErr != nil {
		return nil, r.getUserErr
	}
	if r.user == nil {
		return nil, apperrors.ErrAuthUserNotFound
	}
	return r.user, nil
}
func (r *fakeAuthRepository) GetUserByID(context.Context, int64) (*domain.User, error) {
	if r.user == nil {
		return nil, apperrors.ErrAuthUserNotFound
	}
	return r.user, nil
}
func (r *fakeAuthRepository) CreateRefreshSession(_ context.Context, session domain.RefreshSession) error {
	r.createdSession = session
	return nil
}
func (r *fakeAuthRepository) GetRefreshSessionByTokenID(context.Context, string) (*domain.RefreshSession, error) {
	if r.session == nil {
		return nil, apperrors.ErrAuthInvalidRefresh
	}
	return r.session, nil
}
func (r *fakeAuthRepository) RotateRefreshSession(_ context.Context, old string, replacement domain.RefreshSession) error {
	r.rotatedOldID, r.rotatedSession = old, replacement
	return nil
}
func (r *fakeAuthRepository) RevokeRefreshSession(context.Context, string) error { return nil }
