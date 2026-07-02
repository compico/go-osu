<template>
  <div class="osu-app">
    <!-- Hidden audio element -->
    <audio ref="audioRef" preload="metadata"></audio>

    <!-- ── Left: Player ────────────────────────────────────────────────── -->
    <div class="player-area">
      <Transition name="fade-player">
        <div v-if="selectedDiff" class="now-playing" :key="selectedDiff.difficulty_id">
          <div class="bg-blur" :style="bgStyle"></div>
          <div class="bg-cover" :style="bgStyle"></div>
          <div class="player-content">
            <div class="album-art">
              <img
                  v-if="!bgError"
                  :src="`/api/osu/bg/${selectedGroup?.beatmap_id}.jpg`"
                  class="cover-img"
                  @error="bgError = true"
              />
              <div v-else class="cover-placeholder">
                <i class="bi bi-music-note-beamed"></i>
              </div>
            </div>
            <div class="track-meta">
              <div class="track-title">{{ selectedGroup?.song_name }}</div>
              <div class="track-artist">{{ selectedGroup?.artist_name }}</div>
              <div class="track-diff">
                <span class="badge-diff">{{ selectedDiff.difficulty }}</span>
                <span class="stars" v-if="mainStars">
                  <i class="bi bi-star-fill"></i> {{ mainStars.toFixed(2) }}★
                </span>
              </div>
              <div class="track-stats">
                <span><i class="bi bi-clock"></i> {{ formatTime(selectedDiff.drain_time) }}</span>
                <span><i class="bi bi-circle"></i> {{ selectedDiff.number_of_hitcircles }}</span>
                <span>AR {{ selectedDiff.approach_rate }}</span>
                <span>OD {{ selectedDiff.overall_difficulty }}</span>
                <span>BPM {{ selectedBpm }}</span>
              </div>
            </div>
            <div class="player-controls">
              <button class="ctrl-btn" @click="playPrev"><i class="bi bi-skip-start-fill"></i></button>
              <button class="ctrl-btn play-btn" @click="togglePlay">
                <i :class="playing ? 'bi bi-pause-fill' : 'bi bi-play-fill'"></i>
              </button>
              <button class="ctrl-btn" @click="playNext"><i class="bi bi-skip-end-fill"></i></button>

              <!-- Volume Control -->
              <div class="volume-control">
                <button class="ctrl-btn volume-btn" @click="toggleMute">
                  <i :class="volumeIcon"></i>
                </button>
                <input
                    type="range"
                    class="volume-slider"
                    min="0"
                    max="1"
                    step="0.01"
                    :value="isMuted ? 0 : volume"
                    @input="onVolumeChange"
                />
              </div>
            </div>
          </div>
        </div>
        <div v-else class="no-selection" key="empty">
          <div class="no-sel-inner">
            <i class="bi bi-music-note-list"></i>
            <p>Select a song to play</p>
          </div>
        </div>
      </Transition>
    </div>

    <!-- ── Right: Sidebar ────────────────────────────────────────────────── -->
    <div class="sidebar">
      <div class="search-wrap">
        <div class="search-box">
          <i class="bi bi-search search-icon"></i>
          <input
              v-model="searchQuery"
              type="text"
              class="search-input"
              placeholder="artist, title… stars>6, ar>9, bpm>200"
              @input="debouncedSearch"
          />
          <button v-if="searchQuery" class="clear-btn" @click="clearSearch">
            <i class="bi bi-x"></i>
          </button>
        </div>
        <div class="result-count">
          <span v-if="loading">loading…</span>
          <span v-else>{{ filteredGroups.length }} maps</span>
        </div>
      </div>

      <RecycleScroller
          ref="scrollerEl"
          class="song-list"
          :items="flatList"
          :item-size="null"
          size-field="_size"
          key-field="_key"
      >
        <template #default="{ item, index, active }">
          <div :data-index="index">
            <!-- ── Group row ── -->
            <div
                v-if="item.type === 'group'"
                class="song-item"
                :class="{
                expanded: expandedGroups.has(item.beatmap_id),
                'group-active': selectedGroup?.beatmap_id === item.beatmap_id
              }"
                @click="toggleGroup(item)"
            >
              <div class="thumb">
                <img
                    v-if="!thumbErrors[item.beatmap_id]"
                    :src="`/api/osu/bg/${item.beatmap_id}.jpg`"
                    class="thumb-img"
                    loading="lazy"
                    @error="thumbErrors[item.beatmap_id] = true"
                />
                <div v-else class="thumb-placeholder"><i class="bi bi-music-note"></i></div>
              </div>
              <div class="song-info">
                <div class="song-name">{{ item.song_name }}</div>
                <div class="song-artist">{{ item.artist_name }}</div>
                <div class="song-meta">
                  <span class="diff-count">
                    <i class="bi bi-layers"></i> {{ item.diffs.length }}
                  </span>
                  <span v-if="item._starsMin != null" class="song-stars">
                    <i class="bi bi-star-fill"></i>
                    {{ item._starsMin.toFixed(1) }}
                    <span v-if="item._starsMax !== item._starsMin">– {{ item._starsMax.toFixed(1) }}</span>
                  </span>
                </div>
              </div>
              <div class="expand-icon" :class="{ open: expandedGroups.has(item.beatmap_id) }">
                <i class="bi bi-chevron-right"></i>
              </div>
              <div class="play-indicator" v-if="selectedGroup?.beatmap_id === item.beatmap_id && playing">
                <i class="bi bi-play-fill"></i>
              </div>
            </div>

            <!-- ── Diff row ── -->
            <div
                v-else-if="item.type === 'diff'"
                class="diff-item"
                :class="{ active: selectedDiff?.difficulty_id === item.difficulty_id }"
                @click="selectDiff(item._group, item)"
            >
              <div class="diff-indent"></div>
              <div class="diff-star-bar" :style="{ background: starColor(item._stars) }"></div>
              <div class="diff-info">
                <span class="diff-name">{{ item.difficulty }}</span>
                <span class="diff-stats">
                  <i class="bi bi-star-fill"></i> {{ item._stars?.toFixed(2) ?? '?' }}
                  · AR {{ item.approach_rate }}
                  · {{ formatTime(item.drain_time) }}
                </span>
              </div>
              <div class="play-indicator small" v-if="selectedDiff?.difficulty_id === item.difficulty_id">
                <i :class="playing ? 'bi bi-pause-fill' : 'bi bi-play-fill'"></i>
              </div>
            </div>
          </div>
        </template>
      </RecycleScroller>

      <div v-if="!loading && filteredGroups.length === 0" class="empty-state">
        <i class="bi bi-emoji-frown"></i>
        <p>No maps found</p>
        <small v-if="searchQuery">Try: <code>stars&gt;5</code> · <code>ar&gt;9</code> · <code>bpm&gt;180</code></small>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import Fuse from 'fuse.js'
import { useThemeStore } from '@/stores/theme'
import { usePlayerStore } from '@/composables/usePlayer.ts'
import { useMediaSession } from '@/composables/useMediaSession'

const themeStore = useThemeStore()

// ── State ────────────────────────────────────────────────────────────────────
const groups         = ref([])
const filteredGroups = ref([])
const expandedGroups = ref(new Set())

const selectedGroup = ref(null)
const selectedDiff  = ref(null)

const playing     = ref(false)
const searchQuery = ref('')
const loading     = ref(true)

const scrollerEl = ref(null)
const bgError    = ref(false)
const thumbErrors = reactive({})

const audioRef = ref(null)
const store = usePlayerStore()
const isRestored = ref(false)

let fuse          = null
let debounceTimer = null

const volume = computed(() => store.volume)
const isMuted = computed(() => store.isMuted)

const volumeIcon = computed(() => {
  if (isMuted.value || volume.value === 0) return 'bi bi-volume-mute-fill'
  if (volume.value < 0.3) return 'bi bi-volume-off-fill'
  if (volume.value < 0.7) return 'bi bi-volume-down-fill'
  return 'bi bi-volume-up-fill'
})

function toggleMute() {
  store.toggleMute()
}

function onVolumeChange(event) {
  const newVolume = parseFloat(event.target.value)
  store.setVolume(newVolume)

  if (isMuted.value && newVolume > 0) {
    store.toggleMute()
  }
}

// ── MediaSession Integration ─────────────────────────────────────────────────
useMediaSession(
    audioRef,
    playing,
    selectedDiff,
    selectedGroup,
    playNext,
    playPrev,
    togglePlay
)

// ── Audio Control ────────────────────────────────────────────────────────────
watch(selectedDiff, async (diff) => {
  if (diff && audioRef.value) {
    const url = `/api/osu/songs/${diff.difficulty_id}/track`
    console.log('[Audio] Loading track:', url)

    audioRef.value.src = url
    audioRef.value.load()

    audioRef.value.onerror = (e) => {
      console.error('[Audio] Error loading track:', e)
    }

    try {
      await audioRef.value.play()
      playing.value = true
    } catch (e) {
      console.error('[Audio] Failed to play:', e)
      playing.value = false
    }
  }
})

watch(playing, (isPlaying) => {
  if (!audioRef.value) return

  if (isPlaying) {
    audioRef.value.play().catch(e => {
      console.error('Play failed:', e)
      playing.value = false
    })
  } else {
    audioRef.value.pause()
  }
})

watch([volume, isMuted], ([vol, muted]) => {
  if (audioRef.value) {
    audioRef.value.volume = muted ? 0 : vol
    audioRef.value.muted = muted
  }
}, { immediate: true })

// ── flatList for DynamicScroller ─────────────────────────────────────────────
const flatList = computed(() => {
  const list = []
  for (const g of filteredGroups.value) {
    list.push({ ...g, type: 'group', _key: `g-${g.beatmap_id}`, _size: 68 })
    if (expandedGroups.value.has(g.beatmap_id)) {
      for (const d of g.diffs) {
        list.push({ ...d, type: 'diff', _key: `d-${d.difficulty_id}`, _group: g, _size: 44 })
      }
    }
  }
  return list
})

// ── Auto-scroll: центрирует элемент в списке ─────────────────────────────────
// Считаем offsetTop через накопленные высоты (group=68, diff=44).
// scrollHeight у DynamicScroller корректный т.к. он виртуально резервирует место.
// Один scrollTo smooth — без рывков.
function scrollToSelected(key) {
  const list = flatList.value
  const idx  = list.findIndex(item => item._key === key)
  if (idx === -1 || !scrollerEl.value) return

  const scroller = scrollerEl.value.$el
  if (!scroller || scroller.clientHeight === 0) return

  // Накапливаем высоты — DynamicScroller использует те же значения что и min-item-size
  let offsetTop = 0
  for (let i = 0; i < idx; i++) {
    offsetTop += list[i].type === 'group' ? 68 : 44
  }
  const itemH  = list[idx].type === 'group' ? 68 : 44
  const target = offsetTop - (scroller.clientHeight / 2) + (itemH / 2)

  scroller.scrollTo({ top: Math.max(0, target), behavior: 'smooth' })
}

// ── Load & enrich ─────────────────────────────────────────────────────────────
onMounted(() => { themeStore.apply() })
watch(() => themeStore.theme, () => themeStore.apply())
onMounted(() => {
  if (audioRef.value) {
    audioRef.value.volume = store.volume
    audioRef.value.muted = store.isMuted
  }
})
watch(audioRef, async (audio) => {
  if (!audio || isRestored.value) return

  // Если есть сохраненный трек - загружаем его
  if (store.currentTrack) {
    console.log('[Audio] Restoring track:', store.currentTrack.title)
    audio.src = store.currentTrack.url
    audio.load()

    // Ждем загрузки метаданных чтобы установить currentTime
    audio.addEventListener('loadedmetadata', () => {
      if (store.currentTime > 0 && store.currentTime < audio.duration) {
        audio.currentTime = store.currentTime
      }
      isRestored.value = true
    }, { once: true })
  }
}, { immediate: true })

watch([() => store.volume, () => store.isMuted], ([vol, muted]) => {
  if (audioRef.value) {
    audioRef.value.volume = muted ? 0 : vol
    audioRef.value.muted = muted
  }
}, { immediate: true })


onMounted(async () => {
  try {
    const res  = await fetch('/api/osu/songs')
    const data = await res.json()

    const enriched = data.map(song => {
      const diffs = (song.Beatmaps ?? []).map(bm => ({
        ...bm,
        _stars: bm.osu_mode_stars?.find(s => s.int === 0)?.float ?? null,
        _bpm:   bm.timing_points?.[0]
            ? Math.round(60000 / bm.timing_points[0].beat_length)
            : 0,
      }))

      diffs.sort((a, b) => (a._stars ?? 0) - (b._stars ?? 0))

      const stars = diffs.map(d => d._stars).filter(s => s != null)

      return {
        id:          song.id,
        beatmap_id:  song.beatmap_id,
        song_name:   song.song_name,
        artist_name: song.artist_name,
        diffs,
        _starsMin:   stars.length ? Math.min(...stars) : null,
        _starsMax:   stars.length ? Math.max(...stars) : null,
        _bpm:        diffs[0]?._bpm ?? 0,
        _ar:         Math.max(...diffs.map(d => d.approach_rate ?? 0)),
        _od:         Math.max(...diffs.map(d => d.overall_difficulty ?? 0)),
        _cs:         diffs[0]?.circle_size ?? 0,
        _hp:         diffs[0]?.hp_drain ?? 0,
        _drain:      Math.max(...diffs.map(d => d.drain_time ?? 0)),
        _creator:    diffs[0]?.creator_name ?? '',
        _diffNames:  diffs.map(d => d.difficulty).join(' '),
      }
    })

    groups.value         = enriched
    filteredGroups.value = enriched

    fuse = new Fuse(enriched, {
      keys: [
        { name: 'song_name',   weight: 0.5  },
        { name: 'artist_name', weight: 0.35 },
        { name: '_diffNames',  weight: 0.1  },
        { name: '_creator',    weight: 0.05 },
      ],
      threshold: 0.35,
      shouldSort: true,
      minMatchCharLength: 2,
    })
  } catch (e) {
    console.error('Failed to load songs', e)
  } finally {
    loading.value = false
  }
})

// ── Group expand/collapse ─────────────────────────────────────────────────────
function toggleGroup(group) {
  const wasOpen = expandedGroups.value.has(group.beatmap_id)

  const s = new Set()
  if (!wasOpen) {
    s.add(group.beatmap_id)
    if (selectedGroup.value?.beatmap_id !== group.beatmap_id) {
      selectDiff(group, group.diffs[0], { skipScroll: true })
    }
  }
  expandedGroups.value = s

  const scrollKey = !wasOpen
      ? `d-${group.diffs[0]?.difficulty_id}`
      : `g-${group.beatmap_id}`
  scrollToSelected(scrollKey)
}

// ── Diff selection ────────────────────────────────────────────────────────────
function selectDiff(group, diff, { skipScroll = false } = {}) {
  if (!diff) return
  if (selectedDiff.value?.difficulty_id === diff.difficulty_id) {
    togglePlay()
    return
  }
  selectedGroup.value = group
  selectedDiff.value  = diff

  expandedGroups.value = new Set([group.beatmap_id])

  if (!skipScroll) {
    scrollToSelected(`d-${diff.difficulty_id}`)
  }
}

function togglePlay() {
  playing.value = !playing.value
}

// ── Плоский список дифов для prev/next ───────────────────────────────────────
const allVisibleDiffs = computed(() => {
  const out = []
  for (const g of filteredGroups.value) {
    for (const d of g.diffs) out.push({ group: g, diff: d })
  }
  return out
})

// ── Логика Next / Prev ────────────────────────────────────────────────────────
// Правило: пропускаем соседние дифы пока у них тот же beatmap_id И тот же
// audio_file_name что у текущего. Останавливаемся когда хотя бы одно меняется.

function findNextDiff(direction) {
  const list = allVisibleDiffs.value
  const cur  = selectedDiff.value
  if (!cur || !list.length) return list[0] ?? null

  const curIdx   = list.findIndex(x => x.diff.difficulty_id === cur.difficulty_id)
  if (curIdx === -1) return null

  const curGroupId = list[curIdx].group.beatmap_id
  const curAudio   = cur.audio_file_name

  const step = direction === 'next' ? 1 : -1
  for (let i = curIdx + step; i >= 0 && i < list.length; i += step) {
    const c = list[i]
    const sameGroup = c.group.beatmap_id === curGroupId
    const sameAudio = c.diff.audio_file_name === curAudio
    if (!sameGroup || !sameAudio) return c
  }
  return null
}

function playNext() {
  const target = findNextDiff('next')
  if (target) selectDiff(target.group, target.diff)
}

function playPrev() {
  const target = findNextDiff('prev')
  if (target) selectDiff(target.group, target.diff)
}

// ── Search ────────────────────────────────────────────────────────────────────
const FILTER_RE = /(\w+)\s*(>=|<=|>|<|=)\s*([\d.]+)/g

function parseFilters(query) {
  const filters = []
  const text = query.replace(FILTER_RE, (_, field, op, val) => {
    filters.push({ field: field.toLowerCase(), op, val: parseFloat(val) })
    return ' '
  }).trim()
  return { filters, text }
}

function matchFilter(g, { field, op, val }) {
  if (field === 'stars' || field === 'sr') {
    return g.diffs.some(d => compare(d._stars, op, val))
  }
  const v =
      field === 'ar'                          ? g._ar    :
          field === 'od'                          ? g._od    :
              field === 'cs'                          ? g._cs    :
                  field === 'hp'                          ? g._hp    :
                      field === 'bpm'                         ? g._bpm   :
                          field === 'length' || field === 'drain' ? g._drain : null
  return v != null && compare(v, op, val)
}

function compare(v, op, val) {
  if (v == null) return false
  if (op === '>')  return v >  val
  if (op === '<')  return v <  val
  if (op === '>=') return v >= val
  if (op === '<=') return v <= val
  if (op === '=')  return v === val
  return false
}

function runSearch() {
  const { filters, text } = parseFilters(searchQuery.value)
  let result = groups.value

  if (filters.length) {
    result = result.filter(g => filters.every(f => matchFilter(g, f)))
  }

  if (text.length >= 2) {
    if (filters.length === 0) {
      result = fuse.search(text).map(r => r.item)
    } else {
      const sub = new Fuse(result, {
        keys: [
          { name: 'song_name',   weight: 0.5  },
          { name: 'artist_name', weight: 0.35 },
          { name: '_diffNames',  weight: 0.1  },
          { name: '_creator',    weight: 0.05 },
        ],
        threshold: 0.35,
        minMatchCharLength: 2,
      })
      result = sub.search(text).map(r => r.item)
    }
  }

  filteredGroups.value = result
}

function debouncedSearch() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(runSearch, 150)
}

function clearSearch() {
  searchQuery.value    = ''
  filteredGroups.value = groups.value
  clearTimeout(debounceTimer)
}

// ── Computed ──────────────────────────────────────────────────────────────────
const mainStars   = computed(() => selectedDiff.value?._stars)
const selectedBpm = computed(() => selectedDiff.value?._bpm || '?')
const bgStyle = computed(() =>
    selectedGroup.value && !bgError.value
        ? { backgroundImage: `url(/api/osu/bg/${selectedGroup.value.beatmap_id}.jpg)` }
        : {}
)

// Сбрасываем ошибку при смене песни
watch(() => selectedGroup.value?.beatmap_id, () => { bgError.value = false })

// ── Helpers ───────────────────────────────────────────────────────────────────
function formatTime(seconds) {
  if (!seconds) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

function starColor(stars) {
  if (!stars) return 'var(--bs-border-color)'
  if (stars < 2)  return 'var(--osu-star-1)'
  if (stars < 3)  return 'var(--osu-star-2)'
  if (stars < 4)  return 'var(--osu-star-3)'
  if (stars < 5)  return 'var(--osu-star-4)'
  if (stars < 6)  return 'var(--osu-star-5)'
  if (stars < 7)  return 'var(--osu-star-6)'
  return 'var(--osu-star-7)'
}
</script>

<style>
.song-list .vue-recycle-scroller__item-wrapper { will-change: transform; }
</style>

<style scoped>
/* ── Base ────────────────────────────────────────────────────────────────── */
.osu-app {
  display: flex;
  flex: 1 1 0;     /* растягиваемся в flex-родителе (.main-content > *) */
  height: 100%;
  min-height: 0;
  background: var(--player-bg);
  color: var(--bs-body-color);
  font-family: -apple-system, 'Segoe UI', sans-serif;
  overflow: hidden;
  transition: background 0.25s ease, color 0.25s ease;
}

/* ── Player ──────────────────────────────────────────────────────────────── */
.player-area {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.now-playing {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.bg-blur {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  filter: blur(40px) brightness(0.25) saturate(1.6);
  transform: scale(1.12);
  transition: background-image 0.5s ease;
}

.bg-cover { display: none; }

.player-content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 22px;
  padding: 40px;
}

.album-art {
  width: 160px;
  height: 120px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 6px 30px var(--osu-pink-glow);
  border: 2px solid var(--osu-pink-border);
}

.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;    /* обрезаем под квадрат album-art */
  object-position: center;
  display: block;
  image-rendering: auto;
  /* Апскейл маленьких (160x120) картинок — сглаживание вместо пикселей */
  filter: blur(0px);
}

.cover-placeholder {
  width: 100%;
  height: 100%;
  background: var(--player-cover-ph-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 60px;
  color: var(--osu-pink-glow);
}

.track-meta { text-align: center; }

.track-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--bs-heading-color);
  text-shadow: 0 0 24px var(--osu-pink-glow);
  margin-bottom: 4px;
}

.track-artist { font-size: 13px; color: var(--bs-secondary-color); margin-bottom: 10px; }

.track-diff {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-bottom: 10px;
}

.badge-diff {
  background: var(--osu-pink-subtle);
  border: 1px solid var(--osu-pink-border);
  color: var(--osu-pink);
  padding: 2px 10px;
  border-radius: 20px;
  font-size: 11px;
}

.stars { color: #ffd700; font-size: 13px; }

.track-stats {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  justify-content: center;
  font-size: 11px;
  color: var(--bs-tertiary-color);
}

.track-stats span { display: flex; align-items: center; gap: 4px; }
.player-controls { display: flex; align-items: center; gap: 14px; }

.ctrl-btn {
  background: var(--bs-border-color-translucent);
  border: 1px solid var(--osu-pink-subtle);
  color: var(--bs-secondary-color);
  border-radius: 50%;
  width: 44px; height: 44px;
  display: flex; align-items: center; justify-content: center;
  font-size: 18px;
  cursor: pointer;
  transition: background 0.18s, border-color 0.18s, color 0.18s;
}

.ctrl-btn:hover {
  background: var(--osu-pink-subtle);
  border-color: var(--osu-pink);
  color: var(--bs-heading-color);
}

.play-btn {
  width: 58px; height: 58px; font-size: 24px;
  background: var(--osu-pink-subtle);
  border-color: var(--osu-pink);
  color: var(--osu-pink);
}
.play-btn:hover { background: var(--osu-pink); color: #fff; }

.no-selection { display: flex; align-items: center; justify-content: center; height: 100%; width: 100%; }
.no-sel-inner { text-align: center; color: var(--player-expand-color); }
.no-sel-inner i { font-size: 56px; display: block; margin-bottom: 10px; }

.fade-player-enter-active,
.fade-player-leave-active { transition: opacity 0.3s ease; position: absolute; inset: 0; }
.fade-player-enter-from,
.fade-player-leave-to { opacity: 0; }

/* ── Sidebar ──────────────────────────────────────────────────────────────── */
.sidebar {
  width: 33.333%;
  min-width: 300px;
  min-height: 0;
  background: var(--player-sidebar-bg);
  display: flex;
  flex-direction: column;
  border-left: 1px solid var(--player-border);
  overflow: hidden;
  transition: background 0.25s ease;
}

/* ── Search ───────────────────────────────────────────────────────────────── */
.search-wrap { padding: 14px 12px 8px; border-bottom: 1px solid var(--player-border-inner); flex-shrink: 0; }
.search-box  { position: relative; display: flex; align-items: center; }

.search-icon {
  position: absolute; left: 11px;
  color: var(--bs-tertiary-color); font-size: 12px; pointer-events: none;
}

.search-input {
  width: 100%;
  background: var(--player-search-bg);
  border: 1px solid var(--player-search-border);
  border-radius: 8px;
  color: var(--bs-body-color);
  padding: 8px 34px 8px 32px;
  font-size: 12px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s, background 0.25s;
}
.search-input::placeholder { color: var(--player-text-meta); }
.search-input:focus {
  border-color: var(--osu-pink-border);
  box-shadow: 0 0 0 2px var(--osu-pink-subtle);
}

.clear-btn {
  position: absolute; right: 9px;
  background: none; border: none; color: var(--bs-tertiary-color);
  cursor: pointer; font-size: 16px; padding: 0;
  display: flex; align-items: center; transition: color 0.15s;
}
.clear-btn:hover { color: var(--osu-pink); }
.result-count { font-size: 10px; color: var(--player-text-meta); margin-top: 5px; padding-left: 2px; }

/* ── Song List ────────────────────────────────────────────────────────────── */
.song-list {
  flex: 1;
  height: 0;        /* критично: без этого flex:1 не ограничивает высоту */
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  scrollbar-width: thin;
  scrollbar-color: var(--player-search-border) transparent;
}
.song-list::-webkit-scrollbar { width: 3px; }
.song-list::-webkit-scrollbar-thumb { background: var(--player-search-border); border-radius: 2px; }

/* ── Group row ────────────────────────────────────────────────────────────── */
.song-item {
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 68px;
  padding: 0 10px 0 8px;
  cursor: pointer;
  border-left: 3px solid transparent;
  transform: translateX(-8px);
  transition: transform 0.22s cubic-bezier(0.25,0.46,0.45,0.94), background 0.16s, border-color 0.16s;
  user-select: none;
  box-sizing: border-box;
}

.song-item:hover {
  transform: translateX(0);
  background: var(--player-item-hover-bg);
  border-left-color: var(--osu-pink-border);
}

.song-item.group-active {
  background: var(--player-item-active-bg);
  border-left-color: var(--osu-pink);
}

.song-item.expanded {
  transform: translateX(4px);
  border-left-color: var(--osu-pink-border);
}

.thumb { width: 44px; height: 44px; border-radius: 6px; overflow: hidden; flex-shrink: 0; background: var(--player-thumb-bg); }
.thumb-img { width: 100%; height: 100%; object-fit: cover; display: block; }
.thumb-placeholder { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; color: var(--player-thumb-icon-color); font-size: 17px; }

.song-info { flex: 1; min-width: 0; }

.song-name {
  font-size: 12.5px; font-weight: 600; color: var(--player-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 2px;
}
.song-item.group-active .song-name { color: var(--bs-heading-color); }

.song-artist {
  font-size: 10.5px; color: var(--player-text-secondary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 4px;
}

.song-meta { display: flex; align-items: center; gap: 7px; }

.diff-count {
  font-size: 9.5px; color: var(--osu-pink);
  background: var(--osu-pink-subtle);
  opacity: 0.8;
  padding: 1px 6px; border-radius: 10px;
}

.song-stars { font-size: 9.5px; color: rgba(255,215,0,0.7); }

.expand-icon {
  color: var(--player-expand-color); font-size: 11px; flex-shrink: 0;
  transition: transform 0.22s cubic-bezier(0.25,0.46,0.45,0.94), color 0.16s;
}
.expand-icon.open { transform: rotate(90deg); color: var(--osu-pink); opacity: 0.7; }

/* ── Diff row ─────────────────────────────────────────────────────────────── */
.diff-item {
  display: flex;
  align-items: center;
  min-height: 44px;
  padding: 0 12px 0 0;
  cursor: pointer;
  border-left: 3px solid transparent;
  background: var(--player-diff-bg);
  transform: translateX(-4px);
  transition: transform 0.18s cubic-bezier(0.25,0.46,0.45,0.94), background 0.14s, border-color 0.14s;
  user-select: none;
  box-sizing: border-box;
}

.diff-item:hover {
  transform: translateX(2px);
  background: var(--player-diff-hover-bg);
  border-left-color: var(--osu-pink-subtle);
}

.diff-item.active {
  transform: translateX(8px);
  background: var(--player-diff-active-bg);
  border-left-color: var(--osu-pink);
}

.diff-indent { width: 56px; flex-shrink: 0; }

.diff-star-bar {
  width: 3px; height: 28px;
  border-radius: 2px; flex-shrink: 0;
  margin-right: 10px; opacity: 0.85;
}

.diff-info { flex: 1; min-width: 0; }

.diff-name {
  font-size: 11.5px; font-weight: 500; color: var(--bs-secondary-color);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  display: block; margin-bottom: 2px;
}
.diff-item.active .diff-name { color: var(--bs-heading-color); }

.diff-stats {
  font-size: 9.5px; color: var(--player-text-meta);
  display: flex; align-items: center; gap: 5px;
}
.diff-item.active .diff-stats { color: var(--player-text-secondary); }

/* ── Play indicator ───────────────────────────────────────────────────────── */
.play-indicator {
  font-size: 16px; color: var(--osu-pink); flex-shrink: 0;
  animation: pulse-icon 1.8s ease-in-out infinite;
}
.play-indicator.small { font-size: 13px; }

@keyframes pulse-icon {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.4; }
}

/* ── Empty ────────────────────────────────────────────────────────────────── */
.empty-state { text-align: center; padding: 50px 20px; color: var(--player-text-meta); }
.empty-state i { font-size: 36px; display: block; margin-bottom: 10px; }
.empty-state small { display: block; margin-top: 8px; color: var(--player-text-meta); }
.empty-state code {
  background: var(--player-thumb-bg);
  padding: 1px 5px; border-radius: 4px; font-size: 11px;
  color: var(--osu-pink); opacity: 0.6;
}

.volume-control {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 12px;
  padding-left: 12px;
  border-left: 1px solid var(--osu-pink-subtle);
}

.volume-btn {
  width: 38px;
  height: 38px;
  font-size: 16px;
}

.volume-slider {
  width: 100px;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: var(--player-search-bg);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
  transition: background 0.2s;
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  background: var(--osu-pink);
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  box-shadow: 0 0 0 0 var(--osu-pink-glow);
}

.volume-slider::-webkit-slider-thumb:hover {
  transform: scale(1.2);
  box-shadow: 0 0 8px var(--osu-pink-glow);
}

.volume-slider::-moz-range-thumb {
  width: 14px;
  height: 14px;
  background: var(--osu-pink);
  border: none;
  border-radius: 50%;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}

.volume-slider::-moz-range-thumb:hover {
  transform: scale(1.2);
  box-shadow: 0 0 8px var(--osu-pink-glow);
}

.volume-slider::-webkit-slider-runnable-track {
  background: linear-gradient(to right,
  var(--osu-pink) 0%,
  var(--osu-pink) calc(var(--volume-percent, 100) * 1%),
  var(--player-search-bg) calc(var(--volume-percent, 100) * 1%),
  var(--player-search-bg) 100%
  );
  border-radius: 2px;
}

.volume-slider::-moz-range-track {
  background: var(--player-search-bg);
  border-radius: 2px;
}

.volume-slider::-moz-range-progress {
  background: var(--osu-pink);
  border-radius: 2px;
}
</style>