package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func testDBURL() string {
	u := os.Getenv("TEST_DATABASE_URL")
	if u == "" {
		u = "postgres://music_agent:music_agent@127.0.0.1:5432/music_agent_test?sslmode=disable"
	}
	return u
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDBURL())
	if err != nil {
		t.Skipf("skipping test: cannot connect to postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping test: cannot ping postgres: %v", err)
	}

	_, err = pool.Exec(ctx, `DELETE FROM users`)
	if err != nil {
		t.Skipf("skipping test: cannot clean users table: %v", err)
	}

	return pool
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == "mypassword" {
		t.Error("hash should not equal plaintext password")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("secret123")
	if !CheckPassword(hash, "secret123") {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrongpass") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestRegister(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	user, err := Register(ctx, pool, "testuser", "password123", "Test User")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.UserID == "" {
		t.Error("expected non-empty UserID")
	}
	if user.DisplayName != "Test User" {
		t.Errorf("expected display name 'Test User', got %q", user.DisplayName)
	}

	var storedHash string
	err = pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE username = $1`, "testuser").Scan(&storedHash)
	if err != nil {
		t.Fatalf("failed to query stored hash: %v", err)
	}
	if storedHash == "" || storedHash == "password123" {
		t.Error("password should be stored as a bcrypt hash")
	}
}

func TestLogin(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	_, err := Register(ctx, pool, "logintest", "correctpass", "Login Test")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	user, err := Login(ctx, pool, "logintest", "correctpass")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.UserID == "" {
		t.Error("expected non-empty UserID")
	}
	if user.DisplayName != "Login Test" {
		t.Errorf("expected display name 'Login Test', got %q", user.DisplayName)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	_, err := Register(ctx, pool, "wrongpwuser", "correctpass", "Wrong PW")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err = Login(ctx, pool, "wrongpwuser", "wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password")
	}
	if err != bcrypt.ErrMismatchedHashAndPassword {
		t.Errorf("expected bcrypt.ErrMismatchedHashAndPassword, got %v", err)
	}
}

func TestLoginNonexistentUser(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	_, err := Login(ctx, pool, "nonexistent", "password")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	_, err := Register(ctx, pool, "duplicateuser", "password1", "First")
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err = Register(ctx, pool, "duplicateuser", "password2", "Second")
	if err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestRegisterEmptyPassword(t *testing.T) {
	_, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword with empty string failed: %v", err)
	}
}

func TestCheckPasswordInvalidHash(t *testing.T) {
	if CheckPassword("not-a-valid-hash", "password") {
		t.Error("CheckPassword should return false for invalid hash")
	}
}

func TestRegisterLoginMemFallback(t *testing.T) {
	ctx := context.Background()

	user, err := Register(ctx, nil, "memuser", "mempass", "Mem User")
	if err != nil {
		t.Fatal("memRegister failed:", err)
	}
	if user.UserID == "" {
		t.Error("expected non-empty user ID")
	}

	user2, err := Login(ctx, nil, "memuser", "mempass")
	if err != nil {
		t.Fatal("memLogin failed:", err)
	}
	if user2.DisplayName != "Mem User" {
		t.Errorf("got %s", user2.DisplayName)
	}

	_, err = Login(ctx, nil, "memuser", "wrong")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken("test-user", "password", []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	claims, err := ValidateToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "test-user" {
		t.Errorf("got %s", claims.UserID)
	}
}
