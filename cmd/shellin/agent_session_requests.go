// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/jaycho46/shellin-core/protocol"
)

func registerAgentSession(cfg config, accessToken, sessionID, agentLabel string) error {
	if strings.TrimSpace(accessToken) == "" {
		return errors.New("access token is required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id is required")
	}

	reqBody, err := json.Marshal(protocol.AgentRegisterRequest{
		SessionID:  sessionID,
		AgentLabel: normalizeAgentLabel(agentLabel),
		Transport:  protocol.TransportWebRTCTURN,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.ControlPlaneURL, "/")+"/v1/agents/register", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &controlPlaneStatusError{
			Operation:  "register",
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	return nil
}

func heartbeatAgentSession(cfg config, accessToken, sessionID, agentLabel string) error {
	reqBody, err := json.Marshal(protocol.AgentHeartbeatRequest{
		SessionID:  strings.TrimSpace(sessionID),
		AgentLabel: normalizeAgentLabel(agentLabel),
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.ControlPlaneURL, "/")+"/v1/agents/heartbeat", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return &controlPlaneStatusError{
			Operation:  "heartbeat",
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	return nil
}

func unregisterAgentSession(cfg config, accessToken, sessionID string) error {
	if strings.TrimSpace(accessToken) == "" {
		return nil
	}
	reqBody, err := json.Marshal(protocol.AgentUnregisterRequest{
		SessionID: strings.TrimSpace(sessionID),
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.ControlPlaneURL, "/")+"/v1/agents/unregister", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return &controlPlaneStatusError{
			Operation:  "unregister",
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	return nil
}
