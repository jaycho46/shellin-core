// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v4"
)

type webrtcAgentBridge struct {
	cfg         config
	ptmx        *os.File
	status      *statusPresenter
	sessionID   string
	profilePath string
	authUpdate  func(controlPlaneAuthSession, controlPlaneAuthUpdateMode)
	agentLabel  func() string

	authMu      sync.RWMutex
	authSession controlPlaneAuthSession

	errMu       sync.RWMutex
	lastErr     error
	liveMu      sync.RWMutex
	liveSession *liveWebRTCSession
	connected   atomic.Bool

	closeCh chan struct{}
	doneCh  chan struct{}
}

func newWebRTCAgentBridge(
	cfg config,
	ptmx *os.File,
	status *statusPresenter,
	sessionID string,
	authSession controlPlaneAuthSession,
	profilePath string,
	agentLabel func() string,
	authUpdate func(controlPlaneAuthSession, controlPlaneAuthUpdateMode),
) *webrtcAgentBridge {
	return &webrtcAgentBridge{
		cfg:         cfg,
		ptmx:        ptmx,
		status:      status,
		sessionID:   strings.TrimSpace(sessionID),
		profilePath: strings.TrimSpace(profilePath),
		authSession: authSession,
		authUpdate:  authUpdate,
		agentLabel:  agentLabel,
		closeCh:     make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

func (b *webrtcAgentBridge) Start() {
	go b.run()
}

func (b *webrtcAgentBridge) Close() {
	select {
	case <-b.closeCh:
	default:
		close(b.closeCh)
	}
	<-b.doneCh
}

func (b *webrtcAgentBridge) Connected() bool {
	return b.connected.Load()
}

func (b *webrtcAgentBridge) LastError() error {
	b.errMu.RLock()
	defer b.errMu.RUnlock()
	return b.lastErr
}

func (b *webrtcAgentBridge) setLastError(err error) {
	b.errMu.Lock()
	defer b.errMu.Unlock()
	b.lastErr = err
}

func (b *webrtcAgentBridge) Enqueue(data []byte) {
	if len(data) == 0 {
		return
	}
	b.liveMu.RLock()
	live := b.liveSession
	b.liveMu.RUnlock()
	if live == nil || live.terminalDC == nil || live.terminalDC.ReadyState() != webrtc.DataChannelStateOpen {
		return
	}
	_ = live.terminalDC.Send(data)
}
