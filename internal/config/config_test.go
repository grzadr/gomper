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
