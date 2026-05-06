// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	cliCommandName = "shellin"
)

// Built-in runtime defaults.
// You can override these at build time with:
//
//	go build -ldflags "-X main.defaultControlPlaneURL=... ..."
var (
	defaultControlPlaneURL  = "https://api.shellin.dev"
	defaultShellPath        = "/bin/zsh"
	defaultAnswerTimeout    = "10m"
	defaultUserKey          = ""
	defaultAllowInsecureURL = "false"
	defaultHUD              = "true"
)

type config struct {
	ControlPlaneURL string
	AuthProfilePath string
	Shell           string
	AnswerTimeout   time.Duration
	UserKey         string
	HUD             bool
}

func normalizeServerURL(raw string, allowInsecureHTTP bool) (string, error) {
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", errors.New("missing host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("scheme must be http or https")
	}
	if u.Scheme == "http" && !allowInsecureHTTP && !isLoopbackHost(u.Hostname()) {
		return "", errors.New("refusing non-loopback http URL; use https")
	}
	return raw, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func parseBoolDefault(raw string, fallback bool) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func parseDurationDefault(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func defaultAuthProfilePath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(cfgDir) == "" {
		return ".shellin-auth.json"
	}
	return filepath.Join(cfgDir, cliCommandName, "auth.json")
}

func defaultShellFlagValue() string {
	return resolveDefaultShellPath(defaultShellPath, os.Getenv("SHELL"), runtime.GOOS)
}

func resolveDefaultShellPath(preferred, envShell, goos string) string {
	candidates := []string{preferred, envShell}
	switch goos {
	case "darwin":
		candidates = append(candidates, "/bin/zsh", "zsh", "/bin/bash", "bash", "/bin/sh", "sh")
	case "linux":
		candidates = append(candidates, "/bin/bash", "bash", "/bin/sh", "sh")
	default:
		candidates = append(candidates, "/bin/sh", "sh")
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		resolved, err := exec.LookPath(candidate)
		if err == nil {
			return resolved
		}
	}

	switch goos {
	case "darwin":
		return "/bin/zsh"
	default:
		return "/bin/sh"
	}
}
