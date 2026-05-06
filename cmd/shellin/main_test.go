// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type stubConnectableBridge struct {
	connected bool
	lastErr   error
}

func (b stubConnectableBridge) Connected() bool { return b.connected }

func (b stubConnectableBridge) LastError() error { return b.lastErr }

func TestWaitForBridgeConnectedReturnsNilWhenConnected(t *testing.T) {
	if err := waitForBridgeConnected(stubConnectableBridge{connected: true}, time.Second); err != nil {
		t.Fatalf("waitForBridgeConnected returned error: %v", err)
	}
}

func TestWaitForBridgeConnectedReturnsLastErrorOnTimeout(t *testing.T) {
	want := errors.New("dial signaling websocket failed: EOF")
	err := waitForBridgeConnected(stubConnectableBridge{lastErr: want}, 10*time.Millisecond)
	if !errors.Is(err, want) {
		t.Fatalf("expected wrapped last error, got %v", err)
	}
}

func TestWaitForBridgeConnectedReturnsNonRetryableErrorImmediately(t *testing.T) {
	want := markControlPlaneErrorNonRetryable(errors.New("control plane connect status 401: unauthorized"))
	start := time.Now()
	err := waitForBridgeConnected(stubConnectableBridge{lastErr: want}, time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("expected non-retryable error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("expected immediate return, took %s", elapsed)
	}
}

func TestWaitForBridgeConnectedReturnsTimeoutWhenNoError(t *testing.T) {
	err := waitForBridgeConnected(stubConnectableBridge{}, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out after") {
		t.Fatalf("unexpected error: %v", err)
	}
}
