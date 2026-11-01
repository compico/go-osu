package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// mirror is a fake of the mirror. Handlers can be replaced per beatmap.
type mirror struct {
	t *testing.T

	mu       sync.Mutex
	requests map[string]int
	agents   map[string]bool
	handlers map[string]func(w http.ResponseWriter, attempt int)

	srv *httptest.Server
}

func osuFile(id int) string {
	return fmt.Sprintf("osu file format v14\n\n[Metadata]\nBeatmapID:%d\n", id)
}

func newMirror(t *testing.T) *mirror {
	m := &mirror{
		t:        t,
		requests: map[string]int{},
		agents:   map[string]bool{},
		handlers: map[string]func(http.ResponseWriter, int){},
	}

	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/")

		m.mu.Lock()
		m.requests[id]++
		attempt := m.requests[id]
		m.agents[r.UserAgent()] = true
		handler := m.handlers[id]
		m.mu.Unlock()

		if handler != nil {
			handler(w, attempt)
			return
		}

		n, err := strconv.Atoi(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, osuFile(n))
	}))
	t.Cleanup(m.srv.Close)

	return m
}

func (m *mirror) count(id int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests[strconv.Itoa(id)]
}

func (m *mirror) on(id int, h func(w http.ResponseWriter, attempt int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[strconv.Itoa(id)] = h
}

func newFetcher(m *mirror, dir string) *fetcher {
	return &fetcher{
		client:  m.srv.Client(),
		baseURL: m.srv.URL,
		dir:     dir,
		retries: 3,
		backoff: time.Second,
		sleep:   func(context.Context, time.Duration) error { return nil },
		logf:    func(string, ...any) {},
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFetchAllDownloadsEveryMap(t *testing.T) {
	m := newMirror(t)
	dir := filepath.Join(t.TempDir(), "nested", "test_data")
	f := newFetcher(m, dir)

	sum := f.fetchAll(context.Background(), []int{215, 1695388, 5066071}, 2)

	if sum.downloaded != 3 || sum.cached != 0 || len(sum.failed) != 0 {
		t.Fatalf("summary = %+v, want 3 downloaded", sum)
	}
	for _, id := range []int{215, 1695388, 5066071} {
		if got := readFile(t, filepath.Join(dir, fmt.Sprintf("%d.osu", id))); got != osuFile(id) {
			t.Errorf("%d.osu = %q, want %q", id, got, osuFile(id))
		}
	}
	if !m.agents[userAgent] || len(m.agents) != 1 {
		t.Errorf("User-Agent headers = %v, want only %q", m.agents, userAgent)
	}
}

func TestFetchSkipsExistingFilesUnlessForced(t *testing.T) {
	m := newMirror(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "215.osu")
	if err := os.WriteFile(path, []byte("osu file format v14\nlocal copy\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := newFetcher(m, dir)
	if downloaded, err := f.fetch(context.Background(), 215); err != nil || downloaded {
		t.Fatalf("fetch = (%v, %v), want the existing file to be kept", downloaded, err)
	}
	if m.count(215) != 0 {
		t.Errorf("%d requests for a file that is already there", m.count(215))
	}
	if !strings.Contains(readFile(t, path), "local copy") {
		t.Error("existing file was overwritten")
	}

	f.force = true
	if downloaded, err := f.fetch(context.Background(), 215); err != nil || !downloaded {
		t.Fatalf("forced fetch = (%v, %v), want a download", downloaded, err)
	}
	if got := readFile(t, path); got != osuFile(215) {
		t.Errorf("file after forced fetch = %q", got)
	}
}

func TestFetchReplacesEmptyFile(t *testing.T) {
	m := newMirror(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "215.osu"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if downloaded, err := newFetcher(m, dir).fetch(context.Background(), 215); err != nil || !downloaded {
		t.Fatalf("fetch = (%v, %v), want a download of the empty file", downloaded, err)
	}
}

func TestFetchNotFoundIsPermanentAndDoesNotStopOthers(t *testing.T) {
	m := newMirror(t)
	m.on(404, func(w http.ResponseWriter, _ int) { http.Error(w, "gone", http.StatusNotFound) })
	dir := t.TempDir()

	sum := newFetcher(m, dir).fetchAll(context.Background(), []int{215, 404, 3398802}, 1)

	if sum.downloaded != 2 || len(sum.failed) != 1 || sum.failed[404] == nil {
		t.Fatalf("summary = %+v, want 2 downloaded and 404 failed", sum)
	}
	if !strings.Contains(sum.failed[404].Error(), "not found") {
		t.Errorf("error = %v, want it to say the map was not found", sum.failed[404])
	}
	if m.count(404) != 1 {
		t.Errorf("a missing map was requested %d times, want 1 (no retries)", m.count(404))
	}
}

func TestFetchRetriesServerErrors(t *testing.T) {
	m := newMirror(t)
	m.on(215, func(w http.ResponseWriter, attempt int) {
		switch attempt {
		case 1:
			http.Error(w, "busy", http.StatusServiceUnavailable)
		case 2:
			w.Header().Set("Retry-After", "7")
			http.Error(w, "slow down", http.StatusTooManyRequests)
		default:
			fmt.Fprint(w, osuFile(215))
		}
	})

	var delays []time.Duration
	f := newFetcher(m, t.TempDir())
	f.sleep = func(_ context.Context, d time.Duration) error {
		delays = append(delays, d)
		return nil
	}

	if downloaded, err := f.fetch(context.Background(), 215); err != nil || !downloaded {
		t.Fatalf("fetch = (%v, %v), want success after retries", downloaded, err)
	}
	if m.count(215) != 3 {
		t.Errorf("requests = %d, want 3", m.count(215))
	}
	// The first delay is the plain backoff, the second is what Retry-After asked.
	if len(delays) != 2 || delays[0] != time.Second || delays[1] != 7*time.Second {
		t.Errorf("delays = %v, want [1s 7s]", delays)
	}
}

func TestFetchGivesUpAfterRetries(t *testing.T) {
	m := newMirror(t)
	m.on(215, func(w http.ResponseWriter, _ int) { http.Error(w, "down", http.StatusBadGateway) })
	dir := t.TempDir()
	f := newFetcher(m, dir)
	f.retries = 2

	_, err := f.fetch(context.Background(), 215)
	if err == nil || !strings.Contains(err.Error(), "giving up after 3 attempts") {
		t.Fatalf("err = %v, want it to give up after 3 attempts", err)
	}
	if m.count(215) != 3 {
		t.Errorf("requests = %d, want 3", m.count(215))
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("failed download left files behind: %v", entries)
	}
}

func TestFetchRetryAfterIsCapped(t *testing.T) {
	m := newMirror(t)
	m.on(215, func(w http.ResponseWriter, attempt int) {
		if attempt == 1 {
			w.Header().Set("Retry-After", "86400")
			http.Error(w, "later", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, osuFile(215))
	})

	var delays []time.Duration
	f := newFetcher(m, t.TempDir())
	f.sleep = func(_ context.Context, d time.Duration) error {
		delays = append(delays, d)
		return nil
	}

	if _, err := f.fetch(context.Background(), 215); err != nil {
		t.Fatal(err)
	}
	if len(delays) != 1 || delays[0] != maxRetryAfter {
		t.Errorf("delays = %v, want [%s]", delays, maxRetryAfter)
	}
}

func TestFetchRejectsAnswersThatAreNotOsuFiles(t *testing.T) {
	m := newMirror(t)
	m.on(1, func(w http.ResponseWriter, _ int) { fmt.Fprint(w, "<html><body>Just a moment...</body></html>") })
	m.on(2, func(w http.ResponseWriter, _ int) {})
	m.on(3, func(w http.ResponseWriter, _ int) { http.Error(w, "forbidden", http.StatusForbidden) })
	dir := t.TempDir()
	f := newFetcher(m, dir)

	for _, id := range []int{1, 2, 3} {
		if _, err := f.fetch(context.Background(), id); err == nil {
			t.Errorf("beatmap %d: no error", id)
		}
		if m.count(id) != 1 {
			t.Errorf("beatmap %d: requested %d times, want 1 (not retryable)", id, m.count(id))
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("rejected answers left files behind: %v", entries)
	}
}

func TestLooksLikeOsuFile(t *testing.T) {
	tests := map[string]bool{
		"osu file format v14\n":          true,
		"\xef\xbb\xbfosu file format v9": true,
		"\r\n  osu file format v128":     true,
		"":                               false,
		"<html>":                         false,
		"osu file format":                false,
		"[General]\nosu file format v14": false,
	}
	for in, want := range tests {
		if got := looksLikeOsuFile([]byte(in)); got != want {
			t.Errorf("looksLikeOsuFile(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFetchStopsWhenCancelled(t *testing.T) {
	m := newMirror(t)
	m.on(215, func(w http.ResponseWriter, _ int) { http.Error(w, "down", http.StatusServiceUnavailable) })

	ctx, cancel := context.WithCancel(context.Background())
	f := newFetcher(m, t.TempDir())
	f.retries = 10
	f.sleep = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return ctx.Err()
	}

	if _, err := f.fetch(ctx, 215); err == nil {
		t.Fatal("no error after cancellation")
	}
	if m.count(215) != 1 {
		t.Errorf("requests = %d, want 1", m.count(215))
	}
}

func TestLoadManifest(t *testing.T) {
	write := func(content string) string {
		path := filepath.Join(t.TempDir(), "maps.json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	ids, err := loadManifest(write(`[215, 1695388, 215]`))
	if err != nil || len(ids) != 2 || ids[0] != 215 || ids[1] != 1695388 {
		t.Errorf("ids = %v, err = %v, want [215 1695388]", ids, err)
	}

	for name, content := range map[string]string{
		"empty list": `[]`,
		"not a list": `{"ids": [215]}`,
		"strings":    `["215"]`,
		"negative":   `[215, -4]`,
		"zero":       `[0]`,
		"broken":     `[215,`,
	} {
		if ids, err := loadManifest(write(content)); err == nil {
			t.Errorf("%s: ids = %v, want an error", name, ids)
		}
	}

	if _, err := loadManifest(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("missing manifest: no error")
	}
}

func TestRunEndToEnd(t *testing.T) {
	m := newMirror(t)
	m.on(404, func(w http.ResponseWriter, _ int) { http.NotFound(w, nil) })

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "maps.json")
	if err := os.WriteFile(manifest, []byte(`[215, 2156120, 404]`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, "test_data")
	args := []string{"-manifest", manifest, "-dir", dir, "-base-url", m.srv.URL + "/", "-retries", "0"}

	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), args, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1 because of the missing map\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "3 maps: 2 downloaded, 0 already present, 1 failed") {
		t.Errorf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "beatmap 404:") {
		t.Errorf("stderr = %q, want it to name the failed beatmap", stderr.String())
	}

	// Without the missing map a second run finds everything in place and makes no requests.
	if err := os.WriteFile(manifest, []byte(`[215, 2156120]`), 0o644); err != nil {
		t.Fatal(err)
	}
	before := m.count(215) + m.count(2156120)

	stdout.Reset()
	stderr.Reset()
	if code := run(context.Background(), args, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "2 maps: 0 downloaded, 2 already present, 0 failed") {
		t.Errorf("stdout = %q", stdout.String())
	}
	if after := m.count(215) + m.count(2156120); after != before {
		t.Errorf("second run made %d requests, want none", after-before)
	}
}

func TestRunWithBadArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run(context.Background(), []string{"-nope"}, &stdout, &stderr); code != 2 {
		t.Errorf("unknown flag: exit code = %d, want 2", code)
	}
	if code := run(context.Background(), []string{"-manifest", filepath.Join(t.TempDir(), "missing.json")}, &stdout, &stderr); code != 1 {
		t.Errorf("missing manifest: exit code = %d, want 1", code)
	}
}
