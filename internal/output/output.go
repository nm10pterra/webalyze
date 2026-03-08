package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/nm10pterra/webalyze/internal/runner"
)

type Options struct {
	JSONL          bool
	Color          bool
	Silent         bool
	Verbose        bool
	MatchTech      []string
	FilterTech     []string
	MatchCategory  []string
	FilterCategory []string
}

type jsonLine struct {
	Target     string              `json:"target"`
	URL        string              `json:"url,omitempty"`
	Categories map[string][]string `json:"categories,omitempty"`
	Error      string              `json:"error,omitempty"`
}

type compiledFilters struct {
	matchTechSet      map[string]struct{}
	filterTechSet     map[string]struct{}
	matchCategorySet  map[string]struct{}
	filterCategorySet map[string]struct{}
}

type SummaryAggregator struct {
	opts          Options
	filters       compiledFilters
	categoryCount map[string]int
	totalTech     int
}

type SyncWriter struct {
	mu      sync.Mutex
	writers []io.Writer
}

func NewSyncWriter(writers ...io.Writer) *SyncWriter {
	return &SyncWriter{writers: writers}
}

func (w *SyncWriter) WriteLine(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, writer := range w.writers {
		if _, err := io.WriteString(writer, line); err != nil {
			return err
		}
		if !strings.HasSuffix(line, "\n") {
			if _, err := io.WriteString(writer, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *SyncWriter) WriteString(text string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, writer := range w.writers {
		if _, err := io.WriteString(writer, text); err != nil {
			return err
		}
	}
	return nil
}

func CompileFilters(opts Options) compiledFilters {
	return compiledFilters{
		matchTechSet:      normalizeSet(opts.MatchTech),
		filterTechSet:     normalizeSet(opts.FilterTech),
		matchCategorySet:  normalizeSet(opts.MatchCategory),
		filterCategorySet: normalizeSet(opts.FilterCategory),
	}
}

func NewSummaryAggregator(opts Options) *SummaryAggregator {
	return &SummaryAggregator{
		opts:          opts,
		filters:       CompileFilters(opts),
		categoryCount: map[string]int{},
	}
}

func (s *SummaryAggregator) Add(result runner.Result) {
	if !matchesCompiledFilters(result, s.filters) {
		return
	}
	if result.Err != nil {
		return
	}
	if s.opts.Silent && len(result.Technologies) == 0 {
		return
	}
	for _, tech := range result.Technologies {
		s.totalTech++
		for _, category := range result.TechToCats[tech] {
			s.categoryCount[category]++
		}
	}
}

func (s *SummaryAggregator) Line() string {
	return summaryLineFromCounts(s.totalTech, s.categoryCount, s.opts.Color)
}

func FormatResult(result runner.Result, opts Options) (string, bool, error) {
	return formatResultWithFilters(result, opts, CompileFilters(opts))
}

func formatResultWithFilters(result runner.Result, opts Options, filters compiledFilters) (string, bool, error) {
	if !matchesCompiledFilters(result, filters) {
		return "", false, nil
	}
	if opts.JSONL {
		categoryToTechSet := make(map[string]map[string]struct{})
		for _, tech := range result.Technologies {
			for _, category := range result.TechToCats[tech] {
				if _, ok := categoryToTechSet[category]; !ok {
					categoryToTechSet[category] = make(map[string]struct{})
				}
				categoryToTechSet[category][tech] = struct{}{}
			}
		}

		categories := make(map[string][]string, len(categoryToTechSet))
		for category, techSet := range categoryToTechSet {
			techs := make([]string, 0, len(techSet))
			for tech := range techSet {
				techs = append(techs, tech)
			}
			sort.Strings(techs)
			categories[category] = techs
		}

		line := jsonLine{
			Target:     result.Target,
			URL:        result.URL,
			Categories: categories,
		}
		if result.Err != nil && opts.Verbose {
			line.Error = result.Err.Error()
		}
		encoded, err := json.Marshal(line)
		if err != nil {
			return "", false, fmt.Errorf("json encode result for %q: %w", result.Target, err)
		}
		return string(encoded), true, nil
	}

	if result.Err != nil {
		if opts.Verbose {
			return fmt.Sprintf("%s error: %v", result.Target, result.Err), true, nil
		}
		return "", false, nil
	}
	if opts.Silent && len(result.Technologies) == 0 {
		return "", false, nil
	}

	styledTech := make([]string, 0, len(result.Technologies))
	for _, tech := range result.Technologies {
		styledTech = append(styledTech, styleTechnologyToken(tech, result.TechToCats[tech], opts.Color))
	}
	if len(styledTech) == 0 {
		return result.Target, true, nil
	}
	return fmt.Sprintf("%s %s", result.Target, strings.Join(styledTech, " ")), true, nil
}

func Render(results []runner.Result, opts Options) (string, int, error) {
	var b strings.Builder
	failures := 0
	filters := CompileFilters(opts)
	for _, result := range results {
		if result.Err != nil {
			failures++
		}
	}

	for _, result := range results {
		line, include, err := formatResultWithFilters(result, opts, filters)
		if err != nil {
			return "", failures, err
		}
		if !include {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String(), failures, nil
}

func SummaryLine(results []runner.Result, opts Options) string {
	agg := NewSummaryAggregator(opts)
	for _, result := range results {
		agg.Add(result)
	}
	return agg.Line()
}

func filteredResults(results []runner.Result, opts Options) []runner.Result {
	filters := CompileFilters(opts)

	filtered := make([]runner.Result, 0, len(results))
	for _, result := range results {
		if !matchesCompiledFilters(result, filters) {
			continue
		}
		filtered = append(filtered, result)
	}
	return filtered
}

func matchesCompiledFilters(result runner.Result, filters compiledFilters) bool {
	if !passesFilters(result.Technologies, filters.matchTechSet, filters.filterTechSet) {
		return false
	}
	if !passesFilters(result.Categories, filters.matchCategorySet, filters.filterCategorySet) {
		return false
	}
	return true
}

func summaryLineFromCounts(totalTech int, categoryCount map[string]int, color bool) string {
	if len(categoryCount) == 0 {
		return fmt.Sprintf("Found results %d (none: 0)", totalTech)
	}

	type countPair struct {
		Name  string
		Count int
	}
	pairs := make([]countPair, 0, len(categoryCount))
	for name, count := range categoryCount {
		pairs = append(pairs, countPair{Name: name, Count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Name < pairs[j].Name
	})

	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		categoryName := pair.Name
		if color {
			categoryName = colorize(categoryName, colorForCategory(pair.Name))
		}
		chunk := fmt.Sprintf("%s: %d", categoryName, pair.Count)
		parts = append(parts, chunk)
	}
	return fmt.Sprintf("Found results %d (%s)", totalTech, strings.Join(parts, ", "))
}

const (
	ansiReset   = "\x1b[0m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
	ansiWhite   = "\x1b[37m"
)

var categoryColors = map[string]string{
	"cdn":                   ansiCyan,
	"web servers":           ansiGreen,
	"cms":                   ansiMagenta,
	"javascript frameworks": ansiYellow,
	"programming languages": ansiBlue,
}

func styleTechnologyToken(name string, categories []string, colorEnabled bool) string {
	token := "[" + name + "]"
	if !colorEnabled {
		return token
	}
	return colorize(token, colorForCategory(firstCategory(categories)))
}

func deterministicColor(seed string) string {
	options := []string{ansiGreen, ansiYellow, ansiBlue, ansiMagenta, ansiCyan, ansiRed}
	if seed == "" {
		return ansiBlue
	}
	sum := 0
	for _, b := range []byte(seed) {
		sum += int(b)
	}
	return options[sum%len(options)]
}

func firstCategory(categories []string) string {
	if len(categories) == 0 {
		return ""
	}
	return categories[0]
}

func colorForCategory(category string) string {
	normalized := strings.ToLower(strings.TrimSpace(category))
	if mappedColor, ok := categoryColors[normalized]; ok {
		return mappedColor
	}
	return deterministicColor(normalized)
}

func colorize(s, ansiColor string) string {
	return ansiColor + s + ansiReset
}

func normalizeSet(items []string) map[string]struct{} {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		out[item] = struct{}{}
	}
	return out
}

func passesFilters(tech []string, matchSet, filterSet map[string]struct{}) bool {
	if len(matchSet) == 0 && len(filterSet) == 0 {
		return true
	}

	normalized := make([]string, 0, len(tech))
	for _, name := range tech {
		normalized = append(normalized, strings.ToLower(name))
	}

	if len(matchSet) > 0 {
		matched := false
		for _, item := range normalized {
			if _, ok := matchSet[item]; ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(filterSet) > 0 {
		for _, item := range normalized {
			if _, ok := filterSet[item]; ok {
				return false
			}
		}
	}
	return true
}
