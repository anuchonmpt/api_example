package list

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type SeedRoles struct{}

func (*SeedRoles) GetName() string { return "02_seed_roles" }

func (*SeedRoles) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `INSERT INTO roles (name) VALUES ('admin'), ('user') ON CONFLICT (name) DO NOTHING`)
	return err
}

func (*SeedRoles) Down(context.Context, pgx.Tx) error {
	// Seed rows may be referenced by users, so rollback intentionally preserves them.
	return nil
}
