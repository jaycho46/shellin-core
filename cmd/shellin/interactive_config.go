// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"log"
	"strings"
	"time"
)

func parseInteractiveConfig() config {
	cfg := config{}
	defaultAllowInsecureBool := parseBoolDefault(defaultAllowInsecureURL, false)
	defaultHUDBool := parseBoolDefault(defaultHUD, true)

	cfg.ControlPlaneURL = strings.TrimSpace(defaultControlPlaneURL)
	setFilteredUsage(flag.CommandLine, map[string]bool{
		"user-key": true,
	})
	flag.StringVar(&cfg.AuthProfilePath, "auth-profile", defaultAuthProfilePath(), "Auth profile path used by login/logout and control-plane auth cache")
	flag.StringVar(&cfg.Shell, "shell", defaultShellFlagValue(), "shell to launch in PTY")
	flag.DurationVar(&cfg.AnswerTimeout, "answer-timeout", parseDurationDefault(defaultAnswerTimeout, 10*time.Minute), "How long to wait for a viewer answer before returning to local-only mode")
	flag.StringVar(&cfg.UserKey, "user-key", strings.TrimSpace(defaultUserKey), "User key used to request control-plane auth")
	flag.BoolVar(&cfg.HUD, "hud", defaultHUDBool, "Show bottom HUD in interactive terminal")
	flag.Parse()

	if cfg.ControlPlaneURL == "" {
		return cfg
	}
	controlPlaneURL, err := normalizeServerURL(cfg.ControlPlaneURL, defaultAllowInsecureBool)
	if err != nil {
		log.Fatalf("invalid control plane URL: %v", err)
	}
	cfg.ControlPlaneURL = controlPlaneURL
	return cfg
}
