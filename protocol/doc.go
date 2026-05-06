// SPDX-License-Identifier: AGPL-3.0-or-later

// Package protocol defines the JSON messages shared by the Shellin CLI,
// control plane, signaling server, and viewer clients.
//
// The types in this package are intentionally transport-only DTOs. Validation,
// authorization, and persistence rules live in the packages that own those
// boundaries.
package protocol
