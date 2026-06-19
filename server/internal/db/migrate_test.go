package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dbURL() string {
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://music_agent:music_agent@127.0.0.1:5432/music_agent?sslmode=disable"
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dbURL())
	if err != nil {
		t.Skipf("skipping integration test: cannot connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func existingTables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, name)
	}
	return tables
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

func TestRunMigrations(t *testing.T) {
	pool := newTestPool(t)

	expectedTables := []string{"users", "conversations", "messages", "user_preferences", "user_providers"}

	before := existingTables(t, pool)
	for _, tbl := range expectedTables {
		if contains(before, tbl) {
			t.Fatalf("table %s already exists before migration", tbl)
		}
	}

	if err := RunMigrations(context.Background(), dbURL()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	after := existingTables(t, pool)
	for _, tbl := range expectedTables {
		if !contains(after, tbl) {
			t.Errorf("table %s missing after migration", tbl)
		}
	}

	if err := RunDownMigrations(context.Background(), dbURL()); err != nil {
		t.Fatalf("RunDownMigrations: %v", err)
	}

	fmt.Println("✅ all migration tests passed")
}
