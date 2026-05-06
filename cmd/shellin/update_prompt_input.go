// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

type updatePromptAction uint8

const (
	updatePromptActionNone updatePromptAction = iota
	updatePromptActionUp
	updatePromptActionDown
	updatePromptActionChoose
	updatePromptActionCancel
)

func parseUpdatePromptChoice(input string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "u", "update", "y", "yes":
		return true, true
	case "", "c", "continue", "n", "no":
		return false, true
	default:
		return false, false
	}
}

func readUpdatePromptAction(reader *bufio.Reader) (updatePromptAction, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return updatePromptActionNone, err
	}

	switch b {
	case '\r', '\n':
		return updatePromptActionChoose, nil
	case 'k', 'K':
		return updatePromptActionUp, nil
	case 'j', 'J':
		return updatePromptActionDown, nil
	case 3, 27:
		if b == 3 {
			return updatePromptActionCancel, nil
		}
		next, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return updatePromptActionCancel, nil
			}
			return updatePromptActionNone, err
		}
		if next != '[' {
			return updatePromptActionCancel, nil
		}
		final, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return updatePromptActionCancel, nil
			}
			return updatePromptActionNone, err
		}
		switch final {
		case 'A':
			return updatePromptActionUp, nil
		case 'B':
			return updatePromptActionDown, nil
		default:
			return updatePromptActionNone, nil
		}
	default:
		return updatePromptActionNone, nil
	}
}
