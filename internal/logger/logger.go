// Package logger provides a structured slog-based logger.
//
// Log entries are sent to two destinations:
//   - BrowserHandler: serialises records to JSON and pushes them into a channel
//     consumed by the realtime module (centrifuge → browser console).
//   - os.Stderr: only for Fatal/Error level so the operator sees critical issues
//     without opening a browser.
package logger

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/compico/go-osu/internal/config"
)

// LogEntry is the JSON payload sent to the browser via centrifuge.
type LogEntry struct {
	Level string         `json:"level"`
	Msg   string         `json:"msg"`
	Time  time.Time      `json:"time"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// BrowserHandler implements slog.Handler and pushes JSON-encoded LogEntry
// values into a buffered channel for consumption by the realtime module.
type BrowserHandler struct {
	level  slog.Level
	ch     chan []byte
	attrs  []slog.Attr
	groups []string
}

func newBrowserHandler(level slog.Level) *BrowserHandler {
	return &BrowserHandler{
		level: level,
		ch:    make(chan []byte, 256),
	}
}

// Chan returns the read-only channel of JSON-encoded log entries.
// The realtime module drains this channel and publishes to centrifuge.
func (h *BrowserHandler) Chan() <-chan []byte { return h.ch }

func (h *BrowserHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *BrowserHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]any, r.NumAttrs()+len(h.attrs))
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	entry := LogEntry{
		Level: r.Level.String(),
		Msg:   r.Message,
		Time:  r.Time,
	}
	if len(attrs) > 0 {
		entry.Attrs = attrs
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	select {
	case h.ch <- data:
	default: // channel full — drop rather than block
	}
	return nil
}

func (h *BrowserHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &c
}

func (h *BrowserHandler) WithGroup(name string) slog.Handler {
	c := *h
	c.groups = append(append([]string(nil), h.groups...), name)
	return &c
}

// multiHandler fans out a single slog.Record to multiple handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: hs}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: hs}
}

// New builds a *slog.Logger and the associated BrowserHandler.
//
// The logger fans out to:
//   - BrowserHandler (all levels >= cfg.Level) → centrifuge channel
//   - slog.TextHandler on stderr (Error+ only) → terminal for fatal cases
func New(cfg *config.LogConfig) (*slog.Logger, *BrowserHandler, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, nil, err
	}

	browser := newBrowserHandler(level)

	stderr := slog.NewTextHandler(io.Writer(os.Stderr), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(&multiHandler{
		handlers: []slog.Handler{stderr, browser},
	})

	return logger, browser, nil
}
