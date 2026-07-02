// Package realtime provides an embedded Centrifuge node for real-time
// communication with the browser.
//
// The node runs its own net/http server on a configurable port (separate from
// the main fiber server) so that WebSocket upgrade handling remains unaffected
// by fasthttp's request model.
//
// Channels:
//   - "logs" — JSON-encoded LogEntry values from the logger.BrowserHandler.
//
// Frontend usage (centrifuge-js):
//
//	import { Centrifuge } from 'centrifuge'
//	const c = new Centrifuge('ws://127.0.0.1:3001/connection/websocket')
//	c.newSubscription('logs').on('publication', ctx => console.log(ctx.data)).subscribe()
//	c.connect()
package realtime

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/centrifugal/centrifuge"
	"github.com/compico/go-osu/internal/config"
)

// bindAddr is hardcoded — the realtime server is always localhost-only.
const (
	LogChannel = "logs"
	bindAddr   = "127.0.0.1"
)

// Node wraps a centrifuge.Node and an HTTP server for WebSocket connections.
type Node struct {
	node   *centrifuge.Node
	server *http.Server
	logger *slog.Logger
}

// New creates a centrifuge Node and configures anonymous pub/sub access.
// Call Start to begin accepting connections and Stop for graceful shutdown.
func New(cfg *config.RealtimeConfig, logger *slog.Logger) (*Node, error) {
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel:   centrifuge.LogLevelError,
		LogHandler: newCentrifugeLogAdapter(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("realtime: create node: %w", err)
	}

	// Allow all anonymous connections and subscriptions.
	// Since this is localhost-only, no auth is needed.
	node.OnConnecting(func(_ context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{
			Credentials: &centrifuge.Credentials{UserID: ""},
		}, nil
	})

	node.OnConnect(func(c *centrifuge.Client) {
		c.OnSubscribe(func(_ centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			cb(centrifuge.SubscribeReply{}, nil)
		})
	})

	mux := http.NewServeMux()
	mux.Handle("/connection/websocket", centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}))

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", bindAddr, cfg.Port),
		Handler: mux,
	}

	return &Node{node: node, server: srv, logger: logger}, nil
}

// Start runs the centrifuge node and begins accepting WebSocket connections.
// It is non-blocking: the HTTP listener runs in a separate goroutine.
func (n *Node) Start() error {
	if err := n.node.Run(); err != nil {
		return fmt.Errorf("realtime: run node: %w", err)
	}

	go func() {
		n.logger.Info("realtime server listening", "addr", n.server.Addr)
		if err := n.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			n.logger.Error("realtime server error", "err", err)
		}
	}()

	return nil
}

// Stop shuts down the centrifuge node and the HTTP server gracefully.
func (n *Node) Stop(ctx context.Context) error {
	if err := n.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("realtime: shutdown http: %w", err)
	}
	if err := n.node.Shutdown(ctx); err != nil {
		return fmt.Errorf("realtime: shutdown node: %w", err)
	}
	return nil
}

// Publish sends raw JSON bytes to the given channel.
// Non-blocking: if there are no subscribers the message is silently dropped.
func (n *Node) Publish(channel string, data []byte) error {
	_, err := n.node.Publish(channel, data)
	return err
}

// DrainLogs reads from ch and publishes each entry to the logs channel.
// Runs until ctx is cancelled. Intended to be called in a goroutine.
func (n *Node) DrainLogs(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			if err := n.Publish(LogChannel, data); err != nil {
				// Ignore publish errors — no subscribers is the common case.
				_ = err
			}
		case <-ctx.Done():
			return
		}
	}
}

// centrifugeLogAdapter bridges centrifuge's internal logger to slog.
type centrifugeLogAdapter struct{ logger *slog.Logger }

func newCentrifugeLogAdapter(l *slog.Logger) func(centrifuge.LogEntry) {
	return func(e centrifuge.LogEntry) {
		switch e.Level {
		case centrifuge.LogLevelError:
			l.Error(e.Message, fieldsToArgs(e.Fields)...)
		case centrifuge.LogLevelWarn:
			l.Warn(e.Message, fieldsToArgs(e.Fields)...)
		default:
			l.Debug(e.Message, fieldsToArgs(e.Fields)...)
		}
	}
}

func fieldsToArgs(fields map[string]any) []any {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}
