package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseConfig_Basic(t *testing.T) {
	t.Parallel()

	cfg, err := parseConfig([]string{
		"-i", "example.com",
		"-input", "https://go.dev",
		"-match-tech", "react,nginx",
		"-filter-tech", "wordpress",
		"-mcat", "Web servers",
		"-match-category", "CDN",
		"-fcat", "Blogs",
		"-filter-category", "CMS",
		"-retry", "3",
		"-timeout", "5s",
		"-j",
		"-nc",
		"-silent",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if got, want := len(cfg.Inputs), 2; got != want {
		t.Fatalf("inputs len mismatch got=%d want=%d", got, want)
	}
	if got, want := cfg.Retry, 3; got != want {
		t.Fatalf("retry mismatch got=%d want=%d", got, want)
	}
	if got, want := cfg.Timeout, 5*time.Second; got != want {
		t.Fatalf("timeout mismatch got=%s want=%s", got, want)
	}
	if !cfg.JSONL || !cfg.Silent {
		t.Fatalf("expected jsonl and silent to be true")
	}
	if !cfg.NoColor {
		t.Fatalf("expected no-color to be true")
	}
	if got, want := len(cfg.MatchCategory), 2; got != want {
		t.Fatalf("match category len mismatch got=%d want=%d", got, want)
	}
	if got, want := len(cfg.FilterCategory), 2; got != want {
		t.Fatalf("filter category len mismatch got=%d want=%d", got, want)
	}
}

func TestParseConfig_RetryValidation(t *testing.T) {
	t.Parallel()

	_, err := parseConfig([]string{"-retry", "0"}, &bytes.Buffer{})
	if err == nil {
		t.Fatalf("expected retry validation error")
	}
}

func TestRun_Version(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-version"}, &bytes.Buffer{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 got=%d", code)
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected version output")
	}
}

func TestRun_NoArgsPrintsHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, &bytes.Buffer{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 got=%d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output")
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected help output on stderr")
	}
	if !strings.Contains(stderr.String(), "version:") {
		t.Fatalf("expected banner output on stderr")
	}
}

func TestRun_HelpPrintsBannerAndHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-h"}, &bytes.Buffer{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 got=%d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output")
	}
	if !strings.Contains(stderr.String(), "version:") {
		t.Fatalf("expected banner output on stderr")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected usage output on stderr")
	}
}

func TestRun_JSONLBannerAndSummaryOnStderr(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(
		[]string{"-j", "-i", "://bad-target", "-v"},
		&bytes.Buffer{},
		&stdout,
		&stderr,
	)
	if code != 1 {
		t.Fatalf("expected exit code 1 got=%d", code)
	}
	if !strings.Contains(stdout.String(), "\"target\":\"://bad-target\"") {
		t.Fatalf("expected jsonl result on stdout, got=%q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "version:") {
		t.Fatalf("expected banner on stderr, got=%q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "(") {
		t.Fatalf("expected summary line on stderr, got=%q", stderr.String())
	}
}

func TestRun_SilentSuppressesBannerAndSummary(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(
		[]string{"-j", "-silent", "-i", "://bad-target"},
		&bytes.Buffer{},
		&stdout,
		&stderr,
	)
	if code != 1 {
		t.Fatalf("expected exit code 1 got=%d", code)
	}
	if strings.Contains(stderr.String(), "version:") {
		t.Fatalf("did not expect banner in silent mode, got=%q", stderr.String())
	}
	if strings.Contains(stderr.String(), "(") {
		t.Fatalf("did not expect summary in silent mode, got=%q", stderr.String())
	}
}
