// SPDX-License-Identifier: AGPL-3.0-or-later

// Package signaling hosts the authenticated WebSocket relay used to exchange
// WebRTC offers, answers, ICE candidates, and session lifecycle hints.
//
// Terminal bytes are not carried by this package. They are sent over WebRTC data
// channels after signaling completes.
package signaling
