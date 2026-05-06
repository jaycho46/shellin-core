// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

const (
	hudStatusLabel  = " STATUS "
	hudBaseBG       = "\x1b[48;2;53;53;51m"
	hudBaseFG       = "\x1b[38;2;244;241;234m"
	hudLabelBG      = "\x1b[48;2;78;77;72m"
	hudLabelFG      = "\x1b[38;2;244;241;234m"
	hudSessionBG    = "\x1b[48;2;96;94;88m"
	hudSessionFG    = "\x1b[38;2;250;247;240m"
	hudReadyFG      = "\x1b[38;2;176;173;166m"
	hudConnectedFG  = "\x1b[38;2;121;214;145m"
	hudConnectingFG = "\x1b[38;2;240;197;74m"
	hudErrorFG      = "\x1b[38;2;239;118;118m"
)

type hudDisplayState struct {
	color   string
	text    string
	animate bool
}

type hudLineLayout struct {
	line        string
	leftBadge   string
	statusStart int
	statusText  string
	rightStart  int
	rightBadge  string
}

func formatHUDLine(cols int, statusText, rightText string) hudLineLayout {
	layout := hudLineLayout{statusStart: -1, rightStart: -1}
	if cols <= 0 {
		return layout
	}

	leftBadge := hudStatusLabel
	if len(leftBadge) > cols {
		leftBadge = leftBadge[:cols]
	}

	statusSegment := ""
	if text := strings.TrimSpace(statusText); text != "" {
		statusSegment = " " + text
	}

	rightBadge := hudSessionBadgeText(rightText)
	maxLeftWidth := cols
	if rightBadge != "" {
		maxLeftWidth = cols - len(rightBadge)
		if maxLeftWidth < 0 {
			maxLeftWidth = 0
		}
	}

	leftText := leftBadge + statusSegment
	if len(leftText) > maxLeftWidth {
		leftText = leftText[:maxLeftWidth]
	}
	line := leftText
	if len(line) < cols {
		line += strings.Repeat(" ", cols-len(line))
	}
	if len(line) > cols {
		line = line[:cols]
	}

	if len(leftBadge) > len(line) {
		leftBadge = line
	}
	layout.leftBadge = leftBadge

	statusStart := len(leftBadge)
	if len(leftText) > statusStart {
		statusEnd := statusStart + len(statusSegment)
		if statusEnd > len(leftText) {
			statusEnd = len(leftText)
		}
		if statusEnd > statusStart {
			layout.statusStart = statusStart
			layout.statusText = leftText[statusStart:statusEnd]
		}
	}

	if rightBadge == "" {
		layout.line = line
		return layout
	}
	if len(rightBadge) > cols {
		rightBadge = rightBadge[len(rightBadge)-cols:]
	}
	layout.rightBadge = rightBadge
	layout.rightStart = cols - len(rightBadge)
	if layout.rightStart < 0 {
		layout.rightStart = 0
	}

	lineBytes := []byte(line)
	copy(lineBytes[layout.rightStart:], rightBadge)
	layout.line = string(lineBytes)
	return layout
}

func hudSessionBadgeText(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ""
	}
	return " " + sessionID + " "
}

func hudSessionIDLabel(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(sessionID))
	return fmt.Sprintf("%x", sum[:])[:6]
}

func hudRightText(shellPath, sessionID string) string {
	if shortID := hudSessionIDLabel(sessionID); shortID != "" {
		return strings.ToUpper(shortID)
	}
	return ""
}

func hudStatusDisplay(signalState, p2pState string, animFrame int) hudDisplayState {
	signalState = strings.ToLower(strings.TrimSpace(signalState))
	p2pState = strings.ToLower(strings.TrimSpace(p2pState))

	switch {
	case p2pState == "connected":
		return hudDisplayState{color: hudConnectedFG, text: "connected"}
	case p2pState == "negotiating" || signalState == "viewer_connected" || signalState == "connecting":
		return hudDisplayState{
			color:   hudConnectingFG,
			text:    "connecting" + strings.Repeat(".", animFrame%4),
			animate: true,
		}
	case signalState == "error" || signalState == "failed" || signalState == "closed" || signalState == "disconnected" || p2pState == "failed" || p2pState == "closed":
		return hudDisplayState{color: hudErrorFG, text: "connection error"}
	default:
		return hudDisplayState{color: hudReadyFG, text: "ready for mobile attach"}
	}
}
