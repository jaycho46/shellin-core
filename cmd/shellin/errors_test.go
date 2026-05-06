// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"testing"
)

func TestStartupDisplayForErrorUsesSessionLimitScreen(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("session limit reached: 2 active / 1 max"))
	if !ok {
		t.Fatal("expected session limit screen")
	}
	if display.title != "Session Limit Reached" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.titleFG != hudConnectingFG {
		t.Fatalf("unexpected title color: %q", display.titleFG)
	}
	if display.subtitle != "session limit reached: 2 active / 1 max" {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
	if len(display.steps) != 2 {
		t.Fatalf("unexpected steps: %+v", display.steps)
	}
}

func TestStartupDisplayForErrorUsesSubscriptionRequiredScreen(t *testing.T) {
	display, ok := startupDisplayForError(&subscriptionInactiveError{status: "expired"})
	if !ok {
		t.Fatal("expected subscription required screen")
	}
	if display.title != "Subscription Required" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "Your subscription has expired." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
	if len(display.steps) != 2 || display.steps[0].label != "Purchase or restore an active subscription in shellin app" {
		t.Fatalf("unexpected steps: %+v", display.steps)
	}
}

func TestStartupDisplayForErrorUsesSessionEndedScreen(t *testing.T) {
	display, ok := startupDisplayForError(&sessionTerminatedError{err: errors.New("terminated")})
	if !ok {
		t.Fatal("expected session ended screen")
	}
	if display.title != "Session Ended" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "This terminal session was terminated remotely." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
	if len(display.steps) != 1 || display.steps[0].label != "Run shellin again to start a fresh session" {
		t.Fatalf("unexpected steps: %+v", display.steps)
	}
}

func TestStartupDisplayForErrorUsesLoginRequiredScreen(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("load auth profile failed: open /tmp/auth.json: no such file or directory (run `shellin login <key>`)"))
	if !ok {
		t.Fatal("expected login required screen")
	}
	if display.title != "Login Required" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "No saved login was found on this machine." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
	if len(display.steps) != 2 || display.steps[0].label != "Run shellin login <key>" {
		t.Fatalf("unexpected steps: %+v", display.steps)
	}
}

func TestStartupDisplayForErrorUsesDeviceLoginFailedScreen(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("device login failed: control plane device login status 401: unauthorized"))
	if !ok {
		t.Fatal("expected login failed screen")
	}
	if display.title != "Login Failed" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "Your one-time login key was rejected or has expired." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
	if len(display.steps) != 2 || display.steps[1].label != "Run shellin login <key> again" {
		t.Fatalf("unexpected steps: %+v", display.steps)
	}
}

func TestStartupDisplayForErrorUsesHeartbeatAuthorizationFailedScreen(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("failed to refresh control-plane auth during heartbeat: control plane refresh status 401: unauthorized (fallback user key exchange failed: control plane exchange auth status 401: unauthorized)"))
	if !ok {
		t.Fatal("expected authorization failed screen")
	}
	if display.title != "Authorization Failed" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "Your session authorization expired and automatic recovery failed. Sign in again to restore remote access." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
	if len(display.steps) != 2 || display.steps[0].label != "Run shellin login <key>" {
		t.Fatalf("unexpected steps: %+v", display.steps)
	}
}

func TestStartupDisplayForErrorShowsHeartbeatRefreshFailureDetail(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("failed to refresh control-plane auth during heartbeat: control plane refresh status 401: unauthorized"))
	if !ok {
		t.Fatal("expected authorization failed screen")
	}
	if display.title != "Authorization Failed" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "Your saved session could not be refreshed: control plane refresh status 401: unauthorized. Sign in again to restore remote access." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
}

func TestStartupDisplayForErrorUsesGenericFallbackScreen(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("dial tcp: lookup api.shellin.dev: no such host"))
	if !ok {
		t.Fatal("expected generic startup error screen")
	}
	if display.title != "Unable To Start Session" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "dial tcp: lookup api.shellin.dev: no such host" {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
}

func TestStartupDisplayForErrorShowsStatusRefreshFailureDetail(t *testing.T) {
	display, ok := startupDisplayForError(errors.New("status request failed: status request failed (control plane status 401: unauthorized); refresh failed (control plane refresh status 401: unauthorized)"))
	if !ok {
		t.Fatal("expected authorization failed screen")
	}
	if display.title != "Authorization Failed" {
		t.Fatalf("unexpected title: %q", display.title)
	}
	if display.subtitle != "Your saved session could not be refreshed: control plane refresh status 401: unauthorized. Sign in again to restore remote access." {
		t.Fatalf("unexpected subtitle: %q", display.subtitle)
	}
}
