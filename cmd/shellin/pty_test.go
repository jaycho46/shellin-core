// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestShellExitMessageHandlesInterrupt(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 130")
	err := cmd.Run()
	msg, ok := shellExitMessage(err)
	if !ok {
		t.Fatal("expected friendly shell exit message")
	}
	if msg != "Session interrupted. See you soon. 👋" {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestShellExitMessageHandlesCleanExit(t *testing.T) {
	msg, ok := shellExitMessage(nil)
	if !ok {
		t.Fatal("expected clean exit message")
	}
	if msg != "Session closed. See you soon. 👋" {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestBuildShellEnvironmentInjectsUTF8LocaleWhenMissing(t *testing.T) {
	env := buildShellEnvironment([]string{
		"PATH=/usr/bin",
		"LANG=C",
	})

	expectedLocale := "en_US.UTF-8"
	if runtime.GOOS == "linux" {
		expectedLocale = "C.UTF-8"
	}

	if got, _ := lookupEnv(env, "LANG"); got != expectedLocale {
		t.Fatalf("unexpected LANG: %q", got)
	}
	if got, _ := lookupEnv(env, "LC_CTYPE"); got != expectedLocale {
		t.Fatalf("unexpected LC_CTYPE: %q", got)
	}
}

func TestBuildShellEnvironmentOverridesNonUTF8LCALL(t *testing.T) {
	env := buildShellEnvironment([]string{
		"PATH=/usr/bin",
		"LANG=ko_KR.EUC-KR",
		"LC_ALL=C",
	})

	expectedLocale := "en_US.UTF-8"
	if runtime.GOOS == "linux" {
		expectedLocale = "C.UTF-8"
	}

	if got, _ := lookupEnv(env, "LC_ALL"); got != expectedLocale {
		t.Fatalf("unexpected LC_ALL: %q", got)
	}
}

func TestBuildShellEnvironmentPreservesExistingUTF8Locale(t *testing.T) {
	baseEnv := []string{
		"PATH=/usr/bin",
		"LANG=ko_KR.UTF-8",
		"LC_CTYPE=ko_KR.UTF-8",
	}

	env := buildShellEnvironment(baseEnv)
	if !reflect.DeepEqual(env, baseEnv) {
		t.Fatalf("expected env to be preserved, got %+v", env)
	}
}

func TestResolveDefaultShellPathPrefersConfiguredExecutable(t *testing.T) {
	preferred := filepath.Join(t.TempDir(), "preferred-shell")
	if err := os.WriteFile(preferred, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write preferred shell failed: %v", err)
	}

	resolved := resolveDefaultShellPath(preferred, "/missing/env-shell", "linux")
	if resolved != preferred {
		t.Fatalf("unexpected shell path: got %q want %q", resolved, preferred)
	}
}

func TestResolveDefaultShellPathFallsBackToEnvironmentShell(t *testing.T) {
	envShell := filepath.Join(t.TempDir(), "env-shell")
	if err := os.WriteFile(envShell, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write env shell failed: %v", err)
	}

	resolved := resolveDefaultShellPath("/missing/preferred-shell", envShell, "linux")
	if resolved != envShell {
		t.Fatalf("unexpected shell path: got %q want %q", resolved, envShell)
	}
}

func TestResolveDefaultShellPathFallsBackToPlatformShell(t *testing.T) {
	resolved := resolveDefaultShellPath("/missing/preferred-shell", "/missing/env-shell", "linux")
	if resolved == "" {
		t.Fatal("expected fallback shell path")
	}
	if filepath.Base(resolved) != "bash" && filepath.Base(resolved) != "sh" {
		t.Fatalf("unexpected fallback shell path: %q", resolved)
	}
}
