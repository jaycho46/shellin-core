// SPDX-License-Identifier: AGPL-3.0-or-later

package selfupdate

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	latestManifestPath       = "/downloads/latest.json"
	releaseManifestSigningV1 = "shellin-release-manifest-v1"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

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

func fetchLatestReleaseManifest(baseURL, publicKey string) (releaseManifest, error) {
	manifestURL := strings.TrimRight(strings.TrimSpace(baseURL), "/") + latestManifestPath
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return releaseManifest{}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return releaseManifest{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return releaseManifest{}, fmt.Errorf("fetch latest release manifest status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var manifest releaseManifest
	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return releaseManifest{}, fmt.Errorf("decode latest release manifest failed: %w", err)
	}
	if err := verifyReleaseManifestSignature(manifest, publicKey); err != nil {
		return releaseManifest{}, err
	}
	return manifest, nil
}

func verifyReleaseManifestSignature(manifest releaseManifest, publicKey string) error {
	signature := strings.TrimSpace(manifest.Signature)
	if signature == "" {
		return errors.New("latest release manifest is missing signature")
	}
	pub, err := decodeReleaseManifestPublicKey(publicKey)
	if err != nil {
		return err
	}
	sig, err := decodeBase64(signature)
	if err != nil {
		return fmt.Errorf("decode latest release manifest signature failed: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("latest release manifest signature has invalid length: %d", len(sig))
	}
	if !ed25519.Verify(pub, releaseManifestSigningPayload(manifest), sig) {
		return errors.New("latest release manifest signature verification failed")
	}
	return nil
}

func decodeReleaseManifestPublicKey(raw string) (ed25519.PublicKey, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "ed25519:"))
	if raw == "" {
		return nil, errors.New("release manifest public key is not configured")
	}
	decoded, err := decodeBase64(raw)
	if err != nil {
		return nil, fmt.Errorf("decode release manifest public key failed: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("release manifest public key has invalid length: %d", len(decoded))
	}
	return ed25519.PublicKey(decoded), nil
}

func decodeBase64(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, enc := range encodings {
		decoded, err := enc.DecodeString(raw)
		if err == nil {
			return decoded, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func releaseManifestSigningPayload(manifest releaseManifest) []byte {
	var buf strings.Builder
	buf.WriteString(releaseManifestSigningV1)
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

func (m releaseManifest) assetForPlatform(goos, goarch string) (releaseAsset, error) {
	key := strings.TrimSpace(goos) + "/" + strings.TrimSpace(goarch)
	if asset, ok := m.Platforms[key]; ok {
		if strings.TrimSpace(asset.URL) == "" {
			return releaseAsset{}, fmt.Errorf("latest release asset URL missing for %s", key)
		}
		if strings.TrimSpace(asset.SHA256) == "" {
			return releaseAsset{}, fmt.Errorf("latest release checksum missing for %s", key)
		}
		return asset, nil
	}
	return releaseAsset{}, fmt.Errorf("no release available for %s", key)
}
