package auth

import "context"

type UserInfo struct {
	UserID      string `json:"user_id"`
	Provider    string `json:"provider"`
	ProviderID  string `json:"provider_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type OAuthProvider interface {
	Name() string
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (*UserInfo, error)
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
