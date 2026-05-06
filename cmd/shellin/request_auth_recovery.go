// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	"github.com/jaycho46/shellin-core/protocol"
)

func requestSessionStatusWithAuthSession(cfg config, authSession controlPlaneAuthSession, profilePath string) (protocol.SessionStatusResponse, controlPlaneAuthSession, error) {
	status, err := requestSessionStatus(cfg, authSession.AccessToken)
	if err == nil {
		return status, authSession, nil
	}
	if !isUnauthorizedError(err) || strings.TrimSpace(authSession.RefreshToken) == "" {
		return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, err
	}
	refreshed, refreshErr := refreshControlPlaneAuth(cfg, authSession.RefreshToken)
	if refreshErr != nil {
		latestAuth, changed, loadErr := loadUpdatedControlPlaneAuthFromProfile(cfg.ControlPlaneURL, profilePath, authSession)
		if loadErr != nil {
			return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, fmt.Errorf("status request failed (%v); refresh failed (%v); shared auth reload failed (%v)", err, refreshErr, loadErr)
		}
		if !changed {
			return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, fmt.Errorf("status request failed (%v); refresh failed (%v)", err, refreshErr)
		}

		status, retryErr := requestSessionStatus(cfg, latestAuth.AccessToken)
		if retryErr == nil {
			return status, latestAuth, nil
		}
		if !isUnauthorizedError(retryErr) || strings.TrimSpace(latestAuth.RefreshToken) == "" {
			return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, retryErr
		}

		refreshed, latestRefreshErr := refreshControlPlaneAuth(cfg, latestAuth.RefreshToken)
		if latestRefreshErr != nil {
			return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, fmt.Errorf("status request failed after shared auth reload (%v); refresh failed (%v)", retryErr, latestRefreshErr)
		}
		status, retryErr = requestSessionStatus(cfg, refreshed.AccessToken)
		if retryErr != nil {
			return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, retryErr
		}
		return status, refreshed, nil
	}
	status, err = requestSessionStatus(cfg, refreshed.AccessToken)
	if err != nil {
		return protocol.SessionStatusResponse{}, controlPlaneAuthSession{}, err
	}
	return status, refreshed, nil
}

func requestTerminateAllAgentsWithAuthSession(cfg config, authSession controlPlaneAuthSession, profilePath string) (protocol.AgentTerminateAllResponse, controlPlaneAuthSession, error) {
	result, err := requestTerminateAllAgents(cfg, authSession.AccessToken)
	if err == nil {
		return result, authSession, nil
	}
	if !isUnauthorizedError(err) || strings.TrimSpace(authSession.RefreshToken) == "" {
		return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, err
	}

	refreshed, refreshErr := refreshControlPlaneAuth(cfg, authSession.RefreshToken)
	if refreshErr != nil {
		latestAuth, changed, loadErr := loadUpdatedControlPlaneAuthFromProfile(cfg.ControlPlaneURL, profilePath, authSession)
		if loadErr != nil {
			return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, fmt.Errorf("cleanup request failed (%v); refresh failed (%v); shared auth reload failed (%v)", err, refreshErr, loadErr)
		}
		if !changed {
			return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, fmt.Errorf("cleanup request failed (%v); refresh failed (%v)", err, refreshErr)
		}

		result, retryErr := requestTerminateAllAgents(cfg, latestAuth.AccessToken)
		if retryErr == nil {
			return result, latestAuth, nil
		}
		if !isUnauthorizedError(retryErr) || strings.TrimSpace(latestAuth.RefreshToken) == "" {
			return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, retryErr
		}

		refreshed, latestRefreshErr := refreshControlPlaneAuth(cfg, latestAuth.RefreshToken)
		if latestRefreshErr != nil {
			return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, fmt.Errorf("cleanup request failed after shared auth reload (%v); refresh failed (%v)", retryErr, latestRefreshErr)
		}
		result, retryErr = requestTerminateAllAgents(cfg, refreshed.AccessToken)
		if retryErr != nil {
			return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, retryErr
		}
		return result, refreshed, nil
	}

	result, err = requestTerminateAllAgents(cfg, refreshed.AccessToken)
	if err != nil {
		return protocol.AgentTerminateAllResponse{}, controlPlaneAuthSession{}, err
	}
	return result, refreshed, nil
}
