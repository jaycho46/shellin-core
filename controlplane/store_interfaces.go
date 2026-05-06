// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import "time"

// EntitlementStore defines persistence for user subscription entitlements.
type EntitlementStore interface {
	Add(rawKey string, entitlement UserEntitlement) error
	Upsert(rawKey string, entitlement UserEntitlement) error
	Lookup(rawKey string) (UserEntitlement, bool)
	LookupBySubject(subject string) (UserEntitlement, bool)
	LookupByOriginalTransactionID(originalTransactionID string) (UserEntitlement, bool)
	UpsertByOriginalTransactionID(originalTransactionID string, entitlement UserEntitlement) error
	Count() int
}

// RefreshTokenStoreAPI defines persistence for refresh tokens.
type RefreshTokenStoreAPI interface {
	Issue(subject string, ttl time.Duration) (string, time.Time, error)
	Peek(rawToken string) (string, error)
	Consume(rawToken string) (string, error)
	Cleanup()
}
