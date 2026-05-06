// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/jaycho46/shellin-core/controlplane"
	"github.com/jaycho46/shellin-core/internal/httporigin"
	"github.com/jaycho46/shellin-core/protocol"
)

func (s *controlPlaneServer) handleAgentConnect(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in protocol.AgentConnectRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	sessionID := strings.TrimSpace(in.SessionID)
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	_, created, err := s.deps.agentRegistry.UpsertLimited(entitlement.Subject, sessionID, in.AgentLabel, string(protocol.TransportWebRTCTURN), entitlement.MaxSessions)
	if err != nil {
		writeAgentRegistryError(w, err)
		return
	}
	iceServers, err := issueICE(s.cfg, entitlement.Subject)
	if err != nil {
		if created {
			_ = s.deps.agentRegistry.Remove(entitlement.Subject, sessionID)
		}
		http.Error(w, "failed to issue turn credentials", http.StatusBadGateway)
		return
	}
	signalToken, expiresAt, err := issueSignedToken(s.cfg.GrantHMACSecret, s.cfg.GrantIssuer, s.cfg.SignalAudience, s.cfg.SignalTokenTTL, signedTokenOptions{
		Subject:     entitlement.Subject,
		MaxSessions: entitlement.MaxSessions,
		Role:        string(controlplane.SignalRoleAgent),
		SessionID:   sessionID,
	})
	if err != nil {
		if created {
			_ = s.deps.agentRegistry.Remove(entitlement.Subject, sessionID)
		}
		http.Error(w, "failed to issue signal token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, protocol.AgentConnectResponse{
		SessionID:   sessionID,
		SignalURL:   httporigin.MakeSignalURLWithBaseURL(r, s.cfg.ExternalBaseURL),
		SignalToken: signalToken,
		ICEServers:  iceServers,
		ICEPolicy:   protocol.ICEPolicyAll,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	})
}

func (s *controlPlaneServer) handleAgentAttach(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in protocol.AgentAttachRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	sessionID := strings.TrimSpace(in.SessionID)
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if _, ok := s.deps.agentRegistry.Lookup(entitlement.Subject, sessionID, s.cfg.AgentHeartbeatTimeout); !ok {
		http.Error(w, "agent session not found", http.StatusNotFound)
		return
	}
	iceServers, err := issueICE(s.cfg, entitlement.Subject)
	if err != nil {
		http.Error(w, "failed to issue turn credentials", http.StatusBadGateway)
		return
	}
	signalToken, expiresAt, err := issueSignedToken(s.cfg.GrantHMACSecret, s.cfg.GrantIssuer, s.cfg.SignalAudience, s.cfg.SignalTokenTTL, signedTokenOptions{
		Subject:     entitlement.Subject,
		MaxSessions: entitlement.MaxSessions,
		Role:        string(controlplane.SignalRoleViewer),
		SessionID:   sessionID,
	})
	if err != nil {
		http.Error(w, "failed to issue signal token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, protocol.AgentAttachResponse{
		SessionID:   sessionID,
		SignalURL:   httporigin.MakeSignalURLWithBaseURL(r, s.cfg.ExternalBaseURL),
		SignalToken: signalToken,
		ICEServers:  iceServers,
		ICEPolicy:   protocol.ICEPolicyAll,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	})
}
