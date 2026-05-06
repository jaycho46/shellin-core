// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaycho46/shellin-core/protocol"
)

func TestEnsureControlPlanePreflightRefreshesAndPersistsProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "auth.json")
	if err := saveAuthProfile(profilePath, authProfile{
		ControlPlaneURL: "https://example.com",
		AccessToken:     "expired-access",
		RefreshToken:    "refresh-token-1",
	}); err != nil {
		t.Fatalf("saveAuthProfile failed: %v", err)
	}

	var statusCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		statusCalls++
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if statusCalls == 1 {
			if auth != "Bearer expired-access" {
				t.Fatalf("unexpected first auth header: %q", auth)
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if auth != "Bearer refreshed-access" {
			t.Fatalf("unexpected second auth header: %q", auth)
		}
		_ = json.NewEncoder(w).Encode(protocol.SessionStatusResponse{
			ActiveSessions: 1,
			MaxSessions:    3,
		})
	})
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(protocol.RefreshAuthResponse{
			AccessToken:  "refreshed-access",
			RefreshToken: "refreshed-refresh",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	status, auth, err := ensureControlPlanePreflight(config{
		ControlPlaneURL: srv.URL,
	}, controlPlaneAuthSession{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-token-1",
	}, profilePath)
	if err != nil {
		t.Fatalf("ensureControlPlanePreflight failed: %v", err)
	}
	if status.ActiveSessions != 1 || status.MaxSessions != 3 {
		t.Fatalf("unexpected status: %+v", status)
	}
	if auth.AccessToken != "refreshed-access" || auth.RefreshToken != "refreshed-refresh" {
		t.Fatalf("unexpected auth: %+v", auth)
	}

	profile, err := loadAuthProfile(profilePath)
	if err != nil {
		t.Fatalf("loadAuthProfile failed: %v", err)
	}
	if profile.AccessToken != "refreshed-access" || profile.RefreshToken != "refreshed-refresh" {
		t.Fatalf("unexpected persisted profile: %+v", profile)
	}
}

func TestEnsureControlPlanePreflightReloadsSharedAuthAfterRefreshReuse(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "auth.json")
	var serverURL string

	var statusCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		statusCalls++
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if statusCalls == 1 {
			if auth != "Bearer expired-access" {
				t.Fatalf("unexpected first auth header: %q", auth)
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if auth != "Bearer shared-access" {
			t.Fatalf("unexpected second auth header: %q", auth)
		}
		_ = json.NewEncoder(w).Encode(protocol.SessionStatusResponse{
			ActiveSessions: 0,
			MaxSessions:    3,
		})
	})
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if err := saveAuthProfile(profilePath, authProfile{
			ControlPlaneURL: serverURL,
			AccessToken:     "shared-access",
			RefreshToken:    "shared-refresh",
		}); err != nil {
			t.Fatalf("saveAuthProfile failed: %v", err)
		}
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	serverURL = srv.URL
	if err := saveAuthProfile(profilePath, authProfile{
		ControlPlaneURL: serverURL,
		AccessToken:     "expired-access",
		RefreshToken:    "refresh-token-1",
	}); err != nil {
		t.Fatalf("saveAuthProfile failed: %v", err)
	}

	status, auth, err := ensureControlPlanePreflight(config{
		ControlPlaneURL: srv.URL,
	}, controlPlaneAuthSession{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-token-1",
	}, profilePath)
	if err != nil {
		t.Fatalf("ensureControlPlanePreflight failed: %v", err)
	}
	if status.ActiveSessions != 0 || status.MaxSessions != 3 {
		t.Fatalf("unexpected status: %+v", status)
	}
	if auth.AccessToken != "shared-access" || auth.RefreshToken != "shared-refresh" {
		t.Fatalf("unexpected auth: %+v", auth)
	}
}

func TestEnsureControlPlanePreflightRejectsAtSessionLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(protocol.SessionStatusResponse{
			ActiveSessions: 2,
			MaxSessions:    1,
		})
	}))
	t.Cleanup(srv.Close)

	_, _, err := ensureControlPlanePreflight(config{
		ControlPlaneURL: srv.URL,
	}, controlPlaneAuthSession{
		AccessToken:  "access-token-1",
		RefreshToken: "refresh-token-1",
	}, "")
	if err == nil {
		t.Fatal("expected session limit error")
	}
	if !strings.Contains(err.Error(), "session limit reached: 2 active / 1 max") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureControlPlanePreflightRejectsInactiveSubscription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(protocol.SessionStatusResponse{
			ActiveSessions:     0,
			MaxSessions:        1,
			SubscriptionStatus: "expired",
		})
	}))
	t.Cleanup(srv.Close)

	_, _, err := ensureControlPlanePreflight(config{
		ControlPlaneURL: srv.URL,
	}, controlPlaneAuthSession{
		AccessToken:  "access-token-1",
		RefreshToken: "refresh-token-1",
	}, "")
	if err == nil {
		t.Fatal("expected inactive subscription error")
	}
	if !isSubscriptionInactiveError(err) {
		t.Fatalf("expected subscription inactive error, got %v", err)
	}
	if err.Error() != "Your subscription has expired." {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHeartbeatAgentSessionWithAuthRecoveryStopsOnTerminatedSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "agent session terminated", http.StatusGone)
	}))
	t.Cleanup(srv.Close)

	_, _, err := heartbeatAgentSessionWithAuthRecovery(config{
		ControlPlaneURL: srv.URL,
	}, "", "sess_1", "macbook", controlPlaneAuthSession{
		AccessToken: "access-token-1",
	})
	if err == nil {
		t.Fatal("expected terminated session error")
	}
	if !isNonRetryableControlPlaneError(err) {
		t.Fatalf("expected non-retryable error, got %v", err)
	}
	if !isSessionTerminatedError(err) {
		t.Fatalf("expected terminated session marker, got %v", err)
	}
}

func TestRequestAgentConnectWithAuthSessionSurfacesRefreshFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agents/connect", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expired refresh token", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	_, _, _, err := requestAgentConnectWithAuthSession(config{
		ControlPlaneURL: srv.URL,
	}, controlPlaneAuthSession{
		AccessToken:  "expired-access",
		RefreshToken: "expired-refresh",
	}, "sess_1", "macbook")
	if err == nil {
		t.Fatal("expected connect failure")
	}
	if !strings.Contains(err.Error(), "control plane connect status 401: unauthorized") {
		t.Fatalf("expected connect failure in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "refresh auth failed: control plane refresh status 401: expired refresh token") {
		t.Fatalf("expected refresh failure details in error, got %v", err)
	}
	if !isNonRetryableControlPlaneError(err) {
		t.Fatalf("expected non-retryable connect auth failure, got %v", err)
	}
}

func TestRequestAgentConnect(t *testing.T) {
	var gotRequest protocol.AgentConnectRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/agents/connect" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer access-token-1" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		_ = json.NewEncoder(w).Encode(protocol.AgentConnectResponse{
			SessionID:   "sess_1",
			SignalURL:   "wss://api.example.com/v1/agents/signal",
			SignalToken: "signal-token-1",
			ICEPolicy:   "all",
			ICEServers: []protocol.ICEServer{
				{URLs: []string{"stun:stun.cloudflare.com:3478"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	resp, err := requestAgentConnect(config{ControlPlaneURL: srv.URL}, "access-token-1", "sess_1", "macbook")
	if err != nil {
		t.Fatalf("requestAgentConnect failed: %v", err)
	}
	if gotRequest.SessionID != "sess_1" || gotRequest.AgentLabel != "macbook" {
		t.Fatalf("unexpected request payload: %+v", gotRequest)
	}
	if resp.SignalURL != "wss://api.example.com/v1/agents/signal" || resp.SignalToken != "signal-token-1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRequestAgentConnectWithAuthSessionRefreshesOnUnauthorized(t *testing.T) {
	var connectCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agents/connect", func(w http.ResponseWriter, r *http.Request) {
		connectCalls++
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if connectCalls == 1 {
			if auth != "Bearer expired-access" {
				t.Fatalf("unexpected first auth header: %q", auth)
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if auth != "Bearer refreshed-access" {
			t.Fatalf("unexpected second auth header: %q", auth)
		}
		_ = json.NewEncoder(w).Encode(protocol.AgentConnectResponse{
			SessionID:   "sess_1",
			SignalURL:   "wss://api.example.com/v1/agents/signal",
			SignalToken: "signal-token-2",
			ICEPolicy:   "all",
			ICEServers: []protocol.ICEServer{
				{URLs: []string{"stun:stun.cloudflare.com:3478"}},
			},
		})
	})
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var in protocol.RefreshAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode refresh body failed: %v", err)
		}
		if in.RefreshToken != "refresh-token-1" {
			t.Fatalf("unexpected refresh token: %q", in.RefreshToken)
		}
		_ = json.NewEncoder(w).Encode(protocol.RefreshAuthResponse{
			AccessToken:  "refreshed-access",
			RefreshToken: "refreshed-refresh",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, auth, mode, err := requestAgentConnectWithAuthSession(config{
		ControlPlaneURL: srv.URL,
	}, controlPlaneAuthSession{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-token-1",
	}, "sess_1", "macbook")
	if err != nil {
		t.Fatalf("requestAgentConnectWithAuthSession failed: %v", err)
	}
	if mode != controlPlaneAuthRefreshed {
		t.Fatalf("unexpected auth mode: %v", mode)
	}
	if auth.AccessToken != "refreshed-access" || auth.RefreshToken != "refreshed-refresh" {
		t.Fatalf("unexpected refreshed auth: %+v", auth)
	}
	if resp.SignalToken != "signal-token-2" || connectCalls != 2 {
		t.Fatalf("unexpected response or call count: resp=%+v calls=%d", resp, connectCalls)
	}
}

func TestHeartbeatAgentSessionWithAuthRecoveryReregistersMissingAgent(t *testing.T) {
	var registerCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agents/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "agent session not found", http.StatusNotFound)
	})
	mux.HandleFunc("/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
		registerCalls++
		var in protocol.AgentRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode register body failed: %v", err)
		}
		if in.SessionID != "sess_1" {
			t.Fatalf("unexpected session id: %q", in.SessionID)
		}
		if in.AgentLabel != "macbook" {
			t.Fatalf("unexpected agent label: %q", in.AgentLabel)
		}
		if in.Transport != "webrtc_turn" {
			t.Fatalf("unexpected transport: %q", in.Transport)
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	finalAuth, mode, err := heartbeatAgentSessionWithAuthRecovery(config{
		ControlPlaneURL: srv.URL,
	}, "", "sess_1", "macbook", controlPlaneAuthSession{
		AccessToken: "access-token-1",
	})
	if err != nil {
		t.Fatalf("heartbeatAgentSessionWithAuthRecovery failed: %v", err)
	}
	if mode != controlPlaneAuthUnchanged {
		t.Fatalf("unexpected auth mode: %v", mode)
	}
	if finalAuth.AccessToken != "access-token-1" {
		t.Fatalf("unexpected final auth: %+v", finalAuth)
	}
	if registerCalls != 1 {
		t.Fatalf("expected 1 register call, got %d", registerCalls)
	}
}

func TestHeartbeatAgentSessionWithAuthRecoveryFallsBackToUserKey(t *testing.T) {
	var heartbeatCalls int
	var exchangeCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agents/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		heartbeatCalls++
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if heartbeatCalls == 1 {
			if auth != "Bearer expired-access" {
				t.Fatalf("unexpected first auth header: %q", auth)
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if auth != "Bearer fallback-access" {
			t.Fatalf("unexpected fallback auth header: %q", auth)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mux.HandleFunc("/v1/auth/exchange", func(w http.ResponseWriter, r *http.Request) {
		exchangeCalls++
		var in protocol.ExchangeAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode exchange body failed: %v", err)
		}
		if in.UserKey != "fallback-user-key" {
			t.Fatalf("unexpected user key: %q", in.UserKey)
		}
		_ = json.NewEncoder(w).Encode(protocol.ExchangeAuthResponse{
			AccessToken:  "fallback-access",
			RefreshToken: "fallback-refresh",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	finalAuth, mode, err := heartbeatAgentSessionWithAuthRecovery(config{
		ControlPlaneURL: srv.URL,
		UserKey:         "fallback-user-key",
	}, "", "sess_1", "macbook", controlPlaneAuthSession{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-token-1",
	})
	if err != nil {
		t.Fatalf("heartbeatAgentSessionWithAuthRecovery failed: %v", err)
	}
	if mode != controlPlaneAuthRefreshed {
		t.Fatalf("unexpected auth mode: %v", mode)
	}
	if finalAuth.AccessToken != "fallback-access" || finalAuth.RefreshToken != "fallback-refresh" {
		t.Fatalf("unexpected final auth: %+v", finalAuth)
	}
	if heartbeatCalls != 2 {
		t.Fatalf("expected 2 heartbeat calls, got %d", heartbeatCalls)
	}
	if exchangeCalls != 1 {
		t.Fatalf("expected 1 exchange call, got %d", exchangeCalls)
	}
}

func TestHeartbeatAgentSessionWithAuthRecoveryMarksFallbackFailureNonRetryable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agents/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	mux.HandleFunc("/v1/auth/exchange", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	_, _, err := heartbeatAgentSessionWithAuthRecovery(config{
		ControlPlaneURL: srv.URL,
		UserKey:         "fallback-user-key",
	}, "", "sess_1", "macbook", controlPlaneAuthSession{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-token-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !isNonRetryableControlPlaneError(err) {
		t.Fatalf("expected non-retryable error, got %v", err)
	}
	if !strings.Contains(err.Error(), "fallback user key exchange failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeviceLoginWithKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/device-login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var in protocol.DeviceLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if in.LoginKey != "apple-logo-alert-bank" {
			t.Fatalf("unexpected login key: %q", in.LoginKey)
		}
		_ = json.NewEncoder(w).Encode(protocol.ExchangeAuthResponse{
			AccessToken:  "device-access",
			RefreshToken: "device-refresh",
		})
	}))
	t.Cleanup(srv.Close)

	out, err := deviceLoginWithKey(srv.URL, "apple-logo-alert-bank")
	if err != nil {
		t.Fatalf("deviceLoginWithKey failed: %v", err)
	}
	if out.AccessToken != "device-access" || out.RefreshToken != "device-refresh" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestSaveLoadAuthProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	in := authProfile{
		ControlPlaneURL:       "https://api.example.com",
		AccessToken:           "a1",
		AccessTokenExpiresAt:  "2030-01-01T00:00:00Z",
		RefreshToken:          "r1",
		RefreshTokenExpiresAt: "2030-01-10T00:00:00Z",
		UpdatedAt:             "2030-01-01T00:00:00Z",
	}
	if err := saveAuthProfile(path, in); err != nil {
		t.Fatalf("saveAuthProfile failed: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 mode, got %o", st.Mode().Perm())
	}
	out, err := loadAuthProfile(path)
	if err != nil {
		t.Fatalf("loadAuthProfile failed: %v", err)
	}
	if out.AccessToken != in.AccessToken || out.RefreshToken != in.RefreshToken || out.ControlPlaneURL != in.ControlPlaneURL {
		t.Fatalf("unexpected loaded profile: %+v", out)
	}
}
