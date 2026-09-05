package cmd

import (
	"runtime/debug"
	"strings"
)

// Version can be set at build time via ldflags or module versioning.
var Version string

type buildInfoReader func() (*debug.BuildInfo, bool)

var readBuildInfo buildInfoReader = debug.ReadBuildInfo

func getBuildInfo(reader buildInfoReader) (version, commit, date string) {
	if reader != nil {
		if info, ok := reader(); ok && info != nil {
			if v := strings.TrimSpace(info.Main.Version); v != "" && v != "(devel)" {
				version = v
			}
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					commit = setting.Value
				case "vcs.time":
					date = setting.Value
				case "vcs.modified":
					if setting.Value == "true" {
						commit += "-dirty"
					}
				}
			}
		}
	}
	return version, commit, date
}

// getVersion returns the resolved version according to the following precedence:
// 1. Explicit LDFlags (CI/CD / Release Artifacts)
// 2. runtime/debug.ReadBuildInfo() (go install @tag / @latest)
// 3. VCS Metadata Fallback (Local git builds via vcs.revision & vcs.modified)
// 4. Default Fallback ("dev")
func getVersion() string {
	return resolveVersion(Version, readBuildInfo)
}

func resolveVersion(ldflagVersion string, reader buildInfoReader) string {
	// 1. Explicit LDFlags
	if v := strings.TrimSpace(ldflagVersion); v != "" {
		return v
	}

	version, commit, _ := getBuildInfo(reader)
	if version != "" {
		return version
	}
	if commit != "" && commit != "-dirty" {
		return commit
	}

	// 4. Default Fallback
	return "dev"
}
