package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Version can be set at build time via ldflags:
// -ldflags "-X github.com/grzadr/gomper/cmd.Version=v1.0.0"
var Version string

type buildInfoReader func() (*debug.BuildInfo, bool)

var readBuildInfo buildInfoReader = debug.ReadBuildInfo

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

	// 2 & 3. ReadBuildInfo (module version or VCS metadata)
	if reader != nil {
		if info, ok := reader(); ok && info != nil {
			// 2. Module version from go install @tag / @latest
			if v := strings.TrimSpace(info.Main.Version); v != "" && v != "(devel)" {
				return v
			}

			// 3. VCS Metadata Fallback
			var revision string
			var modified bool
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					revision = strings.TrimSpace(setting.Value)
				case "vcs.modified":
					modified = (strings.TrimSpace(setting.Value) == "true")
				}
			}

			if revision != "" {
				if modified {
					return fmt.Sprintf("%s-dirty", revision)
				}
				return revision
			}
		}
	}

	// 4. Default Fallback
	return "dev"
}
