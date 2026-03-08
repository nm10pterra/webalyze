package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		*s = append(*s, part)
	}
	return nil
}

type config struct {
	Inputs         []string
	ListPath       string
	MatchTech      []string
	FilterTech     []string
	MatchCategory  []string
	FilterCategory []string
	JSONL          bool
	OutputPath     string
	Retry          int
	Timeout        time.Duration
	Workers        int
	FollowRedirect bool
	Silent         bool
	NoColor        bool
	Verbose        bool
	Version        bool
}

func newFlagSet(stderr io.Writer) (*flag.FlagSet, *config, *stringSliceFlag, *stringSliceFlag, *stringSliceFlag, *stringSliceFlag, *stringSliceFlag) {
	cfg := &config{}
	fs := flag.NewFlagSet("webalyze", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var inputs stringSliceFlag
	var matchTech stringSliceFlag
	var filterTech stringSliceFlag
	var matchCategory stringSliceFlag
	var filterCategory stringSliceFlag

	fs.Var(&inputs, "i", "list of targets to process")
	fs.Var(&inputs, "input", "list of targets to process")
	fs.StringVar(&cfg.ListPath, "l", "", "file with targets (one per line)")
	fs.StringVar(&cfg.ListPath, "list", "", "file with targets (one per line)")
	fs.Var(&matchTech, "match-tech", "match targets with specified technologies")
	fs.Var(&filterTech, "filter-tech", "filter targets with specified technologies")
	fs.Var(&matchCategory, "mcat", "match targets with specified categories")
	fs.Var(&matchCategory, "match-category", "match targets with specified categories")
	fs.Var(&filterCategory, "fcat", "filter targets with specified categories")
	fs.Var(&filterCategory, "filter-category", "filter targets with specified categories")
	fs.BoolVar(&cfg.JSONL, "j", false, "write output in jsonl format")
	fs.BoolVar(&cfg.JSONL, "jsonl", false, "write output in jsonl format")
	fs.StringVar(&cfg.OutputPath, "o", "", "write output to file")
	fs.StringVar(&cfg.OutputPath, "output", "", "write output to file")
	fs.IntVar(&cfg.Retry, "retry", 2, "maximum number of retries (must be at least 1)")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "request timeout")
	fs.IntVar(&cfg.Workers, "c", 10, "number of concurrent workers")
	fs.IntVar(&cfg.Workers, "concurrency", 10, "number of concurrent workers")
	fs.BoolVar(&cfg.FollowRedirect, "fr", false, "follow http redirects")
	fs.BoolVar(&cfg.FollowRedirect, "follow-redirects", false, "follow http redirects")
	fs.BoolVar(&cfg.Silent, "silent", false, "only display results")
	fs.BoolVar(&cfg.NoColor, "nc", false, "disable colors in cli output")
	fs.BoolVar(&cfg.NoColor, "no-color", false, "disable colors in cli output")
	fs.BoolVar(&cfg.Verbose, "v", false, "display verbose output")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "display verbose output")
	fs.BoolVar(&cfg.Version, "version", false, "display version")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage:\n  webalyze [flags]\n\n")
		fmt.Fprintln(fs.Output(), "INPUT:")
		fmt.Fprintln(fs.Output(), "  -i, -input string[]       list of targets to process")
		fmt.Fprintln(fs.Output(), "  -l, -list string          file with targets (one per line)")
		fmt.Fprintln(fs.Output(), "")
		fmt.Fprintln(fs.Output(), "MATCHER:")
		fmt.Fprintln(fs.Output(), "  -match-tech string[]      include targets with matching technologies")
		fmt.Fprintln(fs.Output(), "  -mcat, -match-category string[]")
		fmt.Fprintln(fs.Output(), "                           include targets with matching categories")
		fmt.Fprintln(fs.Output(), "")
		fmt.Fprintln(fs.Output(), "FILTER:")
		fmt.Fprintln(fs.Output(), "  -filter-tech string[]     exclude targets with matching technologies")
		fmt.Fprintln(fs.Output(), "  -fcat, -filter-category string[]")
		fmt.Fprintln(fs.Output(), "                           exclude targets with matching categories")
		fmt.Fprintln(fs.Output(), "")
		fmt.Fprintln(fs.Output(), "OUTPUT:")
		fmt.Fprintln(fs.Output(), "  -j, -jsonl                write output in jsonl format")
		fmt.Fprintln(fs.Output(), "  -o, -output string        write output to file")
		fmt.Fprintln(fs.Output(), "  -silent                   only display results in output")
		fmt.Fprintln(fs.Output(), "  -nc, -no-color            disable colors in cli output")
		fmt.Fprintln(fs.Output(), "  -v, -verbose              display verbose output")
		fmt.Fprintln(fs.Output(), "  -version                  display version")
		fmt.Fprintln(fs.Output(), "")
		fmt.Fprintln(fs.Output(), "CONFIG:")
		fmt.Fprintln(fs.Output(), "  -retry int                maximum number of retries for requests (default 2)")
		fmt.Fprintln(fs.Output(), "  -timeout duration         per-request timeout (default 10s)")
		fmt.Fprintln(fs.Output(), "  -c, -concurrency int      number of concurrent workers (default 10)")
		fmt.Fprintln(fs.Output(), "  -fr, -follow-redirects    follow http redirects")
	}

	return fs, cfg, &inputs, &matchTech, &filterTech, &matchCategory, &filterCategory
}

func parseConfig(args []string, stderr io.Writer) (*config, error) {
	fs, cfg, inputs, matchTech, filterTech, matchCategory, filterCategory := newFlagSet(stderr)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	cfg.Inputs = append([]string(nil), *inputs...)
	cfg.MatchTech = append([]string(nil), *matchTech...)
	cfg.FilterTech = append([]string(nil), *filterTech...)
	cfg.MatchCategory = append([]string(nil), *matchCategory...)
	cfg.FilterCategory = append([]string(nil), *filterCategory...)
	if cfg.Retry < 1 {
		return nil, fmt.Errorf("-retry must be at least 1")
	}
	if cfg.Workers < 1 {
		return nil, fmt.Errorf("-concurrency must be at least 1")
	}
	return cfg, nil
}
