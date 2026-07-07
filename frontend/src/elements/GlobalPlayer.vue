<template>
  <div class="global-player" v-if="player.currentTrack">
    <div
        class="timeline-zone"
        @mouseenter="hovering = true"
        @mouseleave="hovering = false"
        @mousedown="startSeek"
    >
      <div class="timeline-track" :class="{ hovering }">
        <div class="timeline-fill" :style="{ width: progressPct + '%' }"></div>
        <div
            class="timeline-handle"
            :class="{ hovering }"
            :style="{ left: progressPct + '%' }"
        ></div>
      </div>
    </div>

    <div class="player-bar">
      <div class="bar-left">
        <div class="bar-cover">
          <img
              v-if="!coverError && player.currentTrack.coverUrl"
              :src="player.currentTrack.coverUrl"
              @error="coverError = true"
          />
          <div v-else class="bar-cover-ph">
            <i class="bi bi-music-note-beamed"></i>
          </div>
        </div>
        <div class="bar-meta">
          <div class="bar-title">{{ player.currentTrack.title }}</div>
          <div class="bar-artist">{{ player.currentTrack.artist }}</div>
          <div class="bar-time">
            {{ formatTime(displayTime) }} / {{ formatTime(player.duration) }}
          </div>
        </div>
      </div>

      <div class="bar-center">
        <button class="bar-btn" @click="player.previous()">
          <i class="bi bi-skip-start-fill"></i>
        </button>
        <button class="bar-btn bar-btn--play" @click="player.toggle()">
          <i v-if="player.isLoading" class="bi bi-arrow-repeat spin"></i>
          <i v-else :class="player.isPlaying ? 'bi bi-pause-fill' : 'bi bi-play-fill'"></i>
        </button>
        <button class="bar-btn" @click="player.next()">
          <i class="bi bi-skip-end-fill"></i>
        </button>
      </div>

      <div class="bar-right">
        <div class="volume-control">
          <button class="bar-btn bar-btn--sm" @click="player.toggleMute()">
            <i :class="volumeIcon"></i>
          </button>
          <input
              type="range"
              class="volume-slider"
              min="0"
              max="1"
              step="0.01"
              :value="sliderPos"
              :style="{ '--volume-percent': sliderPos * 100 }"
              @input="onVolumeChange"
          />
        </div>
        <button class="bar-btn bar-btn--fav" disabled title="Скоро">
          <i class="bi bi-heart"></i>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { usePlayerStore } from '@/stores/player.store'
import { useMediaSession } from '@/composables/useMediaSession'
import { audioService } from '@/services/audio.service'
import { sliderToVolume, volumeToSlider } from '@/utils/volumeCurve'

const player = usePlayerStore()
const hovering = ref(false)
const coverError = ref(false)
const seeking = ref(false)

useMediaSession()

onMounted(() => {
  player.hydrate()
})

watch(() => player.currentTrack, () => { coverError.value = false })

// ── Плавный таймлайн через rAF ────────────────────────────────────
// timeupdate у <audio> срабатывает нерегулярно (~раз в 200-250мс),
// поэтому визуальную позицию гоним отдельным циклом каждый кадр,
// читая currentTime напрямую из аудио-элемента.
const displayTime = ref(0)
let rafId: number | null = null

function tick() {
  if (!seeking.value) {
    displayTime.value = player.isPlaying
        ? audioService.currentTime
        : player.currentTime
  }
  rafId = requestAnimationFrame(tick)
}

onMounted(() => {
  displayTime.value = player.currentTime
  rafId = requestAnimationFrame(tick)
})

onUnmounted(() => {
  if (rafId !== null) cancelAnimationFrame(rafId)
})

const progressPct = computed(() => {
  if (!player.duration) return 0
  return Math.min(100, (displayTime.value / player.duration) * 100)
})

// ── Громкость с нелинейной кривой ──────────────────────────────────
const sliderPos = computed(() =>
    player.isMuted ? 0 : volumeToSlider(player.volume),
)

const volumeIcon = computed(() => {
  if (player.isMuted || player.volume === 0) return 'bi bi-volume-mute-fill'
  if (player.volume < 0.1) return 'bi bi-volume-off-fill'
  if (player.volume < 0.4) return 'bi bi-volume-down-fill'
  return 'bi bi-volume-up-fill'
})

function onVolumeChange(e: Event) {
  const rawSliderPos = parseFloat((e.target as HTMLInputElement).value)
  player.setVolume(sliderToVolume(rawSliderPos))
}

function formatTime(seconds: number) {
  if (!seconds || !isFinite(seconds)) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

function ratioFromEvent(e: MouseEvent, el: HTMLElement) {
  const rect = el.getBoundingClientRect()
  return Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
}

function applySeek(e: MouseEvent, zone: HTMLElement) {
  if (!player.duration) return
  const t = ratioFromEvent(e, zone) * player.duration
  displayTime.value = t
  player.seek(t)
}

function startSeek(e: MouseEvent) {
  if (!player.duration) return
  seeking.value = true
  const zone = e.currentTarget as HTMLElement
  applySeek(e, zone)

  const onMove = (ev: MouseEvent) => applySeek(ev, zone)
  const onUp = () => {
    seeking.value = false
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}
</script>

<style scoped>
.global-player {
  position: fixed;
  left: 0; right: 0; bottom: 0;
  z-index: 1030;
  background: var(--player-sidebar-bg);
  border-top: 1px solid var(--player-border);
}
.timeline-zone {
  position: relative;
  height: 14px;
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 0 2px;
}
.timeline-track {
  position: relative;
  width: 100%;
  height: 3px;
  background: var(--player-search-bg);
  border-radius: 2px;
  transition: height 0.15s ease;
}
.timeline-track.hovering { height: 6px; }
.timeline-fill {
  position: absolute;
  inset: 0 auto 0 0;
  background: var(--osu-pink);
  border-radius: 2px;
  pointer-events: none;
  /* без transition — двигаем через rAF каждый кадр, transition тут будет лагать */
}
.timeline-handle {
  position: absolute;
  top: 50%;
  width: 0; height: 0;
  background: var(--osu-pink);
  border-radius: 50%;
  transform: translate(-50%, -50%);
  transition: width 0.15s ease, height 0.15s ease;
  pointer-events: none;
  box-shadow: 0 0 6px var(--osu-pink-glow);
}
.timeline-handle.hovering { width: 12px; height: 12px; }

.player-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 72px;
  padding: 0 16px;
}
.bar-left { display: flex; align-items: center; gap: 12px; flex: 1 1 0; min-width: 0; }

/* ── Обложка: сохраняем пропорции исходника (160×120 = 4:3) ─────── */
.bar-cover {
  width: 64px;
  aspect-ratio: 4 / 3;
  border-radius: 6px;
  overflow: hidden;
  flex-shrink: 0;
  border: 1px solid var(--osu-pink-border);
}
.bar-cover img { width: 100%; height: 100%; object-fit: cover; display: block; }
.bar-cover-ph {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
  background: var(--player-cover-ph-bg);
  color: var(--osu-pink-glow);
  font-size: 18px;
}
.bar-meta { min-width: 0; }
.bar-title {
  font-size: 13px; font-weight: 600;
  color: var(--player-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.bar-artist {
  font-size: 11px; color: var(--player-text-secondary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.bar-time { font-size: 10px; color: var(--player-text-meta); margin-top: 2px; }

.bar-center { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.bar-right { display: flex; align-items: center; gap: 14px; flex: 1 1 0; justify-content: flex-end; }

.bar-btn {
  background: var(--bs-border-color-translucent);
  border: 1px solid var(--osu-pink-subtle);
  color: var(--bs-secondary-color);
  border-radius: 50%;
  width: 38px; height: 38px;
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; cursor: pointer;
  transition: background 0.18s, border-color 0.18s, color 0.18s;
}
.bar-btn:hover { background: var(--osu-pink-subtle); border-color: var(--osu-pink); color: var(--bs-heading-color); }
.bar-btn:disabled { opacity: 0.4; cursor: default; }
.bar-btn--play {
  width: 46px; height: 46px; font-size: 20px;
  background: var(--osu-pink-subtle);
  border-color: var(--osu-pink);
  color: var(--osu-pink);
}
.bar-btn--play:hover { background: var(--osu-pink); color: #fff; }
.bar-btn--sm { width: 32px; height: 32px; font-size: 13px; }

.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.volume-control { display: flex; align-items: center; gap: 8px; }
.volume-slider {
  width: 90px; height: 4px;
  -webkit-appearance: none; appearance: none;
  background: var(--player-search-bg);
  border-radius: 2px; outline: none; cursor: pointer;
}
.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none; appearance: none;
  width: 12px; height: 12px;
  background: var(--osu-pink);
  border-radius: 50%; cursor: pointer;
}
.volume-slider::-webkit-slider-runnable-track {
  background: linear-gradient(
      to right,
      var(--osu-pink) 0%,
      var(--osu-pink) calc(var(--volume-percent, 100) * 1%),
      var(--player-search-bg) calc(var(--volume-percent, 100) * 1%),
      var(--player-search-bg) 100%
  );
  border-radius: 2px;
}
</style>