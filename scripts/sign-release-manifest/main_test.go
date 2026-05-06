// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReadsPrivateKeyFromStdin(t *testing.T) {
	outDir := t.TempDir()
	for _, name := range []string{
		"shellin-darwin-amd64.tar.gz",
		"shellin-darwin-arm64.tar.gz",
		"shellin-linux-amd64.tar.gz",
		"shellin-linux-arm64.tar.gz",
	} {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte("archive:"+name), 0o644); err != nil {
			t.Fatalf("write archive fixture: %v", err)
		}
	}

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	publicKeyRaw := base64.StdEncoding.EncodeToString(publicKey)
	privateKeyRaw := base64.StdEncoding.EncodeToString(privateKey)

	err = run(
		[]string{outDir, "v1.2.3", "https://shellin.dev", "2026-05-06T00:00:00Z", publicKeyRaw},
		strings.NewReader(privateKeyRaw+"\n"),
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "latest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest releaseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Version != "v1.2.3" {
		t.Fatalf("unexpected version: %q", manifest.Version)
	}
	signature, err := decodeBase64(manifest.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if !ed25519.Verify(publicKey, signingPayload(manifest), signature) {
		t.Fatal("manifest signature did not verify")
	}
	if got := manifest.Platforms["darwin/arm64"].URL; got != "https://shellin.dev/downloads/shellin-darwin-arm64.tar.gz" {
		t.Fatalf("unexpected asset URL: %q", got)
	}
}

func TestReadPrivateKeyRejectsEmptyStdin(t *testing.T) {
	if _, err := readPrivateKey(strings.NewReader(" \n\t")); err == nil {
		t.Fatal("expected empty stdin to fail")
	}
}

func TestRunRejectsLegacyPrivateKeyArgument(t *testing.T) {
	err := run(
		[]string{"out", "v1.2.3", "https://shellin.dev", "2026-05-06T00:00:00Z", "public-key", "private-key"},
		strings.NewReader(""),
	)
	if err == nil {
		t.Fatal("expected legacy private-key argv form to fail")
	}
}
