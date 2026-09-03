package list

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
)

type Migratable interface {
	GetName() string
	Up(context.Context, pgx.Tx) error
	Down(context.Context, pgx.Tx) error
}

var migrationNamePattern = regexp.MustCompile(`^[0-9]{2}_[a-z0-9_]+$`)

func ValidateMigrations(migrations []Migratable) error {
	seen := make(map[string]struct{}, len(migrations))
	for index, migration := range migrations {
		if migration == nil {
			return fmt.Errorf("migration %d is nil", index+1)
		}
		name := migration.GetName()
		if !migrationNamePattern.MatchString(name) {
			return fmt.Errorf("invalid migration name %q", name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate migration %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}
