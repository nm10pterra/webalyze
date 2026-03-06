package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func CollectTargets(direct []string, listPath string, stdin io.Reader, stdinAvailable bool) ([]string, error) {
	var all []string
	all = append(all, direct...)

	if listPath != "" {
		fileTargets, err := readTargetsFromFile(listPath)
		if err != nil {
			return nil, err
		}
		all = append(all, fileTargets...)
	}

	if stdinAvailable && stdin != nil {
		stdinTargets, err := readTargets(stdin)
		if err != nil {
			return nil, err
		}
		all = append(all, stdinTargets...)
	}

	seen := make(map[string]struct{}, len(all))
	out := make([]string, 0, len(all))
	for _, item := range all {
		item = normalizeTarget(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

func readTargetsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open list file: %w", err)
	}
	defer f.Close()
	return readTargets(f)
}

func readTargets(r io.Reader) ([]string, error) {
	var out []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read targets: %w", err)
	}
	return out, nil
}

func normalizeTarget(target string) string {
	target = strings.TrimSpace(target)
	target = strings.Trim(target, "\"'")
	return target
}
