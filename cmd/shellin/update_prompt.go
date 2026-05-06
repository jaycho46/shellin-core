// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func promptForAvailableUpdate(currentVersion, latestVersion string) (bool, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return promptForAvailableUpdateFallback(currentVersion, latestVersion)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return promptForAvailableUpdateFallback(currentVersion, latestVersion)
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(formatUpdateRawTTYText(formatUpdateAvailableMessage(currentVersion, latestVersion)))
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")
	if err := renderUpdateMenu(os.Stdout, 0, false); err != nil {
		return false, err
	}

	selected := 0
	for {
		action, err := readUpdatePromptAction(reader)
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}
		switch action {
		case updatePromptActionUp:
			if selected > 0 {
				selected--
				if err := renderUpdateMenu(os.Stdout, selected, true); err != nil {
					return false, err
				}
			}
		case updatePromptActionDown:
			if selected < len(updateMenuOptions)-1 {
				selected++
				if err := renderUpdateMenu(os.Stdout, selected, true); err != nil {
					return false, err
				}
			}
		case updatePromptActionChoose:
			return selected == 0, nil
		case updatePromptActionCancel:
			return false, nil
		}
		if errors.Is(err, io.EOF) {
			return false, nil
		}
	}
}

func promptForAvailableUpdateFallback(currentVersion, latestVersion string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(formatUpdateAvailableMessage(currentVersion, latestVersion))
	for {
		fmt.Print(formatUpdatePromptLine())

		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}

		updateNow, ok := parseUpdatePromptChoice(input)
		if ok {
			return updateNow, nil
		}
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		fmt.Print("\n")
		fmt.Print(hudConnectingFG)
		fmt.Print("Enter 'y' to install or 'n' to continue.")
		fmt.Print(ansiReset)
		fmt.Print("\n\n")
	}
}
