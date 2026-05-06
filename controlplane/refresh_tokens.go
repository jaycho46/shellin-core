// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
)

type RefreshTokenStore struct {
	mu     sync.Mutex
	tokens map[string]refreshTokenRecord
}

type refreshTokenRecord struct {
	Subject   string
	ExpiresAt time.Time
}

func NewRefreshTokenStore() *RefreshTokenStore {
	return &RefreshTokenStore{
		tokens: make(map[string]refreshTokenRecord),
	}
}

func (s *RefreshTokenStore) Issue(subject string, ttl time.Duration) (string, time.Time, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", time.Time{}, errors.New("subject is required")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("ttl must be > 0")
	}

	token, err := RandomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(ttl)
	tokenHash := HashToken(token)

	s.mu.Lock()
	s.tokens[tokenHash] = refreshTokenRecord{
		Subject:   subject,
		ExpiresAt: expiresAt,
	}
	s.mu.Unlock()

	return token, expiresAt, nil
}

func (s *RefreshTokenStore) Peek(rawToken string) (string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", errors.New("refresh token is required")
	}
	tokenHash := HashToken(rawToken)

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tokens[tokenHash]
	if !ok {
		return "", errors.New("invalid refresh token")
	}

	if now.After(record.ExpiresAt) {
		return "", errors.New("refresh token expired")
	}
	return record.Subject, nil
}

// Consume validates and deletes the refresh token (rotation-safe one-time use).
func (s *RefreshTokenStore) Consume(rawToken string) (string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", errors.New("refresh token is required")
	}
	tokenHash := HashToken(rawToken)

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tokens[tokenHash]
	if !ok {
		return "", errors.New("invalid refresh token")
	}
	delete(s.tokens, tokenHash)

	if now.After(record.ExpiresAt) {
		return "", errors.New("refresh token expired")
	}
	return record.Subject, nil
}

func (s *RefreshTokenStore) Cleanup() {
	cutoff := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, record := range s.tokens {
		if cutoff.After(record.ExpiresAt) {
			delete(s.tokens, hash)
		}
	}
}

func RandomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
