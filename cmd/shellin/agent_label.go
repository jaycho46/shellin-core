// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

type agentLabelTracker struct {
	value atomic.Value
}

func newAgentLabelTracker(initial string) *agentLabelTracker {
	tracker := &agentLabelTracker{}
	tracker.value.Store(normalizeAgentLabel(initial))
	return tracker
}

func (t *agentLabelTracker) Get() string {
	if t == nil {
		return ""
	}
	current, _ := t.value.Load().(string)
	return current
}

func (t *agentLabelTracker) Set(next string) bool {
	if t == nil {
		return false
	}
	normalized := normalizeAgentLabel(next)
	if normalized == "" || normalized == t.Get() {
		return false
	}
	t.value.Store(normalized)
	return true
}

type oscTerminalTitleParser struct {
	pending []byte
}

func (p *oscTerminalTitleParser) Feed(chunk []byte) []string {
	if len(chunk) == 0 {
		return nil
	}

	const maxPending = 4096
	p.pending = append(p.pending, chunk...)
	titles := make([]string, 0, 1)

	for {
		start := bytes.Index(p.pending, []byte{0x1b, ']'})
		if start < 0 {
			if len(p.pending) > maxPending {
				if p.pending[len(p.pending)-1] == 0x1b {
					p.pending = p.pending[len(p.pending)-1:]
				} else {
					p.pending = nil
				}
			}
			break
		}
		if start > 0 {
			p.pending = p.pending[start:]
		}
		if len(p.pending) < 4 {
			break
		}

		separatorOffset := bytes.IndexByte(p.pending[2:], ';')
		if separatorOffset < 0 {
			if len(p.pending) > maxPending {
				p.pending = p.pending[:2]
			}
			break
		}
		separatorIndex := separatorOffset + 2
		titleStart := separatorIndex + 1
		terminatorIndex, terminatorWidth := oscTitleTerminator(p.pending[titleStart:])
		if terminatorIndex < 0 {
			if len(p.pending) > maxPending {
				p.pending = p.pending[:titleStart]
			}
			break
		}

		param := string(p.pending[2:separatorIndex])
		if param == "0" || param == "2" {
			title := normalizeAgentLabel(string(p.pending[titleStart : titleStart+terminatorIndex]))
			if title != "" {
				titles = append(titles, title)
			}
		}

		p.pending = p.pending[titleStart+terminatorIndex+terminatorWidth:]
	}

	return titles
}

func oscTitleTerminator(data []byte) (int, int) {
	for index := 0; index < len(data); index++ {
		switch data[index] {
		case 0x07:
			return index, 1
		case 0x1b:
			if index+1 < len(data) && data[index+1] == '\\' {
				return index, 2
			}
		}
	}
	return -1, 0
}

func normalizeAgentLabel(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 32 || r == 127:
			return -1
		default:
			return r
		}
	}, raw)

	fields := strings.Fields(strings.TrimSpace(cleaned))
	if len(fields) == 0 {
		return ""
	}

	normalized := strings.Join(fields, " ")
	runes := []rune(normalized)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return normalized
}

func defaultAgentLabel() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "agent"
	}
	return normalizeAgentLabel(fmt.Sprintf("%s:%d", strings.TrimSpace(host), os.Getpid()))
}
