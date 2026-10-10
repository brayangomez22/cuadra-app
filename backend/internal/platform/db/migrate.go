package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/brayangomez22/cuadra-app/backend/migrations"
)

// Migrate applies the pending migrations connecting as the schema owner
// (adminURL). If appPassword is not empty it also sets it as app_user's
// password, so the password never lives in a migration and can be rotated by
// running Migrate again.
func Migrate(ctx context.Context, adminURL, appPassword string) error {
	cfg, err := pgx.ParseConfig(adminURL)
	if err != nil {
		return fmt.Errorf("db: parse migration url: %w", err)
	}
	sqlDB := stdlib.OpenDB(*cfg)
	defer func() { _ = sqlDB.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		return fmt.Errorf("db: load migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("db: apply migrations: %w", err)
	}

	if appPassword == "" {
		return nil
	}
	// ALTER ROLE takes no bind parameters: format %L quotes the literal server side.
	var stmt string
	if err := sqlDB.QueryRowContext(ctx, "SELECT format('ALTER ROLE app_user PASSWORD %L', $1::text)", appPassword).Scan(&stmt); err != nil {
		return fmt.Errorf("db: build app_user password statement: %w", err)
	}
	if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("db: set app_user password: %w", err)
	}
	return nil
}
