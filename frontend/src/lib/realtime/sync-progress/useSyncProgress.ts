import { computed, onMounted, onUnmounted, reactive, ref, type Ref } from 'vue'
import { realtimeClient } from '@/lib/realtime'
import type { ProgressEvent, ProgressStage } from '@/lib/realtime'
import { PROGRESS_CHANNEL } from '@/lib/realtime'

export interface UseSyncProgressResult {
  /** Every event received this session, oldest first. Capped at maxEvents. */
  history: Ref<ProgressEvent[]>
  /** Most recent event of any stage — good for a "what's happening now" line. */
  latest: Ref<ProgressEvent | null>
  /**
   * Most recent event per stage. StageWrite is further split by `table` (as
   * `write:beatmaps`, `write:skill_cache`, etc.) so those don't clobber each
   * other — handy for driving multiple progress bars at once.
   */
  byKey: Record<string, ProgressEvent>
  /** done/total from the latest calc_progress event, 0–100, or null before any arrive. */
  calcPercent: Ref<number | null>
  /** True once the current run's `done` stage event has been received. */
  finished: Ref<boolean>
  connected: Ref<boolean>
  reset: () => void
}

function keyFor(ev: ProgressEvent): string {
  return ev.stage === 'write' && ev.table ? `write:${ev.table}` : ev.stage
}

/**
 * Subscribes to the "sync_progress" channel for the lifetime of the
 * enclosing component and exposes both the raw event stream and a couple of
 * derived conveniences for driving progress-bar UI.
 */
export function useSyncProgress(maxEvents = 2000): UseSyncProgressResult {
  const history = ref<ProgressEvent[]>([]) as Ref<ProgressEvent[]>
  const latest = ref<ProgressEvent | null>(null)
  const byKey = reactive<Record<string, ProgressEvent>>({})
  const finished = ref(false)

  let unsubscribe: (() => void) | null = null

  onMounted(async () => {
    unsubscribe = await realtimeClient.subscribe<ProgressEvent>(PROGRESS_CHANNEL, (ev) => {
      latest.value = ev
      byKey[keyFor(ev)] = ev

      history.value.push(ev)
      if (history.value.length > maxEvents) {
        history.value.splice(0, history.value.length - maxEvents)
      }

      if (ev.stage === 'done') {
        finished.value = true
      } else if (ev.stage === 'read_db') {
        // a new run started — clear the "finished" flag from any previous one
        finished.value = false
      }
    })
  })

  onUnmounted(() => {
    unsubscribe?.()
    unsubscribe = null
  })

  const calcPercent = computed<number | null>(() => {
    const ev = byKey['calc_progress' as ProgressStage]
    if (!ev || ev.total <= 0) return null
    return Math.round((ev.done / ev.total) * 100)
  })

  function reset() {
    history.value = []
    latest.value = null
    finished.value = false
    for (const k of Object.keys(byKey)) delete byKey[k]
  }

  return { history, latest, byKey, calcPercent, finished, connected: realtimeClient.connected, reset }
}
