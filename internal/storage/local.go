package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
)

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("local storage root is empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("evaluate storage root: %w", err)
	}
	return &Local{root: resolved}, nil
}

func (s *Local) Put(ctx context.Context, key string, source io.Reader, _ domain.ObjectMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target, err := s.pathForWrite(key)
	if err != nil {
		return err
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return fmt.Errorf("create object directory: %w", err)
	}
	if err := s.verifyInside(parent); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".upload-*")
	if err != nil {
		return fmt.Errorf("create temporary object: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if _, err := io.Copy(temporary, contextReader{ctx: ctx, reader: source}); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write object: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close object: %w", err)
	}
	if err := os.Rename(temporaryName, target); err != nil {
		return fmt.Errorf("publish object: %w", err)
	}
	return nil
}

func (s *Local) Open(ctx context.Context, key string) (*domain.StoredObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, err := s.pathForExisting(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, apperrors.Wrap(apperrors.ErrStorageObjectNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("open object: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("stat object: %w", err)
	}
	return &domain.StoredObject{Body: file, SizeBytes: info.Size(), MediaType: "application/octet-stream"}, nil
}

func (s *Local) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target, err := s.pathForExisting(key)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, apperrors.ErrStorageObjectNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *Local) pathForWrite(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

func (s *Local) pathForExisting(key string) (string, error) {
	target, err := s.pathForWrite(key)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(target)
	if errors.Is(err, os.ErrNotExist) {
		return "", apperrors.Wrap(apperrors.ErrStorageObjectNotFound, err)
	}
	if err != nil {
		return "", fmt.Errorf("evaluate object path: %w", err)
	}
	if err := s.verifyInside(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func (s *Local) verifyInside(candidate string) error {
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return fmt.Errorf("evaluate object directory: %w", err)
	}
	relative, err := filepath.Rel(s.root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("object path escapes storage root")
	}
	return nil
}

func validateKey(key string) error {
	if key == "" || strings.Contains(key, "\\") || strings.Contains(key, "//") || path.IsAbs(key) || path.Clean(key) != key || key == "." || strings.HasPrefix(key, "../") {
		return apperrors.ErrStorageInvalidKey
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
