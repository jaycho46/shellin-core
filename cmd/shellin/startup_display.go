// SPDX-License-Identifier: AGPL-3.0-or-later

package main

type startupStage string

const (
	startupStageAuthorizing     startupStage = "authorizing"
	startupStageCheckingLimits  startupStage = "checking_limits"
	startupStageCreatingSession startupStage = "creating_session"
	startupStageWaitingClient   startupStage = "waiting_client"
	startupStageNegotiating     startupStage = "negotiating"
	startupStageStartingShell   startupStage = "starting_shell"
)

type startupDisplayState struct {
	title    string
	titleFG  string
	subtitle string
	steps    []startupStepState
	animate  bool
}

type startupStepState struct {
	label  string
	status string
}

func startupStatusDisplay(stage startupStage, animFrame int) startupDisplayState {
	switch stage {
	case startupStageAuthorizing:
		return startupDisplayState{
			title:    "Connecting...",
			subtitle: "Authenticating session",
			steps: []startupStepState{
				{label: "Authenticate", status: "active"},
				{label: "Check session limit", status: "pending"},
				{label: "Create relay session", status: "pending"},
			},
			animate: true,
		}
	case startupStageCheckingLimits:
		return startupDisplayState{
			title:    "Connecting...",
			subtitle: "Checking session availability",
			steps: []startupStepState{
				{label: "Authenticate", status: "done"},
				{label: "Check session limit", status: "active"},
				{label: "Create relay session", status: "pending"},
			},
			animate: true,
		}
	case startupStageCreatingSession:
		return startupDisplayState{
			title:    "Connecting...",
			subtitle: "Preparing relay session",
			steps: []startupStepState{
				{label: "Authenticate", status: "done"},
				{label: "Check session limit", status: "done"},
				{label: "Create relay session", status: "active"},
			},
			animate: true,
		}
	case startupStageNegotiating:
		return startupDisplayState{
			title:    "Connecting...",
			subtitle: "Client found. Establishing secure terminal",
			steps: []startupStepState{
				{label: "Authenticate", status: "done"},
				{label: "Check session limit", status: "done"},
				{label: "Create relay session", status: "done"},
			},
			animate: false,
		}
	case startupStageStartingShell:
		return startupDisplayState{
			title:    "Connecting...",
			subtitle: "Connected. Starting terminal",
			steps: []startupStepState{
				{label: "Authenticate", status: "done"},
				{label: "Check session limit", status: "done"},
				{label: "Create relay session", status: "done"},
			},
			animate: false,
		}
	case startupStageWaitingClient:
		fallthrough
	default:
		return startupDisplayState{
			title:    "Connecting...",
			subtitle: "Waiting for client to join",
			steps: []startupStepState{
				{label: "Authenticate", status: "done"},
				{label: "Check session limit", status: "done"},
				{label: "Create relay session", status: "done"},
			},
			animate: false,
		}
	}
}
