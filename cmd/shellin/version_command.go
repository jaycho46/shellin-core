// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/jaycho46/shellin-core/buildinfo"
)

func runVersionCommand(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Print build provenance as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s version [--json]", cliCommandName)
	}
	if *jsonOutput {
		buildinfo.Version = currentCLIVersion()
		buildinfo.ServiceReleaseID = currentCLIVersion()
		return json.NewEncoder(os.Stdout).Encode(buildinfo.Snapshot(cliCommandName))
	}
	fmt.Printf("%s %s\n", cliCommandName, currentCLIVersion())
	return nil
}
