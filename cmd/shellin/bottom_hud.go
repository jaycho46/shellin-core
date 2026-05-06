// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

type bottomHUD struct {
	enabled bool
	out     *lockedStdout

	mu          sync.Mutex
	stopOnce    sync.Once
	signalState string
	p2pState    string
	rightText   string
	animFrame   int
	animStop    chan struct{}
	animDone    chan struct{}
}

func newBottomHUD(enabled bool, out *lockedStdout, rightText string) *bottomHUD {
	return &bottomHUD{
		enabled:     enabled,
		out:         out,
		signalState: "connecting",
		p2pState:    "waiting",
		rightText:   strings.TrimSpace(rightText),
		animStop:    make(chan struct{}),
		animDone:    make(chan struct{}),
	}
}

func (h *bottomHUD) Enabled() bool {
	return h != nil && h.enabled
}

func (h *bottomHUD) Start() {
	if !h.Enabled() {
		return
	}
	h.mu.Lock()
	h.prepareScreenLocked()
	h.renderLocked()
	h.mu.Unlock()
	go h.runAnimation()
}

func (h *bottomHUD) SetSignalState(state string) {
	if !h.Enabled() {
		return
	}
	h.mu.Lock()
	h.signalState = state
	if !hudStatusDisplay(h.signalState, h.p2pState, h.animFrame).animate {
		h.animFrame = 0
	}
	h.renderLocked()
	h.mu.Unlock()
}

func (h *bottomHUD) SetP2PState(state string) {
	if !h.Enabled() {
		return
	}
	h.mu.Lock()
	h.p2pState = state
	if !hudStatusDisplay(h.signalState, h.p2pState, h.animFrame).animate {
		h.animFrame = 0
	}
	h.renderLocked()
	h.mu.Unlock()
}

func (h *bottomHUD) Refresh() {
	if !h.Enabled() {
		return
	}
	h.mu.Lock()
	h.renderLocked()
	h.mu.Unlock()
}

func (h *bottomHUD) Stop() {
	if !h.Enabled() {
		return
	}
	h.stopOnce.Do(func() {
		close(h.animStop)
		<-h.animDone
		_, rows, err := term.GetSize(int(h.out.out.Fd()))
		if err != nil || rows < 1 {
			return
		}
		_, _ = h.out.WriteString(fmt.Sprintf("\x1b7\x1b[r\x1b[%d;1H\x1b[2K\x1b8", rows))
	})
}

func (h *bottomHUD) runAnimation() {
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()
	defer close(h.animDone)

	for {
		select {
		case <-h.animStop:
			return
		case <-ticker.C:
			h.mu.Lock()
			display := hudStatusDisplay(h.signalState, h.p2pState, h.animFrame)
			if display.animate {
				h.animFrame = (h.animFrame + 1) % 4
				h.renderLocked()
			}
			h.mu.Unlock()
		}
	}
}

func (h *bottomHUD) prepareScreenLocked() {
	if !h.Enabled() {
		return
	}
	_, rows, err := term.GetSize(int(h.out.out.Fd()))
	if err != nil || rows < 2 {
		return
	}
	_, _ = h.out.WriteString(fmt.Sprintf("\x1b[2J\x1b[H\x1b[1;%dr\x1b[1;1H", rows-1))
}

func (h *bottomHUD) renderLocked() {
	if !h.Enabled() {
		return
	}
	cols, rows, err := term.GetSize(int(h.out.out.Fd()))
	if err != nil || cols < 1 || rows < 2 {
		return
	}

	display := hudStatusDisplay(h.signalState, h.p2pState, h.animFrame)
	layout := formatHUDLine(cols, display.text, h.rightText)

	var buf strings.Builder
	fmt.Fprintf(&buf, "\x1b7\x1b[1;%dr\x1b[%d;1H\x1b[2K%s%s%s", rows-1, rows, hudBaseBG, hudBaseFG, layout.line)
	if layout.leftBadge != "" {
		fmt.Fprintf(&buf, "\x1b[%d;1H%s%s%s", rows, hudLabelBG, hudLabelFG, layout.leftBadge)
	}
	if layout.statusText != "" {
		fmt.Fprintf(&buf, "\x1b[%d;%dH%s%s%s", rows, layout.statusStart+1, hudBaseBG, display.color, layout.statusText)
	}
	if layout.rightBadge != "" {
		fmt.Fprintf(&buf, "\x1b[%d;%dH%s%s%s", rows, layout.rightStart+1, hudSessionBG, hudSessionFG, layout.rightBadge)
	}
	buf.WriteString(ansiReset)
	buf.WriteString("\x1b8")
	_, _ = h.out.WriteString(buf.String())
}
