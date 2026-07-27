<script setup lang="ts">
import { computed, ref } from 'vue'
import { useLogs } from '@/lib/realtime'

const { logs, connected, clear } = useLogs()
const open = ref(false)

const levelBadge: Record<string, string> = {
  DEBUG: 'text-bg-secondary',
  INFO: 'text-bg-info',
  WARN: 'text-bg-warning',
  ERROR: 'text-bg-danger',
}

function badgeClass(level: string) {
  return levelBadge[level.toUpperCase()] ?? 'text-bg-secondary'
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString()
}

// Счётчик на кнопке имеет смысл, только пока панель закрыта — открыв её,
// пользователь и так всё видит.
const unreadCount = computed(() => (open.value ? 0 : logs.value.length))
</script>

<template>
  <button
      type="button"
      class="btn btn-dark rounded-circle log-toggle position-fixed shadow d-flex align-items-center justify-content-center"
      @click="open = !open"
      :title="open ? 'Hide log' : 'Show log'"
  >
    <!-- простая иконка терминала, без bootstrap-icons -->
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <path d="M6 9l4 3-4 3M13 15h5" />
    </svg>
    <span
        v-if="unreadCount > 0"
        class="position-absolute top-0 start-100 translate-middle badge rounded-pill text-bg-danger"
    >
      {{ unreadCount > 99 ? '99+' : unreadCount }}
    </span>
  </button>

  <Transition name="slide">
    <div v-if="open" class="log-panel position-fixed shadow-lg d-flex flex-column">
      <div class="d-flex align-items-center justify-content-between px-3 py-2 border-bottom">
        <div class="d-flex align-items-center gap-2">
          <strong class="small">Log</strong>
          <span class="badge rounded-pill" :class="connected ? 'text-bg-success' : 'text-bg-secondary'">
            {{ connected ? 'online' : 'offline' }}
          </span>
        </div>
        <div class="d-flex align-items-center gap-2">
          <button type="button" class="btn btn-sm btn-outline-secondary" @click="clear">Clear</button>
          <button type="button" class="btn-close" @click="open = false" aria-label="Close"></button>
        </div>
      </div>

      <div class="flex-grow-1 overflow-auto px-3 py-2 font-monospace log-body">
        <p v-if="!logs.length" class="text-body-secondary small mb-0">Nothing yet</p>
        <div v-for="(entry, i) in logs" :key="i" class="log-entry small mb-1">
          <span class="text-body-secondary">{{ formatTime(entry.time) }}</span>
          <span class="badge ms-2" :class="badgeClass(entry.level)">{{ entry.level }}</span>
          <span class="ms-2">{{ entry.msg }}</span>
          <span v-if="entry.attrs" class="text-body-secondary d-block ms-4">{{ JSON.stringify(entry.attrs) }}</span>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.log-toggle {
  bottom: 100px; /* над нижней панелью GlobalPlayer (86px) */
  right: 20px;
  width: 48px;
  height: 48px;
  z-index: 1060;
}

.log-panel {
  bottom: 160px;
  right: 20px;
  width: min(420px, calc(100vw - 40px));
  height: min(480px, calc(100vh - 200px));
  z-index: 1060;
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: 0.5rem;
}

.log-body {
  font-size: 0.8rem;
}

.slide-enter-active,
.slide-leave-active {
  transition:
      opacity 0.15s ease,
      transform 0.15s ease;
}
.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>