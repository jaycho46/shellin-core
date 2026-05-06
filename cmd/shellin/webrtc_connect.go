// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/jaycho46/shellin-core/protocol"
)

func requestAgentConnectWithAuthSession(cfg config, authSession controlPlaneAuthSession, sessionID, agentLabel string) (protocol.AgentConnectResponse, controlPlaneAuthSession, controlPlaneAuthUpdateMode, error) {
	resp, err := requestAgentConnect(cfg, authSession.AccessToken, sessionID, agentLabel)
	if err == nil {
		return resp, authSession, controlPlaneAuthUnchanged, nil
	}
	if !isUnauthorizedError(err) {
		return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, err
	}

	if strings.TrimSpace(authSession.RefreshToken) != "" {
		refreshed, refreshErr := refreshControlPlaneAuth(cfg, authSession.RefreshToken)
		if refreshErr == nil {
			resp, retryErr := requestAgentConnect(cfg, refreshed.AccessToken, sessionID, agentLabel)
			if retryErr != nil {
				return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markConnectAuthErrorNonRetryable(retryErr)
			}
			return resp, refreshed, controlPlaneAuthRefreshed, nil
		}

		latestAuth, changed, loadErr := loadUpdatedControlPlaneAuthFromProfile(cfg.ControlPlaneURL, cfg.AuthProfilePath, authSession)
		if loadErr != nil {
			return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markConnectAuthErrorNonRetryable(
				fmt.Errorf("%w (refresh auth failed: %v; shared auth reload failed: %v)", err, refreshErr, loadErr),
			)
		}
		if !changed {
			return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markConnectAuthErrorNonRetryable(
				fmt.Errorf("%w (refresh auth failed: %v)", err, refreshErr),
			)
		}
		resp, retryErr := requestAgentConnect(cfg, latestAuth.AccessToken, sessionID, agentLabel)
		if retryErr != nil {
			return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markConnectAuthErrorNonRetryable(retryErr)
		}
		return resp, latestAuth, controlPlaneAuthReloaded, nil
	}

	latestAuth, changed, loadErr := loadUpdatedControlPlaneAuthFromProfile(cfg.ControlPlaneURL, cfg.AuthProfilePath, authSession)
	if loadErr != nil || !changed {
		return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markConnectAuthErrorNonRetryable(err)
	}
	resp, retryErr := requestAgentConnect(cfg, latestAuth.AccessToken, sessionID, agentLabel)
	if retryErr != nil {
		return protocol.AgentConnectResponse{}, controlPlaneAuthSession{}, controlPlaneAuthUnchanged, markConnectAuthErrorNonRetryable(retryErr)
	}
	return resp, latestAuth, controlPlaneAuthReloaded, nil
}

func markConnectAuthErrorNonRetryable(err error) error {
	if !isUnauthorizedError(err) {
		return err
	}
	return markControlPlaneErrorNonRetryable(err)
}

func requestAgentConnect(cfg config, accessToken, sessionID, agentLabel string) (protocol.AgentConnectResponse, error) {
	reqBody, err := json.Marshal(protocol.AgentConnectRequest{
		SessionID:  strings.TrimSpace(sessionID),
		AgentLabel: normalizeAgentLabel(agentLabel),
	})
	if err != nil {
		return protocol.AgentConnectResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.ControlPlaneURL, "/")+"/v1/agents/connect", bytes.NewReader(reqBody))
	if err != nil {
		return protocol.AgentConnectResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return protocol.AgentConnectResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return protocol.AgentConnectResponse{}, &controlPlaneStatusError{
			Operation:  "connect",
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	var out protocol.AgentConnectResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return protocol.AgentConnectResponse{}, err
	}
	if strings.TrimSpace(out.SessionID) == "" || strings.TrimSpace(out.SignalURL) == "" || strings.TrimSpace(out.SignalToken) == "" {
		return protocol.AgentConnectResponse{}, errors.New("control plane connect returned incomplete signaling bootstrap")
	}
	if out.SignalURL, err = normalizeSignalURL(out.SignalURL); err != nil {
		return protocol.AgentConnectResponse{}, fmt.Errorf("control plane connect returned invalid signal_url: %w", err)
	}
	return out, nil
}

func normalizeSignalURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", errors.New("missing host")
	}
	switch u.Scheme {
	case "wss", "ws":
	default:
		return "", errors.New("scheme must be ws or wss")
	}
	return u.String(), nil
}
