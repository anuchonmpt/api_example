package list

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type CreateAuthSchema struct{}

func (*CreateAuthSchema) GetName() string { return "01_create_auth_schema" }

func (*CreateAuthSchema) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(32) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    role_id BIGINT NOT NULL REFERENCES roles(id),
    email VARCHAR(320) NOT NULL,
    password_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS users_email_active_uidx ON users (LOWER(email)) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS refresh_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    token_id VARCHAR(64) NOT NULL UNIQUE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by_token_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS refresh_sessions_user_id_idx ON refresh_sessions (user_id);
CREATE INDEX IF NOT EXISTS refresh_sessions_expires_at_idx ON refresh_sessions (expires_at) WHERE revoked_at IS NULL;
`)
	return err
}

func (*CreateAuthSchema) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
DROP TABLE IF EXISTS refresh_sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;
`)
	return err
}
