// SPDX-License-Identifier: AGPL-3.0-or-later

package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func downloadAndExtractReleaseBinary(asset releaseAsset, commandName string) ([]byte, os.FileMode, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimSpace(asset.URL), nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, 0, fmt.Errorf("download release archive status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	archiveData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if err := verifyReleaseArchiveChecksum(archiveData, asset.SHA256); err != nil {
		return nil, 0, err
	}
	return extractReleaseBinary(archiveData, commandName)
}

func verifyReleaseArchiveChecksum(archiveData []byte, expectedChecksum string) error {
	expected := strings.ToLower(strings.TrimSpace(expectedChecksum))
	if expected == "" {
		return errors.New("expected archive checksum is required")
	}
	sum := sha256.Sum256(archiveData)
	if hex.EncodeToString(sum[:]) != expected {
		return errors.New("release archive checksum mismatch")
	}
	return nil
}

func extractReleaseBinary(archiveData []byte, commandName string) ([]byte, os.FileMode, error) {
	commandName = normalizedCommandName(commandName)
	gzReader, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return nil, 0, fmt.Errorf("open release archive failed: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read release archive failed: %w", err)
		}
		if header.FileInfo().Mode().IsRegular() && filepath.Base(header.Name) == commandName {
			binaryData, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, 0, fmt.Errorf("read release binary failed: %w", err)
			}
			mode := header.FileInfo().Mode().Perm()
			if mode&0o111 == 0 {
				mode |= 0o755
			}
			return binaryData, mode, nil
		}
	}
	return nil, 0, fmt.Errorf("release archive did not contain %s", commandName)
}

func installReplacementBinary(executablePath string, binaryData []byte, mode os.FileMode) error {
	executablePath = strings.TrimSpace(executablePath)
	if executablePath == "" {
		return errors.New("executable path is required")
	}
	if len(binaryData) == 0 {
		return errors.New("replacement binary is empty")
	}

	dir := filepath.Dir(executablePath)
	tmp, err := os.CreateTemp(dir, filepath.Base(executablePath)+".update-*")
	if err != nil {
		return fmt.Errorf("create temporary binary failed: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if mode == 0 {
		mode = 0o755
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary binary mode failed: %w", err)
	}
	if _, err := tmp.Write(binaryData); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary binary failed: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary binary failed: %w", err)
	}
	if err := os.Rename(tmpPath, executablePath); err != nil {
		return fmt.Errorf("replace %s failed: %w", executablePath, err)
	}
	return nil
}
