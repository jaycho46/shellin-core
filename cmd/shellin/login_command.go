// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func runLoginCommand(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	controlPlaneURL := strings.TrimSpace(defaultControlPlaneURL)
	authProfilePath := defaultAuthProfilePath()

	fs.StringVar(&authProfilePath, "auth-profile", authProfilePath, "Auth profile path")

	if len(args) > 1 && !strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		reordered := append([]string{}, args[1:]...)
		reordered = append(reordered, args[0])
		args = reordered
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: %s login [flags] <one-time-login-key>", cliCommandName)
	}
	if strings.TrimSpace(controlPlaneURL) == "" {
		return errors.New("control plane URL is not configured in this build")
	}
	normalizedControlURL, err := normalizeServerURL(controlPlaneURL, parseBoolDefault(defaultAllowInsecureURL, false))
	if err != nil {
		return fmt.Errorf("invalid control plane URL: %w", err)
	}

	authSession, err := deviceLoginWithKey(normalizedControlURL, fs.Arg(0))
	if err != nil {
		return fmt.Errorf("device login failed: %w", err)
	}
	if err := saveAuthProfile(authProfilePath, authProfile{
		ControlPlaneURL:       normalizedControlURL,
		AccessToken:           authSession.AccessToken,
		AccessTokenExpiresAt:  authSession.AccessTokenExpiresAt,
		RefreshToken:          authSession.RefreshToken,
		RefreshTokenExpiresAt: authSession.RefreshTokenExpiresAt,
		UpdatedAt:             time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return fmt.Errorf("save auth profile failed: %w", err)
	}

	fmt.Print(formatLoginSuccessMessage())
	return nil
}

func runLogoutCommand(args []string) error {
	fs := flag.NewFlagSet("logout", flag.ContinueOnError)
	authProfilePath := defaultAuthProfilePath()
	fs.StringVar(&authProfilePath, "auth-profile", authProfilePath, "Auth profile path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := os.Remove(authProfilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("already logged out (%s)\n", authProfilePath)
			return nil
		}
		return fmt.Errorf("failed to remove auth profile: %w", err)
	}
	fmt.Printf("logout succeeded (%s)\n", authProfilePath)
	return nil
}

func formatLoginSuccessMessage() string {
	var buf strings.Builder
	buf.WriteString("\n")
	buf.WriteString("\x1b[32m")
	buf.WriteString(ansiBold)
	buf.WriteString("Login Complete ✨")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	buf.WriteString(hudBaseFG)
	buf.WriteString("You're ready to use ")
	buf.WriteString(cliCommandName)
	buf.WriteString(".")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString(hudBaseFG)
	buf.WriteString("Run `")
	buf.WriteString(cliCommandName)
	buf.WriteString("` to start a new terminal session.")
	buf.WriteString(ansiReset)
	buf.WriteString("\n\n")
	return buf.String()
}
