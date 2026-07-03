package db

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()

	// Restore DB state: each test may run destructive up/down migrations.
	// Ensure the final state is clean so re-runs and inspections are predictable.
	if err := RunDownMigrations(context.Background(), dbURL()); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: RunDownMigrations: %v\n", err)
	}
	if err := RunMigrations(context.Background(), dbURL()); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: RunMigrations: %v\n", err)
	}

	os.Exit(code)
}
