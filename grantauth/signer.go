// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
)

// SignHS256 builds and signs a JWT using the provided claims map.
func SignHS256(secret string, claims map[string]any) (string, error) {
	secretBytes, err := hmacSecretBytes(secret)
	if err != nil {
		return "", err
	}
	if claims == nil {
		return "", errors.New("claims are required")
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := encodedHeader + "." + encodedClaims

	mac := hmac.New(sha256.New, secretBytes)
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}
