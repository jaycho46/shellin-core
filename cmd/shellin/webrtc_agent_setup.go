// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"

	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

func setupWebRTCAgent(
	connect protocol.AgentConnectResponse,
	ptmx *os.File,
	status *statusPresenter,
) (*liveWebRTCSession, error) {
	pc, terminalDC, controlDC, err := createAgentPeerConnection(connect)
	if err != nil {
		return nil, err
	}

	status.SetSignalState("normal")
	status.SetP2PState("waiting")

	live := newLiveWebRTCSession(pc, terminalDC, controlDC, status)
	negotiator := newWebRTCNegotiator(live, ptmx, status)
	negotiator.bindDataChannels()
	negotiator.bindPeerConnection()

	live.onSignal = negotiator.handleSignalMessage
	live.onSignalLost = negotiator.notifySignalingLost
	if err := live.ReconnectSignaling(connect); err != nil {
		_ = pc.Close()
		return nil, err
	}
	return live, nil
}

func createAgentPeerConnection(connect protocol.AgentConnectResponse) (*webrtc.PeerConnection, *webrtc.DataChannel, *webrtc.DataChannel, error) {
	pcConfig := webrtc.Configuration{
		ICEServers: toWebRTCICEServers(connect.ICEServers),
	}
	if connect.ICEPolicy == protocol.ICEPolicyRelay || iceServersAreTURNOnly(connect.ICEServers) {
		pcConfig.ICETransportPolicy = webrtc.ICETransportPolicyRelay
	}
	pc, err := webrtc.NewPeerConnection(pcConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	terminalDC, err := pc.CreateDataChannel("terminal", nil)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	controlDC, err := pc.CreateDataChannel("control", nil)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	return pc, terminalDC, controlDC, nil
}

func newLiveWebRTCSession(pc *webrtc.PeerConnection, terminalDC, controlDC *webrtc.DataChannel, status *statusPresenter) *liveWebRTCSession {
	return &liveWebRTCSession{
		pc:            pc,
		terminalDC:    terminalDC,
		controlDC:     controlDC,
		signalOut:     make(chan protocol.SignalMessage, 128),
		signalingLost: make(chan struct{}, 1),
		reconnectNow:  make(chan struct{}, 1),
		status:        status,
	}
}
