// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"crypto/sha256"
	"encoding/base64"
)

func hashOpaqueToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
