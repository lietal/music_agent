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

func FindOrCreateByProvider(ctx context.Context, db *pgxpool.Pool, provider, providerID, displayName, avatarURL string) (*UserInfo, error) {
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Try to find existing user
	var userID, dn, av string
	err := db.QueryRow(ctx,
		`SELECT id, display_name, avatar_url FROM users WHERE oauth_provider = $1 AND oauth_id = $2`,
		provider, providerID,
	).Scan(&userID, &dn, &av)

	if err == nil {
		// Update last login
		db.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID)
		if displayName != "" && dn == "" {
			dn = displayName
		}
		if avatarURL != "" && av == "" {
			av = avatarURL
		}
		return &UserInfo{UserID: userID, Provider: provider, ProviderID: providerID, DisplayName: dn, AvatarURL: av}, nil
	}

	// Create new user
	err = db.QueryRow(ctx,
		`INSERT INTO users (oauth_provider, oauth_id, display_name, avatar_url)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		provider, providerID, displayName, avatarURL,
	).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("create user by provider: %w", err)
	}

	return &UserInfo{UserID: userID, Provider: provider, ProviderID: providerID, DisplayName: displayName, AvatarURL: avatarURL}, nil
}
