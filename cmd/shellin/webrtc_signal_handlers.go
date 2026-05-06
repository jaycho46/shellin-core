// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

func (n *webrtcNegotiator) handleSignalMessage(msg protocol.SignalMessage) {
	switch msg.Type {
	case protocol.SignalMessageTypeAnswer:
		n.handleAnswer(msg)
	case protocol.SignalMessageTypeCandidate:
		n.handleCandidate(msg)
	case protocol.SignalMessageTypeViewerReady:
		n.handleViewerReady()
	case protocol.SignalMessageTypeDetach, protocol.SignalMessageTypeViewerReplaced:
		n.handleViewerDetach()
	}
}

func (n *webrtcNegotiator) handleAnswer(msg protocol.SignalMessage) {
	if msg.SDP == nil || msg.SDP.SDP == "" {
		return
	}
	if n.pc.SignalingState() != webrtc.SignalingStateHaveLocalOffer {
		return
	}
	if err := n.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: msg.SDP.SDP}); err != nil {
		return
	}
	n.flushPendingCandidates()
}

func (n *webrtcNegotiator) handleCandidate(msg protocol.SignalMessage) {
	if msg.Candidate == nil || msg.Candidate.Candidate == "" {
		return
	}
	candidate := webrtc.ICECandidateInit{
		Candidate:        msg.Candidate.Candidate,
		SDPMid:           msg.Candidate.SDPMid,
		SDPMLineIndex:    msg.Candidate.SDPMLineIndex,
		UsernameFragment: msg.Candidate.UsernameFragment,
	}
	if n.pc.RemoteDescription() == nil || n.pc.SignalingState() == webrtc.SignalingStateHaveLocalOffer {
		n.queuePendingCandidate(candidate)
		return
	}
	_ = n.pc.AddICECandidate(candidate)
}

func (n *webrtcNegotiator) handleViewerReady() {
	n.status.SetSignalState("viewer_connected")
	n.status.SetP2PState("negotiating")
	if err := n.sendOffer(true); err != nil {
		if retryErr := n.sendOffer(false); retryErr != nil {
			n.requestReconnect()
		}
	}
}

func (n *webrtcNegotiator) handleViewerDetach() {
	n.status.SetSignalState("normal")
	_ = n.resetPendingOffer()
	if !n.dcOpen.Load() {
		n.status.SetP2PState("waiting")
	}
	n.requestReconnect()
}
