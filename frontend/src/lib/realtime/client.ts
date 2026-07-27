import { ref, type Ref } from 'vue'
import type { ClientMessage, ServerMessage } from './types'

const WS_PATH = '/ws'
const RECONNECT_DELAY_MS = 1000

type Listener<T> = (data: T) => void

function wsURL(): string {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${location.host}${WS_PATH}`
}

/**
 * Single shared WebSocket connection to the app's realtime hub
 * (internal/realtime, mounted at /ws on the same origin — no separate port).
 * Not exported directly; use `subscribe`/`connected` below.
 */
class RealtimeClient {
  connected: Ref<boolean> = ref(false)

  private ws: WebSocket | null = null
  private connectPromise: Promise<void> | null = null
  private listeners = new Map<string, Set<Listener<unknown>>>()

  private ensureConnected(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) return Promise.resolve()
    if (this.connectPromise) return this.connectPromise

    this.connectPromise = new Promise((resolve) => {
      const socket = new WebSocket(wsURL())

      socket.addEventListener('open', () => {
        this.ws = socket
        this.connected.value = true
        // Re-subscribe to every channel that had listeners before this
        // connect (covers both first connect and reconnect after a drop).
        for (const channel of this.listeners.keys()) {
          this.send({ type: 'subscribe', channel })
        }
        resolve()
      })

      socket.addEventListener('message', (ev) => this.handleMessage(ev.data))

      socket.addEventListener('close', () => {
        this.ws = null
        this.connected.value = false
        this.connectPromise = null
        if (this.listeners.size > 0) {
          setTimeout(() => void this.ensureConnected(), RECONNECT_DELAY_MS)
        }
      })

      socket.addEventListener('error', () => socket.close())
    })

    return this.connectPromise
  }

  private send(msg: ClientMessage) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
    }
  }

  private handleMessage(raw: unknown) {
    if (typeof raw !== 'string') return
    let msg: ServerMessage
    try {
      msg = JSON.parse(raw)
    } catch {
      return
    }
    if (msg.type !== 'publication' || !msg.channel) return

    const set = this.listeners.get(msg.channel)
    if (!set) return
    for (const listener of set) listener(msg.data)
  }

  /**
   * Subscribes to `channel`, connecting the socket on first use. Returns an
   * unsubscribe function — call it (e.g. from onUnmounted) to stop
   * receiving events; the channel is unsubscribed on the wire once its last
   * listener is removed.
   */
  async subscribe<T>(channel: string, listener: Listener<T>): Promise<() => void> {
    let set = this.listeners.get(channel)
    const isNewChannel = !set
    if (!set) {
      set = new Set()
      this.listeners.set(channel, set)
    }
    set.add(listener as Listener<unknown>)

    await this.ensureConnected()
    if (isNewChannel) this.send({ type: 'subscribe', channel })

    return () => {
      set!.delete(listener as Listener<unknown>)
      if (set!.size === 0) {
        this.listeners.delete(channel)
        this.send({ type: 'unsubscribe', channel })
      }
    }
  }
}

export const realtimeClient = new RealtimeClient()
