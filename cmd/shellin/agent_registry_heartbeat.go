// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"log"
	"strings"
	"sync"
	"time"
)

type agentRegistryHeartbeat struct {
	cfg         config
	profilePath string
	sessionID   string
	agentLabel  func() string
	onFatal     func(error)

	tokenMu      sync.RWMutex
	accessToken  string
	refreshToken string

	stop   chan struct{}
	update chan struct{}
	wg     sync.WaitGroup
}

func newAgentRegistryHeartbeat(
	cfg config,
	profilePath string,
	sessionID string,
	initialAuth controlPlaneAuthSession,
	agentLabel func() string,
	onFatal func(error),
) *agentRegistryHeartbeat {
	return &agentRegistryHeartbeat{
		cfg:          cfg,
		profilePath:  strings.TrimSpace(profilePath),
		sessionID:    strings.TrimSpace(sessionID),
		agentLabel:   agentLabel,
		onFatal:      onFatal,
		accessToken:  strings.TrimSpace(initialAuth.AccessToken),
		refreshToken: strings.TrimSpace(initialAuth.RefreshToken),
	}
}

func (h *agentRegistryHeartbeat) Start() error {
	if h == nil || strings.TrimSpace(h.cfg.ControlPlaneURL) == "" || h.accessToken == "" {
		return nil
	}
	if err := registerAgentSession(h.cfg, h.accessToken, h.sessionID, h.currentAgentLabel()); err != nil {
		return err
	}

	h.stop = make(chan struct{})
	h.update = make(chan struct{}, 1)
	h.wg.Add(1)
	go h.run()
	return nil
}

func (h *agentRegistryHeartbeat) Stop() {
	if h == nil || h.stop == nil {
		return
	}
	close(h.stop)
	h.wg.Wait()

	h.tokenMu.RLock()
	finalToken := h.accessToken
	h.tokenMu.RUnlock()
	if err := unregisterAgentSession(h.cfg, finalToken, h.sessionID); err != nil {
		log.Printf("warning: failed to unregister agent listing: %v", err)
	}
}

func (h *agentRegistryHeartbeat) NotifyLabelChanged() {
	if h == nil || h.update == nil {
		return
	}
	select {
	case h.update <- struct{}{}:
	default:
	}
}

func (h *agentRegistryHeartbeat) ApplyAuthUpdate(next controlPlaneAuthSession, mode controlPlaneAuthUpdateMode) {
	switch mode {
	case controlPlaneAuthReloaded:
		h.setAuth(next)
	case controlPlaneAuthRefreshed:
		h.persistAuth(next)
	}
}

func (h *agentRegistryHeartbeat) run() {
	defer h.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	retryCount := 0
	for {
		select {
		case <-h.stop:
			return
		case <-h.update:
		case <-ticker.C:
		}

		currentAuth := h.currentAuth()
		nextAuth, mode, err := heartbeatAgentSessionWithAuthRecovery(
			h.cfg,
			h.profilePath,
			h.sessionID,
			h.currentAgentLabel(),
			currentAuth,
		)
		if err != nil {
			retryCount++
			log.Printf("warning: connection error (%d): %v", retryCount, err)
			if isNonRetryableControlPlaneError(err) {
				log.Printf("warning: stopping control-plane heartbeat retries after authorization fallback failure")
				if h.onFatal != nil {
					h.onFatal(err)
				}
				return
			}
			continue
		}
		retryCount = 0
		h.ApplyAuthUpdate(nextAuth, mode)
	}
}

func (h *agentRegistryHeartbeat) currentAuth() controlPlaneAuthSession {
	h.tokenMu.RLock()
	defer h.tokenMu.RUnlock()
	return controlPlaneAuthSession{
		AccessToken:  h.accessToken,
		RefreshToken: h.refreshToken,
	}
}

func (h *agentRegistryHeartbeat) currentAgentLabel() string {
	if h == nil || h.agentLabel == nil {
		return ""
	}
	return h.agentLabel()
}

func (h *agentRegistryHeartbeat) setAuth(next controlPlaneAuthSession) {
	h.tokenMu.Lock()
	h.accessToken = strings.TrimSpace(next.AccessToken)
	h.refreshToken = strings.TrimSpace(next.RefreshToken)
	h.tokenMu.Unlock()
}

func (h *agentRegistryHeartbeat) persistAuth(next controlPlaneAuthSession) {
	h.setAuth(next)
	if strings.TrimSpace(h.profilePath) == "" {
		return
	}
	if err := saveAuthProfile(h.profilePath, authProfile{
		ControlPlaneURL:       h.cfg.ControlPlaneURL,
		AccessToken:           next.AccessToken,
		AccessTokenExpiresAt:  next.AccessTokenExpiresAt,
		RefreshToken:          next.RefreshToken,
		RefreshTokenExpiresAt: next.RefreshTokenExpiresAt,
		UpdatedAt:             time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		log.Printf("warning: failed to update auth profile after auth refresh: %v", err)
	}
}
