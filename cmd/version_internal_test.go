package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name          string
		ldflagVersion string
		reader        buildInfoReader
		expected      string
	}{
		{
			name:          "LDFlags set explicitly",
			ldflagVersion: "v1.2.3",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "v9.9.9"},
				}, true
			},
			expected: "v1.2.3",
		},
		{
			name:          "LDFlags set with surrounding whitespace",
			ldflagVersion: "  v2.0.0  ",
			reader:        nil,
			expected:      "v2.0.0",
		},
		{
			name:          "Module version from ReadBuildInfo when LDFlags empty",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "v1.5.0"},
				}, true
			},
			expected: "v1.5.0",
		},
		{
			name:          "Module version with whitespace",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "  v1.5.0-rc1  "},
				}, true
			},
			expected: "v1.5.0-rc1",
		},
		{
			name:          "VCS clean build when Module version is (devel)",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "(devel)"},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: "abcdef123456"},
						{Key: "vcs.modified", Value: "false"},
					},
				}, true
			},
			expected: "abcdef123456",
		},
		{
			name:          "VCS dirty build when Module version is (devel)",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "(devel)"},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: "abcdef123456"},
						{Key: "vcs.modified", Value: "true"},
					},
				}, true
			},
			expected: "abcdef123456-dirty",
		},
		{
			name:          "VCS clean build when Module version is empty",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: ""},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: "abcdef123456"},
					},
				}, true
			},
			expected: "abcdef123456",
		},
		{
			name:          "VCS modified is true but no revision falls back to dev",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "(devel)"},
					Settings: []debug.BuildSetting{
						{Key: "vcs.modified", Value: "true"},
					},
				}, true
			},
			expected: "dev",
		},
		{
			name:          "ReadBuildInfo returns ok=false falls back to dev",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return nil, false
			},
			expected: "dev",
		},
		{
			name:          "ReadBuildInfo returns ok=true but info=nil falls back to dev",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return nil, true
			},
			expected: "dev",
		},
		{
			name:          "Nil reader falls back to dev",
			ldflagVersion: "",
			reader:        nil,
			expected:      "dev",
		},
		{
			name:          "Empty BuildInfo settings and devel version falls back to dev",
			ldflagVersion: "",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main:     debug.Module{Version: "(devel)"},
					Settings: []debug.BuildSetting{},
				}, true
			},
			expected: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveVersion(tt.ldflagVersion, tt.reader)
			if result != tt.expected {
				t.Errorf("resolveVersion() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	origVersion := Version
	origReader := readBuildInfo
	defer func() {
		Version = origVersion
		readBuildInfo = origReader
	}()

	t.Run("getVersion uses LDFlags when set", func(t *testing.T) {
		Version = "v3.0.0"
		if got := getVersion(); got != "v3.0.0" {
			t.Errorf("getVersion() = %q, expected %q", got, "v3.0.0")
		}
	})

	t.Run("getVersion uses custom reader when Version is empty", func(t *testing.T) {
		Version = ""
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{Version: "v2.1.0"},
			}, true
		}
		if got := getVersion(); got != "v2.1.0" {
			t.Errorf("getVersion() = %q, expected %q", got, "v2.1.0")
		}
	})
}

func TestGetBuildInfo(t *testing.T) {
	origReader := readBuildInfo
	defer func() {
		readBuildInfo = origReader
	}()

	tests := []struct {
		name        string
		reader      buildInfoReader
		wantVersion string
		wantCommit  string
		wantDate    string
	}{
		{
			name: "Populated VCS settings and module version",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "v1.0.0"},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: "1234567890ab"},
						{Key: "vcs.time", Value: "2026-08-23T20:00:00Z"},
						{Key: "vcs.modified", Value: "true"},
					},
				}, true
			},
			wantVersion: "v1.0.0",
			wantCommit:  "1234567890ab-dirty",
			wantDate:    "2026-08-23T20:00:00Z",
		},
		{
			name: "Clean VCS build with (devel) module version",
			reader: func() (*debug.BuildInfo, bool) {
				return &debug.BuildInfo{
					Main: debug.Module{Version: "(devel)"},
					Settings: []debug.BuildSetting{
						{Key: "vcs.revision", Value: "abcdef123456"},
						{Key: "vcs.time", Value: "2026-08-23T19:00:00Z"},
						{Key: "vcs.modified", Value: "false"},
					},
				}, true
			},
			wantVersion: "",
			wantCommit:  "abcdef123456",
			wantDate:    "2026-08-23T19:00:00Z",
		},
		{
			name: "Reader returns ok=false",
			reader: func() (*debug.BuildInfo, bool) {
				return nil, false
			},
			wantVersion: "",
			wantCommit:  "",
			wantDate:    "",
		},
		{
			name:        "Nil reader",
			reader:      nil,
			wantVersion: "",
			wantCommit:  "",
			wantDate:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readBuildInfo = tt.reader
			v, c, d := getBuildInfo()
			if v != tt.wantVersion {
				t.Errorf("getBuildInfo() version = %q, want %q", v, tt.wantVersion)
			}
			if c != tt.wantCommit {
				t.Errorf("getBuildInfo() commit = %q, want %q", c, tt.wantCommit)
			}
			if d != tt.wantDate {
				t.Errorf("getBuildInfo() date = %q, want %q", d, tt.wantDate)
			}
		})
	}
}

