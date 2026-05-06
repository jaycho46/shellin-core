# Build Provenance

Shellin Core reports build metadata from two places:

- `shellin version --json`
- `GET /version` on the control plane

## Fields

- `service`
- `service_release_id`
- `core_module`
- `core_version`
- `core_sum`
- `build_time`
- `go_version`
- `vcs_revision`
- `vcs_modified`
- `binary_sha256`

For hosted production, `core_module`, `core_version`, and `core_sum` should
match a public release of `github.com/jaycho46/shellin-core`.

## Release Artifacts

`scripts/build-cli-dist.sh` produces release archives. Each archive includes:

- the `shellin` binary
- `LICENSE`, `NOTICE`, and `COMMERCIAL_LICENSE.md`
- `SOURCE.txt` with the release source link and build commit
- third-party notices and copied license files

`scripts/sign-release-manifest` writes `latest.json` with archive URLs,
SHA-256 checksums, and an Ed25519 signature. The CLI verifies the manifest and
archive checksum through `internal/selfupdate` before replacing the local
binary.

## Review Checklist

For a release review, compare:

- `latest.json` signature and archive checksums
- archive checksum against `checksums.txt`
- `SOURCE.txt` release tag and commit
- `shellin version --json` output from the binary
- hosted `GET /version` output, when reviewing the deployed service

`/version` is self-reported by the running service. Treat it as audit evidence,
not remote attestation.
