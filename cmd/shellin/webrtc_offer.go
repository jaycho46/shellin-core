// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

const offerGatheringGracePeriod = 250 * time.Millisecond

func (n *webrtcNegotiator) queuePendingCandidate(candidate webrtc.ICECandidateInit) {
	n.pendingMu.Lock()
	defer n.pendingMu.Unlock()
	n.pendingCandidates = append(n.pendingCandidates, candidate)
}

func (n *webrtcNegotiator) flushPendingCandidates() {
	n.pendingMu.Lock()
	candidates := append([]webrtc.ICECandidateInit(nil), n.pendingCandidates...)
	n.pendingCandidates = n.pendingCandidates[:0]
	n.pendingMu.Unlock()
	for _, candidate := range candidates {
		if err := n.pc.AddICECandidate(candidate); err != nil {
			// Ignore stale candidates from a previous negotiation round.
			continue
		}
	}
}

func (n *webrtcNegotiator) resetPendingOffer() error {
	n.offerMu.Lock()
	defer n.offerMu.Unlock()
	if n.pc.SignalingState() != webrtc.SignalingStateHaveLocalOffer {
		return nil
	}
	if err := n.pc.SetLocalDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeRollback}); err != nil {
		return err
	}
	return nil
}

func (n *webrtcNegotiator) sendOffer(iceRestart bool) error {
	n.offerMu.Lock()
	defer n.offerMu.Unlock()

	if n.pc.SignalingState() == webrtc.SignalingStateHaveLocalOffer {
		if err := n.pc.SetLocalDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeRollback}); err != nil {
			return fmt.Errorf("rollback pending offer: %w", err)
		}
	}

	n.pendingMu.Lock()
	n.pendingCandidates = n.pendingCandidates[:0]
	n.pendingMu.Unlock()

	options := &webrtc.OfferOptions{}
	if iceRestart {
		options.ICERestart = true
	}
	offer, err := n.pc.CreateOffer(options)
	if err != nil {
		return err
	}
	if err := n.pc.SetLocalDescription(offer); err != nil {
		return err
	}

	gatherComplete := webrtc.GatheringCompletePromise(n.pc)
	select {
	case <-gatherComplete:
	case <-time.After(offerGatheringGracePeriod):
	}

	localDesc := n.pc.LocalDescription()
	if localDesc == nil {
		return errors.New("local description is nil")
	}

	select {
	case n.live.signalOut <- protocol.SignalMessage{
		Type: protocol.SignalMessageTypeOffer,
		SDP: &protocol.SDPPayload{
			Type: protocol.SDPTypeOffer,
			SDP:  localDesc.SDP,
		},
	}:
		return nil
	default:
		return errors.New("signaling queue full while sending offer")
	}
}
