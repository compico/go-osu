import { onMounted, onUnmounted, ref, type Ref } from 'vue'
import { realtimeClient } from '@/lib/realtime'
import type { LogEntry } from '@/lib/realtime'
import { LOG_CHANNEL } from '@/lib/realtime'

export interface UseLogsResult {
  /** Push order (oldest first), capped at maxEntries. */
  logs: Ref<LogEntry[]>
  connected: Ref<boolean>
  clear: () => void
}

/**
 * Subscribes to the "logs" channel for the lifetime of the enclosing
 * component and keeps a capped, reactive buffer of LogEntry values — a
 * stand-in for a browser console since none existed yet.
 */
export function useLogs(maxEntries = 500): UseLogsResult {
  const logs = ref<LogEntry[]>([]) as Ref<LogEntry[]>
  let unsubscribe: (() => void) | null = null

  onMounted(async () => {
    unsubscribe = await realtimeClient.subscribe<LogEntry>(LOG_CHANNEL, (entry) => {
      logs.value.push(entry)
      if (logs.value.length > maxEntries) {
        logs.value.splice(0, logs.value.length - maxEntries)
      }
    })
  })

  onUnmounted(() => {
    unsubscribe?.()
    unsubscribe = null
  })

  function clear() {
    logs.value = []
  }

  return { logs, connected: realtimeClient.connected, clear }
}
