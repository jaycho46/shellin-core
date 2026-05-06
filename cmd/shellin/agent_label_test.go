// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"reflect"
	"testing"
)

func TestNormalizeAgentLabelStripsControlCharacters(t *testing.T) {
	got := normalizeAgentLabel("  jay@~/home\t\r\n\x1b]bad  ")
	if got != "jay@~/home ]bad" {
		t.Fatalf("unexpected label: %q", got)
	}
}

func TestOSCTerminalTitleParserTracksSplitSequences(t *testing.T) {
	parser := &oscTerminalTitleParser{}
	if got := parser.Feed([]byte("hello\x1b]0;jay@~/work")); len(got) != 0 {
		t.Fatalf("expected no title yet, got %+v", got)
	}
	got := parser.Feed([]byte("\a$ "))
	if !reflect.DeepEqual(got, []string{"jay@~/work"}) {
		t.Fatalf("unexpected titles: %+v", got)
	}
}
