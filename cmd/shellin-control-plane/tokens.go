// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jaycho46/shellin-core/controlplane"
	"github.com/jaycho46/shellin-core/grantauth"
	"github.com/jaycho46/shellin-core/protocol"
)

type signedTokenOptions struct {
	Subject     string
	MaxSessions int
	Role        string
	SessionID   string
}

func seedUserKeys(cfg config, store *controlplane.UserKeyStore) error {
	specs := []string{cfg.UserKeys}
	if strings.TrimSpace(cfg.UserKeysFile) != "" {
		data, err := os.ReadFile(strings.TrimSpace(cfg.UserKeysFile))
		if err != nil {
			return err
		}
		specs = append(specs, strings.ReplaceAll(string(data), "\n", ","))
	}
	for _, spec := range specs {
		entries, err := controlplane.ParseUserKeySpec(spec, cfg.DefaultMaxSessions)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := store.Add(entry.UserKey, entry.Entitlement); err != nil {
				return err
			}
		}
	}
	return nil
}

func issueAuthTokens(cfg config, refreshStore controlplane.RefreshTokenStoreAPI, subject string, maxSessions int) (protocol.ExchangeAuthResponse, error) {
	accessToken, accessExp, err := issueSignedToken(cfg.GrantHMACSecret, cfg.GrantIssuer, cfg.AccessAudience, cfg.AccessTTL, signedTokenOptions{
		Subject:     subject,
		MaxSessions: maxSessions,
	})
	if err != nil {
		return protocol.ExchangeAuthResponse{}, err
	}
	refreshToken, refreshExp, err := refreshStore.Issue(subject, cfg.RefreshTTL)
	if err != nil {
		return protocol.ExchangeAuthResponse{}, err
	}
	return protocol.ExchangeAuthResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExp.Format(time.RFC3339),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExp.Format(time.RFC3339),
	}, nil
}

func issueSignedToken(secret, issuer, audience string, ttl time.Duration, opts signedTokenOptions) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	claims := map[string]any{
		"sub":          strings.TrimSpace(opts.Subject),
		"aud":          audience,
		"exp":          expiresAt.Unix(),
		"iat":          now.Unix(),
		"nbf":          now.Unix(),
		"jti":          uuid.NewString(),
		"max_sessions": opts.MaxSessions,
	}
	if issuer != "" {
		claims["iss"] = issuer
	}
	if role := strings.TrimSpace(opts.Role); role != "" {
		claims["role"] = role
	}
	if sessionID := strings.TrimSpace(opts.SessionID); sessionID != "" {
		claims["session_id"] = sessionID
	}
	token, err := grantauth.SignHS256(secret, claims)
	return token, expiresAt, err
}

func validateAccessRequest(r *http.Request, validator *grantauth.Validator) (*grantauth.Claims, error) {
	token := bearerToken(r)
	if token == "" {
		return nil, errors.New("missing bearer token")
	}
	return validator.Validate(token)
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(strings.ToLower(auth), strings.ToLower(prefix)) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

func agentStatusResponse(item controlplane.AgentStatus) protocol.AgentStatus {
	return protocol.AgentStatus{
		SessionID:  item.SessionID,
		AgentLabel: item.AgentLabel,
		Transport:  protocol.Transport(item.Transport),
		CreatedAt:  item.CreatedAt.Format(time.RFC3339),
		LastSeenAt: item.LastSeenAt.Format(time.RFC3339),
	}
}

func isEntitlementUsable(entitlement controlplane.UserEntitlement) bool {
	now := time.Now().UTC()
	return strings.TrimSpace(entitlement.Subject) != "" &&
		entitlement.MaxSessions > 0 &&
		entitlement.Active &&
		(entitlement.ExpiresAt.IsZero() || entitlement.ExpiresAt.After(now))
}

func writeAgentRegistryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, controlplane.ErrAgentSessionLimitReached):
		http.Error(w, "session limit reached", http.StatusConflict)
	case errors.Is(err, controlplane.ErrAgentSessionTerminated):
		http.Error(w, "agent session terminated", http.StatusGone)
	default:
		http.Error(w, "invalid request", http.StatusBadRequest)
	}
}
