// SPDX-License-Identifier: AGPL-3.0-or-later

// Command shellin-control-plane runs the local audit/demo control plane.
//
// It wires the public Shellin Core packages together with process-local stores.
// Hosted deployments should replace those stores with durable adapters while
// preserving the same protocol and validation behavior.
package main
