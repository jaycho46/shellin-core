// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import "os"

func main() {
	if dispatchSubcommand(os.Args) {
		return
	}
	runInteractiveSession()
}
