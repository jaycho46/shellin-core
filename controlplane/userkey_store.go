// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"strings"
)

func (s *UserKeyStore) Add(rawKey string, entitlement UserEntitlement) error {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return errors.New("user key is required")
	}
	entitlement, err := NormalizeEntitlement(entitlement)
	if err != nil {
		return err
	}

	keyHash := HashUserKey(rawKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.upsertLocked(keyHash, entitlement, true)
}

func (s *UserKeyStore) Upsert(rawKey string, entitlement UserEntitlement) error {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return errors.New("user key is required")
	}
	entitlement, err := NormalizeEntitlement(entitlement)
	if err != nil {
		return err
	}

	keyHash := HashUserKey(rawKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.upsertLocked(keyHash, entitlement, false)
}

func (s *UserKeyStore) Lookup(rawKey string) (UserEntitlement, bool) {
	keyHash := HashUserKey(strings.TrimSpace(rawKey))
	s.mu.RLock()
	out, ok := s.items[keyHash]
	s.mu.RUnlock()
	return out, ok
}

func (s *UserKeyStore) LookupBySubject(subject string) (UserEntitlement, bool) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return UserEntitlement{}, false
	}
	s.mu.RLock()
	keyHash, ok := s.subjectToID[subject]
	if !ok {
		s.mu.RUnlock()
		return UserEntitlement{}, false
	}
	out, ok := s.items[keyHash]
	s.mu.RUnlock()
	return out, ok
}

func (s *UserKeyStore) LookupByOriginalTransactionID(originalTransactionID string) (UserEntitlement, bool) {
	originalTransactionID = strings.TrimSpace(originalTransactionID)
	if originalTransactionID == "" {
		return UserEntitlement{}, false
	}
	s.mu.RLock()
	keyHash, ok := s.originalTxToID[originalTransactionID]
	if !ok {
		s.mu.RUnlock()
		return UserEntitlement{}, false
	}
	out, ok := s.items[keyHash]
	s.mu.RUnlock()
	return out, ok
}

func (s *UserKeyStore) UpsertByOriginalTransactionID(originalTransactionID string, entitlement UserEntitlement) error {
	originalTransactionID = strings.TrimSpace(originalTransactionID)
	if originalTransactionID == "" {
		return errors.New("original transaction id is required")
	}
	if strings.TrimSpace(entitlement.OriginalTransactionID) == "" {
		entitlement.OriginalTransactionID = originalTransactionID
	}
	entitlement, err := NormalizeEntitlement(entitlement)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	keyHash, ok := s.originalTxToID[originalTransactionID]
	if !ok {
		return errors.New("original transaction id not found")
	}
	return s.upsertLocked(keyHash, entitlement, false)
}

func (s *UserKeyStore) Count() int {
	s.mu.RLock()
	n := len(s.items)
	s.mu.RUnlock()
	return n
}

func (s *UserKeyStore) upsertLocked(keyHash string, entitlement UserEntitlement, failIfExists bool) error {
	existing, exists := s.items[keyHash]
	if exists && failIfExists {
		return errors.New("duplicate user key")
	}

	if existingKeyHash, ok := s.subjectToID[entitlement.Subject]; ok && existingKeyHash != keyHash {
		return errors.New("duplicate subject")
	}
	if entitlement.OriginalTransactionID != "" {
		if existingKeyHash, ok := s.originalTxToID[entitlement.OriginalTransactionID]; ok && existingKeyHash != keyHash {
			return errors.New("duplicate original transaction id")
		}
	}

	if existing.Subject != "" && existing.Subject != entitlement.Subject {
		delete(s.subjectToID, existing.Subject)
	}
	if existing.OriginalTransactionID != "" && existing.OriginalTransactionID != entitlement.OriginalTransactionID {
		delete(s.originalTxToID, existing.OriginalTransactionID)
	}

	s.items[keyHash] = entitlement
	s.subjectToID[entitlement.Subject] = keyHash
	if entitlement.OriginalTransactionID != "" {
		s.originalTxToID[entitlement.OriginalTransactionID] = keyHash
	}
	return nil
}
