package storage

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
)

type s3API interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type S3 struct {
	client s3API
	bucket string
}

func newS3Store(client s3API, bucket string) *S3 { return &S3{client: client, bucket: bucket} }

func (s *S3) Put(ctx context.Context, key string, source io.Reader, metadata domain.ObjectMetadata) error {
	if err := validateKey(key); err != nil {
		return err
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: source, ContentType: aws.String(metadata.MediaType)})
	if err != nil {
		return apperrors.Wrap(apperrors.ErrStorageUnavailable, err)
	}
	return nil
}

func (s *S3) Open(ctx context.Context, key string) (*domain.StoredObject, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		var apiError smithy.APIError
		if errors.As(err, &noSuchKey) || (errors.As(err, &apiError) && (apiError.ErrorCode() == "NoSuchKey" || apiError.ErrorCode() == "NotFound")) {
			return nil, apperrors.Wrap(apperrors.ErrStorageObjectNotFound, err)
		}
		return nil, apperrors.Wrap(apperrors.ErrStorageUnavailable, err)
	}
	mediaType := "application/octet-stream"
	if output.ContentType != nil {
		mediaType = *output.ContentType
	}
	var size int64
	if output.ContentLength != nil {
		size = *output.ContentLength
	}
	return &domain.StoredObject{Body: output.Body, SizeBytes: size, MediaType: mediaType}, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return apperrors.Wrap(apperrors.ErrStorageUnavailable, err)
	}
	return nil
}
