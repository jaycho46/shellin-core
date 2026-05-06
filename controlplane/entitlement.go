// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"strings"
	"time"
)

func NormalizeEntitlement(in UserEntitlement) (UserEntitlement, error) {
	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		return UserEntitlement{}, errors.New("subject is required")
	}
	if in.MaxSessions <= 0 {
		return UserEntitlement{}, errors.New("max sessions must be > 0")
	}
	now := time.Now().UTC()
	out := UserEntitlement{
		Subject:               subject,
		MaxSessions:           in.MaxSessions,
		Active:                in.Active,
		ProductID:             strings.TrimSpace(in.ProductID),
		OriginalTransactionID: strings.TrimSpace(in.OriginalTransactionID),
		ExpiresAt:             in.ExpiresAt.UTC(),
		UpdatedAt:             now,
	}
	if in.ExpiresAt.IsZero() {
		out.ExpiresAt = time.Time{}
	}
	return out, nil
}
