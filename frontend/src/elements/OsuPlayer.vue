<template>
  <div class="osu-app">
    <!-- ── Левая часть: детали выбранной карты ────── -->
    <div class="player-area">
      <MapDetailsPanel
          :group="selectedGroup"
          :diff="selectedDiff"
          :mods-label="currentModsLabel"
          @play="playSelected"
      />
    </div>

    <!-- ── Правая часть: список треков ──────────────────────────────── -->
    <div class="sidebar">
      <div class="search-wrap">
        <div class="search-box">
          <i class="bi bi-search search-icon"></i>
          <input
              v-model="searchQuery"
              type="text"
              class="search-input"
              placeholder="artist, title… stars>6, ar>9, bpm>200, mode=HDDT"
              @input="debouncedSearch"
          />
          <button
              v-if="searchQuery"
              class="clear-btn"
              @click="clearSearch"
          >
            <i class="bi bi-x"></i>
          </button>
        </div>
        <div class="result-count">
          <span v-if="loading && groups.length === 0">loading…</span>
          <span v-else>{{ groups.length }}{{ hasMore ? '+' : '' }} maps</span>
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
        <template #default="{ item, index }">
          <div :data-index="index">
            <!-- ── Group row ── -->
            <div
                v-if="item.type === 'group'"
                class="song-item"
                :class="{
                  expanded: expandedGroups.has(item.beatmap_set_id),
                  'group-active': playerStore.currentTrack?.groupSetId === item.beatmap_set_id,
                  'group-selected': selectedGroup?.beatmap_set_id === item.beatmap_set_id,
                }"
                @click="toggleGroup(item)"
            >
              <div class="thumb">
                <img
                    v-if="!thumbErrors[item.beatmap_set_id]"
                    :src="`/api/osu/bg/${item.beatmap_set_id}.jpg`"
                    class="thumb-img"
                    loading="lazy"
                    @error="thumbErrors[item.beatmap_set_id] = true"
                />
                <div v-else class="thumb-placeholder">
                  <i class="bi bi-music-note"></i>
                </div>
              </div>
              <div class="song-info">
                <div class="song-name">{{ item.song_title }}</div>
                <div class="song-artist">{{ item.artist_name }}</div>
                <div class="song-meta">
                  <span class="diff-count">
                    <i class="bi bi-layers"></i>
                    {{ item.diffs.length }}
                  </span>
                  <span class="song-stars">
                    <i class="bi bi-star-fill"></i>
                    {{ item.stars_min.toFixed(1) }}
                    <span v-if="item.stars_max !== item.stars_min">–{{ item.stars_max.toFixed(1) }}</span>
                  </span>
                </div>
              </div>
              <div
                  class="expand-icon"
                  :class="{ open: expandedGroups.has(item.beatmap_set_id) }"
              >
                <i class="bi bi-chevron-right"></i>
              </div>
              <div
                  class="play-indicator"
                  v-if="playerStore.currentTrack?.groupSetId === item.beatmap_set_id && playerStore.isPlaying"
              >
                <i class="bi bi-play-fill"></i>
              </div>
            </div>

            <!-- ── Diff row ── -->
            <div
                v-else-if="item.type === 'diff'"
                class="diff-item"
                :class="{
                  active: playerStore.currentTrack?.id === String(item.beatmap_id),
                  selected: selectedDiff?.beatmap_id === item.beatmap_id,
                }"
                @click="selectDiff(item._group, item)"
            >
              <div class="diff-indent"></div>
              <div class="diff-star-bar" :style="{ background: starColor(item.stars) }"></div>
              <div class="diff-info">
                <span class="diff-name">{{ item.difficulty }}</span>
                <span class="diff-stats">
                  <i class="bi bi-star-fill"></i>
                  {{ item.stars?.toFixed(2) ?? '?' }} · AR {{ item.approach_rate }} ·
                  {{ formatTime(item.drain_time) }}
                </span>
              </div>
              <div
                  class="play-indicator small"
                  v-if="playerStore.currentTrack?.id === String(item.beatmap_id)"
              >
                <i :class="playerStore.isPlaying ? 'bi bi-pause-fill' : 'bi bi-play-fill'"></i>
              </div>
            </div>
          </div>
        </template>
      </RecycleScroller>

      <div v-if="loading && groups.length > 0" class="loading-more">loading more…</div>

      <div v-if="!loading && groups.length === 0" class="empty-state">
        <i class="bi bi-emoji-frown"></i>
        <p>No maps found</p>
        <small v-if="searchQuery">
          Try: <code>stars&gt;5</code> · <code>ar&gt;9</code> · <code>bpm&gt;180</code> · <code>mode=HDDT</code>
        </small>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import { useThemeStore } from '@/stores/theme'
import { usePlayerStore } from '@/stores/player.store'
import { diffToTrack, useOsuCatalog } from '@/composables/useOsuCatalog'
import MapDetailsPanel from './MapDetailsPanel.vue'

const themeStore = useThemeStore()
const playerStore = usePlayerStore()
const { groups, loading, hasMore, search, loadMore, getEffectiveQuery, getSort } = useOsuCatalog()

const expandedGroups = ref(new Set())
const searchQuery = ref('')
const scrollerEl = ref(null)
const thumbErrors = reactive({})
let debounceTimer = null

// ── Выбор (для левой панели) отдельно от воспроизведения ──────────
// Клик по diff-строке ТОЛЬКО выбирает карту для предпросмотра —
// не трогает текущий трек в глобальном плеере. Запуск — явной кнопкой
// Play в MapDetailsPanel.
const selectedGroup = ref(null)
const selectedDiff = ref(null)

onMounted(() => {
  themeStore.apply()
  nextTick(attachScrollListener)
})

onBeforeUnmount(() => {
  detachScrollListener()
})

// ── Infinite scroll ──────────────────────────────────────────────────
const SCROLL_THRESHOLD_PX = 300
let scrollTarget = null

function handleScroll() {
  if (!scrollTarget) return
  const { scrollTop, clientHeight, scrollHeight } = scrollTarget
  if (scrollHeight - (scrollTop + clientHeight) < SCROLL_THRESHOLD_PX) {
    loadMore()
  }
}

function attachScrollListener() {
  scrollTarget = scrollerEl.value?.$el
  if (!scrollTarget) return
  scrollTarget.addEventListener('scroll', handleScroll)
}

function detachScrollListener() {
  if (scrollTarget) scrollTarget.removeEventListener('scroll', handleScroll)
  scrollTarget = null
}

const flatList = computed(() => {
  const list = []
  for (const g of groups.value) {
    list.push({ ...g, type: 'group', _key: `g-${g.beatmap_set_id}`, _size: 68 })
    if (expandedGroups.value.has(g.beatmap_set_id)) {
      for (const d of g.diffs) {
        list.push({ ...d, type: 'diff', _key: `d-${d.beatmap_id}`, _group: g, _size: 44 })
      }
    }
  }
  return list
})

/** Явный запуск воспроизведения — вызывается только из кнопки Play. */
function playSelected() {
  if (!selectedGroup.value || !selectedDiff.value) return
  const track = diffToTrack(selectedGroup.value, selectedDiff.value)
  playerStore.playFromBrowse(track, { query: getEffectiveQuery(), sort: getSort() })
}

function scrollToSelected(key, { smooth = true } = {}) {
  const list = flatList.value
  const idx = list.findIndex((item) => item._key === key)
  if (idx === -1 || !scrollerEl.value) return
  const scroller = scrollerEl.value.$el
  if (!scroller || scroller.clientHeight === 0) return
  let offsetTop = 0
  for (let i = 0; i < idx; i++) offsetTop += list[i]._size
  const itemH = list[idx]._size
  const target = offsetTop - scroller.clientHeight / 2 + itemH / 2
  scroller.scrollTo({ top: Math.max(0, target), behavior: smooth ? 'smooth' : 'auto' })
}

/** Разворот группы — только показывает диффы и выбирает первую для
 *  предпросмотра в левой панели, воспроизведение не трогает. */
function toggleGroup(group) {
  const wasOpen = expandedGroups.value.has(group.beatmap_set_id)
  const s = new Set()
  if (!wasOpen) {
    s.add(group.beatmap_set_id)
    selectDiffQuiet(group, group.diffs[0])
  }
  expandedGroups.value = s
  const scrollKey = !wasOpen ? `d-${group.diffs[0]?.beatmap_id}` : `g-${group.beatmap_set_id}`
  scrollToSelected(scrollKey)
}

/** Выбор без скролла — используется для авто-выбора первой диффы при
 *  разворачивании группы, где скролл уже управляется отдельно. */
function selectDiffQuiet(group, diff) {
  selectedGroup.value = group
  selectedDiff.value = diff
}

function selectDiff(group, diff) {
  if (!diff) return
  selectDiffQuiet(group, diff)
}

// ── Синхронизация списка/панели с плеером ───────────────────────────
// При смене трека (в т.ч. next/prev из глобального плеера) левая
// панель следует за тем, что реально играет — пока пользователь явно
// не выберет другую карту для предпросмотра.
function findGroupByTrack(track) {
  if (track?.groupSetId == null) return null
  return groups.value.find((g) => g.beatmap_set_id === track.groupSetId) ?? null
}

function syncSelectionToTrack(track, { smooth = true } = {}) {
  const group = findGroupByTrack(track)
  if (!group) return
  const diff = group.diffs.find((d) => String(d.beatmap_id) === track.id)
  expandedGroups.value = new Set([group.beatmap_set_id])
  selectedGroup.value = group
  selectedDiff.value = diff ?? group.diffs[0]
  nextTick(() => scrollToSelected(`d-${track.id}`, { smooth }))
}

let hasInitialSync = false
watch(groups, () => {
  if (hasInitialSync || loading.value) return
  hasInitialSync = true
  if (playerStore.currentTrack) {
    syncSelectionToTrack(playerStore.currentTrack, { smooth: false })
  }
})

watch(
    () => playerStore.currentTrack,
    (track, oldTrack) => {
      if (!hasInitialSync) return
      if (track && track.id !== oldTrack?.id) {
        syncSelectionToTrack(track)
      }
    },
)

function debouncedSearch() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    search(searchQuery.value)
  }, 150)
}

function clearSearch() {
  searchQuery.value = ''
  search('')
  clearTimeout(debounceTimer)
}

function formatTime(seconds) {
  if (!seconds) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

function starColor(stars) {
  if (!stars) return 'var(--bs-border-color)'
  if (stars < 2) return 'var(--osu-star-1)'
  if (stars < 3) return 'var(--osu-star-2)'
  if (stars < 4) return 'var(--osu-star-3)'
  if (stars < 5) return 'var(--osu-star-4)'
  if (stars < 6) return 'var(--osu-star-5)'
  if (stars < 7) return 'var(--osu-star-6)'
  return 'var(--osu-star-7)'
}

const currentModsLabel = computed(() => {
  const match = searchQuery.value.match(/\bmode=([A-Za-z]+)\b/)
  return match ? match[1].toUpperCase() : 'NoMod'
})
</script>

<style scoped>
.osu-app {
  display: flex;
  flex: 1 1 0;
  height: 100%;
  min-height: 0;
  background: var(--player-bg);
  color: var(--bs-body-color);
  font-family: -apple-system, "Segoe UI", sans-serif;
  overflow: hidden;
  transition: background 0.25s ease, color 0.25s ease;
}

/* Пробивает scoping в дочерний компонент RecycleScroller — замена
   прежнего глобального <style> блока без scoped-изоляции. */
.song-list :deep(.vue-recycle-scroller__item-wrapper) {
  will-change: transform;
}

.player-area {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

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

.search-wrap {
  padding: 14px 12px 8px;
  border-bottom: 1px solid var(--player-border-inner);
  flex-shrink: 0;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 11px;
  color: var(--bs-tertiary-color);
  font-size: 12px;
  pointer-events: none;
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

.search-input::placeholder {
  color: var(--player-text-meta);
}

.search-input:focus {
  border-color: var(--osu-pink-border);
  box-shadow: 0 0 0 2px var(--osu-pink-subtle);
}

.clear-btn {
  position: absolute;
  right: 9px;
  background: none;
  border: none;
  color: var(--bs-tertiary-color);
  cursor: pointer;
  font-size: 16px;
  padding: 0;
  display: flex;
  align-items: center;
  transition: color 0.15s;
}

.clear-btn:hover {
  color: var(--osu-pink);
}

.result-count {
  font-size: 10px;
  color: var(--player-text-meta);
  margin-top: 5px;
  padding-left: 2px;
}

.song-list {
  flex: 1;
  height: 0;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  scrollbar-width: thin;
  scrollbar-color: var(--player-search-border) transparent;
}

.song-list::-webkit-scrollbar {
  width: 3px;
}

.song-list::-webkit-scrollbar-thumb {
  background: var(--player-search-border);
  border-radius: 2px;
}

.song-item {
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 68px;
  padding: 0 10px 0 8px;
  cursor: pointer;
  border-left: 3px solid transparent;
  transform: translateX(-8px);
  transition: transform 0.22s cubic-bezier(0.25, 0.46, 0.45, 0.94), background 0.16s, border-color 0.16s;
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

.song-item.group-selected:not(.group-active) {
  border-left-color: var(--osu-pink-subtle);
}

.song-item.expanded {
  transform: translateX(4px);
  border-left-color: var(--osu-pink-border);
}

.thumb {
  width: 44px;
  height: 44px;
  border-radius: 6px;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--player-thumb-bg);
}
.thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.thumb-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--player-thumb-icon-color);
  font-size: 17px;
}

.song-info {
  flex: 1;
  min-width: 0;
}

.song-name {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--player-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 2px;
}

.song-item.group-active .song-name {
  color: var(--bs-heading-color);
}

.song-artist {
  font-size: 10.5px;
  color: var(--player-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.song-meta {
  display: flex;
  align-items: center;
  gap: 7px;
}

.diff-count {
  font-size: 9.5px;
  color: var(--osu-pink);
  background: var(--osu-pink-subtle);
  opacity: 0.8;
  padding: 1px 6px;
  border-radius: 10px;
}

.song-stars {
  font-size: 9.5px;
  color: rgba(255, 215, 0, 0.7);
}

.expand-icon {
  color: var(--player-expand-color);
  font-size: 11px;
  flex-shrink: 0;
  transition: transform 0.22s cubic-bezier(0.25, 0.46, 0.45, 0.94), color 0.16s;
}

.expand-icon.open {
  transform: rotate(90deg);
  color: var(--osu-pink);
  opacity: 0.7;
}

.diff-item {
  display: flex;
  align-items: center;
  min-height: 44px;
  padding: 0 12px 0 0;
  cursor: pointer;
  border-left: 3px solid transparent;
  background: var(--player-diff-bg);
  transform: translateX(-4px);
  transition: transform 0.18s cubic-bezier(0.25, 0.46, 0.45, 0.94), background 0.14s, border-color 0.14s;
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

.diff-item.selected:not(.active) {
  border-left-color: var(--osu-pink-subtle);
  background: var(--player-diff-hover-bg);
}

.diff-indent {
  width: 56px;
  flex-shrink: 0;
}

.diff-star-bar {
  width: 3px;
  height: 28px;
  border-radius: 2px;
  flex-shrink: 0;
  margin-right: 10px;
  opacity: 0.85;
}

.diff-info {
  flex: 1;
  min-width: 0;
}

.diff-name {
  font-size: 11.5px;
  font-weight: 500;
  color: var(--bs-secondary-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  margin-bottom: 2px;
}

.diff-item.active .diff-name {
  color: var(--bs-heading-color);
}

.diff-stats {
  font-size: 9.5px;
  color: var(--player-text-meta);
  display: flex;
  align-items: center;
  gap: 5px;
}

.diff-item.active .diff-stats {
  color: var(--player-text-secondary);
}

.play-indicator {
  font-size: 16px;
  color: var(--osu-pink);
  flex-shrink: 0;
  animation: pulse-icon 1.8s ease-in-out infinite;
}

.play-indicator.small {
  font-size: 13px;
}

@keyframes pulse-icon {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.loading-more {
  text-align: center;
  padding: 10px;
  font-size: 11px;
  color: var(--player-text-meta);
}

.empty-state {
  text-align: center;
  padding: 50px 20px;
  color: var(--player-text-meta);
}

.empty-state i {
  font-size: 36px;
  display: block;
  margin-bottom: 10px;
}

.empty-state small {
  display: block;
  margin-top: 8px;
  color: var(--player-text-meta);
}

.empty-state code {
  background: var(--player-thumb-bg);
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 11px;
  color: var(--osu-pink);
  opacity: 0.6;
}
</style>