package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID
	OAuthProvider string
	OAuthID       string
	DisplayName   string
	AvatarURL     string
	CreatedAt     time.Time
	LastLoginAt   time.Time
}

type Conversation struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	Status    string
	CreatedAt time.Time
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	Role           string
	Content        string
	Metadata       map[string]interface{}
	CreatedAt      time.Time
}

type Preference struct {
	UserID     uuid.UUID
	Key        string
	Polarity   string
	Confidence float64
	Evidence   string
	UpdatedAt  time.Time
}

type Provider struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Provider   string
	ConfigJSON map[string]interface{}
	Status     string
}

type UserRepo interface {
	Create(ctx context.Context, user *User) error
	GetByOAuth(ctx context.Context, provider, oauthID string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type ConversationRepo interface {
	Create(ctx context.Context, conv *Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Conversation, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

type MessageRepo interface {
	Create(ctx context.Context, msg *Message) error
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]*Message, error)
}

type PreferenceRepo interface {
	Upsert(ctx context.Context, pref *Preference) error
	GetByKey(ctx context.Context, userID uuid.UUID, key string) (*Preference, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Preference, error)
	Delete(ctx context.Context, userID uuid.UUID, key string) error
}

type ProviderRepo interface {
	Create(ctx context.Context, p *Provider) error
	GetByProvider(ctx context.Context, userID uuid.UUID, provider string) (*Provider, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Provider, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
