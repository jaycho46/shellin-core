// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v4"
)

type webrtcNegotiator struct {
	pc     *webrtc.PeerConnection
	live   *liveWebRTCSession
	ptmx   *os.File
	status *statusPresenter

	dcOpen atomic.Bool

	offerMu sync.Mutex

	pendingMu         sync.Mutex
	pendingCandidates []webrtc.ICECandidateInit
}

func newWebRTCNegotiator(live *liveWebRTCSession, ptmx *os.File, status *statusPresenter) *webrtcNegotiator {
	return &webrtcNegotiator{
		pc:                live.pc,
		live:              live,
		ptmx:              ptmx,
		status:            status,
		pendingCandidates: make([]webrtc.ICECandidateInit, 0, 8),
	}
}

func (n *webrtcNegotiator) bindDataChannels() {
	n.live.terminalDC.OnOpen(n.handleTerminalOpen)
	n.live.terminalDC.OnClose(n.handleTerminalClose)
	n.live.terminalDC.OnMessage(n.handleTerminalMessage)
	n.live.controlDC.OnMessage(n.handleControlMessage)
}

func (n *webrtcNegotiator) bindPeerConnection() {
	n.pc.OnConnectionStateChange(n.handleConnectionStateChange)
	n.pc.OnICECandidate(n.handleICECandidate)
}

func (n *webrtcNegotiator) notifySignalingLost() {
	select {
	case n.live.signalingLost <- struct{}{}:
	default:
	}
}

func (n *webrtcNegotiator) requestReconnect() {
	select {
	case n.live.reconnectNow <- struct{}{}:
	default:
	}
}
