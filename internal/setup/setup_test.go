package setup_test

import (
	"log/slog"
	"testing"

	"github.com/grzadr/gomper/internal/setup"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := setup.ParseLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestApp_LoggerAndSetLogLevel(t *testing.T) {
	app := setup.NewApp(slog.LevelInfo)
	if app.Logger() == nil {
		t.Fatalf("expected non-nil logger")
	}

	app.SetLogLevel(slog.LevelDebug)
}

func TestNewContext(t *testing.T) {
	ctx, cancel := setup.NewContext()
	defer cancel()

	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
}
