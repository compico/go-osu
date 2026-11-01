// Command fetchrefs downloads the .osu files the skills reference test needs.
//
// The files are not kept in the repository. The IDs of the beatmaps live in
// pkg/osu/skills/reference_maps.json and the files are fetched one by one from
// a mirror into pkg/osu/skills/test_data, where git ignores them. Files that
// are already there are left alone, so a second run (or a CI cache hit) does
// not touch the network.
//
//	go run ./internal/tools/fetchrefs
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultManifest = "pkg/osu/skills/reference_maps.json"
	defaultDir      = "pkg/osu/skills/test_data"
	defaultBaseURL  = "https://mirror.hinamizawa.ai/api/osu"

	// The mirror asks for a descriptive User-Agent instead of the default one
	// of the HTTP library.
	userAgent = "go-osu-fetchrefs/1.0 (+https://github.com/compico/go-osu)"

	// A .osu file is a few hundred KB at most; anything bigger is not one.
	maxFileSize = 16 << 20

	maxRetryAfter = 30 * time.Second

	osuFileMagic = "osu file format v"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

type options struct {
	manifest    string
	dir         string
	baseURL     string
	retries     int
	concurrency int
	timeout     time.Duration
	force       bool
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var opt options

	fs := flag.NewFlagSet("fetchrefs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opt.manifest, "manifest", defaultManifest, "JSON file with the list of beatmap IDs")
	fs.StringVar(&opt.dir, "dir", defaultDir, "directory to download the .osu files into")
	fs.StringVar(&opt.baseURL, "base-url", defaultBaseURL, "mirror URL; the beatmap ID is appended to it")
	fs.IntVar(&opt.retries, "retries", 5, "how many times to retry a failed request")
	fs.IntVar(&opt.concurrency, "concurrency", 4, "how many files to download at once")
	fs.DurationVar(&opt.timeout, "timeout", 30*time.Second, "timeout of a single request")
	fs.BoolVar(&opt.force, "force", false, "download files that already exist too")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ids, err := loadManifest(opt.manifest)
	if err != nil {
		fmt.Fprintf(stderr, "fetchrefs: %v\n", err)
		return 1
	}

	var logMu sync.Mutex // downloads run concurrently and share the writer

	f := &fetcher{
		client:  &http.Client{Timeout: opt.timeout},
		baseURL: strings.TrimRight(opt.baseURL, "/"),
		dir:     opt.dir,
		retries: opt.retries,
		backoff: time.Second,
		force:   opt.force,
		sleep:   sleep,
		logf: func(format string, args ...any) {
			logMu.Lock()
			defer logMu.Unlock()
			fmt.Fprintf(stdout, format+"\n", args...)
		},
	}

	summary := f.fetchAll(ctx, ids, max(1, opt.concurrency))

	fmt.Fprintf(stdout, "%d maps: %d downloaded, %d already present, %d failed\n",
		len(ids), summary.downloaded, summary.cached, len(summary.failed))

	if len(summary.failed) > 0 {
		failedIDs := make([]int, 0, len(summary.failed))
		for id := range summary.failed {
			failedIDs = append(failedIDs, id)
		}
		sort.Ints(failedIDs)

		for _, id := range failedIDs {
			fmt.Fprintf(stderr, "fetchrefs: beatmap %d: %v\n", id, summary.failed[id])
		}
		return 1
	}

	return 0
}

// loadManifest reads a JSON array of beatmap IDs. Duplicates are dropped.
func loadManifest(path string) ([]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var all []int
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("manifest %s must be a JSON array of beatmap IDs: %w", path, err)
	}

	seen := make(map[int]bool, len(all))
	ids := make([]int, 0, len(all))
	for _, id := range all {
		if id <= 0 {
			return nil, fmt.Errorf("manifest %s: invalid beatmap ID %d", path, id)
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("manifest %s has no beatmap IDs", path)
	}

	return ids, nil
}

type fetcher struct {
	client  *http.Client
	baseURL string
	dir     string
	retries int
	backoff time.Duration // delay before the first retry; doubles after each one
	force   bool

	sleep func(ctx context.Context, d time.Duration) error
	logf  func(format string, args ...any)
}

type summary struct {
	downloaded int
	cached     int
	failed     map[int]error
}

// fetchAll downloads every ID with at most concurrency requests in flight.
// A failure of one beatmap does not stop the others.
func (f *fetcher) fetchAll(ctx context.Context, ids []int, concurrency int) summary {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sum = summary{failed: make(map[int]error)}
		sem = make(chan struct{}, concurrency)
	)

	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}

		go func(id int) {
			defer wg.Done()
			defer func() { <-sem }()

			downloaded, err := f.fetch(ctx, id)

			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				sum.failed[id] = err
			case downloaded:
				sum.downloaded++
			default:
				sum.cached++
			}
		}(id)
	}
	wg.Wait()

	return sum
}

func (f *fetcher) path(id int) string {
	return filepath.Join(f.dir, strconv.Itoa(id)+".osu")
}

// fetch makes sure the file of one beatmap is in the directory. It reports
// whether it had to download it.
func (f *fetcher) fetch(ctx context.Context, id int) (downloaded bool, err error) {
	dst := f.path(id)

	if !f.force {
		if info, err := os.Stat(dst); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			return false, nil
		}
	}

	body, err := f.download(ctx, id)
	if err != nil {
		return false, err
	}

	if err := writeFileAtomic(dst, body); err != nil {
		return false, err
	}
	f.logf("downloaded %d (%d bytes)", id, len(body))

	return true, nil
}

// permanentError marks a failure that retrying cannot fix.
type permanentError struct{ error }

func (e permanentError) Unwrap() error { return e.error }

// download requests the file, retrying server errors, rate limiting and network
// failures with a growing delay.
func (f *fetcher) download(ctx context.Context, id int) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= f.retries; attempt++ {
		body, retryAfter, err := f.get(ctx, id)
		if err == nil {
			return body, nil
		}

		var permanent permanentError
		if errors.As(err, &permanent) || ctx.Err() != nil {
			return nil, err
		}
		lastErr = err

		if attempt == f.retries {
			break
		}

		delay := f.backoff << attempt
		if retryAfter > 0 {
			delay = min(retryAfter, maxRetryAfter)
		}
		f.logf("beatmap %d: %v; retrying in %s (%d/%d)", id, err, delay, attempt+1, f.retries)

		if err := f.sleep(ctx, delay); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("giving up after %d attempts: %w", f.retries+1, lastErr)
}

// get makes a single request. retryAfter is what the server asked for, if it
// did.
func (f *fetcher) get(ctx context.Context, id int) (body []byte, retryAfter time.Duration, err error) {
	url := f.baseURL + "/" + strconv.Itoa(id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, permanentError{err}
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/plain, */*")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
	case resp.StatusCode == http.StatusNotFound:
		return nil, 0, permanentError{errors.New("not found on the mirror (the map may have been removed)")}
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return nil, parseRetryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("mirror answered %s", resp.Status)
	default:
		return nil, 0, permanentError{fmt.Errorf("mirror answered %s", resp.Status)}
	}

	body, err = io.ReadAll(io.LimitReader(resp.Body, maxFileSize+1))
	if err != nil {
		return nil, 0, err
	}
	if len(body) > maxFileSize {
		return nil, 0, permanentError{fmt.Errorf("answer is larger than %d bytes, not a .osu file", maxFileSize)}
	}
	if !looksLikeOsuFile(body) {
		return nil, 0, permanentError{errors.New("answer is not a .osu file (it does not start with \"osu file format\")")}
	}

	return body, 0, nil
}

// parseRetryAfter understands the delay-in-seconds form of Retry-After.
func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// looksLikeOsuFile reports whether data starts like a .osu file, which tells it
// apart from an HTML error page served with a 200 status.
func looksLikeOsuFile(data []byte) bool {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	data = bytes.TrimLeft(data, " \t\r\n")
	return bytes.HasPrefix(data, []byte(osuFileMagic))
}

// writeFileAtomic writes data so that a reader, or a run interrupted halfway,
// never sees a partial file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".fetchrefs-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // a no-op once the file has been renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
