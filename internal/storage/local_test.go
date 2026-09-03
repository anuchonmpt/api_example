package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
)

func TestLocalStoragePutOpenDelete(t *testing.T) {
	t.Parallel()
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "documents/1/object", strings.NewReader("hello"), domain.ObjectMetadata{MediaType: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	object, err := store.Open(ctx, "documents/1/object")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = object.Body.Close() }()
	data, err := io.ReadAll(object.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" || object.SizeBytes != 5 {
		t.Fatalf("object = %q size=%d", data, object.SizeBytes)
	}
	if err := store.Delete(ctx, "documents/1/object"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(ctx, "documents/1/object"); !errors.Is(err, apperrors.ErrStorageObjectNotFound) {
		t.Fatalf("Open() error = %v", err)
	}
}

func TestLocalStorageRejectsUnsafeKeys(t *testing.T) {
	t.Parallel()
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "/absolute", "../escape", "documents/../../escape", "documents//file", "."} {
		if err := store.Put(context.Background(), key, strings.NewReader("x"), domain.ObjectMetadata{}); err == nil {
			t.Fatalf("Put(%q) accepted unsafe key", key)
		}
	}
}

func TestLocalStorageRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	store, err := NewLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "link/escaped", strings.NewReader("secret"), domain.ObjectMetadata{}); err == nil {
		t.Fatal("Put followed a symlink outside storage root")
	}
}

func TestLocalStorageHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Put(ctx, "file", strings.NewReader("x"), domain.ObjectMetadata{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put() error = %v", err)
	}
}
