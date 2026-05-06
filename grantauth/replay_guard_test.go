// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"testing"
	"time"
)

func TestReplayGuardConsumesTokenOnceUntilExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	store := NewMemoryReplayStore()
	store.now = func() time.Time { return now }
	guard := NewReplayGuardWithStore(store)

	if err := guard.Consume("jti-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("first consume failed: %v", err)
	}
	if err := guard.Consume("jti-1", now.Add(time.Minute)); err != ErrReplayDetected {
		t.Fatalf("expected replay detection, got %v", err)
	}
}

func TestReplayGuardAllowsTokenReuseAfterExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	store := NewMemoryReplayStore()
	store.now = func() time.Time { return now }
	guard := NewReplayGuardWithStore(store)

	if err := guard.Consume("jti-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("first consume failed: %v", err)
	}

	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	if err := guard.Consume("jti-1", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("expected consume after expiry to succeed, got %v", err)
	}
}

func TestReplayGuardUsesConfiguredStore(t *testing.T) {
	store := &recordingReplayStore{}
	guard := NewReplayGuardWithStore(store)
	expiresAt := time.Now().Add(time.Minute)

	if err := guard.Consume(" jti-1 ", expiresAt); err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if store.tokenID != "jti-1" {
		t.Fatalf("unexpected token id: %q", store.tokenID)
	}
	if store.expiresAt.Before(expiresAt.Add(-time.Second)) {
		t.Fatalf("unexpected expiry: %s", store.expiresAt)
	}
}

func TestReplayGuardRejectsMissingStore(t *testing.T) {
	guard := NewReplayGuardWithStore(nil)

	if err := guard.Consume("jti-1", time.Now().Add(time.Minute)); err != ErrReplayStoreUnavailable {
		t.Fatalf("expected ErrReplayStoreUnavailable, got %v", err)
	}
}

type recordingReplayStore struct {
	tokenID   string
	expiresAt time.Time
}

func (s *recordingReplayStore) Consume(tokenID string, expiresAt time.Time) error {
	s.tokenID = tokenID
	s.expiresAt = expiresAt
	return nil
}
