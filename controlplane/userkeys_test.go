// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import "testing"

func TestParseUserKeySpec(t *testing.T) {
	entries, err := ParseUserKeySpec("uk1:user-a:2, uk2:user-b", 1)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].UserKey != "uk1" || entries[0].Entitlement.Subject != "user-a" || entries[0].Entitlement.MaxSessions != 2 {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if !entries[0].Entitlement.Active {
		t.Fatalf("expected first entry active")
	}
	if entries[1].UserKey != "uk2" || entries[1].Entitlement.Subject != "user-b" || entries[1].Entitlement.MaxSessions != 1 {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
	if !entries[1].Entitlement.Active {
		t.Fatalf("expected second entry active")
	}
}

func TestUserKeyStoreLookup(t *testing.T) {
	store := NewUserKeyStore()
	if err := store.Add("uk_test_1", UserEntitlement{Subject: "user-1", MaxSessions: 3, Active: true}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	got, ok := store.Lookup("uk_test_1")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got.Subject != "user-1" || got.MaxSessions != 3 {
		t.Fatalf("unexpected entitlement: %+v", got)
	}
	if !got.Active {
		t.Fatalf("expected key active")
	}

	if _, ok := store.Lookup("unknown"); ok {
		t.Fatal("expected unknown key to miss")
	}
}

func TestUserKeyStoreLookupBySubject(t *testing.T) {
	store := NewUserKeyStore()
	if err := store.Add("uk_test_1", UserEntitlement{Subject: "user-1", MaxSessions: 2, Active: true}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	got, ok := store.LookupBySubject("user-1")
	if !ok {
		t.Fatal("expected subject to exist")
	}
	if got.MaxSessions != 2 {
		t.Fatalf("unexpected max sessions: %d", got.MaxSessions)
	}
}

func TestUserKeyStoreLookupByOriginalTransactionID(t *testing.T) {
	store := NewUserKeyStore()
	if err := store.Add("uk_test_1", UserEntitlement{
		Subject:               "user-1",
		MaxSessions:           2,
		Active:                true,
		OriginalTransactionID: "otx-1",
	}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	got, ok := store.LookupByOriginalTransactionID("otx-1")
	if !ok {
		t.Fatal("expected original transaction id to exist")
	}
	if got.Subject != "user-1" {
		t.Fatalf("unexpected subject: %s", got.Subject)
	}
}

func TestUserKeyStoreUpsertByOriginalTransactionID(t *testing.T) {
	store := NewUserKeyStore()
	if err := store.Add("uk_test_1", UserEntitlement{
		Subject:               "user-1",
		MaxSessions:           2,
		Active:                true,
		OriginalTransactionID: "otx-1",
	}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	if err := store.UpsertByOriginalTransactionID("otx-1", UserEntitlement{
		Subject:               "user-1",
		MaxSessions:           4,
		Active:                false,
		OriginalTransactionID: "otx-1",
	}); err != nil {
		t.Fatalf("upsert by original tx failed: %v", err)
	}

	got, ok := store.Lookup("uk_test_1")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got.MaxSessions != 4 || got.Active {
		t.Fatalf("unexpected updated entitlement: %+v", got)
	}
}
