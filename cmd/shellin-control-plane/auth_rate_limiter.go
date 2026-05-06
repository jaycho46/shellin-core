// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jaycho46/shellin-core/internal/httporigin"
)

type authFailureLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	records map[string][]time.Time
}

func newAuthFailureLimiter(limit int, window time.Duration) *authFailureLimiter {
	return &authFailureLimiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		records: make(map[string][]time.Time),
	}
}

func (l *authFailureLimiter) Allow(r *http.Request) bool {
	if l == nil {
		return true
	}
	key := l.key(r)
	now := l.currentTime()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneLocked(now)
	return len(l.records[key]) < l.limit
}

func (l *authFailureLimiter) RecordFailure(r *http.Request) {
	if l == nil {
		return
	}
	key := l.key(r)
	now := l.currentTime()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneLocked(now)
	l.records[key] = append(l.records[key], now)
}

func (l *authFailureLimiter) Reset(r *http.Request) {
	if l == nil {
		return
	}
	key := l.key(r)

	l.mu.Lock()
	delete(l.records, key)
	l.mu.Unlock()
}

func (l *authFailureLimiter) key(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	ip := strings.TrimSpace(httporigin.NormalizedRemoteAddrIP(r.RemoteAddr))
	if ip == "" {
		return "unknown"
	}
	return ip
}

func (l *authFailureLimiter) currentTime() time.Time {
	if l.now == nil {
		return time.Now().UTC()
	}
	return l.now().UTC()
}

func (l *authFailureLimiter) pruneLocked(now time.Time) {
	cutoff := now.Add(-l.window)
	for key, attempts := range l.records {
		keep := attempts[:0]
		for _, attempt := range attempts {
			if attempt.After(cutoff) {
				keep = append(keep, attempt)
			}
		}
		if len(keep) == 0 {
			delete(l.records, key)
			continue
		}
		l.records[key] = keep
	}
}
