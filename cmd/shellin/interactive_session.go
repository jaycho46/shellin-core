// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"log"
	"os"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

func runInteractiveSession() {
	cfg := parseInteractiveConfig()

	if err := maybePromptForCLIUpdate(); err != nil {
		log.Printf("warning: update check failed: %v", err)
	}

	useHUD := cfg.HUD && terminalSupportsHUD()
	stdout := &lockedStdout{out: os.Stdout}
	startup := newStartupScreen(useHUD, stdout)
	status := newStatusPresenter(nil)
	if startup.Enabled() {
		status.SetSink(startup)
		startup.SetStage(startupStageAuthorizing)
		startup.Start()
	}

	initialAuth, profilePath, err := resolveControlPlaneAuth(cfg)
	if err != nil {
		fatalStartupError(startup, "session authorization failed", err)
	}
	preflightStatus, finalAuth, err := requestSessionStatusWithAuthSession(cfg, initialAuth, profilePath)
	if err != nil {
		fatalStartupError(startup, "session authorization failed", err)
	}
	if err := persistControlPlaneAuthUpdate(profilePath, cfg.ControlPlaneURL, initialAuth, finalAuth); err != nil {
		log.Printf("warning: failed to update auth profile after auth refresh: %v", err)
	}
	initialAuth = finalAuth
	if startup.Enabled() {
		startup.SetStage(startupStageCheckingLimits)
	}
	if err := validateControlPlanePreflightStatus(preflightStatus); err != nil {
		fatalStartupError(startup, "session authorization failed", err)
	}
	if startup.Enabled() {
		startup.SetStage(startupStageCreatingSession)
	}
	status.SetP2PState("waiting")

	sessionID := uuid.NewString()
	labelTracker := newAgentLabelTracker(defaultAgentLabel())

	cmd := buildShellCommand(cfg.Shell)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("failed to start shell: %v", err)
	}
	defer func() { _ = ptmx.Close() }()

	runtimeFatal := newRuntimeFatalStopper(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = ptmx.Close()
	})
	registryHeartbeat := newAgentRegistryHeartbeat(
		cfg,
		profilePath,
		sessionID,
		initialAuth,
		labelTracker.Get,
		runtimeFatal.Set,
	)

	bridge := newWebRTCAgentBridge(
		cfg,
		ptmx,
		status,
		sessionID,
		initialAuth,
		profilePath,
		labelTracker.Get,
		func(next controlPlaneAuthSession, mode controlPlaneAuthUpdateMode) {
			registryHeartbeat.ApplyAuthUpdate(next, mode)
		},
	)
	bridge.Start()
	defer bridge.Close()

	if err := waitForBridgeConnected(bridge, initialBridgeConnectTimeout); err != nil {
		fatalStartupError(startup, "failed to establish WebRTC agent session", err)
	}
	startup.Stop()
	status.SetSink(nil)

	if err := registryHeartbeat.Start(); err != nil {
		log.Printf("warning: failed to register agent listing: %v", err)
	} else {
		defer registryHeartbeat.Stop()
	}

	hud := newBottomHUD(useHUD, stdout, hudRightText(cfg.Shell, sessionID))
	if hud.Enabled() {
		hud.Start()
		status.SetSink(hud)
		defer hud.Stop()
	}

	stopResizeWatcher := startTerminalResizeWatcher(ptmx, hud)
	defer stopResizeWatcher()

	restore := enterRawTerminal()
	defer func() {
		if restore != nil {
			restore()
		}
	}()

	startTerminalInputPump(ptmx)
	pumpPTYOutput(ptmx, stdout, hud, bridge, labelTracker, registryHeartbeat)

	waitErr := cmd.Wait()
	if hud.Enabled() {
		hud.Stop()
		status.SetSink(nil)
	}
	if restore != nil {
		restore()
		restore = nil
	}
	if fatalErr := runtimeFatal.Err(); fatalErr != nil {
		fatalCLIError(fatalErr)
	}
	if msg, ok := shellExitMessage(waitErr); ok {
		_, _ = stdout.WriteString("\n" + msg + "\n\n")
		return
	}
	if waitErr != nil {
		log.Printf("shell exited: %v", waitErr)
	}
}
