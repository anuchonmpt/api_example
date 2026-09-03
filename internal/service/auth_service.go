package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/api-example/internal/auth"
	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/example/api-example/pkg/validator"
)

type RegisterInput struct{ Email, Password string }
type LoginInput struct{ Email, Password string }

type AuthResult struct {
	User   *domain.User    `json:"user"`
	Tokens *auth.TokenPair `json:"tokens"`
}

type AuthService struct {
	repository domain.AuthRepository
	jwt        *auth.JWTManager
}

func NewAuthService(repository domain.AuthRepository, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{repository: repository, jwt: jwtManager}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	email, err := validator.Email(input.Email)
	if err != nil {
		return nil, apperrors.WithMessage(apperrors.ErrAuthInvalidRequest, err.Error())
	}
	if len(input.Password) < 8 {
		return nil, apperrors.WithMessage(apperrors.ErrAuthInvalidRequest, "password must contain at least 8 characters")
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.repository.CreateUser(ctx, email, hash, domain.RoleUser)
	if err != nil {
		return nil, err
	}
	return s.issueAndPersist(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	email, err := validator.Email(input.Email)
	if err != nil {
		return nil, apperrors.ErrAuthInvalidCredentials
	}
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperrors.ErrAuthUserNotFound) {
			return nil, apperrors.ErrAuthInvalidCredentials
		}
		return nil, err
	}
	if !user.IsActive || user.DeletedAt != nil || auth.VerifyPassword(user.PasswordHash, input.Password) != nil {
		return nil, apperrors.ErrAuthInvalidCredentials
	}
	return s.issueAndPersist(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, token string) (*AuthResult, error) {
	claims, err := s.jwt.ValidateRefreshToken(strings.TrimSpace(token))
	if err != nil {
		return nil, apperrors.ErrAuthInvalidRefresh
	}
	session, err := s.repository.GetRefreshSessionByTokenID(ctx, claims.ID)
	if err != nil {
		if errors.Is(err, apperrors.ErrAuthInvalidRefresh) || errors.Is(err, apperrors.ErrAuthUserNotFound) {
			return nil, apperrors.ErrAuthInvalidRefresh
		}
		return nil, err
	}
	expected, actual := []byte(session.TokenHash), []byte(HashRefreshToken(token))
	if len(expected) != len(actual) || subtle.ConstantTimeCompare(expected, actual) != 1 || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) {
		return nil, apperrors.ErrAuthInvalidRefresh
	}
	user, err := s.repository.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrAuthUserNotFound) {
			return nil, apperrors.ErrAuthInvalidRefresh
		}
		return nil, err
	}
	if !user.IsActive || user.DeletedAt != nil {
		return nil, apperrors.ErrAuthInvalidRefresh
	}
	pair, replacement, err := s.newSession(user)
	if err != nil {
		return nil, err
	}
	if err := s.repository.RotateRefreshSession(ctx, claims.ID, replacement); err != nil {
		return nil, err
	}
	return &AuthResult{User: publicUser(user), Tokens: pair}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	claims, err := s.jwt.ValidateRefreshToken(strings.TrimSpace(token))
	if err != nil {
		return apperrors.ErrAuthInvalidRefresh
	}
	return s.repository.RevokeRefreshSession(ctx, claims.ID)
}

func (s *AuthService) Me(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return publicUser(user), nil
}

func (s *AuthService) issueAndPersist(ctx context.Context, user *domain.User) (*AuthResult, error) {
	pair, session, err := s.newSession(user)
	if err != nil {
		return nil, err
	}
	if err := s.repository.CreateRefreshSession(ctx, session); err != nil {
		return nil, fmt.Errorf("persist refresh session: %w", err)
	}
	return &AuthResult{User: publicUser(user), Tokens: pair}, nil
}

func (s *AuthService) newSession(user *domain.User) (*auth.TokenPair, domain.RefreshSession, error) {
	pair, err := s.jwt.GenerateTokenPair(user)
	if err != nil {
		return nil, domain.RefreshSession{}, err
	}
	claims, err := s.jwt.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		return nil, domain.RefreshSession{}, err
	}
	return pair, domain.RefreshSession{UserID: user.ID, TokenID: claims.ID, TokenHash: HashRefreshToken(pair.RefreshToken), ExpiresAt: claims.ExpiresAt.Time}, nil
}

func HashRefreshToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func publicUser(user *domain.User) *domain.User {
	copy := *user
	copy.PasswordHash = ""
	return &copy
}
