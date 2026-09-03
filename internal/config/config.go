package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Logging  LoggingConfig
	CORS     CORSConfig
}

type AppConfig struct {
	Name        string
	Environment string
	Port        string
	Debug       bool
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int32
	MaxIdleConns    int32
	ConnMaxLifetime time.Duration
}

func (c DatabaseConfig) URL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.Database,
	}
	query := u.Query()
	query.Set("sslmode", c.SSLMode)
	u.RawQuery = query.Encode()
	return u.String()
}

type RedisConfig struct {
	Host                string
	Port                string
	Password            string
	DB                  int
	DocumentQueue       string
	ProcessingQueue     string
	DeadLetterQueue     string
	DocumentMaxAttempts int
}

func (c RedisConfig) Address() string { return net.JoinHostPort(c.Host, c.Port) }

type JWTConfig struct {
	Secret             string
	Issuer             string
	Audience           string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

type StorageConfig struct {
	Driver            string
	LocalRoot         string
	MaxUploadBytes    int64
	AllowedMediaTypes []string
	S3                S3Config
}

type S3Config struct {
	Region          string
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type LoggingConfig struct {
	Level  string
	Format string
}

type CORSConfig struct{ AllowedOrigins []string }

func Load() (*Config, error) {
	_ = godotenv.Load()

	appDebug, err := envBool("APP_DEBUG", true)
	if err != nil {
		return nil, err
	}
	dbPort, err := envInt("DB_PORT", 5432)
	if err != nil {
		return nil, err
	}
	maxOpen, err := envInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return nil, err
	}
	maxIdle, err := envInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return nil, err
	}
	connLifetime, err := envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	if err != nil {
		return nil, err
	}
	redisDB, err := envInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}
	maxAttempts, err := envInt("REDIS_DOCUMENT_MAX_ATTEMPTS", 3)
	if err != nil {
		return nil, err
	}
	accessExpiry, err := envDuration("JWT_ACCESS_TOKEN_EXPIRY", 8*time.Hour)
	if err != nil {
		return nil, err
	}
	refreshExpiry, err := envDuration("JWT_REFRESH_TOKEN_EXPIRY", 720*time.Hour)
	if err != nil {
		return nil, err
	}
	maxUpload, err := envInt64("STORAGE_MAX_UPLOAD_BYTES", 10*1024*1024)
	if err != nil {
		return nil, err
	}
	pathStyle, err := envBool("S3_USE_PATH_STYLE", true)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{Name: env("APP_NAME", "api-example"), Environment: env("APP_ENV", "local"), Port: env("APP_PORT", "8080"), Debug: appDebug},
		Database: DatabaseConfig{
			Host: env("DB_HOST", "localhost"), Port: dbPort, User: env("DB_USER", "postgres"), Password: env("DB_PASSWORD", "postgres"),
			Database: env("DB_NAME", "api_example"), SSLMode: env("DB_SSLMODE", "disable"), MaxOpenConns: int32(maxOpen), MaxIdleConns: int32(maxIdle), ConnMaxLifetime: connLifetime,
		},
		Redis: RedisConfig{
			Host: env("REDIS_HOST", "localhost"), Port: env("REDIS_PORT", "6379"), Password: os.Getenv("REDIS_PASSWORD"), DB: redisDB,
			DocumentQueue: env("REDIS_DOCUMENT_QUEUE", "api_example:documents"), ProcessingQueue: env("REDIS_DOCUMENT_PROCESSING_QUEUE", "api_example:documents:processing"),
			DeadLetterQueue: env("REDIS_DOCUMENT_DEAD_LETTER_QUEUE", "api_example:documents:dead"), DocumentMaxAttempts: maxAttempts,
		},
		JWT: JWTConfig{Secret: os.Getenv("JWT_SECRET"), Issuer: env("JWT_ISSUER", "api-example"), Audience: env("JWT_AUDIENCE", "api-example-client"), AccessTokenExpiry: accessExpiry, RefreshTokenExpiry: refreshExpiry},
		Storage: StorageConfig{
			Driver: env("STORAGE_DRIVER", "local"), LocalRoot: env("STORAGE_LOCAL_ROOT", "./data"), MaxUploadBytes: maxUpload,
			AllowedMediaTypes: envList("STORAGE_ALLOWED_MEDIA_TYPES", []string{"application/pdf", "image/jpeg", "image/png", "text/plain"}),
			S3:                S3Config{Region: env("S3_REGION", "us-east-1"), Bucket: os.Getenv("S3_BUCKET"), Endpoint: os.Getenv("S3_ENDPOINT"), AccessKeyID: os.Getenv("S3_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"), UsePathStyle: pathStyle},
		},
		Logging: LoggingConfig{Level: env("LOG_LEVEL", "info"), Format: env("LOG_FORMAT", "colored_text")},
		CORS:    CORSConfig{AllowedOrigins: envList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"})},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}
	if c.Database.MaxOpenConns <= 0 || c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database pool sizes are invalid")
	}
	if c.Redis.DocumentMaxAttempts <= 0 {
		return fmt.Errorf("REDIS_DOCUMENT_MAX_ATTEMPTS must be positive")
	}
	if c.Storage.MaxUploadBytes <= 0 {
		return fmt.Errorf("STORAGE_MAX_UPLOAD_BYTES must be positive")
	}
	if len(c.Storage.AllowedMediaTypes) == 0 {
		return fmt.Errorf("STORAGE_ALLOWED_MEDIA_TYPES must not be empty")
	}
	switch c.Storage.Driver {
	case "local":
		if strings.TrimSpace(c.Storage.LocalRoot) == "" {
			return fmt.Errorf("STORAGE_LOCAL_ROOT must not be empty")
		}
	case "s3":
		if strings.TrimSpace(c.Storage.S3.Bucket) == "" {
			return fmt.Errorf("S3_BUCKET must not be empty")
		}
	default:
		return fmt.Errorf("STORAGE_DRIVER must be local or s3")
	}
	return nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
}

func envInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func envInt64(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}
