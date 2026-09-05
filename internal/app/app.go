package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/grzadr/gomper/internal/dumper"
	"github.com/grzadr/gomper/internal/filetx"
	"github.com/grzadr/gomper/internal/scanner"
	"github.com/grzadr/gomper/internal/setup"
)

// ListOptions specifies formatting and filtering controls for the list command.
type ListOptions struct {
	LongFormat     bool
	Formatter      ListFormatter
	IgnorePatterns []string
	IgnoreDirs     []string
	IgnoreDotfiles bool
	NameFilters    []string
	Profiles       []string
}

// DumpOptions specifies export settings for the dump command.
type DumpOptions struct {
	Format         OutputFormat
	OutputPath     string
	IgnorePatterns []string
	IgnoreDirs     []string
	IgnoreDotfiles bool
	NameFilters    []string
	Profiles       []string
	Instructions   string
}

// Dumper defines the contract for directory processing operations.
type Dumper interface {
	List(ctx context.Context, out io.Writer, paths []string, opts ListOptions) error
	Dump(ctx context.Context, out io.Writer, paths []string, opts DumpOptions) error
}

// Service implements the Dumper interface cleanly decoupled from transport logic.
type Service struct {
	app *setup.App
}

func NewService(app *setup.App) *Service {
	return new(Service{app: app})
}

func buildCombinedPatterns(profiles []string, userPatterns []string) ([]string, error) {
	var combined []string

	for _, prof := range profiles {
		patterns, err := scanner.LoadProfilePatterns(prof)
		if err != nil {
			return nil, err
		}
		combined = append(combined, patterns...)
	}

	combined = append(combined, userPatterns...)
	return combined, nil
}

// Hook for scanning paths, allowing test overrides for edge cases.
var walkPathsFunc = scanner.WalkPaths

// List iterates over files in the specified paths using scanner.WalkPaths and formats output.
func (s *Service) List(ctx context.Context, out io.Writer, paths []string, opts ListOptions) error {
	logger := s.logger()
	logger.DebugContext(ctx, "starting list operation",
		slog.Any("paths", paths),
		slog.Any("profiles", opts.Profiles),
	)

	patterns, err := buildCombinedPatterns(opts.Profiles, opts.IgnorePatterns)
	if err != nil {
		return err
	}

	filter, err := scanner.NewFilter(scanner.FilterOptions{
		IgnorePatterns: patterns,
		IgnoreDotfiles: opts.IgnoreDotfiles,
		NamePatterns:   opts.NameFilters,
		IgnoreDirs:     opts.IgnoreDirs,
	})
	if err != nil {
		return err
	}

	formatter := opts.Formatter
	if formatter == nil {
		formatter = NewStandardFormatter(opts.LongFormat)
	}

	var scanOpts []scanner.ScanOption
	if formatter.RequiresMetrics() {
		scanOpts = append(scanOpts, scanner.WithComputeMetrics(true))
	}

	if err := formatter.WriteHeader(out); err != nil {
		return err
	}

	for entry, err := range walkPathsFunc(ctx, paths, filter, scanOpts...) {
		if err != nil {
			logger.ErrorContext(ctx, "scan error encountered", slog.String("path", entry.Path), slog.Any("error", err))
			_, _ = fmt.Fprintf(out, "[ERROR] %s: %v\n", entry.Path, err)
			continue
		}

		if entry.Content != nil {
			_ = entry.Content.Close()
		}

		if entry.IsDir {
			continue
		}

		if err := formatter.FormatEntry(out, entry); err != nil {
			return err
		}
	}

	return formatter.Flush(out)
}

// Dump scans paths using scanner.WalkPaths and prepares structure output in the specified format.
func (s *Service) Dump(ctx context.Context, out io.Writer, paths []string, opts DumpOptions) error {
	logger := s.logger()
	logger.DebugContext(ctx, "starting dump operation",
		slog.Any("paths", paths),
		slog.String("format", opts.Format.String()),
		slog.String("output", opts.OutputPath),
	)

	if !opts.Format.IsValid() {
		return fmt.Errorf("unsupported output format: %s", opts.Format)
	}

	patterns, err := buildCombinedPatterns(opts.Profiles, opts.IgnorePatterns)
	if err != nil {
		return err
	}

	filter, err := scanner.NewFilter(scanner.FilterOptions{
		IgnorePatterns: patterns,
		IgnoreDotfiles: opts.IgnoreDotfiles,
		NamePatterns:   opts.NameFilters,
		IgnoreDirs:     opts.IgnoreDirs,
	})

	if err != nil {
		return err
	}

	writeFunc := func(ctx context.Context, w io.Writer) error {
		d := dumper.NewDumper(logger)
		seq := walkPathsFunc(ctx, paths, filter)
		switch opts.Format {
		case FormatXML:
			return d.GenerateXML(ctx, seq, opts.Instructions, w)
		default:
			return d.GenerateMarkdown(ctx, seq, opts.Instructions, w)
		}
	}

	if opts.OutputPath != "" && opts.OutputPath != "stdout" {
		_, _ = fmt.Fprintf(out, "Dumping %d root path(s) in %s format to %s...\n", len(paths), opts.Format, opts.OutputPath)
		return filetx.WriteAtomically(ctx, opts.OutputPath, writeFunc)
	}

	return writeFunc(ctx, out)
}

func (s *Service) logger() *slog.Logger {
	if s.app != nil && s.app.Logger() != nil {
		return s.app.Logger()
	}
	return slog.Default()
}
