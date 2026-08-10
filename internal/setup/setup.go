package setup

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// App encapsulates structured logger and dynamic log level management.
type App struct {
	logLevel *slog.LevelVar
	logger   *slog.Logger
}

// NewApp initializes an App instance with a slog.Logger using a dynamic LevelVar handler.
func NewApp(level slog.Level) *App {
	lvl := new(slog.LevelVar)
	lvl.Set(level)

	return &App{
		logLevel: lvl,
		logger: slog.New(
			slog.NewTextHandler(
				os.Stderr,
				&slog.HandlerOptions{Level: lvl},
			),
		),
	}
}

// ParseLogLevel converts a string ("debug", "info", "warn", "error") into a slog.Level.
func ParseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger returns the underlying *slog.Logger.
func (a *App) Logger() *slog.Logger {
	return a.logger
}

// SetLogLevel dynamically updates the application logging level.
func (a *App) SetLogLevel(level slog.Level) {
	a.logLevel.Set(level)
}

// NewContext returns a signal-aware context for SIGINT and SIGTERM.
func NewContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
