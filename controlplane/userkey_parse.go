// SPDX-License-Identifier: AGPL-3.0-or-later

package controlplane

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ParseUserKeySpec accepts a comma-separated list of entries:
// user_key:subject[:max_sessions]
func ParseUserKeySpec(spec string, defaultMaxSessions int) ([]UserKeyEntry, error) {
	if defaultMaxSessions <= 0 {
		return nil, errors.New("default max sessions must be > 0")
	}

	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}

	result := make([]UserKeyEntry, 0)

	parts := strings.Split(spec, ",")
	for idx, raw := range parts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		chunks := strings.Split(raw, ":")
		if len(chunks) < 2 || len(chunks) > 3 {
			return nil, fmt.Errorf("invalid user key entry #%d: %q", idx+1, raw)
		}
		userKey := strings.TrimSpace(chunks[0])
		subject := strings.TrimSpace(chunks[1])
		if userKey == "" || subject == "" {
			return nil, fmt.Errorf("invalid user key entry #%d: %q", idx+1, raw)
		}
		maxSessions := defaultMaxSessions
		if len(chunks) == 3 {
			n, err := strconv.Atoi(strings.TrimSpace(chunks[2]))
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid max_sessions in entry #%d: %q", idx+1, raw)
			}
			maxSessions = n
		}

		result = append(result, UserKeyEntry{
			UserKey: userKey,
			Entitlement: UserEntitlement{
				Subject:     subject,
				MaxSessions: maxSessions,
				Active:      true,
			},
		})
	}
	return result, nil
}
