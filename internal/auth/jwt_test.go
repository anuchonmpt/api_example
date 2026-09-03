package auth

import (
	"testing"
	"time"

	"github.com/example/api-example/internal/domain"
)

func TestJWTTokenTypes(t *testing.T) {
	t.Parallel()
	manager := NewJWTManager("01234567890123456789012345678901", "issuer", "audience", time.Hour, 24*time.Hour)
	pair, err := manager.GenerateTokenPair(&domain.User{ID: 42, Email: "user@example.com", Role: domain.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	access, err := manager.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if access.UserID != "42" || access.TokenType != string(AccessTokenType) {
		t.Fatalf("access claims = %+v", access)
	}
	refresh, err := manager.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if refresh.ID == "" || refresh.TokenType != string(RefreshTokenType) {
		t.Fatalf("refresh claims = %+v", refresh)
	}
	if _, err := manager.ValidateAccessToken(pair.RefreshToken); err == nil {
		t.Fatal("refresh token accepted as access token")
	}
	if _, err := manager.ValidateRefreshToken(pair.AccessToken); err == nil {
		t.Fatal("access token accepted as refresh token")
	}
}

func TestJWTRejectsWrongAudienceAndExpiredToken(t *testing.T) {
	t.Parallel()
	issuer := NewJWTManager("01234567890123456789012345678901", "issuer", "audience", -time.Second, time.Hour)
	pair, err := issuer.GenerateTokenPair(&domain.User{ID: 1, Email: "user@example.com", Role: domain.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.ValidateAccessToken(pair.AccessToken); err == nil {
		t.Fatal("expired token accepted")
	}

	otherAudience := NewJWTManager("01234567890123456789012345678901", "issuer", "different", time.Hour, time.Hour)
	if _, err := otherAudience.ValidateRefreshToken(pair.RefreshToken); err == nil {
		t.Fatal("wrong audience accepted")
	}
}

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()
	if got := ExtractBearerToken("Bearer token-value"); got != "token-value" {
		t.Fatalf("got %q", got)
	}
	if got := ExtractBearerToken("bearer token-value"); got != "" {
		t.Fatalf("accepted malformed scheme: %q", got)
	}
}
