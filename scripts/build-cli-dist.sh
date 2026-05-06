#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/dist}"
RELEASE_VERSION="${SHELLIN_RELEASE_VERSION:-${GITHUB_REF_NAME:-dev}}"
RELEASE_BASE_URL="${SHELLIN_RELEASE_BASE_URL:-https://shellin.dev}"
UPDATE_PUBLIC_KEY="${SHELLIN_UPDATE_PUBLIC_KEY:-}"
UPDATE_PRIVATE_KEY="${SHELLIN_UPDATE_PRIVATE_KEY:-}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
COMMIT="$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || true)"

if [[ -z "$UPDATE_PUBLIC_KEY" ]]; then
  echo "SHELLIN_UPDATE_PUBLIC_KEY is required to embed the release manifest verification key" >&2
  exit 1
fi
if [[ -z "$UPDATE_PRIVATE_KEY" ]]; then
  echo "SHELLIN_UPDATE_PRIVATE_KEY is required to sign latest.json" >&2
  exit 1
fi
unset SHELLIN_UPDATE_PRIVATE_KEY
export -n UPDATE_PRIVATE_KEY 2>/dev/null || true

mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"
: >"$OUT_DIR/checksums.txt"

build_archive() {
  local goos="$1"
  local goarch="$2"
  local archive_name="shellin-${goos}-${goarch}.tar.gz"
  local archive_path="$OUT_DIR/$archive_name"
  local tmp_dir
  tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/shellin-core-dist.XXXXXX")"

  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
      go build -trimpath \
        -ldflags "-s -w -X main.defaultCLIVersion=$RELEASE_VERSION -X main.defaultDownloadBaseURL=$RELEASE_BASE_URL -X main.defaultReleaseManifestPublicKey=$UPDATE_PUBLIC_KEY -X github.com/jaycho46/shellin-core/buildinfo.Version=$RELEASE_VERSION -X github.com/jaycho46/shellin-core/buildinfo.ServiceReleaseID=$RELEASE_VERSION -X github.com/jaycho46/shellin-core/buildinfo.Commit=$COMMIT -X github.com/jaycho46/shellin-core/buildinfo.BuildTime=$BUILD_TIME" \
        -o "$tmp_dir/shellin" ./cmd/shellin
  )
  cp "$ROOT_DIR/LICENSE" "$tmp_dir/LICENSE"
  cp "$ROOT_DIR/NOTICE" "$tmp_dir/NOTICE"
  cp "$ROOT_DIR/COMMERCIAL_LICENSE.md" "$tmp_dir/COMMERCIAL_LICENSE.md"
  mkdir -p "$tmp_dir/LICENSES"
  cp "$ROOT_DIR/LICENSES/AGPL-3.0-or-later.txt" "$tmp_dir/LICENSES/AGPL-3.0-or-later.txt"
  "$ROOT_DIR/scripts/collect-third-party-licenses.sh" "$tmp_dir"
  {
    echo "Shellin Core source code for this release:"
    echo "https://github.com/jaycho46/shellin-core/tree/$RELEASE_VERSION"
    if [[ -n "$COMMIT" ]]; then
      echo
      echo "Build commit:"
      echo "https://github.com/jaycho46/shellin-core/commit/$COMMIT"
    fi
  } >"$tmp_dir/SOURCE.txt"
  tar -C "$tmp_dir" -czf "$archive_path" \
    shellin \
    LICENSE \
    NOTICE \
    COMMERCIAL_LICENSE.md \
    LICENSES \
    SOURCE.txt \
    THIRD_PARTY_NOTICES.md \
    THIRD_PARTY_LICENSES
  shasum -a 256 "$archive_path" | tee "$archive_path.sha256" >>"$OUT_DIR/checksums.txt"
  rm -rf "$tmp_dir"
}

build_archive darwin amd64
build_archive darwin arm64
build_archive linux amd64
build_archive linux arm64

printf '%s' "$UPDATE_PRIVATE_KEY" | (
  cd "$ROOT_DIR"
  env -u SHELLIN_UPDATE_PRIVATE_KEY -u UPDATE_PRIVATE_KEY \
    go run ./scripts/sign-release-manifest \
    "$OUT_DIR" "$RELEASE_VERSION" "$RELEASE_BASE_URL" "$BUILD_TIME" "$UPDATE_PUBLIC_KEY"
)
