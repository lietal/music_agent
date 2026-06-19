package db

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, databaseURL string) error {
	return runMigrations(ctx, databaseURL, false)
}

func RunDownMigrations(ctx context.Context, databaseURL string) error {
	return runMigrations(ctx, databaseURL, true)
}

func runMigrations(ctx context.Context, databaseURL string, down bool) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("new migrate: %w", err)
	}
	defer m.Close()

	if down {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrate down: %w", err)
		}
		return nil
	}

	for i := 0; i < 10; i++ {
		if err := m.Up(); err == nil {
			return nil
		} else if err == migrate.ErrNoChange {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
