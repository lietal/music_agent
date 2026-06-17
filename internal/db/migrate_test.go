package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dbURL() string {
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		u = "postgres://music_agent:music_agent@127.0.0.1:5432/music_agent?sslmode=disable"
	}
	return u
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL())
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping integration test: cannot ping postgres: %v", err)
	}
	return pool
}

func existingTables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public'`)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		names = append(names, name)
	}
	return names
}

func contains(all []string, want string) bool {
	for _, s := range all {
		if s == want {
			return true
		}
	}
	return false
}

func TestRunMigrations(t *testing.T) {
	pool := newTestPool(t)
	tables := []string{
		"users",
		"conversations",
		"messages",
		"user_preferences",
		"user_providers",
	}

	// Verify tables do not exist before migration
	before := existingTables(t, pool)
	for _, tbl := range tables {
		if contains(before, tbl) {
			t.Fatalf("table %s already exists before migration", tbl)
		}
	}

	// Run up migrations
	if err := RunMigrations(context.Background(), dbURL()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	// Verify all tables exist after migration
	after := existingTables(t, pool)
	for _, tbl := range tables {
		if !contains(after, tbl) {
			t.Errorf("table %s not found after migration", tbl)
		}
	}

	// Verify user_id column exists on each dimension table
	for _, tbl := range []string{"conversations", "messages", "user_preferences", "user_providers"} {
		var exists bool
		err := pool.QueryRow(context.Background(),
			`SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public'
				AND table_name = $1
				AND column_name = 'user_id'
			)`, tbl).Scan(&exists)
		if err != nil {
			t.Fatalf("check user_id on %s: %v", tbl, err)
		}
		if !exists {
			t.Errorf("user_id column missing on table %s", tbl)
		}
	}

	// Run down migrations
	if err := RunDownMigrations(context.Background(), dbURL()); err != nil {
		t.Fatalf("RunDownMigrations: %v", err)
	}

	// Verify tables are gone
	afterDown := existingTables(t, pool)
	for _, tbl := range tables {
		if contains(afterDown, tbl) {
			t.Errorf("table %s still exists after down migration", tbl)
		}
	}

	// Run up again (idempotent)
	if err := RunMigrations(context.Background(), dbURL()); err != nil {
		t.Fatalf("RunMigrations after down: %v", err)
	}

	afterUp := existingTables(t, pool)
	for _, tbl := range tables {
		if !contains(afterUp, tbl) {
			t.Errorf("table %s not found after second migration", tbl)
		}
	}

	// Verify schema_migrations exists
	if !contains(afterUp, "schema_migrations") {
		t.Errorf("schema_migrations table not found after second migration")
	}

	fmt.Println("✅ all migration tests passed")
}
