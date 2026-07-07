import { ref, shallowRef } from 'vue'
import Fuse from 'fuse.js'
import type { Track } from '@/types/player'

export interface EnrichedDiff {
    difficulty_id: number
    difficulty: string
    approach_rate: number
    overall_difficulty: number
    circle_size: number
    hp_drain: number
    drain_time: number
    audio_file_name: string
    creator_name: string
    _stars: number | null
    _bpm: number
}

export interface EnrichedGroup {
    id: number
    beatmap_id: number
    song_name: string
    artist_name: string
    diffs: EnrichedDiff[]
    _starsMin: number | null
    _starsMax: number | null
    _bpm: number
    _ar: number
    _od: number
    _cs: number
    _hp: number
    _drain: number
    _creator: string
    _diffNames: string
}

// singleton-состояние — грузим один раз на всё приложение
const groups = shallowRef<EnrichedGroup[]>([])
const loading = ref(true)
let fuse: Fuse<EnrichedGroup> | null = null
let loadPromise: Promise<void> | null = null

function enrich(raw: any[]): EnrichedGroup[] {
    return raw.map((song) => {
        const diffs: EnrichedDiff[] = (song.Beatmaps ?? []).map((bm: any) => ({
            ...bm,
            _stars: bm.osu_mode_stars?.find((s: any) => s.int === 0)?.float ?? null,
            _bpm: bm.timing_points?.[0]
                ? Math.round(60000 / bm.timing_points[0].beat_length)
                : 0,
        }))
        diffs.sort((a, b) => (a._stars ?? 0) - (b._stars ?? 0))

        const stars = diffs.map((d) => d._stars).filter((s): s is number => s != null)

        return {
            id: song.id,
            beatmap_id: song.beatmap_id,
            song_name: song.song_name,
            artist_name: song.artist_name,
            diffs,
            _starsMin: stars.length ? Math.min(...stars) : null,
            _starsMax: stars.length ? Math.max(...stars) : null,
            _bpm: diffs[0]?._bpm ?? 0,
            _ar: Math.max(...diffs.map((d) => d.approach_rate ?? 0)),
            _od: Math.max(...diffs.map((d) => d.overall_difficulty ?? 0)),
            _cs: diffs[0]?.circle_size ?? 0,
            _hp: diffs[0]?.hp_drain ?? 0,
            _drain: Math.max(...diffs.map((d) => d.drain_time ?? 0)),
            _creator: diffs[0]?.creator_name ?? '',
            _diffNames: diffs.map((d) => d.difficulty).join(' '),
        }
    })
}

async function ensureLoaded(): Promise<void> {
    if (loadPromise) return loadPromise
    loadPromise = (async () => {
        loading.value = true
        try {
            const res = await fetch('/api/osu/songs')
            const data = await res.json()
            const enriched = enrich(data)
            groups.value = enriched
            fuse = new Fuse(enriched, {
                keys: [
                    { name: 'song_name', weight: 0.5 },
                    { name: 'artist_name', weight: 0.35 },
                    { name: '_diffNames', weight: 0.1 },
                    { name: '_creator', weight: 0.05 },
                ],
                threshold: 0.35,
                shouldSort: true,
                minMatchCharLength: 2,
            })
        } catch (e) {
            console.error('Failed to load osu songs catalog', e)
        } finally {
            loading.value = false
        }
    })()
    return loadPromise
}

const FILTER_RE = /(\w+)\s*(>=|<=|>|<|=)\s*([\d.]+)/g

function parseFilters(query: string) {
    const filters: { field: string; op: string; val: number }[] = []
    const text = query
        .replace(FILTER_RE, (_, field, op, val) => {
            filters.push({ field: field.toLowerCase(), op, val: parseFloat(val) })
            return ' '
        })
        .trim()
    return { filters, text }
}

function compare(v: number | null, op: string, val: number): boolean {
    if (v == null) return false
    if (op === '>') return v > val
    if (op === '<') return v < val
    if (op === '>=') return v >= val
    if (op === '<=') return v <= val
    if (op === '=') return v === val
    return false
}

function matchFilter(g: EnrichedGroup, f: { field: string; op: string; val: number }) {
    if (f.field === 'stars' || f.field === 'sr') {
        return g.diffs.some((d) => compare(d._stars, f.op, f.val))
    }
    const v =
        f.field === 'ar' ? g._ar
            : f.field === 'od' ? g._od
                : f.field === 'cs' ? g._cs
                    : f.field === 'hp' ? g._hp
                        : f.field === 'bpm' ? g._bpm
                            : f.field === 'length' || f.field === 'drain' ? g._drain
                                : null
    return v != null && compare(v, f.op, f.val)
}

export function useOsuCatalog() {
    ensureLoaded()

    function search(query: string): EnrichedGroup[] {
        const { filters, text } = parseFilters(query)
        let result = groups.value

        if (filters.length) {
            result = result.filter((g) => filters.every((f) => matchFilter(g, f)))
        }

        if (text.length >= 2 && fuse) {
            if (filters.length === 0) {
                result = fuse.search(text).map((r) => r.item)
            } else {
                const sub = new Fuse(result, {
                    keys: [
                        { name: 'song_name', weight: 0.5 },
                        { name: 'artist_name', weight: 0.35 },
                        { name: '_diffNames', weight: 0.1 },
                        { name: '_creator', weight: 0.05 },
                    ],
                    threshold: 0.35,
                    minMatchCharLength: 2,
                })
                result = sub.search(text).map((r) => r.item)
            }
        }

        return result
    }

    return { groups, loading, search }
}

/** Диф -> Track для плеера */
export function diffToTrack(group: EnrichedGroup, diff: EnrichedDiff): Track {
    return {
        id: String(diff.difficulty_id),
        title: group.song_name,
        artist: group.artist_name,
        album: diff.difficulty,
        duration: diff.drain_time,
        url: `/api/osu/songs/${diff.difficulty_id}/track`,
        coverUrl: `/api/osu/bg/${group.beatmap_id}.jpg`,
        groupKey: `${group.beatmap_id}:${diff.audio_file_name}`,
    }
}

/** Вся группа диффов -> Track[] (для формирования очереди next/prev) */
export function groupToTracks(group: EnrichedGroup): Track[] {
    return group.diffs.map((d) => diffToTrack(group, d))
}
