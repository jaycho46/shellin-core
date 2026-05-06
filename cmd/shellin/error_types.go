// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

type controlPlaneStatusError struct {
	Operation  string
	StatusCode int
	Body       string
}

type nonRetryableControlPlaneError struct {
	err error
}

type sessionTerminatedError struct {
	err error
}

type subscriptionInactiveError struct {
	status string
}

func (e *controlPlaneStatusError) Error() string {
	if e == nil {
		return ""
	}
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("control plane %s status %d", e.Operation, e.StatusCode)
	}
	return fmt.Sprintf("control plane %s status %d: %s", e.Operation, e.StatusCode, body)
}

func (e *nonRetryableControlPlaneError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *nonRetryableControlPlaneError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *sessionTerminatedError) Error() string {
	if e == nil {
		return ""
	}
	return "session was terminated remotely"
}

func (e *sessionTerminatedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *subscriptionInactiveError) Error() string {
	if e == nil {
		return ""
	}
	return inactiveSubscriptionSubtitle(e.status)
}

func markControlPlaneErrorNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	var permanent *nonRetryableControlPlaneError
	if errors.As(err, &permanent) {
		return err
	}
	return &nonRetryableControlPlaneError{err: err}
}

func markControlPlaneErrorSessionTerminated(err error) error {
	if err == nil {
		return nil
	}
	var terminated *sessionTerminatedError
	if errors.As(err, &terminated) {
		return err
	}
	return &sessionTerminatedError{err: err}
}

func isNonRetryableControlPlaneError(err error) bool {
	var permanent *nonRetryableControlPlaneError
	return errors.As(err, &permanent)
}
