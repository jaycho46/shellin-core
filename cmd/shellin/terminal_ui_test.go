// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"testing"
)

func TestHUDStatusDisplayUsesReadyState(t *testing.T) {
	got := hudStatusDisplay("normal", "waiting", 0)
	if got.color != hudReadyFG {
		t.Fatalf("unexpected color: %q", got.color)
	}
	if got.text != "ready for mobile attach" {
		t.Fatalf("unexpected text: %q", got.text)
	}
}

func TestHUDStatusDisplayUsesConnectingState(t *testing.T) {
	got := hudStatusDisplay("connecting", "waiting", 2)
	if got.color != hudConnectingFG {
		t.Fatalf("unexpected color: %q", got.color)
	}
	if got.text != "connecting.." {
		t.Fatalf("unexpected text: %q", got.text)
	}
	if !got.animate {
		t.Fatal("expected connecting state to animate")
	}
}

func TestFormatStartupStepUsesPlainNumberingWithoutStatus(t *testing.T) {
	got := formatStartupStep(1, 2, startupStepState{label: "Close another active terminal session"}, false, 0)
	want := ansiGray + "1." + ansiReset + " " + ansiGray + "Close another active terminal session" + ansiReset
	if got != want {
		t.Fatalf("unexpected step: %q", got)
	}
}

func TestFormatStartupStepSeparatesRunCommand(t *testing.T) {
	got := formatStartupStep(1, 2, startupStepState{label: "Run shellin login <key>"}, false, 0)
	want := ansiGray + "1." + ansiReset + " " + ansiGray + "Run " + ansiReset + ansiBold + hudReadyFG + "`shellin login <key>`" + ansiReset
	if got != want {
		t.Fatalf("unexpected step: %q", got)
	}
}

func TestFormatStartupStepSeparatesRunAgainSuffix(t *testing.T) {
	got := formatStartupStep(2, 2, startupStepState{label: "Run shellin again"}, false, 0)
	want := ansiGray + "2." + ansiReset + " " + ansiGray + "Run " + ansiReset + ansiBold + hudReadyFG + "`shellin`" + ansiReset + " " + ansiGray + "again" + ansiReset
	if got != want {
		t.Fatalf("unexpected step: %q", got)
	}
}

func TestFormatStartupStepSeparatesRunAgainWithTrailingText(t *testing.T) {
	got := formatStartupStep(1, 1, startupStepState{label: "Run shellin again to start a fresh session"}, false, 0)
	want := ansiGray + "1." + ansiReset + " " + ansiGray + "Run " + ansiReset + ansiBold + hudReadyFG + "`shellin`" + ansiReset + " " + ansiGray + "again to start a fresh session" + ansiReset
	if got != want {
		t.Fatalf("unexpected step: %q", got)
	}
}

func TestFormatStartupStepUsesCheckForDoneState(t *testing.T) {
	got := formatStartupStep(1, 3, startupStepState{label: "Authenticate", status: "done"}, false, 0)
	want := "\x1b[32m1/3 [✓] Authenticate" + ansiReset
	if got != want {
		t.Fatalf("unexpected step: %q", got)
	}
}

func TestFormatStartupStepUsesYellowForActiveState(t *testing.T) {
	got := formatStartupStep(1, 3, startupStepState{label: "Authenticate", status: "active"}, false, 0)
	want := hudConnectingFG + "1/3 [...] Authenticate" + ansiReset
	if got != want {
		t.Fatalf("unexpected step: %q", got)
	}
}

func TestStartupStatusDisplayAuthorizingUsesThreeSteps(t *testing.T) {
	got := startupStatusDisplay(startupStageAuthorizing, 0)
	if len(got.steps) != 3 {
		t.Fatalf("unexpected steps: %+v", got.steps)
	}
	if got.steps[0].label != "Authenticate" || got.steps[0].status != "active" {
		t.Fatalf("unexpected first step: %+v", got.steps[0])
	}
	if got.steps[1].label != "Check session limit" || got.steps[1].status != "pending" {
		t.Fatalf("unexpected second step: %+v", got.steps[1])
	}
	if got.steps[2].label != "Create relay session" || got.steps[2].status != "pending" {
		t.Fatalf("unexpected third step: %+v", got.steps[2])
	}
}

func TestStartupStatusDisplayCheckingLimitsStartsAfterAuthenticateCompletes(t *testing.T) {
	got := startupStatusDisplay(startupStageCheckingLimits, 0)
	if len(got.steps) != 3 {
		t.Fatalf("unexpected steps: %+v", got.steps)
	}
	if got.steps[0].label != "Authenticate" || got.steps[0].status != "done" {
		t.Fatalf("unexpected first step: %+v", got.steps[0])
	}
	if got.steps[1].label != "Check session limit" || got.steps[1].status != "active" {
		t.Fatalf("unexpected second step: %+v", got.steps[1])
	}
	if got.steps[2].label != "Create relay session" || got.steps[2].status != "pending" {
		t.Fatalf("unexpected third step: %+v", got.steps[2])
	}
}

func TestStartupStatusDisplayCreatingSessionMarksPriorStepsDone(t *testing.T) {
	got := startupStatusDisplay(startupStageCreatingSession, 0)
	if len(got.steps) != 3 {
		t.Fatalf("unexpected steps: %+v", got.steps)
	}
	if got.steps[0].status != "done" || got.steps[1].status != "done" || got.steps[2].status != "active" {
		t.Fatalf("unexpected step states: %+v", got.steps)
	}
}

func TestBottomHUDStopIsIdempotent(t *testing.T) {
	hud := &bottomHUD{
		enabled:  true,
		out:      &lockedStdout{out: os.Stdout},
		animStop: make(chan struct{}),
		animDone: make(chan struct{}),
	}
	go func() {
		<-hud.animStop
		close(hud.animDone)
	}()

	hud.Stop()
	hud.Stop()
}
