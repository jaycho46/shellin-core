// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"time"
)

func (b *webrtcAgentBridge) run() {
	defer close(b.doneCh)

	backoff := time.Second
	for {
		select {
		case <-b.closeCh:
			b.closeLiveSession()
			return
		default:
		}

		b.status.SetSignalState("connecting")
		connectResp, nextAuth, mode, err := b.requestConnect()
		if err != nil {
			b.connected.Store(false)
			b.setLastError(err)
			b.status.SetSignalState("failed")
			if !sleepWithClose(b.closeCh, backoff) {
				return
			}
			backoff = minDuration(backoff*2, 5*time.Second)
			continue
		}
		b.setAuthSession(nextAuth)
		if mode != controlPlaneAuthUnchanged && b.authUpdate != nil {
			b.authUpdate(nextAuth, mode)
		}

		live, err := setupWebRTCAgent(connectResp, b.ptmx, b.status)
		if err != nil {
			b.connected.Store(false)
			b.setLastError(err)
			b.status.SetSignalState("failed")
			if !sleepWithClose(b.closeCh, backoff) {
				return
			}
			backoff = minDuration(backoff*2, 5*time.Second)
			continue
		}

		b.liveMu.Lock()
		b.liveSession = live
		b.liveMu.Unlock()
		b.connected.Store(true)
		b.setLastError(nil)
		backoff = time.Second

		reconnect := false
	waitForSessionEvents:
		for {
			select {
			case <-b.closeCh:
				b.closeLiveSession()
				return
			case <-live.signalingLost:
				if err := b.recoverLiveSignaling(live); err != nil {
					reconnect = true
					break waitForSessionEvents
				}
			case <-live.reconnectNow:
				reconnect = true
				break waitForSessionEvents
			}
		}

		if reconnect {
			b.connected.Store(false)
			b.closeLiveSession()
			continue
		}
	}
}

func (b *webrtcAgentBridge) closeLiveSession() {
	b.liveMu.Lock()
	live := b.liveSession
	b.liveSession = nil
	b.liveMu.Unlock()
	if live != nil {
		live.Close()
	}
}

func (b *webrtcAgentBridge) recoverLiveSignaling(live *liveWebRTCSession) error {
	if live == nil {
		return errors.New("live session is nil")
	}

	b.status.SetSignalState("connecting")
	connectResp, nextAuth, mode, err := b.requestConnect()
	if err != nil {
		b.setLastError(err)
		b.status.SetSignalState("failed")
		return err
	}
	b.setAuthSession(nextAuth)
	if mode != controlPlaneAuthUnchanged && b.authUpdate != nil {
		b.authUpdate(nextAuth, mode)
	}
	if err := live.ReconnectSignaling(connectResp); err != nil {
		b.setLastError(err)
		b.status.SetSignalState("failed")
		return err
	}
	b.setLastError(nil)
	return nil
}
