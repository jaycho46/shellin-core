// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

func isSessionLimitReachedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "session limit reached")
}

func isSubscriptionInactiveError(err error) bool {
	if err == nil {
		return false
	}
	var inactive *subscriptionInactiveError
	return errors.As(err, &inactive)
}

func isSessionTerminatedError(err error) bool {
	if err == nil {
		return false
	}
	var terminated *sessionTerminatedError
	return errors.As(err, &terminated)
}

func isLoginRequiredError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "load auth profile failed") ||
		strings.Contains(msg, "auth profile is missing tokens") ||
		strings.Contains(msg, fmt.Sprintf("run `%s login <key>`", cliCommandName))
}

func isAuthorizationFailedError(err error) bool {
	if err == nil {
		return false
	}
	return isUnauthorizedError(err) || strings.Contains(strings.ToLower(err.Error()), "unauthorized")
}

func isHeartbeatAuthorizationFailure(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "failed to refresh control-plane auth during heartbeat") ||
		strings.Contains(msg, "heartbeat retry after fallback user key exchange failed")
}

func isDeviceLoginFailedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "device login failed")
}

func sessionLimitSubtitle(err error) string {
	if err == nil {
		return "Your subscription is already using the maximum number of sessions."
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "Your subscription is already using the maximum number of sessions."
	}
	return msg
}

func inactiveSubscriptionSubtitle(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "expired":
		return "Your subscription has expired."
	case "revoked":
		return "No active subscription was found for this account."
	default:
		return "No active subscription was found for this account."
	}
}

func subscriptionInactiveDisplaySubtitle(err error) string {
	var inactive *subscriptionInactiveError
	if errors.As(err, &inactive) {
		return inactiveSubscriptionSubtitle(inactive.status)
	}
	return inactiveSubscriptionSubtitle("")
}

func deviceLoginFailureSubtitle(err error) string {
	if err == nil {
		return "Your login key could not be used."
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "status 401") || strings.Contains(lower, "unauthorized") {
		return "Your one-time login key was rejected or has expired."
	}
	return msg
}

func heartbeatAuthorizationFailureSubtitle(err error) string {
	if err == nil {
		return "Your session authorization expired and automatic recovery failed."
	}
	raw := strings.TrimSpace(err.Error())
	msg := strings.ToLower(raw)
	switch {
	case strings.Contains(msg, "fallback user key exchange failed"):
		return "Your session authorization expired and automatic recovery failed. Sign in again to restore remote access."
	case strings.Contains(msg, "heartbeat retry after fallback user key exchange failed"):
		return "Your session authorization expired and the recovered session could not rejoin the control plane. Sign in again to restore remote access."
	default:
		if detail := heartbeatAuthorizationFailureDetail(raw); detail != "" {
			return fmt.Sprintf("Your saved session could not be refreshed: %s. Sign in again to restore remote access.", detail)
		}
		return "Your session authorization expired. Sign in again to restore remote access."
	}
}

func authorizationFailureSubtitle(err error) string {
	if err == nil {
		return "Your authorization failed. Sign in again to restore remote access."
	}
	if isHeartbeatAuthorizationFailure(err) {
		return heartbeatAuthorizationFailureSubtitle(err)
	}
	raw := strings.TrimSpace(err.Error())
	if detail := statusAuthorizationFailureDetail(raw); detail != "" {
		return fmt.Sprintf("Your saved session could not be refreshed: %s. Sign in again to restore remote access.", detail)
	}
	return raw
}

func heartbeatAuthorizationFailureDetail(msg string) string {
	msg = strings.TrimSpace(msg)
	for _, prefix := range []string{
		"failed to refresh control-plane auth during heartbeat:",
		"heartbeat retry after refresh failed:",
		"heartbeat retry after shared auth refresh failed:",
		"agent heartbeat failed after shared auth reload:",
	} {
		if strings.HasPrefix(strings.ToLower(msg), strings.ToLower(prefix)) {
			return strings.TrimSpace(msg[len(prefix):])
		}
	}
	return ""
}

func statusAuthorizationFailureDetail(msg string) string {
	msg = strings.TrimSpace(msg)
	for _, prefix := range []string{
		"status request failed:",
		"cleanup request failed:",
		"cleanup authorization failed:",
		"status authorization failed:",
	} {
		if strings.HasPrefix(strings.ToLower(msg), strings.ToLower(prefix)) {
			msg = strings.TrimSpace(msg[len(prefix):])
			break
		}
	}

	if detail := parenthesizedSegmentAfter(msg, "refresh failed"); detail != "" {
		return detail
	}
	if detail := parenthesizedSegmentAfter(msg, "status request failed"); detail != "" {
		return detail
	}
	if strings.Contains(strings.ToLower(msg), "control plane refresh status 401") {
		return msg
	}
	return ""
}

func parenthesizedSegmentAfter(msg, marker string) string {
	idx := strings.Index(strings.ToLower(msg), strings.ToLower(marker)+" (")
	if idx < 0 {
		return ""
	}
	start := idx + len(marker) + 2
	if start >= len(msg) {
		return ""
	}
	depth := 1
	for i := start; i < len(msg); i++ {
		switch msg[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(msg[start:i])
			}
		}
	}
	return ""
}
