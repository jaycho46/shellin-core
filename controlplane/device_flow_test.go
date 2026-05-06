// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestLoginKeyStoreIssueConsumeOneTime(t *testing.T) {
	store := NewLoginKeyStore()
	loginKeyPattern := regexp.MustCompile(`^[a-z]+(?:-[a-z]+){3}$`)

	key, expiresAt, err := store.Issue("user_1", 2, 2*time.Minute)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if key == "" {
		t.Fatal("expected issued login key")
	}
	if !loginKeyPattern.MatchString(key) {
		t.Fatalf("expected four-word login key, got %q", key)
	}
	if expiresAt.IsZero() {
		t.Fatal("expected non-zero expiration")
	}

	claims, err := store.Consume(key)
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if claims.Subject != "user_1" || claims.MaxSessions != 2 {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	if _, err := store.Consume(key); err == nil {
		t.Fatal("expected one-time login key consumption to fail on second use")
	}
}

func TestLoginKeyStoreConsumeNormalizesCase(t *testing.T) {
	store := NewLoginKeyStore()

	key, _, err := store.Issue("user_1", 2, 2*time.Minute)
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	claims, err := store.Consume(strings.ToUpper(key))
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if claims.Subject != "user_1" || claims.MaxSessions != 2 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestAgentRegistryListBySubject(t *testing.T) {
	registry := NewAgentRegistry()

	first, err := registry.Upsert("user_1", "sess_1", "macbook", "webrtc_turn")
	if err != nil {
		t.Fatalf("upsert first failed: %v", err)
	}
	if first.CreatedAt.IsZero() || first.LastSeenAt.IsZero() {
		t.Fatalf("expected timestamps: %+v", first)
	}

	time.Sleep(10 * time.Millisecond)

	if _, err := registry.Upsert("user_1", "sess_2", "server", "webrtc_turn"); err != nil {
		t.Fatalf("upsert second failed: %v", err)
	}
	if _, err := registry.Upsert("user_2", "sess_3", "other", "webrtc_turn"); err != nil {
		t.Fatalf("upsert third failed: %v", err)
	}

	items := registry.ListBySubject("user_1", time.Minute)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].SessionID != "sess_2" {
		t.Fatalf("expected most recent first, got %+v", items)
	}

	if !registry.Touch("user_1", "sess_1", "") {
		t.Fatal("expected touch to succeed")
	}
	if !registry.Remove("user_1", "sess_1") {
		t.Fatal("expected remove to succeed")
	}

	afterRemove := registry.ListBySubject("user_1", time.Minute)
	if len(afterRemove) != 1 || afterRemove[0].SessionID != "sess_2" {
		t.Fatalf("unexpected items after remove: %+v", afterRemove)
	}
}

func TestAgentRegistryAllowsSameSessionIDAcrossSubjects(t *testing.T) {
	registry := NewAgentRegistry()

	if _, err := registry.Upsert("user_1", "sess_shared", "macbook", "webrtc_turn"); err != nil {
		t.Fatalf("upsert first failed: %v", err)
	}
	if _, err := registry.Upsert("user_2", "sess_shared", "iphone", "webrtc_turn"); err != nil {
		t.Fatalf("upsert second failed: %v", err)
	}

	user1Items := registry.ListBySubject("user_1", time.Minute)
	if len(user1Items) != 1 || user1Items[0].AgentLabel != "macbook" {
		t.Fatalf("unexpected user_1 items: %+v", user1Items)
	}

	user2Items := registry.ListBySubject("user_2", time.Minute)
	if len(user2Items) != 1 || user2Items[0].AgentLabel != "iphone" {
		t.Fatalf("unexpected user_2 items: %+v", user2Items)
	}
}

func TestAgentRegistryTouchUpdatesLabel(t *testing.T) {
	registry := NewAgentRegistry()

	if _, err := registry.Upsert("user_1", "sess_1", "macbook", "webrtc_turn"); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if !registry.Touch("user_1", "sess_1", "jay@~/work") {
		t.Fatal("expected touch to succeed")
	}

	items := registry.ListBySubject("user_1", time.Minute)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].AgentLabel != "jay@~/work" {
		t.Fatalf("expected updated label, got %+v", items[0])
	}
}

func TestAgentRegistryUpsertLimitedRejectsNewSessionAtCapacity(t *testing.T) {
	registry := NewAgentRegistry()

	if _, _, err := registry.UpsertLimited("user_1", "sess_1", "macbook", "webrtc_turn", 1); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	if _, _, err := registry.UpsertLimited("user_1", "sess_2", "server", "webrtc_turn", 1); !errors.Is(err, ErrAgentSessionLimitReached) {
		t.Fatalf("expected session limit error, got %v", err)
	}
}

func TestAgentRegistryUpsertLimitedAllowsExistingSessionAtCapacity(t *testing.T) {
	registry := NewAgentRegistry()

	if _, _, err := registry.UpsertLimited("user_1", "sess_1", "macbook", "webrtc_turn", 1); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	item, created, err := registry.UpsertLimited("user_1", "sess_1", "jay@~/work", "webrtc_turn", 1)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if created {
		t.Fatal("expected existing session update")
	}
	if item.AgentLabel != "jay@~/work" {
		t.Fatalf("expected updated label, got %+v", item)
	}
}

func TestAgentRegistryTerminateAllMarksSessionsAsTerminated(t *testing.T) {
	registry := NewAgentRegistry()

	if _, _, err := registry.UpsertLimited("user_1", "sess_1", "macbook", "webrtc_turn", 3); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	if _, _, err := registry.UpsertLimited("user_1", "sess_2", "server", "webrtc_turn", 3); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	terminated := registry.TerminateAll("user_1", time.Minute)
	if len(terminated) != 2 {
		t.Fatalf("expected 2 terminated sessions, got %+v", terminated)
	}
	if !registry.IsTerminated("user_1", "sess_1") || !registry.IsTerminated("user_1", "sess_2") {
		t.Fatalf("expected terminated tombstones to be recorded")
	}
	if _, _, err := registry.UpsertLimited("user_1", "sess_1", "macbook", "webrtc_turn", 3); !errors.Is(err, ErrAgentSessionTerminated) {
		t.Fatalf("expected terminated session rejection, got %v", err)
	}
	if registry.Touch("user_1", "sess_1", "macbook") {
		t.Fatal("expected terminated session touch to fail")
	}
}

func TestAgentRegistryCleanupRemovesExpiredTerminationTombstones(t *testing.T) {
	registry := NewAgentRegistry()
	if _, _, err := registry.UpsertLimited("user_1", "sess_1", "macbook", "webrtc_turn", 1); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	registry.Terminate("user_1", "sess_1")

	registry.mu.Lock()
	registry.terminated["user_1"]["sess_1"] = time.Now().UTC().Add(-terminatedAgentSessionRetention - time.Minute)
	registry.mu.Unlock()

	registry.Cleanup(time.Minute)
	if registry.IsTerminated("user_1", "sess_1") {
		t.Fatal("expected expired tombstone to be removed")
	}
}

func TestAgentRegistryLookupHonorsFreshness(t *testing.T) {
	registry := NewAgentRegistry()

	if _, err := registry.Upsert("user_1", "sess_1", "macbook", "webrtc_turn"); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	item, ok := registry.Lookup("user_1", "sess_1", time.Minute)
	if !ok {
		t.Fatal("expected lookup to succeed")
	}
	if item.Transport != "webrtc_turn" {
		t.Fatalf("unexpected transport: %+v", item)
	}

	registry.mu.Lock()
	stale := registry.items["user_1"]["sess_1"]
	stale.LastSeenAt = time.Now().UTC().Add(-2 * time.Minute)
	registry.items["user_1"]["sess_1"] = stale
	registry.mu.Unlock()

	if _, ok := registry.Lookup("user_1", "sess_1", time.Minute); ok {
		t.Fatal("expected stale lookup to fail")
	}
}
