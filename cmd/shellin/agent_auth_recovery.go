// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

func heartbeatAgentSessionWithAuthRecovery(cfg config, profilePath, sessionID, agentLabel string, authSession controlPlaneAuthSession) (controlPlaneAuthSession, controlPlaneAuthUpdateMode, error) {
	tryUserKeyFallback := func(cause error) (controlPlaneAuthSession, controlPlaneAuthUpdateMode, error) {
		if strings.TrimSpace(cfg.UserKey) == "" {
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markControlPlaneErrorNonRetryable(
				fmt.Errorf("failed to refresh control-plane auth during heartbeat: %v", cause),
			)
		}

		fallbackAuth, fallbackErr := exchangeControlPlaneAuth(cfg)
		if fallbackErr != nil {
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markControlPlaneErrorNonRetryable(
				fmt.Errorf(
					"failed to refresh control-plane auth during heartbeat: %v (fallback user key exchange failed: %v)",
					cause,
					fallbackErr,
				),
			)
		}
		if hbErr := heartbeatOrRegisterAgentSession(cfg, fallbackAuth.AccessToken, sessionID, agentLabel); hbErr != nil {
			if fatalErr := terminatedSessionRuntimeError(hbErr); fatalErr != nil {
				return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fatalErr
			}
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markControlPlaneErrorNonRetryable(
				fmt.Errorf("heartbeat retry after fallback user key exchange failed: %w", hbErr),
			)
		}
		return fallbackAuth, controlPlaneAuthRefreshed, nil
	}

	err := heartbeatOrRegisterAgentSession(cfg, authSession.AccessToken, sessionID, agentLabel)
	if err == nil {
		return authSession, controlPlaneAuthUnchanged, nil
	}
	if fatalErr := terminatedSessionRuntimeError(err); fatalErr != nil {
		return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fatalErr
	}
	if !isUnauthorizedError(err) {
		return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fmt.Errorf("agent heartbeat failed: %w", err)
	}

	if strings.TrimSpace(authSession.RefreshToken) != "" {
		refreshed, refreshErr := refreshControlPlaneAuth(cfg, authSession.RefreshToken)
		if refreshErr == nil {
			if hbErr := heartbeatOrRegisterAgentSession(cfg, refreshed.AccessToken, sessionID, agentLabel); hbErr != nil {
				if fatalErr := terminatedSessionRuntimeError(hbErr); fatalErr != nil {
					return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fatalErr
				}
				return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fmt.Errorf("heartbeat retry after refresh failed: %w", hbErr)
			}
			return refreshed, controlPlaneAuthRefreshed, nil
		}

		latestAuth, changed, loadErr := loadUpdatedControlPlaneAuthFromProfile(cfg.ControlPlaneURL, profilePath, authSession)
		if loadErr != nil {
			return tryUserKeyFallback(
				fmt.Errorf("%v (shared auth reload failed: %v)", refreshErr, loadErr),
			)
		}
		if !changed {
			return tryUserKeyFallback(refreshErr)
		}

		if hbErr := heartbeatOrRegisterAgentSession(cfg, latestAuth.AccessToken, sessionID, agentLabel); hbErr == nil {
			return latestAuth, controlPlaneAuthReloaded, nil
		} else if fatalErr := terminatedSessionRuntimeError(hbErr); fatalErr != nil {
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fatalErr
		} else if !isUnauthorizedError(hbErr) {
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fmt.Errorf("agent heartbeat failed after shared auth reload: %w", hbErr)
		}

		if strings.TrimSpace(latestAuth.RefreshToken) == "" {
			return tryUserKeyFallback(refreshErr)
		}
		refreshed, latestRefreshErr := refreshControlPlaneAuth(cfg, latestAuth.RefreshToken)
		if latestRefreshErr != nil {
			return tryUserKeyFallback(latestRefreshErr)
		}
		if hbErr := heartbeatOrRegisterAgentSession(cfg, refreshed.AccessToken, sessionID, agentLabel); hbErr != nil {
			if fatalErr := terminatedSessionRuntimeError(hbErr); fatalErr != nil {
				return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fatalErr
			}
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fmt.Errorf("heartbeat retry after shared auth refresh failed: %w", hbErr)
		}
		return refreshed, controlPlaneAuthRefreshed, nil
	}

	latestAuth, changed, loadErr := loadUpdatedControlPlaneAuthFromProfile(cfg.ControlPlaneURL, profilePath, authSession)
	if loadErr != nil {
		return tryUserKeyFallback(
			fmt.Errorf("%w (shared auth reload failed: %v)", err, loadErr),
		)
	}
	if !changed {
		return tryUserKeyFallback(err)
	}
	if hbErr := heartbeatOrRegisterAgentSession(cfg, latestAuth.AccessToken, sessionID, agentLabel); hbErr != nil {
		if fatalErr := terminatedSessionRuntimeError(hbErr); fatalErr != nil {
			return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fatalErr
		}
		return controlPlaneAuthSession{}, controlPlaneAuthUnchanged, fmt.Errorf("agent heartbeat failed after shared auth reload: %w", hbErr)
	}
	return latestAuth, controlPlaneAuthReloaded, nil
}

func heartbeatOrRegisterAgentSession(cfg config, accessToken, sessionID, agentLabel string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id is required")
	}

	if err := heartbeatAgentSession(cfg, accessToken, sessionID, agentLabel); err != nil {
		if !isNotFoundError(err) {
			return err
		}
		if registerErr := registerAgentSession(cfg, accessToken, sessionID, agentLabel); registerErr != nil {
			return fmt.Errorf("agent registry entry missing; re-register failed: %w", registerErr)
		}
	}
	return nil
}
