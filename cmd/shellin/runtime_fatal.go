// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import "sync"

type runtimeFatalStopper struct {
	mu       sync.Mutex
	err      error
	stopOnce sync.Once
	stop     func()
}

func newRuntimeFatalStopper(stop func()) *runtimeFatalStopper {
	return &runtimeFatalStopper{stop: stop}
}

func (s *runtimeFatalStopper) Set(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
	s.stopOnce.Do(func() {
		if s.stop != nil {
			s.stop()
		}
	})
}

func (s *runtimeFatalStopper) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}
