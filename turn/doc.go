// SPDX-License-Identifier: AGPL-3.0-or-later

// Package turn issues and sanitizes ICE server credentials for Shellin WebRTC
// sessions.
//
// The current implementation supports Cloudflare Calls TURN credentials and
// filters returned ICE URLs before exposing them to clients.
package turn
