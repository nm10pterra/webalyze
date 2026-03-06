package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectTargets_MergesAndDedupes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	listFile := filepath.Join(dir, "targets.txt")
	content := "example.com\n# comment\nhttps://go.dev\nexample.com\n"
	if err := os.WriteFile(listFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write list file: %v", err)
	}

	stdin := strings.NewReader("go.dev\n\nexample.org\n")
	got, err := CollectTargets([]string{"example.com", "  https://go.dev "}, listFile, stdin, true)
	if err != nil {
		t.Fatalf("collect targets: %v", err)
	}

	want := []string{"example.com", "https://go.dev", "go.dev", "example.org"}
	if len(got) != len(want) {
		t.Fatalf("target len mismatch got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("target mismatch at %d got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestCollectTargets_StdinDisabled(t *testing.T) {
	t.Parallel()

	got, err := CollectTargets([]string{"example.com"}, "", strings.NewReader("ignored.com"), false)
	if err != nil {
		t.Fatalf("collect targets: %v", err)
	}
	if len(got) != 1 || got[0] != "example.com" {
		t.Fatalf("unexpected targets: %+v", got)
	}
}
