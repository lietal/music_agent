package tme

import "sync"

type CredentialStore struct {
	mu       sync.RWMutex
	musicid  string
	musickey string
}

func NewCredentialStore() *CredentialStore {
	return &CredentialStore{}
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
}

func (cs *CredentialStore) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.musicid = ""
	cs.musickey = ""
}
