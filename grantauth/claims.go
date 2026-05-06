// SPDX-License-Identifier: AGPL-3.0-or-later

package grantauth

import (
	"encoding/json"
	"errors"
	"strings"
)

func parseAudience(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty aud")
	}

	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		one = strings.TrimSpace(one)
		if one == "" {
			return nil, errors.New("empty aud")
		}
		return []string{one}, nil
	}

	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(many))
	for _, candidate := range many {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		out = append(out, candidate)
	}
	if len(out) == 0 {
		return nil, errors.New("empty aud list")
	}
	return out, nil
}

func containsAudience(aud []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, actual := range aud {
		if strings.EqualFold(strings.TrimSpace(actual), want) {
			return true
		}
	}
	return false
}

func parseUnix(n json.Number) (int64, error) {
	if n == "" {
		return 0, errors.New("missing number")
	}
	return n.Int64()
}
