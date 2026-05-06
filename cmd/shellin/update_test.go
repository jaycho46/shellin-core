// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseUpdatePromptChoice(t *testing.T) {
	t.Parallel()

	if updateNow, ok := parseUpdatePromptChoice("u\n"); !ok || !updateNow {
		t.Fatalf("expected update choice, got ok=%v updateNow=%v", ok, updateNow)
	}
	if updateNow, ok := parseUpdatePromptChoice("c\n"); !ok || updateNow {
		t.Fatalf("expected continue choice, got ok=%v updateNow=%v", ok, updateNow)
	}
	if updateNow, ok := parseUpdatePromptChoice("\n"); !ok || updateNow {
		t.Fatalf("expected enter to continue, got ok=%v updateNow=%v", ok, updateNow)
	}
	if _, ok := parseUpdatePromptChoice("later\n"); ok {
		t.Fatal("expected invalid choice to be rejected")
	}
}

func TestReadUpdatePromptAction(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		action updatePromptAction
	}{
		{name: "enter", input: "\r", action: updatePromptActionChoose},
		{name: "down arrow", input: "\x1b[B", action: updatePromptActionDown},
		{name: "up arrow", input: "\x1b[A", action: updatePromptActionUp},
		{name: "j", input: "j", action: updatePromptActionDown},
		{name: "k", input: "k", action: updatePromptActionUp},
		{name: "escape", input: "\x1b", action: updatePromptActionCancel},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			action, err := readUpdatePromptAction(bufio.NewReader(strings.NewReader(tc.input)))
			if err != nil {
				t.Fatalf("readUpdatePromptAction failed: %v", err)
			}
			if action != tc.action {
				t.Fatalf("unexpected action: got %v want %v", action, tc.action)
			}
		})
	}
}

func TestFormatUpdateAvailableMessage(t *testing.T) {
	t.Parallel()

	output := stripANSI(formatUpdateAvailableMessage("0.0.1", "0.1.1"))
	if !strings.Contains(output, "Update Available") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "Current    0.0.1") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "Latest     0.1.1") {
		t.Fatalf("unexpected output: %q", output)
	}
	if strings.Contains(output, "A new shellin release is available.") {
		t.Fatalf("unexpected output: %q", output)
	}
	if strings.Contains(output, "Action") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestFormatUpdateMenuOptionLine(t *testing.T) {
	t.Parallel()

	selected := stripANSI(formatUpdateMenuOptionLine("Yes, install the update now", true))
	if !strings.Contains(selected, "> Yes, install the update now") {
		t.Fatalf("unexpected selected line: %q", selected)
	}
	if strings.Contains(selected, "Choice") {
		t.Fatalf("unexpected selected line: %q", selected)
	}

	unselected := stripANSI(formatUpdateMenuOptionLine("No, keep using the current version", false))
	if !strings.Contains(unselected, "  No, keep using the current version") {
		t.Fatalf("unexpected unselected line: %q", unselected)
	}
}

func TestFormatUpdateRawTTYText(t *testing.T) {
	t.Parallel()

	got := formatUpdateRawTTYText("a\nb\n")
	if got != "a\r\nb\r\n" {
		t.Fatalf("unexpected raw tty text: %q", got)
	}
}

func TestRunVersionCommand(t *testing.T) {
	t.Parallel()

	originalVersion := defaultCLIVersion
	defaultCLIVersion = "1.2.3"
	t.Cleanup(func() { defaultCLIVersion = originalVersion })

	output, err := captureStdout(t, func() error {
		return runVersionCommand(nil)
	})
	if err != nil {
		t.Fatalf("runVersionCommand failed: %v", err)
	}
	if output != "shellin 1.2.3\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}
