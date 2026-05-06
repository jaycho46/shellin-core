// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"
)

func formatStartupStep(index, total int, step startupStepState, animate bool, animFrame int) string {
	if strings.TrimSpace(step.status) == "" {
		return formatPlainStartupStep(index, step.label)
	}

	statusLabel := "[...]"
	color := ansiGray
	switch step.status {
	case "done":
		statusLabel = "[✓]"
		color = "\x1b[32m"
	case "active":
		statusLabel = "[" + strings.Repeat(".", 1+animFrame%4) + "]"
		color = hudConnectingFG
	case "pending":
		statusLabel = "[ ]"
		color = ansiGray
	}
	if !animate && step.status == "active" {
		statusLabel = "[...]"
	}
	line := fmt.Sprintf("%d/%d %s %s", index, total, statusLabel, step.label)
	return color + line + ansiReset
}

func formatPlainStartupStep(index int, label string) string {
	label = strings.TrimSpace(label)
	prefix := fmt.Sprintf("%s%d.%s ", ansiGray, index, ansiReset)
	if command, suffix, ok := plainStartupCommand(label); ok {
		line := prefix + ansiGray + "Run " + ansiReset + ansiBold + hudReadyFG + "`" + command + "`" + ansiReset
		if suffix != "" {
			line += " " + ansiGray + suffix + ansiReset
		}
		return line
	}
	return prefix + ansiGray + label + ansiReset
}

func plainStartupCommand(label string) (string, string, bool) {
	label = strings.TrimSpace(label)
	if !strings.HasPrefix(label, "Run ") {
		return "", "", false
	}
	command := strings.TrimSpace(strings.TrimPrefix(label, "Run "))
	if command == "" {
		return "", "", false
	}
	if idx := strings.Index(command, " again"); idx >= 0 {
		baseCommand := strings.TrimSpace(command[:idx])
		if baseCommand == "" {
			return "", "", false
		}
		suffix := "again"
		trailing := strings.TrimSpace(command[idx+len(" again"):])
		if trailing != "" {
			suffix += " " + trailing
		}
		return baseCommand, suffix, true
	}
	return command, "", true
}
