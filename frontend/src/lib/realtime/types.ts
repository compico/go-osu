// Mirrors internal/logger.LogEntry (Go).
export interface LogEntry {
  level: string
  msg: string
  time: string // RFC3339, as encoded by encoding/json for time.Time
  attrs?: Record<string, unknown>
}

// Mirrors internal/service.ProgressStage (Go). Deliberately coarse — see
// the Go doc comment on ProgressEvent for why per-beatmap/per-mod-
// combination detail isn't sent at all (16k beatmaps × 36 mod combinations
// is 576k+ events, which floods the browser tab far faster than a progress
// bar can usefully render).
export type ProgressStage = 'read_db' | 'diff' | 'calc_progress' | 'write' | 'done'

// Mirrors internal/service.ProgressEvent (Go). Only beatmap-level counts —
// no beatmap_id/mods. Also rate-limited server-side to a couple of updates
// a second per stage (see emitThrottled in syncer.go), so don't assume
// every logical unit of work gets its own event.
export interface ProgressEvent {
  stage: ProgressStage
  table?: 'beatmapsets' | 'beatmaps' | 'skill_cache'
  done: number
  total: number
  message?: string
  error?: string
  at: string // RFC3339
}

// Mirrors internal/realtime.serverMessage (Go) — every message the server
// sends over the WebSocket.
export interface ServerMessage {
  type: 'publication' | 'subscribed' | 'unsubscribed' | 'error'
  channel?: string
  data?: unknown
  error?: string
}

// Mirrors internal/realtime.clientMessage (Go) — every message the browser
// can send over the WebSocket.
export interface ClientMessage {
  type: 'subscribe' | 'unsubscribe'
  channel: string
}