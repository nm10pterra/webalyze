package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRun_UsesFinalRedirectURLWhenFollowingRedirects(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	results := Run(context.Background(), []string{server.URL + "/redirect"}, Config{
		Retry:          1,
		Timeout:        5 * time.Second,
		Workers:        1,
		FollowRedirect: true,
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got=%d", len(results))
	}
	if results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
	if got, want := results[0].URL, server.URL+"/final"; got != want {
		t.Fatalf("url mismatch got=%q want=%q", got, want)
	}
}

func TestRun_UsesOriginalURLWhenNotFollowingRedirects(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	results := Run(context.Background(), []string{server.URL + "/redirect"}, Config{
		Retry:          1,
		Timeout:        5 * time.Second,
		Workers:        1,
		FollowRedirect: false,
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got=%d", len(results))
	}
	if results[0].Err != nil {
		t.Fatalf("unexpected error: %v", results[0].Err)
	}
	if got, want := results[0].URL, server.URL+"/redirect"; got != want {
		t.Fatalf("url mismatch got=%q want=%q", got, want)
	}
}
