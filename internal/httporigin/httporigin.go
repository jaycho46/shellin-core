// SPDX-License-Identifier: AGPL-3.0-or-later

package httporigin

import (
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
)

func MakeSignalURL(r *http.Request) string {
	scheme := "wss"
	host := EffectiveRequestHost(r)
	if EffectiveRequestScheme(r) == "http" && IsLoopbackHost(HostWithoutPort(host)) {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s/v1/agents/signal", scheme, host)
}

func MakeSignalURLWithBaseURL(r *http.Request, externalBaseURL string) string {
	signalURL, err := SignalURLFromExternalBaseURL(externalBaseURL)
	if err == nil && signalURL != "" {
		return signalURL
	}
	return MakeSignalURL(r)
}

func NormalizeExternalBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := neturl.Parse(raw)
	if err != nil || strings.TrimSpace(u.Host) == "" {
		return "", fmt.Errorf("external base url must include scheme and host")
	}
	switch strings.ToLower(strings.TrimSpace(u.Scheme)) {
	case "https":
	case "http":
		if !IsLoopbackHost(HostWithoutPort(u.Host)) {
			return "", fmt.Errorf("external base url must use https outside loopback")
		}
	default:
		return "", fmt.Errorf("external base url scheme must be http or https")
	}
	if strings.TrimSpace(u.Path) != "" {
		return "", fmt.Errorf("external base url must not include a path")
	}
	return strings.ToLower(strings.TrimSpace(u.Scheme)) + "://" + strings.TrimSpace(u.Host), nil
}

func SignalURLFromExternalBaseURL(raw string) (string, error) {
	baseURL, err := NormalizeExternalBaseURL(raw)
	if err != nil || baseURL == "" {
		return "", err
	}
	u, err := neturl.Parse(baseURL)
	if err != nil {
		return "", err
	}
	scheme := "wss"
	if strings.EqualFold(u.Scheme, "http") && IsLoopbackHost(HostWithoutPort(u.Host)) {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s/v1/agents/signal", scheme, u.Host), nil
}

func OriginAllowedForBaseURL(r *http.Request, externalBaseURL string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	u, err := neturl.Parse(origin)
	if err != nil || strings.TrimSpace(u.Host) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(u.Scheme)) {
	case "http", "https":
	default:
		return false
	}

	if externalBaseURL != "" {
		baseURL, err := NormalizeExternalBaseURL(externalBaseURL)
		if err != nil || baseURL == "" {
			return false
		}
		base, err := neturl.Parse(baseURL)
		if err != nil || strings.TrimSpace(base.Host) == "" {
			return false
		}
		return strings.EqualFold(strings.TrimSpace(u.Scheme), strings.TrimSpace(base.Scheme)) &&
			strings.EqualFold(strings.TrimSpace(u.Host), strings.TrimSpace(base.Host))
	}

	return strings.EqualFold(strings.TrimSpace(u.Scheme), EffectiveRequestScheme(r)) &&
		strings.EqualFold(strings.TrimSpace(u.Host), strings.TrimSpace(EffectiveRequestHost(r)))
}

func EffectiveRequestScheme(r *http.Request) string {
	if r == nil {
		return "http"
	}
	if IsTrustedForwardedSource(NormalizedRemoteAddrIP(r.RemoteAddr)) {
		switch strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))) {
		case "https":
			return "https"
		case "http":
			return "http"
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func EffectiveRequestHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	host := strings.TrimSpace(r.Host)
	if IsTrustedForwardedSource(NormalizedRemoteAddrIP(r.RemoteAddr)) {
		if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
			host = forwardedHost
		}
	}
	return host
}

func HostWithoutPort(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(raw, "[]")
}

func NormalizedRemoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	return strings.Trim(host, "[]")
}

func IsTrustedForwardedSource(ipString string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipString))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func IsLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
