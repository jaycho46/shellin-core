# Threat Model

## Trust Boundaries

- The desktop user trusts the local `shellin` binary to start and bridge a PTY.
- The viewer trusts the mobile app to render terminal output and send intended
  input.
- The control plane is trusted for auth, session discovery, attach bootstrap,
  signaling, and ICE/TURN configuration.
- The control plane should not need terminal plaintext to coordinate a session.

## Server Visibility

The control plane can see:

- account and session identifiers
- agent labels
- auth and signaling token metadata
- SDP and ICE signaling messages
- TURN credential issuance metadata

The control plane should not see:

- terminal keystrokes
- terminal output bytes
- PTY scrollback

## Sensitive Code Paths

- Grant token signing and validation: `grantauth/signer.go`,
  `grantauth/validator.go`, and `cmd/shellin-control-plane/tokens.go`.
- Signaling replay protection: `grantauth/replay_guard.go` and
  `signaling/websocket.go`.
- WebSocket origin handling: `internal/httporigin/httporigin.go` and
  `signaling/websocket.go`.
- Agent registry limits and termination tombstones:
  `controlplane/agent_registry*.go`.
- Terminal byte transport: `cmd/shellin/webrtc_data_channels.go`,
  `cmd/shellin/terminal_pump.go`, and `cmd/shellin/pty.go`.
- TURN credential issuance and filtering: `turn/cloudflare.go`.

## Controls

- Signaling auth requires a bearer token header.
- Query-string signaling tokens are rejected.
- Signaling token IDs are consumed through a replay store before WebSocket
  upgrade.
- Viewer signaling tokens are rejected unless the target agent session exists
  and is fresh.
- External signal URLs and WebSocket origins can be pinned with an explicit
  external base URL.
- Without an explicit external base URL, forwarded host/proto headers are trusted
  only from loopback reverse proxies.
- Non-loopback HTTP external URLs are rejected.
- User-key exchange and device-login failures are rate-limited per remote
  address in the demo control plane.
- Cloudflare ICE responses are sanitized before clients receive them.
- Refresh tokens and one-time login keys are stored as hashes in the demo stores
  and consumed atomically in process.

## Scope Limits

- The iOS app, StoreKit purchase flow, and App Store Server API integration live
  outside this repository.
- Hosted deployments must provide shared, durable storage for replay protection,
  refresh tokens, login keys, entitlement state, and agent registry state.
- TURN relay usage can expose connection metadata even though WebRTC protects
  DataChannel contents.
- Build provenance links a binary to source and release artifacts, but it does
  not replace third-party security review.
