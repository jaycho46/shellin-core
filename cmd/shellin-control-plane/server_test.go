// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaycho46/shellin-core/controlplane"
	"github.com/jaycho46/shellin-core/protocol"
)

const strongControlPlaneTestSecret = "abcdefghijklmnopqrstuvwxyz0123456789"

func TestNewServerExchangesAuthAndServesStatus(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t, 2)
	auth := exchangeTestAuth(t, handler, "uk_test")

	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status returned %d: %s", rec.Code, rec.Body.String())
	}
	var status protocol.SessionStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode status failed: %v", err)
	}
	if status.Subject != "demo-user" || status.MaxSessions != 2 || status.ActiveSessions != 0 {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestNewServerConnectUsesConfiguredExternalSignalURL(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t, 1)
	auth := exchangeTestAuth(t, handler, "uk_test")
	body := bytes.NewBufferString(`{"session_id":"session-1","agent_label":"dev shell"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/agents/connect", body)
	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("connect returned %d: %s", rec.Code, rec.Body.String())
	}
	var out protocol.AgentConnectResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode connect failed: %v", err)
	}
	if out.SignalURL != "wss://api.shellin.dev/v1/agents/signal" {
		t.Fatalf("unexpected signal url: %s", out.SignalURL)
	}
	if out.SignalToken == "" || len(out.ICEServers) == 0 {
		t.Fatalf("incomplete connect response: %+v", out)
	}
}

func TestNewServerAddsSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := newTestServer(t, 1)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health returned %d", rec.Code)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected content type options header: %q", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("unexpected referrer policy header: %q", got)
	}
}

func TestNewServerRateLimitsCredentialFailures(t *testing.T) {
	t.Parallel()

	cfg := testConfig(1)
	cfg.AuthFailureLimit = 2
	cfg.AuthFailureWindow = time.Minute
	handler := newTestServerWithConfig(t, cfg)

	for i := 0; i < cfg.AuthFailureLimit; i++ {
		code, body := exchangeTestAuthStatus(t, handler, "wrong_key", "198.51.100.10:2000")
		if code != http.StatusUnauthorized {
			t.Fatalf("attempt %d returned %d: %s", i+1, code, body)
		}
	}

	code, body := exchangeTestAuthStatus(t, handler, "wrong_key", "198.51.100.10:2000")
	if code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limit, got %d: %s", code, body)
	}
}

func newTestServer(t *testing.T, maxSessions int) http.Handler {
	t.Helper()

	return newTestServerWithConfig(t, testConfig(maxSessions))
}

func testConfig(maxSessions int) config {
	return config{
		Addr:                  "127.0.0.1:0",
		UserKeys:              "uk_test:demo-user",
		DefaultMaxSessions:    maxSessions,
		GrantHMACSecret:       strongControlPlaneTestSecret,
		GrantIssuer:           "shellin.dev",
		AccessAudience:        "control.grants",
		AccessTTL:             10 * time.Minute,
		RefreshTTL:            time.Hour,
		DeviceLoginKeyTTL:     10 * time.Minute,
		AuthFailureLimit:      defaultAuthFailureLimit,
		AuthFailureWindow:     defaultAuthFailureWindow,
		AgentHeartbeatTimeout: 2 * time.Minute,
		SignalAudience:        "webrtc.signal",
		SignalTokenTTL:        2 * time.Minute,
		ExternalBaseURL:       "https://api.shellin.dev",
		CloudflareTURNBaseURL: "https://rtc.live.cloudflare.com",
		CloudflareTURNTTL:     time.Hour,
	}
}

func newTestServerWithConfig(t *testing.T, rawCfg config) http.Handler {
	t.Helper()

	cfg, err := normalizeConfig(rawCfg)
	if err != nil {
		t.Fatalf("normalize config failed: %v", err)
	}
	store := controlplane.NewUserKeyStore()
	if err := seedUserKeys(cfg, store); err != nil {
		t.Fatalf("seed user keys failed: %v", err)
	}
	deps, err := newServerDeps(cfg, store)
	if err != nil {
		t.Fatalf("new server deps failed: %v", err)
	}
	return NewServer(cfg, deps)
}

func exchangeTestAuth(t *testing.T, handler http.Handler, userKey string) protocol.ExchangeAuthResponse {
	t.Helper()

	code, body := exchangeTestAuthStatus(t, handler, userKey, "192.0.2.1:1234")
	if code != http.StatusOK {
		t.Fatalf("exchange returned %d: %s", code, body)
	}
	var out protocol.ExchangeAuthResponse
	if err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&out); err != nil {
		t.Fatalf("decode exchange failed: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatalf("incomplete exchange response: %+v", out)
	}
	return out
}

func exchangeTestAuthStatus(t *testing.T, handler http.Handler, userKey, remoteAddr string) (int, string) {
	t.Helper()

	body := bytes.NewBufferString(`{"user_key":"` + userKey + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/exchange", body)
	req.RemoteAddr = remoteAddr
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec.Code, rec.Body.String()
}
