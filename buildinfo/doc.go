// SPDX-License-Identifier: AGPL-3.0-or-later

// Package buildinfo reports release and source provenance for Shellin binaries.
//
// The package is intentionally small and side-effect free apart from reading the
// current executable hash in Snapshot. Command packages set release fields with
// Go linker flags during the release build.
package buildinfo
