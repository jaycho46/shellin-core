// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"strings"
	"time"
)

func NewValidator(secret, issuer, audience string) (*Validator, error) {
	secretBytes, err := hmacSecretBytes(secret)
	if err != nil {
		return nil, err
	}
	return &Validator{
		secret:    secretBytes,
		issuer:    strings.TrimSpace(issuer),
		audience:  strings.TrimSpace(audience),
		clockSkew: 30 * time.Second,
		now:       time.Now,
	}, nil
}
