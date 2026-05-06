// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"
)

func runCleanupCommand(args []string) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	authProfilePath := defaultAuthProfilePath()
	userKey := strings.TrimSpace(defaultUserKey)
	controlPlaneURL := strings.TrimSpace(defaultControlPlaneURL)

	setFilteredUsage(fs, map[string]bool{
		"user-key": true,
	})
	fs.StringVar(&authProfilePath, "auth-profile", authProfilePath, "Auth profile path")
	fs.StringVar(&userKey, "user-key", userKey, "User key used to request control-plane auth")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s cleanup [flags]", cliCommandName)
	}
	if controlPlaneURL == "" {
		return errors.New("control plane URL is not configured in this build")
	}
	normalizedControlURL, err := normalizeServerURL(controlPlaneURL, parseBoolDefault(defaultAllowInsecureURL, false))
	if err != nil {
		return fmt.Errorf("invalid control plane URL: %w", err)
	}

	cfg := config{
		ControlPlaneURL: normalizedControlURL,
		AuthProfilePath: authProfilePath,
		UserKey:         strings.TrimSpace(userKey),
	}
	initialAuth, profilePath, err := resolveControlPlaneAuth(cfg)
	if err != nil {
		return fmt.Errorf("cleanup authorization failed: %w", err)
	}

	result, finalAuth, err := requestTerminateAllAgentsWithAuthSession(cfg, initialAuth, profilePath)
	if err != nil {
		return fmt.Errorf("cleanup request failed: %w", err)
	}
	if profilePath != "" && !sameControlPlaneAuthSession(finalAuth, initialAuth) {
		if saveErr := saveAuthProfile(profilePath, authProfile{
			ControlPlaneURL:       cfg.ControlPlaneURL,
			AccessToken:           finalAuth.AccessToken,
			AccessTokenExpiresAt:  finalAuth.AccessTokenExpiresAt,
			RefreshToken:          finalAuth.RefreshToken,
			RefreshTokenExpiresAt: finalAuth.RefreshTokenExpiresAt,
			UpdatedAt:             time.Now().UTC().Format(time.RFC3339),
		}); saveErr != nil {
			log.Printf("warning: failed to update auth profile after cleanup refresh: %v", saveErr)
		}
	}

	if result.TerminatedSessions == 0 {
		fmt.Print(formatCleanupCompleteMessage())
		return nil
	}
	fmt.Print(formatCleanupRequestedMessage(result.TerminatedSessions))
	return nil
}

func formatCleanupRequestedMessage(activeSessions int) string {
	var buf strings.Builder
	buf.WriteString("\n")
	buf.WriteString(hudConnectingFG)
	buf.WriteString(ansiBold)
	buf.WriteString("Cleanup Requested")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	buf.WriteString(hudBaseFG)
	buf.WriteString(fmt.Sprintf("Cleanup requested for %d active session(s).", activeSessions))
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString(hudBaseFG)
	buf.WriteString("Active sessions usually shut down within about 30 seconds.")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	return buf.String()
}

func formatCleanupCompleteMessage() string {
	var buf strings.Builder
	buf.WriteString("\n")
	buf.WriteString("\x1b[32m")
	buf.WriteString(ansiBold)
	buf.WriteString("Cleanup Complete")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	buf.WriteString(hudBaseFG)
	buf.WriteString("No active sessions to clean up.")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString(hudBaseFG)
	buf.WriteString("Everything is already clear.")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	return buf.String()
}
