// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jaycho46/shellin-core/internal/selfupdate"

	"golang.org/x/term"
)

func maybePromptForCLIUpdate() error {
	if isUpdateCheckDisabled() {
		return nil
	}
	if parseBoolDefault(os.Getenv(autoUpdateAppliedEnv), false) {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil
	}

	currentVersion := currentCLIVersion()
	if err := selfupdate.ValidateVersion(currentVersion); err != nil {
		return nil
	}

	checkResult, err := selfupdate.Check(selfUpdateOptions(currentVersion, false))
	if err != nil {
		return err
	}
	if !checkResult.UpdateAvailable {
		return nil
	}

	updateNow, err := promptForAvailableUpdate(checkResult.CurrentVersion, checkResult.LatestVersion)
	if err != nil {
		return err
	}
	if !updateNow {
		return nil
	}

	_, err = selfupdate.Perform(selfUpdateOptions(currentVersion, false, func(opts *selfupdate.Options) {
		opts.Args = append([]string{}, os.Args...)
		opts.Reexec = true
	}))
	return err
}

func selfUpdateOptions(currentVersion string, allowUnknownCurrent bool, configure ...func(*selfupdate.Options)) selfupdate.Options {
	opts := selfupdate.Options{
		AppliedEnvName:      autoUpdateAppliedEnv,
		AllowUnknownCurrent: allowUnknownCurrent,
		BaseURL:             updateBaseURL(""),
		CommandName:         cliCommandName,
		CurrentVersion:      currentVersion,
		ManifestPublicKey:   releaseManifestPublicKey(""),
	}
	for _, fn := range configure {
		fn(&opts)
	}
	return opts
}

func runUpdateCommand(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: %s update", cliCommandName)
	}

	result, err := selfupdate.Perform(selfUpdateOptions(currentCLIVersion(), true))
	if err != nil {
		return err
	}

	if result.Updated {
		fmt.Printf("%s updated from %s to %s\n", cliCommandName, result.CurrentVersion, result.LatestVersion)
		return nil
	}
	if result.LatestVersion != "" {
		fmt.Printf("%s is already up to date (%s)\n", cliCommandName, result.LatestVersion)
		return nil
	}
	fmt.Printf("%s is already up to date (%s)\n", cliCommandName, result.CurrentVersion)
	return nil
}
