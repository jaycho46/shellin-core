// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"strings"
	"sync"

	"github.com/jaycho46/shellin-core/protocol"
)

type SignalRole string

const (
	SignalRoleAgent  SignalRole = "agent"
	SignalRoleViewer SignalRole = "viewer"
)

type SignalPeer struct {
	Subject   string
	SessionID string
	Role      SignalRole
	Send      func(protocol.SignalMessage) bool
	Close     func(code int, reason string)
}

type signalSessionKey struct {
	subject   string
	sessionID string
}

type signalSession struct {
	agent  *SignalPeer
	viewer *SignalPeer
}

type SignalHub struct {
	mu       sync.Mutex
	sessions map[signalSessionKey]*signalSession
}

func NewSignalHub() *SignalHub {
	return &SignalHub{
		sessions: make(map[signalSessionKey]*signalSession),
	}
}

func (h *SignalHub) Register(peer *SignalPeer) {
	if peer == nil {
		return
	}

	key := signalSessionKey{
		subject:   strings.TrimSpace(peer.Subject),
		sessionID: strings.TrimSpace(peer.SessionID),
	}
	if key.subject == "" || key.sessionID == "" {
		return
	}

	var oldAgent *SignalPeer
	var oldViewer *SignalPeer
	var notifyAgentDetach *SignalPeer

	h.mu.Lock()
	session := h.sessions[key]
	if session == nil {
		session = &signalSession{}
		h.sessions[key] = session
	}

	switch peer.Role {
	case SignalRoleAgent:
		oldAgent = session.agent
		session.agent = peer
	case SignalRoleViewer:
		oldViewer = session.viewer
		session.viewer = peer
		if oldViewer != nil {
			notifyAgentDetach = session.agent
		}
	}
	h.mu.Unlock()

	if oldAgent != nil && oldAgent != peer && oldAgent.Close != nil {
		oldAgent.Close(4001, "agent replaced")
	}
	if oldViewer != nil && oldViewer != peer {
		if oldViewer.Send != nil {
			oldViewer.Send(protocol.SignalMessage{Type: protocol.SignalMessageTypeViewerReplaced})
		}
		if oldViewer.Close != nil {
			oldViewer.Close(4002, "viewer replaced")
		}
	}
	if notifyAgentDetach != nil && notifyAgentDetach.Send != nil {
		notifyAgentDetach.Send(protocol.SignalMessage{Type: protocol.SignalMessageTypeDetach})
	}
}

func (h *SignalHub) Unregister(peer *SignalPeer) {
	if peer == nil {
		return
	}
	key := signalSessionKey{
		subject:   strings.TrimSpace(peer.Subject),
		sessionID: strings.TrimSpace(peer.SessionID),
	}
	if key.subject == "" || key.sessionID == "" {
		return
	}

	var notifyAgentDetach *SignalPeer
	var closeViewer *SignalPeer

	h.mu.Lock()
	session := h.sessions[key]
	if session == nil {
		h.mu.Unlock()
		return
	}

	switch peer.Role {
	case SignalRoleAgent:
		if session.agent == peer {
			session.agent = nil
			closeViewer = session.viewer
			session.viewer = nil
		}
	case SignalRoleViewer:
		if session.viewer == peer {
			session.viewer = nil
			notifyAgentDetach = session.agent
		}
	}

	if session.agent == nil && session.viewer == nil {
		delete(h.sessions, key)
	}
	h.mu.Unlock()

	if notifyAgentDetach != nil && notifyAgentDetach.Send != nil {
		notifyAgentDetach.Send(protocol.SignalMessage{Type: protocol.SignalMessageTypeDetach})
	}
	if closeViewer != nil && closeViewer.Close != nil {
		closeViewer.Close(4003, "agent disconnected")
	}
}

func (h *SignalHub) Forward(peer *SignalPeer, msg protocol.SignalMessage) bool {
	if peer == nil {
		return false
	}
	key := signalSessionKey{
		subject:   strings.TrimSpace(peer.Subject),
		sessionID: strings.TrimSpace(peer.SessionID),
	}
	if key.subject == "" || key.sessionID == "" {
		return false
	}

	var target *SignalPeer

	h.mu.Lock()
	session := h.sessions[key]
	if session != nil {
		switch peer.Role {
		case SignalRoleAgent:
			target = session.viewer
		case SignalRoleViewer:
			target = session.agent
		}
	}
	h.mu.Unlock()

	if target == nil || target.Send == nil {
		return false
	}
	return target.Send(msg)
}
