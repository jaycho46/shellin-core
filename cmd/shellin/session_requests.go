// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jaycho46/shellin-core/protocol"
)

func requestSessionStatus(cfg config, accessToken string) (protocol.SessionStatusResponse, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(cfg.ControlPlaneURL, "/")+"/v1/status", nil)
	if err != nil {
		return protocol.SessionStatusResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return protocol.SessionStatusResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return protocol.SessionStatusResponse{}, fmt.Errorf("control plane status %d: %s", resp.StatusCode, string(body))
	}

	var out protocol.SessionStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return protocol.SessionStatusResponse{}, err
	}
	if out.MaxSessions <= 0 {
		return protocol.SessionStatusResponse{}, errors.New("control plane returned invalid max_sessions")
	}
	if out.ActiveSessions < 0 {
		return protocol.SessionStatusResponse{}, errors.New("control plane returned invalid active_sessions")
	}
	return out, nil
}

func requestTerminateAllAgents(cfg config, accessToken string) (protocol.AgentTerminateAllResponse, error) {
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.ControlPlaneURL, "/")+"/v1/agents/terminate-all", nil)
	if err != nil {
		return protocol.AgentTerminateAllResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return protocol.AgentTerminateAllResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return protocol.AgentTerminateAllResponse{}, &controlPlaneStatusError{
			Operation:  "cleanup",
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	var out protocol.AgentTerminateAllResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return protocol.AgentTerminateAllResponse{}, err
	}
	if out.TerminatedSessions < 0 {
		return protocol.AgentTerminateAllResponse{}, errors.New("control plane returned invalid terminated_sessions")
	}
	return out, nil
}
