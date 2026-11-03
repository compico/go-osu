<template>
  <div class="strain-graph">
    <div class="graph-header">
      <h3 class="graph-title">Strain graph</h3>
    </div>

    <div v-if="loading" class="graph-note">Calculating…</div>
    <div v-else-if="error" class="graph-note graph-error">Could not load the graph: {{ error }}</div>
    <template v-else-if="data && hasCurves">
      <div class="graph-wrap">
        <svg
          :viewBox="`0 0 ${W} ${H}`"
          class="graph-svg"
          role="img"
          aria-label="Difficulty of each skill over the course of the map"
          @mousemove="onMove"
          @mouseleave="hoverMs = null"
        >
          <!-- y grid -->
          <g class="grid">
            <template v-for="t in yTicks" :key="'y' + t">
              <line :x1="PAD.left" :x2="W - PAD.right" :y1="y(t)" :y2="y(t)" />
              <text :x="PAD.left - 4" :y="y(t) + 3" text-anchor="end">{{ Math.round(t) }}</text>
            </template>
          </g>
          <!-- x axis -->
          <g class="grid">
            <template v-for="t in xTicks" :key="'x' + t">
              <text :x="x(t)" :y="H - 4" text-anchor="middle">{{ formatClock(t) }}</text>
            </template>
          </g>

          <!-- curves -->
          <g v-for="s in shownSkills" :key="s.key" :stroke="s.color" fill="none" class="curve">
            <template v-for="(seg, i) in s.segments" :key="i">
              <polyline v-if="seg.length > 1" :points="seg.map((p) => `${x(p.t)},${y(p.v)}`).join(' ')" />
              <!-- a run of one point (a short stream): a short tick -->
              <line v-else :x1="x(seg[0].t) - 2" :x2="x(seg[0].t) + 2" :y1="y(seg[0].v)" :y2="y(seg[0].v)" />
            </template>
          </g>

          <!-- hover -->
          <line v-if="hoverMs !== null" class="cursor" :x1="x(hoverMs)" :x2="x(hoverMs)" :y1="PAD.top" :y2="H - PAD.bottom" />
        </svg>

        <div v-if="hoverMs !== null" class="tooltip" :style="tooltipStyle">
          <div class="tooltip-time">{{ formatClock(hoverMs) }}</div>
          <div v-for="row in hoverRows" :key="row.key" class="tooltip-row">
            <span class="swatch" :style="{ background: row.color }"></span>
            <span class="tooltip-label">{{ row.label }}</span>
            <span class="tooltip-value">{{ row.value === null ? '–' : row.value.toFixed(0) }}</span>
          </div>
        </div>
      </div>

      <div class="legend">
        <button
          v-for="s in skillList"
          :key="s.key"
          type="button"
          class="legend-item"
          :class="{ off: !s.available || hidden.has(s.key) }"
          :disabled="!s.available"
          :aria-pressed="s.available && !hidden.has(s.key)"
          :title="s.available ? '' : s.unavailableHint"
          @click="toggle(s.key)"
        >
          <span class="swatch" :style="{ background: s.color }"></span>{{ s.label }}
        </button>
      </div>
      <p class="graph-hint">
        Each line's top is that skill's value above. Memory (FL only) adds up over the map; tenacity shows streams.
      </p>
    </template>
    <div v-else-if="data" class="graph-note">Not enough notes to draw a graph.</div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useStrains } from '@/composables/useStrains'
import { SKILL_KEYS, type SkillKey } from '@/types/strains'
import { formatClock, niceStep, splitSegments, toSteps, valueAt } from '@/utils/strainGraph'

const props = defineProps<{
  beatmapId: number | null | undefined
  /** Mods label as in the card: "NoMod" or e.g. "HDDT". */
  mods: string
}>()

const { data, loading, error } = useStrains(
  computed(() => props.beatmapId),
  computed(() => props.mods),
)

const W = 480
const H = 190
const PAD = { left: 30, right: 6, top: 8, bottom: 18 }

const META: Record<SkillKey, { label: string; color: string; hint?: string }> = {
  stamina: { label: 'Stamina', color: '#ef5350' },
  tenacity: { label: 'Tenacity', color: '#ff9800', hint: 'No streams in this map' },
  agility: { label: 'Agility', color: '#fdd835' },
  precision: { label: 'Precision', color: '#9ccc65' },
  reading: { label: 'Reading', color: '#26c6da' },
  memory: { label: 'Memory', color: '#5c6bc0', hint: 'Memory counts only with FL' },
  accuracy: { label: 'Accuracy', color: '#ab47bc' },
  reaction: { label: 'Reaction', color: '#8d9db0' },
}

const hidden = reactive(new Set<SkillKey>())
function toggle(key: SkillKey) {
  if (hidden.has(key)) hidden.delete(key)
  else hidden.add(key)
}

// A gap longer than this (ms of the calculation's timeline) is a break in the
// map: slow passages of a second or two between notes stay one line.
const MIN_BREAK_MS = 3000
const maxGap = computed(() => Math.max(MIN_BREAK_MS, (data.value?.window ?? 400) * 3))

// Lines of a skill. Memory only adds up: one line. Tenacity is on or off: steps,
// one window wide. The others are cut at breaks.
function segmentsOf(s: { key: SkillKey; points: { t: number; v: number }[] }) {
  const toPoints = (pts: { t: number; v: number }[]) => pts.map((p) => ({ time: p.t, value: p.v }))
  const fromPoints = (pts: { time: number; value: number }[]) => pts.map((p) => ({ t: p.time, v: p.value }))

  if (s.key === 'memory') return [s.points]
  if (s.key === 'tenacity') return [fromPoints(toSteps(toPoints(s.points), data.value?.window || 0))]
  return splitSegments(toPoints(s.points), maxGap.value * (data.value?.speed || 1)).map(fromPoints)
}

/** Every skill's points on the beatmap's own timeline (ms). */
const skillList = computed(() => {
  const speed = data.value?.speed || 1
  return SKILL_KEYS.map((key) => {
    const points = (data.value?.[key] ?? []).map((p) => ({ t: p.time * speed, v: p.value }))
    return {
      key,
      label: META[key].label,
      color: META[key].color,
      unavailableHint: META[key].hint ?? '',
      available: points.length > 0,
      points,
    }
  })
})

const hasCurves = computed(() => skillList.value.some((s) => s.points.length > 1))

const shownSkills = computed(() =>
  skillList.value
    .filter((s) => s.available && !hidden.has(s.key))
    .map((s) => ({
      ...s,
      // The running total of memory is one line; the others break at pauses.
      segments: segmentsOf(s),
    })),
)

const endMs = computed(() => {
  let end = 0
  for (const s of skillList.value) {
    const last = s.points[s.points.length - 1]
    if (last) end = Math.max(end, last.t)
  }
  return Math.max(end, 1)
})

const maxValue = computed(() => {
  let top = 0
  for (const s of shownSkills.value) for (const p of s.points) top = Math.max(top, p.v)
  return top > 0 ? top : 1
})

const x = (ms: number) => PAD.left + (ms / endMs.value) * (W - PAD.left - PAD.right)
const y = (v: number) => H - PAD.bottom - (v / maxValue.value) * (H - PAD.top - PAD.bottom)

const yTicks = computed(() => {
  const step = niceStep(maxValue.value, 4)
  const ticks: number[] = []
  for (let t = 0; t <= maxValue.value * 1.0001; t += step) ticks.push(t)
  return ticks
})

const xTicks = computed(() => {
  const step = niceStep(endMs.value / 1000, 6) * 1000
  const ticks: number[] = []
  for (let t = 0; t <= endMs.value; t += step) ticks.push(t)
  return ticks
})

// ── hover ────────────────────────────────────────────────────────────────
const hoverMs = ref<number | null>(null)

function onMove(e: MouseEvent) {
  const svg = e.currentTarget as SVGSVGElement
  const rect = svg.getBoundingClientRect()
  if (rect.width === 0) return
  const px = ((e.clientX - rect.left) / rect.width) * W
  const ms = ((px - PAD.left) / (W - PAD.left - PAD.right)) * endMs.value
  hoverMs.value = ms < 0 || ms > endMs.value ? null : ms
}

const hoverRows = computed(() => {
  const t = hoverMs.value
  if (t === null) return []
  const gap = maxGap.value * (data.value?.speed || 1)
  return skillList.value
    .filter((s) => s.available && !hidden.has(s.key))
    .map((s) => ({
      key: s.key,
      label: s.label,
      color: s.color,
      // The running total of memory holds its value through a break.
      value: valueAt(
        s.points.map((p) => ({ time: p.t, value: p.v })),
        t,
        s.key === 'memory' ? Infinity : gap,
      ),
    }))
})

const tooltipStyle = computed(() => {
  const t = hoverMs.value ?? 0
  const frac = x(t) / W
  return frac > 0.55 ? { right: `${(1 - frac) * 100 + 2}%` } : { left: `${frac * 100 + 2}%` }
})
</script>

<style scoped>
.strain-graph { margin-top: 24px; }
.graph-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.graph-title { font-size: 13px; color: var(--player-text-secondary); margin: 0; }

.graph-note { font-size: 11px; color: var(--player-text-meta); }
.graph-error { color: var(--osu-pink); }

.graph-wrap { position: relative; }
.graph-svg { width: 100%; height: auto; display: block; }

.grid text { font-size: 8px; fill: var(--player-text-meta); }
.grid line { stroke: var(--player-search-bg); stroke-width: 1; }
.curve polyline { stroke-width: 1.5; stroke-linejoin: round; stroke-linecap: round; }
.curve line { stroke-width: 1.5; stroke-linecap: round; }
.cursor { stroke: var(--player-text-meta); stroke-width: 1; stroke-dasharray: 3 3; }

.tooltip {
  position: absolute;
  top: 6px;
  pointer-events: none;
  background: var(--player-thumb-bg, rgba(0, 0, 0, 0.8));
  border: 1px solid var(--player-search-bg);
  border-radius: 8px;
  padding: 6px 8px;
  font-size: 10.5px;
  color: var(--player-text-primary);
  min-width: 110px;
}
.tooltip-time { font-weight: 600; margin-bottom: 4px; }
.tooltip-row { display: flex; align-items: center; gap: 6px; line-height: 1.5; }
.tooltip-label { flex: 1; color: var(--player-text-secondary); }
.tooltip-value { font-variant-numeric: tabular-nums; }

.legend { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 10px; }
.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 10.5px;
  padding: 3px 8px;
  border-radius: 10px;
  border: 1px solid var(--player-search-bg);
  background: transparent;
  color: var(--player-text-primary);
  cursor: pointer;
}
.legend-item.off { opacity: 0.4; }
.legend-item:disabled { cursor: not-allowed; }
.swatch { width: 8px; height: 8px; border-radius: 2px; display: inline-block; flex-shrink: 0; }

.graph-hint { font-size: 10px; color: var(--player-text-meta); margin: 8px 0 0; line-height: 1.4; }
</style>
