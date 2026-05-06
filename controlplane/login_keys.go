// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"strings"
	"sync"
	"time"
)

type LoginKeyClaims struct {
	Subject     string
	MaxSessions int
}

type loginKeyRecord struct {
	Claims    LoginKeyClaims
	ExpiresAt time.Time
}

type LoginKeyStore struct {
	mu    sync.Mutex
	items map[string]loginKeyRecord
}

func NewLoginKeyStore() *LoginKeyStore {
	return &LoginKeyStore{
		items: make(map[string]loginKeyRecord),
	}
}

func (s *LoginKeyStore) Issue(subject string, maxSessions int, ttl time.Duration) (string, time.Time, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", time.Time{}, errors.New("subject is required")
	}
	if maxSessions <= 0 {
		return "", time.Time{}, errors.New("max sessions must be > 0")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("ttl must be > 0")
	}

	expiresAt := time.Now().UTC().Add(ttl)

	record := loginKeyRecord{
		Claims: LoginKeyClaims{
			Subject:     subject,
			MaxSessions: maxSessions,
		},
		ExpiresAt: expiresAt,
	}
	for attempt := 0; attempt < 8; attempt++ {
		raw, err := randomWordLoginKey()
		if err != nil {
			return "", time.Time{}, err
		}
		keyHash := hashOpaqueToken(raw)

		s.mu.Lock()
		if _, exists := s.items[keyHash]; !exists {
			s.items[keyHash] = record
			s.mu.Unlock()
			return raw, expiresAt, nil
		}
		s.mu.Unlock()
	}

	return "", time.Time{}, errors.New("failed to generate unique login key")
}

// Consume validates and deletes the login key (one-time use).
func (s *LoginKeyStore) Consume(rawKey string) (LoginKeyClaims, error) {
	rawKey = normalizeLoginKey(rawKey)
	if rawKey == "" {
		return LoginKeyClaims{}, errors.New("login key is required")
	}
	keyHash := hashOpaqueToken(rawKey)

	now := time.Now().UTC()
	s.mu.Lock()
	record, ok := s.items[keyHash]
	if ok {
		delete(s.items, keyHash)
	}
	s.mu.Unlock()

	if !ok {
		return LoginKeyClaims{}, errors.New("invalid login key")
	}
	if now.After(record.ExpiresAt) {
		return LoginKeyClaims{}, errors.New("login key expired")
	}
	return record.Claims, nil
}

func (s *LoginKeyStore) Cleanup() {
	cutoff := time.Now().UTC()
	s.mu.Lock()
	for keyHash, record := range s.items {
		if cutoff.After(record.ExpiresAt) {
			delete(s.items, keyHash)
		}
	}
	s.mu.Unlock()
}
