package list

import (
	"context"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestMigrationRegistry(t *testing.T) {
	t.Parallel()
	want := []string{"01_create_auth_schema", "02_seed_roles", "03_create_documents"}
	migrations := GetMigrations()
	got := make([]string, len(migrations))
	seen := make(map[string]struct{}, len(migrations))
	validName := regexp.MustCompile(`^[0-9]{2}_[a-z0-9_]+$`)
	for i, migration := range migrations {
		got[i] = migration.GetName()
		if !validName.MatchString(got[i]) {
			t.Fatalf("invalid migration name %q", got[i])
		}
		if _, exists := seen[got[i]]; exists {
			t.Fatalf("duplicate migration %q", got[i])
		}
		seen[got[i]] = struct{}{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("migration order = %v, want %v", got, want)
	}

	files, err := filepath.Glob("[0-9][0-9]_*.go")
	if err != nil {
		t.Fatal(err)
	}
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		fileNames = append(fileNames, strings.TrimSuffix(filepath.Base(file), ".go"))
	}
	sort.Strings(fileNames)
	if !reflect.DeepEqual(fileNames, want) {
		t.Fatalf("migration files = %v, want %v", fileNames, want)
	}
}

func TestValidateMigrationsRejectsDuplicates(t *testing.T) {
	t.Parallel()
	migration := fakeMigration{name: "01_duplicate"}
	if err := ValidateMigrations([]Migratable{migration, migration}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("ValidateMigrations() error = %v", err)
	}
}

type fakeMigration struct{ name string }

func (m fakeMigration) GetName() string                    { return m.name }
func (m fakeMigration) Up(context.Context, pgx.Tx) error   { return nil }
func (m fakeMigration) Down(context.Context, pgx.Tx) error { return nil }
