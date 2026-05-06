#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:?usage: collect-third-party-licenses.sh <output-dir>}"
LICENSE_DIR="$OUT_DIR/THIRD_PARTY_LICENSES"
NOTICE_FILE="$OUT_DIR/THIRD_PARTY_NOTICES.md"

platforms=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
)

rm -rf "$LICENSE_DIR"
mkdir -p "$LICENSE_DIR"

tmp_modules="$(mktemp "${TMPDIR:-/tmp}/shellin-third-party-modules.XXXXXX")"
trap 'rm -f "$tmp_modules"' EXIT

for platform in "${platforms[@]}"; do
  read -r goos goarch <<<"$platform"
  (
    cd "$ROOT_DIR"
    GOOS="$goos" GOARCH="$goarch" go list -deps \
      -f '{{with .Module}}{{printf "%s\t%s\t%s" .Path .Version .Dir}}{{end}}' \
      ./cmd/shellin
  )
done | sort -u >"$tmp_modules"

{
  echo "# Third-Party Notices"
  echo
  echo "This file lists third-party Go modules included in Shellin CLI release"
  echo "archives, plus the Go standard library and runtime used to build the"
  echo "binary. The corresponding license files are included under"
  echo "\`THIRD_PARTY_LICENSES/\` in each archive."
  echo
} >"$NOTICE_FILE"

go_root="$(go env GOROOT)"
if [[ -f "$go_root/LICENSE" ]]; then
  mkdir -p "$LICENSE_DIR/golang.org/toolchain"
  cp "$go_root/LICENSE" "$LICENSE_DIR/golang.org/toolchain/LICENSE"
  if [[ -f "$go_root/PATENTS" ]]; then
    cp "$go_root/PATENTS" "$LICENSE_DIR/golang.org/toolchain/PATENTS"
  fi
  echo "## Go standard library and runtime" >>"$NOTICE_FILE"
  echo >>"$NOTICE_FILE"
  echo "- $(go version)" >>"$NOTICE_FILE"
  echo "- THIRD_PARTY_LICENSES/golang.org/toolchain/LICENSE" >>"$NOTICE_FILE"
  if [[ -f "$go_root/PATENTS" ]]; then
    echo "- THIRD_PARTY_LICENSES/golang.org/toolchain/PATENTS" >>"$NOTICE_FILE"
  fi
  echo >>"$NOTICE_FILE"
fi

while IFS=$'\t' read -r module version dir; do
  [[ -n "$module" && -n "$dir" ]] || continue
  [[ "$module" == "github.com/jaycho46/shellin-core" ]] && continue

  tmp_licenses="$(mktemp "${TMPDIR:-/tmp}/shellin-third-party-licenses.XXXXXX")"
  find "$dir" -maxdepth 2 -type f \
    \( -iname 'LICENSE*' -o -iname 'COPYING*' -o -iname 'NOTICE*' \) |
    sort >"$tmp_licenses"
  [[ -s "$tmp_licenses" ]] || {
    rm -f "$tmp_licenses"
    continue
  }

  module_dir="$LICENSE_DIR/$module"
  mkdir -p "$module_dir"

  version_label="$version"
  [[ -n "$version_label" ]] || version_label="local"
  echo "## $module $version_label" >>"$NOTICE_FILE"
  echo >>"$NOTICE_FILE"

  while IFS= read -r license_file; do
    rel_path="${license_file#"$dir"/}"
    mkdir -p "$module_dir/$(dirname "$rel_path")"
    cp "$license_file" "$module_dir/$rel_path"
    echo "- THIRD_PARTY_LICENSES/$module/$rel_path" >>"$NOTICE_FILE"
  done <"$tmp_licenses"
  echo >>"$NOTICE_FILE"
  rm -f "$tmp_licenses"
done <"$tmp_modules"
