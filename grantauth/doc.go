// SPDX-License-Identifier: AGPL-3.0-or-later

// Package grantauth signs and validates short-lived Shellin grant tokens.
//
// Tokens are HMAC-SHA256 JWTs with issuer, audience, subject, expiry, token ID,
// role, and session claims. Signaling tokens must also pass replay protection
// through a ReplayStore before a WebSocket is accepted.
package grantauth
