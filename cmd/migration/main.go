package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/example/api-example/internal/config"
	"github.com/example/api-example/migrations/list"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var commands = map[string]struct{}{"up": {}, "down": {}, "status": {}, "validate": {}, "dry-run": {}, "check": {}}

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		log.Printf("migration error: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, argv []string) error {
	envName, command, args, err := parseCLIArgs(argv)
	if err != nil {
		return err
	}
	migrations := list.GetMigrations()
	if err := list.ValidateMigrations(migrations); err != nil {
		return err
	}
	if command == "validate" {
		return nil
	}
	if envName != "" && envName != "local" {
		if err := godotenv.Overload(".env." + envName); err != nil {
			return fmt.Errorf("load environment %q: %w", envName, err)
		}
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.Database.URL())
	if err != nil {
		return fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}

	switch command {
	case "up":
		return migrateUp(ctx, pool, migrations)
	case "down":
		steps, err := parseDownSteps(args)
		if err != nil {
			return err
		}
		return migrateDown(ctx, pool, migrations, steps)
	case "status", "dry-run":
		states, err := migrationStates(ctx, pool, migrations)
		if err != nil {
			return err
		}
		for _, state := range states {
			log.Printf("%s - %s", state.Name, state.Status)
		}
		return nil
	case "check":
		return pool.Ping(ctx)
	default:
		return fmt.Errorf("unsupported command %q", command)
	}
}

func parseCLIArgs(argv []string) (envName, command string, args []string, err error) {
	if len(argv) < 2 {
		return "", "", nil, fmt.Errorf("missing migration command")
	}
	if isCommand(argv[1]) {
		return "", argv[1], optionalArgs(argv[2:]), nil
	}
	if len(argv) >= 3 && isCommand(argv[2]) {
		return argv[1], argv[2], optionalArgs(argv[3:]), nil
	}
	return "", "", nil, fmt.Errorf("unsupported migration command")
}

func optionalArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	return args
}

func isCommand(value string) bool { _, ok := commands[value]; return ok }

func parseDownSteps(args []string) (int, error) {
	if len(args) == 0 {
		return 1, nil
	}
	if len(args) != 1 {
		return 0, fmt.Errorf("down accepts one step count")
	}
	steps, err := strconv.Atoi(args[0])
	if err != nil || steps <= 0 {
		return 0, fmt.Errorf("down steps must be a positive integer")
	}
	return steps, nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS migrations (migration_name VARCHAR(255) PRIMARY KEY, executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)
	return err
}

func migrateUp(ctx context.Context, pool *pgxpool.Pool, migrations []list.Migratable) error {
	for _, migration := range migrations {
		var executed bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM migrations WHERE migration_name = $1)`, migration.GetName()).Scan(&executed); err != nil {
			return err
		}
		if executed {
			continue
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if err := migration.Up(ctx, tx); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", migration.GetName(), err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO migrations (migration_name) VALUES ($1)`, migration.GetName()); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func migrateDown(ctx context.Context, pool *pgxpool.Pool, migrations []list.Migratable, steps int) error {
	rows, err := pool.Query(ctx, `SELECT migration_name FROM migrations ORDER BY executed_at DESC, migration_name DESC LIMIT $1`, steps)
	if err != nil {
		return err
	}
	defer rows.Close()
	names := make([]string, 0, steps)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	byName := make(map[string]list.Migratable, len(migrations))
	for _, migration := range migrations {
		byName[migration.GetName()] = migration
	}
	for _, name := range names {
		migration, ok := byName[name]
		if !ok {
			return fmt.Errorf("recorded migration %q is not registered", name)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if err := migration.Down(ctx, tx); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("rollback %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM migrations WHERE migration_name = $1`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

type migrationState struct{ Name, Status string }

func migrationStates(ctx context.Context, pool *pgxpool.Pool, migrations []list.Migratable) ([]migrationState, error) {
	rows, err := pool.Query(ctx, `SELECT migration_name FROM migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	executed := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		executed[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	states := make([]migrationState, 0, len(migrations))
	for _, migration := range migrations {
		status := "PENDING"
		if _, ok := executed[migration.GetName()]; ok {
			status = "EXECUTED"
			delete(executed, migration.GetName())
		}
		states = append(states, migrationState{Name: migration.GetName(), Status: status})
	}
	if len(executed) > 0 {
		unknown := make([]string, 0, len(executed))
		for name := range executed {
			unknown = append(unknown, name)
		}
		return nil, fmt.Errorf("unregistered migration records: %s", strings.Join(unknown, ", "))
	}
	return states, nil
}
