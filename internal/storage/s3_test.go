package storage

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/api-example/internal/domain"
)

func TestS3StorageForwardsStreamingOperations(t *testing.T) {
	t.Parallel()
	fake := &fakeS3{getBody: "stored"}
	store := newS3Store(fake, "bucket")
	ctx := context.Background()
	if err := store.Put(ctx, "documents/1/key", strings.NewReader("upload"), domain.ObjectMetadata{MediaType: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	if fake.putBucket != "bucket" || fake.putKey != "documents/1/key" || fake.putBody != "upload" || fake.contentType != "text/plain" {
		t.Fatalf("put = %+v", fake)
	}
	object, err := store.Open(ctx, "documents/1/key")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(object.Body)
	_ = object.Body.Close()
	if string(data) != "stored" {
		t.Fatalf("body = %q", data)
	}
	if err := store.Delete(ctx, "documents/1/key"); err != nil {
		t.Fatal(err)
	}
	if fake.deleteKey != "documents/1/key" {
		t.Fatalf("delete key = %q", fake.deleteKey)
	}
}

type fakeS3 struct{ putBucket, putKey, putBody, contentType, getBody, deleteKey string }

func (f *fakeS3) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putBucket, f.putKey, f.contentType = *input.Bucket, *input.Key, *input.ContentType
	data, _ := io.ReadAll(input.Body)
	f.putBody = string(data)
	return &s3.PutObjectOutput{}, nil
}
func (f *fakeS3) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	size := int64(len(f.getBody))
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewBufferString(f.getBody)), ContentLength: &size}, nil
}
func (f *fakeS3) DeleteObject(_ context.Context, input *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.deleteKey = *input.Key
	return &s3.DeleteObjectOutput{}, nil
}
