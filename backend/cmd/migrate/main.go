// Command migrate applies the database migrations (make migrate).
//
// It connects as the schema owner (MIGRATION_DATABASE_URL), never as the role
// the API uses. If APP_DB_PASSWORD is set, it also sets app_user's password.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := os.Getenv("MIGRATION_DATABASE_URL")
	if url == "" {
		return errors.New("MIGRATION_DATABASE_URL is required")
	}
	if err := db.Migrate(ctx, url, os.Getenv("APP_DB_PASSWORD")); err != nil {
		return err
	}
	fmt.Println("migrations applied")
	return nil
}
