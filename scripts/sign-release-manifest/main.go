// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const signingContext = "shellin-release-manifest-v1"

type releaseManifest struct {
	Version     string                  `json:"version"`
	PublishedAt string                  `json:"published_at,omitempty"`
	Platforms   map[string]releaseAsset `json:"platforms"`
	Signature   string                  `json:"signature"`
}

type releaseAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func main() {
	if err := run(os.Args[1:], os.Stdin); err != nil {
		fatalf("%v", err)
	}
}

func run(args []string, stdin io.Reader) error {
	if len(args) != 5 {
		return errors.New("usage: sign-release-manifest <out-dir> <version> <base-url> <published-at> <public-key> (private key on stdin)")
	}
	outDir := args[0]
	version := args[1]
	baseURL := strings.TrimRight(args[2], "/")
	publishedAt := args[3]
	privateKeyRaw, err := readPrivateKey(stdin)
	if err != nil {
		return err
	}
	privateKey, err := decodePrivateKey(privateKeyRaw)
	if err != nil {
		return err
	}
	publicKey, err := decodePublicKey(args[4])
	if err != nil {
		return err
	}
	if !bytes.Equal(privateKey.Public().(ed25519.PublicKey), publicKey) {
		return errors.New("SHELLIN_UPDATE_PUBLIC_KEY does not match SHELLIN_UPDATE_PRIVATE_KEY")
	}

	platforms := make(map[string]releaseAsset)
	for _, pair := range [][2]string{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	} {
		goos, goarch := pair[0], pair[1]
		name := fmt.Sprintf("shellin-%s-%s.tar.gz", goos, goarch)
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		sum := sha256.Sum256(data)
		platforms[goos+"/"+goarch] = releaseAsset{
			URL:    fmt.Sprintf("%s/downloads/%s", baseURL, name),
			SHA256: hex.EncodeToString(sum[:]),
		}
	}

	manifest := releaseManifest{
		Version:     version,
		PublishedAt: publishedAt,
		Platforms:   platforms,
	}
	manifest.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, signingPayload(manifest)))
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode latest.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "latest.json"), append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write latest.json: %w", err)
	}
	return nil
}

func readPrivateKey(r io.Reader) (string, error) {
	if r == nil {
		return "", errors.New("SHELLIN_UPDATE_PRIVATE_KEY is required on stdin")
	}
	data, err := io.ReadAll(io.LimitReader(r, 8*1024))
	if err != nil {
		return "", fmt.Errorf("read update private key from stdin: %w", err)
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", errors.New("SHELLIN_UPDATE_PRIVATE_KEY is required on stdin")
	}
	return secret, nil
}

func signingPayload(manifest releaseManifest) []byte {
	var buf strings.Builder
	buf.WriteString(signingContext)
	buf.WriteByte('\n')
	buf.WriteString("version:")
	buf.WriteString(strings.TrimSpace(manifest.Version))
	buf.WriteByte('\n')
	buf.WriteString("published_at:")
	buf.WriteString(strings.TrimSpace(manifest.PublishedAt))
	buf.WriteByte('\n')

	keys := make([]string, 0, len(manifest.Platforms))
	for key := range manifest.Platforms {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		asset := manifest.Platforms[key]
		buf.WriteString("platform:")
		buf.WriteString(strings.TrimSpace(key))
		buf.WriteByte('\n')
		buf.WriteString("url:")
		buf.WriteString(strings.TrimSpace(asset.URL))
		buf.WriteByte('\n')
		buf.WriteString("sha256:")
		buf.WriteString(strings.ToLower(strings.TrimSpace(asset.SHA256)))
		buf.WriteByte('\n')
	}
	return []byte(buf.String())
}

func decodePrivateKey(raw string) (ed25519.PrivateKey, error) {
	decoded, err := decodeBase64(strings.TrimPrefix(strings.TrimSpace(raw), "ed25519:"))
	if err != nil {
		return nil, fmt.Errorf("decode update private key: %w", err)
	}
	switch len(decoded) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(decoded), nil
	default:
		return nil, fmt.Errorf("update private key has invalid length: %d", len(decoded))
	}
}

func decodePublicKey(raw string) (ed25519.PublicKey, error) {
	decoded, err := decodeBase64(strings.TrimPrefix(strings.TrimSpace(raw), "ed25519:"))
	if err != nil {
		return nil, fmt.Errorf("decode update public key: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("update public key has invalid length: %d", len(decoded))
	}
	return ed25519.PublicKey(decoded), nil
}

func decodeBase64(raw string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		decoded, err := enc.DecodeString(strings.TrimSpace(raw))
		if err == nil {
			return decoded, nil
		}
	}
	return nil, errors.New("invalid base64")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
