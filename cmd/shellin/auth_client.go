// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jaycho46/shellin-core/protocol"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

type controlPlaneAuthSession struct {
	AccessToken           string
	AccessTokenExpiresAt  string
	RefreshToken          string
	RefreshTokenExpiresAt string
}

type controlPlaneAuthUpdateMode uint8

const (
	controlPlaneAuthUnchanged controlPlaneAuthUpdateMode = iota
	controlPlaneAuthReloaded
	controlPlaneAuthRefreshed
)

func resolveControlPlaneAuth(cfg config) (controlPlaneAuthSession, string, error) {
	if cfg.ControlPlaneURL == "" {
		return controlPlaneAuthSession{}, "", errors.New("control plane URL is not configured in this build")
	}

	if cfg.UserKey != "" {
		initialAuth, err := exchangeControlPlaneAuth(cfg)
		if err != nil {
			return controlPlaneAuthSession{}, "", err
		}
		return initialAuth, "", nil
	}

	initialAuth, err := loadControlPlaneAuthFromProfile(cfg.ControlPlaneURL, cfg.AuthProfilePath)
	if err != nil {
		return controlPlaneAuthSession{}, "", fmt.Errorf("%w (run `%s login <key>`)", err, cliCommandName)
	}
	return initialAuth, cfg.AuthProfilePath, nil
}

func ensureControlPlanePreflight(cfg config, authSession controlPlaneAuthSession, profilePath string) (protocol.SessionStatusResponse, controlPlaneAuthSession, error) {
	status, finalAuth, err := requestSessionStatusWithAuthSession(cfg, authSession, profilePath)
	if err != nil {
		return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, err
	}
	if err := persistControlPlaneAuthUpdate(profilePath, cfg.ControlPlaneURL, authSession, finalAuth); err != nil {
		log.Printf("warning: failed to update auth profile after auth refresh: %v", err)
	}
	if err := validateControlPlanePreflightStatus(status); err != nil {
		return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, err
	}
	return status, finalAuth, nil
}

func validateControlPlanePreflightStatus(status protocol.SessionStatusResponse) error {
	if !sessionStatusAllowsRelay(status) {
		return &subscriptionInactiveError{
			status: status.SubscriptionStatus,
		}
	}
	if status.ActiveSessions >= status.MaxSessions {
		return fmt.Errorf(
			"session limit reached: %d active / %d max",
			status.ActiveSessions,
			status.MaxSessions,
		)
	}
	return nil
}

func sessionStatusAllowsRelay(status protocol.SessionStatusResponse) bool {
	switch strings.ToLower(strings.TrimSpace(status.SubscriptionStatus)) {
	case "", "active", "grace":
		return true
	default:
		return false
	}
}
