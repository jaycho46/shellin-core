// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jaycho46/shellin-core/buildinfo"
	"github.com/jaycho46/shellin-core/controlplane"
	"github.com/jaycho46/shellin-core/grantauth"
	"github.com/jaycho46/shellin-core/signaling"
)

type serverDeps struct {
	store              *controlplane.UserKeyStore
	refreshStore       *controlplane.RefreshTokenStore
	loginKeyStore      *controlplane.LoginKeyStore
	authFailureLimiter *authFailureLimiter
	agentRegistry      *controlplane.AgentRegistry
	signalHub          *controlplane.SignalHub
	signalReplayGuard  *grantauth.ReplayGuard
	accessValidator    *grantauth.Validator
	signalValidator    *grantauth.Validator
}

type controlPlaneServer struct {
	cfg  config
	deps serverDeps
}

func newServerDeps(cfg config, store *controlplane.UserKeyStore) (serverDeps, error) {
	if store == nil {
		return serverDeps{}, fmt.Errorf("user key store is required")
	}
	accessValidator, err := grantauth.NewValidator(cfg.GrantHMACSecret, cfg.GrantIssuer, cfg.AccessAudience)
	if err != nil {
		return serverDeps{}, fmt.Errorf("failed to initialize access validator: %w", err)
	}
	signalValidator, err := grantauth.NewValidator(cfg.GrantHMACSecret, cfg.GrantIssuer, cfg.SignalAudience)
	if err != nil {
		return serverDeps{}, fmt.Errorf("failed to initialize signal validator: %w", err)
	}
	return serverDeps{
		store:              store,
		refreshStore:       controlplane.NewRefreshTokenStore(),
		loginKeyStore:      controlplane.NewLoginKeyStore(),
		authFailureLimiter: newAuthFailureLimiter(cfg.AuthFailureLimit, cfg.AuthFailureWindow),
		agentRegistry:      controlplane.NewAgentRegistry(),
		signalHub:          controlplane.NewSignalHub(),
		signalReplayGuard:  grantauth.NewReplayGuard(),
		accessValidator:    accessValidator,
		signalValidator:    signalValidator,
	}, nil
}

func NewServer(cfg config, deps serverDeps) http.Handler {
	s := &controlPlaneServer{cfg: cfg, deps: deps}
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return withSecurityHeaders(mux)
}

func (s *controlPlaneServer) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/version", buildinfo.Handler("shellin-control-plane"))
	mux.HandleFunc("/v1/auth/exchange", s.handleAuthExchange)
	mux.HandleFunc("/v1/auth/refresh", s.handleAuthRefresh)
	mux.HandleFunc("/v1/login-keys", s.handleLoginKeys)
	mux.HandleFunc("/v1/auth/device-login", s.handleDeviceLogin)
	mux.HandleFunc("/v1/agents/connect", s.handleAgentConnect)
	mux.HandleFunc("/v1/agents/attach", s.handleAgentAttach)
	mux.HandleFunc("/v1/agents/signal", signaling.Handler(signaling.Options{
		Validator:   s.deps.signalValidator,
		ReplayStore: s.deps.signalReplayGuard,
		Registry:    s.deps.agentRegistry,
		StaleAfter:  s.cfg.AgentHeartbeatTimeout,
		Hub:         s.deps.signalHub,
		ExternalURL: s.cfg.ExternalBaseURL,
	}))
	mux.HandleFunc("/v1/agents/register", s.handleAgentRegister)
	mux.HandleFunc("/v1/agents/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("/v1/agents/unregister", s.handleAgentUnregister)
	mux.HandleFunc("/v1/agents/terminate-all", s.handleAgentTerminateAll)
	mux.HandleFunc("/v1/agents", s.handleAgentList)
	mux.HandleFunc("/v1/status", s.handleStatus)
}

func (s *controlPlaneServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func startCleanupLoop(cfg config, deps serverDeps) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			deps.refreshStore.Cleanup()
			deps.loginKeyStore.Cleanup()
			deps.agentRegistry.Cleanup(cfg.AgentHeartbeatTimeout)
		}
	}()
}
