package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/music-agent/music-agent/internal/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserInfo struct {
	UserID      string `json:"user_id"`
	Provider    string `json:"provider"`
	ProviderID  string `json:"provider_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type contextKey string

const userContextKey contextKey = "user"

func WithUser(ctx context.Context, user *UserInfo) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) *UserInfo {
	user, _ := ctx.Value(userContextKey).(*UserInfo)
	return user
}

// in-memory store for when PostgreSQL is unavailable
var (
	memStore   = map[string]*memUser{}
	memStoreMu sync.RWMutex
)

type memUser struct {
	id, username, passwordHash, displayName string
}

func memRegister(username, password, displayName string) (*UserInfo, error) {
	memStoreMu.Lock()
	defer memStoreMu.Unlock()
	if _, ok := memStore[username]; ok {
		return nil, fmt.Errorf("username already taken")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	memStore[username] = &memUser{id: id, username: username, passwordHash: hash, displayName: displayName}
	return &UserInfo{UserID: id, DisplayName: displayName}, nil
}

func memLogin(username, password string) (*UserInfo, error) {
	memStoreMu.RLock()
	defer memStoreMu.RUnlock()
	u, ok := memStore[username]
	if !ok || !CheckPassword(u.passwordHash, password) {
		return nil, fmt.Errorf("invalid credentials")
	}
	return &UserInfo{UserID: u.id, DisplayName: u.displayName}, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func Register(ctx context.Context, db *pgxpool.Pool, username, password, displayName string) (*UserInfo, error) {
	if db == nil {
		return memRegister(username, password, displayName)
	}
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	var userID, dn string
	err = db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, display_name) VALUES ($1, $2, $3) RETURNING id, display_name`,
		username, hashed, displayName,
	).Scan(&userID, &dn)
	if err != nil {
		return nil, err
	}
	return &UserInfo{UserID: userID, DisplayName: dn}, nil
}

func Login(ctx context.Context, db *pgxpool.Pool, username, password string) (*UserInfo, error) {
	if db == nil {
		return memLogin(username, password)
	}
	var userID, passwordHash, displayName string
	err := db.QueryRow(ctx,
		`SELECT id, password_hash, display_name FROM users WHERE username = $1`,
		username,
	).Scan(&userID, &passwordHash, &displayName)
	if err != nil {
		return nil, err
	}
	if !CheckPassword(passwordHash, password) {
		return nil, bcrypt.ErrMismatchedHashAndPassword
	}
	return &UserInfo{UserID: userID, DisplayName: displayName}, nil
}
