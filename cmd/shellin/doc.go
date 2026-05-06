// SPDX-License-Identifier: AGPL-3.0-or-later

// Command shellin starts the local Shellin agent CLI.
//
// The command owns the local PTY, obtains control-plane auth, registers the
// agent session, negotiates WebRTC with a viewer, and bridges terminal bytes over
// WebRTC DataChannels.
package main
