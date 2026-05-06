// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jaycho46/shellin-core/internal/httporigin"
)

const (
	defaultCloudflareTURNTTL = time.Hour
	maxCloudflareTURNTTL     = 6 * time.Hour
	defaultAuthFailureLimit  = 10
	defaultAuthFailureWindow = 5 * time.Minute
)

type config struct {
	Addr                  string
	UserKeys              string
	UserKeysFile          string
	DefaultMaxSessions    int
	GrantHMACSecret       string
	GrantIssuer           string
	AccessAudience        string
	AccessTTL             time.Duration
	RefreshTTL            time.Duration
	DeviceLoginKeyTTL     time.Duration
	AuthFailureLimit      int
	AuthFailureWindow     time.Duration
	AgentHeartbeatTimeout time.Duration
	SignalAudience        string
	SignalTokenTTL        time.Duration
	ExternalBaseURL       string
	CloudflareTURNBaseURL string
	CloudflareTURNKeyID   string
	CloudflareTURNToken   string
	CloudflareTURNTTL     time.Duration
}

func parseConfig(args []string) (config, error) {
	cfg := config{}
	fs := flag.NewFlagSet("shellin-control-plane", flag.ContinueOnError)
	fs.StringVar(&cfg.Addr, "addr", env("SHELLIN_CONTROL_PLANE_ADDR", "127.0.0.1:8090"), "HTTP listen address")
	fs.StringVar(&cfg.UserKeys, "user-keys", strings.TrimSpace(os.Getenv("SHELLIN_USER_KEYS")), "Comma separated user key entries: user_key:subject[:max_sessions]")
	fs.StringVar(&cfg.UserKeysFile, "user-keys-file", strings.TrimSpace(os.Getenv("SHELLIN_USER_KEYS_FILE")), "Optional file with user key entries")
	fs.IntVar(&cfg.DefaultMaxSessions, "default-max-sessions", 1, "Default max_sessions when not specified per key")
	fs.StringVar(&cfg.GrantHMACSecret, "grant-hmac-secret", strings.TrimSpace(os.Getenv("SHELLIN_GRANT_HMAC_SECRET")), "HMAC secret used to sign access and signaling tokens")
	fs.StringVar(&cfg.GrantIssuer, "grant-issuer", env("SHELLIN_GRANT_ISSUER", "shellin.dev"), "Issuer value added to tokens")
	fs.StringVar(&cfg.AccessAudience, "access-audience", env("SHELLIN_ACCESS_AUDIENCE", "control.grants"), "Audience for control-plane access tokens")
	fs.DurationVar(&cfg.AccessTTL, "access-ttl", durationEnv("SHELLIN_ACCESS_TTL", 10*time.Minute), "Control-plane access token lifetime")
	fs.DurationVar(&cfg.RefreshTTL, "refresh-ttl", durationEnv("SHELLIN_REFRESH_TTL", 720*time.Hour), "Refresh token lifetime")
	fs.DurationVar(&cfg.DeviceLoginKeyTTL, "device-login-key-ttl", durationEnv("SHELLIN_DEVICE_LOGIN_KEY_TTL", 10*time.Minute), "One-time device login key lifetime")
	fs.IntVar(&cfg.AuthFailureLimit, "auth-failure-limit", intEnv("SHELLIN_AUTH_FAILURE_LIMIT", defaultAuthFailureLimit), "Failed credential attempts allowed per remote address")
	fs.DurationVar(&cfg.AuthFailureWindow, "auth-failure-window", durationEnv("SHELLIN_AUTH_FAILURE_WINDOW", defaultAuthFailureWindow), "Window for failed credential attempt throttling")
	fs.DurationVar(&cfg.AgentHeartbeatTimeout, "agent-heartbeat-timeout", durationEnv("SHELLIN_AGENT_HEARTBEAT_TIMEOUT", 2*time.Minute), "Agent heartbeat timeout")
	fs.StringVar(&cfg.SignalAudience, "signal-audience", env("SHELLIN_SIGNAL_AUDIENCE", "webrtc.signal"), "Audience for WebRTC signaling tokens")
	fs.DurationVar(&cfg.SignalTokenTTL, "signal-token-ttl", durationEnv("SHELLIN_SIGNAL_TOKEN_TTL", 2*time.Minute), "WebRTC signaling token lifetime")
	fs.StringVar(&cfg.ExternalBaseURL, "external-base-url", env("SHELLIN_EXTERNAL_BASE_URL", ""), "External http(s) base URL used to generate signaling URLs and validate WebSocket origins")
	fs.StringVar(&cfg.CloudflareTURNBaseURL, "cloudflare-turn-api-base-url", env("SHELLIN_CLOUDFLARE_TURN_API_BASE_URL", "https://rtc.live.cloudflare.com"), "Cloudflare TURN API base URL")
	fs.StringVar(&cfg.CloudflareTURNKeyID, "cloudflare-turn-key-id", strings.TrimSpace(os.Getenv("SHELLIN_CLOUDFLARE_TURN_KEY_ID")), "Cloudflare TURN key ID")
	fs.StringVar(&cfg.CloudflareTURNToken, "cloudflare-turn-api-token", strings.TrimSpace(os.Getenv("SHELLIN_CLOUDFLARE_TURN_API_TOKEN")), "Cloudflare TURN API token")
	fs.DurationVar(&cfg.CloudflareTURNTTL, "cloudflare-turn-ttl", durationEnv("SHELLIN_CLOUDFLARE_TURN_TTL", defaultCloudflareTURNTTL), "Cloudflare TURN credential TTL")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	return normalizeConfig(cfg)
}

func normalizeConfig(cfg config) (config, error) {
	if cfg.GrantHMACSecret == "" {
		return config{}, fmt.Errorf("-grant-hmac-secret or SHELLIN_GRANT_HMAC_SECRET is required")
	}
	if cfg.DefaultMaxSessions <= 0 {
		return config{}, fmt.Errorf("-default-max-sessions must be > 0")
	}
	if cfg.AuthFailureLimit <= 0 {
		return config{}, fmt.Errorf("-auth-failure-limit must be > 0")
	}
	if cfg.AccessTTL <= 0 || cfg.RefreshTTL <= 0 || cfg.SignalTokenTTL <= 0 || cfg.DeviceLoginKeyTTL <= 0 || cfg.AgentHeartbeatTimeout <= 0 {
		return config{}, fmt.Errorf("token and heartbeat durations must be > 0")
	}
	if cfg.AuthFailureWindow <= 0 {
		return config{}, fmt.Errorf("-auth-failure-window must be > 0")
	}
	if cfg.CloudflareTURNTTL <= 0 || cfg.CloudflareTURNTTL > maxCloudflareTURNTTL {
		return config{}, fmt.Errorf("-cloudflare-turn-ttl must be > 0 and <= %s", maxCloudflareTURNTTL)
	}
	normalizedExternalBaseURL, err := httporigin.NormalizeExternalBaseURL(cfg.ExternalBaseURL)
	if err != nil {
		return config{}, fmt.Errorf("invalid -external-base-url: %w", err)
	}
	cfg.ExternalBaseURL = normalizedExternalBaseURL
	return cfg, nil
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func intEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
