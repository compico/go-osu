// Package realtime is an in-process WebSocket pub/sub hub for pushing
// server-side events (logs, sync progress) to the browser. It's mounted
// directly on the existing Fiber app — no separate listener/port, and no
// dependency beyond github.com/gofiber/contrib/v3/websocket (itself a thin
// fasthttp-native wrapper, so this adds nothing indirect beyond what Fiber
// already pulls in).
//
// Mounting:
//
//	node := realtime.New(logger)
//	app.Get("/ws", node.Handler())
//
// Publishing:
//
//	// pre-marshaled JSON, e.g. from logger.BrowserHandler
//	go node.DrainLogs(ctx, browserHandler.Chan())
//
//	// any JSON-marshalable type
//	progressCh := make(chan service.ProgressEvent, 1024)
//	go realtime.Drain(node, ctx, realtime.ProgressChannel, progressCh)
//
// Wire protocol (JSON, one message per WebSocket frame, ws://<host>/ws —
// same origin/port as the rest of the app):
//
//	Client -> Server:
//	  {"type":"subscribe","channel":"logs"}
//	  {"type":"unsubscribe","channel":"logs"}
//
//	Server -> Client:
//	  {"type":"subscribed","channel":"logs"}
//	  {"type":"unsubscribed","channel":"logs"}
//	  {"type":"publication","channel":"logs","data":{...}}
//	  {"type":"error","error":"..."}
//
// A client that never subscribes to anything just sits idle (ping/pong
// keepalive only) — subscribing is required to receive any "publication".
package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

// Node is the pub/sub hub. Create one with New and mount Handler() on a
// route; it needs no Start/Stop of its own since it owns no listener —
// call Close() during graceful shutdown to disconnect clients.
type Node struct {
	hub    *hub
	logger *slog.Logger
}

// New creates a Node. It doesn't open any network resources itself — the
// Fiber app the returned Handler is mounted on does that.
func New(logger *slog.Logger) *Node {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "realtime")
	return &Node{
		hub:    newHub(logger),
		logger: logger,
	}
}

// Handler returns the fiber.Handler to mount on your WebSocket route, e.g.
// app.Get("/ws", node.Handler()). Origins are intentionally left at the
// contrib package's default (allow all): if this stops being localhost-only
// reachable, add websocket.Config{Origins: [...]}.
func (n *Node) Handler() fiber.Handler {
	return websocket.New(n.handleConn)
}

// ClientCount returns the number of currently connected WebSocket clients.
// Handy for a health/debug endpoint.
func (n *Node) ClientCount() int {
	return n.hub.count()
}

// Close disconnects every currently connected client. Call this during
// graceful shutdown, before or alongside app.Shutdown().
func (n *Node) Close() {
	n.hub.closeAll()
}

// Publish sends raw JSON bytes to every client currently subscribed to
// channel. A channel with no subscribers is a no-op; a slow client is
// dropped-for rather than allowed to stall delivery to everyone else (see
// hub.broadcast).
func (n *Node) Publish(channel string, data []byte) error {
	encoded, err := json.Marshal(serverMessage{
		Type:    "publication",
		Channel: channel,
		Data:    data,
	})
	if err != nil {
		return fmt.Errorf("realtime: marshal envelope: %w", err)
	}
	n.hub.broadcast(channel, encoded)
	return nil
}

// Drain reads typed values from ch, JSON-marshals each one, and Publishes
// the result on channel. Runs until ctx is cancelled or ch is closed;
// intended to be called in a goroutine. It's a free function rather than a
// method because Go doesn't allow generic methods on a non-generic
// receiver. Marshal failures are logged and skipped rather than aborting
// the drain loop — one bad event shouldn't kill the feed.
//
// Don't use this for already-marshaled []byte payloads (like log entries) —
// json.Marshal on a []byte base64-encodes it instead of passing it through.
// Use DrainLogs for those.
func Drain[T any](n *Node, ctx context.Context, channel string, ch <-chan T) {
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(v)
			if err != nil {
				n.logger.Error("realtime: failed to marshal value for publish", "channel", channel, "err", err)
				continue
			}
			if err := n.Publish(channel, data); err != nil {
				n.logger.Error("realtime: publish failed", "channel", channel, "err", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// DrainLogs reads from ch and publishes each entry to LogChannel. ch
// already carries pre-marshaled JSON (see logger.BrowserHandler.Handle), so
// this forwards the bytes as-is instead of going through Drain/json.Marshal
// (which would base64-encode them). Runs until ctx is cancelled or ch is
// closed; intended to be called in a goroutine.
func (n *Node) DrainLogs(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			if err := n.Publish(LogChannel, data); err != nil {
				n.logger.Error("realtime: publish failed", "channel", LogChannel, "err", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleConn is the per-connection entry point handed to websocket.New. The
// write pump runs in its own goroutine; this goroutine runs the read loop,
// which also serves as the liveness check via the read deadline + pong
// handler. Whichever side notices the connection is dead calls client.close,
// which tears down both.
func (n *Node) handleConn(conn *websocket.Conn) {
	c := newClient(conn)
	n.hub.register(c)
	defer n.hub.unregister(c)
	defer c.close()

	go n.writePump(c)

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return // disconnected or errored — defers above clean up
		}

		var cm clientMessage
		if err := json.Unmarshal(msg, &cm); err != nil {
			c.enqueue(mustEncode(serverMessage{Type: "error", Error: "invalid message"}))
			continue
		}

		switch cm.Type {
		case "subscribe":
			c.subscribe(cm.Channel)
			c.enqueue(mustEncode(serverMessage{Type: "subscribed", Channel: cm.Channel}))
		case "unsubscribe":
			c.unsubscribe(cm.Channel)
			c.enqueue(mustEncode(serverMessage{Type: "unsubscribed", Channel: cm.Channel}))
		default:
			c.enqueue(mustEncode(serverMessage{Type: "error", Error: "unknown message type: " + cm.Type}))
		}
	}
}

// writePump is the only goroutine allowed to call conn.WriteMessage for this
// client — fasthttp/websocket, like gorilla/websocket, forbids concurrent
// writers on the same connection, so every outbound message (broadcasts and
// pings alike) funnels through here.
func (n *Node) writePump(c *client) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case data := <-c.send:
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		case <-c.done:
			return
		}
	}
}
