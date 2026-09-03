package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/example/api-example/internal/config"
	"github.com/example/api-example/internal/domain"
)

func New(ctx context.Context, cfg appconfig.StorageConfig) (domain.ObjectStorage, error) {
	switch cfg.Driver {
	case "local":
		return NewLocal(cfg.LocalRoot)
	case "s3":
		options := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.S3.Region)}
		if cfg.S3.AccessKeyID != "" || cfg.S3.SecretAccessKey != "" {
			options = append(options, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, "")))
		}
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
		if err != nil {
			return nil, fmt.Errorf("load AWS configuration: %w", err)
		}
		client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
			options.UsePathStyle = cfg.S3.UsePathStyle
			if cfg.S3.Endpoint != "" {
				options.BaseEndpoint = aws.String(cfg.S3.Endpoint)
			}
		})
		return newS3Store(client, cfg.S3.Bucket), nil
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.Driver)
	}
}
