// SPDX-License-Identifier: AGPL-3.0-or-later

package buildinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

const CoreModulePath = "github.com/jaycho46/shellin-core"

var (
	ServiceReleaseID = "dev"
	Version          = "dev"
	Commit           = ""
	BuildTime        = ""
	SourceRepo       = "https://github.com/jaycho46/shellin-core"
)

type VersionResponse struct {
	Service          string `json:"service"`
	ServiceReleaseID string `json:"service_release_id,omitempty"`
	CoreModule       string `json:"core_module"`
	CoreVersion      string `json:"core_version"`
	CoreSum          string `json:"core_sum,omitempty"`
	BuildTime        string `json:"build_time,omitempty"`
	GoVersion        string `json:"go_version"`
	VCSRevision      string `json:"vcs_revision,omitempty"`
	VCSModified      string `json:"vcs_modified,omitempty"`
	BinarySHA256     string `json:"binary_sha256,omitempty"`
	SourceRepo       string `json:"source_repo,omitempty"`
}

func Snapshot(service string) VersionResponse {
	coreVersion, coreSum := coreModuleVersion()
	vcsRevision, vcsModified := vcsInfo()
	if strings.TrimSpace(Commit) != "" {
		vcsRevision = strings.TrimSpace(Commit)
	}
	binarySHA256, _ := executableSHA256()

	releaseID := strings.TrimSpace(ServiceReleaseID)
	if releaseID == "" {
		releaseID = strings.TrimSpace(Version)
	}

	return VersionResponse{
		Service:          strings.TrimSpace(service),
		ServiceReleaseID: releaseID,
		CoreModule:       CoreModulePath,
		CoreVersion:      coreVersion,
		CoreSum:          coreSum,
		BuildTime:        strings.TrimSpace(BuildTime),
		GoVersion:        runtime.Version(),
		VCSRevision:      vcsRevision,
		VCSModified:      vcsModified,
		BinarySHA256:     binarySHA256,
		SourceRepo:       strings.TrimSpace(SourceRepo),
	}
}

func Handler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(Snapshot(service))
	}
}

func coreModuleVersion() (string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return strings.TrimSpace(Version), ""
	}
	if info.Main.Path == CoreModulePath {
		return buildVersion(info.Main.Version), info.Main.Sum
	}
	for _, dep := range info.Deps {
		if dep.Path == CoreModulePath {
			if dep.Replace != nil {
				return buildVersion(dep.Replace.Version), dep.Replace.Sum
			}
			return buildVersion(dep.Version), dep.Sum
		}
	}
	return strings.TrimSpace(Version), ""
}

func vcsInfo() (string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	var revision string
	var modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	return revision, modified
}

func executableSHA256() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func buildVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		if strings.TrimSpace(Version) != "" {
			return strings.TrimSpace(Version)
		}
		return "dev"
	}
	return version
}
