import { ref, watch, type Ref } from 'vue'
import type { StrainsResponse } from '@/types/strains'

const cache = new Map<string, StrainsResponse>()
const CACHE_LIMIT = 50

/** Mods label as shown in the card ("NoMod" or "HDDT") -> the API's mods parameter. */
function modsParam(label: string): string {
  return label === 'NoMod' ? '' : label
}

/**
 * Loads the strain curves of a beatmap for the given mods whenever either
 * changes. A response is cached per beatmap and mods; a request that is
 * superseded by a newer one is aborted, so a slow answer for the previous
 * selection never replaces the current one.
 */
export function useStrains(beatmapId: Ref<number | null | undefined>, modsLabel: Ref<string>) {
  const data = ref<StrainsResponse | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  let controller: AbortController | null = null

  async function load() {
    controller?.abort()
    controller = null

    const id = beatmapId.value
    if (!id) {
      data.value = null
      loading.value = false
      error.value = null
      return
    }

    const mods = modsParam(modsLabel.value)
    const key = `${id}:${mods}`
    const cached = cache.get(key)
    if (cached) {
      data.value = cached
      loading.value = false
      error.value = null
      return
    }

    const mine = new AbortController()
    controller = mine
    data.value = null
    loading.value = true
    error.value = null

    try {
      const params = new URLSearchParams()
      if (mods) params.set('mods', mods)
      const res = await fetch(`/api/osu/songs/${id}/strains?${params}`, { signal: mine.signal })
      if (!res.ok) {
        const body = await res.json().catch(() => null)
        throw new Error(body?.error ?? `HTTP ${res.status}`)
      }
      const parsed = (await res.json()) as StrainsResponse

      if (cache.size >= CACHE_LIMIT) cache.delete(cache.keys().next().value as string)
      cache.set(key, parsed)
      data.value = parsed
    } catch (e) {
      if (mine.signal.aborted) return
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (controller === mine) {
        loading.value = false
        controller = null
      }
    }
  }

  watch([beatmapId, modsLabel], load, { immediate: true })

  return { data, loading, error }
}
