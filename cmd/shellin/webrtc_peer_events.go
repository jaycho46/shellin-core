// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"strings"

	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

func (n *webrtcNegotiator) handleConnectionStateChange(state webrtc.PeerConnectionState) {
	n.status.SetP2PState(strings.ToLower(state.String()))
	if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
		n.requestReconnect()
	}
}

func (n *webrtcNegotiator) handleICECandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}
	cand := candidate.ToJSON()
	select {
	case n.live.signalOut <- protocol.SignalMessage{
		Type: protocol.SignalMessageTypeCandidate,
		Candidate: &protocol.ICECandidate{
			Candidate:        cand.Candidate,
			SDPMid:           cand.SDPMid,
			SDPMLineIndex:    cand.SDPMLineIndex,
			UsernameFragment: cand.UsernameFragment,
		},
	}:
	default:
	}
}
