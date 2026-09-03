package scaffold_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStarterRootContract(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	required := []string{
		"go.mod",
		"Makefile",
		"Dockerfile",
		"docker-compose.yml",
		"sqlc.yaml",
		".env.example",
	}

	for _, name := range required {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := os.Stat(filepath.Join(root, name)); err != nil {
				t.Fatalf("required starter file %s: %v", name, err)
			}
		})
	}

	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	contents := string(data)
	if !strings.Contains(contents, "module github.com/example/api-example") {
		t.Fatal("unexpected module path")
	}
	if !strings.Contains(contents, "go 1.25.0") {
		t.Fatal("unexpected Go version")
	}

	if _, err := os.Stat(filepath.Join(root, ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("starter must not contain nested Git metadata")
	}
}

func TestStarterReuseDocumentationAndIsolation(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", ".."))
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"git init", "github.com/example/api-example", "cp .env.example .env", "STORAGE_DRIVER", "make sqlc-generate", "make docker-up", "make migrate-up", "make test"} {
		if !strings.Contains(string(readme), phrase) {
			t.Fatalf("README is missing %q", phrase)
		}
	}
	parentModule := strings.Join([]string{"github.com", "predicto", "predicto-api"}, "/")
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Name() == ".git" || entry.Name() == ".env" {
			return errors.New("starter contains private or nested repository state: " + path)
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), parentModule) {
			return errors.New("starter imports parent module: " + path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
