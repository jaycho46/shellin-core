// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrReplayDetected         = errors.New("grant token replay detected")
	ErrReplayStoreUnavailable = errors.New("grant replay store unavailable")
)

// ReplayStore atomically records a token ID until expiresAt.
//
// Implementations backed by shared infrastructure must perform this as a
// conditional put: the first caller for a token ID succeeds, and later callers
// fail with ErrReplayDetected until the stored expiry has passed.
type ReplayStore interface {
	Consume(tokenID string, expiresAt time.Time) error
}

type ReplayGuard struct {
	store ReplayStore
}

type MemoryReplayStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
	now  func() time.Time
}

func NewReplayGuard() *ReplayGuard {
	return NewReplayGuardWithStore(NewMemoryReplayStore())
}

func NewReplayGuardWithStore(store ReplayStore) *ReplayGuard {
	return &ReplayGuard{store: store}
}

func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{
		seen: make(map[string]time.Time),
		now:  time.Now,
	}
}

func (g *ReplayGuard) Consume(tokenID string, expiresAt time.Time) error {
	if g == nil || g.store == nil {
		return ErrReplayStoreUnavailable
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return ErrInvalidClaim
	}

	now := time.Now().UTC()
	exp := expiresAt.UTC()
	if exp.Before(now) {
		exp = now
	}
	return g.store.Consume(tokenID, exp)
}

func (s *MemoryReplayStore) Consume(tokenID string, expiresAt time.Time) error {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return ErrInvalidClaim
	}
	if s == nil {
		return ErrReplayStoreUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.now == nil {
		s.now = time.Now
	}
	if s.seen == nil {
		s.seen = make(map[string]time.Time)
	}

	now := s.now().UTC()
	exp := expiresAt.UTC()
	if exp.Before(now) {
		exp = now
	}

	for existingTokenID, existingExpiresAt := range s.seen {
		if !existingExpiresAt.After(now) {
			delete(s.seen, existingTokenID)
		}
	}

	if existingExpiresAt, ok := s.seen[tokenID]; ok && existingExpiresAt.After(now) {
		return ErrReplayDetected
	}

	s.seen[tokenID] = exp
	return nil
}
