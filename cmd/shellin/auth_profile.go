// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type authProfile struct {
	ControlPlaneURL       string `json:"control_plane_url"`
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at,omitempty"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at,omitempty"`
	UpdatedAt             string `json:"updated_at,omitempty"`
}

func persistControlPlaneAuthUpdate(profilePath, controlPlaneURL string, previous, next controlPlaneAuthSession) error {
	if strings.TrimSpace(profilePath) == "" || sameControlPlaneAuthSession(previous, next) {
		return nil
	}
	return saveAuthProfile(profilePath, authProfile{
		ControlPlaneURL:       controlPlaneURL,
		AccessToken:           next.AccessToken,
		AccessTokenExpiresAt:  next.AccessTokenExpiresAt,
		RefreshToken:          next.RefreshToken,
		RefreshTokenExpiresAt: next.RefreshTokenExpiresAt,
		UpdatedAt:             time.Now().UTC().Format(time.RFC3339),
	})
}

func loadControlPlaneAuthFromProfile(controlPlaneURL, profilePath string) (controlPlaneAuthSession, error) {
	profile, err := loadAuthProfile(profilePath)
	if err != nil {
		return controlPlaneAuthSession{}, fmt.Errorf("load auth profile failed: %w", err)
	}
	profileBase := strings.TrimSpace(strings.TrimRight(profile.ControlPlaneURL, "/"))
	controlBase := strings.TrimSpace(strings.TrimRight(controlPlaneURL, "/"))
	if profileBase != "" && !strings.EqualFold(profileBase, controlBase) {
		return controlPlaneAuthSession{}, fmt.Errorf("auth profile control-plane mismatch: profile=%s current=%s", profileBase, controlBase)
	}

	auth := controlPlaneAuthSession{
		AccessToken:           strings.TrimSpace(profile.AccessToken),
		AccessTokenExpiresAt:  strings.TrimSpace(profile.AccessTokenExpiresAt),
		RefreshToken:          strings.TrimSpace(profile.RefreshToken),
		RefreshTokenExpiresAt: strings.TrimSpace(profile.RefreshTokenExpiresAt),
	}
	if auth.AccessToken == "" || auth.RefreshToken == "" {
		return controlPlaneAuthSession{}, errors.New("auth profile is missing tokens")
	}
	return auth, nil
}

func loadUpdatedControlPlaneAuthFromProfile(controlPlaneURL, profilePath string, current controlPlaneAuthSession) (controlPlaneAuthSession, bool, error) {
	if strings.TrimSpace(profilePath) == "" {
		return controlPlaneAuthSession{}, false, nil
	}
	latest, err := loadControlPlaneAuthFromProfile(controlPlaneURL, profilePath)
	if err != nil {
		return controlPlaneAuthSession{}, false, err
	}
	if sameControlPlaneAuthSession(latest, current) {
		return controlPlaneAuthSession{}, false, nil
	}
	return latest, true, nil
}

func sameControlPlaneAuthSession(a, b controlPlaneAuthSession) bool {
	return strings.TrimSpace(a.AccessToken) == strings.TrimSpace(b.AccessToken) &&
		strings.TrimSpace(a.AccessTokenExpiresAt) == strings.TrimSpace(b.AccessTokenExpiresAt) &&
		strings.TrimSpace(a.RefreshToken) == strings.TrimSpace(b.RefreshToken) &&
		strings.TrimSpace(a.RefreshTokenExpiresAt) == strings.TrimSpace(b.RefreshTokenExpiresAt)
}

func loadAuthProfile(path string) (authProfile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return authProfile{}, errors.New("auth profile path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return authProfile{}, err
	}
	var profile authProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return authProfile{}, err
	}
	return profile, nil
}

func saveAuthProfile(path string, profile authProfile) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("auth profile path is required")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	payload, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(payload, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
