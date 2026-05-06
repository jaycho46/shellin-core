// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"time"
)

type connectableBridge interface {
	Connected() bool
	LastError() error
}

func waitForBridgeConnected(bridge connectableBridge, timeout time.Duration) error {
	if bridge == nil {
		return errors.New("bridge is nil")
	}
	deadline := time.Now().Add(timeout)
	for {
		if bridge.Connected() {
			return nil
		}
		if err := bridge.LastError(); isNonRetryableControlPlaneError(err) {
			return err
		}
		if timeout > 0 && time.Now().After(deadline) {
			if err := bridge.LastError(); err != nil {
				return err
			}
			return fmt.Errorf("timed out after %s", timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
