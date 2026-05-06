// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/creack/pty"
)

type controlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

func shellExitMessage(err error) (string, bool) {
	if err == nil {
		return "Session closed. See you soon. 👋", true
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return "", false
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return "", false
	}
	if status.Signaled() && status.Signal() == syscall.SIGINT {
		return "Session interrupted. See you soon. 👋", true
	}
	if status.ExitStatus() == 130 {
		return "Session interrupted. See you soon. 👋", true
	}
	if status.ExitStatus() == 0 {
		return "Session closed. See you soon. 👋", true
	}
	return "", false
}

func resizePTY(f *os.File, reserveBottomLine bool) error {
	size, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		return err
	}
	if reserveBottomLine && size.Rows > 1 {
		size.Rows--
	}
	return pty.Setsize(f, size)
}

func resizePTYFromViewer(f *os.File, cols, rows int) error {
	if cols < 2 || rows < 2 || cols > 1000 || rows > 1000 {
		return fmt.Errorf("invalid resize dimensions cols=%d rows=%d", cols, rows)
	}
	return pty.Setsize(f, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

func buildShellCommand(shellPath string) *exec.Cmd {
	var cmd *exec.Cmd
	base := strings.ToLower(filepath.Base(shellPath))
	switch base {
	case "bash", "zsh", "fish", "ksh", "sh":
		cmd = exec.Command(shellPath, "-i")
	default:
		cmd = exec.Command(shellPath)
	}
	cmd.Env = buildShellEnvironment(os.Environ())
	return cmd
}

func buildShellEnvironment(baseEnv []string) []string {
	if shellEnvironmentUsesUTF8(baseEnv) {
		return append([]string(nil), baseEnv...)
	}

	locale := preferredUTF8Locale()
	env := append([]string(nil), baseEnv...)
	env = upsertEnv(env, "LANG", locale)
	env = upsertEnv(env, "LC_CTYPE", locale)

	if value, ok := lookupEnv(baseEnv, "LC_ALL"); ok && strings.TrimSpace(value) != "" {
		env = upsertEnv(env, "LC_ALL", locale)
	}

	return env
}

func shellEnvironmentUsesUTF8(env []string) bool {
	if value, ok := lookupEnv(env, "LC_ALL"); ok && strings.TrimSpace(value) != "" {
		return localeUsesUTF8(value)
	}
	if value, ok := lookupEnv(env, "LC_CTYPE"); ok && strings.TrimSpace(value) != "" {
		return localeUsesUTF8(value)
	}
	if value, ok := lookupEnv(env, "LANG"); ok && strings.TrimSpace(value) != "" {
		return localeUsesUTF8(value)
	}
	return false
}

func localeUsesUTF8(raw string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	return strings.Contains(normalized, "UTF-8") || strings.Contains(normalized, "UTF8")
}

func preferredUTF8Locale() string {
	if runtime.GOOS == "linux" {
		return "C.UTF-8"
	}
	return "en_US.UTF-8"
}

func lookupEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix), true
		}
	}
	return "", false
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
