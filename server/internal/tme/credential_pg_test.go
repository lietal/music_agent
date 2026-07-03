package tme

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testCredPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, "postgres://music_agent:music_agent@127.0.0.1:5432/music_agent_test_tme?sslmode=disable")
	if err != nil {
		t.Skipf("cannot connect: %v", err)
	}
	t.Cleanup(pool.Close)
	pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS user_credentials (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		musicid TEXT NOT NULL DEFAULT '',
		musickey TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	pool.Exec(ctx, "DELETE FROM user_credentials")
	return pool
}

func TestCredentialStore_PGPersist(t *testing.T) {
	pool := testCredPool(t)
	cs := NewCredentialStoreWithPool(pool)

	if cs.IsLoggedIn() {
		t.Error("should not be logged in initially")
	}

	cs.Set("mid_123", "key_abc")
	if !cs.IsLoggedIn() {
		t.Error("should be logged in after Set")
	}

	mid, mk := cs.Get()
	if mid != "mid_123" || mk != "key_abc" {
		t.Errorf("got %s/%s", mid, mk)
	}
}

func TestCredentialStore_IsExpired(t *testing.T) {
	pool := testCredPool(t)
	cs := NewCredentialStoreWithPool(pool)

	cs.Set("mid_exp", "key_exp")
	if !cs.IsLoggedIn() {
		t.Fatal("should be logged in")
	}

	pool.Exec(context.Background(),
		"UPDATE user_credentials SET updated_at=$1 WHERE musicid='mid_exp'",
		time.Now().Add(-25*time.Hour))

	cs2 := NewCredentialStoreWithPool(pool)
	if cs2.IsLoggedIn() {
		t.Error("should be logged out after 25h")
	}
}

func TestCredentialStore_Logout(t *testing.T) {
	pool := testCredPool(t)
	cs := NewCredentialStoreWithPool(pool)

	cs.Set("mid_logout", "key_logout")
	cs.Logout()

	if cs.IsLoggedIn() {
		t.Error("should be logged out after Logout")
	}

	cs2 := NewCredentialStoreWithPool(pool)
	if cs2.IsLoggedIn() {
		t.Error("should not be logged in from new store after Logout")
	}
}

func TestCredentialStore_TouchActivity(t *testing.T) {
	pool := testCredPool(t)
	cs := NewCredentialStoreWithPool(pool)

	cs.Set("mid_touch", "key_touch")
	pool.Exec(context.Background(),
		"UPDATE user_credentials SET updated_at=$1 WHERE musicid='mid_touch'",
		time.Now().Add(-23*time.Hour))

	cs.TouchActivity()

	cs2 := NewCredentialStoreWithPool(pool)
	if !cs2.IsLoggedIn() {
		t.Error("should still be logged in after TouchActivity")
	}
}
