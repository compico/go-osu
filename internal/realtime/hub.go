package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// Channel names in use by this application. Publish/Drain accept any
// string as a channel — these two are just what's currently published.
const (
	LogChannel      = "logs"
	ProgressChannel = "sync_progress"
)

// sendBuffer is how many outbound messages a single client can have queued
// before broadcast starts dropping messages for it instead of blocking
// delivery to every other client. A slow/backgrounded browser tab shouldn't
// be able to stall the whole feed for everyone else.
const sendBuffer = 128

// pingInterval / pongWait keep idle connections alive through anything that
// silently drops quiet TCP connections (browser backgrounding, proxies),
// and let the server notice and clean up a dead client instead of leaking
// it (and its goroutines) forever.
const (
	pingInterval = 30 * time.Second
	pongWait     = 60 * time.Second
)

// clientMessage is everything the browser can say to us — just
// subscribe/unsubscribe, there's no other client -> server traffic in this
// app.
type clientMessage struct {
	Type    string `json:"type"` // "subscribe" | "unsubscribe"
	Channel string `json:"channel"`
}

// serverMessage is everything we can say to the browser.
type serverMessage struct {
	Type    string          `json:"type"` // "publication" | "subscribed" | "unsubscribed" | "error"
	Channel string          `json:"channel,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func mustEncode(v serverMessage) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		// v only ever contains plain strings and json.RawMessage we produced
		// ourselves — this can't realistically fail, but don't panic over a
		// websocket message if it somehow does.
		return []byte(`{"type":"error","error":"internal encode failure"}`)
	}
	return data
}

// wsConn is the subset of *websocket.Conn (github.com/gofiber/contrib/v3/websocket)
// the hub needs. Kept as an interface purely so hub/client logic can be unit
// tested without a real socket.
type wsConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	SetReadDeadline(t time.Time) error
	SetPongHandler(h func(appData string) error)
	Close() error
}

// client is one connected browser tab.
type client struct {
	conn wsConn
	send chan []byte

	mu       sync.RWMutex
	channels map[string]struct{}

	closeOnce sync.Once
	done      chan struct{}
}

func newClient(conn wsConn) *client {
	return &client{
		conn:     conn,
		send:     make(chan []byte, sendBuffer),
		channels: make(map[string]struct{}),
		done:     make(chan struct{}),
	}
}

func (c *client) subscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.channels[channel]
	return ok
}

func (c *client) subscribe(channel string) {
	c.mu.Lock()
	c.channels[channel] = struct{}{}
	c.mu.Unlock()
}

func (c *client) unsubscribe(channel string) {
	c.mu.Lock()
	delete(c.channels, channel)
	c.mu.Unlock()
}

// enqueue queues data for delivery to this client. Non-blocking: if the
// client's buffer is full, the message is dropped rather than stalling the
// caller — this feed is best-effort, not guaranteed-delivery, same as
// everywhere else progress/logs get published in this app.
func (c *client) enqueue(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// close tears down the connection and signals the write pump to stop.
// Closing conn (rather than just the done channel) is what unblocks a
// ReadMessage call that's currently parked in the read loop — that's the
// only way to make the read loop exit promptly instead of waiting out its
// full pongWait deadline. Safe to call more than once and concurrently.
func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

// hub tracks connected clients and fans out published messages to whichever
// of them are currently subscribed to the target channel.
type hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
	logger  *slog.Logger
}

func newHub(logger *slog.Logger) *hub {
	return &hub{
		clients: make(map[*client]struct{}),
		logger:  logger,
	}
}

func (h *hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *hub) closeAll() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.close()
	}
}

// broadcast sends an already-encoded envelope to every client currently
// subscribed to channel. A channel with no subscribers is simply a no-op —
// same semantics as Node.Publish had with centrifuge before this rewrite.
func (h *hub) broadcast(channel string, encoded []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if !c.subscribed(channel) {
			continue
		}
		if !c.enqueue(encoded) {
			h.logger.Warn("realtime: dropping message for slow client", "channel", channel)
		}
	}
}
