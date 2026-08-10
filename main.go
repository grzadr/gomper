package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/grzadr/gomper/cmd"
	"github.com/grzadr/gomper/internal/setup"
)

func main() {
	ctx, stop := setup.NewContext()
	defer stop()

	app := setup.NewApp(slog.LevelInfo)

	rootCmd := cmd.NewRootCommand(app)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		app.Logger().Error("application execution failed", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
