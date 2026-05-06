// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaycho46/shellin-core/protocol"
)

func TestRunLoginCommandAcceptsKeyBeforeFlags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(protocol.ExchangeAuthResponse{
			AccessToken:  "device-access",
			RefreshToken: "device-refresh",
		})
	}))
	t.Cleanup(srv.Close)

	originalDefault := defaultControlPlaneURL
	defaultControlPlaneURL = srv.URL
	t.Cleanup(func() { defaultControlPlaneURL = originalDefault })

	profilePath := filepath.Join(t.TempDir(), "auth.json")
	out, err := captureStdout(t, func() error {
		return runLoginCommand([]string{"apple-logo-alert-bank", "-auth-profile", profilePath})
	})
	if err != nil {
		t.Fatalf("runLoginCommand failed: %v", err)
	}
	if out != formatLoginSuccessMessage() {
		t.Fatalf("unexpected stdout: %q", out)
	}
	profile, err := loadAuthProfile(profilePath)
	if err != nil {
		t.Fatalf("loadAuthProfile failed: %v", err)
	}
	if profile.AccessToken != "device-access" || profile.RefreshToken != "device-refresh" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestFormatLoginSuccessMessage(t *testing.T) {
	out := formatLoginSuccessMessage()
	if !strings.Contains(out, "Login Complete ✨") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Login Complete ✨"+ansiReset+"\n\n") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "You're ready to use shellin.") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Run `shellin` to start a new terminal session.") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if strings.Contains(out, "Profile:") || strings.Contains(out, "Access token expires at:") {
		t.Fatalf("unexpected stdout: %q", out)
	}
}

func TestFormatCleanupRequestedMessage(t *testing.T) {
	out := formatCleanupRequestedMessage(1)
	if !strings.Contains(out, "Cleanup Requested") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Cleanup Requested"+ansiReset+"\n\n") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Cleanup requested for 1 active session(s).") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Active sessions usually shut down within about 30 seconds.") {
		t.Fatalf("unexpected stdout: %q", out)
	}
}

func TestFormatCleanupCompleteMessage(t *testing.T) {
	out := formatCleanupCompleteMessage()
	if !strings.Contains(out, "Cleanup Complete") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Cleanup Complete"+ansiReset+"\n\n") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "No active sessions to clean up.") {
		t.Fatalf("unexpected stdout: %q", out)
	}
	if !strings.Contains(out, "Everything is already clear.") {
		t.Fatalf("unexpected stdout: %q", out)
	}
}

func TestRunStatusCommandUsesStoredProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer access-token-profile" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		_ = json.NewEncoder(w).Encode(protocol.SessionStatusResponse{
			ActiveSessions:        1,
			MaxSessions:           2,
			ProductID:             "shellin.pro.monthly",
			OriginalTransactionID: "otx-123",
			SubscriptionStatus:    "active",
			ExpiresAt:             "2030-01-02T03:04:05Z",
		})
	}))
	t.Cleanup(srv.Close)

	originalDefault := defaultControlPlaneURL
	defaultControlPlaneURL = srv.URL
	t.Cleanup(func() { defaultControlPlaneURL = originalDefault })

	profilePath := filepath.Join(t.TempDir(), "auth.json")
	if err := saveAuthProfile(profilePath, authProfile{
		ControlPlaneURL: srv.URL,
		AccessToken:     "access-token-profile",
		RefreshToken:    "refresh-token-profile",
	}); err != nil {
		t.Fatalf("saveAuthProfile failed: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runStatusCommand([]string{"-auth-profile", profilePath})
	})
	if err != nil {
		t.Fatalf("runStatusCommand failed: %v", err)
	}
	plainOutput := stripANSI(output)
	if !strings.HasPrefix(output, "\n") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(plainOutput, "Status") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "Status"+ansiReset+"\n\n") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(plainOutput, "Plan       Shellin Pro") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(plainOutput, "Sessions   1 / 2") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(plainOutput, "Ends       Jan 2 2030") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestFormatStatusCommandOutputUsesUnknownPlanWithoutProductID(t *testing.T) {
	out := formatStatusCommandOutput(protocol.SessionStatusResponse{
		ActiveSessions: 2,
		MaxSessions:    3,
	}, time.UTC)

	plainOut := stripANSI(out)
	if !strings.Contains(out, "Status"+ansiReset+"\n\n") || !strings.Contains(plainOut, "Plan       unknown") || !strings.Contains(plainOut, "Sessions   2 / 3") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFormatStatusExpiryUsesRequestedLocation(t *testing.T) {
	loc := time.FixedZone("KST", 9*60*60)
	got := formatStatusExpiry("2030-01-02T03:04:05Z", loc)
	if got != "Jan 2 2030" {
		t.Fatalf("unexpected formatted expiry: %q", got)
	}
}

func TestRunCleanupCommandUsesStoredProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/agents/terminate-all" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer access-token-profile" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		_ = json.NewEncoder(w).Encode(protocol.AgentTerminateAllResponse{
			TerminatedSessions: 2,
		})
	}))
	t.Cleanup(srv.Close)

	originalDefault := defaultControlPlaneURL
	defaultControlPlaneURL = srv.URL
	t.Cleanup(func() { defaultControlPlaneURL = originalDefault })

	profilePath := filepath.Join(t.TempDir(), "auth.json")
	if err := saveAuthProfile(profilePath, authProfile{
		ControlPlaneURL: srv.URL,
		AccessToken:     "access-token-profile",
		RefreshToken:    "refresh-token-profile",
	}); err != nil {
		t.Fatalf("saveAuthProfile failed: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runCleanupCommand([]string{"-auth-profile", profilePath})
	})
	if err != nil {
		t.Fatalf("runCleanupCommand failed: %v", err)
	}
	if output != formatCleanupRequestedMessage(2) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunCleanupCommandPrintsNoSessionsMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(protocol.AgentTerminateAllResponse{})
	}))
	t.Cleanup(srv.Close)

	originalDefault := defaultControlPlaneURL
	defaultControlPlaneURL = srv.URL
	t.Cleanup(func() { defaultControlPlaneURL = originalDefault })

	profilePath := filepath.Join(t.TempDir(), "auth.json")
	if err := saveAuthProfile(profilePath, authProfile{
		ControlPlaneURL: srv.URL,
		AccessToken:     "access-token-profile",
		RefreshToken:    "refresh-token-profile",
	}); err != nil {
		t.Fatalf("saveAuthProfile failed: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runCleanupCommand([]string{"-auth-profile", profilePath})
	})
	if err != nil {
		t.Fatalf("runCleanupCommand failed: %v", err)
	}
	if output != formatCleanupCompleteMessage() {
		t.Fatalf("unexpected output: %q", output)
	}
}
