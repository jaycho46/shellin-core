// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"io"
	"os"
)

func startTerminalInputPump(ptmx *os.File) {
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()
}

func pumpPTYOutput(
	ptmx *os.File,
	stdout *lockedStdout,
	hud *bottomHUD,
	bridge *webrtcAgentBridge,
	labelTracker *agentLabelTracker,
	registryHeartbeat *agentRegistryHeartbeat,
) {
	titleParser := &oscTerminalTitleParser{}
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			for _, title := range titleParser.Feed(chunk) {
				if labelTracker.Set(title) {
					registryHeartbeat.NotifyLabelChanged()
				}
			}
			_, _ = stdout.Write(chunk)
			if hud.Enabled() {
				hud.Refresh()
			}
			bridge.Enqueue(chunk)
		}
		if err != nil {
			return
		}
	}
}
