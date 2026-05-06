// SPDX-License-Identifier: AGPL-3.0-or-later

// Package controlplane contains the auditable in-memory control-plane model used
// by the local Shellin demo server.
//
// Hosted deployments should provide durable implementations around these
// primitives for entitlement, refresh-token, login-key, signaling replay, and
// agent registry state. The package keeps those state transitions explicit so
// production adapters can be reviewed against the same behavior.
package controlplane
