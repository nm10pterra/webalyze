package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
)

type Config struct {
	Retry          int
	Timeout        time.Duration
	Workers        int
	FollowRedirect bool
}

type Result struct {
	Target       string
	URL          string
	Technologies []string
	Categories   []string
	TechToCats   map[string][]string
	Err          error
}

func Run(ctx context.Context, targets []string, cfg Config) []Result {
	results := make([]Result, 0, len(targets))
	var mu sync.Mutex
	RunStream(ctx, targets, cfg, func(result Result) {
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	})
	return results
}

func RunStream(ctx context.Context, targets []string, cfg Config, onResult func(Result)) {
	if len(targets) == 0 || onResult == nil {
		return
	}
	workers := cfg.Workers
	if workers < 1 {
		workers = runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
	}
	if workers > len(targets) {
		workers = len(targets)
	}

	wappalyzerClient, err := wappalyzer.New()
	if err != nil {
		for _, target := range targets {
			onResult(Result{Target: target, Err: fmt.Errorf("init wappalyzer: %w", err)})
		}
		return
	}
	client := newHTTPClient(cfg)

	jobs := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				onResult(processTarget(ctx, client, wappalyzerClient, target, cfg.Retry))
			}
		}()
	}

enqueueLoop:
	for _, target := range targets {
		select {
		case <-ctx.Done():
			break enqueueLoop
		case jobs <- target:
		}
	}
	close(jobs)
	wg.Wait()
}

func processTarget(ctx context.Context, client *http.Client, wappalyzerClient *wappalyzer.Wappalyze, target string, retry int) Result {
	result := Result{Target: target}
	urlCandidates := candidates(target)

	var errs []error
	for _, candidate := range urlCandidates {
		finalURL, tech, cats, techToCats, reqErr := fingerprintURL(ctx, client, wappalyzerClient, candidate, retry)
		if reqErr != nil {
			errs = append(errs, reqErr)
			continue
		}
		result.URL = finalURL
		result.Technologies = tech
		result.Categories = cats
		result.TechToCats = techToCats
		result.Err = nil
		return result
	}
	result.Err = errors.Join(errs...)
	return result
}

func runSequential(ctx context.Context, targets []string, cfg Config) []Result {
	wappalyzerClient, err := wappalyzer.New()
	if err != nil {
		results := make([]Result, 0, len(targets))
		for _, target := range targets {
			results = append(results, Result{Target: target, Err: fmt.Errorf("init wappalyzer: %w", err)})
		}
		return results
	}

	client := newHTTPClient(cfg)

	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		result := Result{Target: target}
		urlCandidates := candidates(target)

		var errs []error
		for _, candidate := range urlCandidates {
			finalURL, tech, cats, techToCats, reqErr := fingerprintURL(ctx, client, wappalyzerClient, candidate, cfg.Retry)
			if reqErr != nil {
				errs = append(errs, reqErr)
				continue
			}
			result.URL = finalURL
			result.Technologies = tech
			result.Categories = cats
			result.TechToCats = techToCats
			result.Err = nil
			break
		}

		if result.URL == "" {
			result.Err = errors.Join(errs...)
		}
		results = append(results, result)
	}

	return results
}

func candidates(target string) []string {
	raw := strings.TrimSpace(target)
	if raw == "" {
		return nil
	}

	u, err := url.Parse(raw)
	if err == nil && u.Scheme != "" {
		return []string{raw}
	}
	return []string{"https://" + raw, "http://" + raw}
}

func fingerprintURL(ctx context.Context, client *http.Client, w *wappalyzer.Wappalyze, targetURL string, retry int) (string, []string, []string, map[string][]string, error) {
	var lastErr error
	for attempt := 1; attempt <= retry; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return "", nil, nil, nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", "webalyze/"+appVersion())

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request attempt %d: %w", attempt, err)
			continue
		}

		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read body attempt %d: %w", attempt, readErr)
			continue
		}

		if resp.StatusCode >= http.StatusInternalServerError && attempt < retry {
			lastErr = fmt.Errorf("status code %d", resp.StatusCode)
			continue
		}

		appInfo := w.FingerprintWithInfo(resp.Header, data)
		tech := make([]string, 0, len(appInfo))
		catSet := make(map[string]struct{})
		techToCats := make(map[string][]string, len(appInfo))
		for name, info := range appInfo {
			tech = append(tech, name)
			perTechSet := make(map[string]struct{})
			for _, cat := range info.Categories {
				cat = strings.TrimSpace(cat)
				if cat == "" {
					continue
				}
				catSet[cat] = struct{}{}
				perTechSet[cat] = struct{}{}
			}
			perTechCats := make([]string, 0, len(perTechSet))
			for cat := range perTechSet {
				perTechCats = append(perTechCats, cat)
			}
			sort.Strings(perTechCats)
			techToCats[name] = perTechCats
		}
		sort.Strings(tech)
		cats := make([]string, 0, len(catSet))
		for cat := range catSet {
			cats = append(cats, cat)
		}
		sort.Strings(cats)
		resultTechToCats := make(map[string][]string, len(tech))
		for _, techName := range tech {
			resultTechToCats[techName] = techToCats[techName]
		}
		return resp.Request.URL.String(), tech, cats, resultTechToCats, nil
	}
	return "", nil, nil, nil, lastErr
}

func appVersion() string {
	return "dev"
}

func newHTTPClient(cfg Config) *http.Client {
	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	if !cfg.FollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return client
}
