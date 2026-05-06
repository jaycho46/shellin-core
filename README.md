# Shellin Core

## What Is Shellin?

Shellin lets you use an existing desktop terminal from an iPhone or iPad. The
desktop runs the `shellin` CLI, which owns the local PTY. The iOS app connects
to that session over WebRTC DataChannels and exchanges terminal input/output,
not a remote desktop video stream.

- Service: [shellin.dev](https://shellin.dev)
- iOS app: [shellin on the App Store](https://apps.apple.com/app/shellin/id6760250186)

This repository is the public Go core used by the Shellin CLI and hosted
service. It contains the shared protocol messages, grant-token validation,
signaling WebSocket handling, TURN response filtering, demo control-plane state,
CLI update verification, and release provenance code.

The production app, website, billing code, subscription reconciliation,
deployment assets, and persistent stores live outside this repository.

## Build And Test

This module targets Go `1.24.2`.

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/shellin ./cmd/shellin-control-plane
git diff --check
```

CI runs the same baseline.

## Run Locally

Start the demo control plane:

```bash
SHELLIN_GRANT_HMAC_SECRET="$(openssl rand -base64 32)" \
SHELLIN_USER_KEYS='uk_local:demo-user:1' \
go run ./cmd/shellin-control-plane
```

Build the CLI against the local control plane:

```bash
go build \
  -ldflags "-X main.defaultControlPlaneURL=http://127.0.0.1:8090 -X main.defaultAllowInsecureURL=true" \
  -o ./output/shellin \
  ./cmd/shellin
```

Log in and start an agent session:

```bash
./output/shellin login uk_local
./output/shellin
```

## Docs

- [Architecture](docs/architecture.md)
- [Threat model](docs/threat-model.md)
- [Release provenance](docs/provenance.md)

## License

Shellin Core is licensed to the public under AGPL-3.0-or-later. The copyright
holder may also use, distribute, and license the same code under separate
commercial or internal terms.

See `LICENSE`, `NOTICE`, `COMMERCIAL_LICENSE.md`, and
[docs/licensing.md](docs/licensing.md).
