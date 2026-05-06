// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"errors"
	"time"
)

var (
	ErrMalformedToken = errors.New("malformed grant token")
	ErrBadSignature   = errors.New("invalid grant token signature")
	ErrExpired        = errors.New("grant token expired")
	ErrNotYetValid    = errors.New("grant token not yet valid")
	ErrInvalidClaim   = errors.New("invalid grant token claim")
)

const MinHMACSecretBytes = 32

type Claims struct {
	Subject     string
	Issuer      string
	Audience    []string
	ExpiresAt   time.Time
	NotBefore   time.Time
	MaxSessions int
	TokenID     string
	Role        string
	SessionID   string
}

type Validator struct {
	secret    []byte
	issuer    string
	audience  string
	clockSkew time.Duration
	now       func() time.Time
}
