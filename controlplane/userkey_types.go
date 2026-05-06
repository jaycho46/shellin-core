// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"sync"
	"time"
)

type UserEntitlement struct {
	Subject               string
	MaxSessions           int
	Active                bool
	ProductID             string
	OriginalTransactionID string
	ExpiresAt             time.Time
	UpdatedAt             time.Time
}

type UserKeyEntry struct {
	UserKey     string
	Entitlement UserEntitlement
}

type UserKeyStore struct {
	mu             sync.RWMutex
	items          map[string]UserEntitlement
	subjectToID    map[string]string
	originalTxToID map[string]string
}

func NewUserKeyStore() *UserKeyStore {
	return &UserKeyStore{
		items:          make(map[string]UserEntitlement),
		subjectToID:    make(map[string]string),
		originalTxToID: make(map[string]string),
	}
}
