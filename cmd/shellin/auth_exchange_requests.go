// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jaycho46/shellin-core/protocol"
)

func exchangeControlPlaneAuth(cfg config) (controlPlaneAuthSession, error) {
	reqBody, err := json.Marshal(protocol.ExchangeAuthRequest{UserKey: cfg.UserKey})
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	endpoint := strings.TrimRight(cfg.ControlPlaneURL, "/") + "/v1/auth/exchange"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return controlPlaneAuthSession{}, fmt.Errorf("control plane exchange status %d: %s", resp.StatusCode, string(body))
	}

	var out protocol.ExchangeAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return controlPlaneAuthSession{}, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return controlPlaneAuthSession{}, errors.New("control plane exchange returned empty access_token")
	}
	if strings.TrimSpace(out.RefreshToken) == "" {
		return controlPlaneAuthSession{}, errors.New("control plane exchange returned empty refresh_token")
	}
	return controlPlaneAuthSession{
		AccessToken:           strings.TrimSpace(out.AccessToken),
		AccessTokenExpiresAt:  strings.TrimSpace(out.AccessTokenExpiresAt),
		RefreshToken:          strings.TrimSpace(out.RefreshToken),
		RefreshTokenExpiresAt: strings.TrimSpace(out.RefreshTokenExpiresAt),
	}, nil
}

func refreshControlPlaneAuth(cfg config, refreshToken string) (controlPlaneAuthSession, error) {
	reqBody, err := json.Marshal(protocol.RefreshAuthRequest{RefreshToken: refreshToken})
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	endpoint := strings.TrimRight(cfg.ControlPlaneURL, "/") + "/v1/auth/refresh"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return controlPlaneAuthSession{}, fmt.Errorf("control plane refresh status %d: %s", resp.StatusCode, string(body))
	}

	var out protocol.RefreshAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return controlPlaneAuthSession{}, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return controlPlaneAuthSession{}, errors.New("control plane refresh returned empty access_token")
	}
	if strings.TrimSpace(out.RefreshToken) == "" {
		return controlPlaneAuthSession{}, errors.New("control plane refresh returned empty refresh_token")
	}
	return controlPlaneAuthSession{
		AccessToken:           strings.TrimSpace(out.AccessToken),
		AccessTokenExpiresAt:  strings.TrimSpace(out.AccessTokenExpiresAt),
		RefreshToken:          strings.TrimSpace(out.RefreshToken),
		RefreshTokenExpiresAt: strings.TrimSpace(out.RefreshTokenExpiresAt),
	}, nil
}

func deviceLoginWithKey(controlPlaneURL, loginKey string) (controlPlaneAuthSession, error) {
	loginKey = strings.TrimSpace(loginKey)
	if loginKey == "" {
		return controlPlaneAuthSession{}, errors.New("login key is required")
	}

	reqBody, err := json.Marshal(protocol.DeviceLoginRequest{LoginKey: loginKey})
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	endpoint := strings.TrimRight(controlPlaneURL, "/") + "/v1/auth/device-login"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return controlPlaneAuthSession{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return controlPlaneAuthSession{}, fmt.Errorf("control plane device login status %d: %s", resp.StatusCode, string(body))
	}

	var out protocol.ExchangeAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return controlPlaneAuthSession{}, err
	}
	if strings.TrimSpace(out.AccessToken) == "" || strings.TrimSpace(out.RefreshToken) == "" {
		return controlPlaneAuthSession{}, errors.New("device login returned empty tokens")
	}
	return controlPlaneAuthSession{
		AccessToken:           strings.TrimSpace(out.AccessToken),
		AccessTokenExpiresAt:  strings.TrimSpace(out.AccessTokenExpiresAt),
		RefreshToken:          strings.TrimSpace(out.RefreshToken),
		RefreshTokenExpiresAt: strings.TrimSpace(out.RefreshTokenExpiresAt),
	}, nil
}
