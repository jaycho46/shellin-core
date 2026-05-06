// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"

	"github.com/jaycho46/shellin-core/protocol"
)

func (s *controlPlaneServer) handleAgentList(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items := s.deps.agentRegistry.ListBySubject(entitlement.Subject, s.cfg.AgentHeartbeatTimeout)
	out := make([]protocol.AgentStatus, 0, len(items))
	for _, item := range items {
		out = append(out, agentStatusResponse(item))
	}
	writeJSON(w, http.StatusOK, protocol.ListAgentsResponse{Agents: out})
}

func (s *controlPlaneServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	entitlement, err := s.validateEntitlement(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items := s.deps.agentRegistry.ListBySubject(entitlement.Subject, s.cfg.AgentHeartbeatTimeout)
	writeJSON(w, http.StatusOK, protocol.SessionStatusResponse{
		Subject:            entitlement.Subject,
		ActiveSessions:     len(items),
		MaxSessions:        entitlement.MaxSessions,
		SubscriptionStatus: "active",
	})
}
