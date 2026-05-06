// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

const strongTestSecret = "abcdefghijklmnopqrstuvwxyz0123456789"

func TestValidateSuccess(t *testing.T) {
	const secret = strongTestSecret
	now := time.Unix(1_800_000_000, 0).UTC()
	token, err := signedToken(secret, map[string]any{
		"sub":          "user_123",
		"iss":          "billing",
		"aud":          "webrtc.signal",
		"exp":          now.Add(10 * time.Minute).Unix(),
		"max_sessions": 2,
		"jti":          "job-1",
		"role":         "agent",
		"session_id":   "sess_123",
	})
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	v, err := NewValidator(secret, "billing", "webrtc.signal")
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	v.now = func() time.Time { return now }
	v.clockSkew = 0

	claims, err := v.Validate(token)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if claims.Subject != "user_123" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
	if claims.MaxSessions != 2 {
		t.Fatalf("unexpected max sessions: %d", claims.MaxSessions)
	}
	if claims.Role != "agent" || claims.SessionID != "sess_123" {
		t.Fatalf("unexpected role/session: %+v", claims)
	}
}

func TestValidateExpired(t *testing.T) {
	const secret = strongTestSecret
	now := time.Unix(1_800_000_000, 0).UTC()
	token, err := signedToken(secret, map[string]any{
		"sub": "user_123",
		"aud": []string{"control.grants"},
		"exp": now.Add(-1 * time.Minute).Unix(),
		"jti": "job-2",
	})
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	v, err := NewValidator(secret, "", "control.grants")
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	v.now = func() time.Time { return now }
	v.clockSkew = 0

	_, err = v.Validate(token)
	if err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestValidateAllowsMissingMaxSessions(t *testing.T) {
	const secret = strongTestSecret
	now := time.Unix(1_800_000_000, 0).UTC()
	token, err := signedToken(secret, map[string]any{
		"sub": "user_123",
		"aud": "control.grants",
		"exp": now.Add(10 * time.Minute).Unix(),
		"jti": "job-3",
	})
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	v, err := NewValidator(secret, "", "control.grants")
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	v.now = func() time.Time { return now }
	v.clockSkew = 0

	claims, err := v.Validate(token)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if claims.MaxSessions != 0 {
		t.Fatalf("expected max_sessions to stay optional, got %d", claims.MaxSessions)
	}
}

func TestValidateRequiresJTI(t *testing.T) {
	const secret = strongTestSecret
	now := time.Unix(1_800_000_000, 0).UTC()
	token, err := signedToken(secret, map[string]any{
		"sub": "user_123",
		"aud": "control.grants",
		"exp": now.Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	v, err := NewValidator(secret, "", "control.grants")
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	v.now = func() time.Time { return now }
	v.clockSkew = 0

	if _, err := v.Validate(token); err == nil {
		t.Fatal("expected missing jti to fail")
	}
}

func TestValidateNotYetValid(t *testing.T) {
	const secret = strongTestSecret
	now := time.Unix(1_800_000_000, 0).UTC()
	token, err := signedToken(secret, map[string]any{
		"sub": "user_123",
		"aud": "control.grants",
		"exp": now.Add(10 * time.Minute).Unix(),
		"nbf": now.Add(5 * time.Minute).Unix(),
		"jti": "job-4",
	})
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}

	v, err := NewValidator(secret, "", "control.grants")
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}
	v.now = func() time.Time { return now }
	v.clockSkew = 0

	_, err = v.Validate(token)
	if err != ErrNotYetValid {
		t.Fatalf("expected ErrNotYetValid, got %v", err)
	}
}

func TestNewValidatorRejectsWeakSecret(t *testing.T) {
	t.Parallel()

	if _, err := NewValidator("short-secret", "", "control.grants"); err == nil {
		t.Fatal("expected weak secret to be rejected")
	}
	if _, err := NewValidator("replace-with-at-least-32-byte-random-secret", "", "control.grants"); err == nil {
		t.Fatal("expected placeholder secret to be rejected")
	}
}

func TestSignHS256RejectsWeakSecret(t *testing.T) {
	t.Parallel()

	_, err := SignHS256("short-secret", map[string]any{"sub": "user_123"})
	if err == nil {
		t.Fatal("expected weak signing secret to be rejected")
	}
}

func signedToken(secret string, claims map[string]any) (string, error) {
	headerJSON, err := json.Marshal(map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := header + "." + payload

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}
