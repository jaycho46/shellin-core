// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"sync"
	"time"
)

var ErrAgentSessionLimitReached = errors.New("agent session limit reached")
var ErrAgentSessionTerminated = errors.New("agent session terminated")

const terminatedAgentSessionRetention = 10 * time.Minute

type AgentStatus struct {
	Subject    string
	SessionID  string
	AgentLabel string
	Transport  string
	CreatedAt  time.Time
	LastSeenAt time.Time
}

type AgentRegistry struct {
	mu         sync.RWMutex
	items      map[string]map[string]AgentStatus
	terminated map[string]map[string]time.Time
}

func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		items:      make(map[string]map[string]AgentStatus),
		terminated: make(map[string]map[string]time.Time),
	}
}

func agentStaleCutoff(now time.Time, staleAfter time.Duration) (time.Time, bool) {
	if staleAfter <= 0 {
		return time.Time{}, false
	}
	return now.Add(-staleAfter), true
}
