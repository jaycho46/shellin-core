// SPDX-License-Identifier: AGPL-3.0-or-later

package turn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jaycho46/shellin-core/protocol"
)

func TestIssueCloudflareCredentialsCallsAPIAndSanitizesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.EscapedPath() != "/v1/turn/keys/key%2Fone/credentials/generate-ice-servers" {
			t.Fatalf("unexpected path: %s", r.URL.EscapedPath())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		var in cloudflareCredentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if in.TTL != 60 {
			t.Fatalf("unexpected ttl: %d", in.TTL)
		}
		if !strings.HasPrefix(in.CustomIdentifier, "sub:") {
			t.Fatalf("unexpected custom identifier: %q", in.CustomIdentifier)
		}
		_ = json.NewEncoder(w).Encode(cloudflareCredentialsResponse{
			Result: &struct {
				ICEServers []protocol.ICEServer `json:"iceServers"`
			}{
				ICEServers: []protocol.ICEServer{
					{
						URLs:       []string{" turn:relay.example.com:3478?transport=udp ", "turn:relay.example.com:53?transport=udp"},
						Username:   " user ",
						Credential: " pass ",
					},
				},
			},
		})
	}))
	defer server.Close()

	oldClient := OutboundHTTPClient
	OutboundHTTPClient = server.Client()
	t.Cleanup(func() { OutboundHTTPClient = oldClient })

	servers, err := IssueCloudflareCredentials(CloudflareConfig{
		APIBaseURL: server.URL,
		KeyID:      "key/one",
		APIToken:   "token-1",
		TTL:        time.Minute,
	}, "user-1")
	if err != nil {
		t.Fatalf("IssueCloudflareCredentials failed: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected stun plus sanitized turn server, got %+v", servers)
	}
	if servers[0].URLs[0] != CloudflareSTUNURL {
		t.Fatalf("expected fallback stun server first, got %+v", servers)
	}
	if got := servers[1].URLs; len(got) != 1 || got[0] != "turn:relay.example.com:3478?transport=udp" {
		t.Fatalf("unexpected sanitized urls: %+v", got)
	}
	if servers[1].Username != "user" || servers[1].Credential != "pass" {
		t.Fatalf("credentials were not trimmed: %+v", servers[1])
	}
}

func TestSanitizeICEServersRejectsNoUsableTURN(t *testing.T) {
	t.Parallel()

	_, err := SanitizeICEServers([]protocol.ICEServer{
		{URLs: []string{"stun:stun.example.com:3478", "turn:relay.example.com:53?transport=udp"}},
	})
	if err == nil || !strings.Contains(err.Error(), "no usable turn urls") {
		t.Fatalf("expected no usable TURN error, got %v", err)
	}
}

func TestValidateICEServerURLRejectsUnexpectedScheme(t *testing.T) {
	t.Parallel()

	if err := ValidateICEServerURL("https://relay.example.com"); err == nil {
		t.Fatal("expected non-ICE scheme to be rejected")
	}
}

func TestUsesBlockedCloudflarePortAllowsPort5349(t *testing.T) {
	t.Parallel()

	if usesBlockedCloudflarePort("turns:relay.example.com:5349?transport=tcp") {
		t.Fatal("expected port 5349 to remain usable")
	}
	if !usesBlockedCloudflarePort("turn:relay.example.com:53?transport=udp") {
		t.Fatal("expected port 53 to be blocked")
	}
}
