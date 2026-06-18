package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	var userID, dn string
	err = db.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, display_name) VALUES ($1, $2, $3)
		 RETURNING id, display_name`,
		username, hashed, displayName,
	).Scan(&userID, &dn)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		UserID:      userID,
		DisplayName: dn,
	}, nil
}

func Login(ctx context.Context, db *pgxpool.Pool, username, password string) (*UserInfo, error) {
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

	return &UserInfo{
		UserID:      userID,
		DisplayName: displayName,
	}, nil
}
