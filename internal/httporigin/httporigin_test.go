// SPDX-License-Identifier: AGPL-3.0-or-later

package httporigin

import (
	"net/http/httptest"
	"testing"
)

func TestMakeSignalURLWithBaseURLUsesExternalBaseURL(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://10.0.0.7/v1/agents/attach", nil)
	req.RemoteAddr = "10.0.0.7:12345"
	req.Host = "internal-lb"
	req.Header.Set("X-Forwarded-Host", "attacker.example")

	if got := MakeSignalURLWithBaseURL(req, "https://api.shellin.dev"); got != "wss://api.shellin.dev/v1/agents/signal" {
		t.Fatalf("unexpected signal url: %s", got)
	}
}

func TestNormalizeExternalBaseURLRejectsPublicHTTP(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeExternalBaseURL("http://api.shellin.dev"); err == nil {
		t.Fatal("expected public http external base url to be rejected")
	}
}

func TestSignalURLFromExternalBaseURLAllowsLoopbackHTTP(t *testing.T) {
	t.Parallel()

	got, err := SignalURLFromExternalBaseURL("http://127.0.0.1:8090")
	if err != nil {
		t.Fatalf("SignalURLFromExternalBaseURL failed: %v", err)
	}
	if got != "ws://127.0.0.1:8090/v1/agents/signal" {
		t.Fatalf("unexpected signal url: %s", got)
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

func TestPrivateForwardedHostIsIgnoredWithoutExternalBaseURL(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://internal-lb/v1/agents/signal", nil)
	req.RemoteAddr = "10.0.0.7:12345"
	req.Header.Set("Origin", "https://attacker.example")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "attacker.example")

	if OriginAllowedForBaseURL(req, "") {
		t.Fatal("expected private forwarded host to be ignored without external base URL")
	}
	if got := EffectiveRequestHost(req); got != "internal-lb" {
		t.Fatalf("unexpected effective host: %q", got)
	}
	if got := EffectiveRequestScheme(req); got != "http" {
		t.Fatalf("unexpected effective scheme: %q", got)
	}
}

func TestLoopbackForwardedHostIsAllowedForLocalProxy(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://127.0.0.1:8090/v1/agents/signal", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Origin", "https://local.shellin.test")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "local.shellin.test")

	if !OriginAllowedForBaseURL(req, "") {
		t.Fatal("expected loopback forwarded host to be allowed")
	}
}
