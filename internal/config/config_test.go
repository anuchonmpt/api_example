package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Port != "8080" {
		t.Fatalf("App.Port = %q, want 8080", cfg.App.Port)
	}
	if cfg.Database.Port != 5432 {
		t.Fatalf("Database.Port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Redis.Address() != "localhost:6379" {
		t.Fatalf("Redis.Address() = %q", cfg.Redis.Address())
	}
	if cfg.JWT.AccessTokenExpiry != 8*time.Hour {
		t.Fatalf("access expiry = %s", cfg.JWT.AccessTokenExpiry)
	}
	if cfg.JWT.RefreshTokenExpiry != 720*time.Hour {
		t.Fatalf("refresh expiry = %s", cfg.JWT.RefreshTokenExpiry)
	}
	if cfg.Storage.Driver != "local" || cfg.Storage.MaxUploadBytes != 10*1024*1024 {
		t.Fatalf("unexpected storage defaults: %+v", cfg.Storage)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("unexpected CORS defaults: %+v", cfg.CORS)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "missing production JWT secret", env: map[string]string{"APP_ENV": "production", "JWT_SECRET": ""}, want: "JWT_SECRET"},
		{name: "short JWT secret", env: map[string]string{"JWT_SECRET": "short"}, want: "at least 32"},
		{name: "invalid access duration", env: map[string]string{"JWT_ACCESS_TOKEN_EXPIRY": "later"}, want: "JWT_ACCESS_TOKEN_EXPIRY"},
		{name: "invalid storage driver", env: map[string]string{"STORAGE_DRIVER": "ftp"}, want: "STORAGE_DRIVER"},
		{name: "missing S3 bucket", env: map[string]string{"STORAGE_DRIVER": "s3", "S3_BUCKET": ""}, want: "S3_BUCKET"},
		{name: "missing CDN URL", env: map[string]string{"STORAGE_DRIVER": "s3", "S3_BUCKET": "bucket", "CDN_URL": ""}, want: "CDN_URL"},
		{name: "invalid CDN URL", env: map[string]string{"STORAGE_DRIVER": "s3", "S3_BUCKET": "bucket", "CDN_URL": "cdn.example.com"}, want: "CDN_URL"},
		{name: "invalid upload limit", env: map[string]string{"STORAGE_MAX_UPLOAD_BYTES": "0"}, want: "STORAGE_MAX_UPLOAD_BYTES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv("APP_ENV", "test")
			t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestDatabaseURLPercentEscapesCredentials(t *testing.T) {
	cfg := DatabaseConfig{
		Host: "db.internal", Port: 5432, User: "api@example.com", Password: "p@ss:/word",
		Database: "sample", SSLMode: "require",
	}
	got := cfg.URL()
	for _, want := range []string{"api%40example.com", "p%40ss%3A%2Fword", "sslmode=require"} {
		if !strings.Contains(got, want) {
			t.Fatalf("URL() = %q, want %q", got, want)
		}
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_NAME", "APP_ENV", "APP_PORT", "APP_DEBUG",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"REDIS_DOCUMENT_QUEUE", "REDIS_DOCUMENT_PROCESSING_QUEUE", "REDIS_DOCUMENT_DEAD_LETTER_QUEUE", "REDIS_DOCUMENT_MAX_ATTEMPTS",
		"JWT_SECRET", "JWT_ISSUER", "JWT_AUDIENCE", "JWT_ACCESS_TOKEN_EXPIRY", "JWT_REFRESH_TOKEN_EXPIRY",
		"STORAGE_DRIVER", "STORAGE_LOCAL_ROOT", "STORAGE_MAX_UPLOAD_BYTES", "STORAGE_ALLOWED_MEDIA_TYPES",
		"CDN_URL",
		"S3_REGION", "S3_BUCKET", "S3_ENDPOINT", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY", "S3_USE_PATH_STYLE",
		"LOG_LEVEL", "LOG_FORMAT",
		"CORS_ALLOWED_ORIGINS",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
