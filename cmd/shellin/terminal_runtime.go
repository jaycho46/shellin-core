// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func terminalSupportsHUD() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func startTerminalResizeWatcher(ptmx *os.File, hud *bottomHUD) func() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return func() {}
	}

	if err := resizePTY(ptmx, hud.Enabled()); err != nil {
		log.Printf("initial resize failed: %v", err)
	}
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			_ = resizePTY(ptmx, hud.Enabled())
			if hud != nil {
				hud.Refresh()
			}
		}
	}()
	sigwinch <- syscall.SIGWINCH

	return func() {
		signal.Stop(sigwinch)
	}
}

func enterRawTerminal() func() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil
	}
	return func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}
}
