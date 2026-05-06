// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateHMACSecret(secret string) error {
	_, err := hmacSecretBytes(secret)
	return err
}

func hmacSecretBytes(secret string) ([]byte, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("secret is required")
	}
	if len([]byte(secret)) < MinHMACSecretBytes {
		return nil, fmt.Errorf("secret must be at least %d bytes", MinHMACSecretBytes)
	}
	if hmacSecretLooksLikePlaceholder(secret) {
		return nil, errors.New("secret must not be a placeholder or development value")
	}
	return []byte(secret), nil
}

func hmacSecretLooksLikePlaceholder(secret string) bool {
	secret = strings.ToLower(strings.TrimSpace(secret))
	for _, marker := range []string{"replace", "change-me", "dev-secret", "test-secret"} {
		if strings.Contains(secret, marker) {
			return true
		}
	}
	return false
}
