// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"net/http"
	"strings"
)

func isUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}
	var statusErr *controlPlaneStatusError
	if errors.As(err, &statusErr) && statusErr != nil {
		return statusErr.StatusCode == http.StatusUnauthorized
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 401") || strings.Contains(msg, "unauthorized")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var statusErr *controlPlaneStatusError
	if errors.As(err, &statusErr) && statusErr != nil {
		return statusErr.StatusCode == http.StatusNotFound
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") || strings.Contains(msg, "not found")
}

func isGoneError(err error) bool {
	if err == nil {
		return false
	}
	var statusErr *controlPlaneStatusError
	if errors.As(err, &statusErr) && statusErr != nil {
		return statusErr.StatusCode == http.StatusGone
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 410")
}

func terminatedSessionRuntimeError(err error) error {
	if !isGoneError(err) {
		return nil
	}
	return markControlPlaneErrorNonRetryable(markControlPlaneErrorSessionTerminated(err))
}
