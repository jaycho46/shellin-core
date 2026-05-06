// SPDX-License-Identifier: AGPL-3.0-or-later

package turn

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/jaycho46/shellin-core/protocol"
)

const CloudflareSTUNURL = "stun:stun.cloudflare.com:3478"

var OutboundHTTPClient = &http.Client{Timeout: 10 * time.Second}

type CloudflareConfig struct {
	APIBaseURL string
	KeyID      string
	APIToken   string
	TTL        time.Duration
}

type cloudflareCredentialsRequest struct {
	TTL              int    `json:"ttl"`
	CustomIdentifier string `json:"customIdentifier,omitempty"`
}

type cloudflareCredentialsResponse struct {
	ICEServers []protocol.ICEServer `json:"iceServers"`
	Result     *struct {
		ICEServers []protocol.ICEServer `json:"iceServers"`
	} `json:"result,omitempty"`
}

func CloudflareEnabled(cfg CloudflareConfig) bool {
	return strings.TrimSpace(cfg.KeyID) != "" && strings.TrimSpace(cfg.APIToken) != ""
}

func IssueCloudflareCredentials(cfg CloudflareConfig, subject string) ([]protocol.ICEServer, error) {
	if !CloudflareEnabled(cfg) {
		return nil, fmt.Errorf("cloudflare turn is not configured")
	}

	ttlSeconds := int(cfg.TTL / time.Second)
	if ttlSeconds <= 0 {
		return nil, fmt.Errorf("invalid cloudflare turn ttl: %s", cfg.TTL)
	}

	reqBody := cloudflareCredentialsRequest{TTL: ttlSeconds}
	if strings.TrimSpace(subject) != "" {
		reqBody.CustomIdentifier = "sub:" + shortHash(subject)
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/") +
		"/v1/turn/keys/" + neturl.PathEscape(strings.TrimSpace(cfg.KeyID)) +
		"/credentials/generate-ice-servers"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := OutboundHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return nil, fmt.Errorf("cloudflare turn api status %d: %s", resp.StatusCode, string(body))
	}

	var out cloudflareCredentialsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&out); err != nil {
		return nil, err
	}
	return SanitizeICEServers(extractCloudflareICEServers(out))
}

func SanitizeICEServers(in []protocol.ICEServer) ([]protocol.ICEServer, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("cloudflare turn api returned no ice servers")
	}

	out := make([]protocol.ICEServer, 0, len(in))
	hasTURN := false
	hasSTUN := false

	for _, server := range in {
		clean := protocol.ICEServer{
			URLs:       make([]string, 0, len(server.URLs)),
			Username:   strings.TrimSpace(server.Username),
			Credential: strings.TrimSpace(server.Credential),
		}
		for _, rawURL := range server.URLs {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				continue
			}
			if usesBlockedCloudflarePort(rawURL) {
				continue
			}
			if err := ValidateICEServerURL(rawURL); err != nil {
				return nil, fmt.Errorf("invalid cloudflare ice url %q: %w", rawURL, err)
			}
			if IsTURNURL(rawURL) {
				hasTURN = true
			}
			if IsSTUNURL(rawURL) {
				hasSTUN = true
			}
			clean.URLs = append(clean.URLs, rawURL)
		}
		if len(clean.URLs) == 0 {
			continue
		}
		out = append(out, clean)
	}

	if !hasSTUN {
		out = append([]protocol.ICEServer{{URLs: []string{CloudflareSTUNURL}}}, out...)
	}
	if !hasTURN {
		return nil, fmt.Errorf("cloudflare turn api returned no usable turn urls")
	}
	return out, nil
}

func ValidateICEServerURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("empty url")
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "stun:"),
		strings.HasPrefix(lower, "stuns:"),
		strings.HasPrefix(lower, "turn:"),
		strings.HasPrefix(lower, "turns:"):
	default:
		return fmt.Errorf("scheme must be stun/stuns/turn/turns")
	}
	return nil
}

func IsTURNURL(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return strings.HasPrefix(raw, "turn:") || strings.HasPrefix(raw, "turns:")
}

func IsSTUNURL(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return strings.HasPrefix(raw, "stun:") || strings.HasPrefix(raw, "stuns:")
}

func extractCloudflareICEServers(in cloudflareCredentialsResponse) []protocol.ICEServer {
	servers := in.ICEServers
	if len(servers) == 0 && in.Result != nil {
		servers = in.Result.ICEServers
	}

	out := make([]protocol.ICEServer, 0, len(servers)+1)
	out = append(out, protocol.ICEServer{URLs: []string{CloudflareSTUNURL}})
	for _, server := range servers {
		urls := make([]string, 0, len(server.URLs))
		for _, rawURL := range server.URLs {
			urls = append(urls, strings.TrimSpace(rawURL))
		}
		out = append(out, protocol.ICEServer{
			URLs:       urls,
			Username:   strings.TrimSpace(server.Username),
			Credential: strings.TrimSpace(server.Credential),
		})
	}
	return out
}

func usesBlockedCloudflarePort(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return hasExactURLPortMarker(lower, ":53") || hasExactURLPortMarker(lower, "=53")
}

func hasExactURLPortMarker(raw, marker string) bool {
	for offset := 0; ; {
		idx := strings.Index(raw[offset:], marker)
		if idx < 0 {
			return false
		}
		end := offset + idx + len(marker)
		if end == len(raw) || strings.ContainsRune("?/&", rune(raw[end])) {
			return true
		}
		offset = end
	}
}

func shortHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:])
	if len(encoded) > 16 {
		return encoded[:16]
	}
	return encoded
}
