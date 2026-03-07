package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"webalyze/internal/input"
	"webalyze/internal/output"
	"webalyze/internal/runner"
)

var appVersion = "dev"

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	startedAt := time.Now()

	helpRequested := false
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			helpRequested = true
			break
		}
	}
	if helpRequested {
		printBanner(stderr, appVersion)
		fs, _, _, _, _, _, _ := newFlagSet(stderr)
		fs.Usage()
		return 0
	}

	if len(args) == 0 {
		printBanner(stderr, appVersion)
		fs, _, _, _, _, _, _ := newFlagSet(stderr)
		fs.Usage()
		return 0
	}

	cfg, err := parseConfig(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		logError(stderr, "flag parsing failed: %v", err)
		return 1
	}
	if cfg.Version {
		fmt.Fprintln(stdout, appVersion)
		return 0
	}

	stdinAvailable := false
	if stat, statErr := os.Stdin.Stat(); statErr == nil {
		stdinAvailable = stat.Mode()&os.ModeCharDevice == 0
	}

	targets, err := input.CollectTargets(cfg.Inputs, cfg.ListPath, stdin, stdinAvailable)
	if err != nil {
		logError(stderr, "input error: %v", err)
		return 1
	}
	if len(targets) == 0 {
		logError(stderr, "no input provided (use -i, -list, or stdin)")
		return 1
	}
	logInfo(stderr, cfg.Verbose && !cfg.Silent, "loaded %d targets", len(targets))

	metaWriter := stdout
	if cfg.JSONL {
		metaWriter = stderr
	}

	outputColorEnabled := false
	if !cfg.JSONL {
		outputColorEnabled = isTerminalWriter(stdout) && !cfg.NoColor
	}
	metaColorEnabled := isTerminalWriter(metaWriter) && !cfg.NoColor

	resultWriters := []io.Writer{stdout}
	var outputFile *os.File
	if cfg.OutputPath != "" {
		outputFile, err = os.Create(cfg.OutputPath)
		if err != nil {
			logError(stderr, "write error: %v", err)
			return 1
		}
		defer outputFile.Close()
		resultWriters = append(resultWriters, outputFile)
	}
	streamWriter := output.NewSyncWriter(resultWriters...)

	if !cfg.Silent {
		printBanner(metaWriter, appVersion)
	}

	renderOpts := output.Options{
		JSONL:          cfg.JSONL,
		Color:          outputColorEnabled,
		Silent:         cfg.Silent,
		Verbose:        cfg.Verbose,
		MatchTech:      cfg.MatchTech,
		FilterTech:     cfg.FilterTech,
		MatchCategory:  cfg.MatchCategory,
		FilterCategory: cfg.FilterCategory,
	}
	summary := output.NewSummaryAggregator(output.Options{
		Color:          metaColorEnabled,
		Silent:         cfg.Silent,
		MatchTech:      cfg.MatchTech,
		FilterTech:     cfg.FilterTech,
		MatchCategory:  cfg.MatchCategory,
		FilterCategory: cfg.FilterCategory,
	})

	var failed atomic.Int64
	var outputErr atomic.Pointer[error]
	runner.RunStream(context.Background(), targets, runner.Config{
		Retry:          cfg.Retry,
		Timeout:        cfg.Timeout,
		Workers:        cfg.Workers,
		FollowRedirect: cfg.FollowRedirect,
	}, func(result runner.Result) {
		if result.Err != nil {
			failed.Add(1)
		}
		summary.Add(result)
		line, include, formatErr := output.FormatResult(result, renderOpts)
		if formatErr != nil {
			errCopy := formatErr
			outputErr.CompareAndSwap(nil, &errCopy)
			return
		}
		if !include {
			return
		}
		if writeErr := streamWriter.WriteLine(line); writeErr != nil {
			errCopy := writeErr
			outputErr.CompareAndSwap(nil, &errCopy)
		}
	})
	if ptr := outputErr.Load(); ptr != nil {
		logError(stderr, "output error: %v", *ptr)
		return 1
	}

	if !cfg.Silent {
		fmt.Fprintln(metaWriter)
		logInfo(metaWriter, true, "%s", summary.Line())
	}
	if cfg.OutputPath != "" {
		logInfo(stderr, cfg.Verbose && !cfg.Silent, "wrote results to %s", cfg.OutputPath)
	}
	logInfo(
		stderr,
		cfg.Verbose && !cfg.Silent,
		"completed in %s (failed: %d)",
		time.Since(startedAt).Round(time.Millisecond),
		failed.Load(),
	)

	if failed.Load() > 0 {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func isTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func logInfo(w io.Writer, enabled bool, format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Fprintf(w, "[INF] "+format+"\n", args...)
}

func logError(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "[ERR] "+format+"\n", args...)
}
