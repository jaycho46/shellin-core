// SPDX-License-Identifier: AGPL-3.0-or-later

package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompareReleaseVersions(t *testing.T) {
	t.Parallel()

	cmp, err := compareReleaseVersions("1.2.3", "1.3.0")
	if err != nil {
		t.Fatalf("compareReleaseVersions failed: %v", err)
	}
	if cmp >= 0 {
		t.Fatalf("expected latest version to be newer, got %d", cmp)
	}

	cmp, err = compareReleaseVersions("v1.3.0", "1.3.0")
	if err != nil {
		t.Fatalf("compareReleaseVersions failed: %v", err)
	}
	if cmp != 0 {
		t.Fatalf("expected equal versions, got %d", cmp)
	}

	cmp, err = compareReleaseVersions("1.3.0-rc1", "1.3.0")
	if err != nil {
		t.Fatalf("compareReleaseVersions failed: %v", err)
	}
	if cmp >= 0 {
		t.Fatalf("expected prerelease to compare lower, got %d", cmp)
	}
}

func TestCheckForAvailableUpdateFindsNewRelease(t *testing.T) {
	t.Parallel()

	archiveData := makeReleaseArchive(t, []byte("new-binary"))
	archiveChecksum := sha256.Sum256(archiveData)
	server, publicKey := newReleaseServer(t, "1.1.0", archiveData, hex.EncodeToString(archiveChecksum[:]))

	result, err := Check(Options{
		AllowUnknownCurrent: false,
		BaseURL:             server.URL,
		CurrentVersion:      "1.0.0",
		ManifestPublicKey:   publicKey,
	})
	if err != nil {
		t.Fatalf("checkForAvailableUpdate failed: %v", err)
	}
	if !result.Checked || !result.UpdateAvailable {
		t.Fatalf("expected update to be available, got %+v", result)
	}
	if result.LatestVersion != "1.1.0" {
		t.Fatalf("unexpected latest version: %+v", result)
	}
}

func TestReleaseManifestAssetForPlatform(t *testing.T) {
	t.Parallel()

	manifest := releaseManifest{
		Platforms: map[string]releaseAsset{
			"darwin/arm64": {
				URL:    "https://example.com/shellin-darwin-arm64.tar.gz",
				SHA256: "darwin-sha",
			},
			"linux/amd64": {
				URL:    "https://example.com/shellin-linux-amd64.tar.gz",
				SHA256: "linux-sha",
			},
		},
	}

	asset, err := manifest.assetForPlatform("linux", "amd64")
	if err != nil {
		t.Fatalf("assetForPlatform failed: %v", err)
	}
	if asset.URL != "https://example.com/shellin-linux-amd64.tar.gz" || asset.SHA256 != "linux-sha" {
		t.Fatalf("unexpected asset: %+v", asset)
	}
}

func TestPerformSelfUpdateInstallsLatestRelease(t *testing.T) {
	t.Parallel()

	releaseBinary := []byte("new-binary")
	archiveData := makeReleaseArchive(t, releaseBinary)
	archiveChecksum := sha256.Sum256(archiveData)
	server, publicKey := newReleaseServer(t, "1.1.0", archiveData, hex.EncodeToString(archiveChecksum[:]))

	executablePath := filepath.Join(t.TempDir(), "shellin")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write executable failed: %v", err)
	}

	result, err := Perform(Options{
		AllowUnknownCurrent: false,
		BaseURL:             server.URL,
		CommandName:         testCommandName,
		CurrentVersion:      "1.0.0",
		ExecutablePath:      executablePath,
		ManifestPublicKey:   publicKey,
		Reexec:              false,
	})
	if err != nil {
		t.Fatalf("performSelfUpdate failed: %v", err)
	}
	if !result.Checked || !result.UpdateAvailable || !result.Updated {
		t.Fatalf("unexpected update result: %+v", result)
	}
	if result.LatestVersion != "1.1.0" {
		t.Fatalf("unexpected latest version: %+v", result)
	}

	updatedBinary, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatalf("read updated executable failed: %v", err)
	}
	if string(updatedBinary) != string(releaseBinary) {
		t.Fatalf("unexpected updated binary: %q", string(updatedBinary))
	}
}

func TestPerformSelfUpdateAllowsUnknownCurrentVersion(t *testing.T) {
	t.Parallel()

	releaseBinary := []byte("new-binary")
	archiveData := makeReleaseArchive(t, releaseBinary)
	archiveChecksum := sha256.Sum256(archiveData)
	server, publicKey := newReleaseServer(t, "1.1.0", archiveData, hex.EncodeToString(archiveChecksum[:]))

	executablePath := filepath.Join(t.TempDir(), "shellin")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write executable failed: %v", err)
	}

	result, err := Perform(Options{
		AllowUnknownCurrent: true,
		BaseURL:             server.URL,
		CommandName:         testCommandName,
		CurrentVersion:      "dev",
		ExecutablePath:      executablePath,
		ManifestPublicKey:   publicKey,
		Reexec:              false,
	})
	if err != nil {
		t.Fatalf("performSelfUpdate failed: %v", err)
	}
	if !result.Updated {
		t.Fatalf("expected update to proceed for unknown current version, got %+v", result)
	}
}

func TestPerformSelfUpdateSkipsWhenAlreadyCurrent(t *testing.T) {
	t.Parallel()

	archiveData := makeReleaseArchive(t, []byte("new-binary"))
	archiveChecksum := sha256.Sum256(archiveData)
	server, publicKey := newReleaseServer(t, "1.1.0", archiveData, hex.EncodeToString(archiveChecksum[:]))

	executablePath := filepath.Join(t.TempDir(), "shellin")
	if err := os.WriteFile(executablePath, []byte("current-binary"), 0o755); err != nil {
		t.Fatalf("write executable failed: %v", err)
	}

	result, err := Perform(Options{
		AllowUnknownCurrent: false,
		BaseURL:             server.URL,
		CommandName:         testCommandName,
		CurrentVersion:      "1.1.0",
		ExecutablePath:      executablePath,
		ManifestPublicKey:   publicKey,
		Reexec:              false,
	})
	if err != nil {
		t.Fatalf("performSelfUpdate failed: %v", err)
	}
	if result.UpdateAvailable || result.Updated {
		t.Fatalf("expected no update, got %+v", result)
	}

	currentBinary, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatalf("read executable failed: %v", err)
	}
	if string(currentBinary) != "current-binary" {
		t.Fatalf("expected binary to stay unchanged, got %q", string(currentBinary))
	}
}

const testCommandName = "shellin"

func makeReleaseArchive(t *testing.T, binaryData []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name: testCommandName,
		Mode: 0o755,
		Size: int64(len(binaryData)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tarWriter.Write(binaryData); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tar close failed: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("gzip close failed: %v", err)
	}
	return buf.Bytes()
}

func TestFetchLatestReleaseManifestRejectsUnsignedManifest(t *testing.T) {
	t.Parallel()

	_, publicKey := newReleaseSigningKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(releaseManifest{
			Version: "1.1.0",
			Platforms: map[string]releaseAsset{
				runtime.GOOS + "/" + runtime.GOARCH: {
					URL:    serverURL(r) + "/downloads/shellin.tar.gz",
					SHA256: strings.Repeat("a", 64),
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	if _, err := fetchLatestReleaseManifest(server.URL, publicKey); err == nil {
		t.Fatal("expected unsigned manifest to be rejected")
	}
}

func TestFetchLatestReleaseManifestRejectsTamperedManifest(t *testing.T) {
	t.Parallel()

	privateKey, publicKey := newReleaseSigningKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := releaseManifest{
			Version:     "1.1.0",
			PublishedAt: "2026-04-30T00:00:00Z",
			Platforms: map[string]releaseAsset{
				runtime.GOOS + "/" + runtime.GOARCH: {
					URL:    serverURL(r) + "/downloads/shellin.tar.gz",
					SHA256: strings.Repeat("a", 64),
				},
			},
		}
		manifest.Signature = signReleaseManifestForTest(privateKey, manifest)
		manifest.Version = "9.9.9"
		_ = json.NewEncoder(w).Encode(manifest)
	}))
	t.Cleanup(server.Close)

	if _, err := fetchLatestReleaseManifest(server.URL, publicKey); err == nil {
		t.Fatal("expected tampered manifest to be rejected")
	}
}

func newReleaseServer(t *testing.T, version string, archiveData []byte, checksum string) (*httptest.Server, string) {
	t.Helper()

	privateKey, publicKey := newReleaseSigningKey(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/downloads/latest.json", func(w http.ResponseWriter, r *http.Request) {
		manifest := releaseManifest{
			Version:     version,
			PublishedAt: "2026-04-30T00:00:00Z",
			Platforms: map[string]releaseAsset{
				runtime.GOOS + "/" + runtime.GOARCH: {
					URL:    serverURL(r) + "/downloads/shellin.tar.gz",
					SHA256: checksum,
				},
			},
		}
		manifest.Signature = signReleaseManifestForTest(privateKey, manifest)
		_ = json.NewEncoder(w).Encode(manifest)
	})
	mux.HandleFunc("/downloads/shellin.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archiveData)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, publicKey
}

func newReleaseSigningKey(t *testing.T) (ed25519.PrivateKey, string) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate signing key failed: %v", err)
	}
	return privateKey, base64.StdEncoding.EncodeToString(publicKey)
}

func signReleaseManifestForTest(privateKey ed25519.PrivateKey, manifest releaseManifest) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, releaseManifestSigningPayload(manifest)))
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
