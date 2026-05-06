// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"io"
	"strings"
)

const ansiBlue = "\x1b[34m"

var updateMenuOptions = []string{
	"Yes, install the update now",
	"No, keep using the current version",
}

func formatUpdateAvailableMessage(currentVersion, latestVersion string) string {
	var buf strings.Builder
	buf.WriteString("\n")
	buf.WriteString(hudConnectingFG)
	buf.WriteString(ansiBold)
	buf.WriteString("Update Available")
	buf.WriteString(ansiReset)
	buf.WriteString("\n")
	buf.WriteString("\n")
	buf.WriteString(formatStatusRow("Current", currentVersion))
	buf.WriteString("\n")
	buf.WriteString(formatStatusRow("Latest", latestVersion))
	buf.WriteString("\n")
	buf.WriteString("\n")
	return buf.String()
}

func formatUpdatePromptLine() string {
	return hudConnectingFG + "Install now? [y]es / [n]o" + ansiReset + ": "
}

func renderUpdateMenu(out io.Writer, selected int, redraw bool) error {
	if redraw {
		if _, err := fmt.Fprintf(out, "\x1b[%dF", updateMenuLineCount()); err != nil {
			return err
		}
	}

	lines := []string{
		formatUpdateMenuOptionLine(updateMenuOptions[0], selected == 0),
		formatUpdateMenuOptionLine(updateMenuOptions[1], selected == 1),
	}
	for _, line := range lines {
		if _, err := fmt.Fprint(out, "\x1b[2K\r"); err != nil {
			return err
		}
		if _, err := fmt.Fprint(out, line); err != nil {
			return err
		}
		if _, err := fmt.Fprint(out, "\r\n"); err != nil {
			return err
		}
	}
	return nil
}

func formatUpdateMenuOptionLine(text string, selected bool) string {
	var buf strings.Builder
	if selected {
		buf.WriteString(ansiBlue)
		buf.WriteString("> ")
		buf.WriteString(text)
		buf.WriteString(ansiReset)
		return buf.String()
	}

	buf.WriteString(hudBaseFG)
	buf.WriteString("  ")
	buf.WriteString(text)
	buf.WriteString(ansiReset)
	return buf.String()
}

func updateMenuLineCount() int {
	return 2
}

func formatUpdateRawTTYText(text string) string {
	return strings.ReplaceAll(text, "\n", "\r\n")
}
