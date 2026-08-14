package config_test

import (
	"testing"

	"github.com/grzadr/gomper/internal/config"
)

func TestConfig_GetEffectiveProfiles(t *testing.T) {
	cfg := config.Config{
		Profile:  []string{"generic"},
		Profiles: []string{"go", "node"},
	}

	effective := cfg.GetEffectiveProfiles()
	if len(effective) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(effective))
	}

	if effective[0] != "generic" || effective[1] != "go" || effective[2] != "node" {
		t.Errorf("unexpected profile slice: %v", effective)
	}
}

func TestConfig_GetEffectiveIgnoreDirs(t *testing.T) {
	cfg := config.Config{
		IgnoreDir:  []string{"bin"},
		IgnoreDirs: []string{"coverage", "build"},
	}

	effective := cfg.GetEffectiveIgnoreDirs()
	if len(effective) != 3 {
		t.Fatalf("expected 3 ignore dirs, got %d", len(effective))
	}

	if effective[0] != "bin" || effective[1] != "coverage" || effective[2] != "build" {
		t.Errorf("unexpected ignore dir slice: %v", effective)
	}
}

func TestConfig_GetEffectiveNameFilters(t *testing.T) {
	cfg := config.Config{
		Name:        []string{`.*\.go$`},
		NameFilter:  []string{`.*\.ts$`},
		NameFilters: []string{`.*\.py$`},
	}

	effective := cfg.GetEffectiveNameFilters()
	if len(effective) != 3 {
		t.Fatalf("expected 3 name filters, got %d", len(effective))
	}

	if effective[0] != `.*\.go$` || effective[1] != `.*\.ts$` || effective[2] != `.*\.py$` {
		t.Errorf("unexpected name filter slice: %v", effective)
	}
}


