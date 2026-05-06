// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"strings"
	"sync"
	"time"
)

type startupScreen struct {
	enabled bool
	out     *lockedStdout

	mu        sync.Mutex
	stage     startupStage
	override  *startupDisplayState
	animFrame int
	stop      chan struct{}
	done      chan struct{}
	ready     chan struct{}
	stopOnce  sync.Once
	readyOnce sync.Once
	stopped   bool
}

func newStartupScreen(enabled bool, out *lockedStdout) *startupScreen {
	return &startupScreen{
		enabled: enabled,
		out:     out,
		stage:   startupStageAuthorizing,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
		ready:   make(chan struct{}),
	}
}

func (s *startupScreen) Enabled() bool {
	return s != nil && s.enabled
}

func (s *startupScreen) Start() {
	if !s.Enabled() {
		return
	}
	s.mu.Lock()
	s.renderLocked()
	s.mu.Unlock()
	go s.runAnimation()
}

func (s *startupScreen) Stop() {
	if !s.Enabled() {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stop)
		<-s.done
		s.mu.Lock()
		s.stopped = true
		s.mu.Unlock()
		_, _ = s.out.WriteString("\x1b[2J\x1b[H")
	})
}

func (s *startupScreen) WaitUntilReady() {
	if !s.Enabled() {
		return
	}
	<-s.ready
}

func (s *startupScreen) MarkReady() {
	if !s.Enabled() {
		return
	}
	s.readyOnce.Do(func() {
		close(s.ready)
	})
}

func (s *startupScreen) SetStage(stage startupStage) {
	if !s.Enabled() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.override = nil
	s.stage = stage
	s.animFrame = 0
	s.renderLocked()
}

func (s *startupScreen) SetSignalState(state string) {
	if !s.Enabled() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	if s.override != nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "viewer_connected":
		s.stage = startupStageNegotiating
	case "connecting":
		s.stage = startupStageCreatingSession
	}
	s.renderLocked()
}

func (s *startupScreen) SetP2PState(state string) {
	if !s.Enabled() {
		return
	}
	state = strings.ToLower(strings.TrimSpace(state))
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	if s.override != nil {
		return
	}
	switch state {
	case "negotiating", "checking", "connecting":
		s.stage = startupStageNegotiating
	case "connected":
		s.stage = startupStageStartingShell
	case "waiting":
		s.stage = startupStageWaitingClient
	}
	s.renderLocked()
}

func (s *startupScreen) runAnimation() {
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			display := startupStatusDisplay(s.stage, s.animFrame)
			if display.animate {
				s.animFrame = (s.animFrame + 1) % 4
				s.renderLocked()
			}
			s.mu.Unlock()
		}
	}
}

func (s *startupScreen) ShowDisplay(display startupDisplayState) {
	if !s.Enabled() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.override = &display
	s.animFrame = 0
	s.renderLocked()
}

func (s *startupScreen) renderLocked() {
	if !s.Enabled() {
		return
	}
	display := startupStatusDisplay(s.stage, s.animFrame)
	if s.override != nil {
		display = *s.override
	}
	lines := make([]string, 0, 8)
	title := ansiBold + display.title + ansiReset
	if display.titleFG != "" {
		title = ansiBold + display.titleFG + display.title + ansiReset
	}
	lines = append(lines, title, "")
	if display.subtitle != "" {
		lines = append(lines, display.subtitle, "")
	}
	for idx, step := range display.steps {
		lines = append(lines, formatStartupStep(idx+1, len(display.steps), step, display.animate, s.animFrame))
	}
	_, _ = s.out.WriteString("\x1b[2J\x1b[H" + strings.Join(lines, "\n"))
}
