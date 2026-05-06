// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"strings"
	"time"
)

func (r *AgentRegistry) IsTerminated(subject, sessionID string) bool {
	subject = strings.TrimSpace(subject)
	sessionID = strings.TrimSpace(sessionID)
	if subject == "" || sessionID == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isTerminatedLocked(subject, sessionID)
}

func (r *AgentRegistry) isTerminatedLocked(subject, sessionID string) bool {
	subjectItems := r.terminated[subject]
	if subjectItems == nil {
		return false
	}
	_, ok := subjectItems[sessionID]
	return ok
}

func (r *AgentRegistry) markTerminatedLocked(subject, sessionID string, now time.Time) {
	subjectItems := r.terminated[subject]
	if subjectItems == nil {
		subjectItems = make(map[string]time.Time)
		r.terminated[subject] = subjectItems
	}
	subjectItems[sessionID] = now
}
