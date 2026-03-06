package output

import (
	"strings"
	"testing"

	"webalyze/internal/runner"
)

func TestRenderPlain_WithFiltersAndVerboseError(t *testing.T) {
	t.Parallel()

	results := []runner.Result{
		{
			Target:       "a.com",
			Technologies: []string{"Cloudflare", "React"},
			Categories:   []string{"CDN", "JavaScript Frameworks"},
			TechToCats: map[string][]string{
				"Cloudflare": []string{"CDN"},
				"React":      []string{"JavaScript Frameworks"},
			},
		},
		{
			Target:       "b.com",
			Technologies: []string{"WordPress"},
			Categories:   []string{"CMS"},
			TechToCats: map[string][]string{
				"WordPress": []string{"CMS"},
			},
		},
		{Target: "c.com", Err: assertErr("dial timeout")},
	}

	out, failed, err := Render(results, Options{
		MatchTech:      []string{"react"},
		FilterTech:     []string{"wordpress"},
		MatchCategory:  []string{"javascript frameworks"},
		FilterCategory: []string{"cms"},
		Verbose:        true,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if failed != 1 {
		t.Fatalf("failed mismatch got=%d want=1", failed)
	}
	if !strings.Contains(out, "a.com [Cloudflare] [React]") {
		t.Fatalf("expected a.com line, got=%q", out)
	}
	if strings.Contains(out, "b.com") {
		t.Fatalf("b.com should be filtered out: %q", out)
	}
	if strings.Contains(out, "c.com error:") {
		t.Fatalf("c.com should not pass match filter: %q", out)
	}
}

func TestRenderPlain_ColorizedByCategory(t *testing.T) {
	t.Parallel()

	results := []runner.Result{
		{
			Target:       "a.com",
			Technologies: []string{"Cloudflare", "React"},
			Categories:   []string{"CDN", "JavaScript Frameworks"},
			TechToCats: map[string][]string{
				"Cloudflare": []string{"CDN"},
				"React":      []string{"JavaScript Frameworks"},
			},
		},
	}

	out, _, err := Render(results, Options{Color: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "\x1b[36m[Cloudflare]\x1b[0m") {
		t.Fatalf("expected CDN colored Cloudflare, got=%q", out)
	}
	if !strings.Contains(out, "\x1b[33m[React]\x1b[0m") {
		t.Fatalf("expected JS Framework colored React, got=%q", out)
	}
}

func TestRenderJSONL_Silent(t *testing.T) {
	t.Parallel()

	results := []runner.Result{
		{
			Target:       "ok.com",
			URL:          "https://ok.com",
			Technologies: []string{"nginx"},
			Categories:   []string{"Web servers"},
			TechToCats: map[string][]string{
				"nginx": []string{"Web servers"},
			},
		},
		{Target: "empty.com", URL: "https://empty.com"},
	}

	out, failed, err := Render(results, Options{JSONL: true, Silent: true})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if failed != 0 {
		t.Fatalf("failed mismatch got=%d want=0", failed)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl line count mismatch got=%d want=2", len(lines))
	}
	if !strings.Contains(lines[0], "\"categories\":{\"Web servers\":[\"nginx\"]}") {
		t.Fatalf("expected category to technologies mapping in jsonl output, got=%q", lines[0])
	}
	if strings.Contains(lines[0], "\"technologies\":") {
		t.Fatalf("did not expect technologies map in jsonl output, got=%q", lines[0])
	}
}

func TestSummaryLine_FormatAndOrdering(t *testing.T) {
	t.Parallel()

	results := []runner.Result{
		{
			Target:       "a.com",
			Technologies: []string{"Cloudflare"},
			Categories:   []string{"CDN", "Security"},
			TechToCats: map[string][]string{
				"Cloudflare": []string{"CDN", "Security"},
			},
		},
		{
			Target:       "b.com",
			Technologies: []string{"Fastly"},
			Categories:   []string{"CDN"},
			TechToCats: map[string][]string{
				"Fastly": []string{"CDN"},
			},
		},
		{
			Target:       "c.com",
			Technologies: []string{"WordPress"},
			Categories:   []string{"CMS"},
			TechToCats: map[string][]string{
				"WordPress": []string{"CMS"},
			},
		},
	}

	got := SummaryLine(results, Options{})
	want := "Found results 3 (CDN: 2, CMS: 1, Security: 1)"
	if got != want {
		t.Fatalf("summary mismatch got=%q want=%q", got, want)
	}
}

func TestSummaryLine_ColorizedCategoryNames(t *testing.T) {
	t.Parallel()

	results := []runner.Result{
		{
			Target:       "a.com",
			Technologies: []string{"Cloudflare"},
			Categories:   []string{"CDN"},
			TechToCats: map[string][]string{
				"Cloudflare": []string{"CDN"},
			},
		},
	}

	got := SummaryLine(results, Options{Color: true})
	if !strings.Contains(got, "\x1b[36mCDN\x1b[0m: 1") {
		t.Fatalf("expected colorized summary category name only, got=%q", got)
	}
	if strings.Contains(got, "\x1b[36mCDN: 1\x1b[0m") {
		t.Fatalf("count should not be colorized, got=%q", got)
	}
}

func TestFormatResult_WithVerboseError(t *testing.T) {
	t.Parallel()

	line, include, err := FormatResult(runner.Result{
		Target: "bad.com",
		Err:    assertErr("dial timeout"),
	}, Options{Verbose: true})
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if !include {
		t.Fatalf("expected line inclusion")
	}
	if !strings.Contains(line, "bad.com error:") {
		t.Fatalf("expected verbose error line, got=%q", line)
	}
}

func TestSummaryAggregator_IncrementalMatchesSummaryLine(t *testing.T) {
	t.Parallel()

	results := []runner.Result{
		{
			Target:       "a.com",
			Technologies: []string{"Cloudflare"},
			Categories:   []string{"CDN", "Security"},
			TechToCats: map[string][]string{
				"Cloudflare": []string{"CDN", "Security"},
			},
		},
		{
			Target:       "b.com",
			Technologies: []string{"Fastly"},
			Categories:   []string{"CDN"},
			TechToCats: map[string][]string{
				"Fastly": []string{"CDN"},
			},
		},
	}

	agg := NewSummaryAggregator(Options{})
	for _, r := range results {
		agg.Add(r)
	}
	if got, want := agg.Line(), SummaryLine(results, Options{}); got != want {
		t.Fatalf("line mismatch got=%q want=%q", got, want)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
