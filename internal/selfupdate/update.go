// SPDX-License-Identifier: AGPL-3.0-or-later

package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const (
	defaultAppliedEnvName = "SHELLIN_AUTO_UPDATE_APPLIED"
	defaultCommandName    = "shellin"
	defaultCurrentVersion = "dev"
)

var execSelf = func(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}

// Options controls release lookup, verification, installation, and optional re-exec.
type Options struct {
	AllowUnknownCurrent bool
	AppliedEnvName      string
	CommandName         string
	ExecutablePath      string
	ManifestPublicKey   string
	Reexec              bool
	Args                []string
	BaseURL             string
	CurrentVersion      string
}

// Result describes the observable result of an update check or installation.
type Result struct {
	Checked         bool
	CurrentVersion  string
	ExecutablePath  string
	LatestVersion   string
	UpdateAvailable bool
	Updated         bool
}

// Check verifies the latest release manifest and reports whether it is newer.
func Check(opts Options) (Result, error) {
	result := Result{
		CurrentVersion: strings.TrimSpace(opts.CurrentVersion),
	}
	if result.CurrentVersion == "" {
		result.CurrentVersion = defaultCurrentVersion
	}

	manifest, err := fetchLatestReleaseManifest(opts.BaseURL, opts.ManifestPublicKey)
	if err != nil {
		return result, err
	}
	result.Checked = true
	result.LatestVersion = strings.TrimSpace(manifest.Version)

	if result.LatestVersion == "" {
		return result, errors.New("latest release manifest did not include a version")
	}
	result.UpdateAvailable = shouldInstallLatestRelease(result.CurrentVersion, result.LatestVersion, opts.AllowUnknownCurrent)
	return result, nil
}

// Perform verifies the latest release and replaces the current executable if needed.
func Perform(opts Options) (Result, error) {
	result, err := Check(opts)
	if err != nil {
		return result, err
	}
	if !result.UpdateAvailable {
		return result, nil
	}

	executablePath := strings.TrimSpace(opts.ExecutablePath)
	if executablePath == "" {
		executablePath, err = resolveExecutablePath()
		if err != nil {
			return result, err
		}
	}
	result.ExecutablePath = executablePath

	manifest, err := fetchLatestReleaseManifest(opts.BaseURL, opts.ManifestPublicKey)
	if err != nil {
		return result, err
	}
	asset, err := manifest.assetForPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return result, err
	}

	commandName := normalizedCommandName(opts.CommandName)
	binaryData, mode, err := downloadAndExtractReleaseBinary(asset, commandName)
	if err != nil {
		return result, err
	}
	if err := installReplacementBinary(executablePath, binaryData, mode); err != nil {
		return result, err
	}

	result.Updated = true
	if opts.Reexec {
		reexecArgs := append([]string{}, opts.Args...)
		if len(reexecArgs) == 0 {
			reexecArgs = append(reexecArgs, os.Args...)
		}
		if len(reexecArgs) == 0 {
			reexecArgs = []string{executablePath}
		}

		env := append([]string{}, os.Environ()...)
		env = upsertEnv(env, appliedEnvName(opts.AppliedEnvName), "1")
		if err := execSelf(executablePath, reexecArgs, env); err != nil {
			return result, fmt.Errorf("updated to %s but failed to relaunch: %w", result.LatestVersion, err)
		}
	}
	return result, nil
}

func resolveExecutablePath() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path failed: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(executablePath)
	if err == nil && strings.TrimSpace(resolvedPath) != "" {
		executablePath = resolvedPath
	}
	return executablePath, nil
}

func normalizedCommandName(raw string) string {
	if commandName := strings.TrimSpace(raw); commandName != "" {
		return commandName
	}
	return defaultCommandName
}

func appliedEnvName(raw string) string {
	if name := strings.TrimSpace(raw); name != "" {
		return name
	}
	return defaultAppliedEnvName
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	next := prefix + value
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			out := append([]string{}, env...)
			out[i] = next
			return out
		}
	}
	return append(append([]string{}, env...), next)
}
