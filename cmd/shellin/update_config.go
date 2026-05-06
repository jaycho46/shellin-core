// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"strings"
)

const (
	autoUpdateAppliedEnv       = "SHELLIN_AUTO_UPDATE_APPLIED"
	autoUpdateBaseURLEnv       = "SHELLIN_BASE_URL"
	updateCheckDisableEnv      = "SHELLIN_DISABLE_UPDATE_CHECK"
	legacyAutoUpdateDisableEnv = "SHELLIN_DISABLE_AUTO_UPDATE"
)

var (
	defaultCLIVersion      = "dev"
	defaultDownloadBaseURL = "https://shellin.dev"
	// Base64-encoded Ed25519 public key. Release builds must set this with
	// -ldflags "-X main.defaultReleaseManifestPublicKey=...".
	defaultReleaseManifestPublicKey = ""
)

func currentCLIVersion() string {
	version := strings.TrimSpace(defaultCLIVersion)
	if version == "" {
		return "dev"
	}
	return version
}

func isUpdateCheckDisabled() bool {
	return parseBoolDefault(os.Getenv(updateCheckDisableEnv), false) ||
		parseBoolDefault(os.Getenv(legacyAutoUpdateDisableEnv), false)
}

func updateBaseURL(override string) string {
	if raw := strings.TrimSpace(override); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	if raw := strings.TrimSpace(os.Getenv(autoUpdateBaseURLEnv)); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	return strings.TrimRight(strings.TrimSpace(defaultDownloadBaseURL), "/")
}

func releaseManifestPublicKey(override string) string {
	if raw := strings.TrimSpace(override); raw != "" {
		return raw
	}
	return strings.TrimSpace(defaultReleaseManifestPublicKey)
}
