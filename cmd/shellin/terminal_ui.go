// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"sync"
	"time"
)

type lockedStdout struct {
	mu  sync.Mutex
	out *os.File
}

func (w *lockedStdout) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.Write(p)
}

func (w *lockedStdout) WriteString(s string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.WriteString(s)
}

type connectionStatusDisplay interface {
	SetSignalState(state string)
	SetP2PState(state string)
}

type statusPresenter struct {
	mu sync.RWMutex

	sink        connectionStatusDisplay
	signalState string
	p2pState    string
}

func newStatusPresenter(initialSink connectionStatusDisplay) *statusPresenter {
	return &statusPresenter{
		sink:        initialSink,
		signalState: "connecting",
		p2pState:    "waiting",
	}
}

func (p *statusPresenter) SetSink(sink connectionStatusDisplay) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.sink = sink
	signalState := p.signalState
	p2pState := p.p2pState
	p.mu.Unlock()

	if sink != nil {
		sink.SetSignalState(signalState)
		sink.SetP2PState(p2pState)
	}
}

func (p *statusPresenter) SetSignalState(state string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.signalState = state
	sink := p.sink
	p.mu.Unlock()
	if sink != nil {
		sink.SetSignalState(state)
	}
}

func (p *statusPresenter) SetP2PState(state string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.p2pState = state
	sink := p.sink
	p.mu.Unlock()
	if sink != nil {
		sink.SetP2PState(state)
	}
}

const (
	initialBridgeConnectTimeout = 30 * time.Second
	ansiReset                   = "\x1b[0m"
	ansiBold                    = "\x1b[1m"
	ansiGray                    = "\x1b[90m"
	ansiCyan                    = "\x1b[36m"
)

func styleBorder(text string, color bool) string {
	if !color {
		return text
	}
	return ansiGray + text + ansiReset
}
