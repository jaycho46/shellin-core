// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import "strings"

func dispatchSubcommand(args []string) bool {
	if len(args) <= 1 {
		return false
	}

	var err error
	commandArgs := args[2:]
	switch strings.TrimSpace(args[1]) {
	case "login":
		err = runLoginCommand(commandArgs)
	case "version":
		err = runVersionCommand(commandArgs)
	case "update":
		err = runUpdateCommand(commandArgs)
	case "logout":
		err = runLogoutCommand(commandArgs)
	case "status":
		err = runStatusCommand(commandArgs)
	case "cleanup":
		err = runCleanupCommand(commandArgs)
	default:
		return false
	}

	if err != nil {
		fatalCLIError(err)
	}
	return true
}
