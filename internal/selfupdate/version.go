// SPDX-License-Identifier: AGPL-3.0-or-later

package selfupdate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type parsedReleaseVersion struct {
	numbers    []int
	prerelease string
}

func shouldInstallLatestRelease(currentVersion, latestVersion string, allowUnknownCurrent bool) bool {
	cmp, err := compareReleaseVersions(currentVersion, latestVersion)
	if err != nil {
		if !allowUnknownCurrent {
			return false
		}
		return strings.TrimSpace(currentVersion) != strings.TrimSpace(latestVersion)
	}
	return cmp < 0
}

// ValidateVersion reports whether raw can participate in release comparison.
func ValidateVersion(raw string) error {
	_, err := parseReleaseVersion(raw)
	return err
}

func compareReleaseVersions(currentVersion, latestVersion string) (int, error) {
	current, err := parseReleaseVersion(currentVersion)
	if err != nil {
		return 0, fmt.Errorf("parse current version: %w", err)
	}
	latest, err := parseReleaseVersion(latestVersion)
	if err != nil {
		return 0, fmt.Errorf("parse latest version: %w", err)
	}

	maxLen := len(current.numbers)
	if len(latest.numbers) > maxLen {
		maxLen = len(latest.numbers)
	}
	for i := 0; i < maxLen; i++ {
		currentPart := 0
		latestPart := 0
		if i < len(current.numbers) {
			currentPart = current.numbers[i]
		}
		if i < len(latest.numbers) {
			latestPart = latest.numbers[i]
		}
		if currentPart < latestPart {
			return -1, nil
		}
		if currentPart > latestPart {
			return 1, nil
		}
	}

	switch {
	case current.prerelease == latest.prerelease:
		return 0, nil
	case current.prerelease == "":
		return 1, nil
	case latest.prerelease == "":
		return -1, nil
	case current.prerelease < latest.prerelease:
		return -1, nil
	case current.prerelease > latest.prerelease:
		return 1, nil
	default:
		return 0, nil
	}
}

func parseReleaseVersion(raw string) (parsedReleaseVersion, error) {
	version := strings.TrimSpace(raw)
	if version == "" {
		return parsedReleaseVersion{}, errors.New("version is required")
	}
	version = strings.TrimPrefix(version, "v")
	if idx := strings.Index(version, "+"); idx >= 0 {
		version = version[:idx]
	}

	prerelease := ""
	if idx := strings.Index(version, "-"); idx >= 0 {
		prerelease = version[idx+1:]
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return parsedReleaseVersion{}, fmt.Errorf("invalid version %q", raw)
	}
	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return parsedReleaseVersion{}, fmt.Errorf("invalid version %q", raw)
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return parsedReleaseVersion{}, fmt.Errorf("invalid version %q", raw)
		}
		numbers = append(numbers, n)
	}

	return parsedReleaseVersion{
		numbers:    numbers,
		prerelease: prerelease,
	}, nil
}
