// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import "github.com/jaycho46/shellin-core/protocol"

func (b *webrtcAgentBridge) requestConnect() (protocol.AgentConnectResponse, controlPlaneAuthSession, controlPlaneAuthUpdateMode, error) {
	authSession := b.getAuthSession()
	return requestAgentConnectWithAuthSession(
		b.cfg,
		authSession,
		b.sessionID,
		b.currentAgentLabel(),
	)
}

func (b *webrtcAgentBridge) currentAgentLabel() string {
	if b.agentLabel == nil {
		return ""
	}
	return b.agentLabel()
}

func (b *webrtcAgentBridge) getAuthSession() controlPlaneAuthSession {
	b.authMu.RLock()
	defer b.authMu.RUnlock()
	return b.authSession
}

func (b *webrtcAgentBridge) setAuthSession(next controlPlaneAuthSession) {
	b.authMu.Lock()
	defer b.authMu.Unlock()
	b.authSession = next
}
