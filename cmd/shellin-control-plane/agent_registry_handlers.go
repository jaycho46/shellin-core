// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"

	"github.com/jaycho46/shellin-core/protocol"
)

func (s *controlPlaneServer) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in protocol.AgentRegisterRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if in.Transport != protocol.TransportWebRTCTURN {
		http.Error(w, "invalid transport", http.StatusBadRequest)
		return
	}
	item, _, err := s.deps.agentRegistry.UpsertLimited(entitlement.Subject, in.SessionID, in.AgentLabel, string(in.Transport), entitlement.MaxSessions)
	if err != nil {
		writeAgentRegistryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentStatusResponse(item))
}

func (s *controlPlaneServer) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in protocol.AgentHeartbeatRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !s.deps.agentRegistry.Touch(entitlement.Subject, in.SessionID, in.AgentLabel) {
		http.Error(w, "agent session not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *controlPlaneServer) handleAgentUnregister(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in protocol.AgentUnregisterRequest
	if err := decodeJSON(r, &in, 16*1024); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	_ = s.deps.agentRegistry.Remove(entitlement.Subject, in.SessionID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *controlPlaneServer) handleAgentTerminateAll(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	terminated := s.deps.agentRegistry.TerminateAll(entitlement.Subject, s.cfg.AgentHeartbeatTimeout)
	writeJSON(w, http.StatusOK, protocol.AgentTerminateAllResponse{TerminatedSessions: len(terminated)})
}
