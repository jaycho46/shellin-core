// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"log"
	"os"

	"golang.org/x/term"
)

func fatalStartupError(startup *startupScreen, prefix string, err error) {
	if display, ok := startupDisplayForError(err); ok && startup != nil && startup.Enabled() {
		startup.ShowDisplay(display)
		_, _ = startup.out.WriteString("\n\n")
		os.Exit(1)
	}
	log.Fatalf("%s: %v", prefix, err)
}

func fatalCLIError(err error) {
	if display, ok := startupDisplayForError(err); ok && term.IsTerminal(int(os.Stdout.Fd())) {
		screen := newStartupScreen(true, &lockedStdout{out: os.Stdout})
		screen.ShowDisplay(display)
		_, _ = screen.out.WriteString("\n\n")
		os.Exit(1)
	}
	log.Fatal(err)
}
