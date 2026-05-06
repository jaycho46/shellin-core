// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"strings"
	"time"
)

type agentSessionUpdate struct {
	subject    string
	sessionID  string
	agentLabel string
	transport  string
}

func newAgentSessionUpdate(subject, sessionID, agentLabel, transport string) agentSessionUpdate {
	return agentSessionUpdate{
		subject:    strings.TrimSpace(subject),
		sessionID:  strings.TrimSpace(sessionID),
		agentLabel: strings.TrimSpace(agentLabel),
		transport:  strings.TrimSpace(transport),
	}
}

func (u agentSessionUpdate) validate() error {
	if u.subject == "" {
		return errors.New("subject is required")
	}
	if u.sessionID == "" {
		return errors.New("session id is required")
	}
	if u.transport == "" {
		return errors.New("transport is required")
	}
	return nil
}

func (r *AgentRegistry) Upsert(subject, sessionID, agentLabel, transport string) (AgentStatus, error) {
	update := newAgentSessionUpdate(subject, sessionID, agentLabel, transport)
	if err := update.validate(); err != nil {
		return AgentStatus{}, err
	}

	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isTerminatedLocked(update.subject, update.sessionID) {
		return AgentStatus{}, ErrAgentSessionTerminated
	}

	item, _ := r.upsertLocked(update, now)
	return item, nil
}

func (r *AgentRegistry) UpsertLimited(subject, sessionID, agentLabel, transport string, maxSessions int) (AgentStatus, bool, error) {
	update := newAgentSessionUpdate(subject, sessionID, agentLabel, transport)
	if err := update.validate(); err != nil {
		return AgentStatus{}, false, err
	}
	if maxSessions <= 0 {
		return AgentStatus{}, false, errors.New("max sessions must be > 0")
	}

	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isTerminatedLocked(update.subject, update.sessionID) {
		return AgentStatus{}, false, ErrAgentSessionTerminated
	}

	subjectItems := r.ensureSubjectItemsLocked(update.subject)
	_, exists := subjectItems[update.sessionID]
	if !exists && len(subjectItems) >= maxSessions {
		return AgentStatus{}, false, ErrAgentSessionLimitReached
	}

	item, created := upsertAgentSessionLocked(subjectItems, update, now)
	return item, created, nil
}

func (r *AgentRegistry) upsertLocked(update agentSessionUpdate, now time.Time) (AgentStatus, bool) {
	subjectItems := r.ensureSubjectItemsLocked(update.subject)
	return upsertAgentSessionLocked(subjectItems, update, now)
}

func (r *AgentRegistry) ensureSubjectItemsLocked(subject string) map[string]AgentStatus {
	subjectItems := r.items[subject]
	if subjectItems == nil {
		subjectItems = make(map[string]AgentStatus)
		r.items[subject] = subjectItems
	}
	return subjectItems
}

func upsertAgentSessionLocked(subjectItems map[string]AgentStatus, update agentSessionUpdate, now time.Time) (AgentStatus, bool) {
	current, exists := subjectItems[update.sessionID]
	if !exists {
		current.CreatedAt = now
		current.Subject = update.subject
	}

	current.SessionID = update.sessionID
	current.AgentLabel = update.agentLabel
	current.Transport = update.transport
	current.LastSeenAt = now

	subjectItems[update.sessionID] = current
	return current, !exists
}
