// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"
	"time"

	"github.com/jaycho46/shellin-core/protocol"
)

func (s *controlPlaneServer) handleAuthExchange(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in protocol.ExchangeAuthRequest
	if err := decodeJSON(r, &in, 32*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !s.allowCredentialAuth(w, r) {
		return
	}
	entitlement, ok := s.deps.store.Lookup(in.UserKey)
	if !ok || !isEntitlementUsable(entitlement) {
		s.deps.authFailureLimiter.RecordFailure(r)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.deps.authFailureLimiter.Reset(r)
	out, err := issueAuthTokens(s.cfg, s.deps.refreshStore, entitlement.Subject, entitlement.MaxSessions)
	if err != nil {
		http.Error(w, "failed to issue tokens", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *controlPlaneServer) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in protocol.RefreshAuthRequest
	if err := decodeJSON(r, &in, 32*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	subject, err := s.deps.refreshStore.Peek(in.RefreshToken)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	entitlement, ok := s.deps.store.LookupBySubject(subject)
	if !ok || !isEntitlementUsable(entitlement) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	consumedSubject, err := s.deps.refreshStore.Consume(in.RefreshToken)
	if err != nil || consumedSubject != subject {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	out, err := issueAuthTokens(s.cfg, s.deps.refreshStore, subject, entitlement.MaxSessions)
	if err != nil {
		http.Error(w, "failed to issue tokens", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, protocol.RefreshAuthResponse(out))
}

func (s *controlPlaneServer) handleLoginKeys(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in protocol.IssueLoginKeyRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ttl := s.cfg.DeviceLoginKeyTTL
	if in.TTLSeconds > 0 {
		ttl = time.Duration(in.TTLSeconds) * time.Second
	}
	if ttl < 30*time.Second || ttl > 24*time.Hour {
		http.Error(w, "ttl_seconds must be between 30 and 86400", http.StatusBadRequest)
		return
	}
	loginKey, expiresAt, err := s.deps.loginKeyStore.Issue(entitlement.Subject, entitlement.MaxSessions, ttl)
	if err != nil {
		http.Error(w, "failed to issue login key", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, protocol.IssueLoginKeyResponse{
		LoginKey:  loginKey,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

func (s *controlPlaneServer) handleDeviceLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in protocol.DeviceLoginRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !s.allowCredentialAuth(w, r) {
		return
	}
	loginClaims, err := s.deps.loginKeyStore.Consume(in.LoginKey)
	if err != nil {
		s.deps.authFailureLimiter.RecordFailure(r)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	entitlement, ok := s.deps.store.LookupBySubject(loginClaims.Subject)
	if !ok || !isEntitlementUsable(entitlement) {
		s.deps.authFailureLimiter.RecordFailure(r)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.deps.authFailureLimiter.Reset(r)
	maxSessions := entitlement.MaxSessions
	if maxSessions <= 0 {
		maxSessions = loginClaims.MaxSessions
	}
	out, err := issueAuthTokens(s.cfg, s.deps.refreshStore, entitlement.Subject, maxSessions)
	if err != nil {
		http.Error(w, "failed to issue tokens", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *controlPlaneServer) allowCredentialAuth(w http.ResponseWriter, r *http.Request) bool {
	if s.deps.authFailureLimiter.Allow(r) {
		return true
	}
	http.Error(w, "too many auth attempts", http.StatusTooManyRequests)
	return false
}
