// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"
)

func startupDisplayForError(err error) (startupDisplayState, bool) {
	if err == nil {
		return startupDisplayState{}, false
	}
	switch {
	case isSessionTerminatedError(err):
		return startupDisplayState{
			title:    "Session Ended",
			titleFG:  hudConnectingFG,
			subtitle: "This terminal session was terminated remotely.",
			steps: []startupStepState{
				{label: fmt.Sprintf("Run %s again to start a fresh session", cliCommandName)},
			},
			animate: false,
		}, true
	case isHeartbeatAuthorizationFailure(err):
		return startupDisplayState{
			title:    "Authorization Failed",
			titleFG:  hudErrorFG,
			subtitle: heartbeatAuthorizationFailureSubtitle(err),
			steps: []startupStepState{
				{label: fmt.Sprintf("Run %s login <key>", cliCommandName)},
				{label: fmt.Sprintf("Run %s again", cliCommandName)},
			},
			animate: false,
		}, true
	case isSessionLimitReachedError(err):
		return startupDisplayState{
			title:    "Session Limit Reached",
			titleFG:  hudConnectingFG,
			subtitle: sessionLimitSubtitle(err),
			steps: []startupStepState{
				{label: "Close another active terminal session"},
				{label: fmt.Sprintf("Run %s again", cliCommandName)},
			},
			animate: false,
		}, true
	case isSubscriptionInactiveError(err):
		return startupDisplayState{
			title:    "Subscription Required",
			titleFG:  hudConnectingFG,
			subtitle: subscriptionInactiveDisplaySubtitle(err),
			steps: []startupStepState{
				{label: "Purchase or restore an active subscription in shellin app"},
				{label: fmt.Sprintf("Run %s again", cliCommandName)},
			},
			animate: false,
		}, true
	case isLoginRequiredError(err):
		return startupDisplayState{
			title:    "Login Required",
			titleFG:  hudConnectingFG,
			subtitle: "No saved login was found on this machine.",
			steps: []startupStepState{
				{label: fmt.Sprintf("Run %s login <key>", cliCommandName)},
				{label: fmt.Sprintf("Run %s again", cliCommandName)},
			},
			animate: false,
		}, true
	case isDeviceLoginFailedError(err):
		return startupDisplayState{
			title:    "Login Failed",
			titleFG:  hudErrorFG,
			subtitle: deviceLoginFailureSubtitle(err),
			steps: []startupStepState{
				{label: "Check that your one-time login key is still valid"},
				{label: fmt.Sprintf("Run %s login <key> again", cliCommandName)},
			},
			animate: false,
		}, true
	case isAuthorizationFailedError(err):
		return startupDisplayState{
			title:    "Authorization Failed",
			titleFG:  hudErrorFG,
			subtitle: authorizationFailureSubtitle(err),
			steps: []startupStepState{
				{label: fmt.Sprintf("Run %s login <key>", cliCommandName)},
				{label: fmt.Sprintf("Run %s again", cliCommandName)},
			},
			animate: false,
		}, true
	default:
		return startupDisplayState{
			title:    "Unable To Start Session",
			titleFG:  hudErrorFG,
			subtitle: strings.TrimSpace(err.Error()),
			steps: []startupStepState{
				{label: "Check your network and control plane settings"},
				{label: fmt.Sprintf("Run %s again", cliCommandName)},
			},
			animate: false,
		}, true
	}
}
