// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (v *Validator) Validate(token string) (*Claims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, ErrMalformedToken
	}

	headerPayload := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrMalformedToken
	}

	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(headerPayload))
	expectedSig := mac.Sum(nil)
	if len(sig) != len(expectedSig) || subtle.ConstantTimeCompare(sig, expectedSig) != 1 {
		return nil, ErrBadSignature
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrMalformedToken
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrMalformedToken
	}
	if !strings.EqualFold(header.Alg, "HS256") {
		return nil, fmt.Errorf("%w: unsupported alg", ErrInvalidClaim)
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrMalformedToken
	}

	var raw struct {
		Sub         string          `json:"sub"`
		Iss         string          `json:"iss"`
		Aud         json.RawMessage `json:"aud"`
		Exp         json.Number     `json:"exp"`
		Nbf         json.Number     `json:"nbf"`
		JTI         string          `json:"jti"`
		MaxSessions int             `json:"max_sessions"`
		Role        string          `json:"role"`
		SessionID   string          `json:"session_id"`
	}
	dec := json.NewDecoder(strings.NewReader(string(payloadBytes)))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, ErrMalformedToken
	}

	if strings.TrimSpace(raw.Sub) == "" {
		return nil, fmt.Errorf("%w: missing sub", ErrInvalidClaim)
	}
	if strings.TrimSpace(raw.JTI) == "" {
		return nil, fmt.Errorf("%w: missing jti", ErrInvalidClaim)
	}

	expUnix, err := parseUnix(raw.Exp)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid exp", ErrInvalidClaim)
	}
	exp := time.Unix(expUnix, 0).UTC()
	now := v.now().UTC()
	if now.After(exp.Add(v.clockSkew)) {
		return nil, ErrExpired
	}

	var nbf time.Time
	if raw.Nbf != "" {
		nbfUnix, err := parseUnix(raw.Nbf)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid nbf", ErrInvalidClaim)
		}
		nbf = time.Unix(nbfUnix, 0).UTC()
		if now.Before(nbf.Add(-v.clockSkew)) {
			return nil, ErrNotYetValid
		}
	}

	if v.issuer != "" && !strings.EqualFold(v.issuer, strings.TrimSpace(raw.Iss)) {
		return nil, fmt.Errorf("%w: issuer mismatch", ErrInvalidClaim)
	}

	aud, err := parseAudience(raw.Aud)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid aud", ErrInvalidClaim)
	}
	if v.audience != "" && !containsAudience(aud, v.audience) {
		return nil, fmt.Errorf("%w: audience mismatch", ErrInvalidClaim)
	}

	return &Claims{
		Subject:     strings.TrimSpace(raw.Sub),
		Issuer:      strings.TrimSpace(raw.Iss),
		Audience:    aud,
		ExpiresAt:   exp,
		NotBefore:   nbf,
		MaxSessions: raw.MaxSessions,
		TokenID:     strings.TrimSpace(raw.JTI),
		Role:        strings.TrimSpace(raw.Role),
		SessionID:   strings.TrimSpace(raw.SessionID),
	}, nil
}
