// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"

	"github.com/jaycho46/shellin-core/protocol"
)

func (n *webrtcNegotiator) handleTerminalOpen() {
	n.dcOpen.Store(true)
	n.status.SetP2PState("connected")
	select {
	case n.live.signalOut <- protocol.SignalMessage{Type: protocol.SignalMessageTypeP2PConnected}:
	default:
	}
}

func (n *webrtcNegotiator) handleTerminalClose() {
	n.dcOpen.Store(false)
	n.status.SetP2PState("waiting")
}

func (n *webrtcNegotiator) handleTerminalMessage(msg webrtc.DataChannelMessage) {
	if len(msg.Data) == 0 {
		return
	}
	_, _ = n.ptmx.Write(msg.Data)
}

func (n *webrtcNegotiator) handleControlMessage(msg webrtc.DataChannelMessage) {
	if len(msg.Data) == 0 {
		return
	}
	var ctrl controlMessage
	if err := json.Unmarshal(msg.Data, &ctrl); err != nil {
		return
	}
	if ctrl.Type == "resize" {
		_ = resizePTYFromViewer(n.ptmx, ctrl.Cols, ctrl.Rows)
	}
}
