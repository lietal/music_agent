package auth

import "context"

type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

func (p *MockProvider) Name() string { return "mock" }

func (p *MockProvider) AuthURL(state string) string {
	return "/api/auth/callback/mock?code=dev&state=" + state
}

func (p *MockProvider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	return &UserInfo{
		UserID:      "dev-user-" + code,
		Provider:    "mock",
		ProviderID:  "dev-user-" + code,
		DisplayName: "Dev User",
		AvatarURL:   "",
	}, nil
}
