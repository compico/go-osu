import { ref, shallowRef } from 'vue'
import { PROGRESS_CHANNEL, realtimeClient } from '@/lib/realtime'
import type { ProgressEvent } from '@/lib/realtime'
import type { Track } from '@/types/player'

export interface SkillsDTO {
    stamina: number
    tenacity: number
    agility: number
    precision: number
    reading: number
    memory: number
    accuracy: number
    reaction: number
}

export interface DiffDTO {
    beatmap_id: number
    difficulty: string
    stars: number
    approach_rate: number
    drain_time: number
    bpm: number
    audio_file_name: string
    creator_name: string
    skills?: SkillsDTO
}

export interface GroupDTO {
    beatmap_set_id: number
    song_title: string
    artist_name: string
    diffs: DiffDTO[]
    stars_min: number
    stars_max: number
}

const groups = shallowRef<GroupDTO[]>([])
const loading = ref(false)
const hasMore = ref(true)

let currentQuery = ''
let sortBy = 'artist_name'
let nextCursor = ''
let requestSeq = 0
let inFlight = false

function buildParams(cursor: string, limit: number): URLSearchParams {
    return new URLSearchParams({
        q: currentQuery,
        sort: sortBy,
        limit: String(limit),
        ...(cursor ? { cursor } : {}),
    })
}

async function loadMore(limit = 50): Promise<void> {
    // inFlight блокирует только повторный вызов для ТЕКУЩЕГО поиска
    // (например, двойной скролл-триггер) — но не блокирует новый search(),
    // тот всегда стартует новый запрос независимо от состояния предыдущего.
    if (inFlight || !hasMore.value) return

    const seq = ++requestSeq
    inFlight = true
    loading.value = true

    try {
        const res = await fetch(`/api/osu/songs/search?${buildParams(nextCursor, limit)}`)
        const page = await res.json()
        if (seq !== requestSeq) return // этот ответ устарел — новый search() уже сбросил состояние

        groups.value = [...groups.value, ...page.items]
        nextCursor = page.next_cursor ?? ''
        hasMore.value = Boolean(page.next_cursor)
    } catch (e) {
        console.error('Failed to load osu songs page', e)
        if (seq === requestSeq) hasMore.value = false
    } finally {
        if (seq === requestSeq) {
            inFlight = false
            loading.value = false
        }
        // если seq !== requestSeq — этот запрос устарел, его inFlight/loading
        // уже обнулены в reset() ниже, трогать нечего
    }
}


function reset(): void {
    requestSeq++
    inFlight = false
    loading.value = false
    groups.value = []
    nextCursor = ''
    hasMore.value = true
}

/** Called on search-box input or sort change. Mods are just part of the
 *  query text (e.g. "mode=HDDT") — the DSL parser on the backend already
 *  extracts them, no separate field needed. */
function search(query: string, opts?: { sort?: string }): void {
    currentQuery = query
    if (opts?.sort !== undefined) sortBy = opts.sort
    reset()
    loadMore()
}

/** The exact query text backing the current result set — pass this to
 *  playerStore.playFromBrowse() so next/prev use the same predicate. */
function getEffectiveQuery(): string {
    return currentQuery
}

function getSort(): string {
    return sortBy
}

let watchingSync = false

/** The catalog is loaded once on first use, so without this it keeps showing
 *  whatever the database held before the library sync started (nothing at
 *  all on the first run). Re-run the current search — same query and sort —
 *  when the sync reports `done`. The subscription lives as long as the page,
 *  like the catalog state itself. */
function reloadWhenSyncFinishes(): void {
    if (watchingSync) return
    watchingSync = true

    void realtimeClient.subscribe<ProgressEvent>(PROGRESS_CHANNEL, (ev) => {
        if (ev.stage === 'done') search(currentQuery)
    })
}

export function useOsuCatalog() {
    reloadWhenSyncFinishes()

    if (groups.value.length === 0 && !loading.value && hasMore.value) {
        loadMore()
    }

    return { groups, loading, hasMore, search, loadMore, getEffectiveQuery, getSort }
}

export function diffToTrack(group: GroupDTO, diff: DiffDTO): Track {
    return {
        id: String(diff.beatmap_id),
        title: group.song_title,
        artist: group.artist_name,
        album: diff.difficulty,
        duration: diff.drain_time,
        url: `/api/osu/songs/${diff.beatmap_id}/track`,
        coverUrl: `/api/osu/bg/${group.beatmap_set_id}.jpg`,
        groupKey: `${group.beatmap_set_id}:${diff.audio_file_name}`,
        groupSetId: group.beatmap_set_id,
    }
}