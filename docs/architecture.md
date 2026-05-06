# Architecture

## Runtime Roles

- `agent`: the desktop `shellin` CLI that owns the local PTY.
- `viewer`: the mobile client that renders terminal output and sends input.
- `control-plane`: the HTTP/WebSocket service that handles auth, agent listing,
  attach bootstrap, signaling, and ICE configuration.

## Session Flow

1. The CLI loads config and obtains control-plane auth.
2. The CLI checks `/v1/status` before starting a session.
3. The CLI starts the local PTY, creates a session ID, and registers the agent.
4. `POST /v1/agents/connect` returns an agent signaling token, signal URL, and
   ICE server list.
5. The CLI opens `GET /v1/agents/signal` with an `Authorization: Bearer ...`
   header.
6. The viewer calls `POST /v1/agents/attach` and opens the same signaling
   endpoint with a viewer token.
7. The signaling package validates claims, consumes the token ID through replay
   protection, and forwards `protocol.SignalMessage` values between peers.
8. WebRTC negotiation creates `terminal` and `control` DataChannels.

Terminal bytes flow over the `terminal` DataChannel. Resize and control
messages flow over the `control` DataChannel. Terminal bytes do not pass through
the control plane or the signaling WebSocket.

## Package Boundaries

- `cmd/shellin`: CLI entrypoint, auth flow, PTY ownership, terminal UI, WebRTC
  agent orchestration, and update prompts.
- `cmd/shellin-control-plane`: local HTTP server with process-local stores.
- `protocol`: JSON DTOs and constants shared by agent, viewer, and control
  plane code.
- `grantauth`: HMAC grant token signing, validation, claim parsing, and replay
  store interfaces.
- `controlplane`: entitlement, refresh-token, login-key, agent registry, and
  signaling hub state transitions.
- `signaling`: authenticated WebSocket relay for WebRTC negotiation messages.
- `turn`: Cloudflare TURN credential issuance and ICE server sanitization.
- `buildinfo`: release/source provenance used by CLI and control-plane version
  endpoints.
- `internal/httporigin`: external URL and WebSocket origin handling.
- `internal/selfupdate`: signed release manifest verification, archive checksum
  validation, binary replacement, and release version comparison.

## Adapter Boundary

The demo control plane uses in-memory stores so the core behavior can run
locally. Hosted deployments replace those stores with durable adapters for:

- entitlement lookup and updates
- refresh token issue/consume/cleanup
- one-time device login keys
- signaling token replay storage
- agent registry and heartbeat state

Multi-instance deployments need shared replay storage. A signaling token ID must
be accepted once globally, not once per process.

The hosted Shellin service also adds the iOS app, StoreKit/App Store handling,
DynamoDB stores, deployment assets, and web pages around this core.

## Verification

Package tests cover the main contracts:

- `signaling/websocket_test.go`: bearer auth, origin handling, peer forwarding,
  and replay rejection.
- `turn/cloudflare_test.go`: TURN credential request construction and ICE server
  sanitization.
- `grantauth/validator_test.go`: token signature, issuer/audience, expiry,
  not-before, token ID, and weak secret rejection.
- `controlplane/*_test.go`: login key, refresh token, and agent registry
  behavior.
- `buildinfo/buildinfo_test.go`: version response shape and release fields.

Local validation:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/shellin ./cmd/shellin-control-plane
git diff --check
```
