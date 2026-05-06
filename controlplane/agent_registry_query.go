// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"sort"
	"strings"
	"time"
)

func (r *AgentRegistry) ListBySubject(subject string, staleAfter time.Duration) []AgentStatus {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil
	}

	staleCutoff, useStaleCutoff := agentStaleCutoff(time.Now().UTC(), staleAfter)

	r.mu.RLock()
	subjectItems := r.items[subject]
	out := make([]AgentStatus, 0, len(subjectItems))
	for _, item := range subjectItems {
		if useStaleCutoff && item.LastSeenAt.Before(staleCutoff) {
			continue
		}
		out = append(out, item)
	}
	r.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeenAt.After(out[j].LastSeenAt)
	})
	return out
}

func (r *AgentRegistry) Lookup(subject, sessionID string, staleAfter time.Duration) (AgentStatus, bool) {
	subject = strings.TrimSpace(subject)
	sessionID = strings.TrimSpace(sessionID)
	if subject == "" || sessionID == "" {
		return AgentStatus{}, false
	}

	staleCutoff, useStaleCutoff := agentStaleCutoff(time.Now().UTC(), staleAfter)

	r.mu.RLock()
	subjectItems := r.items[subject]
	item, ok := subjectItems[sessionID]
	r.mu.RUnlock()
	if !ok {
		return AgentStatus{}, false
	}
	if useStaleCutoff && item.LastSeenAt.Before(staleCutoff) {
		return AgentStatus{}, false
	}
	return item, true
}

func (r *AgentRegistry) Cleanup(staleAfter time.Duration) {
	now := time.Now().UTC()
	cutoff, useStaleCutoff := agentStaleCutoff(now, staleAfter)
	terminatedCutoff := now.Add(-terminatedAgentSessionRetention)

	r.mu.Lock()
	for subject, subjectItems := range r.items {
		if useStaleCutoff {
			for sessionID, item := range subjectItems {
				if item.LastSeenAt.Before(cutoff) {
					delete(subjectItems, sessionID)
				}
			}
		}
		if len(subjectItems) == 0 {
			delete(r.items, subject)
		}
	}
	for subject, terminatedItems := range r.terminated {
		for sessionID, terminatedAt := range terminatedItems {
			if terminatedAt.Before(terminatedCutoff) {
				delete(terminatedItems, sessionID)
			}
		}
		if len(terminatedItems) == 0 {
			delete(r.terminated, subject)
		}
	}
	r.mu.Unlock()
}
