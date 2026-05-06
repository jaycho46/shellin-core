// SPDX-License-Identifier: AGPL-3.0-or-later

package buildinfo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSnapshotUsesConfiguredReleaseFields(t *testing.T) {
	originalReleaseID := ServiceReleaseID
	originalVersion := Version
	originalCommit := Commit
	originalBuildTime := BuildTime
	originalSourceRepo := SourceRepo
	t.Cleanup(func() {
		ServiceReleaseID = originalReleaseID
		Version = originalVersion
		Commit = originalCommit
		BuildTime = originalBuildTime
		SourceRepo = originalSourceRepo
	})

	ServiceReleaseID = " release-1 "
	Version = " v1.2.3 "
	Commit = " abc123 "
	BuildTime = " 2026-05-06T00:00:00Z "
	SourceRepo = " https://example.com/shellin-core "

	got := Snapshot(" shellin ")
	if got.Service != "shellin" {
		t.Fatalf("unexpected service: %q", got.Service)
	}
	if got.ServiceReleaseID != "release-1" {
		t.Fatalf("unexpected release id: %q", got.ServiceReleaseID)
	}
	if got.VCSRevision != "abc123" {
		t.Fatalf("unexpected vcs revision: %q", got.VCSRevision)
	}
	if got.BuildTime != "2026-05-06T00:00:00Z" {
		t.Fatalf("unexpected build time: %q", got.BuildTime)
	}
	if got.SourceRepo != "https://example.com/shellin-core" {
		t.Fatalf("unexpected source repo: %q", got.SourceRepo)
	}
	if got.CoreModule != CoreModulePath {
		t.Fatalf("unexpected core module: %q", got.CoreModule)
	}
	if len(got.BinarySHA256) != 64 {
		t.Fatalf("unexpected binary sha256: %q", got.BinarySHA256)
	}
}

func TestHandlerServesVersionJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	Handler("control-plane")(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("unexpected cache control: %q", got)
	}
	var out VersionResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Service != "control-plane" || out.CoreModule != CoreModulePath {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestHandlerRejectsNonGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/version", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	Handler("control-plane")(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
