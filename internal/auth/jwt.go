package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/example/api-example/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessTokenType  TokenType = "access"
	RefreshTokenType TokenType = "refresh"
)

type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type JWTManager struct {
	secret, issuer, audience string
	accessTTL, refreshTTL    time.Duration
}

func NewJWTManager(secret, issuer, audience string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{secret: secret, issuer: issuer, audience: audience, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (m *JWTManager) GenerateTokenPair(user *domain.User) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(m.accessTTL)
	refreshExpiry := now.Add(m.refreshTTL)
	access, err := m.sign(user, AccessTokenType, accessExpiry, "")
	if err != nil {
		return nil, err
	}
	refreshID, err := randomTokenID()
	if err != nil {
		return nil, err
	}
	refresh, err := m.sign(user, RefreshTokenType, refreshExpiry, refreshID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresAt: accessExpiry}, nil
}

func (m *JWTManager) sign(user *domain.User, tokenType TokenType, expiresAt time.Time, tokenID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: strconv.FormatInt(user.ID, 10), Email: user.Email, Role: string(user.Role), TokenType: string(tokenType),
		RegisteredClaims: jwt.RegisteredClaims{Issuer: m.issuer, Subject: strconv.FormatInt(user.ID, 10), Audience: jwt.ClaimStrings{m.audience}, ExpiresAt: jwt.NewNumericDate(expiresAt), NotBefore: jwt.NewNumericDate(now), IssuedAt: jwt.NewNumericDate(now), ID: tokenID},
	}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("sign %s token: %w", tokenType, err)
	}
	return value, nil
}

func (m *JWTManager) ValidateAccessToken(value string) (*Claims, error) {
	return m.validate(value, AccessTokenType)
}
func (m *JWTManager) ValidateRefreshToken(value string) (*Claims, error) {
	return m.validate(value, RefreshTokenType)
}

func (m *JWTManager) validate(value string, expected TokenType) (*Claims, error) {
	options := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if m.issuer != "" {
		options = append(options, jwt.WithIssuer(m.issuer))
	}
	if m.audience != "" {
		options = append(options, jwt.WithAudience(m.audience))
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(value, claims, func(*jwt.Token) (any, error) { return []byte(m.secret), nil }, options...)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if claims.TokenType != string(expected) {
		return nil, fmt.Errorf("invalid token type: expected %s", expected)
	}
	if claims.UserID == "" {
		return nil, fmt.Errorf("token user ID is missing")
	}
	if expected == RefreshTokenType && claims.ID == "" {
		return nil, fmt.Errorf("refresh token ID is missing")
	}
	return claims, nil
}

func ExtractBearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	value := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if value == "" || strings.Contains(value, " ") {
		return ""
	}
	return value
}

func randomTokenID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("create token ID: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}
