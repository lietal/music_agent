package tme

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CredentialStore struct {
	mu       sync.RWMutex
	musicid  string
	musickey string
	pool     *pgxpool.Pool
	encryptor interface {
		Encrypt(string) (string, error)
		Decrypt(string) (string, error)
	}
}

func NewCredentialStore() *CredentialStore {
	return &CredentialStore{}
}

func NewCredentialStoreWithPool(pool *pgxpool.Pool, encryptor interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}) *CredentialStore {
	cs := &CredentialStore{pool: pool, encryptor: encryptor}
	return cs
}

func (cs *CredentialStore) IsLoggedIn() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.musicid != "" && cs.musickey != ""
}

func (cs *CredentialStore) Get() (musicid, musickey string) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.musicid, cs.musickey
}

func (cs *CredentialStore) Set(musicid, musickey string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.musicid = musicid
	cs.musickey = musickey

	if cs.pool != nil && cs.encryptor != nil {
		cs.saveToDB(musicid, musickey)
	}
}

func (cs *CredentialStore) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.musicid = ""
	cs.musickey = ""

	if cs.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cs.pool.Exec(ctx, "DELETE FROM user_credentials")
	}
}

func (cs *CredentialStore) LoadCredentials() {
	if cs.pool == nil || cs.encryptor == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mid, encKey string
	err := cs.pool.QueryRow(ctx,
		"SELECT musicid, musickey FROM user_credentials ORDER BY updated_at DESC LIMIT 1",
	).Scan(&mid, &encKey)
	if err != nil {
		return
	}

	musickey, err := cs.encryptor.Decrypt(encKey)
	if err != nil {
		return
	}

	cs.mu.Lock()
	cs.musicid = mid
	cs.musickey = musickey
	cs.mu.Unlock()
}

func (cs *CredentialStore) saveToDB(musicid, musickey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	encKey, err := cs.encryptor.Encrypt(musickey)
	if err != nil {
		return
	}

	cs.pool.Exec(ctx,
		"INSERT INTO user_credentials (musicid, musickey) VALUES ($1, $2) ON CONFLICT (musicid) DO UPDATE SET musickey=$2, updated_at=now()",
		musicid, encKey)
}
