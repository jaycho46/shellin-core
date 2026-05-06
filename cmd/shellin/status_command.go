// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jaycho46/shellin-core/protocol"
)

func runStatusCommand(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
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
		return fmt.Errorf("usage: %s status [flags]", cliCommandName)
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
		return fmt.Errorf("status authorization failed: %w", err)
	}

	status, finalAuth, err := requestSessionStatusWithAuthSession(cfg, initialAuth, profilePath)
	if err != nil {
		return fmt.Errorf("status request failed: %w", err)
	}
	if profilePath != "" && (finalAuth.AccessToken != initialAuth.AccessToken || finalAuth.RefreshToken != initialAuth.RefreshToken) {
		if saveErr := saveAuthProfile(profilePath, authProfile{
			ControlPlaneURL:       cfg.ControlPlaneURL,
			AccessToken:           finalAuth.AccessToken,
			AccessTokenExpiresAt:  finalAuth.AccessTokenExpiresAt,
			RefreshToken:          finalAuth.RefreshToken,
			RefreshTokenExpiresAt: finalAuth.RefreshTokenExpiresAt,
			UpdatedAt:             time.Now().UTC().Format(time.RFC3339),
		}); saveErr != nil {
			log.Printf("warning: failed to update auth profile after status refresh: %v", saveErr)
		}
	}

	fmt.Print(formatStatusCommandOutput(status, time.Local))
	return nil
}

func formatStatusCommandOutput(status protocol.SessionStatusResponse, loc *time.Location) string {
	if loc == nil {
		loc = time.Local
	}

	var buf strings.Builder
	buf.WriteString("\n")
	buf.WriteString(ansiCyan)
	buf.WriteString(ansiBold)
	buf.WriteString("Status")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	buf.WriteString(formatStatusRow("Plan", displayPlanName(status.ProductID)))
	buf.WriteString("\n")
	buf.WriteString(formatStatusRow("Sessions", fmt.Sprintf("%d / %d", status.ActiveSessions, status.MaxSessions)))
	if value := formatStatusExpiry(status.ExpiresAt, loc); value != "" {
		buf.WriteString("\n")
		buf.WriteString(formatStatusRow("Ends", value))
	}
	buf.WriteString("\n\n")
	return buf.String()
}

func formatStatusRow(label, value string) string {
	return ansiGray + fmt.Sprintf("%-10s", label) + ansiReset + " " + hudBaseFG + value + ansiReset
}

func displayPlanName(productID string) string {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return "unknown"
	}

	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	rawParts := strings.Fields(replacer.Replace(strings.ToLower(productID)))
	if len(rawParts) == 0 {
		return productID
	}

	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "shellin" {
			continue
		}
		switch part {
		case "lite":
			parts = append(parts, "Lite")
		case "pro":
			parts = append(parts, "Pro")
		case "monthly":
		case "yearly":
		case "annual":
		case "weekly":
		case "daily":
		default:
			parts = append(parts, strings.ToUpper(part[:1])+part[1:])
		}
	}
	if len(parts) == 0 {
		return productID
	}
	return "Shellin " + strings.Join(parts, " ")
}

func formatStatusExpiry(raw string, loc *time.Location) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	expiresAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return expiresAt.In(loc).Format("Jan 2 2006")
}
