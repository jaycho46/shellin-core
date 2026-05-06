// SPDX-License-Identifier: AGPL-3.0-or-later

package signaling

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/jaycho46/shellin-core/controlplane"
	"github.com/jaycho46/shellin-core/grantauth"
	"github.com/jaycho46/shellin-core/protocol"
)

func TestBearerTokenRequiresAuthorizationHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/v1/agents/signal?token=query-token", nil)
	if got := bearerToken(req); got != "" {
		t.Fatalf("expected query token to be ignored, got %q", got)
	}
}

func TestBearerTokenAcceptsBearerHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/v1/agents/signal", nil)
	req.Header.Set("Authorization", "Bearer signal-token")
	if got := bearerToken(req); got != "signal-token" {
		t.Fatalf("unexpected bearer token: %q", got)
	}
}

func TestOriginAllowedForBaseURLUsesConfiguredExternalOrigin(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://internal-lb/v1/agents/signal", nil)
	req.RemoteAddr = "10.0.0.7:12345"
	req.Header.Set("Origin", "https://api.shellin.dev")
	req.Header.Set("X-Forwarded-Host", "attacker.example")

	if !OriginAllowedForBaseURL(req, "https://api.shellin.dev") {
		t.Fatal("expected configured external origin to be allowed")
	}
	req.Header.Set("Origin", "https://attacker.example")
	if OriginAllowedForBaseURL(req, "https://api.shellin.dev") {
		t.Fatal("expected unconfigured external origin to be rejected")
	}
}

func TestHandleWebSocketForwardsSignalMessagesBetweenPeers(t *testing.T) {
	t.Parallel()

	const (
		secret    = "abcdefghijklmnopqrstuvwxyz0123456789"
		issuer    = "shellin.dev"
		audience  = "webrtc.signal"
		subject   = "user-1"
		sessionID = "session-1"
	)
	registry := controlplane.NewAgentRegistry()
	if _, err := registry.Upsert(subject, sessionID, "macbook", string(protocol.TransportWebRTCTURN)); err != nil {
		t.Fatalf("register agent session: %v", err)
	}
	validator, err := grantauth.NewValidator(secret, issuer, audience)
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	server := httptest.NewServer(Handler(Options{
		Validator:   validator,
		ReplayStore: grantauth.NewReplayGuard(),
		Registry:    registry,
		StaleAfter:  time.Minute,
		Hub:         controlplane.NewSignalHub(),
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	agent := dialTestSignalWebSocket(t, wsURL, signedTestSignalToken(t, secret, issuer, audience, subject, string(controlplane.SignalRoleAgent), sessionID, "agent-jti"))
	defer agent.Close()
	viewer := dialTestSignalWebSocket(t, wsURL, signedTestSignalToken(t, secret, issuer, audience, subject, string(controlplane.SignalRoleViewer), sessionID, "viewer-jti"))
	defer viewer.Close()

	if err := viewer.WriteJSON(protocol.SignalMessage{Type: protocol.SignalMessageTypeViewerReady}); err != nil {
		t.Fatalf("write viewer_ready: %v", err)
	}
	if got := readTestSignalMessage(t, agent); got.Type != protocol.SignalMessageTypeViewerReady {
		t.Fatalf("agent received unexpected message: %+v", got)
	}

	offer := protocol.SignalMessage{
		Type: protocol.SignalMessageTypeOffer,
		SDP:  &protocol.SDPPayload{Type: protocol.SDPTypeOffer, SDP: "v=0"},
	}
	if err := agent.WriteJSON(offer); err != nil {
		t.Fatalf("write offer: %v", err)
	}
	got := readTestSignalMessage(t, viewer)
	if got.Type != protocol.SignalMessageTypeOffer || got.SDP == nil || got.SDP.SDP != "v=0" {
		t.Fatalf("viewer received unexpected message: %+v", got)
	}
}

func TestHandleWebSocketRejectsReplayedSignalToken(t *testing.T) {
	t.Parallel()

	const (
		secret    = "abcdefghijklmnopqrstuvwxyz0123456789"
		issuer    = "shellin.dev"
		audience  = "webrtc.signal"
		subject   = "user-1"
		sessionID = "session-1"
	)
	validator, err := grantauth.NewValidator(secret, issuer, audience)
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	server := httptest.NewServer(Handler(Options{
		Validator:   validator,
		ReplayStore: grantauth.NewReplayGuard(),
		Registry:    controlplane.NewAgentRegistry(),
		StaleAfter:  time.Minute,
		Hub:         controlplane.NewSignalHub(),
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	token := signedTestSignalToken(t, secret, issuer, audience, subject, string(controlplane.SignalRoleAgent), sessionID, "same-jti")

	conn := dialTestSignalWebSocket(t, wsURL, token)
	_ = conn.Close()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	_, resp, err := testSignalDialer().Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected replayed token dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for replayed token, got resp=%v err=%v", resp, err)
	}
}

func dialTestSignalWebSocket(t *testing.T, rawURL, token string) *websocket.Conn {
	t.Helper()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	conn, resp, err := testSignalDialer().Dial(rawURL, header)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial websocket failed with status %d: %v", resp.StatusCode, err)
		}
		t.Fatalf("dial websocket failed: %v", err)
	}
	return conn
}

func readTestSignalMessage(t *testing.T, conn *websocket.Conn) protocol.SignalMessage {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var msg protocol.SignalMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read signal message: %v", err)
	}
	return msg
}

func testSignalDialer() *websocket.Dialer {
	return &websocket.Dialer{
		HandshakeTimeout: 2 * time.Second,
		Subprotocols:     []string{"mt-signaling"},
	}
}

func signedTestSignalToken(t *testing.T, secret, issuer, audience, subject, role, sessionID, tokenID string) string {
	t.Helper()

	now := time.Now().UTC()
	token, err := grantauth.SignHS256(secret, map[string]any{
		"sub":        subject,
		"iss":        issuer,
		"aud":        audience,
		"exp":        now.Add(time.Minute).Unix(),
		"nbf":        now.Add(-time.Second).Unix(),
		"jti":        tokenID,
		"role":       role,
		"session_id": sessionID,
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}
