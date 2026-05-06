// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"sort"
	"strings"
	"time"
)

func (r *AgentRegistry) Touch(subject, sessionID, agentLabel string) bool {
	subject = strings.TrimSpace(subject)
	sessionID = strings.TrimSpace(sessionID)
	agentLabel = strings.TrimSpace(agentLabel)
	if subject == "" || sessionID == "" {
		return false
	}

	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()

	subjectItems := r.items[subject]
	if subjectItems == nil {
		return false
	}
	current, ok := subjectItems[sessionID]
	if !ok {
		return false
	}
	if agentLabel != "" {
		current.AgentLabel = agentLabel
	}
	current.LastSeenAt = now
	subjectItems[sessionID] = current
	return true
}

func (r *AgentRegistry) Terminate(subject, sessionID string) bool {
	subject = strings.TrimSpace(subject)
	sessionID = strings.TrimSpace(sessionID)
	if subject == "" || sessionID == "" {
		return false
	}

	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()

	r.markTerminatedLocked(subject, sessionID, now)

	subjectItems := r.items[subject]
	if subjectItems == nil {
		return false
	}
	if _, ok := subjectItems[sessionID]; !ok {
		return false
	}
	delete(subjectItems, sessionID)
	if len(subjectItems) == 0 {
		delete(r.items, subject)
	}
	return true
}

func (r *AgentRegistry) Remove(subject, sessionID string) bool {
	subject = strings.TrimSpace(subject)
	sessionID = strings.TrimSpace(sessionID)
	if subject == "" || sessionID == "" {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	subjectItems := r.items[subject]
	if subjectItems == nil {
		return false
	}
	if _, ok := subjectItems[sessionID]; !ok {
		return false
	}
	delete(subjectItems, sessionID)
	if len(subjectItems) == 0 {
		delete(r.items, subject)
	}
	return true
}

func (r *AgentRegistry) TerminateAll(subject string, staleAfter time.Duration) []AgentStatus {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil
	}

	now := time.Now().UTC()
	staleCutoff, useStaleCutoff := agentStaleCutoff(now, staleAfter)

	r.mu.Lock()
	defer r.mu.Unlock()

	subjectItems := r.items[subject]
	out := make([]AgentStatus, 0, len(subjectItems))
	for sessionID, item := range subjectItems {
		if useStaleCutoff && item.LastSeenAt.Before(staleCutoff) {
			continue
		}
		out = append(out, item)
		r.markTerminatedLocked(subject, sessionID, now)
		delete(subjectItems, sessionID)
	}
	if len(subjectItems) == 0 {
		delete(r.items, subject)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeenAt.After(out[j].LastSeenAt)
	})
	return out
}
