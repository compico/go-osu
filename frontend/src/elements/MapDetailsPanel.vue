<template>
  <div class="map-details">
    <div v-if="!diff" class="details-placeholder">
      <i class="bi bi-bar-chart-line"></i>
      <p>Select a difficulty to see details</p>
    </div>

    <div v-else class="details-content">
      <div class="details-header">
        <h2 class="details-title">{{ group.song_title }}</h2>
        <p class="details-artist">{{ group.artist_name }}</p>
        <p class="details-diff-name">
          <i class="bi bi-star-fill"></i> {{ diff.difficulty }}
        </p>
      </div>

      <button class="play-btn" @click="$emit('play')">
        <i :class="isCurrentAndPlaying ? 'bi bi-pause-fill' : 'bi bi-play-fill'"></i>
        {{ isCurrentAndPlaying ? 'Playing' : (isCurrent ? 'Resume' : 'Play') }}
      </button>

      <div class="details-stats">
        <div class="stat"><span class="stat-label">Stars</span><span class="stat-value">{{ diff.stars?.toFixed(2) ?? '?' }}</span></div>
        <div class="stat"><span class="stat-label">AR</span><span class="stat-value">{{ diff.approach_rate }}</span></div>
        <div class="stat"><span class="stat-label">BPM</span><span class="stat-value">{{ diff.bpm }}</span></div>
        <div class="stat"><span class="stat-label">Length</span><span class="stat-value">{{ formatTime(diff.drain_time) }}</span></div>
        <div class="stat"><span class="stat-label">Mapper</span><span class="stat-value">{{ diff.creator_name }}</span></div>
      </div>

      <div class="skills-block">
        <div class="skills-header">
          <h3 class="skills-title">Skills</h3>
          <span v-if="diff.skills" class="skills-mods-badge">{{ modsLabel }}</span>
        </div>

        <div v-if="diff.skills">
          <div v-for="row in skillRows" :key="row.label" class="skill-row">
            <span class="skill-label">{{ row.label }}</span>
            <div class="skill-bar-track">
              <div class="skill-bar-fill" :style="{ width: row.pct + '%' }"></div>
            </div>
            <span class="skill-value">{{ row.value.toFixed(0) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { usePlayerStore } from '@/stores/player.store'

const props = defineProps({
  group: { type: Object, default: null },
  diff: { type: Object, default: null },
  /** Строка модов как в query, например "HDDT" — только для подписи над skills. */
  modsLabel: { type: String, default: 'NoMod' },
})
defineEmits(['play'])

const playerStore = usePlayerStore()

const isCurrent = computed(() => playerStore.currentTrack?.id === String(props.diff?.beatmap_id))
const isCurrentAndPlaying = computed(() => isCurrent.value && playerStore.isPlaying)

const SKILL_BAR_MAX = 1000

const skillRows = computed(() => {
  if (!props.diff?.skills) return []
  const s = props.diff.skills
  return [
    { label: 'Stamina', value: s.stamina },
    { label: 'Tenacity', value: s.tenacity },
    { label: 'Agility', value: s.agility },
    { label: 'Precision', value: s.precision },
    { label: 'Reading', value: s.reading },
    { label: 'Memory', value: s.memory },
    { label: 'Accuracy', value: s.accuracy },
    { label: 'Reaction', value: s.reaction },
  ].map((r) => ({ ...r, pct: Math.min(100, (r.value / SKILL_BAR_MAX) * 100) }))
})

function formatTime(seconds) {
  if (!seconds) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}
</script>

<style scoped>
.map-details {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  padding: 24px;
}

.details-placeholder {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--player-expand-color);
  text-align: center;
}
.details-placeholder i { font-size: 56px; margin-bottom: 10px; }

.details-content { max-width: 480px; margin: 0 auto; }

.details-header { margin-bottom: 16px; }
.details-title { font-size: 20px; font-weight: 700; color: var(--player-text-primary); margin: 0; }
.details-artist { font-size: 13px; color: var(--player-text-secondary); margin: 4px 0 0; }
.details-diff-name { font-size: 12px; color: var(--osu-pink); margin: 8px 0 0; }

.play-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid var(--osu-pink);
  background: var(--osu-pink-subtle);
  color: var(--osu-pink);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.18s, color 0.18s;
  margin-bottom: 20px;
}
.play-btn:hover { background: var(--osu-pink); color: #fff; }

.details-stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  margin-bottom: 20px;
}
.stat {
  display: flex;
  flex-direction: column;
  background: var(--player-item-hover-bg);
  border-radius: 8px;
  padding: 8px 10px;
}
.stat-label { font-size: 10px; color: var(--player-text-meta); }
.stat-value { font-size: 14px; font-weight: 600; color: var(--player-text-primary); }

.skills-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.skills-title { font-size: 13px; color: var(--player-text-secondary); margin: 0; }
.skills-mods-badge {
  font-size: 10px;
  font-weight: 600;
  color: var(--osu-pink);
  background: var(--osu-pink-subtle);
  padding: 2px 8px;
  border-radius: 10px;
}

.skill-row { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.skill-label { width: 70px; font-size: 10.5px; color: var(--player-text-secondary); flex-shrink: 0; }
.skill-bar-track {
  flex: 1;
  height: 6px;
  background: var(--player-search-bg);
  border-radius: 3px;
  overflow: hidden;
}
.skill-bar-fill {
  height: 100%;
  background: var(--osu-pink);
  border-radius: 3px;
}
.skill-value { width: 34px; text-align: right; font-size: 10.5px; color: var(--player-text-meta); flex-shrink: 0; }

.skills-empty {
  font-size: 11px;
  color: var(--player-text-meta);
  line-height: 1.5;
}
.skills-empty code {
  background: var(--player-thumb-bg);
  padding: 1px 5px;
  border-radius: 4px;
  color: var(--osu-pink);
  opacity: 0.8;
}
</style>